---
slug: baseband
title: Baseband
entry_type: term
category: sdr-dsp
description: Baseband is a signal centred at (or near) zero frequency, after a receiver has mixed the carrier away. SDRs deliver IQ samples at baseband for software to process.
keywords: baseband, zero-IF, complex baseband, IQ, downconversion, DC
aka: [baseband, "complex baseband"]
autolink: true
see_also: [iq-data, intermediate-frequency, local-oscillator, dc-offset]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/iq-data/ }
external:
  - { title: "Baseband (Wikipedia)", url: https://en.wikipedia.org/wiki/Baseband }
---

**Baseband** is a signal centred at (or near) **zero frequency**, after the carrier has
been mixed away. An SDR delivers **complex baseband** [IQ samples](/reference/iq-data/) —
the channel shifted down to 0 Hz — which is the natural form for software to filter and
demodulate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A signal at a high carrier frequency shifted down to sit centred on zero hertz at baseband." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="85" x2="430" y2="85" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="120" y1="85" x2="120" y2="30" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/><text x="120" y="100" text-anchor="middle" font-size="8" fill="currentColor">0 Hz</text>
  <path d="M340 85 L350 50 L360 85 Z" fill="none" stroke="currentColor"/><text x="350" y="42" text-anchor="middle" font-size="8" fill="currentColor">carrier</text>
  <path d="M110 85 L120 45 L130 85 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <path d="M340 60 q-110 -25 -215 -12" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#bbar)"/>
  <text x="240" y="108" text-anchor="middle" font-size="8.5" fill="currentColor">mixed down to baseband (0 Hz)</text>
  <defs><marker id="bbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Baseband is the channel shifted down to 0 Hz; SDRs output complex (IQ) baseband samples.</figcaption>
</figure>

## Overview

Working at complex baseband lets software represent both positive and negative
frequencies around the centre (thanks to [IQ](/reference/iq-data/)), so the whole
captured band is available for [filtering](/reference/digital-filter/) and demodulation.
