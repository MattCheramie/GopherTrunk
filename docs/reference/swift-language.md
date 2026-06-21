---
slug: swift-language
title: Swift
entry_type: language
category: programming-languages
description: Swift is Apple's modern, statically typed, compiled language for iOS and macOS development; it is clean, fast and memory-safe, but its centre of gravity is Apple's platforms.
keywords: Swift, Apple, iOS, macOS, ARC, automatic reference counting, compiled, memory safety, LLVM, mobile
aka: [Swift]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: object-oriented, functional, protocol-oriented" }
  - { label: Typing, value: "Static, strong, inferred" }
  - { label: Appeared, value: "2014 (Apple)" }
  - { label: Designed by, value: "Chris Lattner; Apple" }
  - { label: Compilation, value: "Ahead-of-time to native machine code (LLVM)" }
  - { label: Memory, value: "Automatic reference counting (ARC)" }
  - { label: Notable uses, value: "iOS and macOS apps, Apple-platform software" }
see_also: [kotlin-language, rust-language, memory-management, compiler, type-system, object-oriented-programming, c-language]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.swift.org/
  - https://en.wikipedia.org/wiki/Swift_(programming_language)
---

**Swift** is Apple's modern, statically typed, compiled language for building iOS and
macOS applications.[^wiki] It is clean and fast and prioritises safety, but its centre of
gravity remains Apple's platforms.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Swift source compiles ahead of time to native code via LLVM, with memory managed automatically by reference counting, primarily targeting Apple platforms." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="74" height="38" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="57" y="74">.swift source</text>
    <line x1="94" y1="71" x2="134" y2="71" stroke="currentColor" stroke-width="1.1"/><text x="114" y="63" font-size="8">LLVM</text>
    <rect x="134" y="52" width="86" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="177" y="68">native code</text>
    <line x1="220" y1="71" x2="260" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="260" y="46" width="80" height="50" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="300" y="66">runtime</text><text x="300" y="80" font-size="8">ARC</text>
    <line x1="340" y1="71" x2="380" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="380" y="52" width="60" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="410" y="68">iOS /</text><text x="410" y="80">macOS</text>
  </g>
</svg>
<figcaption>Swift compiles to native code via LLVM, manages memory with reference counting, and targets Apple's platforms.</figcaption>
</figure>

## Overview

Swift [compiles](/reference/compiler/) ahead of time to native machine code through LLVM,
giving it performance well beyond Apple's earlier Objective-C in many cases.[^home] It is
[statically typed](/reference/type-system/) with type inference, supports
[object-oriented](/reference/object-oriented-programming/) and functional styles, and
manages [memory](/reference/memory-management/) automatically through Automatic Reference
Counting (ARC) rather than a tracing garbage collector. The language emphasises safety —
clear handling of optionals and [errors](/reference/error-handling/) is built into the
design.

## Strengths and trade-offs

Swift's strengths are a clean, modern syntax, strong safety guarantees and good native
performance, with first-class tooling on Apple's platforms. The main trade-off is reach:
although Swift is open source and runs on Linux and elsewhere, its ecosystem, libraries
and tooling are overwhelmingly centred on Apple, so it is far less common off those
platforms. In that respect it parallels [Kotlin](/reference/kotlin-language/), the modern
mobile language tied instead to the JVM and Android.

## Where it's used

Swift is the primary language for iOS, iPadOS, macOS, watchOS and tvOS application
development, and for most new software written for Apple's ecosystem. Outside that world
it sees comparatively little use.

## Sources

[^home]: [The Swift programming language](https://www.swift.org/) — official site, documentation, and the language and toolchain.
[^wiki]: [Swift (programming language)](https://en.wikipedia.org/wiki/Swift_(programming_language)) — Wikipedia, for history and design background.
