---
title: "Build in the Open, Part 13: Advanced Git & GitHub Features"
description: Level up beyond commit and push — interactive rebase, cherry-pick, bisect, reflog, worktrees, conflict resolution, plus Discussions, Codespaces, GHCR, and Environments.
keywords: git rebase, interactive rebase, git cherry-pick, git bisect, git reflog, git worktrees, merge conflicts, github discussions, codespaces, ghcr, claude code
category: tutorials
tags: [github, git, advanced, ghcr, codespaces, claude-code]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 13
---

> **TL;DR:** Once `commit`, `push`, and `merge` are muscle memory, a second tier
> of Git tools makes you dramatically more effective: interactive rebase to clean
> history, cherry-pick to move a single commit, stash to park work, bisect to
> find the commit that broke something, reflog to recover "lost" work, and
> worktrees to check out multiple branches at once. On the platform side, GitHub
> offers far more than PRs — Discussions, code search, Codespaces, container/
> package hosting (GHCR), deployment Environments, and a GraphQL API.

**Key takeaways**

- **Interactive rebase** (`git rebase -i`) reorders, squashes, and rewords
  commits to tell a clean story.
- **Bisect** binary-searches your history to pinpoint the commit that introduced
  a bug — automatically.
- **Reflog** is your undo button for almost anything; lost work is rarely
  actually lost.
- **Worktrees** let you have several branches checked out in parallel without
  stashing or re-cloning.
- GitHub is a platform, not just a Git host: Discussions, Codespaces, GHCR, and
  Environments all build on the same repo.
- Conflicts are routine, not a crisis — resolve them deliberately, test, then
  commit.

*This is Part 13 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **Git power tools** — rebase, cherry-pick, stash, bisect, reflog, worktrees,
  tags.
- **Resolving merge conflicts** with confidence instead of dread.
- **Advanced GitHub features** — Discussions, code search, Codespaces, GHCR,
  Environments, GraphQL.
- **How GopherTrunk uses them** — conflict-resolution branches, a published
  container image, and generated protobuf.

## Half one: Git power tools

These are the commands that separate "I can use Git" from "Git works for me."

### Interactive rebase: curate your history

`git rebase -i HEAD~5` opens an editor listing your last five commits. You can
`pick`, `squash` (fold into the previous), `reword` (edit the message),
`reorder`, or `drop` them. This is how you turn a messy "wip", "fix typo", "fix
again" trail into a few clean, logical commits before opening a PR. Rebase
*rewrites history*, so the rule is simple: rebase freely on your own
not-yet-shared branches; never on shared branches others have based work on.

### Cherry-pick: grab one commit

`git cherry-pick <sha>` copies a single commit onto your current branch. Perfect
for back-porting a fix to a release branch, or pulling one useful commit out of
an abandoned branch without merging the rest.

### Stash: park work mid-thought

`git stash` shelves your uncommitted changes so you can switch context — fix an
urgent bug, then `git stash pop` to bring your work back. A clean way to handle
"I need to drop this for five minutes" without making a throwaway commit.

### Bisect: find the commit that broke it

`git bisect` is a hidden superpower. Tell it a known-good commit and a
known-bad one, and it binary-searches between them, checking out a midpoint each
time for you to test. You can even automate it: `git bisect run ./test.sh` lets
the test suite find the culprit hands-free across hundreds of commits in a dozen
steps.

### Reflog: undo almost anything

`git reflog` records *every* move of `HEAD` — including after a bad reset, a
botched rebase, or a deleted branch. Find the SHA you were on and `git reset
--hard <sha>` (or `git checkout`) to get back. Work you think you destroyed is
usually sitting in the reflog for weeks.

### Worktrees: multiple branches at once

`git worktree add ../project-hotfix hotfix-branch` checks out another branch
into a *separate directory* sharing the same repo. No stashing, no second clone
— review a PR in one window while you keep building in another. Especially handy
when a long build or test run shouldn't block other work.

### Resolve conflicts deliberately

