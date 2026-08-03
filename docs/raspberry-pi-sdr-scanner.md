---
layout: page
title: "Run GopherTrunk 24/7 on a Raspberry Pi"
description: "How to run GopherTrunk 24/7 as a headless SDR scanner on a Raspberry Pi 5 — the hardware, power and USB tips, and remote access through GopherTrunk's built-in web console."
keywords: Raspberry Pi SDR scanner, GopherTrunk Raspberry Pi, headless RTL-SDR, 24/7 scanner, Pi 5 SDR, remote scanner web console, always-on P25 decoder, RTL-SDR Pi setup
permalink: /raspberry-pi-sdr-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "Can a Raspberry Pi run GopherTrunk?"
    a: "Yes. GopherTrunk runs on 64-bit Raspberry Pi OS, and a Raspberry Pi 5 (8GB) has ample CPU to decode one or more control channels plus voice around the clock. A single Go binary runs headless with no desktop, so the Pi can sit next to the antenna and be reached over your network."
  - q: "Which Raspberry Pi should I use?"
    a: "A Raspberry Pi 5 with 8GB of RAM is the recommended pick — its faster cores comfortably handle P25/DMR/NXDN decoding and even wideband channelizing. A Pi 4 works for a single site. Give either a good power supply and, ideally, active cooling for 24/7 duty."
  - q: "How do I control a headless Pi scanner?"
    a: "GopherTrunk has a built-in web console. Point any browser on your network at the Pi's address and you get live spectrum, call logs, and controls — no monitor or keyboard needed on the Pi itself. That is what makes an always-on Pi so convenient."
  - q: "Do I need a powered USB hub or extension for the Pi?"
    a: "Often yes. RTL-SDR dongles draw meaningful current and run warm, and plugging one directly into the Pi can cause brownouts or put the hot dongle right against the board. An active (powered) USB extension cable moves the dongle away and stabilizes power."
  - q: "Will a Pi keep up with decoding?"
    a: "For one or two control channels plus voice, easily. A Pi 5 can even channelize several signals from one wideband capture. If you drive a large multi-dongle pool you may want a more powerful host, but for a typical single-site setup a Pi is plenty."
  - q: "Does the Pi need a good power supply?"
    a: "Yes — use the official USB-C supply. Under-powering a Pi 5 while also feeding a power-hungry SDR is the number-one cause of random dropouts and USB errors. Adequate, stable power is the single most important reliability factor for a 24/7 build."
---

# Run GopherTrunk 24/7 on a Raspberry Pi

**A [Raspberry Pi 5](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) plus a
~$35 [RTL-SDR](/reference/rtl-sdr/) is the ideal always-on GopherTrunk scanner** — low
power, silent, headless, and reachable from any browser on your network through
GopherTrunk's [built-in web console](/web.html). Leave it next to the antenna and it
logs every call, day and night, while you check in from your phone or laptop.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Host:** [Raspberry Pi 5, 8GB](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20)
(~$80). **Receiver:** any good [RTL-SDR](/reference/rtl-sdr/) (~$35). **Connect the
dongle** through an
[active USB extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20) (~$15) —
better power, moves the warm dongle off the board. **Control it** from any browser via
the [web console](/web.html) — no monitor on the Pi. **Use a solid power supply** — the
top cause of 24/7 dropouts is under-powering the Pi.
</div>

## Why a Pi for always-on scanning

A desktop PC works, but it is overkill for a job that runs forever. A
[Raspberry Pi](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) is the natural
home for a permanent GopherTrunk decoder because it is:

- **Low power and silent.** A Pi 5 sips a few watts — cheap to leave on 24/7, with no
  fan noise if you cool it passively.
- **Headless by design.** GopherTrunk is a single Go binary; it runs with no desktop and
  is driven entirely through its [web console](/web.html), so the Pi needs no monitor,
  keyboard, or mouse once configured.
- **Small enough to live at the antenna.** Mount it in a closet, attic, or by a window
  right where the feedline comes in, minimizing coax loss to the dongle.
- **Plenty capable.** A Pi 5's cores decode one or more [P25](/reference/project-25/) /
  DMR / NXDN control channels plus voice with headroom to spare.

## The parts list

| Part | Why | Pick | Approx $ |
|---|---|---|---|
| **Raspberry Pi 5 (8GB)** | The always-on host | [Pi 5 8GB](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) | ~$80 |
| **SDR dongle** | Receives the radio | [RTL-SDR Blog V4](/reference/rtl-sdr/) / [NESDR](/reference/nesdr/) | ~$35–40 |
| **Active USB extension** | Stable power, moves the warm dongle away | [Shielded active repeater cable](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20) | ~$15 |
| **Antenna + adapter** | Actually hears signals | [Dipole](/reference/dipole-antenna/) / [discone](/reference/discone-antenna/) | ~$25–40 |
| **Official power supply + cooling** | Reliability | Official USB-C PSU, active cooler | ~$20 |

Full shopping context is in the
[what-you-need checklist](/what-do-i-need-for-gophertrunk/) and the dongle rundown in
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Power and USB — the reliability part

Almost every "my Pi scanner randomly drops out" story comes down to **power and USB**,
not software. Two rules:

> **Feed the Pi properly.** Use the official USB-C supply. A Pi 5 driving a
> power-hungry, warm-running [RTL-SDR](/reference/rtl-sdr/) on a marginal charger will
> brown out and throw USB errors. Adequate, stable power is the single biggest
> reliability factor.

An **[active (powered) USB extension cable](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20)**
does double duty: it provides a cleaner power path to the dongle *and* moves the hot
dongle away from the Pi's board and away from RF-noisy USB 3 ports. If you run several
dongles, use a **powered USB hub** instead so the Pi is not asked to source all that
current itself — see the [multi-dongle setup guide](/multi-dongle-sdr-setup/).

## Remote access through the web console

The payoff of a headless Pi is GopherTrunk's **[web console](/web.html)**. Once the Pi
is running, point any browser on your network at its address and you get live spectrum,
per-call logs with timestamps, and full control — no screen attached to the Pi at all.
Tune it from the couch, check overnight logs from work, or watch calls scroll in on your
phone. This is what turns a Pi in a closet into a genuinely convenient scanner.

## Setup in brief

1. Flash **64-bit Raspberry Pi OS** and boot the Pi headless (enable SSH, join Wi-Fi or
   Ethernet).
2. [Download the GopherTrunk binary](/downloads.html) for ARM64 and follow the
   [hardware setup guide](/hardware.html).
3. Plug the [RTL-SDR](/reference/rtl-sdr/) in through the
   [active extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20); attach
   the antenna.
4. Enter your system's control channel (look it up on
   [RadioReference](https://www.radioreference.com/)), start GopherTrunk, and open the
   [web console](/web.html) from another device.
5. Set it to start on boot so it comes back after any power blip.

## Bottom line

For a scanner that just *runs*, a
**[Raspberry Pi 5](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) + a
[RTL-SDR](/reference/rtl-sdr/) + an
[active USB extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20)** is the
sweet spot: cheap, silent, always on, and controlled entirely through GopherTrunk's
[web console](/web.html). Get the power supply right, place it near the antenna, and you
have a purpose-built digital scanner logging every call around the clock. Start from the
[hardware guide](/hardware.html) and the
[what-you-need checklist](/what-do-i-need-for-gophertrunk/).
