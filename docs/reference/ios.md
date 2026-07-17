---
slug: ios
title: iOS
entry_type: concept
category: hw-mobile
description: iOS is Apple's mobile operating system, running on the iPhone and underpinning iPadOS and watchOS, known for tight hardware-software integration, a curated app store, and a strong security model.
keywords: iOS, Apple, iPhone, iPadOS, watchOS, Swift, App Store, Apple Silicon, mobile OS, walled garden, Secure Enclave
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="The iOS software stack as concentric layers. At the center is the Apple Silicon SoC with its Secure Enclave. Around it, the Darwin XNU kernel; then core services and Cocoa Touch frameworks; then the sandbox that isolates each app; and on the outside, apps installed only through the curated App Store." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" font-family="ui-sans-serif, sans-serif">
    <circle cx="150" cy="90" r="76" stroke-width="1.2"/>
    <circle cx="150" cy="90" r="58" stroke-width="1.2"/>
    <circle cx="150" cy="90" r="40" stroke-width="1.2"/>
    <circle cx="150" cy="90" r="20" stroke-width="1.2" fill-opacity="0.08" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="150" y="93">SoC</text>
    <text x="150" y="140">XNU kernel</text>
    <text x="150" y="160">core services / Cocoa Touch</text>
    <text x="150" y="24">App Store apps (curated)</text>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="270" y="60">Apple controls both the</text>
    <text x="270" y="74">silicon and the software,</text>
    <text x="270" y="88">so each ring is tuned to</text>
    <text x="270" y="102">the one inside it &#8212; a</text>
    <text x="270" y="116">tightly sealed &#8220;walled</text>
    <text x="270" y="130">garden.&#8221;</text>
  </g>
</svg>
<figcaption>iOS wraps Apple's own silicon in concentric layers — kernel, frameworks, per-app sandbox, and a curated App Store on the outside — with Apple controlling every ring, the integration that defines its "walled garden."</figcaption>
</figure>

## Overview

iOS runs only on Apple's own devices, atop Apple Silicon [SoCs](/reference/system-on-a-chip/) descended from the [Arm architecture](/reference/arm-architecture/). Apps are written in Swift or Objective-C and distributed through a curated App Store, with a strong sandbox and hardware-backed security such as the Secure Enclave and mandatory code signing.

The same foundation underpins iPadOS and watchOS, so a developer's skills and much of the code carry across Apple's device line. Because Apple controls both silicon and software, it can tune power and performance closely and push updates to nearly all devices at once — a "walled garden" that trades openness for consistency and a long, uniform support window.

## The Apple OS family

iOS is one member of a family sharing the same core:

| OS | Device | Shared core | Distinct trait |
|----|--------|-------------|----------------|
| iOS | iPhone | Darwin/XNU | Touch-first phone UI |
| iPadOS | iPad | Darwin/XNU | Multitasking, pointer |
| watchOS | Apple Watch | Darwin/XNU | Glanceable, tiny screen |
| macOS | Mac | Darwin/XNU | Full desktop, open files |

The common core is why features and frameworks propagate quickly across Apple's lineup.

## Where it fits

iOS and [Android](/reference/android/) together define the modern [smartphone](/reference/smartphone/) landscape. iOS leans toward control and polish; Android toward openness and variety. iOS's locked-down model and lack of general USB host support make it a poor host for an SDR decode pipeline like GopherTrunk — it is far better suited as a thin client viewing data served by a capture node elsewhere.

## Sources

[^wiki]: [iOS](https://en.wikipedia.org/wiki/IOS) — Wikipedia, on Apple's mobile operating system.
