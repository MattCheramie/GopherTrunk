---
slug: tetra-mobile-network-identity
title: TETRA mobile network identity (MNI)
entry_type: term
category: trunked-radio
description: "The TETRA mobile network identity (MNI) is the MCC + MNC pair that globally names a TETRA network — and, because it seeds the scrambler, a receiver that assumes MNI 0 on a real network cannot descramble its traffic."
keywords: TETRA MNI, mobile network identity, mobile country code, mobile network code, MCC, MNC, extended colour code, scrambler seed, DMO colour recovery, EN 300 392-1
aka: [MNI, mobile network identity, MCC/MNC]
autolink: true
infobox:
  - { label: Composition, value: "MCC (10 bits) + MNC (14 bits)" }
  - { label: Names, value: One TETRA network, globally }
  - { label: Broadcast in, value: SYNC PDU / SYSINFO }
  - { label: Consequence, value: Part of every scrambler seed except the BSCH's }
see_also: [tetra, tetra-extended-colour-code, tetra-sync-pdu, tetra-scrambler, tetra-dmo, color-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Mobile_country_code
---

**The TETRA mobile network identity** (**MNI**) is the pair of numbers that names a
[TETRA](/reference/tetra/) network globally: a 10-bit **mobile country code** (MCC), drawn
from the same ITU country-code plan cellular uses, and a 14-bit **mobile network code**
(MNC) assigned within that country.[^mcc] Every TETRA cell broadcasts its MNI in the
[SYNC PDU](/reference/tetra-sync-pdu/) and SYSINFO, radios are provisioned with the MNI of
their home network, and roaming and authentication decisions key on it. For a listener the
MNI is more than a label, because TETRA folds it into the physical layer: it is the upper
24 bits of the scrambling seed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The 30-bit extended colour code drawn as three concatenated fields — 10-bit MCC and 14-bit MNC forming the MNI, then the 6-bit colour code — feeding the scrambler seed." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="30" y="36" width="120" height="32"/><rect x="150" y="36" width="168" height="32"/><rect x="318" y="36" width="72" height="32"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="90" y="56">MCC (10)</text><text x="234" y="56">MNC (14)</text><text x="354" y="56">colour (6)</text>
  </g>
  <path d="M30 76 L318 76" stroke="currentColor" stroke-opacity="0.5"/>
  <path d="M30 72 L30 80 M318 72 L318 80" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="174" y="92" font-size="8.5" fill="currentColor" text-anchor="middle">mobile network identity (MNI)</text>
  <text x="230" y="110" font-size="8.5" fill="currentColor" text-anchor="middle">all 30 bits together: the extended colour code that seeds the scrambler</text>
</svg>
<figcaption>The MNI is the network's name and two-thirds of its scrambling seed: same bits, two jobs.</figcaption>
</figure>

## Why it matters to a receiver

TETRA scrambles every logical channel except the BSCH with the 30-bit
[extended colour code](/reference/tetra-extended-colour-code/) — `MCC | MNC | colour`,
MSB-first (EN 300 392-2 §8.2.5.2). The 6-bit colour code distinguishes neighbouring cells
*within* a network; the MNI distinguishes networks. The practical consequence: knowing the
colour code alone is not enough to descramble anything. A receiver that searches colour
codes 0–63 while silently assuming MNI 0 is only exploring 64 seeds out of 2³⁰, and on any
real network — whose MNI is never 0 — the true seed is simply not in its search space.
Every candidate then decodes at the CRC chance floor, several rise modestly at once by
coincidence, and none dominates: a pattern easy to misread as a weak signal or as
encryption.

In trunked mode this blind spot cannot open, because the always-decodable BSCH hands over
the real MCC/MNC before anything else is attempted. In
[direct mode](/reference/tetra-dmo/) it can and did: the DMO SCH/S carries MNI 0 *on air*
(that is correct parsing, not a bug — both osmo-tetra-dmo and GopherTrunk agree), so
direct-mode traffic scrambled with a real MNI gives the receiver no over-the-air way to
learn it. GopherTrunk hit exactly this on a Motorola DMO network operating with MCC 250 /
MNC 1: its empirical colour recovery — sweep candidate colours, keep the one that maximises
CRC-valid speech — found no dominant winner until the configured MNI was folded into every
candidate seed (`baseMNI | c`). The failing-first regression
(`TestRecoverDMColourCodeNonZeroMNI`) scrambles with the independent osmo-derived seed for
MCC 250 / MNC 1 / colour 7 and shows the MNI-0 sweep finding nothing while the MNI-folded
sweep recovers the exact seed.

## Relevance to SDR

For trunked TETRA, GopherTrunk needs no MNI configuration — the cold-start chain
(BSCH → SYNC PDU → extended colour code) learns it. For DMO, the `tetra_mcc` and
`tetra_mnc` system-config keys supply the network MNI, which threads through the DMO
pipeline and voice chain into colour recovery and the descramble seed. Diagnostic rule of
thumb for any TETRA decode that sits at the chance floor with healthy sync: before
concluding "encrypted", confirm the full 30-bit seed — colour *and* MNI — is right. The
DMO-side story, including the colour sweep's dominance gate and the on-air captures that
exposed the blind spot, is told in [TETRA DMO facts](/reference/tetra-dmo-facts/).

## Sources

[^mcc]: [Mobile country code](https://en.wikipedia.org/wiki/Mobile_country_code) — Wikipedia, on the ITU country-code plan and per-country network codes that TETRA's MCC/MNC fields reuse.
