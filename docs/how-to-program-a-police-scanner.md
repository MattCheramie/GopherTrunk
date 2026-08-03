---
layout: page
title: "How to Program a Police Scanner"
description: "How to program a police scanner three ways — ZIP-code/HomePatrol, Sentinel/EZ-Scan software with an SD card, or manual frequency and talkgroup entry — plus how GopherTrunk imports a config."
keywords: how to program a police scanner, program scanner by zip code, Uniden Sentinel, HomePatrol programming, manual scanner programming, talkgroup programming, RadioReference download
permalink: /how-to-program-a-police-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "How do I program a police scanner?"
    a: "Three routes. Easiest: enter your ZIP code on a HomePatrol-database Uniden (BCD436HP, BCD536HP, SDS100, SDS200) and it loads nearby systems automatically. Better control: use Uniden Sentinel (or Whistler EZ-Scan) on a PC to build a favorites list and copy it to an SD card. Full control: enter frequencies and talkgroups by hand from RadioReference."
  - q: "How do I program a scanner by ZIP code?"
    a: "On a HomePatrol-based Uniden, choose the location/ZIP setup, type your ZIP or let GPS set it, pick a monitoring range, and the built-in database loads the nearby systems and talkgroups. No frequency tables required. You can then trim which agencies you hear."
  - q: "What is Uniden Sentinel?"
    a: "Sentinel is Uniden's free PC software for the 436/536HP and SDS scanners. It keeps the HomePatrol database updated, builds and edits Favorites Lists, and writes them to the scanner's SD card over USB. Whistler's equivalent for TRX radios is EZ-Scan."
  - q: "Where do I get frequencies and talkgroups?"
    a: "RadioReference.com is the standard source. It lists systems, frequencies, talkgroup IDs, and modes for your county. A premium membership lets Sentinel import that data in bulk instead of typing it in manually."
  - q: "How do I program a trunked system manually?"
    a: "Create the system, enter its control-channel frequencies (or all site frequencies), set the system type (P25, DMR, NXDN), then add the talkgroup IDs you want with their names. The scanner reads the control channel and follows those talkgroups. For DMR/NXDN you may also need the color code or RAN/NAC."
  - q: "Does GopherTrunk need programming like a scanner?"
    a: "GopherTrunk uses a config file instead of on-radio menus. You point it at a system's control-channel frequency and list the talkgroups (or let it log everything), and it decodes and records automatically. You can import site and talkgroup data rather than typing it in — see the import page."
---

# How to Program a Police Scanner

