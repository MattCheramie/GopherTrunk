---
slug: glossary
title: Glossary of Git & GitHub terms
description: Plain-language definitions of Git and GitHub terms — commit, branch, merge, rebase, remote, pull request, Actions, and dozens more — each cross-linked to the learning path.
keywords: git glossary, github terms, git terminology, version control glossary, git dictionary, commit branch merge rebase, pull request definition, github actions terms
level: beginner
status: full
lesson_standalone: true
---

# Glossary of Git & GitHub terms

Every term used across the [learning path](/learn/), defined in plain language and
linked to the lesson where it's explained in full. Skim it as a refresher, or use
your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are grouped by theme and
ordered roughly from fundamentals to workflows.

## Version control fundamentals

**Version control** — A system that records changes to files over time so you can
review history, undo mistakes, and collaborate. See
[What is version control?](/learn/git/what-is-version-control/)

**Git** — The distributed version-control system this path teaches; every clone holds
the full history. See [What is version control?](/learn/git/what-is-version-control/)

**Repository (repo)** — The project folder Git tracks, including all its history,
branches, and configuration stored in the hidden `.git` directory. See
[Your first repository](/learn/git/first-repository/)

**Working tree (working directory)** — The actual files you see and edit on disk, as
checked out from a particular commit. See [Git's mental model](/learn/git/git-mental-model/)

**Staging area (index)** — The in-between space where you assemble exactly the changes
that will go into your next commit. See [Staging & commits](/learn/git/staging-and-commits/)

**Commit** — A saved snapshot of your project at a point in time, with an author,
message, and a link to its parent commit. See [Staging & commits](/learn/git/staging-and-commits/)

**Stage / unstage** — Adding a change to the staging area (`git add`) or removing it
again (`git restore --staged`) before committing. See
[Staging & commits](/learn/git/staging-and-commits/)

**`.gitignore`** — A file listing patterns Git should leave untracked, such as build
output and secrets. See [Ignoring files](/learn/git/gitignore/)

## How Git stores data

**SHA / hash** — The 40-character SHA-1 (or SHA-256) identifier Git computes for every
object; it names a commit, tree, or blob uniquely. Often abbreviated to its first
seven characters. See [Git's mental model](/learn/git/git-mental-model/)

**Blob** — The Git object that stores the raw contents of a single file. See
[Git's mental model](/learn/git/git-mental-model/)

**Tree** — The Git object that represents a directory, listing the blobs and subtrees
inside it. See [Git's mental model](/learn/git/git-mental-model/)

**Ref** — A human-friendly name that points to a commit, such as a branch or tag.
Refs live under `.git/refs`. See [Git's mental model](/learn/git/git-mental-model/)

**HEAD** — A special ref pointing to your current location — usually the tip of the
branch you have checked out. See [Git's mental model](/learn/git/git-mental-model/)

**Detached HEAD** — The state when HEAD points directly at a commit instead of a
branch; new commits aren't on any branch until you make one. See [Branches](/learn/git/branches/)

## Inspecting your work

**`git init`** — The command that creates a new, empty Git repository in the current
folder. See [Your first repository](/learn/git/first-repository/)

**`git status`** — Shows which files are staged, modified, or untracked right now. See
[Status & diffs](/learn/git/status-and-diffs/)

**Diff** — The line-by-line comparison showing what changed between two versions of
your files. See [Status & diffs](/learn/git/status-and-diffs/)

**Hunk** — A contiguous block of changed lines within a diff; you can stage hunks
individually. See [Status & diffs](/learn/git/status-and-diffs/)

**`git log`** — Lists the commit history, most recent first. See
[Viewing history](/learn/git/viewing-history/)

**`git blame`** — Annotates each line of a file with the commit and author that last
changed it. See [Viewing history](/learn/git/viewing-history/)

## Moving around & undoing

**Checkout** — The classic command (`git checkout`) for switching branches or
restoring files; now often split into `switch` and `restore`. See
[Branches](/learn/git/branches/)

**Switch** — The newer command (`git switch`) dedicated to changing the branch you
have checked out. See [Branches](/learn/git/branches/)

**Restore** — The newer command (`git restore`) for discarding changes in the working
tree or unstaging files. See [Undoing changes](/learn/git/undoing-changes/)

**Reset** — Moves the current branch to another commit. `--soft` keeps your changes
staged, `--mixed` (the default) unstages them, and `--hard` discards them entirely.
See [Undoing changes](/learn/git/undoing-changes/)

**Revert** — Creates a new commit that undoes an earlier one, leaving history intact —
the safe way to undo shared commits. See [Undoing changes](/learn/git/undoing-changes/)

**Reflog** — A local log of where HEAD and branches have pointed, letting you recover
commits that seem lost. See [Cherry-pick, bisect & reflog](/learn/git/cherry-pick-bisect-reflog/)

**Stash** — A place to shelve uncommitted changes temporarily so you can switch tasks,
then reapply them later. See [Stashing changes](/learn/git/stashing/)

## Branching & merging

**Branch** — A movable pointer to a line of development, letting you work without
disturbing the main branch. See [Branches](/learn/git/branches/)

**Tag** — A fixed, named pointer to a specific commit, typically marking a release.
See [Releases & tags](/learn/git/releases-and-tags/)

**Merge** — Combining the histories of two branches into one, usually creating a
merge commit. See [Merging branches](/learn/git/merging/)

**Fast-forward** — A merge with no divergence, where Git simply slides the branch
pointer forward instead of making a merge commit. See [Merging branches](/learn/git/merging/)

**Merge base** — The most recent common ancestor of two branches, used as the starting
point for a merge. See [Merging branches](/learn/git/merging/)

**Merge conflict** — When the same lines were changed differently on two branches and
Git can't decide automatically; you resolve it by hand. See
[Resolving merge conflicts](/learn/git/merge-conflicts/)

**Rebase** — Replaying your commits onto a new base to create a linear history, instead
of merging. See [Rebasing](/learn/git/rebasing/)

**Interactive rebase** — A rebase (`git rebase -i`) that lets you reorder, edit,
squash, or drop commits one by one. See [Interactive rebase](/learn/git/interactive-rebase/)

**Squash** — Combining several commits into one, often during an interactive rebase or
when merging a pull request. See [Interactive rebase](/learn/git/interactive-rebase/)

**Fixup** — A commit (or rebase action) that folds changes silently into an earlier
commit, discarding its own message. See [Interactive rebase](/learn/git/interactive-rebase/)

**Cherry-pick** — Copying a single commit from one branch onto another. See
[Cherry-pick, bisect & reflog](/learn/git/cherry-pick-bisect-reflog/)

**Bisect** — A guided binary search (`git bisect`) that finds the commit which
introduced a bug. See [Cherry-pick, bisect & reflog](/learn/git/cherry-pick-bisect-reflog/)

## Remotes & collaboration

**Clone** — Making a complete local copy of a remote repository, including its full
history. See [Working with remotes](/learn/git/remotes/)

**Remote** — A named link to another copy of the repository, usually hosted on a server
like GitHub. See [Working with remotes](/learn/git/remotes/)

**Origin** — The default name Git gives the remote you cloned from. See
[Working with remotes](/learn/git/remotes/)

**Upstream** — The remote that a branch tracks, or the original repo a fork was made
from. See [Working with remotes](/learn/git/remotes/)

**Tracking branch** — A local branch configured to follow a specific remote branch, so
`pull` and `push` know where to go. See [Working with remotes](/learn/git/remotes/)

**Fetch** — Downloading new commits from a remote without changing your working tree.
See [Working with remotes](/learn/git/remotes/)

**Pull** — Fetching from a remote and then merging (or rebasing) those changes into
your branch. See [Working with remotes](/learn/git/remotes/)

**Push** — Uploading your local commits to a remote. See
[Working with remotes](/learn/git/remotes/)

## GitHub basics

**GitHub** — A popular web platform for hosting Git repositories and collaborating
through pull requests, issues, and more. See [What is GitHub?](/learn/git/what-is-github/)

**Fork** — Your own copy of someone else's repository on GitHub, where you can make
changes and propose them back. See [Forking a repository](/learn/git/forking/)

**Pull request (PR)** — A GitHub proposal to merge one branch into another, where
others can review and discuss the changes. See [Pull requests](/learn/git/pull-requests/)

**Draft PR** — A pull request marked as not yet ready for review or merging. See
[Pull requests](/learn/git/pull-requests/)

**Code review** — Teammates reading and commenting on a pull request before it is
merged. See [Pull requests](/learn/git/pull-requests/)

**CODEOWNERS** — A file that designates who must review changes to particular paths in
the repository. See [Branch protection](/learn/git/branch-protection/)

**Issue** — A GitHub tracker item for a bug, task, or feature request. See
[Issues & Projects](/learn/git/issues-and-projects/)

**Label** — A colored tag applied to issues and pull requests for sorting and
filtering. See [Issues & Projects](/learn/git/issues-and-projects/)

**Milestone** — A group of issues and pull requests tied to a target, such as a
release. See [Issues & Projects](/learn/git/issues-and-projects/)

**Projects** — GitHub's built-in boards and tables for planning and tracking work
across issues and PRs. See [Issues & Projects](/learn/git/issues-and-projects/)

**Discussions** — A GitHub forum space for questions and open-ended conversation,
separate from issues. See [GitHub Pages & more](/learn/git/github-pages-and-more/)

**Wiki** — A repository's built-in collection of editable documentation pages. See
[GitHub Pages & more](/learn/git/github-pages-and-more/)

**Gist** — A lightweight, shareable snippet or small file hosted on GitHub. See
[GitHub Pages & more](/learn/git/github-pages-and-more/)

**GitHub Pages** — Free static-site hosting served directly from a repository. See
[GitHub Pages & more](/learn/git/github-pages-and-more/)

## Authentication & security

**SSH key** — A cryptographic key pair used to authenticate to GitHub without typing a
password each time. See [Authentication](/learn/git/authentication/)

**Personal access token (PAT)** — A generated token that stands in for your password
when using HTTPS or the API. See [Authentication](/learn/git/authentication/)

**Branch protection** — Rules that guard a branch, requiring reviews or passing checks
before changes can merge. See [Branch protection](/learn/git/branch-protection/)

**Status check** — An automated test or report that must pass before a pull request can
be merged. See [Branch protection](/learn/git/branch-protection/)

## Automation: CI/CD & Actions

**GitHub Actions** — GitHub's built-in automation platform for running workflows on
events like pushes and pull requests. See [GitHub Actions](/learn/git/github-actions/)

**Workflow** — A configurable automated process defined in a YAML file under
`.github/workflows`. See [GitHub Actions](/learn/git/github-actions/)

**Job** — A set of steps in a workflow that runs on a single runner. See
[GitHub Actions](/learn/git/github-actions/)

**Step** — An individual task within a job, either a shell command or a reusable
action. See [GitHub Actions](/learn/git/github-actions/)

**Runner** — The machine (GitHub-hosted or self-hosted) that executes a job's steps.
See [GitHub Actions](/learn/git/github-actions/)

**CI/CD** — Continuous Integration and Continuous Delivery/Deployment: automatically
building, testing, and shipping code on every change. See
[GitHub Actions](/learn/git/github-actions/)

## Releases & versioning

**Release** — A packaged, named version of a project, usually built around a tag and
including notes and assets. See [Releases & tags](/learn/git/releases-and-tags/)

**Semantic versioning (SemVer)** — A version scheme of MAJOR.MINOR.PATCH that signals
the kind of changes in a release. See [Releases & tags](/learn/git/releases-and-tags/)

**Conventional commits** — A commit-message convention (e.g. `feat:`, `fix:`) that
enables automated changelogs and version bumps. See [Best practices](/learn/git/best-practices/)

## Advanced & workflow

**Submodule** — A repository embedded inside another repository at a pinned commit.
See [Submodules, hooks & worktrees](/learn/git/submodules-hooks-worktrees/)

**Hook** — A script Git runs automatically at certain points, such as before a commit
or after a merge. See [Submodules, hooks & worktrees](/learn/git/submodules-hooks-worktrees/)

**Worktree** — An additional working tree linked to the same repository, letting you
check out several branches at once. See
[Submodules, hooks & worktrees](/learn/git/submodules-hooks-worktrees/)

**Git LFS (Large File Storage)** — An extension that stores large binary files outside
the main repo to keep it lean. See
[Submodules, hooks & worktrees](/learn/git/submodules-hooks-worktrees/)

**Monorepo** — A single repository holding many projects or packages together. See
[Workflow strategies](/learn/git/workflow-strategies/)

**Trunk-based development** — A workflow where everyone commits to one main branch with
very short-lived branches. See [Workflow strategies](/learn/git/workflow-strategies/)

**GitHub Flow** — A lightweight branching workflow: branch off main, open a pull
request, review, and merge. See [Workflow strategies](/learn/git/workflow-strategies/)

**Git Flow** — A more structured branching model with long-lived develop, release, and
hotfix branches. See [Workflow strategies](/learn/git/workflow-strategies/)

---

Didn't find a term? It's probably explained in one of the
[learning-path lessons](/learn/) — or open an
[issue on GitHub](https://github.com/MattCheramie/GopherTrunk/issues) and we'll add it.
