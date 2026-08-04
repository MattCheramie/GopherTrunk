---
slug: motorola-talker-alias-cipher
title: Motorola talker-alias cipher
entry_type: algorithm
category: cryptography
description: The Motorola talker-alias cipher is a proprietary per-byte obfuscation over a radio's display name — a length-seeded 16-bit accumulator, a 256-byte lookup table, and an odd-multiplier step. It is reverse-engineered, unverified, and gated off by default in GopherTrunk.
keywords: Motorola talker alias cipher, per-byte cipher, accumulator, 293, 0x72E9, lookup table, reverse engineered, unverified, CipherVerified, issue 773, P25 alias obfuscation
aka: [Motorola alias cipher, "per-byte alias cipher"]
autolink: true
infobox:
  - { label: Type, value: Proprietary per-byte transform }
  - { label: State, value: 16-bit accumulator + 256-byte LUT }
  - { label: Status, value: "Reverse-engineered, UNVERIFIED" }
  - { label: Default, value: "Gated off (CipherVerified=false)" }
see_also: [p25-talker-alias, motorola-type-ii, radio-id, rc4-cipher, scrambling, p25-encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

The **Motorola talker-alias cipher** is the proprietary per-byte obfuscation Motorola applies
to a radio's [talker alias](/reference/p25-talker-alias/) — the human-readable display name —
before it is fragmented across the air.[^wiki] It is **not** encryption for confidentiality
in the [AES](/reference/p25-encryption/) sense; it is a lightweight, undocumented
[scrambling](/reference/scrambling/) of the alias string that a receiver must reverse to read
the name. Its structure is inferred from public protocol notes: a length-seeded 16-bit
accumulator, a 256-byte substitution table, and an odd-multiplier (modular-inverse-mod-256)
step, with the decoded bytes read as UTF-16 BE.

**This cipher is reverse-engineered and unverified.** GopherTrunk does not decode it by
default and never presents its output as a confirmed name — the caveats below are not
incidental, they are the whole point of the page.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="One byte of the talker-alias cipher: a 16-bit accumulator is stepped by a multiply-and-add, indexes a 256-byte lookup table, combines with an odd-multiplier factor, and yields one decoded byte before the accumulator advances." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="120" height="26" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="80" y="37" text-anchor="middle" font-size="8" fill="currentColor">accum × 293 + 0x72E9</text>
  <path d="M140 33 L175 33" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="175" y="20" width="110" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="37" text-anchor="middle" font-size="8" fill="currentColor">LUT[byte+128]</text>
  <path d="M285 33 L320 33" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="320" y="20" width="130" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="37" text-anchor="middle" font-size="8" fill="currentColor">× odd-multiplier m2</text>
  <path d="M385 46 L385 70 L230 70" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="175" y="70" width="110" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="87" text-anchor="middle" font-size="8" fill="currentColor">decoded byte</text>
  <path d="M175 83 L80 83 L80 46" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="235" y="120" text-anchor="middle" font-size="7.5" fill="currentColor">accum advances by (byte + 1); repeat per byte — UNVERIFIED table &amp; constants</text>
</svg>
<figcaption>The transform chains a stepped accumulator, a table lookup, and an odd-multiplier per byte; both the 256-byte table and the constants are unconfirmed placeholders, so the whole path is gated off.</figcaption>
</figure>

## The per-byte transform

The decode-side transform lives in `internal/radio/p25/motorola/alias.go`. It exists only in
reverse-engineering notes — there is no spec table to point at — so the exact algorithm is
worth showing:

```go
// accum seeded with the encoded length; LUT is a 256-byte signed table.
accum = uint16(len(encoded))
for i, raw := range encoded {
    accum = uint16(uint32(accum)*293 + 0x72E9)      // stepped accumulator
    lut := motorolaAliasLUT[int(int8(raw))+128]     // signed table lookup
    m1 := int8(int(lut) - int(int8(accum>>8)))

    var m2 int8 = 1                                 // odd-multiplier search
    stop := int8(accum | 1)
    increment := stop << 1
    for stop != 1 && m2 != -1 {
        stop += increment
        m2 += 2
    }
    decoded[i] = byte(int8(int(m1) * int(m2)))
    accum = uint16(uint32(accum) + uint32(raw) + 1) // advance
}
```

The `m2` loop finds the multiplicative inverse of `accum | 1` modulo 256 (odd values are
units mod 256), and `m1` mixes the table output with the high byte of the accumulator. The
decoded byte stream is then read as UTF-16 BE and rendered to printable ASCII.

## Verification status — read this

The **SUID framing** around the cipher (the WACN / System / [Radio ID](/reference/radio-id/)
prefix) is verified against real traffic. The **cipher itself is not.** Per GopherTrunk issue
#773: the 256-byte table (`motorolaAliasLUT`) is a placeholder permutation of unconfirmed
provenance, the accumulator constants (`293`, `0x72E9`) are inferred, and the routine decodes
*nothing* on live traffic. The one partial capture available (RID 200062) is mathematically
underdetermined — dozens of distinct constant-sets reproduce its known bytes while disagreeing
on the unknown ones, and one alias character is unrecoverable from that sample — so it cannot
pin the table. This mirrors GopherTrunk's own house rule of labelling
reverse-engineered work as best-effort and not hardware-confirmed.

Because a wrong table could fabricate a plausible name, the decode is **gated behind the
`CipherVerified` constant, which is `false`.** While it is false, `DecodeMessage` never reports
an alias as reliable, and callers must not surface the output as a confirmed name. The
constant is to be flipped to `true` only together with a committed regression fixture mapping
real encoded bytes to the correct plaintext for the same RID — never on inference alone.

## Licensing

A working implementation exists in **SDRTrunk**, but SDRTrunk is GPLv3 and GopherTrunk is
Apache-2.0, so that table and decode **must not be ported** into GopherTrunk. A verified table
therefore requires either ground-truth captures GopherTrunk can solve independently or a
separate licensing path. Until then the cipher stays off, and GopherTrunk instead logs the
raw encoded region so the cryptanalysis can be finished from committed data.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard. The cipher is a proprietary Motorola extension, reverse-engineered and unverified per GopherTrunk issue #773; no vendor specification exists.
