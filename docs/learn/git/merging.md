---
slug: merging
title: Merging & fast-forwards
description: How git merge combines branches — fast-forward merges that just move a pointer versus three-way merges that create a merge commit with two parents.
keywords: git merge, fast-forward merge, three-way merge, merge commit, merge base, no-ff, ff-only, git log graph, combining branches
level: intermediate
status: full
prereq:
  - branches
faq:
  - q: What is a fast-forward merge?
    a: A fast-forward merge happens when the branch you are merging in is directly ahead of your current branch with no diverging commits. Git has nothing to combine, so it simply slides your branch pointer forward to the other branch's commit. No merge commit is created and the history stays perfectly linear.
  - q: What is a three-way merge?
    a: 'A three-way merge happens when both branches have new commits since they diverged. Git finds their common ancestor (the merge base) and combines the two sets of changes, then records the result as a new merge commit that has two parents — one for each branch. The name comes from the three inputs Git compares: the two branch tips and their merge base.'
  - q: What does git merge --no-ff do?
    a: It forces Git to create a merge commit even when a fast-forward would have been possible. Teams use it so that every feature branch leaves a visible merge commit in history, making it easy to see which commits belonged together as one unit of work and to revert a whole feature at once.
  - q: How do I see merge structure in the log?
    a: Run git log --graph --oneline --all. The --graph option draws the branch and merge structure as ASCII lines down the left side, so you can see where branches diverged and where merge commits brought them back together.
---

# Merging & fast-forwards

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git merge <branch>`** brings another branch's work into your current one. If your
branch hasn't moved since the other one branched off, Git does a **fast-forward** —
it just slides your pointer forward, leaving history linear. If *both* branches have
new commits, Git does a **three-way merge**: it finds their common ancestor (the
**merge base**), combines both sets of changes, and records a **merge commit** with
**two parents**. Force a merge commit with **`--no-ff`**, demand a clean fast-forward
with **`--ff-only`**, and read the shape with **`git log --graph`**.
</div>

You've learned that a [branch](/learn/git/branches/) is a pointer that moves forward as
you commit. Merging is how two of those pointers come back together. There are exactly
two ways it can play out, and once you can tell them apart, merge behaviour stops being
mysterious.

## The setup: bringing a feature into main

Merging always means "pull the commits from *that* branch into the branch I'm standing
on." So you first switch to the destination, then name the source:

```bash
$ git switch main
$ git merge feature/login
```

What happens next depends entirely on whether `main` has moved on since `feature/login`
branched off it.

## Fast-forward: the pointer just moves

Suppose you branched `feature/login` off `main`, did your work, and `main` hasn't
received any new commits in the meantime. The history looks like a straight line with
`main` sitting *behind* the feature tip:

```text
C1 ── C2 ── C3 ── C4
      ↑           ↑
     main      feature/login
```

There's nothing to combine — `feature/login` already contains every commit `main` has,
plus more. So Git simply **fast-forwards**: it moves `main` up to the same commit as
`feature/login`.

```bash
$ git merge feature/login
Updating a1b2c3d..f6e5d4c
Fast-forward
 login.js | 38 ++++++++++++++++++++++++++++++++++++++
 1 file changed, 38 insertions(+)
```

No new commit is created, and history stays perfectly linear. Notice the word
**Fast-forward** in the output — that's your signal that no merge commit happened.

## Three-way merge: a commit with two parents

Now the more common real-world case: while you worked on `feature/login`, someone else
landed commits on `main`. The two branches have **diverged**:

```text
            C3 ── C4   ← feature/login
           /
C1 ── C2 ── C5 ── C6   ← main
```

Git can't just slide a pointer — each branch has commits the other lacks. So it
performs a **three-way merge**, comparing three commits:

1. The tip of `main` (`C6`)
2. The tip of `feature/login` (`C4`)
3. Their **merge base** — the last commit they share (`C2`)

Using the merge base as a reference, Git works out what each side changed and combines
them into a new **merge commit** (`M`) that has *two* parents:

```bash
$ git merge feature/login
Merge made by the 'recursive' strategy.
 login.js | 38 ++++++++++++++++++++++++++++++++++++++
 1 file changed, 38 insertions(+)
```

```text
            C3 ── C4 ──┐
           /           ↓
