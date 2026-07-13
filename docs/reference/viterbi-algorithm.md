---
slug: viterbi-algorithm
title: Viterbi algorithm
entry_type: algorithm
category: error-correction
description: The Viterbi algorithm efficiently finds the most likely sequence of states through a trellis, the standard maximum-likelihood way to decode convolutional codes in digital radio.
keywords: Viterbi algorithm, maximum likelihood, trellis, add-compare-select, traceback, soft decision, hard decision, convolutional code, Andrew Viterbi, MLSE
aka: [Viterbi algorithm, Viterbi, Viterbi decoder]
autolink: true
infobox:
  - { label: Type, value: Maximum-likelihood decoder }
  - { label: Decodes, value: Convolutional / trellis codes }
  - { label: Complexity, value: Linear in length, ∝ 2^(K−1) states }
see_also: [convolutional-code, bcjr-algorithm, maximum-likelihood-sequence-estimation, trellis-coded-modulation, forward-error-correction, andrew-viterbi]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Viterbi_algorithm
  - https://en.wikipedia.org/wiki/Viterbi_decoder
---

The **Viterbi algorithm** efficiently finds the most likely sequence of states through a
trellis, given a stream of noisy observations.[^wiki] It is the standard
maximum-likelihood way to decode [convolutional codes](/reference/convolutional-code/), and
it is named for [Andrew Viterbi](/reference/andrew-viterbi/), who published it in 1967.
The same dynamic-programming idea also underlies
[maximum-likelihood sequence estimation](/reference/maximum-likelihood-sequence-estimation/)
for channels with memory and appears far outside radio, from speech recognition to
bioinformatics.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A trellis of states over time with many candidate paths and one highlighted most-likely surviving path." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="100" r="3"/><circle cx="150" cy="40" r="3"/><circle cx="150" cy="100" r="3"/><circle cx="260" cy="40" r="3"/><circle cx="260" cy="100" r="3"/><circle cx="370" cy="40" r="3"/><circle cx="370" cy="100" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.3" stroke-width="1"><line x1="40" y1="40" x2="150" y2="40"/><line x1="40" y1="40" x2="150" y2="100"/><line x1="40" y1="100" x2="150" y2="40"/><line x1="40" y1="100" x2="150" y2="100"/><line x1="150" y1="40" x2="260" y2="40"/><line x1="150" y1="40" x2="260" y2="100"/><line x1="150" y1="100" x2="260" y2="40"/><line x1="150" y1="100" x2="260" y2="100"/><line x1="260" y1="40" x2="370" y2="40"/><line x1="260" y1="40" x2="370" y2="100"/><line x1="260" y1="100" x2="370" y2="40"/><line x1="260" y1="100" x2="370" y2="100"/></g>
  <polyline points="40,100 150,40 260,40 370,100" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="205" y="130" text-anchor="middle" font-size="9" fill="currentColor">most-likely path through the trellis</text>
</svg>
<figcaption>The Viterbi algorithm finds the most-likely sequence through a trellis, decoding convolutional codes.</figcaption>
</figure>

## How it works

The code's memory defines a set of trellis **states** (2^(K−1) of them for constraint
length K), and each received symbol advances the trellis one step. At every step the
decoder computes a **branch metric** — how well each possible transition matches the
received symbol — and runs an **add–compare–select (ACS)** operation for each destination
state: it *adds* each incoming branch metric to the running path metric, *compares* the
competing paths, and *selects* the single best one, discarding the rest. Because only one
*survivor* path is kept per state, the search cost stays linear in the message length
instead of exploding exponentially, yet the surviving global path is provably the
maximum-likelihood sequence.

Once enough steps have accumulated, a **traceback** walks the stored survivor decisions
backward from the best final state to reconstruct the decoded bits. Practical decoders trace
back over a fixed window of roughly five times the constraint length rather than waiting for
the whole message, which bounds latency and memory with negligible loss.

## Variants

- **Hard-decision** decoding feeds the ACS unit sliced bits and uses Hamming distance as the
  metric — simple, but it throws away confidence information.
- **Soft-decision** decoding feeds it the demodulator's real-valued (or quantised) samples
  and uses a Euclidean-style metric, buying roughly 2 dB of coding gain for the same code.
  This is why radios keep soft symbols as far into the pipeline as they can.
- **Contrast with the [BCJR algorithm](/reference/bcjr-algorithm/):** Viterbi minimises the
  *sequence* error probability and emits hard bit decisions, whereas BCJR is a MAP decoder
  that computes the a-posteriori probability of each individual bit — the soft output that
  iterative [turbo](/reference/turbo-code/) and LDPC decoders need. Viterbi is cheaper;
  BCJR gives the soft information.

## In practice

The regular, replicated ACS structure maps cleanly onto hardware, so Viterbi decoders are
routinely built as dedicated blocks in [FPGAs](/reference/field-programmable-gate-array/)
and baseband ASICs; software radios implement the same recursion, often with SIMD to
parallelise the butterfly of ACS updates.

## Relevance to SDR

Viterbi decoding appears wherever convolutional codes do: GSM, IS-95/CDMA, satellite and
deep-space links, 802.11a/g, and, in the scanner world, systems such as
[M17](/reference/m17/) and various trunked-radio signalling paths. GopherTrunk applies
Viterbi decoding on convolutionally coded fields to drive down the error rate before
framing.

## Sources

[^wiki]: [Viterbi algorithm](https://en.wikipedia.org/wiki/Viterbi_algorithm) — Wikipedia, for the maximum-likelihood trellis decoder and its origin. See also [Viterbi decoder](https://en.wikipedia.org/wiki/Viterbi_decoder) for add-compare-select and traceback hardware.
