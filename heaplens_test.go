// ABOUTME: Tests for the main heaplens package, verifying project structure and imports
// ABOUTME: These tests ensure the basic package setup is working correctly

package heaplens_test

import (
	"testing"

	"github.com/prateek/heaplens"
)

func TestProjectStructure(t *testing.T) {
	// Verify the version constant exists and is non-empty
	if heaplens.Version == "" {
		t.Error("Version constant should not be empty")
	}

	// Verify version format (should be semantic versioning)
	expectedPrefix := "0."
	if len(heaplens.Version) < len(expectedPrefix) || heaplens.Version[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Version should start with %q, got %q", expectedPrefix, heaplens.Version)
	}
}

func TestPackageImport(t *testing.T) {
	// This test verifies that the package can be imported and used
	// The actual test is that this file compiles successfully
	t.Log("Package import successful")
}