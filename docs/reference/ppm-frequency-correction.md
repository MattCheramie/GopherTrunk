---
slug: ppm-frequency-correction
title: PPM frequency correction
entry_type: term
category: sdr-dsp
description: PPM frequency correction compensates an SDR's reference-oscillator error, measured in parts per million, so signals appear at their true frequency and digital modes lock.
keywords: PPM, parts per million, frequency error, oscillator drift, calibration, rotating constellation
aka: [PPM correction, PPM]
autolink: true
infobox:
  - { label: Type, value: Calibration parameter }
  - { label: Corrects, value: Reference-oscillator error }
  - { label: Symptom if wrong, value: Rotating constellation, no lock }
see_also: [local-oscillator, frequency, costas-loop, constellation-diagram]
related_lessons:
  - { title: "Calibration & troubleshooting", url: /learn/calibration-troubleshooting/ }
external:
  - { title: "Frequency error (Wikipedia)", url: https://en.wikipedia.org/wiki/Clock_drift }
---

**PPM frequency correction** compensates for the small error in an SDR's reference
oscillator, measured in **parts per million**, so signals land on their **true
frequency**. At UHF a 30 PPM error is several kilohertz — more than a channel's width.

## How it works

Setting the right PPM shifts the [local oscillator](/reference/local-oscillator/) so a
known reference sits exactly where it should. The error drifts a little with temperature,
so warm-up matters.

## Relevance to SDR

A wrong PPM produces the classic *rotating
[constellation](/reference/constellation-diagram/)* that won't lock — fixed by
calibration, not a better antenna.
