---
slug: lora-diagonal-interleaver
title: LoRa diagonal interleaver
entry_type: algorithm
category: error-correction
description: The LoRa diagonal interleaver spreads each Hamming codeword across the symbols of a block along a rotating offset, so a burst that corrupts one symbol touches only one bit of each codeword; it sits between Gray mapping and the Hamming(4+CR,4) FEC in the LoRa PHY chain.
keywords: LoRa interleaver, diagonal interleaver, offset interleaver, LoRa Gray mapping, Hamming 4 CR, coding rate 4/5 4/8, spreading factor, LoRa FEC, Semtech AN1200.22
aka: [LoRa interleaver, "diagonal interleaver", "offset interleaver"]
autolink: true
infobox:
  - { label: Type, value: Diagonal (offset) interleaver }
  - { label: Block, value: "(4+CR) symbols × PPM bits" }
  - { label: FEC, value: "Hamming(4+CR, 4)" }
  - { label: Spec, value: "Semtech AN1200.22" }
see_also: [lora, interleaving, gray-code, hamming-code, forward-error-correction, lora-whitening]
cite_urls:
  - https://en.wikipedia.org/wiki/LoRa
  - https://en.wikipedia.org/wiki/Hamming_code
---

The **LoRa diagonal interleaver** is the bit-permutation that spreads each forward-error-correction
codeword across all the symbols of a block along a rotating offset, so a channel burst that
wipes out one whole symbol damages only a *single bit* of each codeword — exactly the pattern
the [Hamming code](/reference/hamming-code/) around it can repair.[^lora] It is one link in the
[LoRa](/reference/lora/) physical-layer chain: Gray mapping, then this
[interleaving](/reference/interleaving/), then Hamming(4+CR, 4) coding.[^ham]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A grid whose rows are codewords and columns are symbols; a diagonal band shows that bit k of codeword r is placed into symbol k at bit position (r plus k) modulo PPM, so consecutive codewords are staggered by one bit position across the symbols." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.9" fill="none">
    <rect x="60" y="24" width="220" height="110"/>
    <line x1="104" y1="24" x2="104" y2="134"/><line x1="148" y1="24" x2="148" y2="134"/><line x1="192" y1="24" x2="192" y2="134"/><line x1="236" y1="24" x2="236" y2="134"/>
    <line x1="60" y1="46" x2="280" y2="46"/><line x1="60" y1="68" x2="280" y2="68"/><line x1="60" y1="90" x2="280" y2="90"/><line x1="60" y1="112" x2="280" y2="112"/>
  </g>
  <text x="170" y="16" text-anchor="middle" font-size="8" fill="currentColor">symbols (columns) →</text>
  <text x="30" y="82" text-anchor="middle" font-size="8" fill="currentColor" transform="rotate(-90 30 82)">codewords ↓</text>
  <g fill="currentColor" fill-opacity="0.28"><rect x="60" y="24" width="44" height="22"/><rect x="104" y="46" width="44" height="22"/><rect x="148" y="68" width="44" height="22"/><rect x="192" y="90" width="44" height="22"/><rect x="236" y="112" width="44" height="22"/></g>
  <text x="300" y="70" font-size="8" fill="currentColor">symbol[k] bit ((r+k) mod PPM)</text>
  <text x="300" y="86" font-size="8" fill="currentColor">   = codeword[r] bit k</text>
  <text x="300" y="108" font-size="7.5" fill="currentColor">each codeword walks a diagonal</text>
</svg>
<figcaption>Bit k of codeword r lands in symbol k at bit position (r+k) mod PPM; successive codewords are staggered by one position, so their bits trace diagonals and no symbol holds two bits of the same codeword.</figcaption>
</figure>

## How it works

A LoRa interleaver block is a rectangle: **PPM** rows (the spreading factor — or SF−2 in a
reduced-rate block) and **(4+CR)** columns, where CR is the coding rate 1..4. GopherTrunk's
`Interleave` and `Deinterleave` are exact inverses built on one rule:

> `symbol[k]` bit `((r+k) mod ppm)` = `codeword[r]` bit `k`

That `+k` offset per column is what makes it *diagonal* rather than a plain transpose: as the
column index `k` advances, the bit is rotated one more position around the symbol, so a single
codeword's bits are smeared across every symbol at staggered positions. The payoff is fading
resilience — corrupting one received symbol removes at most one bit from any codeword, leaving
the per-codeword Hamming decoder a single-error problem it can solve.

Around the interleaver sit two more transforms, all inverse pairs so the whole chain is
round-trip testable:

| Stage | TX direction | RX direction |
| --- | --- | --- |
| Gray mapping | `v ^ (v>>1)` | cumulative XOR |
| Diagonal interleave | `Interleave` | `Deinterleave` |
| FEC | `HammingEncode4` | `HammingDecode4` |

The [Gray code](/reference/gray-code/) mapping ensures adjacent chirp frequencies differ by one
bit, so a small frequency slip in demodulation costs a single bit rather than several.

## Coding rates and the Hamming code

The FEC is a family of `Hamming(4+CR, 4)` codes selected by the coding-rate field, carrying four
data bits per codeword and CR parity bits:

| CR | Rate | Parity | Capability |
| --- | --- | --- | --- |
| 1 | 4/5 | 1 (even parity) | detect only |
| 2 | 4/6 | 2 | detect |
| 3 | 4/7 | 3 (Hamming(7,4)) | single-error correct |
| 4 | 4/8 | 4 (Hamming(7,4) + overall) | correct + double-error detect |

At CR 3 and 4, GopherTrunk computes a 3-bit syndrome from the parity equations and maps it to the
flipped bit position to correct it; CR 4 adds an overall-parity bit so a corrected single error
can be told apart from an uncorrectable double error. The explicit **header** is always sent at
the most robust rate, 4/8, because a receiver must decode it — to learn the payload length,
coding rate and whether a payload CRC follows — before it knows the payload's own coding rate.

## Calibration caveat

As with the rest of LoRa's reverse-engineered PHY, GopherTrunk implements these transforms as a
self-consistent, round-trip-verified chain and flags exact bit-level interoperability with
Semtech silicon — the interleaver's rotation sign and the codeword bit order — as a calibration
step gated behind captured golden vectors, not something assumed correct from the datasheet.

## Sources

[^lora]: [LoRa](https://en.wikipedia.org/wiki/LoRa) — Wikipedia, on the LoRa CSS physical layer, its interleaving and coding rates.
[^ham]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, on the single-error-correcting block code the LoRa FEC is built from.
