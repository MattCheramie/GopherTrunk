---
slug: costas-loop
title: Costas loop
entry_type: algorithm
category: synchronization
description: A Costas loop is a carrier-recovery feedback loop that locks to a suppressed carrier's phase and frequency using an I/Q phase detector, enabling coherent PSK/QAM demodulation.
keywords: Costas loop, carrier recovery, phase locked loop, PLL, PSK, QAM, coherent demodulation, I/Q phase detector, John Costas, phase error, loop filter
aka: [Costas loop]
autolink: true
infobox:
  - { label: Type, value: Carrier-recovery loop }
  - { label: Recovers, value: Carrier phase and frequency }
  - { label: Used for, value: PSK/QAM coherent demodulation }
see_also: [phase-locked-loop, frequency-locked-loop, phase-shift-keying, cma-equalizer, phase, john-costas]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
related_reading:
  - { title: "SDR Internals, Part 7: Symbol timing & sync recovery", url: /blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Costas_loop
  - https://en.wikipedia.org/wiki/Phase-locked_loop
---

A **Costas loop** is a phase-locked feedback structure that recovers the
[phase](/reference/phase/) and frequency of a *suppressed* carrier — one with no discrete
tone to lock onto — enabling **coherent** [demodulation](/reference/demodulation/) of
[PSK](/reference/phase-shift-keying/) and related phase-modulated signals.[^wiki] It is a
close relative of the [phase-locked loop](/reference/phase-locked-loop/), but instead of
tracking a pilot tone it derives its phase error directly from the I and Q baseband
components of the data itself, which is why it works when the carrier has been fully
absorbed into the modulation.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A Costas loop: the input splits into I and Q arms multiplied by a local oscillator and its 90-degree copy, the two arms are multiplied together to form a phase error that a loop filter feeds back to a numerically controlled oscillator." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <line x1="18" y1="90" x2="52" y2="90" stroke="currentColor" stroke-width="1.1"/><text x="30" y="82">in</text>
    <circle cx="60" cy="90" r="8" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="93">×</text>
    <line x1="60" y1="82" x2="60" y2="45" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="60" cy="90" r="8" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <line x1="52" y1="90" x2="52" y2="135" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="60" cy="140" r="8" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="143">×</text>
    <line x1="68" y1="45" x2="140" y2="45" stroke="currentColor" stroke-width="1.1"/>
    <line x1="68" y1="140" x2="140" y2="140" stroke="currentColor" stroke-width="1.1"/>
    <rect x="140" y="34" width="46" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="163" y="48">LPF I</text>
    <rect x="140" y="129" width="46" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="163" y="143">LPF Q</text>
    <circle cx="230" cy="92" r="9" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="95">×</text>
    <line x1="186" y1="45" x2="230" y2="45" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="45" x2="230" y2="83" stroke="currentColor" stroke-width="1.1"/>
    <line x1="186" y1="140" x2="230" y2="140" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="140" x2="230" y2="101" stroke="currentColor" stroke-width="1.1"/>
    <text x="205" y="30">I (data)</text>
    <rect x="270" y="80" width="52" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="296" y="95">loop filt</text>
    <line x1="239" y1="92" x2="270" y2="92" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/><text x="255" y="84">error</text>
    <rect x="345" y="80" width="46" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="368" y="95">NCO</text>
    <line x1="322" y1="92" x2="345" y2="92" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/>
    <path d="M368 80 V 20 H 60 V 32" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/>
    <text x="215" y="15">cos / sin feedback</text>
    <path d="M60 45 L 52 140" fill="none" stroke="currentColor" stroke-width="0.9" stroke-dasharray="2 2" stroke-opacity="0.5"/><text x="40" y="118" font-size="7">90°</text>
  </g>
  <defs><marker id="clar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A Costas loop mixes the input against a local oscillator and its 90° copy, multiplies the I and Q arms to form a phase error, and feeds that through a loop filter to an NCO that drives the error to zero.</figcaption>
</figure>

## How it works

The input signal is split into two arms and mixed against a locally generated carrier: one
arm against the in-phase reference (cosine), the other against a 90°-shifted quadrature
reference (sine). After lowpass filtering, the in-phase arm carries the recovered data
symbol and the quadrature arm carries mostly the residual phase error. The classic BPSK
**phase detector** multiplies the two arms together (I × Q). When the loop is aligned this
product is near zero; a small carrier offset tilts the constellation and produces a signed
error proportional to `sin(2·Δφ)` for small angles — an S-curve that tells the loop which
way to turn. Because the detector uses the product of I and Q rather than the sign of a
pilot, the error is *insensitive to the data bits* over the region where it is monotonic,
so it recovers phase even though the carrier is suppressed by the modulation.

That error passes through a **loop filter** and drives a numerically controlled oscillator
(the digital equivalent of a VCO), closing the feedback path:

- **Loop-filter order sets the memory.** A first-order (proportional-only) loop tracks a
  static phase offset but leaves a standing error under a constant frequency offset. A
  **second-order** loop adds an integrator, so it drives *both* residual phase and
  frequency error to zero and holds lock against slow drift — the usual choice for SDR.
  The loop bandwidth trades acquisition speed and pull-in range against noise: a wide loop
  grabs a carrier quickly but lets in more phase jitter.
- **Lock and hang-up.** Once locked the [constellation](/reference/constellation-diagram/)
  stops rotating and the I arm delivers clean symbols. A Costas loop has a **phase
  ambiguity** equal to the modulation's symmetry (180° for BPSK, 90° for QPSK), because
  the detector cannot tell those rotations apart — downstream differential decoding or a
  known sync word resolves it. Loops can also **hang up** near a metastable point of the
  S-curve, stalling acquisition until noise nudges them off it.

## Variants

The BPSK Costas loop generalises by changing the phase detector. **QPSK/π4-DQPSK** loops
use a fourth-power or decision-directed detector that exhibits the same error sign across
all four quadrants. A **decision-directed** detector slices the I/Q sample to the nearest
symbol and measures the angle to that decision, giving lower jitter at high SNR at the cost
of needing reliable decisions. When only frequency — not phase — must be pulled in first, a
[frequency-locked loop](/reference/frequency-locked-loop/) is often run ahead of the Costas
loop to reduce the initial offset into the loop's narrower pull-in range. A blind
[CMA equalizer](/reference/cma-equalizer/) is frequently paired with a Costas loop: the
equaliser opens the eye while the Costas loop removes the residual carrier rotation the
equaliser leaves behind.

## Relevance to SDR

Coherent carrier recovery is a prerequisite for demodulating phase-modulated trunking and
data waveforms — P25 C4FM/CQPSK, the π/4-DQPSK layer in P25 and TETRA, and PSK links in
satellite and telemetry systems. Named for [John P. Costas](/reference/john-costas/), the
loop (or an equivalent PLL-style carrier tracker) stabilises a constellation that would
otherwise spin from local-oscillator offset. GopherTrunk's C4FM/PSK demodulators perform
carrier and residual-frequency correction of this kind so the symbol slicer sees a
stationary constellation; the exact tracker is an implementation detail, but the Costas
loop is the canonical textbook form of what that stage does.

## Sources

[^wiki]: [Costas loop](https://en.wikipedia.org/wiki/Costas_loop) — Wikipedia, on the I/Q carrier-recovery feedback loop for coherent PSK demodulation, its phase detector, and its relation to the PLL.
