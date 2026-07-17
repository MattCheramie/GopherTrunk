---
slug: android
title: Android
entry_type: concept
category: hw-mobile
description: Android is the Linux-based, open-source mobile operating system developed by Google, running on the majority of the world's smartphones and tablets as well as watches, TVs, and embedded devices.
keywords: Android, Google, AOSP, Linux mobile, smartphone OS, ART runtime, Kotlin, Java, app store, Wear OS
autolink: true
infobox:
  - { label: Type, value: Mobile operating system }
  - { label: Developer, value: Google (AOSP) }
  - { label: Kernel, value: Linux }
  - { label: First release, value: 2008 }
  - { label: Apps in, value: Kotlin, Java }
see_also: [mobile-operating-system, ios, smartphone, tablet, mobile-app-development, system-on-a-chip]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
  - { title: "Developing for mobile", url: /learn/intro-hardware/developing-for-mobile/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Android_(operating_system)
---

**Android** is the Linux-based, open-source [mobile operating system](/reference/mobile-operating-system/) developed by Google, running on most of the world's smartphones and tablets as well as watches, TVs, and embedded devices.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="The Android software stack drawn as layers. From the bottom: the Linux kernel with device drivers, a hardware abstraction layer, native libraries beside the ART managed runtime, the Java-based application framework, and finally user apps at the top. Each layer only talks to the ones directly around it." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g fill-opacity="0.05" stroke-width="1.2">
      <rect x="40" y="16" width="380" height="26" rx="3"/>
      <rect x="40" y="48" width="380" height="26" rx="3"/>
      <rect x="40" y="80" width="185" height="26" rx="3"/>
      <rect x="235" y="80" width="185" height="26" rx="3"/>
      <rect x="40" y="112" width="380" height="26" rx="3"/>
      <rect x="40" y="144" width="380" height="26" rx="3"/>
    </g>
    <g stroke="none" text-anchor="middle" font-size="9.5">
      <text x="230" y="33">Apps &#183; Play Store &amp; sideloaded</text>
      <text x="230" y="65">Java application framework (Activities, services)</text>
      <text x="132" y="97">Native libraries</text>
      <text x="327" y="97">ART managed runtime</text>
      <text x="230" y="129">Hardware abstraction layer (HAL)</text>
      <text x="230" y="161">Linux kernel &#183; drivers &#183; power &amp; security</text>
    </g>
  </g>
</svg>
<figcaption>Android is a stack of layers: a Linux kernel at the base drives the hardware, a HAL and native libraries sit above it, the ART runtime executes app bytecode, and the Java framework exposes the APIs that user apps are written against.</figcaption>
</figure>

## Overview

Android is built on the Linux kernel with a managed runtime (ART) on top; apps are written mainly in Kotlin or Java against Google's SDK and compiled to bytecode that ART executes. The core platform is released as open source — the Android Open Source Project, AOSP — which lets handset makers ship their own variants, while Google layers proprietary services and the Play Store on top.

Because the source is open and the kernel is Linux, Android is unusually adaptable. Beyond phones it powers watches (Wear OS), TVs, cars (Android Auto/Automotive), and embedded gear, and it runs largely on [Arm](/reference/arm-architecture/)-based [SoCs](/reference/system-on-a-chip/). That openness is also its main tension: dozens of vendors and long update tails produce *fragmentation* across versions and devices that developers must design around.

## Android vs iOS

The two dominant mobile platforms differ most in how open and how uniform they are:

| Aspect | Android | iOS |
|--------|---------|-----|
| Source model | Open (AOSP) | Closed |
| Vendors | Many | Apple only |
| Kernel | Linux | Darwin/XNU |
| App languages | Kotlin, Java | Swift, Objective-C |
| Install sources | Store + sideload | App Store (curated) |
| Fragmentation | High | Low |

Android trades Apple's uniformity for reach and flexibility; iOS trades Android's variety for a single tightly controlled target.

## Where it fits

Android is the open counterweight to Apple's [iOS](/reference/ios/): more device variety and more openness, at the cost of fragmentation across versions and vendors. Its Linux foundation and sideloading make it the more hackable of the two — it is at least conceivable to drive an SDR dongle and a [mobile app](/reference/mobile-app-development/) front end from a rooted Android phone, though GopherTrunk's heavy decode pipeline still prefers a dedicated capture node with a phone acting as a remote web console.

## Sources

[^wiki]: [Android (operating system)](https://en.wikipedia.org/wiki/Android_(operating_system)) — Wikipedia, on Android's design, history, and ecosystem.
