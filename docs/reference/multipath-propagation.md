---
slug: multipath-propagation
title: Multipath propagation
entry_type: term
category: antennas-propagation
description: Multipath propagation is the arrival of a signal via multiple reflected paths, causing fading and intersymbol interference that can degrade digital decoding.
keywords: multipath, fading, reflections, intersymbol interference, equalizer
aka: [multipath, multipath propagation]
autolink: true
infobox:
  - { label: Type, value: Propagation impairment }
  - { label: Causes, value: Reflections off terrain/buildings }
  - { label: Effects, value: Fading, symbol smearing }
see_also: [radio-propagation, cma-equalizer, clock-recovery, radio-horizon]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
external:
  - { title: "Multipath propagation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multipath_propagation }
---

**Multipath propagation** occurs when a signal reaches the receiver by several paths at
once — directly and via reflections off buildings, terrain, and vehicles. The copies
arrive slightly out of step and add or cancel.

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

The interference makes signal strength **fade** and smears digital
[symbols](/reference/symbol-rate/) into one another (intersymbol interference),
degrading decoding. Moving the antenna a short distance can change multipath markedly.

## Relevance to SDR

Multipath is a common reason a strong signal still won't decode; an
[equalizer](/reference/cma-equalizer/) and good [clock recovery](/reference/clock-recovery/)
help combat it.
