---
slug: pi-4-dqpsk
title: π/4-DQPSK
entry_type: technology
category: modulation
description: "π/4-DQPSK is a differential phase-shift keying that rotates the constellation 45° per symbol — the modulation used by TETRA and other digital radio systems."
keywords: pi/4-DQPSK, differential QPSK, TETRA modulation, phase-shift keying, 8-point constellation, differential encoding, NADC, TETRAPOL
aka: ["pi/4-DQPSK", "π/4 DQPSK", differential QPSK]
autolink: true
see_also: [phase-shift-keying, qpsk, differential-decoding, cqpsk, constellation-diagram, tetra]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-shift_keying#%CF%80/4%E2%80%93QPSK
  - https://en.wikipedia.org/wiki/Differential_coding
---

**π/4-DQPSK** (π/4-shifted differential quadrature [phase-shift keying](/reference/phase-shift-keying/))
is a four-symbol phase modulation in which the constellation is **rotated by 45° (π/4)
every symbol** and information is carried in the *change* of phase rather than its
absolute value.[^wiki] It is the air-interface modulation of [TETRA](/reference/tetra/), and
was also used by the North American IS-136 (D-AMPS/NADC) and Japanese PDC cellular systems.

<figure class="figure" markdown="0">
<svg viewBox="0 0 240 240" role="img" aria-label="An eight-point constellation formed by two QPSK sets offset by 45 degrees, as used by pi/4-DQPSK." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="220" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="120" y1="20" x2="120" y2="220" stroke="currentColor" stroke-opacity="0.4"/>
  <g fill="currentColor"><circle cx="190" cy="120" r="4"/><circle cx="120" cy="50" r="4"/><circle cx="50" cy="120" r="4"/><circle cx="120" cy="190" r="4"/></g>
  <g fill="currentColor" fill-opacity="0.55"><circle cx="170" cy="70" r="4"/><circle cx="70" cy="70" r="4"/><circle cx="70" cy="170" r="4"/><circle cx="170" cy="170" r="4"/></g>
  <text x="212" y="134" font-size="10" fill="currentColor">I</text><text x="106" y="30" font-size="10" fill="currentColor">Q</text>
</svg>
<figcaption>π/4-DQPSK alternates between two QPSK constellations offset by 45°, so symbols never pass through the origin.</figcaption>
</figure>

## How it works

Start from ordinary [QPSK](/reference/qpsk/), whose four symbols sit at 45°, 135°, 225° and
315°. π/4-DQPSK uses **two** such constellations: the base one and a copy rotated by 45° (at
0°, 90°, 180°, 270°). The transmitter alternates between them on every symbol, so the visible
eight points are the union of both sets — but on any given symbol only four are legal.

Each dibit of input selects one of four *phase increments* — ±45° or ±135° — that are added to
the phase of the previous symbol. Because every increment is an odd multiple of 45°, consecutive
symbols always sit in the opposite set, guaranteeing a phase change at every symbol boundary and
forcing the alternation. Two consequences follow directly from this rule:

- **A transition on every symbol.** There is no allowed "stay put" increment, so the phase always
  moves. That gives the receiver's [clock-recovery](/reference/clock-recovery/) loop a reliable
  timing event every symbol, even during long constant-data runs.
- **The trajectory never crosses the origin.** The permitted ±45°/±135° steps route the signal
  around the centre of the IQ plane rather than through it, so the amplitude envelope never
  collapses to zero. A more constant envelope means the transmitter's
  [power amplifier](/reference/power-amplifier/) can run closer to saturation efficiently without
  regrowing the spectrum, which matters for battery-powered handhelds.

Information lives in the *difference* between successive phases, so the receiver recovers dibits
by [differentially decoding](/reference/differential-decoding/) — measuring each symbol's phase
relative to the one before it. A slowly varying absolute carrier-phase offset cancels in that
subtraction, so no absolute phase reference is needed.

## Variants

The closely related **π/4-CQPSK** (coherent) form uses the same 45°-shifted geometry but the
phase changes carry the bits directly (Gray-mapped) rather than differentially; some P25-adjacent
literature and GopherTrunk's own [CQPSK](/reference/cqpsk/) page treat the C4FM/CQPSK linear
variant of P25 in this family. Plain **DQPSK** without the π/4 shift lacks the guaranteed
per-symbol transition and can pass through the origin. π/4-DQPSK is nearly always paired with a
[root-raised-cosine](/reference/root-raised-cosine-filter/) [pulse-shaping](/reference/pulse-shaping/)
filter; TETRA uses a roll-off of 0.35.

## In practice

TETRA carries 18 000 symbols per second at two bits each — 36 kbit/s gross — in a 25 kHz channel,
and its four-slot TDMA structure rides on this π/4-DQPSK carrier. The modulation's differential
robustness and near-constant envelope are exactly the properties a professional mobile system
wants: tolerance of a drifting carrier, efficient amplification, and a contained spectrum. The
same reasoning drove its adoption in the first-generation digital cellular standards.

## Relevance to SDR

A software receiver demodulates π/4-DQPSK by recovering symbol timing, sampling the complex
symbol, computing the phase difference from the previous symbol, and mapping that difference to a
dibit — no carrier phase lock required, which makes it forgiving to decode. The
[constellation](/reference/constellation-diagram/) of a healthy π/4-DQPSK signal shows the
distinctive eight-point rosette. GopherTrunk's scope tooling can display that constellation, and
its DSP chain includes the RRC matched filter and differential-phase demodulation this modulation
family needs; TETRA support status is tracked separately in the project's protocol coverage.

## Sources

[^wiki]: [Phase-shift keying — π/4–QPSK](https://en.wikipedia.org/wiki/Phase-shift_keying#%CF%80/4%E2%80%93QPSK) — Wikipedia, for the differential π/4-shifted QPSK definition and constellation geometry.
[^diff]: [Differential coding](https://en.wikipedia.org/wiki/Differential_coding) — Wikipedia, for why encoding information in phase changes removes the need for an absolute phase reference.
