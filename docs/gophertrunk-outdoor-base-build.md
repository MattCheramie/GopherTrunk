---
layout: page
title: "GopherTrunk Outdoor Base Station Build — Rooftop Antenna Parts List"
description: "The complete, priced parts list for a serious rooftop GopherTrunk base station — outdoor base antenna, mast, bracket, low-loss LMR-400 feedline, grounding, lightning arrestor, and mast-mounted LNA feeding a PC or Pi indoors."
keywords: GopherTrunk base station build, outdoor SDR antenna build, rooftop scanner antenna, LMR-400 feedline, discone base antenna, antenna mast install, coax lightning arrestor, mast mounted LNA, SDR grounding
permalink: /gophertrunk-outdoor-base-build/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need for a rooftop GopherTrunk base station?"
    a: "An outdoor base antenna (a wideband discone), a mast and a wall or eave bracket to mount it high, a low-loss LMR-400 feedline run indoors, a grounding kit and coax lightning arrestor for safety, a mast-mounted LNA to recover feedline loss, and a PC or Raspberry Pi indoors running GopherTrunk. Budget about $300–450 depending on feedline length."
  - q: "Why LMR-400 instead of thin coax for a base station?"
    a: "Coax loss climbs steeply with frequency, and trunked systems sit at 700/800 MHz where thin RG58 or RG316 bleeds signal badly over a long run. Low-loss LMR-400 keeps most of the signal on the wire from a rooftop antenna to an indoor receiver. On any run over about 25 feet at UHF, the cable choice matters as much as the antenna."
  - q: "Do I really need a lightning arrestor and grounding?"
    a: "For any permanently mounted outdoor antenna, yes. A coax lightning/surge arrestor at the entry point and a proper ground bond protect your equipment and, more importantly, your home from static discharge and nearby strikes. It is inexpensive insurance that is not optional on a rooftop install — follow local electrical code."
  - q: "Where does the LNA go in a base station build?"
    a: "At the antenna, at the top of the feedline — a mast-mounted LNA amplifies the signal before the coax attenuates it, recovering the feedline loss you cannot avoid on a long run. Power it up the coax with the dongle's bias tee. Do not add gain if strong local signals already threaten to overload the front end."
  - q: "Can one outdoor antenna feed both a scanner and GopherTrunk?"
    a: "Yes, through a splitter, but expect some loss and watch for overload — a strong local signal in a shared feed degrades every receiver on it. For a dedicated GopherTrunk base station, running the antenna straight to one receiver gives the best results. Splitting is covered in the multi-dongle build."
  - q: "What indoor computer does the base station feed?"
    a: "Either a PC you own or a dedicated Raspberry Pi. The base station is the RF front end; the computer is the decoder. A Pi is popular because it can sit near the feedline entry point, keeping the coax run short and letting you run GopherTrunk headless 24/7."
  - q: "Will a base station hear encrypted channels?"
    a: "No — height and gain improve range and signal quality, but no antenna or amplifier can decode AES-encrypted talkgroups. A base station dramatically improves what you hear in the clear, which in most areas is plenty of dispatch and nearly all fire and EMS."
---

# GopherTrunk Outdoor Base Station Build

**This is the flagship build: a wideband [base antenna](/best-scanner-antenna/) up on a
[mast](/reference/antenna-mast/), a low-loss [feedline](/reference/coax-feedline/) run indoors,
proper [grounding](/reference/grounding-kit/) and a
[lightning arrestor](/reference/lightning-arrestor/), and a mast-mounted [LNA](/best-sdr-lna/)
feeding a [PC or Pi](/gophertrunk-sbc-build/) running [GopherTrunk](/downloads.html).** Height
and low loss are what turn a hobby setup into a station that hears the whole system. Budget
about **$300–450**, mostly feedline and mounting.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Antenna:** outdoor wideband [discone](/reference/discone-antenna/) (~$70). **Get it high:**
[mast](/reference/antenna-mast/) + [eave/wall bracket](/reference/eave-mount/) (~$60).
**Low-loss feedline:** [LMR-400](/reference/coax-feedline/) (~$70 for 50 ft). **Safety:**
[grounding kit](/reference/grounding-kit/) + [coax lightning arrestor](/reference/lightning-arrestor/)
(~$45). **Recover feedline loss:** mast-mounted [LNA](/best-sdr-lna/) (~$30). **Indoors:** a
[PC](/gophertrunk-pc-build/) or [Pi](/gophertrunk-sbc-build/) + dongle. **No build decodes
[encryption](/police-scanner-encryption/).**
</div>

