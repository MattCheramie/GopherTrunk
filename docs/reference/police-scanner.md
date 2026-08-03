---
slug: police-scanner
title: Police scanner
entry_type: concept
category: consumer-scanners
description: "A police scanner is a radio receiver that automatically sweeps public-safety, business, and hobby frequencies so you can listen to police, fire, and EMS dispatch in real time — analog or digital."
keywords: police scanner, scanner radio, public safety scanner, trunking scanner, digital scanner, P25 scanner, how police scanners work, listen to police radio
aka: [scanner radio, public-safety scanner, radio scanner]
autolink: true
infobox:
  - { label: What it is, value: Auto-scanning radio receiver }
  - { label: Listens to, value: "Police, fire, EMS, aviation, marine, rail" }
  - { label: Signal types, value: "Analog FM + digital (P25/DMR/NXDN)" }
  - { label: System types, value: "Conventional + trunked" }
  - { label: Cannot do, value: Decode encrypted (AES) traffic }
  - { label: Free alternative, value: "SDR + GopherTrunk" }
see_also: [trunking-scanner, trunked-radio, project-25, p25-phase-2, simulcast, talkgroup, control-channel, uniden-sds200, whistler-trx-2]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Scanner_(radio)
  - https://wiki.radioreference.com/index.php/APCO_Project_25
faq:
  - q: "What is a police scanner?"
    a: "A police scanner is a radio receiver that rapidly sweeps a list of frequencies and stops on any that are active, letting you hear live police, fire, and EMS dispatch. Modern scanners also decode digital voice (P25, DMR, NXDN) and follow trunked-radio systems."
  - q: "Can a police scanner hear encrypted police?"
    a: "No. When an agency encrypts its radio traffic with AES-256, no consumer scanner — and no software-defined radio, including GopherTrunk — can decode it. Encryption is a cryptographic and legal wall (18 U.S.C. §2512). You can still hear the many agencies, plus most fire and EMS, that transmit in the clear."
  - q: "Is it legal to own a police scanner?"
    a: "In the United States, owning a scanner and listening to unencrypted public-safety radio is legal at the federal level. A handful of states restrict using a scanner in a moving vehicle or while committing a crime. See our guide on scanner legality — it is general information, not legal advice."
  - q: "Do I need a digital scanner?"
    a: "If your local police, fire, or EMS have moved to a digital P25, DMR, or NXDN system — most metro areas have — then yes, an analog-only scanner will hear nothing but static on those channels. Check your county on RadioReference before buying."
  - q: "What is the cheapest way to start scanning?"
    a: "A ~$30 RTL-SDR USB dongle plus the free, open-source GopherTrunk software will follow and decode digital trunked systems on a PC you already own. A dedicated handheld scanner is more convenient and portable but starts around $100 for analog and $380+ for digital."
---
**A police scanner** is a radio receiver that automatically sweeps a list of
frequencies and stops on whichever one is active, so you can listen to live
police, fire, and emergency-medical dispatch as it happens.[^wiki] Modern
scanners go far beyond the old "hear a channel" radio: they decode
[digital voice](/reference/project-25/) and follow
[trunked-radio](/reference/trunked-radio/) systems that hop across dozens of
channels.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A police scanner listens to public-safety radio automatically.** It hears
[police, fire, and EMS](/scanner-frequencies/) plus aviation, marine, and rail.
**You need a *digital* scanner** if your agencies run
[P25](/reference/project-25/), [DMR](/reference/dmr/), or
[NXDN](/reference/nxdn/) — most cities do. **No scanner can decode
[encryption](/police-scanner-encryption/) (AES).** **The free alternative** is an
[SDR + GopherTrunk](/police-scanner-vs-sdr/): a ~$30 dongle and open-source
software that records and decodes everything a mid-range scanner does.
</div>

## What a scanner actually does

A traditional receiver sits on one frequency. A scanner **cycles through a
programmed list many times a second** and pauses whenever it detects a
transmission — that is the "scan." When the exchange ends, it resumes sweeping.
That single behavior is why one radio can cover an entire county's dispatch,
fireground, EMS, and public-works channels at once.

