---
layout: page
title: "GopherTrunk Raspberry Pi / SBC Build — Complete 24/7 Scanner Parts List"
description: "A complete, priced parts list for an always-on headless GopherTrunk scanner on a Raspberry Pi 5 or alternative SBC — the board, SDR, antenna, power supply, microSD, case, and cooling for silent 24/7 logging."
keywords: GopherTrunk Raspberry Pi build, SBC SDR scanner, headless RTL-SDR Pi, 24/7 scanner build, Raspberry Pi 5 SDR parts list, always-on P25 decoder, single board computer scanner, Pi 5 scanner kit
permalink: /gophertrunk-sbc-build/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need to build an always-on GopherTrunk Pi scanner?"
    a: "A Raspberry Pi 5 (8GB), a good SDR dongle, an antenna and adapter, plus the Pi essentials: a quality microSD card, the official USB-C power supply, active cooling, and a case. An active USB extension for the dongle rounds it out. Budget about $180–220 all-in; the software is free."
  - q: "Which Raspberry Pi should I use for GopherTrunk?"
    a: "A Raspberry Pi 5 with 8GB of RAM is the recommended pick — its faster cores comfortably decode one or more control channels plus voice around the clock, and can even channelize a wideband capture. A Pi 4 works for a single site if you already own one."
  - q: "Do I need active cooling on a Pi 5 scanner?"
    a: "For 24/7 duty, yes. A Pi 5 under continuous decode load runs warm and will throttle without a heatsink or fan. The official active cooler clips on and keeps it quiet and cool. Passive cooling can work in a cool room but active is the safe choice for always-on."
  - q: "Why an active USB extension for the dongle?"
    a: "Two reasons: it gives the warm, power-hungry dongle a cleaner power path so the Pi does not brown out, and it moves the dongle away from the board and the RF-noisy USB 3 ports. It is the single cheapest reliability upgrade on a Pi build."
  - q: "Can I use an SBC other than a Raspberry Pi?"
    a: "Yes — any ARM64 or x86 single-board computer that runs 64-bit Linux can run the GopherTrunk binary. The Pi 5 is the best-supported and easiest pick, but Orange Pi, Radxa Rock, and mini-PCs all work. Give any of them solid power and cooling for 24/7 use."
  - q: "How do I control a headless Pi scanner with no monitor?"
    a: "Through GopherTrunk's built-in web console. Point any browser on your network at the Pi's address and you get live spectrum, call logs, and controls — no screen, keyboard, or mouse on the Pi itself. That is what makes an always-on Pi so convenient."
  - q: "Will a Pi build decode encrypted talkgroups?"
    a: "No. No SDR or scanner can decode AES-encrypted talkgroups, on a Pi or anything else. An always-on Pi build logs every call transmitted in the clear around the clock — usually plenty of dispatch and nearly all fire and EMS."
---

# GopherTrunk Raspberry Pi / SBC Build

**A [Raspberry Pi 5](/reference/raspberry-pi/) plus a good [dongle](/best-sdr-for-gophertrunk/)
is the ideal always-on GopherTrunk scanner** — silent, low-power, headless, and reachable from
any browser through GopherTrunk's [web console](/web.html). This is the complete parts list for
a dedicated box that logs every call 24/7 next to the antenna. The
[software is free](/downloads.html); the build runs about **$180–220**.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Host:** [Raspberry Pi 5, 8GB](/reference/raspberry-pi/) (~$80). **Receiver:** any good
[RTL-SDR](/reference/rtl-sdr/) (~$35) on an [antenna](/best-sdr-antenna/). **Pi essentials:**
quality microSD, official USB-C PSU, [active cooler](/best-single-board-computer-for-gophertrunk/),
case (~$45 together). **Connect the dongle** through an
[active USB extension](/reference/usb-extension-cable/) (~$15). **Control it** from any browser
via the [web console](/web.html). **Total ~$180–220.** **No build decodes
[encryption](/police-scanner-encryption/).**
</div>

## The complete parts list

| # | Item | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Raspberry Pi 5 (8GB)** | The always-on host | [Raspberry Pi 5, 8GB](/reference/raspberry-pi/) | ~$80 |
| 2 | **SDR dongle** | Receives the radio | [RTL-SDR Blog V4](/reference/rtl-sdr/) / [NESDR](/reference/nesdr/) | ~$35 |
| 3 | **Antenna + adapter** | Actually hears signals | [Dipole](/reference/dipole-antenna/) / [discone](/reference/discone-antenna/) + [SMA kit](/reference/sma-adapter-kit/) | ~$30 |
| 4 | **Official 27W USB-C PSU** | Stable power = reliability | [Raspberry Pi 5 power supply](/best-single-board-computer-for-gophertrunk/) | ~$12 |
| 5 | **microSD card (32–64GB, A2)** | OS + call recordings | [SanDisk Extreme A2 card](/reference/raspberry-pi/) | ~$12 |
| 6 | **Active cooler** | No throttling under 24/7 load | [Official Pi 5 active cooler](/best-single-board-computer-for-gophertrunk/) | ~$8 |
| 7 | **Case** | Protects the board | [Raspberry Pi 5 case](/reference/raspberry-pi/) | ~$12 |
| 8 | **Active USB extension** | Clean power, moves warm dongle away | [Shielded active cable](/reference/usb-extension-cable/) | ~$15 |

