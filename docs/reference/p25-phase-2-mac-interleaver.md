---
slug: p25-phase-2-mac-interleaver
title: P25 Phase 2 MAC interleaver
entry_type: algorithm
category: error-correction
description: The P25 Phase 2 MAC interleaver is the block interleaver that scrambles the position order of a trellis-coded MAC burst so a channel burst error spreads across the Viterbi decoder's input instead of landing as one long unrecoverable run; GopherTrunk models it as a 2-row write-row-major, read-column-major permutation.
keywords: P25 Phase 2 MAC interleaver, block interleaver, 2-row interleaver, burst error spreading, trellis input, deinterleave MAC burst, TIA-102.BBAC
aka: ["MAC burst interleaver", "Phase 2 block interleaver"]
autolink: true
infobox:
  - { label: Type, value: 2-row block interleaver }
  - { label: Rule, value: write row-major, read column-major }
  - { label: Applied to, value: trellis-coded MAC burst }
  - { label: Spec, value: TIA-102.BBAC }
see_also: [interleaving, p25-mac-pdu, p25-trellis-code, viterbi-algorithm, p25-phase-2, pn44-scrambler, p25-reed-solomon]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
  - https://en.wikipedia.org/wiki/Interleaving_(disk_storage)
---

The **P25 Phase 2 MAC interleaver** is the block [interleaver](/reference/interleaving/) that reorders
a trellis-coded [MAC burst](/reference/p25-mac-pdu/) before it goes on air.[^burst] Its purpose is the
classic one: a physical burst error on the channel — a fade, a click, an interfering hit — arrives as a
contiguous run of corrupted dibits, and a run is exactly what the
[Viterbi](/reference/viterbi-algorithm/) trellis decoder handles worst. By permuting the burst at the
transmitter and un-permuting it at the receiver, adjacent on-air errors are pulled apart so they land as
*isolated* errors spread across the trellis decoder's input, where the code can absorb them.[^ileave]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Dibits are written row-major into a two-row matrix and read out column-major, so an on-air burst that hits several consecutive read-order positions maps back to dibits that were originally half a burst apart, spreading the errors across the trellis decoder's input." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="26" font-size="8" fill="currentColor">write row-major →</text>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="20" y="34" width="220" height="44"/>
    <line x1="64" y1="34" x2="64" y2="78"/><line x1="108" y1="34" x2="108" y2="78"/><line x1="152" y1="34" x2="152" y2="78"/><line x1="196" y1="34" x2="196" y2="78"/>
    <line x1="20" y1="56" x2="240" y2="56"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="42" y="50">0</text><text x="86" y="50">1</text><text x="130" y="50">2</text><text x="174" y="50">3</text><text x="218" y="50">4</text>
    <text x="42" y="72">5</text><text x="86" y="72">6</text><text x="130" y="72">7</text><text x="174" y="72">8</text><text x="218" y="72">9</text>
  </g>
  <text x="20" y="104" font-size="8" fill="currentColor">read column-major → 0 5 1 6 2 7 3 8 4 9</text>
  <rect x="20" y="112" width="72" height="20" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
  <text x="56" y="126" text-anchor="middle" font-size="7.5" fill="currentColor">burst hits 5,1</text>
  <path d="M56 132 L56 148" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="20" y="162" font-size="8" fill="currentColor">→ originally-adjacent errors land half a burst apart at the trellis input</text>
</svg>
<figcaption>Writing row-major into a 2-row matrix and reading column-major interleaves the dibits; a burst that corrupts consecutive read-order positions un-maps to originally distant dibits, so the trellis sees isolated errors.</figcaption>
</figure>

## The permutation

GopherTrunk models the MAC interleaver as a **2-row block interleaver**. The dibits are written into a
`2 × N` matrix in row-major order (fill row 0 left to right, then row 1), and read back out in
column-major order (column 0's two entries, then column 1's, and so on). The read index for matrix
position `(row r, column c)` is `c·2 + r`, and the de-interleaver applies the exact inverse — read
column-major back into the row-major grid — so any originally-adjacent pair of dibits ends up half a
burst apart. A slice whose length is not a multiple of the 2 rows is returned **unchanged** rather than
being partially permuted, so a malformed input is a no-op instead of a corrupting operation. Because
`InterleaveMACBurst` and `DeinterleaveMACBurst` are exact inverses, a synthetic fixture round-trips
cleanly whether or not the modelled permutation matches the standard's.

That inverse-pair property is what lets the code carry an explicit caveat: **the exact permutation table
is not in the repository's spec PDFs.** The 2-row model is the project's working assumption — it delivers
the burst-spreading the code exists to provide — and a spec correction is a single local change that both
the encoder and decoder pick up together, with no downstream code needing to know.

## Ordering in the FEC chain

Interleaving is undone *before* trellis decoding and *after* descrambling. The receiver's Phase 2 chain
is: descramble the raw channel dibits with the [PN44 scrambler](/reference/pn44-scrambler/), then
de-interleave, then run the [trellis](/reference/p25-trellis-code/) Viterbi decoder, then check the outer
[Reed-Solomon](/reference/p25-reed-solomon/) code. Placing the de-interleave between the descrambler and
the trellis is precisely what turns a channel burst back into the isolated-error pattern the convolutional
decoder was designed for.

The soft-decision path needs the same permutation applied to a parallel array. `DeinterleaveMACBurstC`
de-interleaves a `complex64` slice — the per-dibit differential soft sample — with the identical
permutation as the hard dibits, keeping each soft reliability value aligned with its dibit through the
interleaver so the soft Viterbi sees consistent inputs.

## Relevance to SDR

`internal/radio/framing/p25p2_interleave.go` implements `InterleaveMACBurst`, `DeinterleaveMACBurst`, and
the soft `DeinterleaveMACBurstC`. Getting the interleaver right is part of what lets a real Phase 2 MAC
burst survive a channel hit and still decode to a valid opcode instead of being discarded — the same
burst-resilience motive that recurs across digital radio. The spec is TIA-102.BBAC, with the exact table
flagged as a working model pending a spec figure.

## Sources

[^burst]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, on why contiguous channel errors are the hard case and how interleaving disperses them.
[^ileave]: [Interleaving](https://en.wikipedia.org/wiki/Interleaving_(disk_storage)) — Wikipedia, on the block-interleaving read/write permutation.
