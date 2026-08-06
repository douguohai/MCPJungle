package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

func TestRunListTools_CollectionUsesEffectiveToolsAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/tool-collections/test-collection":
			_ = json.NewEncoder(w).Encode(types.GetToolCollectionResponse{
				ToolCollection: &types.ToolCollection{Name: "test-collection", Description: "desc"},
			})
		case "/api/v0/tool-collections/test-collection/effective-tools":
			_ = json.NewEncoder(w).Encode(map[string]any{"tools": []string{"from_server", "ghost"}})
		case "/api/v0/tools":
			_ = json.NewEncoder(w).Encode([]*types.Tool{{Name: "from_server", Description: "ok", Enabled: true}, {Name: "other", Description: "skip", Enabled: true}})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origServer := listToolsCmdServerName
	origCollection := listToolsCmdCollectionName
	defer func() {
		apiClient = origClient
		listToolsCmdServerName = origServer
		listToolsCmdCollectionName = origCollection
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	listToolsCmdServerName = ""
	listToolsCmdCollectionName = "test-collection"

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runListTools(cmd, nil); err != nil {
		t.Fatalf("runListTools returned error: %v", err)
	}

	output := out.String()
	if !bytes.Contains([]byte(output), []byte("from_server")) {
		t.Fatalf("expected output to contain resolved tool, got: %s", output)
	}
	if bytes.Contains([]byte(output), []byte("other")) {
		t.Fatalf("did not expect output to contain non-collection tool, got: %s", output)
	}
	if bytes.Contains([]byte(output), []byte("ghost")) {
		t.Fatalf("did not expect output to contain non-existing tool, got: %s", output)
	}
}
