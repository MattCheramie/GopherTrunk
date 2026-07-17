---
slug: touchscreen
title: Touchscreen
entry_type: hardware
category: hw-mobile
description: A touchscreen is a display that also serves as an input device, sensing the position of fingers or a stylus on its surface — the primary interface for phones, tablets, and many embedded devices.
keywords: touchscreen, capacitive, resistive, multitouch, touch panel, display input, stylus, touch sensor, projected capacitive
infobox:
  - { label: Type, value: Display + input device }
  - { label: Common tech, value: Capacitive (projected) }
  - { label: Older tech, value: Resistive }
  - { label: Feature, value: Multitouch }
see_also: [mobile-operating-system, smartphone, tablet, e-reader, foldable-phone, smartwatch]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Touchscreen
---

A **touchscreen** is a display that doubles as an input device, sensing where a finger or stylus touches its surface — the defining interface of modern phones and tablets.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A projected-capacitive touchscreen sensing a fingertip. A grid of transparent horizontal and vertical electrodes is laid over the display. A fingertip touching the glass draws off a tiny amount of charge at the electrodes it covers, changing the capacitance at that crossing. The controller reads which row and column changed to find the touch coordinates." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="0.9" font-family="ui-sans-serif, sans-serif">
    <g stroke-opacity="0.55">
      <line x1="60" y1="44" x2="360" y2="44"/>
      <line x1="60" y1="68" x2="360" y2="68"/>
      <line x1="60" y1="92" x2="360" y2="92"/>
      <line x1="60" y1="116" x2="360" y2="116"/>
      <line x1="96" y1="30" x2="96" y2="128"/>
      <line x1="156" y1="30" x2="156" y2="128"/>
      <line x1="216" y1="30" x2="216" y2="128"/>
      <line x1="276" y1="30" x2="276" y2="128"/>
      <line x1="336" y1="30" x2="336" y2="128"/>
    </g>
    <rect x="46" y="132" width="328" height="12" rx="2" stroke-width="1.1"/>
    <circle cx="216" cy="68" r="16" stroke-width="1.4"/>
    <path d="M216 40 v-16 M216 24 a10 14 0 0 1 10 14" stroke-width="1.4"/>
    <circle cx="216" cy="68" r="3" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="410" y="47">rows</text>
    <text x="216" y="20">fingertip</text>
    <text x="210" y="142">LCD / OLED display beneath</text>
    <text x="216" y="156" font-size="8">capacitance drops where the finger crosses the grid</text>
  </g>
</svg>
<figcaption>A projected-capacitive panel is a grid of transparent electrodes over the display; a fingertip perturbs the capacitance at the row and column it covers, and the controller reads that crossing as a touch coordinate — enabling fast, accurate multitouch.</figcaption>
</figure>

## Overview

Most modern touchscreens are *projected capacitive*: a grid of transparent electrodes detects the tiny change in capacitance a fingertip causes, enabling fast, accurate *multitouch* (pinch, swipe, two-finger gestures). Because it senses charge rather than pressure, the glass can be smooth and rigid and the surface can stay sealed.

Older *resistive* panels sense pressure where two conductive layers physically meet — cheaper and usable with gloves or any stylus, but single-touch and less responsive. In both cases the sensing layer is laminated over an LCD or OLED display, and a dedicated controller reports touch coordinates to the [mobile OS](/reference/mobile-operating-system/) many times a second.

## Capacitive vs resistive

The two families trade responsiveness against cost and glove-friendliness:

| Property | Capacitive | Resistive |
|----------|------------|-----------|
| Senses | Charge (finger) | Pressure |
| Multitouch | Yes | No (typically) |
| Clarity | High | Lower |
| Works with gloves | No (usually) | Yes |
| Cost | Higher | Lower |
| Typical use | Phones, tablets | Kiosks, industrial |

Capacitive won the phone era because multitouch gestures need it; resistive survives where cost or glove use matters more than finesse.

## Where it fits

The touchscreen is what let phones drop physical keyboards and become all-screen devices; it is equally central to [tablets](/reference/tablet/), [e-readers](/reference/e-reader/), [smartwatches](/reference/smartwatch/), and the inner display of a [foldable phone](/reference/foldable-phone/). For a field setup, a small touchscreen on a [Raspberry Pi](/reference/raspberry-pi/) next to the antenna can give a GopherTrunk capture node a self-contained local display without a separate keyboard or mouse.

## Sources

[^wiki]: [Touchscreen](https://en.wikipedia.org/wiki/Touchscreen) — Wikipedia, on capacitive and resistive touchscreen technology.
