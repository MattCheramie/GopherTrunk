---
slug: csbk
title: CSBK
entry_type: term
category: trunked-radio
description: A CSBK (control signalling block) is the single-block control message in DMR, carrying call setup, channel grants, and system data on a Tier III control channel.
keywords: CSBK, control signalling block, DMR control channel, Tier III, channel grant, signalling, BPTC, opcode, Capacity Plus
aka: [CSBK, "control signalling block", "control signaling block"]
autolink: true
see_also: [control-channel, channel-grant, dmr-tier-3, dmr, tsbk, bptc, rest-channel, capacity-plus]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/
---

A **CSBK** (**control signalling block**) is the single-block control message of
[DMR](/reference/dmr/). On a [Tier III](/reference/dmr-tier-3/) control channel, CSBKs
carry call requests, [channel grants](/reference/channel-grant/), and system data — the
DMR counterpart of P25's [TSBK](/reference/tsbk/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A DMR control channel carrying CSBK blocks, one of which grants a traffic channel and slot." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="400" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="46" text-anchor="middle" font-size="8.5" fill="currentColor">DMR Tier III control channel</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="66" width="80" height="26" fill="none"/><rect x="130" y="66" width="90" height="26" fill="currentColor" fill-opacity="0.22"/><rect x="230" y="66" width="80" height="26" fill="none"/><rect x="320" y="66" width="80" height="26" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="80" y="83">CSBK</text><text x="175" y="80">CSBK</text><text x="175" y="90" font-size="7">(grant)</text><text x="270" y="83">CSBK</text><text x="360" y="83">CSBK</text></g>
</svg>
<figcaption>DMR Tier III coordinates the system with CSBKs; a grant CSBK assigns a traffic channel and timeslot.</figcaption>
</figure>

## How it works

A CSBK is a single 96-bit DMR burst payload identified by a *CSBKO* (CSBK opcode) and a
feature-ID field, followed by the message's arguments. Like all DMR signalling, it is
protected by [BPTC](/reference/bptc/) coding and a CRC so a receiver can correct or reject
bursts damaged in transit. The opcode names the message — a grant CSBK carries the target
talkgroup or radio ID plus the assigned traffic channel and timeslot; other CSBKs handle
registration, acknowledgements, and system status. Following the grant CSBKs is how a
decoder tracks a trunked DMR system, just as TSBKs are followed on P25.

Because DMR is two-slot [TDMA](/reference/tdma/), a grant CSBK must specify both the RF
channel and which of its two slots the call will use. On a Tier III system with a dedicated
control channel this is straightforward; on lighter modes the control function itself moves,
which changes where the CSBKs live.

## Variants

- **Tier III standard CSBKs** — the ETSI-defined set for a dedicated-control trunked system.
- **Vendor / feature CSBKs** — Motorola *Capacity Plus* and *Connect Plus*, and Hytera
  systems, extend the CSBK set with proprietary opcodes; [Capacity Plus](/reference/capacity-plus/)
  in particular rotates control onto a [rest channel](/reference/rest-channel/) rather than
  dedicating one.
- **Preamble CSBK** — a lead-in block that wakes and addresses radios before a data or call
  sequence.

## In practice

The CSBK opcode set is the key to identifying and following a DMR trunked system: standard
Tier III systems are well documented, while Capacity Plus and Connect Plus require handling
their vendor-specific blocks and, for Capacity Plus, tracking the moving rest channel. As
with P25, a monitor validates each block and reconstructs system state from the CSBKs that
pass their CRC.

## Relevance to SDR

GopherTrunk deframes the DMR control channel, decodes CSBKs through their BPTC/CRC
protection, and dispatches on the opcode to follow grants onto the correct traffic channel
and slot. Handling both standard Tier III and common vendor CSBK variants is what lets it
track real-world trunked DMR deployments.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR control signalling and Tier III trunking.
[^etsi]: [ETSI TS 102 361-4 (DMR Tier III)](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/) — ETSI, the standard defining DMR trunking CSBK signalling.
