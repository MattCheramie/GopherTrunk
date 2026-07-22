---
title: "Build in the Open, Part 7: GitHub Actions — Which Workflows to Create and Why"
description: Which GitHub Actions workflows most projects need — CI, security scanning, release automation, docs build, scheduled jobs — and how to gate merges with them.
keywords: github actions, ci cd, workflows, continuous integration, required status checks, concurrency, cancel-in-progress, release automation, govulncheck, claude code
category: tutorials
tags: [github, github-actions, ci-cd, automation, devops]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 7
---

> **TL;DR:** CI/CD is just "run my checks automatically." Most projects want a
> handful of workflows: **CI** on every PR and push (build, test, lint, vet),
> **security scanning**, **release automation** triggered by version tags, a
> **docs/site build**, **PR-only or expensive checks**, and **scheduled
> housekeeping** like stale-branch cleanup. Wire the important checks in as
> *required status checks* so a red build can't merge, and use
> `concurrency: cancel-in-progress` so new commits cancel stale runs.

**Key takeaways**

- Start with one CI workflow that builds, tests, vets, and checks formatting on
  every PR — that's 80% of the value.
- Make the checks you trust **required status checks** so the merge button stays
  locked until they're green.
- Trigger releases off **version tags** (`v*.*.*`), not branches, so publishing
  is deliberate.
- Use `paths-ignore` to skip irrelevant runs and `cancel-in-progress` to kill
  superseded ones — both save runner minutes.
- Scheduled (`cron`) jobs handle the chores no human should remember: stale
  branches, nightly rebuilds, dependency audits.

*This is Part 7 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **What CI/CD actually is**, in plain terms.
- **The workflows most projects should have** — and why each exists.
- **Required status checks** — turning a workflow into a merge gate.
- **Concurrency and `cancel-in-progress`** — not wasting runner minutes.
- **How GopherTrunk does it** — a tour of its five real workflows.

## What is CI/CD, really?

**Continuous Integration (CI)** means every change is automatically built and
tested the moment it's proposed, so problems surface in minutes instead of at
release time. **Continuous Delivery/Deployment (CD)** extends that to
automatically packaging and shipping the result.

On GitHub, both are **workflows** — YAML files in `.github/workflows/` that
define *jobs* (groups of *steps*) and the *events* that trigger them
(`on: pull_request`, `on: push`, `on: schedule`, a version tag, a manual
`workflow_dispatch`). The mental model: *an event happens → a workflow runs →
jobs report success or failure.*

<figure class="lab-figure">
<svg viewBox="0 0 680 100" width="680" height="100" role="img" aria-label="A continuous-integration pipeline drawn as ordered stages that a pull request passes through: go vet for static checks, a gofmt format check, go build to compile, go test with the race detector, make integration, and finally the required status check that gates the merge.">
  <rect x="10" y="30" width="102" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="61" y="52" text-anchor="middle" fill="currentColor" font-size="11">go vet</text>
  <text x="61" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">static check</text>
  <line x1="112" y1="52" x2="118" y2="52" stroke="currentColor"/><polygon points="117,48 121,52 117,56" fill="currentColor"/>
  <rect x="121" y="30" width="102" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="172" y="52" text-anchor="middle" fill="currentColor" font-size="11">gofmt</text>
  <text x="172" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">format check</text>
  <line x1="223" y1="52" x2="229" y2="52" stroke="currentColor"/><polygon points="228,48 232,52 228,56" fill="currentColor"/>
  <rect x="232" y="30" width="102" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="283" y="52" text-anchor="middle" fill="currentColor" font-size="11">go build</text>
  <text x="283" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">compile</text>
  <line x1="334" y1="52" x2="340" y2="52" stroke="currentColor"/><polygon points="339,48 343,52 339,56" fill="currentColor"/>
  <rect x="343" y="30" width="102" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="394" y="52" text-anchor="middle" fill="currentColor" font-size="11">go test</text>
  <text x="394" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">-race -count=1</text>
  <line x1="445" y1="52" x2="451" y2="52" stroke="currentColor"/><polygon points="450,48 454,52 450,56" fill="currentColor"/>
  <rect x="454" y="30" width="102" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="505" y="52" text-anchor="middle" fill="currentColor" font-size="11">integration</text>
  <text x="505" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">make integration</text>
  <line x1="556" y1="52" x2="562" y2="52" stroke="currentColor"/><polygon points="561,48 565,52 561,56" fill="currentColor"/>
  <rect x="565" y="30" width="102" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="616" y="52" text-anchor="middle" fill="var(--accent)" font-size="11">merge gate</text>
  <text x="616" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">required ✓</text>
