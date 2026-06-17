---
title: "Build in the Open, Part 12: Optimizing & Securing Your Repository"
description: How to make your GitHub repo discoverable and safe — About box, badges, branch protection, secret scanning, Dependabot, SECURITY.md, and least-privilege tokens.
keywords: github repository security, branch protection, secret scanning, dependabot, sca, security.md, github badges, repository optimization, govulncheck, claude code
category: tutorials
tags: [github, security, repository, badges, dependabot, claude-code]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 12
---

> **TL;DR:** A good public repo does two jobs — it *sells itself* and it
> *protects itself*. Optimize discoverability with a filled-in About box, topics,
> a social-preview image, and honest status badges. Harden it with branch
> protection / required checks, secret scanning + push protection, automated
> dependency-CVE scanning (Dependabot / SCA), a `SECURITY.md` disclosure policy,
> checksummed releases, and least-privilege `GITHUB_TOKEN` permissions in your
> Actions. Most of this is free and takes an afternoon.

**Key takeaways**

- The About box, topics, and a social-preview image are the cheapest SEO and
  first-impression wins you'll ever get.
- Badges should be *honest signals* (CI passing, latest release, license), not
  decoration.
- Branch protection with required status checks stops broken code from reaching
  `main`.
- Turn on secret scanning + push protection so credentials can't be committed in
  the first place.
- Scan dependencies for known CVEs automatically; a `SECURITY.md` tells people
  how to report what you missed.
- Give Actions the *minimum* token permissions they need — default to
  read-only.

*This is Part 12 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **Optimizing for discovery** — the About box, topics, social preview, badges,
  and a tidy repo.
- **Securing the front door** — branch protection and required status checks.
- **Stopping secrets at the source** — secret scanning and push protection.
- **Watching your dependencies** — Dependabot, SCA, and CVE scanning.
- **A disclosure policy** — what `SECURITY.md` should say.
- **Least-privilege Actions** — locking down `GITHUB_TOKEN`.
- **How GopherTrunk does it**, with its real rulesets and CI checks.

## Half one: optimize for discovery

A repo nobody can find or understand is a tree falling in an empty forest. These
are low-effort, high-leverage fixes.

### Fill in the About box

The little gear next to "About" on your repo's homepage controls three things
that GitHub search and Google both index:

- **Description** — one sentence, what it *is* and what makes it different.
- **Website** — your docs or landing page (the one you built in
  [Part 10]({{ '/blog/tutorials/build-in-the-open-10-websites-support-pages-github-pages/' | relative_url }})).
- **Topics** — tags like `golang`, `sdr`, `cli`. These power GitHub topic pages
  and "explore" discovery. Add 5–10 relevant ones.

### Add a social-preview image

Settings → General → Social preview. Upload an image and every time your repo is
shared on social media, Slack, or Discord it renders a rich card instead of a
bare URL. This is a one-time upload that pays off forever.

### Use badges as honest signals

Badges at the top of your README communicate health at a glance: is CI passing,
what's the latest release, what license, what language version. Keep them
*truthful* — a green CI badge that's actually broken is worse than no badge.
Good additions include a code-quality/report-card badge and a docs link.

### Keep the repo tidy

Discoverability is also legibility. A clear directory structure, the standard
"repository health" files (README, LICENSE, CONTRIBUTING, CODE_OF_CONDUCT,
SECURITY, issue/PR templates), and a `.gitignore` that keeps build junk out all
make the repo feel maintained — which is itself a signal to potential users and
contributors.

## Half two: secure the repository

Optimization gets people in the door. Security keeps the door from being kicked
off its hinges.

### Branch protection and required checks

Protect `main` so nobody (including you on a bad day) can push broken code or
force-push history away. The essentials:

- Require changes to come through a pull request.
- Require your CI status checks to pass before merge.
- Block force-pushes (non-fast-forward) and branch deletion.

On GitHub this is "branch protection rules" or the newer, JSON-defined
**rulesets** — which have the advantage that you can commit them to the repo and
review changes to your protection policy like any other code.

### Secret scanning + push protection

Secret scanning watches for committed credentials (API keys, tokens) and alerts
you. **Push protection** goes further — it *blocks the push* before the secret
ever lands in history. Turn both on (free for public repos). The best secret is
the one that never gets committed; the second best is one you're told about
within seconds.

### Dependabot and dependency CVE scanning (SCA)

Your code is only as secure as its dependencies. **Software Composition Analysis
(SCA)** means scanning your dependency tree for known CVEs. Two complementary
tools:

- **Dependabot** — opens PRs to bump dependencies with known vulnerabilities,
  and can keep all deps current on a schedule.
- **A CVE scanner in CI** — fails the build if a vulnerable dependency is in the
  graph, so you find out at PR time, not in production.

### A SECURITY.md disclosure policy

