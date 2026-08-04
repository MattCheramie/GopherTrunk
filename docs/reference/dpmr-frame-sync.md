---
slug: dpmr-frame-sync
title: dPMR Frame Sync
entry_type: term
category: synchronization
description: "dPMR uses three 48-bit frame sync words to mark burst boundaries: FS1 (0x57FF5F75F575) opens a superframe, FS2 (0x5F7F77FD7DFD) marks mid-superframe sync, and FS3 (0x7DDFFD5F55D5) opens a control/CSBK burst."
keywords: dPMR frame sync, FS1 FS2 FS3, 0x57FF5F75F575, 0x5F7F77FD7DFD, 0x7DDFFD5F55D5, dPMR sync word, ETSI TS 102 658, 48-bit sync
aka: ["dPMR sync word", "FS1", "FS2", "FS3"]
autolink: true
infobox:
  - { label: Count, value: three 48-bit words }
  - { label: FS1, value: "0x57FF5F75F575 (superframe start)" }
  - { label: FS3, value: "0x7DDFFD5F55D5 (control/CSBK)" }
  - { label: Spec, value: ETSI TS 102 658 §4.4 }
see_also: [dpmr, dpmr-csbk, frame-synchronization, correlate-access-code, four-fsk, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/DPMR
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

A **dPMR frame sync** is one of three fixed 48-bit patterns that mark burst boundaries on a
[dPMR](/reference/dpmr/) channel, letting a decoder both locate a burst and learn what kind of
burst it is.[^wiki] Unlike a protocol with a single universal sync word, dPMR carries **three**
distinct words — FS1, FS2, and FS3 — and which one a correlator matches is itself information:
FS1 opens a superframe, FS2 marks a mid-superframe resync point, and FS3 opens a control-channel
signalling burst.[^fsync] Each is 48 bits, 24 dibits at dPMR's 4FSK air interface.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="A dPMR superframe drawn as a row of bursts: FS1 opens the superframe, several traffic bursts follow, FS2 provides a mid-superframe resync, and on the control channel FS3 opens a signalling burst; a sliding correlator matches whichever of the three patterns is present and reports both the position and the burst type." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor">
    <rect x="14" y="40" width="46" height="28" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
    <text x="37" y="58" text-anchor="middle">FS1</text>
    <rect x="60" y="40" width="60" height="28" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="90" y="58" text-anchor="middle">burst</text>
    <rect x="120" y="40" width="60" height="28" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="150" y="58" text-anchor="middle">burst</text>
    <rect x="180" y="40" width="46" height="28" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
    <text x="203" y="58" text-anchor="middle">FS2</text>
    <rect x="226" y="40" width="60" height="28" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="256" y="58" text-anchor="middle">burst</text>
    <text x="14" y="88" font-size="7.5">superframe: FS1 start · FS2 mid-superframe resync (every 4th burst)</text>
    <rect x="320" y="40" width="46" height="28" fill="currentColor" fill-opacity="0.32" stroke="currentColor" stroke-width="1.3"/>
    <text x="343" y="58" text-anchor="middle">FS3</text>
    <rect x="366" y="40" width="90" height="28" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="411" y="58" text-anchor="middle">CSBK</text>
    <text x="320" y="88" font-size="7.5">control channel: FS3 opens a signalling burst</text>
  </g>
  <text x="14" y="112" font-size="7.5" fill="currentColor">correlator matches any of FS1/FS2/FS3 → burst position AND burst type</text>
</svg>
<figcaption>dPMR distributes three sync words across its structure: FS1 opens a superframe, FS2 provides a mid-superframe resync so a receiver that missed FS1 can still lock, and FS3 marks a control-channel CSBK burst. Matching one of the three yields both where the burst starts and what it contains.</figcaption>
</figure>

## The three words

GopherTrunk stores the three patterns as canonical 48-bit hex constants and materialises them
MSB-first into 24-dibit arrays for the symbol-domain correlator:

| Word | Hex (48 bits) | Role |
| --- | --- | --- |
| FS1 | `0x57FF5F75F575` | start of a voice/data superframe |
| FS2 | `0x5F7F77FD7DFD` | mid-superframe resync (every 4th burst) |
| FS3 | `0x7DDFFD5F55D5` | start of a control-channel CSBK burst |

Having a separate mid-superframe word (FS2) is what makes dPMR robust to a late lock: a
receiver that joins a call after FS1 has already passed does not have to wait for the next
superframe — it can re-acquire on the next FS2 and slot straight into the burst grid. FS3 is
the one the trunking engine watches most closely, because a control channel is a stream of FS3
bursts, each opening a [CSBK](/reference/dpmr-csbk/) signalling block.

## Detecting a sync

Detection is an [access-code correlation](/reference/correlate-access-code/): the decoder
slides a 24-dibit window over the demodulated stream and, at each position, counts how many
dibits differ from a target pattern; a count at or under the configured tolerance declares a
match at that position. GopherTrunk's `SyncDetector` takes one pattern and a tolerance — a
tolerance of zero demands an exact match — and reports the absolute dibit index where each hit
ends. A control-channel monitor runs an FS3 detector; a call-follower runs FS1 and FS2 to hold
the traffic superframe. The detector's shape deliberately mirrors the Phase 1, DMR, and NXDN
sync detectors so the higher-level state machine treats every protocol the same way.

## Relevance to SDR

`internal/radio/dpmr/sync.go` holds the `FS1Hex` / `FS2Hex` / `FS3Hex` constants, the
`FS1Dibits` / `FS2Dibits` / `FS3Dibits` helpers that expand them, and the tolerant
`SyncDetector`. The intended control-channel flow is FS3 sync detect → 80-bit CSBK slice →
`CSBKFromBits` → ingest, so the sync stage is the first link in the dPMR trunking chain: no FS3
lock means no CSBK, and no CSBK means no grants to follow. Because the three words are distinct
and each is tied to a burst type, dPMR's sync layer does double duty — it recovers timing and
classifies the burst in a single correlation, which is why the decoder can tell a control burst
from a traffic burst before it has decoded a single payload bit.

## Sources

[^wiki]: [dPMR](https://en.wikipedia.org/wiki/DPMR) — Wikipedia, on the ETSI dPMR narrowband digital standard and its burst structure.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on using fixed sync sequences to locate frame boundaries.