## The complete parts list

| # | Item | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Outdoor base antenna** | The whole point — height + wideband coverage | [Discone antenna](/reference/discone-antenna/) | ~$70 |
| 2 | **Antenna mast** | Raises the antenna above the roofline | [Push-up / steel mast](/reference/antenna-mast/) | ~$35 |
| 3 | **Eave / wall bracket** | Anchors the mast to the structure | [Eave mount bracket](/reference/eave-mount/) | ~$25 |
| 4 | **Low-loss feedline** | Keeps signal on the wire at UHF | [LMR-400, 50 ft, N-male](/reference/coax-feedline/) | ~$70 |
| 5 | **Coax lightning arrestor** | Protects gear + home at the entry point | [N-type surge arrestor](/reference/lightning-arrestor/) | ~$30 |
| 6 | **Grounding kit** | Bonds mast + coax shield to ground | [Ground clamp / bonding kit](/reference/grounding-kit/) | ~$15 |
| 7 | **Mast-mounted LNA** | Recovers feedline loss before it happens | [RTL-SDR Blog wideband LNA](/best-sdr-lna/) | ~$30 |
| 8 | **SDR dongle** | Receives the radio | [RTL-SDR Blog V4](/reference/rtl-sdr/) (bias tee for the LNA) | ~$35 |
| 9 | **Indoor computer** | Runs GopherTrunk | A [PC](/gophertrunk-pc-build/) or [Pi](/gophertrunk-sbc-build/) | $0–80 |
| + | **Adapters / weatherproofing** | Join it all, keep water out | [SMA kit](/reference/sma-adapter-kit/) + self-amalgamating tape | ~$20 |

**Running total: ~$330–410**, dominated by feedline length and mounting. A longer coax run or
a taller mast pushes it toward the top of the range.

## The key picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">The antenna</span>
<h3>Wideband discone antenna</h3>
<p class="pick-card__price">around $70</p>
<p>Covers roughly 25–3000 MHz — one outdoor antenna that hears VHF, UHF, and the 700/800 MHz trunked bands. Mounted high on a mast, it is the single biggest upgrade you can make to reception.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CL5ZBN94?tag=gophertrunk-20" rel="nofollow sponsored noopener">Discone on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/discone-antenna/">discone details</a> · <a href="/best-scanner-antenna/">antenna guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Keep signal on the wire</span>
<h3>LMR-400 low-loss feedline (50 ft)</h3>
<p class="pick-card__price">around $70</p>
<p>Ultra-low-loss 50-ohm coax with N-male ends. At 700/800 MHz, thin cable bleeds signal fast over a long run — LMR-400 is what makes a rooftop-to-basement run viable.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B008B49B9I?tag=gophertrunk-20" rel="nofollow sponsored noopener">LMR-400 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/coax-feedline/">feedline details</a> · <a href="/sdr-cables-and-connectors/">cable guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Recover the loss</span>
<h3>RTL-SDR Blog wideband LNA</h3>
<p class="pick-card__price">around $30</p>
<p>Low-noise amplifier, bias-tee powered up the coax. Mount it at the antenna so it lifts the signal <em>before</em> the feedline attenuates it — the correct way to fight loss on a long run.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20" rel="nofollow sponsored noopener">LNA on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/best-sdr-lna/">LNA guide</a> · <a href="/reference/bias-tee/">bias tee</a></p>
</div>
</div>

