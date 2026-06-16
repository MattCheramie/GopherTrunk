---
slug: c4fm
title: C4FM
entry_type: technology
category: modulation
description: C4FM is the four-level continuous-phase FSK modulation used by P25 Phase 1 and Yaesu System Fusion, carrying 2 bits per symbol at 4800 baud.
keywords: C4FM, four-level FSK, 4FSK, P25 Phase 1, System Fusion, 4800 baud, CQPSK
aka: [C4FM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (4-level FSK) }
  - { label: Symbol rate, value: 4800 baud (9600 bps) }
  - { label: Used by, value: P25 Phase 1, System Fusion }
see_also: [frequency-shift-keying, cqpsk, project-25, p25-phase-1, system-fusion-ysf, symbol-rate]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**C4FM** (compatible four-level FM) is the four-level
[FSK](/reference/frequency-shift-keying/) modulation used by
[P25 Phase 1](/reference/p25-phase-1/) and [System Fusion](/reference/system-fusion-ysf/).
The carrier sits at one of four frequency deviations per [symbol](/reference/symbol-rate/),
carrying 2 bits each.

## How it works

C4FM runs at 4800 baud (9600 bps) and is paired with [CQPSK](/reference/cqpsk/): the two
are designed so a single receiver detects both transmit paths. Its four levels appear as
four clusters on a [constellation](/reference/constellation-diagram/) and three stacked
eyes on an [eye diagram](/reference/eye-diagram/).

## Relevance to SDR

Recognising healthy C4FM symbols is central to decoding P25 Phase 1 and Fusion; the
scopes reveal SNR, tuning, and timing problems at a glance.
