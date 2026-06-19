---
slug: constellation-diagram
title: Constellation diagram
entry_type: term
category: modulation
description: A constellation diagram plots a digital signal's symbols on the IQ plane, where position encodes phase and amplitude; tight clusters indicate a clean signal.
keywords: constellation diagram, IQ plane, symbols, signal quality, EVM, modulation
aka: [constellation diagram, constellation]
autolink: true
infobox:
  - { label: Type, value: Signal-quality display }
  - { label: Axes, value: I (in-phase) and Q (quadrature) }
  - { label: Reads, value: Phase (angle), amplitude (radius) }
see_also: [iq-data, eye-diagram, phase-shift-keying, quadrature-amplitude-modulation, phase]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Constellation_diagram
---

A **constellation diagram** plots a digital signal's [symbols](/reference/symbol-rate/)
on the [IQ](/reference/iq-data/) plane: the horizontal axis is I and the vertical axis
is Q, so each point's angle is its [phase](/reference/phase/) and its distance from the
origin is its [amplitude](/reference/amplitude/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="Two four-point constellations: tight clusters labelled clean on the left, smeared clusters labelled noisy on the right." xmlns="http://www.w3.org/2000/svg">
  <g><line x1="20" y1="100" x2="180" y2="100" stroke="currentColor" stroke-opacity="0.3"/><line x1="100" y1="25" x2="100" y2="175" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor"><circle cx="60" cy="60" r="2.5"/><circle cx="58" cy="62" r="2.5"/><circle cx="62" cy="59" r="2.5"/><circle cx="140" cy="60" r="2.5"/><circle cx="138" cy="62" r="2.5"/><circle cx="60" cy="140" r="2.5"/><circle cx="62" cy="138" r="2.5"/><circle cx="140" cy="140" r="2.5"/><circle cx="138" cy="141" r="2.5"/></g>
    <text x="100" y="192" text-anchor="middle" font-size="10" fill="currentColor">clean</text></g>
  <g><line x1="280" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.3"/><line x1="360" y1="25" x2="360" y2="175" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor" fill-opacity="0.7"><circle cx="320" cy="62" r="2.5"/><circle cx="312" cy="70" r="2.5"/><circle cx="330" cy="54" r="2.5"/><circle cx="318" cy="78" r="2.5"/><circle cx="400" cy="60" r="2.5"/><circle cx="408" cy="72" r="2.5"/><circle cx="392" cy="52" r="2.5"/><circle cx="322" cy="140" r="2.5"/><circle cx="330" cy="132" r="2.5"/><circle cx="400" cy="140" r="2.5"/><circle cx="392" cy="148" r="2.5"/></g>
    <text x="360" y="192" text-anchor="middle" font-size="10" fill="currentColor">noisy</text></g>
</svg>
<figcaption>A constellation plots symbols on the IQ plane; tight clusters decode reliably, smeared clusters bring errors.</figcaption>
</figure>

## How it works

A clean signal places symbols in tight, well-separated clusters at their ideal points.
Noise fuzzes the clusters, a tuning offset rotates them, and clipping distorts them —
so the *shape* of the scatter diagnoses the problem.

## Relevance to SDR

GopherTrunk's constellation panel draws this live, making it the first tool for
[tuning a clean lock](/learn/rf-sdr/tuning-with-scopes/).

## Sources

[^wiki]: [Constellation diagram](https://en.wikipedia.org/wiki/Constellation_diagram) — Wikipedia, for the IQ-plane representation and how it reflects signal quality.
