---
slug: motorola-talker-alias-cipher
title: Motorola talker-alias cipher
entry_type: algorithm
category: cryptography
description: The Motorola talker-alias cipher is a proprietary per-byte obfuscation over a radio's display name — a length-seeded 16-bit additive accumulator, a 256-byte lookup table, and an odd-multiplier step. It was recovered by clean-room reverse engineering, verified against real over-the-air data, and is enabled in GopherTrunk.
keywords: Motorola talker alias cipher, per-byte cipher, additive accumulator, 293, 0xC433, lookup table, clean-room reverse engineered, verified, CipherVerified, issue 773, P25 alias obfuscation
aka: [Motorola alias cipher, "per-byte alias cipher"]
autolink: true
infobox:
  - { label: Type, value: Proprietary per-byte transform }
  - { label: State, value: 16-bit additive accumulator + 256-byte LUT }
  - { label: Status, value: "Clean-room recovered, VERIFIED" }
  - { label: Default, value: "Enabled (CipherVerified=true)" }
see_also: [p25-talker-alias, motorola-type-ii, radio-id, rc4-cipher, scrambling, p25-encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

The **Motorola talker-alias cipher** is the proprietary per-byte obfuscation Motorola applies
to a radio's [talker alias](/reference/p25-talker-alias/) — the human-readable display name —
before it is fragmented across the air.[^wiki] It is **not** encryption for confidentiality
in the [AES](/reference/p25-encryption/) sense; it is a lightweight, undocumented
[scrambling](/reference/scrambling/) of the alias string that a receiver must reverse to read
the name. Its structure is a length-seeded 16-bit **additive** accumulator, a 256-byte
substitution table, and an odd-multiplier (modular-inverse-mod-256) step, with the decoded
bytes read as UTF-16 BE.

**This cipher was recovered by clean-room reverse engineering and is verified.** It was
reconstructed from black-box (input → output) observations of an existing decoder driven as an
opaque oracle — no third-party source or table was read or copied — and it decodes a real
over-the-air capture (RID 200062 → "CRIO 0062") with a valid CRC. GopherTrunk decodes it by
default (`CipherVerified = true`). For the full recovery method, reproducible validation, and
the clean-room provenance/licensing record, see
[the clean-room provenance document](https://github.com/MattCheramie/GopherTrunk/blob/main/research/p25-talker-alias-cleanroom-provenance.md).

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="One byte of the talker-alias cipher: a 16-bit accumulator is advanced by an additive step, a 256-byte lookup table combines with the accumulator high byte, an odd-multiplier factor is applied, and one decoded byte is produced before the accumulator advances." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="120" height="26" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="80" y="37" text-anchor="middle" font-size="8" fill="currentColor">W += 293 × (byte+1)</text>
  <path d="M140 33 L175 33" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="175" y="20" width="110" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="37" text-anchor="middle" font-size="8" fill="currentColor">LUT[byte] − (W≫8)</text>
  <path d="M285 33 L320 33" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="320" y="20" width="130" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="37" text-anchor="middle" font-size="8" fill="currentColor">× inv((W&amp;0xFF)|1)</text>
  <path d="M385 46 L385 70 L230 70" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="175" y="70" width="110" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="87" text-anchor="middle" font-size="8" fill="currentColor">decoded byte</text>
  <path d="M175 83 L80 83 L80 46" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="235" y="120" text-anchor="middle" font-size="7.5" fill="currentColor">seed W = 0xC433 + 586·(n−1); repeat per byte — clean-room recovered, verified</text>
</svg>
<figcaption>The transform chains an additive accumulator, a table lookup, and an odd-multiplier per byte; the 256-byte table and constants were recovered clean-room and verified against real over-the-air data, so the path is enabled.</figcaption>
</figure>

## The per-byte transform

The decode-side transform lives in `internal/radio/p25/motorola/alias.go`. There is no vendor
spec — the algorithm below was recovered from behaviour — so it is worth showing:

```go
// W seeded from the character count n = len(encoded)/2; LUT is a 256-byte table.
n := len(encoded) / 2
w := uint16(0xC433 + 586*(n-1))                     // length seed (586 = 2*293)
for i := 0; i < 2*n; i++ {
    // per-byte substitution: odd-multiplier × (table lookup − accumulator high byte)
    out[i] = motorolaAliasInvOdd[byte(w)|1] * (motorolaAliasLUT[encoded[i]] - byte(w>>8))
    w += uint16(293 * (int(encoded[i]) + 1))        // additive accumulator update
}
// pack pairs into UTF-16BE chars; force the high byte to 0xFF when the low byte is >= 0x80.
```

`motorolaAliasInvOdd[(W&0xFF)|1]` is the multiplicative inverse of the accumulator's low byte
(forced odd) modulo 256, and the subtraction mixes in the accumulator's high byte `W>>8`. The
accumulator update is **additive** (`W += 293·(byte+1)`) — an earlier inferred *multiplicative*
form (`accum×293+0x72E9`) was wrong and decoded nothing; the oracle recovery corrected it. The
decoded byte stream is read as UTF-16 BE and rendered to printable ASCII.

## Verification status

Both the **SUID framing** (WACN / System / [Radio ID](/reference/radio-id/) prefix) and the
**cipher itself** are verified. The cipher was recovered by clean-room reverse engineering —
reconstructed from ~22,000 black-box (input → output) observations of an existing decoder
driven as an opaque oracle — and it reproduces that oracle byte-for-byte on a 300-row held-out
set it never trained on (**1242/1242 characters**). Independently, it decodes a **real
over-the-air capture** (RID 200062) to "CRIO 0062" — clean ASCII whose "0062" is the tail of
the radio's own ID — with a CRC-16/GSM that matches the on-air bytes (`0x6A96`). The earlier
worry that the one #376 sample was "underdetermined" was a framing artifact: the cipher length
must be taken from the self-delimiting CRC boundary (stripping the trailing FACCH pad), after
which it decodes cleanly.

The decode is gated behind the `CipherVerified` constant, now **`true`**, standing on a
committed regression fixture (the held-out oracle vectors). While true, a clean-ASCII decode is
reported reliable and the name surfaces; a garbled decode is still flagged unreliable. See the
[clean-room provenance record](https://github.com/MattCheramie/GopherTrunk/blob/main/research/p25-talker-alias-cleanroom-provenance.md)
for the method and reproducible validation.

## Licensing

A working implementation also exists in **SDRTrunk** (GPLv3); GopherTrunk is Apache-2.0, so that
source and table **were not ported**. The cipher was instead recovered by clean-room reverse
engineering: an existing decoder was driven as an opaque black-box oracle — its public decode
accessor only, with **no source, table, or algorithm read or copied** — and the cipher's
structure and 256-byte table were reconstructed as functional facts from the observed
input/output behaviour, then re-implemented independently in Go. The GPL-linked measurement
instruments were quarantined and never committed. The full provenance — the exact boundary of
what was and was not read, and the reproducible validation — is documented in the
[clean-room provenance record](https://github.com/MattCheramie/GopherTrunk/blob/main/research/p25-talker-alias-cleanroom-provenance.md).

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard. The cipher is a proprietary Motorola extension with no vendor specification; it was recovered by clean-room reverse engineering and verified against real over-the-air data per GopherTrunk issue #773.
