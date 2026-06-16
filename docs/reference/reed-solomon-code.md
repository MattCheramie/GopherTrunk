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
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Reed–Solomon error correction (Wikipedia)", url: https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction }
---

**Reed–Solomon** is a block [error-correction](/reference/forward-error-correction/) code
that works on multi-bit **symbols** rather than individual bits, making it especially good
at correcting **bursts** of errors.

## How it works

It adds parity symbols so that a number of symbol errors can be located and corrected. It
is frequently paired with [interleaving](/reference/interleaving/) to spread burst errors
across codewords.

## Relevance to SDR

Reed–Solomon protects parts of [P25](/reference/project-25/) and
[DMR](/reference/dmr/) (and is ubiquitous in storage and broadcast), helping the decoder
recover data on marginal signals.
