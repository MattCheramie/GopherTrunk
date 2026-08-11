---
layout: page
title: "GopherTrunk Portable Build — Field & Battery Scanner Parts List"
description: "A complete, priced parts list for a portable, battery-powered GopherTrunk scanner — laptop or Raspberry Pi, USB-C power bank, portable whip antenna, and dongle for storm-chasing, events, and travel with no wall power."
keywords: GopherTrunk portable build, battery SDR scanner, field SDR kit, storm chasing scanner, portable RTL-SDR, USB power bank SDR, travel scanner build, laptop SDR scanner, go-bag scanner
permalink: /gophertrunk-portable-build/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need for a portable GopherTrunk scanner?"
    a: "A laptop or a Raspberry Pi, a large USB-C power bank to run it off-grid, a portable whip antenna, an SDR dongle, and an adapter. That is a go-bag scanner for storm-chasing, events, or travel — about $120–160 in hardware if you bring your own laptop, and the software is free."
  - q: "How do I power a portable SDR scanner with no outlet?"
    a: "A high-capacity USB-C power-delivery bank. It can run a Raspberry Pi directly for many hours, or top up a laptop between sessions. Size it to your runtime: a 20,000–25,000 mAh PD bank comfortably powers a Pi-based scanner for a full day of intermittent use."
  - q: "What antenna works for a portable build?"
    a: "A telescopic whip like the Nagoya NA-771 is the classic portable pick — it collapses for a bag and extends for reception, and tunes usefully across the VHF/UHF scanner bands. A magnetic-mount antenna on a car roof is a great upgrade when you have a vehicle to work from."
  - q: "Laptop or Raspberry Pi for a portable scanner?"
    a: "A laptop is simplest — screen, keyboard, and power all in one, controlled directly. A Raspberry Pi with a power bank is lighter and sips less power, but you drive it headless from a phone or tablet over its web console. Choose the laptop for convenience, the Pi for minimum weight and runtime."
  - q: "How long will a portable build run on battery?"
    a: "It depends on the battery and the host. A Raspberry Pi draws only a few watts, so a 25,000 mAh power bank can run a Pi scanner most of a day. A laptop running GopherTrunk uses more, but decoding is light work, so a laptop's own battery plus a PD bank to recharge it gets you through a field day."
  - q: "Is a portable build good for storm-chasing?"
    a: "Yes — a battery-powered GopherTrunk rig lets you monitor local public-safety and emergency traffic in the field where there is no power. Pair it with a mag-mount antenna on the vehicle for better reception on the move. As always, it hears only what is transmitted in the clear."
  - q: "Will a portable build decode encrypted traffic?"
    a: "No. Portability changes nothing about encryption — no SDR or scanner can decode AES-encrypted talkgroups. A field rig hears whatever is in the clear, which in most areas includes plenty of dispatch and nearly all fire and EMS."
---

# GopherTrunk Portable Build

**A laptop or [Raspberry Pi](/reference/raspberry-pi/), a [power bank](/hardware.html), a
[portable whip](/best-scanner-antenna/), and a [dongle](/best-sdr-for-gophertrunk/) make a
go-bag GopherTrunk scanner** you can run anywhere — storm-chasing, at events, or traveling,
with no wall power in sight. [GopherTrunk is free](/downloads.html); this field rig is about
**$120–160** in hardware if you bring your own laptop.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Host:** a laptop you own, or a [Raspberry Pi](/reference/raspberry-pi/) (~$80) for minimum
weight. **Power:** a high-capacity [USB-C PD power bank](/hardware.html) (~$40). **Antenna:** a
collapsible [Nagoya NA-771 whip](/reference/whip-antenna/) (~$20), or a mag-mount for the car.
**Receiver:** any good [RTL-SDR](/reference/rtl-sdr/) (~$35) + [SMA adapter](/reference/sma-adapter-kit/).
**Total ~$120–160.** **No build decodes [encryption](/police-scanner-encryption/).**
</div>

## The complete parts list

