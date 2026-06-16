---
slug: intermediate-frequency
title: Intermediate frequency (IF)
entry_type: term
category: sdr-dsp
description: The intermediate frequency (IF) is the fixed lower frequency a superheterodyne receiver mixes the incoming signal down to, where filtering and digitising are easier.
keywords: intermediate frequency, IF, superheterodyne, mixing, low-IF, zero-IF, downconversion
aka: [IF, "intermediate frequency"]
autolink: true
see_also: [superheterodyne-receiver, local-oscillator, baseband, analog-to-digital-converter]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Intermediate frequency (Wikipedia)", url: https://en.wikipedia.org/wiki/Intermediate_frequency }
---

The **intermediate frequency** (**IF**) is the fixed, lower frequency that a
[superheterodyne receiver](/reference/superheterodyne-receiver/) shifts the wanted
signal down to before filtering and digitising. Mixing the variable input frequency to a
**constant IF** lets the receiver use one well-designed filter regardless of the tuned
channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A high radio frequency mixed with a local oscillator to produce a fixed lower intermediate frequency." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="430" y2="80" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="360" y1="80" x2="360" y2="40" stroke="currentColor" stroke-width="2"/><text x="360" y="32" text-anchor="middle" font-size="8" fill="currentColor">RF</text>
  <line x1="110" y1="80" x2="110" y2="50" stroke="currentColor" stroke-width="2"/><text x="110" y="42" text-anchor="middle" font-size="8" fill="currentColor">IF (fixed)</text>
  <path d="M360 78 q-40 -30 -120 -20 q-80 10 -130 20" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#ifar)"/>
  <text x="240" y="104" text-anchor="middle" font-size="8.5" fill="currentColor">mixer + local oscillator shift RF → IF</text>
  <defs><marker id="ifar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Mixing the tuned signal to a constant intermediate frequency lets one filter handle any channel.</figcaption>
</figure>

## Overview

Many SDRs use a **low-IF** or **zero-IF** ([baseband](/reference/baseband/)) architecture,
where the IF is at or near 0 Hz — convenient for the [ADC](/reference/analog-to-digital-converter/)
but introducing a [DC offset](/reference/dc-offset/) artefact to manage.
