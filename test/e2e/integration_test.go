package e2e

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithMockRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")

	baseBranch := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "--abbrev-ref", "HEAD"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	branchName := "feature/test-worktree"
	worktreePath := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "new", "-b", baseBranch, "--cd", branchName))
	expectedPath := filepath.Join(tmpDir, ".worktree", "work", branchName)

	if worktreePath != expectedPath {
		t.Fatalf("Expected worktree path %s, got %s", expectedPath, worktreePath)
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("Expected worktree to exist at %s: %v", worktreePath, err)
	}

	listOutput := mustRunInDir(t, tmpDir, binaryPath, "list")
	if !strings.Contains(listOutput, "Work Worktrees:") {
		t.Fatalf("List output doesn't contain 'Work Worktrees:': %s", listOutput)
	}
	if !strings.Contains(listOutput, branchName) {
		t.Fatalf("List output doesn't contain branch name %q: %s", branchName, listOutput)
	}

	// Cleanup
	_, _ = runInDir(t, tmpDir, "git", "worktree", "remove", worktreePath)
	_, _ = runInDir(t, tmpDir, "git", "branch", "-D", branchName)
}

func TestWorkBranchReuseCdRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")

	baseBranch := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "--abbrev-ref", "HEAD"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	branchName := "feature/reuse-test"
	expectedPath := filepath.Join(tmpDir, ".worktree", "work", branchName)

	worktreePath := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "new", "-b", baseBranch, "--cd", branchName))
	if worktreePath != expectedPath {
		t.Fatalf("Expected worktree path %s, got %s", expectedPath, worktreePath)
	}

	reused := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "new", "--reuse", "--cd", branchName))
	if reused != expectedPath {
		t.Fatalf("Expected reused path %s, got %s", expectedPath, reused)
	}

	cdPath := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "cd", branchName))
	if cdPath != expectedPath {
		t.Fatalf("Expected cd path %s, got %s", expectedPath, cdPath)
	}

	_ = mustRunInDir(t, tmpDir, binaryPath, "remove", branchName)
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Fatalf("Expected worktree to be removed at %s", expectedPath)
	}

	_, _ = runInDir(t, tmpDir, "git", "branch", "-D", branchName)
}

func TestCdOutputOnlyPathWithSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")
	mustRunInDir(t, tmpDir, "git", "commit", "--allow-empty", "-m", "Initial commit")

	baseBranch := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "--abbrev-ref", "HEAD"))

	config := "symlink_files:\n  - shared.txt\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".gh-worktree.yml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "shared.txt"), []byte("shared\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	branchName := "feature/symlink-test"
	cmd := exec.Command(binaryPath, "new", "-b", baseBranch, "--cd", branchName)
	cmd.Dir = tmpDir
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("new --cd failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".worktree", "work", branchName)
	if string(stdout) != expectedPath {
		t.Fatalf("Expected stdout to be only %q, got %q", expectedPath, string(stdout))
	}
	if target, err := os.Readlink(filepath.Join(expectedPath, "shared.txt")); err != nil || target != filepath.Join(tmpDir, "shared.txt") {
		t.Fatalf("Expected shared.txt to be symlinked, got target=%q err=%v", target, err)
	}

	_, _ = runInDir(t, tmpDir, "git", "worktree", "remove", "--force", expectedPath)
}

func TestCleanRemovesMergedWorkWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")

	baseBranch := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "--abbrev-ref", "HEAD"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	branchName := "feature/cleanup-test"
	worktreePath := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "new", "-b", baseBranch, "--cd", branchName))

	if err := os.WriteFile(filepath.Join(worktreePath, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, worktreePath, "git", "add", ".")
	mustRunInDir(t, worktreePath, "git", "commit", "-m", "Feature commit")

	mustRunInDir(t, tmpDir, "git", "merge", "--no-ff", branchName, "-m", "Merge feature")

	_ = mustRunInDir(t, tmpDir, binaryPath, "clean", "--yes", "--pr=false")

	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("Expected merged worktree to be removed at %s", worktreePath)
	}
}

