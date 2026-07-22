---
title: "Build in the Open, Part 8: Testing — How to Build and Write Tests"
description: How to test software well — the testing pyramid, table-driven tests, golden files, race detection, coverage limits, and writing tests with Claude Code.
keywords: how to write tests, testing pyramid, table-driven tests, golden files, race detector, test coverage, integration tests, claude code, github
category: tutorials
tags: [github, claude-code, testing, ci, go, quality]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 8
---

> **TL;DR:** Tests exist to let you change code without fear. Build them in
> layers — lots of fast unit tests, fewer integration tests, a handful of
> end-to-end tests — and make the fast layer the gate your CI enforces on every
> pull request. Write tests as data (table-driven cases), pin tricky outputs
> with golden files, turn the race detector on, and treat coverage as a
> spotlight, not a score. Claude Code is excellent at the tedious parts:
> generating table cases, hunting edge cases, and writing the regression test
> that locks a bug shut.

**Key takeaways**

- The point of a test is *confidence to change things*, not a green checkmark.
- Use the testing pyramid: many unit tests, fewer integration, few end-to-end.
- Make tests data, not code — table-driven cases scale better than copy-paste.
- Gate merges on the fast tests (Part 7's CI); keep slow/hardware tests opt-in.
- Coverage tells you what *ran*, not what's *correct* — use it to find gaps.

*This is Part 8 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **Why test at all** — and what's actually worth testing.
- **The testing pyramid** — unit, integration, end-to-end, and the right mix.
- **How to write good tests** — table-driven cases, fixtures, golden files,
  fakes vs. mocks, the race detector, and coverage.
- **Opt-in and hardware-gated tests** — for things CI can't run.
- **Tests as the CI gate** — connecting this to Part 7's workflows.
- **Writing tests with Claude Code** — where the leverage really is.
- **How GopherTrunk does it**, as a concrete example you can copy.

## Why test at all?

The honest answer isn't "to catch bugs" — it's **to let you change code without
being afraid of it**. A codebase with good tests is one you can refactor,
upgrade, and hand to a stranger, because the tests will shout the moment
something breaks. A codebase without them is one you tiptoe around.

That reframing also tells you *what* to test. You don't test getters and
one-line wrappers; you test the things that would hurt to get wrong:

- **Logic with branches** — anything with `if`/`switch`, math, parsing, or state.
- **Boundaries** — empty input, the maximum, off-by-one, the malformed packet.
- **Bugs you've already hit** — every fixed bug deserves a test so it stays fixed.
- **Contracts** — the promises your public functions and APIs make to callers.

Skip the trivial. A test that can only fail if the language itself is broken is
just maintenance cost.

## What is the testing pyramid?

The testing pyramid is a rule of thumb for the *mix* of tests, from cheap and
plentiful at the bottom to expensive and rare at the top:

1. **Unit tests** (the wide base): one function or type in isolation,
   milliseconds each, no network or disk. You write thousands of these and run
   them constantly.
2. **Integration tests** (the middle): several components wired together — a
   handler plus its database, a pipeline end-to-end — with real-ish parts but no
   real outside world. Slower, fewer.
3. **End-to-end tests** (the tip): the whole system as a user hits it. Slowest,
   flakiest, and you keep only a handful that cover the critical paths.

Most pain in real projects comes from an *inverted* pyramid — a few brittle
end-to-end tests and no unit tests underneath. Push the bulk of your coverage
down to the fast layer, where a failure points straight at the broken function.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The testing pyramid: a wide base of many fast unit tests, a narrower middle band of fewer integration tests, and a small tip of a handful of slow end-to-end tests. GopherTrunk's make test is the unit base, make integration the middle, and the per-protocol lights-up checks the tip.">
  <polygon points="170,20 40,185 300,185" fill="none" stroke="currentColor"/>
  <line x1="127" y1="75" x2="213" y2="75" stroke="currentColor"/>
  <line x1="83" y1="130" x2="257" y2="130" stroke="currentColor"/>
  <text x="170" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="10">e2e</text>
  <text x="170" y="107" text-anchor="middle" fill="currentColor" font-size="11">integration</text>
  <text x="170" y="162" text-anchor="middle" fill="var(--accent)" font-size="12">unit</text>
  <line x1="220" y1="50" x2="330" y2="50" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="336" y="46" fill="var(--fg-muted)" font-size="10">a handful, slowest</text>
  <text x="336" y="60" fill="var(--fg-muted)" font-size="9">per-protocol lights-up</text>
  <line x1="262" y1="103" x2="330" y2="103" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="336" y="99" fill="currentColor" font-size="10">fewer — make integration</text>
  <text x="336" y="113" fill="var(--fg-muted)" font-size="9">wired daemon, no SDR</text>
  <line x1="305" y1="158" x2="330" y2="158" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="336" y="154" fill="var(--accent)" font-size="10">thousands — make test</text>
  <text x="336" y="168" fill="var(--fg-muted)" font-size="9">go test -race, &lt;30s, the CI gate</text>
</svg>
<figcaption>Push the bulk of coverage to the wide base: GopherTrunk's <code>make test</code> unit layer is the gate CI enforces, with tagged integration tests and per-protocol end-to-end checks above it.</figcaption>
</figure>

## How do you write a good test?

A few techniques transfer to any language and test runner.

### Make tests data: table-driven cases

Instead of copy-pasting a test five times with different inputs, define a list
of cases — input, expected output, a name — and loop over them. Adding a case
becomes one line, and the failure message tells you exactly which row broke.
Every mature test suite leans on this.

### Pin tricky outputs with golden files

When the expected output is large or fiddly (rendered HTML, a decoded frame, a
formatted report), store a known-good copy as a *golden file* and assert the
test output matches it. A flag like `-update` regenerates the goldens after an
intentional change. You review the diff in code review like any other change.

### Fakes vs. mocks

Both stand in for real dependencies, but they're not the same:

- A **fake** is a working lightweight implementation — an in-memory database, a
  scripted data source. It behaves; you assert on results.
- A **mock** records calls and lets you assert *that* something was called, with
  what arguments. Useful, but over-mocking produces tests that pass while the
  real system is broken. Prefer fakes when you can.

### Turn on the race detector

If your language has concurrency, it has a race detector or equivalent (Go's
`-race`, ThreadSanitizer, etc.). Run your tests under it. Data races are the
bugs that pass a thousand times and corrupt data on the thousand-and-first;
the detector catches them deterministically.

### Use coverage as a spotlight, not a score

Coverage tells you which lines *executed* during the tests — not whether the
assertions were meaningful. Chasing 100% rewards writing tests for trivial code
and punishes nothing. Use coverage to *find untested branches* you care about,
then decide if they're worth a test. The number is a map, not the territory.

## Opt-in and hardware-gated tests

Some tests can't run in CI: they need a GPU, a USB device, a paid API key, a
specific OS. Don't delete them — **gate** them. Make the test skip by default
unless an environment variable or build flag opts it in. The test lives in the
repo, documents the expected behaviour, and runs on the one machine that can,
while CI stays green and fast.

## Tests as the CI gate

This is where Part 7 pays off. Once you have a fast, reliable test command, you
wire it into a CI workflow that runs on every pull request, and you mark it a
**required status check** so a PR can't merge until it's green. That single rule
turns "we have tests" into "broken code can't reach `main`." The fast layer
(unit + the cheap integration tests) is the gate; the slow and hardware tests
stay opt-in and run out of band.

## Writing tests with Claude Code

Tests are some of the highest-leverage work to hand to Claude Code, because the
work is structured and the right answer is checkable:

- **Generate table cases.** Give it a function and ask for a table-driven test
  covering the obvious paths — it'll scaffold the cases and the loop in seconds.
- **Hunt edge cases.** "What inputs would break this?" surfaces the empty
  string, the negative number, the overflow, the nil you forgot.
- **Write the regression test for a bug.** Describe the bug (or paste the stack
  trace), and ask for a test that fails *before* the fix and passes *after*. Land
  that test in the same PR as the fix — exactly the discipline good projects use.

<figure class="lab-figure">
<svg viewBox="0 0 540 170" width="540" height="170" role="img" aria-label="The red, green, refactor loop: first write a failing test that is red, then write just enough code to make it pass and go green, then refactor while keeping the tests green, and repeat. For a bug fix the failing test comes first and ships in the same pull request as the fix.">
  <rect x="20" y="30" width="150" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="49" text-anchor="middle" fill="var(--accent)" font-size="11">1 · red</text>
  <text x="95" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="9">write a failing test</text>
  <line x1="170" y1="52" x2="187" y2="52" stroke="currentColor"/><polygon points="187,48 193,52 187,56" fill="currentColor"/>
  <rect x="195" y="30" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="270" y="49" text-anchor="middle" fill="currentColor" font-size="11">2 · green</text>
  <text x="270" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="9">make it pass</text>
  <line x1="345" y1="52" x2="362" y2="52" stroke="currentColor"/><polygon points="362,48 368,52 362,56" fill="currentColor"/>
  <rect x="370" y="30" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="445" y="49" text-anchor="middle" fill="currentColor" font-size="11">3 · refactor</text>
  <text x="445" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="9">clean up, stay green</text>
  <line x1="445" y1="74" x2="445" y2="140" stroke="currentColor"/>
  <line x1="445" y1="140" x2="95" y2="140" stroke="currentColor"/>
  <line x1="95" y1="140" x2="95" y2="80" stroke="currentColor"/><polygon points="91,80 95,74 99,80" fill="currentColor"/>
  <text x="270" y="135" text-anchor="middle" fill="var(--fg-muted)" font-size="9">repeat — for a bug fix, the red test lands in the same PR</text>
</svg>
<figcaption>The red→green→refactor loop, and the rule GopherTrunk enforces: a bug fix ships with a regression test that fails before the fix and passes after it, in the same PR.</figcaption>
</figure>

Always read what it produces. Claude is great at breadth; you supply the
judgment about which cases actually matter and whether the assertions are real.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) is a pure-Go SDR
trunking scanner, and its testing maps the pyramid onto a `Makefile` you can read
top to bottom:

