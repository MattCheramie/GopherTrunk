---
slug: iq-data
title: IQ data
entry_type: term
category: sdr-dsp
description: IQ data is the stream of paired in-phase (I) and quadrature (Q) samples an SDR produces, capturing both the amplitude and phase of a signal and enabling negative frequencies.
keywords: IQ data, in-phase quadrature, complex samples, amplitude phase, baseband, constellation
aka: [IQ data, IQ samples]
autolink: true
infobox:
  - { label: Type, value: Complex sample stream }
  - { label: Components, value: I (in-phase), Q (quadrature, 90°) }
  - { label: Encodes, value: Amplitude (radius) + phase (angle) }
see_also: [software-defined-radio, phase, amplitude, constellation-diagram, sample-rate]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/rf-sdr/iq-data/ }
external:
  - { title: "In-phase and quadrature components (Wikipedia)", url: https://en.wikipedia.org/wiki/In-phase_and_quadrature_components }
---

**IQ data** is the stream of paired numbers an SDR produces — **I** (in-phase) and **Q**
(quadrature, 90° apart). Together each pair captures both the
[amplitude](/reference/amplitude/) and [phase](/reference/phase/) of the signal at that
instant.

<figure class="figure" markdown="0">
<svg viewBox="0 0 260 220" role="img" aria-label="The IQ plane with I horizontal and Q vertical; an arrow to a point shows amplitude as length and phase as angle." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="240" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="130" y1="20" x2="130" y2="200" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="232" y="124" font-size="11" fill="currentColor">I</text><text x="116" y="30" font-size="11" fill="currentColor">Q</text>
  <line x1="130" y1="110" x2="195" y2="55" stroke="currentColor" stroke-width="2"/><circle cx="195" cy="55" r="4" fill="currentColor"/>
  <path d="M165 110 A 36 36 0 0 0 150 84" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="150" y="104" font-size="10" fill="currentColor">phase</text>
  <text x="150" y="70" font-size="10" fill="currentColor" transform="rotate(-40 165 80)">amplitude</text>
</svg>
<figcaption>Each IQ sample is a point on the complex plane: distance from the origin is amplitude, angle is phase.</figcaption>
</figure>

## How it works

Treating each sample as a point (I, Q), its distance from the origin is amplitude and its
angle is phase — the basis of the [constellation diagram](/reference/constellation-diagram/).
IQ also lets a receiver distinguish frequencies above and below the tuned centre.

## Relevance to SDR

Everything GopherTrunk does begins with the IQ stream from the radio: tuning, filtering,
demodulation, and the scopes all operate on it.
