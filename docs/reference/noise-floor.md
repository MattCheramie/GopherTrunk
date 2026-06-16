---
slug: noise-floor
title: Noise floor
entry_type: term
category: rf-fundamentals
description: The noise floor is the ever-present background level of thermal and environmental noise in a receiver; a signal must rise above it to be usable.
keywords: noise floor, thermal noise, background noise, sensitivity, dBm
aka: [noise floor]
autolink: true
infobox:
  - { label: Type, value: Background noise level }
  - { label: Unit, value: dBm }
  - { label: Sources, value: Thermal noise, environment, receiver }
see_also: [signal-to-noise-ratio, dbm, low-noise-amplifier, attenuation]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "Noise floor (Wikipedia)", url: https://en.wikipedia.org/wiki/Noise_floor }
---

The **noise floor** is the constant background level of random energy present in any
receiver — thermal noise in the electronics plus environmental RF. It is measured in
[dBm](/reference/dbm/) and sets the bar a signal must clear.

## How it works

Bandwidth, receiver quality, and local interference all raise or lower the floor. A
signal is only useful when it pokes above it; the margin is the
[SNR](/reference/signal-to-noise-ratio/).

## Relevance to SDR

A [low-noise amplifier](/reference/low-noise-amplifier/) and a quiet install lower the
effective floor, while nearby electronics (USB, chargers, LED lighting) raise it.
