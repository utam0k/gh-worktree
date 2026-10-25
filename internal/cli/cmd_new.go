package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/utam0k/gh-worktree/internal/git"
	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executeNew() error {
	fs := newFlagSet("new")

	var baseBranch string
	fs.StringVar(&baseBranch, "base", "", "Base branch to create from (default: main/master)")
	fs.StringVar(&baseBranch, "b", "", "Base branch to create from (default: main/master)")

	var shouldCd bool
	fs.BoolVar(&shouldCd, "cd", false, "Output only the path for cd after creation")

	var reuse bool
	fs.BoolVar(&reuse, "reuse", false, "Reuse existing worktree if it already exists")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printNewHelp()
		return nil
	}

	var branchName string
	if args := fs.Args(); len(args) > 0 {
		branchName = args[0]
	}

	if branchName == "" {
		if !isTerminal() {
			return fmt.Errorf("branch name required in non-interactive mode")
		}

		fmt.Print("Enter new branch name: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read branch name: %w", err)
		}
		branchName = strings.TrimSpace(input)

		if branchName == "" {
			return fmt.Errorf("branch name cannot be empty")
		}
	}

	if baseBranch == "" {
		baseBranch = git.GetDefaultBranch()
	}

	wtManager := worktree.NewManager()

	if reuse {
		if existingPath, ok, err := lookupWorkBranchWorktreePath(wtManager, branchName); err != nil {
			return err
		} else if ok {
			if shouldCd {
				fmt.Print(existingPath)
				return nil
			}
			fmt.Printf("✓ Worktree already exists at %s\n", existingPath)
			fmt.Printf("To enter the worktree: cd %s\n", existingPath)
			return nil
		}
	}

	if !shouldCd {
		fmt.Printf("Creating worktree for branch '%s' from '%s'...\n", branchName, baseBranch)
	}

	if err := wtManager.CreateWorkBranch(branchName, baseBranch); err != nil {
		return err
	}

	worktreePath := wtManager.GetWorkBranchPath(branchName)
	if shouldCd {
		fmt.Print(worktreePath)
		return nil
	}

	fmt.Printf("✓ Successfully created worktree at %s\n", worktreePath)
	fmt.Printf("To enter the worktree: cd %s\n", worktreePath)

	return nil
}

func printNewHelp() {
	fmt.Println("Usage: gh worktree new [branch-name] [-b/--base <branch>] [--cd] [--reuse]")
	fmt.Println()
	fmt.Println("Create a new worktree for development work")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -b, --base <branch>    Base branch to create from (default: main/master)")
	fmt.Println("  --cd                   Output only the path for cd after creation")
	fmt.Println("  --reuse                Reuse existing worktree if it already exists")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gh worktree new feature/awesome-feature")
	fmt.Println("  gh worktree new -b develop feature/new-api")
	fmt.Println("  gh worktree new --cd feature/quick-fix")
	fmt.Println("  gh worktree new --reuse --cd feature/quick-fix")
}
