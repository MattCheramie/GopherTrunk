---
slug: p25-nid-duid
title: P25 NID & DUID
entry_type: algorithm
category: trunked-radio
description: The P25 Network ID (NID) is the 64-bit block after the frame sync that carries the 12-bit NAC and the 4-bit Data Unit ID (DUID), protected by a BCH(63,16) code so a decoder learns which system and which frame type it is looking at.
keywords: P25 NID, DUID, data unit ID, network access code, NAC, BCH 63 16, P25 frame type, HDU LDU TSDU, P25 Phase 1 NID
aka: [NID, DUID, "network identifier", "data unit ID"]
autolink: true
infobox:
  - { label: Size, value: 64 bits after the FSW }
  - { label: Carries, value: NAC (12) + DUID (4) }
  - { label: FEC, value: "BCH(63,16,11) + 1 flag bit" }
  - { label: Spec, value: TIA-102.BAAA §6.2 }
see_also: [p25-frame-sync-word, network-access-code, bch-code, tsbk, p25-logical-data-unit, p25-phase-1]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/BCH_code
---

The **P25 NID** (**Network Identifier**) is the 64-bit block that immediately follows the
[frame sync word](/reference/p25-frame-sync-word/) in every P25 Phase 1 frame.[^wiki] It
carries two things a decoder needs before it can do anything else: the 12-bit
[Network Access Code](/reference/network-access-code/) (NAC) that identifies the system, and
the 4-bit **DUID** (**Data Unit ID**) that names the frame type to follow — a voice frame, a
terminator, or a [trunking](/reference/tsbk/) block. The whole thing is wrapped in a
[BCH(63,16,11)](/reference/bch-code/) code so it survives a noisy channel.[^bch]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The 64-bit P25 NID broken into a 12-bit NAC, a 4-bit DUID, a 47-bit BCH parity field, and a single trailing flag bit that depends on the DUID rather than being an overall parity check." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="52" text-anchor="middle" font-size="9" fill="currentColor">NAC 12</text>
  <rect x="90" y="34" width="46" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="113" y="52" text-anchor="middle" font-size="9" fill="currentColor">DUID 4</text>
  <rect x="136" y="34" width="250" height="28" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="261" y="52" text-anchor="middle" font-size="9" fill="currentColor">BCH(63,16) parity · 47 bits</text>
  <rect x="386" y="34" width="54" height="28" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="413" y="49" text-anchor="middle" font-size="8" fill="currentColor">flag</text>
  <text x="413" y="58" text-anchor="middle" font-size="6.5" fill="currentColor">bit 63</text>
  <text x="20" y="86" font-size="8" fill="currentColor">63 codeword bits corrected by BCH · bit 63 is a per-DUID flag, NOT overall parity</text>
</svg>
<figcaption>The NID packs the NAC and DUID into a 16-bit message, BCH-encodes it to 63 bits, and appends one flag bit whose value is fixed by the DUID — a detail that is easy to get wrong.</figcaption>
</figure>

## How it works

The 16 information bits (12-bit NAC followed by 4-bit DUID) are encoded with a shortened
**BCH(63,16,11)** code — 63 bits able to correct up to 11 bit errors — and a 64th bit is
appended. A receiver runs the BCH decoder over the first 63 bits to recover the NAC and
DUID; if the codeword is beyond the correction radius, the frame is dropped. The DUID then
tells the deframer what comes next: `0x0` HDU, `0x3` TDU, `0x5` LDU1, `0x7` TSDU (the
control channel), `0xA` LDU2, `0xC` PDU, `0xF` TDULC.

The trap is the 64th bit. It is **not** an overall parity check across the 63-bit codeword —
the obvious assumption, and the one that masked GopherTrunk issue #275. It is a fixed flag
whose value is dictated by the DUID: 1 for the two voice frames (LDU1, LDU2) and 0 for
everything else. A decoder that validates it as parity will accept some corrupt frames and
reject some good ones. GopherTrunk encodes the rule directly:

```go
// expectedNIDParity returns the value of the 64th NID bit a P25 Phase 1
// transmitter sets for a given DUID — NOT an overall parity over the
// 63-bit codeword. Per TIA-102.BAAA Annex A and confirmed against OP25.
func expectedNIDParity(duid DUID) byte {
    switch duid {
    case DUIDLogicalLink1, DUIDLogicalLink2:
        return 1
    default:
        return 0
    }
}
```

## In practice

Because the NID is so short and so heavily protected, it is the most reliable thing on a P25
channel — a decoder can usually recover the NAC and DUID even when the frame body is too
corrupted to use. GopherTrunk exploits that: when it cannot fully align a frame, it inspects
*where* the residual bit errors cluster across the 32 NID dibits to tell apart a post-sync
timing slip (errors bunched at one end), a status-symbol phase fault (errors near the tail),
and plain SNR-limited demod corruption (errors spread evenly). That per-dibit error pattern
is how the closest-miss diagnostics localise a lock problem.

## Relevance to SDR

`internal/radio/p25/phase1/nid.go` implements the full NID path: `ParseNID` reads 64 bits,
runs `framing.BCHDecode63_16`, extracts the NAC and DUID, and validates the per-DUID flag;
`NIDFromDibitsWithErrors` additionally returns the 32-entry per-dibit error pattern the
diagnostics use. Getting the flag rule right — flag, not parity — is exactly the kind of
spec detail that is invisible in synthetic round-trip tests (which set the bit the same wrong
way on both ends) yet decides whether real off-air frames decode. Every P25 Phase 1 frame is
gated on a clean NID, so this small block sits on the critical path of the whole decoder.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 data units.
[^bch]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, on the error-correcting code family that protects the NID.
