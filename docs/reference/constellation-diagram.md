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
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/tuning-with-scopes/ }
external:
  - { title: "Constellation diagram (Wikipedia)", url: https://en.wikipedia.org/wiki/Constellation_diagram }
---

A **constellation diagram** plots a digital signal's [symbols](/reference/symbol-rate/)
on the [IQ](/reference/iq-data/) plane: the horizontal axis is I and the vertical axis
is Q, so each point's angle is its [phase](/reference/phase/) and its distance from the
origin is its [amplitude](/reference/amplitude/).

## How it works

A clean signal places symbols in tight, well-separated clusters at their ideal points.
Noise fuzzes the clusters, a tuning offset rotates them, and clipping distorts them —
so the *shape* of the scatter diagnoses the problem.

## Relevance to SDR

GopherTrunk's constellation panel draws this live, making it the first tool for
[tuning a clean lock](/learn/tuning-with-scopes/).
