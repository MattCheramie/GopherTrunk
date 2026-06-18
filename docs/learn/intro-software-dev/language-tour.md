---
slug: language-tour
title: A tour of today's major languages
description: A fair, even-handed survey of the major programming languages — C, C++, Rust, Go, Python, JavaScript/TypeScript, Java/C# and more — with honest strengths, drawbacks and typical uses.
keywords: programming languages compared, C, C++, Rust, Go, Python, JavaScript, TypeScript, Java, C#, Swift, Kotlin, language comparison table
level: intermediate
status: full
faq:
  - q: "Which programming language should a beginner learn first?"
    a: "There's no single answer, but Python is a common first language because it's readable and forgiving, and JavaScript is hard to avoid if you want to build for the web. The skills transfer: once you understand variables, loops, functions and types in one language, picking up the next is far easier."
  - q: "Is Rust replacing C and C++?"
    a: "Not replacing, but increasingly competing for new systems work. Rust offers memory safety without a garbage collector, which addresses a major source of C and C++ bugs. But C and C++ have decades of existing code, tooling and trained engineers, and a steep Rust learning curve slows adoption. Expect coexistence, not replacement."
  - q: "What is the difference between Java and C#?"
    a: "They're close cousins — both statically typed, garbage-collected, JIT-compiled managed languages with large ecosystems aimed at big applications. Java runs on the JVM and dominates enterprise and Android backends; C# runs on .NET and is strong in Windows, game development (Unity) and enterprise. The choice is often ecosystem and platform, not language merit."
---
# A tour of today's major languages

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Every language is a trade-off** — strengths come bundled with real drawbacks. **The big split is control vs convenience** — systems languages vs managed/dynamic ones. **Pick by fit, not fandom** — match the tool to the job, team and platform.
</div>

There are hundreds of programming languages, but you'll choose between a couple
dozen, and realistically a handful for any given project. This is a fair survey of
the major ones — what each is genuinely good at, what it genuinely costs you, and
where it tends to be used. No fanboyism: every language here is loved and cursed by
the people who use it daily, and every one has real drawbacks, including the ones
this site is built in. Use the table at the end as a cheat sheet, and the
[language families](/learn/intro-software-dev/language-families/) lesson for the
deeper "why they differ."

## The systems languages: control and speed

These compile to native code and give you direct control over memory and hardware.
You pay for that control with complexity or risk.

**C** — the bedrock. Almost every operating system, language runtime and embedded
device has C at its core. It's small, fast, portable and *everywhere*. The cost is
**safety**: manual memory management means buffer overflows, use-after-free and
dangling pointers — whole categories of security bugs that C does nothing to stop.
It's still the default for OS kernels, drivers and tight embedded work.

**C++** — C plus enormous power: objects, templates, generics, RAII, a huge standard
library. It can be as fast as C while offering far higher-level abstractions, which
is why it dominates game engines, high-performance trading, browsers and serious DSP.
The drawback is **complexity** — C++ is vast, with decades of accumulated features
and sharp edges; it inherits C's memory-safety risks unless you're disciplined, and
it's genuinely hard to master.

**Rust** — the modern answer to "fast *and* safe." Rust's borrow checker enforces
memory safety at **compile time** with no garbage collector, eliminating data races
and use-after-free *without* a runtime cost. Performance rivals C and C++. The price
is a **steep learning curve** — the borrow checker is famously hard at first — and a
younger (though fast-growing) ecosystem. Increasingly chosen for new systems work,
CLIs, WebAssembly and safety-critical components.

**Go** — built at Google for simplicity at scale. Go is compiled and fast, with
**first-class concurrency** (goroutines and channels), **very fast compiles**, and a
killer deployment story: a **single static binary** you can drop on any machine.
It's deliberately small and easy to learn. The trade-offs are real: a **garbage
collector** (so brief pauses, and less control than C/Rust), **less raw throughput**
than C/C++/Rust in the tightest numeric loops, and a minimalist design some find
limiting — generics only arrived in 2022, and error handling is verbose. Excellent
for network services, CLIs, infrastructure and concurrent stream processing.

## The productivity languages: speed of writing

These trade run-time speed for how fast a human can write and change code.

**Python** — prized for readability and a vast ecosystem. It's a joy to write and
the default for **data science, machine learning, scientific computing, scripting
and automation**, largely thanks to libraries like NumPy, pandas and PyTorch. The
costs: it's **slow** in pure-Python loops, and the **GIL** (global interpreter lock)
limits true multithreaded CPU parallelism in the standard interpreter. You work
around both by delegating heavy work to native libraries (see
[performance vs productivity](/learn/intro-software-dev/performance-vs-productivity/)).

