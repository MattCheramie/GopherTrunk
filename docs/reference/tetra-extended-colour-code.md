---
slug: tetra-extended-colour-code
title: TETRA extended colour code
entry_type: term
category: trunked-radio
description: The TETRA extended colour code is the 30-bit cell identity — mobile country code (10) followed by mobile network code (14) followed by the 6-bit colour code — that seeds the scrambler for every logical channel, learned at cold start from the BSCH.
keywords: TETRA extended colour code, colour code, MCC MNC, scrambler seed, MAC-SYNC, BSCH, cell identity, EN 300 392-2 8.2.5.2
aka: [extended colour code, "TETRA scrambling code", ECC]
autolink: true
infobox:
  - { label: Width, value: 30 bits }
  - { label: Layout, value: "MCC(10) || MNC(14) || colour(6)" }
  - { label: Learned from, value: BSCH / MAC-SYNC }
  - { label: Spec, value: EN 300 392-2 §8.2.5.2 }
see_also: [tetra-scrambler, tetra-mobile-network-identity, tetra-sync-pdu, color-code, tetra-logical-channels, tetra-mac-pdu, control-channel, tetra, tetra-burst-formats, tetra-cmce-mle-pdu]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Mobile_country_code
---

The **TETRA extended colour code** is the 30-bit value that uniquely identifies a cell's scrambling
context and seeds the [TETRA scrambler](/reference/tetra-scrambler/) for every logical channel except the
BSCH.[^tetra] It is built from three fields, MSB-first: the 10-bit **mobile country code** (MCC), the 14-bit
**mobile network code** (MNC), and the 6-bit **[colour code](/reference/color-code/)** that distinguishes
neighbouring cells of the same network.[^mcc] Because the scrambler is seeded from it, a receiver cannot
decode any traffic or signalling channel until it has learned the extended colour code — which it does by
decoding the one channel that is *not* scrambled with it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 430 116" role="img" aria-label="A 30-bit field split into three parts left to right: a 10-bit mobile country code, a 14-bit mobile network code, and a 6-bit colour code, concatenated most-significant-bit first to form the scrambler seed." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="120" height="30" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/>
  <text x="80" y="53" text-anchor="middle" font-size="9" fill="currentColor">MCC · 10</text>
  <rect x="140" y="34" width="168" height="30" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.1"/>
  <text x="224" y="53" text-anchor="middle" font-size="9" fill="currentColor">MNC · 14</text>
  <rect x="308" y="34" width="90" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="353" y="53" text-anchor="middle" font-size="9" fill="currentColor">colour · 6</text>
  <text x="20" y="26" font-size="7.5" fill="currentColor">MSB</text>
  <text x="398" y="26" text-anchor="end" font-size="7.5" fill="currentColor">LSB</text>
  <text x="209" y="86" text-anchor="middle" font-size="8" fill="currentColor">30 bits total → seeds the scrambler LFSR</text>
  <text x="209" y="102" text-anchor="middle" font-size="7.5" fill="currentColor">e(1) is the MCC's high bit; e(30) is the colour code's low bit</text>
</svg>
<figcaption>The extended colour code packs MCC, MNC, and the 6-bit colour code into 30 bits; that packed value, with e(1) in the most-significant position, is what the scrambler's initialisation vector expects.</figcaption>
</figure>

## Cold-start learning

At cold start a TETRA receiver has no configured colour code, so it hunts the
[synchronisation burst](/reference/tetra-burst-formats/), whose BSCH is scrambled with colour code **zero**
— a value any receiver can undo. The BSCH carries a MAC-SYNC broadcast [PDU](/reference/tetra-mac-pdu/) whose
60 decoded bits contain the three components: the 6-bit colour code, plus (further along) the 10-bit MCC and
14-bit MNC. GopherTrunk's `ParseSyncPDU` reads those fields, and `ExtendedColourCode` assembles them:

`(MCC & 0x3FF) << 20 | (MNC & 0x3FFF) << 6 | (colour & 0x3F)`.

Once that value is in hand, every other channel — BNCH SYSINFO, SCH/HD, SCH/F, the AACH, and the traffic
channels — becomes decodable without any operator configuration. This is why the BSCH is the linchpin of
TETRA cold acquisition: it is the bootstrap that turns a locked-but-opaque carrier into a fully readable cell.

## Seeding the scrambler

The packed value carries e(1) in bit 29 (the MCC's most-significant bit) down to e(30) in bit 0 (the colour
code's least-significant bit). The scrambler, however, wants e(i+1) in state bit i, so the value's low 30 bits
are bit-reversed on the way into the LFSR — a detail that, when wrong, is invisible to any round-trip test
because both encode and decode share the reversed seed (see [TETRA scrambler](/reference/tetra-scrambler/)).
Keeping the packing order fixed here — MCC, then MNC, then colour, MSB-first — is what makes the reversed seed
land the colour-code bits where §8.2.5.2's initialisation equation expects them.

## Relevance to SDR

`internal/radio/tetra/sync_pdu.go` implements the whole path: `SyncPDU` holds the parsed MAC-SYNC fields,
`ParseSyncPDU` extracts them from the BSCH bit vector, and both the method form `SyncPDU.ExtendedColourCode`
and the package function `ExtendedColourCode(mcc, mnc, colour)` produce the 30-bit seed. The scanner passes
that seed into `framing.NewScramblerTetra` and the traffic extractor, so the learned identity flows straight
into descrambling. An operator who already knows the cell's MCC/MNC/colour can supply the three components
directly, skipping cold-start learning — but on an unknown system the BSCH decode is what makes everything
else possible.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA standard and its cell-identity broadcast.
[^mcc]: [Mobile country code](https://en.wikipedia.org/wiki/Mobile_country_code) — Wikipedia, on the MCC/MNC pair that identifies a mobile network.
