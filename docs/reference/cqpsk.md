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
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**CQPSK** (compatible QPSK, also linear simulcast modulation, LSM) is the linear
[phase-modulation](/reference/phase-shift-keying/) counterpart to
[C4FM](/reference/c4fm/) used on [P25](/reference/project-25/). It produces the **same
symbol stream** as C4FM so a single demodulator can receive either.

## How it works

CQPSK uses a linear amplifier and shapes the signal so its phase trajectory matches
C4FM's four levels. Simulcast systems favour it because linear modulation behaves
better when overlapping transmitters are received together.

## Relevance to SDR

A P25 receiver can demodulate both C4FM and CQPSK; on the
[constellation](/reference/constellation-diagram/) the recovered symbols look alike.
