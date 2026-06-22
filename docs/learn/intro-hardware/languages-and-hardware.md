---
slug: languages-and-hardware
title: Programming languages & where they run
description: Servers run almost any language while a tiny chip runs C — and hardware is the reason. Learn what "runs on" really means and why constrained devices and powerful machines differ.
keywords: programming languages and hardware, what does runs on mean, compiled vs interpreted, microcontroller programming languages, c c++ rust micropython, languages by platform, bare metal vs runtime
level: beginner
status: full
faq:
  - q: "What does it mean for a language to \"run on\" a device?"
    a: "It means there's a way to turn that language into instructions the device's CPU can execute. Sometimes that's a **compiler** producing a machine-code binary ahead of time; sometimes it's an **interpreter or runtime** installed on the device that reads the code as it goes. A device can only run a language if that translation path exists for its processor and fits in its memory."
  - q: "Why do microcontrollers favor C and C++?"
    a: "Because they're tiny and have no operating system to lean on. **C** and **C++** compile straight to compact, fast machine code with no heavy runtime, so they fit in a few kilobytes and run close to the metal. **Rust** is increasingly used for the same reasons with more safety, and **MicroPython** offers an easier path where a bit more memory is available."
  - q: "Why can a server run almost any language but a microcontroller can't?"
    a: "A server has gigabytes of memory and a full operating system, so it can host the heavy interpreters and runtimes that languages like Python, JavaScript, and Java need. A microcontroller has kilobytes and no OS, so those runtimes simply don't fit. The hardware sets the ceiling on which languages are practical."
  - q: "Do I pick the language or does the hardware pick it for me?"
    a: "Both. On powerful machines you have wide freedom and choose based on the job and your preference. On constrained devices the hardware narrows the field sharply — you'll often be choosing among **C, C++, Rust, or MicroPython** because little else fits. The smaller the device, the more the hardware decides for you."
---
# Programming languages & where they run

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**"Runs on" means a translation path exists** — a compiler or an installed runtime turns code into instructions a given CPU can execute. **Hardware narrows the field** — powerful machines with an OS run almost any language; tiny chips with kilobytes run mostly C, C++, Rust, or MicroPython. **The operating system does heavy lifting** — much of a language's convenience depends on the OS underneath it.
</div>

A web host happily runs PHP and Python; a two-dollar microcontroller almost always runs C. That's not fashion or accident — it's the hardware from the last few lessons making itself felt. This lesson explains what "a language runs on a device" actually means, and why the machine you choose quietly decides which languages are even on the table. (For a proper tour of the languages themselves, the [Intro to Software Development](/learn/intro-software-dev/) path is the place to go; here we focus on the hardware angle.)

## What "runs on X" really means

A CPU only understands machine code — raw numbers specific to its kind of chip. Every programming language needs some way to get there. There are two broad routes:

- **Compiled ahead of time.** A **compiler** translates your source code into a machine-code **binary** before it runs. The binary is built for one kind of processor and runs directly on it, fast and self-contained. C, C++, Rust, and Go work this way.
- **Interpreted or run by a runtime.** Instead of producing a standalone binary, the code is read and executed by another program — an **interpreter** or **runtime** that must already be installed on the device. Python, JavaScript (via Node), and Java work this way (Java compiles to bytecode that its runtime then executes).

So "Python runs on this server" really means "the Python interpreter is installed here, and it can read your script and execute it." That's the catch for small devices: if there isn't room for the interpreter, the language can't run there at all. A compiled binary, by contrast, needs only itself — which is one reason tiny machines lean on compiled languages.

There's also a third, lowest level: **bare-metal** code that runs with no operating system at all, talking straight to the hardware. That's the world of microcontrollers, and it shapes everything about how you program them.

## Why constrained devices favor a short list

A microcontroller has kilobytes of RAM, no operating system, and a slow processor. That rules out the heavy interpreters and runtimes immediately — there's simply nowhere to put them. What's left are languages that compile to small, efficient machine code:

- **C** — the classic choice. Compact, fast, and close to the hardware, with decades of tooling. Most microcontroller code is C.
- **C++** — C with more structure; common on slightly larger embedded projects and the Arduino ecosystem.
- **Rust** — increasingly popular for embedded work: the same low-level control as C, with stronger safety guarantees.
- **MicroPython / CircuitPython** — a trimmed-down Python that fits on more capable microcontrollers (those with tens of kilobytes to spare). Much easier to write, at the cost of speed and memory.

