---
slug: tetra-block-interleaver
title: TETRA block interleaver
entry_type: algorithm
category: error-correction
description: The TETRA block interleaver reorders a channel's coded bits by the rule k = 1 + ((a·i) mod K), spreading burst errors apart before the convolutional decoder; each logical channel carries its own (K, a) pair such as (120, 11) or (432, 103).
keywords: TETRA interleaver, block interleaver, K a interleaving, k = 1 + a i mod K, BSCH SCH interleave, burst error spreading, EN 300 392-2 8.2.4.1
aka: [TETRA (K a) interleaver, "TETRA block interleaving"]
autolink: true
infobox:
  - { label: Rule, value: "k = 1 + ((a·i) mod K)" }
  - { label: Type, value: coprime multiplicative permutation }
  - { label: Channels, value: "BSCH, SCH/HD, SCH/HU, SCH/F" }
  - { label: Spec, value: EN 300 392-2 §8.2.4.1 }
see_also: [interleaving, tetra-logical-channels, tetra-rcpc-code, forward-error-correction, convolutional-code, tetra, tetra-scrambler, tetra-burst-formats]
cite_urls:
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
  - https://en.wikipedia.org/wiki/Interleaving_(disk_storage)
---

The **TETRA block interleaver** reorders a channel's coded bits before transmission so that a physical
**burst** of errors on the air is spread into isolated single errors the [convolutional decoder](/reference/convolutional-code/)
can then correct.[^burst] TETRA uses a *multiplicative* [interleaver](/reference/interleaving/): the bit at
1-indexed input position `i` is written to output position `k = 1 + ((a·i) mod K)`, where `K` is the block
length and `a` is a channel-specific multiplier chosen coprime with `K` so the mapping is a full permutation.[^ileave]
Because a fading channel damages *adjacent* symbols together, scattering them across the block is what makes
the downstream FEC effective — a run of errors on air becomes one error per codeword after de-interleaving.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 132" role="img" aria-label="A contiguous run of damaged bits in the transmitted block maps, through the multiplicative permutation, to positions scattered across the de-interleaved block, so no two damaged bits land next to each other." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="30" font-size="8" fill="currentColor">on air (burst damages adjacent bits)</text>
  <g stroke="currentColor" stroke-width="1">
    <rect x="20" y="36" width="320" height="20" fill="none"/>
    <rect x="140" y="36" width="60" height="20" fill="currentColor" fill-opacity="0.30" stroke="none"/>
  </g>
  <g stroke="currentColor" stroke-width="0.7" fill="none" stroke-dasharray="2 2">
    <path d="M150 56 L70 92"/><path d="M165 56 L200 92"/><path d="M180 56 L300 92"/><path d="M195 56 L120 92"/>
  </g>
  <text x="20" y="108" font-size="8" fill="currentColor">after de-interleave (errors isolated)</text>
  <g stroke="currentColor" stroke-width="1">
    <rect x="20" y="92" width="320" height="20" fill="none"/>
    <rect x="64" y="92" width="12" height="20" fill="currentColor" fill-opacity="0.30" stroke="none"/>
    <rect x="114" y="92" width="12" height="20" fill="currentColor" fill-opacity="0.30" stroke="none"/>
    <rect x="194" y="92" width="12" height="20" fill="currentColor" fill-opacity="0.30" stroke="none"/>
    <rect x="294" y="92" width="12" height="20" fill="currentColor" fill-opacity="0.30" stroke="none"/>
  </g>
  <text x="360" y="76" font-size="7.5" fill="currentColor">k = 1 + (a·i mod K)</text>
</svg>
<figcaption>The multiplicative permutation sends a contiguous run of on-air errors to scattered positions in the de-interleaved block, converting one uncorrectable burst into several correctable single errors.</figcaption>
</figure>

## The (K, a) rule

The interleaver is defined by a single congruence. For each input index `i = 1..K`, the output index is
`k = 1 + ((a·i) mod K)`; the de-interleaver inverts it by walking `i` and reading back from `k`. Choosing `a`
coprime with `K` guarantees the map is a bijection, so the round-trip is the identity. Each logical channel
carries its own `(K, a)` pair, matched to its block length:

```go
// internal/radio/framing/interleave_tetra.go — ETSI EN 300 392-2 §8.3.1.
const (
    InterleaveKBSCH = 120; InterleaveABSCH = 11  // BSCH               §8.3.1.2
    InterleaveKSCHHD = 216; InterleaveASCHHD = 101 // SCH/HD, BNCH, STCH §8.3.1.4.1
    InterleaveKSCHHU = 168; InterleaveASCHHU = 13  // SCH/HU            §8.3.1.4.3
    InterleaveKSCHF = 432; InterleaveASCHF = 103   // SCH/F            §8.3.1.4.5
)
```

The BSCH interleaves 120 bits with `a = 11`; SCH/HD, BNCH, and STCH share a 216-bit block with `a = 101`;
SCH/HU uses `(168, 13)`; and the full-slot SCH/F uses `(432, 103)`. Because a good multiplier scatters bits
far apart — `a` near a fraction of `K` maximises the minimum separation — the pairs are chosen for the
interleaving distance they produce, not arbitrarily.

## Position in the chain

Interleaving is one step in TETRA's type-1-through-type-5 bit-processing chain. On the transmit side the order
is: encode with the [RCPC code](/reference/tetra-rcpc-code/) (type-1 → type-2 → type-3), **interleave** (type-3
→ type-4), then [scramble](/reference/tetra-scrambler/) (type-4 → type-5). A receiver reverses it: descramble,
de-interleave, then Viterbi-decode. Getting the interleaver right matters because it sits *between* the
convolutional decoder and the channel — an incorrect permutation leaves the decoder facing clustered errors it
was never designed to handle, so the FEC fails even on a clean signal.

## Relevance to SDR

`internal/radio/framing/interleave_tetra.go` implements `BlockInterleaveTetra` / `BlockDeinterleaveTetra` and
exports the four `(K, a)` constants, one per [logical channel](/reference/tetra-logical-channels/). Callers pass
the matching `K` and `a` for the channel they are decoding, so the same tiny permutation function serves the
BSCH, the half-slot signalling channels, and the full-slot SCH/F alike. It is a small piece of code whose
correctness is load-bearing: paired with the RCPC decoder and the scrambler, it is what lets a marginal TETRA
burst survive the fading that would otherwise defeat the convolutional code outright.

## Sources

[^burst]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, on why spreading burst errors lets a random-error code correct them.
[^ileave]: [Interleaving](https://en.wikipedia.org/wiki/Interleaving_(disk_storage)) — Wikipedia, on reordering data so that contiguous damage becomes distributed.
