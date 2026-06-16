---
slug: clock-recovery
title: Clock recovery
entry_type: term
category: sdr-dsp
description: Clock recovery determines a digital signal's symbol timing from the signal itself, so the receiver samples each symbol at its centre where the eye is widest.
keywords: clock recovery, symbol timing, timing recovery, symbol synchronization, Gardner, Mueller-Muller
aka: [clock recovery, symbol timing]
autolink: true
infobox:
  - { label: Type, value: Timing-synchronisation stage }
  - { label: Recovers, value: Symbol timing from the signal }
  - { label: Algorithms, value: Gardner, Mueller–Müller }
see_also: [gardner-timing-recovery, mueller-muller-timing-recovery, symbol-rate, eye-diagram, demodulation]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/clock-recovery/ }
external:
  - { title: "Clock recovery (Wikipedia)", url: https://en.wikipedia.org/wiki/Clock_recovery }
---

**Clock recovery** determines a digital signal's [symbol](/reference/symbol-rate/) timing
from the signal itself, since the transmitter's clock is not shared. It lets the receiver
sample each symbol at its **centre**, where the [eye](/reference/eye-diagram/) is widest.

## How it works

A timing-recovery loop watches where transitions fall and nudges the sampling instant to
stay centred, tracking small clock drift. Common algorithms are
[Gardner](/reference/gardner-timing-recovery/) and
[Mueller–Müller](/reference/mueller-muller-timing-recovery/).

## Relevance to SDR

Loss of symbol lock — from low SNR or [multipath](/reference/multipath-propagation/) —
closes the eye and breaks the decode, a key thing the scopes reveal.
