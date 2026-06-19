---
slug: version-control
title: Version control
entry_type: concept
category: testing-delivery
description: Version control is a system that records snapshots of a project's files over time so you can recall any earlier state, see who changed what and why, and collaborate without overwriting each other's work.
keywords: version control, VCS, Git, distributed version control, centralized, SVN, Mercurial, commit, branch, history, source control
aka: [version control "VCS"]
autolink: true
infobox:
  - { label: Category, value: "Developer tooling / collaboration" }
  - { label: Records, value: "Snapshots (commits) over time" }
  - { label: Designs, value: "Local, centralized, distributed" }
  - { label: Dominant tool, value: "Git (distributed)" }
  - { label: Others, value: "Mercurial, Subversion (SVN)" }
see_also: [ci-cd, build-systems, package-manager, semantic-versioning, refactoring]
related_lessons:
  - { title: "What is version control?", url: /learn/git/what-is-version-control/ }
external:
  - { title: "Version control — Wikipedia", url: https://en.wikipedia.org/wiki/Version_control }
---

**Version control** is a system that records snapshots of a project's files over time, so
you can recall any earlier state, see who changed what and why, and collaborate without
overwriting each other's work.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A timeline of commits, each a snapshot, with a branch splitting off and merging back." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="30" y1="60" x2="430" y2="60" stroke="currentColor" stroke-width="1.2"/>
    <circle cx="60" cy="60" r="8" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <circle cx="140" cy="60" r="8" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <circle cx="220" cy="60" r="8" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <circle cx="340" cy="60" r="8" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <circle cx="410" cy="60" r="8" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <path d="M220 60 C260 30 290 30 300 30" fill="none" stroke="currentColor" stroke-width="1.1"/><circle cx="300" cy="30" r="7" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M300 30 C320 30 330 60 340 60" fill="none" stroke="currentColor" stroke-width="1.1"/>
    <text x="230" y="95" font-size="8">commits over time, with a branch that merges back</text>
  </g>
</svg>
<figcaption>Version control strings snapshots into a history you can travel along, branch, and merge.</figcaption>
</figure>

## What it solves

Without version control, keeping old versions means copying and renaming whole folders —
`report_final_v2_FINAL.docx` — and merging a teammate's edits becomes copy-paste and hope.
A version control system replaces that with one working copy and a complete, queryable
**history** alongside it. Each meaningful point is recorded as a **commit**: a snapshot of
the project plus a message saying what changed and why. With that history you can restore
an earlier state, see who changed each line, branch off to try a risky idea safely, and
understand why a line exists by reading the commit that introduced it.

## Centralized vs distributed

Version control evolved through three designs. **Local** tools tracked files on a single
machine. **Centralized** systems (CVS, Subversion/SVN) keep the history on one server that
everyone checks in and out of — enabling collaboration, but with a single point of failure
and a network dependency for most operations. **Distributed** systems (Git, Mercurial)
give every contributor a *full clone* of the repository, history and all, so you can
commit, browse, and branch offline, and no single server's loss erases the work. Git,
built in 2005 for the Linux kernel, won on speed, cheap branching, integrity by content
hashing, and a thriving ecosystem of hosts like GitHub.

## Why it underpins everything

Version control is the backbone of modern development. It is what a
[CI/CD](/reference/ci-cd/) pipeline watches — every push triggers a
[build](/reference/build-systems/) and [test](/reference/unit-testing/) run. It stores the
lockfiles that make [dependency management](/reference/package-manager/) reproducible, and
it holds the tags that mark each release under
[semantic versioning](/reference/semantic-versioning/). Its safety net is also what makes
bold [refactoring](/reference/refactoring/) practical: any change can be reviewed,
reverted, or compared against history.
