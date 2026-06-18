---
slug: dc-offset
title: DC offset (DC spike)
entry_type: term
category: sdr-dsp
description: A DC offset is a constant component in an SDR's IQ stream that shows as a spike at the spectrum centre (0 Hz) — a zero-IF receiver artefact, not a real signal.
keywords: DC offset, DC spike, center spike, zero-IF, IQ imbalance, LO leakage
aka: [DC offset, DC spike, "center spike"]
autolink: true
see_also: [baseband, iq-data, local-oscillator, fft-and-waterfall]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
external:
  - { title: "DC bias (Wikipedia)", url: https://en.wikipedia.org/wiki/DC_bias }
---

A **DC offset** is a constant (zero-frequency) component in the [IQ](/reference/iq-data/)
stream that shows up as a **spike in the exact centre** of the spectrum and waterfall.
It is an artefact of zero-IF/[baseband](/reference/baseband/) receivers — local-oscillator
leakage and converter bias — **not** a real signal on the air.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A spectrum with a sharp narrow spike exactly at the centre frequency, the DC spike, distinct from real signals elsewhere." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="95" x2="430" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M30 88 L80 90 L130 86 L180 89 L230 88 L280 90 L330 87 L380 89 L430 88" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5"/>
  <line x1="230" y1="95" x2="230" y2="35" stroke="currentColor" stroke-width="2.2"/><text x="230" y="28" text-anchor="middle" font-size="8.5" fill="currentColor">DC spike (0 Hz)</text>
  <path d="M150 95 L160 60 L170 95 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="160" y="52" text-anchor="middle" font-size="8" fill="currentColor">real signal</text>
</svg>
<figcaption>The DC spike sits exactly at the tuned centre; it's an artefact to ignore or notch, not a station.</figcaption>
</figure>

## Overview

Operators avoid it by tuning slightly off-centre so the channel of interest doesn't sit
under the spike, or by enabling a DC-blocking [filter](/reference/iir-filter/). It is a
common source of "phantom carrier" confusion for newcomers reading a
[waterfall](/reference/fast-fourier-transform/).
