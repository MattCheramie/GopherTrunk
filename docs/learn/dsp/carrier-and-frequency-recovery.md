---
slug: carrier-and-frequency-recovery
title: Carrier & frequency recovery
description: Correcting the frequency and phase offset between transmitter and receiver — the PLL and Costas loop that lock a carrier before symbols can be read.
keywords: carrier recovery, frequency recovery, phase offset, pll, costas loop, frequency offset correction, carrier sync, phase locked loop
level: advanced
status: full
prereq:
  - demodulation
  - complex-signals-and-iq
faq:
  - q: Why is there a frequency offset between transmitter and receiver?
    a: "The transmitter and receiver each derive their frequencies from independent crystal oscillators, and no two crystals are exactly on frequency — they differ by a few parts per million and drift with temperature. That small mismatch means the received signal arrives slightly off the tuned centre, and its phase constellation slowly rotates until a recovery loop corrects it."
  - q: What is the difference between a PLL and a Costas loop?
    a: "A plain phase-locked loop locks onto a signal that has an actual carrier tone present. A Costas loop is a variant designed for suppressed-carrier signals like PSK, where the modulation itself removes any steady carrier. It compares the in-phase and quadrature components to derive a phase error even though there is no discrete carrier to lock to directly."
---

# Carrier & frequency recovery

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Transmitter and receiver run on **independent oscillators**, so the signal arrives with a
**frequency** and **phase** offset — the [constellation](/learn/dsp/constellations-and-symbol-mapping/)
slowly **spins**. A **phase-locked loop (PLL)** measures that error and drives a local
oscillator to cancel it. For **suppressed-carrier** PSK, where the modulation hides the
carrier, a **Costas loop** derives the error from I and Q. Only once the carrier is locked
can symbols be read reliably.
</div>

[Clock recovery](/learn/dsp/clock-and-symbol-recovery/) found *when* to sample; this
lesson finds the *frequency and phase* the sampled points are referenced to. Both loops
must lock before a decoder can trust its symbols. It builds on
[demodulation](/learn/dsp/demodulation/) and [I/Q](/learn/dsp/complex-signals-and-iq/).

## The offset problem

Your receiver tuned to 851.0125 MHz, but its crystal and the transmitter's disagree by a
few parts per million. The consequence at baseband: the signal is not sitting exactly at
zero, and its phase reference is unknown. On the I/Q plane the whole
[constellation](/learn/dsp/constellations-and-symbol-mapping/) **rotates** — a frequency
offset makes it spin continuously, a phase offset holds it at a fixed wrong angle. Sample
a spinning constellation and every symbol decision is garbage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="Left, four constellation points rotated off the axes by a frequency offset with a curved arrow showing spin; right, the same four points snapped onto the axes after carrier lock." xmlns="http://www.w3.org/2000/svg">
  <g>
    <line x1="20" y1="75" x2="200" y2="75" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="110" y1="15" x2="110" y2="135" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="150" cy="45" r="4" fill="currentColor"/><circle cx="65" cy="40" r="4" fill="currentColor"/>
    <circle cx="70" cy="110" r="4" fill="currentColor"/><circle cx="155" cy="105" r="4" fill="currentColor"/>
    <path d="M175 60 A 35 35 0 0 1 150 90" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="110" y="148" text-anchor="middle" font-size="9" fill="currentColor">offset: spinning</text>
  </g>
  <text x="255" y="79" text-anchor="middle" font-size="15" fill="currentColor">&#8594;</text>
  <text x="255" y="60" text-anchor="middle" font-size="8" fill="currentColor">lock</text>
  <g>
    <line x1="310" y1="75" x2="500" y2="75" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="405" y1="15" x2="405" y2="135" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="445" cy="40" r="4" fill="currentColor"/><circle cx="365" cy="40" r="4" fill="currentColor"/>
    <circle cx="365" cy="110" r="4" fill="currentColor"/><circle cx="445" cy="110" r="4" fill="currentColor"/>
    <text x="405" y="148" text-anchor="middle" font-size="9" fill="currentColor">locked: fixed</text>
  </g>
</svg>
<figcaption>A frequency offset spins the constellation; carrier recovery cancels it so the points sit still where the decision logic expects them.</figcaption>
</figure>

## The phase-locked loop

The workhorse is a **PLL**. It is a feedback loop with three parts:

```text
  incoming phase ->[ phase detector ]-> error ->[ loop filter ]->[ NCO ]-.
                          ^                                              |
                          '----------- corrected phase -----------------'
```

1. A **phase detector** compares the incoming phase to the local oscillator's phase.
2. A **loop filter** smooths that error (setting how fast and how steadily it tracks).
3. A **numerically controlled oscillator (NCO)** adjusts the local phase/frequency to
   drive the error toward zero.

Lock achieved, the NCO now matches the incoming carrier, and multiplying the signal by
its conjugate de-rotates the constellation to a standstill.

## Costas loops for suppressed carriers

PSK and QAM are **suppressed-carrier**: the modulation is symmetric, so averaged over
time there is no steady carrier tone for a plain PLL to grab. A **Costas loop** solves
this by building the phase error from the *data* itself — it multiplies the in-phase and
quadrature arms in a way that yields a usable error signal regardless of which symbol was
sent. It is a PLL specialised for carrying no carrier, and it is what locks the phase of
the PSK-family signals a digital scanner decodes.

## Ordering with clock recovery

Carrier and [clock recovery](/learn/dsp/clock-and-symbol-recovery/) are cooperating loops:
timing recovery decides *when* to sample, carrier recovery decides the *phase reference*
for those samples. Real receivers run them together, sometimes interacting, and only when
**both** are locked does a clean, stationary constellation appear — ready for the
[symbol decisions](/learn/dsp/constellations-and-symbol-mapping/) that follow. A stubborn
frequency offset is a classic cause of a signal that shows energy but never decodes, a
symptom the [troubleshooting](/learn/digital-trunking/troubleshooting-a-decode/) guide
returns to.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a Costas loop recovers phase for suppressed-carrier signals like PSK." markdown="0">
  <p class="knowledge-check__q">Quick check: which loop is designed for suppressed-carrier PSK, where no steady carrier tone exists?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">An anti-aliasing filter</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A Costas loop</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A decimator</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Independent oscillators leave a **frequency/phase offset** that **spins** the constellation.
- A **PLL** — phase detector, loop filter, NCO — measures and cancels the offset.
- **Suppressed-carrier** PSK needs a **Costas loop**, which derives phase error from I and Q.
- Carrier recovery and **clock recovery** must both lock before symbols read cleanly.

Next up: keeping the signal at a usable amplitude as it fades — gain and automatic gain control.
