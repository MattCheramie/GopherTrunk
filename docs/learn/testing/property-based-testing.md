---
slug: property-based-testing
title: Property-based testing
description: Instead of hand-picking examples, state a property that must always hold — round-trips, invariants, oracles — and let the computer generate hundreds of inputs hunting for a counterexample.
keywords: property based testing, property testing go, round trip property, invariants testing, quick check style testing, generative testing, shrinking counterexample
level: intermediate
status: full
prereq:
  - table-driven-tests
---

# Property-based testing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Example-based tests check hand-picked inputs; **property-based tests** state a
claim that must hold for **all** inputs — a **property** — and let the computer
generate hundreds of random cases hunting for a **counterexample**. The classic
properties: **round-trips** (decode(encode(x)) == x), **invariants** (output
sorted, length preserved), and **oracles** (fast code agrees with slow obvious
code). The computer's gift is inputs **you'd never think to try**; your job
shifts from picking cases to articulating truths.
</div>

Every test so far checked inputs a human chose — and humans choose inputs they
can imagine failing. This lesson flips the roles: you state what must *always*
be true, and the machine does the imagining.

## From examples to properties

An example-based test says `ParseFrequency("851.0125MHz") == 851012500`. A
property-based test says something bolder: *for any valid frequency `f`,
formatting it and parsing the result yields `f` again*:

```go
func TestFrequencyRoundTrip(t *testing.T) {
    rng := rand.New(rand.NewSource(1)) // seeded: failures reproduce
    for i := 0; i < 1000; i++ {
        f := rng.Int63n(3_000_000_000) // any frequency up to 3 GHz

        got, err := ParseFrequency(FormatFrequency(f))
        if err != nil {
            t.Fatalf("round trip of %d Hz: parse error: %v", f, err)
        }
        if got != f {
            t.Fatalf("round trip of %d Hz = %d — lost %d Hz", f, got, f-got)
        }
    }
}
```

A thousand frequencies per run, none chosen by a human. When this fails, it
fails on something like `f = 2_147_483_647` or `f = 999_999_999` — a boundary
you didn't hand-pick because you didn't know it *was* a boundary. That's the
core value: the generator has no [confirmation bias](/learn/testing/the-testing-mindset/).
(Plain `math/rand` in a loop is a perfectly good starter kit; dedicated
property-testing libraries add smarter generation and automatic **shrinking** —
whittling a failing input down to its minimal form.)

## The three property families

The hard part of property testing is not the loop — it's finding claims that
are always true. Three families cover most real uses:

| Family | Shape | Radio-flavored example |
|--------|-------|------------------------|
| **Round-trip** | `decode(encode(x)) == x` | Any message frame, packed to bits then unpacked, is unchanged |
| **Invariant** | some fact holds of every output | A channel filter never outputs more samples than it received; a sorted scan list stays sorted after insert |
| **Oracle** | fast implementation agrees with a slow obvious one | The optimized ring-buffer average equals the naive sum-and-divide |

Round-trips are the workhorse for anything with an encode/decode pair — codecs,
serializers, config formats. Invariants suit algorithms with a guarantee you
can state ("output length equals input length", "result is within the input's
min and max"). Oracles are gold when you're optimizing: keep the slow version
as the referee and let randomness search for the input where the clever version
disagrees.

## A warning hidden in the round-trip

Round-trip properties have a blind spot worth naming now, because Unit 6 turns
it into a full lesson. `decode(encode(x)) == x` proves the pair is
*self-consistent* — it does **not** prove either side matches the real-world
format. If your encoder and decoder share the same misreading of the spec, they
round-trip flawlessly while both are wrong; every real message from a real
transmitter fails. GopherTrunk has been genuinely bitten by exactly this shape
— a decoder whose synthetic round-trip tests all passed while real off-air
bursts failed — and the antidote is anchoring at least one test to something
*independent*: a [captured fixture](/learn/testing/golden-files-and-fixtures/),
a reference implementation, a published test vector. The full story is
[the self-consistent synthetic trap](/learn/testing/the-self-consistent-synthetic-trap/).

> Rule of thumb: a round-trip property proves your two functions agree with
> *each other*. Pair it with one real-world sample that proves they agree with
> *reality*.

## Making random failures reproducible

Randomized testing has one operational rule: **a failure you can't re-run is
almost worthless**. Two habits keep property tests debuggable:

1. **Seed the generator** (as above) or log the seed on failure — the failing
   run can then be replayed exactly, which is lesson-one material for Unit 5's
   [reproduction discipline](/learn/testing/reproducing-a-bug/).
2. **Print the failing input in full** in the failure message. The input *is*
   the bug report; once found, it graduates into the example table as a
   permanent [regression row](/learn/testing/regression-tests/) — randomness
   found it once, but you don't want to rely on randomness finding it again.

That last move — random search finds the case, the case becomes a fixed test —
is the standard partnership between the generative and example-based worlds.

<div class="knowledge-check" data-quiz data-correct-msg="Right — self-agreement is all a round-trip can check; only an independent reference (like a real captured message) ties the pair to the actual format." markdown="0">
  <p class="knowledge-check__q">Quick check: a codec's round-trip property passes on 10,000 random messages. What has NOT been shown?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That decode inverts encode on those messages</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">That either function matches the real wire format — both could share the same wrong assumption</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That the code runs without panicking on those messages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Property-based tests** state claims that hold for **all** inputs and let
  generated randomness hunt counterexamples — inputs free of your bias.
- The three families: **round-trip**, **invariant**, **oracle** — most real
  properties are one of these.
- A round-trip proves **self-consistency only**; anchor to an independent
  reference so both sides can't share one wrong assumption.
- **Seed the randomness** and print failing inputs — then promote each found
  case into a permanent regression row.
- Your job shifts from *picking cases* to *articulating truths* — often the
  most clarifying design exercise in this module.

Next up: [Fuzzing](/learn/testing/fuzzing/)
