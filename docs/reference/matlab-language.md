---
slug: matlab-language
title: MATLAB
entry_type: language
category: programming-languages
description: MATLAB is a commercial, matrix-oriented language and environment for engineering, signal processing, and control systems, superb for DSP prototyping but proprietary and licensed.
keywords: MATLAB, matrix, signal processing, DSP, control systems, engineering, numerical computing, Simulink, proprietary, licensed
aka: [MATLAB]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, array/matrix-oriented" }
  - { label: Typing, value: "Dynamic, weak" }
  - { label: Appeared, value: "1984 (MathWorks)" }
  - { label: Designed by, value: "Cleve Moler; MathWorks" }
  - { label: Compilation, value: "Interpreted (with JIT in the runtime)" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "DSP, control systems, engineering analysis" }
see_also: [julia-language, python-language, r-language, interpreter, fast-fourier-transform, software-defined-radio, static-vs-dynamic-typing]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Matching the language to the domain", url: /learn/intro-software-dev/language-for-the-domain/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.mathworks.com/products/matlab.html
  - https://en.wikipedia.org/wiki/MATLAB
---

**MATLAB** is a commercial, matrix-oriented language and environment from MathWorks
built for engineering, signal processing, and control systems.[^wiki] It is superb for DSP
prototyping but proprietary and licensed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A signal enters MATLAB, which applies matrix and FFT operations to produce a frequency spectrum plot." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="70" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">signal</text>
    <line x1="90" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="44" width="120" height="52" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="210" y="66">MATLAB:</text><text x="210" y="80">matrices, FFT</text>
    <line x1="270" y1="70" x2="330" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="330" y="40" width="110" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <line x1="342" y1="92" x2="428" y2="92" stroke="currentColor" stroke-width="1"/><line x1="342" y1="92" x2="342" y2="50" stroke="currentColor" stroke-width="1"/>
    <path d="M348 88 L362 60 L376 80 L390 54 L404 78 L420 66" fill="none" stroke="currentColor" stroke-width="1.3"/>
  </g>
</svg>
<figcaption>MATLAB treats data as matrices and excels at signal-processing analysis like the FFT and its spectra.</figcaption>
</figure>

## Overview

MATLAB ("matrix laboratory") was designed around the matrix as its native data type,
which makes linear-algebra and signal code read almost like the mathematics it
implements.[^home] It pairs the language with a full interactive environment, deep numerical
libraries, toolboxes for control, communications and DSP, and Simulink for
model-based design. That makes it a favourite for **DSP prototyping** and the kind of
[fast Fourier transform](/reference/fast-fourier-transform/) work that underlies
[software-defined radio](/reference/software-defined-radio/).

## Key characteristics

MATLAB is dynamically typed (see [static vs dynamic typing](/reference/static-vs-dynamic-typing/))
and [interpreted](/reference/interpreter/) with a JIT in its runtime.[^home] Its great
drawback is that it is **proprietary and licensed**: it costs money, toolboxes are sold
separately, and code is tied to a vendor's runtime rather than an open ecosystem. It
is also a specialist rather than a general-purpose language. For overlapping work,
open alternatives include [Python](/reference/python-language/) with NumPy/SciPy, the
faster [Julia](/reference/julia-language/), and [R](/reference/r-language/) for
statistics — each trading MATLAB's polished, licensed tooling for openness.

## Sources

[^home]: [MATLAB (MathWorks)](https://www.mathworks.com/products/matlab.html) — official product site, documentation, and the matrix environment and toolboxes.
[^wiki]: [MATLAB](https://en.wikipedia.org/wiki/MATLAB) — Wikipedia, for history, origins, and design background.
