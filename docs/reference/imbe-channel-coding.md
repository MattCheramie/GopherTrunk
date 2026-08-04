---
slug: imbe-channel-coding
title: IMBE channel coding
entry_type: algorithm
category: voice-coding
description: The IMBE channel coding for P25 Phase 1 wraps the 88 vocoder bits in four Golay(23,12) codewords and three Hamming(15,11) codewords, expanding them to the 144 channel bits carried in each voice subframe.
keywords: IMBE channel coding, P25 Phase 1 FEC, Golay 23 12, Hamming 15 11, u0 u7 vectors, 88 to 144 bits, TIA-102.BABA 7.3, unequal error protection, imbe FEC
aka: [IMBE FEC, "IMBE 4400 channel coding", "u_0..u_7 vectors"]
autolink: true
infobox:
  - { label: Input, value: 88 voice bits }
  - { label: Output, value: 144 channel bits }
  - { label: FEC, value: "4× Golay(23,12) + 3× Hamming(15,11)" }
  - { label: Spec, value: TIA-102.BABA §7.3 }
see_also: [imbe, golay-code, hamming-code, forward-error-correction, imbe-scrambler, imbe-interleaver, imbe-parameter-quantization, p25-phase-1, p25-logical-data-unit]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Binary_Golay_code
  - https://en.wikipedia.org/wiki/Hamming_code
---

**IMBE channel coding** is the [forward error correction](/reference/forward-error-correction/)
layer that turns the 88 information bits an [IMBE](/reference/imbe/) vocoder emits every 20 ms
into the 144 channel bits carried in each [P25 Phase 1](/reference/p25-phase-1/) voice
subframe.[^mbe] It is *unequal* protection: the perceptually critical bits get a strong
[Golay(23,12)](/reference/golay-code/) code, the mid-importance bits a lighter
[Hamming(15,11)](/reference/hamming-code/) code, and the least-sensitive bits ride bare. This
graceful-degradation design (TIA-102.BABA §7.3) is why a weak P25 signal warbles rather than
dropping to silence.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 132" role="img" aria-label="The 88 IMBE voice bits split into eight vectors: four twelve-bit vectors each protected by a Golay(23,12) codeword, three eleven-bit vectors each protected by a Hamming(15,11) codeword, and a final seven-bit vector left with no error correction, together forming 144 channel bits." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="15" y="30" width="69" height="30" fill="currentColor" fill-opacity="0.24"/><text x="49" y="44">u0</text><text x="49" y="54" font-size="6.5">Golay 23</text>
    <rect x="84" y="30" width="69" height="30" fill="currentColor" fill-opacity="0.24"/><text x="118" y="44">u1</text><text x="118" y="54" font-size="6.5">Golay 23</text>
    <rect x="153" y="30" width="69" height="30" fill="currentColor" fill-opacity="0.24"/><text x="187" y="44">u2</text><text x="187" y="54" font-size="6.5">Golay 23</text>
    <rect x="222" y="30" width="69" height="30" fill="currentColor" fill-opacity="0.24"/><text x="256" y="44">u3</text><text x="256" y="54" font-size="6.5">Golay 23</text>
    <rect x="291" y="30" width="45" height="30" fill="currentColor" fill-opacity="0.13"/><text x="313" y="44">u4</text><text x="313" y="54" font-size="6.5">Ham 15</text>
    <rect x="336" y="30" width="45" height="30" fill="currentColor" fill-opacity="0.13"/><text x="358" y="44">u5</text><text x="358" y="54" font-size="6.5">Ham 15</text>
    <rect x="381" y="30" width="45" height="30" fill="currentColor" fill-opacity="0.13"/><text x="403" y="44">u6</text><text x="403" y="54" font-size="6.5">Ham 15</text>
    <rect x="426" y="30" width="21" height="30" fill="none"/><text x="436" y="44">u7</text><text x="436" y="54" font-size="6">7</text>
  </g>
  <text x="231" y="80" text-anchor="middle" font-size="8.5" fill="currentColor">88 info bits (12·4 + 11·3 + 7) → 144 channel bits</text>
