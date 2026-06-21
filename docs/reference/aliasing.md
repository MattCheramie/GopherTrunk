---
slug: aliasing
title: Aliasing
entry_type: term
category: sdr-dsp
description: Aliasing is the appearance of out-of-band energy at a false frequency when a signal is sampled too slowly for its bandwidth; anti-alias filtering and adequate sample rate prevent it.
keywords: aliasing, fold-back, anti-alias filter, undersampling, false signal, Nyquist
aka: [aliasing]
autolink: true
infobox:
  - { label: Type, value: Sampling artefact }
  - { label: Cause, value: Sampling below Nyquist for the bandwidth }
  - { label: Prevented by, value: Anti-alias filter + adequate rate }
see_also: [nyquist-theorem, sample-rate, decimation, digital-filter]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Aliasing
---

**Aliasing** is when energy outside the bandwidth a [sample rate](/reference/sample-rate/)
can represent gets **folded** back into the captured spectrum, appearing at a wrong
frequency — a phantom that looks like a real signal.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A captured band between Nyquist edges with a real signal inside and an out-of-band signal folding back to a false position." xmlns="http://www.w3.org/2000/svg">
  <line x1="60" y1="105" x2="420" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="110" y1="20" x2="110" y2="115" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <line x1="330" y1="20" x2="330" y2="115" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="220" y="125" text-anchor="middle" font-size="9" fill="currentColor">captured bandwidth</text>
  <path d="M160 105 L170 55 L180 105 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/><text x="170" y="48" text-anchor="middle" font-size="8" fill="currentColor">real</text>
  <path d="M360 105 L370 70 L380 105 Z" fill="none" stroke="currentColor" stroke-opacity="0.5"/><text x="372" y="63" text-anchor="middle" font-size="8" fill="currentColor">out of band</text>
  <path d="M255 105 L265 78 L275 105 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-dasharray="3 2"/><text x="265" y="71" text-anchor="middle" font-size="8" fill="currentColor">alias!</text>
  <path d="M368 73 q-50 -28 -100 0" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="2 2"/>
</svg>
<figcaption>Aliasing: energy beyond the captured bandwidth folds back to a false position inside it.</figcaption>
</figure>

## How it works

It is the [Nyquist theorem](/reference/nyquist-theorem/) violated. SDR front-ends include
anti-alias filtering, and choosing an adequate sample rate keeps real signals safely
inside the usable window. It also constrains the order of
[filtering and decimation](/reference/decimation/).

## Relevance to SDR

Recognising an alias prevents chasing signals that are not really where they appear.

## Sources

[^wiki]: [Aliasing](https://en.wikipedia.org/wiki/Aliasing) — Wikipedia, on out-of-band energy folding to false frequencies when undersampled.
