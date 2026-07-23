---
slug: clock-and-symbol-recovery
title: Clock & symbol recovery
description: Finding where each symbol begins when the receiver has no shared clock — timing error detectors, interpolation, and the loop that locks onto the data rate.
keywords: clock recovery, symbol recovery, timing recovery, symbol synchronization, gardner detector, timing error, matched filter, dsp clock recovery
level: advanced
status: full
prereq:
  - demodulation
faq:
  - q: Why does a receiver need clock recovery?
    a: The transmitter sends symbols at a precise rate, but the receiver doesn't share its clock and samples on its own slightly different clock. Without correction, the receiver would read symbols at the wrong instants and drift out of step. Clock recovery continuously estimates the correct sampling instant so the decoder reads each symbol at its centre.
  - q: What is a matched filter in this context?
    a: A matched filter is shaped to match the transmitted pulse, and running the signal through it maximizes the signal-to-noise ratio at the symbol centres while suppressing inter-symbol interference. It's applied just before symbol decisions so each symbol is as clean and distinct as possible when the decoder samples it.
---

# Clock & symbol recovery

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The transmitter sends symbols on a clock the receiver doesn't share, so the receiver
must **recover** the timing — figure out where each symbol begins and sample it at its
**centre**. A **timing error detector** measures how early or late the current guess
is, and a feedback **loop** nudges the sampling instant until it **locks**. A
**matched filter** first sharpens each symbol. This is what turns a demodulated wave
into clean bits.
</div>

The demodulator gives you a wiggling signal that steps between symbol levels — but
*when* is each symbol? Without the transmitter's clock, you have to find out. This
lesson is that puzzle and its solution.

## The problem: no shared clock

The transmitter emits, say, 4800 symbols a second on its own crystal. Your receiver
samples on a *different* crystal, a little fast or slow, and started at an arbitrary
moment. If you just read a symbol every so-many samples, you'll gradually drift — and
soon be sampling on the blurry transitions *between* symbols instead of their clean
centres, where decisions go wrong.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A symbol waveform with two sets of sampling instants: some landing at the centres of symbols (good) and some landing on the transitions between them (bad)." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 80 H80 V40 H140 V90 H200 V50 H260 V80 H320 V40 H380 V90 H440" fill="none" stroke="currentColor" stroke-width="1.5" stroke-opacity="0.7"/>
  <g fill="currentColor"><circle cx="50" cy="80" r="4"/><circle cx="110" cy="40" r="4"/><circle cx="170" cy="90" r="4"/><circle cx="230" cy="50" r="4"/></g>
  <text x="130" y="18" text-anchor="middle" font-size="9" fill="currentColor">sampling at centres = good</text>
  <g fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="80" cy="62" r="4"/><circle cx="140" cy="65" r="4"/><circle cx="200" cy="70" r="4"/></g>
  <text x="360" y="18" text-anchor="middle" font-size="9" fill="currentColor">on transitions = errors</text>
</svg>
<figcaption>Read a symbol at its centre (filled dots) and the level is clear; read it on a transition (open dots) and the decision is ambiguous. Clock recovery keeps you on the centres.</figcaption>
</figure>

## The solution: measure the error, correct it

Clock recovery is a feedback **loop**, much like the tuning loops elsewhere in DSP:

1. **Guess** the symbol timing and sample there.
2. A **timing error detector** looks at the samples around the guess and estimates
   whether you're sampling *early* or *late*. (A classic one, the Gardner detector,
   uses the sample halfway between two symbols — it's near zero only when timing is
   right.)
3. A **loop filter** smooths that error estimate and nudges the sampling instant.
4. Repeat every symbol. Once the error settles near zero, the loop has **locked** onto
   the transmitter's rate and tracks its drift automatically.

## Interpolation: sampling between samples

The correct symbol centre rarely lands exactly on one of your samples — it might be
"37% of the way between sample 5 and sample 6." So the loop uses an **interpolator** (a
small filter) to compute the signal's value at that fractional position. This is what
lets timing recovery be smooth and precise rather than jumping a whole sample at a time.

## The matched filter

Just before the decision, the signal usually passes through a **matched filter** shaped
to the transmitted pulse. It maximizes the signal-to-noise ratio right at the symbol
centres and suppresses smearing between neighbouring symbols (inter-symbol
interference), so each symbol is as crisp as possible when sampled. GopherTrunk sizes
this matched filter and the recovery loop from the **channel rate** set by the
[downconverter](/learn/dsp/mixing-and-downconversion/) — which is why its decoder
behaves the same regardless of the original capture rate. The RF path's
[clock recovery](/learn/rf-sdr/clock-recovery/) lesson covers the same loop from the
receiver's view.

<div class="knowledge-check" data-quiz data-correct-msg="Right — clock recovery keeps you sampling at symbol centres despite having no shared clock." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a receiver need clock/symbol recovery?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To increase the sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It has no shared clock, so it must find where each symbol begins</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To remove DC offset</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The receiver has **no shared clock**, so it must **recover** symbol timing or drift
  off the centres.
- A **timing error detector** measures early/late; a **loop** corrects until it
  **locks** and tracks drift.
- An **interpolator** samples between samples for precise, fractional timing.
- A **matched filter** sharpens each symbol first; GopherTrunk sizes it from the
  channel rate.

Next up: keeping the signal at a usable level as it fades — gain and AGC.
