---
slug: flaky-tests
title: Flaky tests
description: Tests that pass sometimes are worse than tests that fail always — where flakiness comes from (timing, order, shared state, real networks), how to hunt it, and why "flake" is not a root cause.
keywords: flaky tests, intermittent test failures, test nondeterminism, race conditions in tests, go test race detector, sleep in tests, test order dependence, quarantine flaky test
level: intermediate
status: full
prereq:
  - continuous-integration
---

# Flaky tests

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **flaky test** passes and fails **without the code changing**. It's worse
than a failing test, because it teaches the team to **re-run instead of read**
— and once red might mean flake, red means nothing. Flakiness has causes, and
they're few: **timing assumptions**, **order dependence**, **shared state**,
**real networks/clocks**, and genuine **race conditions** (which
`go test -race` exists to expose). "It's just flaky" is not a diagnosis — a
flake is a real bug in the test *or the code*, and the second kind has shipped
disasters.
</div>

CI's authority rests on one equivalence: red means broken, green means fine.
This lesson is about the failure mode that dissolves that equivalence — and
why the standard team reflex toward it ("re-run until green") is quietly
corrosive.

## Why sometimes-failing beats always-failing… at causing damage

An always-failing test gets fixed — it's in everyone's face. A flaky test
trains behavior instead. The first time it fails spuriously, someone re-runs
and it passes. The lesson everyone learns: *red might mean nothing*. From
there the decay is mechanical: failures get re-run before they get read; a
**real** regression that fails the same test gets re-run too (twice, three
times — flakes sometimes pass, so eventually it… doesn't, and gets
investigated late, or worse, the PR merges on the lucky green). One habitual
flake makes *every* red ambiguous — the trust that makes
[the pipeline](/learn/testing/continuous-integration/) worth having is gone,
suite-wide, not just for the one test.

## Where flakiness comes from

Nondeterminism doesn't appear from nowhere. Nearly every flake traces to one
of five sources:

| Source | The pattern | The fix |
|--------|-------------|---------|
| **Timing assumptions** | `time.Sleep(100ms)` then assert — enough on your laptop, not on a loaded CI box | Wait on the **condition** (poll with a deadline), never the clock |
| **Order dependence** | Test B passes only because test A ran first and left state | Each test arranges its own world (`t.TempDir()`, fresh structs) |
| **Shared state** | Two tests use one fixed port / global / file path — collide when parallel | Unique per-test resources; port 0 ("assign me any free port") |
| **Real external systems** | Test hits a live network service or depends on wall-clock time/dates | [Doubles](/learn/testing/test-doubles/) at the unit level; controlled local instances for integration |
| **Actual races** | The code under test has a concurrency bug that only sometimes loses | That's not test flakiness — that's a **found bug** |

The first row is the champion by volume. Any test containing a bare sleep is
a bet that the machine will be fast enough today — and CI boxes exist to lose
that bet.

## "Flake" is not a root cause

The most important sentence in this lesson: **a flaky test is a
reproducibility bug, and it lives either in the test or in the code.** The
word "flaky" describes the symptom; it explains nothing. When you actually dig
— and Unit 5's [reproduction techniques](/learn/testing/reproducing-a-bug/)
are exactly the toolkit — you find one of two things:

- **The test is wrong** (a sleep, a shared port, an order dependence). Fix it;
  the suite gets more trustworthy.
- **The code is wrong** — a genuine race or timing hazard that the test is
  *correctly, intermittently detecting*. This is the one that matters:
  the same nondeterminism will fire in production, where it's called an
  outage. The Therac-25 overdoses from
  [the cost lesson](/learn/testing/cost-of-a-bug/) were exactly this shape —
  a race that only fired on certain operator timing.

Go gives you a sharp instrument for the second kind:

```bash
go test -race ./...          # data-race detector on the whole suite
go test -run TestX -count=500  # hammer one test until the flake shows
```

The **race detector** watches actual memory accesses during the run and
reports concurrent unsynchronized access with stack traces — turning "fails
sometimes, can't reproduce" into a named line number. `-count=N` is the
flake-hunter's other friend: run the suspect hundreds of times and measure the
failure rate instead of shrugging at it.

> Rule of thumb: never delete, skip, or `-count`-retry your way past a flake
> until you know *which side* the bug is on. One of the two possibilities is a
> production incident on layaway.

## Managing flakes without losing the suite

Practical triage for a team: when a flake appears, **file it and quarantine
it** — move it out of the merge-blocking path (Go's `t.Skip` with a tracking
issue, or a separate non-required job) so it stops training people to ignore
red, but *keep it visible* so it gets fixed rather than forgotten. Quarantine
with no follow-up is deletion in slow motion, and deletion throws away what
might be your only witness to a real race. Measure, too: CI platforms can
report which tests fail-then-pass on retry; a leaderboard of repeat offenders
turns "the suite feels unreliable" into a short, fixable list.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a flake is a real bug in the test or in the code, and the code-side kind is a production race caught early. 'Flaky' is a symptom, not an explanation." markdown="0">
  <p class="knowledge-check__q">Quick check: a test fails about once in fifty CI runs. What's the correct engineering stance?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Configure CI to auto-retry it — a 1-in-50 failure is noise</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete it — an unreliable test provides no signal</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Investigate which side the nondeterminism is on — it's a bug in the test or a race in the code, and the second ships outages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **flaky test** passes and fails without code changes — and teaches the
  team that *red means re-run*, dissolving the suite's authority.
- Sources are few: **timing sleeps**, **order dependence**, **shared state**,
  **real networks/clocks**, and **genuine races**.
- Wait on **conditions, not clocks** — the single biggest flake killer.
- **"Flaky" is a symptom, not a root cause**: the bug is in the test or in the
  code, and the code-side kind fires in production too — `go test -race` and
  `-count=N` are the hunting tools.
- **Quarantine visibly, then fix** — never silently retry or delete your only
  witness to a race.

Next up: [Required checks & branch protection](/learn/testing/required-checks/)
