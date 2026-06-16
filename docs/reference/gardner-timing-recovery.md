---
slug: gardner-timing-recovery
title: Gardner timing recovery
entry_type: algorithm
category: algorithms
description: Gardner timing recovery is a feedback algorithm that estimates symbol timing error from samples taken at the symbol and half-symbol points, without needing carrier phase.
keywords: Gardner timing recovery, symbol timing, timing error detector, clock recovery, non-data-aided
aka: [Gardner timing recovery, Gardner]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing algorithm }
  - { label: Feature, value: Independent of carrier phase }
  - { label: Use, value: Clock recovery for digital modems }
see_also: [clock-recovery, mueller-muller-timing-recovery, symbol-rate, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/clock-recovery/ }
external:
  - { title: "Gardner (timing recovery) (Wikipedia)", url: https://en.wikipedia.org/wiki/Symbol_synchronization }
---

**Gardner timing recovery** is a feedback algorithm that estimates
[symbol-timing](/reference/clock-recovery/) error using samples at the symbol and
half-symbol instants. A useful property is that it works **independently of carrier
phase**.

## How it works

Its timing-error detector drives a loop that nudges the sampling instant toward the centre
of each [symbol](/reference/symbol-rate/), where the [eye](/reference/eye-diagram/) is
widest, tracking small clock drift.

## Relevance to SDR

Gardner recovery is a common choice in SDR demodulators for locking symbol timing on PSK
and QAM signals.
