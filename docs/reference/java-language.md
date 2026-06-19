---
slug: java-language
title: Java
entry_type: language
category: programming-languages
description: Java is a statically typed, garbage-collected, JIT-compiled language that runs on the JVM, offering real cross-platform portability and a massive, mature enterprise ecosystem.
keywords: Java, JVM, bytecode, JIT, garbage collection, enterprise, Android, write once run anywhere, object-oriented
aka: [Java]
autolink: true
infobox:
  - { label: Paradigm, value: "Object-oriented, imperative, generic" }
  - { label: Typing, value: "Static, strong" }
  - { label: Appeared, value: "1995 (Sun Microsystems)" }
  - { label: Designed by, value: "James Gosling; Sun Microsystems" }
  - { label: Compilation, value: "Compiled to bytecode, JIT-compiled on the JVM" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Enterprise systems, Android backends, big data" }
see_also: [csharp-language, jit-compilation, bytecode, garbage-collection, kotlin-language, object-oriented-programming, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
external:
  - { title: "Java (programming language) — Wikipedia", url: https://en.wikipedia.org/wiki/Java_(programming_language) }
  - { title: "The Java platform — Oracle", url: https://www.java.com/ }
---

**Java** is a statically typed, garbage-collected, object-oriented language that
compiles to bytecode and runs on the Java Virtual Machine (JVM), giving it genuine
"write once, run anywhere" portability across platforms.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Java source compiles once to portable bytecode, which the JVM JIT-compiles to native code on each platform at runtime." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="70" height="38" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">.java</text>
    <line x1="90" y1="71" x2="128" y2="71" stroke="currentColor" stroke-width="1.1"/><text x="109" y="63" font-size="8">javac</text>
    <rect x="128" y="52" width="78" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="167" y="74">bytecode</text>
    <line x1="206" y1="71" x2="244" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="244" y="46" width="70" height="50" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="279" y="66">JVM</text><text x="279" y="80" font-size="8">JIT + GC</text>
    <line x1="314" y1="60" x2="356" y2="42" stroke="currentColor" stroke-width="1.1"/><line x1="314" y1="82" x2="356" y2="100" stroke="currentColor" stroke-width="1.1"/>
    <rect x="356" y="28" width="84" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="398" y="45">Windows</text>
    <rect x="356" y="88" width="84" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="398" y="105">Linux / mac</text>
  </g>
</svg>
<figcaption>Java compiles once to portable bytecode; each platform's JVM JIT-compiles it to native code at runtime.</figcaption>
</figure>

## Overview

Java source compiles to platform-independent [bytecode](/reference/bytecode/), which the
JVM [JIT-compiles](/reference/jit-compilation/) to native code as the program runs. It is
[statically typed](/reference/type-system/), thoroughly
[object-oriented](/reference/object-oriented-programming/), and
[garbage-collected](/reference/garbage-collection/). The combination of a portable
runtime, strong tooling and decades of libraries has made it one of the most widely
deployed languages in large-scale software.

## Strengths and trade-offs

Java's strengths are a massive, mature ecosystem, strong tooling, real cross-platform
portability and battle-tested reliability at scale. The common criticisms are that it is
**verbose** and ceremony-heavy, and that the JVM brings slower startup and higher memory
use than native code. Modern Java has modernised considerably, and on the same platform
[Kotlin](/reference/kotlin-language/) offers a more concise alternative that interoperates
fully with Java code.

## Where it's used

Java runs enormous enterprise systems, Android backends, big-data platforms and
long-lived business applications. Its closest counterpart is [C#](/reference/csharp-language/)
on .NET, which shares almost the same managed, JIT-compiled, garbage-collected design;
the choice between them is usually about ecosystem and platform rather than language
merit.
