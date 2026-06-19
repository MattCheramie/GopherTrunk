---
slug: bytecode
title: Bytecode
entry_type: concept
category: language-internals
description: Bytecode is a compact, portable instruction set produced by a compiler and executed by a virtual machine rather than directly by hardware.
keywords: bytecode, virtual machine, VM, intermediate representation, JVM, CPython, .pyc, IL, portable instructions
aka: [bytecode]
autolink: true
infobox:
  - { label: Category, value: Intermediate representation }
  - { label: Produced by, value: "A compiler" }
  - { label: Run by, value: "A virtual machine (VM)" }
  - { label: Property, value: "Portable across platforms" }
  - { label: Examples, value: "JVM bytecode, CPython .pyc, .NET IL" }
  - { label: Trade-off, value: "Portability vs VM overhead" }
see_also: [compiler, interpreter, jit-compilation, java-language, python-language, csharp-language]
related_lessons:
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
  - { title: "Packaging and distribution", url: /learn/intro-software-dev/packaging-and-distribution/ }
external:
  - { title: "Bytecode — Wikipedia", url: https://en.wikipedia.org/wiki/Bytecode }
---

**Bytecode** is a compact, portable instruction set that sits between human-readable
source and raw machine code: a [compiler](/reference/compiler/) produces it, and a
*virtual machine* (VM) executes it rather than the CPU running it directly.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Source compiles to bytecode, which a virtual machine runs on any platform." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="69">source</text>
    <line x1="90" y1="65" x2="140" y2="65" stroke="currentColor" stroke-width="1.1"/><text x="115" y="57" font-size="8">compile</text>
    <rect x="140" y="45" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="180" y="69">bytecode</text>
    <line x1="220" y1="65" x2="270" y2="65" stroke="currentColor" stroke-width="1.1"/>
    <rect x="270" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="310" y="63">virtual</text><text x="310" y="75">machine</text>
    <line x1="350" y1="65" x2="400" y2="65" stroke="currentColor" stroke-width="1.1"/><text x="378" y="57" font-size="8">any OS</text>
    <rect x="400" y="50" width="40" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="420" y="69">run</text>
  </g>
</svg>
<figcaption>The same bytecode runs on any machine that has the matching virtual machine installed.</figcaption>
</figure>

## How it works

Rather than emit instructions for a specific processor, the compiler emits bytecode —
a low-level program for an idealized, abstract machine. At run time the VM reads that
bytecode and either [interprets](/reference/interpreter/) it or, with
[JIT compilation](/reference/jit-compilation/), translates the hot parts to native
code. Because the bytecode targets the VM and not the hardware, the *same* bytecode
file runs unchanged on any platform where the VM is available.

## Trade-offs

The big win is **portability**: you ship one set of bytecode and it runs anywhere the
VM is installed, sidestepping the per-platform builds that native
[compilation](/reference/compiler/) requires. The cost is the VM layer — interpreting
bytecode adds overhead, and the user must have the runtime installed rather than a
self-contained [static binary](/reference/static-binary/). Bytecode is also more
compact and faster to load than re-parsing source each time.

## In practice

[Java](/reference/java-language/) and other JVM languages compile to JVM bytecode;
[Python](/reference/python-language/) caches `.pyc` bytecode for its VM; and
[C#](/reference/csharp-language/) and the rest of .NET compile to an intermediate
language (IL). This bytecode-plus-VM design is what lets these languages promise
"write once, run anywhere."
