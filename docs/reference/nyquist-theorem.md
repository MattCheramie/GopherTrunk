---
slug: nyquist-theorem
title: Nyquist–Shannon sampling theorem
entry_type: term
category: sdr-dsp
description: The Nyquist–Shannon sampling theorem states that a signal must be sampled at least twice its bandwidth to be represented without loss; under-sampling causes aliasing.
keywords: Nyquist theorem, sampling theorem, Nyquist rate, aliasing, bandwidth, Shannon
aka: [Nyquist theorem, sampling theorem]
autolink: true
infobox:
  - { label: Type, value: Sampling principle }
  - { label: States, value: Sample ≥ 2× the bandwidth }
  - { label: Named for, value: Harry Nyquist, Claude Shannon }
see_also: [sample-rate, aliasing, bandwidth, harry-nyquist, claude-shannon]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Nyquist–Shannon sampling theorem (Wikipedia)", url: https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem }
---

The **Nyquist–Shannon sampling theorem** states that to represent a signal without loss
you must sample at least **twice its [bandwidth](/reference/bandwidth/)**. For IQ
sampling, the practical takeaway is that usable bandwidth ≈
[sample rate](/reference/sample-rate/).

## How it works

Sample too slowly for the bandwidth and information does not merely vanish — it corrupts
the data through [aliasing](/reference/aliasing/), folding out-of-band energy to false
frequencies.

## Relevance to SDR

It is named for [Harry Nyquist](/reference/harry-nyquist/) and
[Claude Shannon](/reference/claude-shannon/), and it sets the floor on the sample rate
needed to capture a given channel.
