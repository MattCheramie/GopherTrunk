---
slug: cyclic-redundancy-check
title: Cyclic redundancy check (CRC)
entry_type: algorithm
category: error-correction
description: A cyclic redundancy check appends a checksum computed by polynomial division so the receiver can detect corrupted frames; it detects but does not correct, and guards P25, DMR, ADS-B, AIS and DSC.
keywords: CRC, cyclic redundancy check, checksum, error detection, FCS, generator polynomial, polynomial division, CRC-16, CRC-24, CRC-CCITT
aka: [cyclic redundancy check, CRC, FCS, frame check sequence]
autolink: true
infobox:
  - { label: Type, value: Error-detection code }
  - { label: Method, value: Polynomial division remainder }
  - { label: Used by, value: P25, DMR, ADS-B, AIS, DSC, AX.25 }
see_also: [forward-error-correction, fire-code, bch-code, ads-b, ais, ax25]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
  - https://en.wikipedia.org/wiki/Mathematics_of_cyclic_redundancy_checks
---

A **cyclic redundancy check** (**CRC**) is an error-*detection* code that appends a short
checksum — the remainder of a **polynomial division** of the message by a fixed generator
polynomial.[^wiki] The receiver repeats the division; a non-zero remainder means the frame was
corrupted in transit, so it is discarded. A CRC **detects** errors but, on its own, does not
correct them, which distinguishes it from [forward error correction](/reference/forward-error-correction/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A data frame with a CRC value appended is divided by a generator polynomial at the receiver, which recomputes the remainder and reports pass or fail." xmlns="http://www.w3.org/2000/svg">
  <rect x="24" y="42" width="200" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="124" y="61" text-anchor="middle" font-size="9" fill="currentColor">data frame</text>
  <rect x="224" y="42" width="72" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.2"/><text x="260" y="61" text-anchor="middle" font-size="9" fill="currentColor">CRC</text>
  <line x1="300" y1="57" x2="336" y2="57" stroke="currentColor" marker-end="url(#crcar)"/>
  <text x="404" y="45" font-size="9" fill="currentColor" text-anchor="middle">÷ generator</text><text x="404" y="58" font-size="9" fill="currentColor" text-anchor="middle">remainder = 0?</text><text x="404" y="71" font-size="9" fill="currentColor" text-anchor="middle">→ pass / fail</text>
  <text x="180" y="100" text-anchor="middle" font-size="9" fill="currentColor">detects corruption; does not correct it</text>
  <defs><marker id="crcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CRC treats the frame as a polynomial, divides by a fixed generator, and sends the remainder so the receiver can verify the frame arrived intact.</figcaption>
</figure>

## How it works

CRCs work in the algebra of polynomials over GF(2) — binary coefficients where addition is
XOR and there are no carries. The message bits are treated as the coefficients of a big
polynomial *M(x)*. That polynomial is shifted left by the CRC width (multiplied by *xⁿ*) and
divided by a fixed **generator polynomial** *G(x)* of degree *n*; the remainder *R(x)* is the
*n*-bit CRC that gets appended. Because the transmitted word `M(x)·xⁿ + R(x)` is now exactly
divisible by *G(x)*, the receiver just divides the whole thing again and checks for a zero
remainder. In hardware or software this "division" is a chain of XORs implemented with a
[linear-feedback shift register](/reference/linear-feedback-shift-register/) or a byte-wise
lookup table, so it costs almost nothing.

The strength of a CRC comes from choosing *G(x)* well. A well-chosen degree-*n* generator
guarantees detection of: **all** single-bit errors; **all** odd numbers of bit errors (if
*G(x)* has *x+1* as a factor); **all** burst errors up to *n* bits long; and any other error
pattern except the vanishingly small fraction (about 2⁻ⁿ) that happens to be a multiple of
*G(x)*. This is far stronger than a simple sum-based checksum for the same number of bits,
which is why CRCs dominate frame validation in communications.

## Variants

Standard generators are known by width and polynomial:

- **CRC-16-CCITT** (`0x1021`) — used as the frame check sequence (FCS) in [AX.25](/reference/ax25/)
  packet radio, [AIS](/reference/ais/), and many HDLC-derived links.
- **CRC-24** — [ADS-B](/reference/ads-b/) / Mode S uses a 24-bit CRC (`0xFFF409`) that is also
  overlaid with the aircraft address, so a valid remainder simultaneously confirms integrity
  and recovers the ICAO address.
- **CRC-16 / CRC-9 / CRC-8** — [DMR](/reference/dmr/) sprinkles several CRC widths across its
  bursts (a 9-bit CRC on the CSBK/data headers, 5- and 8-bit CRCs elsewhere).
- **CRC families in P25** — [Project 25](/reference/project-25/) uses a CRC-16 on packet data
  and header blocks and shorter CRCs on control words.
- **[DSC](/reference/dsc/)** — maritime Digital Selective Calling protects its sequences with a
  parity/error-check scheme in the same spirit.

## In practice

A subtlety worth knowing: many real CRCs are *not* the textbook plain remainder. Protocols add
an **initial fill** (preloading the register with all-ones so leading zeros are covered), a
**final XOR** of the output, and bit-reflection of input and/or output bytes. Getting a CRC to
match a live signal often means matching those parameters exactly, not just the polynomial —
`CRC-16/CCITT-FALSE` vs `CRC-16/X25` differ only in init and reflection yet produce entirely
different check values.

## Relevance to SDR

CRC validation is the last gate before GopherTrunk trusts a decoded frame: after demodulation,
de-interleaving, and any FEC, the CRC says whether the surviving bits are self-consistent.
GopherTrunk recomputes the appropriate CRC for each protocol it decodes — CRC-24 on
[ADS-B](/reference/ads-b/) squitters, CRC-16 on [AIS](/reference/ais/) sentences and
[AX.25](/reference/ax25/) frames, the DMR and P25 header CRCs — and drops frames that fail,
so it reports channel grants, positions, and talkgroups that actually arrived intact rather
than noise that happened to pass framing. Some systems combine a CRC with a burst-correcting
[Fire code](/reference/fire-code/) or a [BCH](/reference/bch-code/) code so the same polynomial
machinery can both correct short bursts and detect what it cannot fix.

## Sources

[^wiki]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, for the polynomial-division construction, standard generators, and detection guarantees.
[^math]: [Mathematics of cyclic redundancy checks](https://en.wikipedia.org/wiki/Mathematics_of_cyclic_redundancy_checks) — Wikipedia, for the GF(2) polynomial algebra, LFSR implementation, and init/reflect/XOR parameters.
