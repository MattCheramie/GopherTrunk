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
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
external:
  - { title: "Digital down converter (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_down_converter }
---

A **digital down-converter** (**DDC**) shifts a chosen channel within a wideband
[IQ](/reference/iq-data/) stream to baseband using a numerically controlled oscillator
(a software [local oscillator](/reference/local-oscillator/)), then
[filters](/reference/digital-filter/) and [decimates](/reference/decimation/) it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 110" role="img" aria-label="Wideband IQ into a numerically controlled oscillator mixer, then a low-pass filter, then decimation, producing one narrow channel." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="40" y="63">wide IQ</text>
    <circle cx="120" cy="58" r="20" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M106 44 L134 72 M134 44 L106 72" stroke="currentColor" stroke-width="1.1"/><text x="120" y="96" font-size="8">NCO</text><line x1="120" y1="92" x2="120" y2="78" stroke="currentColor"/>
    <rect x="172" y="44" width="74" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="209" y="62">low-pass</text>
    <rect x="278" y="44" width="84" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="320" y="62">decimate</text>
    <rect x="394" y="44" width="96" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="442" y="62">one channel</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="70" y1="58" x2="100" y2="58"/><line x1="140" y1="58" x2="171" y2="58"/><line x1="246" y1="58" x2="277" y2="58"/><line x1="362" y1="58" x2="393" y2="58"/></g>
  </g>
</svg>
<figcaption>A digital down-converter shifts a channel to baseband (NCO), filters it, and decimates — the heart of channelising.</figcaption>
</figure>

## How it works

Multiplying the IQ by a rotating tone centres the target channel at zero frequency; a
low-pass filter isolates it and decimation lowers the rate. Many DDCs can run in parallel
from one capture.

## Relevance to SDR

The DDC is how GopherTrunk extracts a control channel and multiple voice channels from a
single wideband capture.
