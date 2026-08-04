---
slug: tetra-traffic-slot-mapping
title: TETRA traffic slot mapping
entry_type: term
category: trunked-radio
description: "TETRA traffic slot mapping is how a voice follower places each burst on a carrier onto the TDMA slot grid: the synchronisation burst anchors the grid one NDB-slot before its frame's TN1 traffic, and the AACH usage marker — not the slot number — is the reliable demux key for concurrent same-carrier calls."
keywords: TETRA traffic slot mapping, TDMA slot grid, synchronisation burst anchor, ndbSBSlotShift, downlink usage marker, concurrent calls, traffic extractor, soft LLR stash, EN 300 392-2
aka: [traffic slot mapping, "TDMA slot grid anchor", "slot demux"]
autolink: true
infobox:
  - { label: Slot span, value: 255 dibits (510 bits) }
  - { label: Grid anchor, value: "Synchronisation burst (SB) in TN1" }
  - { label: Demux key, value: "AACH downlink usage marker" }
  - { label: Spec, value: "ETSI EN 300 392-2 §9 / §21.4.7" }
see_also: [tetra, tetra-aach, tetra-burst-formats, tetra-tchs-speech-coding, tdma, soft-decision, channel-grant, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

**TETRA traffic slot mapping** is how a voice follower, having retuned to a granted traffic
carrier, decides which of the carrier's four [TDMA](/reference/tdma/) timeslots each burst
belongs to and which call it carries.[^tetra][^tdma] A TETRA carrier interleaves four calls'
slots in time, so extracting a single call means demultiplexing the burst stream — and doing
that correctly turns out to depend on *not* trusting the obvious slot number, but on the
per-slot [AACH](/reference/tetra-aach/) usage marker instead.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A TDMA frame of four 255-dibit slots; the synchronisation burst's training sequence sits one NDB-slot before the TN1 traffic burst, so a shift of plus-three aligns the detected anchor to slot 1, and each further slot is 255 dibits on; the AACH usage marker on each burst identifies which call occupies the slot." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="100" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/><text x="70" y="57">TN1 · 255d</text>
    <rect x="120" y="40" width="100" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="170" y="57">TN2</text>
    <rect x="220" y="40" width="100" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="270" y="57">TN3</text>
    <rect x="320" y="40" width="100" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="370" y="57">TN4</text>
  </g>
  <path d="M2 30 L44 30" stroke="currentColor" stroke-width="1.4"/>
  <text x="2" y="24" font-size="7.5" fill="currentColor">SB training seq (one slot early)</text>
  <path d="M23 30 L23 40" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <path d="M23 82 L70 82 L70 68" stroke="currentColor" stroke-width="1" fill="none" marker-end="url(#tsm)"/>
  <defs><marker id="tsm" markerWidth="8" markerHeight="8" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="90" y="92" font-size="7.5" fill="currentColor">+3 (= −1 mod 4) shift → anchor reads as TN1</text>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="70" y="112">AACH → marker 4</text><text x="170" y="112">marker 2 (ctrl)</text><text x="270" y="112">AACH → marker 5</text><text x="370" y="112">idle</text>
  </g>
  <text x="235" y="136" text-anchor="middle" font-size="8" fill="currentColor">route each burst by its usage marker, not its slot number</text>
</svg>
<figcaption>The synchronisation burst's training sequence lands one NDB-slot before the TN1 traffic, so a +3 shift anchors the 255-dibit grid; each burst's AACH usage marker, not its slot index, is what reliably identifies the call it carries.</figcaption>
</figure>

## Anchoring the grid

One TDMA timeslot is **255 dibits** (510 bits); four make a 1020-dibit frame. To place a
burst, the extractor needs an absolute reference for where slot 1 begins. That reference is
the **synchronisation burst (SB)**, transmitted in slot 1 (TN1) of frame 18: its
synchronisation training sequence pins the grid, and a normal burst leading at absolute dibit
`L` then falls in slot `round((L − anchor) / 255) mod 4`.

The subtlety is that the SB's training sequence sits *late* in the SB burst, after the
frequency-correction and BSCH preamble, so the detected training-sequence leading dibit lands
**one NDB-slot before** the TN1 traffic burst's normal-training-sequence position. Adding 3
(≡ −1 mod 4) — the `ndbSBSlotShift` constant — makes a burst one slot after the anchor read
as TN1, matching the control channel's granted timeslots. This was verified against a real
same-carrier capture: with the shift applied, grant timeslot 1 and timeslot 2 line up with
the decoded slots. The anchor is refreshed on every SB (once per multiframe) so it tracks slow
clock drift; until an SB is seen the slot is reported as 0 (unknown), and on a traffic-only
carrier with no SB it stays 0.

## Why the marker, not the slot

The TDMA slot number is kept for telemetry but is **not** a reliable demux key on real air.
The SB anchor's intra-slot rounding jitters a call's bursts across adjacent slot numbers, and
the channel-allocation grant's timeslot field does not map cleanly to the physical slot. The
reliable key is the [AACH downlink usage marker](/reference/tetra-aach/): the AACH decodes in
*every* downlink slot, a marker of 4 or greater identifies the call occupying that slot, and a
granted call's marker matches the marker carried in its [grant](/reference/channel-grant/). So
the voice chain routes each burst by marker, isolating concurrent same-carrier calls that a
slot-number scheme would smear together. When the hard AACH decode misses under load, a
gated soft-decision fallback recovers the marker from the per-symbol confidences rather than
dropping the burst.

## Soft-LLR stashing

To let the traffic path decode [soft-decision](/reference/soft-decision/) TCH/S without
changing the hard dibit contract, the extractor carries the receiver's per-symbol complex
differentials in a buffer kept strictly parallel to its dibit buffer. `StashSoft` hands it the
differentials for the next block, keyed by the same base index; if they ever fall out of
lockstep the soft path is dropped rather than misaligned, and the burst decodes hard-only.
When present, the parallel buffer is sliced by exactly the same BKN1 + BKN2 geometry as the
hard bits to build the descrambled 432-LLR type-5 stream for the burst.

## Relevance to SDR

`internal/radio/tetra/traffic.go` implements the `TrafficExtractor`: it scans the π/4-DQPSK
dibit stream for Normal Continuous Downlink Bursts, anchors the grid with a synchronisation
training-sequence detector, and emits each burst's raw or descrambled traffic frame tagged
with both its TDMA slot (`slotOf`) and its AACH usage marker (`usageOf`), plus the parallel
soft stream (`softFrame`). Getting the `ndbSBSlotShift` and the marker-versus-slot decision
right is what lets GopherTrunk record concurrent TETRA calls on one carrier as separate,
correctly-attributed audio.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA slot and burst structure.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on the four-slot TDMA scheme TETRA uses.
