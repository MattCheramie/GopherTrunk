---
slug: p25-identifier-update
title: P25 Identifier Update (IDEN_UP)
entry_type: term
category: trunked-radio
description: The P25 Identifier Update (IDEN_UP) is the control-channel message that carries a site's band plan — base frequency, channel spacing, transmit offset, and bandwidth — so a decoder can turn a channel number into a tunable frequency.
keywords: P25 Identifier Update, IDEN_UP, band plan, channel plan, base frequency, channel spacing, transmit offset, IDEN_UP_VU, IDEN_UP_TDMA, P25 frequency, channel number to frequency
aka: [IDEN_UP, "identifier update", "band plan"]
autolink: true
infobox:
  - { label: Opcodes, value: "0x3D / 0x34 / 0x33" }
  - { label: Defines, value: One channel-ID band-plan slot }
  - { label: Fields, value: Base, spacing, offset, bandwidth }
  - { label: Spec, value: TIA-102.AABF Tables 14/14a }
see_also: [p25-tsbk-opcodes, channel-grant, tsbk, control-channel, p25-phase-2, system-id, p25-phase-1]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **P25 Identifier Update** (**IDEN_UP**) is the control-channel message that defines a
site's **band plan** — the mapping from an abstract channel number to an actual downlink
frequency.[^wiki] A [channel grant](/reference/channel-grant/) never names a frequency
directly; it carries a 4-bit *channel ID* and a 12-bit *channel number*. The IDEN_UP
supplies the base frequency and step size for each channel ID, so a decoder can compute
`frequency = base + number × spacing` and retune to follow a call. A site repeats these
alongside its status broadcasts, and a receiver accumulates one slot per channel ID.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="An Identifier Update supplies a base frequency and channel spacing for a channel ID; a channel grant's channel number is multiplied by the spacing and added to the base to produce the downlink frequency a receiver tunes." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="200" height="44" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="120" y="38" text-anchor="middle" font-size="8.5" fill="currentColor">IDEN_UP (channel ID 1)</text>
  <text x="120" y="52" text-anchor="middle" font-size="7.5" fill="currentColor">base = 851.0 MHz · step = 12.5 kHz</text>
  <rect x="260" y="20" width="190" height="44" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="355" y="38" text-anchor="middle" font-size="8.5" fill="currentColor">grant: ID 1, number 328</text>
  <text x="355" y="52" text-anchor="middle" font-size="7.5" fill="currentColor">talkgroup 4001</text>
  <path d="M120 64 L120 96 L260 96 M355 64 L355 96 L260 96" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="150" y="96" width="220" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="260" y="115" text-anchor="middle" font-size="8.5" fill="currentColor">851.0 MHz + 328 × 12.5 kHz = 855.1 MHz</text>
  <text x="260" y="140" text-anchor="middle" font-size="7.5" fill="currentColor">receiver tunes here to hear the call</text>
</svg>
<figcaption>The band plan and the grant are separate messages; only together do they yield a frequency, which is why a decoder must collect Identifier Updates before it can act on grants.</figcaption>
</figure>

## The three variants

Three opcodes carry the same frequency fields with different byte-0 semantics, all parsed
into one `IdentifierUpdate` struct in `internal/radio/p25/phase1/identifier.go`:

| Opcode | Mnemonic | Form | Byte-0 low nibble |
|---|---|---|---|
| `0x3D` | IDEN_UP | 700/800/900 MHz | 9-bit bandwidth (×125 Hz), 9-bit offset (×250 kHz) |
| `0x34` | IDEN_UP_VU | VHF/UHF | 4-bit bandwidth code, sign + 13-bit offset (× spacing) |
| `0x33` | IDEN_UP_TDMA | Phase 2 TDMA | 4-bit channel-type code (slot count + access mode) |

## The fields

| Field | Meaning | Encoding |
|---|---|---|
| Channel ID | 4-bit band-plan slot (0–15) | byte 0 high nibble |
| Base frequency | Downlink frequency for channel 0 | 32-bit field × 5 Hz |
| Channel spacing | Channel-to-channel step | 10-bit field × 125 Hz |
| Transmit offset | Signed uplink offset (uplink = downlink + offset) | ±, unit varies by variant |
| Bandwidth | Nominal channel bandwidth | informational only |

The base frequency is transmitted in units of 5 Hz, and the channel spacing in units of
125 Hz. The transmit offset gives the uplink relative to the downlink — a monitor listening
to the downlink does not need it, but it completes the picture of the channel. Bandwidth is
carried for logging only: GopherTrunk's frequency resolver uses just base, spacing, and the
grant's channel number. The [Phase 2](/reference/p25-phase-2/) TDMA variant (`0x33`)
additionally flags the channel ID as TDMA, which the Phase 1 control channel uses to route a
grant on that channel into the Phase 2 voice path rather than the Phase 1 FDMA path.

## Relevance to SDR

GopherTrunk's `BandPlan` accumulates one `IdentifierUpdate` per channel ID and exposes
`Frequency(channelID, channelNumber)` to resolve a grant. Until an IDEN_UP has been seen for
a channel ID, grants on it cannot be tuned — the resolver returns `ErrUnknownChannelID` and
the control channel records a `no-bandplan` decode gap so the missing band plan is visible in
metrics. Because a P25 site rebroadcasts its Identifier Updates continuously, a scanner that
camps on the control channel long enough learns the full band plan and can then follow any
grant. `Snapshot` surfaces the accumulated slots so a discovered system's band plan can be
documented and exported.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard. Field encodings follow TIA-102.AABF Tables 14/14a/16, cross-checked against OP25 and SDRTrunk.
