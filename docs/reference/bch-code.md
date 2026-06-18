---
slug: bch-code
title: BCH code
entry_type: algorithm
category: algorithms
description: BCH codes are a class of cyclic block error-correction codes that can be designed to correct a chosen number of bit errors; POCSAG and DSC use BCH codes.
keywords: BCH code, Bose Chaudhuri Hocquenghem, cyclic code, error correction, POCSAG, DSC
aka: [BCH code]
autolink: true
infobox:
  - { label: Type, value: Cyclic block code }
  - { label: Designed for, value: A target error-correction capability }
  - { label: Used by, value: POCSAG, FLEX, DSC }
see_also: [forward-error-correction, hamming-code, reed-solomon-code, pocsag, dsc]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
external:
  - { title: "BCH code (Wikipedia)", url: https://en.wikipedia.org/wiki/BCH_code }
---

**BCH codes** (Bose–Chaudhuri–Hocquenghem) are a class of cyclic block
[error-correction](/reference/forward-error-correction/) codes that can be constructed to
correct a chosen number of bit errors.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A BCH codeword shown as a block of message bits followed by parity-check bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="230" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="155" y="59" text-anchor="middle" font-size="9" fill="currentColor">message bits (k)</text>
  <rect x="270" y="40" width="150" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="345" y="59" text-anchor="middle" font-size="9" fill="currentColor">parity (n−k)</text>
  <text x="230" y="90" text-anchor="middle" font-size="9" fill="currentColor">e.g. BCH(31,21) in POCSAG</text>
</svg>
<figcaption>BCH codes append algebraically-computed parity bits that correct multiple bit errors per codeword.</figcaption>
</figure>

## How it works

A BCH codeword adds algebraically structured parity bits; the decoder computes a syndrome
to locate and fix errors. [POCSAG](/reference/pocsag/) uses BCH(31,21), and
[DSC](/reference/dsc/) uses a small BCH code.

## Relevance to SDR

BCH decoding lets GopherTrunk recover paging and signalling messages even when a few bits
are corrupted.
