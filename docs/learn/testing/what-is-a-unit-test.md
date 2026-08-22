---
slug: what-is-a-unit-test
title: What is a unit test?
description: One small piece of code, tested in isolation, fast enough to run thousands of times a day — what makes a test a unit test, and why they form the foundation of the pyramid.
keywords: what is a unit test, unit testing explained, unit test definition, test isolation, fast tests, unit test benefits, unit vs integration
level: beginner
status: full
faq:
  - q: What counts as a "unit"?
    a: Usually a function or a small cluster of functions with one job — the smallest piece of code that does something meaningful on its own. The defining property isn't size but isolation, in that a unit test exercises that piece without real databases, networks, files, or radios involved, so a failure can only mean one thing — that piece is wrong.
  - q: Why do unit tests need to be fast?
    a: Because their whole value is in how often you run them. Tests that finish in seconds get run after every small edit, so a mistake is caught minutes after it's made, while the change is still fresh in your head. Tests that take ten minutes get run once a day, and the feedback loop — and the cost curve from earlier in this module — degrades accordingly.
  - q: Do unit tests replace integration tests?
    a: No — they answer different questions. Unit tests prove each piece is right in isolation; integration tests prove the pieces are wired together correctly. A system can have flawless units and still fail at the joints, which is why the test pyramid keeps both, just in different proportions.
---

# What is a unit test?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **unit test** checks **one small piece of code** — typically a function — **in
isolation**: no real database, network, file system, or radio in the loop. Its
defining virtues are **speed** (milliseconds, so you run thousands constantly)
and **precision** (a failure points at one piece, not "somewhere in the
system"). Unit tests form the **base of the pyramid** because those two virtues
are what let you re-verify a whole codebase after every edit.
</div>

Unit 2 is about the workhorse of all testing. Before writing one in Go (two
lessons from now), it's worth being precise about what makes a test a *unit*
test — because the definition is exactly what makes them so valuable.

## The two defining properties

Lots of things get called unit tests. The real definition has two parts:

1. **Small scope.** It exercises one *unit* — usually one function, sometimes a
   small type with its methods. Not the whole program, not a subsystem: one
   piece with one job.
2. **Isolation.** Nothing outside that piece participates. No real database, no
   network call, no file on disk, no attached radio dongle, no shared global
   state. The test supplies inputs directly and inspects outputs directly.

Isolation is the load-bearing property. If a test touches a real database and
fails, the cause could be the code, the database, the connection, the test
data, or last night's schema change. If an isolated test fails, exactly one
thing can be wrong: **the unit**. That precision is what makes unit-test
failures cheap to act on — the failure *is* the diagnosis.

## Why speed is the superpower

An isolated function-level test runs in microseconds to milliseconds. That
sounds like a nicety; it's actually the whole economic engine:

| Suite speed | How often it really gets run | Feedback delay on a mistake |
|-------------|------------------------------|------------------------------|
| 2 seconds | After every edit, reflexively | Minutes |
| 2 minutes | A few times a day | Hours |
| 20 minutes | Before pushing, maybe | A day, plus lost context |

Recall the [cost curve](/learn/testing/cost-of-a-bug/): a bug's price is set by
how long it survives. A fast suite is a bug's shortest possible lifespan — you
break something at 2:14 pm and know at 2:15, while every relevant detail is
still in your head. GopherTrunk's unit suite covers DSP math, protocol framing,
and trunking logic across hundreds of packages, and runs in the time it takes to
stretch — which is precisely why the project can demand it pass
[before every commit](/learn/testing/make-vet-test/).

## What a unit test looks like

Here's the shape, in Go pseudocode you'll write for real in
[Your first Go test](/learn/testing/your-first-go-test/). GopherTrunk decodes
radio messages, so imagine a function that extracts a talkgroup ID from a
decoded message:

```go
func TestTalkgroupFromMessage(t *testing.T) {
    msg := Message{Bits: knownGoodBits} // input built right here, no radio needed

    got := TalkgroupFromMessage(msg)

    if got != 4521 {
        t.Errorf("TalkgroupFromMessage() = %d, want 4521", got)
    }
}
```

Notice the isolation trick: the test doesn't tune a radio to capture a real
message — it **constructs** one in memory. The unit under test can't tell the
difference, and now the test runs identically in a millisecond, on any machine,
with no antenna. Making code *testable* this way — units that accept their
inputs rather than reaching out to fetch them — is a design skill, and
[Fakes, stubs, and mocks](/learn/testing/test-doubles/) covers what to do when a
unit genuinely depends on something slow or unpredictable.

## What unit tests can and can't prove

Honesty about limits keeps the pyramid in perspective:

- **They prove**: each tested unit behaves as specified on the tested inputs.
  Break a unit during a refactor and its test fails within minutes.
- **They can't prove**: that the units are *wired together* correctly, that the
  spec each unit was tested against matches reality, or that the whole system
  serves the user. Units that are all individually "right" against a wrong
  assumption still fail as a system — the Mars Orbiter pattern, and the seed of
  Unit 6's [self-consistent trap](/learn/testing/the-self-consistent-synthetic-trap/).

> Rule of thumb: unit tests are the base of the pyramid, not the whole pyramid.
> Their job is making the *pieces* trustworthy so bigger tests can focus on the
> joints.

<div class="knowledge-check" data-quiz data-correct-msg="Right — isolation means a failure can only implicate the unit itself, which is what makes unit-test failures so cheap to diagnose." markdown="0">
  <p class="knowledge-check__q">Quick check: why is isolation the defining property of a unit test?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes the test file shorter</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A failure can only mean the unit itself is wrong — the failure is the diagnosis</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It guarantees the units work together correctly</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **unit test** checks one small piece — usually a function — in **isolation**
  from databases, networks, files, and hardware.
- Isolation gives **precision**: an isolated failure implicates exactly one
  piece.
- Speed gives the **feedback loop**: millisecond tests get run constantly, so
  bugs die minutes after birth.
- Tests construct inputs **in memory** — a decoder test builds its message
  rather than capturing one off the air.
- Unit tests make the pieces trustworthy; the **joints** belong to integration
  tests, later in the pyramid.

Next up: [Anatomy of a test](/learn/testing/anatomy-of-a-test/)
