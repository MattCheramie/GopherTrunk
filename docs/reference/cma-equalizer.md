---
slug: cma-equalizer
title: CMA equalizer
entry_type: algorithm
category: equalization
description: A constant-modulus-algorithm (CMA) equalizer is a blind adaptive filter that removes multipath and intersymbol interference with no training sequence by driving the output to a constant modulus.
keywords: CMA, constant modulus algorithm, blind equalizer, Godard algorithm, multipath, adaptive filter, intersymbol interference, LMS, stochastic gradient, decision-feedback
aka: [CMA equalizer, constant-modulus algorithm, Godard algorithm]
autolink: true
infobox:
  - { label: Type, value: Blind adaptive equalizer }
  - { label: Counteracts, value: Multipath / intersymbol interference }
  - { label: Blind, value: No training sequence required }
see_also: [adaptive-filter, lms-algorithm, decision-feedback-equalizer, costas-loop, multipath-propagation, constellation-diagram]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
related_reading:
  - { title: "SDR Internals, Part 8: Equalization, diversity & the FFT", url: /blog/deep-dives/sdr-internals-08-equalization-diversity-fft/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Constant_modulus_algorithm
  - https://en.wikipedia.org/wiki/Blind_equalization
---

A **CMA equalizer** uses the **constant-modulus algorithm** — a *blind*
[adaptive filter](/reference/adaptive-filter/) — to counteract
[multipath](/reference/multipath-propagation/) distortion and intersymbol interference
**without a known training sequence**.[^wiki] It exploits a structural property of the
transmitted signal rather than a reference: for constant-envelope and many PSK
modulations, every symbol should land on a circle of fixed radius in the I/Q plane, so the
equaliser can adapt purely from how far the received samples stray off that circle.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A smeared constellation on the left and a tightened constellation on the right after equalisation, with a target circle of constant modulus shown." xmlns="http://www.w3.org/2000/svg">
  <g><line x1="20" y1="95" x2="170" y2="95" stroke="currentColor" stroke-opacity="0.3"/><line x1="95" y1="30" x2="95" y2="160" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="95" cy="95" r="42" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
    <g fill="currentColor" fill-opacity="0.7"><circle cx="58" cy="60" r="2.5"/><circle cx="50" cy="68" r="2.5"/><circle cx="66" cy="54" r="2.5"/><circle cx="132" cy="62" r="2.5"/><circle cx="124" cy="70" r="2.5"/><circle cx="58" cy="130" r="2.5"/><circle cx="66" cy="122" r="2.5"/><circle cx="132" cy="130" r="2.5"/></g>
    <text x="95" y="180" text-anchor="middle" font-size="9" fill="currentColor">multipath-smeared</text></g>
  <line x1="195" y1="95" x2="235" y2="95" stroke="currentColor" marker-end="url(#eqar)"/><text x="215" y="88" text-anchor="middle" font-size="8" fill="currentColor">CMA</text>
  <g><line x1="280" y1="95" x2="440" y2="95" stroke="currentColor" stroke-opacity="0.3"/><line x1="360" y1="30" x2="360" y2="160" stroke="currentColor" stroke-opacity="0.3"/>
    <circle cx="360" cy="95" r="42" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
    <g fill="currentColor"><circle cx="325" cy="62" r="2.5"/><circle cx="395" cy="62" r="2.5"/><circle cx="325" cy="128" r="2.5"/><circle cx="395" cy="128" r="2.5"/></g>
    <text x="360" y="180" text-anchor="middle" font-size="9" fill="currentColor">equalised (on the circle)</text></g>
  <defs><marker id="eqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CMA equaliser blindly undoes channel distortion by pulling every output sample toward a target circle of constant modulus, tightening smeared symbols into clean clusters.</figcaption>
</figure>

## How it works

The equaliser is an FIR filter whose taps are adjusted sample-by-sample to minimise the
**constant-modulus cost function**

`J = E[ (|y|² − R)² ]`,

the mean-squared deviation of the output magnitude `|y|` from a target radius `R` (`R` is a
statistic of the source constellation, e.g. `R = E[|s|⁴]/E[|s|²]`). Note the cost depends
only on the *magnitude* of the output, never on which symbol was sent — that is what makes
it **blind**. A [stochastic-gradient](/reference/lms-algorithm/) update, exactly the same
machinery as the [LMS algorithm](/reference/lms-algorithm/) but with a different error term,
walks the taps downhill:

- **Error term:** `e = y · (R − |y|²)`. When a sample sits too far out (`|y|² > R`) the
  correction pulls it in; too far in, and it pushes it out — driving the cloud onto the
  circle regardless of angle.
- **Tap update:** `w ← w + μ · e* · x`, with step size `μ` trading convergence speed
  against steady-state jitter (the misadjustment), just as in LMS.
- **Convergence is not guaranteed to the global optimum:** the CMA cost surface is
  non-convex, so a badly initialised loop can settle in a local minimum. In practice the
  centre tap is initialised to a single spike and `μ` kept small.

Because CMA only fixes the *magnitude*, it leaves the constellation at an arbitrary rotation
— it removes ISI but not carrier phase. A [Costas loop](/reference/costas-loop/) or other
phase tracker is run alongside to de-rotate the now-open eye.

## Variants

- **Godard / dispersion-order variants.** CMA is the `p = 2` case of Godard's more general
  family of blind constant-modulus criteria.
- **Multi-modulus (MMA).** For QAM, where symbols do *not* all share one radius, a
  multi-modulus algorithm generalises the target to several rings.
- **CMA + DFE.** CMA can pre-open the eye for a
  [decision-feedback equalizer](/reference/decision-feedback-equalizer/), which then uses
  its own decisions to cancel residual post-cursor ISI more aggressively than a linear
  filter can.
- **Trained fallback.** Where a preamble or midamble exists, an LMS/RLS trained equaliser
  converges faster and more reliably; CMA earns its keep precisely when no training symbols
  are available.

## Relevance to SDR

Blind equalisation matters in reflective urban and mobile channels — land-mobile trunking
(P25, DMR, NXDN), broadcast, and microwave links — where echoes smear symbols into
neighbours. Many of these waveforms are constant-envelope or near-constant-modulus, making
them natural CMA candidates. GopherTrunk targets narrowband C4FM/PSK signals whose symbol
rates keep intersymbol interference modest, so its decode chain leans on matched filtering
and timing/carrier recovery rather than a full adaptive equaliser; CMA is the technique you
would reach for if multipath in a given deployment became severe enough to close the eye.

## Sources

[^wiki]: [Constant modulus algorithm](https://en.wikipedia.org/wiki/Constant_modulus_algorithm) — Wikipedia, on the blind adaptive equalization technique, its cost function, and its stochastic-gradient (LMS-family) update.
