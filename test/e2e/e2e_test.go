package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const binaryName = "gh-worktree"

func TestMain(m *testing.M) {
	// Build the binary before running tests
	cmd := exec.Command("go", "build", "-o", binaryName, "../../cmd/gh-worktree")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(binaryName)
	os.Exit(code)
}

func TestCreateListRemoveWorktree(t *testing.T) {
	// Skip if not in a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		t.Skip("Not in a git repository, skipping e2e tests")
	}

	// Skip if gh is not installed
	if err := exec.Command("gh", "--version").Run(); err != nil {
		t.Skip("gh CLI not installed, skipping e2e tests")
	}

	t.Run("help command", func(t *testing.T) {
		output := runCommand(t, "help")
		if !strings.Contains(output, "gh-worktree") {
			t.Errorf("Help output doesn't contain 'gh-worktree'")
		}
		if !strings.Contains(output, "pr") {
			t.Errorf("Help output doesn't contain 'pr' command")
		}
	})

	t.Run("list with no worktrees", func(t *testing.T) {
		output := runCommand(t, "list")
		if !strings.Contains(output, "No worktrees found") && !strings.Contains(output, "PR Worktrees:") {
			t.Errorf("Unexpected list output: %s", output)
		}
	})

	t.Run("pr with no PR number", func(t *testing.T) {
		_, err := runCommandWithError("pr")
		if err == nil {
			t.Error("Expected error when creating PR worktree without PR number")
		}
	})

	t.Run("pr with invalid PR number", func(t *testing.T) {
		_, err := runCommandWithError("pr", "invalid")
		if err == nil {
			t.Error("Expected error when creating PR worktree with invalid PR number")
		}
	})
}

func TestCommandValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no command",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "unknown command",
			args:      []string{"unknown"},
			wantError: true,
		},
		{
			name:      "remove without PR number",
			args:      []string{"remove"},
			wantError: true,
		},
		{
			name:      "remove with invalid PR number",
			args:      []string{"remove", "abc"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runCommandWithError(tt.args...)
			if (err != nil) != tt.wantError {
				t.Errorf("Expected error=%v, got %v", tt.wantError, err)
			}
		})
	}
}

func runCommand(t *testing.T, args ...string) string {
	t.Helper()
	output, err := runCommandWithError(args...)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}
	return output
}

func runCommandWithError(args ...string) (string, error) {
	cmd := exec.Command("./"+binaryName, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
