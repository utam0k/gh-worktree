package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/utam0k/gh-worktree/internal/config"
)

func executeConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Current Configuration:")
	fmt.Println("=====================")
	fmt.Printf("Worktree Base Path: %s\n", cfg.WorktreeBasePath)

	// Show symlink files if configured
	if len(cfg.SymlinkFiles) > 0 {
		fmt.Println("\nSymlink Files:")
		for _, file := range cfg.SymlinkFiles {
			fmt.Printf("   - %s\n", file)
		}
	}

	// Show where the config was loaded from
	fmt.Println("\nConfiguration Sources (in priority order):")
	fmt.Println("1. Environment Variable: GH_WORKTREE_BASE_PATH")
	if envPath := os.Getenv("GH_WORKTREE_BASE_PATH"); envPath != "" {
		fmt.Printf("   ✓ Set to: %s\n", envPath)
	} else {
		fmt.Println("   - Not set")
	}

	fmt.Println("2. Project Config: .gh-worktree.yml")
	if _, err := os.Stat(".gh-worktree.yml"); err == nil {
		fmt.Println("   ✓ Found")
	} else {
		fmt.Println("   - Not found")
	}

	fmt.Println("3. User Config: ~/.config/gh-worktree/config.yml")
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".config", "gh-worktree", "config.yml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("   ✓ Found")
		} else {
			fmt.Println("   - Not found")
		}
	}

	fmt.Println("4. Default: .worktree")

	// Show actual paths
	fmt.Println("\nWorktree Paths:")

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("   Unable to determine git root")
	} else {
		gitRoot := strings.TrimSpace(string(output))
		fmt.Printf("   PR worktrees: %s/%s/pr/\n", gitRoot, cfg.WorktreeBasePath)
		fmt.Printf("   Work branches: %s/%s/work/\n", gitRoot, cfg.WorktreeBasePath)
	}

	return nil
}
