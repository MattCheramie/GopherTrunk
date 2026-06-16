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
  - { title: "How signals travel", url: /learn/propagation/ }
external:
  - { title: "Multipath propagation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multipath_propagation }
---

**Multipath propagation** occurs when a signal reaches the receiver by several paths at
once — directly and via reflections off buildings, terrain, and vehicles. The copies
arrive slightly out of step and add or cancel.

## How it works

The interference makes signal strength **fade** and smears digital
[symbols](/reference/symbol-rate/) into one another (intersymbol interference),
degrading decoding. Moving the antenna a short distance can change multipath markedly.

## Relevance to SDR

Multipath is a common reason a strong signal still won't decode; an
[equalizer](/reference/cma-equalizer/) and good [clock recovery](/reference/clock-recovery/)
help combat it.
