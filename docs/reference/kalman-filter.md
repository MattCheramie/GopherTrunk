---
slug: kalman-filter
title: Kalman filter
entry_type: algorithm
category: estimation-array
description: The Kalman filter is a recursive predict/update estimator that fuses a motion model with noisy measurements to optimally track state in linear-Gaussian systems.
keywords: Kalman filter, recursive estimator, state estimation, predict update, extended Kalman filter, EKF, unscented Kalman filter, UKF, tracking, sensor fusion, Rudolf Kalman
aka: [Kalman filter, KF, linear quadratic estimator, LQE, EKF, UKF]
autolink: true
infobox:
  - { label: Type, value: Recursive state estimator }
  - { label: Optimal for, value: Linear-Gaussian systems }
  - { label: Variants, value: EKF, UKF, particle filter }
see_also: [phase-locked-loop, automatic-frequency-control, multilateration, gps-receiver, adaptive-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Kalman_filter
  - https://www.cs.unc.edu/~welch/media/pdf/kalman_intro.pdf
---

A **Kalman filter** is a recursive estimator that maintains a running belief about
a system's hidden state — position, velocity, frequency, phase — by alternating a
**predict** step (project the state forward with a motion model) and an **update**
step (correct that prediction with a new, noisy measurement).[^wiki] For a linear
system driven by Gaussian noise it is the provably optimal minimum-mean-square-error
estimator, and it needs to store only the current state estimate and its covariance
rather than the whole measurement history.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A two-box loop: a predict step projects the state and grows its uncertainty, an update step blends the prediction with a new measurement weighted by the Kalman gain, and the corrected estimate feeds back into predict." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="55" y="45" width="95" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="102" y="61">predict</text><text x="102" y="74">x̂⁻, P⁻</text>
    <rect x="255" y="45" width="110" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="310" y="61">update</text><text x="310" y="74">gain K · innov.</text>
    <line x1="150" y1="65" x2="254" y2="65" stroke="currentColor" stroke-width="1.1" marker-end="url(#kfar)"/>
    <line x1="200" y1="20" x2="200" y2="64" stroke="currentColor" stroke-width="1.1" marker-end="url(#kfar)"/><text x="200" y="14">measurement z</text>
    <path d="M310 85 V 120 H 102 V 86" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#kfar)"/><text x="205" y="134">corrected estimate x̂</text>
  </g>
  <defs><marker id="kfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The Kalman filter loops predict → update: each measurement nudges the projected state by an amount set by the Kalman gain, then the corrected estimate is projected forward again.</figcaption>
</figure>

## How it works

The filter carries two things: a state vector `x̂` (the best estimate) and a
covariance matrix `P` (how uncertain that estimate is). Each cycle:

- **Predict.** Apply the state-transition model to advance `x̂` one step, and grow
  `P` by the process-noise covariance `Q` — prediction always increases uncertainty.
- **Update.** Compare the actual measurement `z` to what the model predicts; the
  difference is the *innovation*. Compute the **Kalman gain** `K` from the ratio of
  predicted uncertainty to measurement noise `R`, then correct the state by
  `K × innovation` and shrink `P` accordingly.

The gain is the whole story: when the sensor is trusted (small `R`), `K` is large and
the filter follows the measurement; when the sensor is noisy, `K` is small and the
filter leans on its model. It self-tunes this balance every step from the covariances.

## Variants

The basic filter assumes linear dynamics and linear measurements. Real problems —
range from a delay, angle from a bearing, frequency from a tone — are usually
nonlinear, so practical work uses:

- **Extended Kalman filter (EKF).** Linearises the model about the current estimate
  with a Jacobian each step. Cheap and ubiquitous, but can diverge if the linearisation
  is poor.
- **Unscented Kalman filter (UKF).** Propagates a small set of deterministic "sigma
  points" through the true nonlinear model, capturing the mean and covariance to higher
  order without Jacobians — usually more robust than the EKF at similar cost.
- **Particle filter.** Drops the Gaussian assumption entirely and represents the belief
  with weighted samples; handles multimodal, highly nonlinear problems at much higher
  compute cost.

## In practice

Kalman-style recursion appears at both ends of a radio. Inside a demodulator, a tracking
loop like a [phase-locked loop](/reference/phase-locked-loop/) or an
[automatic frequency control](/reference/automatic-frequency-control/) loop is a
degenerate scalar Kalman filter: it predicts the carrier phase/frequency and corrects it
from a phase-error measurement, with the loop bandwidth playing the role of the gain.
Higher up, target trackers fuse successive position or Doppler fixes to smooth a track and
reject outliers, and receiver clock/frequency disciplining fuses GNSS with a local
oscillator the same way.

## Relevance to SDR

Kalman filtering is everywhere state must be tracked through noise: INS/GNSS fusion in a
[GPS receiver](/reference/gps-receiver/), radar and
[multilateration](/reference/multilateration/) target tracking, attitude estimation in
drones, and carrier/timing tracking in modems. GopherTrunk does not run an explicit Kalman
filter in its decode chain — its carrier and symbol-timing recovery use dedicated loop
filters (Costas/PLL, Gardner) rather than a full covariance-propagating estimator — but
those loops are close cousins, and any downstream geolocation or track-fusion layer built
on GT's detections would naturally be Kalman-based. It is worth understanding as the
general theory that the specialised tracking loops in an SDR are special cases of.

## Sources

[^wiki]: [Kalman filter](https://en.wikipedia.org/wiki/Kalman_filter) — Wikipedia, on the recursive predict/update estimator and its EKF/UKF variants.
