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
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Digital filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_filter }
---

A **digital filter** passes some frequencies and attenuates others by performing
arithmetic on a stream of samples — no physical components. The main families are FIR
(finite impulse response) and IIR (infinite impulse response).

## How it works

A low-pass filter keeps frequencies below a cutoff; a band-pass keeps a chosen range. In
an SDR a narrow **channel filter** isolates one signal from a wide capture, often paired
with [decimation](/reference/decimation/).

## Relevance to SDR

Filtering is fundamental to channelising the [IQ](/reference/iq-data/) stream and to pulse
shaping; specialised forms include the [CIC](/reference/cic-filter/),
[root-raised-cosine](/reference/root-raised-cosine-filter/), and
[matched](/reference/matched-filter/) filters.
