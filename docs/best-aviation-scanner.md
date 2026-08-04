---
layout: page
title: "Best Aviation Scanner (Airband)"
description: "The best aviation scanner for airband monitoring in 2026 — the Uniden BC125AT is the classic AM airband handheld for tower, ground, approach and ATC, plus honest picks for pilots and plane spotters."
keywords: best aviation scanner, best airband scanner, air traffic control scanner, BC125AT airband, AM airband scanner, ATC scanner, tower ground approach scanner, aircraft radio scanner
permalink: /best-aviation-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best aviation scanner?"
    a: "For most listeners the Uniden BC125AT is the best aviation scanner — it covers the full 108–137 MHz civil airband in AM mode, has 500 channels, close-call band search, and costs around $110. Airband is amplitude-modulated, so a radio that only does FM will hear nothing."
  - q: "Why do aviation scanners need AM, not FM?"
    a: "Civil aviation voice on 108–137 MHz uses amplitude modulation (AM), a deliberate choice so overlapping transmissions still produce an audible 'heterodyne' rather than one signal capturing the channel. A scanner locked to FM in this band decodes only silence, so AM airband support is the single must-have feature."
  - q: "What frequencies do I program for ATC?"
    a: "Tower, ground, approach/departure, ATIS, and center each have their own frequency at every airport — look them up on RadioReference or LiveATC by airport identifier. Ground is typically 121.x MHz, tower 118–119 MHz, ATIS 127–128 MHz, but they vary by field."
  - q: "Can I hear aircraft with an SDR instead?"
    a: "Yes. A $30 RTL-SDR dongle receives AM airband and, with dump1090-style tools, ADS-B position data at 1090 MHz too. It is the cheapest way to both listen and plot aircraft on a map, though a handheld is far more portable at the fence line."
  - q: "Is listening to air traffic control legal?"
    a: "Receiving unencrypted aviation voice is legal in the United States. Airband is never encrypted, so unlike public-safety radio you will always hear live traffic. Rules on in-vehicle scanner use vary by state; this is not legal advice."
---

# Best Aviation Scanner (Airband)

**The best aviation scanner is any radio that covers 108–137 MHz in AM — and the
one nearly every plane spotter starts with is the [Uniden BC125AT](/reference/uniden-bc125at/).**
Airband is the one detail that trips up first-time buyers: civil aviation voice
is **amplitude-modulated (AM)**, not FM. A scanner that only demodulates FM will
sit dead silent on the tower frequency. Confirm AM airband before anything else.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best overall:** [Uniden BC125AT](/reference/uniden-bc125at/) — full AM airband,
500 channels, ~$110. **Also great:** [BC75XLT](/reference/uniden-bc75xlt/) for a
lighter pocket handheld. **Cheapest way to also plot planes:**
[$30 RTL-SDR](/reference/rtl-sdr/) for airband voice **and** ADS-B. **Must-have
spec:** AM demodulation across **108–137 MHz** — FM-only radios hear nothing.
</div>

## Why airband is different

Every other band a scanner touches — public safety, marine, ham — is FM. Aviation
kept **AM** on purpose. When two pilots key up at once, AM lets the tower hear a
warbling "heterodyne" and ask for a repeat; two FM signals would simply capture
one another and hide the collision. That legacy choice is why your scanner's mode
must be AM, and why a general-purpose FM handheld is useless at the airport fence.

The civil band runs roughly **108–137 MHz**: 108–118 MHz is navigation (VOR/ILS),
and **118–137 MHz** is the voice range you actually want — tower, ground,
clearance, approach/departure, ATIS, and en-route center.

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best overall</span>
<h3>Uniden BC125AT</h3>
<p class="pick-card__price">around $110</p>
<p>The classic airband handheld. Full AM 108–137 MHz, 500 channels, close-call search to grab a nearby transmitter you didn't program.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-bc125at/">BC125AT details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Lightest handheld</span>
<h3>Uniden BC75XLT</h3>
<p class="pick-card__price">around $100</p>
<p>Smaller, weather-alert handheld that still covers AM airband. Fewer channels than the BC125AT but easy to pocket at an airshow.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-bc75xlt/">BC75XLT details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Listen + plot planes</span>
<h3>RTL-SDR + GopherTrunk</h3>
<p class="pick-card__price">around $30</p>
<p>A dongle hears AM airband voice and, with ADS-B tools, plots aircraft at 1090 MHz on a live map from the same antenna.</p>
<a class="btn btn--buy" href="/downloads.html">Get GopherTrunk (free) &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a></p>
</div>
</div>

## Comparison

| Scanner | Form | AM airband | Extras | Approx. price |
|---|---|---|---|---|
| [Uniden BC125AT](/reference/uniden-bc125at/) | Handheld | **Yes (108–137 MHz)** | 500 ch, close call, alpha tags | ~$110 |
| [Uniden BC75XLT](/reference/uniden-bc75xlt/) | Handheld | **Yes** | Weather alert, lighter | ~$100 |
| [Uniden SR30C](/reference/uniden-sr30c/) | Handheld | Yes | Simpler UI, fewer banks | ~$120 |
| **[RTL-SDR](/reference/rtl-sdr/) + [GopherTrunk](/downloads.html)** | PC + dongle | **Yes** | + **ADS-B** mapping, recording | ~$30 |

> **AM is the deal-breaker.** Before you buy any scanner "for aviation," verify it
> demodulates AM across 108–137 MHz. Some budget FM-only radios advertise the
> frequency range but not the mode — and they will hear nothing but static on the
> tower.

## What to actually listen to

- **Tower** — takeoff and landing clearances (typically 118–119 MHz).
- **Ground** — taxi instructions (typically 121.x MHz).
- **Approach / departure** — radar control near the terminal area.
- **ATIS** — the recorded weather/runway loop (127–128 MHz range).
- **Center** — en-route control between airports.

Look up exact frequencies for your field by airport identifier on
[RadioReference](https://www.radioreference.com/), then plug them into your
[scanner's channel list](/scanner-frequencies/). A close-call or band-search mode
(the BC125AT has one) will also catch a strong nearby transmitter you forgot to
program.

## Where an SDR pulls ahead

Aviation is the one hobby where a computer genuinely beats a handheld on features:
the same [$30 RTL-SDR](/reference/rtl-sdr/) that hears AM voice can also decode
**ADS-B** at 1090 MHz and plot every aircraft in range on a live map. Run
[GopherTrunk](/downloads.html) on a PC you already own and you get airband audio
plus recorded, timestamped logs. The trade-off is portability — nothing beats a
BC125AT clipped to your belt at the fence. See our
[scanner vs SDR comparison](/police-scanner-vs-sdr/) for the full picture.

## Bottom line

For pure listening, the **[Uniden BC125AT](/reference/uniden-bc125at/)** is the
best aviation scanner for the money — it does AM airband right, it's portable, and
it's about $110. If you also want to *see* the planes you're hearing, add a
**[$30 RTL-SDR](/reference/rtl-sdr/)** and free
[GopherTrunk](/downloads.html) for ADS-B mapping. Either way, the only spec that
matters first is **AM support across 108–137 MHz** — get that wrong and the radio
never makes a sound.
