---
slug: decimation
title: Decimation
entry_type: term
category: sdr-dsp
description: Decimation reduces a signal's sample rate by keeping every Nth sample after low-pass filtering, shrinking the data rate once a channel has been isolated.
keywords: decimation, downsampling, sample rate reduction, filter then decimate, CPU
aka: [decimation]
autolink: true
infobox:
  - { label: Type, value: DSP operation }
  - { label: Does, value: Lower sample rate after filtering }
  - { label: Order, value: Filter, then decimate }
see_also: [digital-filter, cic-filter, sample-rate, aliasing, digital-down-converter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Decimation (signal processing) (Wikipedia)", url: https://en.wikipedia.org/wiki/Downsampling_(signal_processing) }
---

**Decimation** reduces a signal's [sample rate](/reference/sample-rate/) by keeping only
every Nth sample, after a low-pass [filter](/reference/digital-filter/) removes the
frequencies that would otherwise [alias](/reference/aliasing/).

## How it works

The order matters: **filter first, then decimate**, so out-of-band energy cannot fold
back. Once a narrow channel is isolated, decimating dramatically cuts the data the rest of
the pipeline must process.

## Relevance to SDR

Decimation is what makes running [many channels](/reference/digital-down-converter/) from
one capture computationally feasible.
