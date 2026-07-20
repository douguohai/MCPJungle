package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	InitAdminUsernameEnvVar = "MCPJUNGLE_ADMIN_USERNAME"
	InitAdminPasswordEnvVar = "MCPJUNGLE_ADMIN_PASSWORD"
)

var initServerCmd = &cobra.Command{
	Use:   "init-server",
	Short: "Initialize the MCPJungle Server (for Enterprise Mode only)",
	Long: "If the MCPJungle Server was started in Enterprise Mode, use this command to initialize the server.\n" +
		"Initialization is required before you can use the server.\n\n" +
		"The admin username defaults to \"admin\" (override with " + InitAdminUsernameEnvVar + ").\n" +
		"The admin password is read from " + InitAdminPasswordEnvVar + " and is required.\n",
	RunE: runInitServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "6",
	},
}

func init() {
	rootCmd.AddCommand(initServerCmd)
}

func runInitServer(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing the MCPJungle Server in Enterprise Mode...")

	username := os.Getenv(InitAdminUsernameEnvVar)
	if username == "" {
		username = "admin"
	}
	password := os.Getenv(InitAdminPasswordEnvVar)
	if password == "" {
		return errors.New("admin password is required: set the " + InitAdminPasswordEnvVar + " environment variable")
	}

	resp, err := apiClient.InitServer(username, password)
	if err != nil {
		return fmt.Errorf("failed to initialize the server: %w", err)
	}

	if resp.Message != "" {
		fmt.Println(resp.Message)
	}
	fmt.Printf("Admin username: %s\n", resp.AdminUsername)
	fmt.Println()
	fmt.Println("Open the MCPJungle dashboard at the configured registry URL and log in with the administrator credentials.")
	fmt.Println()
	fmt.Println("All done!")
	return nil
}
