---
slug: why-go
title: Why Go?
description: Where the Go programming language came from, the problems it was built to solve, and why a fast, simple, statically compiled language is a great fit for systems software like GopherTrunk.
keywords: why go, what is golang, go language history, why use go, go vs python, compiled language, go concurrency
level: beginner
status: full
faq:
  - q: Is Go hard to learn?
    a: Go is deliberately one of the smallest mainstream languages. It has about 25 keywords, one loop construct, one way to format code, and no inheritance or exceptions to learn. Most people can read Go within a day and write useful programs within a week — the difficulty is less the language and more the ideas behind it, like concurrency, which this module builds up gradually.
  - q: What is Go used for?
    a: Go is used heavily for servers, networking tools, command-line programs, cloud infrastructure (Docker, Kubernetes, and much of the cloud is written in Go), and performance-sensitive systems software. GopherTrunk uses it for a real-time software-defined-radio engine, where fast concurrent processing of sample streams matters.
  - q: Is Go faster than Python?
    a: For CPU-bound work, yes — usually by a large margin, because Go compiles to native machine code while CPython interprets bytecode. Go also handles many simultaneous tasks efficiently through goroutines. Python is often quicker to prototype in; Go trades a little of that convenience for speed, static typing, and a single deployable binary.
---

# Why Go?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Go** (or "Golang") is a **statically typed, compiled** language built at Google
to make large systems software simple to write and fast to run. Its signature
strengths are **fast compilation**, a **single static binary** as output, built-in
**concurrency** via goroutines, and a deliberately **small** feature set. Those
same traits are why GopherTrunk — a real-time SDR engine — is written in Go.
</div>

This is lesson 1, and its only job is to give you the *why* before the *how*. By
the end you will understand what kind of language Go is, the trade-offs it makes,
and why it shows up in so much of the software running the modern internet.

## Where did Go come from?

Go was created at Google around 2007 and released publicly in 2009. Its designers
— including some of the people behind Unix and C — were frustrated by the same
things every day: builds that took minutes, giant codebases that were hard to
read, and languages that made concurrency painful. Go was their answer: keep the
speed of a compiled language like C, but make it as pleasant to work with as a
scripting language, and make concurrency a first-class part of the language rather
than a library bolted on later.

## What kind of language is Go?

Three words describe most of what matters:

| Trait | What it means | Why it helps |
|-------|---------------|--------------|
| **Compiled** | Your source is turned into native machine code ahead of time | Programs run fast and ship as one file with no interpreter needed |
| **Statically typed** | Every value's type is known at compile time | Whole classes of bugs are caught before the program ever runs |
| **Garbage-collected** | Memory is reclaimed automatically | You get C-like speed without manually freeing memory |

Unlike Python or JavaScript, there is no interpreter shipping alongside your code —
`go build` produces a **single self-contained binary** you can copy to a server and
run. Unlike C or C++, you rarely think about memory management, and builds finish
in seconds.

## Why is Go a good fit for GopherTrunk?

A software-defined-radio scanner has to do a lot at once: pull samples off the
radio, filter and demodulate them, follow control-channel messages, and decode
voice — all in real time, all concurrently. Go fits that shape well:

- **Concurrency is built in.** Handling dozens of channels at once is natural with
  goroutines (Unit 3), instead of wrestling with threads and locks.
- **It is fast enough for DSP.** Native compiled code keeps up with high sample
  rates where an interpreted language would fall behind.
- **It deploys as one binary.** A user can download a single file and run a scanner
  — no runtime to install (you will see how in the [Deployment module](/learn/deployment/)).

## What does Go give up?

Every language trades something. Go deliberately leaves out features other
languages have — there is no class inheritance, no exceptions, and (until
recently) no generics. That can feel spartan if you are used to a big language.
The pay-off is that Go code tends to look the same everywhere, so you can drop into
an unfamiliar codebase and read it. That readability is a feature, not an accident.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a single native binary is one of Go's headline benefits." markdown="0">
  <p class="knowledge-check__q">Quick check: when you run <code>go build</code> on a Go program, what do you get?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A folder of scripts that need an interpreter</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A single self-contained native binary</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Bytecode that runs on a virtual machine</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Go is a **compiled, statically typed, garbage-collected** language built for
  simple, fast systems software.
- Its strengths are **fast builds**, a **single static binary**, **built-in
  concurrency**, and a **small, readable** feature set.
- Those strengths are exactly why GopherTrunk — a real-time SDR engine — is written
  in Go.
- Go trades away big-language features (inheritance, exceptions) for consistency
  and readability.

Next up: install Go and write your very first program.
