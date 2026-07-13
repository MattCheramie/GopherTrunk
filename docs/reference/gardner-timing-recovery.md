---
slug: gardner-timing-recovery
title: Gardner timing recovery
entry_type: algorithm
category: synchronization
description: Gardner timing recovery is a non-data-aided symbol-timing algorithm that estimates clock error from two samples per symbol and works independently of carrier phase.
keywords: Gardner timing recovery, symbol timing, timing error detector, TED, clock recovery, non-data-aided, two samples per symbol, Fred Gardner
aka: [Gardner timing recovery, Gardner TED, Gardner]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing (TED) algorithm }
  - { label: Feature, value: Carrier-phase independent, non-data-aided }
  - { label: Rate, value: 2 samples per symbol }
see_also: [mueller-muller-timing-recovery, early-late-gate, clock-recovery, fred-gardner, symbol-rate, eye-diagram]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
related_reading:
  - { title: "SDR Internals, Part 7: Symbol timing & sync recovery", url: /blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Symbol_synchronization
  - https://ieeexplore.ieee.org/document/1096561
---

**Gardner timing recovery** is a feedback algorithm that estimates
[symbol-timing](/reference/clock-recovery/) error from **two samples per symbol** — one at
the symbol instant and one at the midpoint between symbols.[^wiki] Its defining property is
that it is **non-data-aided** (it needs no known preamble) and **independent of carrier
phase**, so it can lock symbol timing *before* the carrier is fully recovered — a major
practical convenience in an SDR demodulator.[^gardner]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A symbol waveform sampled twice per symbol at the on-time strobe and the midpoint, with the midpoint sample used to sense whether the strobe is early or late." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 95 C 90 95 90 40 150 40 C 210 40 210 95 270 95 C 330 95 330 40 390 40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="90" cy="67" r="3.2"/><circle cx="150" cy="40" r="3.2"/><circle cx="210" cy="67" r="3.2"/><circle cx="270" cy="95" r="3.2"/></g>
  <line x1="150" y1="24" x2="150" y2="110" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <line x1="270" y1="24" x2="270" y2="110" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <line x1="90" y1="50" x2="90" y2="110" stroke="currentColor" stroke-dasharray="2 2" stroke-opacity="0.35"/>
  <line x1="210" y1="50" x2="210" y2="110" stroke="currentColor" stroke-dasharray="2 2" stroke-opacity="0.35"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="150" y="20">strobe (on-time)</text><text x="90" y="122">mid</text><text x="210" y="122">mid</text><text x="270" y="20">strobe</text></g>
  <text x="230" y="136" text-anchor="middle" font-size="8.5" fill="currentColor">e = mid × (late strobe − early strobe)</text>
</svg>
<figcaption>Gardner timing recovery reads the sample halfway between two strobes; when the strobes straddle the symbol correctly the midpoint sits at the zero-crossing, and any offset gives a signed timing error.</figcaption>
</figure>

## How it works

The **timing-error detector (TED)** uses three consecutive samples taken at twice the
symbol rate: the previous on-time sample, the current midpoint sample, and the current
on-time sample. The Gardner error is

`e[k] = x_mid[k] · ( x[k] − x[k−1] )`,

the midpoint sample multiplied by the difference between successive on-time samples. The
intuition: if the strobes are perfectly aligned, each on-time sample lands on a symbol peak
and the midpoint lands on the transition zero-crossing, giving `e ≈ 0`. If the clock is
early or late, the midpoint drifts off the crossing and the product acquires a sign that
tells the loop which way to slide the sampling phase. Crucially, the formula uses only the
real difference of successive samples and the midpoint magnitude, so a constant carrier
phase rotation multiplies both arms equally and **cancels out** — hence its phase
independence.

That error drives a standard second-order loop:

- **Loop filter → interpolator control.** The filtered error steers a fractional-delay
  interpolator (or an NCO controlling a resampler) that shifts the strobe toward the centre
  of the [symbol](/reference/symbol-rate/), where the [eye](/reference/eye-diagram/) is
  widest.
- **Two samples per symbol is the price.** Gardner needs exactly 2 samples/symbol, twice
  the throughput of a one-sample detector, but avoids the matched-filter-and-decision
  overhead of decision-directed schemes.
- **Data-transition dependence.** The error is developed on symbol transitions, so long
  runs of identical symbols starve the detector — pulse shaping and scrambling that keep
  transitions frequent help it stay locked.

## Variants

Gardner's detector is one member of the non-data-aided TED family. The
[early-late gate](/reference/early-late-gate/) detector uses samples symmetrically before
and after the strobe to the same end and is closely related. Where sample budget is tight,
the [Mueller–Müller](/reference/mueller-muller-timing-recovery/) detector achieves timing
recovery with only **one** sample per symbol, but it is *decision-directed* and therefore
sensitive to carrier phase — the opposite trade from Gardner. Zero-crossing detectors are a
degenerate, lower-performance cousin.

## Relevance to SDR

Gardner recovery is a workhorse in SDR demodulators for PSK and QAM waveforms — it is the
default symbol-timing block in many GNU Radio flowgraphs and appears throughout land-mobile
(P25, DMR, NXDN) and satellite receive chains. Its carrier-phase independence lets a
receiver lock timing and carrier as loosely coupled stages rather than one fragile joint
loop. GopherTrunk's C4FM/PSK decoders perform symbol-timing recovery of this non-data-aided,
2-samples-per-symbol kind to place the slicer at the eye centre. The detector is named for
[Floyd M. (Fred) Gardner](/reference/fred-gardner/), who published it in 1986.

## Sources

[^wiki]: [Symbol synchronization](https://en.wikipedia.org/wiki/Symbol_synchronization) — Wikipedia, for symbol-timing recovery including the Gardner timing-error detector and its non-data-aided operation.
[^gardner]: [A BPSK/QPSK timing-error detector for sampled receivers](https://ieeexplore.ieee.org/document/1096561) — F. M. Gardner, *IEEE Trans. Communications*, 1986, the original paper deriving the two-sample, carrier-phase-independent TED.
