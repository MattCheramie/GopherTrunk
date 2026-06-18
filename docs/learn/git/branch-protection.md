---
slug: branch-protection
title: Branch protection, reviews & CODEOWNERS
description: Protect main with branch protection rules and rulesets — require PRs, approvals, passing checks, linear history, signed commits — and auto-request reviews with CODEOWNERS.
keywords: branch protection, github rulesets, require pull request, required status checks, dismiss stale reviews, require signed commits, linear history, codeowners, block force push, required reviews
level: advanced
status: full
prereq:
  - pull-requests
faq:
  - q: What is the difference between a branch protection rule and a ruleset?
    a: Both restrict what can happen to matching branches. Classic branch protection rules are configured per branch pattern under Settings → Branches and apply to a single repository. Rulesets are the newer system that can layer multiple named rule sets, apply at the organisation level across many repos, target branches and tags, and be set to evaluate-only mode for testing. Rulesets are the recommended direction; the protections they offer overlap heavily.
  - q: Why should I protect the main branch?
    a: main is the branch others clone, deploy, and trust. Without protection, anyone with write access can push directly, force-push over history, or merge unreviewed and untested code straight into it. Protecting main forces changes to arrive through reviewed, tested pull requests, so a single mistake or a compromised account can't silently rewrite or break the branch everyone depends on.
  - q: What does the CODEOWNERS file do?
    a: CODEOWNERS maps file paths to the people or teams responsible for them. When a pull request touches a matching path, GitHub automatically requests a review from the listed owners. Combined with a branch protection rule requiring review from Code Owners, it guarantees the right experts must approve changes to their area before merge. It lives in .github/, the repo root, or docs/.
  - q: What is required linear history?
    a: Required linear history forbids merge commits on the protected branch, so its history stays a straight line. With it enabled, pull requests must be merged using squash or rebase rather than a merge commit. This keeps the branch's log simple and easy to read or bisect, at the cost of not preserving the exact branch-merge structure.
---

# Branch protection, reviews & CODEOWNERS

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Branch protection** stops anyone from quietly breaking or rewriting an important
branch like `main`. Configure **rules** (per repo) or **rulesets** (newer, layered,
org-wide) to **require a pull request before merging**, demand **N approving reviews**,
**dismiss stale reviews**, **require status checks** to pass, enforce **linear history**
or **signed commits**, **restrict who can push**, and **block force-pushes and
deletions**. A **`CODEOWNERS`** file maps paths to owners and **auto-requests their
review**. Together with [pull requests](/learn/git/pull-requests/) and
[Actions](/learn/git/github-actions/), this forms a workflow where bad code simply can't
reach `main`.
</div>

You can open [pull requests](/learn/git/pull-requests/) and run
[CI](/learn/git/github-actions/) — but nothing yet *forces* anyone to use them. A
teammate with write access can still push straight to `main`. **Branch protection** is
how you make the safe path the only path. This lesson covers the rules, the
`CODEOWNERS` file, and how they combine.

## Why protect main

`main` is the branch people clone, deploy, and trust to be working. Left open, it's one
careless command away from trouble:

- a **direct push** that skips review and CI;
- a **force-push** that rewrites or erases history others have built on;
- a **deletion** of the branch entirely;
- an **unreviewed, untested merge** of broken code.

Protection converts these from "possible by accident" into "blocked by policy." The
goal isn't distrust — it's making the reviewed-and-tested route the path of least
resistance, even for the repo owner.

## Rules vs rulesets

GitHub offers two systems, configured under **Settings**:

| | Branch protection rules | Rulesets |
|---|---|---|
| Scope | One repository, by branch pattern | Repo **or** organisation-wide |
| Layering | One rule per pattern | Multiple named rulesets stack |
| Targets | Branches | Branches **and** tags |
| Testing | Enforced or off | Can run in **evaluate** (dry-run) mode |

