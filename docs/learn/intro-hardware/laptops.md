---
slug: laptops
title: Laptops
description: A laptop is a whole computer you can carry — learn the portability-for-power trade, its battery and thermal limits, and why it's most developers' default machine.
keywords: laptop computer, laptop vs desktop, portable computer, developer laptop, Apple Silicon, ARM laptop, thermal throttling, battery life
level: beginner
status: full
faq:
  - q: "Is a laptop powerful enough for software development?"
    a: "For most work, easily. A modern laptop runs the same operating systems, languages, and tools as a desktop, and today's chips are fast enough that the vast majority of developers work on one full time. The limits show up only under heavy sustained loads — long compiles, video rendering, or model training — where a desktop's better cooling pulls ahead."
  - q: "Why are Apple Silicon and other ARM laptops a big deal?"
    a: "**Efficiency.** Apple's M-series chips and other ARM-based designs deliver strong performance while drawing little power, which means long battery life and quiet, cool operation. For a portable machine that's the ideal combination — you get desktop-class speed for everyday development without the heat and short battery life that used to come with it."
  - q: "What is thermal throttling, and should I worry about it?"
    a: "**Thermal throttling** is when a laptop slows its processor down to avoid overheating, because its thin case can't shed heat as fast as a desktop's. For short bursts you'll never notice. It only matters if you run the machine flat out for long stretches — a sign that a desktop or a remote server might suit that particular workload better."
---
# Laptops

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A whole computer you can carry** — processor, screen, keyboard, and battery in one portable package. **Good enough for most development** — modern chips, especially efficient Apple Silicon and other ARM designs, handle everyday coding with ease. **The trade is power-per-dollar and headroom** — you pay more for less raw speed, limited upgrades, fewer ports, and slowdowns under sustained heavy load.
</div>

The desktop wins on raw value, but most developers don't work on one — they work on a laptop. This lesson explains why. A laptop folds an entire computer into something you can carry, and that single fact reshapes every trade-off from the last lesson. By the end you'll understand what you gain, what you give up, and why "good enough and everywhere" usually beats "fastest but stuck on a desk."

## What a laptop is

A **laptop** is a personal computer with the screen, keyboard, trackpad, processor, memory, storage, and a **battery** all integrated into one foldable unit. Unlike a desktop, it doesn't need a separate monitor or a wall socket to work — you open the lid and it runs.

That integration is the whole point, and it's also the whole constraint. Everything has to fit in a thin case, sip enough power to last on a battery, and stay cool without big fans. So laptop parts are chosen for efficiency and size as much as for speed, and most of them are soldered in place rather than slotted. A laptop is a desktop's mirror image: where the desktop optimizes for capability, the laptop optimizes for being able to come with you.

## What laptops are for

A laptop suits anyone whose work isn't tied to one spot — which, increasingly, is almost everyone:

- **The default developer machine.** For most programmers a laptop is the primary computer, capable of nearly all day-to-day work.
- **Working anywhere** — at a desk, on a couch, in a café, on a train, or at a client's office. Many laptops also dock to a big monitor and keyboard at home, giving you a near-desktop setup that still unplugs and travels.
- **Students and learners** — one affordable machine that goes to class, the library, and home.

If your work moves with you, a laptop's portability outweighs the raw value of a desktop. If you need maximum sustained power and never leave the desk, the last lesson, [Desktop computers](/learn/intro-hardware/desktop-computers/), made the other case.

## What laptops run

A laptop runs **the same things a desktop does** — it's a general-purpose computer with the same operating-system choices:

- **Windows** — broadest selection of laptops, with WSL for a Linux environment.
- **macOS** — Apple's laptops (MacBook Air and Pro), a popular choice for software and mobile work.
- **Linux** — runs well on most laptops and matches the systems you'll deploy to.

Every language, editor, compiler, and container you'd use on a desktop is available, and for ordinary development the experience is identical. You can absolutely install and develop **GopherTrunk** on a laptop, plug an SDR dongle into a USB port, and run it on the go. The differences from a desktop aren't about *what* you can run — they're about how much, how fast, and for how long, which the next two sections cover.

## Where laptops shine, and where they don't

The cleanest way to understand a laptop is straight against a desktop. Same software, different physics:

| Dimension | Desktop | Laptop |
|-----------|---------|--------|
| **Portability** | None — stays on the desk | Goes anywhere, runs on battery |
| **Performance per dollar** | Best | Lower — you pay for miniaturization |
| **Upgradability** | High — swap most parts | Limited — memory and storage often soldered |
| **Sustained performance** | Holds full speed | May throttle under long heavy loads |
| **Ports** | Many, including lots of USB | Fewer; sometimes needs adapters |
| **All-in-one convenience** | Needs monitor, keyboard, mouse | Screen and keyboard built in |

Read that table as the **portability-for-power trade**. A laptop's strengths — it's portable, it's complete out of the box, and modern ones are genuinely fast enough — are bought with lower value-per-dollar, less room to upgrade, and slowdowns when you push it hard for a long time.

One development worth calling out: **Apple Silicon** (the M-series chips) and other efficient **ARM**-based designs have narrowed the gap dramatically. They deliver strong performance with low power draw, so a modern laptop can be quiet, cool, and last all day on battery while still handling serious work. The trade is real but smaller than it used to be.

## The drawbacks in practice

The limits are the flip side of integration, and they show up in specific situations:

- **Less power per dollar.** Fitting everything into a thin case and a power budget costs money; the same spend buys a faster desktop.
- **Limited upgradeability.** Memory and storage are frequently soldered in, so you choose your specs at purchase and live with them. Plan for the future when you buy.
- **Thermal throttling.** Under sustained heavy load — long compiles, video rendering, ML training — a laptop slows itself to avoid overheating. Fine for bursts, frustrating for marathons.
- **Fewer ports.** Slim machines carry fewer connectors, so you may need a dock or adapters to attach several drives, monitors, or SDR devices at once.

For most developers none of these is a dealbreaker. They simply point at the cases — heavy, sustained, multi-device workloads — where a desktop or a remote server earns its keep. That balancing act is the whole subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a laptop trades some power-per-dollar, upgrade room, and sustained speed for the ability to go anywhere." markdown="0">
  <p class="knowledge-check__q">Quick check: What does a laptop mainly give up compared with a desktop?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The ability to run common programming languages and tools</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Some performance per dollar, upgrade room, and sustained speed — in exchange for portability</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compatibility with Windows, macOS, and Linux</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A laptop is a portable, all-in-one computer** — screen, keyboard, and battery integrated so it runs anywhere.
- **It runs the same software as a desktop** — same operating systems, languages, and tools, including GopherTrunk.
- **Modern chips make it good enough** — efficient Apple Silicon and other ARM designs deliver strong performance with long battery life.
- **The trade is power-per-dollar and headroom** — you pay more for less raw speed and fewer upgrade options.
- **Watch for thermal throttling and ports** — sustained heavy loads slow it down, and thin cases carry fewer connectors.
- **Portability usually wins** — for most developers, "fast enough and everywhere" beats "fastest but fixed."

Next up: how to weigh desktop, laptop, or both against the way you actually work, and which specs really matter for writing software. See [Choosing your development machine](/learn/intro-hardware/choosing-a-dev-machine/).
