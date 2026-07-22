---
title: "Build in the Open, Part 4: Git & GitHub Fundamentals (Using the Web Interface)"
description: A beginner's Git mental model — commits, branches, remotes, the staging area — and how to create a repo, edit files, and open your first pull request entirely in the GitHub web UI.
keywords: git for beginners, github web interface, what is a commit, git branches, what is a remote, gitignore, first pull request, create a github repo, github
category: tutorials
tags: [github, git, getting-started, version-control, claude-code]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 4
---

> **TL;DR:** Git is a time machine for your files: a **commit** is a labeled
> snapshot, a **branch** is a movable line of snapshots you can experiment on
> safely, and a **remote** (like GitHub) is a shared copy everyone syncs to. You
> can do all the basics — create a repo, add a README/.gitignore/license, edit
> and commit files, make a branch, and open a pull request — entirely in the
> GitHub website, no command line required. Learn the mental model first; the
> buttons make sense after that.

**Key takeaways**

- A commit is a snapshot with a message; Git tracks *changes over time*, not
  just the latest files.
- Branches let you work without touching the stable version; `main` stays good
  while you experiment.
- A remote is the shared copy (GitHub). You "push" to share and "pull" to catch
  up.
- The whole beginner loop — edit, commit, branch, pull request — works in the
  browser.
- A `.gitignore` keeps build output and secrets *out* of your repo.

*This is Part 4 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **What Git actually tracks** — commits, the staging area, and why snapshots
  beat "final_v2_FINAL.zip".
- **Why branches exist** and how they keep `main` safe.
- **What a remote is** and how GitHub fits in.
- **The web-based loop**: create a repo, edit, commit, branch, open a PR — all in
  the browser.
- **.gitignore essentials** — what to keep out.
- **How GopherTrunk does it**, as a concrete example.

## What Git actually tracks

Forget the commands for a minute. Git is a system that records the **history of
your files as a series of snapshots**. Each snapshot is a **commit**: the exact
state of every tracked file at that moment, plus a short message describing what
changed and why ("Add login form", "Fix off-by-one in parser").

This is the core idea that makes Git worth learning: you're not saving *files*,
you're saving *the story of how they changed*. You can look at any past commit,
compare two of them, or undo back to one — without ever resorting to a folder
full of `report-final-v3-actually-final.docx`.

