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
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
external:
  - { title: "Nyquist–Shannon sampling theorem (Wikipedia)", url: https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem }
---

The **Nyquist–Shannon sampling theorem** states that to represent a signal without loss
you must sample at least **twice its [bandwidth](/reference/bandwidth/)**. For IQ
sampling, the practical takeaway is that usable bandwidth ≈
[sample rate](/reference/sample-rate/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A fast wave sampled too slowly, with the sample dots tracing a slower false wave." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q12 -30 24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <g fill="currentColor"><circle cx="20" cy="60" r="3"/><circle cx="92" cy="48" r="3"/><circle cx="164" cy="72" r="3"/><circle cx="236" cy="48" r="3"/><circle cx="308" cy="72" r="3"/><circle cx="380" cy="48" r="3"/><circle cx="430" cy="66" r="3"/></g>
  <path d="M20 60 C 92 48, 92 48, 164 72 S 308 72, 380 48" fill="none" stroke="currentColor" stroke-width="1.8" stroke-dasharray="5 3"/>
  <text x="20" y="112" font-size="9" fill="currentColor">too few samples → a false (aliased) low-frequency wave</text>
</svg>
<figcaption>Nyquist: sample at least twice the bandwidth, or fast signals fold back as false low-frequency aliases.</figcaption>
</figure>

## How it works

Sample too slowly for the bandwidth and information does not merely vanish — it corrupts
the data through [aliasing](/reference/aliasing/), folding out-of-band energy to false
frequencies.

## Relevance to SDR

It is named for [Harry Nyquist](/reference/harry-nyquist/) and
[Claude Shannon](/reference/claude-shannon/), and it sets the floor on the sample rate
needed to capture a given channel.
