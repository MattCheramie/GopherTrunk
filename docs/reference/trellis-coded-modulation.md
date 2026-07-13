---
slug: trellis-coded-modulation
title: Trellis-coded modulation (TCM)
entry_type: algorithm
category: error-correction
description: Trellis-coded modulation fuses convolutional coding with the symbol constellation via set partitioning, buying coding gain with no extra bandwidth; P25 Phase 1 trellis-codes its C4FM dibits.
keywords: trellis coded modulation, TCM, Ungerboeck, set partitioning, convolutional, Viterbi, coding gain, P25, C4FM, dibit
aka: [trellis-coded modulation, TCM, Ungerboeck coding]
autolink: true
infobox:
  - { label: Type, value: Coded-modulation scheme }
  - { label: Combines, value: Convolutional coding + constellation mapping }
  - { label: Decoded by, value: Viterbi over a trellis }
see_also: [convolutional-code, viterbi-algorithm, forward-error-correction, c4fm, project-25, p25-phase-1]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trellis_modulation
  - https://en.wikipedia.org/wiki/Trellis_coded_modulation
---

**Trellis-coded modulation** (**TCM**) integrates
[convolutional coding](/reference/convolutional-code/) directly into the modulation's
symbol mapping, so it wins **coding gain without spending extra bandwidth**.[^wiki] Instead of
adding parity bits (which would need more symbols or a wider channel), TCM enlarges the
constellation and lets the code decide which symbol transitions are legal, so redundancy hides
in the *geometry* of the signal rather than in extra bits on the wire. It was introduced by
Gottfried Ungerboeck in the early 1980s and transformed voiceband modems; the same idea now
protects several digital-radio data streams.[^tcm]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A convolutional trellis whose branch labels map onto points of a partitioned constellation, so that valid symbol sequences are maximally separated." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="95" r="3"/><circle cx="150" cy="40" r="3"/><circle cx="150" cy="95" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.5"><line x1="40" y1="40" x2="150" y2="40"/><line x1="40" y1="40" x2="150" y2="95"/><line x1="40" y1="95" x2="150" y2="40"/><line x1="40" y1="95" x2="150" y2="95"/></g>
  <text x="95" y="122" text-anchor="middle" font-size="8" fill="currentColor">trellis (which transitions are legal)</text>
  <line x1="178" y1="67" x2="216" y2="67" stroke="currentColor" marker-end="url(#tcmar)"/>
  <text x="197" y="58" text-anchor="middle" font-size="8" fill="currentColor">map</text>
  <line x1="245" y1="67" x2="435" y2="67" stroke="currentColor" stroke-opacity="0.3"/><line x1="340" y1="27" x2="340" y2="107" stroke="currentColor" stroke-opacity="0.3"/>
  <g fill="currentColor"><circle cx="305" cy="45" r="3"/><circle cx="375" cy="45" r="3"/><circle cx="305" cy="90" r="3"/><circle cx="375" cy="90" r="3"/></g>
  <text x="340" y="122" text-anchor="middle" font-size="8" fill="currentColor">partitioned constellation</text>
  <defs><marker id="tcmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>TCM labels a trellis's branches with constellation points chosen by set partitioning, so the sequences the code allows are the ones farthest apart in signal space.</figcaption>
</figure>

## How it works

The engine of TCM is **set partitioning**. Take a constellation and split it recursively into
subsets, at each step maximising the minimum Euclidean distance *within* a subset. A
convolutional encoder then decides which *subset* the next symbol may come from, while the
uncoded data bits pick a specific point *inside* that subset. Because the code steers the
signal toward transitions that are far apart in Euclidean distance, an error would have to jump
a larger gap than in the uncoded scheme — that gap is the coding gain, typically 3–6 dB, and it
comes without adding symbols. The receiver runs the [Viterbi algorithm](/reference/viterbi-algorithm/)
over the trellis, but crucially it measures **Euclidean** (soft) distance between the received
sample and each candidate point, not Hamming distance on hard-decided bits. This joint
treatment of coding and modulation is exactly why TCM beats a separately-designed code plus
modulation of the same rate.

The key insight Ungerboeck contributed was that you should *not* optimise the code and the
constellation independently. A convolutional code maximises *free Hamming distance*, but on a
Gaussian channel what matters is *free Euclidean distance*. Mapping code branches onto a
partitioned constellation makes those two align, so the trellis's most-likely confusable paths
are also the ones the constellation keeps farthest apart.

## In practice

TCM trades decoder complexity (a Viterbi search with soft metrics) for spectral efficiency, so
it thrives where bandwidth is scarce and a demodulator can afford soft decisions — telephone-line
modems (V.32, V.34), some satellite links, and digital land-mobile radio. Its natural partner
is a modulation whose symbols already carry several bits, so there is room to "spend" one bit on
the code without widening the channel.

## Relevance to SDR

The clearest radio example is [Project 25](/reference/project-25/) **Phase 1**. P25 Phase 1 uses
[C4FM](/reference/c4fm/), a four-level FSK that carries two bits ([dibit](/reference/dibit/)) per
symbol at 4800 symbols/s. The data and control payloads are protected by a rate-1/2
(and, punctured, rate-3/4) **trellis code** that operates directly on those dibit symbols: pairs
of dibits form the code's input, and the trellis constrains which four-level symbol sequences are
valid, so the decoder can recover the payload well below the SNR an uncoded C4FM stream would
need. GopherTrunk implements this P25 trellis decoder in its
[C4FM](/reference/c4fm/) decode chain — after symbol recovery it runs the Viterbi search over the
P25 trellis to reconstruct the [TSBK](/reference/tsbk/) control words and data blocks, which is
part of why a marginal P25 control channel still yields usable channel grants rather than dropping
out entirely.

## Sources

[^wiki]: [Trellis modulation](https://en.wikipedia.org/wiki/Trellis_modulation) — Wikipedia, for combining convolutional coding with the constellation mapping to gain coding gain without extra bandwidth.
[^tcm]: [Trellis coded modulation](https://en.wikipedia.org/wiki/Trellis_coded_modulation) — Wikipedia, for Ungerboeck's set-partitioning method and Euclidean-distance Viterbi decoding.
