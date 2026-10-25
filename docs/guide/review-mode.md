# Review mode

`gh worktree pr` creates worktrees in review mode unless you pass `--no-review`.

## What it changes

In a plain checkout of a PR, the files, the index, and `HEAD` are all at the PR head. `git status` is clean and your editor marks nothing.

In review mode, the files are still the PR head, but `HEAD` and the index are moved to the merge base of the PR and its base branch. That is the commit GitHub compares the PR against, so the difference between `HEAD` and the files is the PR's diff, and git reports it as uncommitted changes:

```console
$ cd "$(gh worktree cd 123)"
$ git status --short
 M README.md
 M internal/cli/cmd_pr.go
?? internal/cli/reuse.go
```

Files the PR modifies or deletes show as modified or deleted. Files the PR adds show as untracked, because the merge base does not have them.

`git diff` prints the PR's changes to existing files. Editors that mark changed lines against `HEAD` or the index, such as gitsigns.nvim or the gutter in VS Code, mark the lines the PR touches. Everything else works as in any checkout: you can build, run the tests, and jump to definitions.

## Your own edits

Review mode records the PR head it was set up from. `sync`, `clean`, and `remove` compare the files with that commit, so they can tell the PR's changes from yours:

- `sync` updates the worktree only if the files still match the recorded head. Otherwise it stops, and `--force` discards your edits.
- `clean` skips the worktree.
- `remove` refuses to remove it.

A new file counts as an edit unless git ignores it.

## Following new commits

```sh
gh worktree sync 123
```

`sync` puts the files back to the recorded head, fetches the PR again, checks out the new head, and sets up review mode again with a new merge base.

## Turning it off and on

```sh
gh worktree pr --no-review 123            # new worktree without review mode
gh worktree pr --reuse --no-review 123    # existing worktree: back to a plain checkout
gh worktree pr --reuse 123                # existing worktree: turn review mode on
```

Turning it on for an existing worktree looks up the PR's base branch on GitHub and uses the commit the worktree is on as the head. If the worktree has edits of your own, review mode is left as it is and a message says so.

## Forks and the base branch

The merge base is computed against the PR's base branch in the base repository. The base repository is the one behind the remote given with `-r`, or behind `origin` (or the first remote) without `-r`. If you work in a fork with the original repository as `upstream`, pass `-r upstream` so the merge base comes from there.

gh-worktree looks for a remote whose URL points at the base repository and uses `refs/remotes/<remote>/<branch>`, fetching it when it is missing. If no remote points there, it fetches the branch from `https://github.com/<owner>/<repo>.git` into `refs/gh-worktree/pr/<number>/base`.

## Where the state is kept

Review mode is recorded in the worktree's own config file, `config.worktree` in the worktree's git directory, so it does not touch the repository's shared config. To look at it from inside the worktree:

```console
$ git config --file "$(git rev-parse --git-path config.worktree)" --get-regexp '^gh-worktree\.review\.'
gh-worktree.review.enabled true
gh-worktree.review.pr 123
gh-worktree.review.head <sha of the PR head>
gh-worktree.review.base-repo owner/repo
gh-worktree.review.base-branch main
gh-worktree.review.base-ref refs/remotes/origin/main
gh-worktree.review.merge-base <sha of the merge base>
```
