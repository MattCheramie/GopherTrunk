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

## How it works

Each symbol is a phase angle, read as a point on the [IQ](/reference/iq-data/) plane. A
[Costas loop](/reference/costas-loop/) recovers the carrier phase so the
[constellation](/reference/constellation-diagram/) lines up. PSK is spectrally
efficient.

## Relevance to SDR

[P25 Phase 2](/reference/p25-phase-2/) uses a PSK variant, and many satellite and
broadcast links are PSK; tracking phase accurately is key to decoding them.
