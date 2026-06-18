---
slug: submodules-hooks-worktrees
title: Submodules, hooks & worktrees
description: Three lesser-known Git features — submodules to nest one repo inside another, hooks to run scripts at key events, and worktrees to check out multiple branches at once.
keywords: git submodule, git submodule add, recurse-submodules, git hooks, pre-commit hook, core.hooksPath, pre-commit framework, git worktree, git worktree add, git subtree
level: advanced
status: full
prereq:
  - first-repository
faq:
  - q: What is a Git submodule?
    a: A submodule is a Git repository nested inside another Git repository at a specific path. The outer (super) project doesn't store the submodule's files directly — it records the submodule's URL and the exact commit it should be checked out at. This lets you include and version a dependency or shared library by reference while keeping its history separate. Submodules are powerful but have sharp edges, so many teams prefer a package manager or git subtree instead.
  - q: Why aren't my Git hooks shared with my team?
    a: Hooks live in .git/hooks, and the .git directory is never committed or cloned, so hooks are strictly local to each clone. To share them, keep the scripts in a tracked directory (for example .githooks) and point Git at it with git config core.hooksPath .githooks, or adopt a framework like pre-commit that installs hooks from a committed config file.
  - q: What is a Git worktree?
    a: A worktree is an additional working directory attached to the same repository, letting you have multiple branches checked out at once without cloning again. git worktree add ../hotfix main creates a second directory on the main branch while your original directory stays on its feature branch. All worktrees share one .git store, so it's far lighter than a second clone and they can't both check out the same branch.
  - q: When should I use a submodule versus a subtree?
    a: Use a submodule when you want to track a dependency by reference, keep its history fully separate, and contribute back to it easily — at the cost of extra commands and a steeper learning curve for the whole team. Use git subtree when you'd rather vendor the dependency's files directly into your repo so clones just work with no special steps, accepting a larger repo and trickier upstream contribution.
---

# Submodules, hooks & worktrees

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three features you won't reach for daily but will be glad to know. **Submodules** nest one
repo inside another at a pinned commit (**`git submodule add`**, clone with
**`--recurse-submodules`**) — flexible but full of gotchas, with **subtree** as a simpler
alternative. **Hooks** are scripts Git runs at events like **`pre-commit`** and
**`pre-push`**; they live in **`.git/hooks`**, so they aren't committed by default — share
them via **`core.hooksPath`** or the **pre-commit framework**. **Worktrees**
(**`git worktree add ../feature feature`**) check out multiple branches at once from one
repo, no re-cloning.
</div>

These three solve specific problems: depending on another repo, automating checks at Git
events, and juggling several branches simultaneously. Each is optional — but each turns an
awkward workflow into a smooth one when its problem is yours.

## Submodules: a repo inside a repo

A **submodule** embeds one Git repository inside another at a fixed path. The outer
"superproject" doesn't copy the inner files — it records the submodule's URL and the exact
**commit** to check out, so the dependency is versioned by reference:

```bash
$ git submodule add https://github.com/acme/shared-ui.git vendor/shared-ui
Cloning into 'vendor/shared-ui'...
$ git status
	new file:   .gitmodules
	new file:   vendor/shared-ui
```

Adding a submodule creates a tracked **`.gitmodules`** file (mapping path → URL) and stages
the submodule as a single entry pointing at one commit. When someone clones the
superproject, the submodule directory is **empty** until they fetch it:

```bash
$ git clone --recurse-submodules https://github.com/acme/app.git   # clone + fetch submodules
$ git submodule update --init --recursive                          # for an already-cloned repo
```

The gotchas are real:

- A submodule is pinned to **one commit**. To bump it, `cd` in, `git pull`, then commit the
  new pointer in the superproject — easy to forget.
- A plain `git clone` (no `--recurse-submodules`) leaves submodules empty, surprising
  teammates.
- Branch switches in the superproject don't automatically update submodule contents.

