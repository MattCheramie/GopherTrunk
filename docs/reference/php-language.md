---
slug: php-language
title: PHP
entry_type: language
category: programming-languages
description: PHP is a dynamic, interpreted scripting language built for the web back end, ubiquitous on shared hosting, historically inconsistent but substantially modernized in versions 7 and 8.
keywords: PHP, web back end, server-side scripting, dynamic typing, interpreted, WordPress, Laravel, Zend Engine, JIT
aka: [PHP]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, object-oriented, scripting" }
  - { label: Typing, value: "Dynamic, weak (optional type hints)" }
  - { label: Appeared, value: "1995 (Rasmus Lerdorf)" }
  - { label: Designed by, value: "Rasmus Lerdorf; later a community/Zend effort" }
  - { label: Compilation, value: "Interpreted (bytecode + JIT since PHP 8)" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Web back ends, WordPress, CMSes" }
see_also: [javascript-language, interpreter, static-vs-dynamic-typing, garbage-collection, jit-compilation, bytecode, rest]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.php.net/
  - https://en.wikipedia.org/wiki/PHP
---

**PHP** is a dynamic, [interpreted](/reference/interpreter/) scripting language built
for the web back end.[^wiki] It is one of the most widely deployed languages on the
internet, historically criticised for inconsistency, but substantially modernised in
versions 7 and 8.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A browser request hits a web server running PHP, which generates an HTML page and sends it back." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="70" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">browser</text>
    <line x1="90" y1="62" x2="160" y2="62" stroke="currentColor" stroke-width="1.1"/><text x="125" y="55" font-size="8">request</text>
    <rect x="160" y="44" width="120" height="52" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="220" y="66">server runs</text><text x="220" y="80">PHP</text>
    <line x1="160" y1="82" x2="90" y2="82" stroke="currentColor" stroke-width="1.1"/><text x="125" y="100" font-size="8">HTML page</text>
    <line x1="280" y1="70" x2="340" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="340" y="50" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="390" y="74">database</text>
  </g>
</svg>
<figcaption>PHP runs on the web server, builds an HTML response per request, and hands it back to the browser.</figcaption>
</figure>

## Overview

PHP was designed to be embedded directly in web pages, and that origin shaped its
ecosystem: it ships with nearly every shared web host, which made it the default for
a huge share of the web.[^home] WordPress, much of the world's content-management software,
and major sites are built on it, and modern frameworks like Laravel and Symfony bring
disciplined structure. It serves dynamic pages and [REST](/reference/rest/) APIs,
typically run per request by the server.

## Key characteristics

PHP is dynamically and weakly typed (see [static vs dynamic typing](/reference/static-vs-dynamic-typing/)),
[garbage-collected](/reference/garbage-collection/), and compiles to
[bytecode](/reference/bytecode/) that an engine executes — with an optional
[JIT](/reference/jit-compilation/) added in PHP 8.[^home] Its reputation for **inconsistency**
is earned: the standard library has irregular function names and argument orders, and
older code mixed logic with HTML freely. Versions 7 and 8 changed the picture
considerably, adding real performance gains, optional type declarations, and cleaner
language features, though its weak typing and accumulated history still draw criticism.
Like [JavaScript](/reference/javascript-language/), it is hard to avoid on the web.

## Sources

[^home]: [PHP: Hypertext Preprocessor](https://www.php.net/) — official site, documentation, and the engine, bytecode, and JIT details.
[^wiki]: [PHP](https://en.wikipedia.org/wiki/PHP) — Wikipedia, for history, design background, and web ubiquity.
