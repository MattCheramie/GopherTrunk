# Prompt: Write the "Build in the Open" blog series

Write a 14-part blog series titled **"Build in the Open: GitHub + Claude Code
from Idea to Release"** for the GopherTrunk Jekyll blog. It teaches how to set
up a GitHub repo and use Claude Code to build software, idea → release.

## Ground rules
- Teach a **generic, transferable strategy** that works for any language/stack.
  In every post, explain the principle generically FIRST, then add a "How
  GopherTrunk does it" subsection using real artifacts from this repo as one
  worked example.
- Reuse the existing publishing pipeline — do not invent a new one. Posts go in
  `docs/_posts/`, future-dated; the daily cron in `.github/workflows/pages.yml`
  drips them out one per day (`future:false` in `docs/_config.yml`).

## Files to create
- 14 posts: `docs/_posts/2026-06-DD-build-in-the-open-NN-slug.md` dated
  consecutively **2026-06-18 → 2026-07-01** (NN = 01–14). Or write undated and
  run `scripts/schedule-series.py --apply 2026-06-18 docs/_posts/*build-in-the-open*.md`.
- Series landing page `docs/blog/series/build-in-the-open.md`, modeled on
  `docs/blog/series/sdr-internals.md` (layout: page, nav_group: Blog,
  permalink `/blog/series/build-in-the-open/`, Liquid loop filtering
  `series == "Build in the Open"` sorted by `series_part`, linking the
  **tutorials** category).

## Post URLs
Posts resolve to `/blog/tutorials/<slug>/` (permalink `/blog/:categories/:title/`
in `docs/_config.yml`). Use that pattern for all internal series-navigation links.

## Front matter (every post)
title (keyword-front-loaded "Build in the Open, Part N: …"), description
(≤155 chars, answer-first, keyword-rich), keywords, category: tutorials, tags,
author: Matt Cheramie, image: /assets/gophertrunk-logo.png,
series: "Build in the Open", series_part: N. Do NOT set `hide_ctas`.

## Body structure (serve skimmer, overviewer, deep-diver — inverted pyramid)
1. **TL;DR** blockquote + **Key takeaways** bullets at the very top (important
   info + one-sentence answer first — what AI search lifts).
2. Italic one-line series blurb + `## In this post` bullet map.
3. Full sections: generic technique, then "How GopherTrunk does it" worked
   example with real commands/code/file paths. Question-style headings.
4. `## FAQ` — 3–5 real "How do I / When should I / What is" Q&As, concise
   self-contained answers (SEO + answer-engine).
5. `## Series navigation` — Part N of 14, prev/next + back-to-Part-1 via
   `relative_url`.
The standard 3 site CTAs (Try / Support / Join) inject automatically via the
layout — just don't suppress them and keep posts multi-section.

## The 14 posts
1. Picking what to build: how pros decide (validate demand, scope an MVP).
2. Choosing your language, platforms & tech stack.
3. Brainstorming features with Claude + writing the README as a roadmap
   (incl. CLAUDE.md).
4. Git & GitHub fundamentals via the web interface.
5. Branching & the three ways to merge to main (many small commits→1 PR; 3–5
   large commits→1 PR; one squashed commit→1 PR) and when to use each.
6. Planning & tracking work + inviting contributors (Issues, labels, milestones,
   project boards, CONTRIBUTING/CoC/CODEOWNERS, collaborator roles).
7. GitHub Actions: which workflows to create and why.
8. Testing: how to build and write tests (pyramid, table-driven, fixtures/golden,
   fakes vs mocks, race, coverage, CI gate, writing tests with Claude Code).
9. Documentation done right: what lives where (Diátaxis).
10. Websites, support pages & GitHub Pages (custom domain, Jekyll, drip-release).
11. Releases: cadence, pre-release vs release, SemVer & changelogs.
12. Optimizing & securing your repository (About/topics/badges; branch protection,
    secret scanning, Dependabot/SCA, SECURITY.md, signed/checksummed releases).
13. Advanced Git & GitHub features (rebase/cherry-pick/bisect/reflog/worktrees;
    Discussions, wikis, code search, Codespaces, Packages/GHCR, Environments).
14. The CLI workflow & Claude Code as your daily driver (gh CLI; Claude Code CLI
    + web sessions, CLAUDE.md, slash commands, hooks).

## GopherTrunk reference facts to cite
Pure-Go `CGO_ENABLED=0` single static ~10MB binary, cross-compiles Linux/macOS/
Windows; Go toolchain pinned 1.25.11; README "What is this?" + Status snapshot
as a living roadmap; branch naming `claude/issue-NNN-desc`, Conventional Commits,
squash-on-merge; 5 workflows (ci.yml with 6 jobs incl. govulncheck + license
audit + multi-OS USB; release.yml; installer.yml PR-only with paths-ignore +
cancel-in-progress; pages.yml daily cron; cleanup-branches.yml); `make test`
(-race), `make integration` + `//go:build integration`, per-protocol
`make integration-cc-<proto>`, golden IQ in `samples/`, env-gated real-hardware
tests; `docs/` ~50 md + 180-entry encyclopedia + 30-lesson learning path;
gophertrunk.org via Jekyll + CNAME, landing page synthesized from README, SEO
plugins, FUNDING.yml (Sponsors + Ko-fi); SemVer v0.4.5, Keep-a-Changelog +
`## [Unreleased]`, tag `vX.Y.Z` → release.yml, `make release-dry-run`, ldflags
version injection, SHA256SUMS, prerelease before v1.0.0; rulesets
main-branch-protection.json + release-tags-protection.json; SECURITY.md threat
model + private advisories + bearer-token/constant-time crypto.

## Finish
Verify locally (Jekyll build with future dates visible: all 14 render, series
page lists all 14 in order, tutorials category + RSS include them), confirm
unique series_part 1–14, run the schedule-series.py dry run to confirm dates
06-18→07-01, then commit on `claude/github-claude-blog-series-lb1apw` with
`docs(blog): add 14-part "Build in the Open" GitHub + Claude Code series` and push.
