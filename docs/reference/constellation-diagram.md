---
slug: constellation-diagram
title: Constellation diagram
entry_type: term
category: modulation
description: A constellation diagram plots a digital signal's symbols on the IQ plane, where position encodes phase and amplitude; tight clusters indicate a clean signal.
keywords: constellation diagram, IQ plane, symbols, signal quality, EVM, modulation, phase, amplitude, scatter plot
aka: [constellation diagram, constellation]
autolink: true
infobox:
  - { label: Type, value: Signal-quality display }
  - { label: Axes, value: I (in-phase) and Q (quadrature) }
  - { label: Reads, value: Phase (angle), amplitude (radius) }
see_also: [iq-data, eye-diagram, error-vector-magnitude, phase-shift-keying, quadrature-amplitude-modulation, qpsk]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Constellation_diagram
  - https://en.wikipedia.org/wiki/Error_vector_magnitude
---

A **constellation diagram** plots a digital signal's [symbols](/reference/symbol-rate/)
on the [IQ](/reference/iq-data/) plane: the horizontal axis is I and the vertical axis
is Q, so each point's angle is its [phase](/reference/phase/) and its distance from the
origin is its [amplitude](/reference/amplitude/).[^wiki] It turns an abstract modulation
scheme into a picture you can read at a glance — the ideal symbol positions are the
"constellation points," and every received symbol lands as a dot near the point it was
meant to be.

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

Each received symbol is a complex sample, I + jQ, taken at the symbol-timing instant after
demodulation. Plotting a few hundred of them builds the scatter. A clean signal places its
dots in tight, well-separated clusters centred on the ideal points; the decoder then slices
each dot to the nearest point to recover the bits. The *shape and spread* of the scatter is a
direct read-out of what is wrong with the link:

- **Noise** fuzzes every cluster outward symmetrically — the classic Gaussian blob. The tighter
  the blob relative to the spacing between points, the higher the signal-to-noise ratio.
- **A carrier-frequency offset** rotates the whole constellation slowly, smearing the clusters
  into rings or arcs until a carrier loop pulls it straight.
- **A phase error** rotates it by a fixed angle; **I/Q imbalance** shears or ellipses it;
  **compression/clipping** pushes outer points inward.
- **Timing error** (sampling off the eye centre) pulls dots toward their neighbours along the
  transition paths.

The distance from where a symbol landed to where it should have landed is the *error vector*,
and averaging its magnitude across many symbols gives the [error vector
magnitude](/reference/error-vector-magnitude/) — the single number that quantifies what the
constellation shows qualitatively.

## Variants

Different modulations have distinctive constellations: [BPSK](/reference/bpsk/) is two points on
the I axis, [QPSK](/reference/qpsk/) four points on a square,
[π/4-DQPSK](/reference/pi-4-dqpsk/) an eight-point rosette formed by two alternating QPSK sets,
and [16-QAM](/reference/quadrature-amplitude-modulation/) a 4×4 grid. Frequency modulations like
[C4FM](/reference/c4fm/) don't produce a stationary constellation in the same way — their state
lives in instantaneous frequency — so those signals are usually judged with an
[eye diagram](/reference/eye-diagram/) instead. Some tools draw *transition trajectories* (the
continuous path the signal takes between symbols) to reveal whether the modulation avoids the
origin, as π/4-DQPSK deliberately does.

## Relevance to SDR

The constellation is the first instrument an operator reaches for when a digital signal will not
decode: it separates "weak but clean" (tight small clusters) from "strong but distorted"
(smeared or rotated clusters), which point to very different fixes — more gain versus retuning,
reducing gain to stop compression, or correcting an [I/Q imbalance](/reference/iq-imbalance/).
GopherTrunk's constellation panel draws this live from the post-demod symbol stream, making it
the primary tool for [tuning a clean lock](/learn/rf-sdr/tuning-with-scopes/) alongside the
eye-diagram and EVM read-out.

## Sources

[^wiki]: [Constellation diagram](https://en.wikipedia.org/wiki/Constellation_diagram) — Wikipedia, for the IQ-plane representation and how it reflects signal quality.
[^evm]: [Error vector magnitude](https://en.wikipedia.org/wiki/Error_vector_magnitude) — Wikipedia, for how symbol-position error is quantified from the constellation.
