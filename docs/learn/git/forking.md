---
slug: forking
title: Forking & the fork workflow
description: How forking works on GitHub — your own server-side copy of a repo. Fork vs clone vs branch, the standard open-source flow, and keeping a fork in sync with upstream.
keywords: github fork, fork workflow, fork vs clone, upstream remote, sync fork, contribute to open source, git fetch upstream, fork and pull request, fork repository
level: intermediate
status: full
prereq:
  - remotes
faq:
  - q: What is the difference between forking and cloning?
    a: A fork is a copy of a repository on GitHub's servers under your own account, created through the GitHub website. A clone is a copy on your local machine, created with git clone. In the standard open-source flow you fork first (server-side copy you can push to) and then clone your fork (local copy to work in). Cloning alone gives you a local copy but no place on the server you're allowed to push to if you don't own the original.
  - q: When should I fork instead of just creating a branch?
    a: Create a branch when you have write access to the repository — that's the normal flow for your own projects and teams. Fork when you do NOT have write access, typically when contributing to someone else's open-source project. The fork gives you a copy you can push to freely, and you then propose your changes back via a pull request.
  - q: What is the upstream remote?
    a: When you fork and clone, your fork becomes the origin remote. upstream is the conventional name you give to a second remote pointing at the ORIGINAL repository you forked from. You add it with git remote add upstream <url> so you can fetch the project's latest changes and keep your fork in sync, since origin (your fork) does not update on its own.
  - q: How do I keep my fork up to date?
    a: Your fork does not automatically track the original. Add the original as an upstream remote, run git fetch upstream to download its latest commits, merge or rebase upstream/main into your local main, and push the result to your fork. GitHub also offers a Sync fork button on the fork's page that does the equivalent without the command line.
---

# Forking & the fork workflow

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **fork** is your own **server-side copy** of someone else's repository, made on
GitHub. You fork when you **can't push** to the original — typically to contribute to
open source. The standard flow is **fork → clone your fork → add an `upstream`
remote → branch → push to your fork → open a pull request**. Because your fork
doesn't update itself, you keep it current by **fetching `upstream`** and merging or
rebasing `upstream/main`, then pushing. GitHub's **Sync fork** button does the same
from the web.
</div>

You can now [push and pull](/learn/git/remotes/) against a repo you own. But what
about contributing to a project you *don't* own — say a popular open-source tool? You
can't push to it directly. Forking is the answer, and it's the backbone of how open
source collaboration works on GitHub.

## What a fork is

A **fork** is a complete copy of a repository placed under *your* account on GitHub's
servers. You create one by clicking **Fork** at the top of any repository's page.
GitHub copies the project to `github.com/<you>/<repo>`, and GitHub remembers the link
back to the original (the **upstream**).

The key point: you have full write access to your fork, even though you have none on
the original. That's what makes contribution possible — you make changes in your
copy, then *propose* them back to the original through a
[pull request](/learn/git/pull-requests/).

## Fork vs clone vs branch — when to use each

These three are easy to muddle. Here's the clean separation:

| Action | Where it lives | Use when |
|--------|---------------|----------|
| **Branch** | Inside one repo | You have write access and want a new line of work |
| **Clone** | Your local machine | You want a local copy of any repo to work in |
| **Fork** | Your GitHub account | You *don't* have write access and want to contribute |

In your own projects and your team's, you just **branch** — no fork needed. You
**fork** specifically when the original repo isn't yours to push to. And you almost
always **clone** afterward, because you still need a local copy to actually edit.

## The standard open-source flow

Here's the full, time-tested sequence for contributing to a project you don't own.

**1. Fork** the project on GitHub (the **Fork** button). You now have
`github.com/you/project`.

**2. Clone your fork** to your machine. Cloning sets up `origin` pointing at *your*
fork:

```bash
$ git clone git@github.com:you/project.git
$ cd project
```

**3. Add the original as `upstream`** so you can pull in its updates later:

```bash
$ git remote add upstream git@github.com:original-owner/project.git
$ git remote -v
origin    git@github.com:you/project.git (fetch)
origin    git@github.com:you/project.git (push)
upstream  git@github.com:original-owner/project.git (fetch)
upstream  git@github.com:original-owner/project.git (push)
```

**4. Create a branch** for your change (never work directly on `main`):

```bash
$ git switch -c fix/typo-in-readme
```

**5. Make your changes, commit, and push to your fork:**

```bash
$ git commit -am "Fix typo in installation instructions"
$ git push -u origin fix/typo-in-readme
```

**6. Open a pull request** from your branch on your fork back to the original
project's `main`. That's the next lesson.

