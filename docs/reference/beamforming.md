---
slug: beamforming
title: Beamforming
entry_type: algorithm
category: estimation-array
description: Beamforming combines the signals of multiple antenna elements with per-element phase and amplitude weights to steer gain toward a direction and place nulls on interferers.
keywords: beamforming, phased array, delay and sum, spatial filtering, MVDR, Capon beamformer, adaptive beamforming, massive MIMO, 5G, direction finding, antenna array
aka: [beamforming, spatial filtering, phased-array combining]
autolink: true
infobox:
  - { label: Type, value: Spatial array combining }
  - { label: Steers, value: Gain and nulls by direction }
  - { label: Families, value: Delay-and-sum, MVDR/Capon }
see_also: [antenna, antenna-gain, music-algorithm, esprit-algorithm, multipath-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Beamforming
  - https://en.wikipedia.org/wiki/Phased_array
---

**Beamforming** is the technique of combining the outputs of several antenna elements —
each with its own phase and amplitude weight — so that signals from a chosen direction add
coherently while signals from other directions partly cancel, synthesising a steerable,
directional beam from an array of otherwise omnidirectional elements.[^wiki] The same
weights can place **nulls** on interferers, so beamforming is spatial filtering: it selects
by direction the way a bandpass filter selects by frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="An incoming plane wave hits four antenna elements at staggered times; per-element delays or phase weights align the copies so they sum in phase, forming a main beam steered toward the source." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="30" y1="20" x2="70" y2="60" stroke="currentColor" stroke-width="1" opacity="0.6"/><line x1="55" y1="15" x2="95" y2="55" stroke="currentColor" stroke-width="1" opacity="0.6"/><text x="45" y="14">wavefront</text>
    <circle cx="40" cy="70" r="3" fill="currentColor"/><circle cx="40" cy="92" r="3" fill="currentColor"/><circle cx="40" cy="114" r="3" fill="currentColor"/><circle cx="40" cy="136" r="3" fill="currentColor"/>
    <rect x="70" y="62" width="42" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="91" y="73">w₁·e^jφ</text>
    <rect x="70" y="84" width="42" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="91" y="95">w₂·e^jφ</text>
    <rect x="70" y="106" width="42" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="91" y="117">w₃·e^jφ</text>
    <rect x="70" y="128" width="42" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="91" y="139">w₄·e^jφ</text>
    <circle cx="175" cy="103" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="175" y="107">Σ</text>
    <line x1="112" y1="70" x2="164" y2="98" stroke="currentColor" stroke-width="1" marker-end="url(#bfar)"/>
    <line x1="112" y1="92" x2="164" y2="101" stroke="currentColor" stroke-width="1" marker-end="url(#bfar)"/>
    <line x1="112" y1="114" x2="164" y2="106" stroke="currentColor" stroke-width="1" marker-end="url(#bfar)"/>
    <line x1="112" y1="136" x2="164" y2="109" stroke="currentColor" stroke-width="1" marker-end="url(#bfar)"/>
    <g transform="translate(300,100)"><path d="M0 0 L70 -30 A76 76 0 0 0 70 30 Z" fill="currentColor" opacity="0.18"/><path d="M0 0 L70 -30 A76 76 0 0 0 70 30 Z" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="45" y="4">main beam</text><text x="10" y="-45">steered gain</text></g>
    <line x1="187" y1="103" x2="250" y2="103" stroke="currentColor" stroke-width="1.1" marker-end="url(#bfar)"/>
  </g>
  <defs><marker id="bfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Beamforming applies a phase/amplitude weight to each element and sums them; the weights that align the wavefront's staggered arrivals build a main beam toward the source.</figcaption>
</figure>

## How it works

A plane wave reaches the elements of an array at slightly different times, so each element
sees the same signal with a direction-dependent phase shift. Beamforming applies a complex
weight `wₙ` to each element and sums the results. Choose the weights to *undo* the phase
shifts for a target direction and those copies add in phase (array gain ≈ number of
elements), while other directions stay misaligned and partially cancel. Steering is just a
matter of changing the weights — electronic, instantaneous, and with no moving parts, which
is why it is called a **phased array**.

## Variants

- **Delay-and-sum (conventional).** Fixed weights that are just the conjugate steering
  vector for the desired angle. Simple and robust, but its nulls fall wherever the array
  geometry puts them, not on the interference.
- **Adaptive — MVDR / Capon.** Minimum-variance distortionless response computes the weights
  from the measured covariance matrix: hold unit gain toward the target while *minimising
  total output power*, which automatically steers deep nulls onto interferers. Sharper and
  far better at rejection than delay-and-sum, but sensitive to steering errors and needs a
  good covariance estimate.
- **Digital vs analog / hybrid.** Weights applied in RF phase shifters (analog), fully in
  DSP after per-element ADCs (digital, most flexible), or a mix (hybrid) as used in
  millimetre-wave 5G to limit the number of expensive RF chains.
- **Transmit beamforming.** The same principle in reverse concentrates radiated power toward
  a receiver — the basis of massive MIMO.

## Relevance to SDR

Beamforming underpins radar, sonar, 5G NR and Wi-Fi massive MIMO, satellite ground
stations, and the front end of direction-finding systems, where it complements subspace
estimators like [MUSIC](/reference/music-algorithm/) and
[ESPRIT](/reference/esprit-algorithm/) that share the same array data. It can also null a
strong [multipath](/reference/multipath-propagation/) reflection or a jammer. All of this
requires a coherent multi-element array with synchronised per-channel receivers — hardware
**GopherTrunk** does not have. GT is a single-front-end trunking receiver and does no
beamforming; the concept is covered here for the broader RF context and its close ties to
[antenna gain](/reference/antenna-gain/) and array processing.

## Sources

[^wiki]: [Beamforming](https://en.wikipedia.org/wiki/Beamforming) — Wikipedia, on phased combining of array elements to steer gain and nulls.
