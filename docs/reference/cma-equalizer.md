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
  - { title: "Tuning for a clean lock", url: /learn/tuning-with-scopes/ }
external:
  - { title: "Constant modulus algorithm (Wikipedia)", url: https://en.wikipedia.org/wiki/Constant_modulus_algorithm }
---

A **CMA equalizer** uses the **constant-modulus algorithm** — a blind adaptive
[filter](/reference/digital-filter/) — to counteract
[multipath](/reference/multipath-propagation/) distortion without needing a known training
sequence.

## How it works

It adjusts its taps to drive the output toward a constant modulus (the property of
constant-envelope modulations), undoing intersymbol interference and tightening the
[constellation](/reference/constellation-diagram/).

## Relevance to SDR

A CMA equalizer can rescue decoding in reflective urban environments where multipath
otherwise smears the symbols.