func TestPRReuseWithoutGH(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")

	commit := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "HEAD"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	prNum := "123"
	prPath1 := filepath.Join(tmpDir, ".worktree", "pr", "origin", "pr-123")
	prPath2 := filepath.Join(tmpDir, ".worktree", "pr", "upstream", "pr-123")

	if err := os.MkdirAll(filepath.Dir(prPath1), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(prPath2), 0755); err != nil {
		t.Fatal(err)
	}

	mustRunInDir(t, tmpDir, "git", "worktree", "add", prPath1, commit)
	mustRunInDir(t, tmpDir, "git", "worktree", "add", prPath2, commit)

	_, err := runInDir(t, tmpDir, binaryPath, "pr", "--reuse", "--cd", prNum)
	if err == nil {
		t.Fatal("Expected error when multiple PR worktrees exist and --remote is not specified")
	}

	// Without review mode, reusing needs neither gh nor the remote to exist.
	reused := strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "pr", "--remote", "origin", "--reuse", "--no-review", "--cd", prNum))
	if reused != prPath1 {
		t.Fatalf("Expected reused path %s, got %s", prPath1, reused)
	}

	// A worktree that is already in review mode is reused as is.
	worktreeConfig := strings.TrimSpace(mustRunInDir(t, prPath2, "git", "rev-parse", "--git-path", "config.worktree"))
	mustRunInDir(t, prPath2, "git", "config", "--file", worktreeConfig, "gh-worktree.review.enabled", "true")
	reused = strings.TrimSpace(mustRunInDir(t, tmpDir, binaryPath, "pr", "--remote", "upstream", "--reuse", "--cd", prNum))
	if reused != prPath2 {
		t.Fatalf("Expected reused path %s, got %s", prPath2, reused)
	}

	_, _ = runInDir(t, tmpDir, "git", "worktree", "remove", prPath1)
	_, _ = runInDir(t, tmpDir, "git", "worktree", "remove", prPath2)
}

func TestCleanRemovesClosedPRWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")

	commit := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "HEAD"))

	// Provide a stub gh command to control PR state responses.
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ghPath := filepath.Join(binDir, "gh")
	ghScript := `#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "pr" ] && [ "${2:-}" = "view" ]; then
  pr="${3:-}"
  case "$pr" in
    123) echo '{"state":"CLOSED"}' ;;
    456) echo '{"state":"OPEN"}' ;;
    *) echo '{"state":"OPEN"}' ;;
  esac
  exit 0
fi
echo "unexpected args: $*" >&2
exit 1
`
	if err := os.WriteFile(ghPath, []byte(ghScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	prClosedPath := filepath.Join(tmpDir, ".worktree", "pr", "pr-123")
	prOpenPath := filepath.Join(tmpDir, ".worktree", "pr", "pr-456")

	if err := os.MkdirAll(filepath.Dir(prClosedPath), 0755); err != nil {
		t.Fatal(err)
	}

	mustRunInDir(t, tmpDir, "git", "worktree", "add", prClosedPath, commit)
	mustRunInDir(t, tmpDir, "git", "worktree", "add", prOpenPath, commit)

	_ = mustRunInDir(t, tmpDir, binaryPath, "clean", "--yes", "--work=false")

	if _, err := os.Stat(prClosedPath); !os.IsNotExist(err) {
		t.Fatalf("Expected closed PR worktree to be removed at %s", prClosedPath)
	}
	if _, err := os.Stat(prOpenPath); err != nil {
		t.Fatalf("Expected open PR worktree to remain at %s: %v", prOpenPath, err)
	}

	// Cleanup for safety in case the test fails mid-way.
	_, _ = runInDir(t, tmpDir, "git", "worktree", "remove", prOpenPath)
}

func TestRemoveAndCleanReviewModeWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	mustRunInDir(t, tmpDir, "git", "init")
	mustRunInDir(t, tmpDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, tmpDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, tmpDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "Initial commit")
	baseCommit := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "HEAD"))

	// The PR adds a file, which shows up as untracked in review mode.
	if err := os.WriteFile(filepath.Join(tmpDir, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, tmpDir, "git", "add", ".")
	mustRunInDir(t, tmpDir, "git", "commit", "-m", "PR commit")
	headCommit := strings.TrimSpace(mustRunInDir(t, tmpDir, "git", "rev-parse", "HEAD"))
	mustRunInDir(t, tmpDir, "git", "reset", "--hard", baseCommit)

	// Provide a stub gh command that reports every PR as closed.
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ghScript := "#!/usr/bin/env bash\necho '{\"state\":\"CLOSED\"}'\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(ghScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set up review mode by hand the way `gh worktree pr` does, since that needs GitHub.
	createReviewWorktree := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		mustRunInDir(t, tmpDir, "git", "worktree", "add", path, headCommit)
		mustRunInDir(t, path, "git", "reset", "--mixed", baseCommit)
		worktreeConfig := strings.TrimSpace(mustRunInDir(t, path, "git", "rev-parse", "--git-path", "config.worktree"))
		mustRunInDir(t, path, "git", "config", "--file", worktreeConfig, "gh-worktree.review.enabled", "true")
		mustRunInDir(t, path, "git", "config", "--file", worktreeConfig, "gh-worktree.review.head", headCommit)
	}

	removePath := filepath.Join(tmpDir, ".worktree", "pr", "pr-123")
	cleanPath := filepath.Join(tmpDir, ".worktree", "pr", "pr-456")
	createReviewWorktree(removePath)
	createReviewWorktree(cleanPath)

	_ = mustRunInDir(t, tmpDir, binaryPath, "remove", "123")
	if _, err := os.Stat(removePath); !os.IsNotExist(err) {
		t.Fatalf("Expected review mode worktree to be removed at %s", removePath)
	}

	_ = mustRunInDir(t, tmpDir, binaryPath, "clean", "--yes", "--work=false")
	if _, err := os.Stat(cleanPath); !os.IsNotExist(err) {
		t.Fatalf("Expected closed PR worktree in review mode to be removed at %s", cleanPath)
	}
}

func TestSyncPRWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	remoteDir := filepath.Join(tmpDir, "remote.git")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	mustRunInDir(t, tmpDir, "git", "init", "--bare", remoteDir)

	mustRunInDir(t, repoDir, "git", "init")
	mustRunInDir(t, repoDir, "git", "config", "user.email", "test@example.com")
	mustRunInDir(t, repoDir, "git", "config", "user.name", "Test User")
	mustRunInDir(t, repoDir, "git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Repo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, repoDir, "git", "add", ".")
	mustRunInDir(t, repoDir, "git", "commit", "-m", "Initial commit")

	baseBranch := strings.TrimSpace(mustRunInDir(t, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	commitA := strings.TrimSpace(mustRunInDir(t, repoDir, "git", "rev-parse", "HEAD"))

	mustRunInDir(t, repoDir, "git", "remote", "add", "origin", remoteDir)
	mustRunInDir(t, repoDir, "git", "push", "-u", "origin", baseBranch)

	mustRunInDir(t, repoDir, "git", "checkout", "-b", "feature/pr-123")
	if err := os.WriteFile(filepath.Join(repoDir, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, repoDir, "git", "add", ".")
	mustRunInDir(t, repoDir, "git", "commit", "-m", "PR commit")
	commitB := strings.TrimSpace(mustRunInDir(t, repoDir, "git", "rev-parse", "HEAD"))

	mustRunInDir(t, repoDir, "git", "push", "origin", "HEAD:refs/pull/123/head")
	mustRunInDir(t, repoDir, "git", "checkout", baseBranch)

	binaryPath := filepath.Join(repoDir, binaryName)
	if err := copyFile(binaryName, binaryPath, 0o755); err != nil {
		t.Fatal(err)
	}

	prWorktreePath := filepath.Join(repoDir, ".worktree", "pr", "pr-123")
	if err := os.MkdirAll(filepath.Dir(prWorktreePath), 0755); err != nil {
		t.Fatal(err)
	}
	mustRunInDir(t, repoDir, "git", "worktree", "add", prWorktreePath, commitA)

	_ = mustRunInDir(t, repoDir, binaryPath, "sync", "123")

	head := strings.TrimSpace(mustRunInDir(t, prWorktreePath, "git", "rev-parse", "HEAD"))
	if head != commitB {
		t.Fatalf("Expected synced HEAD %s, got %s", commitB, head)
	}

	_, _ = runInDir(t, repoDir, "git", "worktree", "remove", prWorktreePath)
}

func mustRunInDir(t *testing.T, dir string, command string, args ...string) string {
	t.Helper()
	output, err := runInDir(t, dir, command, args...)
	if err != nil {
		t.Fatalf("Command failed: %s %s\nError: %v\nOutput:\n%s", command, strings.Join(args, " "), err, output)
	}
	return output
}

func runInDir(t *testing.T, dir string, command string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
