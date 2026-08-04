---
slug: imbe-interleaver
title: IMBE interleaver
entry_type: algorithm
category: voice-coding
description: The IMBE interleaver is the 144-bit permutation that scatters each coded vector's bits across a P25 Phase 1 voice subframe, so a localised burst error spreads over several Golay and Hamming codewords instead of destroying one.
keywords: IMBE interleaver, 144-bit permutation, deinterleave, burst error spreading, P25 Phase 1 interleave, TIA-102.BABA 7.5, DSD iW iX iY iZ, issue 489
aka: [IMBE deinterleaver, "IMBE 4400 interleaver", "§7.5 interleaver"]
autolink: true
infobox:
  - { label: Size, value: 144-bit permutation }
  - { label: Purpose, value: spread burst errors }
  - { label: Receive order, value: "deinterleave → descramble → FEC" }
  - { label: Spec, value: TIA-102.BABA §7.5 }
see_also: [interleaving, imbe-channel-coding, imbe-scrambler, imbe, forward-error-correction, p25-logical-data-unit]
cite_urls:
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
---

The **IMBE interleaver** is the final channel-coding step (TIA-102.BABA §7.5) applied to a
[P25 Phase 1](/reference/imbe/) voice subframe: a fixed permutation that reorders all 144 channel
bits so the bits of any one [FEC](/reference/imbe-channel-coding/) codeword end up scattered across
the whole burst rather than sitting adjacent.[^int] Its purpose is the classic
[interleaving](/reference/interleaving/) trick — a fade or noise burst that would wipe out a run of
consecutive bits instead knocks out a single bit from each of several Golay and Hamming codewords,
where the per-vector FEC can correct it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 140" role="img" aria-label="Adjacent channel bits of a single Golay codeword, shown clustered at the top, are permuted by the interleaver so they land in scattered positions across the 144-bit on-air burst; a burst error striking a contiguous span of the transmitted bits therefore touches only one bit of each codeword." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="iar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor">
    <text x="14" y="26">one codeword's bits</text>
    <text x="14" y="120">on-air burst (interleaved)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="40" y="32" width="16" height="16" fill="currentColor" fill-opacity="0.30"/>
    <rect x="56" y="32" width="16" height="16" fill="currentColor" fill-opacity="0.30"/>
    <rect x="72" y="32" width="16" height="16" fill="currentColor" fill-opacity="0.30"/>
    <rect x="88" y="32" width="16" height="16" fill="currentColor" fill-opacity="0.30"/>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="48" y1="48" x2="120" y2="92" marker-end="url(#iar)"/>
    <line x1="64" y1="48" x2="250" y2="92" marker-end="url(#iar)"/>
    <line x1="80" y1="48" x2="330" y2="92" marker-end="url(#iar)"/>
    <line x1="96" y1="48" x2="420" y2="92" marker-end="url(#iar)"/>
  </g>
  <line x1="20" y1="96" x2="448" y2="96" stroke="currentColor" stroke-width="1.1"/>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="112" y="92" width="16" height="9" fill="currentColor" fill-opacity="0.30"/>
    <rect x="242" y="92" width="16" height="9" fill="currentColor" fill-opacity="0.30"/>
    <rect x="322" y="92" width="16" height="9" fill="currentColor" fill-opacity="0.30"/>
    <rect x="412" y="92" width="16" height="9" fill="currentColor" fill-opacity="0.30"/>
  </g>
</svg>
<figcaption>The interleaver scatters each codeword's adjacent bits across the 144-bit burst, so a contiguous burst error strikes only one bit per codeword — recoverable by the per-vector FEC.</figcaption>
</figure>

## How it works

The permutation is a bijection of the integers 0–143: `imbeDeinterleave[v]` gives the on-air bit
index that supplies vector-order bit `v`. GopherTrunk derives it from DSD's published P25 Phase 1
schedule — the `iW`/`iX`/`iY`/`iZ` tables that map each on-air voice dibit's two bits into the
`imbe_fr[row][col]` vector array — remapped onto this package's `u_0`–`u_7` vector layout. A slice
of the resulting table shows the scatter:

```go
var imbeDeinterleave = [144]int{
    132, 127, 120, 115, 108, 103, 96, 91, 84, 79, 72, 67,
    60, 55, 48, 43, 36, 31, 24, 19, 12, 7, 0, 126,
    // … 120 further entries, one per channel bit …
}
```

Consecutive vector-order positions map to widely separated on-air positions (132, 127, 120, 115,
…), which is exactly the spread that turns an on-air burst into isolated single-bit hits. The DSD
schedule expresses the same permutation over the 72 on-air voice dibits: each dibit's two bits are
placed into `imbe_fr[iW[j]][iX[j]]` and `imbe_fr[iY[j]][iZ[j]]`, and mapping those row/column
coordinates onto this package's `u_0`–`u_7` layout reproduces the table above exactly. Because the
table is a pure permutation, `Deinterleave` and `Interleave` are the same mapping applied in
opposite directions — no separate inverse table is stored, so the two can never drift apart.

## The bijection guard

A permutation table is only correct if it is a genuine bijection — every output index used exactly
once. A table that mapped two vector bits to the same on-air bit (or left a bit unassigned) would
silently corrupt every voice frame while still *looking* plausible. GopherTrunk refuses to ship
such a table: an `init()` guard walks the array at package load, panics if any entry is out of
range, and panics if any on-air index is claimed twice. Failing loudly at startup is preferable to
shipping a subtly-wrong codec that mis-decodes only on real signals. A companion test,
`TestInterleavePermutationIsBijection`, pins the same property in CI.

## The receive order lesson

Interleaving is undone *first* in the receive chain, and the order is not negotiable:
deinterleave → [descramble](/reference/imbe-scrambler/) → per-vector FEC decode. The descrambler's
`u_0` seed is only valid once the bits are back in vector order, so deinterleaving must precede it.
Omitting the deinterleave step entirely was GopherTrunk issue #489: the per-vector FEC then decoded
interleaved garbage and the decoder reported roughly **100% uncorrectable LDUs** on real signals,
even though every synthetic round-trip test passed — a self-consistent bug, because the encode side
had made the same omission. Reinstating the DSD-sourced permutation, verified against
mbelib/DSD-faithful reference vectors, made a real P25 voice subframe decode bit-for-bit.

## Relevance to SDR

Every P25 Phase 1 [LDU](/reference/p25-logical-data-unit/) carries nine 144-bit IMBE subframes,
and each one enters GopherTrunk's decoder through this permutation. The interleaver contributes no
error correction of its own — it only rearranges the burst structure so the Golay and Hamming
stages can do their work — but its correctness is a precondition for everything downstream. A
wrong permutation, or the wrong receive order, presents the FEC with scrambled input and the call
never decodes, which is precisely the failure #489 recorded.

## Sources

[^int]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, on interleaving as a means of spreading burst errors across independent codewords.
