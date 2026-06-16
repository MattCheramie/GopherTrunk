---
slug: low-noise-amplifier
title: Low-noise amplifier (LNA)
entry_type: hardware
category: hardware
description: A low-noise amplifier boosts a weak antenna signal early in the receive chain with minimal added noise, setting much of a receiver's sensitivity.
keywords: LNA, low noise amplifier, noise figure, sensitivity, front end, preamp
aka: [low-noise amplifier, LNA]
autolink: true
infobox:
  - { label: Type, value: RF amplifier }
  - { label: Placed, value: Early in receive chain (near antenna) }
  - { label: Key spec, value: Noise figure }
see_also: [noise-floor, signal-to-noise-ratio, superheterodyne-receiver, bias-tee, antenna]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Low-noise amplifier (Wikipedia)", url: https://en.wikipedia.org/wiki/Low-noise_amplifier }
---

A **low-noise amplifier** (**LNA**) boosts a weak [antenna](/reference/antenna/) signal
**early** in the [receive chain](/reference/superheterodyne-receiver/), adding as little
noise as possible. Because later stages add their own noise, amplifying first preserves
[SNR](/reference/signal-to-noise-ratio/).

## How it works

An LNA's *noise figure* largely determines how weak a signal the whole receiver can
detect — its sensitivity. It is best mounted at the antenna, often powered through the
coax by a [bias tee](/reference/bias-tee/).

## Relevance to SDR

An antenna-mounted LNA can meaningfully improve reception of weak signals, especially
with lossy cable runs — but watch for overload from strong nearby transmitters.
