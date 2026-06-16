---
slug: local-oscillator
title: Local oscillator
entry_type: term
category: sdr-dsp
description: A local oscillator is a tunable reference signal mixed with the incoming signal to shift a chosen band to a lower frequency; its setting is what "tuning" actually changes.
keywords: local oscillator, LO, mixer, tuning, frequency reference, NCO
aka: [local oscillator, LO]
autolink: true
infobox:
  - { label: Type, value: Reference signal source }
  - { label: Role, value: Sets the band shifted down by the mixer }
  - { label: Digital form, value: Numerically controlled oscillator }
see_also: [superheterodyne-receiver, digital-down-converter, frequency, ppm-frequency-correction]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Local oscillator (Wikipedia)", url: https://en.wikipedia.org/wiki/Local_oscillator }
---

A **local oscillator** (**LO**) is a tunable reference signal mixed with the incoming
signal to shift a chosen band down toward baseband. **Tuning a receiver is just changing
the LO frequency.**

## How it works

In hardware the LO drives an analog mixer; in software a numerically controlled
oscillator performs the same shift digitally inside a
[digital down-converter](/reference/digital-down-converter/). LO inaccuracy shows up as a
[PPM frequency error](/reference/ppm-frequency-correction/).

## Relevance to SDR

The LO sets which part of the spectrum lands in the ADC's window, so its accuracy and
stability directly affect tuning.
