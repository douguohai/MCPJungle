// Package toolcollection provides functionality to manage tool collections and their associated MCP proxy servers.
package toolcollection

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/mcpjungle/mcpjungle/pkg/util"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"gorm.io/gorm"
)

var ErrToolCollectionNotFound = fmt.Errorf("tool collection not found: %w", apierrors.ErrNotFound)

// ValidCollectionName is a regex that matches valid tool collection names.
// A valid tool collection name must start with an alphanumeric character and can contain
// alphanumeric characters, underscores, and hyphens.
// This ensures that the collection name can be safely used in URLs.
var ValidCollectionName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ToolCollectionService provides methods to manage tool collections and their associated MCP proxy servers.
type ToolCollectionService struct {
	db *gorm.DB

	mcpService *mcp.MCPService

	// mcpServers manages the MCP proxy servers for all the tool collections
	// key: tool collection name, value: MCP proxy server
	mcpServers map[string]*server.MCPServer
	// mcpServersMu protects access to the mcpServers map
	mcpServersMu sync.RWMutex

	// sseMcpServers manages the SSE MCP proxy servers for all the tool collections
	// key: tool collection name, value: MCP proxy server
	sseMcpServers map[string]*server.MCPServer
	// sseMcpServerMu protects access to the sseMcpServers map
	sseMcpServerMu sync.RWMutex
}

func NewToolCollectionService(db *gorm.DB, mcpService *mcp.MCPService) (*ToolCollectionService, error) {
	s := &ToolCollectionService{
		db:         db,
		mcpService: mcpService,

		mcpServers:   make(map[string]*server.MCPServer),
		mcpServersMu: sync.RWMutex{},

		sseMcpServers:  make(map[string]*server.MCPServer),
		sseMcpServerMu: sync.RWMutex{},
	}

	// register callbacks with mcp service to be notified when a tool gets added/removed
	mcpService.SetToolDeletionCallback(s.handleToolDeletion)
	mcpService.SetToolAdditionCallback(s.handleToolAddition)

	if err := s.initToolCollectionMCPServers(); err != nil {
		return nil, fmt.Errorf("failed to initialize tool collection MCP servers: %w", err)
	}
	return s, nil
}

// CreateToolCollection creates a new tool collection in the database and a Proxy MCP server that just exposes the specified tools.
func (s *ToolCollectionService) CreateToolCollection(collection *model.ToolCollection) error {
	// validate the tool collection name
	if len(collection.Name) == 0 {
		return fmt.Errorf("tool collection name cannot be empty: %w", apierrors.ErrInvalidInput)
	}
	if !ValidCollectionName.MatchString(collection.Name) {
		return fmt.Errorf(
			"invalid collection name: name must start with an alphanumeric character and "+
				"can only contain alphanumeric characters, underscores, and hyphens: %w",
			apierrors.ErrInvalidInput,
		)
	}

	// resolve all effective tools for this collection
	toolNames, err := collection.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return fmt.Errorf("failed to resolve effective tools: %w", err)
	}
	if len(toolNames) == 0 {
		return fmt.Errorf(
			"tool collection must contain at least one tool after resolving servers and exclusions: %w",
			apierrors.ErrInvalidInput,
		)
	}

	// create the proxy MCP servers that expose only specified tools
	mcpServer := s.newMCPServer(collection.Name)
	sseMcpServer := s.newSseMCPServer(collection.Name)

	// populate the MCP servers with the specified tools
	// this also has a side effect of validating that the tools exist in mcpjungle.
	// if a tool does not exist, return an error without creating the collection.
	for _, name := range toolNames {
		tool, exists := s.mcpService.GetToolInstance(name)
		if !exists {
			return fmt.Errorf("tool %s does not exist or is disabled: %w", name, apierrors.ErrInvalidInput)
		}

		parentServer, err := s.mcpService.GetToolParentServer(name)
		if err != nil {
			return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", name, err)
		}

		if parentServer.Transport == types.TransportSSE {
			sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
		} else {
			mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
		}
	}

	// first, add the tool collection to the database
	// this also checks for uniqueness of the collection's name
	if err := s.db.Create(collection).Error; err != nil {
		return fmt.Errorf("failed to create tool collection: %w", err)
	}

	// finally, add the proxy MCPs to the tool collection MCPs manager so that it is ready to serve
	s.addToolCollectionMCPServer(collection.Name, mcpServer)
	s.addToolCollectionSseMCPServer(collection.Name, sseMcpServer)

	return nil
}

