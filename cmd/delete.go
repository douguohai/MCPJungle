package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{Use: "delete", Short: "Delete entities from mcpjungle", Annotations: map[string]string{"group": string(subCommandGroupAdvanced), "order": "5"}}
var deleteToolGroupCmd = &cobra.Command{Use: "group [name]", Args: cobra.ExactArgs(1), Short: "Delete a tool group", RunE: runDeleteToolGroup}

func init() { deleteCmd.AddCommand(deleteToolGroupCmd); rootCmd.AddCommand(deleteCmd) }

func runDeleteToolGroup(cmd *cobra.Command, args []string) error {
	if err := apiClient.DeleteToolGroup(args[0]); err != nil {
		return fmt.Errorf("failed to delete the tool group: %w", err)
	}
	cmd.Printf("Tool group '%s' deleted successfully!\n", args[0])
	return nil
}
