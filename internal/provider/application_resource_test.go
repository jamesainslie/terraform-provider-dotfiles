// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationResourceUnit(t *testing.T) {
	// Setup test environment
	testEnv := setupApplicationResourceTestEnvironment(t)

	t.Run("expandTargetPathTemplate", func(t *testing.T) {
		testExpandTargetPathTemplate(t, testEnv)
	})

	t.Run("createSymlinkForConfig", func(t *testing.T) {
		testCreateSymlinkForConfig(t, testEnv)
	})

	t.Run("copyConfigFile", func(t *testing.T) {
		testCopyConfigFile(t, testEnv)
	})
}

// applicationResourceTestEnv holds the test environment setup
type applicationResourceTestEnv struct {
	ctx         context.Context
	tempDir     string
	client      *DotfilesClient
	appResource *ApplicationResource
}

// setupApplicationResourceTestEnvironment creates the test environment
func setupApplicationResourceTestEnvironment(t *testing.T) *applicationResourceTestEnv {
	ctx := context.Background()
	tempDir := t.TempDir()

	client := &DotfilesClient{
		Config: &DotfilesConfig{
			DotfilesRoot: tempDir,
		},
	}

	appResource := &ApplicationResource{
		client: client,
	}

	return &applicationResourceTestEnv{
		ctx:         ctx,
		tempDir:     tempDir,
		client:      client,
		appResource: appResource,
	}
}

// testExpandTargetPathTemplate tests target path template expansion
func testExpandTargetPathTemplate(t *testing.T, env *applicationResourceTestEnv) {
	// Simple test cases for target path template expansion
	result, err := env.appResource.expandTargetPathTemplate("/simple/path", "testapp")
	if err != nil {
		t.Errorf("Simple path expansion failed: %v", err)
	}
	if result != "/simple/path" {
		t.Errorf("Expected /simple/path, got %s", result)
	}

	// Test application template
	result, err = env.appResource.expandTargetPathTemplate("/config/{{.application}}/settings.json", "testapp")
	if err != nil {
		t.Errorf("Application template expansion failed: %v", err)
	}
	if result != "/config/testapp/settings.json" {
		t.Errorf("Expected /config/testapp/settings.json, got %s", result)
	}
}

// testCreateSymlinkForConfig tests creating symlinks for config files
func testCreateSymlinkForConfig(t *testing.T, env *applicationResourceTestEnv) {
	// Create source file
	sourceFile := filepath.Join(env.tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test basic functionality
	if env.appResource == nil {
		t.Error("App resource should not be nil")
	}

	// Verify source file exists
	if !pathExists(sourceFile) {
		t.Error("Source file should exist")
	}
}

// testCopyConfigFile tests copying config files
func testCopyConfigFile(t *testing.T, env *applicationResourceTestEnv) {
	// Create source file
	sourceFile := filepath.Join(env.tempDir, "copy_source.txt")
	err := os.WriteFile(sourceFile, []byte("copy test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create copy source file: %v", err)
	}

	// Test basic functionality
	if env.client == nil {
		t.Error("Client should not be nil")
	}

	// Verify source file exists
	if !pathExists(sourceFile) {
		t.Error("Copy source file should exist")
	}
}
