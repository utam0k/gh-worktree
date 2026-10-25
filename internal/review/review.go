package review

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitutil "github.com/utam0k/gh-worktree/internal/git"
)

const (
	keyEnabled    = "gh-worktree.review.enabled"
	keyPR         = "gh-worktree.review.pr"
	keyHead       = "gh-worktree.review.head"
	keyBaseRepo   = "gh-worktree.review.base-repo"
	keyBaseBranch = "gh-worktree.review.base-branch"
	keyBaseRef    = "gh-worktree.review.base-ref"
	keyMergeBase  = "gh-worktree.review.merge-base"
)

type ReviewInfo struct {
	Enabled    bool
	PRNumber   string
	HeadSHA    string
	BaseRepo   string
	BaseBranch string
	BaseRef    string
	MergeBase  string
}

func ResolveBaseRef(path string, prNumber string, baseRepo string, baseBranch string) (string, error) {
	baseRepo = strings.TrimSpace(baseRepo)
	baseBranch = strings.TrimSpace(baseBranch)
	if baseRepo == "" {
		return "", fmt.Errorf("base repository is required")
	}
	if baseBranch == "" {
		return "", fmt.Errorf("base branch is required")
	}

	remote, err := findRemoteForRepo(path, baseRepo)
	if err != nil {
		return "", err
	}
	if remote != "" {
		ref := fmt.Sprintf("refs/remotes/%s/%s", remote, baseBranch)
		if ok, err := refExists(path, ref); err != nil {
			return "", err
		} else if !ok {
			if err := fetchRef(path, remote, baseBranch, ref); err != nil {
				return fetchBaseByURL(path, prNumber, baseRepo, baseBranch)
			}
		}
		return ref, nil
	}

	return fetchBaseByURL(path, prNumber, baseRepo, baseBranch)
}

func Enable(path string, prNumber string, headSHA string, baseRepo string, baseBranch string, baseRef string) (ReviewInfo, error) {
	headSHA = strings.TrimSpace(headSHA)
	baseRef = strings.TrimSpace(baseRef)
	if headSHA == "" {
		return ReviewInfo{}, fmt.Errorf("head SHA is required")
	}
	if baseRef == "" {
		return ReviewInfo{}, fmt.Errorf("base ref is required")
	}

	mergeBase, err := gitOutput(path, "merge-base", baseRef, headSHA)
	if err != nil {
		return ReviewInfo{}, fmt.Errorf("failed to resolve merge-base: %w", err)
	}

	if err := gitRun(path, "reset", "--mixed", mergeBase); err != nil {
		return ReviewInfo{}, fmt.Errorf("failed to apply review mode: %w", err)
	}

	info := ReviewInfo{
		Enabled:    true,
		PRNumber:   prNumber,
		HeadSHA:    headSHA,
		BaseRepo:   baseRepo,
		BaseBranch: baseBranch,
		BaseRef:    baseRef,
		MergeBase:  mergeBase,
	}
	if err := writeConfig(path, info); err != nil {
		return ReviewInfo{}, err
	}

	return info, nil
}

func Disable(path string) error {
	info, ok, err := Read(path)
	if err != nil {
		return err
	}
	if ok && info.HeadSHA != "" {
		if err := gitRun(path, "reset", "--hard", info.HeadSHA); err != nil {
			return fmt.Errorf("failed to reset to head: %w", err)
		}
	}
	for _, key := range []string{keyEnabled, keyPR, keyHead, keyBaseRepo, keyBaseBranch, keyBaseRef, keyMergeBase} {
		_ = gitConfigRun(path, "--unset-all", key)
	}
	return nil
}

func Read(path string) (ReviewInfo, bool, error) {
	enabledValue, ok, err := gitConfigGet(path, keyEnabled)
	if err != nil {
		return ReviewInfo{}, false, err
	}
	if !ok || !isTruthy(enabledValue) {
		return ReviewInfo{}, false, nil
	}

	prNumber, _ := gitConfigGetValue(path, keyPR)
	headSHA, _ := gitConfigGetValue(path, keyHead)
	baseRepo, _ := gitConfigGetValue(path, keyBaseRepo)
	baseBranch, _ := gitConfigGetValue(path, keyBaseBranch)
	baseRef, _ := gitConfigGetValue(path, keyBaseRef)
	mergeBase, _ := gitConfigGetValue(path, keyMergeBase)

	info := ReviewInfo{
		Enabled:    true,
		PRNumber:   prNumber,
		HeadSHA:    headSHA,
		BaseRepo:   baseRepo,
		BaseBranch: baseBranch,
		BaseRef:    baseRef,
		MergeBase:  mergeBase,
	}
	return info, true, nil
}

