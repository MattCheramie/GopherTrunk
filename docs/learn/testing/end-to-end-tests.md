---
slug: end-to-end-tests
title: End-to-end tests
description: Drive the whole assembled system the way a user would — the most convincing tests and the most expensive ones. What E2E tests are for, why they're kept few, and how to keep them from rotting.
keywords: end to end testing, e2e tests, system testing, smoke test, e2e vs integration, testing whole system, browser testing basics
level: intermediate
status: full
prereq:
  - integration-tests
---

# End-to-end tests

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **end-to-end (E2E) test** exercises the **whole assembled system** through
its real entry points — launch the binary, hit the real API, click the real UI —
and checks what a **user** would observe. It's the only test that proves *the
product works*, which makes it the most convincing layer; it's also the
slowest, most brittle, and vaguest about causes, which is why the pyramid keeps
it to a **handful of critical journeys** plus a fast **smoke test**, not
hundreds of cases.
</div>

Top of the pyramid. Unit tests proved the pieces, integration tests proved the
joints — but a user doesn't run pieces or joints. They run the *program*. One
layer of testing has to do the same.

## What "end to end" means

Everything real, nothing replaced. For a web app that means a scripted browser
against a genuinely running server; for a program like GopherTrunk it means: build the
actual binary, launch it with a real config, feed it real input, and assert on
what comes out the far end — the log lines, the API responses, the recording
files on disk. The test enters through the front door (CLI flags, HTTP
endpoints, the UI) and observes through the front door, exactly as a user
would. Internal functions are off-limits; if the behavior isn't visible from
outside, an E2E test can't and shouldn't check it.

```go
func TestDaemon_StartsAndServesStatus(t *testing.T) {
    bin := buildBinary(t)                                   // the real executable
    cmd := exec.Command(bin, "-config", "testdata/replay.yaml")
    startAndWait(t, cmd)                                     // real process, real config
    defer cmd.Process.Kill()

    resp := httpGet(t, "http://127.0.0.1:8080/api/v1/status") // the real API surface
    if resp.StatusCode != 200 {
        t.Fatalf("GET /status = %d, want 200", resp.StatusCode)
    }
}
```

Every layer participates: flag parsing, config loading, startup ordering, the
HTTP server, shutdown. A bug in *any* of them fails this test — which is both
its power and its problem.

## Why E2E tests are kept few

That "any layer can fail it" cuts both ways:

- **Slow.** Building, launching, and waiting takes seconds to minutes per test.
  A thousand of these is an overnight run, not a feedback loop.
- **Vague on failure.** A red unit test names a function. A red E2E test says
  "the product is broken somewhere" — and the diagnosis begins. You'll use
  Unit 5's [debugging skills](/learn/testing/reproducing-a-bug/) on E2E
  failures more than any other kind.
- **Brittle.** Real processes mean real ports, real timing, real filesystems —
  the natural habitat of [flaky tests](/learn/testing/flaky-tests/). A test
  that waits "2 seconds, which is surely enough" for startup is a coin flip on
  a loaded CI machine.
- **Expensive to maintain.** Change the CLI flags or the API shape and every
  E2E script that touched them needs updating.

So the guidance is ruthless selectivity: cover the **critical user journeys** —
the five or ten flows that, if broken, mean the product is broken. For a
trunking scanner: *starts with a valid config; locks a control channel; follows
a call and writes a tagged recording; serves the web UI; shuts down cleanly*.
Everything subtler belongs lower on the pyramid, where tests are cheap and
failures are sharp.

## The smoke test: E2E's minimum viable form

The highest-value-per-second E2E test is the **smoke test**: does the program
*start at all* and answer one trivial request? It sounds too small to matter.
It catches an outsized share of real-world breakage — a missing file in the
release, a bad default, a migration that fails on boot, a flag typo — because
it's the only test that runs the true startup path. Plumbing metaphor: before
checking the fixtures, blow smoke through the pipes and see if any escapes.

> Rule of thumb: one honest smoke test of the shipped artifact is worth more
> than a hundred additional unit tests, because it's the only thing verifying
> what users actually receive.

## Keeping the top of the pyramid healthy

Three habits keep a small E2E suite trustworthy:

1. **Wait on conditions, not clocks.** Poll "is the port answering?" with a
   deadline instead of sleeping a fixed 2 seconds — the number-one flakiness
   cure.
2. **Make failures forensic.** On failure, dump the process's logs and exit
   status into the test output. A vague layer needs rich evidence.
3. **Run them on every merge, not every save.** Like
   [integration tests](/learn/testing/integration-tests/) they live behind a
   gate — in CI on pull requests, and locally when you've touched what they
   cover.

<div class="knowledge-check" data-quiz data-correct-msg="Right — E2E failures are convincing but vague, and E2E runs are slow, so the pyramid reserves them for the critical journeys." markdown="0">
  <p class="knowledge-check__q">Quick check: why keep only a handful of end-to-end tests when they're the most realistic kind?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because E2E tests can't detect real bugs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because they require special hardware to run</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because they're slow, brittle, and vague about causes — realism is bought at the price of feedback quality</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **E2E tests** run the whole real system through its real entry points and
  assert what a **user** would observe.
- They're the only tests that prove **the product** works — and the slowest,
  brittlest, vaguest layer.
- Reserve them for **critical journeys**; push everything subtler down the
  pyramid.
- The **smoke test** — does it start and answer? — is the cheapest E2E test and
  catches an outsized share of release breakage.
- Health habits: **wait on conditions**, dump forensics on failure, run on
  merges not saves.

Next up: [Regression tests & failing first](/learn/testing/regression-tests/)
