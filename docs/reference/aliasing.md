---
slug: aliasing
title: Aliasing
entry_type: term
category: sdr-dsp
description: Aliasing is the appearance of out-of-band energy at a false frequency when a signal is sampled too slowly for its bandwidth; anti-alias filtering and adequate sample rate prevent it.
keywords: aliasing, fold-back, anti-alias filter, undersampling, false signal, Nyquist
aka: [aliasing]
autolink: true
infobox:
  - { label: Type, value: Sampling artefact }
  - { label: Cause, value: Sampling below Nyquist for the bandwidth }
  - { label: Prevented by, value: Anti-alias filter + adequate rate }
see_also: [nyquist-theorem, sample-rate, decimation, digital-filter]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Aliasing (Wikipedia)", url: https://en.wikipedia.org/wiki/Aliasing }
---

**Aliasing** is when energy outside the bandwidth a [sample rate](/reference/sample-rate/)
can represent gets **folded** back into the captured spectrum, appearing at a wrong
frequency — a phantom that looks like a real signal.

## How it works

It is the [Nyquist theorem](/reference/nyquist-theorem/) violated. SDR front-ends include
anti-alias filtering, and choosing an adequate sample rate keeps real signals safely
inside the usable window. It also constrains the order of
[filtering and decimation](/reference/decimation/).

## Relevance to SDR

Recognising an alias prevents chasing signals that are not really where they appear.
