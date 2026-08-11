---
layout: page
title: "Antenna Mast & Mounting Hardware Guide (2026)"
description: "How to mount an outdoor scanner antenna for GopherTrunk in 2026 — masts, tripods, eave and chimney mounts, grounding, and coax lightning arrestors, with picks and a full bill of materials."
keywords: antenna mast, antenna mounting hardware, scanner antenna mount, outdoor antenna install, tripod mount, eave mount, chimney mount, antenna grounding kit, coax lightning arrestor, guy wire kit, SDR antenna mast
permalink: /antenna-mast-and-mounting-guide/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need to mount an outdoor scanner antenna?"
    a: "Four things: a way up (a mast), a way to hold the mast (a tripod, eave bracket, or chimney mount), a way to stay safe (a grounding kit and a coax lightning arrestor), and — for anything tall — a guy wire kit. A typical safe base install runs about $130–200 in hardware on top of the antenna and coax, and height is the single biggest improvement to what your SDR hears."
  - q: "How high should I mount the antenna?"
    a: "As high and clear of obstructions as you safely can. VHF/UHF reception is line-of-sight, so getting a discone above the roofline matters more than antenna gain or a bigger SDR. Twenty to thirty feet clears most single-storey rooflines; past that you are into guyed-mast territory and need to think seriously about wind load and grounding."
  - q: "Do I really need to ground the mast and add an arrestor?"
    a: "Yes, for any outdoor antenna. Grounding the mast drains static (which also lowers the noise floor) and gives surge a path to earth, and a coax lightning arrestor at the building entry protects the feedline's centre conductor. In the US, antenna grounding is required by NEC Article 810. It is cheap insurance for your gear and your house — and for real storm risk, unplug the feedline."
  - q: "Tripod, eave mount, or chimney mount — which one?"
    a: "Eave/wall bracket if you have a vertical surface near the roofline and want no roof penetration; tripod if the only clear spot is a flat roof or deck; chimney strap mount if you have a sound chimney near the ridge and want no drilling at all. All three carry the same short mast — pick by what your structure offers."
  - q: "Will a better mast help me decode encrypted systems?"
    a: "No. Height and mounting improve signal, not access. No SDR or scanner can decode AES-encrypted talkgroups, no matter how good the antenna install is. A better mast helps you hear the clear P25, DMR, NXDN, and TETRA systems in your area more reliably; it does nothing for keyed encryption."
  - q: "Can I mount an LNA on the mast?"
    a: "Yes, and it is often the right move on a long feedline. A mast-mounted low-noise amplifier applies its gain before the coax loss, preserving weak signals. Power it up the coax with a bias tee. Grounding and an arrestor matter even more once there is active electronics at the top of the mast."
---

# Antenna Mast & Mounting Hardware Guide (2026)

