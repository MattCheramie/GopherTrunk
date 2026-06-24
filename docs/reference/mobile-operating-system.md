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

## Overview

Like any [operating system](/reference/operating-system/), a mobile OS schedules processes, manages memory, and brokers access to hardware. What distinguishes it is the emphasis: a [touchscreen](/reference/touchscreen/)-first interface, aggressive power management to stretch [battery](/reference/battery-technology/) life, and a strict per-app sandbox so untrusted code from an app store cannot reach the rest of the device. Two platforms dominate: [Android](/reference/android/) and [iOS](/reference/ios/). Apps are built with [mobile app development](/reference/mobile-app-development/) tools specific to each.

## Where it fits

The mobile OS is the layer between the [SoC](/reference/system-on-a-chip/) and the apps a user sees. It decides which background work runs, when the cellular and Wi-Fi radios wake, and how quickly the screen sleeps — choices that directly govern how long a charge lasts. Running a full SDR decode pipeline like GopherTrunk on such a device is unusual: the power budget and locked-down background limits favor a small Linux board near the antenna instead.

## Sources

[^wiki]: [Mobile operating system](https://en.wikipedia.org/wiki/Mobile_operating_system) — Wikipedia, on mobile OS design and examples.
