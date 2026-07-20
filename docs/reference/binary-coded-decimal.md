---
slug: binary-coded-decimal
title: Binary-coded decimal (BCD)
entry_type: concept
category: hw-foundations
description: Binary-coded decimal stores each decimal digit 0-9 in its own 4-bit nibble, leaving six codes unused; it trades range for trivial decimal display and appears in POCSAG, DSC, RDS, and clock chips.
keywords: BCD, binary-coded decimal, packed BCD, unpacked BCD, nibble, decimal digit, POCSAG, DSC, RDS, real-time clock
aka: [BCD, Binary-coded decimal, packed BCD]
autolink: true
infobox:
  - { label: Type, value: Numeric encoding }
  - { label: Unit, value: "One decimal digit per 4-bit nibble" }
  - { label: Unused codes, value: "1010–1111 (six of sixteen)" }
  - { label: Used by, value: "POCSAG, DSC, RDS, RTC chips" }
see_also: [bits-and-bytes, pocsag, dsc]
cite_urls:
  - https://en.wikipedia.org/wiki/Binary-coded_decimal
  - https://www.itu.int/rec/R-REC-M.493/en
---

**Binary-coded decimal** (**BCD**) stores each decimal digit 0–9 in its own 4-bit
[nibble](/reference/bits-and-bytes/), from `0000` to `1001`.[^wiki] Because a nibble can
express sixteen values but only ten digits are used, the six codes `1010`–`1111` go unused —
so BCD wastes about 17% of the available range in exchange for a direct, conversion-free
mapping between stored bits and the decimal digits a human reads.

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 150" role="img" aria-label="The decimal number one nine seven mapped to three 4-bit nibbles: 0001, 1001, 0111, each digit in its own nibble." xmlns="http://www.w3.org/2000/svg">
  <g font-size="12" fill="currentColor" text-anchor="middle">
    <text x="70" y="34">1</text><text x="210" y="34">9</text><text x="350" y="34">7</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1"><line x1="70" y1="42" x2="70" y2="60" marker-end="url(#rel_bcd)"/><line x1="210" y1="42" x2="210" y2="60" marker-end="url(#rel_bcd)"/><line x1="350" y1="42" x2="350" y2="60" marker-end="url(#rel_bcd)"/></g>
  <g font-size="13" fill="currentColor" text-anchor="middle" font-family="monospace">
    <rect x="20" y="66" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.2"/><text x="70" y="88">0001</text>
    <rect x="160" y="66" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.2"/><text x="210" y="88">1001</text>
    <rect x="300" y="66" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.2"/><text x="350" y="88">0111</text>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="70" y="118">nibble</text><text x="210" y="118">nibble</text><text x="350" y="118">nibble</text>
    <text x="210" y="140" fill-opacity="0.85">each decimal digit occupies its own 4 bits</text>
  </g>
  <defs><marker id="rel_bcd" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The decimal number 197 in BCD: each digit maps straight to its own nibble — 1→0001, 9→1001, 7→0111 — with no binary-to-decimal arithmetic needed.</figcaption>
</figure>

## How it works

In pure binary the number 197 is `11000101`, and recovering its decimal digits takes repeated
division. BCD skips that: each digit is encoded independently, so `197` is simply `0001 1001
0111`. Reading a digit out to a seven-segment display, rounding to a decimal place, or
transmitting a digit string becomes a lookup with no conversion. The trade is density — BCD
needs four bits per digit where binary would pack the same value more tightly, and a quarter of
each nibble's codes are illegal.

BCD comes in two layouts. **Packed BCD** stores two digits per byte, one in each nibble, which
is the compact form used in most protocols and storage. **Unpacked BCD** stores a single digit
per byte, wasting the high nibble but simplifying byte-at-a-time processing. Arithmetic on BCD
requires a decimal-adjust step whenever a nibble would exceed `1001`, which is why processors
historically included a decimal-adjust instruction.

## In practice

BCD is favoured wherever decimal digits are displayed or transmitted directly and a
binary-to-decimal conversion would be a nuisance. Numeric **[POCSAG](/reference/pocsag/)** pages
carry their digits as BCD, as do the maritime identities and call fields of
**[DSC](/reference/dsc/)** (Digital Selective Calling). The Radio Data System (**RDS**) uses BCD
for time and date, and real-time-clock chips store the seconds, minutes, hours, and calendar in
BCD so firmware can read the time out digit by digit. In each case the payload is meant to be
read as decimal, so encoding it as decimal from the start is simpler than repeatedly converting.

## Relevance to SDR

A decoder that pulls a numeric page or a DSC identity off the air must know the field is BCD,
not binary — misreading a packed BCD byte as an ordinary integer yields nonsense. Recognising the
`1010`–`1111` codes as invalid also serves as a cheap sanity check on a recovered bitstream.
GopherTrunk's trunking focus centres on P25/DMR/NXDN/TETRA control data, but BCD is exactly the
kind of low-level digit encoding a paging or maritime decoder built alongside it must parse
correctly.

## Sources

[^wiki]: [Binary-coded decimal](https://en.wikipedia.org/wiki/Binary-coded_decimal) — Wikipedia, for the nibble-per-digit encoding, unused codes, and the packed vs unpacked forms.
[^dsc]: [ITU-R M.493, Digital Selective-Calling system](https://www.itu.int/rec/R-REC-M.493/en) — the recommendation defining the DSC message format that carries maritime identities as decimal digits.
