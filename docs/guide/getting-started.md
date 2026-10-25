# Getting started

## Requirements

- git
- [GitHub CLI](https://cli.github.com/), logged in with `gh auth login`

## Install

```sh
gh extension install utam0k/gh-worktree
```

Prebuilt binaries are published for Linux, macOS, Windows, and FreeBSD. Development and testing happen on Linux.

To upgrade later:

```sh
gh extension upgrade worktree
```

## Check out a pull request

Inside a clone of the repository, run:

```sh
gh worktree pr 123
```

This fetches `refs/pull/123/head` and adds a worktree at `.worktree/pr/pr-123`. No local branch is created. The worktree is a detached checkout of the PR head, and your current checkout is not touched.

Leave out the number to pick from the open PRs. Type to filter by title, branch, author, or number, move with the arrow keys, and press Enter.

```sh
gh worktree pr
```

If the PR lives in the repository behind another remote, such as `upstream` when you work in a fork, name the remote:

```sh
gh worktree pr -r upstream 456
```

The repository is then taken from that remote's URL, and the worktree goes to `.worktree/pr/upstream/pr-456`.

New PR worktrees are in [review mode](./review-mode) unless you pass `--no-review`.

## Start a branch

```sh
gh worktree new feature/awesome
gh worktree new -b develop hotfix/urgent
```

The branch is created from the default branch (the one `origin/HEAD` points to, otherwise `main`, otherwise `master`), or from the branch given with `-b`. The worktree goes to `.worktree/work/<branch>`.

## Move between worktrees

`gh worktree cd` prints the path of a worktree, which you pass to `cd`:

```sh
cd "$(gh worktree cd 123)"               # PR #123
cd "$(gh worktree cd feature/awesome)"   # a branch
cd "$(gh worktree cd)"                   # pick from a list
```

`pr` and `new` take `--cd` to print only the path of the worktree they create. Add `--reuse` to get the existing worktree instead of an error when there already is one:

```sh
cd "$(gh worktree pr --reuse --cd 123)"
```

### `gh workspace cd`

With this function in `~/.bashrc` or `~/.zshrc`, `gh workspace cd 123` changes to the worktree directly:

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

## Where worktrees live

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

Run gh-worktree from your main checkout. Paths are relative to the top of the checkout you run it in, so running it inside one of its worktrees creates a nested `.worktree/` there. The directory can be changed in the [configuration](./configuration).

## Clean up

```sh
gh worktree remove 123            # one worktree
gh worktree clean --dry-run       # list what clean would remove
gh worktree clean
```

`clean` removes worktrees of PRs that are no longer open and of branches already merged into the default branch, as long as they have no local changes. Branches are kept. See [`clean`](./commands#clean) for the details.
