---
slug: uniden-bcd996xt
title: Uniden BCD996XT
entry_type: hardware
category: consumer-scanners
description: "The Uniden BCD996XT is a P25 Phase I base/mobile digital police scanner with 25,000 channels, TrunkTracker, GPS support and Close Call — the older, value-priced sibling of the BCD996P2."
keywords: Uniden BCD996XT, BCD996XT scanner, P25 Phase 1 base scanner, TrunkTracker, GPS scanner, Close Call, Dynamic Memory Architecture, Uniden BCD996XT review
aka: [BCD996XT]
autolink: true
affiliate: true
product:
  name: "Uniden BCD996XT"
  brand: Uniden
  category: Police scanner (base/mobile)
  lowPrice: "330"
  highPrice: "430"
  url: https://www.amazon.com/s?k=Uniden+BCD996XT+scanner&tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital base/mobile scanner }
  - { label: Modes, value: "P25 Phase I, TrunkTracker" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "Manual / software (no ZIP DB)" }
  - { label: Extras, value: "25,000 channels, GPS-ready, Close Call, S.A.M.E." }
  - { label: Price, value: around $380 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Uniden+BCD996XT+scanner&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd996p2, uniden-bcd536hp, uniden-sds200, uniden-bearcat-bct15x, police-scanner, p25-phase-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://wiki.radioreference.com/index.php/BCD996XT
faq:
  - q: "What's the difference between the BCD996XT and the BCD996P2?"
    a: "The BCD996XT is P25 Phase I only; the newer BCD996P2 adds P25 Phase II (TDMA/X2-TDMA), which most metro systems now use for voice. If your system has moved to Phase II, buy the BCD996P2. The BCD996XT is the value pick where the local system is still Phase I, or as a used bargain."
  - q: "Does the BCD996XT decode P25 Phase II?"
    a: "No. It handles P25 Phase I conventional and trunked, plus Motorola/EDACS/LTR and analog via TrunkTracker, but not the Phase II TDMA voice that many upgraded systems now carry. For Phase II you want the BCD996P2, an SDS-series Uniden, or a free SDR running GopherTrunk."
  - q: "How do I program a BCD996XT?"
    a: "It has no HomePatrol ZIP-code database, so you enter frequencies and talkgroups by hand from a source like RadioReference, or build favorites on a PC with third-party software (like ARC-XT/FreeSCAN) and load them over USB. Its GPS input can then auto-enable systems as you drive."
  - q: "Does the BCD996XT decode DMR or NXDN?"
    a: "No — it predates those modes on Uniden hardware. For DMR and NXDN, look at a Whistler TRX-series scanner or a free SDR running GopherTrunk. And no scanner of any age can decode AES encryption."
  - q: "Is the BCD996XT still worth buying?"
    a: "Only if your local system is P25 Phase I or analog and you find one cheap. For a current build, the BCD996P2 (Phase II) or a $30 SDR plus free GopherTrunk gets you more, decodes Phase II and DMR/NXDN, and records every call."
---
**The Uniden BCD996XT** is a digital base/mobile police scanner that follows and
decodes [P25 Phase I](/reference/p25-phase-1/) with **TrunkTracker**, holds
**25,000 channels** across Uniden's Dynamic Memory Architecture, and adds **GPS
support** and **Close Call** near-signal capture.[^rr] It is the older, value-priced
predecessor of the [BCD996P2](/reference/uniden-bcd996p2/) — same base/mobile body,
but **Phase I only**, with no P25 Phase II TDMA voice.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Uniden+BCD996XT+scanner&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**P25 Phase I base — the value/used pick.** The BCD996XT decodes
[P25 Phase I](/reference/p25-phase-1/) with **TrunkTracker**, **25,000 channels**,
**GPS** and **Close Call**, but **not** [Phase II](/reference/p25-phase-2/) TDMA — so
if your metro has upgraded, buy the [BCD996P2](/reference/uniden-bcd996p2/) instead.
**No ZIP database:** program it manually or from a PC. **Fair
[simulcast](/reference/simulcast/).** **~$380.** **No encryption** — like every scanner
(and every SDR), it can't decode [AES](/police-scanner-encryption/). For a free path
that also does Phase II and DMR, compare [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The BCD996XT was Uniden's APCO-25 base/mobile flagship before the Phase II era. It
introduced **Dynamic Memory Architecture** — objects (systems, sites, groups,
channels) allocated from a shared 25,000-channel pool instead of fixed banks — plus
**GPS-aware scanning** that enables and mutes systems by your location as you drive.
Its handheld contemporary is the [BCD436HP](/reference/uniden-bcd436hp/) generation's
predecessor line; today its direct replacement is the
[BCD996P2](/reference/uniden-bcd996p2/), which drops into the same role but adds
[P25 Phase II](/reference/p25-phase-2/).

Like the other non-SDS Unidens, its conventional front end gives *fair*
[simulcast](/reference/simulcast/) — acceptable in many areas, but it distorts on the
toughest simulcast systems, where only the True I/Q
[SDS200](/reference/uniden-sds200/) holds a clean lock.

## Modes &amp; systems it decodes

- **[P25 Phase I](/reference/p25-phase-1/)** — conventional and
  [trunked](/reference/trunked-radio/); this is the ceiling — **no Phase II**.
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM, all followed by
  **TrunkTracker** across **25,000 channels**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels, plus **Close Call** to grab a nearby transmitter you don't have listed.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** and **no P25 Phase II** — for
  any of those, step up to a newer scanner or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

No ZIP-code database, so two paths:

1. **PC software.** Build favorites lists on a computer — frequencies,
   [talkgroups](/reference/talkgroup/), and scan settings — with Uniden's tools or a
   third-party editor, and load them over USB. The practical way to set it up.
2. **Manual entry.** Key in frequencies and talkgroups by hand from a source like
   RadioReference. Slower, but works with no PC. GPS then auto-enables systems by
   location.

## GopherTrunk alternative

The BCD996XT is the *older turnkey* answer. **GopherTrunk** is the *free* one, and it
does what this scanner can't. An [RTL-SDR](/reference/rtl-sdr/) (~$30) running
GopherTrunk decodes [P25](/reference/project-25/) **Phase I and II** — plus
[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/) the BCD996XT never supported — and
**auto-discovers and follows** [trunked systems](/reference/trunked-radio/) rather than
making you hand-enter [talkgroups](/reference/talkgroup/). It also **records, logs and
timestamps every call**, follows **unlimited** systems at once, and streams to a **web
console**.

Where the BCD996XT still wins: **no PC required**, standalone base operation, and
FCC-certified turnkey hardware. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try it,
[download GopherTrunk](/downloads.html) first.

## Who it's for

- **Buy the BCD996XT** only if your local system is **P25 Phase I** or analog and you
  find one at a good (often used) price.
  <a class="btn btn--buy" href="https://www.amazon.com/s?k=Uniden+BCD996XT+scanner&tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BCD996P2](/reference/uniden-bcd996p2/)** instead for the same base/mobile
  role **with P25 Phase II**, or the [SDS200](/reference/uniden-sds200/) for True I/Q
  simulcast and DMR/NXDN.
- **Go free** with a [SDR + GopherTrunk](/police-scanner-vs-sdr/). Comparing? See
  [best police scanners](/best-police-scanners/).

> **Amazon note.** The BCD996XT is a legacy model, so Amazon stock is mostly
> third-party and used — the button is a tagged search that resolves to current
> listings rather than a single page that may be out of stock.

## Sources

[^rr]: [BCD996XT](https://wiki.radioreference.com/index.php/BCD996XT) — The RadioReference Wiki, on the BCD996XT's P25 Phase I support, TrunkTracker, 25,000-channel Dynamic Memory Architecture, GPS, and Close Call.