```text
  upstream (original)  ◄── pull request ──  origin (your fork)  ◄── push ──  your laptop
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 176" role="img" aria-label="The fork and pull-request flow. Your laptop pushes to your fork, called origin, on GitHub. Your fork opens a pull request to the upstream original repository. A return arrow along the bottom shows git fetch upstream syncing changes from upstream back down to your laptop." xmlns="http://www.w3.org/2000/svg">
  <g>
    <rect x="14" y="44" width="112" height="46" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
    <rect x="179" y="44" width="112" height="46" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
    <rect x="344" y="44" width="112" height="46" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  </g>
  <g text-anchor="middle" fill="currentColor">
    <text x="70" y="66" font-size="9" font-weight="600">your laptop</text>
    <text x="70" y="80" font-size="8" fill-opacity="0.85">local clone</text>
    <text x="235" y="66" font-size="9" font-weight="600">your fork</text>
    <text x="235" y="80" font-size="8" fill-opacity="0.85">origin</text>
    <text x="400" y="66" font-size="9" font-weight="600">upstream</text>
    <text x="400" y="80" font-size="8" fill-opacity="0.85">the original</text>
  </g>
  <g stroke="currentColor" stroke-width="1.5" fill="none">
    <line x1="126" y1="60" x2="176" y2="60" marker-end="url(#fork_ar)"/>
    <line x1="291" y1="60" x2="341" y2="60" marker-end="url(#fork_ar)"/>
    <path d="M400 90 V128 H70 V98" marker-end="url(#fork_ar)"/>
  </g>
  <g text-anchor="middle" fill="currentColor" font-size="8.5">
    <text x="151" y="52">push</text>
    <text x="316" y="52">pull request</text>
    <text x="235" y="146">git fetch upstream — keep your fork in sync</text>
  </g>
  <defs><marker id="fork_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>You can't push to the original, so you push to your own <strong>fork</strong> (origin) and propose changes back with a <strong>pull request</strong>. Because the fork doesn't update itself, you periodically <strong>fetch upstream</strong> and merge or rebase its latest commits back down — the loop that keeps a long-lived fork current.</figcaption>
</figure>

## Keeping a fork in sync

Here's the catch with forks: **they don't update themselves**. While you work, the
original project keeps moving, and your fork's `main` slowly falls behind. Before
starting new work, refresh it.

First, fetch the latest from `upstream`:

```bash
$ git fetch upstream
remote: Enumerating objects: 47, done.
From github.com:original-owner/project
   3a1f9c2..8b4d6e0  main       -> upstream/main
```

Switch to your local `main` and bring in those commits. You can **merge**:

```bash
$ git switch main
$ git merge upstream/main
Updating 3a1f9c2..8b4d6e0
Fast-forward
 src/app.js | 12 ++++++++++--
 1 file changed, 10 insertions(+), 2 deletions(-)
```

or **rebase** for a straight-line history (see [rebasing](/learn/git/rebasing/)):

```bash
$ git rebase upstream/main
Successfully rebased and updated refs/heads/main.
```

Finally, push the updated `main` to your fork so `origin` matches too:

```bash
$ git push origin main
```

Now your fork is current, and new feature branches start from up-to-date code.

## The Sync fork button

If you'd rather not touch the command line, GitHub offers a shortcut. On your fork's
page, when it's behind the original, you'll see a **Sync fork** notice with an
**Update branch** button. Clicking it brings your fork's `main` up to date with
upstream directly on the server.

It's convenient for keeping the *fork* current, but remember it only updates the copy
on GitHub — you'll still need to `git pull` afterward to refresh your *local* clone.
For anything involving conflicts, the command-line flow above gives you more control.
(Unsure what "merge" or "rebase" mean here? The
[glossary](/learn/git/glossary/) has short definitions.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — fork when you can't push to the original repo." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to contribute a bug fix to a popular project you don't own. What do you do first?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Fork the repo to your own account, then clone your fork</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Push a new branch straight to the original repo</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Email the maintainer your changed files</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **fork** is your own server-side copy of a repo, made on GitHub when you can't push
  to the original.
- **Branch** when you have write access; **fork** when you don't; **clone** to get a local
  copy either way.
- The flow: **fork → clone → add `upstream` → branch → push to your fork → open a PR**.
- Forks don't auto-update — **`git fetch upstream`**, merge or rebase `upstream/main`, then push.
- The **Sync fork** button updates the fork on GitHub; you still `git pull` to refresh locally.

Next up: proposing your changes back with [pull requests & code review](/learn/git/pull-requests/).
