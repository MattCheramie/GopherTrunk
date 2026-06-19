---
slug: kotlin-language
title: Kotlin
entry_type: language
category: programming-languages
description: Kotlin is a statically typed, concise JVM language from JetBrains that interoperates fully with Java and is Google's preferred language for Android development.
keywords: Kotlin, JVM, Android, JetBrains, null safety, coroutines, Java interoperability, statically typed, bytecode
aka: [Kotlin]
autolink: true
infobox:
  - { label: Paradigm, value: "Object-oriented, functional, imperative" }
  - { label: Typing, value: "Static, strong, inferred" }
  - { label: Appeared, value: "2011 (JetBrains)" }
  - { label: Designed by, value: "JetBrains" }
  - { label: Compilation, value: "Compiled to JVM bytecode (also JS/native)" }
  - { label: Memory, value: "Garbage-collected (JVM)" }
  - { label: Notable uses, value: "Android apps, server back ends, multiplatform" }
see_also: [java-language, type-system, bytecode, jit-compilation, garbage-collection, object-oriented-programming, functional-programming]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Type systems", url: /learn/intro-software-dev/type-systems/ }
external:
  - { title: "Kotlin (programming language) — Wikipedia", url: https://en.wikipedia.org/wiki/Kotlin_(programming_language) }
  - { title: "Kotlin programming language", url: https://kotlinlang.org/ }
---

**Kotlin** is a statically typed, concise programming language from JetBrains that
runs on the Java Virtual Machine, interoperates fully with [Java](/reference/java-language/),
and is Google's preferred language for Android development.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Kotlin and Java source both compile to JVM bytecode, which the Java Virtual Machine runs, showing their interoperability." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="20" width="80" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="60" y="41">.kt files</text>
    <rect x="20" y="86" width="80" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="107">.java files</text>
    <line x1="100" y1="37" x2="180" y2="62" stroke="currentColor" stroke-width="1.1"/>
    <line x1="100" y1="103" x2="180" y2="78" stroke="currentColor" stroke-width="1.1"/>
    <rect x="180" y="52" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="225" y="74">bytecode</text>
    <line x1="270" y1="70" x2="320" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="320" y="50" width="120" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="380" y="74">JVM runs it</text>
  </g>
</svg>
<figcaption>Kotlin and Java both compile to JVM bytecode, so the two can call each other freely in one project.</figcaption>
</figure>

## Overview

Kotlin compiles to the same JVM [bytecode](/reference/bytecode/) as Java and is
executed by the same [JIT-compiling](/reference/jit-compilation/) runtime, so it
inherits Java's mature ecosystem, libraries, and tooling while shedding much of the
language's verbosity. A Kotlin file can call Java code and vice versa, which let
teams adopt it incrementally rather than rewriting. Beyond the JVM, Kotlin can also
target JavaScript and native binaries, enabling shared "multiplatform" code.

## Key characteristics

Kotlin's [type system](/reference/type-system/) is static with strong inference, and
its standout feature is **null safety**: the type system distinguishes nullable from
non-nullable references, catching a whole class of null-pointer errors at compile
time. It blends [object-oriented](/reference/object-oriented-programming/) and
[functional](/reference/functional-programming/) styles, and its coroutines provide
lightweight asynchronous concurrency. The honest drawbacks: it still depends on the
JVM's [garbage collector](/reference/garbage-collection/) and startup overhead, the
non-JVM targets are less mature, and compile times can lag Java's. Its centre of
gravity remains Android and JVM back-end work rather than general-purpose ubiquity.
