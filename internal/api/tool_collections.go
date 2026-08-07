package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) createToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.ToolCollection
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := s.toolCollectionService.CreateToolCollection(&input); err != nil {
			handleServiceError(c, err)
			return
		}
		resp := &types.CreateToolCollectionResponse{
			ToolCollectionEndpoints: getToolCollectionEndpoints(c, input.Name),
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// listToolCollectionsHandler handles returns a list of all tool collections.
// This API only provides basic information about each tool collection, ie, name and description.
func (s *Server) listToolCollectionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		collections, err := s.toolCollectionService.ListToolCollections()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := make([]*types.ToolCollection, len(collections))
		for i, coll := range collections {
			resp[i] = &types.ToolCollection{
				Name:        coll.Name,
				Description: coll.Description,
			}

			collTools, err := coll.GetTools()
			if err != nil {
				slog.Error("listToolCollections GetTools failed", "collection", coll.Name, "error", err)
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": "internal server error"},
				)
				return
			}
			resp[i].IncludedTools = collTools

			collServers, err := coll.GetServers()
			if err != nil {
				slog.Error("listToolCollections GetServers failed", "collection", coll.Name, "error", err)
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": "internal server error"},
				)
				return
			}
			resp[i].IncludedServers = collServers

			collExcluded, err := coll.GetExcludedTools()
			if err != nil {
				slog.Error("listToolCollections GetExcludedTools failed", "collection", coll.Name, "error", err)
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": "internal server error"},
				)
				return
			}
			resp[i].ExcludedTools = collExcluded
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) getToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		collection, err := s.toolCollectionService.GetToolCollection(name)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := &types.GetToolCollectionResponse{
			ToolCollection: &types.ToolCollection{
				Name:        collection.Name,
				Description: collection.Description,
			},
			ToolCollectionEndpoints: getToolCollectionEndpoints(c, collection.Name),
		}

		// Get included tools
		var tools []string
		tools, err = collection.GetTools()
		if err != nil {
			slog.Error("getToolCollection GetTools failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.IncludedTools = tools

		// Get included servers
		var servers []string
		servers, err = collection.GetServers()
		if err != nil {
			slog.Error("getToolCollection GetServers failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.IncludedServers = servers

		// Get excluded tools
		var excludedTools []string
		excludedTools, err = collection.GetExcludedTools()
		if err != nil {
			slog.Error("getToolCollection GetExcludedTools failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.ExcludedTools = excludedTools

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) getToolCollectionEffectiveToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		tools, err := s.toolCollectionService.ResolveEffectiveTools(name)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"tools": tools})
	}
}

func (s *Server) deleteToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		err := s.toolCollectionService.DeleteToolCollection(name)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		// TODO: return 404 if the collection did not exist.
		//  The tool collection service should return ErrToolCollectionNotFound if the collection does not exist.
		//  The CLI should then handle this and output "collection does not exist".
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) updateToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "collection name is required"})
			return
		}

		var input model.ToolCollection
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		originalConf, err := s.toolCollectionService.UpdateToolCollection(name, &input)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		// create and send response object
		resp := &types.UpdateToolCollectionResponse{
			Name: name,
			Old: &types.ToolCollection{
				Name:        originalConf.Name,
				Description: originalConf.Description,
			},
			New: &types.ToolCollection{
				Name:        input.Name,
				Description: input.Description,
			},
		}

		var origTools []string
		origTools, err = originalConf.GetTools()
		if err != nil {
			slog.Error("updateToolCollection GetTools orig failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.Old.IncludedTools = origTools

		var origServers []string
		origServers, err = originalConf.GetServers()
		if err != nil {
			slog.Error("updateToolCollection GetServers orig failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.Old.IncludedServers = origServers

		var origExcluded []string
		origExcluded, err = originalConf.GetExcludedTools()
		if err != nil {
			slog.Error("updateToolCollection GetExcludedTools orig failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.Old.ExcludedTools = origExcluded

		var newTools []string
		newTools, err = input.GetTools()
		if err != nil {
			slog.Error("updateToolCollection GetTools new failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.New.IncludedTools = newTools

		var newServers []string
		newServers, err = input.GetServers()
		if err != nil {
			slog.Error("updateToolCollection GetServers new failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.New.IncludedServers = newServers

		var newExcluded []string
		newExcluded, err = input.GetExcludedTools()
		if err != nil {
			slog.Error("updateToolCollection GetExcludedTools new failed", "collection", name, "error", err)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}
		resp.New.ExcludedTools = newExcluded

		c.JSON(http.StatusOK, resp)
	}
}

// toolCollectionMCPServerCallHandler handles incoming MCP requests from for a specific tool collection.
func (s *Server) toolCollectionMCPServerCallHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get the Proxy MCP server for the specified tool collection
		collectionName := c.Param("name")
		collectionMcpServer, exists := s.toolCollectionService.GetToolCollectionMCPServer(collectionName)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool collection not found: %s", collectionName)})
			return
		}

		// serve the MCP request using the MCP server
		// TODO: Make this API more efficient
		// This api sits in the hot path because we expect high traffic on MCP tool calling.
		// It is inefficient to create a new StreamableHTTPServer for each request.
		// Maybe pre-create a StreamableHTTPServer for each tool collection and store it in the ToolCollectionMCPServer struct?
		streamableServer := server.NewStreamableHTTPServer(collectionMcpServer)
		streamableServer.ServeHTTP(c.Writer, c.Request)
	}
}

// getCollectionSseServer returns a server.SSEServer for a specific collection, creating one if it doesn't already exist.
// It ensures that each tool collection has its own SSE server with the correct dynamic base path.
func (s *Server) getCollectionSseServer(collectionName string) (*server.SSEServer, error) {
	// Try to get existing server first
	if serverVal, ok := s.collectionSseServers.Load(collectionName); ok {
		return serverVal.(*server.SSEServer), nil
	}

	// Get the sse MCP proxy server for the collection
	collectionSseMcpServer, exists := s.toolCollectionService.GetToolCollectionSseMCPServer(collectionName)
	if !exists {
		return nil, fmt.Errorf("tool collection not found: %s", collectionName)
	}

	// Create new server with the correct dynamic base path
	sseServer := server.NewSSEServer(
		collectionSseMcpServer,
		server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
			// Return the collection-specific base path
			return fmt.Sprintf("%s/collections/%s", V0PathPrefix, collectionName)
		}),
	)

	// Store for future use
	s.collectionSseServers.Store(collectionName, sseServer)

	return sseServer, nil
}

// toolCollectionSseMCPServerCallHandler handles SSE connection requests (/sse) for a specific tool collection.
func (s *Server) toolCollectionSseMCPServerCallHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionName := c.Param("name")

		collectionSseMcpServer, err := s.getCollectionSseServer(collectionName)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": fmt.Sprintf("failed to get sse server for collection %s: %v", collectionName, err)},
			)
			return
		}

		collectionSseMcpServer.SSEHandler().ServeHTTP(c.Writer, c.Request)
	}
}

