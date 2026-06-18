---
slug: rebasing
title: Rebasing & a linear history
description: How git rebase replays your commits onto a new base for a clean linear history — merge vs rebase trade-offs, the golden rule, and safe force-pushing.
keywords: git rebase, rebase vs merge, linear history, force-with-lease, golden rule of rebasing, git pull rebase, rebase continue skip abort, force push
level: advanced
status: full
prereq:
  - merging
  - branches
faq:
  - q: What does git rebase actually do?
    a: Rebase takes the commits unique to your branch, sets them aside, moves your branch to start from the tip of another branch, and then replays your saved commits one by one on top of that new base. Because each replayed commit has a different parent, it gets a brand-new commit hash — the originals are abandoned. The result is a linear history with no merge commit.
  - q: When should I rebase instead of merge?
    a: Rebase when you want a clean, linear history and you are working on commits that only you have — for example, tidying up a local feature branch before sharing it, or updating it onto the latest main. Merge when you want to preserve the true historical context of when work happened in parallel, or when the commits have already been shared with others.
  - q: What is the golden rule of rebasing?
    a: Never rebase commits that you have already pushed or shared with others. Rebasing rewrites history by creating new commits, so anyone who based work on the old commits will have a diverged, conflicting history. Only rebase commits that still live solely on your own machine.
  - q: Why use --force-with-lease instead of --force?
    a: After rebasing a branch you have already pushed, you must force-push because the history changed. --force-with-lease is the safe version — it refuses to overwrite the remote if someone else has pushed new commits since you last fetched, protecting their work. Plain --force overwrites unconditionally and can silently destroy a teammate's commits.
---

# Rebasing & a linear history

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Rebase** replays your branch's commits onto a new base, giving each a **new hash** and
producing a **linear history** with no merge commit. **`git rebase main`** updates your
feature branch to start from the latest `main`. The trade-off versus
[merging](/learn/git/merging/): rebase reads cleanly but rewrites history and discards
the record of parallel work. Resolve mid-rebase conflicts with
**`--continue` / `--skip` / `--abort`**. **The golden rule: never rebase commits you've
already pushed or shared.** When you must push a rebased branch, do it safely with
**`--force-with-lease`**, and keep history tidy on pull with **`git pull --rebase`**.
</div>

Merging preserves history exactly as it happened, merge commits and all. Rebasing offers
a different aesthetic: a single straight line, as if you'd written your work on top of
the latest code from the start. Both are valid; the skill is knowing which to reach for
and respecting the one rule that keeps rebasing safe.

## What rebase actually does

Picture a feature branch that split off `main`, after which `main` gained new commits:

```text
            C3 ── C4   ← feature
           /
C1 ── C2 ── C5 ── C6   ← main
```

A merge would join these with a merge commit. A **rebase** instead *moves the base of
your branch*. Git sets your unique commits (`C3`, `C4`) aside, fast-forwards your branch
to the tip of `main` (`C6`), then **replays** `C3` and `C4` on top one at a time:

```bash
$ git switch feature
$ git rebase main
Successfully rebased and updated refs/heads/feature.
```

```text
C1 ── C2 ── C5 ── C6 ── C3' ── C4'   ← feature
                        ↑
                  replayed, new hashes
```

Notice `C3'` and `C4'` — the prime marks matter. Because each replayed commit now has a
*different parent*, Git computes a **new hash** for it. The original `C3`/`C4` are
abandoned. Same changes, brand-new commits, perfectly linear result.

## Merge vs rebase: the trade-off

| | Merge | Rebase |
|--|-------|--------|
| History shape | Branching, with merge commits | Single straight line |
| Preserves *when* work happened | Yes | No — looks sequential |
| Commit hashes | Unchanged | Rewritten (new hashes) |
| Safe on shared commits | Yes | **No** (see golden rule) |
| Best for | Recording true context | A clean, readable history |

There's no universally correct choice. Many teams **rebase local feature branches** to
tidy them, then **merge** into `main` (often with [`--no-ff`](/learn/git/merging/)) to
record the integration point. The point is to choose deliberately, not to treat one as
always better.

## Resolving conflicts mid-rebase

Because rebase replays commits one by one, a conflict can pop up at *each* replayed
commit. The flow mirrors a normal [conflict resolution](/learn/git/merge-conflicts/),
but you finish with `--continue` rather than a commit:

```bash
$ git rebase main
Auto-merging config.yml
CONFLICT (content): Merge conflict in config.yml
error: could not apply c3b2a19... Add login route
```

Your three options at this pause:

```bash
$ git add config.yml          # after editing to resolve
$ git rebase --continue       # apply this commit, move to the next

$ git rebase --skip           # drop the current commit entirely
$ git rebase --abort          # bail out, restore the branch as it was
```

`--abort` is the full safety hatch: it returns your branch to exactly where it stood
before you started the rebase, just like `git merge --abort`. Work through the conflicts
commit by commit, and the rebase finishes.

## The golden rule

This is the one rule that, once internalised, keeps you out of trouble:

> **Never rebase commits that you've already pushed or shared.**

The reason follows from how rebase works. Rebasing creates **new commits with new
hashes** and abandons the originals. If those originals were already on a shared remote
and a teammate built on them, your rewritten history now disagrees with theirs — they'll
see duplicated commits and painful conflicts. Rebase freely on commits that live *only*
on your machine; leave shared history alone (or coordinate explicitly).

## Force-pushing safely

There's a legitimate exception: a feature branch that's *yours*, already pushed for a
[pull request](/learn/git/pull-requests/), that you rebase to update onto `main`. Now the
remote's history and yours have diverged, so a normal push is rejected. You must force —
but force *safely* with **`--force-with-lease`**:

```bash
$ git push --force-with-lease
```

`--force-with-lease` overwrites the remote branch **only if** no one else has pushed to
it since your last fetch. If a teammate snuck in a commit, the push is refused and their
work is protected. Plain `git push --force` skips that check and can silently obliterate
someone else's commits — reach for `--force-with-lease` every time.

## git pull --rebase

`git pull` is "fetch then merge," which can sprinkle little merge commits into your
history every time you sync. To keep your local line straight, rebase your local commits
on top of the freshly fetched ones instead:

```bash
$ git pull --rebase
```

Make it the default for all pulls if you prefer linear history:

```bash
$ git config --global pull.rebase true
```

## Where to go next

This lesson covered rebasing onto a new base. Git can also rebase a branch *onto itself*
to reorder, squash, reword, and drop commits — an enormously useful cleanup tool. That's
**[interactive rebase](/learn/git/interactive-rebase/)**, covered later in the path. Any
term here still unclear? The [glossary](/learn/git/glossary/) defines them.

<div class="knowledge-check" data-quiz data-correct-msg="Right — never rebase commits you've already pushed or shared." markdown="0">
  <p class="knowledge-check__q">Quick check: which commits is it safe to rebase?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Any commits, as long as you force-push afterward</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Commits already pushed and merged into main</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Commits that still live only on your own machine</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Rebase** replays your commits onto a new base, giving them **new hashes** and a **linear history**.
- **Merge vs rebase**: merge preserves true context; rebase reads cleaner but rewrites history.
- Resolve mid-rebase with **`--continue` / `--skip` / `--abort`**.
- **The golden rule:** never rebase commits you've already **pushed or shared**.
- When you must push a rebased branch, use **`--force-with-lease`**, never plain `--force`.
- **`git pull --rebase`** keeps your local history straight when syncing.

Next up: parking unfinished work without committing it — [stashing work in progress](/learn/git/stashing/).
