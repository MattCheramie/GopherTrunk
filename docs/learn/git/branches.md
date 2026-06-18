---
slug: branches
title: Branches explained
description: How Git branches really work — a branch is just a movable pointer to a commit. Learn HEAD, creating and switching, renaming, deleting, and detached HEAD.
keywords: git branch, what is a git branch, git switch, git checkout -b, HEAD, detached HEAD, delete branch, rename branch, branch pointer
level: intermediate
status: full
prereq:
  - git-mental-model
  - staging-and-commits
faq:
  - q: What is a Git branch, really?
    a: A branch is just a lightweight, movable pointer to a single commit — a 41-byte file containing a commit hash. It is not a copy of your files. When you commit on a branch, Git creates the new commit and moves the branch pointer forward to it. This is why creating a branch in Git is instant and costs almost nothing, unlike older version control systems that copied the whole project.
  - q: What is the difference between git switch and git checkout?
    a: They overlap, but git switch is the newer, narrower command for changing branches (added in Git 2.23), while git checkout is the older command that does many unrelated things — switching branches, restoring files, and detaching HEAD. Use git switch <name> to move between branches and git switch -c <name> to create one. git checkout -b <name> still works and does the same thing.
  - q: What does detached HEAD mean?
    a: Detached HEAD means HEAD points directly at a commit instead of at a branch name. You get there by checking out a commit hash or a tag. You can still look around and even make commits, but those commits are not on any branch, so they can be lost when you switch away. To keep work made in this state, create a branch with git switch -c <name> before leaving.
  - q: When should I use git branch -D instead of -d?
    a: Use the lowercase -d to delete a branch safely — Git refuses if the branch has commits not yet merged elsewhere, protecting you from losing work. Use the uppercase -D only when you are certain you want to discard those commits, since it forces the deletion regardless of merge state.
---

# Branches explained

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **branch** in Git is not a copy of your project — it's just a **movable pointer**
to one commit, stored as a tiny file. **HEAD** is the pointer that says *where you
are*. Make a commit and the current branch pointer slides forward to it. Create
branches with **`git switch -c <name>`** (or the older `git checkout -b`), list with
`git branch`, and delete with **`-d`** (safe) or **`-D`** (force). If HEAD ever points
straight at a commit instead of a branch, you're in a **detached HEAD** — calm down,
make a branch, and you're back on track.
</div>

In older version control systems a branch meant copying the whole project, so people
branched rarely. Git turns that on its head: a branch is so cheap that branching
becomes a daily habit. To use branches confidently you only need one mental picture —
a branch is a sticky note pointing at a commit. This lesson makes that picture concrete.

## A branch is a movable pointer

You met the [commit graph](/learn/git/git-mental-model/) already: each commit points
back to its parent, forming a chain of snapshots. A **branch** is simply a *name* that
points at one commit in that graph — usually the latest one on a line of work. On disk
it really is a 41-byte file holding a commit hash:

```bash
$ cat .git/refs/heads/main
a1b2c3d4e5f60718293a4b5c6d7e8f9001234567
```

That's the whole branch. There is no folder of files hidden behind it, no expensive
copy. This is why `git branch new-idea` returns instantly even on a huge repository —
it writes one small file. Contrast that with the heavy "copy the trunk" branches of
systems like older SVN, where branching was something you thought twice about.

## HEAD: where you are right now

If branches are pointers to commits, **HEAD** is the pointer to your *current branch*.
It answers the question "where am I, and what will my next commit attach to?" Normally
HEAD points at a branch, and that branch points at a commit:

```bash
$ cat .git/HEAD
ref: refs/heads/main
```

So `HEAD → main → a1b2c3d`. When you switch branches, HEAD is what moves to point at a
different branch name, and Git updates your working files to match that branch's commit.

## What happens to the pointer when you commit

This is the moment everything clicks. Say HEAD points at `main`, which points at commit
`C2`. You stage some changes and commit:

```bash
$ git commit -m "Add login form"
[main d4e5f6a] Add login form
 2 files changed, 38 insertions(+)
```

Git creates a new commit `C3` whose parent is `C2`, then **moves `main` forward** to
point at `C3`. HEAD still points at `main`, so it follows along automatically:

```text
before:   C1 ── C2          after:   C1 ── C2 ── C3
                ↑                                ↑
               main                            main  ← HEAD
```

Every commit you make slides the current branch pointer one step forward. Switch to a
different branch and commit, and *that* branch's pointer moves instead — the two lines
of work diverge from their shared history.

## Creating, switching, listing

The modern command for moving between branches is `git switch`. Create-and-switch in
one step with `-c`:

```bash
$ git switch -c feature/login
Switched to a new branch 'feature/login'
```

That's equivalent to the older, still-common form `git checkout -b feature/login`. To
switch to an existing branch, drop the `-c`:

```bash
$ git switch main
Switched to branch 'main'
```

List your branches with `git branch`; the current one is marked with an asterisk:

```bash
$ git branch
* feature/login
  main
```

| Goal | Command |
|------|---------|
| Create a branch (don't switch) | `git branch <name>` |
| Create and switch | `git switch -c <name>` |
| Switch to existing | `git switch <name>` |
| List local branches | `git branch` |
| List all, including remotes | `git branch -a` |

## Renaming and deleting

Rename the branch you're on with `-m` (move):

```bash
$ git branch -m feature/login feature/auth
```

To delete a branch, use `-d`. Git protects you here: it **refuses** if the branch has
commits that aren't merged anywhere else, so you can't silently throw away work.

```bash
$ git branch -d feature/auth
error: the branch 'feature/auth' is not fully merged.
hint: If you are sure you want to delete it, run 'git branch -D feature/auth'.
```

When you genuinely want to discard those commits, the uppercase `-D` forces it:

```bash
$ git branch -D feature/auth
Deleted branch feature/auth (was d4e5f6a).
```

Rule of thumb: reach for `-d` by default and only escalate to `-D` when you've decided
the commits are disposable.

## Detached HEAD, calmly

Sometimes HEAD points *directly* at a commit instead of at a branch name. This is a
**detached HEAD**, and it happens when you check out a specific commit or a tag rather
than a branch:

```bash
$ git switch --detach a1b2c3d
HEAD is now at a1b2c3d Initial layout

$ git status
HEAD detached at a1b2c3d
```

Nothing is broken. You can look around, build, and even commit. The catch is that any
commits you make here belong to *no branch*, so when you switch away, Git can't find
them by name and they may eventually be garbage-collected. It's perfect for a quick
look at an old state, risky as a place to do real work.

To get out and *keep* anything you committed, give the current spot a branch name
before you leave:

```bash
$ git switch -c experiment   # keeps commits made while detached
$ # or, to discard and return to your branch:
$ git switch main
```

If you only browsed and made no commits, simply `git switch main` and forget it ever
happened. (Unsure what "garbage-collected" or "ref" mean? The
[glossary](/learn/git/glossary/) has you covered.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — a branch is just a movable pointer to a commit." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a Git branch, fundamentally?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A lightweight movable pointer to a commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A full copy of every file in the project</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A backup stored on GitHub</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **branch** is a tiny movable pointer to a commit — not a copy of your files.
- **HEAD** points at your current branch and answers "where am I?".
- Each **commit** moves the current branch pointer forward; HEAD follows.
- Create and switch with **`git switch -c <name>`** (or `git checkout -b`).
- Delete safely with **`-d`**; force with **`-D`** only to discard work.
- **Detached HEAD** means HEAD points at a commit, not a branch — make a branch to keep any work.

Next up: bringing two branches back together with [merging and fast-forwards](/learn/git/merging/).
