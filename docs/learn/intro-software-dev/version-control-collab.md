---
slug: version-control-collab
title: Version control & collaboration
description: Why every project lives in version control — history, branching, safety — plus centralized vs distributed VCS, Git, and how teams collaborate with pull requests and review.
keywords: version control, Git, distributed version control, centralized version control, branching, pull requests, code review, collaboration, commit history, merge requests
level: beginner
status: full
faq:
  - q: "Why use version control for a project that's just me?"
    a: "Even solo, version control gives you a complete, undoable history of every change, the freedom to experiment on a branch without fear, and a safety net when something breaks — you can see exactly what changed and roll back. It also makes future collaboration or open-sourcing effortless because the history already exists. The cost is near zero; the upside is large."
  - q: "What's the difference between centralized and distributed version control?"
    a: "In a centralized system (like older Subversion) there is one central server holding the history, and you check files out from it. In a distributed system like Git, every clone is a full copy of the entire history, so you can commit, branch, and view history offline. Distributed systems are more resilient and enable flexible collaboration flows, which is a big reason Git won."
  - q: "Are pull requests a Git feature?"
    a: "Not exactly. Git itself handles commits, branches, and merging. Pull requests (or merge requests) are a feature of hosting platforms like GitHub and GitLab built on top of Git — they wrap a proposed merge with discussion, code review, and automated checks. Git provides the plumbing; the platform provides the collaboration workflow."
---
# Version control & collaboration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Version control is non-negotiable** — every serious project keeps a full, undoable history. **Git won** — distributed, fast, and the de facto standard. **Collaboration flows** — branches, pull requests, and code review let many people work without chaos.
</div>

If there's one tool every software project uses, it's **version control.** It records the complete history of your code — who changed what, when, and why — lets people work in parallel without overwriting each other, and gives you a safety net to undo mistakes. It's so foundational that "the project lives in version control" is just assumed. This lesson explains *why* that's true and sketches how teams collaborate on top of it. It's an intro: the [Git & GitHub path](/learn/git/) covers the mechanics in depth, and we'll point you there for the details.

## Why every project lives in version control

Imagine writing software with no version control: files named `decoder_final_v2_REALLY_final.go`, no record of why a line changed, and a teammate emailing you a zip that silently clobbers your afternoon's work. That's the world version control abolishes. A **version control system (VCS)** gives you four things that are hard to live without:

- **History** — a timestamped, attributed log of every change, so you can answer "what changed and why?" and trace a bug to the commit that introduced it.
- **Branching** — the ability to develop a feature or experiment in isolation, on a separate line, without destabilizing the working version.
- **Collaboration** — many people editing the same project at once, with the system merging their work and flagging genuine conflicts.
- **Safety** — every committed state is recoverable. Delete the wrong thing? Roll back. Break the build? Bisect the history to find the culprit.

These benefits aren't just for teams. Even working solo, a VCS turns "I broke something an hour ago and don't know what" into a two-command investigation. The cost of adopting it is almost nothing; the cost of skipping it compounds with every change.

## Centralized vs distributed

There are two broad architectures for a VCS, and the difference shaped the modern tooling.

**Centralized** systems (such as the older Subversion, or CVS before it) keep the single source of truth on a central server. You check out files, make changes, and commit back to the server. There's exactly one repository, and you generally need to be connected to it to do meaningful work. Simple to reason about, but the server is a single point of failure and offline work is limited.

**Distributed** systems (Git, Mercurial) give *every* clone a complete copy of the entire history. When you clone a repo, you get all the commits, all the branches, the whole timeline. You can commit, branch, view history, and diff entirely offline; you only need the network to share with others. This makes the system resilient — every clone is effectively a backup — and unlocks flexible workflows where people sync peer-to-peer or through one or more shared remotes.

| | Centralized | Distributed |
| --- | --- | --- |
| History location | One central server | Every clone has the full history |
| Offline work | Limited | Full commit/branch/log offline |
| Single point of failure | The server | None — every clone is a backup |
| Examples | Subversion, CVS | Git, Mercurial |

