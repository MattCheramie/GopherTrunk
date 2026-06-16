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
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Cascaded integrator–comb filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter }
---

A **CIC filter** (cascaded integrator–comb) is a [digital filter](/reference/digital-filter/)
built from only integrators and combs — **no multipliers** — making it very efficient for
large-ratio [decimation](/reference/decimation/) or interpolation.

## How it works

Integrator stages run at the high rate and comb stages at the low rate, changing the
[sample rate](/reference/sample-rate/) cheaply. Its gentle passband is usually followed by
a short compensation FIR filter.

## Relevance to SDR

CIC filters are common in the front of an SDR channeliser, where huge decimation ratios
must be done with minimal computation.
