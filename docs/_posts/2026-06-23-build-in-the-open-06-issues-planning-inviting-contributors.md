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

<figure class="lab-figure">
<svg viewBox="0 0 660 100" width="660" height="100" role="img" aria-label="The issue lifecycle as a left-to-right pipeline: an issue is opened, then triaged and labeled by type, area and signal, then attached to a milestone and assigned, then linked from a pull request with Closes hash-N, and finally auto-closed when that pull request merges.">
  <rect x="6" y="30" width="116" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="64" y="52" text-anchor="middle" fill="currentColor" font-size="11">opened</text>
  <text x="64" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bug / feature</text>
  <line x1="122" y1="52" x2="132" y2="52" stroke="currentColor"/><polygon points="132,48 138,52 132,56" fill="currentColor"/>
  <rect x="138" y="30" width="116" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="196" y="52" text-anchor="middle" fill="currentColor" font-size="11">triaged</text>
  <text x="196" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">type · area · signal</text>
  <line x1="254" y1="52" x2="264" y2="52" stroke="currentColor"/><polygon points="264,48 270,52 264,56" fill="currentColor"/>
  <rect x="270" y="30" width="116" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="328" y="52" text-anchor="middle" fill="currentColor" font-size="11">milestone</text>
  <text x="328" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ assignee</text>
  <line x1="386" y1="52" x2="396" y2="52" stroke="currentColor"/><polygon points="396,48 402,52 396,56" fill="currentColor"/>
  <rect x="402" y="30" width="116" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="460" y="52" text-anchor="middle" fill="var(--accent)" font-size="11">pull request</text>
  <text x="460" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Closes #N</text>
  <line x1="518" y1="52" x2="528" y2="52" stroke="currentColor"/><polygon points="528,48 534,52 528,56" fill="currentColor"/>
  <rect x="534" y="30" width="116" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="592" y="52" text-anchor="middle" fill="var(--accent)" font-size="11">merged</text>
  <text x="592" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">auto-closed</text>
</svg>
<figcaption>An issue's path on GopherTrunk: opened, triaged with the type·area·signal labels, milestoned, then closed automatically when a <code>Closes #N</code> pull request merges to <code>main</code>.</figcaption>
</figure>

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

<figure class="lab-figure">
<svg viewBox="0 0 640 208" width="640" height="208" role="img" aria-label="The contributor funnel: each step from finding the repository to a merged pull request is paired with the file that lowers its friction. Find the repo maps to README and topics; want to help maps to CONTRIBUTING.md; pick a task maps to the good first issue label; submit maps to the issue and PR templates; get reviewed maps to CODEOWNERS.">
  <text x="120" y="22" text-anchor="middle" fill="var(--fg-muted)" font-size="10">contributor step</text>
  <text x="491" y="22" text-anchor="middle" fill="var(--fg-muted)" font-size="10">what lowers the friction</text>
  <rect x="24" y="34" width="192" height="26" rx="5" fill="none" stroke="currentColor"/><text x="120" y="51" text-anchor="middle" fill="currentColor" font-size="11">find the repo</text>
  <line x1="216" y1="47" x2="356" y2="47" stroke="currentColor"/><polygon points="356,43 366,47 356,51" fill="currentColor"/>
  <rect x="366" y="34" width="250" height="26" rx="5" fill="none" stroke="currentColor"/><text x="491" y="51" text-anchor="middle" fill="currentColor" font-size="11">README + topics</text>
  <rect x="24" y="68" width="192" height="26" rx="5" fill="none" stroke="currentColor"/><text x="120" y="85" text-anchor="middle" fill="currentColor" font-size="11">want to help</text>
  <line x1="216" y1="81" x2="356" y2="81" stroke="currentColor"/><polygon points="356,77 366,81 356,85" fill="currentColor"/>
  <rect x="366" y="68" width="250" height="26" rx="5" fill="none" stroke="currentColor"/><text x="491" y="85" text-anchor="middle" fill="currentColor" font-size="11">CONTRIBUTING.md</text>
  <rect x="24" y="102" width="192" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="120" y="119" text-anchor="middle" fill="var(--accent)" font-size="11">pick a task</text>
  <line x1="216" y1="115" x2="356" y2="115" stroke="currentColor"/><polygon points="356,111 366,115 356,119" fill="currentColor"/>
  <rect x="366" y="102" width="250" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="491" y="119" text-anchor="middle" fill="var(--accent)" font-size="11">good first issue</text>
  <rect x="24" y="136" width="192" height="26" rx="5" fill="none" stroke="currentColor"/><text x="120" y="153" text-anchor="middle" fill="currentColor" font-size="11">submit a PR</text>
  <line x1="216" y1="149" x2="356" y2="149" stroke="currentColor"/><polygon points="356,145 366,149 356,153" fill="currentColor"/>
  <rect x="366" y="136" width="250" height="26" rx="5" fill="none" stroke="currentColor"/><text x="491" y="153" text-anchor="middle" fill="currentColor" font-size="11">issue + PR templates</text>
  <rect x="24" y="170" width="192" height="26" rx="5" fill="none" stroke="currentColor"/><text x="120" y="187" text-anchor="middle" fill="currentColor" font-size="11">get reviewed</text>
  <line x1="216" y1="183" x2="356" y2="183" stroke="currentColor"/><polygon points="356,179 366,183 356,187" fill="currentColor"/>
  <rect x="366" y="170" width="250" height="26" rx="5" fill="none" stroke="currentColor"/><text x="491" y="187" text-anchor="middle" fill="currentColor" font-size="11">CODEOWNERS</text>
</svg>
<figcaption>Every step of the funnel is backed by a file that removes a question: the pair a project should ship first is <code>CONTRIBUTING.md</code> plus a few <code>good first issue</code> tickets.</figcaption>
</figure>

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
