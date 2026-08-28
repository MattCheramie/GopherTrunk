---
slug: fractionally-spaced-equalizer
title: Fractionally-spaced equalizer (FSE)
entry_type: algorithm
category: equalization
description: A fractionally-spaced equalizer places its taps closer than one symbol apart — typically T/2 — so it can synthesize the receive matched filter, correct pulse-shape mismatch, and shrug off sampling-phase errors a symbol-spaced equalizer cannot.
keywords: fractionally spaced equalizer, FSE, T/2 equalizer, symbol-spaced equalizer, matched filter synthesis, pulse shape mismatch, CMA, timing phase, adaptive equalizer, P25 CQPSK
aka: [FSE, T/2 equalizer, fractionally spaced equalizer]
autolink: true
infobox:
  - { label: Type, value: Adaptive equalizer structure }
  - { label: Tap spacing, value: "A fraction of a symbol (usually T/2)" }
  - { label: Advantage, value: Synthesizes the matched filter; timing-phase insensitive }
  - { label: Cost, value: 2× taps, needs a leakage term against tap drift }
see_also: [cma-equalizer, adaptive-filter, matched-filter, root-raised-cosine-filter, cqpsk, snapshot-equalization]
cite_urls:
  - https://en.wikipedia.org/wiki/Equalization_(communications)
  - https://en.wikipedia.org/wiki/Blind_equalization
---

**A fractionally-spaced equalizer** (**FSE**) is an [adaptive
equalizer](/reference/adaptive-filter/) whose taps are spaced *closer than one symbol
apart* — most commonly half a symbol, "T/2" — so it operates on two or more samples per
symbol instead of one.[^wiki] The extra resolution is not a luxury: a symbol-spaced
equalizer can only correct the channel as seen *at* the symbol instants, which makes it
blind to anything happening between them — the shape of the pulse itself, and the exact
sampling phase. An FSE sees the waveform between symbols too, and that lets it do jobs a
symbol-spaced filter structurally cannot.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A symbol waveform sampled once per symbol misses the pulse shape between symbol instants; sampling twice per symbol lets the equalizer taps see and correct the shape itself." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 80 C 50 20 80 20 110 80 C 140 130 170 130 200 80 C 230 30 260 30 290 80 C 320 125 350 125 380 80" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.7"/>
  <g fill="currentColor"><circle cx="65" cy="35" r="3"/><circle cx="155" cy="117" r="3"/><circle cx="245" cy="42" r="3"/><circle cx="335" cy="114" r="3"/></g>
  <g fill="currentColor" fill-opacity="0.5"><circle cx="110" cy="80" r="3"/><circle cx="200" cy="80" r="3"/><circle cx="290" cy="80" r="3"/><circle cx="380" cy="80" r="3"/></g>
  <text x="65" y="24" font-size="8" fill="currentColor" text-anchor="middle">on-time (T)</text>
  <text x="203" y="98" font-size="8" fill="currentColor">midpoint (T/2)</text>
  <text x="230" y="134" font-size="8.5" fill="currentColor" text-anchor="middle">two tap phases per symbol see the pulse, not just its symbol-instant values</text>
</svg>
<figcaption>Symbol-spaced taps touch only the dark samples; the T/2 taps also weight the midpoints, giving the equalizer authority over the pulse shape between decisions.</figcaption>
</figure>

## How it works

The FSE is an FIR filter running at 2× the symbol rate whose output is read once per
symbol; adaptation (LMS against training, or blind
[CMA](/reference/cma-equalizer/)) updates the T/2-spaced taps exactly as it would
symbol-spaced ones. The structural wins:

- **It synthesizes the receive [matched filter](/reference/matched-filter/) implicitly.**
  The optimal receiver front end is matched filter + symbol-rate sampler; a T/2 FSE spans
  both roles, so if the receiver's fixed filter is matched to the *wrong* pulse, the FSE
  quietly learns the correction. A symbol-spaced equalizer cannot — the mismatch lives
  between its taps.
- **It is insensitive to sampling phase.** Symbol-spaced equalization degrades with timing
  offset because aliasing of the (excess-bandwidth) pulse folds differently at each phase;
  T/2 sampling keeps the excess band un-aliased, so residual timing error becomes just
  another linear distortion the taps absorb.
- **It equalizes before decimation**, so channel nulls that land between symbol-rate alias
  points remain correctable.

The costs are 2× the taps and one genuine trap: fractional spacing enlarges the blind cost
function's null space — tap combinations that leave the output modulus unchanged — so on a
clean channel the taps can random-walk. A small **leakage** term (`w ← (1−leak)·w` each
update) shrinks unexcited taps toward zero: real ISI sustains taps, noise alone does not.

## Relevance to SDR

GopherTrunk's FSE (`internal/dsp/equalizer/fse.go`) exists because of a pulse-mismatch bug
class no symbol-spaced filter could fix: the P25 [CQPSK](/reference/cqpsk/)/LSM receiver
matched-filters with a [root-raised-cosine](/reference/root-raised-cosine-filter/), but a
real P25 [C4FM](/reference/c4fm/) transmitter shapes a different (CPM) pulse — the outer
±1800 Hz rails arrive under-shot and the constellation closes, and a symbol-spaced CMA
cannot open it (issue #492). The T/2 blind FSE consumes the timing loop's on-time sample
*and* the half-symbol-earlier midpoint interpolant, corrects the pulse mismatch and any
multipath ISI, and runs ahead of the carrier loop (the CMA cost is rotation-invariant, with
the centre tap phase-pinned so tap drift is not read as a frequency offset). It is always-on
for P25 CQPSK/LSM. Note the contrast with the TETRA paths, where the differential decoder
imposes its own constraint and the equalizers are applied as frozen snapshots — see
[snapshot equalization](/reference/snapshot-equalization/).

## Sources

[^wiki]: [Equalization (communications)](https://en.wikipedia.org/wiki/Equalization_(communications)) — Wikipedia, on adaptive equalizer structures including fractionally-spaced designs and their timing-phase insensitivity.
