package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mcpjungle/mcpjungle/internal/configresolver"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{Use: "create", Short: "Create entities in mcpjungle", Annotations: map[string]string{"group": string(subCommandGroupAdvanced), "order": "4"}}
var createToolGroupCmd = &cobra.Command{
	Use: "group --conf <file>", Short: "Create a Group of MCP Tools",
	Long: "Create a tool collection from a configuration file.", RunE: runCreateToolGroup,
}
var createToolGroupConfigFilePath string

func init() {
	createToolGroupCmd.Flags().StringVarP(&createToolGroupConfigFilePath, "conf", "c", "", "Path to the Group configuration file")
	_ = createToolGroupCmd.MarkFlagRequired("conf")
	createCmd.AddCommand(createToolGroupCmd)
	rootCmd.AddCommand(createCmd)
}

func runCreateToolGroup(cmd *cobra.Command, args []string) error {
	group, err := readToolGroupConfig(createToolGroupConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", createToolGroupConfigFilePath, err)
	}
	resp, err := apiClient.CreateToolGroup(group)
	if err != nil {
		return fmt.Errorf("failed to create tool group: %w", err)
	}
	cmd.Printf("Tool Group %s created successfully\n%s\n", group.Name, resp.StreamableHTTPEndpoint)
	return nil
}

func readToolGroupConfig(filePath string) (*types.ToolGroup, error) {
	var input types.ToolGroup
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &input, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return &input, fmt.Errorf("failed to parse config file: %w", err)
	}
	if err := configresolver.ResolveEnvVars(&input); err != nil {
		return &input, fmt.Errorf("failed to resolve config file environment variables: %w", err)
	}
	return &input, nil
}
