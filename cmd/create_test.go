package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "create", createCmd.Use)
	testhelpers.AssertEqual(t, "Create entities in mcpjungle", createCmd.Short)

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupAdvanced)},
		{Key: "order", Expected: "4"},
	}
	testhelpers.TestCommandAnnotations(t, createCmd.Annotations, annotationTests)

	// Test subcommands count
	subcommands := createCmd.Commands()
	testhelpers.AssertEqual(t, 3, len(subcommands))
}

func TestCreateUserSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "user [username]", createUserCmd.Use)
	testhelpers.AssertEqual(t, "Create a new user (Enterprise mode)", createUserCmd.Short)
	testhelpers.AssertNotNil(t, createUserCmd.Long)
	testhelpers.AssertTrue(t, len(createUserCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createUserCmd.RunE)
	testhelpers.AssertNotNil(t, createUserCmd.Args)
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "group --conf <file>", createToolGroupCmd.Use)
	testhelpers.AssertEqual(t, "Create a Group of MCP Tools", createToolGroupCmd.Short)
	testhelpers.AssertNotNil(t, createToolGroupCmd.Long)
	testhelpers.AssertTrue(t, len(createToolGroupCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)

	// Test command flags
	confFlag := createToolGroupCmd.Flags().Lookup("conf")
	testhelpers.AssertNotNil(t, confFlag)
	testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")
}

func TestCreateCommandVariables(t *testing.T) {
	// Test that command variables are properly initialized to empty values
	testhelpers.AssertEqual(t, "", createToolGroupConfigFilePath)
}

// Integration tests for create commands
func TestCreateCommandIntegration(t *testing.T) {
	// Verify that createCmd is properly added to rootCmd
	testhelpers.AssertNotNil(t, createCmd)

	// Test all create subcommands are properly configured
	subcommands := createCmd.Commands()
	expectedSubcommands := []string{"user", "device-token", "group"}

	testhelpers.AssertEqual(t, len(expectedSubcommands), len(subcommands))

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		testhelpers.AssertTrue(t, found, "Expected subcommand '"+expected+"' not found")
	}
}

// Test argument validation
func TestCreateCommandArgumentValidation(t *testing.T) {
	// Test that commands properly validate arguments
	testhelpers.AssertNotNil(t, createUserCmd.Args)

	// Test various invalid input scenarios
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"empty args", []string{}, true},
		{"too many args", []string{"arg1", "arg2", "arg3"}, true},
		{"valid single arg", []string{"valid-arg"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test user command args validation
			if createUserCmd.Args != nil {
				err := createUserCmd.Args(createUserCmd, tc.args)
				if tc.expectError {
					testhelpers.AssertError(t, err)
				} else {
					testhelpers.AssertNoError(t, err)
				}
			}
		})
	}
}
