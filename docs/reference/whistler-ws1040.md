---
slug: whistler-ws1040
title: Whistler WS1040
entry_type: hardware
category: consumer-scanners
description: "The Whistler WS1040 is a handheld P25 Phase I digital police scanner with object-oriented memory, V-Scanner storage, Spectrum Sweeper and S.A.M.E. weather alerts — the value/used entry point."
keywords: Whistler WS1040, WS1040 scanner, handheld P25 Phase 1 scanner, object oriented scanner, V-Scanner, Spectrum Sweeper, Skywarn, Whistler WS1040 review
aka: [WS1040]
autolink: true
affiliate: true
product:
  name: "Whistler WS1040"
  brand: Whistler
  category: Police scanner (handheld)
  lowPrice: "250"
  highPrice: "330"
  url: https://www.amazon.com/dp/B00IID3OAY?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner }
  - { label: Modes, value: "P25 Phase I, analog" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "Object-oriented / PC software" }
  - { label: Extras, value: "V-Scanner storage, Spectrum Sweeper, S.A.M.E." }
  - { label: Price, value: around $290 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00IID3OAY?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [whistler-ws1088, whistler-ws1065, uniden-bcd325p2, uniden-bcd436hp, police-scanner, p25-phase-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://wiki.radioreference.com/index.php/WS1040
faq:
  - q: "Does the Whistler WS1040 decode P25 Phase II?"
    a: "No. The WS1040 is P25 Phase I only, plus analog. Many metro systems now carry voice on Phase II, which it can't follow — for that step up to the WS1088, a Uniden BCD436HP, or a free SDR running GopherTrunk. Buy the WS1040 for a Phase I / analog area or as an inexpensive used unit."
  - q: "What is V-Scanner storage on the WS1040?"
    a: "V-Scanner lets the WS1040 hold up to 21 saved scanner configurations, so one radio can carry setups for different cities or activities and switch between them — over 38,000 scannable objects in total across those folders."
  - q: "How do I program a WS1040?"
    a: "It uses Whistler's object-oriented model. You can enter frequencies and talkgroups by hand, or manage objects and scanlists on a PC with the free EZ Scan software and load them over the USB interface. There's no built-in ZIP-code database."
  - q: "Does the WS1040 decode DMR, NXDN, or encryption?"
    a: "No. It covers P25 Phase I and analog only — no DMR, no NXDN, no P25 Phase II. And like every consumer scanner and every SDR, it cannot decode AES-encrypted traffic."
  - q: "Is the WS1040 worth buying today?"
    a: "Only where the local system is P25 Phase I or analog, or as a cheap used handheld to learn on. For a current build, a Phase II scanner or a $30 SDR plus free GopherTrunk decodes far more — Phase II, DMR, NXDN — and records every call."
---
**The Whistler WS1040** is a handheld digital police scanner that follows and decodes
[P25 Phase I](/reference/p25-phase-1/) and analog, using Whistler's **object-oriented**
memory with **V-Scanner** multi-configuration storage, a **Spectrum Sweeper** for
finding nearby transmitters, and **S.A.M.E.** weather alerts.[^rr] It is the older,
Phase I–only sibling of the [WS1088](/reference/whistler-ws1088/) — the value and used
entry point into Whistler's digital line.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00IID3OAY?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**P25 Phase I handheld — the value/used pick.** The WS1040 decodes
[P25 Phase I](/reference/p25-phase-1/) and analog with
[object-oriented](/reference/talkgroup/) memory, **V-Scanner** folders, **Spectrum
Sweeper** and **S.A.M.E.** — but **not** [Phase II](/reference/p25-phase-2/), DMR or
NXDN. If your metro has upgraded, get the [WS1088](/reference/whistler-ws1088/)
instead. **Fair [simulcast](/reference/simulcast/). ~$290. No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a free
path that also does Phase II and DMR, compare [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The WS1040 is Whistler's Phase I digital handheld, descended from the GRE/Radio Shack
platform. Its hallmark is the **object-oriented** approach: every system, group and
channel is an "object" you organise freely, rather than fixed banks. **V-Scanner**
extends that to **21 stored configurations** — think of them as folders for different
cities or activities — giving over 38,000 scannable objects in total. **Spectrum
Sweeper** rapidly sweeps the bands for a strong nearby signal you don't have listed,
Whistler's analog answer to Close Call.

Its ceiling is [P25 Phase I](/reference/p25-phase-1/): it does **not** decode Phase II
TDMA voice, so on an upgraded system it will follow the control channel but miss the
calls. Its front end is conventional, so [simulcast](/reference/simulcast/) performance
is *fair*. For Phase II, move to the [WS1088](/reference/whistler-ws1088/) or a
[BCD436HP](/reference/uniden-bcd436hp/); for the toughest simulcast, only a True I/Q
[SDS100](/reference/uniden-sds100/) holds a clean lock.

## Modes &amp; systems it decodes

- **[P25 Phase I](/reference/p25-phase-1/)** — conventional and
  [trunked](/reference/trunked-radio/); this is the ceiling — **no Phase II**.
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM.
- **Weather (S.A.M.E. alerts)** with a **Skywarn** button, plus air, marine, rail and
  business channels; **Spectrum Sweeper** for finding nearby transmitters.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** and **no P25 Phase II** — for
  any of those, step up to a newer scanner or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

No ZIP-code database, so two paths through Whistler's object model:

1. **EZ Scan software (free).** Build objects and scanlists on a PC — frequencies,
   [talkgroups](/reference/talkgroup/), scan settings — and load them over USB into
   your V-Scanner folders. The practical way to set it up.
2. **Manual entry.** Key in frequencies and talkgroups by hand from a source like
   RadioReference.

## GopherTrunk alternative

The WS1040 is the *cheap, older turnkey* answer. **GopherTrunk** is the *free* one, and
it does what this scanner can't. An [RTL-SDR](/reference/rtl-sdr/) (~$30) running
GopherTrunk decodes [P25](/reference/project-25/) **Phase I and II** — plus
[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/) the WS1040 never supported — and
**auto-discovers and follows** [trunked systems](/reference/trunked-radio/). It also
**records, logs and timestamps every call**, follows **unlimited** systems at once, and
streams to a **web console**.

Where the WS1040 still wins: **no PC required** and true pocket portability. The honest
head-to-head is in [police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try the
free path, [download GopherTrunk](/downloads.html) and pair it with a $30 dongle first.

## Who it's for

- **Buy the WS1040** only if your local system is **P25 Phase I** or analog and you find
  one cheap (often used) to learn on.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00IID3OAY?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [WS1088](/reference/whistler-ws1088/)** instead for the same handheld with
  **P25 Phase II**, or a [BCD436HP](/reference/uniden-bcd436hp/) for Uniden's workflow.
- **Go free** with a [SDR + GopherTrunk](/police-scanner-vs-sdr/). Comparing? See
  [best police scanners](/best-police-scanners/).

## Sources

[^rr]: [WS1040](https://wiki.radioreference.com/index.php/WS1040) — The RadioReference Wiki, on the WS1040's P25 Phase I support, object-oriented programming, V-Scanner storage, Spectrum Sweeper, and S.A.M.E. weather alerts.
