---
slug: root-raised-cosine-filter
title: Root-raised-cosine filter
entry_type: algorithm
category: sdr-dsp
description: A root-raised-cosine (RRC) filter is a pulse-shaping filter used at both transmitter and receiver to limit bandwidth while minimising intersymbol interference.
keywords: root raised cosine, RRC, pulse shaping, intersymbol interference, matched filter, roll-off
aka: [root-raised-cosine filter, RRC filter]
autolink: true
infobox:
  - { label: Type, value: Pulse-shaping filter }
  - { label: Goal, value: Limit bandwidth, minimise ISI }
  - { label: Used, value: TX and RX (matched pair) }
see_also: [matched-filter, digital-filter, symbol-rate, eye-diagram]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Root-raised-cosine filter (Wikipedia)", url: https://en.wikipedia.org/wiki/Root-raised-cosine_filter }
---

A **root-raised-cosine** (**RRC**) filter is a pulse-shaping
[filter](/reference/digital-filter/) applied at both transmitter and receiver. Split
across the link, the two halves combine into a raised-cosine response that limits
bandwidth while minimising **intersymbol interference**.

## How it works

The roll-off factor trades [bandwidth](/reference/bandwidth/) against pulse compactness.
The receiver's RRC also acts as a [matched filter](/reference/matched-filter/),
maximising SNR at the sampling instant — visible as a clean
[eye diagram](/reference/eye-diagram/).

## Relevance to SDR

Applying the correct RRC is part of demodulating digital signals that use it, sharpening
[symbol](/reference/symbol-rate/) decisions.