// IsReviewOnlyDirty reports whether the working tree still matches the PR head,
// i.e. whether every change in it comes from review mode rather than local edits.
func IsReviewOnlyDirty(path string, info ReviewInfo) (bool, error) {
	if !info.Enabled || info.HeadSHA == "" {
		return false, nil
	}

	// Files added by the PR are untracked in review mode, so neither the index
	// nor `git diff <head>` sees them. Snapshot the working tree into a scratch
	// index instead and compare the resulting tree with the PR head's tree.
	tmpDir, err := os.MkdirTemp("", "gh-worktree-index-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmpDir)
	tmpIndex := filepath.Join(tmpDir, "index")

	// Start from a copy of the real index so unchanged files are not rehashed.
	indexPath, err := gitOutput(path, "rev-parse", "--git-path", "index")
	if err != nil {
		return false, fmt.Errorf("failed to resolve index path: %w", err)
	}
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(path, indexPath)
	}
	if data, err := os.ReadFile(indexPath); err == nil {
		if err := os.WriteFile(tmpIndex, data, 0o600); err != nil {
			return false, err
		}
	}

	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndex)
	addCmd := exec.Command("git", "-C", path, "add", "-A")
	addCmd.Env = env
	if output, err := addCmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to snapshot worktree: %s", strings.TrimSpace(string(output)))
	}
	writeCmd := exec.Command("git", "-C", path, "write-tree")
	writeCmd.Env = env
	worktreeTree, err := writeCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to snapshot worktree: %w", err)
	}

	headTree, err := gitOutput(path, "rev-parse", info.HeadSHA+"^{tree}")
	if err != nil {
		return false, fmt.Errorf("failed to resolve head tree: %w", err)
	}
	return strings.TrimSpace(string(worktreeTree)) == headTree, nil
}

func findRemoteForRepo(path string, targetRepo string) (string, error) {
	output, err := gitOutput(path, "remote")
	if err != nil {
		return "", err
	}
	remotes := strings.Fields(output)
	for _, remote := range remotes {
		url, err := gitOutput(path, "remote", "get-url", remote)
		if err != nil {
			continue
		}
		repo, err := gitutil.ParseGitHubRepo(strings.TrimSpace(url))
		if err != nil {
			continue
		}
		if strings.EqualFold(repo, targetRepo) {
			return remote, nil
		}
	}
	return "", nil
}

func fetchBaseByURL(path string, prNumber string, baseRepo string, baseBranch string) (string, error) {
	baseRef := fmt.Sprintf("refs/gh-worktree/pr/%s/base", prNumber)
	repoURL := fmt.Sprintf("https://github.com/%s.git", baseRepo)
	if err := fetchRef(path, repoURL, baseBranch, baseRef); err != nil {
		return "", err
	}
	return baseRef, nil
}

func fetchRef(path string, remoteOrURL string, sourceRef string, destRef string) error {
	spec := fmt.Sprintf("%s:%s", sourceRef, destRef)
	cmd := exec.Command("git", "-C", path, "fetch", remoteOrURL, spec)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("git fetch failed: %s", errMsg)
		}
		return fmt.Errorf("git fetch failed: %w", err)
	}
	return nil
}

func refExists(path string, ref string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "show-ref", "--verify", "--quiet", ref)
	if err := cmd.Run(); err == nil {
		return true, nil
	} else if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to verify ref %s: %w", ref, err)
	}
	return false, nil
}

func writeConfig(path string, info ReviewInfo) error {
	if err := gitConfigRun(path, keyEnabled, "true"); err != nil {
		return err
	}
	if info.PRNumber != "" {
		if err := gitConfigRun(path, keyPR, info.PRNumber); err != nil {
			return err
		}
	}
	if info.HeadSHA != "" {
		if err := gitConfigRun(path, keyHead, info.HeadSHA); err != nil {
			return err
		}
	}
	if info.BaseRepo != "" {
		if err := gitConfigRun(path, keyBaseRepo, info.BaseRepo); err != nil {
			return err
		}
	}
	if info.BaseBranch != "" {
		if err := gitConfigRun(path, keyBaseBranch, info.BaseBranch); err != nil {
			return err
		}
	}
	if info.BaseRef != "" {
		if err := gitConfigRun(path, keyBaseRef, info.BaseRef); err != nil {
			return err
		}
	}
	if info.MergeBase != "" {
		if err := gitConfigRun(path, keyMergeBase, info.MergeBase); err != nil {
			return err
		}
	}
	return nil
}

func gitConfigGet(path string, key string) (string, bool, error) {
	configPath, err := worktreeConfigPath(path)
	if err != nil {
		return "", false, err
	}
	cmd := exec.Command("git", "-C", path, "config", "--file", configPath, "--get", key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return "", false, nil
			}
			msg := strings.TrimSpace(string(output))
			if strings.Contains(msg, "unable to read config file") || strings.Contains(msg, "could not lock config file") {
				return "", false, nil
			}
		}
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return "", false, fmt.Errorf("failed to read config %s: %s", key, msg)
		}
		return "", false, fmt.Errorf("failed to read config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), true, nil
}

func gitConfigGetValue(path string, key string) (string, error) {
	value, _, err := gitConfigGet(path, key)
	return value, err
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func gitRun(path string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return err
	}
	return nil
}

func gitConfigRun(path string, args ...string) error {
	configPath, err := worktreeConfigPath(path)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", append([]string{"-C", path, "config", "--file", configPath}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return err
	}
	return nil
}

func worktreeConfigPath(path string) (string, error) {
	configPath, err := gitOutput(path, "rev-parse", "--git-path", "config.worktree")
	if err != nil {
		return "", fmt.Errorf("failed to resolve worktree config path: %w", err)
	}
	if strings.TrimSpace(configPath) == "" {
		return "", fmt.Errorf("worktree config path is empty")
	}
	return configPath, nil
}

func gitOutput(path string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("%s", errMsg)
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
