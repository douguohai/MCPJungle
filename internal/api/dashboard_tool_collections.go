package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type dashboardToolCollectionCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

type dashboardToolCollectionTool struct {
	Name          string `json:"name"`
	CanonicalName string `json:"canonical_name"`
	Server        string `json:"server"`
	Description   string `json:"description,omitempty"`
}

type dashboardToolCollection struct {
	Name                   string                        `json:"name"`
	Description            string                        `json:"description,omitempty"`
	ToolCount              int                           `json:"tool_count"`
	Tools                  []dashboardToolCollectionTool `json:"tools"`
	StreamableHTTPEndpoint string                        `json:"streamable_http_endpoint"`
	SSEEndpoint            string                        `json:"sse_endpoint"`
	SSEMessageEndpoint     string                        `json:"sse_message_endpoint"`
}

type dashboardToolCollectionsResponse struct {
	ToolCollections []dashboardToolCollection     `json:"tool_collections"`
	EmptyState      *types.DashboardEmptyState    `json:"empty_state,omitempty"`
}

func (s *Server) dashboardToolCollectionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		collections, err := s.toolCollectionService.ListToolCollections()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := dashboardToolCollectionsResponse{
			ToolCollections: make([]dashboardToolCollection, 0, len(collections)),
		}
		for _, collection := range collections {
			item, err := s.buildDashboardToolCollection(c, collection)
			if err != nil {
				handleServiceError(c, err)
				return
			}
			resp.ToolCollections = append(resp.ToolCollections, item)
		}

		if len(resp.ToolCollections) == 0 {
			resp.EmptyState = &types.DashboardEmptyState{
				Title:       "No tool collections configured yet.",
				Description: "Create a tool collection to expose a focused subset of MCP tools.",
				Commands: []string{
					"mcpjungle create collection --conf collection.json",
					"mcpjungle list collections",
					"mcpjungle get collection <collection-name>",
				},
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardGetToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		collection, err := s.toolCollectionService.GetToolCollection(c.Param("name"))
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp, err := s.buildDashboardToolCollection(c, *collection)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardCreateToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dashboardToolCollectionCreateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		includedTools, err := json.Marshal(input.Tools)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tools payload"})
			return
		}

		collection := &model.ToolCollection{
			Name:          input.Name,
			Description:   input.Description,
			IncludedTools: includedTools,
		}
		if err := s.toolCollectionService.CreateToolCollection(collection); err != nil {
			handleServiceError(c, err)
			return
		}

		resp, err := s.buildDashboardToolCollection(c, *collection)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func (s *Server) dashboardDeleteToolCollectionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.toolCollectionService.DeleteToolCollection(c.Param("name")); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func (s *Server) buildDashboardToolCollection(c *gin.Context, collection model.ToolCollection) (dashboardToolCollection, error) {
	toolNames, err := collection.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return dashboardToolCollection{}, err
	}

	tools := make([]dashboardToolCollectionTool, 0, len(toolNames))
	for _, toolName := range toolNames {
		item := dashboardToolCollectionTool{
			CanonicalName: toolName,
			Name:          toolName,
			Server:        "Unknown",
		}
		if tool, err := s.mcpService.GetTool(toolName); err == nil {
			item.Name = tool.Name
			if server, serverErr := s.mcpService.GetToolParentServer(toolName); serverErr == nil {
				item.Server = server.Name
			}
			item.CanonicalName = toolName
			item.Description = tool.Description
		}
		respName := item.Name
		parts := strings.SplitN(toolName, "__", 2)
		if len(parts) == 2 {
			respName = parts[1]
		}
		item.Name = respName
		tools = append(tools, item)
	}

	endpoints := getToolCollectionEndpoints(c, collection.Name)

	return dashboardToolCollection{
		Name:                   collection.Name,
		Description:            collection.Description,
		ToolCount:              len(tools),
		Tools:                  tools,
		StreamableHTTPEndpoint: endpoints.StreamableHTTPEndpoint,
		SSEEndpoint:            endpoints.SSEEndpoint,
		SSEMessageEndpoint:     endpoints.SSEMessageEndpoint,
	}, nil
}
