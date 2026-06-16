---
slug: digital-down-converter
title: Digital down-converter (DDC)
entry_type: term
category: sdr-dsp
description: A digital down-converter shifts a channel within a wideband IQ stream to baseband using a numerically controlled oscillator, then filters and decimates it for processing.
keywords: digital down converter, DDC, NCO, channelizer, mixing, decimation
aka: [digital down-converter, DDC]
autolink: true
infobox:
  - { label: Type, value: DSP block }
  - { label: Does, value: Shift channel to baseband, filter, decimate }
  - { label: Uses, value: Numerically controlled oscillator }
see_also: [local-oscillator, decimation, digital-filter, iq-data, demodulation]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Digital down converter (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_down_converter }
---

A **digital down-converter** (**DDC**) shifts a chosen channel within a wideband
[IQ](/reference/iq-data/) stream to baseband using a numerically controlled oscillator
(a software [local oscillator](/reference/local-oscillator/)), then
[filters](/reference/digital-filter/) and [decimates](/reference/decimation/) it.

## How it works

Multiplying the IQ by a rotating tone centres the target channel at zero frequency; a
low-pass filter isolates it and decimation lowers the rate. Many DDCs can run in parallel
from one capture.

## Relevance to SDR

The DDC is how GopherTrunk extracts a control channel and multiple voice channels from a
single wideband capture.
