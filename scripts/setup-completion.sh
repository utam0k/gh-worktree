#!/usr/bin/env bash

# Setup script for gh-worktree command completion

set -e

# Detect shell
SHELL_NAME="$(basename "${SHELL:-}")"
if [ "$SHELL_NAME" = "zsh" ]; then
    SHELL_NAME="zsh"
elif [ "$SHELL_NAME" = "bash" ]; then
    SHELL_NAME="bash"
else
    echo "Unsupported shell. Only bash and zsh are supported."
    exit 1
fi

echo "Setting up gh-worktree completion for $SHELL_NAME..."

# Setup for gh extension completion
# gh uses a special completion system that hooks into the shell's completion
case "$SHELL_NAME" in
    bash)
        # For bash, we need to hook into gh's completion system
        cat << 'EOF'
To enable completion for gh-worktree, add the following to your ~/.bashrc:

# gh-worktree completion
if command -v gh &> /dev/null; then
    eval "$(gh completion -s bash)"
    # Additional completion for gh worktree subcommands
    _gh_worktree() {
        local cur="${COMP_WORDS[COMP_CWORD]}"
        local prev="${COMP_WORDS[COMP_CWORD-1]}"
        
        if [[ "$prev" == "gh-worktree" ]]; then
            COMPREPLY=( $(compgen -W "pr new list ls clean remove rm cd sync config help" -- "$cur") )
        fi
    }
    complete -F _gh_worktree gh-worktree
fi
EOF
        ;;
    zsh)
        # For zsh, we need to hook into gh's completion system
        cat << 'EOF'
To enable completion for gh-worktree, add the following to your ~/.zshrc:

# gh-worktree completion
if command -v gh &> /dev/null; then
    eval "$(gh completion -s zsh)"
    # Additional completion for gh worktree subcommands
    compdef '_gh_worktree() {
        local -a commands
        commands=(
            "pr:Create a worktree for PR review"
            "new:Create a new worktree for development work"
            "list:List all worktrees"
            "ls:List all worktrees"
            "clean:Find and remove unnecessary worktrees"
            "remove:Remove the worktree for the specified PR"
            "rm:Remove the worktree for the specified PR"
            "cd:Output the path to cd into the worktree"
            "sync:Fetch and update an existing PR worktree"
            "config:Show current configuration"
            "help:Show help message"
        )
        
        if (( CURRENT == 2 )); then
            _describe -t commands "gh-worktree commands" commands
        fi
    }' gh-worktree
fi
EOF
        ;;
esac

echo ""
echo "After adding the above to your shell configuration, restart your shell or run:"
echo "  source ~/.${SHELL_NAME}rc"
