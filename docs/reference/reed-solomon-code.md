---
slug: reed-solomon-code
title: Reed–Solomon code
entry_type: algorithm
category: algorithms
description: Reed–Solomon is a block error-correction code that operates on symbols rather than bits, excelling at correcting bursts of errors; used in P25, DMR, storage, and broadcast.
keywords: Reed-Solomon, RS code, block code, burst error, symbols, FEC
aka: [Reed–Solomon code, Reed-Solomon]
autolink: true
infobox:
  - { label: Type, value: Block error-correction code }
  - { label: Operates on, value: Symbols (good vs bursts) }
  - { label: Used by, value: P25, DMR, storage, broadcast }
see_also: [forward-error-correction, bch-code, golay-code, interleaving]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Reed–Solomon error correction (Wikipedia)", url: https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction }
---

**Reed–Solomon** is a block [error-correction](/reference/forward-error-correction/) code
that works on multi-bit **symbols** rather than individual bits, making it especially good
at correcting **bursts** of errors.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A codeword of data symbols followed by parity symbols, with the parity able to correct a burst of damaged symbols." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="30" y="40" width="34" height="30" fill="none"/><rect x="64" y="40" width="34" height="30" fill="none"/><rect x="98" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="132" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="166" y="40" width="34" height="30" fill="none"/><rect x="200" y="40" width="34" height="30" fill="none"/>
    <rect x="250" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="284" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="318" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/>
  </g>
  <text x="132" y="90" text-anchor="middle" font-size="9" fill="currentColor">data symbols (shaded = damaged)</text>
  <text x="301" y="90" text-anchor="middle" font-size="9" fill="currentColor">parity</text>
</svg>
<figcaption>Reed–Solomon adds parity symbols that correct bursts of damaged symbols — used across P25 and DMR.</figcaption>
</figure>

## How it works

It adds parity symbols so that a number of symbol errors can be located and corrected. It
is frequently paired with [interleaving](/reference/interleaving/) to spread burst errors
across codewords.

## Relevance to SDR

Reed–Solomon protects parts of [P25](/reference/project-25/) and
[DMR](/reference/dmr/) (and is ubiquitous in storage and broadcast), helping the decoder
recover data on marginal signals.