## Why height and low loss win

A base station beats a desk setup for two physical reasons: the antenna is **higher** (more
line of sight to distant sites) and the RF chain is engineered for **low loss** at UHF. Both
are things no receiver can fix downstream.

- **Get the antenna above the roofline.** A [discone](/reference/discone-antenna/) on a
  [mast](/reference/antenna-mast/) anchored by an [eave or wall bracket](/reference/eave-mount/)
  clears obstructions and adds range. Height is free signal — spend effort here first. See the
  [mast and mounting guide](/antenna-mast-and-mounting-guide/).
- **Run low-loss feedline.** [Coax loss climbs with frequency](/sdr-cables-and-connectors/),
  and trunked systems sit right where thin cable bleeds worst. Use
  [LMR-400](https://www.amazon.com/dp/B008B49B9I?tag=gophertrunk-20) for the run from roof to
  receiver, keep it as short as the layout allows, and weatherproof every outdoor junction with
  self-amalgamating tape.
- **Amplify at the antenna, not the desk.** A mast-mounted
  [LNA](https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20) recovers feedline loss because
  it lifts the signal *before* the coax attenuates it. Power it up the coax with the dongle's
  [bias tee](/reference/bias-tee/) — the [RTL-SDR Blog V4](/reference/rtl-sdr/) has one built in.
  Do not add gain if strong local signals already threaten [overload](/sdr-filters/).

## Safety — not optional on a rooftop

> **Ground it and fuse it.** Any permanently mounted outdoor antenna needs a
> [grounding kit](/reference/grounding-kit/) bonding the mast and coax shield to ground, and a
> [coax lightning/surge arrestor](/reference/lightning-arrestor/) at the point the feedline
> enters the building. This protects your equipment and your home from static buildup and nearby
> strikes. Follow local electrical code, and if you are not comfortable with the install, hire a
> pro. This is the one part of the build you do not skip or cheap out on.

## Feed a PC or a Pi indoors

The base station is the **RF front end**; the computer is the **decoder**. Run the feedline to
either a [PC you own](/gophertrunk-pc-build/) or a dedicated
[Raspberry Pi](/gophertrunk-sbc-build/) near the entry point. A Pi is popular here: place it
right where the coax comes in to keep the run short, and run [GopherTrunk](/downloads.html)
headless 24/7 through its [web console](/web.html). Want to cover several sites from this one
antenna? Split it into a [multi-dongle pool](/gophertrunk-multi-dongle-build/) — mind the
[overload](/multi-dongle-sdr-setup/) a shared feed can cause.

## Where to buy

The heart of the build is the [**discone antenna**](https://www.amazon.com/dp/B0CL5ZBN94?tag=gophertrunk-20)
(~$70) and a run of [**LMR-400 feedline**](https://www.amazon.com/dp/B008B49B9I?tag=gophertrunk-20)
(~$70), with a mast, bracket, grounding kit, arrestor, and a mast-mounted
[**LNA**](https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20) completing the front end.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CL5ZBN94?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost to
you. It never changes what we recommend.*

## Bottom line

A **high [discone](/reference/discone-antenna/) + low-loss
[LMR-400](/reference/coax-feedline/) + a mast-mounted [LNA](/best-sdr-lna/)**, grounded and
arrestor-protected, feeding a [PC](/gophertrunk-pc-build/) or [Pi](/gophertrunk-sbc-build/) is
the best-hearing GopherTrunk build there is — about **$300–450**, mostly feedline and mounting.
Spend on height and low loss first; they are what a receiver cannot fix. And remember: more
range still means more clear traffic, never [encrypted](/police-scanner-encryption/). Compare
the other setups on the [build-lists hub](/gophertrunk-build-lists/).
