package cli

import (
	"fmt"

	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executeList() error {
	wtManager := worktree.NewManager()

	worktrees, err := wtManager.List()
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	// Separate PR and work worktrees
	var prWorktrees []worktree.WorktreeInfo
	var workWorktrees []worktree.WorktreeInfo

	for _, wt := range worktrees {
		if wt.Type == "pr" {
			prWorktrees = append(prWorktrees, wt)
		} else if wt.Type == "work" {
			workWorktrees = append(workWorktrees, wt)
		}
	}

	// Print PR worktrees
	if len(prWorktrees) > 0 {
		fmt.Println("PR Worktrees:")
		fmt.Printf("%-10s %-10s %-20s %-15s %s\n", "PR", "REMOTE", "BRANCH", "COMMIT", "PATH")
		fmt.Printf("%-10s %-10s %-20s %-15s %s\n", "---", "------", "------", "------", "----")

		for _, wt := range prWorktrees {
			pr := fmt.Sprintf("#%s", wt.PRNumber)
			remote := wt.Remote
			if remote == "" {
				remote = "-"
			}
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			// Truncate branch name if too long
			if len(branch) > 18 {
				branch = branch[:15] + "..."
			}

			fmt.Printf("%-10s %-10s %-20s %-15s %s\n", pr, remote, branch, wt.Commit, wt.DisplayPath)
		}
	}

	// Print work worktrees
	if len(workWorktrees) > 0 {
		if len(prWorktrees) > 0 {
			fmt.Println() // Add blank line between sections
		}
		fmt.Println("Work Worktrees:")
		fmt.Printf("%-30s %-15s %s\n", "BRANCH", "COMMIT", "PATH")
		fmt.Printf("%-30s %-15s %s\n", "------", "------", "----")

		for _, wt := range workWorktrees {
			branch := wt.Branch
			// Truncate branch name if too long
			if len(branch) > 28 {
				branch = branch[:25] + "..."
			}

			fmt.Printf("%-30s %-15s %s\n", branch, wt.Commit, wt.DisplayPath)
		}
	}

	return nil
}
