---
slug: icom-id-52a
title: Icom ID-52A
entry_type: hardware
category: ham-radios
description: "The Icom ID-52A is a dual-band D-STAR handheld with an IP67 waterproof body, a big color display, GPS and Bluetooth — now transitioning to the ID-52A PLUS with enhanced terminal-mode features."
keywords: Icom ID-52A, ID-52A review, ID-52A PLUS, D-STAR handheld, IP67 ham radio, waterproof ham handheld, Icom D-STAR HT, best D-STAR radio 2026, dual band handheld ham radio
aka: [ID-52A, ID-52A PLUS, ID-52]
autolink: true
affiliate: true
product:
  name: "Icom ID-52A"
  brand: Icom
  category: Ham handheld transceiver
  lowPrice: "580"
  highPrice: "750"
  url: https://www.amazon.com/dp/B0DC8MSVJ4?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 144/430 MHz; RX 0.5–579 MHz" }
  - { label: Modes, value: "FM, D-STAR DV/DR; AM/FM/WFM receive" }
  - { label: Power, value: 5 W }
  - { label: Programming, value: "Icom CS-52/CS-52PLUS, RT Systems (no CHIRP)" }
  - { label: Price, value: "around $580 (original) / $700–750 (PLUS)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0DC8MSVJ4?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">ID-52A PLUS on Amazon &rarr;</a>" }
see_also: [kenwood-th-d75a, yaesu-ft-5dr, yaesu-ft-60r, kenwood-th-f6a, rtl-sdr, d-star]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.icomamerica.com/lineup/products/ID-52APLUS/
faq:
  - q: "What is the difference between the ID-52A and the ID-52A PLUS?"
    a: "Same radio family. The PLUS is Icom's 2024 refresh with enhanced Terminal and Access Point (DV gateway) features and USB-C-era conveniences; Icom America's lineup now lists the PLUS. Remaining new stock of the original still sells for around $580, the PLUS for $700–750."
  - q: "Is the ID-52A waterproof?"
    a: "Yes — IP67, rated for 1 m of water for 30 minutes. That's a real advantage over the Kenwood TH-D75A (IP54/55, splash-resistant only) if your D-STAR radio lives outdoors."
  - q: "Does the ID-52A do APRS?"
    a: "Not conventional APRS — there's no packet TNC. It reports position via D-PRS over D-STAR instead. If APRS is your priority, the Kenwood TH-D75A is the clear pick."
  - q: "Does the ID-52A work with CHIRP?"
    a: "No. Program it with Icom's free CS-52 (CS-52PLUS for the PLUS) software, RT Systems, or by editing the microSD card contents directly."
  - q: "Do I need a license to use the ID-52A?"
    a: "Transmitting on the ham bands requires an FCC amateur license (Technician class or higher). Listening — and this radio's 0.5–579 MHz receiver with air band is good at it — requires none."
---
**The Icom ID-52A** is the [D-STAR](/reference/d-star/) handheld built for the
outdoors: dual-band 144/430 MHz at 5 W, a genuinely waterproof **IP67** body,
the first large color display on a D-STAR HT, wideband 0.5–579 MHz receive with
AM air band, and GPS plus Bluetooth built in.[^icom] The family is currently
transitioning: the original ID-52A (~$580 while stock lasts) is being superseded
by the **ID-52A PLUS** (~$700–750), Icom's 60th-anniversary refresh with
enhanced terminal-mode and DV-gateway features.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0DC8MSVJ4?tag=gophertrunk-20" rel="nofollow sponsored noopener">ID-52A PLUS on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The best D-STAR handheld for hard outdoor use.** IP67 waterproofing (the
[TH-D75A](/reference/kenwood-th-d75a/) is only splash-resistant), a superb color
UI, easy reflector access, and a bigger stock battery (1,880 mAh) than Kenwood's.
**No APRS TNC** — position goes out as D-PRS over D-STAR — no 220 MHz, and
D-STAR is its only digital mode. **~$580 for remaining original stock, $700–750
for the PLUS.** Transmitting takes an FCC Technician license; listening doesn't.
</div>

