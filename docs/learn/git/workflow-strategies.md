---
slug: workflow-strategies
title: "Branching strategies: trunk, GitHub Flow & Git Flow"
description: Compare trunk-based development, GitHub Flow, and Git Flow — the trade-offs, how branch protection and CI support each, and how to choose by team size and release cadence.
keywords: branching strategy, trunk-based development, GitHub Flow, Git Flow, feature flags, release branch, hotfix branch, develop branch, continuous delivery, branching model
level: intermediate
status: full
prereq:
  - branches
  - pull-requests
faq:
  - q: What is trunk-based development?
    a: Trunk-based development keeps a single long-lived branch (the trunk, usually main) that everyone integrates into frequently — typically at least daily — through very short-lived branches or even direct commits. Unfinished features are hidden behind feature flags rather than kept on long branches, which keeps merges small and avoids painful divergence. It pairs naturally with continuous integration and continuous delivery.
  - q: What is the difference between GitHub Flow and Git Flow?
    a: GitHub Flow is lightweight - branch off main, open a pull request, review, merge, and deploy, with main always deployable. Git Flow is heavier, with two long-lived branches (main and develop) plus dedicated feature, release, and hotfix branches and a defined process for moving between them. GitHub Flow suits continuous web deployment; Git Flow suits versioned software with scheduled releases, at the cost of more overhead.
  - q: When is Git Flow's complexity worth it?
    a: Git Flow earns its complexity when you ship discrete, versioned releases that must be maintained in parallel — desktop or mobile apps, libraries, firmware, or any product where several versions are live at once and need hotfixes independently. For a continuously deployed web service, Git Flow's develop branch and release ceremony usually add overhead without benefit; trunk-based development or GitHub Flow fit better.
  - q: How do branch protection and CI support a branching strategy?
    a: Branch protection enforces the rules a strategy assumes — requiring pull request reviews and passing status checks before anything lands on the protected branch, and blocking force-pushes. CI runs the tests and linters that prove a branch is safe to merge, which is exactly what makes "main is always deployable" credible. Together they let small, frequent merges stay safe, which is what trunk-based and GitHub Flow depend on.
---

# Branching strategies: trunk, GitHub Flow & Git Flow

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A branching strategy is a team agreement about *how* you use [branches](/learn/git/branches/)
and [pull requests](/learn/git/pull-requests/). **Trunk-based development** keeps everyone
on one trunk with tiny, frequent merges and hides unfinished work behind **feature flags**.
**GitHub Flow** is the lightweight default: branch → PR → review → merge → deploy, with
`main` always deployable. **Git Flow** adds `develop`, plus `feature`/`release`/`hotfix`
branches — powerful for **versioned releases** but heavy for continuous deployment.
**[Branch protection](/learn/git/branch-protection/)** and **CI** are what make small,
frequent merges safe. Choose by **team size and release cadence**, then be consistent.
</div>

Git doesn't impose a workflow — branches are just pointers. A *strategy* is the convention
your team layers on top: which branches are long-lived, how work flows into them, and when
you ship. The three below cover most teams. Picking one deliberately matters more than
which one you pick.

## Trunk-based development

Everyone integrates into a single trunk (`main`) **frequently** — at least daily — through
**very short-lived** branches that live hours, not weeks. Work too big to finish in a day
is merged incrementally and hidden behind a **feature flag** until it's ready to switch on:

```bash
$ git switch -c quick-fix
# ... small change, opened as a PR, reviewed, merged the same day ...
$ git switch main
$ git pull
```

Because branches barely diverge, merges are trivial and [merge
conflicts](/learn/git/merge-conflicts/) are rare. The trade-off is discipline: it demands
strong CI, [feature flags](/learn/git/glossary/) for incomplete work, and a culture of
small changes. It's the model behind continuous delivery and the choice of many
high-velocity teams.

## GitHub Flow

The pragmatic default for most web projects, and the one this path has quietly taught
throughout. There's one long-lived branch — `main`, always deployable — and a single,
simple loop:

