---
slug: amplitude-phase-shift-keying
title: Amplitude and phase-shift keying (APSK)
entry_type: technology
category: modulation
description: APSK places constellation points on concentric rings — amplitude sets the ring, phase sets the position — giving a near-constant envelope that suits satellite power amplifiers; DVB-S2/S2X use 16APSK and 32APSK.
keywords: APSK, amplitude and phase-shift keying, 16APSK, 32APSK, concentric rings, DVB-S2, DVB-S2X, travelling-wave tube, constellation, satellite modulation
aka: [APSK, Amplitude and phase-shift keying, 16APSK, 32APSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (amplitude + phase) }
  - { label: Constellation, value: Concentric rings }
  - { label: Common orders, value: "16APSK (4+12), 32APSK (4+12+16)" }
  - { label: Used by, value: DVB-S2 / DVB-S2X }
see_also: [phase-shift-keying, 8psk, qpsk, quadrature-amplitude-modulation, constellation-diagram, dvb-s, spectral-efficiency]
cite_urls:
  - https://en.wikipedia.org/wiki/Amplitude_and_phase-shift_keying
  - https://en.wikipedia.org/wiki/DVB-S2
---

**Amplitude and phase-shift keying** (**APSK**) arranges its
[constellation](/reference/constellation-diagram/) points on several **concentric rings**:
the signal amplitude selects which ring a symbol lands on, and its phase selects the
position around that ring.[^wiki] It is a deliberate compromise between pure
[phase-shift keying](/reference/phase-shift-keying/) — a single ring, as in
[QPSK](/reference/qpsk/) or [8PSK](/reference/8psk/) — and
[quadrature amplitude modulation](/reference/quadrature-amplitude-modulation/), whose
rectangular grid spends more amplitude levels.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 240" role="img" aria-label="A 16APSK constellation on the IQ plane: an inner ring of four points and an outer ring of twelve points, both centred on the origin." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="270" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="210" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="262" y="134" font-size="10" fill="currentColor">I</text><text x="136" y="30" font-size="10" fill="currentColor">Q</text>
  <circle cx="150" cy="120" r="38" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
  <circle cx="150" cy="120" r="88" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
  <g fill="currentColor">
    <circle cx="177" cy="93" r="3.4"/><circle cx="123" cy="93" r="3.4"/><circle cx="123" cy="147" r="3.4"/><circle cx="177" cy="147" r="3.4"/>
    <circle cx="238" cy="120" r="3.4"/><circle cx="226" cy="76" r="3.4"/><circle cx="194" cy="44" r="3.4"/><circle cx="150" cy="32" r="3.4"/><circle cx="106" cy="44" r="3.4"/><circle cx="74" cy="76" r="3.4"/><circle cx="62" cy="120" r="3.4"/><circle cx="74" cy="164" r="3.4"/><circle cx="106" cy="196" r="3.4"/><circle cx="150" cy="208" r="3.4"/><circle cx="194" cy="196" r="3.4"/><circle cx="226" cy="164" r="3.4"/>
  </g>
  <text x="150" y="230" text-anchor="middle" font-size="8" fill="currentColor">16APSK: inner ring of 4, outer ring of 12</text>
</svg>
<figcaption>16APSK places 4 points on an inner ring and 12 on an outer ring; amplitude picks the ring and phase the position, keeping only two amplitude levels.</figcaption>
</figure>

## How it works

Each APSK order is described by how its points split across rings. **16APSK** uses a **4+12**
layout — four points on the inner ring, twelve on the outer — carrying four bits per symbol.
**32APSK** adds a third ring for a **4+12+16** layout, carrying five bits per symbol. Because
almost every symbol sits at one of just two or three radii, the transmitted envelope stays
close to constant. That matters for the **travelling-wave-tube amplifiers (TWTAs)** aboard
satellites: those amplifiers are run near saturation for efficiency, where they are strongly
nonlinear and compress amplitude. A many-level QAM grid, with its wide spread of amplitudes,
suffers badly under that compression; APSK's few tight rings pass through with far less
distortion.

The cost is noise margin. Splitting power across rings and phases packs the points more
closely than a robust QPSK constellation, so APSK needs a higher
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) than QPSK or 8PSK to hit the same
error rate. It buys extra [spectral efficiency](/reference/spectral-efficiency/) only when the
link budget can spare the SNR.

## In practice

APSK is the workhorse of modern satellite transmission. **[DVB-S2](/reference/dvb-s/)** and its
extension DVB-S2X adopt 16APSK and 32APSK (and, in S2X, higher orders up to 256APSK) for their
higher-throughput modes, selected by adaptive coding and modulation when the measured link can
support them; the more robust QPSK and 8PSK modes are used when it cannot. The ring radii and
the pairing with a specific forward-error-correction code rate are jointly optimised so the
constellation performs well through the satellite's nonlinear amplifier.

## Relevance to SDR

On a constellation display APSK is unmistakable: instead of PSK's single ring or QAM's square
grid, it shows two or three clean rings of points. A software receiver locking a DVB-S2 carrier
must know the exact APSK order and ring-ratio to demap symbols correctly. APSK falls outside
GopherTrunk's land-mobile trunking focus — those modes use C4FM and π/4-DQPSK — so it is
documented here to complete the modulation family and to mark where satellite links move beyond
phase-only keying.

## Sources

[^wiki]: [Amplitude and phase-shift keying](https://en.wikipedia.org/wiki/Amplitude_and_phase-shift_keying) — Wikipedia, for the concentric-ring constellations, the 16APSK/32APSK layouts, and the nonlinear-amplifier motivation.
[^s2]: [DVB-S2](https://en.wikipedia.org/wiki/DVB-S2) — Wikipedia, for the adoption of 16APSK and 32APSK in the DVB-S2/S2X higher-throughput modes.
