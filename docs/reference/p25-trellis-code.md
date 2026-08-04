---
slug: p25-trellis-code
title: P25 trellis code
entry_type: algorithm
category: error-correction
description: The P25 trellis code is the table-driven 4-state ½-rate convolutional code from TIA-102.BAAA-A Annex A that protects TSBK and other data blocks, mapping each input dibit through a state transition to an output dibit pair and decoded with Viterbi.
keywords: P25 trellis code, half-rate trellis, 4-state trellis, TIA-102 Annex A, TSBK FEC, constellation table, Viterbi decode, P25 convolutional code
aka: ["P25 1/2-rate trellis", "4-state trellis code", "TIA-102 Annex A trellis"]
autolink: true
infobox:
  - { label: Type, value: 4-state ½-rate trellis (table-driven) }
  - { label: TSBK use, value: 48 info dibits → 98 channel dibits }
  - { label: Decoder, value: Viterbi (hard or soft) }
  - { label: Spec, value: TIA-102.BAAA-A Annex A }
see_also: [p25-tsbk-interleaver, tsbk, viterbi-algorithm, convolutional-code, trellis-coded-modulation, forward-error-correction, p25-phase-2]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Convolutional_code
  - https://en.wikipedia.org/wiki/Viterbi_algorithm
---

The **P25 trellis code** is the 4-state ½-rate [convolutional code](/reference/convolutional-code/)
defined by TIA-102.BAAA-A Annex A that protects the [TSBK](/reference/tsbk/) and other P25
data-block channels.[^conv] It is deliberately **not** the textbook (7,5)-octal convolutional
code: instead of a generator polynomial, it is table-driven — the encoder's state *is* the most
recent input dibit, and each transition emits a two-dibit output pair looked up from a fixed
16-entry constellation table. Decoding is the [Viterbi algorithm](/reference/viterbi-algorithm/),
which finds the most-likely input sequence through the trellis.[^vit]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="One input dibit selects the next state (equal to that dibit) and, together with the current state, indexes a 16-entry constellation table that yields an output pair of a high dibit and a low dibit; a Viterbi decoder walks the resulting trellis to recover the inputs." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="45" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">state (2b)</text>
  <rect x="20" y="80" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="97" text-anchor="middle" font-size="8.5" fill="currentColor">input dibit</text>
  <path d="M90 58 L150 65" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M90 93 L150 78" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="150" y="52" width="120" height="40" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="210" y="70" text-anchor="middle" font-size="8.5" fill="currentColor">states[cur][next]</text>
  <text x="210" y="84" text-anchor="middle" font-size="8" fill="currentColor">→ pairs[idx]</text>
  <path d="M270 72 L330 72" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <rect x="330" y="45" width="110" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">hi dibit</text>
  <rect x="330" y="80" width="110" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="97" text-anchor="middle" font-size="8.5" fill="currentColor">lo dibit</text>
  <text x="150" y="125" font-size="8" fill="currentColor">next state = input dibit · one dibit in → two dibits out (rate ½)</text>
</svg>
<figcaption>Each transition uses the current state and the input dibit to index a 16-entry constellation table, emitting a high/low output dibit pair; the next state is simply the input dibit, giving a memory-1, 4-state machine decoded by Viterbi.</figcaption>
</figure>

## How it works

The encoder holds a state in `{0,1,2,3}` equal to the previous input dibit. For each new input
dibit `d`, the next state is `d`, an index is read from `states[cur][d]`, and the two output
dibits `(hi, lo)` come from `pairs[idx]`. Because one input dibit produces two output dibits,
the code is rate ½. For a TSBK, 48 information dibits (12 bytes) encode to 98 channel dibits:
the 48 data transitions plus one **finisher** transition (input dibit 0) that flushes the
machine back to state 0, giving the decoder a known terminal state. The two spec-locked tables
are the whole code — a 4×4 state/index table and the 16-entry output constellation:

```go
// TIA-102.BAAA-A Annex A Table A.1, from internal/radio/p25/phase1/trellis.go
var trellisStates = [4][4]int{ // [cur][next] → constellation index
    {0, 15, 12, 3}, {4, 11, 8, 7}, {13, 2, 1, 14}, {9, 6, 5, 10},
}
var trellisPairs = [16][2]uint8{ // index → (hi, lo) output dibit
    {0b00, 0b10}, {0b10, 0b10}, {0b01, 0b11}, {0b11, 0b11},
    {0b11, 0b10}, {0b01, 0b10}, {0b10, 0b11}, {0b00, 0b11},
    {0b11, 0b01}, {0b01, 0b01}, {0b10, 0b00}, {0b00, 0b00},
    {0b00, 0b01}, {0b10, 0b01}, {0b01, 0b00}, {0b11, 0b00},
}
```

The decoder runs a hard-decision Viterbi over the 98 channel dibits: at each stage it computes,
for all four possible successor states, the path metric as the accumulated dibit-distance
between the expected output pair and the received pair (distance 0/1/2 by how many of the two
bits differ), keeps the survivor into each state, and back-traces from the flushed terminal
state 0. The surviving path's total metric doubles as a confidence figure — zero means a clean
channel; positive values count corrected dibit errors.

## Variants and what is implemented

P25 defines this **½-rate** trellis for control and short data blocks, and a separate
**¾-rate** trellis (TIA-102.BAAA-A Annex A) for the confirmed [packet-data](/reference/tsbk/)
channel, which packs more information per transition at lower coding gain. GopherTrunk
implements only the ½-rate code; the ¾-rate PDU variant is not present, so confirmed-data PDU
payloads are not trellis-decoded. GopherTrunk also adds a true soft-decision decoder
(`DecodeP25TrellisSoftC`) that takes complex differential samples and uses a magnitude-weighted
correlation branch metric, so an ambiguous near-zero symbol contributes little and a strong one
dominates — information a hard slicer throws away.

## Relevance to SDR

The same tables back two paths: `internal/radio/p25/phase1/trellis.go` for Phase 1 TSBKs, and
the shared `internal/radio/framing/p25_trellis.go` primitives that [P25 Phase 2](/reference/p25-phase-2/)
reuses for MAC PDU channel coding. On air the trellis code is applied *after* the
[block interleaver](/reference/p25-tsbk-interleaver/), so on receive GopherTrunk deinterleaves
the 98 channel dibits first, then Viterbi-decodes them back to the 48 information dibits — the
combination is what lets a scanner recover a control-channel grant through a burst of channel
errors instead of dropping the block.

## Sources

[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the convolutional-coding family the P25 trellis belongs to.
[^vit]: [Viterbi algorithm](https://en.wikipedia.org/wiki/Viterbi_algorithm) — Wikipedia, on the maximum-likelihood trellis decoder used here.
