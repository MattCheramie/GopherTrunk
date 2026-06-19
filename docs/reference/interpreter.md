---
slug: interpreter
title: Interpreter
entry_type: concept
category: language-internals
description: An interpreter is a program that reads source code (or bytecode) and executes it directly at run time, rather than compiling it to a native binary first.
keywords: interpreter, interpreted language, run time, bytecode, virtual machine, CPython, REPL, dynamic execution
aka: [interpreter]
autolink: true
infobox:
  - { label: Category, value: Language tooling }
  - { label: Input, value: "Source code or bytecode" }
  - { label: When it runs, value: "At run time, statement by statement" }
  - { label: Trade-off, value: "Flexibility & fast iteration vs slower execution" }
  - { label: Needs on target, value: "The interpreter installed" }
  - { label: Interpreted languages, value: "Python, Ruby, classic JavaScript" }
see_also: [compiler, bytecode, jit-compilation, static-vs-dynamic-typing, python-language, ruby-language, javascript-language]
related_lessons:
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
external:
  - { title: "Interpreter (computing) — Wikipedia", url: https://en.wikipedia.org/wiki/Interpreter_(computing) }
---

**An interpreter** is a program that reads source code and executes it directly,
statement by statement, every time the program runs — instead of translating the
whole program to a native binary ahead of time the way a [compiler](/reference/compiler/)
does.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An interpreter reads source statements one at a time and runs each immediately." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="20" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="35">line 1</text>
    <rect x="20" y="54" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="69">line 2</text>
    <rect x="20" y="88" width="70" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="103">line 3</text>
    <line x1="90" y1="31" x2="190" y2="60" stroke="currentColor" stroke-width="1.1"/><line x1="90" y1="65" x2="190" y2="65" stroke="currentColor" stroke-width="1.1"/><line x1="90" y1="99" x2="190" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="190" y="48" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="235" y="69">interpreter</text>
    <line x1="280" y1="65" x2="340" y2="65" stroke="currentColor" stroke-width="1.1"/><text x="310" y="57" font-size="8">runs now</text>
    <rect x="340" y="48" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="69">result</text>
  </g>
</svg>
<figcaption>The interpreter reads each statement and executes it immediately, with no separate build step.</figcaption>
</figure>

## How it works

A pure interpreter walks the source program and performs each operation as it
encounters it. In practice most modern "interpreted" languages take a half-step
first: they compile the source to compact [bytecode](/reference/bytecode/) for a
*virtual machine*, and the VM interprets that bytecode. CPython, for example,
compiles `.py` files to `.pyc` bytecode and runs it on the Python VM. Either way,
translation happens during execution rather than once up front.

## Trade-offs

Interpretation inverts the compiled story. Execution is **slower**, because the
interpreter adds overhead to every operation, and the user must have the interpreter
installed. In exchange you get **fast iteration** — no separate build step, so you
edit and re-run instantly — and **flexibility**, including the runtime dynamism that
[dynamic typing](/reference/static-vs-dynamic-typing/) allows and interactive
read-eval-print loops (REPLs). [JIT compilation](/reference/jit-compilation/) is a
hybrid that keeps the portable bytecode but compiles hot paths to native code to
recover speed.

## In practice

[Python](/reference/python-language/), [Ruby](/reference/ruby-language/), and classic
[JavaScript](/reference/javascript-language/) are the canonical interpreted languages.
Their flexibility and quick edit-run cycle make them popular for scripting, data work,
and rapid prototyping, where developer speed matters more than raw execution speed.