// UpdateToolCollection updates an existing tool collection without causing any downtime for its MCP proxy servers.
// It returns the configuration of the original tool collection before the update.
// If the tool collection does not exist, it returns ErrToolCollectionNotFound.
func (s *ToolCollectionService) UpdateToolCollection(name string, updatedCollection *model.ToolCollection) (*model.ToolCollection, error) {
	oldCollection, err := s.GetToolCollection(name)
	if err != nil {
		if errors.Is(err, ErrToolCollectionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to retrieve the tool collection: %w", err)
	}

	// determine which tools were added or removed from the collection
	oldToolNames, err := oldCollection.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools of original collection: %w", err)
	}
	updatedToolNames, err := updatedCollection.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools of the updated collection: %w", err)
	}

	toolsAdded, toolsRemoved := util.DiffTools(oldToolNames, updatedToolNames)

	// if nothing was actually changed in the collection, no need to proceed further
	if updatedCollection.Description == oldCollection.Description && len(toolsAdded) == 0 && len(toolsRemoved) == 0 {
		return oldCollection, nil
	}

	// determine the changes to make to the tool collection's proxy MCP server instances (normal + SSE)
	// all changes are ultimately made at the end of this method to avoid inconsistent state in case of errors.
	mcpServer, exists := s.GetToolCollectionMCPServer(name)
	if !exists {
		return nil, fmt.Errorf("MCP server for tool collection %s does not exist", name)
	}
	sseMcpServer, exists := s.GetToolCollectionSseMCPServer(name)
	if !exists {
		return nil, fmt.Errorf("SSE MCP server for tool collection %s does not exist", name)
	}

	// tools added to the collection must be added to its MCP server instances
	var sseToolsToAdd, normalToolsToAdd []mcpgo.Tool
	for _, toolName := range toolsAdded {
		tool, exists := s.mcpService.GetToolInstance(toolName)
		if !exists {
			return nil, fmt.Errorf("tool %s does not exist or is disabled: %w", toolName, apierrors.ErrInvalidInput)
		}

		parentServer, err := s.mcpService.GetToolParentServer(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent MCP server of the tool %s: %w", toolName, err)
		}

		if parentServer.Transport == types.TransportSSE {
			sseToolsToAdd = append(sseToolsToAdd, tool)
		} else {
			normalToolsToAdd = append(normalToolsToAdd, tool)
		}
	}

	// tools removed from the collection must be removed from its MCP server instances
	var sseToolsToRemove, normalToolsToRemove []string
	for _, toolName := range toolsRemoved {
		parentServer, err := s.mcpService.GetToolParentServer(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent MCP server of the tool %s: %w", toolName, err)
		}

		if parentServer.Transport == types.TransportSSE {
			sseToolsToRemove = append(sseToolsToRemove, toolName)
		} else {
			normalToolsToRemove = append(normalToolsToRemove, toolName)
		}
	}

	// make all the changes together to avoid inconsistent state in case of errors
	mcpServer.DeleteTools(normalToolsToRemove...)
	sseMcpServer.DeleteTools(sseToolsToRemove...)

	for _, tool := range normalToolsToAdd {
		mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
	}
	for _, tool := range sseToolsToAdd {
		sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
	}

	// as a final step, update the tool collection record in the database
	// we only persist this update after successfully updating the in-memory state

	// ensure the collection name remains unchanged in the db record
	updatedCollection.Name = name
	if err := s.db.Model(&model.ToolCollection{}).Where("name = ?", name).Updates(updatedCollection).Error; err != nil {
		return nil, fmt.Errorf("failed to update tool collection in DB: %w", err)
	}

	return oldCollection, nil
}

