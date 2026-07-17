---
slug: mobile-app-development
title: Mobile app development
entry_type: concept
category: hw-mobile
description: Mobile app development is building software for phones and tablets through three broad approaches — native, cross-platform, and progressive web apps — each with different trade-offs.
keywords: mobile app development, native app, cross-platform, PWA, progressive web app, Flutter, React Native, iOS, Android
aka: [Mobile app development, Mobile development, App development]
infobox:
  - { label: Type, value: Software development concept }
  - { label: Targets, value: Phones, tablets }
  - { label: Native, value: Swift, Kotlin/Java }
  - { label: Cross-platform, value: Flutter, React Native }
  - { label: Web, value: PWA }
see_also: [smartphone, tablet, swift-language, kotlin-language, java-language, javascript-language]
related_lessons:
  - { title: "Developing for mobile", url: /learn/intro-hardware/developing-for-mobile/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Mobile_app_development
---

**Mobile app development** is building software for
[smartphones](/reference/smartphone/) and [tablets](/reference/tablet/), and it
comes in three broad flavors.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Three approaches to mobile app development converging on the same two devices. Native code is written separately for iOS and Android. A cross-platform framework compiles one shared codebase to both. A progressive web app runs inside the browser on either. All three ultimately deliver to a phone and a tablet." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <rect x="30" y="24" width="120" height="28" rx="3"/>
    <rect x="30" y="70" width="120" height="28" rx="3"/>
    <rect x="30" y="116" width="120" height="28" rx="3"/>
    <rect x="330" y="52" width="46" height="66" rx="4"/>
    <rect x="390" y="60" width="52" height="50" rx="4"/>
    <line x1="150" y1="38" x2="330" y2="70"/>
    <line x1="150" y1="84" x2="330" y2="84"/>
    <line x1="150" y1="130" x2="330" y2="100"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="90" y="42">native (Swift / Kotlin)</text>
    <text x="90" y="88">cross-platform</text>
    <text x="90" y="134">PWA (web)</text>
    <text x="353" y="128">phone</text>
    <text x="416" y="120">tablet</text>
  </g>
</svg>
<figcaption>Native, cross-platform, and web (PWA) are three routes to the same destination — a phone and a tablet — trading how much they share against how deeply they reach the hardware.</figcaption>
</figure>

## Overview

The first is **native**: code written for one platform alone, with full hardware
access and the best performance — [Swift](/reference/swift-language/) for iOS,
[Kotlin](/reference/kotlin-language/) or [Java](/reference/java-language/) for
Android. The second is **cross-platform**: a shared codebase that runs on both,
such as Flutter or React Native (the latter built on
[JavaScript](/reference/javascript-language/)). The third is the **PWA**, or
progressive web app — a website that behaves like an installed app.

Which flavor a team picks turns on how much they value reach against depth, and
how much duplicated work they can afford. The build work itself always happens on
a [personal computer](/reference/personal-computer/); the phone or tablet is only
the deployment target, tested against emulators and physical devices.

## Comparing the approaches

The three trade reach against depth in a fairly clean way:

| Approach | Codebases | Hardware access | Performance | App store |
|----------|-----------|-----------------|-------------|-----------|
| Native | One per platform | Full | Best | Yes |
| Cross-platform | One shared | Most | Very good | Yes |
| PWA | One (web) | Limited | Good | Not required |

Native gives the deepest access and smoothest feel but doubles the work; a PWA
is lightest and store-free but lives inside the browser's limits; cross-platform
sits in between.

## Where it fits

Mobile app development is how the client side of a mobile experience gets built —
including a lightweight front end for viewing data a server produces. A GopherTrunk
decoder typically exposes a web console, so the simplest "app" for checking it from
a phone is a PWA pointed at that console; a native or cross-platform app would only
be worth the effort if it needed deeper device features like background
notifications. The heavy decoding stays on the capture node, not the handset.

## Sources

[^wiki]: [Mobile app development](https://en.wikipedia.org/wiki/Mobile_app_development) — Wikipedia, on native, cross-platform, and web approaches.
