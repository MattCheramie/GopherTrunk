---
slug: testing
title: Testing — unit, integration & beyond
description: The test pyramid explained — unit, integration and end-to-end tests, plus TDD, regression tests, mocking, code coverage and its limits, and golden-file testing for signal code.
keywords: software testing, test pyramid, unit tests, integration tests, end-to-end tests, test-driven development, regression tests, mocking, code coverage, golden files
level: intermediate
status: full
faq:
  - q: "What's the difference between unit, integration, and end-to-end tests?"
    a: "A unit test checks one small piece (a function or class) in isolation. An integration test checks that several pieces work together correctly — say, a parser feeding a decoder. An end-to-end test exercises the whole system the way a user would, from input to output. Unit tests are fast and numerous; end-to-end tests are slow and few, which is why the test pyramid favors many unit tests at the base."
  - q: "Does 100% code coverage mean my code is bug-free?"
    a: "No. Coverage only measures which lines ran during tests, not whether your assertions actually checked the right things. You can execute every line and still assert nothing meaningful, miss edge cases, or have wrong expected values. Coverage is a useful signal for finding untested code, but it's a floor, not a guarantee of correctness."
  - q: "What is a golden file and when would I use one?"
    a: "A golden file is a known-good reference output saved alongside your tests. The test runs your code against a fixed input and compares the result byte-for-byte (or value-for-value) against the golden file. It's ideal when output is large or complex and hard to eyeball — like the decoded result of a captured radio signal — because it pins down exactly what \"correct\" means and flags any regression instantly."
---
# Testing — unit, integration & beyond

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The test pyramid** — many fast unit tests, fewer integration tests, a handful of end-to-end. **Tests pin down "correct"** — golden files and regression tests catch breakage you'd never spot by eye. **Coverage is a floor, not a guarantee** — running a line isn't the same as checking it.
</div>

Testing is how you turn "I think it works" into "I know it works, and I'll know the moment it stops." It's not a phase you tack on at the end — good teams write tests alongside the code, and a [CI pipeline](/learn/intro-software-dev/build-and-ci-cd/) runs them automatically on every change. This lesson covers the kinds of tests and how they fit together, a few essential techniques, and the limits of the metric everyone loves to quote: code coverage. The motivating example throughout is signal-processing code, where bugs are nearly impossible to catch by reading.

## The test pyramid

Not all tests are equal, and a healthy test suite balances three layers — usually drawn as a pyramid, wide at the bottom and narrow at the top.

- **Unit tests** (the base) check a single small piece — one function or class — in isolation. They're fast, focused, and numerous; a good suite has thousands. When one fails, it points straight at the broken unit.
- **Integration tests** (the middle) check that several pieces work *together* — a parser handing frames to a decoder, code talking to a database. Fewer of these, and slower, because they exercise real seams between components.
- **End-to-end (E2E) tests** (the tip) drive the whole system the way a user would, from input all the way to final output. They give the most confidence that the product actually works, but they're slow, brittle, and expensive to maintain — so you keep them few.

```
        /\        E2E         (few, slow, high confidence)
       /  \
      /----\      Integration (some, medium)
     /      \
    /--------\    Unit        (many, fast, focused)
```

The pyramid shape is the guidance: lean on lots of fast unit tests at the base, fewer integration tests in the middle, and only a thin layer of E2E at the top. An "inverted pyramid" — mostly slow E2E tests — is a classic anti-pattern: the suite becomes slow and flaky, and a single failure tells you little about *where* the problem is.

## Test-driven development, briefly

**Test-driven development (TDD)** flips the usual order: you write a failing test *first*, then write just enough code to make it pass, then refactor. The rhythm is "red, green, refactor." Writing the test first forces you to clarify what the code should do before you build it, and it guarantees the test actually fails when the behavior is missing (a test that passes before you write the code is testing nothing). TDD isn't mandatory and isn't always the right fit, but the discipline of specifying behavior up front is valuable even when you don't follow it strictly.

## Regression tests and why they matter

A **regression** is when something that used to work breaks. A **regression test** is one you add specifically to lock in a fixed bug so it can never silently return. The workflow is simple and powerful: a bug is reported, you write a test that reproduces it (and fails), you fix the code until the test passes, and now that test guards the behavior forever. Over time your regression tests accumulate into a record of every mistake you've made — and a wall preventing you from making them twice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the pyramid favors many fast, focused unit tests at the base and only a few slow end-to-end tests at the top." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the test pyramid recommend?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Mostly slow end-to-end tests, with very few unit tests</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Many fast unit tests at the base, fewer integration, a handful of end-to-end</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An equal number of every kind of test, regardless of speed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Mocking, fakes, and isolation