// ResolveEffectiveTools resolves all effective tools for the specified tool collection.
// The resulting list is sorted for deterministic API responses and tests.
func (s *ToolCollectionService) ResolveEffectiveTools(name string) ([]string, error) {
	collection, err := s.GetToolCollection(name)
	if err != nil {
		return nil, err
	}

	tools, err := collection.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools for collection %s: %w", name, err)
	}

	sort.Strings(tools)
	return tools, nil
}

// GetToolCollection retrieves a tool collection by name from the database.
func (s *ToolCollectionService) GetToolCollection(name string) (*model.ToolCollection, error) {
	var collection model.ToolCollection
	if err := s.db.Where("name = ?", name).First(&collection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrToolCollectionNotFound
		}
		return nil, err
	}
	return &collection, nil
}

// ListToolCollections retrieves all tool collections from the database.
func (s *ToolCollectionService) ListToolCollections() ([]model.ToolCollection, error) {
	var collections []model.ToolCollection
	if err := s.db.Find(&collections).Error; err != nil {
		return nil, err
	}
	return collections, nil
}

func (s *ToolCollectionService) DeleteToolCollection(name string) error {
	s.deleteToolCollectionMCPServers(name)

	err := s.db.Unscoped().Where("name = ?", name).Delete(&model.ToolCollection{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete tool collection: %w", err)
	}
	return nil
}

// GetToolCollectionMCPServer retrieves the MCP proxy server for a given tool collection name.
func (s *ToolCollectionService) GetToolCollectionMCPServer(name string) (*server.MCPServer, bool) {
	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()
	mcpServer, exists := s.mcpServers[name]
	return mcpServer, exists
}

// GetToolCollectionSseMCPServer retrieves the SSE MCP proxy server for a given tool collection name.
func (s *ToolCollectionService) GetToolCollectionSseMCPServer(name string) (*server.MCPServer, bool) {
	s.sseMcpServerMu.RLock()
	defer s.sseMcpServerMu.RUnlock()
	mcpServer, exists := s.sseMcpServers[name]
	return mcpServer, exists
}

// newMCPServer creates a new MCP proxy server for a given tool collection name.
// The advertised version is tied to the mcpjungle server version (pkg/version)
// so each tool-collection proxy reports the host's version instead of a hardcoded
// string.
func (s *ToolCollectionService) newMCPServer(collectionName string) *server.MCPServer {
	return server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for tool collection: %s", collectionName),
		version.GetVersion(),
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
}

// newSseMCPServer creates a new SSE MCP proxy server for a given tool collection name.
func (s *ToolCollectionService) newSseMCPServer(collectionName string) *server.MCPServer {
	return server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for SSE transport for tool collection: %s", collectionName),
		version.GetVersion(),
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
}

// addToolCollectionMCPServer adds or updates the MCP proxy server for a given tool collection name.
// If a collection with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolCollectionService) addToolCollectionMCPServer(name string, mcpServer *server.MCPServer) {
	s.mcpServersMu.Lock()
	defer s.mcpServersMu.Unlock()
	s.mcpServers[name] = mcpServer
}

// addToolCollectionSseMCPServer adds or updates the SSE MCP proxy server for a given tool collection name.
// If a collection with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolCollectionService) addToolCollectionSseMCPServer(name string, mcpServer *server.MCPServer) {
	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()
	s.sseMcpServers[name] = mcpServer
}

// deleteToolCollectionMCPServers removes the MCP proxy servers for a given tool collection name.
func (s *ToolCollectionService) deleteToolCollectionMCPServers(name string) {
	// first, acquire both locks to ensure complete cleanup of the collection
	s.mcpServersMu.Lock()
	defer s.mcpServersMu.Unlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	// proceed to delete both normal & sse proxies for the collection, then release the locks
	delete(s.mcpServers, name)
	delete(s.sseMcpServers, name)
}

