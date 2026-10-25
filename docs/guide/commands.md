# Commands

A number given as a target means a PR, anything else a branch name. `gh worktree help` prints a summary of all commands, and most commands accept `--help`.

## pr

```
gh worktree pr [<number>] [-r <remote>] [--cd] [--reuse] [--no-review]
```

Creates a worktree for a pull request at `<base>/pr/pr-<number>`, or `<base>/pr/<remote>/pr-<number>` with `-r`.

| Option | Description |
|---|---|
| `-r`, `--remote <name>` | Take the PR from the repository behind this remote. |
| `--cd` | Print only the path of the worktree. Needs a PR number. |
| `--reuse` | If a worktree for the PR exists, use it instead of failing. |
| `--no-review` | Check out the PR head without [review mode](./review-mode). |

Without a number, up to 30 open PRs are listed to pick from. This needs a terminal.

## new

```
gh worktree new [<branch>] [-b <base>] [--cd] [--reuse]
```

Creates a branch and a worktree for it at `<base>/work/<branch>`.

| Option | Description |
|---|---|
| `-b`, `--base <branch>` | Branch to start from. Defaults to the default branch. |
| `--cd` | Print only the path of the worktree. |
| `--reuse` | If a worktree for the branch exists, use it instead of failing. |

Without a branch name, it asks for one. This needs a terminal.

## list

```
gh worktree list
```

Alias: `ls`. Lists the PR worktrees and branch worktrees under the base directory.

```
PR Worktrees:
PR         REMOTE     BRANCH               COMMIT          PATH
---        ------     ------               ------          ----
#123       -          (detached)           72d59db         .worktree/pr/pr-123
#456       upstream   (detached)           72d59db         .worktree/pr/upstream/pr-456

Work Worktrees:
BRANCH                         COMMIT          PATH
------                         ------          ----
feature/awesome                72d59db         .worktree/work/feature/awesome
```

## cd

```
gh worktree cd [<target>] [-r <remote>] [--work <branch>]
```

Prints the path of a worktree, for use as `cd "$(gh worktree cd 123)"`.

| Option | Description |
|---|---|
| `-r`, `--remote <name>` | Use the PR worktree for this remote. |
| `--work <branch>` | Treat the target as a branch name, for branches whose names are numbers. |

Without a target, the worktrees are listed to pick from. The list is drawn on stderr, so only the chosen path ends up in `$(...)`. This needs a terminal.

## sync

```
gh worktree sync <number> [-r <remote>] [--force]
```

Fetches `refs/pull/<number>/head` again and checks out the new head in the PR's worktree. The PR is fetched from the remote given with `-r`, otherwise from the remote in the worktree's path, otherwise from `origin` (or the first remote).

| Option | Description |
|---|---|
| `-r`, `--remote <name>` | Select the worktree and the remote to fetch from. |
| `--force` | Discard local changes in the worktree. |

See [following new commits](./review-mode#following-new-commits) for how this works in review mode.

## remove

```
gh worktree remove <target> [-r <remote>] [--work <branch>]
```

Alias: `rm`. Removes a worktree with `git worktree remove`. The branch is kept.

| Option | Description |
|---|---|
| `-r`, `--remote <name>` | Remove the PR worktree for this remote. |
| `--work <branch>` | Treat the target as a branch name. |

A worktree with local changes is not removed. Commit, stash, or discard them first. In review mode, the PR's changes do not count as local changes. If worktrees for the same PR number exist for more than one remote, `remove` and `sync` ask you to pass `-r`.

## clean

```
gh worktree clean [--pr=false] [--work=false] [--dry-run] [-y]
```

Runs `git worktree prune`, then removes:

- PR worktrees whose PR is no longer open, as reported by `gh pr view`,
- branch worktrees whose branch is merged into the default branch, that is, an ancestor of the local default branch, or of `origin/<default branch>` when there is no local one.

Worktrees with local changes are skipped and listed. Branches are kept. A branch merged by squash or rebase is not an ancestor of the default branch, so `clean` does not see it as merged.

| Option | Description |
|---|---|
| `--pr=false` | Leave PR worktrees alone. |
| `--work=false` | Leave branch worktrees alone. |
| `--dry-run` | Only list what would be removed. |
| `-y`, `--yes` | Do not ask for confirmation. Required without a terminal. |

## config

```
gh worktree config
```

Shows the configuration in effect and where it came from. See [configuration](./configuration).

## Without a terminal

When stdin is not a terminal, nothing is prompted for: `pr` needs a number, `new` a branch name, `cd` a target, and `clean` needs `--yes`.
