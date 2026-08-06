package types

// ToolCollection represents a collection of MCP Tools.
// A collection can contain a subset of all available tools in the MCPJungle system.
// This allows you to expose a limited set of tools to certain mcp clients.
// This struct is also the basis for the JSON configuration file used to register a new tool collection.
type ToolCollection struct {
	// Name is the unique name of the tool collection (mandatory).
	Name string `json:"name"`
	// IncludedTools is a list of tools included in this collection.
	IncludedTools []string `json:"included_tools,omitempty"`
	// IncludedServers is a list of MCP server names. All tools from these servers will be included.
	IncludedServers []string `json:"included_servers,omitempty"`
	// ExcludedTools is a list of tools to exclude from the collection (useful with IncludedServers).
	ExcludedTools []string `json:"excluded_tools,omitempty"`

	Description string `json:"description"`
}

// ToolCollectionEndpoints contains the endpoints a MCP client can use to access a tool collection.
type ToolCollectionEndpoints struct {
	StreamableHTTPEndpoint string `json:"streamable_http_endpoint"`
	SSEEndpoint            string `json:"sse_endpoint"`
	SSEMessageEndpoint     string `json:"sse_message_endpoint"`
}

type CreateToolCollectionResponse struct {
	*ToolCollectionEndpoints
}

type GetToolCollectionResponse struct {
	*ToolCollection
	*ToolCollectionEndpoints
}

// UpdateToolCollectionResponse contains the old and new configuration of a tool collection after a successful update.
type UpdateToolCollectionResponse struct {
	Name string `json:"name"`

	// Old contains the original configuration of the tool collection.
	Old *ToolCollection `json:"old"`
	// New contains the now-live configuration of the tool collection.
	New *ToolCollection `json:"new"`
}
