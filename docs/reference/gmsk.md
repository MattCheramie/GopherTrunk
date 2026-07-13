---
slug: gmsk
title: GMSK
entry_type: technology
category: modulation
description: GMSK (Gaussian minimum-shift keying) is a continuous-phase FSK variant with a Gaussian pulse-shaping filter, giving a compact spectrum; used by AIS, GSM, and D-STAR.
keywords: GMSK, Gaussian minimum shift keying, continuous phase, AIS, GSM, D-STAR, pulse shaping, MSK, BT product, constant envelope
aka: [GMSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (CPM) }
  - { label: Feature, value: Gaussian-filtered, compact spectrum }
  - { label: Used by, value: AIS, GSM, D-STAR }
see_also: [frequency-shift-keying, ais, d-star, root-raised-cosine-filter, minimum-shift-keying, continuous-phase-modulation, gfsk, intersymbol-interference, four-fsk]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Minimum-shift_keying
  - https://en.wikipedia.org/wiki/Continuous_phase_modulation
---

**GMSK** (Gaussian minimum-shift keying) is a continuous-phase
[FSK](/reference/frequency-shift-keying/) variant in which the data is passed through a
Gaussian filter before modulation, smoothing phase transitions for a **compact
spectrum** and **constant envelope**.[^wiki] It is the Gaussian-shaped refinement of
[minimum-shift keying](/reference/minimum-shift-keying/) (MSK), and one of the most widely
deployed digital modulations in history thanks to GSM.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A smooth continuous phase trajectory with no abrupt jumps, illustrating Gaussian-filtered MSK." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 60 C 60 30, 90 30, 120 60 S 180 90, 220 75 C 260 62, 270 35, 310 35 S 380 80, 440 55" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="20" y="100" font-size="10" fill="currentColor">phase is continuous and smoothly filtered — compact spectrum</text>
</svg>
<figcaption>GMSK is continuous-phase FSK with Gaussian pulse shaping, giving a narrow spectrum; AIS uses it.</figcaption>
</figure>

## How it works

GMSK starts from MSK, which is the special case of continuous-phase 2FSK where the
frequency deviation is exactly one-quarter of the bit rate — the *minimum* separation
that keeps the two tones orthogonal. At that deviation each bit advances the carrier phase
by exactly ±90°, so MSK can equivalently be viewed as a form of offset
[QPSK](/reference/qpsk/) with sinusoidal pulse shaping. GMSK adds one step: it passes the
data through a **Gaussian low-pass filter** before the phase integrator. The Gaussian
filter has no overshoot and a very fast spectral roll-off, so it suppresses the
side-lobes that plain MSK still radiates, buying an even more compact spectrum.

The filter's aggressiveness is set by the **BT product** — the Gaussian's bandwidth times
the bit period. A small BT (GSM uses 0.3) gives a very narrow spectrum but spreads each
symbol's energy across neighbours, deliberately introducing controlled
[intersymbol interference](/reference/intersymbol-interference/) that the receiver must
untangle (often with a Viterbi/MLSE equaliser). A larger BT keeps the eye open at the cost
of wider occupied bandwidth. Because the phase path is smooth and the amplitude never
varies, GMSK keeps a **constant envelope**, which — like [C4FM](/reference/c4fm/) — lets
it be amplified by efficient non-linear power amplifiers, a decisive advantage for
battery-powered handsets.

## Variants

GMSK sits in the [continuous-phase modulation](/reference/continuous-phase-modulation/)
(CPM) family alongside MSK, [C4FM](/reference/c4fm/), and
[GFSK](/reference/gfsk/) (Gaussian FSK, the near-identical scheme Bluetooth uses — the
line between "GFSK" and "GMSK" is mostly whether the deviation index is set to the MSK
value of 0.5). Compared with [4FSK](/reference/four-fsk/) modes, GMSK is binary — 1 bit
per symbol — trading throughput for an exceptionally clean, robust, constant-envelope
signal.

## Relevance to SDR

GMSK carries [AIS](/reference/ais/) ship transponders (9600 bps GMSK on marine VHF), GSM
cellular, [D-STAR](/reference/d-star/) amateur digital voice, and many satellite and
telemetry links. A software receiver demodulates it either coherently (recovering carrier
phase and detecting the ±90° increments) or non-coherently as an FSK discriminator
followed by a matched filter and slicer. GopherTrunk uses a GMSK demodulator in its
[AIS](/reference/ais/) pipeline, recovering the underlying bit transitions after
frequency/phase detection; it does not target GSM, which is out of scope.

## Sources

[^wiki]: [Minimum-shift keying](https://en.wikipedia.org/wiki/Minimum-shift_keying) — Wikipedia, for MSK/GMSK, Gaussian shaping, the BT product, and the constant-envelope property.
[^cpm]: [Continuous phase modulation](https://en.wikipedia.org/wiki/Continuous_phase_modulation) — Wikipedia, for the CPM family GMSK belongs to and its controlled-ISI behaviour.
