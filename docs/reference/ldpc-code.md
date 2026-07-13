---
slug: ldpc-code
title: Low-density parity-check (LDPC) code
entry_type: algorithm
category: error-correction
description: LDPC codes use a sparse parity-check matrix decoded by iterative belief propagation on a Tanner graph, approaching the Shannon limit in Wi-Fi, 5G NR data, DVB-S2, and 10GBASE-T.
keywords: LDPC code, low-density parity-check, sparse parity-check matrix, Tanner graph, belief propagation, sum-product, min-sum, iterative decoding, Gallager, Wi-Fi, 5G NR, DVB-S2
aka: [LDPC code, low-density parity-check code, Gallager code]
autolink: true
infobox:
  - { label: Type, value: Iteratively decoded block code }
  - { label: Decoded by, value: Belief propagation on a Tanner graph }
  - { label: Used by, value: Wi-Fi, 5G NR data, DVB-S2, 10GBASE-T }
see_also: [forward-error-correction, turbo-code, polar-code, bcjr-algorithm, convolutional-code, ofdm]
cite_urls:
  - https://en.wikipedia.org/wiki/Low-density_parity-check_code
  - https://ieeexplore.ieee.org/document/1057683
---

A **low-density parity-check (LDPC) code** is a linear block code defined by a **sparse**
parity-check matrix — one with very few 1s — decoded by passing probabilistic messages back
and forth on a bipartite graph until they settle.[^wiki] First described by Gallager in 1962
and rediscovered in the 1990s, LDPC codes reach within a fraction of a decibel of the
**Shannon limit** while decoding with cheap, parallelisable arithmetic, which is why they now
carry the highest-throughput channels in [Wi-Fi](/reference/wi-fi/), 5G, and satellite.[^gallager]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A Tanner graph with variable nodes for code bits connected sparsely to check nodes for parity equations; messages pass along the edges during belief-propagation decoding." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="230" y="18">check nodes (parity equations)</text>
    <text x="230" y="168">variable nodes (code bits)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="90" y="40" width="16" height="16"/><rect x="220" y="40" width="16" height="16"/><rect x="350" y="40" width="16" height="16"/>
  </g>
  <g fill="currentColor">
    <circle cx="60" cy="130" r="6"/><circle cx="130" cy="130" r="6"/><circle cx="200" cy="130" r="6"/><circle cx="270" cy="130" r="6"/><circle cx="340" cy="130" r="6"/><circle cx="400" cy="130" r="6"/>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.6">
    <line x1="98" y1="56" x2="60" y2="124"/><line x1="98" y1="56" x2="200" y2="124"/><line x1="98" y1="56" x2="340" y2="124"/>
    <line x1="228" y1="56" x2="130" y2="124"/><line x1="228" y1="56" x2="270" y2="124"/><line x1="228" y1="56" x2="400" y2="124"/>
    <line x1="358" y1="56" x2="60" y2="124"/><line x1="358" y1="56" x2="130" y2="124"/><line x1="358" y1="56" x2="270" y2="124"/>
  </g>
</svg>
<figcaption>An LDPC code is a Tanner graph: each check node enforces one parity equation over the few bits it touches, and decoding passes soft messages along the sparse edges.</figcaption>
</figure>

## How it works

The code is fully specified by its parity-check matrix **H**: a valid codeword is any bit
vector that satisfies every parity equation (every row of **H**). "Low density" means each
equation involves only a handful of bits and each bit appears in only a handful of equations.
Draw a **Tanner graph** with one *variable node* per code bit and one *check node* per parity
equation, connecting them wherever **H** has a 1 — the graph is sparse.

Decoding is iterative **belief propagation** (the sum-product algorithm):

- Each variable node starts with the channel's soft estimate (an LLR) of its bit.
- Variable nodes send their beliefs to the check nodes they touch; each check node combines
  the incoming messages to tell every connected bit what value would make its parity equation
  hold.
- Bits gather these hints, update their beliefs, and send again. As with turbo decoding, each
  message carries only *extrinsic* information — a node never echoes back what a neighbour just
  told it.
- After each round the current hard decisions are checked against **H**; when all parity
  equations are satisfied (or an iteration cap is hit) decoding stops.

Because the graph is sparse and every node's update is local, the whole thing parallelises
beautifully in hardware. A cheaper approximation, **min-sum**, replaces the exact check-node
math with a minimum and a sign product, trading a small loss for much simpler logic.

## Variants

Codes are called *regular* when every bit and every check has the same number of edges, and
*irregular* when the degrees vary — carefully chosen irregular degree distributions give the
best threshold performance. Practical standards use **quasi-cyclic** LDPC codes, whose H is
built from small circulant blocks so the same hardware handles many block lengths and rates,
and rate is adapted by adding or shortening parity blocks rather than heavy puncturing.

## Relevance to SDR

LDPC codes protect the data channels of 5G NR, the high-throughput modes of
[Wi-Fi](/reference/wi-fi/) (802.11n/ac/ax), DVB-S2/S2X satellite broadcast, and 10GBASE-T
Ethernet — almost always paired with [OFDM](/reference/ofdm/) waveforms. They are a
capacity-approaching alternative to [turbo codes](/reference/turbo-code/) and
[polar codes](/reference/polar-code/), differing mainly in graph structure and decoder. The
land-mobile and aviation formats GopherTrunk decodes rely on block and convolutional FEC
rather than LDPC, so GT does not implement an LDPC decoder; it is documented here as a pillar
of modern [forward error correction](/reference/forward-error-correction/) in broadband and
satellite systems.

## Sources

[^wiki]: [Low-density parity-check code](https://en.wikipedia.org/wiki/Low-density_parity-check_code) — Wikipedia, for the sparse-matrix definition and belief-propagation decoding.
[^gallager]: [Low-density parity-check codes](https://ieeexplore.ieee.org/document/1057683) — R. G. Gallager, IRE Trans. Information Theory (1962), the originating work.
