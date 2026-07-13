---
slug: turbo-code
title: Turbo code
entry_type: algorithm
category: error-correction
description: Turbo codes are parallel concatenated convolutional codes with an interleaver, decoded iteratively by exchanging soft information; they approach the Shannon limit in 3G/4G and deep-space links.
keywords: turbo code, parallel concatenated convolutional code, PCCC, interleaver, iterative decoding, extrinsic information, BCJR, near-Shannon, 3G, 4G LTE, deep space
aka: [turbo code, PCCC, parallel concatenated convolutional code]
autolink: true
infobox:
  - { label: Type, value: Iteratively decoded FEC }
  - { label: Built from, value: Two convolutional codes + interleaver }
  - { label: Used by, value: 3G/4G, deep-space telemetry }
see_also: [convolutional-code, bcjr-algorithm, interleaving, ldpc-code, viterbi-algorithm, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Turbo_code
  - https://ieeexplore.ieee.org/document/397441
---

A **turbo code** is a powerful [forward error correction](/reference/forward-error-correction/)
scheme built by running the same data through two [convolutional encoders](/reference/convolutional-code/),
one directly and one through an [interleaver](/reference/interleaving/) that scrambles the bit
order, then transmitting both sets of parity bits.[^wiki] Its breakthrough was the *decoder*:
two soft-input/soft-output [BCJR](/reference/bcjr-algorithm/) decoders that trade reliability
estimates back and forth, iterating until they converge — a process whose feedback loop gave
turbo codes their name and let practical codes get within a fraction of a decibel of the
**Shannon limit**.[^bglt]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="An encoder splits into a direct convolutional encoder and an interleaved one producing two parity streams; the decoder loops two soft BCJR decoders exchanging extrinsic information through an interleaver and deinterleaver." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="230" y="14" text-anchor="middle" font-size="9" fill="currentColor">encoder</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="150" y="24" width="70" height="26"/>
    <rect x="150" y="60" width="70" height="26"/>
    <rect x="60" y="42" width="60" height="26"/>
  </g>
  <text x="90" y="59" text-anchor="middle" font-size="8" fill="currentColor">interleave</text>
  <text x="185" y="41" text-anchor="middle" font-size="8" fill="currentColor">conv enc 1</text>
  <text x="185" y="77" text-anchor="middle" font-size="8" fill="currentColor">conv enc 2</text>
  <line x1="30" y1="55" x2="58" y2="55" stroke="currentColor" stroke-width="1.2" marker-end="url(#tcar)"/>
  <text x="25" y="58" text-anchor="end" font-size="8" fill="currentColor">data</text>
  <line x1="120" y1="55" x2="148" y2="63" stroke="currentColor" stroke-width="1.2" marker-end="url(#tcar)"/>
  <line x1="30" y1="37" x2="148" y2="37" stroke="currentColor" stroke-width="1.2" marker-end="url(#tcar)"/>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="120" y="120" width="80" height="28"/>
    <rect x="270" y="120" width="80" height="28"/>
  </g>
  <text x="160" y="138" text-anchor="middle" font-size="8" fill="currentColor">BCJR dec 1</text>
  <text x="310" y="138" text-anchor="middle" font-size="8" fill="currentColor">BCJR dec 2</text>
  <path d="M200 128 C 235 110, 235 110, 268 128" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#tcar)"/>
  <path d="M270 142 C 235 160, 235 160, 202 142" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#tcar)"/>
  <text x="235" y="105" text-anchor="middle" font-size="8" fill="currentColor">extrinsic LLRs (interleave / deinterleave)</text>
  <text x="235" y="170" text-anchor="middle" font-size="8" fill="currentColor">iterate</text>
</svg>
<figcaption>Two convolutional codes share an interleaver at the encoder; at the decoder two BCJR stages loop, passing extrinsic information until they agree.</figcaption>
</figure>

## How it works

The encoder is deliberately simple. The information bits are sent once (the *systematic*
part), and two recursive convolutional encoders each add parity — the second working on an
interleaved copy so its parity protects a completely different ordering of the same bits. If
a burst of noise wipes out a run of bits, it damages consecutive positions for one encoder but
*scattered* positions for the other, so the two views rarely fail in the same place.

Decoding is where the power lives:

- Decoder 1 runs BCJR on the systematic bits plus parity 1 and produces a soft LLR for each
  bit, then strips out the part it was *given* to keep only its new **extrinsic** information.
- That extrinsic information is interleaved and handed to decoder 2 as a *prior*, which runs
  BCJR on the interleaved bits plus parity 2 and returns its own extrinsic estimate.
- Deinterleave, feed back to decoder 1, and repeat. Each pass sharpens the LLRs; after a
  handful of iterations the bits are decided by sign.

Exchanging only *extrinsic* information (never a decoder's own input echoed back) is what
keeps the loop from reinforcing its own mistakes and lets it converge — the same principle
that makes [LDPC](/reference/ldpc-code/) belief propagation work.

## In practice

Turbo codes trade latency and decoder complexity for coding gain: the iterations and the
double BCJR pass cost far more than a one-shot [Viterbi](/reference/viterbi-algorithm/)
decode, and the interleaver adds delay, so they suit throughput- and power-limited links more
than low-latency ones. Performance hinges on the interleaver design, which sets the code's
minimum distance and its error floor. Rate is tuned by *puncturing* the parity streams.

## Relevance to SDR

Turbo codes carried the data channels of 3G (UMTS/HSPA) and 4G LTE and are standard for
CCSDS deep-space telemetry, where every fraction of a decibel of coding gain saves antenna
size or transmit power. 5G NR later moved its data channels to LDPC and control to
[polar codes](/reference/polar-code/), but turbo remains widespread in deployed cellular and
space systems. The land-mobile and aviation formats GopherTrunk decodes use block and plain
convolutional FEC, not turbo codes, so GT does not implement a turbo decoder; the topic is
covered here as a landmark in near-capacity coding and the primary application of BCJR.

## Sources

[^wiki]: [Turbo code](https://en.wikipedia.org/wiki/Turbo_code) — Wikipedia, for the parallel-concatenated structure and iterative decoding.
[^bglt]: [Near Shannon limit error-correcting coding and decoding: Turbo-codes](https://ieeexplore.ieee.org/document/397441) — Berrou, Glavieux, Thitimajshima, ICC (1993), the introducing paper.
