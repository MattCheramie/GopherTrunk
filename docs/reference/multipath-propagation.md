---
slug: multipath-propagation
title: Multipath propagation
entry_type: term
category: propagation
description: Multipath propagation is the arrival of a signal via multiple reflected paths, causing fading and intersymbol interference that can degrade digital decoding.
keywords: multipath, fading, reflections, intersymbol interference, equalizer, delay spread, Rayleigh fading, Doppler
aka: [multipath, multipath propagation]
autolink: true
infobox:
  - { label: Type, value: Propagation impairment }
  - { label: Causes, value: Reflections off terrain/buildings }
  - { label: Effects, value: Fading, symbol smearing }
see_also: [radio-propagation, rayleigh-fading, doppler-shift, mimo, cma-equalizer, clock-recovery, radio-horizon]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multipath_propagation
  - https://en.wikipedia.org/wiki/Delay_spread
---

**Multipath propagation** occurs when a signal reaches the receiver by several paths at
once — directly and via reflections off buildings, terrain, and vehicles.[^wiki] The copies
arrive slightly out of step and add or cancel, so the received amplitude and phase are the
vector sum of many delayed echoes rather than a single clean wave.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A transmitter and receiver with a direct path plus a reflected path bouncing off a building, arriving slightly delayed." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="110" x2="40" y2="70" stroke="currentColor" stroke-width="2"/><text x="40" y="125" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <line x1="420" y1="110" x2="420" y2="70" stroke="currentColor" stroke-width="2"/><text x="420" y="125" text-anchor="middle" font-size="9" fill="currentColor">RX</text>
  <line x1="48" y1="75" x2="412" y2="75" stroke="currentColor" stroke-width="1.4" marker-end="url(#mpar)"/><text x="200" y="68" font-size="9" fill="currentColor">direct</text>
  <rect x="225" y="20" width="40" height="24" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
  <path d="M48 70 L245 44 L412 70" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#mpar)"/><text x="150" y="40" font-size="9" fill="currentColor">reflected (delayed)</text>
  <defs><marker id="mpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Multipath: copies arrive by direct and reflected paths and interfere, causing fading and decode errors.</figcaption>
</figure>

## How it works

Each reflected path travels a different distance, so its copy arrives with a different delay
and phase. Where copies arrive in phase they reinforce; where they arrive out of phase they
cancel, and the result is **fading** — deep, position-dependent dips in signal strength.
Because the phase relationships depend on wavelength, the fading pattern is finely grained:
moving a receive antenna a fraction of a wavelength can turn a dead null into a strong
signal, which is why nudging the antenna a few centimetres sometimes fixes reception.

Two numbers characterize a multipath channel:

- **Delay spread** — the time between the earliest and latest significant echo. When the
  delay spread is comparable to a symbol period, one symbol's echoes smear into the next,
  producing **intersymbol interference** that a decoder sees as errors.[^ds]
- **Fading rate** — how fast the fades come and go as the receiver, transmitter, or
  reflectors move. Motion also imposes a [Doppler shift](/reference/doppler-shift/) on each
  path, and the spread of those shifts sets how quickly the channel changes.

When many comparable echoes arrive with no dominant direct ray, the summed amplitude follows
**[Rayleigh fading](/reference/rayleigh-fading/)** statistics; when a strong line-of-sight
path is present, the milder Rician distribution applies instead.

## In practice

Multipath is worst in dense urban and indoor settings and mildest with a high, clear
line-of-sight path. Digital systems fight it in several ways: **equalizers** such as a
[CMA equalizer](/reference/cma-equalizer/) estimate and undo the channel's smearing; robust
**[clock recovery](/reference/clock-recovery/)** keeps symbol timing locked through fades;
and multi-antenna techniques exploit it rather than merely tolerating it. [MIMO](/reference/mimo/)
famously turns rich multipath into a benefit, using the independent paths to carry parallel
data streams, while simpler [antenna diversity](/reference/antenna-diversity/) just picks
whichever of two antennas is not in a fade.

## Relevance to SDR

Multipath is a common reason a strong signal still won't decode; the meter reads plenty of
power, yet symbols arrive smeared and the [BER](/reference/bit-error-rate/) stays high. An
[equalizer](/reference/cma-equalizer/) and good [clock recovery](/reference/clock-recovery/)
help combat it, and moving or re-siting the antenna can change the multipath picture
dramatically. GopherTrunk's [C4FM](/reference/c4fm/)/[CQPSK](/reference/cqpsk/) demodulators
include timing recovery and equalization stages precisely because land-mobile signals so
often arrive through this kind of reflective channel; when decoding fails on a strong
carrier, multipath is one of the first suspects.

## Sources

[^wiki]: [Multipath propagation](https://en.wikipedia.org/wiki/Multipath_propagation) — Wikipedia, on multiple-path arrival, fading, and intersymbol interference.
[^ds]: [Delay spread](https://en.wikipedia.org/wiki/Delay_spread) — Wikipedia, on how echo delay relative to the symbol period produces intersymbol interference.
