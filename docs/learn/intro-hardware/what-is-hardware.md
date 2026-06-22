---
slug: what-is-hardware
title: What is computer hardware?
description: Computer hardware is the physical machine that runs software. Learn the hardware/software split, what actually counts as a computer, and why the same job can run on a server, a laptop, or a chip the size of a stamp.
keywords: what is computer hardware, hardware vs software, what is a computer, types of computer hardware, embedded computer, general purpose computer, hardware for developers
level: beginner
status: full
faq:
  - q: "What is the difference between hardware and software?"
    a: "**Hardware** is the physical machine — the chips, boards, wires, and screens you can touch. **Software** is the set of instructions that tells that hardware what to do. The same hardware can run completely different software, and the same software can often run on very different hardware, which is exactly why this whole path is about choosing the right physical machine for a job."
  - q: "Is a smartphone or a microcontroller really a computer?"
    a: "Yes. Anything with a processor, memory, and the ability to run instructions is a computer. A data-center server, a laptop, a phone, a Raspberry Pi, and a two-dollar microcontroller are all computers — they differ in scale, not in kind. They all fetch instructions and data from memory and execute them, just at wildly different sizes, speeds, and prices."
  - q: "Why does the hardware I choose matter if my code is the same?"
    a: "Because hardware sets the limits your code lives inside: how fast it runs, how much memory and storage it has, whether it can reach the internet, how much power it draws, and what it costs to run. The same program can be trivial on a server and impossible on a microcontroller. Choosing hardware well is choosing the constraints you'll build within."
---
# What is computer hardware?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Hardware is the physical machine; software is the instructions that drive it.** **A computer is anything with a processor, memory, and the ability to run instructions** — a server, a laptop, a phone, and a microcontroller are all computers, differing in scale, not in kind. **The hardware you choose sets the limits** your software lives inside: speed, memory, connectivity, power, and cost.
</div>

This is lesson 1 of the path. Before we tour servers, laptops, phones, and tiny chips, it's worth being precise about the word that ties them all together: hardware. Every program you've ever used was running on some physical machine — and the range of machines that can run software is far wider than most people realize. By the end of this lesson you'll have a clear definition of hardware, a working idea of what counts as a "computer," and a feel for why the machine you pick shapes everything you build on top of it.

## Hardware you can touch, software you can't

Start with the fundamental split. **Hardware** is everything physical: the processor that does the calculating, the memory chips that hold data, the storage that keeps it when the power is off, the board that wires them together, and the ports, screens, and sensors bolted on around them. If you can drop it and crack it, it's hardware.

**Software** is the set of instructions that tells that hardware what to do. It has no mass and no shape — you can copy it infinitely and send it around the world in seconds, and the same software can often run on millions of different machines.

The relationship between them is the key idea: *the hardware is general, the software is specific*. A processor doesn't "know" how to be a web server or a thermostat or a radio scanner. It only knows how to do a small set of simple operations — add two numbers, compare them, move data, jump to another instruction. Software is what arranges billions of those tiny steps into something useful. (If you want the software side of this story in depth, the [Intro to Software Development](/learn/intro-software-dev/) path picks it up there; here we stay focused on the machine.)

That generality is why this path exists. Because the same software can run on radically different hardware, you get a *choice* — and that choice has real consequences for cost, speed, and what's even possible.

## What actually counts as a computer?

People hear "computer" and picture a laptop or a desktop tower. But the honest definition is much broader:

> A computer is any device with a processor and memory that can fetch instructions and data and execute them.

By that definition, all of these are computers:

- The rack-mounted **server** in a data center serving a website to thousands of people at once.
- The **laptop** you might be reading this on.
- The **smartphone** in your pocket.
- A **Raspberry Pi** the size of a credit card running a home media server.
- A two-dollar **microcontroller** inside a smart light bulb, with less memory than a single photo from your phone.

They differ enormously in scale — a big server can have a *million* times the memory of a small microcontroller — but not in kind. Every one of them follows the same basic loop: pull instructions from memory and carry them out, over and over, very fast. We'll meet the parts that make up that loop in the next lesson, [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/).

It's useful to split that range into two loose families:

- **General-purpose computers** are built to run whatever software you load onto them — servers, PCs, phones, and single-board computers all qualify. You install new programs without changing the machine.
- **Embedded computers** are built into a product to do one job — the chip in a microwave, a car's engine controller, or a sensor node. Microcontrollers are the classic example. They still run software, but it's usually a single program baked in at the factory.

The line between these blurs (a Raspberry Pi can do either), but it's a helpful first cut, and it maps neatly onto the modules ahead.

## The same job, very different machines

Here's what makes hardware choice interesting: a surprising number of jobs can run on almost *any* of these machines — the question is which one fits best.

Take a simple example: reading a temperature every minute and logging it. You could do that on a $1,000 server, a $35 Raspberry Pi, or a $4 microcontroller. All three can do the job. But the server is absurd overkill and wastes electricity around the clock; the microcontroller sips power and costs pocket change but can't easily store years of history or show you a web dashboard; the Pi sits comfortably in between. None is "right" in the abstract — the right answer depends on your constraints.

This is the theme the whole path returns to. **GopherTrunk**, the software-defined-radio software these docs are about, is a good illustration: it's just software, so it can run on a beefy desktop, a laptop, or a Raspberry Pi sitting right at the antenna. Each of those is a legitimate choice with different trade-offs in cost, performance, and convenience — exactly the kind of decision Module 7 will teach you to make deliberately.

## Why the hardware sets the limits

If software is general and portable, why does the hardware matter so much? Because the physical machine defines the box your software has to live inside. A few dimensions recur so often that we'll spend a whole lesson on them later ([Cost, power & performance trade-offs](/learn/intro-hardware/cost-power-performance/)), but here they are in brief:

| Dimension | What it means | Why it matters |
|-----------|---------------|----------------|
| **Performance** | How fast the processor is and how many things it can do at once | Sets what's feasible — video editing needs far more than blinking an LED |
| **Memory & storage** | How much data the machine can hold while working, and keep long-term | Some programs simply won't fit on a small device |
| **Connectivity** | Whether and how it reaches networks and other devices | A web server is useless offline; a sensor may need Wi-Fi or none at all |
| **Power** | How much electricity it draws | Decides whether it runs on a battery for years or needs the wall and a fan |
| **Cost** | Price to buy and to keep running | A choice you can afford once may be ruinous at a thousand units |

The same program can be trivial on one machine and impossible on another. Choosing hardware well is really choosing *which constraints you want to build within* — and that's a skill, not a guess.

<div class="knowledge-check" data-quiz data-correct-msg="Right — anything with a processor and memory that runs instructions is a computer, from a server to a microcontroller." markdown="0">
  <p class="knowledge-check__q">Quick check: Which of these counts as a "computer" in the sense this path uses?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Only desktop PCs and servers</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only devices with a keyboard and screen</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Anything with a processor and memory that fetches and runs instructions — including phones and microcontrollers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Hardware vs software** — hardware is the physical, general-purpose machine; software is the specific instructions that drive it.
- **A computer is anything that runs instructions** — servers, PCs, phones, single-board computers, and microcontrollers are all computers, differing in scale, not in kind.
- **General-purpose vs embedded** — some machines run whatever you load; others are built into a product to do one job.
- **The same job fits many machines** — the interesting question is which one fits *best* for your constraints.
- **Hardware sets the limits** — performance, memory, connectivity, power, and cost define the box your software lives in.

Next up: a look inside every one of these machines at the four parts they all share. See [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/).
