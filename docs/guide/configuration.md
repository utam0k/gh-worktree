# Configuration

There are two settings: where worktrees are created, and which files are symlinked into new worktrees.

## Config files

gh-worktree reads the first of these files that exists. They are not merged.

1. `.gh-worktree.yml`, looked up from the current directory to the top of the repository
2. `~/.config/gh-worktree/config.yml`

The `GH_WORKTREE_BASE_PATH` environment variable overrides `worktree_base_path` from either file.

If the file cannot be parsed, the defaults are used and no symlinks are created. `gh worktree config` shows the parse error.

## `worktree_base_path`

The directory worktrees are created in, relative to the top of the repository. The default is `.worktree`. PR worktrees go to `<base>/pr/` and branch worktrees to `<base>/work/`.

```yaml
worktree_base_path: .worktree
```

## `symlink_files`

Paths, relative to the top of the repository, that are symlinked from the main checkout into every new worktree. This is meant for files that are not in git, such as `.env` or editor settings.

```yaml
symlink_files:
  - .env
  - .vscode
```

Paths that do not exist in the main checkout are skipped. If a path already exists in the new worktree, for example because it is tracked, you are asked whether to replace it with a symlink or keep it.

A symlink that git does not ignore is an untracked file, so `clean`, `sync`, and `remove` treat it as a local change. Ignore these paths in `.gitignore` or `.git/info/exclude`.

## Checking the configuration

```console
$ gh worktree config
Current Configuration:
=====================
Worktree Base Path: .worktree

Symlink Files:
   - .env
   - .vscode

Configuration Sources (in priority order):
1. Environment Variable: GH_WORKTREE_BASE_PATH
   - Not set
2. Project Config: .gh-worktree.yml
   ✓ Found
3. User Config: ~/.config/gh-worktree/config.yml
   - Not found
4. Default: .worktree

Worktree Paths:
   PR worktrees: /home/you/src/my-project/.worktree/pr/
   Work branches: /home/you/src/my-project/.worktree/work/
```
