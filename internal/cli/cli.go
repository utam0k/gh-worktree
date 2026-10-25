package cli

import (
	"fmt"
	"os"
)

func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return fmt.Errorf("no command specified")
	}

	command := os.Args[1]

	switch command {
	case "pr":
		return executePR()
	case "new":
		return executeNew()
	case "list", "ls":
		return executeList()
	case "clean":
		return executeClean()
	case "remove", "rm":
		return executeRemove()
	case "cd":
		return executeCd()
	case "sync":
		return executeSync()
	case "config":
		return executeConfig()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println(`gh-worktree - Manage git worktrees for GitHub PR reviews

Usage:
  gh worktree <command> [arguments]

Commands:
  pr [pr-number]             Create a worktree for PR review (interactive if omitted)
    -r, --remote <name>        Specify git remote name (e.g., origin, upstream)
    --cd                       Output only the path (non-interactive)
    --reuse                    Reuse existing worktree if it already exists
    --no-review                Disable review mode (default: enabled)
  new [branch-name]          Create a new worktree for development work
    -b, --base <branch>        Base branch to create from (default: main/master)
    --cd                       Output path for cd after creation
    --reuse                    Reuse existing worktree if it already exists
  list, ls                   List all worktrees
  clean                      Find and remove unnecessary worktrees
  remove, rm <target>        Remove a worktree (PR number or work branch name)
    -r, --remote <name>        Specify git remote name (optional)
    --work <branch>            Specify a work branch name explicitly
  cd [target]                Output the path to cd into a worktree (interactive if omitted)
    -r, --remote <name>        Specify git remote name (optional)
    --work <branch>            Specify a work branch name explicitly
  sync <pr-number>           Fetch and update an existing PR worktree
    -r, --remote <name>        Specify git remote name (optional)
    --force                    Discard local changes in the worktree
  config                     Show current configuration
  help                       Show this help message

Examples:
  gh worktree pr 123                        Create a worktree for PR #123
  gh worktree pr                            Select a PR interactively
  gh worktree pr -r upstream 123            Create from upstream remote
  gh worktree pr --cd 123                   Create and output path for cd
  gh worktree pr --reuse --cd 123           Reuse existing and output path
  gh worktree new feature/awesome           Create worktree for new feature branch
  gh worktree new -b develop hotfix/urgent  Create hotfix from develop branch
  gh worktree new --cd feature/new          Create and output path for cd
  gh worktree new --reuse --cd feature/new  Reuse existing and output path
  gh worktree list                          List all worktrees
  gh worktree clean --dry-run               Show removable worktrees
  gh worktree remove 123                    Remove the worktree for PR #123
  gh worktree remove -r upstream 123        Remove the worktree for PR #123 from upstream
  gh worktree remove feature/my-branch      Remove the worktree for a work branch
  cd $(gh worktree cd 123)                  Change directory to PR #123 worktree
  cd $(gh worktree cd feature/my-branch)    Change directory to a work branch worktree
  cd $(gh worktree cd)                      Select worktree interactively and cd
  gh worktree sync 123                      Sync an existing PR worktree

Shell Integration:
  Add this function to your shell profile (~/.bashrc or ~/.zshrc):
  
  # gh workspace cd support
  gh() {
    if [ "$1" = "workspace" ] && [ "$2" = "cd" ]; then
      shift 2
      local dir
      dir=$(command gh worktree cd "$@") && cd "$dir"
    else
      command gh "$@"
    fi
  }
  
  Then use:
    gh workspace cd 123    # cd to PR #123
    gh workspace cd        # Select interactively`)
}
