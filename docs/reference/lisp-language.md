---
slug: lisp-language
title: LISP
entry_type: language
category: programming-languages
description: LISP (1958) is one of the oldest high-level languages; it treats code and data as the same kind of list, pioneered the functional style, and became the language of early AI research.
keywords: LISP, Lisp, functional programming, symbolic computing, artificial intelligence, John McCarthy, S-expressions, macros, code as data
aka: [Lisp]
autolink: true
infobox:
  - { label: Paradigm, value: "Functional, symbolic, multi-paradigm" }
  - { label: Typing, value: "Dynamic, strong" }
  - { label: Appeared, value: "1958 (MIT)" }
  - { label: Designed by, value: "John McCarthy" }
  - { label: Compilation, value: "Interpreted or compiled (varies by dialect)" }
  - { label: Memory, value: "Garbage-collected (pioneered GC)" }
  - { label: Notable uses, value: "AI research, symbolic computing, language design" }
see_also: [fortran-language, cobol-language, functional-programming, garbage-collection, interpreter, static-vs-dynamic-typing, python-language]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "The birth of programming languages", url: /learn/intro-software-dev/birth-of-languages/ }
external:
  - { title: "Lisp (programming language) — Wikipedia", url: https://en.wikipedia.org/wiki/Lisp_(programming_language) }
  - { title: "Common Lisp", url: https://common-lisp.net/ }
---

**LISP** (LISt Processor), created by John McCarthy in 1958, is one of the oldest
high-level languages. It treats code and data as the same kind of list, pioneered the
[functional](/reference/functional-programming/) style, and became the language of
early artificial-intelligence research.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A nested parenthesised LISP expression is shown as a tree of lists, illustrating that code and data share one structure." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="76">(+ 1 (* 2 3))</text>
    <line x1="140" y1="72" x2="180" y2="72" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="230" cy="40" r="14" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="43">+</text>
    <circle cx="290" cy="90" r="14" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="290" y="93">1</text>
    <circle cx="360" cy="90" r="14" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="360" y="93">*</text>
    <circle cx="330" cy="40" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="330" y="43">2</text>
    <circle cx="420" cy="40" r="13" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="420" y="43">3</text>
    <line x1="230" y1="54" x2="285" y2="78" stroke="currentColor" stroke-width="1"/><line x1="240" y1="48" x2="352" y2="80" stroke="currentColor" stroke-width="1"/>
    <line x1="356" y1="78" x2="334" y2="52" stroke="currentColor" stroke-width="1"/><line x1="364" y1="78" x2="416" y2="52" stroke="currentColor" stroke-width="1"/>
  </g>
</svg>
<figcaption>A LISP expression is a nested list; because code and data share one form, programs can manipulate programs.</figcaption>
</figure>

## History

LISP took a path entirely different from its contemporaries
[FORTRAN](/reference/fortran-language/) and [COBOL](/reference/cobol-language/).
Programs are written as parenthesised lists, and crucially the *code* is itself a list
— so a LISP program can read, build, and transform other LISP programs as ordinary
data. From that idea came macros, the functional style, automatic
[garbage collection](/reference/garbage-collection/) (which LISP pioneered), and the
interactive read-eval-print loop. It became the working language of early AI labs and
seeded ideas that mainstream languages adopted only decades later.

## Legacy and use today

LISP is dynamically typed (see [static vs dynamic typing](/reference/static-vs-dynamic-typing/)),
runs [interpreted](/reference/interpreter/) or compiled depending on the dialect, and
survives in families like Common Lisp, Scheme, and Clojure. Its honest drawbacks are
real: the parenthesis-heavy syntax is an acquired taste, and its mainstream use is
niche today. But its influence is enormous — first-class functions, garbage
collection, and treating code as data have all flowed into modern languages such as
[Python](/reference/python-language/) and JavaScript, making LISP one of the most
quietly influential designs in computing.
