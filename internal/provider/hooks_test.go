// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestExecutePostCommands tests the post-creation hooks functionality
func TestExecutePostCommands(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty commands list", func(t *testing.T) {
		commands := types.ListNull(types.StringType)
		err := executePostCommands(ctx, commands, "test")
		if err != nil {
			t.Errorf("Expected no error for empty commands, got: %v", err)
		}
	})

	t.Run("Simple command execution", func(t *testing.T) {
		// Create a list of commands
		commandValues := []attr.Value{
			types.StringValue("echo 'test command'"),
		}
		commands, _ := types.ListValue(types.StringType, commandValues)

		err := executePostCommands(ctx, commands, "test")
		if err != nil {
			t.Errorf("Simple command should succeed, got error: %v", err)
		}
	})

	t.Run("Multiple command execution", func(t *testing.T) {
		// Create a list of commands
		commandValues := []attr.Value{
			types.StringValue("echo 'first command'"),
			types.StringValue("echo 'second command'"),
		}
		commands, _ := types.ListValue(types.StringType, commandValues)

		err := executePostCommands(ctx, commands, "test")
		if err != nil {
			t.Errorf("Multiple commands should succeed, got error: %v", err)
		}
	})

	t.Run("Command failure", func(t *testing.T) {
		// Create a command that should fail
		commandValues := []attr.Value{
			types.StringValue("false"), // Command that always returns exit code 1
		}
		commands, _ := types.ListValue(types.StringType, commandValues)

		err := executePostCommands(ctx, commands, "test")
		if err == nil {
			t.Error("Expected error for failing command, but got none")
		}
	})

	t.Run("File operations with commands", func(t *testing.T) {
		// Test file operations combined with commands
		commandValues := []attr.Value{
			types.StringValue("touch /tmp/test-hook-file"),
			types.StringValue("echo 'hook executed' > /tmp/test-hook-file"),
		}
		commands, _ := types.ListValue(types.StringType, commandValues)

		err := executePostCommands(ctx, commands, "test")
		if err != nil {
			t.Errorf("File operation commands should succeed, got error: %v", err)
		}

		// Cleanup
		cleanupCommands := []attr.Value{
			types.StringValue("rm -f /tmp/test-hook-file"),
		}
		cleanupList, _ := types.ListValue(types.StringType, cleanupCommands)
		executePostCommands(ctx, cleanupList, "cleanup")
	})
}

// TestExecuteShellCommand tests individual shell command execution
func TestExecuteShellCommand(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "Echo command",
			command:     "echo 'test'",
			expectError: false,
		},
		{
			name:        "True command",
			command:     "true",
			expectError: false,
		},
		{
			name:        "False command",
			command:     "false",
			expectError: true,
		},
		{
			name:        "Empty command",
			command:     "",
			expectError: true,
		},
		{
			name:        "Nonexistent command",
			command:     "this-command-does-not-exist-12345",
			expectError: true,
		},
		{
			name:        "Complex shell command",
			command:     "echo 'hello' | grep 'hello'",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := executeShellCommand(ctx, tc.command)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for command '%s', but got none", tc.command)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for command '%s', but got: %v", tc.command, err)
			}
		})
	}
}

// TestPostCreationHooksIntegration tests hooks in the context of file operations
func TestPostCreationHooksIntegration(t *testing.T) {
	t.Run("FileResource with post-create hooks", func(t *testing.T) {
		// Test the integration of post-creation hooks in file operations
		model := &EnhancedFileResourceModel{
			FileResourceModel: FileResourceModel{
				ID:         types.StringValue("test-file"),
				Repository: types.StringValue("test-repo"),
				Name:       types.StringValue("test-file"),
				SourcePath: types.StringValue("test.txt"),
				TargetPath: types.StringValue("/tmp/test-file"),
			},
			PostCreateCommands: func() types.List {
				commands := []attr.Value{
					types.StringValue("echo 'File created' > /tmp/test-hook-log"),
					types.StringValue("touch /tmp/test-file && chmod 644 /tmp/test-file"),
				}
				list, _ := types.ListValue(types.StringType, commands)
				return list
			}(),
			PostUpdateCommands: func() types.List {
				commands := []attr.Value{
					types.StringValue("echo 'File updated' >> /tmp/test-hook-log"),
				}
				list, _ := types.ListValue(types.StringType, commands)
				return list
			}(),
			PreDestroyCommands: func() types.List {
				commands := []attr.Value{
					types.StringValue("echo 'File will be destroyed' >> /tmp/test-hook-log"),
					types.StringValue("test -f /tmp/test-file && cp /tmp/test-file /tmp/test-file.backup || echo 'File not found for backup'"),
				}
				list, _ := types.ListValue(types.StringType, commands)
				return list
			}(),
		}

		ctx := context.Background()

		// Test post-create hooks
		err := executePostCommands(ctx, model.PostCreateCommands, "post-create")
		if err != nil {
			t.Errorf("Post-create hooks should succeed: %v", err)
		}

		// Test post-update hooks
		err = executePostCommands(ctx, model.PostUpdateCommands, "post-update")
		if err != nil {
			t.Errorf("Post-update hooks should succeed: %v", err)
		}

		// Test pre-destroy hooks
		err = executePostCommands(ctx, model.PreDestroyCommands, "pre-destroy")
		if err != nil {
			t.Errorf("Pre-destroy hooks should succeed: %v", err)
		}

		// Cleanup
		cleanupCommands := []attr.Value{
			types.StringValue("rm -f /tmp/test-hook-log /tmp/test-file /tmp/test-file.backup"),
		}
		cleanupList, _ := types.ListValue(types.StringType, cleanupCommands)
		executePostCommands(ctx, cleanupList, "cleanup")
	})
}

// TestHooksValidation tests validation of hook commands
func TestHooksValidation(t *testing.T) {
	t.Run("Safe command validation", func(t *testing.T) {
		// Test that basic commands work
		safeCommands := []string{
			"echo 'hello'",
			"touch /tmp/test",
			"chmod 644 /tmp/test",
			"ls -la",
			"pwd",
		}

		ctx := context.Background()
		for _, cmd := range safeCommands {
			err := executeShellCommand(ctx, cmd)
			if err != nil {
				t.Errorf("Safe command '%s' should not fail: %v", cmd, err)
			}
		}
	})

	t.Run("Command with pipes and redirects", func(t *testing.T) {
		// Test complex shell operations
		complexCommands := []string{
			"echo 'test' | wc -l",
			"echo 'hello world' | grep 'world'",
			"ls /tmp | head -5",
		}

		ctx := context.Background()
		for _, cmd := range complexCommands {
			err := executeShellCommand(ctx, cmd)
			if err != nil {
				t.Errorf("Complex command '%s' should not fail: %v", cmd, err)
			}
		}
	})
}