The pattern is clear: the smaller and cheaper the device, the shorter the menu, and the closer to the metal you work. On the very smallest chips, you're essentially choosing between C, C++, and Rust.

## Why big machines run almost anything

Move up the spectrum and the constraint vanishes. A server, desktop, or laptop has gigabytes of memory and a full operating system (Linux, Windows, macOS) that manages the hardware for every program. That OS is what makes hosting big runtimes practical, so these machines run essentially the whole language landscape:

- **Python** — data, scripting, web backends, automation.
- **JavaScript / Node** — web front and back ends.
- **Go** — networked services and command-line tools (GopherTrunk's own world).
- **Java, C#** — large applications and enterprise systems.
- **C, C++, Rust** — performance-critical work, the same languages the small devices use.

Here the hardware barely limits you, so the choice comes down to the job and your preference rather than what fits. GopherTrunk is a good example: it's software-defined-radio software that can run on a server, desktop, or laptop, where there's ample room for whatever runtime and libraries it needs — a freedom that simply doesn't exist on a microcontroller.

## Mobile and the role of the operating system

Phones and tablets sit in between, and they show how much the **operating system** shapes the picture. They're powerful enough to run many languages, but their platforms steer you toward specific ones for building real apps:

- **Swift** — Apple's language for iOS and iPadOS apps.
- **Kotlin** (and Java) — the standard for Android apps.
- Cross-platform frameworks let you use other languages, but the native pair above are the defaults.

The deeper point is that the OS does enormous heavy lifting. On a server or phone, the operating system manages memory, files, networking, and lets many programs share the machine — so your language can lean on all of that. On a bare-metal microcontroller there's no OS, so the language (usually C) has to deal with the hardware directly. Much of what makes a language feel "easy" is really the operating system underneath it doing the hard parts.

## Platform to language at a glance

| Platform | Typical languages | Why |
|----------|-------------------|-----|
| **Cloud / server / VPS** | Python, JavaScript/Node, Go, Java, C#, PHP, Rust | Full OS and ample memory host any runtime |
| **Desktop / laptop** | Same as servers, plus native app languages | Powerful, OS-managed, few limits |
| **Single-board computer (Linux)** | Python, JavaScript, Go, C/C++ | A real OS on small hardware — most server languages work |
| **Phone / tablet** | Swift (iOS), Kotlin/Java (Android) | Platform-defined native toolchains |
| **Microcontroller** | C, C++, Rust, MicroPython | No OS, kilobytes of RAM — only compact, compiled (or tiny interpreted) languages fit |

Read top to bottom and the menu shrinks as the hardware shrinks. A single-board computer like a Raspberry Pi sits at an interesting crossover: it's small and cheap, but it runs a full Linux OS, so it can use the same easygoing languages as a server — which is part of why it's such a useful bridge between the two worlds.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a microcontroller has no OS and only kilobytes of memory, so it can't host the heavy runtimes that languages like Python need." markdown="0">
  <p class="knowledge-check__q">Quick check: Why does a microcontroller typically run C rather than Python?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Python hasn't been invented for small devices yet</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It has no operating system and only kilobytes of memory, so it can't host Python's heavy runtime — but it can run compact compiled C</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">C is the only programming language that exists for hardware</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **"Runs on" means a path to machine code** — either a compiler builds a self-contained binary, or an installed interpreter/runtime executes the code as it goes.
- **Compiled binaries travel light** — they need only themselves, which is why tiny devices favor compiled C, C++, and Rust.
- **Constrained devices have a short menu** — kilobytes of RAM and no OS limit microcontrollers to C, C++, Rust, or MicroPython.
- **Powerful machines run almost anything** — a full OS and gigabytes of memory let servers and PCs host any runtime, so choice comes down to the job.
- **The OS does the heavy lifting** — much of a language's convenience comes from the operating system beneath it, which bare-metal devices lack.

Next up: cost, electricity, performance, control, and effort — the handful of trade-offs that recur in every hardware decision you'll make. See [Cost, power & performance trade-offs](/learn/intro-hardware/cost-power-performance/).
