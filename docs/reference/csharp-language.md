---
slug: csharp-language
title: C#
entry_type: language
category: programming-languages
description: C# is Microsoft's statically typed, garbage-collected, JIT-compiled language on the .NET platform, the backbone of Windows enterprise software and, via Unity, much of the game industry.
keywords: C#, C sharp, csharp, .NET, CLR, JIT, garbage collection, Unity, enterprise, Microsoft, managed language
aka: ["C#", "C sharp", csharp]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: object-oriented, functional, imperative" }
  - { label: Typing, value: "Static, strong (with dynamic opt-in)" }
  - { label: Appeared, value: "2000 (Microsoft)" }
  - { label: Designed by, value: "Anders Hejlsberg; Microsoft" }
  - { label: Compilation, value: "Compiled to IL bytecode, JIT-compiled on the CLR" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Windows enterprise software, games (Unity), web backends" }
see_also: [java-language, jit-compilation, bytecode, garbage-collection, typescript-language, object-oriented-programming, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://learn.microsoft.com/en-us/dotnet/csharp/
  - https://en.wikipedia.org/wiki/C_Sharp_(programming_language)
---

**C#** (pronounced "C sharp") is Microsoft's statically typed, garbage-collected,
object-oriented language for the .NET platform.[^wiki] It is closely comparable to
[Java](/reference/java-language/) — a managed, JIT-compiled language with a large
ecosystem — with arguably more modern language features.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="C# source compiles to intermediate-language bytecode that the .NET CLR JIT-compiles to native code, with garbage-collected memory." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="70" height="38" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">.cs source</text>
    <line x1="90" y1="71" x2="130" y2="71" stroke="currentColor" stroke-width="1.1"/><text x="110" y="63" font-size="8">compile</text>
    <rect x="130" y="52" width="86" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="173" y="68">IL bytecode</text>
    <line x1="216" y1="71" x2="256" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="256" y="46" width="80" height="50" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="296" y="66">.NET CLR</text><text x="296" y="80" font-size="8">JIT + GC</text>
    <line x1="336" y1="71" x2="376" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="376" y="52" width="64" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="408" y="74">native</text>
  </g>
</svg>
<figcaption>C# compiles to IL bytecode that the .NET runtime JIT-compiles to native code, with automatic memory management.</figcaption>
</figure>

## Overview

C# compiles to intermediate-language [bytecode](/reference/bytecode/) that the .NET
Common Language Runtime (CLR) [JIT-compiles](/reference/jit-compilation/) to native code
at runtime.[^docs] It is [statically typed](/reference/type-system/),
[garbage-collected](/reference/garbage-collection/), and broadly
[object-oriented](/reference/object-oriented-programming/) with strong functional
features. Its lead designer, Anders Hejlsberg, also went on to create
[TypeScript](/reference/typescript-language/), and the two share a family resemblance in
their type systems.

## Strengths and trade-offs

C#'s strengths are a modern, expressive language design, a big ecosystem, and excellent
tooling, with trade-offs very similar to [Java](/reference/java-language/): managed
execution means convenience and safety at the price of a runtime, slower cold starts and
higher memory use than native code. Historically the platform was Windows-centric, but
.NET is now genuinely cross-platform, running on Linux and macOS as well.

## Where it's used

C# is the backbone of Windows enterprise software and a major language for web backends
and cloud services on .NET. Through the **Unity** engine it also underpins a large share
of the game industry. As with Java, the practical choice between the two often comes down
to ecosystem and platform rather than the languages themselves.

## Sources

[^docs]: [The C# programming language](https://learn.microsoft.com/en-us/dotnet/csharp/) — Microsoft's official C# and .NET documentation.
[^wiki]: [C Sharp (programming language)](https://en.wikipedia.org/wiki/C_Sharp_(programming_language)) — Wikipedia, for history and design background.
