---
slug: integration-tests
title: Integration tests
description: Testing real pieces wired together — databases, files, networks — to catch the bugs that live at the joints. The trade of realism against speed, and how to keep both kinds of tests in one repo.
keywords: integration testing, integration test go, testing with real database, build tags go test, integration vs unit test, testing the joints, go integration test setup
level: intermediate
status: full
faq:
  - q: What's the difference between a unit test and an integration test?
    a: A unit test checks one piece in isolation, replacing its dependencies with test doubles; an integration test wires two or more real pieces together — code plus a real database, real files, a real subprocess — and checks the joint between them. Unit tests are faster and pinpoint failures; integration tests catch the mismatches doubles can hide, like wrong SQL, wrong file formats, or wrong assumptions about a protocol.
  - q: Why not make every test an integration test, since they're more realistic?
    a: Cost. Integration tests are slower (seconds instead of milliseconds), need more setup, and fail more broadly — a red integration test says "something in this assembly is wrong" where a unit test names the function. The pyramid answer is to prove each piece with many cheap unit tests, then spend a smaller number of integration tests specifically on the joints.
  - q: How do projects keep slow integration tests from slowing everyone down?
    a: By separating them from the default test run. In Go the common tools are build tags or environment-variable skips, so go test runs the fast suite by default and an explicit command opts into the slow one. GopherTrunk does exactly this — make vet test runs constantly, while make integration runs the daemon and replay suites when the paths they cover have changed.
---

# Integration tests

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **integration test** wires **real pieces together** — your code plus an
actual database, real files, a live subprocess — and verifies the **joint**
between them. It exists because unit tests, by design, replace those joints with
[doubles](/learn/testing/test-doubles/) built from your possibly-wrong beliefs.
The price is **speed and precision**: integration tests run in seconds and fail
broadly. So projects run them as a **separate, opt-in suite** — GopherTrunk's
`make integration` — while unit tests stay in the constant loop.
</div>

Unit 2 built the base of the pyramid. This unit climbs it, starting one level
up: the tests that check whether your trustworthy pieces are actually connected
correctly.

## The bugs that live at the joints

Recall the deal you struck with test doubles: the fake radio answered instantly,
the fake store never failed, and both behaved *exactly as you believed* the real
thing would. Every one of those beliefs is a place reality can differ:

- Your SQL is subtly wrong in a way your in-memory fake store never was.
- The real config file has Windows line endings; your fixture string didn't.
- The real subprocess buffers its output; your fake returned it immediately.
- Two components each handle timestamps "correctly" — in different time zones
  (the Mars Climate Orbiter pattern from
  [the cost lesson](/learn/testing/cost-of-a-bug/), at unit scale).

None of these can *possibly* fail a unit test, because the unit test replaced
the very thing that's wrong. Integration tests exist to put reality back in.

## What one looks like

Suppose GopherTrunk-style code logs decoded calls to SQLite. The unit tests
cover the logic against a fake store; one integration test proves the real
plumbing:

```go
func TestCallLog_RoundTrip_SQLite(t *testing.T) {
    db := openSQLite(t, t.TempDir()+"/calls.db") // real database file
    log := NewCallLog(db)

    want := Call{Talkgroup: 4521, Freq: 851_012_500, Duration: 7 * time.Second}
    if err := log.Record(want); err != nil {
        t.Fatalf("Record() error: %v", err)
    }

    got, err := log.ByTalkgroup(4521)
    if err != nil {
        t.Fatalf("ByTalkgroup() error: %v", err)
    }
    if len(got) != 1 || got[0].Freq != want.Freq {
        t.Errorf("ByTalkgroup(4521) = %+v, want one call at %d Hz", got, want.Freq)
    }
}
```

Note the shape: still arrange-act-assert, still `go test` — but the store is a
**real SQLite file** (in `t.TempDir()`, which the test framework wipes
afterward). This test catches wrong SQL, schema drift, and driver quirks that
no fake could. A *round trip* — write through one path, read through another,
compare — is the classic integration pattern, because it verifies both sides of
a joint against each other.

## The cost, and how to pay it

| | Unit test | Integration test |
|--|-----------|------------------|
| Speed | µs–ms | ms–seconds (sometimes minutes) |
| Failure points at | One function | Some joint in the assembly |
| Needs | Nothing external | Real DB / files / processes, setup + teardown |
| Flakiness risk | Minimal | Real — timing, ports, leftover state |
| Count in a healthy repo | Thousands | Dozens–hundreds |

Two consequences follow. First, **write fewer, aim them at joints**: each
integration test should cover a boundary the unit suite structurally cannot,
not re-prove logic the units already proved. Second, **keep them out of the
default loop**. If `go test ./...` takes ten minutes, developers stop running
it, and the feedback loop that makes the whole system work dies. The Go
convention is an opt-in gate — a build tag or an environment check:

```go
//go:build integration
```

```bash
go test ./...                      # fast suite: every edit
go test -tags=integration ./...    # full suite: before merge, in CI
```

GopherTrunk splits exactly this way: `make vet test` is the constant,
must-be-green loop, and **`make integration`** runs the daemon and replay
suites — real decode pipelines fed by recorded radio captures — when the
daemon, DSP, or replay paths change. That replay machinery is interesting
enough to get [its own lesson](/learn/testing/replay-integration-tests/) in
Unit 6.

> Rule of thumb: unit tests prove the pieces; integration tests prove the
> joints; run the first constantly and the second deliberately.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the unit suite replaced the store with a fake built from the same beliefs as the code, so only a test against the real store can catch the mismatch." markdown="0">
  <p class="knowledge-check__q">Quick check: a query works against the in-memory fake store but fails against real SQLite. Why did the unit suite miss it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The unit tests didn't have enough assertions</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The unit tests replaced the database with a double that shared the code's wrong belief — the joint was never exercised</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Code coverage on the package was too low</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Integration tests** wire real components together and verify the
  **joints** — where doubles' beliefs and reality can differ.
- The **round trip** (write via one path, read via another) is the classic
  pattern.
- They cost **speed and precision**, so write fewer and aim each at a boundary
  units can't reach.
- Keep them **opt-in** (build tags) so the fast loop stays fast — GopherTrunk's
  `make vet test` vs `make integration` split.
- Use `t.TempDir()` and thorough teardown so real resources don't leak between
  runs.

Next up: [End-to-end tests](/learn/testing/end-to-end-tests/)
