---
slug: wearable-computer
title: Wearable computer
entry_type: hardware
category: hw-mobile
description: A wearable computer is a small computing device worn on the body — on the wrist, head, ear, or clothing — combining sensors and radios to provide hands-free, always-available computing.
keywords: wearable computer, wearable, smartwatch, smart glasses, hearable, fitness band, body-worn computer, AR headset
aka: [Wearable]
infobox:
  - { label: Type, value: Body-worn computer }
  - { label: Worn on, value: Wrist, head, ear, clothing }
  - { label: Forms, value: Watches, glasses, earbuds, bands }
  - { label: Key constraint, value: Size & battery }
see_also: [smartwatch, smartphone, battery-technology, system-on-a-chip, near-field-communication, gps-receiver]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Wearable_computer
---

A **wearable computer** is a small computing device worn on the body — on the wrist, head, ear, or clothing — built to provide hands-free, always-available computing.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Four wearable form factors around a simple human figure. Smart glasses on the head, a smart earbud at the ear, a smartwatch on the wrist, and a fitness band or clip on the body. Each is labeled, showing that wearables are defined by where on the body they are worn." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <circle cx="150" cy="46" r="20"/>
    <path d="M150 66 v46 M150 78 l-30 18 M150 78 l30 18 M150 112 l-18 30 M150 112 l18 30"/>
    <path d="M132 44 h36" stroke-width="1.4"/>
    <circle cx="127" cy="44" r="4"/>
    <circle cx="173" cy="44" r="4"/>
    <ellipse cx="171" cy="52" rx="4" ry="5"/>
    <rect x="114" y="90" width="14" height="12" rx="2"/>
    <rect x="164" y="126" width="16" height="12" rx="3"/>
    <g stroke-width="0.8">
      <line x1="180" y1="40" x2="250" y2="32"/>
      <line x1="178" y1="54" x2="250" y2="66"/>
      <line x1="114" y1="96" x2="60" y2="96"/>
      <line x1="180" y1="132" x2="250" y2="132"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="256" y="35">smart glasses (head)</text>
    <text x="256" y="69">hearable (ear)</text>
    <text x="56" y="99" text-anchor="end">smartwatch (wrist)</text>
    <text x="256" y="135">fitness band / clip (body)</text>
  </g>
</svg>
<figcaption>Wearables are grouped by where they sit on the body — glasses on the head, hearables at the ear, a smartwatch on the wrist, a band or clip on the torso — each a small sensor-and-radio package under tight size and battery limits.</figcaption>
</figure>

## Overview

Wearables span [smartwatches](/reference/smartwatch/) and fitness bands, smart glasses and AR headsets, and "hearables" such as smart earbuds. All share the same constraints: a tiny [SoC](/reference/system-on-a-chip/), a small or absent display, low-power radios like Bluetooth and [NFC](/reference/near-field-communication/), and a [battery](/reference/battery-technology/) measured in fractions of a phone's.

Many lean on a paired [smartphone](/reference/smartphone/) for heavy lifting and connectivity, acting as a sensor-rich satellite rather than a standalone computer. The design pressure is relentless: every added feature costs volume, weight, and battery on a device that must be comfortable to wear all day, so wearables sense and relay far more than they compute.

## Wearable form factors

The category spans several body positions, each suited to different jobs:

| Form | Worn on | Typical job |
|------|---------|-------------|
| Smartwatch | Wrist | Notify, sensors, apps |
| Fitness band | Wrist | Step & heart tracking |
| Smart glasses | Head | Display, camera, AR |
| Hearable | Ear | Audio, voice assistant |
| Smart clothing | Body | Biometrics, motion |

What unites them is not a screen or a chip but the fact of being worn — placement on the body is the defining trait.

## Where it fits

The category is defined by being *on* the body rather than in a pocket, which favors continuous sensing — motion, heart rate, location via [GPS](/reference/gps-receiver/) — over raw performance. The extreme size and power limits make wearables the most constrained tier of personal computing, suited to capturing and relaying data, not to running compute-heavy workloads like an SDR decode pipeline on their own.

## Sources

[^wiki]: [Wearable computer](https://en.wikipedia.org/wiki/Wearable_computer) — Wikipedia, on body-worn computing devices and their forms.
