package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWorktreePath(t *testing.T) {
	tests := []struct {
		name     string
		prNumber string
		setup    func()
		cleanup  func()
		contains string
	}{
		{
			name:     "default path",
			prNumber: "123",
			setup:    func() {},
			cleanup:  func() {},
			contains: ".worktree/pr/pr-123",
		},
		{
			name:     "with environment variable",
			prNumber: "456",
			setup: func() {
				os.Setenv("GH_WORKTREE_BASE_PATH", ".custom-wt")
			},
			cleanup: func() {
				os.Unsetenv("GH_WORKTREE_BASE_PATH")
			},
			contains: ".custom-wt/pr/pr-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			manager := NewManager()
			path := manager.GetWorktreePath(tt.prNumber)
			if !strings.Contains(path, tt.contains) {
				t.Errorf("GetWorktreePath() = %v, want to contain %v", path, tt.contains)
			}
		})
	}
}

func TestGetWorktreePathWithRemote(t *testing.T) {
	tests := []struct {
		name       string
		prNumber   string
		remoteName string
		setup      func()
		cleanup    func()
		contains   string
	}{
		{
			name:       "with origin remote",
			prNumber:   "123",
			remoteName: "origin",
			setup:      func() {},
			cleanup:    func() {},
			contains:   ".worktree/pr/origin/pr-123",
		},
		{
			name:       "with upstream remote",
			prNumber:   "456",
			remoteName: "upstream",
			setup:      func() {},
			cleanup:    func() {},
			contains:   ".worktree/pr/upstream/pr-456",
		},
		{
			name:       "empty remote defaults to origin",
			prNumber:   "789",
			remoteName: "",
			setup:      func() {},
			cleanup:    func() {},
			contains:   ".worktree/pr/origin/pr-789",
		},
		{
			name:       "custom base path",
			prNumber:   "100",
			remoteName: "fork",
			setup: func() {
				os.Setenv("GH_WORKTREE_BASE_PATH", ".my-worktrees")
			},
			cleanup: func() {
				os.Unsetenv("GH_WORKTREE_BASE_PATH")
			},
			contains: ".my-worktrees/pr/fork/pr-100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			manager := NewManager()
			path := manager.GetWorktreePathWithRemote(tt.prNumber, tt.remoteName)
			if !strings.Contains(path, tt.contains) {
				t.Errorf("GetWorktreePathWithRemote() = %v, want to contain %v", path, tt.contains)
			}
		})
	}
}

func TestGetWorkBranchPath(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		setup      func()
		cleanup    func()
		contains   string
	}{
		{
			name:       "simple branch name",
			branchName: "feature-x",
			setup:      func() {},
			cleanup:    func() {},
			contains:   ".worktree/work/feature-x",
		},
		{
			name:       "branch with slashes",
			branchName: "feature/new-feature",
			setup:      func() {},
			cleanup:    func() {},
			contains:   ".worktree/work/feature/new-feature",
		},
		{
			name:       "custom base path",
			branchName: "bugfix-123",
			setup: func() {
				os.Setenv("GH_WORKTREE_BASE_PATH", ".branches")
			},
			cleanup: func() {
				os.Unsetenv("GH_WORKTREE_BASE_PATH")
			},
			contains: ".branches/work/bugfix-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			manager := NewManager()
			path := manager.GetWorkBranchPath(tt.branchName)
			if !strings.Contains(path, tt.contains) {
				t.Errorf("GetWorkBranchPath() = %v, want to contain %v", path, tt.contains)
			}
		})
	}
}

func TestPathConsistency(t *testing.T) {
	manager := NewManager()

	t.Run("PR paths are consistent", func(t *testing.T) {
		prNumber := "999"

		// These should produce the same path when no remote is specified
		path1 := manager.getWorktreePath(prNumber)
		path2 := manager.GetWorktreePath(prNumber)

		if path1 != path2 {
			t.Errorf("Inconsistent PR paths: private=%v, public=%v", path1, path2)
		}
	})

	t.Run("Work branch paths are consistent", func(t *testing.T) {
		branchName := "test-consistency"

		// These should produce the same path
		path1 := manager.getWorkBranchPath(branchName)
		path2 := manager.GetWorkBranchPath(branchName)

		if path1 != path2 {
			t.Errorf("Inconsistent work branch paths: private=%v, public=%v", path1, path2)
		}
	})
}

func TestPathStructure(t *testing.T) {
	manager := NewManager()

	t.Run("PR path structure", func(t *testing.T) {
		tests := []struct {
			prNumber   string
			remoteName string
			wantSuffix string
		}{
			{"123", "", "pr/pr-123"},
			{"456", "origin", "pr/origin/pr-456"},
			{"789", "upstream", "pr/upstream/pr-789"},
		}

		for _, tt := range tests {
			var path string
			if tt.remoteName == "" {
				path = manager.GetWorktreePath(tt.prNumber)
			} else {
				path = manager.GetWorktreePathWithRemote(tt.prNumber, tt.remoteName)
			}

			if !strings.HasSuffix(path, tt.wantSuffix) {
				t.Errorf("Path %v doesn't have expected suffix %v", path, tt.wantSuffix)
			}
		}
	})

	t.Run("Work branch path structure", func(t *testing.T) {
		tests := []struct {
			branchName string
			wantSuffix string
		}{
			{"feature-x", "work/feature-x"},
			{"bugfix/issue-123", "work/bugfix/issue-123"},
			{"release-v1.0", "work/release-v1.0"},
		}

		for _, tt := range tests {
			path := manager.GetWorkBranchPath(tt.branchName)
			if !strings.HasSuffix(path, tt.wantSuffix) {
				t.Errorf("Path %v doesn't have expected suffix %v", path, tt.wantSuffix)
			}
		}
	})
}

func TestConfigFileIntegration(t *testing.T) {
	t.Run("respects project config file", func(t *testing.T) {
		// Create a temporary config file
		configContent := `worktree_base_path: .test-worktrees`
		if err := os.WriteFile(".gh-worktree.yml", []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(".gh-worktree.yml")

		manager := NewManager()
		path := manager.GetWorktreePath("100")
		if !strings.Contains(path, ".test-worktrees/pr/pr-100") {
			t.Errorf("Path doesn't respect config file: %v", path)
		}
	})
}

func TestAbsolutePaths(t *testing.T) {
	manager := NewManager()

	t.Run("paths are absolute when git root is available", func(t *testing.T) {
		// This test will only work in a git repository
		path := manager.GetWorktreePath("123")

		// If we're in a git repo, the path should be absolute
		// Otherwise, it falls back to relative path
		if filepath.IsAbs(path) {
			// Verify it contains the expected structure
			if !strings.Contains(path, ".worktree/pr/pr-123") {
				t.Errorf("Absolute path doesn't contain expected structure: %v", path)
			}
		} else {
			// Verify relative path structure
			if path != ".worktree/pr/pr-123" {
				t.Errorf("Relative path doesn't match expected: %v", path)
			}
		}
	})
}
