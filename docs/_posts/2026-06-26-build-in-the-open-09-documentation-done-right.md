---
title: "Build in the Open, Part 9: Documentation Done Right — What Lives Where"
description: How to organize project documentation — README, CONTRIBUTING, SECURITY, CHANGELOG, in-repo docs — using the Diátaxis framework so docs are findable and don't rot.
keywords: project documentation, README, CONTRIBUTING, Diátaxis, docs framework, technical writing, changelog, code of conduct, open source docs, claude code, github
category: tutorials
tags: [github, claude-code, documentation, writing, open-source]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 9
---

> **TL;DR:** Documentation isn't one thing — it's several kinds with different
> jobs and different homes. The README is your front door; standard files like
> CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, CHANGELOG, and LICENSE each answer one
> recurring question; everything deeper lives in an in-repo `/docs` folder. Use
> the **Diátaxis** framework — tutorials, how-to guides, reference, explanation —
> to decide what kind of doc you're writing and where it belongs. Keep docs next
> to the code so they version and rot together, and write each piece for one
> specific reader.

**Key takeaways**

- Docs come in types; put each type where readers expect to find it.
- The README is the front door — what, why, and a fast path to "it works."
- Diátaxis splits docs into tutorials, how-to, reference, and explanation.
- Keep docs in the repo, next to the code, so they version with it and rot less.
- Every doc has one reader and one job — write to that, not to everyone at once.

*This is Part 9 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **The standard files** every repo should have — and who each is for.
- **What goes in the README** versus everywhere else.
- **The Diátaxis framework** for organizing the deeper `/docs`.
- **Why docs belong next to the code**, and how to keep them from rotting.
- **How GopherTrunk does it**, as a concrete example you can copy.

## The standard files and who they're for

A handful of files have become conventions because each answers a question
someone will inevitably ask. GitHub even surfaces several of them in its UI:

- **README** — *"What is this and how do I start?"* The front door. Everyone
  reads it first.
- **CONTRIBUTING** — *"How do I help?"* Build, test, branch, and PR conventions
  for would-be contributors.
- **SECURITY** — *"I found a vulnerability, now what?"* The private disclosure
  process, so bugs don't get reported publicly.
- **CODE_OF_CONDUCT** — *"What behaviour is expected here?"* The community
  ground rules.
- **CHANGELOG** — *"What changed between versions?"* For users deciding whether
  to upgrade (more on this in Part 11).
- **LICENSE** — *"What am I allowed to do with this?"* The legal terms. Without
  it, the default is "no rights granted."

You don't need all of them on day one, but knowing the slot each fills means you
never have to wonder *where* a given piece of information should go.

## What belongs in the README?

The README is the highest-traffic page you will ever write, so it earns the most
care. It should answer, fast and in order:

1. **What is this?** One or two sentences, no jargon.
2. **Why does it exist / why would I use it?** The problem it solves.
3. **How do I get to "it works"?** Install or run, in the fewest steps possible.
4. **Where do I go next?** Links out to the deeper docs — *not* the docs
   themselves.

The README's job is to get someone oriented and then *hand them off*. The
moment it tries to be the complete manual, it becomes too long to be the front
door. Link out to `/docs` for everything past "hello world."

## The Diátaxis framework: four kinds of docs

Past the README, documentation gets disorganized because people mix four
fundamentally different things into one page. **Diátaxis** is a widely used
framework that names them and keeps them apart:

- **Tutorials** — *learning-oriented.* A guided lesson for a newcomer: "build
  your first X." The reader is a student; success is that they finish and feel
  capable.
- **How-to guides** — *task-oriented.* Steps to accomplish a specific goal the
  reader already has: "how to configure TLS." The reader knows what they want.
- **Reference** — *information-oriented.* Dry, complete, accurate descriptions:
  every flag, every field, every endpoint. The reader is looking something up.
- **Explanation** — *understanding-oriented.* The why and the how-it-works:
  architecture, design decisions, background. The reader wants the mental model.

