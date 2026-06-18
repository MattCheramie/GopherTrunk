---
slug: cqpsk
title: CQPSK
entry_type: technology
category: modulation
description: CQPSK (compatible QPSK) is the linear phase-modulation counterpart to C4FM used on P25, producing the same symbols so one demodulator can handle both transmit paths.
keywords: CQPSK, compatible QPSK, LSM, linear simulcast modulation, P25, phase modulation
aka: [CQPSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (phase) }
  - { label: Related to, value: C4FM (same symbols) }
  - { label: Used by, value: P25 (linear/simulcast path) }
see_also: [phase-shift-keying, c4fm, project-25, constellation-diagram]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**CQPSK** (compatible QPSK, also linear simulcast modulation, LSM) is the linear
[phase-modulation](/reference/phase-shift-keying/) counterpart to
[C4FM](/reference/c4fm/) used on [P25](/reference/project-25/). It produces the **same
symbol stream** as C4FM so a single demodulator can receive either.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 210" role="img" aria-label="A QPSK constellation with four points and arcs showing the linear phase transitions between them." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="270" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="190" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M205 55 A 78 78 0 0 1 205 155" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
  <g fill="currentColor"><circle cx="205" cy="55" r="5"/><circle cx="95" cy="55" r="5"/><circle cx="95" cy="155" r="5"/><circle cx="205" cy="155" r="5"/></g>
  <text x="150" y="205" text-anchor="middle" font-size="9" fill="currentColor">linear phase transitions (compatible with C4FM detection)</text>
</svg>
<figcaption>CQPSK conveys the same symbols as C4FM on a linear (phase) path, so one receiver design handles both.</figcaption>
</figure>

## How it works

CQPSK uses a linear amplifier and shapes the signal so its phase trajectory matches
C4FM's four levels. Simulcast systems favour it because linear modulation behaves
better when overlapping transmitters are received together.

## Relevance to SDR

A P25 receiver can demodulate both C4FM and CQPSK; on the
[constellation](/reference/constellation-diagram/) the recovered symbols look alike.
