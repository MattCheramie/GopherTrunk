---
slug: interleaving
title: Interleaving
entry_type: algorithm
category: algorithms
description: Interleaving reorders bits or symbols before transmission so that a burst of errors is spread across multiple codewords, letting error-correction codes recover it.
keywords: interleaving, de-interleaving, burst error, FEC, fading, robustness
aka: [interleaving]
autolink: true
infobox:
  - { label: Type, value: Error-resilience technique }
  - { label: Spreads, value: Burst errors across codewords }
  - { label: Paired with, value: FEC codes }
see_also: [forward-error-correction, reed-solomon-code, bch-code, multipath-propagation]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Interleaving_(data)
---

**Interleaving** reorders bits or symbols before transmission and restores their order on
receive, so that a **burst** of errors (from fading or interference) is spread thinly
across many codewords rather than overwhelming one.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Bits written into a matrix by rows and read out by columns, so that a burst of errors is spread across many codewords." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="60" y="30" width="120" height="80"/>
    <line x1="60" y1="50" x2="180" y2="50"/><line x1="60" y1="70" x2="180" y2="70"/><line x1="60" y1="90" x2="180" y2="90"/>
    <line x1="90" y1="30" x2="90" y2="110"/><line x1="120" y1="30" x2="120" y2="110"/><line x1="150" y1="30" x2="150" y2="110"/>
  </g>
  <text x="120" y="125" text-anchor="middle" font-size="8" fill="currentColor">write rows → read columns</text>
  <line x1="200" y1="70" x2="250" y2="70" stroke="currentColor" marker-end="url(#ilar)"/>
  <text x="360" y="60" text-anchor="middle" font-size="9" fill="currentColor">a burst of errors is</text>
  <text x="360" y="74" text-anchor="middle" font-size="9" fill="currentColor">spread thin across</text>
  <text x="360" y="88" text-anchor="middle" font-size="9" fill="currentColor">many codewords</text>
  <defs><marker id="ilar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Interleaving reorders bits so a burst of errors is scattered, letting the error-correction code cope.</figcaption>
</figure>

## How it works

Because [FEC](/reference/forward-error-correction/) codes correct only a limited number of
errors per codeword, distributing a burst lets each codeword's correction cope. The
receiver de-interleaves before decoding.

## Relevance to SDR

Interleaving makes digital radio robust to [multipath](/reference/multipath-propagation/)
fading and short interference bursts.

## Sources

[^wiki]: [Interleaving (data)](https://en.wikipedia.org/wiki/Interleaving_(data)) — Wikipedia, for reordering data to spread burst errors across codewords.
