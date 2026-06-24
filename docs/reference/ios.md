---
slug: ios
title: iOS
entry_type: concept
category: hw-mobile
description: iOS is Apple's mobile operating system, running on the iPhone and underpinning iPadOS and watchOS, known for tight hardware-software integration, a curated app store, and a strong security model.
keywords: iOS, Apple, iPhone, iPadOS, Swift, App Store, Apple Silicon, mobile OS, walled garden
autolink: true
infobox:
  - { label: Type, value: Mobile operating system }
  - { label: Developer, value: Apple }
  - { label: Runs on, value: iPhone (iPad, Watch kin) }
  - { label: First release, value: 2007 }
  - { label: Apps in, value: Swift, Objective-C }
see_also: [mobile-operating-system, android, smartphone, tablet, mobile-app-development, system-on-a-chip]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
  - { title: "Developing for mobile", url: /learn/intro-hardware/developing-for-mobile/ }
cite_urls:
  - https://en.wikipedia.org/wiki/IOS
---

**iOS** is Apple's [mobile operating system](/reference/mobile-operating-system/), introduced with the first iPhone in 2007 and known for tight integration between Apple's hardware and software.[^wiki]

## Overview

iOS runs only on Apple's own devices, atop Apple Silicon [SoCs](/reference/system-on-a-chip/) descended from the [Arm architecture](/reference/arm-architecture/). Apps are written in Swift or Objective-C and distributed through a curated App Store, with a strong sandbox and hardware-backed security (Secure Enclave, code signing). The same foundation underpins iPadOS and watchOS. Because Apple controls both silicon and software, it can tune power and performance closely — a "walled garden" that trades openness for consistency.

## Where it fits

iOS and [Android](/reference/android/) together define the modern [smartphone](/reference/smartphone/) landscape. iOS leans toward control and polish; Android toward openness and variety. iOS's locked-down model and lack of general USB host support make it a poor host for an SDR decode pipeline like GopherTrunk — it is far better suited as a thin client viewing data served by a capture node elsewhere.

## Sources

[^wiki]: [iOS](https://en.wikipedia.org/wiki/IOS) — Wikipedia, on Apple's mobile operating system.
