package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/utam0k/gh-worktree/internal/git"
	"github.com/utam0k/gh-worktree/internal/github"
	"github.com/utam0k/gh-worktree/internal/review"
	"github.com/utam0k/gh-worktree/internal/worktree"
)

type cleanCandidate struct {
	worktree worktree.WorktreeInfo
	reason   string
}

type cleanSkip struct {
	worktree worktree.WorktreeInfo
	reason   string
}

func executeClean() error {
	fs := newFlagSet("clean")

	var includePR bool
	fs.BoolVar(&includePR, "pr", true, "Check PR worktrees")

	var includeWork bool
	fs.BoolVar(&includeWork, "work", true, "Check work worktrees")

	var dryRun bool
	fs.BoolVar(&dryRun, "dry-run", false, "Show candidates without removing anything")

	var yes bool
	fs.BoolVar(&yes, "yes", false, "Remove without confirmation (required in non-interactive mode)")
	fs.BoolVar(&yes, "y", false, "Remove without confirmation (required in non-interactive mode)")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printCleanHelp()
		return nil
	}

	if !includePR && !includeWork {
		return fmt.Errorf("nothing to do (both --pr and --work are disabled)")
	}

	// Best-effort cleanup of stale worktree metadata.
	if err := exec.Command("git", "worktree", "prune").Run(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	currentRoot := ""
	if output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		currentRoot = strings.TrimSpace(string(output))
	}

	wtManager := worktree.NewManager()
	worktrees, err := wtManager.List()
	if err != nil {
		return err
	}

	baseBranch := git.GetDefaultBranch()
	baseRef, baseRefErr := resolveBaseRef(baseBranch)
	if baseRefErr != nil {
		baseRef = ""
	}

	ghAvailable := true
	if _, err := exec.LookPath("gh"); err != nil {
		ghAvailable = false
	}
	ghClient := github.NewClient()

	var candidates []cleanCandidate
	var skipped []cleanSkip

	for _, wt := range worktrees {
		if currentRoot != "" && wt.Path == currentRoot {
			continue
		}

		switch wt.Type {
		case "pr":
			if !includePR {
				continue
			}
			if !ghAvailable {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: "gh not found; cannot check PR state"})
				continue
			}

			repo := ""
			if wt.Remote != "" {
				if r, err := git.GetRepoFromRemote(wt.Remote); err == nil {
					repo = r
				}
			}

			state, err := ghClient.GetPRStateWithRepo(wt.PRNumber, repo)
			if err != nil && repo != "" {
				// Fallback to default repo resolution.
				state, err = ghClient.GetPRStateWithRepo(wt.PRNumber, "")
			}
			if err != nil {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("failed to check PR state: %v", err)})
				continue
			}

			if strings.EqualFold(state, "OPEN") {
				continue
			}

			clean, err := isWorktreeClean(wt.Path)
			if err != nil {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("failed to check status: %v", err)})
				continue
			}
			if !clean {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("worktree has local changes (PR state=%s)", state)})
				continue
			}

			candidates = append(candidates, cleanCandidate{worktree: wt, reason: fmt.Sprintf("PR state=%s", state)})

		case "work":
			if !includeWork {
				continue
			}

			if wt.Branch == "" {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: "branch unknown"})
				continue
			}
			if baseRef == "" {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: "default branch not found"})
				continue
			}

			merged, err := isBranchMergedInto(wt.Branch, baseRef)
			if err != nil {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("failed to check merge status: %v", err)})
				continue
			}
			if !merged {
				continue
			}

			clean, err := isWorktreeClean(wt.Path)
			if err != nil {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("failed to check status: %v", err)})
				continue
			}
			if !clean {
				skipped = append(skipped, cleanSkip{worktree: wt, reason: fmt.Sprintf("worktree has local changes (merged into %s)", baseBranch)})
				continue
			}

			candidates = append(candidates, cleanCandidate{worktree: wt, reason: fmt.Sprintf("merged into %s", baseBranch)})
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No removable worktrees found")
		if len(skipped) > 0 {
			printCleanSkipped(skipped)
		}
		return nil
	}

	printCleanCandidates(candidates)
	if len(skipped) > 0 {
		fmt.Println()
		printCleanSkipped(skipped)
	}

	if dryRun {
		return nil
	}

	if !yes {
		if !isTerminal() {
			return fmt.Errorf("--yes required in non-interactive mode")
		}
		ok, err := confirmClean(len(candidates))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled")
			return nil
		}
	}

	var failures []string
	for _, c := range candidates {
		if err := wtManager.RemovePath(c.worktree.Path); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", c.worktree.Path, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to remove %d worktrees:\n%s", len(failures), strings.Join(failures, "\n"))
	}

	fmt.Printf("✓ Removed %d worktrees\n", len(candidates))
	return nil
}

