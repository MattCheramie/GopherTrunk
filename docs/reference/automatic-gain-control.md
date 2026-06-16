---
slug: automatic-gain-control
title: Automatic gain control (AGC)
entry_type: term
category: sdr-dsp
description: Automatic gain control adjusts amplification to keep a signal at a usable level; in SDR it can be hardware or software, and is often set manually for stable decoding.
keywords: AGC, automatic gain control, gain, headroom, clipping, pumping
aka: [automatic gain control, AGC]
autolink: true
infobox:
  - { label: Type, value: Gain-management technique }
  - { label: Goal, value: Keep level usable, avoid clipping }
  - { label: Note, value: Manual gain often best for decoding }
see_also: [dbfs, analog-to-digital-converter, noise-floor, ppm-frequency-correction]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/gain-and-agc/ }
external:
  - { title: "Automatic gain control (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_gain_control }
---

**Automatic gain control** (**AGC**) adjusts amplification to keep a signal at a usable
level — high enough above the [noise floor](/reference/noise-floor/) but below the
[ADC](/reference/analog-to-digital-converter/)'s clipping ceiling (0
[dBFS](/reference/dbfs/)).

## How it works

AGC can live in the tuner hardware or in software. For decoding a fixed system it can
"pump" — ramping up in quiet moments and clamping on strong bursts — which is why a
well-chosen **manual gain** is often preferred.

## Relevance to SDR

Setting gain correctly is the single setting beginners most often get wrong; see the
gain lesson for a practical routine.
