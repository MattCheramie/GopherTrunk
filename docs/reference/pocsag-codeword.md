---
slug: pocsag-codeword
title: POCSAG codeword
entry_type: algorithm
category: paging-data
description: The POCSAG codeword is the 32-bit on-wire unit of the POCSAG paging protocol — one flag bit, 20 data bits, 10 BCH parity bits and an overall even-parity bit — carried in 16-codeword batches behind the 0x7CD215D8 sync word.
keywords: POCSAG codeword, POCSAG batch, frame synchronization codeword, 0x7CD215D8, 0x7A89C197, BCH 31 21, POCSAG address message, RIC, CCIR 584, ITU-R M.584
aka: [POCSAG codeword, "POCSAG batch", "frame synchronisation codeword"]
autolink: true
infobox:
  - { label: On-wire size, value: 32 bits per codeword }
  - { label: FEC, value: "BCH(31,21) + even parity" }
  - { label: Sync word, value: "0x7CD215D8" }
  - { label: Spec, value: "CCIR R.584 / ITU-R M.584" }
see_also: [pocsag, flex-protocol-coding, bch-code, forward-error-correction, frame-synchronization, correlate-access-code]
cite_urls:
  - https://en.wikipedia.org/wiki/POCSAG
  - https://en.wikipedia.org/wiki/BCH_code
---

The **POCSAG codeword** is the 32-bit unit that carries every address and every character of
a [POCSAG](/reference/pocsag/) page: one flag bit, 20 data bits, 10
[BCH](/reference/bch-code/) parity bits, and a trailing overall even-parity bit.[^pocsag]
Codewords never travel alone — they are packed 16 at a time into a *batch* that opens with a
fixed synchronisation codeword, the landmark a receiver locks onto before it reads
anything.[^bch]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A 32-bit POCSAG codeword split into a 1-bit flag, 20 data bits, 10 BCH parity bits and a trailing overall even-parity bit, with a batch above it showing one sync codeword followed by sixteen data codewords arranged as eight frames of two." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="40" height="22" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
  <text x="40" y="35" text-anchor="middle" font-size="8" fill="currentColor">SC</text>
  <rect x="60" y="20" width="380" height="22" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="250" y="35" text-anchor="middle" font-size="8" fill="currentColor">16 codewords = 8 frames × 2</text>
  <text x="20" y="58" font-size="7.5" fill="currentColor">one batch (sync + 16 codewords)</text>
  <rect x="20" y="78" width="34" height="30" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="37" y="97" text-anchor="middle" font-size="8" fill="currentColor">flag</text>
  <rect x="54" y="78" width="200" height="30" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/>
  <text x="154" y="97" text-anchor="middle" font-size="9" fill="currentColor">20 data bits</text>
  <rect x="254" y="78" width="150" height="30" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/>
  <text x="329" y="97" text-anchor="middle" font-size="9" fill="currentColor">10 BCH parity</text>
  <rect x="404" y="78" width="36" height="30" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.1"/>
  <text x="422" y="94" text-anchor="middle" font-size="7" fill="currentColor">even</text>
  <text x="422" y="103" text-anchor="middle" font-size="6.5" fill="currentColor">parity</text>
  <text x="20" y="128" font-size="7.5" fill="currentColor">bit 31 flag · bits 30..11 data · bits 10..1 BCH(31,21) · bit 0 overall even parity</text>
</svg>
<figcaption>A POCSAG batch is one sync codeword and sixteen data codewords; each data codeword is a flag, 20 data bits, a BCH(31,21) parity field and one overall even-parity bit.</figcaption>
</figure>

## How it works

The 31 high bits of the codeword are a **BCH(31,21)** code: 21 information bits (the flag
plus 20 data bits) followed by 10 parity bits, able to correct up to two bit errors per
codeword. GopherTrunk's `framing.BCHDecode31_21` finds the nearest valid codeword; when the
received word is more than two bits from any of them it reports the codeword uncorrectable so
the caller can drop it. The generator polynomial is g(x) = x¹⁰ + x⁹ + x⁸ + x⁶ + x⁵ + x³ + 1,
the constant `0x769` from CCIR Recommendation 584. The 32nd bit is a plain even-parity check
over the 31 above it — a last, cheap way to notice a single residual flip that BCH re-encoding
did not catch.

The flag bit names the codeword's shape. The layout of the 21 information bits differs by
shape:

| Field | Address codeword | Message codeword |
| --- | --- | --- |
| Flag (bit 20 of info) | `0` | `1` |
| Payload | 18-bit address + 2-bit function | 20-bit message field |
| Recovered by | `Address`, `Func` | `MessageBits` |

Two whole codewords are reserved as fixed patterns: the **sync codeword** `0x7CD215D8` (the
Frame Synchronization Codeword, FSC) that begins every batch, and the **idle codeword**
`0x7A89C197` that fills unused frame slots. A receiver treats the idle pattern as "skip" — it
carries the address flag bit but does not correspond to a real pager address.

## Batch and addressing

A batch is `8 frames × 2 codewords` behind the sync word, and the *frame slot* an address
codeword lands in is part of the address. The codeword carries only the 18 high address bits;
the batch layer combines them with the frame index 0..7 to reconstruct the full 21-bit RIC
(Radio Identity Code) the paging network uses — GopherTrunk does this in
`ReconstructRIC`. An address codeword opens a page, subsequent message codewords in the batch
extend it, and the next address or idle codeword (or an uncorrectable one) terminates it. The
2-bit function code (A/B/C/D) traditionally selects tone, numeric or alphanumeric delivery;
GopherTrunk maps function B to a numeric decode and C to alphanumeric, falling back to
alphanumeric for the ambiguous codes.

## In practice

The classic field gotcha is **polarity**. POCSAG is FSK over FM, and an FM discriminator can
hand the bit slicer the whole stream inverted. Rather than force the operator to flip a
switch, GopherTrunk's `Syncer` is polarity-agnostic: while hunting it compares its sliding
32-bit window against `SyncCodeword` *and* against the codeword's bit-inverse
(`^window == SyncCodeword`), records which polarity matched, and XORs every subsequent bit of
the batch with that polarity before parsing. This is the same trick every robust FSK-over-FM
paging decoder uses, and it is what lets a scanner lock POCSAG without knowing the receiver's
sense in advance. The [FLEX codeword](/reference/flex-protocol-coding/) reuses the identical
BCH(31,21) primitive but reverses the bit order, so the two share GopherTrunk's tested BCH
code with a bit-reversal wrapper.

## Sources

[^pocsag]: [POCSAG](https://en.wikipedia.org/wiki/POCSAG) — Wikipedia, on the CCIR Radiopaging Code No. 1, its batch and codeword structure, and the sync/idle codewords.
[^bch]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, on the error-correcting family that protects each POCSAG codeword.
