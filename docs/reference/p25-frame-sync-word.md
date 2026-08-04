---
slug: p25-frame-sync-word
title: P25 frame sync word
entry_type: term
category: synchronization
description: The P25 frame sync word (FSW) is the fixed 48-bit pattern 0x5575F5FF77FF that opens every P25 Phase 1 frame, letting a decoder find where a frame begins in the dibit stream.
keywords: P25 frame sync word, FSW, 0x5575F5FF77FF, P25 sync pattern, frame synchronization P25, C4FM dibit mapping, sync correlation, P25 Phase 1 sync
aka: [FSW, "P25 sync word", "frame sync word"]
autolink: true
infobox:
  - { label: Length, value: 48 bits (24 dibits) }
  - { label: Value, value: "0x5575F5FF77FF" }
  - { label: Spec, value: TIA-102.BAAA §6.1.1 }
  - { label: Precedes, value: NID → frame body }
see_also: [correlate-access-code, frame-synchronization, c4fm, p25-nid-duid, p25-phase-1, four-fsk]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

The **P25 frame sync word** (**FSW**) is the fixed 48-bit pattern **`0x5575F5FF77FF`** that
opens every [P25](/reference/project-25/) Phase 1 frame.[^wiki] Before a decoder can read a
[NID](/reference/p25-nid-duid/), a voice frame, or a
[TSBK](/reference/tsbk/), it must know *where* the frame starts in the incoming symbol
stream — and the FSW is the landmark it searches for, exactly the job of
[frame synchronization](/reference/frame-synchronization/).[^fsync]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="The 48-bit P25 frame sync word 0x5575F5FF77FF shown as 24 C4FM dibits, followed by the 64-bit NID and then the frame body; a sliding correlator locks onto the sync word to mark the frame boundary." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="150" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="95" y="47" text-anchor="middle" font-size="9" fill="currentColor">FSW · 48 bits</text>
  <rect x="170" y="30" width="120" height="26" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="47" text-anchor="middle" font-size="9" fill="currentColor">NID · 64 bits</text>
  <rect x="290" y="30" width="160" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="370" y="47" text-anchor="middle" font-size="9" fill="currentColor">frame body</text>
  <text x="20" y="74" font-size="8" font-family="monospace" fill="currentColor">0x5575F5FF77FF  =  +3 +3 +3 +3 +3 −3 …  (24 dibits)</text>
  <path d="M40 96 L120 96 L120 84" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <path d="M95 112 L95 100" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="95" y="124" text-anchor="middle" font-size="8" fill="currentColor">correlator peak = frame start</text>
</svg>
<figcaption>Every P25 Phase 1 frame opens with the same 24-dibit sync word; a sliding correlator locks onto it to mark the boundary, then the decoder reads the NID and body that follow.</figcaption>
</figure>

## How it works

A P25 Phase 1 receiver produces a stream of **dibits** (two bits at a time) from the
[C4FM](/reference/c4fm/) demodulator — the four transmit deviations map to dibit values by
the TIA-102 rule `+3→01, +1→00, −1→10, −3→11`. The FSW is 24 of those dibits in a fixed
order, so the decoder runs a sliding [access-code correlation](/reference/correlate-access-code/):
at each new dibit it compares the last 24 dibits against the known pattern and declares a
frame boundary wherever the number of mismatches falls under a threshold (GopherTrunk
defaults to tolerating 4 of 24). The sync word is chosen for a sharp autocorrelation peak,
so a genuine alignment stands out clearly from noise.

Because the FSW pattern is designed to be recognisable even under error, the correlator can
lock a frame a few dB into the noise where a strict all-or-nothing match would fail — the
same tolerance that lets a scanner hold a marginal P25 signal.

## In practice

Two subtleties make real-world FSW search harder than the textbook version:

- **Polarity / rotation ambiguity.** An FM discriminator can hand the decoder the dibit
  alphabet cyclically rotated — most commonly a whole-alphabet polarity flip (`dibit + 2`)
  from a conjugated I/Q input. GopherTrunk's detector tries cyclic rotations and records
  which one matched. On a true C4FM stream only rotation 0 (identity) and rotation 2
  (polarity flip) are physically possible; the CQPSK / π/4-DQPSK path has a genuine
  four-fold phase ambiguity and needs all four. Allowing the non-physical rotations 1 and 3
  on C4FM actually *hurt*: the downstream [BCH](/reference/bch-code/) decoder would sometimes
  miscorrect a misaligned window into a parity-valid pseudo-[NID](/reference/p25-nid-duid/) at
  the wrong rotation (the root of GopherTrunk issue #275).
- **Status symbols.** P25 inserts an interstitial
  [status symbol](/reference/p25-status-symbols/) into the stream at a fixed cadence, so the
  raw symbol positions do not map one-to-one onto frame bits — the deframer must account for
  them when it reads the NID and body after the sync.

## Relevance to SDR

The FSW is the front door of GopherTrunk's P25 Phase 1 decoder. `internal/radio/p25/phase1`
holds the canonical 24-dibit `FrameSyncWord`, the `SymbolToDibit` mapping, and a
`SyncDetector` that slides the pattern over the demodulated dibits and reports each hit's
position, matched rotation, and correlation margin — the last of which the measurement
harness turns into a sync-health metric (a margin pressed against the threshold means lock
is barely holding). Every P25 Phase 1 frame type — HDU, LDU1/LDU2, TDU, TSDU, PDU — begins
at an FSW hit, so getting this search right, rotations and all, is what lets everything
downstream decode at all.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 frame structure.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on using a fixed sync sequence to locate frame boundaries in a bit stream.
