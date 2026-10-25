<h1 align="center">
  <img src="docs/public/logo.svg" width="128" alt=""><br>
  gh-worktree
</h1>

A [GitHub CLI](https://cli.github.com/) extension that checks out each pull request you review and each branch you start into its own [git worktree](https://git-scm.com/docs/git-worktree). The checkout you are working in is left alone, so there is nothing to stash and no branch to switch.

`gh worktree pr 123` puts PR #123 in `.worktree/pr/pr-123`, and `gh worktree new feature/foo` creates the branch in `.worktree/work/feature/foo`. PR worktrees open in review mode: the PR's changes show up as uncommitted changes, so `git status`, `git diff`, and the change markers in your editor show what the PR changes. `sync` brings a PR worktree up to date with the PR, and `clean` removes worktrees of closed PRs and merged branches.

Documentation: https://utam0k.github.io/gh-worktree/

## Installation

```sh
gh extension install utam0k/gh-worktree
```

Requirements: `git`, plus `gh` logged in to GitHub (`gh auth login`).

Prebuilt binaries are published for Linux, macOS, Windows, and FreeBSD. Development and testing happen on Linux.

To upgrade:

```sh
gh extension upgrade worktree
```

## Where worktrees live

Worktrees are created under `.worktree/` at the top of your repository:

```
<repo>/
└── .worktree/
    ├── pr/
    │   ├── pr-123/              # gh worktree pr 123
    │   └── upstream/
    │       └── pr-456/          # gh worktree pr -r upstream 456
    └── work/
        └── feature/awesome/     # gh worktree new feature/awesome
```

The location can be changed; see [Configuration](#configuration).

## Usage

### Review a pull request

```sh
gh worktree pr 123                # create a worktree for PR #123
gh worktree pr                    # pick one of the open PRs interactively
gh worktree pr -r upstream 123    # PR #123 of the repository behind the "upstream" remote
gh worktree pr --reuse 123        # use the existing worktree if there is one
gh worktree pr --no-review 123    # plain checkout of the PR head (no review mode)
```

Without `-r`, `gh` chooses the repository the same way `gh pr view` does. With `-r <remote>`, the repository comes from that remote's URL and the worktree is placed under `pr/<remote>/`.

The PR is fetched directly from `refs/pull/<number>/head`, so no local branch is created; the worktree is a detached checkout of the PR head.

#### Review mode

Review mode is on by default. In review mode, the files in the worktree are the PR head, while `HEAD` and the index point at the merge base of the PR and its base branch. The whole PR then appears as uncommitted changes:

```sh
cd "$(gh worktree cd 123)"
git status          # files changed by the PR
git diff            # the PR's diff, like on GitHub (new files show up as untracked)
```

Editors and plugins that mark changed lines (for example gitsigns.nvim or VS Code's gutter) then highlight exactly what the PR changes.

`sync`, `clean`, and `remove` know about review mode: changes that come from review mode are not treated as local edits, while files you edit yourself are still protected.

Use `--no-review` for a regular checkout. Running `gh worktree pr --reuse --no-review 123` on an existing review-mode worktree switches it back.

### Develop on a new branch

```sh
gh worktree new feature/awesome            # new branch from the default branch
gh worktree new -b develop hotfix/urgent   # new branch from "develop"
gh worktree new                            # prompt for a branch name
gh worktree new --reuse feature/awesome    # use the existing worktree if there is one
```

The default base branch is the one `origin/HEAD` points to, falling back to `main`, then `master`.

### Move between worktrees

`gh worktree cd` prints the path of a worktree, so it can be combined with `cd`:

```sh
cd "$(gh worktree cd 123)"               # PR #123
cd "$(gh worktree cd -r upstream 456)"   # PR #456 from the "upstream" remote
cd "$(gh worktree cd feature/awesome)"   # a development branch
cd "$(gh worktree cd)"                   # pick interactively
```

A number is treated as a PR and anything else as a branch name. Use `--work <branch>` for a branch whose name is a number.

`pr` and `new` accept `--cd` to print only the path, which makes "create or reuse, then enter" a one-liner:

```sh
cd "$(gh worktree pr --reuse --cd 123)"
cd "$(gh worktree new --reuse --cd feature/awesome)"
```

#### Shell integration

To have `gh workspace cd` change the directory directly, add this function to `~/.bashrc` or `~/.zshrc`:

```sh
gh() {
  if [ "$1" = "workspace" ] && [ "$2" = "cd" ]; then
    shift 2
    local dir
    dir=$(command gh worktree cd "$@") && cd "$dir"
  else
    command gh "$@"
  fi
}
```

```sh
gh workspace cd 123    # cd into PR #123's worktree
gh workspace cd        # pick a worktree interactively
```

### List, update, and remove worktrees

```sh
gh worktree list                     # list PR and development worktrees (alias: ls)
gh worktree sync 123                 # update PR #123's worktree to the latest PR head
gh worktree sync --force 123         # ...discarding local changes
gh worktree remove 123               # remove PR #123's worktree (alias: rm)
gh worktree remove feature/awesome   # remove a development worktree
```

`remove` refuses to delete a worktree that has local changes. Commit, stash, or discard them first.

### Clean up

```sh
gh worktree clean --dry-run   # show what would be removed
gh worktree clean             # remove after confirmation
gh worktree clean --yes       # remove without asking (required when not on a terminal)
```

`clean` removes a worktree only when it has no local changes and:

- it is a PR worktree whose PR is no longer open (closed or merged), or
- it is a development worktree whose branch is merged into the default branch.

Branches are never deleted. Branches merged by squash or rebase are not recognized as merged. Use `--pr=false` or `--work=false` to check only one kind.

### Command reference

| Command | Description | Options |
|---|---|---|
| `pr [number]` | Create a worktree for a PR | `-r, --remote <name>`, `--cd`, `--reuse`, `--no-review` |
| `new [branch]` | Create a worktree with a new branch | `-b, --base <branch>`, `--cd`, `--reuse` |
| `list`, `ls` | List worktrees | |
| `cd [target]` | Print the path of a worktree | `-r, --remote <name>`, `--work <branch>` |
| `sync <number>` | Update a PR worktree to the latest PR head | `-r, --remote <name>`, `--force` |
| `remove`, `rm <target>` | Remove a worktree | `-r, --remote <name>`, `--work <branch>` |
| `clean` | Remove worktrees that are no longer needed | `--pr`, `--work`, `--dry-run`, `-y, --yes` |
| `config` | Show the effective configuration | |
| `help` | Show help | |

`gh worktree help` shows the full usage, and most commands also accept `--help`.

## Configuration

Settings are read from the first of these that exists:

1. `.gh-worktree.yml`, searched from the current directory up to the top of the repository
2. `~/.config/gh-worktree/config.yml`

The `GH_WORKTREE_BASE_PATH` environment variable overrides `worktree_base_path` from either file.

```yaml
# Where worktrees are created, relative to the top of the repository.
# Default: .worktree
worktree_base_path: .worktree

# Files and directories to symlink from the main checkout into every new worktree,
# e.g. untracked local settings you want everywhere.
symlink_files:
  - .env
  - .vscode
```

Entries in `symlink_files` that do not exist in the main checkout are skipped. If a file already exists in the new worktree, you are asked whether to replace it with a symlink.

`gh worktree config` shows the configuration in effect. See [`.gh-worktree.example.yml`](.gh-worktree.example.yml) for an example.

## Shell completion

Basic bash and zsh completion scripts are in [`completions/`](completions). Run `scripts/setup-completion.sh` for setup instructions.

## Development

Requires Go 1.24 or later.

```sh
make build    # build ./gh-worktree
make test     # unit and end-to-end tests
make fmt      # gofmt
make vet      # go vet
```

To try a local build as the `gh worktree` extension:

```sh
make build
gh extension install .
```

Releases are built by [cli/gh-extension-precompile](https://github.com/cli/gh-extension-precompile): pushing a tag such as `v0.1.0` publishes binaries for all supported platforms.

## License

[MIT](LICENSE)
