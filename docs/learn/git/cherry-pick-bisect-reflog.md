---
slug: cherry-pick-bisect-reflog
title: Cherry-pick, bisect & the reflog
description: Three Git power tools — cherry-pick to copy a commit to another branch, bisect to binary-search for the commit that broke something, and the reflog to recover lost work.
keywords: git cherry-pick, cherry-pick range, git bisect, git bisect run, find bad commit, git reflog, recover lost commits, recover after reset hard, recover deleted branch
level: advanced
status: full
prereq:
  - rebasing
  - undoing-changes
faq:
  - q: What does git cherry-pick do?
    a: Cherry-pick takes the changes introduced by one existing commit and applies them as a brand-new commit on your current branch. It's how you copy a single fix from one branch to another without merging the whole branch. The new commit has a different hash from the original because it sits on a different parent. You can cherry-pick a range of commits at once, and use -x to record which commit the change came from.
  - q: How does git bisect find a bad commit?
    a: Bisect performs a binary search through your history. You mark one commit as bad (broken) and an older one as good (working), and Git checks out a commit halfway between them. You test it and tell Git good or bad; it halves the range again, and repeats. In about log2(N) steps it pinpoints the exact commit that introduced the problem. You can automate the testing with git bisect run by supplying a script that exits zero for good and non-zero for bad.
  - q: What is the git reflog?
    a: The reflog is a local log of every position HEAD (and each branch tip) has pointed to — every commit, checkout, reset, rebase, and merge. Even when commits are no longer reachable from any branch, the reflog still references them for a while (90 days by default for reachable entries), so you can find and recover work that seems lost after a bad reset --hard or rebase.
  - q: Can I recover commits after git reset --hard?
    a: Usually yes. reset --hard moves your branch and discards uncommitted changes, but it does not immediately delete the commits it moved away from — they're still in the reflog. Run git reflog, find the HEAD@{n} entry from just before the reset, and either git reset --hard that entry to move your branch back, or git branch recovered <sha> to save it. Uncommitted changes that were never committed, however, are not recoverable this way.
---

# Cherry-pick, bisect & the reflog

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three commands that get you out of trouble. **`git cherry-pick <sha>`** copies one
commit's changes onto your current branch as a new commit — perfect for porting a single
fix without merging a whole branch (`-x` records the origin; ranges work too).
**`git bisect`** binary-searches history to find the commit that introduced a bug: mark
**`bad`** and **`good`** endpoints and Git walks you to the culprit, or
**`git bisect run <script>`** automates it. And **`git reflog`** is the safety net —
it logs everywhere `HEAD` has been, so you can **recover "lost" commits or branches**
after a bad `reset --hard` or rebase.
</div>

Most days you commit, branch, and merge. Some days something breaks, a fix needs to go to
two branches at once, or a `reset --hard` eats work you wanted. These three tools handle
exactly those days. None of them is obscure once you've reached for them in anger.

## Cherry-pick: copy one commit elsewhere

[Merging](/learn/git/merging/) brings *all* of a branch's commits across. Sometimes you
want just one — a hotfix made on `main` that also belongs on a release branch, say.
**`git cherry-pick`** replays a single commit's changes onto your current branch:

```bash
$ git switch release-1.4
$ git cherry-pick a1b2c3d
[release-1.4 9e8d7c6] Fix null check in payment handler
 Date: Mon Jun 15 10:22:01 2026 +0000
 1 file changed, 3 insertions(+), 1 deletion(-)
```

The new commit `9e8d7c6` carries the same diff as `a1b2c3d` but a **different hash**,
because it sits on a different parent. Useful variants:

```bash
$ git cherry-pick a1b2c3d^..e4f5g6h   # a range of commits (exclusive of a1b2c3d's parent)
$ git cherry-pick -x a1b2c3d          # append "(cherry picked from commit a1b2c3d)" to the message
$ git cherry-pick -n a1b2c3d          # apply changes but don't commit yet
```

If the change doesn't apply cleanly you get a conflict, resolved just like any
[merge conflict](/learn/git/merge-conflicts/), then:

```bash
$ git add payment.js
$ git cherry-pick --continue   # or --abort to bail, --skip to drop this one
```

Use cherry-pick sparingly — copying the same change to many branches creates duplicate
commits that can complicate later merges. For a single targeted fix, though, it's exactly
right.

## Bisect: binary-search for the breaking commit

A test passed last week and fails today, and there are 200 commits in between. Checking
them one by one is madness; **`git bisect`** finds the culprit in about
log<sub>2</sub>(N) steps — roughly 8 tests across those 200 commits.

