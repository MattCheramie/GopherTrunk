---
slug: rudolf-kalman
title: Rudolf Kalman
entry_type: person
category: people
description: "Rudolf Kalman (1930–2016) was a Hungarian-American engineer who devised the Kalman filter, the recursive optimal estimator used for tracking and navigation."
keywords: Rudolf Kalman, Kalman filter, recursive estimation, state-space, optimal filtering, control theory, navigation, tracking
aka: [Rudolf Kalman, Rudolf E. Kalman]
autolink: true
infobox:
  - { label: Lived, value: "1930–2016" }
  - { label: Field, value: "Control theory, engineering" }
  - { label: Known for, value: "The Kalman filter" }
see_also: [kalman-filter, norbert-wiener, adaptive-filter, automatic-frequency-control, phase-locked-loop]
cite_urls:
  - https://en.wikipedia.org/wiki/Rudolf_E._K%C3%A1lm%C3%A1n
  - https://en.wikipedia.org/wiki/Kalman_filter
---

**Rudolf Kalman** (1930–2016) was a Hungarian-American electrical engineer and
mathematician who devised the **[Kalman filter](/reference/kalman-filter/)**, a
recursive algorithm that produces the optimal estimate of a changing system's state
from a stream of noisy measurements.[^wiki][^filt] The filter is one of the most widely
applied results of twentieth-century engineering, steering spacecraft, aircraft, and
satellite-navigation receivers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A predict-then-update loop: a prediction step feeds an update step that ingests a new measurement and returns a corrected estimate, which feeds back to the next prediction." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="40" y="45" width="130" height="45" fill="none" stroke="currentColor"/>
  <text x="105" y="72" text-anchor="middle" font-size="11" fill="currentColor">predict</text>
  <rect x="290" y="45" width="130" height="45" fill="none" stroke="currentColor"/>
  <text x="355" y="72" text-anchor="middle" font-size="11" fill="currentColor">update</text>
  <line x1="170" y1="67" x2="290" y2="67" stroke="currentColor" marker-end="url(#rkar)"/>
  <line x1="355" y1="90" x2="355" y2="120" stroke="currentColor"/>
  <line x1="355" y1="120" x2="105" y2="120" stroke="currentColor"/>
  <line x1="105" y1="120" x2="105" y2="90" stroke="currentColor" marker-end="url(#rkar)"/>
  <line x1="355" y1="20" x2="355" y2="45" stroke="currentColor" marker-end="url(#rkar)"/>
  <text x="355" y="15" text-anchor="middle" font-size="10" fill="currentColor">measurement</text>
</svg>
<figcaption>The Kalman filter alternates a prediction step with a measurement-driven update, carrying its estimate forward recursively.</figcaption>
</figure>

## Life and work

Kalman was born in Budapest, emigrated to the United States as a teenager, and earned
degrees from MIT and Columbia. He developed his filter around 1959–1960 while at the
Research Institute for Advanced Study in Baltimore, publishing the landmark paper
"A New Approach to Linear Filtering and Prediction Problems" in 1960.[^wiki] The idea
was not welcomed universally at first, but a visit to NASA's Ames Research Center led to
its adoption in the Apollo guidance computer, and the filter's success there secured its
reputation. Kalman later held professorships at Stanford, the University of Florida, and
ETH Zürich, and received the U.S. National Medal of Science.

## Contribution

Kalman's filter solves the same minimum-mean-square-error problem that
[Norbert Wiener](/reference/norbert-wiener/) had posed, but reframes it in a way that is
far more practical. Instead of a fixed frequency-domain filter for a stationary signal,
Kalman describes the system with a **state-space model**: a small set of state variables
(position, velocity, phase, and so on) that evolve in time, plus a measurement equation
linking those states to what the sensors observe. The filter then runs a two-step loop.
The **predict** step propagates the current estimate and its uncertainty forward using
the model; the **update** step blends in each new measurement, weighting model and
measurement by their relative uncertainties through a quantity called the **Kalman gain**.

Because the whole computation is recursive — each estimate depends only on the previous
estimate and the latest measurement, not on the entire history — it runs in bounded
memory and constant time per step, making it ideal for real-time embedded use. It also
naturally handles non-stationary signals, the case Wiener's original formulation could
not.

## Legacy

The Kalman filter is arguably the most-used estimation algorithm ever devised. Nonlinear
variants — the extended Kalman filter and the unscented Kalman filter — extend it to
systems that do not fit the linear-Gaussian assumptions, and it sits at the core of
sensor fusion in robotics, avionics, and consumer electronics. Its lineage runs directly
back through Wiener's optimal-filtering work and forward into the [adaptive
filter](/reference/adaptive-filter/) tradition, where estimator gains are learned from
data.

## Relevance to SDR

Software radios lean on Kalman-style estimation whenever a slowly varying quantity must
be tracked through noise. GPS and GNSS receivers use Kalman filters to fuse pseudorange
and Doppler measurements into a position-velocity-time solution. Carrier- and
timing-recovery loops — a [phase-locked loop](/reference/phase-locked-loop/) or an
[automatic frequency control](/reference/automatic-frequency-control/) stage — are, in
effect, simple recursive estimators of phase and frequency, and can be derived as reduced
Kalman filters. GopherTrunk does not run a general Kalman filter in its decode chain, but
its tracking loops embody the same predict-then-correct principle Kalman formalised.

## Sources

[^wiki]: [Rudolf E. Kálmán](https://en.wikipedia.org/wiki/Rudolf_E._K%C3%A1lm%C3%A1n) — Wikipedia, for biography and the 1960 paper.
[^filt]: [Kalman filter](https://en.wikipedia.org/wiki/Kalman_filter) — Wikipedia, for the predict/update recursion and the Kalman gain.
