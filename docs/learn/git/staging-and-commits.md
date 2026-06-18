---
slug: staging-and-commits
title: The staging area & commits
description: Master git add and git commit — the staging area as a draft of your next snapshot, the commit message flow, the -a and --amend shortcuts, and partial staging with git add -p.
keywords: git add, git commit, staging area, git index, git commit -m, git commit -a, git commit --amend, git add -p, partial staging, atomic commit
level: beginner
status: full
prereq:
  - first-repository
faq:
  - q: What does git add actually do?
    a: It copies the current state of a file into the staging area (the index), marking it for inclusion in your next commit. Staging is a separate step from committing on purpose — it lets you assemble exactly the set of changes you want to record, even if your working tree contains other edits you're not ready to commit.
  - q: What is the difference between git commit -m and plain git commit?
    a: "`git commit -m \"message\"` records the commit with the message you supply inline. Plain `git commit` opens your configured editor so you can write a longer, multi-line message — useful when a one-liner can't capture the why."
  - q: When should I use git commit --amend?
    a: Use `--amend` to fix the *most recent* commit — to correct its message or fold in a change you forgot. It replaces the last commit with a new one. Avoid amending commits you've already pushed and shared, because it rewrites history that others may have.
  - q: What does git add -p do?
    a: It stages changes *interactively, hunk by hunk*, so you can include some edits from a file while leaving others unstaged. It's the key to splitting a messy working tree into clean, focused commits.
---

# The staging area & commits

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git add`** moves changes into the **staging area** — a *draft of your next
snapshot*. **`git commit`** records that draft as a permanent snapshot with a
message. Use **`-m`** for a quick one-line message, or plain `git commit` to open an
editor. **`git commit -a`** stages-and-commits tracked files in one go; **`--amend`**
fixes the last commit. **`git add -p`** stages selectively, hunk by hunk, so you can
build clean, **atomic** commits.
</div>

In [Your first repository](/learn/git/first-repository/) you created a repo with an
untracked `README.md`. Now you'll bring it under version control by **staging** and
**committing** it, using the [three-area model](/learn/git/git-mental-model/).

## The staging area: a draft of your next snapshot

The **staging area** (also called the **index**) sits between your working tree and
the repository. Think of it as a *draft* you're assembling for your next commit. This
two-step flow — stage, then commit — is what lets you record *some* of your changes
now and save the rest for a separate, focused commit later.

```text
working tree  ── git add ──▶  staging area  ── git commit ──▶  repository
```

## Staging with git add

Stage the README, then check status:

```bash
$ git add README.md
$ git status
On branch main

No commits yet

Changes to be committed:
  (use "git rm --cached <file>..." to unstage)
        new file:   README.md
```

`README.md` has moved from *Untracked* to **Changes to be committed** — it's now in
the staging area, ready to be snapshotted. You can stage everything at once with
`git add .`, but naming files keeps you deliberate about what goes in.

## Recording a snapshot with git commit

Commit the staged change with an inline message:

```bash
$ git commit -m "Add project README"
[main (root-commit) 0a1b2c3] Add project README
 1 file changed, 1 insertion(+)
 create mode 100644 README.md
```

You've made your first commit. `git status` will now report a clean working tree,
and `git log` will show the snapshot you just recorded (see
[Viewing history](/learn/git/viewing-history/)).

## The editor flow for longer messages

For anything beyond a one-liner, run `git commit` with no `-m`. Git opens your
configured `core.editor` with a template:

```text

# Please enter the commit message for your changes. Lines starting
# with '#' will be ignored, and an empty message aborts the commit.
#
# On branch main
# Changes to be committed:
#       modified:   README.md
```

Write a short **summary line**, leave a blank line, then a body explaining the *why*.
Save and close the editor to finish; leave it empty to abort the commit.

## Handy shortcuts: -a and --amend

Two flags you'll reach for often:

- **`git commit -a -m "..."`** stages *all modified, already-tracked* files and
  commits them in one step. Note it does **not** pick up brand-new (untracked) files —
  those still need a `git add` first.
- **`git commit --amend`** replaces the most recent commit. Use it to fix a typo in
  the last message, or to fold in a file you forgot:

```bash
$ git add forgotten-file.txt
$ git commit --amend -m "Add project README and license"
```

Only amend commits you haven't shared yet — amending rewrites history, which causes
trouble for anyone who already has the old commit (more in
[Undoing changes](/learn/git/undoing-changes/)).

## Partial staging with git add -p

Sometimes your working tree holds two unrelated edits in the same file and you want
them in *separate* commits. `git add -p` stages **interactively, one hunk at a
time**:

```bash
$ git add -p
diff --git a/app.py b/app.py
@@ -10,6 +10,7 @@
+    log.debug("start")        # change 1
...
Stage this hunk [y,n,q,a,d,s,e,?]?
```

Answer **y** to stage a hunk, **n** to skip it, **s** to split a large hunk into
smaller ones, or **?** for the full list. This is the core skill behind keeping
commits focused.

## What makes a clean, atomic commit

A good commit is **atomic**: it captures *one* logical change and would still make
sense on its own. A few quick guidelines:

- One concern per commit — don't mix a bug fix with a refactor.
- A clear summary line in the imperative mood ("Add login form," not "Added").
- It leaves the project in a working state.

This is just a taste — the full treatment, including message conventions, lives in
[Best practices](/learn/git/best-practices/). To inspect what you've staged versus
changed before committing, head to [Status & diffs](/learn/git/status-and-diffs/).
The [glossary](/learn/git/glossary/) defines any unfamiliar term.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the staging area is the draft of your next commit." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the staging area best described as?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A backup of your most recent commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A draft of your next snapshot</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A remote copy of the repository</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`git add`** stages changes into the **index** — a draft of the next snapshot.
- **`git commit`** records that draft; **`-m`** for a one-liner, plain `commit` for
  the editor flow.
- **`-a`** stages-and-commits tracked files; **`--amend`** fixes the last commit
  (only if unshared).
- **`git add -p`** stages selectively, hunk by hunk.
- Aim for clean, **atomic** commits — one logical change each.

Next up: reading status and diffs to see exactly what changed.
