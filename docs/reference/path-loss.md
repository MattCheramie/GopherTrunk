---
slug: path-loss
title: Path loss
entry_type: term
category: rf-fundamentals
description: Path loss is the attenuation a radio signal suffers travelling from transmitter to receiver, dominated by the spreading of energy over distance and obstacles in the way.
keywords: path loss, free space loss, propagation loss, distance, dB
aka: [path loss]
autolink: true
infobox:
  - { label: Type, value: Propagation attenuation }
  - { label: Unit, value: Decibels (dB) }
  - { label: Grows with, value: Distance and frequency }
see_also: [attenuation, radio-propagation, decibel, signal-to-noise-ratio]
related_lessons:
  - { title: "How signals travel", url: /learn/propagation/ }
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "Path loss (Wikipedia)", url: https://en.wikipedia.org/wiki/Path_loss }
---

**Path loss** is the [attenuation](/reference/attenuation/) a signal experiences
travelling from transmitter to receiver. It is dominated by the spreading of energy
over distance, plus losses from terrain, buildings, and foliage.

## How it works

In free space, power falls with the square of distance, and loss rises with
[frequency](/reference/frequency/); real environments add much more. Path loss can
total 100+ dB over a few kilometres, which is why link budgets are done in
[decibels](/reference/decibel/).

## Relevance to SDR

Path loss explains why a distant or obstructed system arrives near the
[noise floor](/reference/noise-floor/), and why antenna height and a clear path
([propagation](/reference/radio-propagation/)) matter so much.