**Programming a scanner just means telling it which systems and talkgroups to listen
to — and modern radios make that as easy as typing your ZIP code.** There are three
routes, from zero-effort to full manual control, plus GopherTrunk's config-file
approach if you're running an [SDR](/reference/software-defined-radio/) instead.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Route 1 — ZIP code:** HomePatrol-database Unidens
([BCD436HP](/reference/uniden-bcd436hp/), [BCD536HP](/reference/uniden-bcd536hp/),
[SDS100](/reference/uniden-sds100/), [SDS200](/reference/uniden-sds200/)) load nearby
systems from a ZIP — no frequency tables. **Route 2 — software:** Uniden **Sentinel**
(or Whistler **EZ-Scan**) builds favorites on a PC → SD card. **Route 3 — manual:**
enter frequencies and [talkgroups](/reference/talkgroup/) by hand. Source data:
[RadioReference](https://www.radioreference.com/). **GopherTrunk** uses a
[config file / import](/import.html) instead.
</div>

## First: get the data (RadioReference)

Every route needs the same raw material — your county's systems, frequencies, and
talkgroups. The standard source is
[RadioReference.com](https://www.radioreference.com/): find your state → county, and
note the system type, [control-channel](/reference/control-channel/) frequencies,
[talkgroup](/reference/talkgroup/) IDs, and mode ([P25](/reference/project-25/),
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/)). Browsing is free; a small premium
membership unlocks **bulk downloads** that Sentinel can import automatically, saving a
lot of typing.

> **Know the system before you program it.** Whether it's
> [trunked](/reference/trunked-radio/), digital, or
> [simulcast](/reference/simulcast/) determines what you enter — start with
> [how police scanners work](/how-police-scanners-work/) and the
> [buying guide](/scanner-buying-guide/).

## Route 1: ZIP code (easiest — HomePatrol database)

The simplest scanners to program are Uniden's **HomePatrol-database** models:
[BCD436HP](/reference/uniden-bcd436hp/), [BCD536HP](/reference/uniden-bcd536hp/),
[SDS100](/reference/uniden-sds100/), [SDS200](/reference/uniden-sds200/), and the
[HomePatrol-2](/reference/uniden-home-patrol-2/). They ship with a nationwide database
baked in.

1. **Enter your location.** Choose the ZIP-code/location setup and type your ZIP (or
   let GPS set it on models that support it).
2. **Pick a range.** Set how far out to monitor (e.g. your city vs the whole metro).
3. **Let it load.** The scanner pulls the nearby systems and talkgroups from its
   database and starts scanning immediately.
4. **Trim it.** Use avoid/lockout to hide agencies you don't care about.

That's it — no frequency tables, no talkgroup IDs typed by hand. This is why these
radios are the [beginner recommendation](/best-police-scanners/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check BCD436HP price on Amazon &rarr;</a>

## Route 2: software + SD card (Sentinel / EZ-Scan)

For more control — custom favorites lists, edited talkgroup names, quick database
updates — program on a PC and copy to the scanner:

- **Uniden Sentinel** (free) for the 436/536HP and SDS series. It keeps the
  HomePatrol database current, builds **Favorites Lists**, and writes them to the
  radio's SD card over USB.
- **Whistler EZ-Scan** for the [TRX-1](/reference/whistler-trx-1/) /
  [TRX-2](/reference/whistler-trx-2/), which include [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/).

Typical flow: update the database → import your county (or a RadioReference download)
→ curate which systems and talkgroups you keep → save to SD → reinsert in the
scanner. This is the sweet spot for most enthusiasts: database convenience with
hand-tuned lists.

## Route 3: manual frequency and talkgroup entry

Older or value radios ([BCD996P2](/reference/uniden-bcd996p2/),
[BCD325P2](/reference/uniden-bcd325p2/), [BC125AT](/reference/uniden-bc125at/)) are
programmed channel by channel — by hand or with free third-party software.

**Conventional (analog or digital):**

1. Create a system/bank.
2. Enter each **frequency** and, for digital, the mode.
3. Name the channels.

**Trunked:**

1. Create the trunked system and set its type ([P25](/reference/project-25/), DMR, NXDN).
2. Enter the **[control-channel](/reference/control-channel/)** frequencies (or all
   site frequencies).
3. Add the **[talkgroup](/reference/talkgroup/) IDs** you want, with names.
4. For DMR/NXDN, add the **color code** or **RAN/[NAC](/reference/network-access-code/)**
   where required.

The scanner then reads the control channel and follows those talkgroups. Third-party
software (e.g. ARC/ProScan for Uniden) makes bulk entry far less tedious than the
keypad.

## Route comparison

| Route | Effort | Control | Best for | Radios |
|---|---|---|---|---|
| **ZIP code** | **Lowest** | Low | Beginners, fast start | 436/536HP, SDS, HomePatrol-2 |
| **Sentinel / EZ-Scan** | Medium | **High** | Enthusiasts, curated lists | 436/536HP, SDS, TRX |
| **Manual entry** | High | **Full** | Value radios, precise setups | 996P2, 325P2, BC125AT |
| **[GopherTrunk](/import.html)** | Config file | **Full + logging** | PC/SDR users | [RTL-SDR](/reference/rtl-sdr/) |

## How GopherTrunk does it instead

Running an [SDR](/reference/software-defined-radio/) with
[GopherTrunk](/downloads.html)? There are **no on-radio menus** — you edit a **config
file** (or use the guided setup) and point it at a system:

- **Give it the [control channel](/reference/control-channel/).** Set the system's
  control-channel frequency and type; GopherTrunk finds the grants and follows the
  calls automatically.
- **List talkgroups — or don't.** Name the [talkgroups](/reference/talkgroup/) you
  care about, or let it **log everything** and sort it out later. Every call is
  recorded and timestamped.
- **Import instead of retype.** Bring in site and talkgroup data rather than entering
  it by hand — see the [import guide](/import.html) and [downloads](/downloads.html) to
  get set up.

One config drives unlimited talkgroups and multiple systems at once, with a
[web console](/web.html) to manage it. The trade-offs vs a hardware scanner are laid
out in [scanner vs SDR](/police-scanner-vs-sdr/).

## A note on encryption

No amount of correct programming makes an **[encrypted](/police-scanner-encryption/)**
talkgroup audible. You can enter it perfectly and still get silence — AES can't be
decoded by any scanner or SDR. Program the **clear** talkgroups and don't chase the
locked ones.

## Bottom line

**Programming is just telling the radio what to listen to, and the modern easy path is
a ZIP code.** Use [RadioReference](https://www.radioreference.com/) for the data, then
pick your route: ZIP-code loading on a [436/536HP or SDS](/best-police-scanners/),
curated lists via **Sentinel**/**EZ-Scan** to an SD card, or full manual entry on a
value radio. On an [SDR](/reference/rtl-sdr/), skip the menus entirely — point
[GopherTrunk](/downloads.html) at a [control channel](/reference/control-channel/),
[import](/import.html) your talkgroups, and it records every call. Just remember that
no programming route unlocks [encryption](/police-scanner-encryption/).
