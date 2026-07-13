---
slug: bit-error-rate
title: Bit error rate (BER)
entry_type: term
category: rf-metrics
description: Bit error rate is the fraction of received bits that are wrong, the primary measure of digital link quality, plotted against Eb/N0 as a waterfall curve.
keywords: bit error rate, BER, error rate, digital link quality, waterfall curve, Eb/N0, coding gain, BER floor, symbol error rate
aka: [BER]
autolink: true
infobox:
  - { label: Symbol, value: BER }
  - { label: Unit, value: Dimensionless ratio }
  - { label: Formula, value: "errored bits / total bits" }
see_also: [eb-n0, forward-error-correction, signal-to-noise-ratio, soft-decision, error-vector-magnitude]
cite_urls:
  - https://en.wikipedia.org/wiki/Bit_error_rate
---

**Bit error rate** (**BER**) is the number of received bits that differ from the bits
that were sent, divided by the total number of bits — the single most direct measure
of a digital link's health.[^wiki] A BER of 10⁻³ means one bit in a thousand is wrong;
a clean link might run 10⁻⁶ or better. BER is what
[forward error correction](/reference/forward-error-correction/) exists to reduce, and
what every other link metric ultimately tries to predict.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A log-scale waterfall curve of bit error rate falling steeply as Eb/N0 increases, with a coded curve to the left of an uncoded curve showing coding gain." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="20" x2="50" y2="165" stroke="currentColor" stroke-opacity="0.6"/>
  <line x1="50" y1="165" x2="430" y2="165" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="8" y="26" font-size="9" fill="currentColor">10⁻¹</text>
  <text x="8" y="100" font-size="9" fill="currentColor">BER</text>
  <text x="8" y="163" font-size="9" fill="currentColor">10⁻⁶</text>
  <text x="230" y="185" font-size="10" fill="currentColor">Eb/N0 (dB) →</text>
  <path d="M70 30 C150 40, 210 70, 300 160" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="300" y="150" font-size="9" fill="currentColor">uncoded</text>
  <path d="M60 30 C110 42, 150 75, 210 160" fill="none" stroke="currentColor" stroke-width="1.6" stroke-dasharray="4 3"/>
  <text x="120" y="130" font-size="9" fill="currentColor">coded</text>
  <line x1="210" y1="150" x2="300" y2="150" stroke="currentColor" stroke-dasharray="2 2" stroke-opacity="0.7"/>
  <text x="230" y="145" font-size="8" fill="currentColor">coding gain</text>
</svg>
<figcaption>A BER waterfall curve: error rate plunges as Eb/N0 rises. Coding shifts the curve left — the horizontal gap at a target BER is coding gain.</figcaption>
</figure>

## How it works

BER is estimated by comparing received bits against a known transmitted sequence and
counting mismatches over a long enough run to be statistically meaningful — to confirm
a BER of 10⁻⁶ with confidence you must observe on the order of 10⁷ or more bits. For a
given modulation, BER is a smooth function of the per-bit
[signal-to-noise ratio](/reference/signal-to-noise-ratio/), expressed as
[Eb/N0](/reference/eb-n0/). Plotting BER (log scale) against Eb/N0 (dB) yields the
characteristic **waterfall curve**: nearly flat and high at low SNR, then plunging
almost vertically once enough energy per bit is present. The steepness is why digital
links have a "cliff" — a couple of dB can move a channel from unusable to flawless.

Different modulations trace different waterfalls. Robust binary schemes like
[BPSK](/reference/bpsk/) tolerate low Eb/N0; dense
[QAM](/reference/quadrature-amplitude-modulation/) constellations pack more bits per
symbol but need higher Eb/N0 for the same BER because their points sit closer together.

## Variants

- **Uncoded (raw) BER** — measured on the demodulated bitstream before decoding.
  This reflects the channel and modulation directly.
- **Coded (post-FEC) BER** — after error correction. FEC shifts the waterfall left;
  the horizontal distance to reach a target BER is the **coding gain**, typically
  several dB.
- **Symbol / frame / block error rate** — related metrics counting whole wrong symbols,
  frames, or blocks rather than individual bits, often more meaningful for
  packet systems.
- **BER floor** — an irreducible error rate that persists no matter how high the SNR
  climbs, caused by implementation flaws such as
  [ISI](/reference/intersymbol-interference/), phase noise, or timing jitter.

## In practice

- Standards specify a target BER (or residual BER after FEC) as the pass/fail criterion
  for [receiver sensitivity](/reference/receiver-sensitivity/) — for example a BER of
  5% is a common P25 sensitivity reference point.
- [Soft-decision](/reference/soft-decision/) decoding lowers BER for the same channel
  by feeding the decoder confidence values instead of hard 0/1 calls, typically buying
  ~2 dB over hard decisions.

## Relevance to SDR

BER is the honest bottom line for any digital decoder. In trunked systems —
[P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), [NXDN](/reference/nxdn/),
[TETRA](/reference/tetra/) — a rising BER on the [control channel](/reference/control-channel/)
means missed channel grants and lost calls long before voice becomes unintelligible.
[GopherTrunk](/reference/software-defined-radio/) does not usually surface a raw BER
number to users, but the same physics governs its decode chain: as demod SNR falls, bit
errors climb, the [FEC](/reference/forward-error-correction/) eventually cannot keep up,
and frames fail their [CRC](/reference/cyclic-redundancy-check/) checks. Understanding
the waterfall explains why marginal signals fail abruptly rather than gracefully — GT
is riding the cliff.

## Sources

[^wiki]: [Bit error rate](https://en.wikipedia.org/wiki/Bit_error_rate) — Wikipedia, definition, measurement, and the BER-versus-Eb/N0 relationship.
