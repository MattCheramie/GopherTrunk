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

## How it works

Its [sample rate](/reference/sample-rate/) sets how much
[bandwidth](/reference/bandwidth/) can be captured (per the
[Nyquist theorem](/reference/nyquist-theorem/)), and its range defines full scale —
exceed it and the signal **clips** at 0 [dBFS](/reference/dbfs/).

## Relevance to SDR

Setting [gain](/reference/automatic-gain-control/) so strong signals stay below the ADC's
ceiling, without burying weak ones in noise, is central to clean reception.