</svg>
<figcaption>The <code>build-test</code> job of <code>ci.yml</code> as a stage pipeline: vet, format, build, race-tested, integrated — and only a green run clears the required-status-check merge gate.</figcaption>
</figure>

## Which workflows should most projects have?

You don't need everything on day one, but here's the menu and why each item
earns its place.

### (a) CI on every PR and push

The non-negotiable one. On every pull request (and every push to `main`), it
should at least:

- **build** the project,
- **run the tests**,
- **lint/format-check** (e.g. `gofmt`, `eslint`, `black --check`),
- **vet/static-analyze** for obvious mistakes.

This is the workflow you'll make a *required status check* so nothing red merges.

### (b) Security scanning

Automated checks for known-vulnerable dependencies (SCA), leaked secrets, and
license problems. These catch a class of issue humans reliably miss, and they're
cheap to run on every PR.

### (c) Release automation on tags

When you push a version tag (`v1.2.0`), a workflow builds your artifacts for each
platform, generates checksums and release notes, and uploads them to a GitHub
Release. Tagging — not pushing to a branch — is the trigger, which keeps
publishing a deliberate act. (Full release mechanics are
[Part 11]({{ '/blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/' | relative_url }}).)

### (d) Docs / site build

If you publish a site (GitHub Pages, a docs portal), a workflow builds and
deploys it when the docs change. We cover Pages in
[Part 10]({{ '/blog/tutorials/build-in-the-open-10-websites-support-pages-github-pages/' | relative_url }}).

### (e) PR-only or expensive checks

Some checks are too slow or platform-specific to run everywhere. Scope them to
pull requests (and skip them when only docs change) so they protect merges
without slowing every push.

### (f) Scheduled jobs

A `cron` schedule runs a workflow on a timer — nightly dependency audits, a
daily site rebuild, a weekly report. Great for anything that should happen
"regularly" without a human triggering it.

### (g) Housekeeping

Automation for chores: deleting merged branches, closing stale issues, labeling.
Often `workflow_dispatch` (manual, on demand) with a dry-run option so you can
preview before it acts.

## How do I make a check block merges?

A workflow that runs but doesn't block anything is just advice. To make it a
gate, mark its jobs as **required status checks** in your branch-protection
settings (Part 12). Once a job is required, GitHub disables the merge button
until that job reports success for the PR's head commit.

One gotcha worth knowing now: a workflow *skipped* by `paths-ignore` does **not**
report success — so if a check is required, skipping it can leave a PR stuck as
"blocked." Either don't `paths-ignore` a required workflow, or make the required
check one that always runs.

## What is concurrency and cancel-in-progress?

