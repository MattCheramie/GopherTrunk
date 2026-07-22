---
title: "Build in the Open, Part 11: Releases — Cadence, Pre-Release vs. Release, SemVer & Changelogs"
description: How to ship software releases the right way — Semantic Versioning, pre-releases, git tags, Keep a Changelog notes, and automated, checksummed builds.
keywords: semantic versioning, semver, software releases, pre-release, git tags, changelog, keep a changelog, release notes, github releases, claude code
category: tutorials
tags: [github, releases, semver, changelog, ci-cd, claude-code]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 11
---

> **TL;DR:** A release is just a *tagged, named, downloadable snapshot* of your
> project plus notes that tell people what changed. Use Semantic Versioning
> (`MAJOR.MINOR.PATCH`) so the number means something, release often instead of
> hoarding changes, ship pre-releases (`alpha`/`beta`/`rc`, or any `0.x` while
> you're still stabilizing) to set expectations, keep a human-readable changelog
> in the [Keep a Changelog](https://keepachangelog.com/) format, and automate
> the build so a single `git tag` produces signed, checksummed artifacts.

**Key takeaways**

- SemVer encodes a *promise*: bump MAJOR for breaking changes, MINOR for new
  features, PATCH for fixes.
- Anything below `1.0.0` is implicitly a pre-release — the public API may still
  change. Ship a `0.x` or `-rc` build before you commit to `v1.0.0`.
- Maintain a `CHANGELOG.md` with an `## [Unreleased]` section you fill in as you
  go, so cutting a release is just renaming a heading.
- Let a tag trigger the build. Publishing artifacts by hand is how you ship the
  wrong binary at 2 a.m.
- Always ship checksums (`SHA256SUMS`) so users can verify what they downloaded.

*This is Part 11 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **What a release actually is** — and why "tag, notes, artifacts" is the whole
  game.
- **Semantic Versioning** — what each number promises, and how to bump it.
- **Pre-releases** — alpha/beta/rc, the `0.x` convention, and why you want one
  before `v1.0.0`.
- **Release cadence** — why shipping small and often beats the big-bang release.
- **Writing changelogs and release notes** people will actually read.
- **Automating the build** so a tag produces checksummed artifacts.
- **How GopherTrunk does it**, end to end, with real commands.

## What is a release, really?

Strip away the ceremony and a release is three things:

1. **A tag** — an immutable git pointer (`v0.4.5`) to one exact commit.
2. **A name and notes** — a human-readable summary of what changed.
3. **Artifacts** — the built binaries, archives, or packages users download,
   plus checksums so they can verify integrity.

Everything else — version policy, changelogs, automation — exists to make those
three things trustworthy and repeatable. If you get the tag right and the build
is reproducible from it, you can always reconstruct exactly what a user is
running.

## Semantic Versioning: make the number mean something

[Semantic Versioning](https://semver.org/) (SemVer) is a contract written into
the version number `MAJOR.MINOR.PATCH`:

- **MAJOR** (`1.x.x` → `2.0.0`): you broke backward compatibility. Users must
  read the notes before upgrading.
- **MINOR** (`1.2.x` → `1.3.0`): you added functionality, backward-compatibly.
  Safe to upgrade.
- **PATCH** (`1.2.3` → `1.2.4`): you fixed a bug, no API change. Always safe.

The discipline is what makes this useful. If a "patch" quietly changes behavior,
the number is lying and your users stop trusting it. When in doubt about whether
something is a feature or a fix, ask: *could this surprise someone who upgrades
without reading?* If yes, it's at least a MINOR.

<figure class="lab-figure">
<svg viewBox="0 0 660 168" width="660" height="168" role="img" aria-label="The three SemVer bump rules applied to version 1.2.3: a breaking change bumps MAJOR to 2.0.0 and resets minor and patch; a backward-compatible feature bumps MINOR to 1.3.0 and resets patch; a bug fix bumps PATCH to 1.2.4.">
  <rect x="270" y="12" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="27" text-anchor="middle" fill="currentColor" font-size="13">1.2.3</text>
  <text x="330" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">current version</text>
  <line x1="300" y1="46" x2="150" y2="84" stroke="currentColor"/><polygon points="153,79 143,86 156,88" fill="currentColor"/>
  <line x1="330" y1="46" x2="330" y2="84" stroke="currentColor"/><polygon points="326,84 330,94 334,84" fill="currentColor"/>
  <line x1="360" y1="46" x2="510" y2="84" stroke="currentColor"/><polygon points="504,79 517,86 507,88" fill="currentColor"/>
  <rect x="26" y="96" width="196" height="58" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="124" y="114" text-anchor="middle" fill="var(--accent)" font-size="11">MAJOR → 2.0.0</text>
  <text x="124" y="130" text-anchor="middle" fill="currentColor" font-size="9">breaking change</text>
  <text x="124" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="8">read notes before upgrade</text>
  <rect x="232" y="96" width="196" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="114" text-anchor="middle" fill="currentColor" font-size="11">MINOR → 1.3.0</text>
  <text x="330" y="130" text-anchor="middle" fill="currentColor" font-size="9">new feature, compatible</text>
  <text x="330" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="8">safe to upgrade</text>
  <rect x="438" y="96" width="196" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="536" y="114" text-anchor="middle" fill="currentColor" font-size="11">PATCH → 1.2.4</text>
  <text x="536" y="130" text-anchor="middle" fill="currentColor" font-size="9">bug fix, no API change</text>
  <text x="536" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="8">always safe</text>
</svg>
<figcaption>What each number promises: bump MAJOR for a breaking change, MINOR for a backward-compatible feature, PATCH for a fix — and reset the lower fields to zero.</figcaption>
</figure>

A subtle but important rule: **everything below `1.0.0` is a free-for-all.**
SemVer §4 says the public API should not be considered stable until `1.0.0`. A
`0.x` version is itself a signal — "I'm still figuring out the shape of this,
expect change." That's not a cop-out; it's an honest label.

## Pre-releases: alpha, beta, rc — and the 0.x convention

A **pre-release** is a version you ship knowing it's not the final cut. SemVer
gives you a suffix for this (`1.0.0-alpha`, `1.0.0-beta.2`, `1.0.0-rc.1`), and
the convention is a rough confidence ladder:

- **alpha** — feature-incomplete, expect bugs, for early testers.
- **beta** — feature-complete, hunting for bugs.
- **rc** (release candidate) — believed shippable; if nothing breaks, this
  *becomes* the release.

Two reasons to bother. First, it sets expectations honestly. Second — and this
is the one people skip — **a pre-release exercises your release machinery on
real infrastructure before it matters.** The first time you push a tag and watch
the build run is exactly when you find out your version-injection is broken or
your artifact paths are wrong. Far better to discover that on `v0.99.0` than on
`v1.0.0`.

## Release often, in small bites

The instinct to batch up "enough" changes into a big release is a trap. Large
releases are riskier (more surface area to regress), harder to write notes for,
and slower to get feedback on. Shipping small and often means each release is
easy to reason about, easy to roll back, and keeps users in the habit of
upgrading. Your cadence doesn't have to be fixed — "whenever there's something
worth shipping" is a perfectly good policy for a side project.

## Write a changelog humans will read

A `CHANGELOG.md` is the project's release diary. The
[Keep a Changelog](https://keepachangelog.com/) format is the de-facto standard
and it's dead simple:

- Newest version at the top.
- An `## [Unreleased]` section you append to *as you merge changes* — so the work
  is already done when you cut a release.
- Changes grouped under `### Added`, `### Changed`, `### Fixed`, `### Removed`,
  `### Deprecated`, `### Security`.

Write entries for *users*, not for git. "Fixed null pointer in handler" is a
commit message; "Scanner no longer crashes when a talkgroup has no name" is a
changelog entry. When you cut a release, you rename `[Unreleased]` to the new
version with a date and open a fresh `[Unreleased]`. That's it.

GitHub Releases can also auto-generate notes from merged PR titles, which is a
great complement to a hand-written changelog: the changelog tells the *story*,
the auto-notes give the *complete* list.

## Automate the build off the tag

Manual release builds are where mistakes live. The pattern that scales:

1. You push a tag matching a pattern (e.g. `v*.*.*`).
2. A CI workflow triggers on that tag, builds every target, injects the version
   into the binary, computes checksums, and creates the GitHub Release with the
   artifacts attached.

Inject the version at build time rather than hard-coding it in source — that way
the binary can report exactly which release and commit it came from. In Go this
is `-ldflags "-X pkg.Version=..."`; other ecosystems have equivalents. And always
publish a `SHA256SUMS` file so anyone can verify the download wasn't corrupted or
tampered with.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="The release timeline: an Unreleased changelog section fills up as PRs merge; pushing a v-prefixed tag triggers CI to build artifacts and checksums; a 0.x or -rc tag publishes as a pre-release, while a clean v1.0.0-plus tag publishes as the latest stable release, and the Unreleased heading is renamed to the dated version.">
  <line x1="24" y1="40" x2="636" y2="40" stroke="var(--fg-muted)"/><polygon points="636,36 646,40 636,44" fill="var(--fg-muted)"/>
  <rect x="24" y="52" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="99" y="70" text-anchor="middle" fill="currentColor" font-size="10">## [Unreleased]</text>
  <text x="99" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">filled as PRs merge</text>
  <text x="99" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Added · Fixed · Changed</text>
  <line x1="174" y1="75" x2="206" y2="75" stroke="currentColor"/><polygon points="206,71 216,75 206,79" fill="currentColor"/>
  <rect x="216" y="52" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="291" y="70" text-anchor="middle" fill="var(--accent)" font-size="10">git tag vX.Y.Z</text>
  <text x="291" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">on: push tags v*.*.*</text>
  <text x="291" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">triggers release.yml</text>
  <line x1="366" y1="75" x2="398" y2="75" stroke="currentColor"/><polygon points="398,71 408,75 398,79" fill="currentColor"/>
  <rect x="408" y="52" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="483" y="70" text-anchor="middle" fill="currentColor" font-size="10">CI build matrix</text>
  <text x="483" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">ldflags version stamp</text>
  <text x="483" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+ SHA256SUMS</text>
  <line x1="483" y1="98" x2="483" y2="120" stroke="currentColor"/><polygon points="479,120 483,130 487,120" fill="currentColor"/>
  <rect x="330" y="132" width="146" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="403" y="148" text-anchor="middle" fill="var(--fg-muted)" font-size="10">v0.x / -rc.1</text>
  <text x="403" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="8">prerelease: true</text>
  <rect x="490" y="132" width="146" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="563" y="148" text-anchor="middle" fill="var(--accent)" font-size="10">v1.0.0+</text>
  <text x="563" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="8">latest stable release</text>
</svg>
<figcaption>One tag drives the whole timeline: the <code>[Unreleased]</code> changelog fills as PRs merge, a pushed <code>v*.*.*</code> tag triggers the build, and the version pattern decides pre-release versus latest.</figcaption>
</figure>

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) is on **SemVer
`v0.4.5`** — squarely in the honest `0.x` "still stabilizing" range, with the
public API and config schema free to evolve until `v1.0.0`. Here's the full
release machinery, all real:

**The changelog is the source of truth.** `CHANGELOG.md` opens by declaring it
follows Keep a Changelog and Semantic Versioning, and the top of the file is
always an `## [Unreleased]` section. Contributors add a line there as part of
every PR — `CONTRIBUTING.md` literally lists "Add a line to `CHANGELOG.md` under
`## [Unreleased]`" as step 3 of sending a pull request. Cutting `v0.4.5` was just
promoting that section to a dated heading.

**A tag triggers everything.** `.github/workflows/release.yml` is wired to
`on: push: tags: ["v*.*.*"]` (plus a manual `workflow_dispatch` button). Pushing
`vX.Y.Z` builds Windows, Linux, and macOS artifacts for both `amd64` and `arm64`
in parallel, then a final `release` job flattens them and publishes the GitHub
Release.

**Version is injected at link time.** Each build computes the version, short
commit SHA, and UTC build time and bakes them in via ldflags:

```text
-ldflags "-X github.com/MattCheramie/GopherTrunk/internal/version.Version=${VERSION} \
          -X github.com/MattCheramie/GopherTrunk/internal/version.Commit=${COMMIT} \
          -X github.com/MattCheramie/GopherTrunk/internal/version.BuildTime=${BUILD_TIME}"
```

So `gophertrunk version` always reports exactly which release and commit you're
running.

**Rehearse before you tag.** `CONTRIBUTING.md`'s "Cutting a release" section
tells you to dry-run first:

```sh
make release-dry-run VERSION=v0.99.0
```

That builds `dist/dry-run/gophertrunk` with the same ldflags the real workflow
uses, runs `./gophertrunk version` against it, and writes a `SHA256SUMS` file —
so packaging or ldflags breakage surfaces *before* a tag is cut.

**Pre-release before v1.0.0.** The repo's stated policy is that "the first
production release should be a prerelease (e.g. `v0.99.0`)" so the full workflow
runs end-to-end against live GitHub Actions before `v1.0.0` ships. And the
release job encodes the SemVer pre-release rule in code: any `v0.x` tag *or* any
tag with a hyphen suffix (`v1.0.0-rc.1`) is published with `prerelease: true`;
only a clean `v1.0.0+` lands as the "latest" stable release.

**Checksums and tag protection.** The final job runs `sha256sum * > SHA256SUMS`
over every artifact, and every release ends with the line "All artifacts are
SHA-256-checksummed in `SHA256SUMS`." Tags themselves are protected: the
`.github/rulesets/release-tags-protection.json` ruleset matches `refs/tags/v*`
and blocks both `deletion` and `update`, so a published release tag can never be
silently moved to point at different code.

You don't need a multi-OS matrix to copy this. The portable moves are: pick
SemVer, keep an `[Unreleased]` changelog, trigger the build off a tag, inject the
version, ship checksums, and rehearse with a pre-release.

## FAQ

**When should I release v1.0.0?**
When you're willing to promise API stability — that you won't make breaking
changes without bumping to `2.0.0`. If you're still reshaping the public
interface, stay on `0.x`. There's no rush; plenty of widely-used software lives
happily at `0.x` for years.

**Do I need a changelog if GitHub auto-generates release notes?**
They serve different purposes. Auto-generated notes list every merged PR — great
for completeness. A hand-written `CHANGELOG.md` curates the *user-facing story*
and groups changes by type. The best setup uses both, which is exactly what
GopherTrunk does (`generate_release_notes: true` alongside `CHANGELOG.md`).

**What's the difference between a tag and a release?**
A tag is a git concept: an immutable pointer to a commit. A "release" is a
GitHub feature layered on top — a tag plus a title, notes, and downloadable
artifacts. You can have tags with no release, but not a release without a tag.

**Why inject the version instead of writing it in a file?**
A version file can drift from reality — someone forgets to bump it, or a local
build reports the wrong number. Injecting `Version`/`Commit`/`BuildTime` at link
time means the binary always tells the truth about exactly what produced it.

**Should pre-releases show up as the "latest" download?**
No. Mark them `prerelease: true` so users who just want the stable build aren't
handed a release candidate. GopherTrunk's release job does this automatically for
any `0.x` or hyphen-suffixed tag.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Websites, Support Pages & GitHub Pages]({{ '/blog/tutorials/build-in-the-open-10-websites-support-pages-github-pages/' | relative_url }})
· Next →
[Part 12: Optimizing & Securing Your Repository]({{ '/blog/tutorials/build-in-the-open-12-optimizing-securing-your-repository/' | relative_url }})