// toolCollectionSseMCPServerCallMessageHandler handles SSE connection requests (/message) for a specific tool collection.
func (s *Server) toolCollectionSseMCPServerCallMessageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionName := c.Param("name")

		collectionSseMcpServer, err := s.getCollectionSseServer(collectionName)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": fmt.Sprintf("failed to get sse server for collection: %s", collectionName)},
			)
			return
		}

		collectionSseMcpServer.MessageHandler().ServeHTTP(c.Writer, c.Request)
	}
}

// getToolCollectionEndpoints deduces the proxy MCP server endpoint URLs for a given tool collection.
// It returns the streamable HTTP endpoint and the SSE endpoints
func getToolCollectionEndpoints(c *gin.Context, collectionName string) *types.ToolCollectionEndpoints {
	// This logic of creating the API endpoints is duplicated from internal/api/server.go
	// TODO: centralize this logic into one place and use that everywhere.
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	endpointURL := &url.URL{
		Scheme: scheme,
		Host:   c.Request.Host,
		Path:   fmt.Sprintf("%s/collections/%s", V0PathPrefix, collectionName),
	}
	baseEndpoint := endpointURL.String()

	return &types.ToolCollectionEndpoints{
		StreamableHTTPEndpoint: baseEndpoint + "/mcp",
		SSEEndpoint:            baseEndpoint + "/sse",
		SSEMessageEndpoint:     baseEndpoint + "/message",
	}
}
