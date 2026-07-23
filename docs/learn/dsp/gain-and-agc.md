---
slug: gain-and-agc
title: Gain & automatic gain control
description: Keeping a signal at a usable amplitude as it fades — how an AGC loop works and why a demodulator needs stable levels to decide symbols correctly.
keywords: agc, automatic gain control, gain, signal level, agc loop, dsp gain, level control, fading, symbol decision
level: intermediate
status: full
prereq:
  - demodulation
faq:
  - q: What is automatic gain control?
    a: Automatic gain control (AGC) is a feedback loop that continuously adjusts a signal's amplitude to hold it near a target level. As the received signal fades or surges, the AGC turns its gain down or up to compensate, so downstream stages always see a roughly constant level regardless of how strong the signal arrives.
  - q: Why does a demodulator need a stable signal level?
    a: A symbol decision compares the signal against fixed decision levels — for C4FM, the four levels the frequency can take. If the overall amplitude drifts up or down, those comparisons shift and symbols get misread. AGC holds the level steady so the decision thresholds stay valid as the signal fades in and out.
---

# Gain & automatic gain control

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Received signals **fade and surge**, but a demodulator compares against **fixed
decision levels**, so it needs a **stable amplitude**. **Automatic gain control (AGC)**
is a feedback loop that measures the signal's level and adjusts **gain** to hold it near
a target — turning gain down on strong signals, up on weak ones. It's what keeps symbol
decisions valid as conditions change.
</div>

The last piece before bits are reliable: level. This lesson explains why amplitude
matters to a decoder and how AGC keeps it under control.

## Why level matters

A [symbol decision](/learn/dsp/demodulation/) works by comparing the signal to fixed
thresholds — for C4FM, deciding which of four frequency levels a symbol sits at. Those
thresholds assume a known amplitude. But a real signal's strength **varies constantly**:
a vehicle drives behind a hill, [multipath](/learn/rf-sdr/propagation/) fades in and
out, the transmitter is near or far. If the amplitude drifts, the fixed decision levels
no longer line up with the signal's levels, and symbols get misread.

## AGC: a feedback loop for level

**Automatic gain control** solves this the same way clock recovery solves timing — a
feedback loop:

```text
1. measure the signal's current level (its power or magnitude)
2. compare to the target level
3. if too high, reduce gain; if too low, increase gain
4. repeat continuously
```

The result is a signal held near a constant amplitude no matter how it arrives, so the
demodulator's decision levels stay valid.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="An input signal whose amplitude fades and surges over time, passing through an AGC block, emerging with a steady constant amplitude." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 60 q 15 -35 30 0 t 30 0 t 30 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M100 60 q 15 -12 30 0 t 30 0 t 30 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="95" y="100" text-anchor="middle" font-size="9" fill="currentColor">fading input</text>
  <rect x="210" y="42" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="255" y="63" text-anchor="middle" font-size="11" fill="currentColor">AGC</text>
  <line x1="190" y1="59" x2="210" y2="59" stroke="currentColor" stroke-width="1.5" marker-end="url(#g1)"/>
  <line x1="300" y1="59" x2="330" y2="59" stroke="currentColor" stroke-width="1.5" marker-end="url(#g1)"/>
  <path d="M340 60 q 15 -22 30 0 t 30 0 t 30 0 t 30 0 t 30 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="430" y="100" text-anchor="middle" font-size="9" fill="currentColor">steady output</text>
  <defs><marker id="g1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>AGC turns a fading, surging signal into one held at a steady level — so the demodulator's fixed decision thresholds keep working.</figcaption>
</figure>

## The tuning tradeoff

An AGC has an **attack** and **decay** speed — how quickly it reacts. Too fast and it
chases every noise wiggle, distorting the signal; too slow and it can't keep up with
real fades. The loop is tuned to track genuine level changes while ignoring
sample-to-sample noise — a balance, like every feedback loop in this unit.

## Where it sits in the chain

AGC typically runs on the isolated channel just before or alongside demodulation, and
its level target is set from the **channel rate** and expected signal, so — like the
[clock recovery loop](/learn/dsp/clock-and-symbol-recovery/) — the receiver behaves
consistently regardless of the capture rate. The RF path frames the analog side of the
same idea in [gain & AGC](/learn/rf-sdr/gain-and-agc/); here the point is that stable
level is a *precondition* for reliable symbol decisions, and AGC is what guarantees it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — AGC holds the level steady so fixed symbol-decision thresholds stay valid." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a demodulator need AGC?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To shift the channel to zero frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To hold amplitude steady so fixed decision levels stay valid as the signal fades</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To recover the symbol clock</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Received signals **fade and surge**, but decisions use **fixed levels** — so
  amplitude must be **stable**.
- **AGC** is a feedback loop: measure level, compare to target, adjust **gain**,
  repeat.
- Its **attack/decay** speed is tuned to track real fades without chasing noise.
- Stable level is a **precondition** for reliable symbol decisions.

Next up: Unit 5 — see the whole chain assembled in GopherTrunk's real code.
