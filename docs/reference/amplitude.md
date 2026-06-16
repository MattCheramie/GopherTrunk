---
slug: amplitude
title: Amplitude
entry_type: term
category: rf-fundamentals
description: Amplitude is the magnitude or strength of a wave; in radio it corresponds to signal power and is the quantity varied by amplitude modulation.
keywords: amplitude, signal strength, magnitude, power, AM
infobox:
  - { label: Type, value: Wave property }
  - { label: Represents, value: Strength / power }
  - { label: Reported as, value: Power level (dBm / dBFS) }
see_also: [phase, decibel, dbm, amplitude-modulation, signal-to-noise-ratio]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/radio-waves/ }
external:
  - { title: "Amplitude (Wikipedia)", url: https://en.wikipedia.org/wiki/Amplitude }
---

**Amplitude** is the magnitude — the "height" — of a wave. For a
[radio wave](/reference/radio-wave/) it corresponds to signal strength, which a
receiver reports as a power level in [dBm](/reference/dbm/) or
[dBFS](/reference/dbfs/).

## How it works

A larger amplitude carries more power. As a signal spreads from a transmitter and
passes obstacles, its amplitude falls ([path loss](/reference/path-loss/)), which is
why distant signals are weaker.

## Relevance to SDR

Varying amplitude is the basis of [amplitude modulation](/reference/amplitude-modulation/)
and part of what an [IQ](/reference/iq-data/) sample encodes (its distance from the
origin). A signal's amplitude relative to the [noise floor](/reference/noise-floor/)
sets its [SNR](/reference/signal-to-noise-ratio/).
