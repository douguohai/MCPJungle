package cmd

import (
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entities like MCP servers, tools, etc",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "3",
	},
}

var (
	listToolsCmdServerName     string
	listToolsCmdCollectionName string
)

var (
	listPromptsCmdServerName   string
	listResourcesCmdServerName string
)

var listToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools",
	Long: "List tools available either from a specific MCP server, tool collection, or across " +
		"all MCP servers registered in mcpjungle.\n\n" +
		"NOTE: When using --collection flag, this command only displays tools that currently exist " +
		"in mcpjungle and are part of the collection.\n" +
		"So if, for example, the collection includes a tool that has been deleted, this command won't display it.\n" +
		"To get the full list of tools included in a collection, use the `get collection` command instead.",
	RunE: runListTools,
}

var listPromptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "List available prompts",
	Long:  "List prompt templates available either from a specific MCP server or across all MCP servers in mcpjungle.",
	RunE:  runListPrompts,
}

var listResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List available resources",
	Long:  "List resources available either from a specific MCP server or across all MCP servers in mcpjungle.",
	RunE:  runListResources,
}

var listServersCmd = &cobra.Command{
	Use:   "servers",
	Short: "List registered MCP servers",
	RunE:  runListServers,
}

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users (Enterprise mode)",
	Long:  "List users that are authorized to access MCPJungle.",
	RunE:  runListUsers,
}

var listCollectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "List tool collections",
	RunE:  runListCollections,
}

var listDeviceTokensCmd = &cobra.Command{
	Use:   "device-tokens",
	Short: "List your device tokens (Enterprise mode)",
	Long:  "List your device tokens. Admins see all tokens.",
	RunE:  runListDeviceTokens,
}

func init() {
	listToolsCmd.Flags().StringVar(
		&listToolsCmdServerName,
		"server",
		"",
		"Filter tools by server name",
	)
	listToolsCmd.Flags().StringVar(
		&listToolsCmdCollectionName,
		"collection",
		"",
		"Filter tools by tool collection name",
	)

	listPromptsCmd.Flags().StringVar(
		&listPromptsCmdServerName,
		"server",
		"",
		"Filter prompts by server name",
	)

	listResourcesCmd.Flags().StringVar(
		&listResourcesCmdServerName,
		"server",
		"",
		"Filter resources by server name",
	)

	listCmd.AddCommand(listToolsCmd)
	listCmd.AddCommand(listPromptsCmd)
	listCmd.AddCommand(listResourcesCmd)
	listCmd.AddCommand(listServersCmd)
	listCmd.AddCommand(listUsersCmd)
	listCmd.AddCommand(listCollectionsCmd)
	listCmd.AddCommand(listDeviceTokensCmd)

	rootCmd.AddCommand(listCmd)
}

func runListTools(cmd *cobra.Command, args []string) error {
	// If both server and collection flags are provided, reject the request.
	if listToolsCmdServerName != "" && listToolsCmdCollectionName != "" {
		return fmt.Errorf("using both --server and --collection flags together is currently not supported")
	}

	var tools []*types.Tool
	var err error
	var contextInfo string

	if listToolsCmdCollectionName != "" {
		// Get tools from specific collection
		collection, err := apiClient.GetToolCollection(listToolsCmdCollectionName)
		if err != nil {
			return fmt.Errorf("failed to get tool collection '%s': %w", listToolsCmdCollectionName, err)
		}

		effectiveTools, err := apiClient.GetToolCollectionEffectiveTools(listToolsCmdCollectionName)
		if err != nil {
			return fmt.Errorf("failed to resolve effective tools for collection '%s': %w", listToolsCmdCollectionName, err)
		}

		// Get all tools first, then filter by collection's effective tools.
		// This is necessary because a collection might contain tools that do not currently exist in mcpjungle.
		// for eg- the tool was deleted after collection creation or the collection includes a non-existent tool.
		// ListTools only returns tools that actually exist in mcpjungle, so we must cross-check.
		allTools, err := apiClient.ListTools("")
		if err != nil {
			return fmt.Errorf("failed to list all tools: %w", err)
		}

		// Create a map for efficient lookup
		effectiveToolsMap := make(map[string]bool)
		for _, toolName := range effectiveTools {
			effectiveToolsMap[toolName] = true
		}

		// Filter tools that are in the collection
		for _, tool := range allTools {
			if effectiveToolsMap[tool.Name] {
				tools = append(tools, tool)
			}
		}

		contextInfo = fmt.Sprintf("Tools in collection '%s'", listToolsCmdCollectionName)
		if collection.Description != "" {
			contextInfo += fmt.Sprintf(" (%s)", collection.Description)
		}
	} else {
		// no collection specified, list tools from specific server (if flag is set) or all servers
		tools, err = apiClient.ListTools(listToolsCmdServerName)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		if listToolsCmdServerName != "" {
			contextInfo = fmt.Sprintf("Tools from server '%s'", listToolsCmdServerName)
		}
	}

	if len(tools) == 0 {
		if listToolsCmdCollectionName != "" {
			cmd.Printf("There are no valid tools in collection '%s'\n", listToolsCmdCollectionName)
		} else if listToolsCmdServerName != "" {
			cmd.Printf("There are no tools from mcp server '%s'\n", listToolsCmdServerName)
		} else {
			cmd.Println("There are currently no tools in the registry")
		}
		return nil
	}

	// Display context information if filtering is applied
	if contextInfo != "" {
		cmd.Printf("%s:\n\n", contextInfo)
	}

	for i, t := range tools {
		ed := "ENABLED"
		if !t.Enabled {
			ed = "DISABLED"
		}
		cmd.Printf("%d. %s  [%s]\n", i+1, t.Name, ed)
		cmd.Println(t.Description)
		cmd.Println()
	}

	cmd.Println("Run 'usage <tool name>' to see a tool's usage or 'invoke <tool name>' to call one")

	return nil
}

