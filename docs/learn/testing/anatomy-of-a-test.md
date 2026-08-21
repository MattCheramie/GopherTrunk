---
slug: anatomy-of-a-test
title: Anatomy of a test
description: Arrange, act, assert — the three-part shape of every good test, plus how to name tests, why each test should check one behavior, and what a great failure message looks like.
keywords: arrange act assert, test structure, test naming, one assertion per test, given when then, test anatomy, writing good tests
level: beginner
status: full
prereq:
  - what-is-a-unit-test
---

# Anatomy of a test

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Nearly every good test has the same three-beat shape: **arrange** the inputs and
world, **act** by calling the code under test once, **assert** that the result
matches the prediction. Each test should verify **one behavior**, its **name**
should state that behavior, and its **failure message** should hand the reader
the got-vs-want facts. Write tests so that when one fails at 2 a.m. six months
from now, the failure alone tells the story.
</div>

Tests are code, and like all code they can be well or badly built. The good news:
unlike most code, good tests follow one near-universal shape. Learn it once and
you can read — and write — tests in any language.

## Arrange, act, assert

Every test answers *given this situation, when this happens, then this should be
true*. In code, those three beats are called **arrange, act, assert** (you'll
also hear the equivalent *given/when/then*):

```go
func TestFrequencyFormat(t *testing.T) {
    // Arrange: build the exact input situation.
    hz := int64(851_012_500)

    // Act: call the code under test — once.
    got := FormatFrequency(hz)

    // Assert: compare against the prediction.
    want := "851.0125 MHz"
    if got != want {
        t.Errorf("FormatFrequency(%d) = %q, want %q", hz, got, want)
    }
}
```

Keeping the beats in order and visually distinct isn't ceremony — it's what
makes a test *skimmable*. A reader (including future-you) should identify the
situation, the action, and the expectation in five seconds. When a test
interleaves setup with assertions, or acts three times with checks in between,
understanding a failure means mentally executing the test — the very cost tests
exist to remove.

## One behavior per test

The tempting anti-pattern is the mega-test: one function that arranges
everything and asserts twenty facts. It seems efficient. It's a trap, for two
reasons:

- **Failures blur.** Go stops reporting a test at its first fatal failure, so a
  mega-test failing on assertion #3 tells you nothing about #4–#20. Twenty small
  tests fail independently and give you the full picture in one run.
- **Names lie.** A test checking twenty things can only be called
  `TestEverything`, which tells a reader nothing when it turns red.

"One behavior" doesn't mean literally one `if` — checking that a returned
struct's three fields are all correct is one behavior. It means one *claim*: a
test should be summarizable in a sentence, and that sentence should be its name.

## Names are documentation

Test names are the one place Go convention encourages long names. The pattern
worth adopting: **`Test<Thing>_<Situation>_<Expectation>`**, flexed to taste.

| Weak name | Strong name |
|-----------|-------------|
| `TestDecode` | `TestDecode_TruncatedMessage_ReturnsError` |
| `TestParse2` | `TestParseFrequency_MHzSuffix_ConvertsToHz` |
| `TestEdgeCase` | `TestChunkAudio_EmptyInput_ReturnsNoChunks` |

Run `go test`, and a failing strong name *is* the bug report: you know what
broke, under which conditions, before reading a line of code. A directory of
strong test names doubles as a specification of the package — many developers
read a package's `_test.go` files before its documentation, because tests can't
drift out of date without failing.

## Failure messages: write for the 2 a.m. reader

The assert beat deserves extra care, because its output is read at the worst
possible time — when something is broken. The Go convention is the
**got/want** message, and its three ingredients are exactly what a diagnosis
needs:

```go
t.Errorf("FormatFrequency(%d) = %q, want %q", hz, got, want)
```

That prints *what was called with which input* (`FormatFrequency(851012500)`),
*what came back* (`"851.01 MHz"`), and *what was expected* (`"851.0125 MHz"`).
Compare `t.Errorf("wrong output")` — true, and useless. The difference between
those two messages is the difference between a thirty-second fix and a
debugging session.

> Rule of thumb: a failure message should let someone diagnose the problem
> **without opening the test file**. Input, got, want — all three, every time.

One more Go-specific choice: `t.Errorf` records a failure and keeps going;
`t.Fatalf` records it and stops the test immediately. Use `Fatalf` when
continuing makes no sense (the arrange step itself failed, or the thing you're
about to dereference is nil); use `Errorf` otherwise, so one run reports every
broken expectation.

<div class="knowledge-check" data-quiz data-correct-msg="Right — input, got, and want turn a red test into a diagnosis; a bare 'failed' turns it into a debugging session." markdown="0">
  <p class="knowledge-check__q">Quick check: which failure message best serves the person who reads it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>t.Errorf("test failed")</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>t.Errorf("bad value: %v", got)</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="correct"><code>t.Errorf("Decode(%q) = %d, want %d", input, got, want)</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Tests follow **arrange → act → assert**: set up the situation, call the code
  once, check the prediction.
- **One behavior per test** — failures stay sharp, and every test is
  summarizable in a sentence.
- That sentence is the **name**: `TestThing_Situation_Expectation` turns a red
  test into a bug report.
- Failure messages carry **input, got, and want** — diagnosable without opening
  the file.
- `t.Errorf` continues, `t.Fatalf` stops — use fatal only when continuing is
  meaningless.

Next up: [Your first Go test](/learn/testing/your-first-go-test/)
