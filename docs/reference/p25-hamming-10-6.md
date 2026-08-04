---
slug: p25-hamming-10-6
title: P25 Hamming(10,6,3)
entry_type: algorithm
category: error-correction
description: The shortened Hamming(10,6,3) code is the single-error-correcting inner FEC P25 applies to each 6-bit fragment of the Link Control and Encryption Sync words inside an LDU — 24 codewords across the 240-bit LC/ES field.
keywords: P25 Hamming 10 6, shortened Hamming, Hamming(10,6,3), single error correcting, LC inner FEC, encryption sync FEC, syndrome decode, TIA-102 BAAA
aka: ["Hamming(10,6,3)", "P25 short Hamming", "10-6 Hamming code"]
autolink: true
infobox:
  - { label: Parameters, value: "(10,6,3) — 6 data + 4 parity" }
  - { label: Corrects, value: 1 bit error per codeword }
  - { label: Per LC/ES field, value: 24 codewords (240 bits) }
  - { label: Spec, value: TIA-102.BAAA }
see_also: [hamming-code, p25-link-control-word, p25-encryption-sync, p25-reed-solomon, p25-logical-data-unit, forward-error-correction]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Hamming_code
  - https://en.wikipedia.org/wiki/Hamming_distance
---

The **P25 Hamming(10,6,3)** code is the single-error-correcting inner
[Hamming code](/reference/hamming-code/) that P25 (TIA-102.BAAA) applies to each 6-bit fragment
of the [Link Control](/reference/p25-link-control-word/) and
[Encryption Sync](/reference/p25-encryption-sync/) words carried inside an
[LDU](/reference/p25-logical-data-unit/).[^ham] Each codeword is 6 data bits plus 4 parity
bits; a full 240-bit LC/ES field is 24 of them, and the recovered 24 × 6 = 144 bits become the
symbols of the outer [Reed-Solomon](/reference/p25-reed-solomon/) code that cleans up whatever
the Hamming layer misses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A 10-bit codeword of six data bits followed by four parity bits; each data bit contributes a fixed 4-bit column to the parity, and on decode the received-versus-recomputed parity XOR gives a syndrome that names the single flipped bit." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor">
    <rect x="20" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="35" y="47" text-anchor="middle">d0</text>
    <rect x="50" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="65" y="47" text-anchor="middle">d1</text>
    <rect x="80" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="95" y="47" text-anchor="middle">d2</text>
    <rect x="110" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="125" y="47" text-anchor="middle">d3</text>
    <rect x="140" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="155" y="47" text-anchor="middle">d4</text>
    <rect x="170" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="185" y="47" text-anchor="middle">d5</text>
    <rect x="200" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="215" y="47" text-anchor="middle">p0</text>
    <rect x="230" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="245" y="47" text-anchor="middle">p1</text>
    <rect x="260" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="275" y="47" text-anchor="middle">p2</text>
    <rect x="290" y="30" width="30" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="305" y="47" text-anchor="middle">p3</text>
  </g>
  <text x="20" y="80" font-size="8" fill="currentColor">syndrome = recomputed parity XOR received parity → 4-bit value naming the one flipped bit (0 = clean)</text>
</svg>
<figcaption>Six data bits and four parity bits make a distance-3 codeword; the 4-bit syndrome uniquely identifies which of the ten positions flipped, letting the decoder correct any single-bit error.</figcaption>
</figure>

## How it works

The code is defined by six fixed 4-bit **parity columns**, one per data bit; the parity nibble
is the XOR of the columns of the set data bits. The four parity bits themselves occupy the four
unit columns `0x8/0x4/0x2/0x1`, so all ten columns are distinct and nonzero — the property that
makes a distance-3 (single-error-correcting) code. GopherTrunk's parity columns are:

| Data bit | d0 | d1 | d2 | d3 | d4 | d5 |
|----------|----|----|----|----|----|----|
| Parity column | 0x7 | 0xB | 0xD | 0xE | 0x3 | 0x5 |

On decode, the receiver recomputes parity over the six received data bits and XORs it against
the four received parity bits to form a 4-bit **syndrome**. A zero syndrome means the codeword
is clean. A nonzero syndrome is matched against the six data columns and the four unit columns:
whichever column it equals names the single flipped bit, which is then corrected. Because the
code has distance 3, it can *correct* one error or *detect* two, but a two-bit error lands on a
syndrome that points at the wrong single bit — which is precisely why P25 wraps 24 of these
codewords in an outer RS code that operates on whole 6-bit symbols.

## In practice

The 6-bit data width is not an accident: each Hamming codeword's six data bits are exactly one
GF(2⁶) symbol of the outer [RS(24,12,13)](/reference/p25-reed-solomon/) (Link Control) or
RS(24,16,9) (Encryption Sync) codeword. So the two layers compose cleanly — the inner Hamming
pass fixes scattered single-bit hits and hands 24 six-bit symbols up, and the outer RS pass
corrects up to *t* whole symbols the Hamming layer got wrong or silently miscorrected. This
concatenation is what keeps a [talkgroup](/reference/talkgroup/) or crypto
[Message Indicator](/reference/p25-encryption-sync/) intact through the marginal SNR where a
bare Hamming layer would let bit errors slip through into the payload.

## Relevance to SDR

`internal/radio/p25/phase1/hamming10_6.go` implements this locally rather than reusing the
framing package's (15,11) and (13,9) shortenings, because P25's LC/ES fragments are exactly
6 bits wide. `lcInnerDecode` runs the 24 codewords across a 240-bit LC/ES field and returns the
144 data bits plus a corrected-error count, which the [Link Control](/reference/p25-link-control-word/)
and Encryption Sync parsers feed straight into the outer RS layer. It is a small, table-driven
code, but getting its parity columns right is a precondition for reading any voice-channel
metadata on P25 Phase 1.

## Sources

[^ham]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, on the single-error-correcting block codes this shortening belongs to.
[^dist]: [Hamming distance](https://en.wikipedia.org/wiki/Hamming_distance) — Wikipedia, on the minimum-distance-3 property that bounds correction to one bit.