To test a unit *in isolation*, you often need to stand in for its dependencies — you don't want a unit test hitting a real network, a real database, or a real radio. **Test doubles** fill that gap:

- A **fake** is a lightweight working implementation — for example, a sample source that replays a file instead of reading a live SDR.
- A **mock** records how it was called and lets you assert that your code interacted with it correctly (it called `send()` once, with these arguments).
- A **stub** simply returns canned responses.

This is exactly where the [dependency inversion](/learn/intro-software-dev/solid/) idea pays off: if your code depends on an interface rather than a concrete radio, a test can pass in a fake source and run deterministically, with no hardware in the loop. Good seams make code testable; testable code tends to be well-designed code.

## The motivating case: testing a signal decoder

Here's where testing stops being abstract. Signal-processing and DSP code is **notoriously hard to debug by eye.** A demodulator emits streams of numbers; a decoder turns noisy samples into bits and then into messages. If a sign is flipped, a filter coefficient is off, or the timing recovery drifts, the output is *subtly* wrong — not a crash, just garbage that looks plausible. You cannot reliably stare at a buffer of floats and know whether it's right.

The answer is **golden-file testing.** You capture a real signal once — a recording of an actual over-the-air transmission whose correct decoded content you've verified — and save both the raw capture and the expected decoded output (the "golden" reference) in your test suite. The test then runs the decoder against the captured samples and compares its output against the golden file:

```go
func TestDecodeP25(t *testing.T) {
    samples := loadCapture("testdata/p25_voice.iq")   // known input
    got := Decode(samples)
    want := loadGolden("testdata/p25_voice.expected") // verified correct output
    if !bytes.Equal(got, want) {
        t.Fatalf("decoded output drifted from golden reference")
    }
}
```

This is how you *trust* a decoder. The moment a refactor, a dependency bump, or a "harmless" tweak changes the output by a single byte, the test goes red and tells you instantly. For a project like GopherTrunk, a library of golden sample captures — one per protocol, per edge case — is the difference between confident change and praying nothing broke. It's the testing technique that makes signal code maintainable at all. The [RF & SDR path](/learn/rf-sdr/) goes deeper into the signal side.

## Code coverage and its limits

**Code coverage** measures the percentage of your code that runs during the test suite — which lines, branches, or functions were exercised. It's genuinely useful for *finding untested code*: a module sitting at 0% coverage is a red flag, and watching coverage tells you where to aim next.

But coverage is widely misread. It measures whether a line *ran*, not whether your test *checked* anything meaningful about it. You can hit 100% coverage with tests that assert nothing, miss every edge case, or compare against wrong expected values. High coverage with weak assertions is false confidence. Treat coverage as a **floor** — a signal for gaps — never as a target to chase or a proof of correctness. A smaller suite of sharp, well-asserted tests beats a sprawling one written to game a number.

## Tests run automatically

The final piece: tests only protect you if they run, every time. That's the job of **continuous integration** — on every push, the CI server builds the code and runs the whole suite, so a regression is caught within minutes of being introduced rather than weeks later in production. Tests plus CI is the combination that lets a team move fast *without* breaking things. We cover that machinery next, and how tests interact with graceful failure is the subject of [robustness & error handling](/learn/intro-software-dev/robustness-and-errors/).

## Recap

- **The test pyramid** — many fast unit tests at the base, fewer integration tests, a thin layer of slow end-to-end tests on top.
- **TDD** — write the failing test first, then the code; it clarifies intent and guarantees the test really tests something.
- **Regression tests** — lock in every fixed bug so it can never silently come back.
- **Mocks and fakes** — stand in for real dependencies so units can be tested in isolation, deterministically.
- **Golden files** — pin down "correct" for output you can't eyeball; the key to trusting signal/DSP decoders against known captures.
- **Coverage is a floor** — it shows what ran, not what was checked; never mistake a high number for correctness.

Next up: how code gets compiled, tested, and shipped automatically — [build systems, CI/CD & automation](/learn/intro-software-dev/build-and-ci-cd/).
