---
slug: numerically-controlled-oscillator
title: Numerically controlled oscillator (NCO)
entry_type: term
category: sdr-dsp
description: A numerically controlled oscillator (NCO) generates a digital sine/cosine of programmable frequency using a phase accumulator — the tunable mixer at the heart of an SDR's digital down-converter.
keywords: NCO, numerically controlled oscillator, phase accumulator, DDS, digital mixing, frequency synthesis
aka: [NCO, "numerically controlled oscillator", DDS]
autolink: true
see_also: [digital-down-converter, local-oscillator, decimation, iq-data]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Numerically controlled oscillator (Wikipedia)", url: https://en.wikipedia.org/wiki/Numerically-controlled_oscillator }
---

A **numerically controlled oscillator** (**NCO**) generates a digital sine and cosine
of any programmable frequency from a **phase accumulator** — a counter that adds a fixed
step each sample and looks up the corresponding sine value. It is the software
equivalent of a [local oscillator](/reference/local-oscillator/) and the tunable mixer
inside a [digital down-converter](/reference/digital-down-converter/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A phase accumulator adding a step each sample, feeding a sine lookup that outputs a digital tone." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="120" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="90" y="58">phase accumulator</text><text x="90" y="70" font-size="7.5">+ step each sample</text>
    <rect x="190" y="44" width="90" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="235" y="64">sine LUT</text>
    <line x1="150" y1="60" x2="189" y2="60" stroke="currentColor" marker-end="url(#ncoar)"/>
    <line x1="280" y1="60" x2="320" y2="60" stroke="currentColor" marker-end="url(#ncoar)"/>
  </g>
  <path d="M325 60 q8 -16 16 0 t16 0 t16 0 t16 0 t16 0 t16 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <defs><marker id="ncoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An NCO steps a phase accumulator each sample and looks up the sine — a precisely tunable digital oscillator.</figcaption>
</figure>

## Overview

Changing the accumulator's step instantly retunes the NCO, which is how an SDR
**digitally tunes** to a channel: multiply the [IQ](/reference/iq-data/) stream by the
NCO's output to shift that channel to [baseband](/reference/baseband/) before filtering.