func runListServers(cmd *cobra.Command, args []string) error {
	servers, err := apiClient.ListServers()
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("There are no MCP servers in the registry")
		return nil
	}
	for i, s := range servers {
		fmt.Printf("%d. %s\n", i+1, s.Name)

		if s.Description != "" {
			fmt.Println(s.Description)
		}

		fmt.Println("Transport: " + s.Transport)

		t, _ := types.ValidateTransport(s.Transport)
		if t == types.TransportStreamableHTTP || t == types.TransportSSE {
			fmt.Println("URL: " + s.URL)
		} else {
			if len(s.Args) > 0 {
				fmt.Println("Command: " + s.Command + " " + strings.Join(s.Args, " "))
			} else {
				fmt.Println("Command: " + s.Command)
			}

			if len(s.Env) > 0 {
				fmt.Printf("Environment variables: %s\n", s.Env)
			}
		}

		if i < len(servers)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runListUsers(cmd *cobra.Command, args []string) error {
	users, err := apiClient.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		cmd.Println("There are no users in the registry")
		return nil
	}
	for i, u := range users {
		if u.Role == string(types.UserRoleSystemAdmin) {
			cmd.Printf("%d. %s  [SYSTEM ADMIN]\n", i+1, u.Username)
		} else {
			cmd.Printf("%d. %s\n", i+1, u.Username)
		}

		if i < len(users)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListCollections(cmd *cobra.Command, args []string) error {
	collections, err := apiClient.ListToolCollections()
	if err != nil {
		return fmt.Errorf("failed to list tool collections: %w", err)
	}

	if len(collections) == 0 {
		cmd.Println("There are no tool collections in the registry")
		return nil
	}
	for i, coll := range collections {
		cmd.Printf("%d. %s\n", i+1, coll.Name)
		if coll.Description != "" {
			cmd.Println(coll.Description)
		}

		if i < len(collections)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListPrompts(cmd *cobra.Command, args []string) error {
	prompts, err := apiClient.ListPrompts(listPromptsCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		cmd.Println("No prompts found")
		return nil
	}
	for i, p := range prompts {
		ed := "ENABLED"
		if !p.Enabled {
			ed = "DISABLED"
		}
		cmd.Printf("%d. %s  [%s]\n", i+1, p.Name, ed)
		if p.Description != "" {
			cmd.Println(p.Description)
		}
		cmd.Println()
	}

	cmd.Println("Run 'get prompt <prompt name>' to retrieve a prompt template")

	return nil
}

func runListResources(cmd *cobra.Command, args []string) error {
	resources, err := apiClient.ListResources(listResourcesCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resources) == 0 {
		cmd.Println("No resources found")
		return nil
	}
	for i, r := range resources {
		cmd.Printf("%d. %s\n", i+1, r.Name)
		cmd.Printf("   URI: %s\n", r.URI)
		if r.Description != "" {
			cmd.Println("   Description: ", r.Description)
		}
		cmd.Println()
	}

	return nil
}

func runListDeviceTokens(cmd *cobra.Command, args []string) error {
	tokens, err := apiClient.ListDeviceTokens()
	if err != nil {
		return fmt.Errorf("failed to list device tokens: %w", err)
	}
	if len(tokens) == 0 {
		cmd.Println("You have no device tokens")
		return nil
	}
	for i, t := range tokens {
		status := t.Status
		if status == "" {
			status = "active"
		}
		cmd.Printf("%d. %s  [%s] (scope: %s)\n", i+1, t.Name, status, t.ScopeMode)
	}
	return nil
}
