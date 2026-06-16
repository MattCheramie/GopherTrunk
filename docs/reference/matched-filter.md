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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A noisy input on the left and a sharp correlation peak on the right after matched filtering." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 70 L40 64 L60 78 L80 60 L100 80 L120 66 L140 74 L160 62 L180 78 L200 68" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.6"/>
  <text x="110" y="105" text-anchor="middle" font-size="9" fill="currentColor">noisy input</text>
  <line x1="225" y1="65" x2="260" y2="65" stroke="currentColor" marker-end="url(#mfar)"/>
  <line x1="280" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M280 88 L350 88 L360 30 L370 88 L440 88" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="105" text-anchor="middle" font-size="9" fill="currentColor">correlation peak</text>
  <defs><marker id="mfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A matched filter maximises SNR for a known pulse shape, producing a sharp peak the detector can time on.</figcaption>
</figure>

## How it works

By correlating the incoming signal with the expected pulse, it concentrates energy at the
ideal sampling instant, giving the cleanest possible symbol decision. For RRC-shaped
signals, the receiver's [RRC](/reference/root-raised-cosine-filter/) is the matched
filter.

## Relevance to SDR

Matched filtering is a standard receive step that improves
[demodulation](/reference/demodulation/) of weak digital signals such as
[AIS](/reference/ais/) and [APRS](/reference/aprs/).