The power of Diátaxis is diagnostic: when a doc feels muddled, it's usually
trying to be two of these at once. A tutorial bloated with reference detail
loses the beginner; a reference padded with tutorial hand-holding annoys the
expert. Split them, and each gets clearer.

<figure class="lab-figure">
<svg viewBox="0 0 520 236" width="520" height="236" role="img" aria-label="The Diátaxis framework as four quadrants: tutorials are learning-oriented, how-to guides are task-oriented, explanation is understanding-oriented and reference is information-oriented. The top row is practical and the bottom row is theoretical; the left column serves study and the right column serves work.">
  <text x="245" y="20" text-anchor="middle" fill="var(--fg-muted)" font-size="9">practical — steps you follow</text>
  <rect x="50" y="30" width="190" height="82" rx="6" fill="none" stroke="currentColor"/>
  <text x="145" y="66" text-anchor="middle" fill="var(--accent)" font-size="12">Tutorials</text>
  <text x="145" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9">learning-oriented</text>
  <rect x="250" y="30" width="190" height="82" rx="6" fill="none" stroke="currentColor"/>
  <text x="345" y="66" text-anchor="middle" fill="currentColor" font-size="12">How-to guides</text>
  <text x="345" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9">task-oriented</text>
  <rect x="50" y="120" width="190" height="82" rx="6" fill="none" stroke="currentColor"/>
  <text x="145" y="156" text-anchor="middle" fill="currentColor" font-size="12">Explanation</text>
  <text x="145" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="9">understanding-oriented</text>
  <rect x="250" y="120" width="190" height="82" rx="6" fill="none" stroke="currentColor"/>
  <text x="345" y="156" text-anchor="middle" fill="currentColor" font-size="12">Reference</text>
  <text x="345" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="9">information-oriented</text>
  <text x="245" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="9">theoretical — knowledge you absorb</text>
  <text x="16" y="145" fill="var(--fg-muted)" font-size="9" transform="rotate(-90 16 145)">study ← → work</text>
</svg>
<figcaption>Diátaxis names four kinds of doc so each stays pure; a page that feels muddled is usually two quadrants at once and should be split.</figcaption>
</figure>

## Keep docs next to the code

Wherever it's reasonable, keep documentation **in the repository**, in a `/docs`
folder, versioned alongside the code:

- **It versions with the code.** A doc change rides in the same commit and PR as
  the behaviour change, so the docs for v2 describe v2.
- **It's reviewable.** Doc changes go through the same review as code changes —
  reviewers can flag a stale instruction the way they'd flag a bug.
- **It rots less.** Docs in a separate wiki or external site drift, because
  nobody updates them when they change the code. Docs in the diff are right
  there, demanding to be updated.

The discipline that keeps docs alive is a simple rule in your CONTRIBUTING file:
*if your change alters user-visible behaviour, update the docs in the same PR.*
That turns documentation from a someday chore into part of "done."

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) treats docs as a
first-class part of the repo, and the layout maps cleanly onto everything above:

- **The README is the front door — and literally so.** Its
  [Pages landing page]({{ '/' | relative_url }}) is *synthesized from the README
  at build time* (see Part 10), so the front door and the homepage never drift
  apart. The README links out to the deeper docs rather than absorbing them.

<figure class="lab-figure">
<svg viewBox="0 0 640 120" width="640" height="120" role="img" aria-label="The source-to-site build flow: README.md is synthesized into docs/index.md, which sits alongside the docs folder of learn, reference and guides pages; Jekyll builds those Markdown sources; and GitHub Pages serves the result at gophertrunk.org.">
  <rect x="10" y="34" width="120" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="70" y="58" text-anchor="middle" fill="currentColor" font-size="11">README.md</text>
  <text x="70" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the front door</text>
  <text x="149" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">synthesize</text>
  <line x1="130" y1="60" x2="162" y2="60" stroke="currentColor"/><polygon points="162,56 168,60 162,64" fill="currentColor"/>
  <rect x="168" y="34" width="150" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="243" y="55" text-anchor="middle" fill="currentColor" font-size="11">docs/</text>
  <text x="243" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="8">index.md · learn</text>
  <text x="243" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reference · guides</text>
  <line x1="318" y1="60" x2="350" y2="60" stroke="currentColor"/><polygon points="350,56 356,60 350,64" fill="currentColor"/>
  <rect x="356" y="34" width="110" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="411" y="58" text-anchor="middle" fill="currentColor" font-size="11">Jekyll</text>
  <text x="411" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">build</text>
  <line x1="466" y1="60" x2="498" y2="60" stroke="currentColor"/><polygon points="498,56 504,60 498,64" fill="currentColor"/>
  <rect x="504" y="34" width="126" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="567" y="58" text-anchor="middle" fill="var(--accent)" font-size="11">gophertrunk.org</text>
  <text x="567" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="9">GitHub Pages</text>
