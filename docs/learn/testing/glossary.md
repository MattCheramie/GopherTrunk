---
slug: glossary
title: Glossary of testing terms
description: Plain-language definitions for every term in the testing module — unit test, assertion, fixture, mock, coverage, regression, CI, flaky test, fuzzing, bisect, root cause, and more — cross-linked to the lessons that explain them.
keywords: testing glossary, unit test definition, regression test, test fixture, mock stub fake, code coverage, flaky test, fuzzing, git bisect, root cause analysis, ci glossary, failing first
level: beginner
status: full
lesson_standalone: true
---

# Glossary of testing terms

Every term used across the [Testing &amp; Software Quality](/learn/testing/)
module, defined in plain language and linked to the lesson where it's explained
in full. Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump
to a word. Terms are grouped by theme, roughly in the order the module
introduces them.

> Looking for reference material beyond this module's terms? The
> [Field Guide](/reference/) covers the wider software-development, RF, and
> hardware vocabulary GopherTrunk's documentation draws on.

## Bugs & quality basics

**Bug** — Catch-all word for a mistake, the defect it left in code, and the
failure that defect causes. See [What is a bug?](/learn/testing/what-is-a-bug/).

**Defect (fault)** — The wrong code itself, sitting silently in the source —
possibly for years — until conditions expose it. See
[What is a bug?](/learn/testing/what-is-a-bug/).

**Failure** — The observable wrong behavior a defect produces at runtime, often
far from where the defect lives. See
[What is a bug?](/learn/testing/what-is-a-bug/).

**Edge case** — A rare or extreme input — empty, zero, maximum, malformed —
where bugs concentrate because nobody imagined it. See
[Why does software break?](/learn/testing/why-software-breaks/).

**Regression** — A bug in behavior that used to work, usually introduced by a
later change. See [Why does software break?](/learn/testing/why-software-breaks/).

**Cost curve** — The same defect costs orders of magnitude more the later it's
found — seconds in the editor, disasters in production. See
[What does a bug cost?](/learn/testing/cost-of-a-bug/).

