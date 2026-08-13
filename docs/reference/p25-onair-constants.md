---
slug: p25-onair-constants
title: P25 on-air constants
entry_type: term
category: fn-protocol
description: "The P25 constants that only real on-air signal validates: the NID's BCH generator and bit 64, the TSBK trailer CRC variant, status symbols, C4FM pulse shaping, the Phase 2 sync word, and rotation-recovery sign."
keywords: p25, bch generator, nid, duid, crc-ccitt, tsbk trailer, status symbols, c4fm, inverse sinc, raised cosine, phase 2 sync word, dibit remap, rotation recovery, tia-102
see_also: [bch-code, crc-16-ccitt, p25-frame-sync-word, p25-nid-duid, p25-status-symbols, p25-phase-2-sync-word, p25-tsbk-opcodes, p25-trellis-code, dibit, p25-site-identity-semantics, p25-demod-mode-selection, signal-signatures, encrypted-call-handling]
---

**P25 on-air constants** are the handful of protocol values a decoder can get *almost* right
and still pass every unit test — because a round-trip test that encodes and decodes with the
same wrong constant is self-consistent. Each value below was wrong (or missing) in GopherTrunk
at some point, survived a green test suite, and was only exposed by real captured signal.
Fifteen seconds of ground-truth IQ was worth more than any synthetic harness
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)).

## The NID: BCH generator and bit 64

The 64-bit [NID](/reference/p25-nid-duid/) protects NAC + DUID with a
[BCH(63,16,11) code](/reference/bch-code/). The correct generator polynomial constant is
`0xCD930BDD3B2B` — derived by multiplying the minimal polynomials of α, α³, …, α²¹ over
GF(2⁶) with primitive polynomial `x⁶+x+1`. GopherTrunk shipped with a wrong generator that
decoded its own encoder's output perfectly and nothing on air.

The 64th NID bit is **not an even-parity bit**. It is a fixed per-DUID flag: `0` for
HDU/TDU/TSDU/PDU/TDULC, `1` for LDU1/LDU2
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)).

## The TSBK trailer CRC

The [TSBK](/reference/p25-tsbk-opcodes/) trailer uses the *augmented-codeword* variant of
[CRC-16/CCITT](/reference/crc-16-ccitt/): initial value 0, MSB-first, final XOR `0xFFFF`,
evaluated over all 12 bytes with an expected result of 0. That is **not** CRC-CCITT/FALSE —
same `0x1021` polynomial, different procedure, different result. A decoder checking the wrong
variant rejects every valid TSBK on air while validating its own synthetic ones
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)).

## Status symbols land inside the NID

P25 interleaves a 2-bit status symbol every 70 information bits through *every* data unit,
TSDUs included ([status symbols](/reference/p25-status-symbols/)). The 24-dibit
[frame sync word](/reference/p25-frame-sync-word/) is contiguous — the first status symbol
lands at dibit 35 — so FSW correlation works perfectly while roughly 21 of the 32 NID dibits
are misaligned. "Sync detects, NID never decodes" is the signature
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)).

## C4FM is not a root-raised-cosine pair

Per TIA-102.BAAA, the C4FM transmit shaping is a raised cosine (α = 0.2) multiplied by an
inverse-sinc, and the receive filter is a **sinc** — not a matched RRC pair. Modelling both
ends as RRC is self-consistent in a synthetic harness but leaves ≈ 5.75% residual
intersymbol interference on a real signal, which showed up as a hard error-count ceiling on
NID BCH decode ([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)).

## Phase 2: the sync word and the dibit remap

Two independent constants broke P25 Phase 2 decode, and both passed round-trip tests
([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)):

- **The outbound [Phase 2 sync word](/reference/p25-phase-2-sync-word/) is `0x575D57F7FF`**
  (40 bits / 20 [dibits](/reference/dibit/)), per TIA-102.BBAC and cross-checked against
  independent decoders. GopherTrunk carried a 48-bit value in the 40-bit field; the top byte
  was silently truncated, leaving a pattern that was neither the standard sync nor anything
  transmitted on air. Every test passed because the encoder injected sync from the same
  constant.
- **The two negative-phase symbols decode 2↔3 swapped vs TIA-102** in a plain quadrant
  slicer; the remap `[0, 1, 3, 2]` (its own inverse) must be applied. Phase 1 CQPSK already
  carried this remap; Phase 2 was missing it. Signature: superframes lock, but payload fields
  decode to garbage — e.g. an algorithm ID smeared uniformly across `0x00–0xFF` instead of a
  valid registry value (see [encrypted-call handling](/reference/encrypted-call-handling/)).

## Rotation recovery must add, not subtract

The sync detector defines rotation `k` by `(received + k) mod 4 == canonical`, so recovery
must **add** `k` to each dibit. Computing `(4 − k) & 3` instead is off by exactly 2 for odd
rotations only. The C4FM path can only produce *even* rotations (I/Q conjugation negates
symbols, a +2 shift), which the buggy code handled correctly by coincidence; CQPSK quadrant
slips are 90° — odd — so the bug surfaced only on
[simulcast/CQPSK sites](/reference/p25-demod-mode-selection/)
([#297](https://github.com/MattCheramie/GopherTrunk/issues/297)).

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| FSW correlates, NID never decodes | Weak signal, BCH bug | Status symbols misalign the NID (first one at dibit 35) | Strip status symbols before the NID ([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)) |
| NID BCH errors floor at a fixed count on air, perfect in tests | Marginal RF | RRC modelled at both ends; real RX filter is a sinc | Use the TIA-102.BAAA filters ([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)) |
| Valid-looking TSBKs all fail CRC | Bit errors | Wrong CRC-CCITT variant | Augmented-codeword CRC: init 0, final XOR `0xFFFF`, expects 0 ([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)) |
| Phase 2 never finds sync on air, round-trips fine | RF / timing | 48-bit constant truncated in a 40-bit field | Sync is `0x575D57F7FF` ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)) |
| Phase 2 superframes lock but fields are garbage | Encryption, interference | Missing 2↔3 dibit remap vs TIA-102 | Apply `[0,1,3,2]` after the slicer ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)) |
| Only simulcast/CQPSK sites fail to lock | Simulcast distortion | Odd-rotation recovery subtracted `k` | Recovery adds `k` ([#297](https://github.com/MattCheramie/GopherTrunk/issues/297)) |

The meta-lesson: none of these constants can be validated by a test that encodes and decodes
with the same table. Validate against real captures or an independent implementation — see
[signal signatures](/reference/signal-signatures/) for the diagnostic fingerprints.

## Provenance

- [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) — the first P25 control-channel lock: BCH generator, NID bit 64, TSBK trailer CRC, status symbols, C4FM pulse shaping.
- [#813](https://github.com/MattCheramie/GopherTrunk/issues/813) — Phase 2 encrypted-call metadata: the truncated sync constant and the missing dibit remap.
- [#297](https://github.com/MattCheramie/GopherTrunk/issues/297) — odd-rotation recovery sign bug that only simulcast (CQPSK) sites could expose.
