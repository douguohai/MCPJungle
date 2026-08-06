package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestCreateToolCollection(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		expectedResponse := &types.CreateToolCollectionResponse{
			ToolCollectionEndpoints: &types.ToolCollectionEndpoints{
				StreamableHTTPEndpoint: "/api/v0/tool-collections/test-collection",
				SSEEndpoint:            "/api/v0/tool-collections/test-collection/sse",
				SSEMessageEndpoint:     "/api/v0/tool-collections/test-collection/sse/message",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-collections") {
				t.Errorf("Expected path to end with /tool-collections, got %s", r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify request body
			var toolCollection types.ToolCollection
			if err := json.NewDecoder(r.Body).Decode(&toolCollection); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			if toolCollection.Name != "test-collection" {
				t.Errorf("Expected Name 'test-collection', got %s", toolCollection.Name)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		toolCollection := &types.ToolCollection{
			Name:          "test-collection",
			Description:   "Test tool collection",
			IncludedTools: []string{"tool1", "tool2"},
		}

		response, err := client.CreateToolCollection(toolCollection)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if response.StreamableHTTPEndpoint != expectedResponse.StreamableHTTPEndpoint {
			t.Errorf("Expected StreamableHTTPEndpoint %s, got %s", expectedResponse.StreamableHTTPEndpoint, response.StreamableHTTPEndpoint)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid tool collection configuration"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		toolCollection := &types.ToolCollection{
			Name:          "test-collection",
			Description:   "Test tool collection",
			IncludedTools: []string{"tool1"},
		}

		response, err := client.CreateToolCollection(toolCollection)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if response != nil {
			t.Error("Expected nil response on error")
		}

		expectedError := "request failed with status: 400, message: Invalid tool collection configuration"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetToolCollection(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedCollection := &types.ToolCollection{
			Name:          "test-collection",
			Description:   "Test tool collection",
			IncludedTools: []string{"tool1", "tool2"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-collections/test-collection") {
				t.Errorf("Expected path to end with /tool-collections/test-collection, got %s", r.URL.Path)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedCollection)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		collection, err := client.GetToolCollection("test-collection")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if collection.Name != expectedCollection.Name {
			t.Errorf("Expected Name %s, got %s", expectedCollection.Name, collection.Name)
		}
		if collection.Description != expectedCollection.Description {
			t.Errorf("Expected Description %s, got %s", expectedCollection.Description, collection.Description)
		}
		if len(collection.IncludedTools) != len(expectedCollection.IncludedTools) {
			t.Errorf("Expected IncludedTools length %d, got %d", len(expectedCollection.IncludedTools), len(collection.IncludedTools))
		}
	})

	t.Run("collection not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool collection not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		collection, err := client.GetToolCollection("non-existent-collection")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if collection != nil {
			t.Error("Expected nil collection on error")
		}

		expectedError := "request failed with status: 404, message: Tool collection not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestDeleteToolCollection(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		collectionName := "test-collection"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", r.Method)
			}
			expectedPath := "/api/v0/tool-collections/" + collectionName
			if !strings.HasSuffix(r.URL.Path, expectedPath) {
				t.Errorf("Expected path to end with %s, got %s", expectedPath, r.URL.Path)
			}

			// Return success response
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteToolCollection(collectionName)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("collection not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool collection not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteToolCollection("non-existent-collection")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		expectedError := "request failed with status: 404, message: Tool collection not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestListToolCollections(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		expectedCollections := []*types.ToolCollection{
			{
				Name:          "collection1",
				Description:   "First collection",
				IncludedTools: []string{"tool1"},
			},
			{
				Name:          "collection2",
				Description:   "Second collection",
				IncludedTools: []string{"tool2", "tool3"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-collections") {
				t.Errorf("Expected path to end with /tool-collections, got %s", r.URL.Path)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedCollections)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		collections, err := client.ListToolCollections()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(collections) != len(expectedCollections) {
			t.Errorf("Expected %d collections, got %d", len(expectedCollections), len(collections))
		}

		for i, collection := range collections {
			if collection.Name != expectedCollections[i].Name {
				t.Errorf("Expected collection[%d].Name %s, got %s", i, expectedCollections[i].Name, collection.Name)
			}
		}
	})

	t.Run("empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		collections, err := client.ListToolCollections()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(collections) != 0 {
			t.Errorf("Expected empty list, got %d collections", len(collections))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		collections, err := client.ListToolCollections()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if collections != nil {
			t.Error("Expected nil collections on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetToolCollectionEffectiveTools(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedTools := []string{"alpha", "beta"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-collections/test-collection/effective-tools") {
				t.Errorf("Expected path to end with /tool-collections/test-collection/effective-tools, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"tools": expectedTools})
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.GetToolCollectionEffectiveTools("test-collection")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(tools) != len(expectedTools) {
			t.Fatalf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}
		for i := range expectedTools {
			if tools[i] != expectedTools[i] {
				t.Errorf("Expected tools[%d] to be %s, got %s", i, expectedTools[i], tools[i])
			}
		}
	})

	t.Run("collection not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool collection not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.GetToolCollectionEffectiveTools("missing")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if tools != nil {
			t.Fatal("Expected nil tools on error")
		}
	})
}