When someone finds a vulnerability, they need a *private* way to tell you.
`SECURITY.md` documents that path. Good ones include: which versions are
supported, how to report privately (GitHub's private security advisories are the
standard), what's in and out of scope, and rough response timelines. This turns
a scary surprise into a managed process.

### Least-privilege Actions tokens

Every workflow run gets a `GITHUB_TOKEN`. By default it can be broad. Set
`permissions:` to the minimum each workflow needs — read-only for CI, `contents:
write` only for the job that publishes releases. A leaked or exploited workflow
with read-only scope can do far less damage than one with write-everything.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) treats both halves as
first-class. Here's the real configuration.

**Honest badges.** The README header carries six: a live CI status badge
(`actions/workflows/ci.yml/badge.svg`), a release badge
(`shields.io/github/v/release` with `include_prereleases&sort=semver` — so it
honestly shows the `0.x` pre-release state), an Apache-2.0 license badge, a
Go-version badge pulled from `go.mod`, a **Go Report Card** code-quality badge,
and a docs badge linking to gophertrunk.org. Every one is a live signal, not a
static image.

**Protection as committed rulesets.** Branch and tag protection live in the repo
as JSON, reviewable like code:

- `.github/rulesets/main-branch-protection.json` targets the default branch and
  requires pull requests, resolves review threads
  (`required_review_thread_resolution: true`), blocks `deletion` and
  `non_fast_forward`, and — crucially — requires three status checks to pass
  before merge: `build-test`, `usb-windows`, and `usb-macos`.
- `.github/rulesets/release-tags-protection.json` (from
  [Part 11]({{ '/blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/' | relative_url }}))
  matches `refs/tags/v*` and blocks both `deletion` and `update`, so published
  release tags are immutable.

**CVE scanning is a required check.** The CI workflow has a dedicated `vulncheck`
job that installs and runs `govulncheck ./...` — Go's official vulnerability
scanner — enumerating known CVEs across the direct and transitive dependency
graph. Because `build-test` and the USB jobs are required, a failing security
scan keeps a PR from merging. There's also a `licenses` job that regenerates the
third-party license inventory with `go-licenses`, backstopping the hand-curated
table.

**A real disclosure policy.** `SECURITY.md` is not boilerplate — it opens with a
*threat model* (an operator running the daemon on a private network or single
host they own), lists supported versions, and routes reports through **GitHub's
private security advisory** workflow with explicit "do not open a public issue"
guidance. It enumerates what's in scope (auth bypass, path traversal, SQL
injection, DoS, memory-safety bugs) and out (operator misconfiguration,
`auth.mode: disabled`), and publishes a severity-based response timeline
(critical: ≤7 days initial, ≤30 days fix).

**Crypto and secrets done right.** The API uses bearer-token auth with
*constant-time comparison* (`crypto/subtle.ConstantTimeCompare`), so
token-comparison side channels are out of scope by construction, per
`SECURITY.md`. And the one real build secret — the RadioReference app key — is
never committed; it's injected at build time via ldflags
(`-X ...radioreference.DefaultAppKey=${RR_APP_KEY}`) from a GitHub Actions
secret. The `release.yml` workflow also declares `permissions: contents: write`
explicitly rather than inheriting broad defaults, and ships SHA-256 checksums
with every release.

The portable lesson: commit your protection rules, make a CVE scan a required
check, write a real `SECURITY.md`, keep secrets out of source, and scope your
tokens tight.

## FAQ

**Are these features free?**
For public repositories, yes — branch protection, rulesets, secret scanning,
push protection, and Dependabot are all free on GitHub. CVE scanners like
`govulncheck` are free open-source tools you run in CI.

**What's the difference between Dependabot and a CVE scanner like govulncheck?**
Dependabot proactively opens PRs to *update* vulnerable or outdated
dependencies. A scanner like `govulncheck` runs in CI and *fails the build* when
a known-vulnerable dependency is present. They're complementary: one fixes,
one gates.

**Why use rulesets instead of classic branch protection?**
Rulesets can be exported as JSON and committed to the repo, so changes to your
protection policy are reviewable and versioned like any other code. GopherTrunk
keeps both `main` and tag protection under `.github/rulesets/`.

**What belongs in SECURITY.md?**
Supported versions, a private reporting channel (GitHub advisories are
standard), what's in and out of scope, and rough response timelines. A short
threat model up front helps reporters know what you actually care about.

**How do I keep secrets out of my repo?**
Never commit them. Store them as GitHub Actions secrets and inject them at build
or runtime, turn on push protection to block accidental commits, and prefer
build-time injection (like GopherTrunk's ldflags app-key) over checked-in config.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Releases — SemVer & Changelogs]({{ '/blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/' | relative_url }})
· Next →
[Part 13: Advanced Git & GitHub Features]({{ '/blog/tutorials/build-in-the-open-13-advanced-git-and-github-features/' | relative_url }})
