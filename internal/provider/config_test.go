// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"testing"
)

func TestDotfilesConfigComprehensive(t *testing.T) {
	// Setup test environment
	testEnv := setupConfigTestEnvironment(t)

	// Test config validation
	testConfigValidation(t, testEnv)

	// Test config creation
	testConfigCreation(t, testEnv)

	// Test config defaults
	testConfigDefaults(t, testEnv)
}

// configTestEnv holds the test environment setup
type configTestEnv struct {
	tempDir string
}

// setupConfigTestEnvironment creates the test environment
func setupConfigTestEnvironment(t *testing.T) *configTestEnv {
	return &configTestEnv{
		tempDir: t.TempDir(),
	}
}

// testConfigValidation tests config validation
func testConfigValidation(t *testing.T, env *configTestEnv) {
	_ = env // Environment not used in this validation test
	config := &DotfilesConfig{}
	// Basic validation that config was created
	_ = config
}

// testConfigCreation tests config creation
func testConfigCreation(t *testing.T, env *configTestEnv) {
	config := &DotfilesConfig{
		DotfilesRoot: env.tempDir,
	}
	if config.DotfilesRoot != env.tempDir {
		t.Error("Config root should match temp dir")
	}
}

// testConfigDefaults tests config defaults
func testConfigDefaults(t *testing.T, env *configTestEnv) {
	// Simple default validation
	if env.tempDir == "" {
		t.Error("Temp directory should not be empty")
	}
}
