---
layout: page
title: "What Hardware Do I Need to Run GopherTrunk? (Complete Checklist)"
description: "Everything you need to get GopherTrunk decoding police, fire, and EMS radio — the SDR dongle, antenna, adapters, computer, and optional LNA/filters — with tested picks and Amazon links."
keywords: what do I need for GopherTrunk, SDR scanner starter kit, RTL-SDR setup, GopherTrunk hardware, SDR scanning kit, police scanner SDR setup, RTL-SDR antenna adapter
permalink: /what-do-i-need-for-gophertrunk/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need to run GopherTrunk?"
    a: "Four things: a computer (any modern PC, Mac, or a Raspberry Pi), an SDR USB dongle (a ~$35 RTL-SDR is ideal), an antenna, and usually an SMA adapter or cable to connect them. GopherTrunk itself is free. That's a complete digital-scanning setup for well under $100."
  - q: "How much does a GopherTrunk setup cost?"
    a: "If you already own a computer, about $40–70 total: ~$35 for a good RTL-SDR, ~$25 for a dipole antenna kit, and a few dollars for adapters. A dedicated Raspberry Pi for 24/7 use adds ~$80. The software is free and open source."
  - q: "Do I need a special computer?"
    a: "No. GopherTrunk runs on Windows, macOS, and Linux, and on a Raspberry Pi 4 or 5 for always-on use. Any machine from the last decade with a spare USB port is fine — decoding one or two control channels is light work."
  - q: "Do I need an antenna, or is the one in the box enough?"
    a: "The tiny antenna bundled with cheap dongles will hear strong local signals but little else. A ~$25 telescopic dipole kit dramatically improves reception, and an outdoor discone is better still. The antenna matters more than the dongle."
  - q: "What connects the antenna to the dongle?"
    a: "Most SDRs use an SMA connector. Coax antennas often use BNC, N, F (TV), or UHF (PL-259). A cheap SMA adapter kit bridges whatever you have. See our cables and connectors guide."
  - q: "Can I hear encrypted police with this setup?"
    a: "No. No SDR or scanner can decode AES-encrypted talkgroups — it's a cryptographic and legal wall. This setup hears everything still transmitted in the clear, which in most areas includes plenty of dispatch and nearly all fire/EMS."
---

# What Hardware Do I Need to Run GopherTrunk?

**Four things get you decoding police, fire, and EMS radio with GopherTrunk: a
computer, an [SDR](/reference/software-defined-radio/) dongle, an antenna, and a
connector or two.** GopherTrunk is [free, open-source software](/downloads.html), so the
only money is a bit of hardware — a complete digital-scanning rig costs well under $100
if you already own a computer.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The four essentials:** ① a **computer** (PC/Mac/Linux, or a
[Raspberry Pi](/raspberry-pi-sdr-scanner/) for 24/7), ② an **[SDR dongle](/best-sdr-for-gophertrunk/)**
(~$35 [RTL-SDR](/reference/rtl-sdr/)), ③ an **[antenna](/best-sdr-antenna/)** (~$25 dipole
kit), ④ an **[SMA adapter/cable](/sdr-cables-and-connectors/)**. **Optional:** an
[LNA](/best-sdr-lna/) or [filter](/sdr-filters/) for weak signals or overload.
**Total ~$40–70** + free software. **No setup decodes [encryption](/police-scanner-encryption/).**
</div>

## The complete checklist

| # | What | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Computer** | Runs GopherTrunk | A PC you own, or a [Raspberry Pi 5](/raspberry-pi-sdr-scanner/) for 24/7 | $0–80 |
| 2 | **SDR dongle** | Receives the radio | [RTL-SDR Blog V4](/reference/rtl-sdr/) or [NESDR SMArt v5](/reference/nesdr/) | ~$35–40 |
| 3 | **Antenna** | Actually hears signals | [Dipole kit](/best-sdr-antenna/) (indoor) or [discone](/reference/discone-antenna/) (outdoor) | ~$25–40 |
| 4 | **Adapter/cable** | Connects antenna to dongle | [SMA adapter kit](/sdr-cables-and-connectors/) | ~$13 |
| + | **LNA** (optional) | Boosts weak signals | [RTL-SDR Blog Wideband LNA](/best-sdr-lna/) | ~$30 |
| + | **Filter** (optional) | Fixes FM/AM overload | [Broadcast notch filter](/sdr-filters/) | ~$25 |
| + | **USB extension** (optional) | Antenna near a window, PC elsewhere | [Active shielded USB cable](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20) | ~$15 |

