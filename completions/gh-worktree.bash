#!/usr/bin/env bash

# Bash completion for gh-worktree

_gh_worktree_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="pr new list ls clean remove rm cd sync config help"

    case "${prev}" in
        worktree|gh-worktree)
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
        pr|new|clean|remove|rm|cd)
            # Arg completion could be smarter (PR numbers, branches, etc.)
            # but that would require git/GitHub access.
            return 0
            ;;
    esac
}

# Register the completion function for gh worktree
complete -F _gh_worktree_completions gh-worktree
