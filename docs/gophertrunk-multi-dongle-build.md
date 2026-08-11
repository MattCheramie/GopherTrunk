---
layout: page
title: "GopherTrunk Multi-Dongle Build — Wideband & Multi-Site Parts List"
description: "A complete, priced parts list for a multi-SDR GopherTrunk build — a powered USB hub, several RTL-SDR dongles, antennas, a splitter, and enough compute to cover multiple sites or control channels at once."
keywords: GopherTrunk multi-dongle build, multiple RTL-SDR setup, wideband SDR build, powered USB hub SDR, multi-site scanner, SDR pool build, antenna splitter SDR, several SDR dongles, GopherTrunk multi-SDR
permalink: /gophertrunk-multi-dongle-build/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need for a multi-dongle GopherTrunk build?"
    a: "A powered (active) USB hub, two or more RTL-SDR dongles, an antenna feed for each (or one antenna and a splitter), a computer or strong SBC with enough USB and CPU, and the usual adapters. Budget about $180–260 depending on how many dongles you run. GopherTrunk coordinates the whole pool by serial and role."
  - q: "Why do I need a powered USB hub for multiple dongles?"
    a: "Several RTL-SDRs draw more current than an unpowered hub or a Raspberry Pi can safely supply, and they run warm. A quality powered (active) USB hub gives each dongle clean, stable power and physical spacing so they run cooler and couple less noise into each other. It is the foundation of a reliable multi-dongle build."
  - q: "How do I tell identical dongles apart in GopherTrunk?"
    a: "Give each RTL-SDR a unique serial with rtl_eeprom, then reference dongles by serial in GopherTrunk's config. Without unique serials the OS enumerates identical dongles in an unpredictable order, so setting serials is the first step in any multi-dongle build — then 'the control dongle' is always the same physical stick."
  - q: "Should I use several dongles or one wideband Airspy?"
    a: "It depends on how your channels are spread. A pool of narrow dongles is best when a system's sites or control channels are far apart across a band — one dongle each. A single wideband Airspy is better when the channels you want fall within one ~10 MHz window, because GopherTrunk can channelize several from that one capture."
  - q: "Can I share one antenna across multiple dongles?"
    a: "Yes, with a splitter, but expect some loss and watch for overload — a strong local signal in a shared feed degrades every dongle on it. Filter strong locals ahead of the split and keep gain conservative. For channels on very different bands, separate antennas often work better than one shared feed."
  - q: "Do I need more computer for a multi-dongle build?"
    a: "Somewhat. Each dongle adds a decode workload, so a large pool wants a capable host — a strong laptop, a desktop, or a Pi 5 for a modest pool. A single Pi 5 handles a few dongles comfortably; a big multi-site pool is happier on a desktop or a machine with plenty of USB bandwidth and cores."
  - q: "Will running more dongles let me hear encrypted channels?"
    a: "No. More receivers means more channels and sites covered at once, but no number of dongles can decode AES-encrypted talkgroups. A multi-dongle build simply hears more of what is transmitted in the clear, across more of the system, at the same time."
---

# GopherTrunk Multi-Dongle Build

**GopherTrunk can drive a whole pool of SDRs at once**, so one instance covers multiple sites
or [control channels](/reference/project-25/) simultaneously. This build is the hardware for
that: a [powered USB hub](/multi-dongle-sdr-setup/), several [dongles](/best-rtl-sdr/), an
antenna feed for each, and enough compute to keep up. [GopherTrunk is free](/downloads.html);
the pool is about **$180–260** depending on dongle count.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Foundation:** a quality **powered USB hub** (~$35) — several dongles overdraw an unpowered
one. **Receivers:** 2–4 [RTL-SDRs](/reference/rtl-sdr/) (~$35 each), each a **unique serial**
and **role**. **Antennas:** one per band, or one feed + a [splitter](/sdr-cables-and-connectors/).
**Compute:** a capable [PC](/gophertrunk-pc-build/) or [Pi 5](/gophertrunk-sbc-build/). **Watch
[overload](/sdr-filters/)** on shared feeds. **No build decodes
[encryption](/police-scanner-encryption/).**
</div>

## The complete parts list

| # | Item | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Powered USB hub** | Clean, stable power for every dongle | [10-port powered USB 3.0 hub](/multi-dongle-sdr-setup/) | ~$35 |
| 2 | **SDR dongles (×2–4)** | One per site/control channel | [RTL-SDR Blog V4](/reference/rtl-sdr/) / [NESDR](/reference/nesdr/) | ~$35 each |
| 3 | **Antennas** | Feed each band you cover | [Discone](/reference/discone-antenna/) / [dipole](/reference/dipole-antenna/) per band | ~$25–70 |
| 4 | **Antenna splitter** (if sharing) | Split one feed to several dongles | [2-way SMA splitter](/sdr-cables-and-connectors/) | ~$15 |
| 5 | **SMA adapters / jumpers** | Join hub, splitter, and dongles | [SMA adapter kit](/reference/sma-adapter-kit/) + [pigtails](/reference/coax-pigtail/) | ~$20 |
| 6 | **Capable computer** | Decodes the whole pool | A [PC](/gophertrunk-pc-build/) or [Pi 5](/gophertrunk-sbc-build/) | $0–80 |
| + | **Broadcast notch filter** (optional) | Protect a shared feed from overload | [FM block filter](/sdr-filters/) | ~$25 |
| + | **Wideband Airspy** (alternative) | Channelize several channels from one capture | [Airspy](/reference/airspy/) | ~$170 |