- **The fast unit layer is `make test`.** It runs
  `go test -race -count=1 ./...` — every package, under the race detector, with
  caching disabled — and finishes in **under 30 seconds**. This is the command
  contributors run constantly and the one CI gates on.
- **Integration tests are build-tagged so they don't slow the unit run.**
  `make integration` runs the tests behind `//go:build integration`, booting the
  wired daemon end-to-end (no real SDR) and asserting the engine, recorder, call
  log, metrics, and API all agree on a synthetic call. The build tag keeps them
  out of the default `make test` path.
- **Per-protocol "lights up" checks.** There's a focused integration target per
  trunked protocol — `make integration-cc-nxdn`, `make integration-cc-dmr`,
  `make integration-cc-tetra`, `make integration-cc-p25p2`, and so on — each of
  which synthesizes IQ for that protocol's control channel and asserts the real
  pipeline recovers the lock. That's an end-to-end critical path, isolated per
  protocol so a failure points right at the broken decoder.
- **Table-driven tests and `t.Parallel()` are house rules.** `CONTRIBUTING.md`
  spells it out: tests are "parallel where it's safe (`t.Parallel()`),
  table-driven for any function with more than two interesting inputs,
  `t.Helper()` on helper functions so failure locations surface correctly."
- **Golden IQ captures live under `samples/`.** Known-good signal captures act
  as fixtures the decoders run against — golden files for radio.
