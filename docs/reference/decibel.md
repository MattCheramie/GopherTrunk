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
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Decibel
---

The **decibel** (**dB**) is a logarithmic unit expressing the ratio between two power
levels: *dB = 10·log₁₀(P₁/P₂)*.[^wiki] Radio relies on it because signal powers span an
enormous range and because gains and losses then simply **add**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A ladder showing that each +10 dB step multiplies power by ten and +3 dB roughly doubles it." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <line x1="60" y1="20" x2="60" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
    <text x="70" y="30">+30 dB = ×1000</text>
    <text x="70" y="58">+20 dB = ×100</text>
    <text x="70" y="86">+10 dB = ×10</text>
    <text x="70" y="114">+3 dB ≈ ×2</text>
    <line x1="55" y1="26" x2="65" y2="26" stroke="currentColor"/>
    <line x1="55" y1="54" x2="65" y2="54" stroke="currentColor"/>
    <line x1="55" y1="82" x2="65" y2="82" stroke="currentColor"/>
    <line x1="55" y1="110" x2="65" y2="110" stroke="currentColor"/>
  </g>
</svg>
<figcaption>Decibels are logarithmic: every +10 dB is ten times the power, and gains and losses simply add.</figcaption>
</figure>

## How it works

Two anchors cover most mental arithmetic: **+3 dB ≈ double the power**, and **+10 dB =
ten times**. A chain of amplifier gains and cable losses becomes a running sum in dB.

## Relevance to SDR

Absolute power is given in [dBm](/reference/dbm/); digital headroom in
[dBFS](/reference/dbfs/). [Antenna gain](/reference/antenna-gain/),
[path loss](/reference/path-loss/), and [SNR](/reference/signal-to-noise-ratio/) are
all expressed in decibels.

## Sources

[^wiki]: [Decibel](https://en.wikipedia.org/wiki/Decibel) — Wikipedia, definition of the logarithmic ratio unit and its conventions.
