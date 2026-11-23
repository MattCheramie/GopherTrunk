---
slug: dpmr-channel-coding
title: dPMR Channel Coding
entry_type: term
category: error-correction
description: "dPMR protects its signalling with a CRC (CRC7 or CRC8), a shortened Hamming (12,8) code on every byte, a block interleaver (12 x 6 or 12 x 10) and the x^9 + x^5 + 1 scrambler. Voice frames carry the AMBE+2 vocoder's own Golay coding. There is no convolutional code."
keywords: dPMR channel coding, dPMR FEC, Hamming 12 8, CRC7, CRC8, interleaving, scrambler x9 x5 1, ETSI TS 102 658
aka: ["dPMR FEC", "dPMR channel coding"]
autolink: true
infobox:
  - { label: Code, value: "shortened Hamming (12,8) per byte" }
  - { label: Check, value: "CRC7 (x^7+x^3+1) or CRC8 (x^8+x^2+x+1)" }
  - { label: Interleave, value: "12 x 6 or 12 x 10 block" }
  - { label: Scrambler, value: "x^9 + x^5 + 1, preset all ones" }
  - { label: Spec, value: ETSI TS 102 658 clause 7 }
see_also: [forward-error-correction, dpmr, dpmr-csbk, dpmr-frame-sync, interleaving, cyclic-redundancy-check]
cite_urls:
  - https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf
  - https://en.wikipedia.org/wiki/Hamming_code
---

[dPMR](/reference/dpmr/) codes every signalling field the same way, in four steps (ETSI
TS 102 658 clause 7):[^etsi]

1. **CRC.** A 7-bit CRC (x^7 + x^3 + 1) or an 8-bit one (x^8 + x^2 + x + 1), register preset to
   zero, appended to the information bits.
2. **Hamming (12,8).** The result is cut into bytes and each byte gets four parity bits from a
   shortened Hamming code (generator x^4 + x + 1), which corrects one error per 12-bit block.
3. **Interleave.** The blocks are written down the columns of a 12-row matrix and read out row by
   row, so a burst of errors on the air lands in different blocks.
4. **Scramble.** The interleaved bits are XORed with x^9 + x^5 + 1 started from all ones.

| Field | Information | CRC | Blocks | Interleave | Where |
|-------|-------------|-----|--------|------------|-------|
| Control channel (CCH) | 41 bits | CRC7 | 6 | 12 x 6 | every payload frame |
| Message / header (MI, HI) | 72 bits | CRC8 | 10 | 12 x 10 | headers; Mode 3 beacon messages, sent twice |
| End data | 17 bits | CRC7 | 3, sent twice | none | the end frame |
| SYScast | 17 bits each | CRC7 | 3, sent twice | 12 x 6 | Mode 3 beacon; the three are scrambled as one 216-bit block |
| Appended data | 72 bits | CRC8 | 10 | 12 x 10 | in place of a SYScast, or after a header |

The voice payload is not scrambled and is not coded by any of this: each 72-bit TCH block is an
AMBE+2 vocoder frame with the vocoder's own Golay protection, laid out exactly as DMR's.

## In GopherTrunk

`internal/radio/dpmr/airframe.go` codes the control channel and `beacon_air.go` the message and
SYScast fields. Both are pinned by vectors that the reference decoders' own routines decode
(DSD-FME for the control channel, dsdcc for the message field), compiled outside the repository.
The two references disagree on one point: DSD-FME's CRC8 routine presets its register to all ones,
but DSD-FME never calls it; dsdcc's, which decodes received headers, presets it to zero, and so
does GopherTrunk. This page used to describe a "rate-3/4 convolutional code"; dPMR has none, and
that description came from an earlier model removed in issue #1374.

## Sources

[^etsi]: [ETSI TS 102 658 V2.6.1](https://www.etsi.org/deliver/etsi_ts/102600_102699/102658/02.06.01_60/ts_102658v020601p.pdf) — clause 7 (CRC, Hamming, scrambling, interleaving and the per-field coding) and clause 11 (worked construction of each frame).