func printCleanHelp() {
	fmt.Println("Usage: gh worktree clean [--pr[=true|false]] [--work[=true|false]] [--dry-run] [-y/--yes]")
	fmt.Println()
	fmt.Println("Find and remove unnecessary worktrees managed by gh-worktree")
	fmt.Println()
	fmt.Println("Rules (safe defaults):")
	fmt.Println("  - PR worktrees: remove when PR is not OPEN and worktree has no local changes")
	fmt.Println("  - Work worktrees: remove when branch is merged into default branch and worktree has no local changes")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --pr                 Check PR worktrees (default: true)")
	fmt.Println("  --work               Check work worktrees (default: true)")
	fmt.Println("  --dry-run            Show candidates without removing")
	fmt.Println("  -y, --yes            Remove without confirmation (required in non-interactive mode)")
	fmt.Println("  -h, --help           Show this help message")
}

func printCleanCandidates(candidates []cleanCandidate) {
	fmt.Println("Removable worktrees:")
	for _, c := range candidates {
		fmt.Printf("  - %s (%s)\n", formatWorktreeLabel(c.worktree), c.reason)
	}
}

func printCleanSkipped(skipped []cleanSkip) {
	fmt.Println("Skipped:")
	for _, s := range skipped {
		fmt.Printf("  - %s (%s)\n", formatWorktreeLabel(s.worktree), s.reason)
	}
}

func formatWorktreeLabel(wt worktree.WorktreeInfo) string {
	switch wt.Type {
	case "pr":
		remote := wt.Remote
		if remote == "" {
			remote = "-"
		}
		return fmt.Sprintf("[PR #%s %s] %s", wt.PRNumber, remote, wt.DisplayPath)
	case "work":
		return fmt.Sprintf("[WORK %s] %s", wt.Branch, wt.DisplayPath)
	default:
		return wt.DisplayPath
	}
}

func confirmClean(count int) (bool, error) {
	fmt.Printf("Remove %d worktrees? [y/N]: ", count)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	choice := strings.TrimSpace(strings.ToLower(input))
	return choice == "y" || choice == "yes", nil
}

func isWorktreeClean(path string) (bool, error) {
	dirty, err := worktreeHasChanges(path)
	if err != nil {
		return false, err
	}
	if !dirty {
		return true, nil
	}

	info, ok, err := review.Read(path)
	if err != nil {
		return false, err
	}
	if ok {
		reviewOnly, err := review.IsReviewOnlyDirty(path, info)
		if err != nil {
			return false, err
		}
		if reviewOnly {
			return true, nil
		}
	}

	return false, nil
}

func resolveBaseRef(baseBranch string) (string, error) {
	if baseBranch == "" {
		return "", fmt.Errorf("empty base branch")
	}

	candidates := []string{
		"refs/heads/" + baseBranch,
		"refs/remotes/origin/" + baseBranch,
	}

	for _, ref := range candidates {
		if err := exec.Command("git", "show-ref", "--verify", "--quiet", ref).Run(); err == nil {
			return ref, nil
		}
	}
	return "", fmt.Errorf("base branch ref not found: %s", baseBranch)
}

func isBranchMergedInto(branchName string, baseRef string) (bool, error) {
	branchRef := "refs/heads/" + branchName
	if err := exec.Command("git", "show-ref", "--verify", "--quiet", branchRef).Run(); err != nil {
		return false, fmt.Errorf("branch ref not found: %s", branchName)
	}

	cmd := exec.Command("git", "merge-base", "--is-ancestor", branchRef, baseRef)
	runErr := cmd.Run()
	if runErr == nil {
		return true, nil
	} else if exitErr, ok := runErr.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("merge-base failed: %w", runErr)
	}
	return false, fmt.Errorf("merge-base failed: %w", runErr)
}
