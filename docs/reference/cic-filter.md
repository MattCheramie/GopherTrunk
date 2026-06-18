---
slug: cic-filter
title: CIC filter
entry_type: algorithm
category: sdr-dsp
description: A cascaded integrator–comb (CIC) filter is a multiplier-free filter ideal for efficient large-ratio decimation or interpolation in SDR hardware and software.
keywords: CIC filter, cascaded integrator comb, decimation, interpolation, multiplier-free, Hogenauer
aka: [CIC filter]
autolink: true
infobox:
  - { label: Type, value: Decimation/interpolation filter }
  - { label: Feature, value: No multipliers (adds/delays only) }
  - { label: Use, value: Large-ratio rate change }
see_also: [decimation, digital-filter, sample-rate, digital-down-converter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
external:
  - { title: "Cascaded integrator–comb filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter }
---

A **CIC filter** (cascaded integrator–comb) is a [digital filter](/reference/digital-filter/)
built from only integrators and combs — **no multipliers** — making it very efficient for
large-ratio [decimation](/reference/decimation/) or interpolation.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A cascaded integrator-comb block diagram: integrators, a rate change, then comb stages." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="45" width="50" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="64">∫</text>
    <rect x="90" y="45" width="50" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="115" y="64">∫</text>
    <rect x="155" y="45" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="185" y="63">↓ R</text>
    <rect x="230" y="45" width="50" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="255" y="64">comb</text>
    <rect x="290" y="45" width="50" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="315" y="64">comb</text>
    <text x="385" y="64">efficient<tspan x="385" dy="12">decimator</tspan></text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="80" y1="60" x2="89" y2="60"/><line x1="140" y1="60" x2="154" y2="60"/><line x1="215" y1="60" x2="229" y2="60"/><line x1="280" y1="60" x2="289" y2="60"/><line x1="340" y1="60" x2="350" y2="60"/></g>
  </g>
</svg>
<figcaption>A CIC filter uses only adders and delays (no multipliers), making it a cheap high-ratio decimator.</figcaption>
</figure>

## How it works

Integrator stages run at the high rate and comb stages at the low rate, changing the
[sample rate](/reference/sample-rate/) cheaply. Its gentle passband is usually followed by
a short compensation FIR filter.

## Relevance to SDR

CIC filters are common in the front of an SDR channeliser, where huge decimation ratios
must be done with minimal computation.
