---
slug: worked-examples
title: "Worked examples: picking hardware for real projects"
description: The decision framework run end to end on four real projects — a blog, a battery sensor, a home media server, and a GopherTrunk scanner — from requirements to a justified choice.
keywords: hardware worked examples, choosing hardware examples, blog hosting choice, ESP32 sensor project, home media server hardware, GopherTrunk hardware, SDR scanner hardware, decision framework example
level: advanced
status: full
faq:
  - q: "What hardware should I use for a personal blog?"
    a: "The smallest thing that serves the pages. A static blog is happy on [shared web hosting](/learn/intro-hardware/web-hosting/) or a static host for a few dollars a month; only if you need server-side features or full control do you step up to a small [VPS](/learn/intro-hardware/vps/). A blog almost never justifies more than that."
  - q: "Why an ESP32 for a battery temperature sensor instead of a Raspberry Pi?"
    a: "Because a Pi is a full computer that would flatten a battery in a day, while an [ESP32](/learn/intro-hardware/esp32/) wakes, takes a reading, sends it over Wi-Fi, and drops into deep sleep drawing microamps — running for months on a small battery. The job is tiny and physical; the [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) is the right scale, and the Pi is overkill."
  - q: "Why can't GopherTrunk run on a cloud VPS?"
    a: "Because the SDR dongle is a **USB device that must be physically plugged into the machine doing the capture**, and you can't plug hardware into a server in someone else's data center. The capture tier has to be a machine you physically touch — a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) at the antenna or a local PC. The cloud can store and serve recordings, but it can never capture."
  - q: "Is a repurposed mini-PC a good home server?"
    a: "Often, yes. For a media-and-backup [home server](/learn/intro-hardware/home-servers/), a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) handles light duty cheaply, while a repurposed mini-PC or NAS gives you more storage, transcoding muscle, and headroom. The choice turns on how much you store and how heavy the workload is."
---
# Worked examples: picking hardware for real projects

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The framework in action** — four projects taken from requirements through hard constraints to a justified choice. **Hard constraints decide fast** — a USB dongle, a battery, a public service each collapse the field on their own. **Smallest thing that works wins** — a static host for a blog, an ESP32 for a sensor, a Pi at the antenna for a scanner.
</div>

This is the final lesson of the path, and the most practical. We take the [decision framework](/learn/intro-hardware/decision-framework/) and run it end to end on four real projects, each laid out the same way — requirements, then the hard constraints that eliminate options, then the choice and why. Watch how often a single pass/fail fact does most of the work, and how the answer is almost always the *simplest* thing that clears the bar.

## Example 1: a personal blog

**Requirements.** Serve a handful of articles to modest traffic; cheap; near-zero maintenance; no physical hardware; reachable publicly 24/7.

**Hard constraints.** It must be a public, always-on service — so a home machine on a flaky connection is out, and so is anything battery-powered or device-bound. Beyond that, the constraints are loose: almost any host qualifies, which tells you the choice will be decided by simplicity and cost, not capability.

**The choice.** A static site on [shared web hosting](/learn/intro-hardware/web-hosting/) (or a static host), a few dollars a month, with the provider managing the machine entirely. **Why:** the job is tiny and read-only, so paying for a server you administer yourself buys nothing. Only if you later need server-side features or full control would you step up to a small [VPS](/learn/intro-hardware/vps/). The lesson: when nothing is at an extreme, the *smallest, simplest* option wins — exactly as the framework predicts.

## Example 2: a battery-powered temperature/soil sensor

**Requirements.** Read temperature and soil moisture in a garden bed; report readings over Wi-Fi every few minutes; run for months on a small battery; cost a few dollars; live outdoors, unattended.

**Hard constraints.** Two are decisive. It must touch the physical world (sensors), which rules out any remote cloud machine. And it must run for *months on a battery*, which rules out full computers and even an [SBC](/learn/intro-hardware/what-is-an-sbc/) — a Raspberry Pi would drain the battery in a day. Together these collapse the field to one region: [microcontrollers](/learn/intro-hardware/what-is-a-microcontroller/).

**The choice.** An [ESP32](/learn/intro-hardware/esp32/) running a deep-sleep cycle. **Why:** it has Wi-Fi built in for the reporting requirement, and its deep-sleep mode draws microamps between readings — wake, sample, transmit, sleep — which is what makes months on a battery possible. An [Arduino](/learn/intro-hardware/arduino/) would suit a purely offline version, but the Wi-Fi requirement makes the ESP32 the natural pick. The hard constraints did all the work; by the time we weighed anything, the field was already one part.

