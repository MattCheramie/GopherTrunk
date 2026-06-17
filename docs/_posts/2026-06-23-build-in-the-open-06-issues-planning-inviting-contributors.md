---
title: "Build in the Open, Part 6: Planning & Tracking Work — and Inviting Contributors"
description: How to plan work with GitHub Issues, labels, milestones, and Projects, link issues to PRs, and onboard contributors with CONTRIBUTING, templates, and roles.
keywords: github issues, labels, milestones, github projects, closes issue, good first issue, contributing.md, code of conduct, codeowners, inviting contributors, claude code
category: tutorials
tags: [github, issues, planning, contributors, open-source]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 6
---

> **TL;DR:** Track work where the code lives. GitHub **Issues** capture *what*
> needs doing, **labels** sort them, **milestones** group them toward a release,
> and a **Project board** shows them moving from "todo" to "done." Link each PR
> to its issue with `Closes #N` so merging auto-closes the work. To invite
> contributors, make the path obvious: a `CONTRIBUTING.md`, a code of conduct,
> issue/PR templates, and a few `good first issue` tickets do most of the work.

**Key takeaways**

- A good issue states the problem, the expected behaviour, and how you'll know
  it's fixed — not just "X is broken."
- Labels are a taxonomy (type + area + priority); keep it small enough to
  actually use.
- Milestones answer "what's in the next release?"; Projects answer "what's
  moving right now?"
- `Closes #N` in a PR body auto-closes the issue on merge — the single most
  useful issue-to-PR link.
- The contributor funnel is real: lower every step from "I want to help" to
  "my PR merged."

*This is Part 6 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **Writing good issues** that someone (including future you) can act on.
- **Labels, milestones, and Projects** — what each is for and when to reach for
  it.
- **Linking issues to PRs** so merging closes the loop automatically.
- **Inviting contributors**: the docs and signals that lower the barrier.
- **Collaborator roles** and how much access to hand out.
- **How GopherTrunk does it** — issue-referencing commits, branch naming, and
  the scope rules that keep contributions reviewable.

## What makes a good issue?

An issue is a unit of trackable work. The difference between a useful issue and
noise is structure. A good bug report has:

1. **What happened** — the observed behaviour, with steps to reproduce.
2. **What you expected** — so the gap is unambiguous.
3. **Context** — version, OS, config, logs.
4. **Acceptance criteria** — how everyone will know it's fixed.

A good *feature* issue leads with the problem, not the solution: "operators
can't tell which site a call came from" is more useful than "add a site-ID
field," because it leaves room for the right design. **Issue templates** (in
`.github/ISSUE_TEMPLATE/`) bake this structure in so reporters fill the right
fields without being told.

## Labels, milestones, and Projects — what's each for?

These three tools overlap in newcomers' minds. They're distinct.

### Labels: a small taxonomy

Labels classify issues. The trap is making too many. A taxonomy that stays
usable usually has three axes:

- **Type:** `bug`, `enhancement`, `docs`, `question`.
- **Area:** the subsystem — `area/ui`, `area/parser`, `area/ci`.
- **Signal:** `good first issue`, `help wanted`, `priority/high`.

`good first issue` deserves special mention: it's the label new contributors
filter on, and GitHub surfaces it in its "contribute" UI. Curate a handful of
genuinely small, well-described tickets under it.

### Milestones vs. Projects

- A **milestone** is a bucket of issues/PRs tied to a target — usually a
  release like `v0.5.0`. It answers *"what ships next?"* and shows a progress
  bar.
