---
slug: cross-compilation
title: Cross-compilation
entry_type: concept
category: language-internals
description: Cross-compilation is building an executable on one platform that is meant to run on a different platform, such as producing a Windows binary from a Linux host.
keywords: cross-compilation, cross compiler, target platform, host platform, toolchain, GOOS GOARCH, build target, portability
aka: ["cross-compiling", "cross compiler"]
autolink: true
infobox:
  - { label: Category, value: Build process }
  - { label: Host, value: "Machine you build on" }
  - { label: Target, value: "Machine the output runs on" }
  - { label: Used for, value: "Releasing for many OS/CPU combinations" }
  - { label: First-class in, value: "Go (GOOS / GOARCH)" }
  - { label: Common with, value: "Embedded and mobile development" }
see_also: [compiler, static-binary, go-language, rust-language, c-language]
related_lessons:
  - { title: "Packaging and distribution", url: /learn/intro-software-dev/packaging-and-distribution/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Cross_compiler
---

**Cross-compilation** is building an executable on one platform — the *host* — that is
meant to run on a *different* platform — the *target* — such as producing a Windows or
macOS binary from a Linux machine without ever touching those operating systems.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="One source tree compiled on a Linux host produces binaries targeting Linux, macOS, and Windows." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="68">source on</text><text x="60" y="80">Linux host</text>
    <line x1="100" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="50" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="190" y="74">compiler</text>
    <line x1="230" y1="62" x2="290" y2="28" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="70" x2="290" y2="70" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="78" x2="290" y2="112" stroke="currentColor" stroke-width="1.1"/>
    <rect x="290" y="14" width="120" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="350" y="31">Linux binary</text>
    <rect x="290" y="57" width="120" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="350" y="74">macOS binary</text>
    <rect x="290" y="100" width="120" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="350" y="117">Windows binary</text>
  </g>
</svg>
<figcaption>One host compiles binaries for several target operating systems and CPU architectures.</figcaption>
</figure>

## How it works

A normal [compiler](/reference/compiler/) emits machine code for the machine it runs
on. A cross-compiling toolchain instead targets a chosen operating system and CPU
architecture, generating code and linking against the right libraries for that target.
The output is a binary that will not run on the host but will run on the target.[^wiki] This
is essential whenever the target cannot easily build for itself — embedded devices,
phones, or simply a release pipeline that produces every platform's download from one
build machine.

## Trade-offs

Cross-compilation removes the need to own or maintain a machine of every target type,
making multi-platform releases a single automated step. The complications come from
*platform-specific dependencies*: code that links against C libraries or calls
operating-system APIs may need matching target headers and libraries, and that setup
can be fiddly. Languages with little native dependency and a pure toolchain cross-compile
most smoothly, especially when they also emit a [static binary](/reference/static-binary/).[^wiki]

## In practice

[Go](/reference/go-language/) makes cross-compilation a one-line affair — setting
`GOOS` and `GOARCH` builds for any supported platform from any host — which is part of
why GopherTrunk can ship Linux, macOS, and Windows downloads from one build.
[Rust](/reference/rust-language/) supports many targets through its toolchain, and
[C](/reference/c-language/) cross-compiles with dedicated toolchains, though external
dependencies make it more involved.

## Sources

[^wiki]: [Cross compiler](https://en.wikipedia.org/wiki/Cross_compiler) — Wikipedia, on building executables for a target platform different from the host and why it matters for releases and embedded work.
