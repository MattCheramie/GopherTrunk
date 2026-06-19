---
slug: static-binary
title: Static binary
entry_type: concept
category: language-internals
description: A static binary is an executable that bundles all the code it needs at build time, so it runs without depending on shared libraries or a separate runtime on the target machine.
keywords: static binary, static linking, dynamic linking, shared library, self-contained executable, no dependencies, distribution, deployment
aka: ["static binary", "statically linked binary"]
autolink: true
infobox:
  - { label: Category, value: Executable packaging }
  - { label: Definition, value: "Executable with dependencies bundled in" }
  - { label: Contrast, value: "Dynamically linked binary" }
  - { label: Benefit, value: "Runs with no external runtime or libraries" }
  - { label: Cost, value: "Larger file, rebuild to update a dependency" }
  - { label: Common in, value: "Go, Rust, C" }
see_also: [compiler, cross-compilation, jit-compilation, go-language, rust-language, c-language]
related_lessons:
  - { title: "Packaging and distribution", url: /learn/intro-software-dev/packaging-and-distribution/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
external:
  - { title: "Static library — Wikipedia", url: https://en.wikipedia.org/wiki/Static_library }
---

**A static binary** is an executable that bundles all the code it needs at build time,
so it runs on its own — no shared libraries to find, no interpreter or runtime to
install on the target machine.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A static binary contains its libraries inside it and runs alone, unlike a dynamic binary that depends on external shared libraries." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="95" y="20" font-size="8" fill-opacity="0.7">static</text>
    <rect x="40" y="28" width="110" height="60" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.3"/><text x="95" y="46">app code</text>
    <rect x="55" y="54" width="36" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="73" y="69" font-size="7">lib A</text>
    <rect x="99" y="54" width="36" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="117" y="69" font-size="7">lib B</text>
    <text x="95" y="104" font-size="7" fill-opacity="0.7">runs alone</text>
    <text x="345" y="20" font-size="8" fill-opacity="0.7">dynamic</text>
    <rect x="290" y="40" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="330" y="64">app code</text>
    <line x1="370" y1="50" x2="410" y2="40" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/><line x1="370" y1="70" x2="410" y2="80" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/>
    <rect x="410" y="28" width="40" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="430" y="43" font-size="7">lib A</text>
    <rect x="410" y="68" width="40" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="430" y="83" font-size="7">lib B</text>
  </g>
</svg>
<figcaption>A static binary carries its libraries inside it; a dynamic one depends on shared libraries present on the machine.</figcaption>
</figure>

## How it works

When a [compiler](/reference/compiler/) builds a program, it links in the libraries the
code calls. With **dynamic linking** those libraries stay separate and are loaded from
the system at run time, so the binary depends on the right versions being installed.
With **static linking** the linker copies the needed library code directly into the
executable, producing one self-contained file. The result needs nothing external — not
even a language runtime, unlike [bytecode](/reference/bytecode/) that requires a virtual
machine.

## Trade-offs

The payoff is **deployment simplicity**: copy one file to a matching machine and run
it, with no installer, no dependency hunt, and no "works on my machine" version skew.
The costs are a **larger file** (every dependency is included) and that updating a
bundled library means rebuilding the binary rather than patching a shared one. Paired
with [cross-compilation](/reference/cross-compilation/), a static binary makes
shipping for many platforms a single, clean step.

## In practice

[Go](/reference/go-language/) produces a static binary by default, which is a major
reason it suits CLIs, network services, and tools like GopherTrunk that ship to users
who just want to run them. [Rust](/reference/rust-language/) and
[C](/reference/c-language/) can link statically as well, trading file size for the same
dependency-free convenience.