- A **Project** (the boards under the repo's Projects tab) is a live view of
  work — columns like *Todo / In progress / In review / Done* that cards move
  across. It answers *"what's happening right now?"*

Use milestones for *scope*, Projects for *flow*. A small project can run on
issues + milestones alone and add a board only when several things are in flight
at once.

## How do I link an issue to a pull request?

The magic words go in the PR description (or a commit message):

- `Closes #42`, `Fixes #42`, `Resolves #42` — GitHub **auto-closes** issue #42
  when the PR merges into the default branch.
- `Refs #42` or a bare `#42` — creates a link *without* auto-closing, for
  "related to" references.

This is the highest-leverage habit on this whole list: it ties the *why* (the
issue) to the *what* (the PR) permanently, and it keeps your issue tracker
honest by closing work as it lands instead of leaving stale tickets.

## How do I invite contributors?

People help projects where helping is *easy*. Think of it as a funnel — every
friction point loses people — and reduce friction at each step:

- **They find the repo.** A clear README and topics (Part 12) get them in the
  door.
- **They want to help.** A `CONTRIBUTING.md` tells them how to set up, build,
  test, and submit — so they don't have to ask.
- **They pick something.** `good first issue` and `help wanted` labels give them
  a safe entry point.
- **They behave well together.** A `CODE_OF_CONDUCT.md` sets expectations and
  makes the space welcoming.
- **They submit.** Issue and PR templates make their first contribution land in
  the right shape.
- **The right people review.** A `CODEOWNERS` file auto-requests review from the
  maintainers of the touched code.

You don't need all of these on day one. A `CONTRIBUTING.md` and a couple of
good-first-issues are the highest-value starting pair.

## How much access should I give collaborators?

GitHub's repository roles are tiered, and the rule is **least privilege**:

- **Read** — can clone and open issues/PRs (the default for fork-based
  contributors; they don't need more).
- **Triage** — can manage issues and PRs without write access to code.
- **Write** — can push branches and merge PRs. Hand this out once someone has a
  track record.
- **Maintain / Admin** — settings, secrets, protected branches. Keep this
  circle tiny.

Outside contributors fork the repo and open PRs from their fork — they never
need write access for that. Branch protection (Part 12) means even collaborators
with Write can't bypass review and CI.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) wires planning
straight into its commit and branch conventions:

- **Commits reference their issue.** The history is full of issue-linked
  squash commits — `feat: expose P25 site identity in grant events and via
  /api/v1/sites (#698)`, `fix(sdr): serialize StreamIQ re-open behind async
  teardown (#686)`, `storage: persist mid-call backfilled RID + encryption on
  call end (#696)`. The `(#NNN)` suffix ties every change on `main` back to the
  tracked work that motivated it.
- **Branch names embed the issue number.** Work lands on branches like
  `claude/issue-698-features-…` and `claude/issue-686-fix-…`, so the branch, the
  PR, and the issue all share one identifier from the first commit to the merge.
- **CONTRIBUTING.md sets the scope contract.** Its "How changes are scoped"
  section is the planning rulebook:
  [`CONTRIBUTING.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)
  requires **one logical change per PR**, says **bug fixes ship with a
  regression test** that fails without the fix and passes with it, and keeps
  **refactors in a separate PR, never bundled with a behaviour change.** New
  features that span more than one package are designed in an issue or plan
  *first*, then landed as incremental PRs that each close one tier.
- **The PR template carries a 5-point test plan.** Every PR fills in
  [`.github/pull_request_template.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/.github/pull_request_template.md):
  `make vet test` green locally, `make integration` if the daemon changed,
  tests added/updated, a real-hardware smoke test if SDR/USB/vocoder code
  changed, plus a `## [Unreleased]` CHANGELOG bullet and a **Linked issues**
  line (`Closes #NNN`). The template *is* the funnel — a contributor who fills
  it in produces a reviewable PR by default.
- **Hardware contributions have an opt-in path.** Because not every contributor
  owns an SDR, CONTRIBUTING.md documents env-var-gated real-hardware tests
  (e.g. the Airspy suite skips unless `GOPHERTRUNK_AIRSPY_REAL=1`, with
  companion vars like `GOPHERTRUNK_AIRSPY_REAL_CENTER_HZ` and
  `GOPHERTRUNK_AIRSPY_REAL_RATE_HZ`). The default test run stays green for
  everyone; hardware validation is available to those who can run it.

The lesson transfers to any project: track work in issues, link every PR back to
one, write down how to contribute, and keep each contribution small enough to
review.

## FAQ

**What's the difference between a milestone and a project board?**
A milestone groups issues toward a target (usually a release) and shows a
progress bar — it answers "what ships next?" A Project board shows work moving
through columns like Todo → In progress → Done — it answers "what's happening
right now?"

**How do I make a PR close an issue automatically?**
Put a closing keyword and the issue number in the PR description: `Closes #42`,
`Fixes #42`, or `Resolves #42`. When the PR merges into the default branch,
GitHub closes the issue for you. Use `Refs #42` to link without closing.

**What is a "good first issue" and why does it matter?**
It's a label for small, well-described tasks suitable for newcomers. GitHub
surfaces these in its contribute UI, so curating a few genuinely beginner-sized
tickets is one of the cheapest ways to attract first-time contributors.

**Do contributors need write access to my repo?**
No. Outside contributors fork the repo and open PRs from their fork — that needs
only the default Read access. Grant Write only to trusted regulars, and keep
Admin to a tiny circle.

**What's the minimum to make a repo contributor-friendly?**
A clear `README`, a `CONTRIBUTING.md` that explains setup/build/test/submit, and
a couple of `good first issue` tickets. Add a code of conduct, templates, and
CODEOWNERS as the project grows.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Branching & the Three Ways to Merge to Main]({{ '/blog/tutorials/build-in-the-open-05-three-ways-to-merge-to-main/' | relative_url }})
· Next →
[Part 7: GitHub Actions — Which Workflows to Create and Why]({{ '/blog/tutorials/build-in-the-open-07-github-actions-which-workflows/' | relative_url }})