One subtlety beginners hit: between "I changed a file" and "it's in a commit"
there's a middle step called the **staging area**. You *stage* the specific
changes you want to include, then *commit* them together. It lets you commit one
logical change at a time even if you edited several files. (In the GitHub web UI
this is mostly hidden — committing in the browser stages and commits in one
action — but it's worth knowing the term exists.)

## Why branches exist

A **branch** is a movable pointer to a line of commits. Every repo has a default
branch, almost always called `main`, which holds the version you consider good.

Branches matter because they let you work on something risky or unfinished
**without putting `main` at risk**. You create a branch off `main`, make commits
on it, and `main` stays exactly as it was. If the experiment works, you merge it
back; if it doesn't, you throw the branch away and nothing of value is lost. This
is why teams (and Claude Code) work on feature branches: `main` stays
releasable while work happens to the side.

The mental picture: `main` is the trunk of a tree; a branch grows off it, gets
its own commits, and eventually either grafts back in (merge) or gets pruned.

## What a remote is

So far everything could live on your laptop. A **remote** is a copy of the repo
hosted somewhere shared — GitHub being the most common. The remote is how you
back up your work, collaborate, and publish.

Two verbs connect you to it:

- **Push** — send your local commits up to the remote so others (and your other
  machines) can see them.
- **Pull** — bring the remote's new commits down to your copy so you're up to
  date.

When you work entirely in the GitHub website, *the remote is the only copy you're
touching* — every edit and commit happens directly on GitHub, so there's no
separate push/pull step. That's exactly why the web interface is such a friendly
on-ramp: you get commits, branches, and pull requests without yet installing Git
or learning the command line.

<figure class="lab-figure">
<svg viewBox="0 0 640 176" width="640" height="176" role="img" aria-label="The local repository and the GitHub remote as two synced copies. Locally, edited working files are staged and then committed into the repo's history. Push sends local commits up to the GitHub remote; pull brings the remote's new commits down; clone makes the first local copy from the remote.">
  <rect x="14" y="40" width="270" height="120" rx="8" fill="none" stroke="currentColor"/>
  <text x="149" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="9">local repo · your laptop</text>
  <rect x="30" y="70" width="72" height="34" rx="5" fill="none" stroke="currentColor"/><text x="66" y="90" text-anchor="middle" fill="currentColor" font-size="9">working files</text>
  <line x1="102" y1="87" x2="126" y2="87" stroke="currentColor"/><polygon points="126,83 136,87 126,91" fill="currentColor"/>
  <rect x="136" y="70" width="60" height="34" rx="5" fill="none" stroke="currentColor"/><text x="166" y="86" text-anchor="middle" fill="currentColor" font-size="9">staging</text><text x="166" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="8">area</text>
  <line x1="196" y1="87" x2="220" y2="87" stroke="currentColor"/><polygon points="220,83 230,87 220,91" fill="currentColor"/>
  <rect x="230" y="70" width="42" height="34" rx="5" fill="none" stroke="currentColor"/><text x="251" y="90" text-anchor="middle" fill="currentColor" font-size="9">commit</text>
  <text x="149" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="8">history of snapshots</text>
  <rect x="430" y="40" width="196" height="120" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="528" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">GitHub remote (origin)</text>
  <text x="528" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the shared copy</text>
  <line x1="288" y1="76" x2="426" y2="76" stroke="var(--accent)"/><polygon points="426,72 436,76 426,80" fill="var(--accent)"/>
  <text x="358" y="70" text-anchor="middle" fill="var(--accent)" font-size="9">push</text>
  <line x1="426" y1="106" x2="288" y2="106" stroke="currentColor"/><polygon points="288,102 278,106 288,110" fill="currentColor"/>
  <text x="358" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">pull</text>
  <line x1="426" y1="142" x2="288" y2="142" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="288,138 278,142 288,146" fill="var(--fg-muted)"/>
  <text x="358" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="8">clone (first copy)</text>
</svg>
<figcaption>The local repo and the GitHub remote are two synced copies: you stage and commit locally, then <code>push</code> commits up and <code>pull</code> new ones down, while <code>clone</code> makes the first local copy. Working purely in the web UI, the remote is the only copy you touch.</figcaption>
</figure>

## The whole loop in the GitHub web interface

Here's the complete beginner workflow, no terminal required.

**1. Create the repository.** On GitHub, click **New repository**, give it a
name, choose public or private, and — importantly — check the boxes to **add a
README**, a **.gitignore** (pick the template for your language), and a
**license**. Those three files give you a real starting commit instead of an
empty repo.

**2. Edit a file in the browser.** Open any file and click the pencil
(**Edit**) icon. GitHub gives you a text editor right in the page. Make your
change.

**3. Commit it.** Scroll down to the **Commit changes** box. Write a short,
descriptive message, choose "commit directly to `main`" *or* "create a new
branch for this commit," and click commit. That's a real commit, recorded in
history.

**4. Create a branch.** Use the branch dropdown (top-left of the file list, says
`main`), type a new branch name, and press enter. You're now working off to the
side. Make your edits and commits here; `main` is untouched.

**5. Open your first pull request (PR).** A **pull request** proposes merging your
branch's changes into `main`. After committing to a branch, GitHub shows a
**Compare & pull request** prompt — click it, write a title and a short
description of *why* the change matters, and click **Create pull request**. Now
your change is visible, reviewable, and (once approved) mergeable. PRs are the
heart of collaborating on GitHub, and we'll dig into merge strategies in
[Part 5]({{ '/blog/tutorials/build-in-the-open-05-three-ways-to-merge-to-main/' | relative_url }}).

That's the entire loop: **edit → commit → branch → pull request**, all from a
web browser.

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="The beginner GitHub web loop, all in the browser: edit a file with the pencil icon, commit the change with a message, create a branch to work off to the side while main stays safe, and open a pull request proposing the branch be merged into main. Every step happens directly on the remote.">
  <rect x="14" y="52" width="112" height="44" rx="6" fill="none" stroke="currentColor"/><text x="70" y="72" text-anchor="middle" fill="currentColor" font-size="10">edit file</text><text x="70" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pencil icon</text>
  <line x1="126" y1="74" x2="158" y2="74" stroke="currentColor"/><polygon points="158,70 168,74 158,78" fill="currentColor"/>
  <rect x="168" y="52" width="112" height="44" rx="6" fill="none" stroke="currentColor"/><text x="224" y="72" text-anchor="middle" fill="currentColor" font-size="10">commit</text><text x="224" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">with a message</text>
  <line x1="280" y1="74" x2="312" y2="74" stroke="currentColor"/><polygon points="312,70 322,74 312,78" fill="currentColor"/>
  <rect x="322" y="52" width="112" height="44" rx="6" fill="none" stroke="currentColor"/><text x="378" y="72" text-anchor="middle" fill="currentColor" font-size="10">new branch</text><text x="378" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">main stays safe</text>
  <line x1="434" y1="74" x2="466" y2="74" stroke="var(--accent)"/><polygon points="466,70 476,74 466,78" fill="var(--accent)"/>
  <rect x="476" y="52" width="130" height="44" rx="6" fill="none" stroke="var(--accent)"/><text x="541" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">pull request</text><text x="541" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">propose merge → main</text>
  <text x="310" y="28" text-anchor="middle" fill="var(--fg-muted)" font-size="9">all in the browser — the remote is the only copy</text>
</svg>
<figcaption>The whole beginner loop in the browser: edit → commit → new branch → pull request, every step happening directly on the GitHub remote.</figcaption>
</figure>

## .gitignore essentials

Not everything belongs in a repo. A **`.gitignore`** file lists patterns Git
should *not* track. Two categories matter most:

- **Generated output** — compiled binaries, build folders, dependency caches.
  These are produced from your source, so committing them just bloats the repo
  and causes pointless conflicts. Examples: `node_modules/`, `dist/`, a compiled
  binary, `*.test`.
- **Secrets and local junk** — `.env` files with credentials, editor settings
  (`.idea/`, `.vscode/`), OS cruft (`.DS_Store`). Secrets *especially* must never
  be committed; once pushed, treat them as compromised.

The GitHub "new repository" flow offers a language-appropriate `.gitignore`
template, which is a great starting point you then tweak as your project grows.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) follows these exact
fundamentals at scale.