**Rulesets** are the newer, more flexible direction and can apply the same protections
across many repos at once. The individual controls below exist in both systems — the
choice is mostly about scale and management.

## The core protections

A typical protected `main` enables several of these together:

- **Require a pull request before merging** — no direct pushes; every change arrives via
  a PR.
- **Require approvals** — set a number (e.g. **2**); the PR can't merge until that many
  reviewers approve.
- **Dismiss stale pull request approvals when new commits are pushed** — if the author
  pushes more changes after approval, the approvals reset so reviewers re-check the new
  code.
- **Require status checks to pass** — pick the [CI checks](/learn/git/github-actions/)
  that must be green; the Merge button stays disabled until they are. Optionally
  **require branches to be up to date** before merging.
- **Require conversation resolution** — all PR review comments must be resolved first.
- **Require linear history** — forbids merge commits, forcing squash or rebase merges so
  the log stays a straight line.
- **Require signed commits** — every commit must carry a verified GPG/SSH signature,
  proving authorship.
- **Block force pushes** and **block deletions** — protect the branch's history and its
  very existence.
- **Restrict who can push** — limit direct pushes (where allowed) to named people or
  teams.

You can also choose whether these rules **apply to administrators** — for real safety,
they should.

## The CODEOWNERS file

Requiring reviews is good; requiring the *right* reviewers is better. The **`CODEOWNERS`**
file maps file paths to the people or teams responsible for them. Place it in `.github/`,
the repo root, or `docs/`. The syntax is a path pattern followed by owners:

```text
# Default owners for everything in the repo
*               @octo-org/maintainers

# Front-end code
/src/ui/        @alice @octo-org/frontend

# Anything in docs
*.md            @octo-org/docs-team

# A specific file
/Dockerfile     @bob
```

The **last matching pattern wins**, so put broad defaults first and specifics later.
When a PR changes a matching path, GitHub **automatically requests a review** from the
listed owners. Pair this with the branch protection option **Require review from Code
Owners**, and a PR touching `/src/ui/` simply cannot merge without an approval from a
front-end owner.

## How it all combines into a safe workflow

Stack these pieces and you get a workflow where mistakes are caught by machinery, not
luck:

1. A developer pushes a branch and opens a [pull request](/learn/git/pull-requests/) —
   direct pushes to `main` are blocked.
2. [GitHub Actions](/learn/git/github-actions/) runs the test workflow; its **required
   status checks** must pass.
3. **`CODEOWNERS`** auto-requests review from the right people; the rule requires **N
   approvals**, and a later push **dismisses stale ones**.
4. Conversations must be resolved, and the branch must be up to date.
5. Only when every gate is green does the **Merge** button light up, producing a clean,
   **linear** (or signed) history.

The result: `main` always reflects reviewed, tested code, and no single action — careless
or malicious — can quietly compromise it. (Unsure about "force-push," "status check," or
"squash merge"? The [glossary](/learn/git/glossary/) defines them.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — CODEOWNERS auto-requests review from the owners of the changed paths." markdown="0">
  <p class="knowledge-check__q">Quick check: what does adding a <code>CODEOWNERS</code> file do for a pull request?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Auto-requests review from the owners of the changed files</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Automatically merges the PR once CI passes</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Signs every commit on the branch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Protect `main`** so changes can't bypass review and CI or rewrite history.
- Use **rules** (per repo) or **rulesets** (newer, layered, org-wide) to enforce
  protections.
- Common gates: **require a PR**, **N approvals**, **dismiss stale reviews**, **required
  status checks**, **linear history**, **signed commits**, and **block force-pushes /
  deletions**.
- **`CODEOWNERS`** maps paths to owners and **auto-requests** their review.
- Combined with [PRs](/learn/git/pull-requests/) and
  [Actions](/learn/git/github-actions/), this makes the reviewed, tested path the only
  way into `main`.

Next up: rewriting history precisely with [interactive rebase](/learn/git/interactive-rebase/).
