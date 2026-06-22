---
slug: sbc-use-cases-and-limits
title: SBC use cases, strengths & drawbacks
description: Single-board computers excel at home labs, media servers, robots, and edge sensors. Learn where SBCs win and the heat, power, storage, and CPU limits that don't.
keywords: SBC use cases, Raspberry Pi projects, SBC strengths and weaknesses, SD card wear, thermal throttling Raspberry Pi, home server, edge AI, SBC vs mini PC, when to use a microcontroller
level: intermediate
status: full
faq:
  - q: "What are single-board computers good for?"
    a: "SBCs shine at small, always-on jobs: home servers and NAS-lite storage, media centers, retro gaming, robotics, edge AI and vision, IoT gateways, SDR scanners, and learning Linux. They suit anything that needs a real computer with hardware I/O but doesn't need a full PC's power, size, or electricity bill."
  - q: "Why do Raspberry Pis sometimes become unreliable?"
    a: "The most common culprit is the **microSD card**. SD cards wear out with repeated writes and can corrupt, especially under heavy logging or sudden power loss. Other causes are an **underpowered power supply** and **thermal throttling** when the chip overheats. A good supply, a heatsink or fan, and booting from an SSD fix most reliability problems."
  - q: "When should I use a mini-PC or a microcontroller instead of an SBC?"
    a: "Step **up** to a mini-PC or server when you need heavy, sustained compute — lots of RAM, fast storage, or serious CPU/GPU. Step **down** to a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) when the job is simple, must run on a battery for a long time, or needs to start instantly. The SBC is the middle ground, not the universal answer."
---
# SBC use cases, strengths & drawbacks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SBCs win at small, always-on jobs** — home servers, media centers, robotics, edge AI, IoT gateways, and SDR scanners. **Their strengths are cheap, low-power, full-OS, and GPIO** in a small, quiet package. **Their drawbacks are real** — SD-card wear, limited CPU/RAM, thermal throttling, and fussy power supplies. **Know when to step up or down** — a mini-PC for heavy compute, a microcontroller for simple, battery, instant-on jobs.
</div>

By now you know what an SBC is, which boards to consider, and how to program one. This final lesson of the module is about judgment: where an SBC is the right tool, where it quietly disappoints, and how to recognize the moment you should reach for something bigger or smaller instead. Getting this right is the difference between a project that runs happily for years and one that mysteriously falls over every few weeks.

## What SBCs are great for

An SBC's sweet spot is a job that needs a *real computer with hardware access*, but not a full PC's power or footprint. That covers a remarkable range:

- **Home server / NAS-lite** — file sharing, backups, a personal cloud, ad-blocking DNS, or a small website for the house.
- **Media center** — running media software on a TV, cheaply and silently.
- **Retro gaming** — emulating old consoles, a hugely popular hobby use.
- **Robotics** — the brain of a robot, using GPIO to drive motors and read sensors while running higher-level logic in Python.
- **Edge AI and vision** — running a camera and a model locally (especially on a GPU-equipped board like a Jetson) so decisions happen on-device.
- **IoT gateway** — collecting data from many small sensors and microcontrollers, then forwarding it to the network.
- **SDR scanner** — a **GopherTrunk** node at the antenna with a USB SDR dongle, scanning around the clock.
- **Learning Linux** — a cheap, disposable machine to experiment on without risking your main computer.

The thread connecting these is *always-on, low-stakes, hardware-adjacent* work. None of them needs a fast desktop; all of them benefit from a small, quiet, inexpensive box that can sit somewhere and just run. For more on picking the right tier for a given project, the final module starts at [Start with requirements](/learn/intro-hardware/start-with-requirements/).

## The strengths that make them shine

It's worth naming exactly *why* SBCs fit those jobs so well. Five strengths do most of the work:

- **Cheap** — a capable board costs less than a video game, so experimenting (and replacing one) is painless.
- **Low power** — a few watts means you can leave it on permanently for pennies, and even run it from a battery or solar in the field.
- **Full operating system** — real Linux brings the whole software world: any language, networking, Docker, services. (See [Programming & running software on an SBC](/learn/intro-hardware/programming-an-sbc/).)
- **GPIO** — the pins that let it touch the physical world, the thing a laptop can't easily do.
- **Small and quiet** — credit-card-sized and usually fanless, it tucks into places a PC never could.

