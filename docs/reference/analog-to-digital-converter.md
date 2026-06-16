---
slug: analog-to-digital-converter
title: Analog-to-digital converter (ADC)
entry_type: term
category: sdr-dsp
description: An analog-to-digital converter samples a continuous signal into discrete numbers; in an SDR its sample rate sets capture bandwidth and its full scale sets the clipping ceiling.
keywords: ADC, analog to digital converter, sampling, quantization, bit depth, clipping, dBFS
aka: [analog-to-digital converter, ADC]
autolink: true
infobox:
  - { label: Type, value: Sampling device }
  - { label: Sets, value: Capture bandwidth (rate), clipping (full scale) }
  - { label: Overflow, value: Clipping at 0 dBFS }
see_also: [sample-rate, nyquist-theorem, dbfs, automatic-gain-control, software-defined-radio]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Analog-to-digital converter (Wikipedia)", url: https://en.wikipedia.org/wiki/Analog-to-digital_converter }
---

An **analog-to-digital converter** (**ADC**) measures a continuous signal many times per
second, turning it into a stream of numbers. In an SDR it produces the
[IQ](/reference/iq-data/) samples software works on.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A smooth analog wave overlaid with a stair-step series of sampled values taken at regular intervals." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 70 C 90 15, 160 15, 230 70 S 370 125, 440 70" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <polyline points="20,70 50,52 80,33 110,28 140,40 170,58 200,70 230,70 260,82 290,98 320,103 350,95 380,80 410,70 440,62" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g fill="currentColor"><circle cx="50" cy="52" r="2.5"/><circle cx="110" cy="28" r="2.5"/><circle cx="170" cy="58" r="2.5"/><circle cx="290" cy="98" r="2.5"/><circle cx="350" cy="95" r="2.5"/></g>
  <text x="20" y="118" font-size="9" fill="currentColor">regular samples turn the continuous wave into numbers</text>
</svg>
<figcaption>The ADC measures the signal at a fixed rate, converting the continuous waveform into digital samples.</figcaption>
</figure>

## How it works

Its [sample rate](/reference/sample-rate/) sets how much
[bandwidth](/reference/bandwidth/) can be captured (per the
[Nyquist theorem](/reference/nyquist-theorem/)), and its range defines full scale —
exceed it and the signal **clips** at 0 [dBFS](/reference/dbfs/).

## Relevance to SDR

Setting [gain](/reference/automatic-gain-control/) so strong signals stay below the ADC's
ceiling, without burying weak ones in noise, is central to clean reception.
