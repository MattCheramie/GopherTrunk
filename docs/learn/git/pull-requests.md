---
slug: pull-requests
title: Pull requests & code review
description: Pull requests explained — how to open one, write a great description, link issues, run reviews, and choose merge vs squash vs rebase. The heart of GitHub collaboration.
keywords: github pull request, pull request, code review, base vs compare branch, draft pull request, squash merge, rebase and merge, closes issue, request reviewers, merge pr
level: intermediate
status: full
prereq:
  - remotes
  - forking
faq:
  - q: What is a pull request?
    a: A pull request (PR) is a request to merge the commits on one branch into another, wrapped in a discussion. It shows the diff between the two branches, lets people leave review comments, runs automated checks like tests, and provides a button to merge when everyone's happy. Despite the name, it's really a "please merge my changes" proposal — a collaboration and review surface, not a Git command.
  - q: What is the difference between the base and compare branch?
    a: The base branch is where you want your changes to end up — usually main. The compare branch (also called the head branch) is the one holding your new commits. A pull request proposes merging the compare branch into the base branch, and the diff you see is "what changes the base branch would receive."
  - q: 'What does Closes #123 do in a pull request?'
    a: 'Writing a closing keyword like Closes #123, Fixes #123, or Resolves #123 in a pull request description links the PR to that issue, and when the PR is merged GitHub automatically closes the issue. It keeps your code change and the task it addresses connected, with no manual cleanup.'
  - q: Should I use merge commit, squash, or rebase and merge?
    a: A merge commit keeps every commit from the branch plus a merge commit recording the join. Squash and merge combines all the branch's commits into a single tidy commit on the base. Rebase and merge replays the branch's commits onto the base with no merge commit, keeping a linear history. Squash is popular for feature branches with messy work-in-progress commits; merge commits preserve full detail; pick what fits your team's history style.
---

# Pull requests & code review

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **pull request (PR)** asks to merge one branch (the **compare** branch) into another
(the **base**, usually `main`), wrapped in **discussion, review, and CI**. Open one by
pushing a branch and picking base vs compare; write a clear title and description, and
link issues with **`Closes #123`**. Mark unfinished work as a **draft**. Reviewers
leave **line comments** and **suggested changes**, then **Approve** or **Request
changes**. **Required status checks** can block merging until tests pass. When merging,
choose **merge commit**, **squash**, or **rebase-and-merge** depending on the history
you want — then **delete the branch**.
</div>

You've [forked, branched, and pushed](/learn/git/forking/). Now comes the moment your
work meets the project: the pull request. It's where code gets reviewed, discussed,
tested, and finally merged. Understanding PRs well is what separates "I can use Git"
from "I can collaborate on GitHub."

## What a pull request actually is

A **pull request** is a proposal: *please merge the commits on this branch into that
branch.* But it's far more than a merge button. A PR bundles together:

- the **diff** between the two branches, so reviewers see exactly what changes,
- a **discussion thread** for questions and decisions,
- **code review** with inline comments and approvals,
- **automated checks** (tests, linters) run by [GitHub Actions](/learn/git/github-actions/),
- and the **merge** controls once it's approved.

Despite the name, nothing is "pulled" by you — you're asking the project to pull your
work in. It works the same whether the branch lives in the same repo (team workflow)
or in a [fork](/learn/git/forking/) (open-source workflow).

## Opening a pull request

After you push a branch, GitHub usually shows a **Compare & pull request** banner.
Click it, or go to the **Pull requests** tab and choose **New pull request**. You'll
pick two branches:

- **Base branch** — where your changes should land, typically `main`.
- **Compare branch** — the branch holding your new commits.

```text
  base:  main          ◄── merge into
  compare: feature/login   (your commits)
```

GitHub shows the resulting diff so you can sanity-check it before submitting. When
contributing from a fork, the base is the *original* repo's `main` and the compare is
your fork's branch.

## Writing a good title and description

A PR is read by humans, so make their job easy.

