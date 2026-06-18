---
slug: what-is-software
title: What is software?
description: Software is instructions and data living in memory. Learn the hardware/software split, the stored-program idea, and how software stacks in layers.
keywords: what is software, hardware vs software, stored program, von Neumann architecture, instructions and data, firmware, operating system, software-defined radio
level: beginner
status: full
faq:
  - q: "What is the difference between hardware and software?"
    a: "**Hardware** is the physical machine — chips, wires, memory, screens — the parts you can touch. **Software** is the set of instructions that tells that hardware what to do. The same hardware can run completely different software, which is exactly why one laptop can be a word processor one minute and a radio receiver the next."
  - q: "What does \"stored-program\" mean?"
    a: "It means a computer keeps both its instructions (the program) and the data those instructions work on in the same memory. Because instructions are just numbers in memory, a program can be loaded, replaced, or even modified like any other data. This idea, often credited to the **von Neumann** design, is why we can install new apps without rewiring anything."
  - q: "Is software-defined radio really \"just software\"?"
    a: "Mostly. A small amount of hardware turns radio waves into a stream of numbers, and from there the demodulating, filtering, and decoding is done by software. Functions that used to need dedicated circuits — mixers, filters, detectors — become math running on a regular CPU, which is why a cheap dongle plus a laptop can do what once filled a rack."
---
# What is software?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Hardware vs software** — hardware is the physical machine; software is the instructions that drive it. **Stored-program idea** — instructions and data both live in memory as numbers. **Layers** — firmware, operating system, and applications stack on top of each other to make a machine usable.
</div>

This is lesson 1 of the path. Before we talk about languages, history, or design, it's worth being precise about the thing this whole field is built on: software. We use the word loosely — "the software crashed," "update your software" — but underneath it is one of the most elegant ideas of the twentieth century. By the end of this lesson you'll understand what a program actually *is*, why a single machine can do thousands of unrelated jobs, and why modern radios are increasingly made of code rather than copper.

## Hardware you can touch, software you can't

Start with the obvious split. **Hardware** is everything physical: the processor, the memory chips, the storage drive, the keyboard, the antenna plugged into the back. If you drop it and it breaks, it was hardware.

**Software** is the set of instructions that tells hardware what to do. It has no mass and no shape. You can copy it infinitely at almost no cost, send it around the world in seconds, and run the exact same software on millions of different machines.

The crucial relationship between them is this: *the hardware is general, the software is specific*. A processor doesn't "know" how to be a spreadsheet or a video game or a radio. It only knows how to do a small set of very simple operations — add two numbers, compare two numbers, move a number from here to there, jump to a different instruction. Software is what arranges millions of those tiny operations into something useful.

That generality is the whole magic trick. The same laptop can be a calculator, a music studio, and — with the right software and a small USB device — a wideband radio scanner. Nothing about the hardware changed. Only the instructions did.

## What is a program, really?

A **program** is an ordered list of instructions for the machine to carry out. The raw form those instructions take is **machine code** — numbers that the processor reads directly. Each number means something specific to that particular kind of chip: "load this value," "add," "store the result," "if zero, jump."