When you push three commits in quick succession, you don't want three full CI
runs racing — the first two are obsolete. A `concurrency` group with
`cancel-in-progress: true` cancels any in-flight run in the same group when a new
one starts, so only the latest commit's run survives. It's the single easiest way
to cut wasted runner minutes on active PRs. (For deploys, you usually set
`cancel-in-progress: false` so you never kill a half-finished publish.)

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) runs **five**
workflows in
[`.github/workflows/`](https://github.com/MattCheramie/GopherTrunk/tree/main/.github/workflows),
one for each need above.

<figure class="lab-figure">
<svg viewBox="0 0 660 208" width="660" height="208" role="img" aria-label="The trigger matrix mapping each GitHub event to the workflow it starts: pull request and push to main start ci.yml, a version tag starts release.yml, a pull request starts installer.yml, a push to docs plus a daily cron start pages.yml, and a manual workflow_dispatch starts cleanup-branches.yml.">
  <text x="125" y="22" text-anchor="middle" fill="var(--fg-muted)" font-size="10">trigger event</text>
  <text x="520" y="22" text-anchor="middle" fill="var(--fg-muted)" font-size="10">workflow (.github/workflows/)</text>
  <rect x="20" y="34" width="210" height="26" rx="5" fill="none" stroke="currentColor"/><text x="125" y="51" text-anchor="middle" fill="currentColor" font-size="10">pull_request · push main</text>
  <line x1="230" y1="47" x2="380" y2="47" stroke="currentColor"/><polygon points="380,43 390,47 380,51" fill="currentColor"/>
  <rect x="390" y="34" width="260" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="520" y="51" text-anchor="middle" fill="var(--accent)" font-size="11">ci.yml — merge gate</text>
  <rect x="20" y="68" width="210" height="26" rx="5" fill="none" stroke="currentColor"/><text x="125" y="85" text-anchor="middle" fill="currentColor" font-size="10">tag v*.*.*</text>
  <line x1="230" y1="81" x2="380" y2="81" stroke="currentColor"/><polygon points="380,77 390,81 380,85" fill="currentColor"/>
  <rect x="390" y="68" width="260" height="26" rx="5" fill="none" stroke="currentColor"/><text x="520" y="85" text-anchor="middle" fill="currentColor" font-size="11">release.yml</text>
  <rect x="20" y="102" width="210" height="26" rx="5" fill="none" stroke="currentColor"/><text x="125" y="119" text-anchor="middle" fill="currentColor" font-size="10">pull_request (not docs)</text>
  <line x1="230" y1="115" x2="380" y2="115" stroke="currentColor"/><polygon points="380,111 390,115 380,119" fill="currentColor"/>
  <rect x="390" y="102" width="260" height="26" rx="5" fill="none" stroke="currentColor"/><text x="520" y="119" text-anchor="middle" fill="currentColor" font-size="11">installer.yml — PR-only</text>
  <rect x="20" y="136" width="210" height="26" rx="5" fill="none" stroke="currentColor"/><text x="125" y="153" text-anchor="middle" fill="currentColor" font-size="10">push docs/** · daily cron</text>
  <line x1="230" y1="149" x2="380" y2="149" stroke="currentColor"/><polygon points="380,145 390,149 380,153" fill="currentColor"/>
  <rect x="390" y="136" width="260" height="26" rx="5" fill="none" stroke="currentColor"/><text x="520" y="153" text-anchor="middle" fill="currentColor" font-size="11">pages.yml</text>
  <rect x="20" y="170" width="210" height="26" rx="5" fill="none" stroke="currentColor"/><text x="125" y="187" text-anchor="middle" fill="currentColor" font-size="10">workflow_dispatch</text>
  <line x1="230" y1="183" x2="380" y2="183" stroke="currentColor"/><polygon points="380,179 390,183 380,187" fill="currentColor"/>
  <rect x="390" y="170" width="260" height="26" rx="5" fill="none" stroke="currentColor"/><text x="520" y="187" text-anchor="middle" fill="currentColor" font-size="11">cleanup-branches.yml</text>
</svg>
<figcaption>GopherTrunk's five workflows, each bound to the event that should start it — the trigger, not the workflow, is what makes a release deliberate and a chore scheduled.</figcaption>
</figure>

### `ci.yml` — the merge gate

Triggered on every `pull_request` and on `push` to `main`, with several jobs:

- **`build-test`** (Ubuntu): `go vet ./...`, a `gofmt` check that fails on any
  unformatted file, `go build ./...`, `go test -race -count=1 ./...`, then
  `make integration`.
- **`usb-windows`** and **`usb-macos`**: build/vet/test the pure-Go USB
  transport (`internal/sdr/rtlsdr/usb/...`) under `CGO_ENABLED=0` on
  *native* Windows and macOS runners — because that code has platform-specific
  backends (WinUSB, IOKit) that must compile everywhere.
- **`dvsi`**: builds and tests the patent-encumbered DVSI vocoder backend behind
  `-tags dvsi` using a mock transport, so the wire protocol is covered without
  the real chip.
- **`integration`**: runs `make test-integration` across the whole module so a
  future `//go:build integration` test in a new package can't silently skip CI.
- **`vulncheck`**: installs and runs `govulncheck ./...` — a failing CVE scan
  **blocks merge**.
- **`licenses`**: regenerates the dependency-license inventory with
  `go-licenses` (`make licenses`) and diffs it against the committed copy
  (currently `continue-on-error: true` while upstream has Go 1.25 false
  positives).

The ruleset at `.github/rulesets/main-branch-protection.json` makes
**`build-test`, `usb-windows`, and `usb-macos`** required, so a red build on any
of the three locks the merge button. Notably, `ci.yml` deliberately does *not*
use `paths-ignore` — a comment in the file explains that skipping a *required*
check would leave even docs-only PRs stuck as `mergeable_state=blocked`.

### `release.yml` — release automation on tags

Triggered by pushing a `v*.*.*` tag (or manually via `workflow_dispatch`). Three
platform jobs build with `CGO_ENABLED=0` — `windows` (installer + portable ZIPs,
amd64 and arm64), `linux` (amd64/arm64 tarballs), and `macos` (Intel/Apple
Silicon) — and a final `release` job flattens the artifacts, writes
`SHA256SUMS`, marks any `v0.x` tag as a pre-release, and publishes the GitHub
Release. Version metadata is injected at build time via `-ldflags`.

### `installer.yml` — a PR-only, expensive check

"Windows installer (PR)" builds the full Inno Setup installer on every pull
request, so a break in the Windows packaging surfaces *before* a release tag.
It's the textbook expensive/PR-only workflow: it `paths-ignore`s `**/*.md`,
`docs/**`, `LICENSE`, and `config.example.yaml` (a docs change can't break the
installer), and it sets `concurrency` with `cancel-in-progress: true` so a quick
rebase doesn't burn idle `windows-latest` minutes. It's *not* a required check,
precisely because it can be skipped on docs PRs.

### `pages.yml` — scheduled docs build + blog drip

Builds `docs/` as a Jekyll site and deploys to GitHub Pages. It triggers on
pushes that touch `docs/**` or `README.md`, **and on a daily `cron` at
`0 16 * * *` (11:00 America/Chicago)**. That schedule is what powers this very
blog series: posts are committed future-dated, Jekyll's `future: false` hides
them, and each daily run reveals the posts that have come due — one per day, no
draft branches. Its `concurrency` uses `cancel-in-progress: false` so a deploy is
never interrupted.

### `cleanup-branches.yml` — housekeeping

A manual (`workflow_dispatch`) job that lists feature branches whose PRs have
merged and deletes them, with a `dry_run` input (default `true`) so you preview
the plan in the run summary before anything is removed. It skips protected,
open-PR, and no-PR branches — exactly the careful, opt-in housekeeping pattern.

Swap Go for your stack and the shape is identical: one CI gate, one security
pass, tag-driven releases, a site build, a PR-only heavy check, and a scheduled
chore.

## FAQ

**What's the minimum CI workflow a project needs?**
One workflow, triggered on `pull_request`, that builds the project, runs the
tests, and checks formatting/linting. Make it a required status check and you've
captured most of CI's value.

**How do I stop a workflow from blocking merges when it doesn't apply?**
Be careful: a workflow skipped via `paths-ignore` does *not* report success, so
if it's a *required* check the PR stays blocked. Either don't make path-filtered
workflows required, or keep required checks on a job that always runs.

**What does `concurrency: cancel-in-progress` do?**
It cancels any in-flight run in the same concurrency group when a newer run
starts, so rapid pushes don't pile up redundant runs. Use `true` for CI/PR
checks; use `false` for deploys you don't want interrupted.

**How should releases be triggered — on push or on tag?**
On a version tag (e.g. `on: push: tags: ["v*.*.*"]`). Tagging is a deliberate
act, which keeps you from publishing a release on every commit to `main`.

**What runs a workflow on a schedule?**
An `on: schedule` trigger with a `cron` expression (in UTC). GitHub's scheduler
is best-effort and may be delayed under load, which is fine for daily chores like
a docs rebuild or a dependency audit.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Planning & Tracking Work — and Inviting Contributors]({{ '/blog/tutorials/build-in-the-open-06-issues-planning-inviting-contributors/' | relative_url }})
· Next →
[Part 8: Testing — How to Build and Write Tests]({{ '/blog/tutorials/build-in-the-open-08-testing-how-to-write-tests/' | relative_url }})
