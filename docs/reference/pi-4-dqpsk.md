---
slug: pi-4-dqpsk
title: π/4-DQPSK
entry_type: technology
category: modulation
description: π/4-DQPSK (π/4-shifted differential quadrature phase-shift keying) is a phase modulation that rotates the constellation by 45° each symbol, used by TETRA and other digital systems.
keywords: pi/4-DQPSK, differential QPSK, TETRA modulation, phase-shift keying, 8-point constellation
aka: ["pi/4-DQPSK", "π/4 DQPSK", differential QPSK]
autolink: true
see_also: [phase-shift-keying, cqpsk, constellation-diagram, tetra]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "π/4-QPSK (Wikipedia)", url: https://en.wikipedia.org/wiki/Phase-shift_keying#%CF%80/4%E2%80%93QPSK }
---

**π/4-DQPSK** (π/4-shifted differential quadrature [phase-shift keying](/reference/phase-shift-keying/))
is a four-symbol phase modulation in which the constellation is **rotated by 45° (π/4)
every symbol** and information is carried in the *change* of phase rather than its
absolute value. It is the air-interface modulation of [TETRA](/reference/tetra/).

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

## Overview

By alternating between two QPSK constellations offset by 45°, π/4-DQPSK guarantees a
phase transition at every symbol (helping [clock recovery](/reference/clock-recovery/))
while avoiding transitions through the origin, which keeps the signal's amplitude
envelope more constant and easier to amplify efficiently.

## Relevance

Differential encoding means the receiver only needs to measure phase *changes*, so it
tolerates a constant carrier-phase offset without an absolute phase reference. This
robustness is why TETRA and several other professional systems adopted it.
