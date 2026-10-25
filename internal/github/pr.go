package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type PR struct {
	HeadRefName    string `json:"headRefName"`
	HeadRepository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"headRepository"`
}

type PRListItem struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	State string `json:"state"`
}

type PRData struct {
	CommitSHA   string
	RefName     string
	BaseRepo    string
	BaseRefName string
}

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GetPRWithRepo(prNumber string, repo string) (*PR, error) {
	args := []string{"pr", "view", prNumber, "--json", "headRefName,headRepository"}
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

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR information: %w", err)
	}

	return &pr, nil
}

func (c *Client) ListPRsWithRepo(repo string) ([]PRListItem, error) {
	args := []string{"pr", "list", "--json", "number,title,headRefName,author,state", "--limit", "30"}
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
			return nil, fmt.Errorf("failed to list PRs: %s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	var prs []PRListItem
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}

	return prs, nil
}

func (c *Client) GetPRStateWithRepo(prNumber string, repo string) (string, error) {
	args := []string{"pr", "view", prNumber, "--json", "state"}
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
			return "", fmt.Errorf("failed to get PR state: %s", strings.TrimSpace(errMsg))
		}
		return "", fmt.Errorf("failed to get PR state: %w", err)
	}

	var pr struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(output, &pr); err != nil {
		return "", fmt.Errorf("failed to parse PR state: %w", err)
	}

	return pr.State, nil
}
