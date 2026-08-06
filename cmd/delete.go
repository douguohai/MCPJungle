package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete entities from mcpjungle",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "5",
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "user [username]",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a user (Enterprise mode)",
	Long:  "Delete a user from mcpjungle.\nThis instantly revokes all access of this user.",
	RunE:  runDeleteUser,
}

var deleteToolGroupCmd = &cobra.Command{
	Use:   "group [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a tool group",
	Long: "Delete a tool group from mcpjungle.\n" +
		"Once you delete a group, its endpoint is no longer available.\n" +
		"So make sure no MCP clients are relying on the endpoint before you delete a group.\n" +
		"NOTE: This command only deletes the group itself, not the tools included in it.\n" +
		"Tools are only deleted when you deregister a MCP server from mcpjungle.",
	RunE: runDeleteToolGroup,
}

var deleteDeviceTokenCmd = &cobra.Command{
	Use:   "device-token [id]",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a device token",
	Long:  "Permanently delete a device token by its ID. This cannot be undone.",
	RunE:  runDeleteDeviceToken,
}

func init() {
	deleteCmd.AddCommand(deleteUserCmd)
	deleteCmd.AddCommand(deleteDeviceTokenCmd)
	deleteCmd.AddCommand(deleteToolGroupCmd)

	rootCmd.AddCommand(deleteCmd)
}

func runDeleteUser(cmd *cobra.Command, args []string) error {
	username := args[0]
	if err := apiClient.DeleteUser(username); err != nil {
		return fmt.Errorf("failed to delete the user: %w", err)
	}
	cmd.Printf("User '%s' deleted successfully (if they existed)\n", username)
	return nil
}

func runDeleteToolGroup(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := apiClient.DeleteToolGroup(name); err != nil {
		return fmt.Errorf("failed to delete the tool group: %w", err)
	}
	cmd.Printf("Tool group '%s' deleted successfully!\n", name)
	return nil
}

func runDeleteDeviceToken(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token id: %s", args[0])
	}
	if err := apiClient.DeleteDeviceToken(uint(id)); err != nil {
		return fmt.Errorf("failed to delete device token: %w", err)
	}
	cmd.Println("Device token deleted successfully")
	return nil
}
