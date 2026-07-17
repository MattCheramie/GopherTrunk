---
slug: smartwatch
title: Smartwatch
entry_type: hardware
category: hw-mobile
description: A smartwatch is a wrist-worn computer that pairs with or extends a smartphone, combining a small touchscreen, sensors, and radios to show notifications, track fitness, and run lightweight apps.
keywords: smartwatch, wearable, Apple Watch, Wear OS, fitness tracker, wrist computer, heart rate sensor, smart watch, SpO2, accelerometer
infobox:
  - { label: Type, value: Wrist-worn computer }
  - { label: Display, value: Small touchscreen }
  - { label: Sensors, value: Heart rate, GPS, accelerometer }
  - { label: Examples, value: Apple Watch, Wear OS }
see_also: [wearable-computer, smartphone, battery-technology, touchscreen, mobile-operating-system, gps-receiver]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Smartwatch
---

A **smartwatch** is a wrist-worn computer that pairs with or extends a [smartphone](/reference/smartphone/), packing a small [touchscreen](/reference/touchscreen/), sensors, and radios into a watch-sized case.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A smartwatch and its sensor array. On the left, the watch body with a small touchscreen and a strap. On the right, callouts list what the watch senses and connects with: a touchscreen display and low-power SoC on the front, and on the back against the skin an optical heart-rate sensor, an accelerometer and gyroscope, a GPS receiver, and Bluetooth and Wi-Fi radios, all fed by a tiny battery." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <path d="M108 30 h44 l6 18 v78 l-6 18 h-44 l-6 -18 v-78 z"/>
    <rect x="104" y="52" width="52" height="70" rx="6" fill="currentColor" fill-opacity="0.05"/>
    <circle cx="130" cy="145" r="7"/>
    <circle cx="126" cy="141" r="1.6" fill="currentColor"/>
    <circle cx="134" cy="141" r="1.6" fill="currentColor"/>
    <circle cx="130" cy="149" r="1.6" fill="currentColor"/>
    <g stroke-width="0.8">
      <line x1="158" y1="60" x2="250" y2="46"/>
      <line x1="158" y1="80" x2="250" y2="72"/>
      <line x1="158" y1="100" x2="250" y2="98"/>
      <line x1="137" y1="145" x2="250" y2="124"/>
      <line x1="158" y1="115" x2="250" y2="150"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="256" y="49">touchscreen + low-power SoC</text>
    <text x="256" y="75">accelerometer / gyroscope</text>
    <text x="256" y="101">GPS receiver</text>
    <text x="256" y="127">optical heart-rate sensor</text>
    <text x="256" y="153">Bluetooth / Wi-Fi radios</text>
  </g>
</svg>
<figcaption>A smartwatch crams a display, a low-power SoC, motion and heart-rate sensors, GPS, and short-range radios into a watch case — the whole array kept alive by a battery a fraction the size of a phone's.</figcaption>
</figure>

## Overview

A smartwatch is a tightly constrained [wearable computer](/reference/wearable-computer/): a low-power [SoC](/reference/system-on-a-chip/), a small display, Bluetooth and often Wi-Fi or cellular, and sensors such as an optical heart-rate monitor, accelerometer, gyroscope, and [GPS receiver](/reference/gps-receiver/). It runs a purpose-built [mobile OS](/reference/mobile-operating-system/) — for example watchOS or Wear OS — surfacing notifications, fitness tracking, payments, and small apps.

Everything is shaped by a tiny [battery](/reference/battery-technology/), forcing aggressive power management and, on many models, a daily charge. Much of the heavy work is offloaded to a paired phone over Bluetooth; the watch handles glanceable interaction and continuous sensing against the skin, which the phone in a pocket cannot do.

## Smartwatch vs band vs phone

The wrist tier trades capability for size at each step down from the phone:

| Trait | Smartwatch | Fitness band | Smartphone |
|-------|-----------|--------------|------------|
| Display | Full touchscreen | Small or none | Large |
| Apps | Yes | Fixed functions | Full stores |
| Sensors | Rich (HR, GPS…) | Basic (HR, steps) | Rich |
| Battery | ~1 day | Days–week | ~1 day |
| Standalone | Some (cellular) | No | Yes |

A band strips the smartwatch down to sensing and battery life; the phone is the full computer both lean on.

## Where it fits

The smartwatch sits at the smallest, most personal end of the mobile spectrum: not a replacement for a phone but a glanceable extension of it, and a hub for health sensors worn against the skin. Its sealed, battery-limited design makes it purely a consumer endpoint — useful as a remote notifier for an alert, not as a place to run real compute like an SDR pipeline.

## Sources

[^wiki]: [Smartwatch](https://en.wikipedia.org/wiki/Smartwatch) — Wikipedia, on smartwatch hardware and platforms.
