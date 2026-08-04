---
slug: dmr-csbk-crc
title: DMR CSBK CRC
entry_type: term
category: error-correction
description: The DMR CSBK CRC is a 16-bit CRC-CCITT over a control block's leading 80 bits, with a fixed 0x5A5A mask XORed onto the checksum so a receiver only accepts a burst that is both intact and actually a CSBK.
keywords: DMR CSBK CRC, CRC-16 CCITT, 0x5A5A mask, ETSI TS 102 361-1, CRC mask, control block CRC, block type discrimination
aka: ["CSBK CRC", "CSBK CRC-16", "CRC mask 0x5A5A"]
autolink: true
infobox:
  - { label: Polynomial, value: CRC-CCITT (0x1021) }
  - { label: Init, value: "0x0000" }
  - { label: Covers, value: leading 80 bits (10 octets) }
  - { label: Mask, value: "XOR 0x5A5A (Table B.21)" }
see_also: [csbk, cyclic-redundancy-check, bptc, dmr-tier-3, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236101/
---

The **DMR CSBK CRC** is the 16-bit [cyclic redundancy check](/reference/cyclic-redundancy-check/)
that closes every [CSBK](/reference/csbk/) (Control Signalling Block).[^wiki] It is a plain
CRC-CCITT (polynomial `0x1021`, initial value `0x0000`) computed over the block's leading 80
bits, but with one twist that trips up naive implementations: before transmission a fixed
**`0x5A5A`** mask is XORed onto the 16-bit checksum, per ETSI TS 102 361-1 §B.3.11, Table
B.21.[^etsi] A receiver recomputes the CRC over the recovered bits, applies the same mask,
and accepts the block only on an exact match.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A 96-bit CSBK information block: the first 80 bits are the leader, opcode, feature ID and payload, over which a CRC-CCITT is computed with initial value zero; the resulting 16-bit checksum is XORed with the constant 0x5A5A and stored in the final 16 bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="300" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="170" y="47" text-anchor="middle" font-size="8.5" fill="currentColor">bits 0–79 · leader + opcode + FID + payload</text>
  <rect x="320" y="30" width="120" height="26" rx="3" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="380" y="47" text-anchor="middle" font-size="8.5" fill="currentColor">CRC-16 · bits 80–95</text>
  <path d="M170 56 L170 72 L360 72 L360 62" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="170" y="90" font-size="8" fill="currentColor">CRC-CCITT (0x1021, init 0x0000)</text>
  <text x="170" y="104" font-size="8" fill="currentColor">stored = CRC XOR 0x5A5A</text>
  <text x="20" y="122" font-size="7.5" fill="currentColor">match on both intact bits AND correct block type; a wrong mask rejects every real CSBK</text>
</svg>
<figcaption>The CRC covers the block's first 80 bits; XORing the fixed 0x5A5A mask onto the checksum before storing it ties acceptance to the block being a genuine CSBK, not merely error-free.</figcaption>
</figure>

## What the mask is for

A bare CRC answers one question: were these bits received unaltered? DMR carries several
kinds of block on the same physical channel, all protected by the same CRC-CCITT, so a bare
check cannot tell a corrupted CSBK apart from an intact block of some other type that happens
to satisfy the polynomial. The mask solves this by giving each block type its own additive
constant. A CSBK uses `0x5A5A`; a receiver that computes the CRC and expects the CSBK mask
will reject anything whose stored value was masked for a different type, even when the bits
themselves are error-free. The check therefore verifies *integrity and identity* at once —
"these bits are intact **and** this is a CSBK" — which is exactly what a control-channel
decoder needs before it acts on an opcode. XOR-ing a fixed constant does not change the CRC's
error-detection strength; it only shifts which received value counts as valid.

## Getting the convention right

The precise convention is load-bearing, and the wrong one fails in a way that hides during
testing. GopherTrunk once used an "init `0xFFFF`, store the bitwise complement" scheme; it
passed every synthesized round-trip fixture — because the encoder and decoder made the same
assumption — yet rejected **every** real off-air CSBK, since genuine transmitters use init
`0x0000` with the `0x5A5A` XOR. The current constants were pinned against real ETSI Tier III
control-channel bursts (Aloha and Preamble blocks that decode cleanly through the
[BPTC](/reference/bptc/) layer first). The lesson mirrors the CRC-family rule elsewhere in the
codebase: a self-consistent encode/decode pair proves nothing about on-air interoperability,
so a CRC convention has to be validated against real captures, not just its own round trip.

## Relevance to SDR

`internal/radio/dmr/tier3/csbk.go` implements the check. `ParseCSBK` takes the 96
information bits recovered from the BPTC decode, computes
`framing.CRCCCITTWithInit(info[:10], 0x0000) ^ 0x5A5A`, compares it against the stored 16-bit
trailer, and returns a `CRCError` (with the partially-parsed block preserved for diagnostics)
on any mismatch:

```go
const csbkCRCMask uint16 = 0x5A5A
storedCRC := binary.BigEndian.Uint16(info[10:12])
want := framing.CRCCCITTWithInit(info[:10], 0x0000) ^ csbkCRCMask
```

The CRC sits after BPTC error correction in the decode chain: BPTC repairs recoverable bit
errors across the 196-bit burst, and this CRC is the final gate that decides whether the
recovered 96-bit block is trustworthy enough to dispatch on. Because a control-channel decoder
follows grants and tracks system state from these blocks, a block that fails the CRC is
dropped rather than acted on — a wrong mask constant silently starves the whole trunking
engine of input.

## Sources

[^wiki]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on CRC computation and the role of an XOR output constant.
[^etsi]: [ETSI TS 102 361-1](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236101/) — ETSI, the DMR air-interface standard defining the CSBK CRC mask in §B.3.11 / Table B.21.