- **Real-hardware tests are env-gated.** `make test-airspy-real` sets
  `GOPHERTRUNK_AIRSPY_REAL=1`; the package skips entirely unless that variable is
  set, so the test ships in the repo but "never runs in CI," exactly as
  `CONTRIBUTING.md` says. Overrides like `GOPHERTRUNK_AIRSPY_REAL_BIAS_TEE=1`
  toggle extra checks. CI runs `make test`, `make integration`, and
  `make test-dvsi` on every PR — a green CI is required before merge.

You don't need a radio for any of this to transfer. Swap "decode a control
channel" for "render an invoice" or "parse a config file": fast unit tests as
the gate, tagged integration tests for the wiring, golden files for fiddly
output, and opt-in tests for whatever CI can't reach.

## FAQ

**How many tests is enough?**
Enough that you'd trust a stranger to refactor your code and find out from a
failing test, not from production. That usually means solid unit coverage of
your branching logic plus a few integration tests over the critical paths — not
a coverage percentage you chase for its own sake.

**Should I write tests before or after the code?**
Either works; the discipline that matters is that tests exist and run in CI.
Test-first (TDD) helps when the design is unclear; test-after is fine when it
isn't. For bug fixes, always write the failing test first — it proves the bug is
real and that your fix actually fixes it.

**What's the difference between a fake and a mock?**
A fake is a real (if simplified) working implementation you assert results
against — an in-memory store, a scripted source. A mock records calls so you can
assert that something was invoked. Fakes tend to produce more durable tests;
heavy mocking can make tests pass while the system is broken.

**How do I test something that needs hardware or a paid API?**
Gate it. Make the test skip unless an environment variable or build tag opts it
in, the way GopherTrunk gates its Airspy tests behind
`GOPHERTRUNK_AIRSPY_REAL=1`. The test stays in the repo and runs where it can,
while CI stays fast and green.

**Can Claude Code write my tests for me?**
It can write most of the scaffolding — table cases, edge cases, regression tests
for a described bug — very well, and very fast. You still review the result and
decide which cases matter. Treat it as a fast pair, not an oracle.

## Series navigation

**Part 8 of 14** · ←
[Part 7]({{ '/blog/tutorials/build-in-the-open-07-github-actions-which-workflows/' | relative_url }})
· Next →
[Part 9: Documentation Done Right]({{ '/blog/tutorials/build-in-the-open-09-documentation-done-right/' | relative_url }})
