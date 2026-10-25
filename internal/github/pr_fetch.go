package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func (c *Client) FetchPRWithRepo(prNumber string, repo string, remoteName string) (*PRData, error) {
	// Use gh to fetch the PR metadata including the commit SHA and base info
	args := []string{"pr", "view", prNumber, "--json", "headRefName,headRepository,headRepositoryOwner,headRefOid,baseRefName,baseRefOid"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("failed to get PR information: %s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("failed to get PR information: %w", err)
	}

	// Parse the JSON to get the commit SHA and branch info
	var prInfo struct {
		HeadRefOid          string `json:"headRefOid"`
		HeadRefName         string `json:"headRefName"`
		HeadRepositoryOwner struct {
			Login string `json:"login"`
		} `json:"headRepositoryOwner"`
		HeadRepository struct {
			Name string `json:"name"`
		} `json:"headRepository"`
		BaseRefName string `json:"baseRefName"`
		BaseRefOid  string `json:"baseRefOid"`
	}

	if err := json.Unmarshal(output, &prInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PR information: %w", err)
	}

	// Determine which remote to use
	remote := remoteName
	if remote == "" {
		// Try to find a remote that matches the PR's repository
		remoteCmd := exec.Command("git", "remote", "-v")
		remoteOutput, err := remoteCmd.Output()
		if err == nil {
			lines := strings.Split(string(remoteOutput), "\n")
			targetRepo := fmt.Sprintf("%s/%s", prInfo.HeadRepositoryOwner.Login, prInfo.HeadRepository.Name)

			for _, line := range lines {
				parts := strings.Fields(line)
				if len(parts) >= 2 && strings.Contains(parts[1], targetRepo) {
					remote = parts[0]
					break
				}
			}
		}

		// Default to origin if we couldn't find a matching remote
		if remote == "" {
			remote = "origin"
		}
	}

	// Fetch the PR ref using the determined remote
	refSpec := fmt.Sprintf("+refs/pull/%s/head:refs/remotes/%s/pr/%s", prNumber, remote, prNumber)
	fetchCmd := exec.Command("git", "fetch", remote, refSpec)
	var fetchStderr bytes.Buffer
	fetchCmd.Stderr = &fetchStderr

	if err := fetchCmd.Run(); err != nil {
		// If fetch with PR ref fails, try fetching the branch directly
		fetchCmd = exec.Command("git", "fetch", remote, prInfo.HeadRefName)
		fetchCmd.Stderr = &fetchStderr

		if err := fetchCmd.Run(); err != nil {
			errMsg := fetchStderr.String()
			if errMsg != "" {
				return nil, fmt.Errorf("failed to fetch PR: %s", strings.TrimSpace(errMsg))
			}
			return nil, fmt.Errorf("failed to fetch PR: %w", err)
		}
	}

	baseRepo := ""
	if repo != "" {
		baseRepo = repo
	}

	return &PRData{
		CommitSHA:   prInfo.HeadRefOid,
		RefName:     prInfo.HeadRefName,
		BaseRepo:    baseRepo,
		BaseRefName: prInfo.BaseRefName,
	}, nil
}
