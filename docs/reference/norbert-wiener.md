---
slug: norbert-wiener
title: Norbert Wiener
entry_type: person
category: people
description: "Norbert Wiener (1894–1964) was an American mathematician who founded cybernetics and derived the Wiener filter for optimal signal estimation in noise."
keywords: Norbert Wiener, cybernetics, Wiener filter, Wiener-Hopf equation, optimal filtering, MMSE, stochastic processes, signal estimation
aka: [Norbert Wiener]
autolink: true
infobox:
  - { label: Lived, value: "1894–1964" }
  - { label: Field, value: "Mathematics, engineering" }
  - { label: Known for, value: "Cybernetics; the Wiener filter" }
see_also: [matched-filter, adaptive-filter, kalman-filter, lms-algorithm, rudolf-kalman]
cite_urls:
  - https://en.wikipedia.org/wiki/Norbert_Wiener
  - https://en.wikipedia.org/wiki/Wiener_filter
---

**Norbert Wiener** (1894–1964) was an American mathematician who founded the field of
**cybernetics** and derived the **Wiener filter**, the linear filter that minimises the
mean-square error between a desired signal and an estimate formed from a noisy
observation.[^wiki][^filt] His work turned the intuitive idea of "cleaning up" a signal
into a precise optimisation problem, and the optimal-estimation viewpoint he introduced
underpins much of modern communications and control theory.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A signal plus noise entering a Wiener filter block, which outputs an estimate that minimises mean-square error against the desired signal." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nwar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="20" y="60" font-size="11" fill="currentColor">signal</text>
  <text x="20" y="95" font-size="11" fill="currentColor">+ noise</text>
  <line x1="70" y1="70" x2="150" y2="70" stroke="currentColor" marker-end="url(#nwar)"/>
  <rect x="150" y="45" width="130" height="50" fill="none" stroke="currentColor"/>
  <text x="215" y="74" text-anchor="middle" font-size="11" fill="currentColor">Wiener filter</text>
  <line x1="280" y1="70" x2="360" y2="70" stroke="currentColor" marker-end="url(#nwar)"/>
  <text x="405" y="74" text-anchor="middle" font-size="11" fill="currentColor">estimate</text>
  <text x="215" y="120" text-anchor="middle" font-size="9" fill="currentColor">minimises E[(desired − estimate)²]</text>
</svg>
<figcaption>The Wiener filter is the linear filter that produces the minimum-mean-square-error estimate of a desired signal buried in noise.</figcaption>
</figure>

## Life and work

Wiener was a child prodigy who earned his PhD from Harvard at eighteen and spent most
of his career at MIT. He made deep contributions to pure mathematics — Brownian motion,
harmonic analysis, and the Wiener–Khinchin theorem relating a signal's autocorrelation
to its power spectrum — before turning, during and after World War II, to the practical
problem of predicting and smoothing noisy time series.[^wiki] A wartime project on
automatic fire-control for anti-aircraft guns, where a gun must be aimed at where an
evading aircraft *will* be, crystallised his thinking about prediction, feedback, and
noise.

## Contribution

Two strands of Wiener's work are central to signal processing.

The first is the **Wiener filter**. Given a desired signal corrupted by additive noise,
Wiener asked which linear, time-invariant filter produces the estimate with the smallest
average squared error. The answer, expressed through the **Wiener–Hopf equation**, is a
filter whose frequency response weights each band by the ratio of signal power to total
(signal-plus-noise) power at that frequency: bands where the signal dominates pass
almost untouched, bands buried in noise are suppressed.[^filt] This minimum-mean-square-error
(MMSE) criterion is the optimality standard against which later estimators are measured,
and the [matched filter](/reference/matched-filter/) can be viewed as its special case
when the goal is detection rather than waveform recovery.

The second is **cybernetics**, the study of control and communication in animals and
machines, which he named in his 1948 book of that title. Cybernetics framed feedback,
information, and regulation as one subject spanning engineering and biology, and helped
put the concept of information on an equal footing with energy and matter in the sciences.

## Legacy

The Wiener filter is the historical root of statistical, or "optimal", filtering. Its
central limitation — it assumes the signal and noise are stationary, so a single
fixed filter serves for all time — was lifted a decade later by
[Rudolf Kalman](/reference/rudolf-kalman/), whose [Kalman filter](/reference/kalman-filter/)
recasts the same MMSE goal in a recursive, state-space form that tracks changing
signals in real time. When the signal statistics are unknown or drift, an
[adaptive filter](/reference/adaptive-filter/) approximates the Wiener solution by
learning its coefficients from the data; the [LMS algorithm](/reference/lms-algorithm/)
is a stochastic-gradient method that descends toward the Wiener optimum sample by sample.
Every one of these traces its optimality criterion back to Wiener.

## Relevance to SDR

MMSE thinking is everywhere in a software radio. Channel equalisers that undo
multipath, noise-reduction stages, and interference cancellers are all, at heart,
attempts to build a Wiener filter for a channel whose statistics must be estimated on
the fly — which is exactly what adaptive and Kalman-based methods do. The
Wiener–Khinchin relationship also justifies the everyday practice of estimating a
signal's power spectrum by Fourier-transforming its autocorrelation. GopherTrunk does
not implement a general Wiener filter by name, but its symbol-timing and carrier-recovery
loops and its equalisation stages all rest on the estimation-in-noise foundation Wiener
laid.

## Sources

[^wiki]: [Norbert Wiener](https://en.wikipedia.org/wiki/Norbert_Wiener) — Wikipedia, for biography, cybernetics, and the Wiener–Khinchin theorem.
[^filt]: [Wiener filter](https://en.wikipedia.org/wiki/Wiener_filter) — Wikipedia, for the MMSE criterion and the Wiener–Hopf solution.