## Overview

Where Kenwood's [TH-D75A](/reference/kenwood-th-d75a/) is the data-mode Swiss
Army knife, the ID-52A is the D-STAR radio you can drop in a creek. Icom gave it
a 2.3-inch color transflective LCD (readable in sunlight), simultaneous V/V,
U/U and V/U dual watch, 1,000 memories, a microSD slot, and one of the cleanest
D-STAR implementations in the business — repeater lists, reflector linking and
the share-picture function all work the way the manual says they do.

The PLUS transition matters mostly for terminal and access-point operation
(working D-STAR gateways over the internet with no local repeater): the PLUS
enhances those features, and it's what Icom America's lineup page now lists.
Buying the original at a discount is a fine move if you mainly use local
repeaters; buy the PLUS if gateway operation is the point. One era-specific
annoyance on pre-PLUS units: charging is Micro-USB, which owners rightly
dislike.

## Modes &amp; features

- **[D-STAR](/reference/d-star/) DV/DR** with easy reflector access; PLUS adds
  enhanced Terminal/Access Point modes.
- **Wideband RX 0.5–579 MHz** — AM, FM and WFM including the air band.
- **IP67** — 1 m / 30 min submersion rating, the best waterproofing in the
  current D-STAR field.
- **GPS and Bluetooth** built in; **D-PRS** position reporting (no conventional
  [APRS](/reference/aprs/) TNC).
- **5 W** on 144/430 MHz; BP-272 1,880 mAh battery (3,150 mAh BP-307 optional).

## Programming

Icom's free CS-52 software (CS-52PLUS for the PLUS) handles memories and
settings over USB or by writing the microSD card directly; RT Systems covers it
too. **CHIRP does not support it** — plan on Icom's tools.

## GopherTrunk alternative

GopherTrunk receives; it can't transmit, so it is no substitute for an ID-52A.
What it *is* good for is finding out whether $600+ of D-STAR hardware will have
anyone to talk to. A ~$30 [RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk
lets you monitor your local repeaters and digital ham traffic — recording and
logging every transmission, which no HT does — so you know whether
[D-STAR](/reference/d-star/), [DMR](/reference/dmr/) or [C4FM](/reference/c4fm/)
is the live mode in your area before you commit to one. Start with
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the ID-52A** if you want D-STAR in a waterproof, sunlight-readable,
  field-first package — SOTA, backpacking, marine environments.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0DC8MSVJ4?tag=gophertrunk-20" rel="nofollow sponsored noopener">ID-52A PLUS on Amazon &rarr;</a>
- **Buy the [Kenwood TH-D75A](/reference/kenwood-th-d75a/)** instead if APRS,
  packet/Winlink, 220 MHz or SSB/CW receive matter more than waterproofing.
- **Skip both** if your area runs DMR or C4FM — see the
  [AT-D878UVII Plus](/reference/anytone-at-d878uvii-plus/) and
  [FT-5DR](/reference/yaesu-ft-5dr/), and the full rankings in
  [best handheld ham radios](/best-handheld-ham-radios/).

> **Buying note.** The button links the confirmed **ID-52A PLUS** (60th
> Anniversary Edition) listing. Remaining new stock of the original ID-52A
> sells through ham dealers around $580; on Amazon,
> <a href="https://www.amazon.com/s?k=icom+id-52a&tag=gophertrunk-20" rel="nofollow sponsored noopener">search current ID-52A listings</a>.

## Sources

[^icom]: [Icom ID-52A PLUS product page](https://www.icomamerica.com/lineup/products/ID-52APLUS/) — Icom America, on D-STAR DV/DR and terminal modes, 0.5–579 MHz receive, IP67 rating, GPS/Bluetooth, and the PLUS refresh.
