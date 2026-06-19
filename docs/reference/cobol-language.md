---
slug: cobol-language
title: COBOL
entry_type: language
category: programming-languages
description: COBOL (1959) is an English-like high-level language built for business data processing; shaped by Grace Hopper, it still runs huge amounts of banking and government software.
keywords: COBOL, business data processing, Grace Hopper, English-like syntax, mainframe, banking, legacy code, high-level language
aka: [Cobol]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, procedural" }
  - { label: Typing, value: "Static, strong" }
  - { label: Appeared, value: "1959 (CODASYL committee)" }
  - { label: Designed by, value: "Committee; influenced by Grace Hopper" }
  - { label: Compilation, value: "Compiled to native code" }
  - { label: Memory, value: "Static (no garbage collector)" }
  - { label: Notable uses, value: "Banking, government, mainframe business systems" }
see_also: [fortran-language, lisp-language, c-language, compiler, imperative-programming, type-system, java-language]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "The birth of programming languages", url: /learn/intro-software-dev/birth-of-languages/ }
external:
  - { title: "COBOL — Wikipedia", url: https://en.wikipedia.org/wiki/COBOL }
  - { title: "GnuCOBOL", url: https://gnucobol.sourceforge.io/ }
---

**COBOL** (COmmon Business-Oriented Language), introduced in 1959, is a deliberately
English-like high-level language built for business data processing. Shaped by the
work of Grace Hopper and a committee, it still runs huge amounts of banking and
government software today.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="English-like COBOL statements are compiled and run on a mainframe to process business records like payroll and accounts." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="48" width="110" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="75" y="66">ADD PAY</text><text x="75" y="80">TO TOTAL.</text>
    <line x1="130" y1="70" x2="185" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="185" y="48" width="90" height="44" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="74">mainframe</text>
    <line x1="275" y1="70" x2="330" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="330" y="48" width="110" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="66">business</text><text x="385" y="80">records</text>
  </g>
</svg>
<figcaption>COBOL's English-like syntax was meant to make business logic readable; it still drives mainframe data processing.</figcaption>
</figure>

## History

COBOL used English-like syntax on purpose, so business processes would be readable by
non-specialists, not just programmers. A [compiler](/reference/compiler/) translates
those verbose, statement-style programs into machine code. It was standardised by a
committee under pressure from the U.S. government to create a common business language,
and Grace Hopper's earlier work on English-like programming heavily influenced its
design. It became, alongside [FORTRAN](/reference/fortran-language/) for science and
[LISP](/reference/lisp-language/) for AI, one of the defining languages of the first
era of high-level programming.

## Legacy and use today

COBOL is [statically typed](/reference/type-system/),
[imperative](/reference/imperative-programming/), and compiled to native code without a
[garbage collector](/reference/garbage-collection/). Astonishingly, vast amounts of
banking, insurance, and government software still run on it, processing daily
transactions on mainframes. That is also its core problem: the code is decades old,
the pool of programmers who know it is shrinking, and its verbosity and age make it a
poor fit for new work, which has largely moved to languages like
[Java](/reference/java-language/) and [C](/reference/c-language/). Its endurance is a
reminder that working software rarely gets rewritten just because it is old.
