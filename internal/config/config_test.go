package config

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWorktreeBasePath(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("default value", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Setenv("GH_WORKTREE_BASE_PATH", "")
		if got := GetWorktreeBasePath(); got != ".worktree" {
			t.Fatalf("GetWorktreeBasePath() = %v, want %v", got, ".worktree")
		}
	})

	t.Run("environment variable", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Setenv("GH_WORKTREE_BASE_PATH", ".my-worktrees")
		if got := GetWorktreeBasePath(); got != ".my-worktrees" {
			t.Fatalf("GetWorktreeBasePath() = %v, want %v", got, ".my-worktrees")
		}
	})
}

func TestLoad(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("default config", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Setenv("GH_WORKTREE_BASE_PATH", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != ".worktree" {
			t.Fatalf("Load() WorktreeBasePath = %v, want %v", cfg.WorktreeBasePath, ".worktree")
		}
	})

	t.Run("valid yaml config", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Setenv("GH_WORKTREE_BASE_PATH", "")

		content := `worktree_base_path: custom-path`
		if err := os.WriteFile(".gh-worktree.yml", []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != "custom-path" {
			t.Fatalf("Load() WorktreeBasePath = %v, want %v", cfg.WorktreeBasePath, "custom-path")
		}
	})

	t.Run("invalid yaml config", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		content := `invalid yaml content: [}`
		if err := os.WriteFile(".gh-worktree.yml", []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(); err == nil {
			t.Fatal("Load() expected error but got none")
		}
	})

	t.Run("empty config file", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		t.Setenv("GH_WORKTREE_BASE_PATH", "")

		if err := os.WriteFile(".gh-worktree.yml", []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != ".worktree" {
			t.Fatalf("Load() WorktreeBasePath = %v, want %v", cfg.WorktreeBasePath, ".worktree")
		}
	})

	t.Run("env var overrides config file", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		content := `worktree_base_path: .from-config`
		if err := os.WriteFile(".gh-worktree.yml", []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GH_WORKTREE_BASE_PATH", ".from-env")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != ".from-env" {
			t.Fatalf("Expected env var to override config file, got %v", cfg.WorktreeBasePath)
		}
	})

	t.Run("user config directory", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(tempDir))
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("GH_WORKTREE_BASE_PATH", "")

		configDir := filepath.Join(homeDir, ".config", "gh-worktree")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		configFile := filepath.Join(configDir, "config.yml")
		content := `worktree_base_path: .user-worktrees`
		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != ".user-worktrees" {
			t.Fatalf("Expected user config to be loaded, got %v", cfg.WorktreeBasePath)
		}
	})

	t.Run("discovers config from git root when in subdir", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not found")
		}

		repoDir := t.TempDir()
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(repoDir))

		mustRun(t, repoDir, "git", "init")
		mustRun(t, repoDir, "git", "config", "user.email", "test@example.com")
		mustRun(t, repoDir, "git", "config", "user.name", "Test User")
		mustRun(t, repoDir, "git", "config", "commit.gpgsign", "false")

		if err := os.WriteFile(filepath.Join(repoDir, ".gh-worktree.yml"), []byte("worktree_base_path: .from-root\n"), 0644); err != nil {
			t.Fatal(err)
		}

		subDir := filepath.Join(repoDir, "a", "b")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(originalCwd) })

		t.Setenv("GH_WORKTREE_BASE_PATH", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}
		if cfg.WorktreeBasePath != ".from-root" {
			t.Fatalf("Load() WorktreeBasePath = %v, want %v", cfg.WorktreeBasePath, ".from-root")
		}
	})
}

func mustRun(t *testing.T, dir string, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("Command failed: %s %s\nError: %v\nOutput:\n%s", command, strings.Join(args, " "), err, out.String())
	}
}
