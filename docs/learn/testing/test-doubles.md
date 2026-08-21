---
slug: test-doubles
title: Fakes, stubs, and mocks
description: Test doubles stand in for slow or unpredictable dependencies — real radios, networks, clocks — so units can be tested in isolation. Go's interfaces make swapping them in natural.
keywords: test doubles, fakes stubs mocks difference, go interface testing, mocking in go, fake dependency, stub vs mock, dependency injection go
level: intermediate
status: full
prereq:
  - what-is-a-unit-test
  - table-driven-tests
---

# Fakes, stubs, and mocks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a unit depends on something slow, unpredictable, or unavailable — a radio,
a network, a clock — you test it against a **test double**: a stand-in that
implements the same **interface**. A **stub** returns canned answers; a
**fake** is a small working implementation (a map posing as a database); a
**mock** additionally records and verifies *how it was called*. Go makes
substitution natural: define the dependency as an interface, and anything with
the right methods can play the part.
</div>

Unit tests demand isolation, but real code has dependencies — GopherTrunk's
scanner logic ultimately talks to actual radio hardware. You can't ship an SDR
dongle with the test suite. This lesson is the standard solution, and it happens
to teach one of Go's central design ideas along the way.

## The problem, concretely

Say the scanner has logic deciding whether to retune when a control channel goes
quiet. Its natural form reaches straight for hardware:

```go
func (s *Scanner) CheckChannel() error {
    strength := s.device.ReadSignalStrength() // real SDR hardware!
    if strength < s.threshold {
        return s.device.Retune(s.nextFreq)
    }
    return nil
}
```

Testing this as written needs a physical radio, an antenna, and a transmitter
that conveniently goes quiet on cue. The *decision logic* — the part that has
bugs — is trivial to test; the dependency is the whole problem.

## Interfaces: the substitution point

The fix is to make the unit depend on a **capability**, not a device. In Go
that's an interface:

```go
type SignalSource interface {
    ReadSignalStrength() float64
    Retune(freqHz int64) error
}
```

The production scanner is built with the real SDR driver, which satisfies
`SignalSource`. The test builds it with a double:

```go
type fakeSource struct {
    strength float64
    retunedTo []int64
}

func (f *fakeSource) ReadSignalStrength() float64 { return f.strength }
func (f *fakeSource) Retune(hz int64) error {
    f.retunedTo = append(f.retunedTo, hz)
    return nil
}

func TestCheckChannel_WeakSignalRetunes(t *testing.T) {
    src := &fakeSource{strength: -120.0} // arrange: a silent channel
    s := NewScanner(src, WithThreshold(-100.0))

    if err := s.CheckChannel(); err != nil { // act
        t.Fatalf("CheckChannel() error: %v", err)
    }

    if len(src.retunedTo) != 1 { // assert
        t.Errorf("retuned %d times, want 1", len(src.retunedTo))
    }
}
```

Fifteen lines replace a hardware lab. Because Go interfaces are satisfied
*implicitly* — any type with the right methods qualifies, no `implements`
declaration — the production code never knows or cares that a double exists.
This pattern of handing a unit its dependencies instead of letting it construct
them is **dependency injection**, and it's covered from the design side in the
[Go interfaces lesson](/learn/programming-go/interfaces/).

## Stub, fake, mock — the vocabulary

People use these words loosely, but the distinctions are useful:

| Double | What it does | Use when |
|--------|--------------|----------|
| **Stub** | Returns canned answers; remembers nothing | The unit just needs *some* input ("signal is weak") |
| **Fake** | A real, working, lightweight implementation — a map posing as a store, an in-memory queue | The unit genuinely interacts with the dependency |
| **Mock** | Records calls and lets the test assert on them ("Retune was called once, with 851 MHz") | The *interaction itself* is the behavior under test |

The `fakeSource` above blurs stub and mock — canned strength, recorded retunes —
which is normal: in Go these are usually a dozen hand-written lines, not
framework artifacts, and you build exactly the double the test needs.

## The dial worth watching: how much to verify

Stub/fake testing asserts on **outcomes** ("the scanner ended up on the new
frequency"). Mock-style testing asserts on **interactions** ("Retune was called
exactly once, before ReadSignalStrength"). Prefer outcomes when you can:
interaction assertions couple the test to the unit's *current internal
choreography*, so a harmless refactor — same result, different call sequence —
turns tests red. A test suite that breaks on every refactor teaches people to
ignore it.

> Rule of thumb: assert on what the code *achieved*, not how it went about it —
> unless "how" is the actual contract (retuning twice would double-toggle real
> hardware, say).

One caution from the other direction: a double is only as truthful as your
model of the real thing. If the fake answers instantly while the real radio
takes 50 ms, timing bugs live on untested. The joints where doubles stood in
are exactly what [integration tests](/learn/testing/integration-tests/) exist
to cover — and a double whose wrong assumption *matches* the code's wrong
assumption is the seed of Unit 6's
[self-consistent trap](/learn/testing/the-self-consistent-synthetic-trap/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — implicit interface satisfaction means any type with the right methods can stand in, with no changes to production code." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes test doubles especially natural in Go?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Interfaces are satisfied implicitly, so any type with the right methods can stand in for the real dependency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Go ships an official mocking framework in the standard library</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Go tests are allowed to access private fields of other packages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Test doubles** stand in for slow, unpredictable, or unavailable
  dependencies so units stay isolated and fast.
- **Stub** = canned answers; **fake** = working lightweight implementation;
  **mock** = records and verifies interactions.
- The substitution point is an **interface**; Go's implicit satisfaction makes
  doubles a dozen hand-written lines.
- Prefer asserting **outcomes** over interactions — refactor-proof tests stay
  trusted.
- Doubles encode your *beliefs* about the dependency; integration tests check
  those beliefs against the real thing.

Next up: [Code coverage](/learn/testing/code-coverage/)
