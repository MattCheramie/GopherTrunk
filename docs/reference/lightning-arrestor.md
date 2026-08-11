---
slug: lightning-arrestor
title: Coax lightning arrestor
entry_type: hardware
category: mounting
description: "A coax lightning arrestor sits in the feedline where it enters a building and shunts surge to ground with a gas-discharge tube — the cheap in-line insurance for an SDR's front end."
keywords: lightning arrestor, coax surge protector, coax lightning arrester, gas discharge tube, GDT arrestor, SO-239 lightning arrestor, N-type surge protector, antenna surge protector, feedline protection
aka: [surge arrestor, coax surge protector, lightning arrester, GDT protector]
autolink: true
affiliate: true
product:
  name: "Proxicast Coaxial Lightning Arrester (PL-259 / SO-239)"
  brand: Proxicast
  category: Coax surge / lightning arrestor
  lowPrice: "20"
  highPrice: "32"
  url: https://www.amazon.com/dp/B0CBW5TXKV?tag=gophertrunk-20
infobox:
  - { label: Type, value: Gas-discharge coax arrestor }
  - { label: Connectors, value: UHF (PL-259/SO-239), N, or SMA }
  - { label: Mount at, value: Feedline building entry, bonded to ground }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0CBW5TXKV?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [grounding-kit, antenna-mast, coaxial-cable, n-type-connector, tripod-mount, discone-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Surge_arrester
  - https://en.wikipedia.org/wiki/Gas-filled_tube
faq:
  - q: "Do I need a lightning arrestor for an SDR antenna?"
    a: "If the antenna is outdoors, yes. A coax lightning arrestor (around $20–30) installs in-line where the feedline enters the building and shunts surge to ground through a gas-discharge tube before it reaches your SDR. It is cheap insurance for a $35 dongle and, more importantly, for the equipment and house behind it. Bond it to the same ground as the mast."
  - q: "How does a coax lightning arrestor work?"
    a: "Most use a gas-discharge tube between the coax centre conductor and the grounded shell. Under normal signal levels it is an open circuit and passes RF with minimal loss; when a surge spikes the voltage past its breakdown, the gas ionises and conducts, dumping the transient to ground. It is a fast, self-resetting shunt — the same idea as a lightning rod, applied to your feedline."
  - q: "Where do I install the arrestor?"
    a: "At the point where the feedline enters the building, mounted to a grounded bulkhead or ground plate and bonded with a short heavy wire to the same ground rod as the mast. Putting it at the entry keeps surge outside your walls; a short, direct ground bond is what makes it work, since a fast transient sees the inductance of a long or bent wire."
  - q: "Does an arrestor replace grounding the mast?"
    a: "No — they work together. The grounding kit bonds the mast and bleeds static; the arrestor protects the inner conductor of the coax, which grounding the shield alone does not. And neither fully stops a direct strike — for real storm risk, unplug the feedline. The surest lightning protection is still an antenna cable that is not connected to anything."
---

A **coax lightning arrestor** is a small in-line device that sits in the
[feedline](/reference/coaxial-cable/) where it enters a building and shunts a voltage surge to
ground before it can reach your receiver. Inside is usually a **gas-discharge tube (GDT)**
between the coax centre conductor and a grounded shell: invisible to normal signals, an instant
short to a spike. For any outdoor [antenna](/best-scanner-antenna/), it is the cheapest
insurance you can buy for your [SDR](/best-sdr-for-gophertrunk/)'s front end — and for the
gear and house behind it.

## How it works

Under normal conditions the arrestor is an open circuit across the line: the gas tube does not
conduct, and RF passes through with only a small insertion loss, typically a fraction of a dB
across HF through UHF. When a nearby lightning strike or a static build-up drives the line
voltage past the tube's breakdown threshold — often around 90 V — the gas **ionises and
conducts**, dumping the transient into the grounded shell and clamping the voltage the
downstream equipment sees. Once the surge passes, the gas de-ionises and the tube resets
itself, ready for the next event. On many models the GDT is a replaceable cartridge, since a
big hit can wear it out.

Crucially, the arrestor protects the **centre conductor** of the coax. Grounding the mast and
the cable shield — which a [grounding kit](/reference/grounding-kit/) does — leaves the inner
conductor unprotected; the arrestor closes that gap.

## How it mounts

Install it **where the feedline enters the building**, bolted to a grounded bulkhead or ground
plate and bonded with a **short, heavy, straight** wire to the same ground rod the
[mast](/reference/antenna-mast/) uses. The short bond is not a detail — a fast transient sees
the inductance of every inch and every bend, so a long or curly ground lead badly undercuts the
device. Match the connectors to your run: UHF (PL-259/SO-239) is common on scanner and ham
gear, [N-type](/reference/n-type-connector/) on low-loss outdoor cable, and SMA on the SDR end,
with adapters as needed from the [cables and connectors](/sdr-cables-and-connectors/) kit.

## Relevance to GopherTrunk

The arrestor is transparent to [GopherTrunk](/downloads.html) — it passes signal and only acts
during a surge — but it is what lets you responsibly leave an outdoor antenna connected to a
24/7 [Raspberry Pi scanner](/raspberry-pi-sdr-scanner/). A $35 dongle running unattended on a
rooftop [discone](/reference/discone-antenna/) is exactly the setup that a static discharge or
nearby strike can quietly kill; the arrestor plus a grounded mast is the standard, inexpensive
defence. It has nothing to do with [encryption](/police-scanner-encryption/) — that is a
separate wall — but it keeps the hardware alive to decode everything that is in the clear.

## Where to buy

A **gas-discharge coax arrestor** like the Proxicast (around $25) is the standard pick:
low insertion loss, a replaceable GDT element, and UHF/N/SMA connector options to match your
feedline. Mount it at the building entry, bonded short and heavy to the mast's ground.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CBW5TXKV?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

Pair it with a [grounding kit](/reference/grounding-kit/); see the
[mast and mounting guide](/antenna-mast-and-mounting-guide/) for the full safe stack and the
[outdoor base build](/gophertrunk-outdoor-base-build/) for a parts list.

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^surge]: [Surge arrester](https://en.wikipedia.org/wiki/Surge_arrester) — Wikipedia, on shunting transient overvoltage to ground to protect downstream equipment.
[^gdt]: [Gas-filled tube](https://en.wikipedia.org/wiki/Gas-filled_tube) — Wikipedia, on the gas-discharge element that conducts once its breakdown voltage is exceeded.
