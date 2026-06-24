---
slug: android
title: Android
entry_type: concept
category: hw-mobile
description: Android is the Linux-based, open-source mobile operating system developed by Google, running on the majority of the world's smartphones and tablets as well as watches, TVs, and embedded devices.
keywords: Android, Google, AOSP, Linux mobile, smartphone OS, ART runtime, Kotlin, Java, app store
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

**Android** is the Linux-based, open-source [mobile operating system](/reference/mobile-operating-system/) developed by Google, running on most of the world's smartphones and tablets.[^wiki]

## Overview

Android is built on the Linux kernel with a managed runtime (ART) on top; apps are written mainly in Kotlin or Java against Google's SDK. The core platform is released as open source (the Android Open Source Project, AOSP), which lets handset makers ship their own variants, while Google layers proprietary services and the Play Store on top. Beyond phones it powers watches (Wear OS), TVs, cars, and embedded gear. It runs largely on [Arm](/reference/arm-architecture/)-based [SoCs](/reference/system-on-a-chip/).

## Where it fits

Android is the open counterweight to Apple's [iOS](/reference/ios/): more device variety and more openness, at the cost of fragmentation across versions and vendors. Its Linux foundation and sideloading make it the more hackable of the two — it is at least conceivable to drive an SDR dongle and a [mobile app](/reference/mobile-app-development/) front end from a rooted Android phone, though GopherTrunk's heavy decode pipeline still prefers a dedicated capture node.

## Sources

[^wiki]: [Android (operating system)](https://en.wikipedia.org/wiki/Android_(operating_system)) — Wikipedia, on Android's design, history, and ecosystem.
