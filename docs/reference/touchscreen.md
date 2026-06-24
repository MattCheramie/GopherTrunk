---
slug: touchscreen
title: Touchscreen
entry_type: hardware
category: hw-mobile
description: A touchscreen is a display that also serves as an input device, sensing the position of fingers or a stylus on its surface — the primary interface for phones, tablets, and many embedded devices.
keywords: touchscreen, capacitive, resistive, multitouch, touch panel, display input, stylus, touch sensor
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

## Overview

Most modern touchscreens are *projected capacitive*: a grid of transparent electrodes detects the tiny change in capacitance a fingertip causes, enabling fast, accurate *multitouch* (pinch, swipe, two-finger gestures). Older *resistive* panels sense pressure where two conductive layers meet — cheaper and usable with gloves or any stylus, but single-touch and less responsive. The panel is laminated over an LCD or OLED display, and a controller reports touch coordinates to the [mobile OS](/reference/mobile-operating-system/).

## Where it fits

The touchscreen is what let phones drop physical keyboards and become all-screen devices; it is equally central to [tablets](/reference/tablet/), [e-readers](/reference/e-reader/), [smartwatches](/reference/smartwatch/), and the inner display of a [foldable phone](/reference/foldable-phone/). For a field setup, a small touchscreen on a [Raspberry Pi](/reference/raspberry-pi/) next to the antenna can give a GopherTrunk capture node a self-contained local display without a separate keyboard or mouse.

## Sources

[^wiki]: [Touchscreen](https://en.wikipedia.org/wiki/Touchscreen) — Wikipedia, on capacitive and resistive touchscreen technology.
