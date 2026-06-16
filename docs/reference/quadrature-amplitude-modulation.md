---
slug: quadrature-amplitude-modulation
title: Quadrature amplitude modulation (QAM)
entry_type: technology
category: modulation
description: QAM combines amplitude and phase modulation to pack many bits per symbol; higher-order QAM carries more data but needs a higher SNR to decode.
keywords: QAM, quadrature amplitude modulation, 16-QAM, 64-QAM, constellation, bits per symbol
aka: [quadrature amplitude modulation, QAM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Phase and amplitude }
  - { label: Used by, value: Wi-Fi, cable, LTE, broadcast }
see_also: [phase-shift-keying, frequency-shift-keying, constellation-diagram, signal-to-noise-ratio, iq-data]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Quadrature amplitude modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation }
---

**Quadrature amplitude modulation** (**QAM**) varies **both** the
[phase](/reference/phase/) and [amplitude](/reference/amplitude/) of a carrier, packing
many states into the [IQ](/reference/iq-data/) plane — 16-QAM (4 bits/symbol), 64-QAM,
and higher.

## How it works

More bits per symbol means more data in the same [bandwidth](/reference/bandwidth/), but
the states sit closer together, so QAM needs a higher
[SNR](/reference/signal-to-noise-ratio/) to tell them apart.

## Relevance to SDR

QAM appears in Wi-Fi, cable, and LTE rather than scanner voice traffic, but the same
[constellation](/reference/constellation-diagram/) idea applies to reading its quality.
