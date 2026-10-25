package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetRemoteURL returns the URL for the specified git remote
func GetRemoteURL(remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	cmd := exec.Command("git", "remote", "get-url", remoteName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL for %s: %w", remoteName, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ParseGitHubRepo extracts owner/repo from a GitHub URL
func ParseGitHubRepo(remoteURL string) (string, error) {
	// Remove .git suffix if present
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Handle HTTPS URLs: https://github.com/owner/repo
	httpsRegex := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)`)
	if matches := httpsRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	// Handle SSH URLs: git@github.com:owner/repo
	sshRegex := regexp.MustCompile(`git@github\.com:([^/]+)/([^/]+)`)
	if matches := sshRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	// Handle SSH URLs with ssh:// prefix: ssh://git@github.com/owner/repo
	sshPrefixRegex := regexp.MustCompile(`ssh://git@github\.com/([^/]+)/([^/]+)`)
	if matches := sshPrefixRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	return "", fmt.Errorf("could not parse GitHub repository from URL: %s", remoteURL)
}

// GetRepoFromRemote gets the GitHub repository (owner/repo) from a git remote
func GetRepoFromRemote(remoteName string) (string, error) {
	url, err := GetRemoteURL(remoteName)
	if err != nil {
		return "", err
	}

	return ParseGitHubRepo(url)
}

// GetDefaultRemote returns the default remote (usually origin)
func GetDefaultRemote() string {
	// Check if origin exists
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if err := cmd.Run(); err == nil {
		return "origin"
	}

	// If origin doesn't exist, get the first remote
	cmd = exec.Command("git", "remote")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(remotes) > 0 && remotes[0] != "" {
		return remotes[0]
	}

	return ""
}