## Example 3: a home media + backup server

**Requirements.** Store and stream a media library across the house; hold automatic backups of family devices; run quietly 24/7; modest budget; live in a closet; minimal fuss.

**Hard constraints.** It must run always-on and serve local devices, which favors a low-power machine in the house and rules out a battery device or microcontroller (no full OS, no real storage). It needs a full operating system and real disks. None of this forces a single answer, so this one is decided more on trade-offs than elimination.

**The choice.** A [home server](/learn/intro-hardware/home-servers/) — either a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) for light duty, or a repurposed mini-PC/NAS for heavier use. **Why:** weigh the trade-offs from [cost, power & performance](/learn/intro-hardware/cost-power-performance/). A Pi sips power and costs little, perfect if you mostly stream to one device at a time and store a modest library. If you need lots of storage or on-the-fly transcoding for several streams, a mini-PC or NAS earns its slightly higher power draw and price. Both are far cheaper to run over a year than leaving a full [desktop](/learn/intro-hardware/desktop-computers/) on around the clock. Pick the smaller one and step up only if it can't keep up.

## Example 4: a GopherTrunk SDR scanner

**Requirements.** Capture and decode trunked radio from an antenna; the **SDR dongle plugs into the machine over USB**; sit near the antenna (often a roof or attic); run continuously; let you view recordings remotely; hobby budget, one unit.

**Hard constraints.** The decisive one is physical: the USB dongle must be attached to the machine doing the capture. That single fact **eliminates any cloud [VPS](/learn/intro-hardware/vps/) or shared host outright** — you cannot plug a USB device into a server you don't physically touch. The capture machine must be on-site, near the antenna, running a full OS to host GopherTrunk.

**The choice.** A [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) mounted at the antenna with the dongle attached, or a [laptop](/learn/intro-hardware/laptops/)/[desktop](/learn/intro-hardware/desktop-computers/) with the dongle plugged in locally. **Why:** the Pi is low-power, cheap, and small enough to live up by the feedline running 24/7 — ideal for a permanent install. A PC suits an occasional bench setup where you don't mind the dongle on your desk. As [combining tiers](/learn/intro-hardware/combining-tiers/) showed, you can layer a server (the same Pi, a home server, or a VPS) behind it to *store and serve* recordings, and view them from your phone anywhere — the cloud just can't be the *capture* tier. This is the path's recurring lesson in one project: a hard physical constraint pins the choice, and everything else layers around it.

## The pattern across all four

| Project | Decisive constraint | Choice | Tier |
|---------|--------------------|--------|------|
| Personal blog | Public 24/7, no physical hw | [Shared/static hosting](/learn/intro-hardware/web-hosting/) (or small [VPS](/learn/intro-hardware/vps/)) | Cloud |
| Battery sensor | Battery life + physical sensing | [ESP32](/learn/intro-hardware/esp32/) with deep sleep | Edge device |
| Home media server | Always-on, full OS, local serving | [Pi](/learn/intro-hardware/raspberry-pi-and-family/) or mini-PC [home server](/learn/intro-hardware/home-servers/) | Local server |
| GopherTrunk scanner | USB dongle must be attached | [Pi](/learn/intro-hardware/raspberry-pi-and-family/) at antenna / local PC | Edge + local server |

In every case the same shape holds: a hard constraint narrowed the field, the trade-offs broke any remaining tie, and the winner was the simplest thing that did the job. That's the whole skill.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the USB dongle must be physically attached, so a remote cloud VPS can never be the capture machine." markdown="0">
  <p class="knowledge-check__q">Quick check: Why is a cloud VPS ruled out for the GopherTrunk capture machine?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Cloud VPSes are always too slow for radio decoding</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The SDR dongle must be physically plugged in, and you can't attach USB hardware to a remote server</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A VPS can't run a full operating system</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Run the framework every time** — requirements, then hard constraints, then trade-offs, then the simplest fit; it works on any project.
- **A blog wants the least** — static or shared hosting beats a server you don't need to manage.
- **A battery sensor is a microcontroller job** — the ESP32's deep sleep is what makes months on a battery possible.
- **A home server balances trade-offs** — a Pi for light duty, a mini-PC or NAS when storage and transcoding demand it.
- **GopherTrunk is pinned by physics** — the USB dongle forces an on-site capture machine; the cloud can store and serve but never capture.

That's the path. Keep the [glossary](/learn/intro-hardware/glossary/) handy as a reference, and revisit any [module from the hub](/learn/intro-hardware/) as your projects evolve.