**Verification** — Checking software against its stated expectations ("built
the thing right?") — what tests do. See
[What is testing?](/learn/testing/what-is-testing/).

**Validation** — Checking that software solves the user's actual problem
("built the right thing?") — needs reality, not just tests. See
[What is testing?](/learn/testing/what-is-testing/).

**Test pyramid** — Many fast unit tests at the base, fewer integration tests,
a handful of end-to-end tests on top. See
[What is testing?](/learn/testing/what-is-testing/).

## Unit testing

**Unit test** — A fast test of one small piece of code in isolation — no real
databases, networks, files, or hardware. See
[What is a unit test?](/learn/testing/what-is-a-unit-test/).

**Arrange, act, assert** — The three-beat shape of nearly every test: set up,
call the code once, check the prediction. See
[Anatomy of a test](/learn/testing/anatomy-of-a-test/).

**Assertion** — The checked prediction in a test — the comparison that makes a
wrong answer detectable. See
[Anatomy of a test](/learn/testing/anatomy-of-a-test/).

**`go test`** — Go's built-in test runner: finds `TestXxx` functions in
`_test.go` files, no framework needed. See
[Your first Go test](/learn/testing/your-first-go-test/).

**`testing.T`** — The handle every Go test receives: `t.Errorf`/`t.Fatalf` to
report failures, `t.Run` for subtests. See
[Your first Go test](/learn/testing/your-first-go-test/).

**Table-driven test** — Go's signature idiom: cases as a slice of structs, one
loop running the same body over every row. See
[Table-driven tests](/learn/testing/table-driven-tests/).

**Test double** — Any stand-in for a real dependency, swapped in through an
interface so units stay isolated. See
[Fakes, stubs, and mocks](/learn/testing/test-doubles/).

**Stub** — A double that returns canned answers and remembers nothing. See
[Fakes, stubs, and mocks](/learn/testing/test-doubles/).

**Fake** — A double that's a real, lightweight working implementation — a map
posing as a database. See
[Fakes, stubs, and mocks](/learn/testing/test-doubles/).

**Mock** — A double that records how it was called, so the test can assert on
the interactions. See [Fakes, stubs, and mocks](/learn/testing/test-doubles/).

**Code coverage** — The percentage of statements executed by tests: a
flashlight for finding untested code, not a certificate of quality. See
[Code coverage](/learn/testing/code-coverage/).

## Bigger tests

**Integration test** — A test wiring real components together — code plus real
databases, files, or processes — to verify the joints doubles can't reach. See
[Integration tests](/learn/testing/integration-tests/).

**End-to-end (E2E) test** — A test driving the whole assembled system through
its real entry points, asserting what a user would observe. See
[End-to-end tests](/learn/testing/end-to-end-tests/).

**Regression test** — A test pinning a fixed bug's exact triggering input into
the suite, so the bug can never silently return. See
[Regression tests &amp; failing first](/learn/testing/regression-tests/).

**Failing first** — Writing the regression test before the fix and watching it
fail — proof you reproduced the bug and that the test can detect it. See
[Regression tests &amp; failing first](/learn/testing/regression-tests/).

**Fixture** — Stored input data a test loads — in Go, conventionally from
`testdata/`. See
[Golden files &amp; fixtures](/learn/testing/golden-files-and-fixtures/).

**Golden file** — A stored expected output compared byte-for-byte against what
the code produces, regenerated only via a deliberate `-update` flag. See
[Golden files &amp; fixtures](/learn/testing/golden-files-and-fixtures/).

**Property-based testing** — Stating a claim that must hold for all inputs and
letting generated random cases hunt for a counterexample. See
[Property-based testing](/learn/testing/property-based-testing/).

**Round-trip property** — `decode(encode(x)) == x` for any x — the workhorse
property for codec pairs, which proves self-consistency only. See
[Property-based testing](/learn/testing/property-based-testing/).

**Fuzzing** — Hammering code with mutated, mostly-malformed inputs hunting for
crashes; built into Go as `go test -fuzz`. See
[Fuzzing](/learn/testing/fuzzing/).

## Quality tooling

**Static analysis** — Finding bugs by reading source without running it —
covering all code, tested or not. See
[Linters &amp; static analysis](/learn/testing/linters-and-static-analysis/).

**Linter** — A static-analysis tool flagging suspicious patterns; staticcheck
and golangci-lint extend Go's built-in vet. See
[Linters &amp; static analysis](/learn/testing/linters-and-static-analysis/).

**`go vet`** — Go's standard analyzer — printf mismatches, copied locks,
unreachable code — with near-zero false positives. See
[Linters &amp; static analysis](/learn/testing/linters-and-static-analysis/).

**gofmt** — Go's canonical, non-configurable formatter: one automatic style,
noise-free diffs, no arguments. See
[Formatters &amp; style](/learn/testing/formatters-and-style/).

**Code review** — A second human reading a change before merge — the layer
that catches wrong assumptions and tests that don't test. See
[Code review](/learn/testing/code-review/).

**Continuous integration (CI)** — Automatically building and testing every
push on a clean machine, reporting checks on each pull request. See
[Continuous integration](/learn/testing/continuous-integration/).

**Flaky test** — A test that passes and fails without the code changing —
teaching the team that red means re-run. See
[Flaky tests](/learn/testing/flaky-tests/).

**Race detector** — Go's `-race` mode: reports unsynchronized concurrent
memory access with stack traces during a run. See
[Flaky tests](/learn/testing/flaky-tests/).

**Branch protection** — Platform rules that enforce checks and reviews: no
merge until required checks pass and approvals land. See
[Required checks &amp; branch protection](/learn/testing/required-checks/).

**Required check** — A CI job explicitly marked merge-blocking; unmarked jobs
report but block nothing — the green-checkmark-red-build trap. See
[Required checks &amp; branch protection](/learn/testing/required-checks/).

## Debugging

**Reproduction** — Making a failure happen on demand — the precondition for
experimenting and for knowing a fix worked. See
[Reproducing a bug](/learn/testing/reproducing-a-bug/).

**Minimal reproduction** — The smallest input, code path, and environment that
still triggers the bug, found by cutting the scenario in half repeatedly. See
[Reproducing a bug](/learn/testing/reproducing-a-bug/).

**"Works on my machine"** — Evidence that the trigger lives in an
environmental difference — a clue to diff environments, not a verdict. See
[Reproducing a bug](/learn/testing/reproducing-a-bug/).

**Stack trace** — The chain of calls at a failure, innermost frame first —
read top-down to the first frame in your own code. See
[Reading error messages &amp; stack traces](/learn/testing/reading-error-messages/).

**Panic** — Go's runtime crash: a diagnosis line with the values involved,
then per-goroutine stack traces. See
[Reading error messages &amp; stack traces](/learn/testing/reading-error-messages/).

**Print debugging** — Statements showing what the program actually does —
sharpest when each print tests a hypothesis and bisects the pipeline. See
[Print debugging &amp; logging](/learn/testing/print-debugging-and-logging/).

**Structured logging** — Permanent, leveled, key-value log output (Go's
`slog`) that can be queried — the field failure's only witness. See
[Print debugging &amp; logging](/learn/testing/print-debugging-and-logging/).

**Debugger** — A tool that pauses a live program at breakpoints and exposes
all state, with line-by-line stepping; Go's is Delve (`dlv`). See
[Using a debugger](/learn/testing/using-a-debugger/).

**Bisecting** — Binary-searching history for the first bad commit with
`git bisect` — ~10 tests per 1,000 commits, automatable via `bisect run`. See
[Bisecting history](/learn/testing/bisecting-history/).

**Root cause** — The point in a bug's cause chain where a fix would have
prevented it — usually a wrong assumption at a trust boundary. See
[Root-cause analysis](/learn/testing/root-cause-analysis/).

**Five whys** — Repeatedly asking "why?" down the cause chain, past the code
into the conditions that let the defect ship. See
[Root-cause analysis](/learn/testing/root-cause-analysis/).

## GopherTrunk practice

**`make vet test`** — GopherTrunk's per-commit gate: `go vet` plus the full
unit suite in one command, green before any commit. See
[The make vet test gate](/learn/testing/make-vet-test/).

**`make integration`** — The heavier opt-in suite — daemon and replay tests —
run when the daemon, DSP, or replay paths change. See
[The make vet test gate](/learn/testing/make-vet-test/).

**IQ capture** — A recorded file of raw SDR samples: the radio moment frozen,
replayable bit-identically forever. See
[Replay: testing a radio without a radio](/learn/testing/replay-integration-tests/).

**Replay test** — An integration test feeding a capture file through the same
decode pipeline live radio uses — whole-decoder determinism. See
[Replay: testing a radio without a radio](/learn/testing/replay-integration-tests/).

**Self-consistent synthetic trap** — A test whose encode and decode sides
share the same wrong assumption: they agree with each other, pass every
round-trip, and fail against the real world. See
[The self-consistent synthetic trap](/learn/testing/the-self-consistent-synthetic-trap/).

**Independent reference** — Expected values your own misunderstanding cannot
have produced — real captures, reference implementations, published test
vectors. See
[The self-consistent synthetic trap](/learn/testing/the-self-consistent-synthetic-trap/).

**Capture-gated verification** — A fix isn't verified — and its issue isn't
closed — until a failing-first regression test passes *and* the symptom is
shown gone against real captured data or by the reporter. See
[Capture-gated verification](/learn/testing/capture-gated-verification/).

**Narrow commit** — A bug fix shipped as one focused commit — fix plus its
failing-first test, no bundled refactors. See
[Write your first regression test](/learn/testing/your-first-regression-test/).
