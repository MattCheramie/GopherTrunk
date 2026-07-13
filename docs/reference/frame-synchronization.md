---
slug: frame-synchronization
title: Frame synchronization
entry_type: term
category: sdr-dsp
description: "Frame synchronization locates frame boundaries in a symbol stream by detecting a known sync word, so a decoder can align to fields, headers, and payload."
keywords: frame synchronization, frame sync, sync word, sync pattern, framing, unique word, frame alignment, marker sequence, sync detection, packet boundary
aka: [frame sync, framing, sync-word detection]
autolink: true
infobox:
  - { label: Type, value: Receiver alignment stage }
  - { label: Uses, value: Known sync word / marker }
  - { label: Recovers, value: Frame boundary & bit phase }
see_also: [preamble-correlation, clock-recovery, matched-filter, barker-code, differential-decoding, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

**Frame synchronization** is the step that locates the boundaries of each frame in a demodulated
symbol stream by detecting a known **sync word** (a fixed marker sequence the transmitter
inserts), so the decoder knows where fields, headers, and payload begin.[^wiki] Bit- and
symbol-timing recovery tell a receiver *when* each symbol occurs; frame sync tells it *which*
symbol starts the frame. Without it a perfectly demodulated bit stream is an unaligned blur —
the right bits with no map — because the receiver has no idea which bit is bit zero.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A continuous symbol stream in which a fixed sync word is detected, marking the boundary after which header and payload fields are correctly aligned." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="45" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  <text x="55" y="62" font-size="8" fill="currentColor" text-anchor="middle">…noise…</text>
  <rect x="90" y="45" width="86" height="26" fill="currentColor" opacity="0.85"/>
  <text x="133" y="62" font-size="8.5" fill="currentColor" text-anchor="middle">SYNC WORD</text>
  <rect x="176" y="45" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="211" y="62" font-size="8.5" fill="currentColor" text-anchor="middle">header</text>
  <rect x="246" y="45" width="140" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="316" y="62" font-size="8.5" fill="currentColor" text-anchor="middle">payload</text>
  <rect x="386" y="45" width="54" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="413" y="62" font-size="8" fill="currentColor" text-anchor="middle">CRC</text>
  <line x1="176" y1="82" x2="176" y2="100" stroke="currentColor" stroke-width="1"/>
  <text x="176" y="114" font-size="8.5" fill="currentColor" text-anchor="middle">frame boundary</text>
</svg>
<figcaption>Detecting the fixed sync word marks the frame boundary; every field after it is then read at a known offset.</figcaption>
</figure>

## How it works

The transmitter prefixes each frame (or periodically inserts) a fixed bit or symbol pattern — the
**sync word**, also called a unique word or frame-alignment word — chosen to be unlikely to occur
in random data. The receiver slides a [correlator](/reference/preamble-correlation/) along the
incoming stream, comparing each candidate window against the known pattern. When the correlation
peaks above a threshold, the frame boundary is declared at that position. A good sync word has a
sharp autocorrelation peak and low sidelobes so the peak is unambiguous even under noise and a
few bit errors; short optimal sequences such as [Barker codes](/reference/barker-code/) are used
for exactly this property.

Two design knobs govern the detector:

- **Threshold / Hamming distance** — matching is rarely exact under noise, so the detector allows
  up to a set number of mismatched bits. Too tight and it misses real frames (missed detection);
  too loose and random data triggers it (false alarm). The sync word's length and distance
  properties set how far these can be pushed apart.
- **Acquisition vs tracking** — on first lock the receiver searches every bit position
  ("hunt" state); once synced it need only check the expected position each frame ("lock"
  state), which is cheaper and rejects spurious matches elsewhere in the stream. Many systems
  require several consecutive confirmations before declaring lock and several misses before
  dropping it, giving a flywheel that rides through brief dropouts.

A related subtlety is **phase ambiguity**: [PSK](/reference/phase-shift-keying/) demodulators can
lock with an arbitrary constellation rotation, so the recovered bits may be inverted or rotated.
The sync word resolves this too — the pattern only correlates in the correct rotation, and
[differential decoding](/reference/differential-decoding/) or testing all rotations recovers the
right one.

## Relevance to SDR

Essentially every digital radio protocol carries a frame sync. P25 uses a 48-bit frame sync at
the head of each frame; DMR embeds 24-bit sync patterns that also distinguish voice from data and
base from mobile; NXDN, TETRA, POCSAG (its unique word), and packet formats like AX.25 (flag
bytes) all rely on marker detection to align. A decoder's first job after demodulation and
[clock recovery](/reference/clock-recovery/) is to find these markers. GopherTrunk's protocol
decoders depend directly on frame synchronization: locating the P25/DMR/NXDN sync pattern is what
lets the control-channel and voice-frame parsers align to the bit fields they read, and losing
sync is the event that sends the decoder back into its hunt state.

## Sources

[^wiki]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on sync words and frame-boundary detection in digital communication.
