---
slug: bptc
title: BPTC
entry_type: algorithm
category: error-correction
description: BPTC (Block Product Turbo Code) is the (196,96) product code that protects DMR data and control bursts by applying Hamming parity across both the rows and columns of an interleaved bit grid.
keywords: BPTC, block product turbo code, BPTC(196,96), DMR FEC, product code, Hamming code, interleaving, row column parity
aka: ["BPTC", "BPTC(196,96)", "block product turbo code"]
autolink: true
infobox:
  - { label: Type, value: Product (turbo product) code }
  - { label: Size, value: BPTC(196,96) — 96 data bits per burst }
  - { label: Built from, value: Hamming rows × Hamming columns }
see_also: [forward-error-correction, hamming-code, interleaving, dmr, csbk]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Product_code
---

**BPTC** (**Block Product Turbo Code**), in its **BPTC(196,96)** form, is the
[forward-error-correction](/reference/forward-error-correction/) scheme that protects
[DMR](/reference/dmr/) data and control bursts.[^wiki] It is a **product code**: bits are laid out
in a grid and a short [Hamming code](/reference/hamming-code/) is applied *both* across each row
*and* down each column, so the two dimensions cover each other's blind spots — an error a row check
misses is very likely caught by the column check that crosses it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 180" role="img" aria-label="A grid of data bits with a Hamming parity block computed across each row and a parity block computed down each column, forming a two-dimensional product code." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none"><rect x="40" y="20" width="160" height="100"/>
    <line x1="80" y1="20" x2="80" y2="120"/><line x1="120" y1="20" x2="120" y2="120"/><line x1="160" y1="20" x2="160" y2="120"/>
    <line x1="40" y1="45" x2="200" y2="45"/><line x1="40" y1="70" x2="200" y2="70"/><line x1="40" y1="95" x2="200" y2="95"/></g>
  <rect x="200" y="20" width="40" height="100" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="220" y="136" text-anchor="middle" font-size="8" fill="currentColor">row parity</text>
  <rect x="40" y="120" width="160" height="28" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="120" y="138" text-anchor="middle" font-size="8" fill="currentColor">column parity</text>
  <text x="150" y="168" text-anchor="middle" font-size="8" fill="currentColor">every bit is covered by a row code AND a column code</text>
</svg>
<figcaption>BPTC arranges the burst's bits in a matrix and Hamming-codes both rows and columns; the crossing checks let the decoder localise and correct errors either pass alone would miss.</figcaption>
</figure>

## How it works

A DMR burst carries 196 coded bits in its payload (98 before and 98 after the sync/embedded
signalling), of which 96 are actual information bits — hence "(196,96)". BPTC first
**de-interleaves** those 196 bits, reversing the standard's bit permutation to place them into a
logical grid, then decodes the grid in two directions. Each **row** is a shortened
Hamming(15,11) codeword and each **column** is a shortened Hamming(13,9) codeword; both are
distance-3 codes, so each can locate and flip a single error in its line. Running the row pass and
the column pass means most single and many multiple error patterns get corrected, because an error
that survives its row lands at the intersection of a column that can still catch it. The 96 recovered
data bits are then read out in their defined order to form the link-control or data payload.

The name calls it a "turbo" product code by analogy: strong product codes can be decoded
*iteratively*, passing soft reliability information back and forth between the row and column
decoders in the spirit of true turbo codes. DMR's practical decoders usually run a single hard-decision
row-then-column pass, which is enough for the short Hamming components at DMR's block sizes, but the
product structure is what gives the label its "turbo."

## In practice

BPTC's real strength shows up against **bursts**. On its own a Hamming code fails the moment two
errors land next to each other; the built-in [interleaving](/reference/interleaving/) and the second
(column) dimension between them convert a physical burst on the channel into isolated single errors,
one per row codeword, which the Hamming passes then mop up. That robustness is exactly what a fading,
mobile DMR channel demands — and it matters most for the [CSBK](/reference/csbk/) control-signalling
blocks that carry channel grants and keep a Tier III trunked system coordinated, where losing a
burst can mean missing a call entirely.

## Relevance to SDR

BPTC(196,96) is core to decoding DMR. GopherTrunk implements the full chain — de-interleave the 196
payload bits, run the row and column Hamming decoders, extract the 96 information bits, and hand them
up as a [CSBK](/reference/csbk/), voice link-control, or data header — for both direct-mode and
trunked [DMR](/reference/dmr/) traffic. Getting the de-interleave permutation and the two Hamming
component definitions exactly right is what lets GopherTrunk correct real bit hits on a marginal DMR
signal instead of discarding the burst, which is often the difference between reporting a talkgroup
and hearing nothing. The same product-code idea — short block codes crossed in two dimensions with
interleaving — recurs across digital radio wherever burst resilience is needed on a short frame.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the DMR standard whose 196-bit data and control bursts BPTC(196,96) protects.
[^prod]: [Product code](https://en.wikipedia.org/wiki/Product_code) — Wikipedia, for the row-and-column product-code construction and its iterative (turbo) decoding.
