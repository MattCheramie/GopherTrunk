---
slug: ruby-language
title: Ruby
entry_type: language
category: programming-languages
description: Ruby is a dynamic, object-oriented, interpreted language designed for programmer happiness and readability, best known for the Ruby on Rails web framework.
keywords: Ruby, Ruby on Rails, dynamic typing, object-oriented, interpreted, scripting, Matz, developer happiness, metaprogramming
aka: [Ruby]
autolink: true
infobox:
  - { label: Paradigm, value: "Object-oriented, dynamic, scripting" }
  - { label: Typing, value: "Dynamic, strong, duck-typed" }
  - { label: Appeared, value: "1995 (Japan)" }
  - { label: Designed by, value: "Yukihiro \"Matz\" Matsumoto" }
  - { label: Compilation, value: "Interpreted (bytecode VM, JIT in recent versions)" }
  - { label: Memory, value: "Garbage-collected" }
  - { label: Notable uses, value: "Web back ends (Rails), scripting, automation" }
see_also: [python-language, object-oriented-programming, interpreter, static-vs-dynamic-typing, garbage-collection, bytecode, package-manager]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://www.ruby-lang.org/
  - https://en.wikipedia.org/wiki/Ruby_(programming_language)
---

**Ruby** is a dynamic, [object-oriented](/reference/object-oriented-programming/),
[interpreted](/reference/interpreter/) language created by Yukihiro "Matz" Matsumoto
with an explicit goal of programmer happiness, and it is best known as the language
behind the Ruby on Rails web framework.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Concise Ruby source runs through an interpreter, which produces a working web application." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="68">readable</text><text x="60" y="80">.rb code</text>
    <line x1="100" y1="70" x2="160" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="130" y="62" font-size="8">run</text>
    <rect x="160" y="50" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="210" y="74">interpreter</text>
    <line x1="260" y1="70" x2="320" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="320" y="50" width="120" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="380" y="68">web app</text><text x="380" y="80">(Rails)</text>
  </g>
</svg>
<figcaption>Ruby code is read and executed by an interpreter; Rails is the framework that made it famous.</figcaption>
</figure>

## Overview

Ruby prizes readability and expressiveness over raw speed. Its syntax reads almost
like English, nearly everything is an object, and its flexible metaprogramming lets
libraries build clean, domain-specific interfaces.[^home] That power is what made **Ruby on
Rails** so influential: the framework's "convention over configuration" approach let
small teams ship web applications fast, and it shaped a generation of web frameworks
in other languages. Gems, installed through the RubyGems [package manager](/reference/package-manager/),
provide a large library ecosystem.

## Key characteristics

Ruby is dynamically typed (see [static vs dynamic typing](/reference/static-vs-dynamic-typing/))
and [garbage-collected](/reference/garbage-collection/), with code run by a
[bytecode](/reference/bytecode/) virtual machine that has gained a JIT in recent
releases.[^home] The trade-offs are well known: like other dynamic languages such as
[Python](/reference/python-language/), it is comparatively **slow** in CPU-bound work
and its loose typing lets some bugs reach runtime. Its popularity has cooled from the
Rails peak, and its centre of gravity stays in web back ends, scripting, and
automation rather than systems or numeric work.

## Sources

[^home]: [Ruby programming language](https://www.ruby-lang.org/) — official site, documentation, and the dynamic object model and runtime.
[^wiki]: [Ruby (programming language)](https://en.wikipedia.org/wiki/Ruby_(programming_language)) — Wikipedia, for history, Matz's design goals, and Rails.
