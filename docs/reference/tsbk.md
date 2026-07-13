---
slug: tsbk
title: TSBK
entry_type: term
category: trunked-radio
description: A TSBK (trunking signalling block) is the unit of control-channel signalling in P25 — the message that announces channel grants, registrations, and system parameters.
keywords: TSBK, trunking signalling block, P25 control channel, channel grant, single block, signalling, opcode, CRC, multi-block
aka: [TSBK, "trunking signalling block", "trunking signaling block"]
autolink: true
see_also: [control-channel, channel-grant, project-25, p25-phase-1, csbk, cyclic-redundancy-check, system-id, wacn]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
---

A **TSBK** (**trunking signalling block**) is the unit of control-channel signalling in
[P25](/reference/project-25/).[^wiki] Each TSBK is a short, error-protected message on the
[control channel](/reference/control-channel/) carrying one piece of system business — a
[channel grant](/reference/channel-grant/), a registration, an
[affiliation](/reference/affiliation/), or a system parameter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A control channel carrying a stream of TSBK blocks, one of which is a channel grant." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="400" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="46" text-anchor="middle" font-size="8.5" fill="currentColor">P25 control channel</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="66" width="80" height="26" fill="none"/><rect x="130" y="66" width="80" height="26" fill="currentColor" fill-opacity="0.22"/><rect x="220" y="66" width="80" height="26" fill="none"/><rect x="310" y="66" width="80" height="26" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="80" y="83">TSBK</text><text x="170" y="80">TSBK</text><text x="170" y="90" font-size="7">(grant)</text><text x="260" y="83">TSBK</text><text x="350" y="83">TSBK</text></g>
</svg>
<figcaption>The P25 control channel is a stream of TSBKs; one announces a grant pointing radios to a voice channel.</figcaption>
</figure>

## How it works

A TSBK is a fixed-length block: a manufacturer/feature flag, an *opcode* naming the message
type, a payload of arguments, and a [CRC](/reference/cyclic-redundancy-check/) so a receiver
can reject blocks corrupted by noise. The opcode tells the decoder how to interpret the
payload — a Group Voice Channel Grant carries a talkgroup and a channel number; a Unit
Registration Response carries a source ID; system-service messages carry the
[WACN](/reference/wacn/), [system ID](/reference/system-id/), and channel-plan parameters a
radio needs to map channel numbers to frequencies. The control channel sends these blocks
back-to-back at a steady rate, so a decoder simply reads block after block and acts on the
opcodes it recognises.

A radio (or monitor) validates each block's CRC, discards the bad ones, and reassembles
the rest into a running picture of the system: which calls are active, which units are
affiliated, and how the frequencies are laid out. Decoding TSBKs is exactly how a scanner
follows a P25 system — read the grant TSBKs and retune to the assigned
[voice channel](/reference/voice-channel/) in step with the radios.

## Variants

- **Single-block TSBK** — the common case; one message in one block.
- **Multi-block PDU** — longer control messages that span several blocks for larger
  payloads such as extended system data.
- **P25 Phase 2 MAC PDU** — Phase 2 replaces the TSBK with *MAC* messages that serve the
  same coordinating role on its TDMA control path.
- **DMR CSBK** — DMR's [CSBK](/reference/csbk/) is the direct counterpart on DMR Tier III.

## In practice

Because TSBKs are self-contained and CRC-checked, a monitor can extract useful activity
from a P25 control channel even under marginal signal — good blocks get through, bad ones
are dropped, and the stream is redundant enough that missed grants are often re-announced.
The opcode set is standardised, but manufacturers add proprietary opcodes (notably
Motorola), so a complete decoder tracks both the TIA-standard and vendor messages.

## Relevance to SDR

GopherTrunk demodulates the P25 control channel, deframes it into TSBKs, checks each CRC,
and dispatches on the opcode to drive its trunk-following engine — following grants,
recording affiliations, and learning the channel plan. Phase 2 systems are handled through
the analogous MAC messages. This TSBK parsing is the core of GopherTrunk's P25 support.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its control-channel signalling.
[^crc]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on the error-detecting code that protects each TSBK.
