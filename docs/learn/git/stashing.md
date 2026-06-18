---
slug: stashing
title: Stashing work in progress
description: How git stash parks dirty, uncommitted changes so you can switch branches or pull cleanly — push, list, pop vs apply, untracked files, and named stashes.
keywords: git stash, git stash pop, stash vs apply, git stash list, stash untracked files, git stash push message, git stash drop, work in progress
level: intermediate
status: full
prereq:
  - staging-and-commits
faq:
  - q: What does git stash do?
    a: git stash takes your uncommitted changes — both staged and unstaged modifications to tracked files — and tucks them away on a stack, then resets your working directory to a clean state matching the last commit. Your changes are saved and recoverable, but your working tree is now clean, so you can switch branches or pull without committing half-finished work.
  - q: What is the difference between git stash pop and git stash apply?
    a: Both reapply your stashed changes to the working directory. The difference is what happens to the stash entry afterward — pop removes it from the stack once it applies cleanly, while apply leaves it on the stack so you can reuse it elsewhere. Use pop for the normal case and apply when you want to apply the same stash to more than one branch.
  - q: Does git stash include untracked files?
    a: By default, no — git stash only saves changes to files Git already tracks. To include new, untracked files use git stash -u (or --include-untracked). To also include ignored files, use -a (--all). New files left out by accident are a common reason a stash seems incomplete.
  - q: How do I stash only some files?
    a: Pass the paths you want to git stash push, for example git stash push src/login.js. Only the listed paths are stashed; everything else stays in your working directory. You can add a message at the same time with -m to label what the stash contains.
---

# Stashing work in progress

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git stash`** parks your dirty, uncommitted changes on a stack and gives you a clean
working tree — perfect for quickly switching branches or pulling without committing
half-done work. Bring it back with **`git stash pop`** (apply *and* remove) or
**`git stash apply`** (apply, keep on the stack). Inspect with **`git stash list`** and
**`git stash show -p`**, label with **`-m "msg"`**, and tidy up with **`drop`** /
**`clear`**. By default stash ignores **untracked** files — add **`-u`** to include them.
You can even stash specific paths.
</div>

You're mid-change when something urgent lands: switch branches to review a colleague's
work, or pull in updates before you go further. But Git won't let you switch with
conflicting uncommitted changes, and you're not ready to commit a mess. **Stashing** is
the answer — a temporary shelf for work in progress. It builds on
[staging and commits](/learn/git/staging-and-commits/).

## Parking changes with git stash

Running `git stash` (a shorthand for `git stash push`) takes your modifications to
tracked files — both staged and unstaged — sets them aside, and restores your working
directory to match the last commit:

```bash
$ git stash
Saved working directory and index state WIP on feature/login: a1b2c3d Add login route

$ git status
On branch feature/login
nothing to commit, working tree clean
```

Your tree is clean, but nothing is lost — the changes are safely tucked onto the stash
stack. You can now switch branches, pull, or do whatever needed a clean tree.

## Listing and inspecting stashes

The stash is a stack; each entry has an index, newest first. List them:

```bash
$ git stash list
stash@{0}: WIP on feature/login: a1b2c3d Add login route
stash@{1}: On main: experimental cache layer
```

To see *what's actually in* a stash before reapplying it, use `show`. Add `-p` for the
full patch:

```bash
$ git stash show -p stash@{0}
diff --git a/login.js b/login.js
index 2f1e0d9..a7c9e02 100644
--- a/login.js
+++ b/login.js
@@ -10,6 +10,9 @@ function login(user) {
+  // validate before submit
+  if (!user.email) return false;
```

## pop vs apply

Both commands reapply a stash's changes to your working directory. The difference is
whether the stash entry survives:

| Command | Reapplies changes | Removes stash entry |
|---------|-------------------|---------------------|
| `git stash pop` | Yes | Yes (if it applies cleanly) |
| `git stash apply` | Yes | No — entry stays on the stack |

```bash
$ git stash pop
On branch feature/login
Changes not staged for commit:
        modified:   login.js
Dropped stash@{0} (b8f3a1c...)
```

Use **`pop`** for the normal "get back to work" case. Use **`apply`** when you want the
same stashed changes on *more than one* branch, since it leaves the entry available. By
default both act on the newest stash (`stash@{0}`); name an index to pick another, e.g.
`git stash pop stash@{1}`.

## Dropping and clearing

Once a stash has served its purpose you can remove it explicitly:

```bash
$ git stash drop stash@{1}   # delete one entry
$ git stash clear            # delete ALL stash entries (irreversible)
```

`pop` drops automatically on success; `apply` doesn't, so you'll often `drop` afterward.
Be careful with `clear` — it wipes the entire stack and there's no undo.

## Untracked files: the -u flag

A common surprise: by default, stash saves only files Git already **tracks**. Brand-new,
untracked files are left sitting in your working directory. Include them with `-u`
(`--include-untracked`):

```bash
$ git stash -u
Saved working directory and index state WIP on feature/login: a1b2c3d Add login route
```

If you also need ignored files (rare), `-a` (`--all`) includes those too. If a stash ever
seems to have "lost" a file, an untracked file left behind is the usual culprit.

## Labels and stashing specific paths

A bare `WIP on ...` message is hard to recognise later. Give the stash a name with `-m`:

```bash
$ git stash push -m "half-finished email validation"
Saved working directory and index state On feature/login: half-finished email validation
```

You can also stash *just some files* by listing their paths — everything else stays put
in your working directory:

```bash
$ git stash push -m "just the login tweaks" src/login.js
```

This is handy when one change is ready to keep working on but another needs to be set
aside.

## Two everyday uses

The two situations stash was made for:

- **Switch branches fast.** You're partway through a feature when a bug report comes in.
  `git stash`, switch to a fix branch, deal with it, switch back, `git stash pop` — and
  you're exactly where you left off.
- **Pull into a dirty tree.** Git refuses to pull if incoming changes would clobber your
  uncommitted edits. `git stash`, `git pull`, `git stash pop` lets you sync cleanly, then
  resume. (Note: `pop` can itself raise a [merge conflict](/learn/git/merge-conflicts/)
  if your stash and the new commits touched the same lines — resolve it the usual way.)

See the [glossary](/learn/git/glossary/) for any term that's still unclear.

<div class="knowledge-check" data-quiz data-correct-msg="Right — apply leaves the stash on the stack; pop removes it after applying." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to apply a stash to two different branches. Which command keeps the stash available afterward?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">git stash pop</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">git stash apply</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">git stash clear</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`git stash`** parks uncommitted tracked changes and leaves a clean working tree.
- **`pop`** applies *and removes* a stash; **`apply`** applies but keeps it on the stack.
- Inspect with **`git stash list`** and **`git stash show -p`**.
- Include untracked files with **`-u`**; remove entries with **`drop`** or **`clear`**.
- Label with **`-m`** and stash specific files by listing their **paths**.
- Classic uses: **switch branches fast** and **pull into a dirty tree**.

Next up: Module 4 begins — meet the hosting side of Git with [what is GitHub?](/learn/git/what-is-github/).
