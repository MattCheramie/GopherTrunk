---
slug: continuous-integration
title: Continuous integration
description: Every push builds and tests automatically on a clean machine — pipelines, checks on pull requests, and why CI is the team's shared safety net rather than a bureaucratic hurdle.
keywords: continuous integration, ci pipeline, github actions ci, ci for go, automated builds, works on my machine, ci checks pull request, keep main green
level: intermediate
status: full
prereq:
  - integration-tests
---

# Continuous integration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Continuous integration (CI)** runs your build, linters, and tests
**automatically on every push**, on a **clean machine** that has none of your
laptop's accumulated state — which is why it catches "works on my machine"
that local runs can't. A **pipeline** of jobs reports pass/fail **checks** on
each pull request, making every change's health visible before merge. CI's
prime directive is **keep main green**: a red main blocks everyone, so fixing
it outranks feature work. Speed matters — slow pipelines rot into ignored
pipelines.
</div>

Everything this module has taught — tests, vet, formatting, review — shares a
weakness: it runs when someone remembers to run it. CI removes the
*remembering*. It's the machinery that turns individual discipline into a
team-wide guarantee.

## What CI actually does

Wire your repository to a CI service — GitHub Actions is the one you'll meet
first, and [the Git module covers its mechanics](/learn/git/github-actions/) —
and every push triggers a **pipeline**: a fresh virtual machine checks out
your exact commit and runs the project's checks from zero. A representative Go
pipeline:

```yaml
# .github/workflows/ci.yml (abridged)
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 'stable' }
      - run: test -z "$(gofmt -l .)"   # formatting is canonical
      - run: go vet ./...              # static analysis
      - run: go test ./...             # the unit suite
      - run: go build ./...            # it actually compiles, from scratch
```

Each job reports a green ✓ or red ✗ **check** on the commit and its pull
request. Reviewers see at a glance whether a change passes everything before
spending human attention on it — machines gate first, humans judge second.

## The clean-machine effect

Why does CI catch things your own `go test ./...` didn't? Because your laptop
is a **dirty environment** in the technical sense: it has files, environment
variables, cached builds, globally installed tools, and local edits you've
forgotten about. Code can *depend* on that state without anyone noticing — the
config file that exists only on your disk, the dependency you installed by
hand a year ago, the test that passes because a previous run left data behind.

The CI machine has none of it. Starting from a bare checkout, every hidden
dependency fails loudly — which converts "works on my machine" from a shrug
into a diagnosis: *whatever my machine has that the clean one doesn't is part
of the bug*. CI is also where the slower suites live:
[integration and end-to-end tests](/learn/testing/integration-tests/) too
heavy for the every-save loop run on every PR instead, where their minutes
cost no one's flow. GopherTrunk's CI follows exactly this split — the fast
`vet`+unit gate plus heavier jobs (integration, cross-platform builds, web
console typechecks) that no contributor could be expected to run by hand every
time.

## Keep main green

CI's social contract has one central rule: **the main branch passes, always.**
Everything flows from it:

- Everyone branches from main. If main is red, every developer starts broken
  and can't tell their new failures from the inherited ones.
- Releases cut from main. A red main means you *can't ship*, even urgently.
- A green main makes every failure *attributable*: your PR's red is your
  change's red — nothing else could have caused it.

Hence the etiquette: a change that breaks main gets fixed or **reverted
immediately** — fixing main outranks feature work, whoever broke it. This
isn't bureaucracy; it's what keeps the safety net load-bearing. The mechanism
that *enforces* green-before-merge — required checks and branch protection —
gets [its own lesson](/learn/testing/required-checks/), including the sneaky
failure mode where a "green" PR hides a red job nobody made required.

> Rule of thumb: a red main is a site-wide outage for the team. Treat it with
> outage urgency.

## Speed is a feature of the pipeline itself

A pipeline's value decays with its latency. At 5 minutes, developers wait for
green before merging. At 45, they context-switch, stack changes on unverified
changes, and pressure grows to merge "it'll probably pass." The
[cost-curve](/learn/testing/cost-of-a-bug/) logic applies to the pipeline too:
feedback delayed is feedback degraded. Standard levers, in order: **cache**
dependencies and build artifacts between runs; **parallelize** independent
jobs (lint ∥ test ∥ build); run the fast checks first so obvious failures
report in seconds; and hunt down [flaky tests](/learn/testing/flaky-tests/) —
the next lesson — because a pipeline that needs re-running is a pipeline twice
as slow and half as trusted.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a clean machine has none of your laptop's hidden state, so dependencies on that state fail loudly instead of silently passing." markdown="0">
  <p class="knowledge-check__q">Quick check: tests pass on your machine but fail in CI. What's the most likely category of cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The CI service runs a broken copy of Go</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">CI machines execute tests in a random order to save money</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The code or tests depend on something your machine has that a clean checkout doesn't</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **CI** runs build + vet + tests **automatically on every push**, reporting
  pass/fail **checks** on each pull request.
- The **clean machine** is the point: hidden dependencies on your laptop's
  state fail loudly — "works on my machine" becomes a diagnosis.
- Slow suites (integration, E2E, cross-platform) live in CI, where their
  minutes don't break anyone's flow.
- **Keep main green** — red main is a team-wide outage; fix or revert
  immediately, whoever caused it.
- **Pipeline speed is a feature**: cache, parallelize, fail fast, and kill
  flakes before they kill trust.

Next up: [Flaky tests](/learn/testing/flaky-tests/)
