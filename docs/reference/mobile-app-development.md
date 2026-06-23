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

## Overview

The first is **native**: code written for one platform alone, with full hardware
access and the best performance — [Swift](/reference/swift-language/) for iOS,
[Kotlin](/reference/kotlin-language/) or [Java](/reference/java-language/) for
Android. The second is **cross-platform**: a shared codebase that runs on both,
such as Flutter or React Native (the latter built on
[JavaScript](/reference/javascript-language/)). The third is the **PWA**, or
progressive web app — a website that behaves like an installed app.

## Trade-offs

The three trade off reach against depth. Native gives the deepest hardware access
and smoothest performance but doubles the work to cover both platforms.
Cross-platform writes once and ships everywhere at some cost in polish and
native-feature access. A PWA is the lightest to build and needs no app store, but
runs inside the browser's limits. The build work itself happens on a
[personal computer](/reference/personal-computer/); the phone or tablet is the
target.

## Sources

[^wiki]: [Mobile app development](https://en.wikipedia.org/wiki/Mobile_app_development) — Wikipedia, on native, cross-platform, and web approaches.
