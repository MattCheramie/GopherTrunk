---
slug: computing-hardware-story
title: From looms to microprocessors
description: The hardware lineage behind computing — punch-card looms, Babbage's engine, vacuum tubes, the transistor, integrated circuits, Moore's Law, and the chip.
keywords: history of computing hardware, Jacquard loom, Babbage Analytical Engine, Ada Lovelace, ENIAC, transistor 1947, integrated circuit, Moore's Law, Intel 4004, microprocessor
level: beginner
status: full
faq:
  - q: "Who invented the transistor and when?"
    a: "The transistor was invented at **Bell Labs in 1947** by John Bardeen, Walter Brattain, and William Shockley. It replaced bulky, hot, fragile vacuum tubes with a small, reliable solid-state switch, and it's the single device that made cheap, dense computing possible. Every modern chip is built from billions of transistors."
  - q: "What was the first commercial microprocessor?"
    a: "The **Intel 4004**, released in 1971, is widely considered the first commercial microprocessor — a complete central processing unit on a single chip. It was modest by today's standards, but it proved you could put a whole CPU on one piece of silicon, which set the path toward personal computers and the cheap embedded chips inside today's devices."
  - q: "What is Moore's Law?"
    a: "**Moore's Law** is the observation, made by Intel co-founder Gordon Moore in 1965, that the number of transistors on a chip roughly doubles every couple of years. It isn't a law of physics — it's an industry trend — but for decades it drove steadily cheaper, faster computing, which is exactly why a powerful radio-processing CPU now costs almost nothing."
---
# From looms to microprocessors

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Punch cards predate computers** — programmable looms and tabulators showed machines could follow encoded instructions. **The transistor changed everything** — the 1947 solid-state switch made cheap, dense electronics possible. **Moore's Law** — decades of doubling transistor counts turned room-sized machines into thirty-dollar dongles.
</div>

Software is instructions, but instructions need a machine to run on. This lesson traces that machine's lineage — from a weaving loom in 1804 to the chip in the device on your desk. It's a story of one idea (encode a task, let a machine carry it out) riding successive waves of better, smaller, cheaper hardware. The payoff at the end is concrete: it's why a cheap USB stick can now do the radio work that once filled an equipment rack. If you haven't yet, the previous lesson on [what software is](/learn/intro-software-dev/what-is-software/) sets up the hardware/software split this builds on.

## Punch cards before computers

The deep ancestor of the program isn't electronic at all — it's a loom. In 1804 **Joseph-Marie Jacquard** demonstrated a loom controlled by **punch cards**: holes punched in stiff cards told the loom which threads to raise on each pass, letting it weave intricate patterns automatically. Swap the cards, weave a different pattern. That's a *program* — a removable set of instructions changing what a machine produces — a century and a half before the first computer.

That punch-card idea proved astonishingly durable. It reappeared in computing repeatedly, and well into the twentieth century "writing a program" literally meant punching holes in cards.

## Babbage's engines and the first algorithm

In the 1830s, English mathematician **Charles Babbage** designed the **Analytical Engine** — a mechanical, general-purpose computer driven by punch cards, with separate units for processing ("the mill") and storage ("the store"). It was never fully built in his lifetime, but on paper it had the bones of a modern computer: it could be programmed to carry out arbitrary calculations.

Working with Babbage, **Ada Lovelace** wrote notes on the engine that included an algorithm for computing Bernoulli numbers — widely regarded as the first published computer program. She also grasped something Babbage barely emphasized: that such a machine could manipulate *any* symbols, not just numbers — music or logic, in principle — anticipating general-purpose computing by a century. We profile her further in [the pioneers](/learn/intro-software-dev/the-pioneers/).

## Tabulators and the rise of data processing

The next leap was commercial. For the 1890 US Census, **Herman Hollerith** built electric **tabulating machines** that read data from punch cards, dramatically speeding up what had been a years-long manual count. His company eventually merged into what became **IBM**. For the first half of the twentieth century, "data processing" largely meant rooms full of punch-card tabulators — programmable in a limited way, and enormously profitable.

## Vacuum tubes: the first electronic computers

The machines so far were mechanical or electromechanical — gears, relays, moving parts. The first fully *electronic* computers replaced moving switches with **vacuum tubes**, which switch electric current with no mechanical motion and so run far faster.