**The best upgrade to what your [SDR scanner](/best-sdr-for-gophertrunk/) hears is not a
bigger dongle — it is getting the antenna higher and mounting it safely.** VHF and UHF
reception is line-of-sight, so lifting a [discone](/reference/discone-antenna/) above the
roofline buys more signal than [gain](/reference/antenna-gain/), a
[filter](/sdr-filters/), or an [LNA](/best-sdr-lna/) usually can. This guide ties together the
hardware that gets an antenna up and keeps it there: the [mast](/reference/antenna-mast/), the
[mount](/reference/tripod-mount/) that holds it, the [grounding](/reference/grounding-kit/) and
[arrestor](/reference/lightning-arrestor/) that keep it safe, and the
[guying](/reference/guy-wire-kit/) that keeps it standing.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Height wins** — get the antenna above the roofline first. **Pick one support:**
[eave bracket](/reference/eave-mount/) (no roof drilling), [tripod](/reference/tripod-mount/)
(roof/deck), or [chimney mount](/reference/chimney-mount/) (no drilling at all). **Always
ground it:** a [grounding kit](/reference/grounding-kit/) plus a
[coax lightning arrestor](/reference/lightning-arrestor/) at the building entry — required by
NEC 810. **Guy anything tall** with a [guy wire kit](/reference/guy-wire-kit/). **No mast
decodes [encryption](/police-scanner-encryption/).**
</div>

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">The pole</span>
<h3>Easy Up 20' 9" push-up mast</h3>
<p class="pick-card__price">around $80</p>
<p>Telescoping steel mast that collapses to about 5 ft and extends to clear a roofline. The convenient one-person way to get real height for a discone or vertical.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B019FVJKAA?tag=gophertrunk-20" rel="nofollow sponsored noopener">Mast on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/antenna-mast/">antenna mast details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">No roof drilling</span>
<h3>20" J-pole eave bracket</h3>
<p class="pick-card__price">around $16</p>
<p>Lag it to a wall, eave, or gable and clamp a short mast. The simplest, cheapest support when you have a vertical surface near the roofline.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0785K81YJ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Bracket on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/eave-mount/">eave mount details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Safety, non-negotiable</span>
<h3>Coax lightning arrestor</h3>
<p class="pick-card__price">around $25</p>
<p>Gas-discharge arrestor at the building entry shunts surge to ground before it reaches your SDR. Pair with a grounded mast — cheap insurance for a 24/7 receiver.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CBW5TXKV?tag=gophertrunk-20" rel="nofollow sponsored noopener">Arrestor on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/lightning-arrestor/">arrestor details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Keep it standing</span>
<h3>3-way guy wire kit</h3>
<p class="pick-card__price">around $35</p>
<p>Guy ring, steel wire, turnbuckles, and clamps to brace a tall mast against wind. Needed once more than ~10–15 ft of pole is exposed above its support.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B011AEQVNY?tag=gophertrunk-20" rel="nofollow sponsored noopener">Guy kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/guy-wire-kit/">guy wire details</a></p>
</div>
</div>

## The four jobs of an outdoor install

Every safe base-antenna install does four things. Get all four and a $35 dongle on a rooftop
[discone](/reference/discone-antenna/) will run [GopherTrunk](/downloads.html) as well as far
pricier hardware.

1. **Get up** — a [mast](/reference/antenna-mast/) lifts the antenna above the roofline and
   nearby clutter. Height is the whole point.
2. **Hold on** — a [tripod](/reference/tripod-mount/), [eave bracket](/reference/eave-mount/),
   or [chimney mount](/reference/chimney-mount/) anchors the mast to the structure, with
   [U-bolts and clamps](/reference/mounting-hardware/) doing the fastening.
3. **Stay safe** — a [grounding kit](/reference/grounding-kit/) bonds the mast to earth and a
   [coax lightning arrestor](/reference/lightning-arrestor/) protects the feedline where it
   enters the building.
4. **Stay up** — a [guy wire kit](/reference/guy-wire-kit/) braces anything tall against wind.

## Bill of materials

A representative safe base install. Prices are approximate and exclude the antenna and coax
themselves — see the [scanner antenna guide](/best-scanner-antenna/) and
[coax reference](/reference/coaxial-cable/) for those.

| Part | Role | Pick | Approx price |
|---|---|---|---|
| [Mast](/reference/antenna-mast/) | Height | Easy Up 20' 9" push-up | ~$80 |
| [Eave mount](/reference/eave-mount/) *or* [tripod](/reference/tripod-mount/) *or* [chimney mount](/reference/chimney-mount/) | Support | J-pole bracket / 3' tripod / chimney Y-mount | ~$16 / ~$40 / ~$35 |
| [Mounting hardware](/reference/mounting-hardware/) | Fasten antenna & mast | V-jaw clamp + U-bolt kit | ~$13 |
| [Grounding kit](/reference/grounding-kit/) | Static + surge to earth | Ground rod, clamp, 8 AWG wire | ~$30 |
| [Lightning arrestor](/reference/lightning-arrestor/) | Protect feedline centre | Gas-discharge coax arrestor | ~$25 |
| [Guy wire kit](/reference/guy-wire-kit/) | Brace a tall mast | 3-way down guy set | ~$35 |
| [Coax](/reference/coaxial-cable/) + [N connectors](/reference/n-type-connector/) | Feedline | Low-loss run, weatherproofed | varies |

