---
slug: julia-language
title: Julia
entry_type: language
category: programming-languages
description: Julia is a high-performance, dynamically typed language for numerical and scientific computing that aims to be as fast as C while as easy to write as Python.
keywords: Julia, scientific computing, numerical computing, high performance, JIT, multiple dispatch, dynamic typing, data science
aka: [Julia]
autolink: true
infobox:
  - { label: Paradigm, value: "Multiple dispatch, functional, imperative" }
  - { label: Typing, value: "Dynamic with optional type annotations" }
  - { label: Appeared, value: "2012 (MIT)" }
  - { label: Designed by, value: "Jeff Bezanson, Stefan Karpinski, Viral Shah, Alan Edelman" }
  - { label: Compilation, value: "Just-in-time (JIT) to native code" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Numerical/scientific computing, data science" }
see_also: [python-language, r-language, matlab-language, jit-compilation, static-vs-dynamic-typing, garbage-collection, compiler]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://julialang.org/
  - https://en.wikipedia.org/wiki/Julia_(programming_language)
---

**Julia** is a high-performance, dynamically typed language designed for numerical and
scientific computing, aiming to be as fast as [C](/reference/c-language/) while as
easy to write as [Python](/reference/python-language/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Python-like Julia source is JIT compiled to native code, reaching speeds close to C." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="90" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="65" y="68">easy</text><text x="65" y="80">.jl code</text>
    <line x1="110" y1="70" x2="175" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="142" y="62" font-size="8">JIT</text>
    <rect x="175" y="50" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="225" y="68">native</text><text x="225" y="80">machine code</text>
    <line x1="275" y1="70" x2="340" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="340" y="50" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="390" y="68">C-class</text><text x="390" y="80">speed</text>
  </g>
</svg>
<figcaption>Julia JIT-compiles high-level code to native machine code, chasing C-level speed from a Python-like syntax.</figcaption>
</figure>

## Overview

Julia was created to end the "two-language problem," where scientists prototype in a
convenient language and then rewrite hot paths in a fast one. It compiles
[just in time](/reference/jit-compilation/) to native code, and its **multiple
dispatch** model — choosing a method based on the types of all arguments — lets
generic, readable code specialise into tight machine code.[^home] The result is high-level
syntax that can rival compiled languages on numeric workloads.

## Key characteristics

Julia is dynamically typed with optional annotations (see
[static vs dynamic typing](/reference/static-vs-dynamic-typing/)) and
[garbage-collected](/reference/garbage-collection/).[^home] Its drawbacks are practical: the
**ecosystem is smaller** than Python's or R's, and JIT compilation causes "time to
first plot" latency, where the first run of new code pays a compilation cost. For
numerical and scientific work it competes with [Python](/reference/python-language/)
(broader libraries), [R](/reference/r-language/) (statistics depth), and
[MATLAB](/reference/matlab-language/) (mature, licensed engineering tooling), trading a
younger ecosystem for performance and openness.

## Sources

[^home]: [The Julia programming language](https://julialang.org/) — official site, documentation, and the multiple-dispatch and JIT model.
[^wiki]: [Julia (programming language)](https://en.wikipedia.org/wiki/Julia_(programming_language)) — Wikipedia, for history, origins, and design goals.
