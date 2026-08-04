---
slug: p25-phase-2-sync-word
title: P25 Phase 2 sync word
entry_type: term
category: synchronization
description: The P25 Phase 2 sync words are the 20-dibit patterns (outbound 0x575D57F7FF, inbound 0xDFF57D75DF5D) that mark sub-frame boundaries in the two-slot TDMA dibit stream; a canonical [0,1,3,2] dibit remap undoes the DQPSK quadrant transpose before the search.
keywords: P25 Phase 2 sync word, 0x575D57F7FF, frame sync magic, 20 dibit sync, canonical dibit remap, DQPSK quadrant transpose, TDMA sync search, TIA-102.BBAC
aka: ["Phase 2 frame sync", "P25P2 frame sync magic", "outbound sync word"]
autolink: true
infobox:
  - { label: Length, value: 20 dibits (40 bits) }
  - { label: Outbound, value: "0x575D57F7FF" }
  - { label: Inbound, value: "0xDFF57D75DF5D" }
  - { label: Spec, value: TIA-102.BBAC }
see_also: [p25-phase-2, p25-phase-2-hdqpsk, tdma, pi-4-dqpsk, differential-decoding, p25-isch, p25-phase-2-superframe]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

The **P25 Phase 2 sync word** is the fixed 20-dibit (40-bit) pattern that opens a Phase 2
sub-frame and tells the decoder where a burst begins in the [H-DQPSK](/reference/p25-phase-2-hdqpsk/)
dibit stream.[^wiki] Two patterns exist because the link is [TDMA](/reference/tdma/) and two-directional:
the **outbound** (base → subscriber) sync is `0x575D57F7FF` and the **inbound** (subscriber → base)
sync is `0xDFF57D75DF5D`, so a receiver locks onto the correct side of the link.[^fsync] The outbound
value is the authoritative TIA-102.BBAC frame-sync magic, the same constant OP25 and SDRTrunk carry.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 160" role="img" aria-label="The raw DQPSK slicer emits the sync word with its two negative-phase dibits transposed, reading 0x565956A6AA; a fixed [0,1,3,2] remap swaps the dibit values 2 and 3 to recover the canonical outbound value 0x575D57F7FF that the detector matches." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="26" width="200" height="26" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="120" y="43" text-anchor="middle" font-size="8.5" fill="currentColor">slicer output 0x565956A6AA</text>
  <path d="M228 39 L262 39" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M256 35 L262 39 L256 43" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <rect x="268" y="26" width="182" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="359" y="43" text-anchor="middle" font-size="8.5" fill="currentColor">canonical 0x575D57F7FF</text>
  <text x="245" y="20" text-anchor="middle" font-size="7.5" fill="currentColor">[0,1,3,2]</text>
  <text x="20" y="82" font-size="8" fill="currentColor">remap fixes dibit labels 2 ↔ 3 (the two negative-phase quadrants) for the WHOLE stream</text>
  <rect x="20" y="100" width="90" height="24" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
  <text x="65" y="116" text-anchor="middle" font-size="8" fill="currentColor">sync · 20 dibits</text>
  <rect x="110" y="100" width="60" height="24" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="140" y="116" text-anchor="middle" font-size="8" fill="currentColor">ISCH</text>
  <rect x="170" y="100" width="230" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="285" y="116" text-anchor="middle" font-size="8" fill="currentColor">MAC / voice payload</text>
  <text x="20" y="146" font-size="8" fill="currentColor">a sliding detector matches within a dibit-mismatch tolerance to anchor the sub-frame grid</text>
</svg>
<figcaption>The differential slicer emits the sync 2↔3 transposed; a fixed [0,1,3,2] remap restores the standard value so the detector, ISCH and MAC FEC all match the same canonical bits.</figcaption>
</figure>

## The dibit remap

Phase 2 rides a [differentially decoded](/reference/differential-decoding/) constellation, and the
quadrant slicer GopherTrunk uses assigns the two *negative-phase* symbols the dibit values 3 and 2
where the TIA-102 dibit convention uses 2 and 3. The consequence is subtle: the raw slicer output for
genuine air is the true bits with **2 and 3 transposed**, so the standard sync `0x575D57F7FF` comes
off the wire reading `0x565956A6AA`. Rather than matching the transposed constant — which would lock
sync but leave the whole MAC and voice payload swapped, so fields like ALGID/KID never populate — the
Phase 2 receiver applies a fixed **`[0,1,3,2]` remap** (`canonicalDibitRemap`, mirroring the verified
Phase 1 CQPSK remap from issue #492) to *every* dibit before emitting it. After the remap the sync
detector, the [ISCH](/reference/p25-isch/) decode, and the MAC FEC all match against the authoritative
canonical values, and the transpose is corrected once for the entire chain.

A historical bug is instructive: before this was understood, the outbound constant was a garbled
48-bit value (`0x575F7DFF77FF`) that was neither the standard nor its transpose. It never matched real
air, so Phase 2 superframes never locked — yet every synthesized round-trip test passed, because the
test encoder used the same wrong constant on both ends (a self-consistent bug). The fix pins the sync
to exactly 20 dibits (40 bits) from the low bits of the constant.

## Sync search on the TDMA slots

`SyncDetector` slides a 20-dibit window over the demodulated stream and reports each absolute dibit
index where the pattern matches within a configurable *tolerance* of dibit mismatches (the Phase 2
control path uses a tolerance of 2). This is the same detector shape as the Phase 1, DMR and NXDN sync
searches, so the higher-level state machine stays uniform across protocols. Every hit anchors a
sub-frame: the [superframe](/reference/p25-phase-2-superframe/) decoder places its 0..11 sub-frame grid
on a sync match, then reads the ISCH that immediately follows to learn whether the sub-frame carries
voice or a MAC PDU. Because the outbound sync recurs on a fixed cadence within the superframe, a stable
lock lets the decoder track both TDMA slots without re-hunting.

The inbound sync is used only by a diagnostic detector and is not yet verified against real uplink
captures or this decoder's dibit convention — GopherTrunk does not rely on it for decode.

## Relevance to SDR

`internal/radio/p25/phase2/sync.go` holds the canonical outbound and inbound constants,
`OutboundSyncDibits`/`InboundSyncDibits` materialisers, and the `SyncDetector`. Getting both the
constant *and* the `[0,1,3,2]` remap right is what turns a locked-but-garbled Phase 2 channel into one
whose MAC opcodes and voice actually decode; the sync word is the landmark everything downstream is
sliced relative to. The spec is TIA-102.BBAC.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 framing.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on locating frame boundaries with a fixed sync sequence.
