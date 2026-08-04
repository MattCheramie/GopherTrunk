---
slug: dmr-mbc
title: DMR Multi-Block Control
entry_type: term
category: trunked-radio
description: A DMR Multi-Block Control (MBC) message spreads a control payload too large for one CSBK across a header burst and one or more continuation bursts, the last flagged LB=1, each block carried by the same BPTC coding as a CSBK.
keywords: DMR MBC, Multi-Block Control, MBC header, MBC continuation, last block LB, BPTC, ETSI TS 102 361-1, Tier III control
aka: ["MBC", "Multi-Block Control", "multi-block CSBK"]
autolink: true
infobox:
  - { label: Structure, value: header + continuation(s) }
  - { label: Terminator, value: "last block flagged LB=1" }
  - { label: Per-block FEC, value: BPTC(196,96), like a CSBK }
  - { label: Spec, value: TS 102 361-1 §9 / 361-4 }
see_also: [csbk, dmr-tier-3, bptc, channel-grant, control-channel, color-code, dmr-csbk-payloads]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236101/
---

A **DMR Multi-Block Control** (**MBC**) message carries a control payload too large to fit in
a single [CSBK](/reference/csbk/).[^wiki] Where a CSBK is one 96-bit block, an MBC spreads its
content across a **header block** followed by one or more **continuation blocks**, with the
final block flagged as the last (LB=1). Each block is carried by the same
[BPTC(196,96)](/reference/bptc/) coding as a CSBK, so a receiver decodes and error-checks every
block the same way, then reassembles the whole message once the terminator arrives.[^etsi]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A Multi-Block Control message shown as a header block carrying the opcode and feature ID, followed by two continuation blocks, the last of which sets the last-block flag; a bracket beneath shows the blocks reassembled into one control payload." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="24" y="26" width="120" height="30" fill="currentColor" fill-opacity="0.22"/><text x="84" y="41">Header</text><text x="84" y="52" font-size="7">opcode · FID</text>
    <rect x="156" y="26" width="120" height="30" fill="none"/><text x="216" y="45">Continuation</text>
    <rect x="288" y="26" width="120" height="30" fill="currentColor" fill-opacity="0.14"/><text x="348" y="41">Continuation</text><text x="348" y="52" font-size="7">LB = 1</text>
  </g>
  <path d="M24 66 L24 74 L408 74 L408 66" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <path d="M216 74 L216 84" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="216" y="98" text-anchor="middle" font-size="8" fill="currentColor">reassembled control payload</text>
  <text x="216" y="112" text-anchor="middle" font-size="7.5" fill="currentColor">each block BPTC-decoded and CRC-checked before assembly</text>
</svg>
<figcaption>An MBC begins with a header block and ends at the block whose last-block flag is set; the receiver concatenates the blocks, each independently BPTC-protected, into one message.</figcaption>
</figure>

## Assembly and lifecycle

The assembler keys each in-progress message on the burst's [colour code](/reference/color-code/),
so two messages interleaved on the same physical channel but different colour codes do not
collide. A header block (slot-type `DTMBCHeader`) opens a fresh assembly and records the CSBK-style
leader — the last-block and protect flags, the CSBKO opcode, and the feature-set ID — from its
first two octets. Each continuation block (`DTMBCContinuation`) appends its raw content. When a
block arrives with LB=1, the assembly is closed and dispatched; the opcode and reassembled
payload identify the message.

Two guards keep a lost burst from wedging the decoder. A block limit (8) caps an assembly whose
terminator never arrives, and an age limit (one second) evicts a stale partial assembly so a
dropped LB=1 on one message cannot shadow the next header on the same colour code. A continuation
that arrives with no open header — the receiver joined mid-message or lost the header burst — has
nothing to attach to and is discarded. This mirrors the general trunking posture: never act on a
half-assembled control message.

## What GopherTrunk parses

The header block's octets 2–9 reuse the validated 8-octet CSBK grant layout, so the
channel-grant opcodes (private and talkgroup voice grants, and the data grants) are structurally
parsed straight out of an MBC header and published exactly as their single-block CSBK
counterparts would be — a voice grant retunes and follows, a data grant is only recorded for the
band-plan learner. Every other opcode is assembled and surfaced at debug level but not
force-parsed. This is deliberate honesty: there is no real off-air MBC capture pinning the
multi-block payload layouts or the last-block CRC convention, and the CSBK CRC history in this
codebase is a standing warning that an unverified convention can silently reject every genuine
frame. Rather than guess a layout and emit garbage, the assembler leans on each block's BPTC FEC
for protection and logs the unrecognised message "pending capture validation" — the same cautious
stance the Connect Plus and Hytera vendor CSBKs get.

## Relevance to SDR

`internal/radio/dmr/tier3/mbc.go` implements the assembler: `ParseMBCBlock` reads one decoded
12-byte block, `handleMBC` routes header and continuation bursts and enforces the block and age
limits, and `dispatchMBC` concatenates the blocks and dispatches the grant opcodes through the
shared `ParseTVGrant` / `ParsePVGrant` / `ParseDataGrant` parsers. Because each block passes
through the same `decodeInfoBlock` BPTC path as a CSBK, a burst too corrupted to correct is
rejected before it ever reaches the assembler, so an MBC is only ever built from blocks that
already cleared their per-block FEC. The result is a decoder that can follow the large control
messages standard Tier III occasionally needs without pretending to understand payload layouts it
has not yet been able to validate against real signals.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR control signalling and Tier III trunking.
[^etsi]: [ETSI TS 102 361-1](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236101/) — ETSI, §9 defining the multi-block control structure carried over the DMR air interface.
