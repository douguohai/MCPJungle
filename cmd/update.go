package cmd

import (
	"fmt"

	"github.com/mcpjungle/mcpjungle/pkg/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{Use: "update", Short: "Update entities", Annotations: map[string]string{"group": string(subCommandGroupAdvanced), "order": "8"}}
var updateToolGroupCmd = &cobra.Command{Use: "group", Short: "Update a tool group", RunE: runUpdateGroup}
var updateToolGroupConfigFilePath string

func init() {
	updateToolGroupCmd.Flags().StringVarP(&updateToolGroupConfigFilePath, "conf", "c", "", "Path to the Group configuration file")
	_ = updateToolGroupCmd.MarkFlagRequired("conf")
	updateCmd.AddCommand(updateToolGroupCmd)
	rootCmd.AddCommand(updateCmd)
}

func runUpdateGroup(cmd *cobra.Command, args []string) error {
	updated, err := readToolGroupConfig(updateToolGroupConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", updateToolGroupConfigFilePath, err)
	}
	resp, err := apiClient.UpdateToolGroup(updated)
	if err != nil {
		return fmt.Errorf("failed to update tool group %s: %w", updated.Name, err)
	}
	added, removed := util.DiffTools(resp.Old.IncludedTools, resp.New.IncludedTools)
	cmd.Printf("Tool Group %s updated successfully (tools added: %d, removed: %d)\n", resp.Name, len(added), len(removed))
	return nil
}
