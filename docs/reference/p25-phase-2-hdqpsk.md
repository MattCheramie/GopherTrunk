---
slug: p25-phase-2-hdqpsk
title: P25 Phase 2 H-DQPSK
entry_type: term
category: modulation
description: H-DQPSK is the two-slot TDMA waveform of P25 Phase 2 — a π/8-shifted differential QPSK running at 6000 symbols/second (12000 bit/s) per carrier, replacing the C4FM/FDMA physical layer of Phase 1.
keywords: P25 Phase 2 H-DQPSK, harmonized DQPSK, HCPM, 6000 sym/s, 12000 bps, two-slot TDMA, pi/8 DQPSK, TIA-102.BBAC, P25 Phase 2 modulation
aka: [H-DQPSK, "Harmonized DQPSK", "P25 Phase 2 waveform", HCPM]
autolink: true
infobox:
  - { label: Symbol rate, value: 6000 sym/s (12000 bit/s) }
  - { label: Constellation, value: π/8-shifted DQPSK, α = 0.20 }
  - { label: Access, value: 2-slot TDMA }
  - { label: Spec, value: TIA-102.BBAC }
see_also: [p25-phase-2, pi-4-dqpsk, tdma, fdma, p25-phase-1, differential-decoding, p25-phase-2-sync-word, p25-isch]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Phase-shift_keying
---

**H-DQPSK** (**Harmonized Differential QPSK**) is the physical-layer waveform of
[P25 Phase 2](/reference/p25-phase-2/).[^wiki] Where [Phase 1](/reference/p25-phase-1/) puts one
9600 bit/s C4FM channel on a 12.5 kHz carrier ([FDMA](/reference/fdma/)), Phase 2 packs *two*
independent voice paths onto the same 12.5 kHz carrier by running a faster **6000 symbol/second**
(**12000 bit/s**) [π/8-shifted DQPSK](/reference/pi-4-dqpsk/) waveform and time-dividing it into two
slots — [TDMA](/reference/tdma/).[^psk] Each symbol carries a dibit (2 bits), and the information
rides in the *phase change* from one symbol to the next, so the receiver is a
[differential decoder](/reference/differential-decoding/) rather than a coherent one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A single 12.5 kHz carrier is time-divided into two TDMA slots that alternate every 30 milliseconds; the H-DQPSK constellation is a QPSK ring offset by pi over eight, decoded by differential phase between successive symbols." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="24" width="180" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="110" y="41" text-anchor="middle" font-size="9" fill="currentColor">slot 1 · 30 ms</text>
  <rect x="200" y="24" width="180" height="26" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="290" y="41" text-anchor="middle" font-size="9" fill="currentColor">slot 2 · 30 ms</text>
  <text x="20" y="66" font-size="8" fill="currentColor">one 12.5 kHz carrier · 6000 sym/s · 12000 bit/s shared between the two slots</text>
  <circle cx="360" cy="128" r="34" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <g fill="currentColor"><circle cx="392" cy="115" r="2.4"/><circle cx="392" cy="141" r="2.4"/><circle cx="328" cy="115" r="2.4"/><circle cx="328" cy="141" r="2.4"/></g>
  <line x1="360" y1="128" x2="392" y2="115" stroke="currentColor" stroke-width="1"/>
  <text x="360" y="176" text-anchor="middle" font-size="8" fill="currentColor">π/8-offset DQPSK · dibit = phase step</text>
  <text x="130" y="128" text-anchor="middle" font-size="8" fill="currentColor">two voice paths</text>
  <text x="130" y="140" text-anchor="middle" font-size="8" fill="currentColor">per carrier</text>
</svg>
<figcaption>Phase 2 doubles a carrier's capacity by TDMA — two 30 ms slots share one 6000 sym/s H-DQPSK waveform — instead of Phase 1's single continuous C4FM channel.</figcaption>
</figure>

## Modulation and rate

The Phase 2 downlink (base → subscriber) is a **π/8-shifted DQPSK** constellation: the four QPSK
points sit on a ring, and successive symbols step by odd multiples of π/4 around a fixed π/8 offset.
That offset keeps the trajectory off the origin, bounding the envelope so a linear power amplifier
stays efficient — the same reasoning behind [π/4-DQPSK](/reference/pi-4-dqpsk/) in TETRA and IS-136,
of which this is the π/8 cousin. The uplink is defined as a compatible **H-CPM** (Harmonized
Continuous Phase Modulation) constant-envelope waveform so a battery subscriber can run its PA in
saturation; "harmonized" refers to reconciling the two vendor proposals that merged into the
standard. A GopherTrunk receiver treats the demodulated downlink as a differential dibit stream and
does not model the CPM uplink except as a diagnostic sync target.

At 6000 sym/s with 2 bits per symbol, one carrier delivers 12000 bit/s of coded channel, which the
TDMA structure splits into two ~6000 bit/s logical slots — enough for one
[AMBE+2](/reference/ambe-plus-2/) vocoder stream plus signalling each. Contrast Phase 1: 4800 sym/s
C4FM ([4-FSK](/reference/four-fsk/) family) at 9600 bit/s, one voice path per carrier. The net effect
is a doubling of channel capacity on the same spectrum, which is the whole point of Phase 2.

## Demodulating it

GopherTrunk demodulates H-DQPSK with a single reusable primitive. `NewPiOver4DQPSK` builds a
root-raised-cosine matched filter (roll-off α = 0.20 for Phase 2) feeding a differential slicer;
the constructor takes a `rotation` argument, and passing `math.Pi/8` selects the Phase 2 offset
(passing `math.Pi/4` gives the TETRA/IS-136 variant from the same code). At each symbol the decoder
computes `arg(s · conj(last)) − rotation`, wraps to [−π, π], and picks the nearest of the four
quadrants to emit a dibit. A soft-decision path (`DecodeBoth`) additionally exports the raw complex
differential `s · conj(last)`, whose real and imaginary parts are the two on-air bits' reliabilities
— the log-likelihood input the MAC trellis's soft Viterbi uses to recover ~1.5–2 dB the hard slicer
throws away.

Because the modulation is *differential*, the receiver never has to resolve an absolute carrier
phase, which is what makes the four-fold QPSK phase ambiguity harmless: a whole-constellation rotation
cancels in the `s · conj(last)` product. It does, however, mean the dibit *labels* the slicer assigns
can differ from the standard's by a quadrant transpose, which is why the Phase 2 front end applies a
fixed dibit remap before anything downstream reads the bits (see the
[Phase 2 sync word](/reference/p25-phase-2-sync-word/)).

## Relevance to SDR

H-DQPSK is the entry point of GopherTrunk's Phase 2 decoder: `internal/dsp/demod/dqpsk.go` and
`piover4_dqpsk.go` turn IQ into the 6000 sym/s dibit stream that the sync detector, ISCH decode,
and MAC FEC all consume. Getting the rotation (π/8, not π/4) and the RRC roll-off right is what lets
a real two-slot signal lock at all; everything from the superframe grid to the vocoder sits on top of
this waveform. The spec is TIA-102.BBAC.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 and its two-slot TDMA physical layer.
[^psk]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, on differential and π/4-shifted QPSK constellations.
