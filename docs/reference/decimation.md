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
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 5: Tuning & channelization", url: /blog/deep-dives/sdr-internals-05-tuning-channelization/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Downsampling_(signal_processing)
---

**Decimation** reduces a signal's [sample rate](/reference/sample-rate/) by keeping only
every Nth sample, after a low-pass [filter](/reference/digital-filter/) removes the
frequencies that would otherwise [alias](/reference/aliasing/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A dense row of samples reduced to a sparse row by keeping every fourth sample after filtering." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="30" cy="40" r="3"/><circle cx="55" cy="40" r="3"/><circle cx="80" cy="40" r="3"/><circle cx="105" cy="40" r="3"/><circle cx="130" cy="40" r="3"/><circle cx="155" cy="40" r="3"/><circle cx="180" cy="40" r="3"/><circle cx="205" cy="40" r="3"/><circle cx="230" cy="40" r="3"/><circle cx="255" cy="40" r="3"/><circle cx="280" cy="40" r="3"/><circle cx="305" cy="40" r="3"/></g>
  <text x="350" y="44" font-size="9" fill="currentColor">high rate</text>
  <line x1="160" y1="55" x2="160" y2="78" stroke="currentColor" marker-end="url(#dcar)"/><text x="200" y="72" font-size="9" fill="currentColor">keep every 4th (÷4)</text>
  <g fill="currentColor"><circle cx="30" cy="95" r="3"/><circle cx="130" cy="95" r="3"/><circle cx="230" cy="95" r="3"/><circle cx="330" cy="95" r="3"/></g>
  <text x="370" y="99" font-size="9" fill="currentColor">low rate</text>
  <defs><marker id="dcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Decimation lowers the sample rate after filtering, cutting the data the rest of the pipeline must process.</figcaption>
</figure>

## How it works

The order matters: **filter first, then decimate**, so out-of-band energy cannot fold
back. Once a narrow channel is isolated, decimating dramatically cuts the data the rest of
the pipeline must process.

## Relevance to SDR

Decimation is what makes running [many channels](/reference/digital-down-converter/) from
one capture computationally feasible.

## Sources

[^wiki]: [Decimation (signal processing)](https://en.wikipedia.org/wiki/Downsampling_(signal_processing)) — Wikipedia, on filter-then-downsample and anti-alias requirements.
