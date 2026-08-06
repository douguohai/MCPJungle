package types

import (
	"encoding/json"
	"testing"
)

func TestToolCollection(t *testing.T) {
	t.Parallel()

	// Test struct creation
	collection := ToolCollection{
		Name:          "test-collection",
		IncludedTools: []string{"tool1", "tool2"},
	}

	if collection.Name != "test-collection" {
		t.Errorf("Expected Name to be 'test-collection', got %s", collection.Name)
	}
	if len(collection.IncludedTools) != 2 {
		t.Errorf("Expected IncludedTools to have 2 items, got %d", len(collection.IncludedTools))
	}
}

func TestToolCollectionJSONMarshaling(t *testing.T) {
	t.Parallel()

	collection := ToolCollection{
		Name:          "json-collection",
		IncludedTools: []string{"json-tool1"},
		Description:   "Collection for JSON testing",
	}

	data, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCollection: %v", err)
	}

	expected := `{"name":"json-collection","included_tools":["json-tool1"],"description":"Collection for JSON testing"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestToolCollectionJSONMarshalingWithNewFields(t *testing.T) {
	t.Parallel()

	collection := ToolCollection{
		Name:            "advanced-collection",
		IncludedTools:   []string{"manual-tool1"},
		IncludedServers: []string{"time", "deepwiki"},
		ExcludedTools:   []string{"time__convert_time"},
		Description:     "Collection with server inclusion and exclusion",
	}

	data, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCollection: %v", err)
	}

	// Unmarshal to verify all fields are present
	var unmarshaled ToolCollection
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ToolCollection: %v", err)
	}

	if unmarshaled.Name != collection.Name {
		t.Errorf("Expected Name %s, got %s", collection.Name, unmarshaled.Name)
	}
	if len(unmarshaled.IncludedTools) != 1 || unmarshaled.IncludedTools[0] != "manual-tool1" {
		t.Errorf("Expected IncludedTools [manual-tool1], got %v", unmarshaled.IncludedTools)
	}
	if len(unmarshaled.IncludedServers) != 2 {
		t.Errorf("Expected 2 IncludedServers, got %v", unmarshaled.IncludedServers)
	}
	if len(unmarshaled.ExcludedTools) != 1 || unmarshaled.ExcludedTools[0] != "time__convert_time" {
		t.Errorf("Expected ExcludedTools [time__convert_time], got %v", unmarshaled.ExcludedTools)
	}
}

func TestCreateToolCollectionResponse(t *testing.T) {
	t.Parallel()

	// Test struct creation
	response := CreateToolCollectionResponse{
		ToolCollectionEndpoints: &ToolCollectionEndpoints{
			StreamableHTTPEndpoint: "/api/tool-collections/test-collection/stream",
			SSEEndpoint:            "/api/tool-collections/test-collection/sse",
			SSEMessageEndpoint:     "/api/tool-collections/test-collection/sse/message",
		},
	}

	if response.StreamableHTTPEndpoint != "/api/tool-collections/test-collection/stream" {
		t.Errorf("Expected StreamableHTTPEndpoint to be '/api/tool-collections/test-collection/stream', got %s", response.StreamableHTTPEndpoint)
	}
}

func TestCreateToolCollectionResponseJSONMarshaling(t *testing.T) {
	t.Parallel()

	response := CreateToolCollectionResponse{
		ToolCollectionEndpoints: &ToolCollectionEndpoints{
			StreamableHTTPEndpoint: "/api/tool-collections/json-collection/stream",
			SSEEndpoint:            "/api/tool-collections/json-collection/sse",
			SSEMessageEndpoint:     "/api/tool-collections/json-collection/sse/message",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal CreateToolCollectionResponse: %v", err)
	}

	expected := `{"streamable_http_endpoint":"/api/tool-collections/json-collection/stream","sse_endpoint":"/api/tool-collections/json-collection/sse","sse_message_endpoint":"/api/tool-collections/json-collection/sse/message"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestGetToolCollectionResponse(t *testing.T) {
	t.Parallel()

	// Test struct creation
	toolCollection := &ToolCollection{
		Name:          "get-collection",
		IncludedTools: []string{"get-tool1"},
		Description:   "Collection for get testing",
	}

	response := GetToolCollectionResponse{
		ToolCollection: toolCollection,
		ToolCollectionEndpoints: &ToolCollectionEndpoints{
			StreamableHTTPEndpoint: "/api/tool-collections/get-collection/stream",
			SSEEndpoint:            "/api/tool-collections/get-collection/sse",
			SSEMessageEndpoint:     "/api/tool-collections/get-collection/sse/message",
		},
	}

	if response.ToolCollection != toolCollection {
		t.Error("Expected ToolCollection pointer to match")
	}
	if response.StreamableHTTPEndpoint != "/api/tool-collections/get-collection/stream" {
		t.Errorf("Expected StreamableHTTPEndpoint to be '/api/tool-collections/get-collection/stream', got %s", response.StreamableHTTPEndpoint)
	}
}
