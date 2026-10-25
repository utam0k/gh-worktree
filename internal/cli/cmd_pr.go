package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/utam0k/gh-worktree/internal/git"
	"github.com/utam0k/gh-worktree/internal/github"
	"github.com/utam0k/gh-worktree/internal/review"
	"github.com/utam0k/gh-worktree/internal/ui"
	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executePR() error {
	fs := newFlagSet("pr")

	var remoteName string
	fs.StringVar(&remoteName, "remote", "", "Specify git remote name (e.g., origin, upstream)")
	fs.StringVar(&remoteName, "r", "", "Specify git remote name (e.g., origin, upstream)")

	var flagCd bool
	fs.BoolVar(&flagCd, "cd", false, "Output only the worktree path (non-interactive)")

	var reuse bool
	fs.BoolVar(&reuse, "reuse", false, "Reuse existing worktree if it already exists")

	var noReview bool
	fs.BoolVar(&noReview, "no-review", false, "Disable review mode (default: enabled)")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printPRHelp()
		return nil
	}

	args := fs.Args()
	var prNumber string
	if len(args) > 0 {
		prNumber = args[0]
	}

	pathOnly := flagCd || os.Getenv("GH_WORKTREE_CD") == "1"
	if flagCd && prNumber == "" {
		return fmt.Errorf("PR number required with --cd")
	}

	if prNumber == "" {
		if !isTerminal() {
			return fmt.Errorf("PR number required in non-interactive mode")
		}
		pathOnly = false
	} else {
		if _, err := strconv.Atoi(prNumber); err != nil {
			return fmt.Errorf("invalid PR number: %s", prNumber)
		}
	}

	reviewEnabled := !noReview

	wtManager := worktree.NewManager()

	// Look for an existing worktree before touching the remote or GitHub,
	// so that --reuse does not need them unless review mode must be enabled.
	if reuse && prNumber != "" {
		if found, err := reuseExistingPRWorktree(wtManager, prNumber, remoteName, reviewEnabled, pathOnly); found || err != nil {
			return err
		}
	}

	// Get repository from remote if specified
	var repo string
	if remoteName != "" {
		var err error
		repo, err = git.GetRepoFromRemote(remoteName)
		if err != nil {
			return fmt.Errorf("failed to get repository from remote '%s': %w", remoteName, err)
		}
		if !pathOnly {
			fmt.Printf("Using remote '%s' (%s)\n", remoteName, repo)
		}
	}
	// If no remote specified, let gh determine the repository

	// If no PR number provided, show interactive selection
	if prNumber == "" {
		selectedPR, err := selectPRWithRepo(repo)
		if err != nil {
			return err
		}
		prNumber = selectedPR

		if reuse {
			if found, err := reuseExistingPRWorktree(wtManager, prNumber, remoteName, reviewEnabled, pathOnly); found || err != nil {
				return err
			}
		}
	}

	ghClient := github.NewClient()

	if !pathOnly {
		fmt.Printf("Fetching PR #%s...\n", prNumber)
	}

	prData, err := ghClient.FetchPRWithRepo(prNumber, repo, remoteName)
	if err != nil {
		return err
	}

	if !pathOnly {
		if prData.RefName != "" {
			fmt.Printf("PR branch: %s\n", prData.RefName)
		}
	}

	if prData.BaseRepo == "" {
		if baseRepo, err := resolveBaseRepo(repo, remoteName); err == nil {
			prData.BaseRepo = baseRepo
		}
	}

	if !pathOnly {
		shortSHA := prData.CommitSHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}
		fmt.Printf("Creating worktree at commit %s...\n", shortSHA)
	}

	var worktreePath string
	if remoteName != "" {
		if err := wtManager.CreateWithRemote(prNumber, prData.CommitSHA, remoteName); err != nil {
			return err
		}
		worktreePath = wtManager.GetWorktreePathWithRemote(prNumber, remoteName)
	} else {
		if err := wtManager.Create(prNumber, prData.CommitSHA); err != nil {
			return err
		}
		worktreePath = wtManager.GetWorktreePath(prNumber)
	}

	if reviewEnabled {
		if err := applyReviewMode(worktreePath, prNumber, prData, true); err != nil {
			return err
		}
	}

	if pathOnly {
		fmt.Print(worktreePath)
		return nil
	}

	fmt.Printf("✓ Successfully created worktree for PR #%s at %s\n", prNumber, worktreePath)
	fmt.Printf("To enter the worktree: cd %s\n", worktreePath)
	fmt.Printf("Or use: cd $(gh worktree cd %s)\n", prNumber)

	return nil
}

func printPRHelp() {
	fmt.Println("Usage: gh worktree pr [pr-number] [--cd] [--reuse] [--no-review] [-r/--remote <name>]")
	fmt.Println()
	fmt.Println("Create a worktree for PR review")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -r, --remote <name>    Specify git remote name (e.g., origin, upstream)")
	fmt.Println("  --cd                   Output only the path (non-interactive)")
	fmt.Println("  --reuse                Reuse existing worktree if it already exists")
	fmt.Println("  --no-review            Disable review mode (default: enabled)")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gh worktree pr 123                Create a worktree for PR #123")
	fmt.Println("  gh worktree pr -r upstream 123    Create from upstream remote")
	fmt.Println("  gh worktree pr --cd 123           Create and output path for cd")
	fmt.Println("  gh worktree pr --reuse --cd 123   Reuse existing and output path")
	fmt.Println("  gh worktree pr --no-review 123    Create without review mode")
	fmt.Println("  gh worktree pr                    Select a PR interactively")
}

