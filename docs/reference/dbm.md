---
slug: dbm
title: dBm
entry_type: term
category: rf-fundamentals
description: dBm is power expressed in decibels relative to one milliwatt, giving an absolute signal-strength figure; received radio signals are negative dBm values.
keywords: dBm, decibel milliwatt, absolute power, received signal strength
aka: [dBm]
autolink: true
infobox:
  - { label: Type, value: Absolute power unit }
  - { label: Reference, value: 1 milliwatt }
  - { label: Examples, value: "0 dBm = 1 mW; −80 dBm ≈ solid signal" }
see_also: [decibel, dbfs, noise-floor, signal-to-noise-ratio]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "dBm (Wikipedia)", url: https://en.wikipedia.org/wiki/DBm }
---

**dBm** is power expressed in [decibels](/reference/decibel/) relative to **one
milliwatt**, making it an *absolute* measure of signal strength rather than a mere
ratio. 0 dBm equals 1 mW; +30 dBm is 1 watt.

## How it works

Because received radio signals are tiny fractions of a milliwatt, they are **negative**
dBm values — and the one closer to zero is stronger (−70 dBm beats −90 dBm by 100×).

## Relevance to SDR

Receiver meters report signal and [noise-floor](/reference/noise-floor/) levels in dBm;
their difference is the [SNR](/reference/signal-to-noise-ratio/) that determines whether
a signal decodes.