</svg>
<figcaption>One source of truth: the README is synthesized into <code>docs/index.md</code> at build time, so the homepage and the front door can never drift apart.</figcaption>
</figure>

- **The standard files are all present.** `CONTRIBUTING.md` documents the build,
  test, branch, and PR conventions (including the table of `make` targets and the
  opt-in hardware-test env vars); `SECURITY.md` carries the threat model and
  private-disclosure process; `CHANGELOG.md` follows Keep a Changelog with an
  `## [Unreleased]` section; `LICENSE` sets the terms.
- **The `docs/` folder is large and structured** — around **50 markdown files**.
  It's clearly organized along Diátaxis lines:
  - **Tutorials / learning:** a **30-lesson learning path** under `docs/learn/rf-sdr/`
    (from `what-is-sdr.md` through `clock-recovery.md`, `digital-voice.md`, and a
    `glossary.md`) — the learning-oriented tier.
  - **Reference:** a **~180-entry Field Guide** under
    `docs/reference/` — the information-oriented tier you look things up in.
  - **How-to guides:** install guides per platform (`install-linux.md`,
    `install-macos.md`, `install-windows.md`), getting-started pages, and
    operational guides (`guide-basics.md`, `guide-intermediate.md`,
    `guide-advanced.md`, `hardening.md`).
  - **Explanation:** `architecture.md` for the design, plus living-status docs
    `status.md` and `roadmap.md` for where the project is and where it's going.
- **Everything is reachable from the README**, which links out to install, the
  guides, the learning path, the reference, and the status/roadmap docs — the
  front door pointing at every room in the house.

You don't need 50 files to apply this. Even a five-file project benefits from a
README that hands off, the standard community files, and a `/docs` folder where
each page knows whether it's a tutorial, a how-to, reference, or explanation.

## FAQ

**What's the minimum documentation a project needs?**
A README that says what the project is and how to run it, and a LICENSE so people
know what they're allowed to do. Add CONTRIBUTING and SECURITY the moment you
invite outside contributors, and a CHANGELOG when you start cutting releases.

**What is the Diátaxis framework?**
A way to organize documentation into four distinct types — tutorials (learning),
how-to guides (tasks), reference (lookup), and explanation (understanding) — each
written for a different reader need. Its main value is spotting when one doc is
trying to do two of those jobs and should be split.

**Should documentation live in the repo or on a separate site?**
Keep the source in the repo, next to the code, so docs version and get reviewed
alongside changes. You can still *publish* them to a website (Part 10 covers
that) — but the source of truth belongs in the diff, where it's least likely to
rot.

**How do I keep docs from going stale?**
Make updating them part of "done." A line in CONTRIBUTING — "update the docs in
the same PR as any user-visible change" — plus reviewers who enforce it does more
than any tooling. Docs that travel with the code stay closer to the truth.

**Where does the README stop and `/docs` begin?**
The README orients and hands off: what, why, get-it-running, and links onward.
Anything past the first successful run — full configuration, guides, reference,
architecture — belongs in `/docs`, so the front door stays short enough to be a
front door.

## Series navigation

**Part 9 of 14** · ←
[Part 8]({{ '/blog/tutorials/build-in-the-open-08-testing-how-to-write-tests/' | relative_url }})
· Next →
[Part 10: Websites, Support Pages & GitHub Pages]({{ '/blog/tutorials/build-in-the-open-10-websites-support-pages-github-pages/' | relative_url }})