If that's more ceremony than you want, **`git subtree`** is the common alternative — it
merges the dependency's files directly into your repo, so clones just work with no extra
commands (at the cost of a bigger repo and trickier upstream contributions). For most
language ecosystems, a proper **package manager** beats both.

## Hooks: scripts that fire on Git events

**Hooks** are scripts Git runs automatically at points in its workflow. They live in
**`.git/hooks`**, where Git ships disabled samples:

```bash
$ ls .git/hooks
applypatch-msg.sample   pre-commit.sample    pre-push.sample
commit-msg.sample       pre-rebase.sample    prepare-commit-msg.sample
```

Remove the `.sample` suffix and make the file executable to activate it. The handy ones:

| Hook | Fires | Typical use |
|------|-------|-------------|
| `pre-commit` | before a commit is created | run linters/formatters; reject on failure |
| `commit-msg` | after the message is written | enforce a message format (e.g. Conventional Commits) |
| `pre-push` | before a push uploads | run the test suite; block a broken push |

A minimal `pre-commit` that blocks the commit (non-zero exit) when linting fails:

```bash
#!/bin/sh
npm run lint || {
  echo "Lint failed — commit aborted."
  exit 1
}
```

The catch: **`.git` is never committed or cloned**, so hooks are strictly local — your
teammates won't get them automatically. Two ways to share:

```bash
$ git config core.hooksPath .githooks   # track scripts in .githooks/ and point Git at them
```

Keep the scripts in a committed `.githooks/` directory and have everyone run that one
config line (or set it in a setup script). Better still for teams, adopt the **pre-commit
framework** (`pre-commit`): you commit a `.pre-commit-config.yaml` listing the checks, and
`pre-commit install` wires up the hook for each developer from that shared config.

## Worktrees: multiple branches checked out at once

Need to fix an urgent bug while your feature branch sits mid-edit? The old answer was
`git stash` or a second clone. **Worktrees** are cleaner: one extra working directory
attached to the *same* repository, on a *different* branch:

```bash
$ git worktree add ../hotfix main
Preparing worktree (checking out 'main')
HEAD is now at 9f8e7d6 Release 1.4
```

Now `../hotfix` is a full working directory on `main` while your original directory stays
on `feature` — both backed by one shared `.git` store, so it's far lighter than cloning
again. Create a directory *and* a new branch in one step, and tidy up when done:

```bash
$ git worktree add -b experiment ../experiment   # new branch "experiment" in ../experiment
$ git worktree list
/home/me/app           a1b2c3d [feature]
/home/me/hotfix        9f8e7d6 [main]
/home/me/experiment    a1b2c3d [experiment]
$ git worktree remove ../hotfix                  # remove when finished
```

The one rule: **a branch can be checked out in only one worktree at a time**, which keeps
them from stepping on each other. Worktrees pair nicely with [bisect](/learn/git/cherry-pick-bisect-reflog/)
or a long-running build you don't want to interrupt your main work.

<div class="knowledge-check" data-quiz data-correct-msg="Right — .git is never cloned, so hooks in .git/hooks stay local unless you share them via core.hooksPath or a framework." markdown="0">
  <p class="knowledge-check__q">Quick check: why don't your teammates automatically get the pre-commit hook you set up?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hooks only run on the default branch</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Hooks live in .git/hooks, and .git is never committed or cloned</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hooks require a paid GitHub plan to share</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Submodules** nest a repo at a **pinned commit**: **`git submodule add`**, clone with **`--recurse-submodules`**, fetch later with **`git submodule update --init`**. Mind the gotchas; consider **subtree** or a package manager.
- **Hooks** run scripts on events like **`pre-commit`**, **`commit-msg`**, **`pre-push`**; they live in **`.git/hooks`** and aren't committed — share via **`core.hooksPath`** or the **pre-commit framework**.
- **Worktrees** (**`git worktree add ../dir branch`**) check out multiple branches at once from one repo; a branch can live in only one worktree.

Next up: choosing how your team branches and ships — [branching strategies: trunk, GitHub Flow & Git Flow](/learn/git/workflow-strategies/).