```text
main ──●──────────────●─────────►  (always deployable)
        \            /
         ●──●──●────●   feature branch → PR → review → merge → deploy
```

1. Branch off `main` for any change.
2. Push and open a [pull request](/learn/git/pull-requests/).
3. Review and let CI run.
4. Merge into `main`.
5. Deploy `main`.

It's easy to teach, keeps `main` releasable, and fits teams that deploy often. Branches are
short-lived (though usually longer than trunk-based) and there's no separate integration
branch to manage.

## Git Flow

Git Flow is the heavyweight, built for **versioned, scheduled releases**. It uses two
long-lived branches plus three kinds of supporting branch:

| Branch | Lives | Purpose |
|--------|-------|---------|
| `main` | forever | production; every commit is a tagged release |
| `develop` | forever | integration branch for the next release |
| `feature/*` | short | one feature, branched from and merged to `develop` |
| `release/*` | short | stabilise a release; branched from `develop`, merged to `main` and back |
| `hotfix/*` | short | urgent production fix; branched from `main`, merged to `main` and `develop` |

```bash
$ git switch develop
$ git switch -c feature/payments     # work happens off develop
# ... later, cut a release ...
$ git switch -c release/1.5 develop  # stabilise, then merge to main and tag
```

That structure is genuinely useful when several released versions are live at once and need
independent hotfixes — desktop apps, mobile apps, libraries, firmware. For a continuously
deployed web service, the `develop` branch and release ceremony usually add overhead with
little payoff.

## How branch protection and CI support each

None of these strategies is safe on trust alone.
[Branch protection](/learn/git/branch-protection/) enforces the rules a strategy *assumes*
— requiring PR review and **passing status checks** before anything lands on the protected
branch, and blocking force-pushes. **CI** runs the tests and linters that *prove* a branch
is mergeable, which is what makes "`main` is always deployable" credible rather than
aspirational.

The faster the strategy, the more it leans on this safety net: trunk-based development is
only sane *because* CI catches regressions on every tiny merge. Add automated checks via
[GitHub Actions](/learn/git/github-actions/) and require them in branch protection, and
small frequent merges stay trustworthy.

## Choosing one

There's no universally best model — fit it to your context:

| If your team… | Lean toward |
|---------------|-------------|
| Deploys a web app continuously, small/medium team | **GitHub Flow** |
| Deploys many times a day, mature CI, uses feature flags | **Trunk-based** |
| Ships versioned releases maintained in parallel | **Git Flow** |
| Is a solo project or learning | **GitHub Flow** (simplest that scales) |

Rules of thumb: **shorter release cadence and larger teams** favour shorter-lived branches
(trunk-based / GitHub Flow); **discrete versioned releases** justify Git Flow's structure.
Whatever you pick, the real win is **consistency** — a team that agrees on one model and
documents it beats a "perfect" model nobody follows the same way. Define how you
[merge vs rebase](/learn/git/rebasing/) too, and write it down in your contributing guide.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Git Flow's develop and release branches earn their keep when several versioned releases are maintained in parallel." markdown="0">
  <p class="knowledge-check__q">Quick check: which situation most justifies Git Flow's extra complexity?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A continuously deployed web app on a small team</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Versioned software with several releases live and hotfixed in parallel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A solo project just getting started</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **branching strategy** is a team convention over Git's branches and [pull requests](/learn/git/pull-requests/) — pick one deliberately.
- **Trunk-based**: one trunk, tiny frequent merges, **feature flags** for unfinished work; needs strong CI.
- **GitHub Flow**: branch → PR → review → merge → deploy, `main` always deployable — the lightweight default.
- **Git Flow**: `main` + `develop` plus `feature`/`release`/`hotfix` — worth it for **versioned, parallel releases**.
- **[Branch protection](/learn/git/branch-protection/)** + **CI** enforce and prove the safety each model assumes.
- Choose by **team size and release cadence**, then stay **consistent**.

Next up: the habits that make any strategy work — [commit hygiene & best practices](/learn/git/best-practices/).
