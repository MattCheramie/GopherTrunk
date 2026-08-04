---
slug: p25-phase-2-superframe
title: P25 Phase 2 superframe
entry_type: term
category: trunked-radio
description: The P25 Phase 2 superframe is the 360 ms frame that organises the two-slot TDMA traffic channel — 12 sub-frames of 30 ms (2160 dibits / 4320 bits) alternating between the two timeslots, each sub-frame typed by its ISCH as voice or MAC.
keywords: P25 Phase 2 superframe, 360 ms superframe, 12 sub-frames, 4320 bits, SlotType, two-slot TDMA, ultraframe, TIA-102.BBAB 7
aka: ["Phase 2 superframe", "P25P2 superframe"]
autolink: true
infobox:
  - { label: Duration, value: 360 ms }
  - { label: Structure, value: 12 sub-frames × 30 ms }
  - { label: Size, value: 2160 dibits (4320 bits) }
  - { label: Spec, value: TIA-102.BBAB §7 }
see_also: [tdma, p25-phase-2, p25-isch, p25-phase-2-sync-word, pn44-scrambler, p25-mac-pdu, p25-sacch-facch, p25-phase-2-hdqpsk]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

The **P25 Phase 2 superframe** is the 360 ms frame that organises the two-slot
[TDMA](/reference/tdma/) traffic channel.[^wiki] Running the
[H-DQPSK](/reference/p25-phase-2-hdqpsk/) waveform at 6000 symbols/second, the superframe carries **12
sub-frames of 30 ms each**, alternating between the two timeslots, and each sub-frame is typed by the
[ISCH](/reference/p25-isch/) that prefixes it as either a voice sub-frame or a MAC-signalling
sub-frame.[^tdma] Its span — 2160 dibits, or **4320 bits** — is exactly the period of the
[PN44 scrambler](/reference/pn44-scrambler/), which restarts at the top of every superframe, so the
superframe is the natural unit both the framing and the descrambler are keyed to.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A 360 millisecond superframe divided into 12 sub-frames of 30 milliseconds, alternating between timeslot 1 and timeslot 2; each sub-frame is 180 dibits and is typed voice or MAC by its ISCH, and the PN44 scrambler restarts at the start of the superframe." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="20" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/><text x="38" y="49">TS1</text>
    <rect x="56" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="74" y="49">TS2</text>
    <rect x="92" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/><text x="110" y="49">TS1</text>
    <rect x="128" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="146" y="49">TS2</text>
    <rect x="164" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/><text x="182" y="49">TS1</text>
    <rect x="200" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="218" y="49">TS2</text>
    <rect x="236" y="34" width="18" height="24" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/><text x="245" y="49">…</text>
    <rect x="254" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/><text x="272" y="49">TS1</text>
    <rect x="290" y="34" width="36" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="308" y="49">TS2</text>
  </g>
  <text x="20" y="78" font-size="8" fill="currentColor">12 sub-frames × 30 ms = 360 ms · 180 dibits each · 2160 dibits = 4320 bits</text>
  <path d="M20 92 L20 100 L326 100 L326 92" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="173" y="114" text-anchor="middle" font-size="8" fill="currentColor">PN44 scrambler restarts here, once per superframe</text>
  <text x="20" y="138" font-size="7.5" fill="currentColor" font-style="italic">the 12-superframe "ultraframe" (≈4.32 s) is spec-defined but not modeled in GopherTrunk</text>
</svg>
<figcaption>The 360 ms superframe holds 12 alternating-timeslot sub-frames (4320 bits total); the PN44 sequence resets at its start, and its ISCH typing steers each sub-frame to voice or MAC decode.</figcaption>
</figure>

## Timing and structure

The superframe's dimensions all follow from the 6000 sym/s symbol rate:

| Quantity | Value |
| --- | --- |
| Superframe duration | 360 ms |
| Sub-frame duration | 30 ms |
| Sub-frames per superframe | 12 |
| Timeslots | 2 (alternating) |
| Symbol rate | 6000 sym/s |
| Dibits per sub-frame | 180 |
| Dibits per superframe | 2160 |
| Bits per superframe | 4320 |

Each sub-frame is composed of either a voice slot (4 voice frames, or 2 voice frames plus MAC) or a MAC
slot, and the [ISCH](/reference/p25-isch/)'s 4-bit `SlotType` names which. GopherTrunk models the
subset the trunking layer cares about: `Voice4V`, `Voice2V`, and the MAC types `MAC_PTT`, `MAC_END`,
`MAC_IDLE`, `MAC_ACTIVE`, `MAC_HANGTIME`, `MAC_SIGNALING`, and `MAC_END_CONT`. Voice and signalling
therefore interleave *within* a slot's stream across the superframe rather than living on separate
channels — an active call's voice sub-frames are punctuated by the MAC sub-frames that carry its grant
updates, hang-time, and end-of-transmission.

The `SuperframeDecoder` anchors the 0..11 sub-frame grid on a [sync-word](/reference/p25-phase-2-sync-word/)
match. The sub-frame index at which the outbound sync recurs (`SyncSubframeIndex = 0`) is the project's
working value: the spec fixes the exact cadence, but the repo has no figure for it, so the decoder
anchors on *every* match and derives the whole slice geometry from that one constant — a single-line
fix if a real capture disagrees.

## The ultraframe (not modeled)

Above the superframe, TIA-102.BBAB defines an **ultraframe** of 12 superframes — roughly 4.32 seconds —
used chiefly to schedule slower housekeeping such as encryption-sync repetition across the traffic
channel. **GopherTrunk does not model the ultraframe.** Nothing in the Phase 2 decode path depends on
ultraframe alignment: the encryption sync a follower needs arrives in the `MAC_PTT` message that opens a
transmission (see the [MAC PDU](/reference/p25-mac-pdu/)), and the PN44 descrambler is keyed to the
360 ms superframe, not the ultraframe. The absence is a deliberate scope choice, noted here so a reader
does not mistake it for an oversight.

## Relevance to SDR

`internal/radio/p25/phase2/superframe.go` holds the superframe constants and the `SlotType` enum, and
the superframe decoder slices a structured superframe out of the raw dibit stream so the trunking engine
can follow a call across both slots. The 4320-bit period tying the superframe to the PN44 sequence is
what keeps framing and descrambling in lock-step. The spec is TIA-102.BBAB §7.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 and its TDMA traffic channel.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on dividing a carrier into recurring timeslots.