People almost never write those numbers by hand anymore (we'll see why in [From machine code to high-level languages](/learn/intro-software-dev/birth-of-languages/)). But it's important to know that, at the very bottom, that's all a running program is — a sequence of numeric instructions being fetched and executed, one after another, billions of times per second.

Here's a deliberately tiny example of what an instruction stream looks like in human-readable assembly form, where each line maps to one machine operation:

```text
LOAD  R1, 5      ; put the number 5 in register R1
LOAD  R2, 3      ; put the number 3 in register R2
ADD   R3, R1, R2 ; R3 = R1 + R2  (now R3 holds 8)
STORE R3, total  ; write R3 into the memory slot named "total"
```

Real programs are these same primitive steps stacked into the millions. The art of software is composing simplicity into complexity.

## The stored-program idea

Early computing machines were often wired for a single task. To make them do something else, you physically rewired them or fed in a new arrangement of switches and cables — a process that could take days.

The breakthrough was the **stored-program** concept, associated with the design John von Neumann described in 1945 (building on ideas already circulating among the ENIAC team). The insight is deceptively simple:

> Store the program in the computer's memory, in the same place and the same form as the data.

Because an instruction is just a number, and data is just numbers, there's no fundamental difference between them. The machine pulls instructions out of memory the same way it pulls out data. This is the **von Neumann architecture**, and almost every computer you've ever used follows it.

The consequences are enormous:

- **You can change software instantly.** Loading a new program is just writing new numbers into memory — no rewiring.
- **Programs can manipulate other programs.** Compilers, operating systems, and updaters all work because code is just data that other code can read and write.
- **One machine, endless purposes.** The same physical computer runs anything you load into it.

Instructions and data sharing one memory is why your computer can download a brand-new application this afternoon and run it without any physical change. The machine didn't learn a new skill — you just handed it a new list of numbers to follow.

## Software comes in layers

You rarely deal with raw machine code because software is built in **layers**, each one hiding the messiness of the layer below it. From the metal upward:

| Layer | What it is | Example |
|-------|-----------|---------|
| **Firmware** | Tiny software baked into a hardware device, telling it how to behave at the lowest level | The code inside an SSD, a Wi-Fi chip, or an SDR dongle |
| **Operating system** | Manages the hardware and shares it among programs | Linux, Windows, macOS, Android |
| **Libraries / runtime** | Reusable building blocks programs lean on | A DSP filtering library, a graphics toolkit |
| **Applications** | The programs you actually open and use | A browser, a game, GopherTrunk |

Each layer trusts the one beneath it and serves the one above it. When you click "play" in an app, that request travels down through libraries, the operating system, firmware, and finally the hardware — and the result travels back up. You see a single smooth action; underneath, every layer did its job.

This layering is also how a complete beginner can build something real. You don't have to understand transistors to write an app, because dozens of layers of other people's software stand between your code and the silicon.

## When the hardware moves into the software

Here's the idea this whole site circles back to. For most of radio's history, a receiver was *hardware*: physical tuners, mixers, and filters made of coils and capacitors, each soldered for one job. Want a different band or mode? Different hardware.

**Software-defined radio (SDR)** flips that. A minimal piece of hardware grabs the radio signal and turns it into a stream of numbers (we'll meet the [radio waves](/learn/rf-sdr/radio-waves/) themselves in the RF path). Everything after that — tuning, filtering, demodulating, decoding — becomes *software*: math running on an ordinary CPU. The filter that used to be a physical coil is now a few lines of code processing those numbers.

SDR is the purest expression of the lesson here: *if you can describe a job as instructions operating on data, you can move it out of hardware and into software*. That's why a thirty-dollar USB dongle plus a laptop can now scan, decode, and analyze signals that once demanded a rack of specialized equipment. The radio became software. GopherTrunk lives in exactly this world.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in a stored-program computer, instructions and data both live in memory as numbers." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the core idea of the stored-program (von Neumann) architecture?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Software must be permanently wired into the hardware</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A program's instructions and its data both live in the same memory, as numbers</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Each program needs its own dedicated processor</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Hardware vs software** — hardware is the physical, general-purpose machine; software is the specific instructions that drive it.
- **A program is instructions** — at bottom, a running program is a stream of simple numeric operations the processor executes one after another.
- **Stored-program idea** — instructions and data share one memory as numbers, so software can be loaded, replaced, and even written by other software.
- **One machine, many jobs** — that shared memory is why a single computer can run any program you give it without physical change.
- **Layers** — firmware, operating system, libraries, and applications stack up, each hiding the complexity below.
- **SDR** — software-defined radio shows the endpoint of "move it into software": the receiver becomes math on a CPU.

Next up: how the physical machine got here — from Jacquard looms and Babbage's engines through the transistor, the microprocessor, and the cheap chips that made SDR possible. See [From looms to microprocessors](/learn/intro-software-dev/computing-hardware-story/).
