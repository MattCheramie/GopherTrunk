---
slug: go-language
title: Go
entry_type: language
category: programming-languages
description: Go (Golang) is a statically typed, compiled language from Google designed for simple, reliable, concurrent systems software; it builds to a single static binary and has goroutines and channels built in.
keywords: Go, Golang, goroutines, channels, garbage collection, static binary, concurrency, Google, compiled language, cross-compilation
aka: [Go, Golang]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, concurrent, structured" }
  - { label: Typing, value: "Static, strong, inferred" }
  - { label: Appeared, value: "2009 (Google)" }
  - { label: Designed by, value: "Robert Griesemer, Rob Pike, Ken Thompson" }
  - { label: Compilation, value: "Ahead-of-time to a native static binary" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Concurrency, value: "Goroutines + channels (CSP)" }
  - { label: Notable uses, value: "GopherTrunk, Docker, Kubernetes" }
see_also: [goroutines, garbage-collection, concurrency, static-binary, cross-compilation, compiler, type-system, c-language, rust-language]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
  - { title: "Concurrency models", url: /learn/intro-software-dev/concurrency-models/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://go.dev/ref/spec
  - https://go.dev/
  - https://en.wikipedia.org/wiki/Go_(programming_language)
---

**Go** (often called **Golang** after its website) is a statically typed, compiled
programming language created at Google for building simple, reliable, and efficient
systems software.[^wiki] GopherTrunk itself is written in Go.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Source files compile ahead of time into one static binary that runs many concurrent goroutines communicating over a channel." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="60" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="50" y="74">.go files</text>
    <line x1="80" y1="70" x2="120" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="100" y="62" font-size="8">build</text>
    <rect x="120" y="50" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="160" y="68">static</text><text x="160" y="80">binary</text>
    <line x1="200" y1="70" x2="240" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="240" y="20" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="275" y="35">goroutine</text>
    <rect x="240" y="58" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="275" y="73">goroutine</text>
    <rect x="240" y="96" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="275" y="111">goroutine</text>
    <line x1="330" y1="31" x2="370" y2="69" stroke="currentColor" stroke-width="1.1"/><line x1="330" y1="69" x2="370" y2="69" stroke="currentColor" stroke-width="1.1"/><line x1="330" y1="107" x2="370" y2="69" stroke="currentColor" stroke-width="1.1"/>
    <rect x="370" y="55" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="405" y="72">channel</text>
  </g>
</svg>
<figcaption>Go compiles ahead of time into one self-contained binary that runs many lightweight goroutines communicating over channels.</figcaption>
</figure>

## Overview

Go pairs the performance of a [compiled](/reference/compiler/) language with the
simplicity of a deliberately small syntax.[^spec] Programs build [ahead of time](/reference/jit-compilation/)
into a single [static binary](/reference/static-binary/) with no runtime dependency,
which you can drop onto any matching machine and run. That same toolchain makes
[cross-compilation](/reference/cross-compilation/) — building a Windows or macOS
binary from a Linux host — a one-line affair.

## Key characteristics

Go's headline feature is first-class [concurrency](/reference/concurrency/): cheap
[goroutines](/reference/goroutines/) and channels make concurrent stream processing
natural.[^home] It is [garbage-collected](/reference/garbage-collection/), so there is no
manual memory management, and it has a [static type system](/reference/type-system/)
with inference that keeps code terse. The trade-offs are real — the garbage collector
introduces brief pauses, raw numeric throughput trails [C](/reference/c-language/) and
[Rust](/reference/rust-language/), and the minimalist design (generics arrived only in
2022) can feel limiting.

## Why GopherTrunk uses it

A self-contained static binary ships effortlessly to Linux, macOS, and Windows with no
installer or shared libraries, and goroutines map cleanly onto the many concurrent
decode pipelines an SDR scanner runs at once — making Go a strong fit for
infrastructure, network services, CLIs, and stream engines like this one.

## Sources

[^spec]: [The Go Programming Language Specification](https://go.dev/ref/spec) — the authoritative definition of the language and its type system.
[^home]: [The Go programming language](https://go.dev/) — official site, documentation, and the goroutine/channel concurrency model.
[^wiki]: [Go (programming language)](https://en.wikipedia.org/wiki/Go_(programming_language)) — Wikipedia, for history and design background.
