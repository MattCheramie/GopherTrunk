---
slug: phase-locked-loop
title: Phase-locked loop (PLL)
entry_type: algorithm
category: synchronization
description: A phase-locked loop is a feedback system whose oscillator tracks the phase and frequency of an input, used for carrier recovery, clock recovery, and frequency synthesis.
keywords: phase-locked loop, PLL, phase detector, loop filter, VCO, NCO, carrier recovery, frequency synthesis, loop bandwidth, loop order, lock range, Costas loop
aka: [PLL]
autolink: true
infobox:
  - { label: Type, value: Feedback tracking loop }
  - { label: Locks, value: Phase (and thus frequency) }
  - { label: Used for, value: Carrier recovery, clock recovery, frequency synthesis }
see_also: [costas-loop, numerically-controlled-oscillator, frequency-locked-loop, automatic-frequency-control, clock-recovery, local-oscillator]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-locked_loop
  - https://en.wikipedia.org/wiki/Control_theory
---

A **phase-locked loop (PLL)** is a feedback control system that drives an internal
oscillator to match the **phase** — and therefore the frequency — of a reference
input.[^wiki] It compares the input phase against the oscillator's phase, filters
the error, and steers the oscillator to null that error, producing a local copy
that stays synchronized with the incoming signal. PLLs are the backbone of
carrier recovery, symbol-[clock recovery](/reference/clock-recovery/), and
frequency synthesis across nearly all radio and clocking systems.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A feedback loop: reference and oscillator feed a phase detector, whose error passes through a loop filter to a controlled oscillator that feeds back to the detector." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pllar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="70" y="42" width="74" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="107" y="56">phase</text><text x="107" y="67">detector</text>
    <rect x="190" y="42" width="74" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="227" y="56">loop</text><text x="227" y="67">filter</text>
    <rect x="310" y="42" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="353" y="56">VCO / NCO</text><text x="353" y="67">oscillator</text>
    <line x1="18" y1="59" x2="69" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#pllar)"/><text x="40" y="51">ref</text>
    <line x1="144" y1="59" x2="189" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#pllar)"/><text x="167" y="51">error</text>
    <line x1="264" y1="59" x2="309" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#pllar)"/><text x="287" y="51">tune</text>
    <line x1="396" y1="59" x2="430" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#pllar)"/><text x="415" y="51">out</text>
    <path d="M353 76 V 112 H 107 V 77" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#pllar)"/>
    <text x="230" y="126">phase feedback</text>
  </g>
</svg>
<figcaption>A PLL nulls the phase difference between a reference and its own oscillator: the phase detector's error is smoothed by the loop filter and fed back to tune the VCO/NCO.</figcaption>
</figure>

## How it works

Three blocks in a loop:

- **Phase detector** — outputs a signal proportional to the phase difference
  between the input and the oscillator (a multiplier, an XOR, or a
  phase-frequency detector, depending on the design).
- **Loop filter** — a low-pass (usually proportional-plus-integral) filter that
  removes detector noise and sets the loop's dynamics. Its integrator lets the loop
  hold a nonzero frequency offset with zero steady-state phase error.
- **Controlled oscillator** — a VCO in analog hardware or, in DSP, a
  [numerically-controlled oscillator](/reference/numerically-controlled-oscillator/)
  whose instantaneous phase is advanced by the filtered error.

When the error is nonzero the loop nudges the oscillator until the phases agree;
at that point it is **locked** and the oscillator frequency equals the input
frequency. Because phase is the integral of frequency, locking phase automatically
locks frequency.

## Loop order and bandwidth

Two knobs dominate PLL behaviour:

- **Order/type.** A first-order loop is simple but leaves a static phase offset
  under a frequency error. A **second-order type-II** loop (one extra integrator in
  the filter) tracks a constant frequency offset with zero phase error and is the
  workhorse for carrier and clock recovery. Its dynamics are set by a natural
  frequency and a **damping factor** (≈0.707 is a common critically-damped choice).
- **Loop bandwidth.** Wide bandwidth locks fast and tracks jitter/Doppler but lets
  in more noise; narrow bandwidth is cleaner but slower to acquire and has a smaller
  **pull-in range**. Many receivers open the bandwidth to acquire, then narrow it to
  track — or precede the PLL with a [frequency-locked loop](/reference/frequency-locked-loop/)
  or [AFC](/reference/automatic-frequency-control/) to remove coarse offset first,
  since a PLL alone has a limited frequency pull-in range.

## Variants

- **Costas loop** — a PLL variant whose phase detector is built to recover a
  *suppressed* carrier from a modulated signal, so it can lock to
  [PSK](/reference/phase-shift-keying/) where there is no discrete carrier tone.
  See [costas-loop](/reference/costas-loop/).
- **Frequency synthesizer** — a PLL with a frequency divider in the feedback path
  multiplies a stable reference up to a tunable output, the basis of the
  [local oscillators](/reference/local-oscillator/) in nearly every radio.
- **All-digital PLL** — the entire loop implemented in DSP around an NCO, the norm
  in software radio.

## Relevance to SDR

PLLs (and their Costas cousin) recover the carrier for phase-modulated modes,
discipline sample clocks, and — in hardware — synthesize the tuning frequency of
SDR front-ends. In a trunking decoder a carrier-tracking loop keeps a
[constellation](/reference/constellation-diagram/) from rotating so symbol decisions
stay valid. GopherTrunk uses digital carrier/timing feedback loops in its C4FM and
π-4-DQPSK demodulation chains; the PLL is the general pattern those loops instantiate.

## Sources

[^wiki]: [Phase-locked loop](https://en.wikipedia.org/wiki/Phase-locked_loop) — Wikipedia, on the phase-detector/loop-filter/oscillator feedback structure, loop order, and lock behaviour.
