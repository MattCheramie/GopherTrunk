---
slug: make-vet-test
title: The make vet test gate
description: GopherTrunk's rule that vet plus unit tests must be green before any commit — what the command runs, why one command instead of two habits, and why the gate is non-negotiable.
keywords: make vet test, gophertrunk testing, commit gate, go vet before commit, makefile test target, pre-commit checks go, always green rule
level: intermediate
status: full
faq:
  - q: What does GopherTrunk's make vet test actually run?
    a: Two things, in sequence, across the whole repository — go vet's static analysis, then the full unit-test suite. One command, one green-or-red answer. The project's standing rule is that it must pass before any commit; the slower daemon and replay integration suites live behind a separate make integration target, run when the daemon, DSP, or replay paths change.
  - q: Why bundle vet and the tests into a single command?
    a: Because a gate people must remember two halves of is a gate people half-run. A single memorable command has no partial compliance — you either ran the gate or you didn't — and the same one line serves as the contributor documentation, the CI recipe, and the habit. Checks that are easy to run completely are the ones that actually get run.
  - q: Why is the gate per-commit instead of per-release or per-PR?
    a: Because every commit is a point someone may later build on, bisect through, or ship from. A broken commit poisons all three — most expensively bisection, where a history of sometimes-broken commits can no longer answer "which change broke this?" cleanly. Per-commit green keeps the whole history load-bearing.
---

# The make vet test gate

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Unit 6 studies quality practice in a real codebase. GopherTrunk's foundation
is one rule: **`make vet test` must be green before any commit** — static
analysis plus the full unit suite, bundled into **one command** so there's
nothing to remember, skip, or half-run. Slow suites are split into
**`make integration`** so the gate stays fast enough to obey. The rule is
per-commit because commits are what people **build on, bisect through, and
ship from** — and it's non-negotiable because a gate with exceptions isn't a
gate.
</div>

Five units of concepts; one unit of practice. Everything from here on is how a
real, working codebase — the one this site documents — wires those concepts
into daily habit. It starts with the smallest possible policy, which does the
most work.

## One command, one answer

GopherTrunk's contributor documentation reduces the whole of Units 2 and 4 to
a single line:

```bash
make vet test    # must be green before any commit
```

Under the hood it's exactly what this module taught, run across the whole
repo: [`go vet`](/learn/testing/linters-and-static-analysis/) first, then the
[unit suite](/learn/testing/what-is-a-unit-test/). While iterating you run
single packages (`go test ./internal/scanner/ccdecoder/...`); before
committing, you run the gate. (Make targets are just named command recipes —
the [toolchain lesson](/learn/programming-go/the-go-toolchain/) covers the
plumbing.)

The bundling *is* the design. A policy of "run vet, and also run the tests"
has two halves, and two-halved habits decay into whichever half is
remembered. `make vet test` has no halves: you ran the gate or you didn't,
and the answer is one green or one red. The same one line is simultaneously
the habit, the contributor doc, and the [CI job](/learn/testing/continuous-integration/)
— so what you check locally and what the machine enforces can't drift apart.

## The speed contract — and the split it forces

A per-commit gate lives or dies on speed: developers commit many times a day,
and a slow gate becomes a skipped gate, then a dead letter. That constraint
forces an architecture you've already met — the
[pyramid's split](/learn/testing/integration-tests/) between fast and slow
suites, here made policy:

| Command | Runs | When |
|---------|------|------|
| `go test ./internal/...` (one package) | The code you're editing | Constantly, while iterating |
| **`make vet test`** | Vet + all unit tests | **Before every commit — must be green** |
| `make integration` | Daemon, replay, end-to-end decode suites | When the daemon, DSP, or replay paths changed; in CI |

Notice what the split protects in *both* directions. The gate stays fast
because the [replay suites](/learn/testing/replay-integration-tests/) —
minutes of decoding recorded radio captures — live outside it. And the heavy
suites stay *run* because they have their own named command and their own
trigger rule ("touched the daemon or DSP? run it"), instead of being an
unenforced "please also remember."

## Why per-commit, and why no exceptions

Per-commit sounds strict until you list what a commit *is* in a working
repository — three roles, each poisoned by a broken one:

- **A base.** Teammates branch from it; a red base means everyone starts
  broken and can't attribute their own failures — the
  [keep-main-green](/learn/testing/continuous-integration/) argument, pushed
  down to every commit.
- **A bisection sample.** [`git bisect`](/learn/testing/bisecting-history/)
  interrogates history commit by commit; each broken commit is a `skip`, and
  enough of them turn "the exact commit" into "one of these seven." A
  history of green commits is a *searchable* history — an asset the project
  buys one gate-run at a time and cashes during its worst debugging days.
- **A shippable point.** Releases and rollbacks want to treat any commit as
  potentially deployable; "green except sometimes" means checking, which
  means sometimes not checking.

And why tolerate no exceptions — not "it's just a comment change," not "I'm
in a hurry"? Because exception-granting is a judgment call made exactly when
judgment is worst (tired, hurried, sure it's fine — the
[builder brain](/learn/testing/the-testing-mindset/) at its most confident),
and because each granted exception re-prices the next one. The gate's entire
value is that green is *unconditional*: anyone can build on, bisect through,
or ship from any commit without asking permission or checking provenance. A
rule this cheap to follow — seconds per commit — buys that property outright.

> Rule of thumb: make the mandatory path *one short command*, and keep it
> fast enough that skipping it saves nothing worth having.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the gate must be fast enough to run before every commit, so the slow replay and daemon suites are split into make integration with their own trigger rule." markdown="0">
  <p class="knowledge-check__q">Quick check: why aren't GopherTrunk's replay integration suites part of the per-commit make vet test gate?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They're less important than the unit tests</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A per-commit gate must stay fast or it gets skipped — slow suites run as make integration when the paths they cover change</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Integration tests can't run on developer machines at all</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk's foundation rule: **`make vet test` green before any commit** —
  vet plus the full unit suite, one command, one answer.
- **Bundling prevents decay**: no halves to forget, and habit, docs, and CI
  share the same line.
- The **speed contract** forces the healthy split: fast gate per commit,
  **`make integration`** for the heavy daemon/replay suites with their own
  trigger rule.
- Per-commit green keeps every commit **buildable-on, bisectable, and
  shippable** — the searchable-history payoff arrives on your worst debugging
  day.
- **No exceptions**, because exceptions are judged by builder brain at its
  most confident — and a gate with exceptions is a suggestion.

Next up: [Replay: testing a radio without a radio](/learn/testing/replay-integration-tests/)
