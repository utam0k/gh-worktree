package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/utam0k/gh-worktree/internal/ui"
	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executeCd() error {
	fs := newFlagSet("cd")

	var remoteName string
	fs.StringVar(&remoteName, "remote", "", "Specify git remote name (optional)")
	fs.StringVar(&remoteName, "r", "", "Specify git remote name (optional)")

	var workBranch string
	fs.StringVar(&workBranch, "work", "", "Work branch name (optional)")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printCdHelp()
		return nil
	}

	wtManager := worktree.NewManager()

	args := fs.Args()
	if workBranch != "" && len(args) > 0 {
		return fmt.Errorf("unexpected argument: %s", args[0])
	}

	if workBranch != "" && remoteName != "" {
		return fmt.Errorf("--remote cannot be used with --work")
	}

	if len(args) == 0 {
		if workBranch != "" {
			path, err := findWorkBranchWorktreePath(wtManager, workBranch)
			if err != nil {
				return err
			}
			fmt.Print(path)
			return nil
		}

		worktrees, err := wtManager.List()
		if err != nil {
			return err
		}

		if len(worktrees) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		if !isTerminal() {
			return fmt.Errorf("target required in non-interactive mode")
		}

		selectedWT, err := selectWorktree(worktrees)
		if err != nil {
			return err
		}

		fmt.Print(selectedWT.Path)
		return nil
	}

	target := args[0]
	if _, err := strconv.Atoi(target); err != nil {
		if remoteName != "" {
			return fmt.Errorf("--remote cannot be used with work branches")
		}
		path, err := findWorkBranchWorktreePath(wtManager, target)
		if err != nil {
			return err
		}
		fmt.Print(path)
		return nil
	}
	prNumber := target

	var worktreePath string
	if remoteName != "" {
		worktreePath = wtManager.GetWorktreePathWithRemote(prNumber, remoteName)
	} else {
		// Try to find the worktree from the list
		worktrees, err := wtManager.List()
		if err != nil {
			return err
		}

		for _, wt := range worktrees {
			if wt.PRNumber == prNumber {
				worktreePath = wt.Path
				break
			}
		}

		if worktreePath == "" {
			worktreePath = wtManager.GetWorktreePath(prNumber)
		}
	}

	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree for PR #%s not found", prNumber)
	}

	fmt.Print(worktreePath)
	return nil
}

func printCdHelp() {
	fmt.Println("Usage: gh worktree cd [target] [-r/--remote <name>] [--work <branch>]")
	fmt.Println()
	fmt.Println("Output the path to cd into the worktree (interactive if omitted)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -r, --remote <name>    Specify git remote name (optional)")
	fmt.Println("  --work <branch>        Specify a work branch name explicitly")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cd $(gh worktree cd 123)")
	fmt.Println("  cd $(gh worktree cd -r upstream 123)")
	fmt.Println("  cd $(gh worktree cd feature/my-branch)")
	fmt.Println("  cd $(gh worktree cd)")
}

func selectWorktree(worktrees []worktree.WorktreeInfo) (*worktree.WorktreeInfo, error) {
	selector := ui.NewWorktreeSelector(worktrees)
	return selector.SelectWorktree()
}
