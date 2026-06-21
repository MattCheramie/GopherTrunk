---
slug: javascript-language
title: JavaScript
entry_type: language
category: programming-languages
description: JavaScript is a dynamically typed, JIT-compiled language that runs natively in every web browser and, via Node.js, on servers, making it the de facto language of the web.
keywords: JavaScript, JS, ECMAScript, Node.js, npm, browser, dynamic typing, JIT, web, V8
aka: [JavaScript, JS, ECMAScript]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: event-driven, functional, object-oriented" }
  - { label: Typing, value: "Dynamic, weak (loose) typing" }
  - { label: Appeared, value: "1995 (Netscape)" }
  - { label: Designed by, value: "Brendan Eich" }
  - { label: Compilation, value: "JIT-compiled by modern engines (e.g. V8)" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Web front-ends, Node.js servers, tooling" }
see_also: [typescript-language, jit-compilation, garbage-collection, static-vs-dynamic-typing, async-programming, python-language, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Type systems", url: /learn/intro-software-dev/type-systems/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://developer.mozilla.org/en-US/docs/Web/JavaScript
  - https://en.wikipedia.org/wiki/JavaScript
---

**JavaScript** (JS) is a dynamically typed, garbage-collected programming language and
the only language that runs natively in every web browser; with Node.js it also runs on
servers, letting one language cover the whole stack.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="JavaScript runs in both the browser and on the server via Node.js, drawing on the npm ecosystem." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="170" y="54" width="120" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="75">JavaScript</text>
    <line x1="170" y1="71" x2="110" y2="40" stroke="currentColor" stroke-width="1.1"/>
    <line x1="290" y1="71" x2="350" y2="40" stroke="currentColor" stroke-width="1.1"/>
    <rect x="40" y="22" width="80" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="42">browser</text>
    <rect x="340" y="22" width="90" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="42">Node.js</text>
    <line x1="230" y1="88" x2="230" y2="110" stroke="currentColor" stroke-width="1.1"/>
    <rect x="180" y="110" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="126">npm ecosystem</text>
  </g>
</svg>
<figcaption>JavaScript runs in the browser and on the server (Node.js), sharing the vast npm ecosystem.</figcaption>
</figure>

## Overview

Modern JavaScript engines such as V8 [JIT-compile](/reference/jit-compilation/) the
language to native code at runtime, so it is far faster than its scripting origins
suggest.[^mdn] It is [dynamically typed](/reference/static-vs-dynamic-typing/) and
[garbage-collected](/reference/garbage-collection/), with an event-driven,
[asynchronous](/reference/async-programming/) model at its core that suits I/O-bound web
work. The standard is formally called ECMAScript, and the npm registry is among the
largest package ecosystems in existence.

## Strengths and trade-offs

Its defining strength is ubiquity: it is the one language guaranteed to run in every
browser, and with Node.js you can write both client and server in it. The drawbacks are
decades of quirky design decisions and famously loose, weak typing that lets subtle bugs
slip through at runtime. [TypeScript](/reference/typescript-language/) — JavaScript with
a static [type system](/reference/type-system/) layered on top — addresses the typing
problem and compiles back down to plain JavaScript, which is why it has become the
default for serious front-end and Node projects.

## Where it's used

JavaScript powers virtually all interactive web front-ends, a large share of server
backends through Node.js, and much of modern build tooling. For larger codebases teams
increasingly write [TypeScript](/reference/typescript-language/) and ship the compiled
JavaScript, but the runtime everywhere remains JavaScript itself.

## Sources

[^mdn]: [JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript) — MDN Web Docs, the reference documentation for the language and its runtime behaviour.
[^wiki]: [JavaScript](https://en.wikipedia.org/wiki/JavaScript) — Wikipedia, for history and design background.