**Running total (hub + two dongles + shared antenna): ~$180.** Each extra dongle adds ~$35;
covering more bands adds antennas. A wideband [Airspy](/reference/airspy/) is an alternative
strategy, not an add-on — see below.

## The key picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">The foundation</span>
<h3>Powered USB 3.0 hub (10-port)</h3>
<p class="pick-card__price">around $35</p>
<p>Its own 60W supply and per-port switches give every dongle clean, stable power and physical spacing — so the host never has to source all that current and the sticks run cooler and quieter. The part a multi-dongle build is built on.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0797NZFYP?tag=gophertrunk-20" rel="nofollow sponsored noopener">Powered hub on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/multi-dongle-sdr-setup/">multi-dongle how-to</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Standardize the pool</span>
<h3>NooElec NESDR SMArt v5</h3>
<p class="pick-card__price">around $35 each</p>
<p>Its R820T2/R860 tuner is always in stock, so you can re-order matching dongles for years. The right pick when you are buying several and want them identical and re-orderable.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/nesdr/">NESDR details</a> · <a href="/best-rtl-sdr/">dongle guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Wideband alternative</span>
<h3>Airspy (wideband)</h3>
<p class="pick-card__price">around $170</p>
<p>Grabs up to ~10 MHz at once so GopherTrunk can channelize several control channels from one capture — one receiver doing the job of several dongles, when the channels fall within one window.</p>
<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+SDR+receiver&tag=gophertrunk-20" rel="nofollow sponsored noopener">Search Airspy on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/airspy/">Airspy details</a></p>
</div>
</div>

## Two ways to cover more than one channel

There are two strategies, and many builds combine them — the full treatment is in the
[multi-dongle and wideband setup guide](/multi-dongle-sdr-setup/):

1. **A pool of narrow dongles.** Each [RTL-SDR](/reference/rtl-sdr/) tunes its own frequency —
   ideal when a system's sites or control channels are spread far across a band. Assign a dongle
   per site, or one to the control channel and another to follow voice.
2. **One wideband receiver, channelized.** A single [Airspy](/reference/airspy/) captures up to
   ~10 MHz and GopherTrunk **channelizes** it, extracting several control channels from that one
   stream in software — ideal when the channels you want fall inside one window.

| Approach | Best when | Hardware |
|---|---|---|
| Dongle pool | Sites/channels spread across a band | Several [RTL-SDRs](/reference/rtl-sdr/) + powered hub |
| Wideband channelize | Channels within ~10 MHz | One [Airspy](/reference/airspy/) |
| Hybrid | Big multi-site systems | Airspy + dongles for the outliers |

## Serials, roles, and power

> **Set unique serials first.** Fresh [RTL-SDRs](/reference/rtl-sdr/) often share the same
> default serial, so the OS enumerates them in an unpredictable order. Use `rtl_eeprom` to give
> each a unique serial, then refer to each by serial in GopherTrunk. Now "the control dongle" is
> always the same physical stick, and you assign **roles** — control tracking, voice following,
> wideband capture — that GopherTrunk coordinates so grants are never missed.

The [powered hub](https://www.amazon.com/dp/B0797NZFYP?tag=gophertrunk-20) is not optional.
Several dongles draw more current than an unpowered hub — or a bare
[Raspberry Pi](/gophertrunk-sbc-build/) — can safely source, and they run warm. A good powered
hub gives each stick clean power and spacing so they stay cool and quiet.

## Overload on shared feeds

> **A strong local signal poisons every channel in a shared capture.** When you split one
> antenna across dongles, or channelize many signals out of one wideband
> [Airspy](/reference/airspy/) stream, an overloading broadcast FM, pager, or AM transmitter
> degrades *all* of them at once. Put a [broadcast notch or bandpass filter](/sdr-filters/)
> ahead of the split, keep gain conservative on wideband captures, and check the noise floor
> before blaming the decoder.

For channels on very different bands, separate antennas often beat one shared feed. When you do
share, a [2-way splitter](/sdr-cables-and-connectors/) and a filter keep the pool healthy.

## Where to buy

Build on a [**powered USB hub**](https://www.amazon.com/dp/B0797NZFYP?tag=gophertrunk-20)
(~$35), standardize on identical [**NESDR SMArt v5**](https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20)
dongles (~$35 each) so you can re-order them, and feed them from antennas per band or one shared
feed and a splitter. Prefer one wideband receiver? Step up to an
[Airspy](/reference/airspy/) instead.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0797NZFYP?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost to
you. It never changes what we recommend.*

## Bottom line

A **[powered hub](/multi-dongle-sdr-setup/) + several [dongles](/best-rtl-sdr/) with unique
serials + a capable [computer](/gophertrunk-pc-build/)** lets one GopherTrunk instance cover
multiple sites or [control channels](/reference/project-25/) at once — about **$180–260**
depending on dongle count. Feed the pool from a powered hub, [filter](/sdr-filters/) strong
locals before any split, and consider a wideband [Airspy](/reference/airspy/) if your channels
sit in one window. More receivers hear more of the *clear* system — never
[encryption](/police-scanner-encryption/). Compare the other setups on the
[build-lists hub](/gophertrunk-build-lists/).