The famous example is **ENIAC** (completed in 1945), built at the University of Pennsylvania. It contained roughly 18,000 vacuum tubes, filled a large room, and consumed enormous power. It was a thousand times faster than the relay machines before it — and a maintenance nightmare, since tubes burned out constantly. Vacuum tubes proved electronic computing worked, but they were hot, bulky, fragile, and power-hungry. Something had to replace them.

## The transistor and the integrated circuit

That replacement arrived in **1947** at **Bell Labs**, where Bardeen, Brattain, and Shockley invented the **transistor** — a tiny solid-state switch with no vacuum, no filament, and no moving parts. It does the same job as a tube but is smaller, cooler, more reliable, and far cheaper. The transistor is arguably the most important invention of the century, and everything that follows is a consequence of it.

The next step was packing many transistors together. In the late 1950s, Jack Kilby (Texas Instruments) and Robert Noyce (Fairchild) independently developed the **integrated circuit (IC)** — multiple transistors and their connections fabricated together on a single piece of silicon. Instead of wiring components by hand, you could *print* an entire circuit at once. That made it possible to keep cramming in more and more transistors:

- **Vacuum tube** — one switch, the size of a small light bulb.
- **Discrete transistor** — one switch, the size of a pea.
- **Integrated circuit** — many switches on one chip.
- **Modern chip** — billions of switches on a fingernail-sized die.

## Moore's Law and the microprocessor

In 1965, Intel co-founder **Gordon Moore** observed that the number of transistors on a chip was doubling roughly every couple of years, and predicted it would continue. That trend — **Moore's Law** — held for decades and became the metronome of the whole industry. It isn't a physical law; it's a self-reinforcing industry expectation. But its effect was real: computing got exponentially cheaper and faster, year after year.

That trend made the **microprocessor** possible — an entire CPU on one chip. The **Intel 4004** (1971) is generally credited as the first commercial example. Within a few years, microprocessors powered the first **personal computers**, putting a machine that once filled a room onto a desk, and then into a pocket.

## Why a $30 dongle can replace a rack

Here's where this lineage pays off for radio. Doing software-defined radio means two things have to be cheap and fast: an **analog-to-digital converter (ADC)** that turns the incoming signal into a stream of numbers, and a **CPU** fast enough to crunch those numbers in real time using **digital signal processing (DSP)**.

For most of the twentieth century, neither was cheap. High-speed ADCs and processors fast enough to demodulate a signal in software cost a fortune, so radios stayed hardware: physical mixers, filters, and detectors, each soldered for one job.

Moore's Law dissolved that barrier. Fast ADCs became commodity parts, and ordinary CPUs grew powerful enough to do real-time DSP. The result is the modern SDR: a tiny board that digitizes the airwaves, handed off to software running on a regular laptop. A receiver that once meant a rack of specialized gear is now a **thirty-dollar USB dongle plus free software** — a direct dividend of the entire chain from Jacquard's cards to today's silicon. GopherTrunk exists because that hardware finally got cheap enough.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the 1947 transistor replaced the vacuum tube with a small, cheap, reliable solid-state switch." markdown="0">
  <p class="knowledge-check__q">Quick check: Which invention replaced the hot, fragile vacuum tube and made cheap, dense computing possible?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The punch card</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The transistor (Bell Labs, 1947)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The vacuum-tube amplifier</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Punch cards came first** — Jacquard's loom (1804) showed a machine following removable, encoded instructions: a program before computers.
- **Babbage and Lovelace** — the Analytical Engine sketched a general-purpose computer, and Ada Lovelace wrote the first algorithm for it.
- **Tabulators** — Hollerith's punch-card machines industrialized data processing and seeded IBM.
- **Tubes then transistors** — ENIAC proved electronic computing; the 1947 transistor made it small, cheap, and reliable.
- **ICs and Moore's Law** — integrated circuits packed transistors together, and decades of doubling drove exponential cost and speed gains.
- **The dividend** — cheap fast ADCs and CPUs are why a thirty-dollar SDR dongle now does what once needed a rack.

Next up: with the machine in place, how did we learn to talk to it? We trace the climb from raw machine code through assembly to high-level languages in [From machine code to high-level languages](/learn/intro-software-dev/birth-of-languages/).
