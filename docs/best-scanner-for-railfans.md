---
layout: page
title: "Best Scanner for Railfans (2026)"
description: "The best scanner for railfans in 2026: AAR road and EOT channels live in analog VHF 160–161 MHz, so a cheap Uniden BC125AT or SR30C is ideal. AAR channel numbering explained, plus what to program."
keywords: best scanner for railfans, railfan scanner, train scanner, AAR channels, railroad frequencies, EOT frequency, Uniden BC125AT, VHF railroad scanner, road channel
permalink: /best-scanner-for-railfans/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best scanner for railfans?"
    a: "A ~$110 Uniden BC125AT. Railroad road and EOT channels are analog VHF in the 160–161 MHz band, so an inexpensive analog scanner hears everything a $600 digital scanner would — Close Call also grabs a train's active road channel automatically as it passes."
  - q: "What frequencies do railroads use?"
    a: "The Association of American Railroads (AAR) assigns 97 channels across 160.215–161.565 MHz, plus end-of-train (EOT) telemetry near 457/452 MHz. Road channels carry train-to-dispatcher voice; a railfan mainly programs the AAR road channels for the subdivision they're watching."
  - q: "Do railroads use digital or encrypted radio?"
    a: "The vast majority of North American railroad voice is still analog FM on the AAR VHF channels — an analog scanner hears it. Some yards and PTC data are digital, and a few operations encrypt, but standard road-channel dispatch voice remains in the clear analog."
  - q: "How does AAR channel numbering work?"
    a: "The AAR channel plan numbers 97 VHF channels (AAR 2 through AAR 97). Railfans quote a subdivision's frequency as its AAR number — a railroad might use 'AAR 66' (161.010 MHz). Programming by AAR number and by frequency both land on the same channel."
  - q: "Is Close Call useful for railfans?"
    a: "Very. Close Call detects a strong nearby transmission and instantly tunes it, so when a train passes on an unknown road channel the BC125AT grabs it automatically — no need to know the subdivision's assignment in advance."
---

# Best Scanner for Railfans (2026)

**Railfanning is the rare hobby where the cheapest scanner is also the best one.**
North American railroad voice lives on analog VHF channels that a $110 radio hears
perfectly — spending more buys digital decoding you'll never use trackside. Here's
the ideal railfan setup and how the AAR channel plan works.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best railfan scanner:** [Uniden BC125AT](/reference/uniden-bc125at/) (~$110) —
Close Call grabs a passing train's channel automatically. **Budget alternative:**
[Uniden SR30C](/reference/uniden-sr30c/). **Where trains live:** analog VHF
**160.215–161.565 MHz** (AAR channels) + EOT telemetry. **Mostly analog and in the
clear** — a digital scanner is wasted money here.
</div>

## Why analog wins for railfans

The [Association of American Railroads](https://www.aar.org/) assigns a fixed VHF
channel plan, and railroad **road channels — the train-to-dispatcher voice you came
to hear — are analog FM**. That means an inexpensive analog scanner receives
exactly what a $600 [digital P25](/reference/project-25/) unit would trackside.
Paying for digital, DMR, or NXDN decoding you'll never key is the classic railfan
overspend. Check the [frequency guide](/scanner-frequencies/) for the bands.

## The AAR channel plan, explained

The AAR numbers **97 channels (AAR 2 through AAR 98)** across **160.215 to 161.565
MHz**, spaced 15 kHz apart. Railfans refer to a subdivision by its AAR number as
shorthand:

- **AAR channel number** ↔ **frequency** are two names for the same slot. "AAR 66"
  is 161.010 MHz; program either and you land on the same channel.
- **Road channels** carry mainline train-to-dispatcher voice — the primary railfan
  target.
- **Yard / switching channels** are separate AAR assignments used within terminals.
- **EOT (end-of-train) telemetry** near 457/452 MHz reports the rear-unit brake
  pressure — data bursts, not voice, but a sign a train is near.

A subdivision typically uses one or two road channels; look up the line you're
watching on [RadioReference](https://www.radioreference.com/) and program those.

## Top picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best for railfans</span>
<h3>Uniden BC125AT</h3>
<p class="pick-card__price">around $110</p>
<p>The trackside classic. Covers the full AAR VHF band, and Close Call instantly grabs a passing train's road channel even if you don't know the subdivision's assignment.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-bc125at/">BC125AT details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Budget alternative</span>
<h3>Uniden SR30C</h3>
<p class="pick-card__price">around $120</p>
<p>Simple analog handheld that covers the AAR band. A fine grab-and-go if you want a second radio or a spare in the bag.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-sr30c/">SR30C details</a></p>
</div>
</div>

## Railfan scanner comparison

| Scanner | Covers AAR VHF | Close Call | Digital (wasted here) | Approx. price |
|---|---|---|---|---|
| [BC125AT](/reference/uniden-bc125at/) | **Yes** | **Yes** | No | ~$110 |
| [SR30C](/reference/uniden-sr30c/) | **Yes** | Basic | No | ~$120 |
| A $500 digital scanner | Yes | Yes | Yes (unused) | ~$500+ |

> **You don't need digital for trains.** Standard road-channel dispatch is analog
> and in the clear. A [BC125AT](/reference/uniden-bc125at/) hears it as well as any
> flagship. Some yards run digital or PTC data, and a rare operation encrypts, but
> for mainline railfanning analog is the whole game.

## Grabbing a channel with Close Call

The BC125AT's **Close Call** feature detects a strong nearby signal and tunes it
instantly. Trackside, that means when a train rolls by on a road channel you
haven't programmed, the scanner **catches it automatically** — you don't need the
subdivision's AAR assignment memorized. Set Close Call to the VHF band and let it
sweep as trains approach.

## Logging trains with GopherTrunk

Railfans who like to document runs can add a [$30 SDR](/reference/rtl-sdr/) with
free [GopherTrunk](/downloads.html) on a laptop to **record and timestamp every
road-channel transmission**, so you can match audio to a passing consist later. It
even monitors many AAR channels at once — handy on a busy junction — where a single
handheld can only follow one. It needs a running PC, so it's a complement to the
trackside [BC125AT](/reference/uniden-bc125at/), not a replacement; see
[scanner vs SDR](/police-scanner-vs-sdr/).

## Bottom line

For railfans, the **[Uniden BC125AT](/reference/uniden-bc125at/)** is the best
scanner precisely because it's cheap — railroad road channels are analog VHF in the
160–161 MHz AAR band, so a $110 radio hears everything a flagship would, and Close
Call grabs passing trains automatically. Look up your subdivision's
[AAR frequencies](/scanner-frequencies/), skip the digital premium, and save the
money for a trip to the mainline. For the full lineup, see the
[best police scanners guide](/best-police-scanners/).
