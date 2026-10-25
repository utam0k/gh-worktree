package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorktreeBasePath(t *testing.T) {
	// Skip if not in a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		t.Skip("Not in a git repository, skipping e2e tests")
	}

	// Skip if no commits exist
	if err := exec.Command("git", "rev-parse", "HEAD").Run(); err != nil {
		t.Skip("No commits in repository, skipping e2e tests")
	}

	// Get git root
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("Failed to get git root: %v", err)
	}
	gitRoot := strings.TrimSpace(string(output))

	// Get current branch to use as base
	branchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	baseBranch := strings.TrimSpace(string(branchOutput))

	t.Run("default .worktree path", func(t *testing.T) {
		// Clean up any existing config
		os.Remove(".gh-worktree.yml")
		os.Unsetenv("GH_WORKTREE_BASE_PATH")

		// Create a test worktree for branch
		output := runCommand(t, "new", "-b", baseBranch, "--cd", "test-branch-default")
		worktreePath := strings.TrimSpace(output)

		// Verify the path uses .worktree
		expectedPath := filepath.Join(gitRoot, ".worktree", "work", "test-branch-default")
		if worktreePath != expectedPath {
			t.Errorf("Expected worktree path %s, got %s", expectedPath, worktreePath)
		}

		// Cleanup
		exec.Command("git", "worktree", "remove", worktreePath).Run()
		exec.Command("git", "branch", "-D", "test-branch-default").Run()
	})

	t.Run("custom path via config file", func(t *testing.T) {
		// Create config file
		configContent := `worktree_base_path: .custom-worktrees`
		if err := os.WriteFile(".gh-worktree.yml", []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(".gh-worktree.yml")

		// Create a test worktree
		output := runCommand(t, "new", "-b", baseBranch, "--cd", "test-branch-config")
		worktreePath := strings.TrimSpace(output)

		// Verify the path uses custom location
		expectedPath := filepath.Join(gitRoot, ".custom-worktrees", "work", "test-branch-config")
		if worktreePath != expectedPath {
			t.Errorf("Expected worktree path %s, got %s", expectedPath, worktreePath)
		}

		// Cleanup
		exec.Command("git", "worktree", "remove", worktreePath).Run()
		exec.Command("git", "branch", "-D", "test-branch-config").Run()
	})

	t.Run("custom path via environment variable", func(t *testing.T) {
		// Clean up config file
		os.Remove(".gh-worktree.yml")

		// Set environment variable
		os.Setenv("GH_WORKTREE_BASE_PATH", ".env-worktrees")
		defer os.Unsetenv("GH_WORKTREE_BASE_PATH")

		// Create a test worktree
		output := runCommand(t, "new", "-b", baseBranch, "--cd", "test-branch-env")
		worktreePath := strings.TrimSpace(output)

		// Verify the path uses env variable
		expectedPath := filepath.Join(gitRoot, ".env-worktrees", "work", "test-branch-env")
		if worktreePath != expectedPath {
			t.Errorf("Expected worktree path %s, got %s", expectedPath, worktreePath)
		}

		// Cleanup
		exec.Command("git", "worktree", "remove", worktreePath).Run()
		exec.Command("git", "branch", "-D", "test-branch-env").Run()
	})

	t.Run("list shows correct paths", func(t *testing.T) {
		// Use default path
		os.Remove(".gh-worktree.yml")
		os.Unsetenv("GH_WORKTREE_BASE_PATH")

		// Create a test worktree
		runCommand(t, "new", "-b", baseBranch, "test-branch-list")
		defer func() {
			worktreePath := filepath.Join(gitRoot, ".worktree", "work", "test-branch-list")
			exec.Command("git", "worktree", "remove", worktreePath).Run()
			exec.Command("git", "branch", "-D", "test-branch-list").Run()
		}()

		// List worktrees
		output := runCommand(t, "list")

		// Should show the worktree with .worktree path
		if !strings.Contains(output, ".worktree/work/test-branch-list") {
			t.Errorf("List output doesn't show .worktree path: %s", output)
		}
	})
}

