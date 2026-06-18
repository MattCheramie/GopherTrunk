---
slug: what-is-version-control
title: Why version control?
description: A plain-language introduction to version control — what a VCS is, the pain it removes, and why distributed Git became the default tool for tracking your work over time.
keywords: version control, what is version control, VCS, Git, distributed version control, centralized version control, SVN, Mercurial, source control, why use Git
level: beginner
status: full
faq:
  - q: What is version control in simple terms?
    a: Version control is a system that records changes to your files over time so you can recall any earlier version, see who changed what, and work on the same project alongside other people without overwriting each other. Think of it as an unlimited, organised undo history for an entire project rather than a single document.
  - q: What is the difference between Git and GitHub?
    a: Git is the version control *tool* that runs on your computer and tracks your project's history. GitHub is a *website* that hosts Git repositories online so you can back them up and collaborate. You can use Git with no internet and no GitHub account at all. GitHub is one of several hosts (GitLab and Bitbucket are others) built around Git.
  - q: Is version control just a backup?
    a: No. A backup captures the current state of your files. Version control captures the full *history* of how they got there — every snapshot, with a message explaining why — and lets you branch, merge, and compare. It complements backups but does not replace off-site backups of your machine.
  - q: What does "distributed" mean for Git?
    a: In a distributed VCS, every clone is a complete copy of the project, including its entire history. You can commit, branch, and browse history offline, and there is no single server whose failure loses your work. Centralized systems like SVN keep the history on one server that you must reach to do most operations.
---

# Why version control?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **version control system (VCS)** records **snapshots** of your project over time so
you can travel back to any earlier state, see exactly what changed and why, and
collaborate without stepping on each other. It replaces fragile habits like
`report_final_v2_FINAL.docx` and emailing zip files. **Git** is a **distributed**
VCS: every copy holds the full history, so you can work offline and branch cheaply.
Git is the tool; **Git is not GitHub**, and it is not a backup service.
</div>

This is the first lesson of the path, and it's the *why* behind everything that
follows. By the end you'll understand what a VCS actually does, the problems it
solves, and why Git in particular became the near-universal default.

## What problem does version control solve?

Picture a project folder without version control. To keep an old version "just in
case," you copy the whole folder and rename it. Soon you have a graveyard:

```text
report.docx
report_v2.docx
report_final.docx
report_final_v2_FINAL.docx
report_final_v2_FINAL_actually.docx
```

Which one is current? What changed between two of them? If a teammate emails you
*their* copy, how do you merge their edits with yours? You can't, reliably — you
end up copy-pasting and hoping. And if you delete the wrong file, the work is gone.

A version control system removes all of that pain. It keeps **one** working copy of
the project and a complete, queryable **history** alongside it. You no longer name
versions by hand; the system tracks them for you.

## What is a snapshot, and what is history?

Each time you reach a meaningful point, you record a **commit** — a snapshot of the
whole project at that moment, plus a short message saying what you did and why.
String those snapshots together and you get the project's **history**: a timeline
you can travel along.

With that history you can:

| Want to… | Version control lets you… |
|----------|---------------------------|
| Undo a change from last week | Restore an earlier snapshot |
| Know who broke something | See who changed each line, and when |
| Try a risky idea safely | Branch off, experiment, throw it away |
| Understand *why* a line exists | Read the commit message that introduced it |

You'll learn to make these snapshots in [Staging & commits](/learn/git/staging-and-commits/)
and to explore them in [Viewing history](/learn/git/viewing-history/).

## Local, centralized, and distributed VCS

Version control evolved through three broad designs:

- **Local** — the earliest tools (like RCS) tracked versions of files on a single
  machine only. Fine for one person, useless for a team.
- **Centralized (CVCS)** — tools like **CVS** and **Subversion (SVN)** keep the
  history on one central server. Everyone checks files in and out from it. This
  enabled real collaboration, but the server is a single point of failure, and
  most operations need a network connection.
- **Distributed (DVCS)** — tools like **Git** and **Mercurial** give every
  contributor a *full clone* of the repository, history and all.

The diagram below contrasts the last two:

```text
Centralized (SVN)            Distributed (Git)
  ┌─────────┐                  ┌─────────┐
  │ server  │                  │ server  │  (e.g. GitHub)
  └────┬────┘                  └──┬───┬──┘
   ┌───┴───┐                   ┌──┴─┐ ┌┴───┐
   │ you   │ (no history)      │you │ │peer│  each has FULL history
   └───────┘                   └────┘ └────┘
```

## What does "distributed" buy you?

Because your clone contains the entire history, you can do almost everything
**offline**: commit, view history, compare versions, and create branches without
talking to any server. You only need the network to *share* — to push your work
out or pull others' in. We cover that in [Remotes](/learn/git/remotes/).

Distribution also makes **branching cheap**. Spinning up a branch to try an idea
is nearly instant and costs almost nothing, which encourages safe experimentation.
That single property shapes the way teams use Git, and it's the foundation of the
[branches](/learn/git/branches/) and [workflow strategies](/learn/git/workflow-strategies/)
lessons later in the path.

## Why did Git become the default?

Git was created in 2005 to manage the Linux kernel, so it was built for speed,
huge histories, and thousands of contributors from day one. A few reasons it won:

- **Fast and local** — most commands work against your own disk, not a server.
- **Cheap branching and merging** — encouraging the branch-and-merge habit.
- **Integrity by design** — every snapshot is identified by a content hash, so
  corruption is detectable (more on that in [How Git thinks](/learn/git/git-mental-model/)).
- **A thriving ecosystem** — hosts like **GitHub**, GitLab, and Bitbucket, plus
  deep tooling in every editor and CI system.

Mercurial is a fine DVCS with a similar model; Git simply gathered the larger
community, and network effects did the rest.

## Git is not GitHub, and not a backup

Two clarifications worth nailing down early, because they trip up newcomers:

- **Git ≠ GitHub.** Git is the tool on your machine. GitHub is a *hosting service*
  built around Git for storing and collaborating on repositories online. You'll
  meet it properly in [What is GitHub?](/learn/git/what-is-github/). You can use
  Git for years without ever opening GitHub.
- **Git ≠ a backup service.** Git records the history of files you *commit*, but
  it won't save uncommitted work, and a repo that lives only on your laptop dies
  with your laptop. Pushing to a remote gives you an off-site copy, but you still
  want ordinary backups too.

Unsure of a term? The [glossary](/learn/git/glossary/) defines everything in one
place.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in a distributed VCS every clone holds the complete history." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes Git a <em>distributed</em> version control system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It can only be used with GitHub</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every clone contains the project's full history</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It stores history on one central server</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **version control system** tracks **snapshots** of a project over time so you
  can travel through its **history**.
- It removes the pain of manual versioning, lost work, and clashing edits.
- VCS designs went **local → centralized (SVN) → distributed (Git, Mercurial)**.
- **Distributed** means a full local history: work offline, branch cheaply.
- Git won on speed, cheap branching, integrity, and ecosystem.
- **Git ≠ GitHub** and **Git ≠ a backup**.

Next up: getting Git onto your machine and configured the right way.
