---
slug: compiled-vs-interpreted
title: Compiled, interpreted & JIT
description: How source code becomes something a machine runs — ahead-of-time compilation, interpreters and bytecode VMs, and just-in-time compilation, with their real trade-offs.
keywords: compiled language, interpreted language, JIT compilation, bytecode, virtual machine, ahead of time compilation, language performance, static binary
level: beginner
status: full
faq:
  - q: "What is the difference between a compiled and an interpreted language?"
    a: "A **compiled** language is translated to machine code ahead of time (C, Rust, Go), producing a binary the CPU runs directly. An **interpreted** language is read and executed at run time by another program, the interpreter (classic Python, Ruby). The line is fuzzy — many \"interpreted\" languages compile to bytecode first."
  - q: "What does JIT compilation do?"
    a: "**Just-in-time (JIT)** compilation translates code to machine code *while the program runs*, often optimising the hot paths it observes. The JVM, JavaScript's V8 and .NET use it to combine portability with near-native speed, at the cost of slower startup and higher memory use."
  - q: "Why do compiled languages suit real-time radio?"
    a: "Real-time DSP needs **predictable, low-latency performance** with no surprise pauses. Ahead-of-time compiled languages like C, Rust and Go produce native code with no interpreter overhead and tighter control over timing, so samples are processed before the next buffer arrives."
---
# Compiled, interpreted & JIT

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Compiled** — translated to machine code before you ship it. **Interpreted** — executed by another program at run time. **JIT** — compiles hot paths on the fly for portability *and* speed.
</div>

A CPU only understands **machine code** — raw numeric instructions for that exact
processor. The source code you write is for humans. The bridge between the two is
where languages differ most, and the choice shapes how fast your program runs,
how quickly it starts, how easily it ships to other machines, and how portable it
is. There are three broad strategies — **ahead-of-time compilation**,
**interpretation**, and **just-in-time compilation** — and most real systems blur
the lines between them.

## Ahead-of-time compilation

A **compiler** translates your entire source program into machine code *before*
it ever runs, producing an executable binary. This is **ahead-of-time (AOT)**
compilation, used by **C, C++, Rust and Go**.

```text
source.go  ──►  compiler  ──►  native binary  ──►  CPU runs it directly
            (once, on your machine)        (every time, no translation)
```

The payoff:

- **Speed** — the CPU runs native instructions with no translation layer in the
  way; compilers also optimise aggressively (inlining, vectorising, eliminating
  dead code).
- **Predictable performance** — what you compiled is what runs, every time.
- **No runtime dependency on a toolchain** — the user does not need a compiler or
  interpreter installed.

The costs are a **compile step** between editing and running (slower iteration on
huge codebases) and **platform-specific binaries** — a binary built for x86-64
Linux will not run on an ARM Mac without recompiling.

## Interpreters and bytecode VMs

An **interpreter** reads your source and executes it directly, statement by
statement, every time the program runs. **Python and Ruby** are the canonical
examples. In practice most modern "interpreted" languages do a half-step first:
they compile source to **bytecode** — a compact, portable instruction set for a
**virtual machine (VM)** — and the VM interprets that bytecode. CPython compiles
`.py` files to `.pyc` bytecode and runs it on the Python VM.

The trade-offs invert the compiled story:

- **Slower execution** — the VM adds overhead interpreting each operation.
- **Fast iteration and flexibility** — no separate build step; you can change code
  and re-run instantly, and the language can do dynamic things at run time.
- **Portability** — the *same* bytecode runs anywhere the VM is installed; you ship
  source or bytecode, not a per-platform binary. But the user must have the
  interpreter installed.

## Just-in-time compilation

**Just-in-time (JIT)** compilation is the hybrid that tries to get both speed and
portability. The program ships as bytecode, but as it runs the **JIT compiler**
translates frequently-executed ("hot") sections into native machine code on the
fly — and can optimise based on what it actually observes happening.

- The **JVM** (Java, Kotlin, Scala) compiles bytecode to native code via its HotSpot
  JIT.
- **V8** powers JavaScript in Chrome and Node.js with aggressive JIT optimisation.
- **.NET** JIT-compiles its intermediate language (IL).

The result is often near-native steady-state speed. The costs are **slower
startup** (the JIT has to warm up before hot paths are optimised) and **higher
memory use** (it holds both bytecode and generated native code). This is why a
long-running server loves a JIT but a short command-line tool may not — it exits
before warm-up pays off.

| Approach | Examples | Speed | Startup | Distribution |
|----------|----------|-------|---------|--------------|
| AOT compiled | C, Rust, Go | fastest, predictable | instant | per-platform binary |
| Interpreted / bytecode VM | Python, Ruby | slowest | fast | source + interpreter |
| JIT | JVM, V8, .NET | near-native after warm-up | slow warm-up | bytecode + VM |

<div class="knowledge-check" data-quiz data-correct-msg="Right — a JIT compiles hot paths to native code while the program runs, trading slower startup for higher steady-state speed." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a just-in-time (JIT) compiler do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Translates all source to a native binary before shipping</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Compiles frequently-run code to native machine code while the program is running</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Interprets source line by line without ever producing machine code</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## The trade-off, summarised

No approach is "best" — each optimises different things:

- **Raw, predictable speed:** AOT compiled wins. The work is done up front and never
  repeated.
- **Developer iteration speed and dynamism:** interpreters win. Edit, run, repeat.
- **Long-running throughput with portability:** JIT wins, once warmed up.
- **Startup latency:** AOT is instant; JIT is worst.
- **Distribution simplicity:** AOT can ship a single self-contained file; interpreted
  and JIT generally require a runtime on the target machine.

## The line is blurrier than it looks

It is tempting to file each language under one heading, but the boundaries are
soft, and modern toolchains deliberately blur them:

