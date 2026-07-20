package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mcpjungle/mcpjungle/cmd/config"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var (
	loginUsername string
	loginPassword string
)

var loginCmd = &cobra.Command{
	Use:   "login [token]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Log in to MCPJungle (Enterprise mode)",
	Long: "Log in to your MCPJungle account.\n\n" +
		"Password login (obtains a short-lived session token):\n" +
		"  mcpjungle login --username admin --password $MCPJUNGLE_PASSWORD\n\n" +
		"Use an existing PAT or token directly:\n" +
		"  mcpjungle login <token>\n\n" +
		"The password may also be read from the MCPJUNGLE_PASSWORD environment variable.",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "7",
	},
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&loginUsername, "username", "", "username for password login")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "password for password login (or set MCPJUNGLE_PASSWORD)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	var token string
	if len(args) == 1 {
		// A token (PAT or session JWT) was provided directly.
		token = args[0]
	} else {
		// Username + password login -> session JWT.
		username := strings.TrimSpace(loginUsername)
		if username == "" {
			return fmt.Errorf("--username is required for password login (or pass a token as an argument)")
		}
		password := loginPassword
		if password == "" {
			password = os.Getenv("MCPJUNGLE_PASSWORD")
		}
		if password == "" {
			return fmt.Errorf("password is required: pass --password or set MCPJUNGLE_PASSWORD")
		}
		t, err := apiClient.Login(username, password)
		if err != nil {
			return fmt.Errorf("failed to log in: %w", err)
		}
		token = t
		cmd.Println("Logged in as", username)
	}

	// Verify the token is accepted by the server.
	user, err := apiClient.Whoami(token)
	if err != nil {
		return fmt.Errorf("login succeeded but failed to verify credentials: %w", err)
	}
	if user == nil {
		return fmt.Errorf("invalid credentials")
	}
	cmd.Println("Authenticated as", user.Username)
	if user.Role == string(types.UserRoleAdmin) {
		cmd.Println("You are an administrator of MCPJungle")
	}

	cfg := &config.ClientConfig{
		RegistryURL: apiClient.BaseURL(),
		AccessToken: token,
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save client configuration: %w", err)
	}

	cfgPath, err := config.AbsPath()
	if err != nil {
		return fmt.Errorf("failed to get client configuration path: %w", err)
	}
	fmt.Println("Your credentials have been saved to", cfgPath)

	return nil
}