**Running total: ~$204.** Already own a Pi or an antenna? Subtract those lines. This
complements the how-to at [running GopherTrunk 24/7 on a Raspberry Pi](/raspberry-pi-sdr-scanner/) —
that page walks the setup; this page is the shopping list.

## The picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">The host</span>
<h3>Raspberry Pi 5 (8GB)</h3>
<p class="pick-card__price">around $80</p>
<p>Faster cores that decode one or more P25/DMR/NXDN control channels plus voice around the clock — with headroom to channelize a wideband capture. The best-supported SBC for GopherTrunk.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Pi 5 8GB on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/raspberry-pi/">Pi details</a> · <a href="/best-single-board-computer-for-gophertrunk/">SBC guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Reliability upgrade</span>
<h3>Active USB extension cable</h3>
<p class="pick-card__price">around $15</p>
<p>Gives the warm, power-hungry dongle a cleaner power path so the Pi does not brown out, and moves it off the board and away from RF-noisy USB 3 ports. The cheapest reliability win on a Pi build.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Active cable on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/usb-extension-cable/">extension details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">The receiver</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $35</p>
<p>1 ppm TCXO, built-in HF, front-end filtering, switchable bias tee to power a mast LNA up the coax. Rock-steady lock for a scanner that has to hold a control channel for weeks.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/best-rtl-sdr/">dongle guide</a></p>
</div>
</div>

## Why a Pi for always-on scanning

A desktop PC works, but it is overkill for a job that runs forever. A
[Raspberry Pi 5](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) is the natural home
for a permanent GopherTrunk decoder because it is **low-power and silent** (a few watts, cheap
to leave on), **headless by design** (a single Go binary driven entirely through the
[web console](/web.html) — no monitor needed), **small enough to live at the antenna** (mount
it in a closet or attic where the feedline comes in, minimizing coax loss), and **plenty
capable** (its cores decode multiple channels with headroom). Full context:
[running GopherTrunk on a Raspberry Pi](/raspberry-pi-sdr-scanner/).

## Power, cooling, and the SD card — the reliability parts

Almost every "my Pi scanner randomly drops out" story comes down to **power, cooling, or the
SD card**, not software.

> **Feed the Pi properly.** Use the **official 27W USB-C supply**. A Pi 5 driving a warm,
> power-hungry [RTL-SDR](/reference/rtl-sdr/) on a marginal charger will brown out and throw
> USB errors. Adequate, stable power is the single biggest reliability factor. The
> [active USB extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20) helps here
> too by giving the dongle its own clean power path.

- **Cool it.** Under continuous decode load a Pi 5 runs hot and throttles without cooling. The
  **official active cooler** clips on and keeps it quiet — the safe choice for 24/7 duty. See
  the [best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/) guide.
- **Use a quality microSD.** A good **A2-rated card** (SanDisk Extreme or similar, 32–64GB)
  handles the OS and call recordings without the corruption cheap cards suffer under constant
  writes. This is not the place to save $4.

## Antenna and the rest of the chain

The Pi build shares the same RF chain as any other: a [dipole](/reference/dipole-antenna/)
indoors to start, upgrading to a roof-mounted [discone](/reference/discone-antenna/) or a
full [outdoor base build](/gophertrunk-outdoor-base-build/) for real range, joined with an
[SMA adapter kit](/reference/sma-adapter-kit/). Because the Pi is small enough to sit right
at the antenna, you can keep the coax run short and send the *network* the long way — the
right way to fight [feedline loss](/sdr-cables-and-connectors/).

## Setup in brief

1. Flash **64-bit Raspberry Pi OS** to the microSD and boot headless (enable SSH, join the
   network).
2. [Download the GopherTrunk binary](/downloads.html) for ARM64 and follow the
   [hardware setup guide](/hardware.html).
3. Plug the [dongle](/reference/rtl-sdr/) in through the
   [active extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20); attach the
   antenna.
4. Enter your control channel from [RadioReference](https://www.radioreference.com/), start
   GopherTrunk, and open the [web console](/web.html) from another device.
5. Set it to start on boot so it recovers after any power blip.

## Where to buy

Start the build with the [**Raspberry Pi 5, 8GB**](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20)
(~$80) and the [**active USB extension**](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20)
(~$15), then add a dongle, antenna, PSU, microSD, cooler, and case per the list.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost to
you. It never changes what we recommend.*

## Bottom line

For a scanner that just *runs*, a **[Pi 5](/reference/raspberry-pi/) + a
[dongle](/best-rtl-sdr/) + an [active USB extension](/reference/usb-extension-cable/)**, fed by
a real power supply and cooled properly, is the sweet spot: cheap, silent, always on, and
controlled entirely through the [web console](/web.html) — about **$180–220** all-in. Follow
the [Raspberry Pi how-to](/raspberry-pi-sdr-scanner/) to set it up, and remember nothing here
beats [encryption](/police-scanner-encryption/). Compare the other setups on the
[build-lists hub](/gophertrunk-build-lists/).
