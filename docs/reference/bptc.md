---
slug: bptc
title: BPTC
entry_type: algorithm
category: algorithms
description: BPTC (block product turbo code) is the (196,96) forward-error-correction scheme that protects DMR data and control bursts using interleaved row and column parity.
keywords: BPTC, block product turbo code, BPTC(196,96), DMR FEC, product code, interleaving
aka: ["BPTC", "BPTC(196,96)", "block product turbo code"]
autolink: true
see_also: [forward-error-correction, hamming-code, interleaving, dmr, csbk]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

**BPTC** (**block product turbo code**), specifically **BPTC(196,96)**, is the
[forward-error-correction](/reference/forward-error-correction/) scheme that protects
[DMR](/reference/dmr/) data and control bursts.[^wiki] It arranges bits in a grid and applies
parity across **both rows and columns**, so the two passes can correct errors the other
misses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 170" role="img" aria-label="A grid of data bits with parity computed across each row and down each column." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none"><rect x="40" y="20" width="160" height="100"/>
    <line x1="80" y1="20" x2="80" y2="120"/><line x1="120" y1="20" x2="120" y2="120"/><line x1="160" y1="20" x2="160" y2="120"/>
    <line x1="40" y1="45" x2="200" y2="45"/><line x1="40" y1="70" x2="200" y2="70"/><line x1="40" y1="95" x2="200" y2="95"/></g>
  <rect x="200" y="20" width="40" height="100" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="220" y="135" text-anchor="middle" font-size="8" fill="currentColor">row parity</text>
  <rect x="40" y="120" width="160" height="28" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="120" y="138" text-anchor="middle" font-size="8" fill="currentColor">column parity</text>
</svg>
<figcaption>BPTC computes parity across both rows and columns of a bit grid, with interleaving to spread bursts.</figcaption>
</figure>

## Overview

The "product" structure plus [interleaving](/reference/interleaving/) makes BPTC robust
against the burst errors typical of a fading mobile channel — important for the
[CSBK](/reference/csbk/) control messages that keep a trunked DMR system coordinated.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the DMR standard whose data and control bursts BPTC(196,96) protects.
