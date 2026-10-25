package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executeRemove() error {
	fs := newFlagSet("remove")

	var remoteName string
	fs.StringVar(&remoteName, "remote", "", "Specify git remote name (e.g., origin, upstream)")
	fs.StringVar(&remoteName, "r", "", "Specify git remote name (e.g., origin, upstream)")

	var workBranch string
	fs.StringVar(&workBranch, "work", "", "Work branch name (optional)")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printRemoveHelp()
		return nil
	}

	args := fs.Args()
	if workBranch != "" && len(args) > 0 {
		return fmt.Errorf("unexpected argument: %s", args[0])
	}

	if remoteName != "" && workBranch != "" {
		return fmt.Errorf("--remote cannot be used with --work")
	}

	if workBranch != "" {
		wtManager := worktree.NewManager()
		path, err := findWorkBranchWorktreePath(wtManager, workBranch)
		if err != nil {
			return err
		}
		if err := wtManager.RemovePath(path); err != nil {
			return err
		}
		fmt.Printf("✓ Successfully removed worktree for branch %s\n", workBranch)
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("target required")
	}

	target := args[0]
	if _, err := strconv.Atoi(target); err != nil {
		if remoteName != "" {
			return fmt.Errorf("--remote cannot be used with work branches")
		}
		wtManager := worktree.NewManager()
		path, err := findWorkBranchWorktreePath(wtManager, target)
		if err != nil {
			return err
		}
		if err := wtManager.RemovePath(path); err != nil {
			return err
		}
		fmt.Printf("✓ Successfully removed worktree for branch %s\n", target)
		return nil
	}
	prNumber := target

	wtManager := worktree.NewManager()

	// Prefer removing the actual worktree path from `git worktree list` so config changes don't break removal.
	worktrees, err := wtManager.List()
	if err == nil {
		var matches []worktree.WorktreeInfo
		for _, wt := range worktrees {
			if wt.Type != "pr" || wt.PRNumber != prNumber {
				continue
			}
			if remoteName != "" && wt.Remote != remoteName {
				continue
			}
			matches = append(matches, wt)
		}

		if remoteName == "" && len(matches) > 1 {
			remotes := make([]string, 0, len(matches))
			seen := make(map[string]struct{}, len(matches))
			for _, wt := range matches {
				remote := wt.Remote
				if remote == "" {
					remote = "(none)"
				}
				if _, ok := seen[remote]; ok {
					continue
				}
				seen[remote] = struct{}{}
				remotes = append(remotes, remote)
			}
			return fmt.Errorf("multiple worktrees found for PR #%s (remotes: %s); specify --remote", prNumber, strings.Join(remotes, ", "))
		}

		if len(matches) == 1 {
			if err := wtManager.RemovePath(matches[0].Path); err != nil {
				return err
			}
			if remoteName != "" {
				fmt.Printf("✓ Successfully removed worktree for PR #%s (%s)\n", prNumber, remoteName)
			} else {
				fmt.Printf("✓ Successfully removed worktree for PR #%s\n", prNumber)
			}
			return nil
		}
	}

	// Fallback: remove based on computed path
	if remoteName != "" {
		if err := wtManager.RemoveWithRemote(prNumber, remoteName); err != nil {
			return err
		}
		fmt.Printf("✓ Successfully removed worktree for PR #%s (%s)\n", prNumber, remoteName)
		return nil
	}

	if err := wtManager.Remove(prNumber); err != nil {
		return err
	}
	fmt.Printf("✓ Successfully removed worktree for PR #%s\n", prNumber)
	return nil
}

func printRemoveHelp() {
	fmt.Println("Usage: gh worktree remove <target> [-r/--remote <name>] [--work <branch>]")
	fmt.Println()
	fmt.Println("Remove a PR worktree (by number) or a work worktree (by branch name)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -r, --remote <name>    Specify git remote name (e.g., origin, upstream)")
	fmt.Println("  --work <branch>        Specify a work branch name explicitly")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gh worktree remove 123              Remove PR #123 worktree")
	fmt.Println("  gh worktree remove -r upstream 123  Remove PR #123 from upstream remote")
	fmt.Println("  gh worktree remove feature/my-branch         Remove worktree for a work branch")
}