**JavaScript** — the language of the web, and the *only* one that runs natively in
every browser. With Node.js it runs servers too, so you can use one language across
the whole stack. The ecosystem (npm) is enormous. The drawbacks are decades of
quirky design decisions and famously loose dynamic typing that lets subtle bugs
through. **TypeScript** — JavaScript with a static type layer — addresses the typing
problem and has become the default for serious front-end and Node work; it compiles
away to plain JavaScript. See [type systems](/learn/intro-software-dev/type-systems/).

## The managed enterprise languages

**Java** — statically typed, garbage-collected, JIT-compiled on the JVM. Its
strengths are a **massive, mature ecosystem**, strong tooling, real cross-platform
portability ("write once, run anywhere") and battle-tested reliability at scale. It
runs enormous enterprise systems and Android backends. Critics find it **verbose**
and ceremony-heavy, with slower startup and higher memory use than native code,
though modern Java has modernised considerably.

**C#** — Microsoft's JVM-equivalent on .NET. Very similar trade-offs to Java —
managed, typed, big ecosystem — with arguably more modern language features. It's
the backbone of Windows enterprise software and, via **Unity**, much of the game
industry. Historically Windows-centric, though .NET is now genuinely cross-platform.

## Specialist and mobile languages

A few you'll meet in specific domains:

- **Swift** — Apple's modern, safe language for iOS and macOS apps. Clean and fast,
  but its centre of gravity is Apple's platforms.
- **Kotlin** — a more concise, safer language on the JVM; Google's preferred language
  for Android, and usable anywhere Java is.
- **Julia** — designed for high-performance numerical and scientific computing, aiming
  to be as fast as C while as easy as Python. Smaller ecosystem.
- **R** — a statistics-and-data specialist, beloved by statisticians and researchers;
  excellent for analysis and plotting, less suited to general programming.
- **MATLAB** — a commercial powerhouse for engineering, signal processing and control
  systems; superb for DSP prototyping, but proprietary and licensed.

## The comparison at a glance

| Language | Strengths | Drawbacks | Typical use |
|---|---|---|---|
| **C** | tiny, fast, ubiquitous, portable | memory-unsafe, low-level, manual | OS, drivers, embedded |
| **C++** | fast + powerful abstractions | complex, sharp edges, unsafe by default | engines, DSP, browsers |
| **Rust** | memory-safe *and* fast, no GC | steep learning curve, younger ecosystem | systems, CLIs, WASM |
| **Go** | simple, great concurrency, fast compiles, single binary | GC pauses, less raw speed than C/Rust, minimalist | services, CLIs, infra, stream engines |
| **Python** | readable, huge data/ML ecosystem | slow loops, GIL limits CPU threads | data science, ML, scripting |
| **JS / TS** | runs in every browser, full-stack, huge npm; TS adds types | JS quirks, loose typing (TS mitigates) | web front-end, Node backends |
| **Java** | mature, portable, scalable, strong tooling | verbose, heavier startup/memory | enterprise, Android backends |
| **C#** | modern, big ecosystem, Unity | historically Windows-leaning | enterprise, games |
| **Swift / Kotlin** | safe, modern mobile languages | tied to Apple / JVM-Android | iOS / Android apps |
| **Julia / R / MATLAB** | strong numeric/DSP/stats | niche, smaller (or licensed) ecosystems | science, signal processing |

<div class="knowledge-check" data-quiz data-correct-msg="Right — Rust enforces memory safety at compile time with its borrow checker and no garbage collector, at the cost of a steep learning curve." markdown="0">
  <p class="knowledge-check__q">Quick check: what is Rust's defining strength?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's the easiest language to learn for total beginners</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Memory safety without a garbage collector, enforced at compile time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs natively inside every web browser</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Reading the table fairly

Two cautions before you use this as a scorecard. First, **drawbacks are contextual** —
Go's garbage collector is a non-issue for a web service and a real consideration for a
hard real-time DSP loop; Python's speed is irrelevant for a script and decisive for an
inner loop. A "con" only counts if it touches *your* requirements. Second, **ecosystem
and team often outweigh raw language merit** — the "best" language with no libraries and
no one to write it loses to a good-enough one with both. The next lesson turns this
survey into something more actionable: which languages each *domain* actually reaches
for, and why.

## Recap

- **Systems languages** (C, C++, Rust, Go) — control and speed; C is unsafe, C++ is
  complex, Rust is safe-but-steep, Go is simple-but-GC'd.
- **Productivity languages** (Python, JS/TS) — fast to write; Python is slow with a
  GIL, JS is quirky, TypeScript adds the types JS lacks.
- **Managed enterprise** (Java, C#) — portable, scalable, big ecosystems; verbose and
  heavier than native code.
- **Specialists** — Swift/Kotlin for mobile, Julia/R/MATLAB for numeric and DSP work.
- **Judge in context** — every drawback is conditional, and ecosystem plus team
  frequently outweigh raw language differences.

Next up: which of these each field actually picks, and the reasons behind it —
[matching language to domain](/learn/intro-software-dev/language-for-the-domain/).