A merge conflict just means two branches changed the same lines. It's routine.
The calm workflow: open each conflicted file, find the `<<<<<<<` / `=======` /
`>>>>>>>` markers, decide what the *correct* combined result is (not just "pick a
side"), remove the markers, **run your tests**, then `git add` and commit. The
fear comes from treating conflicts as rare emergencies; resolve a few
deliberately and they become mundane.

## Half two: advanced GitHub features

The platform layered on top of Git is deep. The high-value pieces:

- **Discussions** — a forum attached to your repo for Q&A, ideas, and
  announcements, separate from the issue tracker (issues are for actionable work;
  discussions are for conversation).
- **Wikis** — long-form docs that live alongside the repo without cluttering it.
- **Code search & navigation** — GitHub's indexed search and "go to definition"
  make a large codebase browsable in the web UI.
- **Codespaces** — a full, cloud-hosted dev environment spun up from your repo in
  a browser; great for contributors who don't want to set up a local toolchain.
- **GHCR / Packages** — GitHub Container Registry and package hosting, so you can
  publish Docker images and language packages right next to your code.
- **Environments + environment secrets** — named deploy targets (staging,
  production) with their own protected secrets and approval gates.
- **The GraphQL API** — query exactly the repo data you need in one request;
  more powerful than the REST API for complex automation.
- **Insights, saved replies, keyboard shortcuts** — the quality-of-life layer.
  Press <kbd>?</kbd> on any GitHub page to see the shortcuts.

You won't use all of these on day one, but knowing they exist means you reach for
the right tool when the need shows up.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk)'s history and tooling
show several of these in action.

**Conflict-resolution branches in the open.** The git log makes the merge
workflow visible — there's a real branch
`claude/pr-674-merge-conflicts-bmivyc` whose merge commit
(`1a56ec8 Merge pull request #708`) exists *specifically* to reconcile a
long-running branch against an evolved `main`, including a
`Merge remote-tracking branch 'origin/claude/issue-376-fix-q1l2cc'`. Conflicts
aren't hidden; they're resolved on a dedicated branch and merged cleanly.

**Squash-friendly, near-linear history.** As covered in
[Part 5]({{ '/blog/tutorials/build-in-the-open-05-three-ways-to-merge-to-main/' | relative_url }}),
the maintainer squashes on merge to keep `main` legible — `CONTRIBUTING.md` notes
"force-push is fine on feature branches; the maintainer squashes on merge to keep
`main` history clean." That discipline is exactly what makes a `git bisect` over
`main` fast and meaningful: each commit is one logical, testable change.

**A published container image.** GopherTrunk ships a Docker image to GitHub's
container registry — `docker-compose.yml` pulls
`image: ghcr.io/mattcheramie/gophertrunk:latest`, built from a multi-stage
`Dockerfile` (`golang:1.25-bookworm` builder → `debian:bookworm-slim` runtime).
That's GHCR/Packages doing real work: container hosting next to the code, no
separate registry account.

**Generated code regenerated reproducibly.** The gRPC API is defined in `.proto`
files under `proto/`, and the generated Go lands in `internal/api/pb/v1` via a
single `make proto` (which shells out to `protoc` with the Go and gRPC plugins).
Keeping generation behind one command means the generated tree is reproducible
and easy to review when it changes — the same legibility principle as squashed
commits, applied to machine-generated code.

The takeaway: you don't have to adopt everything at once. Learn rebase and reflog
first (they'll save you weekly), keep history clean enough that bisect is useful,
and reach for the platform features — GHCR, Codespaces, Discussions — when a
concrete need appears.

## FAQ

**Is rebase dangerous?**
Only if you rebase *shared* history. On your own feature branch that nobody else
has pulled, rebase freely to clean things up. The rule: never rewrite commits
others have already based work on. And if you do make a mistake, `git reflog`
will get you back.

**When should I use cherry-pick vs. merge?**
Merge when you want a whole branch's changes. Cherry-pick when you want *one*
specific commit — typically back-porting a single fix to a release branch
without dragging along everything else on the source branch.

**How is git bisect faster than reading the log?**
It binary-searches. Across 1,000 commits it finds the culprit in about 10 test
runs instead of checking each one. With `git bisect run <script>` it's fully
automated — you walk away and it reports the exact commit.

**What's the difference between GitHub Issues and Discussions?**
Issues track *actionable work* — bugs and tasks that get closed when done.
Discussions are for open-ended conversation — questions, ideas, announcements —
that may never "close." Use both for what they're built for.

**What is GHCR?**
GitHub Container Registry — free container image hosting tied to your repo. You
push Docker images to `ghcr.io/<owner>/<image>` and pull them anywhere, exactly
how GopherTrunk's `docker-compose.yml` references
`ghcr.io/mattcheramie/gophertrunk:latest`.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Optimizing & Securing Your Repository]({{ '/blog/tutorials/build-in-the-open-12-optimizing-securing-your-repository/' | relative_url }})
· Next →
[Part 14: The CLI Workflow & Claude Code as Your Daily Driver]({{ '/blog/tutorials/build-in-the-open-14-cli-workflow-claude-code-daily-driver/' | relative_url }})