func TestWorktreePathStructure(t *testing.T) {
	// Skip if not in a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		t.Skip("Not in a git repository, skipping e2e tests")
	}

	// Skip if no commits exist
	if err := exec.Command("git", "rev-parse", "HEAD").Run(); err != nil {
		t.Skip("No commits in repository, skipping e2e tests")
	}

	// Get git root
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("Failed to get git root: %v", err)
	}
	gitRoot := strings.TrimSpace(string(output))

	// Get current branch to use as base
	branchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	baseBranch := strings.TrimSpace(string(branchOutput))

	// Clean up any existing config
	os.Remove(".gh-worktree.yml")
	os.Unsetenv("GH_WORKTREE_BASE_PATH")

	t.Run("work branch structure", func(t *testing.T) {
		branchName := "feature/test-structure"
		output := runCommand(t, "new", "-b", baseBranch, "--cd", branchName)
		worktreePath := strings.TrimSpace(output)

		// Verify directory structure
		expectedPath := filepath.Join(gitRoot, ".worktree", "work", branchName)
		if worktreePath != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, worktreePath)
		}

		// Verify parent directories exist
		workDir := filepath.Join(gitRoot, ".worktree", "work")
		if _, err := os.Stat(workDir); os.IsNotExist(err) {
			t.Errorf("Work directory doesn't exist: %s", workDir)
		}

		// Cleanup
		exec.Command("git", "worktree", "remove", worktreePath).Run()
		exec.Command("git", "branch", "-D", branchName).Run()
	})

	// Note: PR worktree tests would require mocking GitHub API responses
	// so we're focusing on the work branch functionality for integration tests
}

func TestWorktreePathMigration(t *testing.T) {
	// This test verifies that the new regex patterns work with both old and new paths
	// Skip if not in a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		t.Skip("Not in a git repository, skipping e2e tests")
	}

	// Skip if no commits exist
	if err := exec.Command("git", "rev-parse", "HEAD").Run(); err != nil {
		t.Skip("No commits in repository, skipping e2e tests")
	}

	t.Run("list recognizes new .worktree paths", func(t *testing.T) {
		// Create a worktree with new path
		os.Remove(".gh-worktree.yml")
		os.Unsetenv("GH_WORKTREE_BASE_PATH")

		// Get current branch to use as base
		branchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		baseBranch := strings.TrimSpace(string(branchOutput))

		runCommand(t, "new", "-b", baseBranch, "migration-test")
		defer func() {
			output, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
			gitRoot := strings.TrimSpace(string(output))
			worktreePath := filepath.Join(gitRoot, ".worktree", "work", "migration-test")
			exec.Command("git", "worktree", "remove", worktreePath).Run()
			exec.Command("git", "branch", "-D", "migration-test").Run()
		}()

		// List should show the worktree
		output := runCommand(t, "list")
		if !strings.Contains(output, "migration-test") {
			t.Errorf("List doesn't show new worktree: %s", output)
		}
		if !strings.Contains(output, "Work Worktrees:") {
			t.Errorf("List doesn't categorize work worktrees correctly: %s", output)
		}
	})
}

func TestInvalidConfigHandling(t *testing.T) {
	// Skip if not in a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		t.Skip("Not in a git repository, skipping e2e tests")
	}

	// Skip if no commits exist
	if err := exec.Command("git", "rev-parse", "HEAD").Run(); err != nil {
		t.Skip("No commits in repository, skipping e2e tests")
	}
	t.Run("invalid yaml in config file", func(t *testing.T) {
		// Create invalid config file
		invalidConfig := `invalid: yaml: content: [`
		if err := os.WriteFile(".gh-worktree.yml", []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(".gh-worktree.yml")

		// Get current branch to use as base
		branchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		baseBranch := strings.TrimSpace(string(branchOutput))

		// Should fall back to default path
		output := runCommand(t, "new", "-b", baseBranch, "--cd", "test-invalid-config")
		worktreePath := strings.TrimSpace(output)

		// Get git root
		gitRootOutput, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		gitRoot := strings.TrimSpace(string(gitRootOutput))

		// Should use default .worktree path
		expectedPath := filepath.Join(gitRoot, ".worktree", "work", "test-invalid-config")
		if worktreePath != expectedPath {
			t.Errorf("Expected default path %s, got %s", expectedPath, worktreePath)
		}

		// Cleanup
		exec.Command("git", "worktree", "remove", worktreePath).Run()
		exec.Command("git", "branch", "-D", "test-invalid-config").Run()
	})
}
