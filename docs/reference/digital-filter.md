---
slug: digital-filter
title: Digital filter
entry_type: term
category: sdr-dsp
description: A digital filter passes some frequencies and attenuates others by arithmetic on a sample stream; low-pass, band-pass, and channel filters isolate signals in an SDR.
keywords: digital filter, FIR, IIR, low-pass, band-pass, channel filter, DSP
aka: [digital filter]
autolink: true
infobox:
  - { label: Type, value: DSP operation }
  - { label: Kinds, value: FIR, IIR; low-pass, band-pass }
  - { label: Use, value: Isolate a channel, shape pulses }
see_also: [decimation, cic-filter, root-raised-cosine-filter, matched-filter, digital-down-converter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_filter
---

A **digital filter** passes some frequencies and attenuates others by performing
arithmetic on a stream of samples — no physical components.[^wiki] The main families are FIR
(finite impulse response) and IIR (infinite impulse response).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A frequency-response curve that passes a band of frequencies and attenuates those outside it." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="110" x2="440" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M40 95 L150 95 C 180 95, 180 35, 210 35 L 270 35 C 300 35, 300 95, 330 95 L 440 95" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="28" text-anchor="middle" font-size="10" fill="currentColor">passband</text>
  <text x="90" y="86" font-size="9" fill="currentColor">rejected</text><text x="390" y="86" font-size="9" fill="currentColor">rejected</text>
  <text x="240" y="130" text-anchor="middle" font-size="9" fill="currentColor">frequency →</text>
</svg>
<figcaption>A digital filter passes a chosen band of frequencies and attenuates the rest — isolating one channel.</figcaption>
</figure>

## How it works

A low-pass filter keeps frequencies below a cutoff; a band-pass keeps a chosen range. In
an SDR a narrow **channel filter** isolates one signal from a wide capture, often paired
with [decimation](/reference/decimation/).

## Relevance to SDR

Filtering is fundamental to channelising the [IQ](/reference/iq-data/) stream and to pulse
shaping; specialised forms include the [CIC](/reference/cic-filter/),
[root-raised-cosine](/reference/root-raised-cosine-filter/), and
[matched](/reference/matched-filter/) filters.

## Sources

[^wiki]: [Digital filter](https://en.wikipedia.org/wiki/Digital_filter) — Wikipedia, on FIR/IIR families and frequency response.
