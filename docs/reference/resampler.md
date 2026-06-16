---
slug: resampler
title: Resampler
entry_type: algorithm
category: sdr-dsp
description: A resampler converts a stream from one sample rate to another — essential when an SDR's native rate doesn't match the rate a decoder needs.
keywords: resampler, resampling, sample-rate conversion, interpolation, decimation, fractional resampling
aka: [resampler, resampling]
autolink: true
see_also: [sample-rate, decimation, digital-filter, nyquist-theorem]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Sample-rate conversion (Wikipedia)", url: https://en.wikipedia.org/wiki/Sample-rate_conversion }
---

A **resampler** converts a sample stream from one [sample rate](/reference/sample-rate/)
to another. SDRs rarely produce exactly the rate a decoder wants (a P25 channel needs a
multiple of 4800 baud, say), so a resampler bridges the two — by a whole-number ratio or
a fractional one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An input stream of samples at one spacing converted to an output stream at a different spacing." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="30" cy="40" r="3"/><circle cx="58" cy="40" r="3"/><circle cx="86" cy="40" r="3"/><circle cx="114" cy="40" r="3"/><circle cx="142" cy="40" r="3"/><circle cx="170" cy="40" r="3"/></g>
  <text x="100" y="26" font-size="8.5" fill="currentColor">input rate</text>
  <line x1="200" y1="55" x2="240" y2="55" stroke="currentColor" marker-end="url(#rsar)"/><text x="220" y="48" text-anchor="middle" font-size="8" fill="currentColor">resample</text>
  <g fill="currentColor"><circle cx="280" cy="80" r="3"/><circle cx="320" cy="80" r="3"/><circle cx="360" cy="80" r="3"/><circle cx="400" cy="80" r="3"/></g>
  <text x="350" y="100" font-size="8.5" fill="currentColor">output rate</text>
  <defs><marker id="rsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A resampler changes the sample rate (here reducing it) while preserving the underlying signal.</figcaption>
</figure>

## Overview

Resampling combines interpolation, filtering, and [decimation](/reference/decimation/);
done carelessly it causes [aliasing](/reference/aliasing/), so a resampler always
includes an anti-alias [filter](/reference/digital-filter/).