// initToolCollectionMCPServers initializes the MCP proxy servers for all existing tool collections in the database.
// It initializes both the mcpServers and sseMcpServers.
func (s *ToolCollectionService) initToolCollectionMCPServers() error {
	collections, err := s.ListToolCollections()
	if err != nil {
		return fmt.Errorf("failed to list tool collections from DB: %w", err)
	}

	for _, collection := range collections {
		mcpServer := s.newMCPServer(collection.Name)
		sseMcpServer := s.newSseMCPServer(collection.Name)

		toolNames, err := collection.ResolveEffectiveTools(s.mcpService)
		if err != nil {
			// If resolution of any server or specific tool fails for a collection, the error is logged and
			// the collection is added as an empty MCP server to the proxy.
			// This is not ideal, but it ensures that mcpjungle server startup doesn't fail.
			// TODO: Change design to include all other servers & tools that were resolved.
			// See https://github.com/mcpjungle/MCPJungle/issues/233
			log.Printf(
				"[ERROR] failed to resolve effective tools for tool collection %s during startup; the tool collection will be initialized as empty: %v",
				collection.Name,
				err,
			)
			s.addToolCollectionMCPServer(collection.Name, mcpServer)
			s.addToolCollectionSseMCPServer(collection.Name, sseMcpServer)
			continue
		}
		// TODO: Log a warning if a collection has no tools, ie, len(toolNames) == 0

		for _, name := range toolNames {
			tool, exists := s.mcpService.GetToolInstance(name)
			if !exists {
				// it is possible that a tool collection contains a tool that does not exist.
				// this should not prevent server startup, so just skip instead of returning an error.
				// TODO: Add a warning log here.
				continue
			}

			parentServer, err := s.mcpService.GetToolParentServer(name)
			if err != nil {
				return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", name, err)
			}

			if parentServer.Transport == types.TransportSSE {
				sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
			} else {
				mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
			}
		}

		s.addToolCollectionMCPServer(collection.Name, mcpServer)
		s.addToolCollectionSseMCPServer(collection.Name, sseMcpServer)
	}

	return nil
}

// handleToolDeletion is a callback that is called when one or more tools is deleted or disabled.
// It removes the tools from all tool collection MCP proxy servers.
func (s *ToolCollectionService) handleToolDeletion(tools ...string) {
	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	for _, mcpServer := range s.mcpServers {
		mcpServer.DeleteTools(tools...)
	}

	for _, sseMcpServer := range s.sseMcpServers {
		sseMcpServer.DeleteTools(tools...)
	}
}

// handleToolAddition is a callback that is called when a tool is added or (re)enabled in mcpjungle.
// this callback adds the new tool to MCP proxy servers of all collections that include it.
func (s *ToolCollectionService) handleToolAddition(newTool string) error {
	// get all tool collections from the database
	collections, err := s.ListToolCollections()
	if err != nil {
		return fmt.Errorf("failed to list tool collections from DB: %w", err)
	}

	// find all collections that include the added tool
	collectionsToUpdate := make([]string, 0, len(collections))
	for i := range collections {
		name := collections[i].Name
		collectionTools, err := collections[i].ResolveEffectiveTools(s.mcpService)
		if err != nil {
			return fmt.Errorf("failed to resolve effective tools for collection %s: %w", name, err)
		}
		for _, t := range collectionTools {
			if t != newTool {
				continue
			}
			// current collection includes the added tool, so add the tool instance to the collection's MCP server
			collectionsToUpdate = append(collectionsToUpdate, name)
			// no need to check other tools in this collection anymore, so exit the loop and move on to the next collection
			break
		}
	}

	newToolInstance, exists := s.mcpService.GetToolInstance(newTool)
	if !exists {
		// this should not happen because the tool should exist if we are in this callback
		return fmt.Errorf("tool instance %s does not exist", newTool)
	}

	parentServer, err := s.mcpService.GetToolParentServer(newTool)
	if err != nil {
		return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", newTool, err)
	}

	// add the new tool instance to all relevant MCP proxy servers
	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	for _, name := range collectionsToUpdate {
		if parentServer.Transport == types.TransportSSE {
			sseMcpServer, exists := s.sseMcpServers[name]
			if exists {
				sseMcpServer.AddTool(newToolInstance, s.mcpService.MCPProxyToolCallHandler)
			}
			continue
		}

		mcpServer, exists := s.mcpServers[name]
		if exists {
			mcpServer.AddTool(newToolInstance, s.mcpService.MCPProxyToolCallHandler)
		}
	}

	return nil
}