</svg>
<figcaption>Eight vectors carry the 88 IMBE bits: u0–u3 in Golay(23,12) codewords, u4–u6 in Hamming(15,11) codewords, and u7 unprotected — 144 channel bits in all.</figcaption>
</figure>

## How it works

The 88 bits are grouped into eight vectors named `u_0` through `u_7`. The four most significant
vectors carry 12 bits each and are encoded as Golay(23,12,7) codewords — 23 bits able to correct
up to 3 errors. The next three carry 11 bits each as Hamming(15,11,3) codewords — 15 bits
correcting a single error. The last vector, `u_7`, is 7 bits with no coding at all, because it
holds the least perceptually sensitive spectral detail. The arithmetic closes exactly:
`12×4 + 11×3 + 7 = 88` information bits become `23×4 + 15×3 + 7 = 144` channel bits.

| Vector | Info bits | Channel bits | Code |
|--------|-----------|--------------|------|
| u_0 … u_3 | 12 each | 23 each | Golay(23,12,7) |
| u_4 … u_6 | 11 each | 15 each | Hamming(15,11,3) |
| u_7 | 7 | 7 | none |

GopherTrunk decodes each vector by nearest-codeword search over a precomputed table (4096 Golay
codewords, 2048 Hamming codewords), which for errors inside the correction radius returns the
unique correct data and a corrected-error count. `u_0` matters most: its Golay data seeds the
[descrambler](/reference/imbe-scrambler/), so the decoder corrects `u_0` first, before it can
trust any of `u_1`–`u_6`. The generator itself is a fixed table — each of the 12 Golay data bits
contributes a fixed 11-bit parity pattern, XOR-accumulated:

```go
// 11-bit parity contribution of each Golay(23,12) data bit, MSB-first.
var golayGenerator = [12]uint16{
    0x63a, 0x31d, 0x7b4, 0x3da, 0x1ed, 0x6cc,
    0x366, 0x1b3, 0x6e3, 0x54b, 0x49f, 0x475,
}
```

## Two different Golay codes in one repo

A subtle trap sits under this page. GopherTrunk carries **two** unrelated Golay/Hamming
implementations, and they are *not* interchangeable. The `internal/radio/framing` package holds
systematic Golay(24,12) and Hamming(15,11) codes used for P25 and DMR *link control* framing. The
vocoder uses the separate `internal/voice/imbe/p25fec.go` code, transcribed from mbelib's `ecc.c`
in the exact bit order real IMBE transmitters use. Although framing's Golay generator list equals
`golayGenerator` shifted by one bit, the two associate generator rows with data bits in *opposite*
order, so they are different codes in practice: a clean real-air IMBE codeword decodes to the
*wrong* data under framing's Golay. That mismatch was the root of GopherTrunk issue #489, verified
against a real P25 voice capture and the mbelib reference decoder. When touching this path, never
reach for the framing package's codec — the vocoder needs the mbelib-order one, and only that one.

## Relevance to SDR

Channel coding is the first thing GopherTrunk's IMBE receiver runs on each 144-bit subframe, after
the [interleaver](/reference/imbe-interleaver/) and [scrambler](/reference/imbe-scrambler/) layers
have been undone. The corrected-error counts it returns feed the frame-repeat logic upstream: an
uncorrectable vector marks a bad frame that the synthesizer replays and fades rather than voicing.
Each P25 Phase 1 [LDU](/reference/p25-logical-data-unit/) carries nine such subframes, so getting
the Golay and Hamming math bit-exact — in the mbelib order, not the framing order — is what lets
real off-air voice decode at all.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the IMBE vocoder family and its role in P25 Phase 1.
[^golay]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, on the (23,12,7) perfect code used for the u_0–u_3 vectors.
[^hamming]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, on the single-error-correcting family used for u_4–u_6.
