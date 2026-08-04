---
slug: p25-tsbk-interleaver
title: P25 TSBK interleaver
entry_type: algorithm
category: error-correction
description: The P25 TSBK interleaver is the 98-dibit block permutation (TIA-102.BAAA-A Annex A) applied around the trellis code, scattering a contiguous on-air error burst into isolated single-dibit errors the Viterbi decoder can still correct.
keywords: P25 TSBK interleaver, block interleaver, 98 dibit permutation, deinterleave, TIA-102 Annex A, burst error, trellis interleave, P25 data block
aka: ["TSBK block interleaver", "98-dibit interleaver", "P25 data-block interleaver"]
autolink: true
infobox:
  - { label: Block size, value: 98 dibits (196 bits) }
  - { label: Sits, value: around the ½-rate trellis code }
  - { label: Purpose, value: break contiguous error bursts }
  - { label: Spec, value: TIA-102.BAAA-A Annex A }
see_also: [p25-trellis-code, tsbk, interleaving, viterbi-algorithm, forward-error-correction, p25-tsbk-opcodes]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
  - https://en.wikipedia.org/wiki/Interleaving_(disk_storage)
---

The **P25 TSBK interleaver** is the 98-dibit block [interleaving](/reference/interleaving/)
permutation from TIA-102.BAAA-A Annex A that P25 applies around the
[trellis code](/reference/p25-trellis-code/) protecting a [TSBK](/reference/tsbk/) or other
data block.[^burst] Its only job is to spread the dibits out: a contiguous fade or click on the
channel corrupts a run of *adjacent* on-air dibits, and reordering those dibits before decode
scatters that run into isolated single-dibit errors spaced far apart — exactly the error
pattern the [Viterbi](/reference/viterbi-algorithm/) trellis decoder is good at correcting.[^int]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A contiguous burst of corrupted channel dibits, after deinterleaving by the fixed 98-entry permutation, becomes several isolated single-dibit errors spread across the coding-order block, which the Viterbi decoder then corrects." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="26" font-size="8.5" fill="currentColor">channel order (as received)</text>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="20" y="32" width="18" height="18"/><rect x="38" y="32" width="18" height="18"/>
    <rect x="56" y="32" width="18" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="74" y="32" width="18" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="92" y="32" width="18" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="110" y="32" width="18" height="18"/><rect x="128" y="32" width="18" height="18"/>
    <rect x="146" y="32" width="18" height="18"/><rect x="164" y="32" width="18" height="18"/>
  </g>
  <text x="60" y="66" font-size="7.5" fill="currentColor">← burst →</text>
  <path d="M240 41 L280 41" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="300" y="26" font-size="8.5" fill="currentColor">deinterleave → coding order</text>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="290" y="32" width="16" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="306" y="32" width="16" height="18"/><rect x="322" y="32" width="16" height="18"/>
    <rect x="338" y="32" width="16" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="354" y="32" width="16" height="18"/><rect x="370" y="32" width="16" height="18"/>
    <rect x="386" y="32" width="16" height="18" fill="currentColor" fill-opacity="0.4"/>
    <rect x="402" y="32" width="16" height="18"/><rect x="418" y="32" width="16" height="18"/>
  </g>
  <text x="20" y="96" font-size="8" fill="currentColor">98-entry fixed permutation · adjacent errors become spaced-out singles the Viterbi decoder mops up</text>
</svg>
<figcaption>The deinterleaver reverses a fixed 98-entry permutation, so a run of adjacent corrupted channel dibits lands as widely separated single errors in coding order — the pattern the trellis Viterbi decoder corrects best.</figcaption>
</figure>

## How it works

The interleaver is a pure permutation of positions — no arithmetic, just a lookup table.
GopherTrunk stores two 98-entry tables that are inverses of each other: `tsbkInterleavePerm`
for the encoder (`channel[i] = coding[perm[i]]`) and `tsbkDeinterleavePerm` for the decoder
(`coding[i] = channel[perm[i]]`). A build-time test asserts they invert. The permutation is a
column/row transpose of the 98-dibit block; laid out as consecutive dibits it reads:

```go
// TIA-102.BAAA-A Annex A, from internal/radio/p25/phase1/interleaver.go
// Encoder side: channel[i] = coding[tsbkInterleavePerm[i]].
var tsbkInterleavePerm = [98]int{
    0, 1, 8, 9, 16, 17, 24, 25, 32, 33, 40, 41, 48, 49, 56, 57, 64, 65, 72, 73, 80, 81, 88, 89, 96, 97,
    2, 3, 10, 11, 18, 19, 26, 27, 34, 35, 42, 43, 50, 51, 58, 59, 66, 67, 74, 75, 82, 83, 90, 91,
    4, 5, 12, 13, 20, 21, 28, 29, 36, 37, 44, 45, 52, 53, 60, 61, 68, 69, 76, 77, 84, 85, 92, 93,
    6, 7, 14, 15, 22, 23, 30, 31, 38, 39, 46, 47, 54, 55, 62, 63, 70, 71, 78, 79, 86, 87, 94, 95,
}
```

The stride of 8 between the first row's entries (`0, 8, 16, …`) is the interleaver's depth: two
dibits that were adjacent in coding order end up separated by four positions on air, and vice
versa, which is what converts a physical burst into scattered singles.

## In practice

The interleaver only makes sense as one half of a pair with the trellis code, and order
matters. On the encode side the information dibits are trellis-encoded to 98 channel dibits
*first*, then interleaved for transmission; on receive GopherTrunk reverses that — `DeinterleaveTSBK`
restores coding order, then `DecodeTrellis` runs Viterbi. Deinterleaving *after* the trellis
decode, or skipping it, would leave the burst intact and the Viterbi decoder facing a dense
cluster of adjacent errors it cannot resolve. The permutation is cross-verified in the source
against three independent P25 implementations (kchmck/p25.rs, OP25, DSDPlus), because a single
transposed index silently corrupts every block while synthetic round-trips still pass.

## Relevance to SDR

`internal/radio/p25/phase1/interleaver.go` implements `InterleaveTSBK` / `DeinterleaveTSBK`,
the deinterleave step that precedes the [trellis](/reference/p25-trellis-code/) Viterbi decoder
on GopherTrunk's control-channel path. Together they are what lets a scanner pull a
[channel grant](/reference/channel-grant/) [TSBK opcode](/reference/p25-tsbk-opcodes/) out of a
marginal signal — the interleaver earns its keep precisely on the fading, bursty channels where
losing a control block means missing a call. The same burst-breaking idea recurs throughout
digital radio wherever a short block code must survive channel bursts.

## Sources

[^burst]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, on why interleaving lets short codes survive burst errors.
[^int]: [Interleaving](https://en.wikipedia.org/wiki/Interleaving_(disk_storage)) — Wikipedia, on reordering symbols to spread out contiguous errors.
