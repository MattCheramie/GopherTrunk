---
layout: page
title: "Police, Fire & EMS Scanner Frequencies by Service"
description: "A reference of US scanner frequencies by service: police/fire/EMS VHF/UHF and 700/800 MHz, aviation, marine VHF, railroad AAR channels, NOAA weather, FRS/GMRS and CB — plus finding your local ones."
keywords: scanner frequencies, police scanner frequencies, fire scanner frequencies, EMS frequencies, NOAA weather frequencies, marine VHF frequencies, railroad frequencies, aviation frequencies, 800 MHz public safety
permalink: /scanner-frequencies/
nav_group: Hardware
affiliate: true
faq:
  - q: "What frequencies do police use?"
    a: "US police operate across several bands depending on the era and area: VHF high band (136–174 MHz), UHF (450–470 MHz), and 700/800 MHz public-safety trunked systems (769–775/799–805 and 806–824/851–869 MHz). There is no single 'police frequency' — it varies by agency, so look yours up on RadioReference."
  - q: "What frequency is NOAA weather radio?"
    a: "NOAA Weather Radio broadcasts on seven channels between 162.400 and 162.550 MHz (162.400, 162.425, 162.450, 162.475, 162.500, 162.525, 162.550). Every modern scanner and most weather radios receive them, and each transmitter uses whichever channel avoids overlap with neighbors."
  - q: "What frequencies do fire and EMS use?"
    a: "Fire and EMS appear on VHF low band (about 33–46 MHz on older systems), VHF high band (150–174 MHz, common for fireground), UHF (450–470 MHz), and 700/800 MHz trunked systems. EMS often also uses the VHF/UHF MED channels for hospital coordination."
  - q: "Can I scan aviation and marine on a police scanner?"
    a: "Yes, if the scanner covers those bands and modes. Aviation uses AM between 108–137 MHz, and marine uses FM on VHF 156–162 MHz. Most wideband scanners like the Uniden BC125AT cover both; make sure the radio does AM for air band."
  - q: "How do I find the exact frequencies for my area?"
    a: "Use RadioReference.com — search your county to get the actual agency frequencies, trunked system talkgroups, and encryption status. The band plans on this page tell you where to look; RadioReference tells you exactly what your local agencies use."
  - q: "What are FRS and GMRS frequencies?"
    a: "FRS and GMRS share 22 channels in the 462–467 MHz UHF range, used by consumer 'walkie-talkie' radios. They're unencrypted and easy to scan, useful for events, stores, and neighborhood traffic."
---

# Police, Fire & EMS Scanner Frequencies by Service

**There is no single "police frequency" — US public safety and the services
around it are spread across a dozen bands from 30 MHz to 900 MHz, and knowing
*which band a service lives on* is what turns a scanner from a toy into a tool.**
This page is a band-plan reference; for the exact channels your local agencies
use, pair it with [RadioReference](/reference/radioreference/).

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Public safety** lives on VHF high (136–174), UHF (450–470), and 700/800 MHz
trunked systems. **Aviation** is AM 108–137, **marine** is FM 156–162,
**railroad** uses the AAR channels near 160–161, **NOAA weather** sits at
162.400–162.550. **[Frequency bands](/reference/frequency-bands/)** determine what
your scanner needs to cover. **Look up your exact channels** on
[RadioReference](/reference/radioreference/) — this page tells you where to look.
</div>

## Public-safety bands at a glance

Police, fire, and EMS have migrated across bands over the decades. Older systems
sit low; modern trunked systems sit high.

| Band | Range | Typical mode | Who's there |
|---|---|---|---|
| **VHF low** | 30–50 MHz | Analog FM | Legacy fire/EMS, some rural, state agencies |
| **VHF high** | 136–174 MHz | FM / P25 | Fire, EMS, police, public works — very common |
| **UHF** | 450–470 MHz | FM / P25 / DMR | Police, fire, hospitals, business |
| **UHF-T** | 470–512 MHz | Trunked | Urban public safety in some metros |
| **700 MHz** | 769–775 / 799–805 | [P25](/reference/project-25/) trunked | Modern interoperable public safety |
| **800 MHz** | 806–824 / 851–869 | P25 / [SmartNet](/reference/smartnet-smartzone/) | Big-city trunked systems |

> **Trunked vs. conventional.** On VHF/UHF you'll often find *conventional*
> channels — one frequency, one purpose. On 700/800 MHz you'll almost always find
> *[trunked](/reference/trunked-radio/)* systems that pool many
> [talkgroups](/reference/talkgroup/) across a few frequencies, which is why a
> trunk-tracking scanner (or GopherTrunk) matters there.

