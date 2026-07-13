---
slug: phase-shift-keying
title: Phase-shift keying (PSK)
entry_type: technology
category: modulation
description: Phase-shift keying (PSK) is digital modulation that switches a carrier's phase between fixed angles; variants like QPSK and CQPSK appear in P25 Phase 2 and satellite links.
keywords: PSK, phase shift keying, BPSK, QPSK, 8PSK, CQPSK, DQPSK, constellation, carrier recovery, differential encoding
aka: [phase-shift keying, PSK, QPSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Carrier phase (discrete angles) }
  - { label: Used by, value: P25 Phase 2, satellite, broadcast }
see_also: [frequency-shift-keying, quadrature-amplitude-modulation, cqpsk, phase, costas-loop, constellation-diagram, bpsk, qpsk, 8psk, iq-modulation, pi-4-dqpsk, differential-decoding]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Phase-shift_keying
  - https://en.wikipedia.org/wiki/Costas_loop
---

**Phase-shift keying** (**PSK**) is digital [modulation](/reference/modulation/) that
switches a [carrier](/reference/carrier-wave/)'s [phase](/reference/phase/) between
fixed angles while amplitude stays constant.[^wiki] Two phases is
[BPSK](/reference/bpsk/) (1 bit/symbol); four is [QPSK](/reference/qpsk/) (2 bits/symbol);
eight is [8PSK](/reference/8psk/) (3 bits/symbol). It is one of the most spectrally
efficient and noise-robust digital modulations, which is why it dominates satellite and
deep-space links.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 220" role="img" aria-label="A QPSK constellation with four points at the diagonals of the IQ plane, each labelled with a two-bit value." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="110" x2="270" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="200" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="262" y="124" font-size="10" fill="currentColor">I</text><text x="136" y="30" font-size="10" fill="currentColor">Q</text>
  <g fill="currentColor"><circle cx="210" cy="55" r="5"/><circle cx="90" cy="55" r="5"/><circle cx="90" cy="165" r="5"/><circle cx="210" cy="165" r="5"/></g>
  <g font-size="10" fill="currentColor"><text x="218" y="50">01</text><text x="66" y="50">11</text><text x="66" y="182">10</text><text x="218" y="182">00</text></g>
</svg>
<figcaption>PSK encodes bits in the carrier's phase; QPSK uses four phase points (2 bits each). P25 Phase 2 uses a PSK variant.</figcaption>
</figure>

## How it works

Each symbol is a phase angle, read as a point on the [IQ](/reference/iq-data/) plane by
[quadrature (IQ) modulation](/reference/iq-modulation/). BPSK places two points at 0°
and 180°; QPSK places four at 45°, 135°, 225°, 315° so that adjacent points differ by
one bit ([Gray coding](/reference/gray-code/)), which minimises the bit errors caused by
a phase slip to a neighbouring point. Because all points sit at the same radius, PSK
shares FSK's constant-envelope-friendly behaviour up to a point — though the *transitions*
between points pass through lower amplitude, so it is not perfectly constant-envelope
unless the trajectory is shaped to avoid the origin.

The central challenge is **carrier recovery**. The receiver has no separate copy of the
transmitter's carrier phase, so it must regenerate it from the signal itself. A
[Costas loop](/reference/costas-loop/) does this by feeding back a phase error computed
from the recovered symbols, locking the local oscillator so the
[constellation](/reference/constellation-diagram/) lines up on the decision axes.[^costas]
A subtlety is *phase ambiguity*: a squaring or Costas loop can lock at any of the
symmetric phases (0°/90°/180°/270° for QPSK), so systems either send a known preamble to
resolve the rotation or use **differential encoding**
([differential decoding](/reference/differential-decoding/)), where information rides in
the *change* of phase between symbols rather than its absolute value — immune to a
constant rotation at the cost of about 2–3 dB in noise performance.

## Variants

- **[BPSK](/reference/bpsk/)** — 2 phases, most robust, used by GPS C/A code and PSK31.
- **[QPSK](/reference/qpsk/)** — 4 phases, doubles the rate at the same SNR as BPSK per
  bit; ubiquitous in satellite and cellular.
- **[8PSK](/reference/8psk/)** — 8 phases, 3 bits/symbol, used by EDGE and DVB-S2.
- **π/4-DQPSK** — [π/4-DQPSK](/reference/pi-4-dqpsk/), a differentially encoded QPSK that
  never passes through the origin; carries TETRA and P25 Phase 2's H-DQPSK inbound.
- **[CQPSK](/reference/cqpsk/)** — P25's linear-simulcast QPSK, symbol-compatible with
  C4FM.

Higher-order PSK packs more bits but crowds the phase circle, so 16-PSK and beyond are
rarely used — past 8PSK, [QAM](/reference/quadrature-amplitude-modulation/) (which also
uses amplitude) makes better use of the same SNR.

## Relevance to SDR

[P25 Phase 2](/reference/p25-phase-2/) uses a π/4-DQPSK-derived PSK variant, and TETRA,
most geostationary and LEO satellite downlinks, and digital broadcast (DVB-S) are PSK, so
tracking phase accurately with a Costas or [PLL](/reference/phase-locked-loop/) is central
to decoding them. On an SDR the recovered symbols cluster tightly when locked and smear
into rings when the carrier loop is unlocked — a quick visual diagnosis. GopherTrunk
handles the CQPSK/C4FM-symbol path of P25; the linear PSK physical layers of Phase 2 and
TETRA are more demanding and are tracked in its status notes rather than fully claimed.

## Sources

[^wiki]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, for the definition and the BPSK/QPSK/8PSK variants.
[^costas]: [Costas loop](https://en.wikipedia.org/wiki/Costas_loop) — Wikipedia, for carrier phase recovery and the phase-ambiguity that motivates differential encoding.
