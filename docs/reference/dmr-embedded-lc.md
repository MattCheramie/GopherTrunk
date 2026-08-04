---
slug: dmr-embedded-lc
title: DMR embedded LC
entry_type: algorithm
category: error-correction
description: The DMR embedded Link Control is a 72-bit PDU protected by a variable BPTC(128,72) — Hamming(16,11,4) on seven rows, even column parity on the eighth, plus a 5-bit mod-31 checksum — reassembled from the four 32-bit fragments carried by voice bursts B–E.
keywords: DMR embedded LC, BPTC 128 72, embedded link control, Hamming 16 11, column parity, five-bit checksum, mod 31, ETSI TS 102 361-1 Annex B C
aka: ["embedded LC", "embedded link control", "BPTC(128,72)"]
autolink: true
infobox:
  - { label: Code, value: "BPTC(128,72), variable-length" }
  - { label: Rows, value: "7 × Hamming(16,11,4) + column parity" }
  - { label: Check, value: 5-bit mod-31 checksum }
  - { label: Spec, value: ETSI TS 102 361-1 Annex B/C }
see_also: [bptc, dmr-emb, dmr-full-link-control, dmr-voice-superframe, hamming-code, cyclic-redundancy-check, interleaving, forward-error-correction, dmr]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Hamming_code
---

The **DMR embedded LC** is the [Full Link Control](/reference/dmr-full-link-control/) PDU a voice
call carries *inside* its own signalling, protected by a variable-length **BPTC(128,72)** product
code.[^wiki] A [voice superframe](/reference/dmr-voice-superframe/) cannot spare a whole burst for
link control, so it spreads a 72-bit LC block across the 32-bit [EMB](/reference/dmr-emb/)
fragments of bursts B, C, D and E; the receiver concatenates those four fragments into 128 channel
bits and runs a [BPTC](/reference/bptc/)-style two-dimensional decode to recover the 72 LC bits.[^ham]

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 200" role="img" aria-label="An 8-row by 16-column matrix: rows 0 through 6 are each a Hamming(16,11,4) codeword with data in the left eleven columns and parity in the right five, row 7 holds even column parity over the rows above, and five cells in column 10 of rows 2 through 6 hold a 5-bit checksum." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.9" fill="none">
    <rect x="30" y="20" width="220" height="140"/>
    <line x1="30" y1="40" x2="250" y2="40"/><line x1="30" y1="60" x2="250" y2="60"/><line x1="30" y1="80" x2="250" y2="80"/><line x1="30" y1="100" x2="250" y2="100"/><line x1="30" y1="120" x2="250" y2="120"/><line x1="30" y1="140" x2="250" y2="140"/>
    <line x1="181" y1="20" x2="181" y2="160"/>
  </g>
  <rect x="30" y="20" width="151" height="120" fill="currentColor" fill-opacity="0.08"/>
  <rect x="181" y="20" width="69" height="120" fill="currentColor" fill-opacity="0.20"/>
  <text x="105" y="14" text-anchor="middle" font-size="8" fill="currentColor">11 data cols</text>
  <text x="215" y="14" text-anchor="middle" font-size="8" fill="currentColor">5 parity</text>
  <rect x="30" y="140" width="220" height="20" fill="currentColor" fill-opacity="0.20"/>
  <text x="140" y="154" text-anchor="middle" font-size="8" fill="currentColor">row 7 · column parity</text>
  <text x="266" y="80" font-size="7.5" fill="currentColor" transform="rotate(90 266 80)">rows 0–6: Hamming(16,11,4)</text>
  <text x="140" y="182" text-anchor="middle" font-size="7.5" fill="currentColor">col 10 of rows 2–6 → 5-bit mod-31 checksum</text>
</svg>
<figcaption>The 128 channel bits deinterleave into an 8×16 matrix: seven Hamming rows, an eighth column-parity row, and five checksum cells; the 72 LC bits are read from the data columns.</figcaption>
</figure>

## The code

GopherTrunk implements the decode in `internal/radio/framing/embedded_bptc.go`, matching the
de-facto MMDVM reference verified against ETSI Annex B/C:

1. **Deinterleave** the 128 on-air bits into an 8-row × 16-column matrix with `data[b] = onair[a]`,
   `b += 16` wrapping `b -= 127` when `b > 127`.
2. **Row Hamming.** Rows 0–6 are each a shortened Hamming(16,11,4) codeword — data in columns
   0–10, parity in columns 11–15. The code is SEC-DED: `decodeEmb16114` corrects one bit error per
   row (by trial flip until the 5-bit syndrome clears) and detects two; any uncorrectable row fails
   the block.
3. **Column parity.** Row 7 holds even parity down each column, so every column of the full 8-row
   matrix must XOR to 0 — a second [interleaving](/reference/interleaving/) dimension that catches
   errors a row pass alone misses.
4. **5-bit checksum.** The 72 LC bits are read from rows 0–6 columns 0–10, *except* that column 10
   of rows 2–6 carries the five bits of a checksum. That checksum is the sum of the nine LC octets
   modulo 31 (`embFiveBit`); it must match the recovered LC.

The **72 LC bits** come out of `extractEmbLC` in the order the `embLCRanges` table defines — rows
0–6 columns 0–10, but only columns 0–9 for rows 2–6. `DecodeEmbeddedLC` returns the LC and a
corrected-bit count, or −1 if any row was uncorrectable, the column parity failed, *or* the
checksum did not verify.

## Why three checks

The three integrity layers are complementary. The row Hamming codes locate and flip isolated bit
errors; the column parity crosses them so a burst that survives one row lands under a column that
can still catch it; and the mod-31 checksum is an independent end-to-end verifier that guards
against a Hamming pass mis-correcting into a valid-but-wrong codeword. Only when all three agree
does GopherTrunk treat the 72 bits as a genuine LC. `ReassembleEmbeddedLC`
(`internal/radio/dmr/emb.go`) then parses those bits as a Full LC — and because this whole block is
the integrity gate, the [EMB](/reference/dmr-emb/)'s own QR(16,7) FEC and the per-context
[RS(12,9)](/reference/dmr-rs-12-9/) parity can be deferred without admitting bad LCs.

## Relevance to SDR

The embedded LC is what lets GopherTrunk *label* a DMR voice call it is already decoding. Burst A's
sync word is BS-sourced and identical on both timeslots, so it cannot say which talkgroup a call
belongs to; the embedded LC's destination and source addresses can, and its successful BPTC + CRC
decode is authoritative enough that the interleaved voice decoder locks its TDMA cadence the moment
one appears (see `resolveAndSlice`). Recovering it correctly from four fragments spread across four
bursts — deinterleave, two Hamming/parity dimensions, and the mod-31 check — is the difference
between reporting a talkgroup and a source radio versus hearing anonymous audio.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its embedded link control.
[^ham]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, on the row component code of the embedded BPTC.
