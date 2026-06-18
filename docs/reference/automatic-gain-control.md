---
slug: automatic-gain-control
title: Automatic gain control (AGC)
entry_type: term
category: sdr-dsp
description: Automatic gain control adjusts amplification to keep a signal at a usable level; in SDR it can be hardware or software, and is often set manually for stable decoding.
keywords: AGC, automatic gain control, gain, headroom, clipping, pumping
aka: [automatic gain control, AGC]
autolink: true
infobox:
  - { label: Type, value: Gain-management technique }
  - { label: Goal, value: Keep level usable, avoid clipping }
  - { label: Note, value: Manual gain often best for decoding }
see_also: [dbfs, analog-to-digital-converter, noise-floor, ppm-frequency-correction]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/rf-sdr/gain-and-agc/ }
external:
  - { title: "Automatic gain control (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_gain_control }
---

**Automatic gain control** (**AGC**) adjusts amplification to keep a signal at a usable
level — high enough above the [noise floor](/reference/noise-floor/) but below the
[ADC](/reference/analog-to-digital-converter/)'s clipping ceiling (0
[dBFS](/reference/dbfs/)).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An input whose amplitude varies wildly, and an output of roughly constant amplitude after AGC." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="22" font-size="9" fill="currentColor">input (varying level)</text>
  <path d="M20 45 q6 -6 12 0 t12 0 q6 -22 12 0 t12 0 q6 -22 12 0 t12 0 q6 -4 12 0 t12 0 q6 -4 12 0 t12 0 q6 -16 12 0 t12 0 q6 -16 12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="20" y="95" font-size="9" fill="currentColor">output (levelled)</text>
  <path d="M20 118 q6 -13 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
</svg>
<figcaption>AGC continuously adjusts gain so the output level stays roughly constant despite a fading input.</figcaption>
</figure>

## How it works

AGC can live in the tuner hardware or in software. For decoding a fixed system it can
"pump" — ramping up in quiet moments and clamping on strong bursts — which is why a
well-chosen **manual gain** is often preferred.

## Relevance to SDR

Setting gain correctly is the single setting beginners most often get wrong; see the
gain lesson for a practical routine.