Put together, those traits explain the SBC's whole reason for existing: it's the cheapest, smallest way to get a *complete computer* into a project.

## The drawbacks that decide the limits

The same smallness that's a strength is also where the limits live. An SBC is not a small PC, and treating it like one leads to grief:

- **SD-card wear and reliability** — the microSD card is the weakest link. Cards wear out under heavy writes and can corrupt on sudden power loss. Heavy logging, databases, and swap all shorten their life. Booting from an SSD and minimizing writes helps a lot.
- **Limited CPU and RAM** — even a Pi 5 is far slower and has far less memory than a modern desktop. Demanding workloads will crawl or simply won't fit.
- **Thermal throttling** — under sustained load the chip heats up and deliberately slows itself to avoid damage. Without a heatsink or fan, you lose the performance you paid for.
- **Power-supply pickiness** — SBCs are surprisingly fussy about power. An underpowered or low-quality supply causes random crashes, corruption, and "undervoltage" warnings that look like software bugs but aren't.
- **Not for heavy compute** — video transcoding at scale, large databases, big ML training, or anything needing serious sustained horsepower belongs on a bigger machine.

| | Strengths | Drawbacks |
|---|-----------|-----------|
| **Cost** | Very cheap to buy and run | — |
| **Power** | A few watts; battery/solar capable | Fussy supplies cause crashes |
| **Software** | Full Linux, all the usual tools | — |
| **Hardware I/O** | GPIO, I2C, SPI, UART | — |
| **Performance** | Plenty for light always-on jobs | Limited CPU/RAM; thermal throttling |
| **Storage** | microSD is cheap and simple | SD wear and corruption risk |
| **Size** | Small, quiet, fanless | Cramped for heavy workloads |

Most of these are manageable — a good power supply, a heatsink, an SSD, and modest expectations turn a flaky Pi into a rock-solid one. The mistake is ignoring them and then blaming your code.

## When to step up or down

The deepest skill is knowing when an SBC *isn't* the answer. The decision usually goes one of two ways:

**Step up to a mini-PC or server** when you need heavy, sustained compute — lots of RAM, fast storage, real CPU or GPU horsepower. If your "home server" is transcoding several video streams at once or running a big database, a small x86 mini-PC will be far happier than a Pi, often without much more power draw.

**Step down to a microcontroller** when the job is simple, must sip power for a very long time, or needs to start instantly and never crash. A microcontroller running one program on bare metal can live on a coin cell for months, boot in milliseconds, and never suffer SD-card corruption — because there's no SD card and no OS to corrupt. If all you need is "read a sensor and blink a light," an SBC is overkill. That's the subject of the next module, beginning with [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/).

The SBC is a brilliant middle ground, not a universal tool. Reaching for it deliberately — and knowing the two neighbors it sits between — is exactly the kind of choice this path is training you to make.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the microSD card is the usual reliability weak point, worn down by writes and vulnerable to power loss." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the most common reliability weakness of a typical SBC like a Raspberry Pi?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It has no operating system, so it crashes constantly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The microSD card wears out with writes and can corrupt on power loss</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It cannot connect to a network or run services</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SBCs win at small, always-on jobs** — home servers, media centers, retro gaming, robotics, edge AI, IoT gateways, SDR scanners, and learning Linux.
- **Their strengths** — cheap, low-power, a full OS, GPIO, and small and quiet, all at once.
- **Their drawbacks** — SD-card wear, limited CPU/RAM, thermal throttling, fussy power supplies, and no head for heavy compute.
- **Most drawbacks are manageable** — a good supply, cooling, an SSD, and realistic expectations make an SBC reliable.
- **Know your neighbors** — step up to a mini-PC for heavy compute, step down to a microcontroller for simple, battery, instant-on jobs.

Next up: the smaller, simpler sibling of the SBC — the chip that runs one program on bare metal. See [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/).
