---
layout: home

hero:
  name: gh-worktree
  text: Git worktrees for pull requests and branches
  tagline: A GitHub CLI extension. Each PR you review and each branch you start gets its own directory, and the checkout you are working in is left alone.
  image:
    src: /logo.svg
    alt: ""
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/utam0k/gh-worktree
---

## Install

```sh
gh extension install utam0k/gh-worktree
```

## Example

```console
$ gh worktree pr 123                  # check out PR #123
$ gh worktree pr -r upstream 456      # PR #456 of the repository behind "upstream"
$ gh worktree new feature/awesome     # start a new branch
$ gh worktree list
PR Worktrees:
PR         REMOTE     BRANCH               COMMIT          PATH
---        ------     ------               ------          ----
#123       -          (detached)           72d59db         .worktree/pr/pr-123
#456       upstream   (detached)           72d59db         .worktree/pr/upstream/pr-456

Work Worktrees:
BRANCH                         COMMIT          PATH
------                         ------          ----
feature/awesome                72d59db         .worktree/work/feature/awesome
$ cd "$(gh worktree cd 123)"
```

PR worktrees open in [review mode](/guide/review-mode) unless you pass `--no-review`. The files are the PR head while `HEAD` and the index stay at the merge base, so `git status`, `git diff`, and the change markers in your editor show what the PR changes.

When you are done, `gh worktree clean` removes the worktrees of PRs that have been closed or merged and of branches that have been merged, as long as they have no local changes.