Three capabilities separate a modern scanner from a toy:

- **Digital decode.** Public safety has largely migrated from analog FM to
  digital modes — chiefly [APCO Project 25 (P25)](/reference/project-25/), plus
  [DMR](/reference/dmr/) and [NXDN](/reference/nxdn/) on business and some public
  systems. A digital scanner demodulates those bitstreams back into voice.
- **Trunk tracking.** On a [trunked system](/reference/trunked-radio/), a talkgroup
  does not own a channel — a [control channel](/reference/control-channel/) assigns
  a free voice channel for each transmission. A trunking scanner reads that control
  channel and follows the [talkgroup](/reference/talkgroup/) as it hops, so a
  conversation sounds continuous.
- **Smart programming.** The best units program from your **ZIP code** against a
  built-in database of every known system in the USA and Canada, instead of
  hand-entering frequencies.

## Analog vs digital, conventional vs trunked

Two independent axes decide whether a given scanner can hear a given agency:

| | Conventional | Trunked |
|---|---|---|
| **Analog FM** | Any scanner, incl. $100 handhelds | Needs trunk-tracking (e.g. [BearTracker BCT15X](/reference/uniden-bearcat-bct15x/)) |
| **Digital (P25/DMR/NXDN)** | Needs a digital scanner | Needs a **digital trunking** scanner (SDS/BCD/TRX series) |

The bottom-right box — **digital + trunked** — is where most modern police and
fire live, and it is the reason a cheap analog scanner is often useless in a
city. See [digital vs analog scanners](/digital-vs-analog-scanner/) for how to
tell what you need.

## What a scanner cannot do

**No scanner can decode encrypted traffic.** When an agency keys up AES-256
encryption, the scanner correctly identifies the [talkgroup](/reference/talkgroup/)
but plays silence — it lacks the key, and obtaining or using one without
authorization is a federal crime. This limit is **identical for software-defined
radios**: [GopherTrunk](/police-scanner-vs-sdr/) is a receiver too, so it decodes
clear and even scrambled traffic but never keyed encryption. Anyone selling a
"scanner that hears encrypted police" is selling a fiction. See
[police scanner encryption explained](/police-scanner-encryption/).

## Buying one — or building the free alternative

Dedicated scanners are turnkey, portable, and need no computer. Uniden's
[SDS100](/reference/uniden-sds100/) (handheld) and
[SDS200](/reference/uniden-sds200/) (base/mobile) lead on tough
[simulcast](/reference/simulcast/) systems; the
[BCD436HP](/reference/uniden-bcd436hp/) /
[BCD536HP](/reference/uniden-bcd536hp/) add ZIP-code programming; Whistler's
[TRX-1](/reference/whistler-trx-1/) / [TRX-2](/reference/whistler-trx-2/) include
DMR and NXDN at no extra cost. Our [best police scanners](/best-police-scanners/)
guide ranks them all.

The **free alternative** is a [software-defined radio](/reference/software-defined-radio/)
— a ~$30 [RTL-SDR](/reference/rtl-sdr/) — running **GopherTrunk**, which follows
and decodes P25, DMR, NXDN, and TETRA, and additionally **records, logs, and
timestamps every call** across unlimited channels. It trades portability and a
turnkey experience for zero cost and far more data. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/).

## Bottom line

A police scanner is the fastest way to hear your community's live public-safety
radio. Match the radio to your agencies — **digital and trunk-tracking if they've
gone P25/DMR/NXDN**, analog if they haven't — accept that **encrypted channels are
off-limits to everyone**, and remember that a dongle plus
[GopherTrunk](/downloads.html) does the same decoding for free if you have a PC to
spare.

## Sources

[^wiki]: [Scanner (radio)](https://en.wikipedia.org/wiki/Scanner_(radio)) — Wikipedia, on scanning receivers and how they sweep frequencies.
