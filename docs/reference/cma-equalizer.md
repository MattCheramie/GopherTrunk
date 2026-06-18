---
slug: cma-equalizer
title: CMA equalizer
entry_type: algorithm
category: sdr-dsp
description: A constant-modulus-algorithm (CMA) equalizer is a blind adaptive filter that counteracts multipath distortion without a training sequence, improving digital decoding.
keywords: CMA, constant modulus algorithm, blind equalizer, multipath, adaptive filter, intersymbol interference
aka: [CMA equalizer, constant-modulus algorithm]
autolink: true
infobox:
  - { label: Type, value: Blind adaptive equalizer }
  - { label: Counteracts, value: Multipath / intersymbol interference }
  - { label: Blind, value: No training sequence required }
see_also: [multipath-propagation, demodulation, clock-recovery, constellation-diagram]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
external:
  - { title: "Constant modulus algorithm (Wikipedia)", url: https://en.wikipedia.org/wiki/Constant_modulus_algorithm }
---

A **CMA equalizer** uses the **constant-modulus algorithm** — a blind adaptive
[filter](/reference/digital-filter/) — to counteract
[multipath](/reference/multipath-propagation/) distortion without needing a known training
sequence.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A smeared constellation on the left and a tightened constellation on the right after equalisation." xmlns="http://www.w3.org/2000/svg">
  <g><line x1="20" y1="95" x2="170" y2="95" stroke="currentColor" stroke-opacity="0.3"/><line x1="95" y1="30" x2="95" y2="160" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor" fill-opacity="0.7"><circle cx="58" cy="60" r="2.5"/><circle cx="50" cy="68" r="2.5"/><circle cx="66" cy="54" r="2.5"/><circle cx="132" cy="62" r="2.5"/><circle cx="124" cy="70" r="2.5"/><circle cx="58" cy="130" r="2.5"/><circle cx="66" cy="122" r="2.5"/><circle cx="132" cy="130" r="2.5"/></g>
    <text x="95" y="180" text-anchor="middle" font-size="9" fill="currentColor">multipath-smeared</text></g>
  <line x1="195" y1="95" x2="235" y2="95" stroke="currentColor" marker-end="url(#eqar)"/>
  <g><line x1="280" y1="95" x2="440" y2="95" stroke="currentColor" stroke-opacity="0.3"/><line x1="360" y1="30" x2="360" y2="160" stroke="currentColor" stroke-opacity="0.3"/>
    <g fill="currentColor"><circle cx="325" cy="62" r="2.5"/><circle cx="395" cy="62" r="2.5"/><circle cx="325" cy="128" r="2.5"/><circle cx="395" cy="128" r="2.5"/></g>
    <text x="360" y="180" text-anchor="middle" font-size="9" fill="currentColor">equalised</text></g>
  <defs><marker id="eqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CMA equaliser blindly undoes channel distortion (multipath), pulling smeared symbols back to tight clusters.</figcaption>
</figure>

## How it works

It adjusts its taps to drive the output toward a constant modulus (the property of
constant-envelope modulations), undoing intersymbol interference and tightening the
[constellation](/reference/constellation-diagram/).

## Relevance to SDR

A CMA equalizer can rescue decoding in reflective urban environments where multipath
otherwise smears the symbols.
