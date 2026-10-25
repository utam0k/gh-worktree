package worktree

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/utam0k/gh-worktree/internal/config"
	"github.com/utam0k/gh-worktree/internal/review"
)

func NewManager() *Manager {
	gitRoot, _ := getGitRoot()
	basePath := config.GetWorktreeBasePath()
	if basePath == "" {
		basePath = ".worktree"
	}
	return &Manager{gitRoot: gitRoot, basePath: basePath}
}

type Manager struct {
	gitRoot  string
	basePath string
}

func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *Manager) Create(prNumber string, commitHash string) error {
	return m.createAtPath(m.getWorktreePath(prNumber), commitHash)
}

func (m *Manager) CreateWithRemote(prNumber string, commitHash string, remoteName string) error {
	return m.createAtPath(m.getWorktreePathWithRemote(prNumber, remoteName), commitHash)
}

func (m *Manager) createAtPath(worktreePath string, commitHash string) error {
	cmd := exec.Command("git", "worktree", "add", worktreePath, commitHash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("failed to create worktree: %s", msg)
		}
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	if err := m.createSymlinks(worktreePath); err != nil {
		return fmt.Errorf("failed to create symlinks: %w", err)
	}

	return nil
}

type WorktreeInfo struct {
	PRNumber    string
	Remote      string
	Path        string // Absolute path
	DisplayPath string // Relative path for display
	Branch      string
	Commit      string
	Type        string // "pr" or "work"
}

func (m *Manager) List() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	basePath := m.basePath

	// Regex patterns for different worktree types
	// Match both old (.git/worktrees) and new (.worktree) paths
	prRegex := regexp.MustCompile(`(?:worktrees|` + regexp.QuoteMeta(basePath) + `)/pr/(?:([^/]+)/)?pr-(\d+)`)
	workRegex := regexp.MustCompile(`(?:worktrees|` + regexp.QuoteMeta(basePath) + `)/work/(.+)`)

	entries := strings.Split(string(output), "\n\n")
	for _, entry := range entries {
		if entry == "" {
			continue
		}

		lines := strings.Split(entry, "\n")
		if len(lines) < 3 {
			continue
		}

		// Parse worktree path
		path := strings.TrimPrefix(lines[0], "worktree ")

		// Parse commit hash
		commit := strings.TrimPrefix(lines[1], "HEAD ")
		if len(commit) > 7 {
			commit = commit[:7]
		}

		// Parse branch name
		branch := ""
		if len(lines) >= 3 && strings.HasPrefix(lines[2], "branch ") {
			branch = strings.TrimPrefix(lines[2], "branch refs/heads/")
		}

		// Make path relative for display
		displayPath := path
		if m.gitRoot != "" {
			if rel, err := filepath.Rel(m.gitRoot, path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				displayPath = rel
			}
		} else if cwd, err := os.Getwd(); err == nil && strings.HasPrefix(path, cwd) {
			displayPath = strings.TrimPrefix(path, cwd+string(os.PathSeparator))
		}

		// Check if it's a PR worktree
		if matches := prRegex.FindStringSubmatch(path); len(matches) >= 3 {
			remote := matches[1] // May be empty for old format
			prNumber := matches[2]

			worktrees = append(worktrees, WorktreeInfo{
				PRNumber:    prNumber,
				Remote:      remote,
				Path:        path,
				DisplayPath: displayPath,
				Branch:      branch,
				Commit:      commit,
				Type:        "pr",
			})
		} else if matches := workRegex.FindStringSubmatch(path); len(matches) >= 2 {
			// It's a work branch worktree
			branchName := matches[1]

			worktrees = append(worktrees, WorktreeInfo{
				PRNumber:    "",
				Remote:      "",
				Path:        path,
				DisplayPath: displayPath,
				Branch:      branchName,
				Commit:      commit,
				Type:        "work",
			})
		}
	}

	return worktrees, nil
}

func (m *Manager) Remove(prNumber string) error {
	return m.removeAtPath(m.getWorktreePath(prNumber))
}

func (m *Manager) RemoveWithRemote(prNumber string, remoteName string) error {
	return m.removeAtPath(m.getWorktreePathWithRemote(prNumber, remoteName))
}

func (m *Manager) RemovePath(worktreePath string) error {
	return m.removeAtPath(worktreePath)
}

func (m *Manager) removeAtPath(worktreePath string) error {
	// git refuses to remove a dirty worktree, and review mode makes the PR's
	// changes look like local edits. Leave review mode first if that is all there is.
	if info, ok, err := review.Read(worktreePath); err == nil && ok {
		if reviewOnly, err := review.IsReviewOnlyDirty(worktreePath, info); err == nil && reviewOnly {
			if err := review.Disable(worktreePath); err != nil {
				return fmt.Errorf("failed to leave review mode: %w", err)
			}
		}
	}

	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("failed to remove worktree: %s", msg)
		}
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

func (m *Manager) GetWorktreePath(prNumber string) string {
	return m.getWorktreePath(prNumber)
}

func (m *Manager) GetWorktreePathWithRemote(prNumber string, remoteName string) string {
	return m.getWorktreePathWithRemote(prNumber, remoteName)
}

func (m *Manager) getWorktreePath(prNumber string) string {
	if m.gitRoot == "" {
		return filepath.Join(m.basePath, "pr", fmt.Sprintf("pr-%s", prNumber))
	}
	return filepath.Join(m.gitRoot, m.basePath, "pr", fmt.Sprintf("pr-%s", prNumber))
}

