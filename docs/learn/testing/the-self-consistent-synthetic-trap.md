---
slug: the-self-consistent-synthetic-trap
title: The self-consistent synthetic trap
description: A test whose encoder and decoder share the same wrong assumption passes while the real world fails — GopherTrunk's most instructive bug pattern, and the independent references that break it.
keywords: self consistent test trap, round trip test blind spot, synthetic test data, independent reference testing, encoder decoder same assumption, test oracle problem, conformance testing
level: advanced
status: full
prereq:
  - property-based-testing
  - replay-integration-tests
---

# The self-consistent synthetic trap

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The most dangerous green test is the **self-consistent** one: its encode side
and decode side **share the same wrong assumption**, so they agree with each
other perfectly while both disagree with reality. Round-trips pass, sweeps
pass, everything passes — and every real input fails. The trap is invisible
from inside because **both sides move together**; the cure is an
**independent reference**: real captured data, a reference implementation
built by someone else, published test vectors — anything your own
misunderstanding **cannot have contaminated**. GopherTrunk treats this as its
most instructive recurring bug pattern, because it has paid for the lesson
more than once.
</div>

This is the intellectual center of Unit 6 — a failure mode subtle enough that
it defeats everything taught so far *while leaving all of it green*. It's the
reason the previous lesson insisted on captured-from-air fixtures and the
reason the next lesson's verification rule exists.

## The shape of the trap

You're implementing a decoder for a specified format — a radio protocol, a
file format, a wire encoding. To test it, you naturally also write the
encoder, and then the obvious
[round-trip property](/learn/testing/property-based-testing/):

```go
func TestFrameRoundTrip(t *testing.T) {
    for _, f := range manyGeneratedFrames() {
        got, err := Decode(Encode(f))
        if err != nil {
            t.Fatalf("Decode(Encode(%v)): %v", f, err)
        }
        if got != f {
            t.Errorf("round trip: got %v, want %v", got, f)
        }
    }
}
```

Ten thousand frames, all green. Now the trap: **both functions were written by
the same mind reading the same spec** — every misreading you made, you made
*twice, symmetrically*. Misread a bit order? Encode writes it wrong; decode
reads it wrong; the errors cancel; green. Wrong scrambling constant? Both
sides use it; green. The test doesn't check your code against the *format* —
it checks your code against *itself*, and code always agrees with itself.
Then the first real transmitter — which obeys the actual spec — sends a frame,
and decode fails on all of it.

What makes this the *most* dangerous pattern in the module: every signal you
trust is lying in the same direction. Coverage is high. The suite is green.
The [failing-first regression test](/learn/testing/regression-tests/) you
write... uses your encoder, and passes. Even your careful
[debugging](/learn/testing/root-cause-analysis/) inspects the decode against
your own encode and finds them consistent. The system of checks has a closed
loop in it, and inside a closed loop, wrong is indistinguishable from right.

## Paid-for examples

GopherTrunk's engineering notes treat this pattern as a named recurring
enemy, because real time was lost to it in materially different dresses:

- **The descramble skip.** A voice decoder skipped a descrambling step in one
  specific configuration — a shortcut inherited from a context where it was
  safe. Synthetic round-trip tests scrambled *and* descrambled consistently,
  so they passed either way; the asymmetry only existed on real air, where
  transmitters always scramble. Every synthetic test green; every real
  transmission undecodable. The regression test that finally caught it had to
  make the encode side behave like *reality* (always scramble) instead of
  like the decoder's assumption.
- **The wrong checksum flavor.** A channel decoder implemented a CRC as a
  standard shift-register scheme where the spec actually meant a fixed
  parity-check table. Self-consistent round-trips — same wrong CRC on both
  sides — passed for months while every on-air burst was silently rejected as
  corrupt.
- **The fake server that agreed too much.** Not radio at all: a driver spoke
  a binary RPC protocol to a hardware server, and its unit tests ran against
  a hand-written fake server. The driver used a wrong message opcode — and
  the fake, written from the same author's same reading, *checked for the
  same wrong opcode*. Both sides moved together; tests green; real hardware
  silently did the wrong thing. The fix pinned the constants against the
  upstream project's literals — an independent reference — because that's the
  only thing opcode drift *can't* fool.

Different subsystems, one shape: **the test's two sides shared authorship,
therefore shared assumptions, therefore could not disagree.**

## Breaking the loop: independence

The escape is never "test harder" — more self-consistent tests are more
green. It's importing something your misunderstanding **cannot have
touched**:

| Independent reference | What it anchors | Cost |
|-----------------------|-----------------|------|
| **Real captured data** | The actual behavior of real transmitters — the ultimate authority | Must be obtained; conditions can't be ordered up |
| **A reference implementation** (the standard body's codec, a mature independent decoder) | Same inputs → outputs must match bit-for-bit | Integration work; build quirks |
| **Published test vectors** (spec appendices: input X ⇒ output Y) | Exact conformance points chosen by the format's authors | Cheap when they exist |
| **A second independent implementation** (different author, different reading) | Two minds rarely misread identically | Expensive; still not proof |

GopherTrunk's voice codec work shows the full discipline: the vocoder was
validated by feeding **the standard body's own reference codec** and the
clean-room implementation identical bitstreams and requiring **bit-identical
output** — then validated *again* end-to-end against real captures. Note the
layering: the reference implementation proves conformance to the spec; the
capture proves the spec was the right thing to conform to, all the way down
the pipeline.

> Rule of thumb: every encode/decode pair needs at least one test whose
> expected values **you did not produce** — a real capture, a reference
> output, a published vector. Round-trips verify consistency; only
> independence verifies correctness.

The general form goes beyond radio. A serializer tested only against its own
deserializer; a client tested against a mock server written from the same API
misreading ([test doubles'](/learn/testing/test-doubles/) warning, fully
grown); a physics engine tested against its author's re-derivation of the
same wrong equation. Wherever both sides of an agreement share an author,
ask the question this lesson exists to install: ***what, in this test, did I
not write?***

<div class="knowledge-check" data-quiz data-correct-msg="Right — shared authorship means shared assumptions: both sides encode the same misreading, the errors cancel, and the tests check your code against itself instead of the real format." markdown="0">
  <p class="knowledge-check__q">Quick check: why can a decoder pass ten thousand round-trip tests and still fail on every real transmission?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Real transmissions carry noise that no software test can represent</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The test's encoder shares the decoder's misreading — the errors cancel, so the suite verifies self-consistency, not the real format</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Round-trip tests only exercise the happy path of the decoder</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Self-consistent tests** pit your code against your own assumptions —
  encode and decode share a misreading, errors cancel, everything passes,
  reality fails.
- The trap is invisible **from inside**: coverage, round-trips, even
  regression tests built on your own encoder all stay green.
- GopherTrunk paid for this lesson repeatedly — descrambling skips, wrong
  CRC schemes, a fake server that checked the same wrong opcode.
- The cure is an **independent reference**: real captures, reference
  implementations, published vectors — expected values **you didn't produce**.
- The standing question for any test of an agreement: *what part of this did
  I not write?*

Next up: [Capture-gated verification](/learn/testing/capture-gated-verification/)
