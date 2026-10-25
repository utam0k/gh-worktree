package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorktreeBasePath string   `yaml:"worktree_base_path"`
	SymlinkFiles     []string `yaml:"symlink_files"`
}

func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func findProjectConfigPath() (string, bool) {
	const configName = ".gh-worktree.yml"

	gitRoot, err := findGitRoot()
	if err != nil {
		if _, err := os.Stat(configName); err == nil {
			return configName, true
		}
		return "", false
	}

	cwd, err := os.Getwd()
	if err != nil {
		configPath := filepath.Join(gitRoot, configName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, true
		}
		return "", false
	}

	cwd = filepath.Clean(cwd)
	gitRoot = filepath.Clean(gitRoot)
	dir := cwd

	for {
		configPath := filepath.Join(dir, configName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, true
		}

		if dir == gitRoot {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", false
}

func Load() (*Config, error) {
	cfg := &Config{
		WorktreeBasePath: ".worktree",
	}

	// Try to load project config (.gh-worktree.yml) from current directory up to git root.
	if configPath, ok := findProjectConfigPath(); ok {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		// Try to load from user config only if project config doesn't exist
		if homeDir, err := os.UserHomeDir(); err == nil {
			configPath := filepath.Join(homeDir, ".config", "gh-worktree", "config.yml")
			if data, err := os.ReadFile(configPath); err == nil {
				if err := yaml.Unmarshal(data, cfg); err != nil {
					return nil, err
				}
			}
		}
	}

	// Environment variable always takes priority
	if envPath := os.Getenv("GH_WORKTREE_BASE_PATH"); envPath != "" {
		cfg.WorktreeBasePath = envPath
	}

	return cfg, nil
}

func GetWorktreeBasePath() string {
	cfg, err := Load()
	if err != nil {
		return ".worktree"
	}
	return cfg.WorktreeBasePath
}
