package git

import (
	"os/exec"
	"strings"
)

// GetDefaultBranch returns the default branch (main or master)
func GetDefaultBranch() string {
	// Try to get the default branch from origin
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		// Remove "origin/" prefix
		if strings.HasPrefix(branch, "origin/") {
			return strings.TrimPrefix(branch, "origin/")
		}
	}

	// Check if main exists
	cmd = exec.Command("git", "rev-parse", "--verify", "refs/heads/main")
	if err := cmd.Run(); err == nil {
		return "main"
	}

	// Check if master exists
	cmd = exec.Command("git", "rev-parse", "--verify", "refs/heads/master")
	if err := cmd.Run(); err == nil {
		return "master"
	}

	// Fallback to current branch
	return GetCurrentBranch()
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "main" // Fallback
	}

	return strings.TrimSpace(string(output))
}

// ListBranches returns all local branches
func ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	return branches, nil
}

// ListRemoteBranches returns all remote branches
func ListRemoteBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	return branches, nil
}