func (m *Manager) getWorktreePathWithRemote(prNumber string, remoteName string) string {
	if remoteName == "" {
		remoteName = "origin"
	}
	if m.gitRoot == "" {
		return filepath.Join(m.basePath, "pr", remoteName, fmt.Sprintf("pr-%s", prNumber))
	}
	return filepath.Join(m.gitRoot, m.basePath, "pr", remoteName, fmt.Sprintf("pr-%s", prNumber))
}

// CreateWorkBranch creates a new worktree for development work
func (m *Manager) CreateWorkBranch(branchName string, baseBranch string) error {
	worktreePath := m.getWorkBranchPath(branchName)

	// Create worktree with new branch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	// Create symlinks for configured files
	if err := m.createSymlinks(worktreePath); err != nil {
		return fmt.Errorf("failed to create symlinks: %w", err)
	}

	return nil
}

// GetWorkBranchPath returns the path for a work branch worktree
func (m *Manager) GetWorkBranchPath(branchName string) string {
	return m.getWorkBranchPath(branchName)
}

func (m *Manager) getWorkBranchPath(branchName string) string {
	if m.gitRoot == "" {
		return filepath.Join(m.basePath, "work", branchName)
	}
	return filepath.Join(m.gitRoot, m.basePath, "work", branchName)
}

// createSymlinks creates symbolic links for configured files
func (m *Manager) createSymlinks(worktreePath string) error {
	cfg, err := config.Load()
	if err != nil {
		// The base path already falls back to the default on an unreadable
		// config (see config.GetWorktreeBasePath); likewise create no symlinks
		// rather than fail after the worktree has been created.
		return nil
	}

	// If no symlink files configured, return early
	if len(cfg.SymlinkFiles) == 0 {
		return nil
	}

	mainRoot := m.gitRoot
	if mainRoot == "" {
		root, err := getGitRoot()
		if err != nil {
			return fmt.Errorf("failed to get git root: %w", err)
		}
		mainRoot = root
	}

	var skippedFiles []string
	var createdLinks []string

	for _, file := range cfg.SymlinkFiles {
		sourcePath := filepath.Join(mainRoot, file)
		targetPath := filepath.Join(worktreePath, file)

		// Check if source file exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// Skip if source doesn't exist
			continue
		}

		// Create parent directory if needed
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file, err)
		}

		// Check if target exists
		targetStat, err := os.Lstat(targetPath)
		if err == nil {
			// Target exists, need to handle it
			action, err := handleExistingFile(targetPath, sourcePath, targetStat)
			if err != nil {
				return err
			}

			switch action {
			case "skip":
				skippedFiles = append(skippedFiles, file)
				continue
			case "replace":
				// Remove the existing file
				if err := os.RemoveAll(targetPath); err != nil {
					return fmt.Errorf("failed to remove existing file %s: %w", file, err)
				}
			}
		}

		// Create symlink
		if err := os.Symlink(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", file, err)
		}
		createdLinks = append(createdLinks, file)
	}

	// Report results on stderr so that `--cd` prints nothing but the path on stdout
	if len(createdLinks) > 0 {
		fmt.Fprintln(os.Stderr, "\nCreated symlinks:")
		for _, file := range createdLinks {
			fmt.Fprintf(os.Stderr, "  ✓ %s\n", file)
		}
	}

	if len(skippedFiles) > 0 {
		fmt.Fprintln(os.Stderr, "\nSkipped files (already exist):")
		for _, file := range skippedFiles {
			fmt.Fprintf(os.Stderr, "  - %s\n", file)
		}
	}

	return nil
}

// handleExistingFile prompts the user about what to do with an existing file
func handleExistingFile(targetPath, sourcePath string, targetStat os.FileInfo) (string, error) {
	// Check if it's already a symlink to the same target
	if targetStat.Mode()&os.ModeSymlink != 0 {
		currentTarget, err := os.Readlink(targetPath)
		if err == nil && currentTarget == sourcePath {
			// Already correct symlink, skip silently
			return "skip", nil
		}
	}

	// Display file information
	fmt.Fprintf(os.Stderr, "\nFound existing file at: %s\n", targetPath)

	fileType := "Regular file"
	if targetStat.IsDir() {
		fileType = "Directory"
	} else if targetStat.Mode()&os.ModeSymlink != 0 {
		fileType = "Symbolic link"
		if target, err := os.Readlink(targetPath); err == nil {
			fileType = fmt.Sprintf("Symbolic link -> %s", target)
		}
	}

	fmt.Fprintf(os.Stderr, "Type: %s", fileType)
	if !targetStat.IsDir() {
		fmt.Fprintf(os.Stderr, " (%d bytes)", targetStat.Size())
	}
	fmt.Fprintf(os.Stderr, ", modified %s\n", targetStat.ModTime().Format("2006-01-02 15:04:05"))

	// Interactive prompt
	fmt.Fprintln(os.Stderr, "\nWhat would you like to do?")
	fmt.Fprintln(os.Stderr, "  [s] Skip this file (default)")
	fmt.Fprintln(os.Stderr, "  [r] Replace with symlink")
	fmt.Fprint(os.Stderr, "\nChoice [s/r]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(strings.ToLower(input))

	switch choice {
	case "r":
		return "replace", nil
	case "s", "":
		return "skip", nil
	default:
		fmt.Fprintln(os.Stderr, "Invalid choice, skipping file")
		return "skip", nil
	}
}
