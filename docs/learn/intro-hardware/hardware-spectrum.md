---
slug: hardware-spectrum
title: The hardware spectrum — cloud to microcontroller
description: From rented cloud servers down to two-dollar chips, every computing platform sits on one spectrum. See them ordered by power, cost, and how much of the machine you actually control.
keywords: hardware spectrum, cloud to microcontroller, computing platforms compared, server vs sbc vs microcontroller, who manages the hardware, choosing a computing platform, hardware tiers
level: beginner
status: full
faq:
  - q: "What is the hardware spectrum?"
    a: "It's a way of laying every computing platform on a single line, from the most powerful and abstract (rented cloud servers) down to the smallest and most direct (microcontrollers). Ordering them by **power**, **cost**, and **how much you control** makes it clear that there's no single best machine — only the one that fits a given job's constraints."
  - q: "Is a more powerful platform always better?"
    a: "No, and this is the central idea of the whole path. A cloud server is wildly overpowered for blinking an LED, and a microcontroller can't host a website. **The right tool depends on your constraints** — cost, power draw, performance, control, and effort. More power usually means more cost, more electricity, and sometimes less direct control."
  - q: "What does \"control vs managed\" mean on this spectrum?"
    a: "At the cloud end, someone else owns and maintains the physical machine — you just rent capacity and they handle hardware, cooling, and replacement. At the microcontroller end, you own and control everything down to the individual pins. Moving down the spectrum trades convenience for control: less is managed for you, but more is yours to decide and maintain."
  - q: "Where does GopherTrunk fit on this spectrum?"
    a: "Almost anywhere. Because **GopherTrunk** is software-defined-radio software, it can run on a dedicated server, a desktop, a laptop, or a Raspberry Pi sitting right at the antenna. The same job lands at very different points on the spectrum depending on whether you want raw power, portability, or a cheap node out in the field."
---
# The hardware spectrum — cloud to microcontroller

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**One spectrum, many platforms** — cloud, VPS, dedicated server, home server, desktop, laptop, tablet, phone, single-board computer, and microcontroller all sit on one line. **Three axes order it** — raw power, cost, and how much of the machine you control versus someone else manages. **The right tool depends on constraints** — there's no best platform, only the best fit for a job.
</div>

You now know the four building blocks every computer shares. This lesson zooms back out and lays every platform this path covers onto a single map. It's the frame the rest of the course hangs on: each later module takes one stretch of this spectrum and explores it in depth. Get this map in your head now and every lesson afterward will have a place to sit.

## The spectrum, end to end

Here is the full range, ordered from the most powerful and abstract down to the smallest and most direct:

**Cloud / web hosting → VPS → dedicated server → home server → desktop → laptop → tablet / phone → single-board computer → microcontroller**

At the top, you don't own a machine at all — you rent a slice of someone else's, and they handle the physical hardware. In the middle sit the personal computers most people picture. Toward the bottom, the machines get small, cheap, and physical: boards you hold in your hand and wire to the world. Nothing about this is a quality ranking. It's a map of trade-offs, and which end you reach for depends entirely on what you're building.

## Ordering by raw power

The most obvious way to read the spectrum is by capability — CPU, memory, storage, and how much work the machine can do at once. This tracks closely with the building blocks from the last lesson.

| Platform | Relative power | What that buys you |
|----------|----------------|---------------------|
| **Cloud / web hosting** | Effectively unlimited (scales on demand) | Serve millions of users; grow instantly |
| **Dedicated / home server** | Very high | Heavy always-on workloads, lots of storage |
| **Desktop** | High | Demanding local work — video, compiling, simulation |
| **Laptop** | High, with limits | Most development and daily work, on the move |
| **Tablet / phone** | Moderate, battery-bound | Apps, sensors, communication in your pocket |
| **Single-board computer** | Low but real | A full Linux machine for tens of dollars |
| **Microcontroller** | Tiny | One small job, sipping power |

Power falls off by orders of magnitude from top to bottom — but so does what you need to pay for and carry.

## Ordering by cost and power draw

