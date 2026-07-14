---
slug: deframing
title: Deframing
entry_type: concept
category: sdr-app-building
description: Deframing is the receiver step that finds frame boundaries in a symbol stream — locating sync words, aligning to slots, and extracting the bits that belong to one frame.
keywords: deframing, deframer, frame extraction, sync word search, slot alignment, frame boundary, bit alignment, packet extraction, receiver framing
aka: [deframer, frame extraction, frame recovery]
autolink: true
infobox:
  - { label: Type, value: RX framing stage }
  - { label: Finds, value: Frame boundaries in a bit stream }
  - { label: Next, value: FEC decode / parse }
see_also: [correlate-access-code, frame-synchronization, packet-framing, clock-recovery, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Frame_synchronization
  - https://en.wikipedia.org/wiki/Frame_(networking)
---

**Deframing** is the receiver step that finds where each frame begins and ends in a continuous
symbol or bit stream — locating [sync words](/reference/frame-synchronization/), aligning to
time slots, and slicing out the bits that belong to one frame.[^fsync] It is the inverse of
the transmitter's framing: the demodulator and [clock recovery](/reference/clock-recovery/)
produce an unbroken stream of symbols with no inherent marks showing where a message starts,
and deframing restores that structure so the payload can be decoded.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A continuous bit stream is scanned for a sync word; once found, the deframer cuts out the header and payload of one frame and passes it on." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dfrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="8" y="26" font-size="7" fill="currentColor">stream</text>
  <g font-family="monospace" font-size="8" fill="currentColor">
    <text x="8" y="42">…011 [SYNC] hdr payload crc [SYNC] hdr…</text>
  </g>
  <rect x="48" y="32" width="42" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <line x1="69" y1="46" x2="69" y2="66" stroke="currentColor" stroke-width="1" marker-end="url(#dfrar)"/>
  <text x="69" y="80" font-size="7" fill="currentColor" text-anchor="middle">correlate for sync</text>
  <rect x="120" y="90" width="220" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <line x1="122" y1="98" x2="150" y2="98" stroke="currentColor" stroke-width="0.7"/>
  <text x="180" y="101" font-size="7" fill="currentColor">one aligned frame → decode</text>
</svg>
<figcaption>Deframing correlates for the sync word, then cuts one aligned frame out of the running stream.</figcaption>
</figure>

## How it works

The stream that reaches the deframer has bits but no punctuation. Deframing puts the
punctuation back:

- **Find the sync word.** Every frame carries a known fixed pattern — an access code or frame
  sync word. A [sliding correlator](/reference/correlate-access-code/) runs the stream past
  that pattern and flags each position where they match above a threshold. That match is the
  frame's anchor point.
- **Establish bit alignment.** The sync-word position also resolves byte/dibit phase: from the
  anchor, the receiver knows which bit is bit 0 of the header, so subsequent fields land in the
  right place.
- **Slot the bits.** In TDMA systems, frames arrive interleaved across time slots; the
  deframer uses the sync position and the known frame geometry to assign each burst to its
  slot before assembling a logical channel.
- **Extract the frame body.** Counting the standard-defined field lengths from the anchor, the
  deframer copies out header, payload, and check bits and hands them to
  [FEC](/reference/forward-error-correction/) decoding and parsing.

## In practice

Robust deframing tolerates errors in the sync word itself: at low SNR the pattern arrives with
bit flips, so correlation uses a threshold that accepts a few mismatches rather than requiring
an exact hit, and often maintains a *flywheel* — once locked, it predicts where the next sync
should be and trusts that timing even if one instance is corrupted. It must also reject false
matches, which is why sync words are chosen for low autocorrelation sidelobes (like
[Barker codes](/reference/barker-code/)).

## Relevance to SDR

Deframing sits between symbol recovery and payload decode in every framed protocol — P25's
Frame Sync, DMR's SYNC patterns, TETRA's training sequences, [HDLC](/reference/hdlc/) flags in
AX.25. Getting it right is the difference between a locked, decoding receiver and a stream of
noise.

**GopherTrunk** deframes across all its protocols. `internal/dsp/sync` provides a generic
`Correlator` that slides a soft-symbol pattern to flag sync positions, and each radio under
`internal/radio` (P25, DMR, NXDN, TETRA, D-STAR, YSF) has its own sync and framing code that
takes those anchors, resolves slot and bit alignment, and extracts frame bodies for decoding.
The [packet framing](/reference/packet-framing/) layout the transmitter used is what tells the
deframer where each field lives.

## Sources

[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on locating frame boundaries via a known sync sequence.
[^frame]: [Frame (networking)](https://en.wikipedia.org/wiki/Frame_(networking)) — Wikipedia, on the delimited unit deframing extracts from the stream.