| # | Item | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Host: laptop or Pi** | Runs GopherTrunk in the field | A laptop you own, or a [Raspberry Pi 5](/reference/raspberry-pi/) | $0–80 |
| 2 | **USB-C PD power bank** | Off-grid power for hours | [High-capacity 20–25k mAh PD bank](/hardware.html) | ~$40 |
| 3 | **Portable whip antenna** | Packs small, extends to receive | [Nagoya NA-771 telescopic whip](/reference/whip-antenna/) | ~$20 |
| 4 | **SDR dongle** | Receives the radio | [NESDR SMArt v5](/reference/nesdr/) / [RTL-SDR Blog V4](/reference/rtl-sdr/) | ~$35 |
| 5 | **SMA adapter kit** | Joins antenna to dongle | [16-piece SMA adapter kit](/reference/sma-adapter-kit/) | ~$13 |
| + | **Mag-mount antenna** (optional) | Better reception from a vehicle | [Mobile scanner antenna](/reference/mobile-scanner-antenna/) | ~$25 |
| + | **Short USB extension** (optional) | Place the dongle by a window | [Active USB cable](/reference/usb-extension-cable/) | ~$15 |

**Running total (with a laptop you own): ~$108.** Add a Pi if you want the lightest,
longest-running host, or a mag-mount if you will work from a vehicle.

## The picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Packs small</span>
<h3>Nagoya NA-771 telescopic whip</h3>
<p class="pick-card__price">around $20</p>
<p>The classic portable scanner antenna — collapses for a bag, extends for reception, and tunes usefully across the VHF/UHF scanner bands. The right first antenna for a field rig.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20" rel="nofollow sponsored noopener">NA-771 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/whip-antenna/">whip details</a> · <a href="/best-scanner-antenna/">antenna guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Always in stock</span>
<h3>NooElec NESDR SMArt v5</h3>
<p class="pick-card__price">around $35</p>
<p>Compact, rugged aluminium case, 0.5 ppm TCXO for a steady lock even as temperatures swing in the field. A great travel dongle that holds a control channel outdoors.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/nesdr/">NESDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Lightest host</span>
<h3>Raspberry Pi 5 (8GB)</h3>
<p class="pick-card__price">around $80</p>
<p>Runs for hours off a power bank and weighs almost nothing. Drive it headless from your phone or tablet over the web console — the minimum-weight, maximum-runtime portable host.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Pi 5 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/raspberry-pi/">Pi details</a> · <a href="/gophertrunk-sbc-build/">Pi build</a></p>
</div>
</div>

## Laptop or Pi — pick your host

- **Laptop** — the simplest field rig. Screen, keyboard, and battery in one; drive
  [GopherTrunk](/downloads.html) directly. A [USB-C PD power bank](/hardware.html) tops it up
  between sessions. Best when convenience matters more than weight.
- **Raspberry Pi + power bank** — the lightest, longest-running option. A
  [Pi](/reference/raspberry-pi/) sips a few watts, so a big PD bank runs it most of a day; you
  drive it headless from a phone or tablet over the [web console](/web.html). Build it out from
  the [Raspberry Pi build](/gophertrunk-sbc-build/), then just add the battery and a portable
  antenna.

## Power in the field

The battery is what makes this build portable. A **high-capacity USB-C power-delivery bank**
(20,000–25,000 mAh) runs a [Pi](/reference/raspberry-pi/)-based scanner most of a day of
intermittent use, or recharges a laptop between listening sessions. Decoding a control channel
is light electrical work — the host's idle draw, not GopherTrunk, sets your runtime. Size the
bank to your day and bring a second if you are out for a weekend.

## Antenna in the field

A collapsible [Nagoya NA-771 whip](https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20)
lives in the bag and extends when you stop to listen. Working from a vehicle? A
[magnetic-mount mobile antenna](/reference/mobile-scanner-antenna/) on the roof is a real
upgrade — the metal roof acts as a ground plane and height helps. Join whatever you bring with
the [SMA adapter kit](/reference/sma-adapter-kit/); details in the
[cables and connectors guide](/sdr-cables-and-connectors/).

## Where to buy

Start with the [**Nagoya NA-771 whip**](https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20)
(~$20) and a [**NESDR SMArt v5**](https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20)
(~$35), add a big USB-C power bank, and run it off the laptop or
[Pi](https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20) you choose as the host.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost to
you. It never changes what we recommend.*

## Bottom line

A **laptop or [Pi](/reference/raspberry-pi/) + a [power bank](/hardware.html) + a
[portable whip](/reference/whip-antenna/) + a [dongle](/best-rtl-sdr/)** is a complete field
scanner for about **$120–160** — a go-bag GopherTrunk rig for storms, events, and travel where
there is no outlet. Add a mag-mount when you have a vehicle, and remember that going portable
changes nothing about [encryption](/police-scanner-encryption/): you hear what is in the clear.
Compare the other setups on the [build-lists hub](/gophertrunk-build-lists/).
