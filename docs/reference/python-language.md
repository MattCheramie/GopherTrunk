---
slug: python-language
title: Python
entry_type: language
category: programming-languages
description: Python is a dynamically typed, interpreted, general-purpose language prized for readability and a vast ecosystem, and is the default tool for data science, machine learning, scripting and automation.
keywords: Python, CPython, dynamic typing, interpreted, data science, machine learning, NumPy, pandas, PyTorch, GIL, scripting
aka: [Python, CPython]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: imperative, object-oriented, functional" }
  - { label: Typing, value: "Dynamic, strong, duck-typed (optional hints)" }
  - { label: Appeared, value: "1991 (CWI, Netherlands)" }
  - { label: Designed by, value: "Guido van Rossum" }
  - { label: Compilation, value: "Compiled to bytecode, run on an interpreter (CPython)" }
  - { label: Memory, value: "Garbage-collected (reference counting + cycle detection)" }
  - { label: Notable uses, value: "Data science, ML, scripting, automation, web backends" }
see_also: [interpreter, bytecode, garbage-collection, static-vs-dynamic-typing, javascript-language, julia-language, r-language, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.python.org/
  - https://en.wikipedia.org/wiki/Python_(programming_language)
---

**Python** is a dynamically typed, interpreted, general-purpose programming language
designed for readability, with a deliberately clean syntax and one of the largest
library ecosystems of any language.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Python source is compiled to bytecode and executed by the interpreter, with heavy work delegated to fast native libraries." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="64" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="52" y="74">.py source</text>
    <line x1="84" y1="70" x2="124" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="104" y="62" font-size="8">compile</text>
    <rect x="124" y="50" width="70" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="159" y="74">bytecode</text>
    <line x1="194" y1="70" x2="234" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="234" y="50" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="274" y="68">interpreter</text><text x="274" y="80" font-size="8">(CPython)</text>
    <line x1="314" y1="70" x2="354" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="334" y="62" font-size="8">calls</text>
    <rect x="354" y="50" width="86" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="397" y="68">native libs</text><text x="397" y="80" font-size="8">NumPy / C</text>
  </g>
</svg>
<figcaption>Python compiles to bytecode for an interpreter and leans on fast native libraries for heavy numeric work.</figcaption>
</figure>

## Overview

Python source is compiled to [bytecode](/reference/bytecode/) and run by an
[interpreter](/reference/interpreter/) — most commonly CPython, the reference
implementation.[^home] It uses [dynamic typing](/reference/static-vs-dynamic-typing/) with
optional type hints, and is [garbage-collected](/reference/garbage-collection/) through
reference counting plus a cycle detector. The result is a language that is quick to
write and forgiving to learn, which is why it is a common first language and the glue
of choice for automation and scripting.

## Strengths and trade-offs

Python's strengths are readability, a gentle learning curve, and an enormous ecosystem —
libraries such as NumPy, pandas and PyTorch make it the default for data science,
machine learning and scientific computing. The trade-offs are real: pure-Python loops
are **slow** compared with [compiled](/reference/compiler/) languages, and the global
interpreter lock (GIL) limits true multithreaded CPU [parallelism](/reference/concurrency/)
in the standard interpreter. The usual remedy is to delegate heavy work to native
libraries written in [C](/reference/c-language/), keeping the convenient Python layer on
top.

## Where it's used

Python dominates data science, machine learning and scientific computing, and is a
mainstay of scripting, automation, glue code and web backends. Its speed limits rarely
matter for scripts or for workloads that push the real computation into native code; in
the niche where raw numeric throughput is decisive, alternatives like
[Julia](/reference/julia-language/) or a compiled language are reached for instead.

## Sources

[^home]: [The Python programming language](https://www.python.org/) — official site, documentation, and the CPython reference implementation.
[^wiki]: [Python (programming language)](https://en.wikipedia.org/wiki/Python_(programming_language)) — Wikipedia, for history and design background.