- **`main` plus feature branches.** The stable, releasable version lives on
  `main`; all work happens on short-lived feature branches that merge back via
  pull requests. `main` always stays in a shippable state.

- **A clear branch-naming convention.** Branches follow a
  `<topic>/<short-description>` shape — and for AI-assisted work the project uses
  `claude/issue-NNN-desc` (for example, a branch tied to issue 698). A consistent
  naming scheme makes it obvious at a glance what a branch is for and which issue
  it addresses. The
  [CONTRIBUTING guide](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)
  spells the convention out.

- **A `.gitignore` tuned for its stack.** GopherTrunk's
  [`.gitignore`](https://github.com/MattCheramie/GopherTrunk/blob/main/.gitignore)
  keeps Go build output out of the repo — `/bin/`, `/dist/`, stray `gophertrunk`
  binaries, `*.test`, `*.out`, `*.prof` — alongside local secrets (`.env`,
  `.env.*`), editor folders (`.idea/`, `.vscode/`), OS files (`.DS_Store`), and
  the Jekyll docs-site build output (`docs/_site/`, `.jekyll-cache/`). The web
  console has its own `web/.gitignore` ignoring `node_modules/` and the built
  `dist/`. Generated and secret stuff stays out; source stays in.

You don't need a project this size to use the same moves. Create a repo with a
README, license, and `.gitignore`; keep `main` stable; do your work on a named
branch; and open a PR — all from the browser if you like. The command line can
come later.

## FAQ

**What is the difference between Git and GitHub?**
Git is the version-control tool that records your file history as commits.
GitHub is a website that hosts Git repositories — a *remote* — adding
collaboration features like pull requests, issues, and code review on top. You
can use Git without GitHub, but GitHub needs Git underneath.

**Do I need to install Git or use the command line to start?**
No. You can create a repository, edit and commit files, make branches, and open
pull requests entirely in the GitHub website. The web interface is a friendly
on-ramp; the command line is worth learning later but isn't required to begin.

**What is a pull request and why use one?**
A pull request proposes merging the commits on one branch into another (usually
into `main`). It makes your change visible and reviewable before it lands, which
is how teams collaborate safely and keep `main` stable. You create one from the
branch dropdown's "Compare & pull request" prompt.

**What should go in a .gitignore file?**
Anything that's generated or secret: compiled binaries and build folders
(`dist/`, `node_modules/`, `*.test`), dependency caches, credential files
(`.env`), editor configs (`.idea/`, `.vscode/`), and OS cruft (`.DS_Store`).
Committed source stays in; everything reproducible or sensitive stays out.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Brainstorming Features with Claude & Writing the README as Your Roadmap]({{ '/blog/tutorials/build-in-the-open-03-brainstorm-with-claude-readme-roadmap/' | relative_url }})
· Next →
[Part 5: Branching & the Three Ways to Merge to Main]({{ '/blog/tutorials/build-in-the-open-05-three-ways-to-merge-to-main/' | relative_url }})
