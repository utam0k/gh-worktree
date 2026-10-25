package review

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableDisableReview(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "commit.gpgsign", "false")

	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base file: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "base")

	baseSHA := mustGitOutput(t, dir, "rev-parse", "HEAD")
	runGit(t, dir, "branch", "base", baseSHA)

	if err := os.WriteFile(filePath, []byte("head\n"), 0644); err != nil {
		t.Fatalf("write head file: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "head")

	headSHA := mustGitOutput(t, dir, "rev-parse", "HEAD")

	info, err := Enable(dir, "123", headSHA, "owner/repo", "main", "refs/heads/base")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	gotHead := mustGitOutput(t, dir, "rev-parse", "HEAD")
	if gotHead != baseSHA {
		t.Fatalf("HEAD = %s, want %s", gotHead, baseSHA)
	}

	status := mustGitOutput(t, dir, "status", "--porcelain")
	if strings.TrimSpace(status) == "" {
		t.Fatalf("expected dirty status after review enable")
	}

	reviewOnly, err := IsReviewOnlyDirty(dir, info)
	if err != nil {
		t.Fatalf("IsReviewOnlyDirty failed: %v", err)
	}
	if !reviewOnly {
		t.Fatalf("expected review-only dirty state")
	}

	if err := os.WriteFile(filePath, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("write modified file: %v", err)
	}

	reviewOnly, err = IsReviewOnlyDirty(dir, info)
	if err != nil {
		t.Fatalf("IsReviewOnlyDirty after edit failed: %v", err)
	}
	if reviewOnly {
		t.Fatalf("expected non-review dirty state after edit")
	}

	if err := Disable(dir); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	gotHead = mustGitOutput(t, dir, "rev-parse", "HEAD")
	if gotHead != headSHA {
		t.Fatalf("HEAD after disable = %s, want %s", gotHead, headSHA)
	}

	if _, ok, err := Read(dir); err != nil {
		t.Fatalf("Read failed: %v", err)
	} else if ok {
		t.Fatalf("expected review config to be cleared")
	}
}

func TestIsReviewOnlyDirtyWithAddedAndDeletedFiles(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(dir, "removed.txt"), []byte("base\n"), 0644); err != nil {
		t.Fatalf("write base file: %v", err)
	}
	runGit(t, dir, "add", "removed.txt")
	runGit(t, dir, "commit", "-m", "base")
	baseSHA := mustGitOutput(t, dir, "rev-parse", "HEAD")

	// The PR deletes one file and adds another.
	runGit(t, dir, "rm", "removed.txt")
	addedPath := filepath.Join(dir, "added.txt")
	if err := os.WriteFile(addedPath, []byte("head\n"), 0644); err != nil {
		t.Fatalf("write added file: %v", err)
	}
	runGit(t, dir, "add", "added.txt")
	runGit(t, dir, "commit", "-m", "head")
	headSHA := mustGitOutput(t, dir, "rev-parse", "HEAD")

	info, err := Enable(dir, "123", headSHA, "owner/repo", "main", baseSHA)
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	assertReviewOnly := func(want bool, msg string) {
		t.Helper()
		got, err := IsReviewOnlyDirty(dir, info)
		if err != nil {
			t.Fatalf("IsReviewOnlyDirty failed: %v", err)
		}
		if got != want {
			t.Fatalf("%s: IsReviewOnlyDirty = %v, want %v", msg, got, want)
		}
	}

	assertReviewOnly(true, "untouched review worktree")

	strayPath := filepath.Join(dir, "stray.txt")
	if err := os.WriteFile(strayPath, []byte("local\n"), 0644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}
	assertReviewOnly(false, "with an extra local file")
	if err := os.Remove(strayPath); err != nil {
		t.Fatalf("remove stray file: %v", err)
	}

	if err := os.Remove(addedPath); err != nil {
		t.Fatalf("remove added file: %v", err)
	}
	assertReviewOnly(false, "with a file added by the PR deleted locally")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := gitRun(dir, args...); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

func mustGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitOutput(dir, args...)
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(out)
}
