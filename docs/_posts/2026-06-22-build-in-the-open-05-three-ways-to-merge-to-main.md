---
title: "Build in the Open, Part 5: Branching & the Three Ways to Merge to Main"
description: How to use Git branches and pick the right GitHub merge strategy — merge commit, squash & merge, or rebase & merge — with a clear "use this when" for each.
keywords: git branching, merge to main, squash and merge, rebase and merge, merge commit, pull request, draft pr, required status checks, github, claude code
category: tutorials
tags: [github, git, branching, pull-requests, workflow]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 5
---

> **TL;DR:** A branch is a cheap, isolated copy of your work; a pull request is
> how it gets back into `main`. GitHub gives you three merge buttons — **Create
> a merge commit**, **Squash and merge**, and **Rebase and merge** — and they
> match three real ways of shaping work: many small commits, a few large logical
> commits, or one clean squashed commit per feature. Pick one default for the
> repo, and let the *shape of the work* pick the exception.

**Key takeaways**

- Always branch off `main`; never commit straight to it. The branch is your
  scratchpad, the PR is your gate.
- **Squash and merge** keeps `main` linear and readable — one feature, one
  commit. It's the safest default for most projects.
- **Rebase and merge** preserves a curated set of logical commits without a
  merge bubble — use it when each commit is a reviewable step.
- **Create a merge commit** keeps the full branch history and records *when*
  things merged — use it for long-lived or release branches.
- Required status checks and review turn the merge button from a hope into a
  guarantee.

*This is Part 5 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **What a branch actually is** — and why you always make one.
- **The pull request** as the gate: review, status checks, draft PRs.
- **The three ways to merge to main**, mapped to GitHub's three merge buttons,
  with a "use this when" for each.
- **How to choose a default** for your repo and when to break it.
- **How GopherTrunk does it** — squash-on-merge, Conventional Commits, and a
  "one logical change per PR" rule.

## What is a branch, and why always make one?

A **branch** is a movable pointer to a line of commits. When you branch off
`main`, you get an isolated copy of the project's history that you can commit to
freely without touching the shared, "known-good" `main` branch. If the
experiment works, you merge it back; if it doesn't, you delete the branch and
nothing is lost.

The rule experienced teams follow is simple: **`main` is always releasable, so
nobody commits to it directly.** All work happens on a branch, and the only way
in is a pull request. That single discipline buys you code review, automated
testing, and a clean audit trail — for free.

A good branch is *one thing*: one bug fix, one feature, one refactor. A branch
named `fix-timezone-parsing` should not also quietly reformat three unrelated
files. Keeping branches narrow is what makes the merge decision below easy.

## The pull request is the gate

A **pull request (PR)** proposes merging your branch into `main`. It's where
three safety mechanisms live:

- **Review.** A second pair of eyes (or, on a solo project, your own
  cooled-off self the next morning) reads the diff and approves or requests
  changes.
- **Required status checks.** GitHub can *block* the merge button until your
  CI — build, tests, linters — reports success. We cover building those checks
  in [Part 7]({{ '/blog/tutorials/build-in-the-open-07-github-actions-which-workflows/' | relative_url }}).
- **Draft PRs.** Open a PR as a **draft** when the work isn't done but you want
  CI to run and reviewers to peek early. You can't merge a draft until you mark
  it "Ready for review" — a built-in guardrail against merging half-finished
  work.

Crucially, the *shape* of the commits inside your branch and the *merge button*
you press are two different decisions. The next sections connect them.

## The three ways to merge to main

There are three honest ways to land a branch, defined by how many commits you
want to appear on `main`. GitHub exposes each as a merge-button option.

### 1. Many small commits → one PR

You committed often while exploring — "wip", "try other approach", "fix typo",
"actually fix it". That's a perfectly good way to *work*; it's a terrible way to
leave `main` looking. You have two choices at merge time:

- **Squash and merge** collapses all of it into a single commit on `main`. The
  messy journey stays on the branch (and in the closed PR), `main` stays clean.
- **Create a merge commit** keeps every commit *and* adds a merge commit that
  records the join. Honest, but it makes `main` noisy.

**Use this when:** the work was exploratory or incremental and the individual
commits aren't worth preserving. Reach for **squash and merge** here — it's the
most common real-world case.

### 2. Three to five large logical commits → one PR

Sometimes a feature genuinely has a few distinct, reviewable steps: "add the
parser", "wire it into the pipeline", "add the config flag". Each commit
compiles, passes tests, and tells a story. You curated this history on purpose —
often with an interactive rebase to tidy it before opening the PR.

**Use this when:** each commit is a meaningful, self-contained step a future
reader (or `git bisect`) would want to land on individually. Reach for **rebase
and merge** to replay those commits onto `main` with no merge bubble, or a
careful **merge commit** if you also want to record the integration point.

### 3. One squashed / "special" commit → one PR

The cleanest possible `main`: **one feature equals one commit.** Every entry in
`git log main` is a complete, shippable unit you could revert in one move. This
is what most polished open-source projects present, regardless of how messy the
branch was underneath.

