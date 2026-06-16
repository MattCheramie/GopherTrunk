---
slug: eye-diagram
title: Eye diagram
entry_type: term
category: modulation
description: An eye diagram overlays many symbol periods of a demodulated signal against time; a wide-open "eye" indicates good timing and noise margin for decoding.
keywords: eye diagram, eye pattern, symbol timing, noise margin, jitter, 4FSK
aka: [eye diagram, eye pattern]
autolink: true
infobox:
  - { label: Type, value: Signal-quality display }
  - { label: Axes, value: Amplitude vs. time (overlaid) }
  - { label: Open eye, value: Good timing/noise margin }
see_also: [constellation-diagram, clock-recovery, symbol-rate, c4fm]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/tuning-with-scopes/ }
external:
  - { title: "Eye pattern (Wikipedia)", url: https://en.wikipedia.org/wiki/Eye_pattern }
---

An **eye diagram** overlays many short segments of a demodulated signal, each one
symbol period long, so they stack into characteristic "eye" shapes between the symbol
levels.

## How it works

The **wider and taller the eye opening**, the more margin the decoder has to sample
each [symbol](/reference/symbol-rate/) correctly. Noise and timing
[jitter](/reference/clock-recovery/) close the eye. A 4-level signal like
[C4FM](/reference/c4fm/) shows three stacked eyes.

## Relevance to SDR

GopherTrunk's eye-diagram panel shows timing and noise margin at a glance, complementing
the [constellation](/reference/constellation-diagram/) for diagnosing a marginal signal.
