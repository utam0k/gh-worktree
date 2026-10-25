package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/utam0k/gh-worktree/internal/worktree"
)

func lookupPRWorktreePath(wtManager *worktree.Manager, prNumber string, remoteName string) (string, bool, error) {
	wt, ok, err := lookupPRWorktreeInfo(wtManager, prNumber, remoteName)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return wt.Path, true, nil
}

func lookupPRWorktreeInfo(wtManager *worktree.Manager, prNumber string, remoteName string) (*worktree.WorktreeInfo, bool, error) {
	worktrees, err := wtManager.List()
	if err != nil {
		return nil, false, err
	}

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
		return nil, false, fmt.Errorf("multiple worktrees found for PR #%s (remotes: %s); specify --remote", prNumber, strings.Join(remotes, ", "))
	}

	if len(matches) == 1 {
		return &matches[0], true, nil
	}
	return nil, false, nil
}

func lookupWorkBranchWorktreePath(wtManager *worktree.Manager, branchName string) (string, bool, error) {
	worktrees, err := wtManager.List()
	if err == nil {
		for _, wt := range worktrees {
			if wt.Type == "work" && wt.Branch == branchName {
				return wt.Path, true, nil
			}
		}
	}

	path := wtManager.GetWorkBranchPath(branchName)
	if _, statErr := os.Stat(path); statErr == nil {
		return path, true, nil
	}

	if err != nil {
		return "", false, err
	}
	return "", false, nil
}

func findWorkBranchWorktreePath(wtManager *worktree.Manager, branchName string) (string, error) {
	path, ok, err := lookupWorkBranchWorktreePath(wtManager, branchName)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("worktree for branch %s not found", branchName)
	}
	return path, nil
}
