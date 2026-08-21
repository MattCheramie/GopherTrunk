---
slug: required-checks
title: Required checks & branch protection
description: Which CI jobs actually gate a merge — and how a green checkmark can hide a red build in a job nobody made required. Branch protection as the enforcement layer for everything in this unit.
keywords: required checks github, branch protection rules, merge gating, ci required status checks, non-required jobs, green checkmark red build, protect main branch
level: intermediate
status: full
prereq:
  - continuous-integration
  - code-review
---

# Required checks & branch protection

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
CI jobs only *report*; **branch protection** is what makes them *gate*: the
platform refuses to merge until the **required checks** pass and a review
approves. The catch every team learns the hard way: **only checks explicitly
marked required block anything** — a PR can wear a green "mergeable" state
while a **non-required job is red**, and that red ships. Audit the required
set against what you actually build, and when reading a PR, **read the checks,
not the checkmark**.
</div>

This unit built four guards — linting, formatting, review, CI. This closing
lesson is about the bolt that fastens them to the door: the configuration that
decides which guards can actually stop a merge, and the quiet failure mode
when that configuration drifts out of sync with reality.

## From reporting to gating

By default, CI results are informational: red jobs decorate the pull request,
and a human can merge anyway — deadline pressure guarantees someone will.
**Branch protection** (GitHub's term; every platform has an equivalent — the
[Git module](/learn/git/branch-protection/) covers the settings screen) flips
selected rules from advice to enforcement on a branch, typically `main`:

- **Required status checks** — named CI jobs that must pass before merging is
  possible.
- **Required reviews** — at least N approvals from
  [reviewers](/learn/testing/code-review/).
- **No direct pushes / no force pushes** — all changes travel through a PR,
  and history can't be rewritten out from under everyone.

With this in place, "keep main green" stops being etiquette and becomes
physics: the merge button is disabled until the gates open. This is the
enforcement layer the whole unit has been implying — an optional check
protects nothing, because the day it's inconvenient is exactly the day it
matters.

## The trap: green checkmark, red build

Now the sharp edge. Required checks are an **explicit allowlist of job
names**. Jobs not on the list still run, still report — and **block nothing**.
The platform's summary optimizes for the merge question, so a PR whose
required jobs pass shows mergeable-green *even while a non-required job sits
red* below the fold.

This is not hypothetical; it's a bug class with a known signature, and
GopherTrunk hit it for real. The project's web console is TypeScript, and its
CI tests transpiled the code without *typechecking* it — so type errors sailed
through the test jobs. The only job that ran a full production build (and thus
the typechecker) was a packaging job **nobody had marked required**. A change
landed with three type errors; every required job passed; the PR "passed CI"
and merged — red job and all. The next person to run a real build found the
project wouldn't compile. Two lessons were extracted, and both generalize:

1. **The required set must cover what "working" means.** If releases need
   `npm run build` to succeed, some required job must run `npm run build` —
   not a proxy for it, the thing itself.
2. **Read a PR's checks, not its checkmark.** The one-glance summary answers
   "may this merge?"; only the full list answers "is this healthy?" A red
   anything deserves an explanation before merge, required or not.

> Rule of thumb: a check that can't block a merge is documentation, not
> protection. Decide deliberately which is which — don't let the default
> decide.

## Auditing the gate

The required set drifts: jobs get renamed (silently detaching from the
required list, which matches by name), new suites get added as non-required
"for now," and forever is made of for-nows. A periodic audit is cheap.
Questions worth asking of any repo you work on:

| Question | Failure it prevents |
|----------|---------------------|
| Does some required job run the *real* production build for every artifact we ship? | The typecheck story above |
| Do required names still match the workflow's actual job names? | A renamed job that gates nothing while everyone assumes it does |
| Is anything red-on-every-PR sitting in non-required limbo? | Normalized failure — the team has learned to un-see a red |
| Are [quarantined flakes](/learn/testing/flaky-tests/) tracked, or just parked outside the gate? | Quarantine becoming silent deletion |

The deeper principle closing this unit: **a safety system is only as strong as
its enforcement path.** Tests you don't run, linters that only warn, reviews
you can skip, checks that can't block — each is a guard that works right up
until the day it's needed. The configuration is part of the system; audit it
like code.

<div class="knowledge-check" data-quiz data-correct-msg="Right — only jobs explicitly marked required can block; everything else reports and is ignorable, which is how a red build merges under a green checkmark." markdown="0">
  <p class="knowledge-check__q">Quick check: a PR shows a green mergeable state, yet one CI job in its list is red. How is that possible?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The platform re-ran the red job in the background and it passed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The red job isn't in the required-checks list, so it reports without blocking</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Green summaries are computed before all jobs finish, so it's a display lag</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Branch protection** turns checks and reviews from advice into a **gate**:
  no merge until required checks pass and approvals land.
- Only checks **explicitly marked required** block anything — non-required
  jobs report red and merge anyway.
- Hence the trap: **green checkmark over a red build**, exactly how a
  typecheck-shaped hole once let a broken build onto GopherTrunk's main.
- Defenses: required set covers the **real build**, names stay synced, and
  reviewers **read the checks, not the checkmark**.
- The unit's closing law: a safety system is only as strong as its
  **enforcement path** — audit the configuration like code.

Next up: [Reproducing a bug](/learn/testing/reproducing-a-bug/)
