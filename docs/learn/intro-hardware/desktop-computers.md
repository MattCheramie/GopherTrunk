---
slug: desktop-computers
title: Desktop computers
description: A desktop computer trades portability for raw value — the most performance per dollar, easy upgrades, and steady cooling that make it a developer's workhorse.
keywords: desktop computer, desktop vs laptop, performance per dollar, upgradable PC, mini PC, all-in-one computer, developer workstation, always-on home machine
level: beginner
status: full
faq:
  - q: "Why would a developer choose a desktop over a laptop?"
    a: "**Value and headroom.** For the same money a desktop gives you a faster processor, more memory, and better cooling, so it holds its top speed during long compiles or renders. It's also easy to upgrade and repair. The trade is that it stays on your desk — you give up portability to get more machine."
  - q: "What is a mini PC, and does it count as a desktop?"
    a: "Yes. A **mini PC** is a small mains-powered box with no built-in screen — think of a compact desktop without the tower. It gives up some upgrade room and top-end power for a tiny footprint, but it runs the same desktop operating systems and is a great quiet, always-on machine for something like a home server."
  - q: "Can a desktop run my software-defined radio setup?"
    a: "Very well. A desktop's USB ports connect straight to an **SDR** dongle, and its sustained processing power handles the heavy real-time math of demodulating and decoding signals. Because it can stay on around the clock, a desktop also makes a fine always-on host for **GopherTrunk** sitting near the antenna."
---
# Desktop computers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best performance per dollar** — a fixed, mains-powered machine spends your money on speed instead of miniaturization. **Upgradable and repairable** — swap the memory, storage, or graphics card instead of buying a whole new computer. **Sustained performance** — roomy cases cool better, so a desktop holds its top speed through long jobs. The cost is that it doesn't move and it draws wall power.
</div>

This module is about the computers you sit in front of. We start with the desktop because it's the most capable of the personal machines and the easiest to reason about: it stays put, it plugs into the wall, and almost every part inside it can be replaced. By the end of this lesson you'll know what counts as a desktop, what people actually do with one, what it runs, and where it beats every other kind of machine for a developer.

## What a desktop is

A **desktop computer** is a personal computer designed to live in one place and run on mains power rather than a battery. The classic form is the **tower**: a rectangular case holding the processor, memory, storage, and expansion slots, with a separate monitor, keyboard, and mouse plugged in. But "desktop" really describes a category, and it comes in a few shapes:

- **Tower** — the traditional box, with the most room to cool and upgrade.
- **All-in-one** — the computer built into the back of the monitor (an iMac is the familiar example), tidier but harder to expand.
- **Mini PC** — a small box the size of a paperback, low-power and quiet, with little or no room inside for upgrades.

What unites them is the trade they make: because a desktop never has to fit in a bag or run off a battery, its designers can prioritize raw capability, cooling, and serviceability over size and weight. That single decision is the source of every strength below.

## What desktops are for

A desktop suits any job where you stay in one place and want as much computer as your budget allows. Common use cases:

- **Heavy development** — compiling large projects, running several services or virtual machines at once, and keeping dozens of browser tabs and tools open without slowing down.
- **Gaming** — where a powerful, well-cooled graphics card matters most.
- **Video editing, 3D, and rendering** — workloads that run for minutes or hours and reward sustained speed.
- **Machine learning** — training and running models that lean hard on a fast GPU and plenty of memory.
- **Multi-monitor work** — desktops drive several large screens easily, which suits coding, trading, and design.
- **An always-on home machine** — a desktop or mini PC left running can act as a home server, file store, or — relevant here — a radio host.

If your work never needs to leave the desk, a desktop gives you the most for your money. If it does need to travel, that's the next lesson, [Laptops](/learn/intro-hardware/laptops/).

## What desktops run

A desktop is a general-purpose computer, so it runs essentially **anything**. The main choice you make is the operating system, and all three major options are first-class development platforms:

- **Windows** — the widest hardware and game support; pairs with WSL (Windows Subsystem for Linux) to give developers a real Linux environment alongside it.
- **macOS** — Apple's desktops only, but a polished Unix-based system popular for software and mobile development.
- **Linux** — free, highly customizable, and the same system most servers run, which makes it a natural choice for backend and systems work.

On top of whichever you pick, every programming language and tool is available: editors and IDEs, compilers and interpreters, containers, databases, and version control. There's no language a desktop can't handle — its job is to be the comfortable, powerful place where you write and test software, including the **GopherTrunk** SDR software these docs describe. You'd typically install GopherTrunk on a desktop, plug an SDR dongle into a USB port, and have plenty of processing headroom to spare.

## Where desktops shine

The desktop's advantages all flow from being a fixed, mains-powered box with room inside:

| Strength | Why it holds | What it means for you |
|----------|--------------|------------------------|
| **Performance per dollar** | No battery or miniaturization tax on the parts | More speed and memory for the same budget |
| **Upgradable and repairable** | Standard parts in standard slots | Add memory or a new GPU instead of replacing the machine |
| **Sustained performance** | Big cases and fans keep parts cool | Holds full speed through long compiles and renders |
| **Connectivity** | Plenty of ports, including many USB | Connect SDR hardware, drives, and several monitors at once |
| **Always-on duty** | Built to run on wall power indefinitely | Doubles as a quiet home server or radio host |

That last point is worth dwelling on for this site. Because a desktop is happy running around the clock and has USB ports right there, it makes an excellent always-on **GopherTrunk** host — left near the antenna, processing signals continuously, while you connect to it from elsewhere.

## The drawbacks

A desktop's strengths come with real costs, and they're the reason laptops exist:

- **Not portable.** It stays where you put it. If you need to work in different places, a desktop simply can't follow you.
- **It takes a desk.** A tower, monitor, and peripherals occupy permanent space and add cable clutter.
- **It needs mains power.** No battery means a power cut stops it dead, and it draws steadily from the wall — a consideration for an always-on machine you'll meet again in [Cost, power & performance trade-offs](/learn/intro-hardware/cost-power-performance/).

None of these are flaws so much as the flip side of the trade. A desktop chose capability over mobility; if mobility is what you need, you look elsewhere.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a fixed, mains-powered box with room to cool and upgrade gives the most performance per dollar." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the desktop's biggest advantage over a laptop?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs software no laptop can run</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It offers more performance per dollar, plus easier upgrades and better sustained speed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses far less electricity than a laptop</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A desktop is a fixed, mains-powered PC** — tower, all-in-one, or mini PC, designed to stay in one place.
- **It runs anything** — Windows, macOS, or Linux, and every language and tool on top.
- **Best performance per dollar** — no battery or size tax means more machine for the money.
- **Upgradable and well-cooled** — swap parts to extend its life, and keep full speed through long jobs.
- **A natural always-on host** — plenty of USB ports and happy to run continuously, which suits an SDR setup like GopherTrunk.
- **The cost is mobility** — it doesn't move, takes desk space, and needs wall power.

Next up: the computer you can carry, and the trade it makes to fit in a bag. See [Laptops](/learn/intro-hardware/laptops/).
