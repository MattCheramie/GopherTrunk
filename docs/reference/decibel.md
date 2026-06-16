---
slug: decibel
title: Decibel (dB)
entry_type: term
category: rf-fundamentals
description: The decibel is a logarithmic unit expressing the ratio between two power levels; in radio it makes gains, losses, and huge dynamic ranges easy to add and compare.
keywords: decibel, dB, logarithmic, ratio, gain, loss, power
aka: [decibel, dB]
autolink: true
infobox:
  - { label: Type, value: Logarithmic ratio unit }
  - { label: Formula, value: "dB = 10·log10(P1/P2)" }
  - { label: Rules, value: "+3 dB ≈ ×2, +10 dB = ×10" }
see_also: [dbm, dbfs, signal-to-noise-ratio, noise-floor, path-loss, antenna-gain]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "Decibel (Wikipedia)", url: https://en.wikipedia.org/wiki/Decibel }
---

The **decibel** (**dB**) is a logarithmic unit expressing the ratio between two power
levels: *dB = 10·log₁₀(P₁/P₂)*. Radio relies on it because signal powers span an
enormous range and because gains and losses then simply **add**.

## How it works

Two anchors cover most mental arithmetic: **+3 dB ≈ double the power**, and **+10 dB =
ten times**. A chain of amplifier gains and cable losses becomes a running sum in dB.

## Relevance to SDR

Absolute power is given in [dBm](/reference/dbm/); digital headroom in
[dBFS](/reference/dbfs/). [Antenna gain](/reference/antenna-gain/),
[path loss](/reference/path-loss/), and [SNR](/reference/signal-to-noise-ratio/) are
all expressed in decibels.
