package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestConfigCommand(t *testing.T) {
	t.Run("show default config", func(t *testing.T) {
		// Clean up any existing config
		os.Remove(".gh-worktree.yml")
		os.Unsetenv("GH_WORKTREE_BASE_PATH")

		output := runCommand(t, "config")

		// Check essential output
		if !strings.Contains(output, "Current Configuration:") {
			t.Errorf("Output doesn't contain 'Current Configuration:'")
		}
		if !strings.Contains(output, "Worktree Base Path: .worktree") {
			t.Errorf("Output doesn't show default base path")
		}
		if !strings.Contains(output, "Configuration Sources") {
			t.Errorf("Output doesn't show configuration sources")
		}
		if !strings.Contains(output, "Worktree Paths:") {
			t.Errorf("Output doesn't show worktree paths")
		}
	})

	t.Run("show config with project file", func(t *testing.T) {
		// Create config file
		configContent := `worktree_base_path: .test-worktrees`
		if err := os.WriteFile(".gh-worktree.yml", []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(".gh-worktree.yml")

		output := runCommand(t, "config")

		// Check config is loaded
		if !strings.Contains(output, "Worktree Base Path: .test-worktrees") {
			t.Errorf("Output doesn't show custom base path from config file")
		}
		if !strings.Contains(output, "Project Config: .gh-worktree.yml") {
			t.Errorf("Output doesn't mention project config")
		}
		if !strings.Contains(output, "✓ Found") {
			t.Errorf("Output doesn't show project config as found")
		}
	})

	t.Run("show config with environment variable", func(t *testing.T) {
		// Set environment variable
		os.Setenv("GH_WORKTREE_BASE_PATH", ".env-worktrees")
		defer os.Unsetenv("GH_WORKTREE_BASE_PATH")

		output := runCommand(t, "config")

		// Check env var is used
		if !strings.Contains(output, "Worktree Base Path: .env-worktrees") {
			t.Errorf("Output doesn't show base path from environment variable")
		}
		if !strings.Contains(output, "✓ Set to: .env-worktrees") {
			t.Errorf("Output doesn't show environment variable value")
		}
	})

	t.Run("environment variable overrides config file", func(t *testing.T) {
		// Create config file
		configContent := `worktree_base_path: .file-worktrees`
		if err := os.WriteFile(".gh-worktree.yml", []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(".gh-worktree.yml")

		// Set environment variable
		os.Setenv("GH_WORKTREE_BASE_PATH", ".env-override")
		defer os.Unsetenv("GH_WORKTREE_BASE_PATH")

		output := runCommand(t, "config")

		// Check env var takes priority
		if !strings.Contains(output, "Worktree Base Path: .env-override") {
			t.Errorf("Environment variable should override config file")
		}
	})
}

func TestConfigCommandPaths(t *testing.T) {
	// Skip if not in a git repository
	if err := os.Getenv("CI"); err != "" {
		t.Skip("Skipping path tests in CI environment")
	}

	t.Run("show absolute paths", func(t *testing.T) {
		os.Remove(".gh-worktree.yml")
		os.Unsetenv("GH_WORKTREE_BASE_PATH")

		output := runCommand(t, "config")

		// Should show absolute paths with git root
		if !strings.Contains(output, "/.worktree/pr/") {
			t.Errorf("Output doesn't show absolute PR path")
		}
		if !strings.Contains(output, "/.worktree/work/") {
			t.Errorf("Output doesn't show absolute work path")
		}
	})
}
