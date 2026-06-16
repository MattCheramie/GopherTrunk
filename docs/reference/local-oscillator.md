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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A mixer combining an incoming RF signal and a local-oscillator tone to produce a shifted lower frequency." xmlns="http://www.w3.org/2000/svg">
  <text x="30" y="45" font-size="9" fill="currentColor">RF in</text>
  <line x1="30" y1="55" x2="150" y2="55" stroke="currentColor" stroke-width="1.3"/>
  <circle cx="175" cy="55" r="22" fill="none" stroke="currentColor" stroke-width="1.4"/><path d="M160 40 L190 70 M190 40 L160 70" stroke="currentColor" stroke-width="1.2"/>
  <text x="175" y="100" font-size="9" fill="currentColor" text-anchor="middle">LO (tunable)</text><line x1="175" y1="92" x2="175" y2="78" stroke="currentColor"/>
  <line x1="197" y1="55" x2="320" y2="55" stroke="currentColor" stroke-width="1.3" marker-end="url(#loar)"/>
  <text x="330" y="51" font-size="9" fill="currentColor">shifted</text><text x="330" y="63" font-size="9" fill="currentColor">(IF/baseband)</text>
  <text x="120" y="30" font-size="9" fill="currentColor" text-anchor="middle">tuning = changing the LO</text>
  <defs><marker id="loar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The local oscillator sets the tuned frequency: the mixer shifts the chosen band down for digitising.</figcaption>
</figure>

## How it works

In hardware the LO drives an analog mixer; in software a numerically controlled
oscillator performs the same shift digitally inside a
[digital down-converter](/reference/digital-down-converter/). LO inaccuracy shows up as a
[PPM frequency error](/reference/ppm-frequency-correction/).

## Relevance to SDR

The LO sets which part of the spectrum lands in the ADC's window, so its accuracy and
stability directly affect tuning.
