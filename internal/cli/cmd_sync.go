package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/utam0k/gh-worktree/internal/git"
	"github.com/utam0k/gh-worktree/internal/review"
	"github.com/utam0k/gh-worktree/internal/worktree"
)

func executeSync() error {
	fs := newFlagSet("sync")

	var remoteName string
	fs.StringVar(&remoteName, "remote", "", "Specify git remote name (optional)")
	fs.StringVar(&remoteName, "r", "", "Specify git remote name (optional)")

	var force bool
	fs.BoolVar(&force, "force", false, "Discard local changes in the worktree")

	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	fs.BoolVar(&help, "h", false, "Show help")

	if err := parseFlagSet(fs, os.Args[2:]); err != nil {
		return err
	}
	if help {
		printSyncHelp()
		return nil
	}

	args := fs.Args()
	if len(args) == 0 {
		return fmt.Errorf("PR number required")
	}
	prNumber := args[0]
	if _, err := strconv.Atoi(prNumber); err != nil {
		return fmt.Errorf("invalid PR number: %s", prNumber)
	}

	wtManager := worktree.NewManager()
	wt, ok, err := lookupPRWorktreeInfo(wtManager, prNumber, remoteName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("worktree for PR #%s not found", prNumber)
	}

	remote := remoteName
	if remote == "" {
		remote = wt.Remote
	}
	if remote == "" {
		remote = git.GetDefaultRemote()
	}
	if remote == "" {
		return fmt.Errorf("no git remote found (specify --remote)")
	}

	reviewInfo, reviewEnabled, err := review.Read(wt.Path)
	if err != nil {
		return err
	}

	if reviewEnabled {
		if reviewInfo.HeadSHA == "" {
			return fmt.Errorf("review metadata is missing head SHA; re-run gh worktree pr %s", prNumber)
		}
		if !force {
			reviewOnly, err := review.IsReviewOnlyDirty(wt.Path, reviewInfo)
			if err != nil {
				return err
			}
			if !reviewOnly {
				return fmt.Errorf("worktree has local changes; use --force to discard them")
			}
		}
		if err := gitResetHard(wt.Path, reviewInfo.HeadSHA); err != nil {
			return err
		}
	}

	refSpec := fmt.Sprintf("+refs/pull/%s/head:refs/remotes/%s/pr/%s", prNumber, remote, prNumber)
	fetchCmd := exec.Command("git", "fetch", remote, refSpec)
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch PR #%s from %s: %w\nOutput: %s", prNumber, remote, err, strings.TrimSpace(string(fetchOutput)))
	}

	ref := fmt.Sprintf("refs/remotes/%s/pr/%s", remote, prNumber)
	checkoutArgs := []string{"-C", wt.Path, "checkout", "--detach", ref}
	if force && !reviewEnabled {
		checkoutArgs = []string{"-C", wt.Path, "checkout", "-f", "--detach", ref}
	}

	checkoutCmd := exec.Command("git", checkoutArgs...)
	checkoutOutput, err := checkoutCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update worktree: %w\nOutput: %s", err, strings.TrimSpace(string(checkoutOutput)))
	}

	shortSHA, _ := gitOutputStrict(wt.Path, "rev-parse", "--short", "HEAD")

	if reviewEnabled {
		baseRepo := reviewInfo.BaseRepo
		if baseRepo == "" {
			if wt.Remote != "" {
				if repo, err := git.GetRepoFromRemote(wt.Remote); err == nil {
					baseRepo = repo
				}
			}
			if baseRepo == "" {
				if defaultRemote := git.GetDefaultRemote(); defaultRemote != "" {
					if repo, err := git.GetRepoFromRemote(defaultRemote); err == nil {
						baseRepo = repo
					}
				}
			}
		}

		baseRef := reviewInfo.BaseRef
		if baseRepo != "" && reviewInfo.BaseBranch != "" {
			resolved, err := review.ResolveBaseRef(wt.Path, prNumber, baseRepo, reviewInfo.BaseBranch)
			if err != nil {
				return err
			}
			baseRef = resolved
		}
		if baseRef == "" {
			return fmt.Errorf("base ref is missing; re-run gh worktree pr %s", prNumber)
		}
		newHead, err := gitOutputStrict(wt.Path, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if _, err := review.Enable(wt.Path, prNumber, newHead, baseRepo, reviewInfo.BaseBranch, baseRef); err != nil {
			return err
		}
	}

	if shortSHA != "" {
		fmt.Printf("✓ Synced PR #%s to %s\n", prNumber, shortSHA)
	} else {
		fmt.Printf("✓ Synced PR #%s\n", prNumber)
	}

	return nil
}

func printSyncHelp() {
	fmt.Println("Usage: gh worktree sync <pr-number> [-r/--remote <name>] [--force]")
	fmt.Println()
	fmt.Println("Fetch and update an existing PR worktree to the latest PR head commit")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -r, --remote <name>    Specify git remote name (optional)")
	fmt.Println("  --force                Discard local changes in the worktree")
	fmt.Println("  -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gh worktree sync 123")
	fmt.Println("  gh worktree sync -r upstream 123")
	fmt.Println("  gh worktree sync --force 123")
}

func gitResetHard(path string, ref string) error {
	cmd := exec.Command("git", "-C", path, "reset", "--hard", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reset worktree: %w\nOutput: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitOutputStrict(path string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read git output: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
