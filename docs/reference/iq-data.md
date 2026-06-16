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
  - { title: "IQ data & complex signals", url: /learn/iq-data/ }
external:
  - { title: "In-phase and quadrature components (Wikipedia)", url: https://en.wikipedia.org/wiki/In-phase_and_quadrature_components }
---

**IQ data** is the stream of paired numbers an SDR produces — **I** (in-phase) and **Q**
(quadrature, 90° apart). Together each pair captures both the
[amplitude](/reference/amplitude/) and [phase](/reference/phase/) of the signal at that
instant.

## How it works

Treating each sample as a point (I, Q), its distance from the origin is amplitude and its
angle is phase — the basis of the [constellation diagram](/reference/constellation-diagram/).
IQ also lets a receiver distinguish frequencies above and below the tuned centre.

## Relevance to SDR

Everything GopherTrunk does begins with the IQ stream from the radio: tuning, filtering,
demodulation, and the scopes all operate on it.
