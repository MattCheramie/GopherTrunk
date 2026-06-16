---
slug: dbfs
title: dBFS
entry_type: term
category: rf-fundamentals
description: dBFS is level expressed in decibels relative to digital full scale, used inside an SDR; 0 dBFS is the ADC's maximum, and exceeding it causes clipping.
keywords: dBFS, decibels full scale, ADC, clipping, headroom, digital level
aka: [dBFS]
autolink: true
infobox:
  - { label: Type, value: Digital level unit }
  - { label: Reference, value: ADC full scale (0 dBFS max) }
  - { label: Risk at 0 dBFS, value: Clipping / distortion }
see_also: [decibel, dbm, analog-to-digital-converter, automatic-gain-control]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/gain-and-agc/ }
external:
  - { title: "dBFS (Wikipedia)", url: https://en.wikipedia.org/wiki/DBFS }
---

**dBFS** (decibels relative to **full scale**) measures level inside the digital domain
of an SDR. Here **0 dBFS is the ceiling** — the largest value the
[ADC](/reference/analog-to-digital-converter/) can represent — and real samples sit
below it as negative numbers.

## How it works

If a signal reaches 0 dBFS the converter **clips**, flattening peaks and spraying
distortion across the spectrum. Leaving headroom below 0 dBFS keeps the signal clean.

## Relevance to SDR

[Gain](/reference/automatic-gain-control/) should be set so the strongest signal peaks
comfortably under 0 dBFS. dBm describes the world outside the radio;
[dBFS](/reference/dbfs/) describes the world inside it.
