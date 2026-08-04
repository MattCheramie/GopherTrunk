---
slug: p25-header-data-unit
title: P25 Header Data Unit
entry_type: term
category: trunked-radio
description: The P25 Header Data Unit (HDU) opens a voice transmission, carrying the 72-bit Message Indicator plus the algorithm, key, and talkgroup IDs a receiver needs to set up decryption, protected by inner Golay and outer RS(36,20,17).
keywords: P25 HDU, header data unit, message indicator, ALGID, KID, key ID, TGID, RS 36 20, Golay HDU, encryption header, TIA-102 BAAA
aka: [HDU, "header data unit", "P25 voice header"]
autolink: true
infobox:
  - { label: DUID, value: 0x0 }
  - { label: Carries, value: "MI (72), ALGID, KID, TGID" }
  - { label: FEC, value: "Golay inner + RS(36,20,17) outer" }
  - { label: Spec, value: TIA-102.BAAA §7 }
see_also: [p25-reed-solomon, p25-encryption-sync, p25-logical-data-unit, golay-code, p25-nid-duid, talkgroup]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

The **P25 Header Data Unit** (**HDU**, [DUID](/reference/p25-nid-duid/) `0x0`) is the frame that
opens a P25 Phase 1 voice transmission, sent once before the stream of
[LDU](/reference/p25-logical-data-unit/) voice frames begins.[^wiki] Its job is setup: it
carries the 72-bit Message Indicator (the cryptographic initialisation vector) together with
the algorithm, key, and talkgroup identifiers a receiver needs to prepare decryption *before*
the first voice frame arrives. The whole payload is protected by an inner
[Golay](/reference/golay-code/) code and an outer [RS(36,20,17)](/reference/p25-reed-solomon/)
Reed-Solomon code.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The P25 HDU payload as a 72-bit Message Indicator followed by an 8-bit manufacturer ID, an 8-bit algorithm ID, a 16-bit key ID, and a 16-bit talkgroup ID, together 120 bits or 20 six-bit symbols, extended to 36 symbols by RS(36,20) parity." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="150" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="95" y="47" text-anchor="middle" font-size="9" fill="currentColor">Message Indicator · 72</text>
  <rect x="170" y="30" width="40" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="190" y="47" text-anchor="middle" font-size="8" fill="currentColor">MFID</text>
  <rect x="210" y="30" width="42" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="231" y="47" text-anchor="middle" font-size="8" fill="currentColor">ALGID</text>
  <rect x="252" y="30" width="46" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="275" y="47" text-anchor="middle" font-size="8" fill="currentColor">KID 16</text>
  <rect x="298" y="30" width="52" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="324" y="47" text-anchor="middle" font-size="8" fill="currentColor">TGID 16</text>
  <rect x="350" y="30" width="90" height="26" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/>
  <text x="395" y="47" text-anchor="middle" font-size="8" fill="currentColor">RS parity</text>
  <text x="20" y="80" font-size="8" fill="currentColor">120 information bits = 20 GF(2⁶) symbols → RS(36,20,17) → 36 symbols, each Golay-protected on air</text>
</svg>
<figcaption>The HDU packs a 72-bit Message Indicator and the ALGID/KID/TGID identifiers into 120 information bits — twenty six-bit symbols — which RS(36,20,17) extends to 36 symbols, each further protected by an inner Golay codeword.</figcaption>
</figure>

## How it works

After the [frame sync](/reference/p25-frame-sync-word/) and NID, the HDU's information field is
120 bits, which map exactly onto the 20 six-bit information symbols of the outer RS code:

| Field | Bits | Purpose |
|-------|------|---------|
| Message Indicator (MI) | 72 | Crypto initialisation vector / keystream sync |
| Manufacturer ID (MFID) | 8 | Vendor namespace |
| Algorithm ID (ALGID) | 8 | Encryption algorithm (0x80 = clear/unencrypted) |
| Key ID (KID) | 16 | Which key within the algorithm |
| Talkgroup ID (TGID) | 16 | Destination [talkgroup](/reference/talkgroup/) |

Those 20 symbols are encoded to a 36-symbol RS(36,20,17) codeword — able to correct up to 8
symbol errors — and each symbol is then wrapped in an inner shortened
[Golay](/reference/golay-code/) codeword and interleaved for transmission. On receive the chain
reverses: Golay-decode each inner codeword, reassemble the 36 RS symbols, and RS-decode to
recover the 20 information symbols. The MI and ALGID/KID together tell the receiver whether the
following voice is encrypted and, if so, how to initialise the keystream; a clear-mode HDU
(ALGID 0x80) simply announces the talkgroup.

## In practice

The HDU is a convenience, not a necessity, for a scanner. The same talkgroup and encryption
identifiers reappear inside every [LDU2](/reference/p25-logical-data-unit/)'s
[Encryption Sync](/reference/p25-encryption-sync/) field throughout the call, so a decoder that
tunes in mid-transmission — having missed the one-shot header — still recovers them. What the
HDU adds is *early* setup: a receiver that catches it can begin decryption from the very first
voice frame rather than waiting for the next LDU2.

GopherTrunk today implements the HDU's **protecting code** — the RS(36,20,17) encoder,
verifier, and corrector in the framing package — but does **not** yet have a dedicated HDU frame
parser that strips the inner Golay layer, reassembles the payload, and surfaces the MI/ALGID/
KID/TGID fields. The outer FEC primitive is in place and round-trip tested; consuming a full
on-air HDU frame is the remaining gap.

## Relevance to SDR

The RS(36,20,17) support lives in `internal/radio/framing/rs_gf64.go` alongside the Link Control
and Encryption Sync codes, sharing the same GF(2⁶) field and Berlekamp-Massey / Chien / Forney
decoder. Because GopherTrunk reads the same encryption metadata from each call's LDU2
[Encryption Sync](/reference/p25-encryption-sync/) fields, missing the HDU parser does not lose
information over the life of a call — but a dedicated HDU parser would let the decoder report a
call's algorithm and talkgroup from its opening frame instead of waiting for the first LDU2.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 data units.
[^rs]: [Reed-Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, on the RS(36,20,17) outer code that protects the HDU.
