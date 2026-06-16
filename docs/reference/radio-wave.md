---
slug: radio-wave
title: Radio wave
entry_type: term
category: rf-fundamentals
description: A radio wave is electromagnetic radiation in the radio frequency range, used to carry information wirelessly by varying its amplitude, frequency, or phase.
keywords: radio wave, electromagnetic radiation, RF, carrier, propagation
aka: [radio wave, radio waves]
autolink: true
infobox:
  - { label: Type, value: Electromagnetic radiation }
  - { label: Frequency range, value: ~3 kHz – 300 GHz }
  - { label: Speed, value: ~299,792,458 m/s (vacuum) }
see_also: [electromagnetic-spectrum, frequency, wavelength, carrier-wave, modulation]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/radio-waves/ }
external:
  - { title: "Radio wave (Wikipedia)", url: https://en.wikipedia.org/wiki/Radio_wave }
---

A **radio wave** is electromagnetic radiation whose [frequency](/reference/frequency/)
lies in the radio range of the [electromagnetic spectrum](/reference/electromagnetic-spectrum/).
Radio waves travel at the speed of light and carry information wirelessly when their
[amplitude](/reference/amplitude/), frequency, or [phase](/reference/phase/) is varied
— a process called [modulation](/reference/modulation/).

## How it works

A transmitter drives alternating current into an [antenna](/reference/antenna/),
radiating a self-propagating electric and magnetic field. A distant antenna converts
the passing field back into a tiny current that a receiver amplifies and decodes.

## Relevance to SDR

Radio waves are the raw input to any receiver. An SDR digitises a slice of them into
[IQ data](/reference/iq-data/) for software to process.
