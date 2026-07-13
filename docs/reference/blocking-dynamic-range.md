---
slug: blocking-dynamic-range
title: Blocking dynamic range (BDR)
entry_type: term
category: rf-metrics
description: Blocking dynamic range is the span in decibels between a receiver's noise floor and the strong off-channel signal level that desensitizes it, measuring resistance to strong-signal overload.
keywords: blocking dynamic range, BDR, blocking, receiver blocking, desensitization, strong signal handling, reciprocal mixing, dynamic range, off-channel rejection
aka: [BDR, blocking, blocking level]
autolink: true
infobox:
  - { label: Type, value: Receiver figure of merit }
  - { label: Unit, value: Decibels (dB) }
  - { label: Bounds, value: Noise floor to blocking level }
see_also: [dynamic-range, desensitization, spurious-free-dynamic-range, receiver-sensitivity, 1-db-compression-point, intermodulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Dynamic_range#Radio
  - https://en.wikipedia.org/wiki/Desensitization_(telecommunications)
---

**Blocking dynamic range** (**BDR**) is the range in
[decibels](/reference/decibel/) between a receiver's
[noise floor](/reference/noise-floor/) and the level of a strong off-channel signal
that [desensitizes](/reference/desensitization/) it by a defined amount — usually
1 dB or 3 dB of sensitivity loss.[^wiki] It measures how loud an unwanted neighbor a
receiver can tolerate before a weak wanted signal starts to fade, and it is one of the
core dimensions of a receiver's overall [dynamic range](/reference/dynamic-range/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A vertical level scale showing the noise floor at the bottom and the blocking level near the top, with the gap between them labelled blocking dynamic range." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bdrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="120" y1="20" x2="120" y2="150" stroke="currentColor" stroke-opacity="0.6"/>
  <line x1="110" y1="140" x2="300" y2="140" stroke="currentColor"/>
  <text x="305" y="144" font-size="10" fill="currentColor">noise floor</text>
  <line x1="110" y1="40" x2="300" y2="40" stroke="currentColor"/>
  <text x="305" y="44" font-size="10" fill="currentColor">blocking level</text>
  <line x1="120" y1="138" x2="120" y2="42" stroke="currentColor" stroke-width="2" marker-end="url(#bdrar)" marker-start="url(#bdrar)"/>
  <rect x="60" y="80" width="40" height="18" fill="none"/>
  <text x="128" y="94" font-size="11" fill="currentColor">BDR (dB)</text>
</svg>
<figcaption>Blocking dynamic range is the vertical span from the noise floor up to the off-channel level that causes a defined loss of sensitivity.</figcaption>
</figure>

## How it works

The lower bound of BDR is the receiver's own noise floor, which fixes the weakest
signal it can hear — its [sensitivity](/reference/receiver-sensitivity/). The upper
bound is the **blocking level**: the power of an interfering carrier, placed at a
stated offset (say 20 kHz or 100 kHz away), that reduces the wanted signal's output by
the reference amount. Subtract the two, in dB, and you have BDR.

A high blocking level, and therefore a wide BDR, requires two things. The front end
must stay linear — high [1 dB compression point](/reference/1-db-compression-point/)
and [third-order intercept](/reference/third-order-intercept/) — so the strong signal
does not compress stage gain. And the [local oscillator](/reference/local-oscillator/)
must be spectrally clean, because at large offsets the blocking level is set by
**reciprocal mixing**: the strong signal beats against the LO's
[phase noise](/reference/phase-noise/) and dumps broadband energy into the passband.
Below a certain offset, LO phase noise usually dominates the limit; farther out,
front-end compression does.

BDR is offset-dependent, so a meaningful figure always names the spacing. It is a
sibling of [spurious-free dynamic range](/reference/spurious-free-dynamic-range/),
which instead measures the level at which internally generated
[intermodulation](/reference/intermodulation/) products emerge from the noise. A
receiver can have excellent BDR yet poor SFDR, or vice versa; a good design needs both.

## In practice

- Narrow front-end selectivity — a [cavity](/reference/cavity-filter/),
  [helical](/reference/helical-filter/), or [SAW filter](/reference/saw-filter/) —
  raises the blocking level for out-of-band interferers by attenuating them before the
  first active stage.
- A cleaner LO, such as one referenced to a [TCXO](/reference/tcxo/) or
  [OCXO](/reference/ocxo/), pushes back the reciprocal-mixing limit and widens BDR at
  close offsets.
- Wideband direct-sampling SDRs typically show lower BDR than a well-filtered
  superheterodyne set, because their entire input reaches the
  [ADC](/reference/analog-to-digital-converter/) unfiltered.

## Relevance to SDR

Blocking dynamic range predicts how a scanner behaves at busy multi-transmitter sites,
where a nearby high-power emitter sits only tens of kilohertz from the channel you want.
[P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), and
[TETRA](/reference/tetra/) trunked systems are frequently deployed alongside strong
paging, broadcast, or cellular transmitters, so BDR often decides whether an SDR can
follow the control channel at all. For a receiver-only decoder like
[GopherTrunk](/reference/software-defined-radio/), BDR is a property of the RF hardware
and its filtering, not of the decode software — but it directly bounds what GT can work
with. When a strong neighbor blocks the front end, the wanted signal never reaches the
DSP intact, and no amount of downstream processing recovers it.

## Sources

[^wiki]: [Dynamic range — Radio](https://en.wikipedia.org/wiki/Dynamic_range#Radio) — Wikipedia, definitions of blocking and dynamic range in receivers.