C1 ── C2 ── C5 ── C6 ── M   ← main
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 205" role="img" aria-label="A three-way merge. Commit C2 is the common merge base; from it the feature line (C3, C4) and the main line (C5, C6) have diverged. A merge commit M joins them, with arrows to two parents — the tip of main and the tip of feature — becoming the new tip of main." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <line x1="108" y1="120" x2="52" y2="120" marker-end="url(#mrg_ar)"/>
    <line x1="198" y1="120" x2="132" y2="120" marker-end="url(#mrg_ar)"/>
    <line x1="278" y1="120" x2="222" y2="120" marker-end="url(#mrg_ar)"/>
    <line x1="200" y1="60" x2="133" y2="111" marker-end="url(#mrg_ar)"/>
    <line x1="278" y1="56" x2="222" y2="56" marker-end="url(#mrg_ar)"/>
    <line x1="369" y1="94" x2="302" y2="115" marker-end="url(#mrg_ar)"/>
    <line x1="369" y1="82" x2="302" y2="61" marker-end="url(#mrg_ar)"/>
  </g>
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <circle cx="40" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="40" y="123">C1</text>
    <circle cx="120" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="120" y="123">C2</text>
    <circle cx="210" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="210" y="123">C5</text>
    <circle cx="290" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="290" y="123">C6</text>
    <circle cx="210" cy="56" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="210" y="59">C3</text>
    <circle cx="290" cy="56" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="290" y="59">C4</text>
    <circle cx="380" cy="88" r="12" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.6"/><text x="380" y="91">M</text>
  </g>
  <g font-size="8.5" text-anchor="middle" font-weight="600">
    <line x1="290" y1="44" x2="290" y2="36" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
    <rect x="258" y="18" width="64" height="18" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="290" y="30" fill="currentColor">feature</text>
    <line x1="380" y1="100" x2="380" y2="110" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
    <rect x="348" y="110" width="64" height="18" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="380" y="122" fill="currentColor">main</text>
    <line x1="120" y1="132" x2="120" y2="150" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
    <rect x="82" y="150" width="76" height="18" rx="4" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/><text x="120" y="162" fill="currentColor">merge base</text>
  </g>
  <text x="250" y="192" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">the merge commit M records two parents — one tip from each diverged branch</text>
  <defs><marker id="mrg_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Because both lines gained commits after they split at the <strong>merge base</strong> (C2), Git can't just move a pointer. It compares each tip against that base, combines the two sets of changes, and records the result as a <strong>merge commit</strong> (M) with <strong>two parents</strong> — which is what a fork-and-rejoin looks like in the graph.</figcaption>
</figure>

If both sides changed the *same lines*, Git can't decide automatically and you get a
[merge conflict](/learn/git/merge-conflicts/) — the subject of the next lesson.

## --no-ff: forcing a merge commit on purpose

Sometimes a fast-forward is *possible* but a team would rather not have it. With
**`--no-ff`** Git creates a merge commit even when it could have fast-forwarded:

```bash
$ git merge --no-ff feature/login
Merge made by the 'ort' strategy.
```

Why want this? The merge commit becomes a visible marker that "these commits were one
feature." It groups the work in the history, makes the feature easy to identify, and
lets you revert the whole thing with a single `git revert -m 1 <merge>`. Many teams
configure `--no-ff` for merges into `main` for exactly this clarity.

## --ff-only: refuse to create a merge commit

The opposite preference is "only merge if it's a clean fast-forward; otherwise stop and
let me decide." That's **`--ff-only`**:

```bash
$ git merge --ff-only feature/login
fatal: Not possible to fast-forward, aborting.
```

This is a guardrail. It's common in scripts and in `git pull` configuration to avoid
surprise merge commits — if the branches have diverged, Git refuses rather than quietly
making a merge you didn't intend.

| Option | Behaviour |
|--------|-----------|
| (default) | Fast-forward if possible, otherwise three-way merge |
| `--no-ff` | Always create a merge commit |
| `--ff-only` | Fast-forward only; abort if not possible |

## Seeing the result with git log --graph

To actually *see* whether you got a straight line or a merge commit, draw the history:

```bash
$ git log --graph --oneline --all
*   8d3f1a2 (HEAD -> main) Merge branch 'feature/login'
|\
| * f6e5d4c Add login form
| * c3b2a19 Add login route
* | a7c9e02 Update README
|/
* 2f1e0d9 Initial layout
```

The lines on the left show where `feature/login` split off and where the merge commit
(`8d3f1a2`, with its `|\` fork) brought it back. A fast-forward merge, by contrast,
shows no fork at all — just one unbroken column. Reach for `git log --graph` whenever
you want to confirm what a merge actually did.

<div class="knowledge-check" data-quiz data-correct-msg="Right — diverged branches need a three-way merge, which creates a two-parent merge commit." markdown="0">
  <p class="knowledge-check__q">Quick check: both branches gained new commits after they diverged. What kind of merge does Git perform?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A fast-forward — it just moves the pointer</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A three-way merge with a two-parent merge commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">No merge is possible until you rename a branch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`git merge <branch>`** pulls another branch's commits into the one you're on.
- A **fast-forward** just slides your pointer forward — no merge commit, linear history.
- A **three-way merge** combines diverged work using the **merge base** and creates a **merge commit with two parents**.
- **`--no-ff`** always makes a merge commit (grouping a feature); **`--ff-only`** refuses anything but a fast-forward.
- **`git log --graph --oneline --all`** shows the branch-and-merge shape.

Next up: what to do when both sides edited the same lines — [resolving merge conflicts](/learn/git/merge-conflicts/).
