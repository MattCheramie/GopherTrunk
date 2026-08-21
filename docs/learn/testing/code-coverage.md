---
slug: code-coverage
title: Code coverage
description: What coverage measures, how to read go test -cover and the HTML heat map, and why 100% is the wrong goal — coverage finds untested code, it doesn't certify tested code.
keywords: code coverage go, go test -cover, coverage html report, coverage percentage meaning, 100 percent coverage myth, coverprofile, test coverage explained
level: intermediate
status: full
prereq:
  - your-first-go-test
---

# Code coverage

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Code coverage** measures which lines of code your tests *executed* — in Go,
via **`go test -cover`** and the **`-coverprofile`** HTML heat map. Coverage is
excellent at its real job: **finding untested code**. It is terrible at the job
people assign it: certifying quality — a line can be executed by a test that
**asserts nothing**. Use coverage as a **flashlight**, not a **target**: chasing
a number produces assertion-free tests that inflate confidence without adding
any.
</div>

After a few lessons of writing tests, a natural question: how much of the code
do my tests actually reach? Tooling answers precisely. Interpreting the answer
wisely is the skill this lesson is really about.

## What coverage measures

Run tests with the flag:

```bash
go test -cover ./...
```

```text
ok   example/freqfmt    0.003s  coverage: 78.6% of statements
```

Go instruments the build, runs the suite, and reports the percentage of
**statements executed at least once** during the run. For the map of *which*
statements, write a profile and open the HTML view:

```bash
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

Your source renders with executed code in green and never-executed code in red.
That red is the genuinely valuable output — more on it in a moment.

## What coverage does not measure

Here is the whole trap in four lines. This test yields identical coverage to a
rigorous one:

```go
func TestFormatMHz(t *testing.T) {
    FormatMHz(851_012_500) // executed: covered. Checked: nothing.
}
```

Every statement in `FormatMHz` ran, so every statement is "covered" — and the
function could return garbage, panic-adjacent nonsense, or the works of
Shakespeare without this test noticing. **Coverage counts execution, not
verification.** A covered line means *a test caused this line to run*, nothing
more. Whether any prediction was checked — the thing that
[makes a test a test](/learn/testing/what-is-testing/) — is invisible to the
metric.

This isn't a corner case; it's what reliably happens when coverage becomes a
target. Set a team goal of 90% and humans will produce the cheapest tests that
execute lines — often assertion-light, meaning-free ones. The number rises;
safety doesn't. (Goodhart's law: when a measure becomes a target, it stops
being a good measure.)

## The right way to use it: as a flashlight

Point coverage at your code and look at what's **red**. Red answers a question
no other tool answers: *what have I never tested at all?* Typical finds:

- **Error paths.** The happy path is green; every `if err != nil` branch is
  red. Untested error handling is where crashes hide — an error path that's
  never run may itself be broken (Go's
  [error-handling lesson](/learn/programming-go/error-handling-patterns/) pairs
  well here).
- **A forgotten branch.** One arm of a `switch` over message types never
  executes — no test ever feeds that message type.
- **Dead code.** Red that *can't* be reached is a cleanup candidate.

Then apply judgment, because not all red deserves tests. A trivial getter?
Leave it. The branch that handles a truncated radio message? That red is a bug
report waiting to happen — write the test. Criticality, not the percentage,
decides.

> Rule of thumb: read the red, not the number. "What untested code matters
> most?" is a great weekly question; "how do we hit 90%?" is a bad one.

## So what's a good number?

Honest answer: it depends on what the package does, and the number is the least
interesting part. Mature Go projects commonly sit anywhere from 60% to 90%,
higher in pure-logic packages (parsers, math — cheap to test, costly to get
wrong) and lower at the hardware and OS edges where
[integration tests](/learn/testing/integration-tests/) carry the load instead.
A DSP function's package can reasonably run near-complete coverage; the code
that opens a USB radio device cannot, and pretending otherwise produces
ceremony, not safety. 100% overall is the wrong goal everywhere: the marginal
cost of the last percent is enormous, and the marginal safety is ~zero — effort
that [regression tests](/learn/testing/regression-tests/) for real bugs would
repay far better.

Where coverage *shines* as a number is **direction**: a package that drifts
from 80% to 60% over six months is accumulating untested change — worth a look
regardless of what the "right" level is.

<div class="knowledge-check" data-quiz data-correct-msg="Right — coverage counts executed statements only; whether anything was asserted about their behavior is invisible to it." markdown="0">
  <p class="knowledge-check__q">Quick check: a package reports 100% coverage. What do you actually know?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Every behavior of the package has been verified correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The package has no bugs on the tested inputs</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every statement executed during tests — possibly without a single assertion checking the results</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Coverage** = percentage of statements *executed* by tests: `go test
  -cover`, with `-coverprofile` + `go tool cover -html` for the heat map.
- It measures **execution, not verification** — an assertion-free test covers
  lines while checking nothing.
- Used as a **target**, it corrupts (Goodhart); used as a **flashlight**, the
  red regions reveal untested error paths, branches, and dead code.
- Judge red by **criticality**: test the truncated-message branch, skip the
  getter.
- Watch the **trend** more than the level; spend the last-percent effort on
  real regression tests instead.

Next up: [Integration tests](/learn/testing/integration-tests/)