- **Title**: a concise summary of the change, like a good commit subject —
  *"Add rate limiting to the login endpoint."*
- **Description**: explain *what* changed and *why*, note anything reviewers should
  watch for, and link related work.

Crucially, link the issue your PR addresses using a **closing keyword**. Writing
`Closes #123` (or `Fixes #123` / `Resolves #123`) in the description ties the PR to
that [issue](/learn/git/issues-and-projects/) and **auto-closes it** when the PR
merges:

```text
Add rate limiting to the login endpoint

Limits login attempts to 5 per minute per IP to slow down
credential-stuffing attacks.

Closes #123
```

## Draft PRs and the review flow

If your work isn't ready for merging but you want early feedback or CI to run, open it
as a **draft pull request** (choose **Create draft pull request** from the dropdown).
Drafts can't be merged until you click **Ready for review**, signalling clearly that
it's still in progress.

Once it's open for review, the collaboration begins:

- **Request reviewers** from the sidebar to ask specific people to look.
- Reviewers leave **line comments** anchored to specific lines of the diff.
- They can offer **suggested changes** — concrete edits you accept with one click.
- When done, a reviewer submits one of three verdicts:

| Verdict | Meaning |
|---------|---------|
| **Comment** | Feedback only, no explicit approval or block |
| **Approve** | Looks good to merge |
| **Request changes** | Must be addressed before merging |

You respond by pushing more commits to the same branch — the PR updates
automatically — and conversations get resolved as issues are fixed.

## Status checks and protection

Many repos run **required status checks**: automated tests and linters that must pass
before the merge button unlocks. You'll see them at the bottom of the PR with green
ticks or red crosses, and clicking through shows the logs.

These checks are powered by [GitHub Actions](/learn/git/github-actions/) and enforced
by [branch protection](/learn/git/branch-protection/) rules — a maintainer can require
that checks pass *and* that a reviewer approves before anyone can merge to `main`. For
now, just know: red checks usually mean "fix this first."

## Merge options and cleanup

When a PR is approved and checks are green, you merge. GitHub offers three strategies,
and the choice shapes your project's history:

| Option | What it does to history |
|--------|-------------------------|
| **Create a merge commit** | Keeps all the branch's commits *plus* a merge commit recording the join |
| **Squash and merge** | Combines every commit on the branch into one tidy commit on the base |
| **Rebase and merge** | Replays the branch's commits onto the base with no merge commit (linear) |

A rough guide: **squash** is great for feature branches full of "wip" and "fix typo"
commits you don't want in `main`; a **merge commit** preserves the full detail and
the shape of the branch; **rebase and merge** keeps history perfectly linear. Teams
usually settle on one default.

After merging, **delete the branch** — GitHub shows a button right there. The work is
safely in `main`, so the branch has done its job:

```bash
$ git switch main
$ git pull                       # bring the merge down locally
$ git branch -d feature/login    # tidy up your local copy too
```

(Need the precise difference between merge and rebase? The
[glossary](/learn/git/glossary/) and the [merging](/learn/git/merging/) and
[rebasing](/learn/git/rebasing/) lessons go deeper.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — squash collapses the branch's commits into one." markdown="0">
  <p class="knowledge-check__q">Quick check: your feature branch has ten messy "wip" commits and you want just one clean commit on main. Which merge option?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Create a merge commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Rebase and merge</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Squash and merge</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **pull request** proposes merging the **compare** branch into the **base**, with
  discussion, review, and CI attached.
- Pick **base vs compare**, write a clear title and description, and link issues with
  **`Closes #123`** to auto-close them.
- Use **draft** PRs for work in progress; reviewers leave line comments and **Approve**
  or **Request changes**.
- **Required status checks** can block merging until tests pass.
- Choose **merge commit**, **squash**, or **rebase-and-merge** for the history you want, then
  **delete the branch**.

That completes Module 4. Next module moves into automation and project tooling, starting
with [issues & projects](/learn/git/issues-and-projects/).