## Police, fire & EMS detail

Because agencies choose their own band, the same city can have police on 800 MHz
and volunteer fire on VHF. Expect a mix.

| Service | Common bands | Notes |
|---|---|---|
| **Police** | VHF high, UHF, 700/800 MHz | Increasingly P25 trunked; some [encrypted](/police-scanner-encryption/) |
| **Fire dispatch** | VHF high, UHF, 700/800 MHz | Often stays clear — see [fire & EMS scanner](/fire-ems-scanner/) |
| **Fireground / tactical** | VHF high (151–159), UHF | Simplex on scene; the audio you want at a working fire |
| **EMS / ambulance** | VHF, UHF, 700/800 MHz | Plus the MED channels below |
| **EMS MED channels** | ~155 MHz and 462/467 UHF MED | Hospital ↔ ambulance coordination |

Many of these carry an analog [CTCSS/DCS](/reference/ctcss/) sub-tone; a scanner
receives them regardless, but knowing the tone helps identify a channel.

## Aviation, marine, rail, weather

The non-public-safety spectrum is where the reliably-unencrypted action is — none
of these is normally encrypted.

| Service | Frequencies | Mode | Notes |
|---|---|---|---|
| **Aviation (air band)** | 108–137 MHz | **AM** | Towers, approach, en-route; needs an AM-capable scanner |
| **Marine VHF** | 156.000–162.025 MHz | FM | [Marine VHF](/reference/marine-vhf/); Ch 16 (156.800) is distress/calling |
| **Railroad (AAR)** | ~160.215–161.565 MHz | FM | 97 AAR channels; road, yard, and defect-detector traffic |
| **NOAA weather** | 162.400–162.550 MHz | FM | Seven channels; continuous forecasts and alerts |

**NOAA weather channels** in full: 162.400, 162.425, 162.450, 162.475, 162.500,
162.525, and 162.550 MHz. Any nearby transmitter uses one of these; your scanner's
"WX" button just cycles them.

## Consumer & business bands

Easy listening that's always in the clear — good for events, neighborhoods, and
testing a new antenna.

| Service | Frequencies | Notes |
|---|---|---|
| **FRS / GMRS** | 462–467 MHz (22 channels) | Consumer walkie-talkies; GMRS is licensed, higher power |
| **MURS** | 151.820–154.600 MHz (5 channels) | License-free VHF business/personal |
| **CB radio** | 26.965–27.405 MHz (40 channels) | AM (and SSB); trucker and hobby traffic |
| **Business / itinerant** | VHF & UHF (e.g. 151, 154, 464, 469) | Stores, security, construction |

## How to find *your* local frequencies

Band plans tell you where to point the radio; they don't tell you what your city
dispatches on. For that, one resource does the job.

- **[RadioReference](/reference/radioreference/) is the database.** Search your
  county to get actual agency frequencies, trunked
  [system IDs](/reference/system-id/) and [talkgroups](/reference/talkgroup/),
  modes, and — critically — **encryption status** per channel.
- **Note the four facts that matter:** analog vs. digital, conventional vs.
  [trunked](/reference/trunked-radio/), [simulcast](/reference/simulcast/) or not,
  and clear vs. [encrypted](/police-scanner-encryption/). Those decide what
  hardware you need.
- **Match the scanner to the plan.** If everything local is analog VHF/UHF, a
  ~$110 [Uniden BC125AT](https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20)
  covers it. If it's P25 trunked, you need a digital trunk-tracker like the
  [BCD436HP](https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20) — or an SDR
  running [GopherTrunk](/police-scanner-vs-sdr/). See the full
  [best police scanners](/best-police-scanners/) guide.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check BC125AT price on Amazon &rarr;</a>

> **Program from the database, not by hand.** Uniden's ZIP-code scanners and
> GopherTrunk both pull from public frequency data, so you rarely need to type
> [frequencies](/reference/frequency/) manually — but understanding the band plan
> helps you know what you're hearing.

## Bottom line

US public safety spans **VHF low, VHF high, UHF, and 700/800 MHz**, while the
reliably-clear services — **aviation (AM 108–137), marine (156–162), railroad
(160–161), and NOAA weather (162.400–162.550)** — round out a rich listening
picture. Use this reference to know *where* each service lives, then look up the
*exact* channels for your county on [RadioReference](/reference/radioreference/),
and match your radio to what's actually on the air. Worried some of it is locked?
Read [police scanner encryption explained](/police-scanner-encryption/).
