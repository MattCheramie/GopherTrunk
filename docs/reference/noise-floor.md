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
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
external:
  - { title: "Noise floor (Wikipedia)", url: https://en.wikipedia.org/wiki/Noise_floor }
---

The **noise floor** is the constant background level of random energy present in any
receiver — thermal noise in the electronics plus environmental RF. It is measured in
[dBm](/reference/dbm/) and sets the bar a signal must clear.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A wandering noisy baseline across frequency, marked as the noise floor, with no signal present." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 95 L60 90 L90 99 L120 88 L150 96 L180 86 L210 98 L240 90 L270 100 L300 88 L330 95 L360 87 L390 97 L420 91" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="30" y1="93" x2="430" y2="93" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="30" y="40" font-size="11" fill="currentColor">noise floor — the level a signal must rise above</text>
  <text x="230" y="118" text-anchor="middle" font-size="10" fill="currentColor">frequency →</text>
</svg>
<figcaption>The noise floor is the ever-present background from receiver and environmental noise.</figcaption>
</figure>

## How it works

Bandwidth, receiver quality, and local interference all raise or lower the floor. A
signal is only useful when it pokes above it; the margin is the
[SNR](/reference/signal-to-noise-ratio/).

## Relevance to SDR

A [low-noise amplifier](/reference/low-noise-amplifier/) and a quiet install lower the
effective floor, while nearby electronics (USB, chargers, LED lighting) raise it.