You tell Git one **bad** commit (where the bug exists) and one **good** commit (where it
didn't), and it checks out the midpoint for you to test:

```bash
$ git bisect start
$ git bisect bad                  # current commit is broken
$ git bisect good v1.3.0          # this old tag was fine
Bisecting: 97 revisions left to test after this (roughly 7 steps)
[c4d5e6f] Refactor session store
```

Build and test that checkout, then report the result. Git halves the range and checks out
the next midpoint:

```bash
$ git bisect good                 # this midpoint works — bug is newer
Bisecting: 48 revisions left to test after this (roughly 6 steps)
...
$ git bisect bad                  # this one's broken — bug is older
...
c4d5e6f7 is the first bad commit
commit c4d5e6f7
    Refactor session store
$ git bisect reset                # return to where you started
```

`git bisect reset` restores your original `HEAD`. When you can express "is it broken?" as
a command, automate the whole search with **`git bisect run`** — supply a script (or test
command) that **exits 0 for good, non-zero for bad**:

```bash
$ git bisect start HEAD v1.3.0    # bad then good in one line
$ git bisect run npm test
running 'npm test'
...
c4d5e6f7 is the first bad commit
bisect found first bad commit
```

Git churns through the candidates unattended and prints the first bad commit. (Use exit
code **125** in your script to mark a commit as "skip — can't test this one.")

## The reflog: your local undo history

Branches and tags point at commits, but Git also keeps a private log of **every place
`HEAD` has been** — each commit, checkout, [reset](/learn/git/undoing-changes/),
[rebase](/learn/git/rebasing/), and merge. That's the **reflog**, and it's the single most
reassuring command in Git:

```bash
$ git reflog
7c8d9e0 HEAD@{0}: reset: moving to HEAD~3
4a5b6c7 HEAD@{1}: commit: Add tests for login
3f2e1d0 HEAD@{2}: commit: Add session timeout
a1b2c3d HEAD@{3}: commit: Add login form
9f8e7d6 HEAD@{4}: checkout: moving from main to feature
```

Each `HEAD@{n}` is a reachable point in time. The reflog is **local only** — it lives in
your clone, isn't pushed, and (for reachable entries) is kept about 90 days before
garbage collection.

## Recovering "lost" commits and branches

The reflog turns most disasters into a two-line fix. Say a `git reset --hard HEAD~3`
threw away three good commits:

```bash
$ git reset --hard HEAD~3        # oops — those three commits are gone from the branch
$ git reflog
7c8d9e0 HEAD@{0}: reset: moving to HEAD~3
4a5b6c7 HEAD@{1}: commit: Add tests for login   ← the tip before the reset
$ git reset --hard HEAD@{1}      # move the branch back to before the reset
HEAD is now at 4a5b6c7 Add tests for login
```

The commits were never deleted — `reset --hard` only moved the branch pointer. The same
trick rescues a **deleted branch**: find its old tip in the reflog and re-create it:

```bash
$ git reflog | grep feature
4a5b6c7 HEAD@{6}: checkout: moving from feature to main
$ git branch feature 4a5b6c7     # branch back into existence
```

It even undoes a bad [interactive rebase](/learn/git/interactive-rebase/) — reset to the
`rebase (start)` entry's parent. The one thing the reflog *can't* recover is work that was
**never committed**: `reset --hard` discards uncommitted changes for good, so commit (or
[stash](/learn/git/stashing/)) before you do anything risky.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the reflog still references the pre-reset commit, so you can move the branch back." markdown="0">
  <p class="knowledge-check__q">Quick check: you run git reset --hard HEAD~3 by mistake. How do you get the three committed commits back?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They're gone — reset --hard deletes commits permanently</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Run git cherry-pick HEAD~3 to copy them back</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Find the pre-reset tip in git reflog and reset --hard to it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`git cherry-pick <sha>`** copies one commit's changes onto the current branch as a new commit; ranges, **`-x`**, and conflict resolution all work.
- **`git bisect`** binary-searches between a **`bad`** and a **`good`** commit to pinpoint what broke; **`git bisect run <script>`** automates it (exit 0 = good).
- **`git reflog`** logs every position **`HEAD`** has held — local, ~90 days.
- The reflog **recovers** commits and branches lost to a bad **`reset --hard`** or rebase: find the old tip and reset or re-branch to it.
- It can't recover **uncommitted** work — commit or [stash](/learn/git/stashing/) before risky operations.

Next up: lesser-known plumbing for big projects — [submodules, hooks & worktrees](/learn/git/submodules-hooks-worktrees/).
