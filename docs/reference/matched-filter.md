---
slug: matched-filter
title: Matched filter
entry_type: algorithm
category: sdr-dsp
description: A matched filter is the optimal linear filter for maximising signal-to-noise ratio against a known pulse shape, used in receivers to sharpen symbol detection.
keywords: matched filter, optimal filter, SNR, correlation, pulse shape, detection
aka: [matched filter]
autolink: true
infobox:
  - { label: Type, value: Optimal detection filter }
  - { label: Maximises, value: SNR at the sampling instant }
  - { label: Form, value: Time-reversed copy of the pulse }
see_also: [root-raised-cosine-filter, digital-filter, signal-to-noise-ratio, demodulation]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Matched filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Matched_filter }
---

A **matched filter** is the linear filter that maximises
[SNR](/reference/signal-to-noise-ratio/) against a *known* signal shape. Its impulse
response is a time-reversed copy of the pulse it is matched to.

## How it works

By correlating the incoming signal with the expected pulse, it concentrates energy at the
ideal sampling instant, giving the cleanest possible symbol decision. For RRC-shaped
signals, the receiver's [RRC](/reference/root-raised-cosine-filter/) is the matched
filter.

## Relevance to SDR

Matched filtering is a standard receive step that improves
[demodulation](/reference/demodulation/) of weak digital signals such as
[AIS](/reference/ais/) and [APRS](/reference/aprs/).
