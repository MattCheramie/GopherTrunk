---
slug: motorola-osw
title: Motorola OSW
entry_type: term
category: trunked-radio
description: The Motorola Outbound Status Word (OSW) is the 32-bit control-channel signalling block of Motorola Type II SmartNet/SmartZone systems, carrying a 16-bit address and a command field split into a 12-bit opcode and a 4-bit logical channel, protected by two BCH(64,16,11) codewords.
keywords: Motorola OSW, outbound status word, SmartNet, SmartZone, Motorola Type II, BCH 64 16 11, control channel opcode, group voice channel grant, Motorola sync A4D7AA
aka: [OSW, "outbound status word", "Motorola OSW"]
autolink: true
infobox:
  - { label: Length, value: 32 bits (Address 16 + Command 16) }
  - { label: Command, value: 12-bit opcode + 4-bit LCN }
  - { label: Sync, value: "0xA4D7AA (24 bits)" }
  - { label: FEC, value: "two BCH(64,16,11) codewords" }
see_also: [motorola-type-ii, smartnet-smartzone, trunked-radio, control-channel, channel-grant, bch-code, edacs-control-channel-word, mpt-1327-codeword, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/BCH_code
---

The **Motorola OSW** (**Outbound Status Word**) is the 32-bit control-channel signalling
block of [Motorola Type II](/reference/motorola-type-ii/) trunking — the
[SmartNet / SmartZone](/reference/smartnet-smartzone/) systems that dominate US public-safety
radio.[^trs] Each OSW rides the outbound [control channel](/reference/control-channel/) after
sync, BCH error correction, and de-interleaving, and announces one piece of system business:
a voice or data [channel grant](/reference/channel-grant/), a system-ID broadcast, an
adjacent-site pointer, an affiliation, or an idle heartbeat. The 32 information bits are
carried by two [BCH(64,16,11)](/reference/bch-code/) codewords so the word survives a noisy
channel.[^bch]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A Motorola control-channel frame: a 24-bit sync word 0xA4D7AA followed by two BCH(64,16,11) codewords whose 16 information bits each combine into the 32-bit OSW, which splits into a 16-bit address and a 16-bit command that itself holds a 12-bit opcode and a 4-bit logical channel number." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="24" width="66" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="49" y="41" text-anchor="middle" font-size="8" fill="currentColor">sync · 24</text>
  <rect x="90" y="24" width="120" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="150" y="41" text-anchor="middle" font-size="8" fill="currentColor">BCH(64,16,11) #1</text>
  <rect x="210" y="24" width="120" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="270" y="41" text-anchor="middle" font-size="8" fill="currentColor">BCH(64,16,11) #2</text>
  <path d="M90 64 L210 64" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  <path d="M210 64 L330 64" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  <rect x="90" y="78" width="150" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="165" y="95" text-anchor="middle" font-size="8" fill="currentColor">Address 16</text>
  <rect x="240" y="78" width="130" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="290" y="95" text-anchor="middle" font-size="8" fill="currentColor">opcode 12</text>
  <rect x="370" y="78" width="60" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="400" y="95" text-anchor="middle" font-size="8" fill="currentColor">LCN 4</text>
  <text x="240" y="122" text-anchor="middle" font-size="7.5" fill="currentColor">Command = opcode (Command &gt;&gt; 4) · LCN (Command &amp; 0xF)</text>
</svg>
<figcaption>Each Motorola control frame carries two BCH(64,16,11) codewords whose 16-bit info fields combine into the 32-bit OSW: a 16-bit address plus a command that splits into a 12-bit opcode and a 4-bit logical channel number.</figcaption>
</figure>

## Field layout

The OSW is 32 bits: the upper 16 carry an **Address** (a talkgroup or radio ID, depending on
the opcode), and the lower 16 carry a **Command** that combines a 12-bit opcode with a 4-bit
per-opcode parameter. For voice and data grants that nibble is the LCN — the logical channel
number that indexes the operator's band plan. GopherTrunk derives the opcode as
`Command >> 4` and the LCN as `Command & 0xF`.

## Opcodes

The Motorola opcode space is large and vendor-extended; GopherTrunk models the SmartZone
subset most useful for trunk-following:

| Opcode | Value | Meaning |
|---|---|---|
| GroupVoiceChannelGrant | 0x308 | Group voice [channel grant](/reference/channel-grant/) |
| GroupVoiceChannelGrantUpdate | 0x309 | Grant continuation / update |
| PrivateCallGrant | 0x30B | Unit-to-unit call grant |
| DataChannelGrant | 0x310 | Data channel grant |
| AffiliationResponse | 0x320 | Unit affiliation response |
| AdjacentSiteStatus | 0x31B | Neighbour-site announcement |
| SystemIDExtended | 0x080 | Extended system-ID broadcast |
| Encryption | 0x140 | Encryption signalling |
| Emergency | 0x300 | Emergency call |
| Idle | 0x28D / 0x290 | Control-channel idle / heartbeat |

A voice grant carries the talkgroup in the Address field and the LCN in the Command nibble;
the follower maps that LCN through the band plan and retunes. A `SystemIDExtended` OSW puts
the system identifier in the Address field, with the Command nibble carrying a system-ID
variant. Strict-mode operators reject any OSW whose opcode is outside the recognised set —
an unknown opcode signals bit errors or a misaligned codeword pair.

## Forward error correction

Each OSW frame carries **two BCH(64,16,11) codewords** back-to-back. A BCH(64,16,11) codeword
is the 63-bit BCH(63,16,11) codeword — 16 information bits, correcting up to 11 errors — plus
a trailing overall-even-parity bit, giving 64 wire bits. GopherTrunk's `BCHDecode64_16`
splits off the parity bit, runs the inner BCH(63,16,11) decoder over the top 63 bits, then
recomputes the parity over the corrected codeword and folds any mismatch into the error count.
The two codewords' 16-bit info fields combine into the OSW's 32-bit {Address, Command} field.
The 24-bit outbound sync is **0xA4D7AA** (binary `1010 0100 1101 0111 1010 1010`); a sliding
[detector](/reference/frame-synchronization/) locks onto it before the adapter slices the
post-sync codewords.

## In practice

The field-position and opcode assignments follow the most-cited public reference (sdrtrunk),
not a paywalled Motorola specification, so they are best-effort — cross-check against a fresh
live capture before trusting an unusual opcode. Because Motorola systems continuously
re-announce active calls, a monitor recovers most activity even under marginal signal: the
BCH pair repairs typical bit hits, and the redundant grant stream fills gaps a dropped frame
would otherwise leave.

## Relevance to SDR

`internal/radio/motorola/osw.go` holds the OSW packing (`ParseOSW`, `OSWFromBits`);
`opcodes.go` interprets the Command field into structured grants, system IDs, and adjacent-site
descriptors; and `internal/radio/framing/bch.go` (`BCHDecode64_16`) implements the FEC. These
let GopherTrunk follow a Motorola Type II / SmartZone system — read the grant OSWs, map each
LCN through the band plan, and retune to the assigned voice channel in step with the radios.

## Sources

[^trs]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on trunking and the Motorola Type II / SmartNet family.
[^bch]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, on the error-correcting code that protects each OSW codeword.