## 1. A computer

GopherTrunk runs on **Windows, macOS, and Linux**, and on a **Raspberry Pi 4 or 5** for
always-on monitoring. Decoding one or two control channels is light work — almost any
machine from the last decade with a free USB port will do. For a dedicated, low-power
box that runs 24/7 next to the antenna, a [Raspberry Pi 5](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20)
is ideal; see [running GopherTrunk on a Raspberry Pi](/raspberry-pi-sdr-scanner/).

## 2. The SDR dongle

This is the receiver. A good **[RTL-SDR](/reference/rtl-sdr/)** — the
[RTL-SDR Blog V4](https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20) or a
[NooElec NESDR SMArt v5](https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20) — is
the sweet spot at ~$35–40: shielded, temperature-stable, and more than enough to follow
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/). Step up to
an [Airspy](/reference/airspy/) only for tough RF or wideband capture. Full comparison:
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD7558GT?tag=gophertrunk-20" rel="nofollow sponsored noopener">RTL-SDR Blog V4 + antenna kit on Amazon &rarr;</a>

> **Buy the bundle.** The [RTL-SDR Blog V4 with the dipole antenna kit](https://www.amazon.com/dp/B0CD7558GT?tag=gophertrunk-20)
> gets you the dongle *and* a proper antenna in one box — the simplest way to start.

## 3. An antenna

**The antenna matters more than the dongle.** The stub that ships with cheap sticks
hears strong locals and little else. A **[telescopic dipole kit](https://www.amazon.com/dp/B075445JDF?tag=gophertrunk-20)**
(~$25) you can tune to your band is a huge upgrade; a roof-mounted
**[discone](https://www.amazon.com/dp/B0CL5ZBN94?tag=gophertrunk-20)** covers 25–3000 MHz
for serious coverage. Details and mounting: [best SDR antenna](/best-sdr-antenna/).

## 4. Adapters and cables

Most SDRs use an **[SMA connector](/reference/sma-connector/)**. Antennas and coax often
end in BNC, N, F (TV-style), or UHF/PL-259. A cheap
**[SMA adapter kit](https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20)** bridges
whatever you've got, and a [pigtail or extension cable](/sdr-cables-and-connectors/) lets
you place the antenna where the signal is. Full guide:
[SDR cables and connectors](/sdr-cables-and-connectors/).

## Optional upgrades

- **[LNA (low-noise amplifier)](/best-sdr-lna/)** — helps a weak, distant control channel,
  especially with feedline loss to a rooftop antenna. Powered over the dongle's
  [bias tee](/reference/bias-tee/). Don't overdo gain — it can *cause* overload.
- **[Broadcast notch filter](/sdr-filters/)** — if a strong FM or AM station desensitizes
  your dongle (common in cities), a notch filter restores weak-signal reception.
- **[Active USB extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20)** —
  put the dongle at the window/antenna and the computer wherever's convenient.

## Put it together

1. Screw the antenna (via any needed adapter) onto the SDR.
2. Plug the SDR into the computer.
3. [Download GopherTrunk](/downloads.html) and follow the
   [hardware setup guide](/hardware.html) (on Windows, bind the driver with
   [Zadig](/reference/zadig/) first).
4. Enter your system's control channel — look it up on
   [RadioReference](https://www.radioreference.com/) — and start decoding.

New to how any of this works? Start with the [RF & SDR learning path](/learn/rf-sdr/) and
[what is trunked radio](/learn/digital-trunking/).

## Bottom line

A **computer + a ~$35 [RTL-SDR](/reference/rtl-sdr/) + a ~$25
[antenna](/best-sdr-antenna/) + a few-dollar [adapter](/sdr-cables-and-connectors/)** is
the entire shopping list for a GopherTrunk digital scanner — under $100, or the price of
the free software plus one bundle. Add an [LNA](/best-sdr-lna/) or
[filter](/sdr-filters/) only if your reception needs it. Everything hits the same
[encryption](/police-scanner-encryption/) wall, so buy for the traffic that's in the clear.
