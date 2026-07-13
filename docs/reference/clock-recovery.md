---
slug: clock-recovery
title: Clock recovery
entry_type: term
category: sdr-dsp
description: Clock recovery determines a digital signal's symbol timing from the signal itself, so the receiver samples each symbol at its centre where the eye is widest.
keywords: clock recovery, symbol timing, timing recovery, symbol synchronization, Gardner, Mueller-Muller, timing error detector, interpolation
aka: [clock recovery, symbol timing]
autolink: true
infobox:
  - { label: Type, value: Timing-synchronisation stage }
  - { label: Recovers, value: Symbol timing from the signal }
  - { label: Algorithms, value: Gardner, Mueller–Müller }
see_also: [gardner-timing-recovery, mueller-muller-timing-recovery, symbol-rate, eye-diagram, demodulation, frame-synchronization]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
related_reading:
  - { title: "SDR Internals, Part 7: Symbol timing & sync recovery", url: /blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Clock_recovery
  - https://en.wikipedia.org/wiki/Eye_pattern
---

**Clock recovery** determines a digital signal's [symbol](/reference/symbol-rate/) timing
from the signal itself, since the transmitter's clock is not shared with the receiver.[^wiki] It
lets the receiver sample each symbol at its **centre**, where the [eye](/reference/eye-diagram/)
is widest and the decision is most reliable. Without it, samples drift across symbol boundaries
and the decoded stream collapses even when the signal is otherwise clean.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 140" role="img" aria-label="An eye diagram with a dashed line at the centre showing where clock recovery aims the sampling instant." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-opacity="0.85">
    <path d="M40 30 C120 30 120 110 200 110 C280 110 280 30 360 30"/>
    <path d="M40 110 C120 110 120 30 200 30 C280 30 280 110 360 110"/>
  </g>
  <line x1="200" y1="20" x2="200" y2="120" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="200" y="136" text-anchor="middle" font-size="10" fill="currentColor">recovered clock samples at the eye centre</text>
</svg>
<figcaption>Clock recovery finds the symbol rhythm from the signal itself so each symbol is sampled at its centre.</figcaption>
</figure>

## How it works

The transmitter and receiver run independent oscillators, so the receiver knows the *nominal*
symbol rate but not the exact phase — where within each symbol period the centre falls — nor the
precise rate, which differs by a few parts per million and slowly drifts. Clock recovery is a
feedback loop that closes both gaps:

- A **timing-error detector** (TED) estimates whether the current sampling instant is early or
  late relative to the symbol centre, usually by examining the signal near the transitions
  between symbols, where timing error shows up most strongly.
- A **loop filter** smooths that noisy error estimate into a stable correction, setting the
  loop's tracking bandwidth: wide enough to follow drift, narrow enough to reject noise.
- An **interpolator or controlled resampler** shifts the effective sampling phase by a fraction
  of a sample so the next symbol is read at its centre, and the loop repeats.

The result tracks slow clock drift automatically. The loop must first *acquire* lock (pull the
sampling phase to the centre) and then *track* it; a preamble of known alternating symbols in
the transmitted frame gives it clean transitions to lock onto quickly.

## Variants

- **[Gardner](/reference/gardner-timing-recovery/)** — a non-data-aided detector that needs two
  samples per symbol and is insensitive to carrier phase, which makes it a common first choice
  for QPSK and PSK modes.
- **[Mueller–Müller](/reference/mueller-muller-timing-recovery/)** — a decision-directed detector
  that works at one sample per symbol but assumes the carrier is already phase-locked.
- **Early–late gate** and **zero-crossing** detectors — older, intuitive schemes that compare
  samples taken slightly before and after the estimated centre, or watch where the waveform
  crosses zero.

The choice trades sample-rate cost, sensitivity to residual carrier error, and acquisition
speed. All feed the same kind of loop filter and interpolator.

## In practice

Clock recovery comes after demodulation and before [frame synchronization](/reference/frame-synchronization/):
timing recovery finds *where* each symbol is, frame sync then finds *which* symbol starts the
frame. It is one of the most failure-prone stages to watch, because it can hold a false lock —
sampling consistently off-centre — and still produce a plausible-looking but wrong bit stream.
An [eye diagram](/reference/eye-diagram/) is the direct diagnostic: a wide-open eye with the
sample points clustered at its centre means the loop is locked correctly.

## Relevance to SDR

Every digital mode GopherTrunk decodes runs a timing-recovery loop between the demodulator and
the symbol slicer. Loss of symbol lock — from low SNR or
[multipath](/reference/multipath-propagation/) — closes the eye and breaks the decode, which is
exactly what the receiver scopes are meant to reveal. Because the loop tracks small rate offsets,
it also absorbs residual clock error that survives coarse
[PPM correction](/reference/ppm-frequency-correction/) upstream.

## Sources

[^wiki]: [Clock recovery](https://en.wikipedia.org/wiki/Clock_recovery) — Wikipedia, on recovering symbol timing from a received signal.
[^eye]: [Eye pattern](https://en.wikipedia.org/wiki/Eye_pattern) — Wikipedia, on the eye diagram whose centre the recovered clock samples.
