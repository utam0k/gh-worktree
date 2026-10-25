#compdef gh-worktree

# Zsh completion for gh-worktree

_gh_worktree() {
    local -a commands
    commands=(
        'pr:Create a new worktree for the specified PR'
        'new:Create a new worktree for development work'
        'list:List all worktrees created for PRs'
        'ls:List all worktrees created for PRs'
        'clean:Find and remove unnecessary worktrees'
        'remove:Remove the worktree for the specified PR'
        'rm:Remove the worktree for the specified PR'
        'cd:Output the path to cd into a worktree'
        'sync:Fetch and update an existing PR worktree'
        'config:Show current configuration'
        'help:Show help message'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'gh-worktree commands' commands
    elif (( CURRENT == 3 )); then
        case "$words[2]" in
            pr|remove|rm|cd|sync)
                _message "PR number (optional for cd)"
                ;;
            new)
                _message "Branch name"
                ;;
        esac
    fi
}

_gh_worktree "$@"
