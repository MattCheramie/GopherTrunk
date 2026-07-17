---
slug: mobile-operating-system
title: Mobile operating system
entry_type: concept
category: hw-mobile
description: A mobile operating system is the system software that runs phones and tablets, managing the touchscreen, radios, apps, and power on hardware that is battery-powered and always connected.
keywords: mobile operating system, mobile OS, Android, iOS, smartphone OS, tablet OS, app sandbox, power management
infobox:
  - { label: Type, value: Operating system }
  - { label: Runs on, value: Phones & tablets }
  - { label: Leading examples, value: Android, iOS }
  - { label: Key concerns, value: Touch UI, power, security }
see_also: [android, ios, operating-system, smartphone, tablet, mobile-app-development]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Mobile_operating_system
---

A **mobile operating system** is the system software that runs a phone or tablet — managing the touchscreen, radios, applications, and battery on hardware that is portable, always connected, and tightly power-constrained.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A mobile operating system shown as the middle layer between hardware and apps. At the bottom sits the SoC, radios, and sensors. In the middle, the mobile OS, whose three central jobs are labeled: a touch-first user interface, aggressive power management, and a per-app security sandbox. At the top are the user's apps, isolated from one another." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <rect x="40" y="16" width="380" height="26" rx="3"/>
    <rect x="40" y="56" width="380" height="52" rx="3" fill="currentColor" fill-opacity="0.05"/>
    <rect x="40" y="122" width="380" height="26" rx="3"/>
    <line x1="230" y1="42" x2="230" y2="56" stroke-width="0.9"/>
    <line x1="230" y1="108" x2="230" y2="122" stroke-width="0.9"/>
    <g stroke-width="0.9">
      <rect x="60" y="70" width="100" height="26" rx="3"/>
      <rect x="180" y="70" width="100" height="26" rx="3"/>
      <rect x="300" y="70" width="100" height="26" rx="3"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="230" y="33">apps (isolated from one another)</text>
    <text x="230" y="53">mobile OS</text>
    <text x="110" y="86">touch UI</text>
    <text x="230" y="86">power mgmt</text>
    <text x="350" y="86">app sandbox</text>
    <text x="230" y="139">SoC &#183; radios &#183; sensors</text>
  </g>
</svg>
<figcaption>A mobile OS is the layer between the hardware and the user's apps; its three defining jobs are a touch-first interface, aggressive power management, and a sandbox that isolates every app from the rest of the device.</figcaption>
</figure>

## Overview

Like any [operating system](/reference/operating-system/), a mobile OS schedules processes, manages memory, and brokers access to hardware. What distinguishes it is the emphasis: a [touchscreen](/reference/touchscreen/)-first interface, aggressive power management to stretch [battery](/reference/battery-technology/) life, and a strict per-app sandbox so untrusted code from an app store cannot reach the rest of the device.

Two platforms dominate: [Android](/reference/android/) and [iOS](/reference/ios/). Both descend from established kernels (Linux and Darwin) but reshape everything above the kernel around touch, mobility, and battery. Apps are built with [mobile app development](/reference/mobile-app-development/) tools specific to each, and both platforms curate distribution to keep the sandbox meaningful.

## Mobile OS vs desktop OS

The same fundamentals, tuned for very different constraints:

| Concern | Mobile OS | Desktop OS |
|---------|-----------|------------|
| Primary input | Touchscreen | Keyboard & mouse |
| Power | Battery-critical | Mains, relaxed |
| App model | Sandboxed, curated | Mostly open install |
| Background work | Tightly limited | Freely allowed |
| Updates | Whole-OS images | Piecemeal packages |

The restrictions that make a mobile OS efficient and secure are exactly what make it an awkward host for open-ended background workloads.

## Where it fits

The mobile OS is the layer between the [SoC](/reference/system-on-a-chip/) and the apps a user sees. It decides which background work runs, when the cellular and Wi-Fi radios wake, and how quickly the screen sleeps — choices that directly govern how long a charge lasts. Running a full SDR decode pipeline like GopherTrunk on such a device is unusual: the power budget and locked-down background limits favor a small Linux board near the antenna, with the phone used only as a client onto it.

## Sources

[^wiki]: [Mobile operating system](https://en.wikipedia.org/wiki/Mobile_operating_system) — Wikipedia, on mobile OS design and examples.
