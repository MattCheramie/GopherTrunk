---
slug: mueller-muller-timing-recovery
title: Mueller–Müller timing recovery
entry_type: algorithm
category: algorithms
description: Mueller–Müller timing recovery is a decision-directed symbol-timing algorithm that needs only one sample per symbol, making it efficient for many digital demodulators.
keywords: Mueller-Muller, timing recovery, decision directed, symbol timing, clock recovery, one sample per symbol
aka: [Mueller–Müller timing recovery, Mueller-Muller]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing algorithm }
  - { label: Feature, value: One sample per symbol (decision-directed) }
  - { label: Use, value: AIS, APRS, paging demodulators }
see_also: [clock-recovery, gardner-timing-recovery, symbol-rate, ais, aprs]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/clock-recovery/ }
external:
  - { title: "Symbol synchronization (Wikipedia)", url: https://en.wikipedia.org/wiki/Symbol_synchronization }
---

**Mueller–Müller timing recovery** is a decision-directed symbol-timing algorithm that
needs only **one sample per symbol**, making it computationally efficient.

## How it works

It uses current and previous symbol decisions to estimate the timing error and drive a
loop that keeps sampling at the symbol centre — at the cost of needing reasonably reliable
decisions to start.

## Relevance to SDR

GopherTrunk uses Mueller–Müller recovery in decoders such as [AIS](/reference/ais/),
[APRS](/reference/aprs/), and signalling pipelines.
