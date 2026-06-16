---
slug: phase-shift-keying
title: Phase-shift keying (PSK)
entry_type: technology
category: modulation
description: Phase-shift keying (PSK) is digital modulation that switches a carrier's phase between fixed angles; variants like QPSK and CQPSK appear in P25 Phase 2 and satellite links.
keywords: PSK, phase shift keying, BPSK, QPSK, CQPSK, constellation, carrier recovery
aka: [phase-shift keying, PSK, QPSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Carrier phase (discrete angles) }
  - { label: Used by, value: P25 Phase 2, satellite, broadcast }
see_also: [frequency-shift-keying, quadrature-amplitude-modulation, cqpsk, phase, costas-loop, constellation-diagram]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Phase-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Phase-shift_keying }
---

**Phase-shift keying** (**PSK**) is digital [modulation](/reference/modulation/) that
switches a [carrier](/reference/carrier-wave/)'s [phase](/reference/phase/) between
fixed angles while amplitude stays constant. Two phases is BPSK; four is QPSK (2 bits
per symbol).

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

Each symbol is a phase angle, read as a point on the [IQ](/reference/iq-data/) plane. A
[Costas loop](/reference/costas-loop/) recovers the carrier phase so the
[constellation](/reference/constellation-diagram/) lines up. PSK is spectrally
efficient.

## Relevance to SDR

[P25 Phase 2](/reference/p25-phase-2/) uses a PSK variant, and many satellite and
broadcast links are PSK; tracking phase accurately is key to decoding them.