The same spectrum looks different through the lens of money and electricity, and the two often move together. Cloud platforms cost little upfront but charge ongoing for what you use. A microcontroller costs a few dollars once and draws almost no power. In between, machines vary in both upfront price and the running cost of keeping them powered.

| Platform | Upfront cost | Ongoing cost / power draw |
|----------|--------------|----------------------------|
| **Cloud / web hosting** | Near zero | Monthly fees; scales with use |
| **VPS** | Near zero | Modest fixed monthly fee |
| **Dedicated server** | Low (rented) to high (owned) | Significant, always-on |
| **Home server** | One-time hardware | Your electricity bill, 24/7 |
| **Desktop** | Moderate to high | Wall power when in use |
| **Laptop** | Moderate to high | Battery, then wall |
| **Phone / tablet** | Moderate to high | Battery, frugal |
| **Single-board computer** | Tens of dollars | A few watts |
| **Microcontroller** | A few dollars | Milliwatts — can run for years on a battery |

This is why a job that's "free" to prototype on a microcontroller can be expensive to run as an always-on home server — the cost shifts from upfront to ongoing. We'll dig into exactly these tensions in the next lesson.

## Ordering by control vs management

The third axis is the least obvious and often the most important: **how much of the machine is yours to control, versus managed for you by someone else.**

At the cloud end, you control almost nothing physical. A hosting provider owns the hardware, cools it, replaces failed drives, and keeps it online — you just deploy software onto it. That's enormously convenient, but you can't touch the metal, and you're trusting someone else's decisions. As you move down the spectrum, more is handed to you: a home server is yours to maintain entirely, and a microcontroller is yours down to the last pin. You decide everything — and you're responsible for everything.

| Platform | Who manages the hardware | What you control |
|----------|--------------------------|-------------------|
| **Cloud / web hosting** | Provider, fully | Just your code and config |
| **VPS** | Provider owns metal; you run the OS | The whole software stack |
| **Dedicated server** | Provider (rented) or you (owned) | The full machine |
| **Home server / desktop / laptop** | You | Everything |
| **SBC / microcontroller** | You | Everything, down to the pins |

Convenience and control sit at opposite ends. Neither is "better" — a beginner shipping a website wants the cloud's convenience; a tinkerer reading sensors wants the microcontroller's directness.

## The right tool depends on constraints

Put the three readings together and the theme of the whole path appears: **there is no best platform, only the best fit.** A platform that's perfect for one job is absurd for another. GopherTrunk makes this concrete — the same software-defined-radio software can run on a dedicated server (maximum power, always on), a laptop (portable, good for experimenting), or a Raspberry Pi bolted next to the antenna (cheap, low-power, out in the field). All three are legitimate; the right one depends on your constraints.

The modules ahead each take one region of this spectrum: servers and hosting, personal computers, mobile devices, single-board computers, and microcontrollers — then Module 7 turns the whole map into a method for choosing deliberately. Keep this spectrum in mind as you go; everything else is a closer look at one stretch of it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — moving down the spectrum trades convenience for control: less is managed for you, but more is yours to decide." markdown="0">
  <p class="knowledge-check__q">Quick check: As you move from the cloud end toward the microcontroller end of the spectrum, what generally happens?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The machine becomes more powerful and more expensive to buy</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">More of the machine becomes yours to control, and more is your responsibility to manage</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You give up control to the hosting provider</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **One spectrum** — cloud, VPS, dedicated and home servers, desktops, laptops, tablets, phones, SBCs, and microcontrollers all sit on a single line from most powerful to smallest.
- **Power axis** — capability falls off by orders of magnitude from top to bottom, and so does what you must pay for and carry.
- **Cost and power axis** — the cloud trades near-zero upfront cost for ongoing fees; a microcontroller is cheap once and sips electricity.
- **Control axis** — the cloud manages everything for you; the microcontroller hands you everything, convenience traded for control.
- **The right tool depends on constraints** — there's no best platform, and GopherTrunk can run almost anywhere on this map depending on the job.

Next up: why a web host runs PHP and a microcontroller runs C — how the hardware you've just mapped shapes which languages are even practical. See [Programming languages & where they run](/learn/intro-hardware/languages-and-hardware/).
