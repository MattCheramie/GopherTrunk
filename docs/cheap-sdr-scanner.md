---
layout: page
title: "Cheapest Way to Scan Digital Radio (SDR + GopherTrunk)"
description: "The cheapest way to scan digital P25, DMR, and NXDN radio — a ~$30 RTL-SDR plus free GopherTrunk, far under a $380+ digital scanner. Budget breakdown and the honest trade-offs."
keywords: cheap SDR scanner, cheapest digital scanner, RTL-SDR P25, budget police scanner, scan P25 cheap, DMR NXDN SDR, GopherTrunk cheap, $30 scanner, digital scanner alternative
permalink: /cheap-sdr-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the cheapest way to scan digital police radio?"
    a: "A ~$30 RTL-SDR dongle plus free, open-source GopherTrunk is the cheapest route to decoding digital P25, DMR, and NXDN — a fraction of the $380-plus a dedicated digital trunking scanner costs. You supply a computer you already own; the software is free."
  - q: "How much does a cheap SDR scanner cost?"
    a: "About $30–60 all in if you own a computer: roughly $30 for an RTL-SDR, ~$18–25 for a decent antenna, and a few dollars for an SMA adapter. GopherTrunk itself costs nothing. That undercuts even entry-level analog-only scanners while doing digital that they cannot."
  - q: "Can a cheap RTL-SDR really decode P25 and DMR?"
    a: "Yes. GopherTrunk does the digital decoding in software, so a ~$30 8-bit RTL-SDR handles P25 Phase I/II, DMR, and NXDN on a clean signal. The dongle just delivers samples; the decoding smarts are free software, which is why it is so cheap."
  - q: "What is the catch with a cheap SDR scanner?"
    a: "It needs a computer — a PC you own, or a ~$80 Raspberry Pi for always-on use — so it is not a grab-and-go handheld. It also has a weaker simulcast front end than a premium scanner and cannot decode AES encryption. For a fixed home or desk setup, none of that matters."
  - q: "Is a cheap SDR better than a cheap scanner?"
    a: "For digital systems, usually yes — most inexpensive scanners are analog-only and cannot follow P25/DMR/NXDN at all, while a ~$30 RTL-SDR plus GopherTrunk can. A cheap scanner wins only if you specifically need a portable, no-computer, battery-powered box."
  - q: "Can a cheap SDR decode encrypted channels?"
    a: "No. No SDR or scanner at any price can decode AES-encrypted talkgroups. A cheap SDR hears everything transmitted in the clear, which in most areas still includes plenty of dispatch and nearly all fire/EMS — exactly what a $380 scanner would also hear."
---

# Cheapest Way to Scan Digital Radio (SDR + GopherTrunk)

**The cheapest way to scan digital P25, DMR, and NXDN radio is a ~$30
[RTL-SDR](/reference/rtl-sdr/) dongle plus free, open-source
[GopherTrunk](/downloads.html) — a small fraction of the $380-plus a dedicated digital
trunking scanner costs.** The decoding lives in free software, so the only money is a
cheap USB dongle and an antenna.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Cheapest digital setup:** a ~$30 [RTL-SDR](/reference/rtl-sdr/) + free
[GopherTrunk](/downloads.html). **Total ~$30–60** with an antenna and adapter, vs
**$380+** for a digital scanner. **Decodes** P25 Phase I/II, DMR, NXDN in software.
**The catch:** it needs a computer (a PC you own, or a
[~$80 Raspberry Pi](/raspberry-pi-sdr-scanner/)). **No SDR at any price decodes
[AES encryption](/police-scanner-encryption/).**
</div>

## The whole cost, laid out

Because GopherTrunk is free, the "scanner" is just a handful of cheap parts:

| Part | Why | Pick | Approx $ |
|---|---|---|---|
| **SDR dongle** | The receiver + digital decoder (in software) | [NooElec NESDR Nano 3](https://www.amazon.com/dp/B073JZ8CC2?tag=gophertrunk-20) / [NESDR SMArt v5](/reference/nesdr/) | ~$30–35 |
| **Antenna** | Actually hears signals | [Nagoya NA-771 whip](https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20) or a [dipole kit](/reference/dipole-antenna/) | ~$18–25 |
| **SMA adapter** | Connects antenna to dongle | [SMA adapter kit](https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20) | ~$13 |
| **GopherTrunk** | Does all the decoding | [Free download](/downloads.html) | $0 |
| **Computer** | Runs the software | A PC you already own | $0 |
| | | **Total** | **~$30–60** |

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">Cheapest solid dongle on Amazon &rarr;</a>

The absolute floor is a bare [NESDR Nano 3](https://www.amazon.com/dp/B073JZ8CC2?tag=gophertrunk-20)
around $30; spend a few dollars more on a [NESDR SMArt v5](/reference/nesdr/) and you get
a shielded case and a stable TCXO that make tracking a control channel far more reliable.
Either way you are **an order of magnitude under a digital scanner.**

## Why it is so much cheaper than a scanner

A dedicated [digital police scanner](/best-police-scanners/) has to build the P25/DMR/NXDN
decoder, the trunk-tracking logic, the screen, the battery, and the case into one
handheld — and P25 decoding carries licensing costs. That is why entry-level *digital*
scanners start around $380 and good ones run higher. Cheap scanners under ~$100 are
**analog-only** and cannot follow digital trunked systems at all.

GopherTrunk moves all of that — the decoding, the trunk-tracking, the logging, the
display — into **free software running on a computer you already own**. The
[RTL-SDR](/reference/rtl-sdr/) only has to deliver raw samples, which an 8-bit dongle does
just fine on a clean signal. So you pay ~$30 for the antenna-to-USB hardware and nothing
for the brains.

> **Cheap dongle, real digital.** A ~$30 [RTL-SDR](/reference/rtl-sdr/) plus GopherTrunk
> decodes [P25 Phase I and II](/best-sdr-for-p25-trunking/), DMR, and NXDN — the exact
> digital modes that put a scanner into the $380+ bracket. The savings come from free
> software, not from cutting capability.

## The honest trade-offs

Cheap is real, but so are the compromises. Be clear-eyed:

- **It needs a computer.** This is a fixed or desk setup, not a grab-and-go handheld. A
  PC you own costs nothing; for always-on use add a
  [~$80 Raspberry Pi](/raspberry-pi-sdr-scanner/).
- **No battery, less portable.** A scanner clips to your belt at the track or a parade; an
  SDR does not (short of a laptop or Pi power bank rig).
- **Weaker simulcast front end.** On tough [simulcast](/reference/simulcast/) systems a
  premium Uniden SDS scanner still has the edge. A cheap dongle is fine on clean single
  sites.
- **No encryption, ever.** Like every receiver, it cannot decode
  [AES-encrypted](/police-scanner-encryption/) talkgroups — but neither can the $380
  scanner.

None of these matter for the common case: **a home or desk digital scanner that logs
everything in the clear for the price of a pizza.** The full head-to-head is in
[police scanner vs SDR](/police-scanner-vs-sdr/), and the budget-scanner angle in
[cheap police scanner](/cheap-police-scanner/).

## What you actually get for ~$30

Do not mistake cheap for limited. On top of decoding the same digital modes as an
expensive scanner, GopherTrunk **records and timestamps every call, follows unlimited
talkgroups, logs to a database, and serves a [web console](/raspberry-pi-sdr-scanner/)**
you can reach from your phone — features that cost extra or do not exist on many
dedicated scanners. It is genuinely more capable in software than hardware many times its
price, provided you can sit it next to a computer.

## Bottom line

If you want to hear digital P25/DMR/NXDN and spend as little as possible, buy a **~$30
[RTL-SDR](/reference/rtl-sdr/)** (a [NESDR Nano 3](https://www.amazon.com/dp/B073JZ8CC2?tag=gophertrunk-20)
or the sturdier [SMArt v5](https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20)), add
a cheap [antenna](https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20) and
[adapter](https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20), and run
[free GopherTrunk](/downloads.html) on a computer you already have. That is ~$30–60 versus
$380+ for a digital scanner — the cheapest real route into digital monitoring. Just
remember it needs a PC and cannot break [encryption](/police-scanner-encryption/). Full
shopping list: [what do I need for GopherTrunk](/what-do-i-need-for-gophertrunk/).