- **CPython compiles before it interprets.** The "interpreted" label hides a real
  compile step to bytecode; the interpreter runs the bytecode, not your source
  text directly.
- **Java is compiled *and* JIT-compiled.** Source compiles ahead of time to
  bytecode, which the JVM then JIT-compiles to native code as it runs — two
  translation stages.
- **Many compiled languages ship an interpreter too.** Rust and Go have fast
  build-and-run modes for quick iteration; some C++ environments offer interpreters
  for experimentation.
- **AOT for traditionally-JIT languages.** GraalVM can compile Java ahead of time
  to a native image for instant startup, trading away some peak throughput.

The useful mental model is therefore not "which box is this language in" but
"where does translation to machine code happen — before run time, during it, or
never (the interpreter does it on the fly)?" That question, plus how much the
runtime optimises, predicts the speed, startup and distribution characteristics
better than any single label. When you read that a language is "fast" or "slow",
ask *which* of these stages it is doing and *when*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 476 214" role="img" aria-label="Three translation paths compared across a build-time and run-time divider: ahead-of-time compilation turns source to a native binary before running; a bytecode VM translates on the fly every run; and a JIT ships bytecode but compiles hot paths to native while the program runs." xmlns="http://www.w3.org/2000/svg">
  <line x1="300" y1="36" x2="300" y2="206" stroke="currentColor" stroke-width="1.1" stroke-dasharray="5 4" stroke-opacity="0.6"/>
  <text x="176" y="30" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">before run · build time</text>
  <text x="388" y="30" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">during run · run time</text>
  <g text-anchor="middle" fill="currentColor" font-size="7">
    <text x="30" y="62" font-size="8.5" font-weight="600">AOT</text><text x="30" y="72">C·Go·Rust</text>
    <text x="30" y="122" font-size="8.5" font-weight="600">VM</text><text x="30" y="132">Python</text>
    <text x="30" y="182" font-size="8.5" font-weight="600">JIT</text><text x="30" y="192">JVM·V8</text>
  </g>
  <g text-anchor="middle" font-size="8" fill="currentColor">
    <rect x="68" y="44" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="91" y="63">source</text>
    <rect x="124" y="44" width="60" height="30" rx="4" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.3"/><text x="154" y="60" font-weight="600">compiler</text><text x="154" y="70" font-size="6.5">translate</text>
    <rect x="196" y="44" width="66" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="229" y="60">native</text><text x="229" y="70" font-size="7">binary</text>
    <rect x="410" y="44" width="52" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="436" y="63">CPU</text>
    <rect x="68" y="104" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="91" y="123">source</text>
    <rect x="124" y="104" width="60" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="154" y="123">bytecode</text>
    <rect x="318" y="104" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.3"/><text x="378" y="120" font-weight="600">VM interprets</text><text x="378" y="130" font-size="6.5">translate each run</text>
    <rect x="68" y="164" width="56" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="96" y="183">bytecode</text>
    <rect x="318" y="164" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.3"/><text x="378" y="180" font-weight="600">VM · JIT</text><text x="378" y="190" font-size="6.5">hot paths to native, mid-run</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="114" y1="59" x2="124" y2="59" marker-end="url(#ci_ar)"/>
    <line x1="184" y1="59" x2="196" y2="59" marker-end="url(#ci_ar)"/>
    <line x1="262" y1="59" x2="410" y2="59" marker-end="url(#ci_ar)"/>
    <line x1="114" y1="119" x2="124" y2="119" marker-end="url(#ci_ar)"/>
    <line x1="184" y1="119" x2="318" y2="119" marker-end="url(#ci_ar)"/>
    <line x1="124" y1="179" x2="318" y2="179" marker-end="url(#ci_ar)"/>
  </g>
  <defs><marker id="ci_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The useful question isn't which box a language sits in but <strong>where translation to machine code happens</strong>. An ahead-of-time compiler does it once at build time, so the CPU later runs native code directly. A bytecode VM translates on the fly, every run. A JIT ships bytecode but compiles the hot paths to native while the program runs — portability up front, near-native speed once warmed up.</figcaption>
</figure>

## Why this matters for GopherTrunk

Two concerns from radio software make the choice concrete.

First, **real-time DSP** demands predictable, low-latency performance. A
software-defined radio produces samples continuously, and each buffer must be
filtered and demodulated before the next one arrives. An interpreter's per-operation
overhead, or a JIT pausing to recompile, can blow the timing budget and **drop
samples**. Ahead-of-time compiled languages — C, Rust, Go — give native speed and
tighter timing control, which is why DSP cores are almost always compiled. (The
related worry of garbage-collection pauses is covered in
[memory management](/learn/intro-software-dev/memory-management/).)

Second, **distribution**. Because Go compiles ahead of time and can produce a
**single statically-linked binary** with no external runtime, you can build one
file and hand it to a user on Linux, macOS or Windows — no interpreter, no
dependency hunt. That is a real advantage for shipping a tool like GopherTrunk to
people who just want to run it, a theme we pick up in
[packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/).

## Recap

- **A CPU only runs machine code** — the strategy for getting there is what separates
  languages.
- **AOT compiled** (C, Rust, Go) — fast, predictable, instant startup; ships as a
  per-platform binary.
- **Interpreted / bytecode VM** (Python, Ruby) — slower but flexible with fast
  iteration; needs the interpreter installed.
- **JIT** (JVM, V8, .NET) — bytecode plus on-the-fly native compilation; near-native
  speed after a slow warm-up.
- **Real-time radio wants compiled** — predictable timing avoids dropped samples.
- **One static Go binary is easy to ship** — no runtime to install on the target
  machine.

Next up: how languages catch mistakes before they ever run — [type systems and
safety](/learn/intro-software-dev/type-systems/).
