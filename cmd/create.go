package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mcpjungle/mcpjungle/internal/configresolver"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create entities in mcpjungle",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "4",
	},
}

var createUserCmd = &cobra.Command{
	Use:  "user [username]",
	Args: cobra.ExactArgs(1),
	Short: "Create a new user (Enterprise mode)",
	Long: "Create a new standard user in MCPJungle.\n" +
		"A user can make authenticated requests to the MCPJungle API server and perform limited actions like:\n" +
		"- List and view MCP servers & tools\n" +
		"- Check tool usage and invoke them\n\n" +
		"This command is only available in Enterprise mode.",
	RunE: runCreateUser,
}

var createToolGroupCmd = &cobra.Command{
	Use:   "group --conf <file>",
	Short: "Create a Group of MCP Tools",
	Long: "Create a new Group of MCP Tools by supplying a configuration file.\n" +
		"A group lets you expose only a handful of Tools that you choose.\n" +
		"This limits the number of tools your MCP client sees, increasing calling accuracy of the LLM.\n\n" +
		"You can include tools by:\n" +
		"  - Specifying individual tools with 'included_tools'\n" +
		"  - Including all tools from servers with 'included_servers'\n" +
		"  - Excluding specific tools with 'excluded_tools'\n\n" +
		"Once you create a tool group, it is accessible as a streamable http MCP server at the following endpoint:\n" +
		"    /v0/groups/{group_name}/mcp\n",
	RunE: runCreateToolGroup,
}

var createDeviceTokenName string
var createDeviceTokenScope string
var createDeviceTokenServers string

var createDeviceTokenCmd = &cobra.Command{
	Use:   "device-token --name <name>",
	Short: "Create a new device token (Enterprise mode)",
	Long: "Create a new device token for authenticating MCP proxy requests.\n" +
		"The raw token is shown once at creation — copy it before closing.\n" +
		"Use 'inherit_all' scope to inherit your permission groups, or 'restricted' to limit to specific servers.",
	RunE: runCreateDeviceToken,
}

var (
	createToolGroupConfigFilePath string
)

func init() {
	createToolGroupCmd.Flags().StringVarP(
		&createToolGroupConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the Group",
	)
	_ = createToolGroupCmd.MarkFlagRequired("conf")

	createDeviceTokenCmd.Flags().StringVar(&createDeviceTokenName, "name", "", "Name for the device token (required)")
	_ = createDeviceTokenCmd.MarkFlagRequired("name")
	createDeviceTokenCmd.Flags().StringVar(&createDeviceTokenScope, "scope", "inherit_all", "Scope mode: inherit_all or restricted")
	createDeviceTokenCmd.Flags().StringVar(&createDeviceTokenServers, "servers", "", "Comma-separated MCP server names (required for restricted scope)")

	createCmd.AddCommand(createUserCmd)
	createCmd.AddCommand(createDeviceTokenCmd)
	createCmd.AddCommand(createToolGroupCmd)

	rootCmd.AddCommand(createCmd)
}

func runCreateUser(cmd *cobra.Command, args []string) error {
	user := &types.CreateOrUpdateUserRequest{
		Username: args[0],
	}
	resp, err := apiClient.CreateUser(user)
	if err != nil {
		return err
	}

	cmd.Printf("User '%s' created successfully (role: %s)\n", user.Username, resp.Role)

	return nil
}

func runCreateDeviceToken(cmd *cobra.Command, args []string) error {
	scopeMode := createDeviceTokenScope
	if scopeMode != "inherit_all" && scopeMode != "restricted" {
		return fmt.Errorf("invalid --scope: must be inherit_all or restricted")
	}

	var restrictedNames []string
	if scopeMode == "restricted" {
		if createDeviceTokenServers == "" {
			return fmt.Errorf("--servers is required when --scope is restricted")
		}
		for _, name := range strings.Split(createDeviceTokenServers, ",") {
			restrictedNames = append(restrictedNames, strings.TrimSpace(name))
		}
	}

	raw, token, err := apiClient.CreateDeviceToken(createDeviceTokenName, scopeMode, restrictedNames)
	if err != nil {
		return fmt.Errorf("failed to create device token: %w", err)
	}

	cmd.Println("Device token created successfully!")
	cmd.Printf("  Name: %s\n", token.Name)
	cmd.Printf("  Scope: %s\n", token.ScopeMode)
	if scopeMode == "restricted" {
		cmd.Printf("  Restricted servers: %s\n", createDeviceTokenServers)
	}
	cmd.Println()
	cmd.Println("⚠️  The token below is shown only once. Copy it now!")
	cmd.Printf("  %s\n", raw)
	return nil
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

	cmd.Printf("Tool Group %s created successfully\n", group.Name)
	cmd.Print("It is now accessible at the following streamable http endpoint:\n\n")
	cmd.Println("    " + resp.StreamableHTTPEndpoint + "\n")

	cmd.Print("Tools using the SSE (server-sent events) transport are accessible at:\n\n")
	cmd.Println("    " + resp.SSEEndpoint)
	cmd.Println("    " + resp.SSEMessageEndpoint + "\n")

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
