---
slug: testing-and-debugging-dsp
title: Testing & debugging DSP
description: Golden vectors, offline replay of captures, and the invariants that catch a broken signal path — how you prove a decoder actually works.
keywords: testing dsp, debugging dsp, golden vectors, reference vectors, replay capture, cfile replay, dsp invariants, regression test, rate invariance
level: advanced
status: full
prereq:
  - dsp-in-gophertrunk
faq:
  - q: What is a golden vector in DSP testing?
    a: "A golden vector is a saved reference: a known input and the exact output a correct implementation must produce for it. A regression test feeds the input through the code and compares against the stored output. Because DSP is deterministic arithmetic, any drift from the golden output signals a change in behaviour — it turns a subtle numeric bug into a hard test failure you can catch automatically."
  - q: Why replay a recorded capture instead of testing with live radio?
    a: "A live signal is never the same twice, so a bug that appears on the air is nearly impossible to reproduce or to prove fixed. A recorded capture — a .cfile of raw I/Q — is a fixed, repeatable stimulus. Replaying it runs the exact same samples through the pipeline every time, so you can reproduce a failure deterministically, bisect it, and confirm a fix by re-running the same file."
---

# Testing & debugging DSP

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
DSP is deterministic arithmetic, which makes it **testable**. **Golden vectors** pin
known input→output pairs so any behaviour drift fails a test. **Offline replay** of a
recorded **`.cfile`** capture turns an un-reproducible on-air bug into a repeatable one.
And **invariants** — properties that must always hold, like GopherTrunk's
**rate-invariance** — catch whole classes of failure. Together they let you *prove* a
decoder works instead of hoping.
</div>

The final skill in this module isn't an algorithm — it's how you know any of them are
correct. This lesson draws the [whole pipeline](/learn/dsp/dsp-in-gophertrunk/) together
under the question: how do you *verify* a signal path? It reflects the practices
GopherTrunk's own repository enforces.

## Golden vectors: deterministic reference outputs

Because a DSP block is pure arithmetic, the same input always yields the same output. That
determinism is a gift for testing. A **golden vector** captures a known input and the
correct output beside it; a regression test runs the input through the code and asserts
the result matches.

```text
input vector   ->  [ DSP under test ]  ->  output
                                              ||  compare
golden output  ------------------------------ '
mismatch => a behaviour change => test fails
```

This is exactly the project's rule for a bug fix: a regression test that **fails without
the fix and passes with it**. If you can't write a vector that fails first, you haven't
truly reproduced the bug — a discipline that stops "fixes" for problems that were never
understood.

## Offline replay: making the un-reproducible repeatable

The hardest bugs appear on live signals that never recur. The cure is to **record** the
raw I/Q — a **`.cfile`** of [complex samples](/learn/dsp/complex-signals-and-iq/) — once,
then **replay** that same file through the pipeline as often as you like. GopherTrunk has a
dedicated `replay` path for exactly this: the same samples, the same result, every run. A
live mystery becomes a deterministic test you can bisect and, crucially, re-run to *confirm*
a fix rather than guess at one.

## Invariants: properties that must always hold

Beyond specific vectors, the strongest tests assert **invariants** — statements true for
*every* valid input. GopherTrunk's headline example is **rate-invariance**: because the
[downconverter](/learn/dsp/dsp-in-gophertrunk/) normalizes every channel to a fixed channel
rate, a given channel must decode to the *same* in-channel quality whether it was captured
at 2.4 MS/s or 10 MS/s and decimated down. A test that asserts this pins the whole
front-end's behaviour with one property.

That invariant is also a **diagnostic**. Read the table:

| Symptom | What the invariant tells you |
|---------|------------------------------|
| Fails live at high rate, **reproduces** in offline replay | the fault is in the **captured data** (front-end overload, phase noise) — not the DSP |
| Fails live but **cannot** be reproduced offline from the same file | the DSP is fine; look upstream at capture/timing |
| Same file decodes differently across runs | non-determinism — a real bug in the code |

The project used precisely this reasoning on a field issue: a channel that locked from a
2.4 MS/s capture but not a 10 MS/s one. Decimating the high-rate file with an *independent*
resampler and replaying it through the proven path reproduced the **same** deficit — proving
the ~10 dB shortfall was baked into the samples (front-end reciprocal mixing), not
GopherTrunk's downconverter.

## A workable debugging loop

Putting it together, the reliable loop is:

1. **Capture** the failing signal to a `.cfile`.
2. **Replay** it to reproduce the failure deterministically.
3. Write a **golden-vector** or **invariant** test that fails on it.
4. Fix until the test passes — and only then claim it fixed.

That last point is a hard rule in this project: a fix isn't verified until a failing-first
test passes *and* the symptom is shown resolved. It is why the same discipline that grades
a link — [SNR, EVM, BER](/learn/dsp/snr-evm-and-ber/) — also grades your *changes* to the
code that decodes it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — reproducing a high-rate failure in offline replay points at the captured data, not the steady-state DSP." markdown="0">
  <p class="knowledge-check__q">Quick check: a bug appears only at a high capture rate but reproduces in offline replay of that file. Where is the fault?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">In the steady-state DSP algorithms</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In the captured data itself — front-end overload or phase noise, not the DSP</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In the display code only</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- DSP is **deterministic**, so **golden vectors** (input→output) turn numeric drift into test failures.
- **Offline replay** of a **`.cfile`** makes an un-reproducible on-air bug repeatable.
- **Invariants** like **rate-invariance** pin whole behaviours and double as diagnostics.
- A fix is verified only when a **failing-first** test passes and the symptom is shown gone.

Next up: the last lesson — how numbers are stored, and why it matters for performance.
