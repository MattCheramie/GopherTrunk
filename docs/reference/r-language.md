---
slug: r-language
title: R
entry_type: language
category: programming-languages
description: R is a dynamic, interpreted language specialised for statistics, data analysis, and plotting, beloved by statisticians and researchers but less suited to general programming.
keywords: R language, statistics, data analysis, statistical computing, plotting, data science, CRAN, dynamic typing, interpreted
aka: [R]
autolink: true
infobox:
  - { label: Paradigm, value: "Functional, dynamic, array-oriented" }
  - { label: Typing, value: "Dynamic, weak" }
  - { label: Appeared, value: "1993 (University of Auckland)" }
  - { label: Designed by, value: "Ross Ihaka, Robert Gentleman" }
  - { label: Compilation, value: "Interpreted" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Statistics, research, data visualisation" }
see_also: [python-language, julia-language, matlab-language, interpreter, static-vs-dynamic-typing, garbage-collection, functional-programming]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Matching the language to the domain", url: /learn/intro-software-dev/language-for-the-domain/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.r-project.org/
  - https://en.wikipedia.org/wiki/R_(programming_language)
---

**R** is a dynamic, [interpreted](/reference/interpreter/) language specialised for
statistics, data analysis, and plotting.[^wiki] It is beloved by statisticians and
researchers, and is excellent for analysis and visualisation but less suited to
general-purpose programming.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A data table flows into R for statistical analysis, which produces a plotted chart." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="68">data</text><text x="55" y="80">table</text>
    <line x1="90" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="46" width="110" height="48" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="205" y="66">R: stats</text><text x="205" y="80">&amp; models</text>
    <line x1="260" y1="70" x2="320" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="320" y="40" width="120" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <line x1="332" y1="92" x2="428" y2="92" stroke="currentColor" stroke-width="1"/><line x1="332" y1="92" x2="332" y2="50" stroke="currentColor" stroke-width="1"/>
    <path d="M338 86 L360 70 L382 78 L404 56 L422 60" fill="none" stroke="currentColor" stroke-width="1.3"/>
  </g>
</svg>
<figcaption>R turns tabular data into statistical models and publication-quality plots.</figcaption>
</figure>

## Overview

R grew out of the academic statistics community and remains organised around that
work.[^home] Its data frames, vectorised operations, and enormous library of statistical
methods make it a natural fit for exploratory analysis, modelling, and producing
charts. Packages distributed through CRAN cover an extraordinary range of statistical
and domain-specific techniques, and tools like ggplot2 are renowned for the quality
of their graphics.

## Key characteristics

R is dynamically typed (see [static vs dynamic typing](/reference/static-vs-dynamic-typing/)),
[interpreted](/reference/interpreter/), and [garbage-collected](/reference/garbage-collection/),
with a [functional](/reference/functional-programming/) and array-oriented bent.[^home] Its
strengths are also its limits: it is purpose-built for statistics, so it is less
comfortable as a general-purpose language, and pure-R loops are **slow**, pushing heavy
work into vectorised or native code. For numerical and data work it competes with
[Python](/reference/python-language/) (which has a larger general ecosystem) and the
faster [Julia](/reference/julia-language/); for engineering signal work many reach for
[MATLAB](/reference/matlab-language/) instead.

## Sources

[^home]: [The R Project for Statistical Computing](https://www.r-project.org/) — official site, documentation, and the CRAN ecosystem.
[^wiki]: [R (programming language)](https://en.wikipedia.org/wiki/R_(programming_language)) — Wikipedia, for history, origins, and statistical focus.