**Use this when:** you want a linear, scannable `main` history and you don't need
intermediate commits preserved. Reach for **squash and merge** and write the
squash commit message carefully — it's the *one* line that survives.

### Mapping it to GitHub's three buttons

| Merge button | What lands on `main` | Best for |
| --- | --- | --- |
| **Create a merge commit** | All branch commits + a merge commit | Long-lived / release branches; preserving exact history |
| **Squash and merge** | One new commit | Exploratory work; "one feature, one commit"; linear history |
| **Rebase and merge** | Each branch commit, replayed, no merge commit | A small curated set of logical commits |

You set which options are even *allowed* in **Settings → General → Pull
Requests**. Many teams disable the ones they don't want so nobody picks the
wrong one by accident.

## How do I choose a default — and when do I break it?

Pick **one** default merge method for the whole repo and use it 95% of the time.
For most projects, that default should be **squash and merge**, because:

- It makes `main` a clean list of features and fixes, not a transcript of
  someone's afternoon.
- It makes reverting trivial — one commit, one `git revert`.
- It pairs naturally with **Conventional Commits** (e.g. `feat:`, `fix:`,
  `docs:`), where the single squash message becomes the changelog entry.

Break the default only when the work's shape demands it: a carefully sequenced
refactor that benefits from rebase-and-merge, or a release/integration branch
where a merge commit's record of "what merged when" is the whole point.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) keeps `main` linear
with **squash-on-merge** as the standing rule. Its
[`CONTRIBUTING.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)
spells out both the branch discipline and the merge policy:

- **One logical change per PR.** The guide's rule of thumb: *"if you can't
  summarise the change in one sentence in the imperative mood, the change is
  doing more than one thing and should be split."* Bug fixes ship as one commit
  with a regression test; refactors are a separate PR, never bundled with a
  behaviour change.
- **Messy branches are fine; `main` is not.** Contributors are told force-push is
  fine on feature branches because *"the maintainer squashes on merge to keep
  `main` history clean."* That's case 1 above, applied as policy.
- **Conventional Commits.** Look at the real history and the pattern is obvious —
  `feat: expose P25 site identity in grant events and via /api/v1/sites (#698)`,
  `fix(sdr): serialize StreamIQ re-open behind async teardown (#686)`,
  `ci(pages): run docs rebuild daily at 11:00 Central`. Each squashed commit is a
  typed, scoped, one-line summary that doubles as a changelog line.
- **The merge button is gated, not free.** GopherTrunk's branch-protection
  ruleset (`.github/rulesets/main-branch-protection.json`) requires the
  `build-test`, `usb-windows`, and `usb-macos` status checks to pass and requires
  review threads resolved before the merge button unlocks — and it allows all
  three merge methods (`"allowed_merge_methods": ["merge", "squash", "rebase"]`)
  so the maintainer can pick per-PR while defaulting to squash.
- **A test plan rides along.** Every PR fills in the template at
  [`.github/pull_request_template.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/.github/pull_request_template.md),
  whose **Test plan** checklist asks for `make vet test` green locally,
  `make integration` if the daemon changed, added/updated tests, and a hardware
  smoke test if SDR/USB/vocoder code changed. The shape of the work is reviewed
  *before* anyone reaches for the merge button.

You're not building a radio, but the moves port directly: branch per change,
keep each PR to one logical thing, gate the merge with CI and review, and squash
unless the work's structure earns an exception.

## FAQ

**Should I squash, rebase, or merge-commit by default?**
Default to **squash and merge** for a clean, linear `main` where one feature is
one commit. Use **rebase and merge** when a few curated commits each deserve to
survive, and a **merge commit** for long-lived or release branches where the
record of when things merged matters.

**What's the difference between rebase-and-merge and a merge commit?**
Rebase-and-merge replays your branch's commits directly onto `main` with no extra
commit, giving a straight line. A merge commit keeps your commits *and* adds a
new commit recording the join, which preserves the exact branch topology but
makes the history non-linear.

**What is a draft pull request and when should I open one?**
A draft PR runs your CI and lets reviewers see the work, but its merge button is
disabled until you mark it "Ready for review." Open one when you want early
feedback or to confirm checks pass before the work is finished.

**Can I commit directly to main?**
You can, but you shouldn't. Branch-protection rules (Part 12) can forbid it
outright. Routing every change through a branch and PR gives you review,
automated testing, and an easy path to revert.

**How many commits should one PR have?**
As many as the work honestly needs — but the PR should still represent **one
logical change**. If your commits span two unrelated concerns, that's two PRs,
not one.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Git & GitHub Fundamentals (the Web Interface)]({{ '/blog/tutorials/build-in-the-open-04-git-github-fundamentals-web-interface/' | relative_url }})
· Next →
[Part 6: Planning & Tracking Work — and Inviting Contributors]({{ '/blog/tutorials/build-in-the-open-06-issues-planning-inviting-contributors/' | relative_url }})