Skip the guy kit for a short mast on a solid bracket; skip nothing on the safety line. A
minimal wall-bracket build lands near **$130–160**; a taller guyed rooftop build nearer
**$180–220** before antenna and cable.

## Picking a support: eave vs tripod vs chimney

- **[Eave / J-mount bracket](/reference/eave-mount/)** — simplest and cheapest, and **no roof
  penetration**. Best when you have a wall, eave, or gable end near the roofline that you can
  lag into solid framing. Two brackets spaced apart steady a taller pole.
- **[Rooftop tripod](/reference/tripod-mount/)** — three legs lag-bolted and **sealed** through
  the roof into rafters. Best when the only clear high spot is a flat roof or deck. Its spread
  base steadies a short mast where a single pole could not stand.
- **[Chimney strap mount](/reference/chimney-mount/)** — two steel straps wrap the chimney,
  **no drilling at all**, and a chimney is often the roof's highest easy anchor. Needs a
  structurally sound chimney; space the straps as far apart as it allows.

All three carry the same short mast. If more than ~10–15 ft of pole sticks up above the
support, add a [guy wire kit](/reference/guy-wire-kit/) regardless of which support you chose.

## Do not skip grounding and the arrestor

An ungrounded rooftop antenna is a liability to your receiver and your house. Two cheap parts
close that gap and often quiet the noise floor as a bonus:

- A **[grounding kit](/reference/grounding-kit/)** — ground rod, clamp, and heavy 8 AWG (or
  larger) copper wire — bonds the [mast](/reference/antenna-mast/) to earth. Keep the run
  **short, straight, and heavy**; a fast surge sees the inductance of every bend. Bonding also
  bleeds off the static charge that builds on an isolated antenna and raises the noise floor.
- A **[coax lightning arrestor](/reference/lightning-arrestor/)** at the building entry, bonded
  to the same ground, protects the **centre conductor** of the coax — which grounding the shield
  alone does not. Its gas-discharge tube is transparent to signal and an instant short to a
  spike.

In the US this is not just good practice — antenna grounding is required by **NEC Article 810**.
And for a genuine storm overhead, the surest protection is still an unplugged feedline.

## Where GopherTrunk fits

None of this hardware talks to [GopherTrunk](/downloads.html) — the software decodes whatever
the [SDR](/best-sdr-for-gophertrunk/) captures — but the mounting stack is where most real-world
decode reliability is won. A marginal [P25](/best-sdr-for-p25-trunking/) or TETRA site that
garbles from an indoor whip often locks cleanly once the antenna clears the roofline, because
the gain is in raw carrier-to-noise, upstream of every DSP trick. Before reaching for a
[filter](/sdr-filters/) or an [LNA](/best-sdr-lna/) to chase a weak system, get the antenna
higher and mount it well. What no mast can do is defeat
[encryption](/police-scanner-encryption/) — that stays a separate, immovable wall.

## Bottom line

Buy **height first**: a [push-up mast](/reference/antenna-mast/) and a support that suits your
structure — [eave bracket](/reference/eave-mount/), [tripod](/reference/tripod-mount/), or
[chimney mount](/reference/chimney-mount/). Then buy **safety, never optional**: a
[grounding kit](/reference/grounding-kit/) and a
[coax lightning arrestor](/reference/lightning-arrestor/). Add a
[guy wire kit](/reference/guy-wire-kit/) for anything tall and keep decent
[mounting hardware](/reference/mounting-hardware/) on hand. For a complete parts list wired to
an SDR, see the [outdoor base build](/gophertrunk-outdoor-base-build/), the
[what-you-need checklist](/what-do-i-need-for-gophertrunk/), and the
[hardware setup guide](/hardware.html).