<div class="knowledge-check" data-quiz data-correct-msg="Right — in a distributed VCS like Git, every clone holds the complete history, so you can work and commit offline." markdown="0">
  <p class="knowledge-check__q">Quick check: what defines a distributed version control system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">All history lives only on a central server you must stay connected to</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every clone holds a complete copy of the project's history</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can only ever have one branch at a time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Git — the de facto standard

In practice, "version control" today almost always means **Git.** Created by Linus Torvalds in 2005 to manage Linux kernel development, Git is distributed, extremely fast, and ruthlessly good at branching and merging — which made experimentation cheap and collaboration scalable. It became the overwhelming default, and the hosting platforms built around it (GitHub, GitLab, Bitbucket) turned it into the backbone of modern collaboration and open source.

A few core concepts you'll meet everywhere:

- A **commit** is a snapshot of your project at a point in time, with a message explaining the change.
- A **branch** is a movable pointer to a line of development — cheap to create, so you make one per feature or fix.
- **Merging** combines the work from one branch into another, and Git only bothers you when two changes genuinely conflict.
- A **remote** is a shared copy (often on GitHub) that the team pushes to and pulls from.

We won't reteach Git here — the dedicated path does that properly. Start with [what version control is](/learn/git/what-is-version-control/) for the foundations and [branches](/learn/git/branches/) for the model that makes parallel work possible.

## How teams collaborate

Version control enables collaboration, but teams add a *workflow* on top to keep it orderly. The common shape, often called a feature-branch or pull-request workflow, looks like this:

1. **Branch** off the main line to work on a feature or fix in isolation, so the shared `main` branch always stays in a working state.
2. **Commit** your changes to that branch as you go, in small, well-described steps.
3. **Open a pull request** (PR) — or "merge request" on GitLab — proposing that your branch be merged into `main`.
4. **Review** — teammates read the change, leave comments, and request fixes. Automated checks (build and tests) usually run here too.
5. **Merge** once approved and green, folding the work into `main`.

The pull request is where collaboration becomes a conversation rather than a file dump. **Code review** — a second pair of eyes before code lands — catches bugs, spreads knowledge across the team, and keeps quality consistent. It's one of the highest-value habits in software, and it's nearly free.

This shows up directly in a project like GopherTrunk: a contributor adding support for a new radio protocol opens a branch, builds the decoder, and submits a PR. Reviewers can see *exactly* what changed, run the automated tests against sample captures, and discuss the approach — all without anyone risking the stability of `main`. For the full mechanics, see [pull requests](/learn/git/pull-requests/) in the Git path.

## Where to go deeper

This lesson is deliberately a tour, not a manual. Branching strategies, resolving merge conflicts, rebasing, and the day-to-day commands all live in the dedicated [Git & GitHub path](/learn/git/). If you take one action from this lesson, make it this: put your next project in Git from commit one. The habit pays for itself almost immediately, and it's the substrate that the rest of this module — [testing](/learn/intro-software-dev/testing/) and [build & CI/CD](/learn/intro-software-dev/build-and-ci-cd/) — builds on.

## Recap

- **Version control is foundational** — it gives you history, branching, collaboration, and safety; every serious project uses it.
- **Solo or team, it pays off** — even alone, an undoable history and fearless experimentation are worth the near-zero cost.
- **Centralized vs distributed** — centralized keeps history on one server; distributed (Git) gives every clone the full history and offline power.
- **Git is the standard** — distributed, fast, great at branching; the platforms around it power modern collaboration and open source.
- **Collaboration flows** — branch, commit, open a pull request, review, merge; code review is a high-value, low-cost habit.
- **Go deeper in the Git path** — this lesson is an intro; the [Git & GitHub path](/learn/git/) covers the mechanics properly.

Next up: how to know your code actually works — [testing: unit, integration & beyond](/learn/intro-software-dev/testing/).
