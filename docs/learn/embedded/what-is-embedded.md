---
slug: what-is-embedded
title: What is an embedded system?
description: An embedded system is a computer hidden inside a device, dedicated to one job. Learn what separates embedded computers from desktops, where they hide in everyday life, and why constraints define them.
keywords: embedded system, what is an embedded system, embedded computer, dedicated computer, firmware, single-board computer, real-time, appliance
level: beginner
status: full
faq:
  - q: What is an embedded system in simple terms?
    a: An embedded system is a computer built into a larger device to do one specific job — the computer in your washing machine, router, car, or thermostat. Unlike a desktop or laptop, you don't install arbitrary programs on it or use it for general tasks; it runs the software it shipped with, usually invisibly, for the life of the product.
  - q: Is a Raspberry Pi an embedded system?
    a: It can be. A Raspberry Pi on your desk running a web browser is being used as a small general-purpose computer. The same Pi mounted in a box, running one program at boot with no monitor attached — say, a radio scanner — is being used as an embedded system. "Embedded" describes the role and the design, not the chip.
  - q: How is embedded software different from normal software?
    a: Embedded software runs under constraints — limited CPU, memory, and storage, sometimes strict timing deadlines, and often no person around to click "retry" when something fails. It has to start itself at power-on, run unattended for months, and recover from problems on its own. Those constraints shape everything this module teaches.
---

# What is an embedded system?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **embedded system** is a computer built into a device to do **one dedicated job**
— invisible, unattended, and always on. Most of the computers you own are embedded:
your router, car, TV, and microwave all contain them. What defines them is not a
particular chip but **constraints** (limited CPU, memory, power) and a **role**:
run one workload reliably with nobody watching. A **single-board computer** like the
Raspberry Pi is the friendliest way in — a real Linux computer small and cheap enough
to dedicate to a single task, like running a radio scanner around the clock.
</div>

This is lesson 1, and its job is to reframe what "a computer" means. By the end you'll
see the computers hiding in everything around you, understand what makes a system
"embedded," and know why this module ends with a Raspberry Pi decoding trunked radio
24/7 — a textbook embedded project.

## Where are the computers you don't see?

Count the computers in your home and you'll probably say two or three — a laptop, a
phone, maybe a desktop. The real number is likely dozens. There's a computer in your
Wi-Fi router, your TV, your thermostat, your car (often more than fifty of them), your
washing machine, your smart speaker, your printer, and your microwave. None of them
have a keyboard. None of them look like computers. All of them boot software, execute
instructions, and talk to hardware exactly like your laptop does.

These are **embedded systems**: computers *embedded inside* a product, doing the one
job the product exists for. The router's computer routes packets. The thermostat's
computer reads a temperature sensor and switches a relay. You never "use the computer"
— you use the device, and the computer is invisible.

## What makes a system "embedded"?

There's no sharp legal definition, but embedded systems share a cluster of traits:

| Trait | Desktop / laptop | Embedded system |
|-------|------------------|-----------------|
| **Purpose** | General — runs anything you install | Dedicated — one job, fixed at design time |
| **Interface** | Screen, keyboard, mouse | Often none, or a few buttons and LEDs |
| **Operation** | Attended — a person is present | Unattended — runs alone for months or years |
| **Startup** | A person logs in and launches apps | Boots straight into its job at power-on |
| **Resources** | Generous CPU, RAM, storage | Just enough for the job, chosen for cost and power |

The heart of it is the middle column versus the right one: **dedication and
unattendedness**. An embedded system is designed around a single workload, starts that
workload itself when power arrives, and keeps it running without human help. When it
fails, there's no one sitting in front of it to notice — so it has to be built to
recover on its own. That last point drives an entire unit of this module
(watchdogs, monitoring, and self-healing services).

## Why do constraints define embedded work?

Embedded computers are chosen to be *just powerful enough*. A thermostat maker won't
pay for a desktop-class processor when a $2 chip does the job — and the cheaper chip
also draws less power, makes less heat, and fits in a smaller box. So embedded work is
the art of living inside a budget: limited **CPU**, limited **memory**, limited
**storage** (often a memory card that wears out if abused), and sometimes a limited
**power** supply.

That sounds like a hardship, but it's really a discipline. You'll meet it concretely in
Unit 6, where a Raspberry Pi has to run GopherTrunk's real-time signal processing:
the CPU budget decides what sample rates and how many simultaneous channels the little
board can decode. Constraints force you to understand what your software actually
costs — a lesson that makes you better on big computers too.

> Rule of thumb: an embedded system should be sized so its job fits comfortably, not
> barely. A board pinned at 100% CPU has no headroom for the busy moment that
> matters most.

## Is a Raspberry Pi embedded, then?

The Raspberry Pi and boards like it — **single-board computers**, or SBCs — sit right
on the boundary, which is what makes them such a good classroom. Out of the box a Pi is
a small general-purpose Linux computer: plug in a monitor and keyboard and it browses
the web. But almost nobody deploys one that way. In real projects the Pi is mounted in
a case with no screen, boots straight into one workload, and runs untouched for months
— which is to say, it's *used as* an embedded system.

That dual nature is the pedagogy of this whole module: you get the full comfort of
Linux (a shell, packages, [SSH](/learn/linux-cli/ssh-and-remote/), real debugging
tools) while practising genuinely embedded habits — headless operation, services that
start at boot, surviving power cuts, and living within a small CPU. The
[Computer Hardware module](/learn/intro-hardware/what-is-an-sbc/) covers where SBCs sit
in the wider hardware landscape; here we'll actually build with one.

## What does an embedded project look like end to end?

Every embedded project — commercial or hobbyist — answers the same five questions,
and they map exactly onto this module's units:

1. **What hardware?** A board with enough compute, plus storage, power, and cooling
   (Units 1–2).
2. **What software base?** Usually Linux, installed and run **headless** — no monitor,
   managed over the network (Unit 3).
3. **What does it talk to?** Sensors, radios, and peripherals over GPIO, USB, and
   serial buses (Unit 4).
4. **How does it survive?** Heat, storage wear, crashes, and the fact that nobody is
   watching (Unit 5).
5. **What's the job?** For us: GopherTrunk decoding trunked radio from an RTL-SDR,
   serving a web console to your LAN, around the clock (Unit 6).

Hold that shape in your head. Every lesson from here on is filling in one of those
boxes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — dedication to one job, running unattended, is the defining trait." markdown="0">
  <p class="knowledge-check__q">Quick check: what most fundamentally makes a computer "embedded"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses an ARM processor instead of an Intel one</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It is physically smaller than a desktop PC</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It is dedicated to one job and runs unattended inside a device</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **embedded system** is a computer inside a device, dedicated to **one job** —
  and most of the computers you own are embedded.
- The defining traits are **dedication** and **unattended operation**: boot straight
  into the job, run for months, recover without help.
- **Constraints** — CPU, memory, storage, power — are the defining discipline of
  embedded work, not an inconvenience.
- A **single-board computer** like the Raspberry Pi is a general-purpose Linux machine
  usually *deployed as* an embedded system — the best of both worlds for learning.
- The module's destination is a classic embedded build: a Pi running **GopherTrunk**
  as a 24/7 scanner appliance.

Next up: [SBC vs microcontroller vs PC](/learn/embedded/sbc-vs-microcontroller-vs-pc/).