// reuseExistingPRWorktree reports whether a worktree for the PR already exists.
// If it does, its review mode is brought in line with the flags and its path is printed.
func reuseExistingPRWorktree(wtManager *worktree.Manager, prNumber string, remoteName string, reviewEnabled bool, pathOnly bool) (bool, error) {
	existingPath, ok, err := lookupPRWorktreePath(wtManager, prNumber, remoteName)
	if err != nil || !ok {
		return false, err
	}

	if err := reconcileReviewMode(existingPath, prNumber, remoteName, reviewEnabled); err != nil {
		if !errors.Is(err, errReviewLocalChanges) {
			return true, err
		}
		if !pathOnly {
			fmt.Printf("! Worktree has local changes; review mode unchanged\n")
		}
	}

	if pathOnly {
		fmt.Print(existingPath)
		return true, nil
	}
	fmt.Printf("✓ Worktree already exists for PR #%s at %s\n", prNumber, existingPath)
	fmt.Printf("To enter the worktree: cd %s\n", existingPath)
	fmt.Printf("Or use: cd $(gh worktree cd %s)\n", prNumber)
	return true, nil
}

// reconcileReviewMode applies the requested review mode to an existing worktree.
// GitHub is only queried when review mode has to be newly enabled.
func reconcileReviewMode(path string, prNumber string, remoteName string, enable bool) error {
	if !enable {
		return applyReviewMode(path, prNumber, nil, false)
	}
	if _, ok, err := review.Read(path); err != nil || ok {
		return err
	}

	var repo string
	if remoteName != "" {
		var err error
		repo, err = git.GetRepoFromRemote(remoteName)
		if err != nil {
			return fmt.Errorf("failed to get repository from remote '%s': %w", remoteName, err)
		}
	}
	prData, err := github.NewClient().FetchPRWithRepo(prNumber, repo, remoteName)
	if err != nil {
		return err
	}
	if prData.BaseRepo == "" {
		if baseRepo, err := resolveBaseRepo(repo, remoteName); err == nil {
			prData.BaseRepo = baseRepo
		}
	}

	// Review what is checked out, not the latest PR head; moving the
	// worktree forward is `gh worktree sync`'s job.
	head, err := gitOutputStrict(path, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	prData.CommitSHA = head
	return applyReviewMode(path, prNumber, prData, true)
}

var errReviewLocalChanges = errors.New("worktree has local changes")

func applyReviewMode(path string, prNumber string, prData *github.PRData, enable bool) error {
	if enable {
		info, ok, err := review.Read(path)
		if err != nil {
			return err
		}
		if !ok {
			dirty, err := worktreeHasChanges(path)
			if err != nil {
				return err
			}
			if dirty {
				return errReviewLocalChanges
			}
		} else {
			reviewOnly, err := review.IsReviewOnlyDirty(path, info)
			if err != nil {
				return err
			}
			if !reviewOnly {
				return errReviewLocalChanges
			}
		}

		baseRepo := strings.TrimSpace(prData.BaseRepo)
		baseBranch := strings.TrimSpace(prData.BaseRefName)
		if baseRepo == "" || baseBranch == "" {
			return fmt.Errorf("failed to resolve base repository or branch for PR #%s", prNumber)
		}
		baseRef, err := review.ResolveBaseRef(path, prNumber, baseRepo, baseBranch)
		if err != nil {
			return err
		}
		_, err = review.Enable(path, prNumber, prData.CommitSHA, baseRepo, baseBranch, baseRef)
		return err
	}

	info, ok, err := review.Read(path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	reviewOnly, err := review.IsReviewOnlyDirty(path, info)
	if err != nil {
		return err
	}
	if !reviewOnly {
		return errReviewLocalChanges
	}
	return review.Disable(path)
}

func worktreeHasChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func resolveBaseRepo(repo string, remoteName string) (string, error) {
	if repo != "" {
		return repo, nil
	}
	if remoteName != "" {
		return git.GetRepoFromRemote(remoteName)
	}
	if defaultRemote := git.GetDefaultRemote(); defaultRemote != "" {
		return git.GetRepoFromRemote(defaultRemote)
	}
	return "", fmt.Errorf("no git remote found to resolve base repository")
}

func selectPRWithRepo(repo string) (string, error) {
	ghClient := github.NewClient()

	fmt.Println("Fetching PR list...")
	prs, err := ghClient.ListPRsWithRepo(repo)
	if err != nil {
		return "", err
	}

	if len(prs) == 0 {
		return "", fmt.Errorf("no open pull requests found")
	}

	selector := ui.NewPRSelector(prs)
	return selector.SelectPR()
}
