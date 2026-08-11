---
slug: mobile-scanner-antenna
title: Mobile scanner antenna
entry_type: hardware
category: antennas
description: "A mobile scanner antenna is a magnetic-mount whip that sticks to a vehicle roof and uses the metal body as a ground plane, giving far better reception than a handheld while scanning on the move."
keywords: mobile scanner antenna, magnetic mount antenna, mag mount, vehicle scanner antenna, car scanner antenna, dual band mag mount, Tram 1185, mobile whip, ground plane vehicle
aka: [mag-mount scanner antenna, magnetic mount antenna, car scanner antenna, vehicle scanner antenna]
autolink: true
affiliate: true
product:
  name: "Tram 1185 dual-band magnetic-mount mobile antenna (BNC)"
  brand: Tram
  category: Magnetic-mount mobile scanner antenna
  lowPrice: "22"
  highPrice: "32"
  url: https://www.amazon.com/dp/B01DY8FNAG?tag=gophertrunk-20
infobox:
  - { label: Type, value: Magnetic-mount vertical whip }
  - { label: Ground, value: Vehicle roof / body }
  - { label: Pattern, value: Omnidirectional, vertical }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01DY8FNAG?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [whip-antenna, ground-plane-antenna, handheld-scanner-antenna, base-scanner-antenna, polarization, coaxial-cable]
cite_urls:
  - https://en.wikipedia.org/wiki/Whip_antenna
  - https://en.wikipedia.org/wiki/Ground_plane
faq:
  - q: "Which antenna should I buy for scanning in a car?"
    a: "A dual-band magnetic-mount whip like the Tram 1185 (around $27) is the standard mobile scanner antenna: it sticks to the roof with a magnet, uses the car body as a ground plane, and covers the VHF and UHF land-mobile bands where most scanning happens. It hears far better than a handheld held inside the cabin, because it is outside the metal box and has a real ground plane under it. Match the connector to your scanner — BNC for a Uniden BC125AT, or an adapter for an SMA radio."
  - q: "Why is a mag-mount so much better than a handheld inside the car?"
    a: "Two reasons. A vehicle is a metal box that shields the radio inside it, so an antenna in the cabin is fighting a Faraday cage; a roof-mounted whip is out in the clear. And the steel roof gives the quarter-wave whip the large ground plane it needs to work properly, which a handheld never has. The combination is often a night-and-day improvement on weak signals."
  - q: "Will a magnet mount damage or scratch my roof?"
    a: "A quality mag-mount has a protective boot or padded base, so it grips firmly at highway speed without scratching if the roof is clean. Grit trapped under the magnet is what causes swirl marks, so wipe the spot first. On a non-metal roof (aluminum or fiberglass, common on some newer vehicles) a magnet will not stick or ground properly — you would need a different mount and an antenna with its own ground plane."
  - q: "Where should I route the coax and mount the whip?"
    a: "Put the magnet as close to the center of the roof as you can, so the metal ground plane extends evenly around the base. Route the thin coax under a door or trunk-lid rubber seal — most mag-mount cables are designed to survive that pinch — and keep the run to the scanner short. Avoid coiling excess cable near the radio."
---

A **mobile scanner antenna** is a magnetic-mount [whip](/reference/whip-antenna/) that
sticks to a vehicle's roof and turns the car body into a
[ground plane](/reference/ground-plane-antenna/). It is the middle rung of scanner
antennas — between the compromised [handheld](/reference/handheld-scanner-antenna/) whip
and a fixed [rooftop](/reference/base-scanner-antenna/) install — and for anyone scanning
on the move it is transformative, because it puts a properly grounded vertical outside the
metal box that would otherwise shield the receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A side view of a car with a magnetic-mount whip standing on the roof, a coax cable running from its base down into the cabin, and the roof labeled as the ground plane." xmlns="http://www.w3.org/2000/svg">
  <path d="M70 150 L110 150 C 120 120, 150 110, 200 110 L300 110 C 340 110, 360 130, 390 150 L410 150" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="70" y1="150" x2="410" y2="150" stroke="currentColor" stroke-width="2"/>
  <circle cx="150" cy="163" r="13" fill="none" stroke="currentColor" stroke-width="2"/>
  <circle cx="330" cy="163" r="13" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M245 110 C 245 70, 247 55, 255 30" fill="none" stroke="currentColor" stroke-width="3"/>
  <text x="260" y="50" font-size="10" fill="currentColor">λ/4 whip</text>
  <rect x="238" y="106" width="14" height="8" fill="currentColor" opacity="0.7"/>
  <text x="255" y="128" font-size="9" fill="currentColor">magnet base</text>
  <text x="120" y="102" font-size="9" fill="currentColor">roof = ground plane</text>
  <path d="M244 112 C 220 125, 210 135, 205 150" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <text x="150" y="140" font-size="8" fill="currentColor">coax into cabin</text>
</svg>
<figcaption>A mag-mount whip stands on the roof and uses the steel body as its ground plane, hearing far better than a scanner held inside the shielded cabin.</figcaption>
</figure>

## How it works

The whip itself is an ordinary quarter-wave [monopole](/reference/monopole-antenna/): it
needs a conducting plane beneath it to mirror the element into an effective half-wave
[dipole](/reference/dipole-antenna/). A vehicle roof is a large, well-connected steel
sheet — an excellent ground plane — and the magnetic base couples to it capacitively
through the paint, completing the RF path without drilling a hole. That is the whole
appeal of a mag-mount: full quarter-wave performance with a peel-and-stick install and no
body damage.

Most scanner mag-mounts are **dual-band** designs resonant near both the VHF (136–174 MHz)
and UHF (400–520 MHz) land-mobile bands, so one antenna covers the bulk of what a scanner
monitors. The pattern is omnidirectional in azimuth and vertically
[polarized](/reference/polarization/), matching P25, DMR, and NXDN traffic from any
bearing — exactly what you want while driving through changing coverage. As with any
mobile install, the [coax](/reference/coaxial-cable/) is thin to slip under a door seal,
so keep the run short and do not add loss with long extensions.

Two practical limits are worth knowing. First, the antenna wants the roof **center**: a
whip near an edge sees an asymmetric ground plane and a skewed pattern. Second, a magnet
will not grip or ground a **non-metal roof** — aluminum or composite panels, increasingly
common on newer vehicles — where you need a through-glass, lip, or permanent mount and an
antenna with its own ground system instead.

## Relevance to GopherTrunk

A mobile antenna most often reaches GopherTrunk through a **mobile SDR install**: an
[RTL-SDR](/reference/rtl-sdr/) or better front end riding in the car on a
[Raspberry Pi](/reference/raspberry-pi/), fed by a roof-mounted whip. Everything that
makes a mag-mount good for a scanner applies to the SDR — a grounded outdoor vertical
delivers a stronger, cleaner signal than any antenna trapped in the cabin, improving lock
on weak [control channels](/reference/control-channel/) as coverage shifts along a route.
GopherTrunk decodes whatever the front end captures and imposes no antenna requirement,
but the mobile antenna is frequently the difference between holding a
[trunking site](/reference/trunking-site/) and losing it between towns. It cannot, of
course, defeat [encryption](/police-scanner-encryption/) — no antenna decodes AES.

## Where to buy

For scanning from a vehicle, a dual-band magnetic-mount whip like the **Tram 1185**
(around $27) is the standard pick: it sticks to the roof, uses the body as a ground plane,
and covers the VHF/UHF bands where most scanning lives — a large step up from a
[handheld](/reference/handheld-scanner-antenna/) whip held inside the cabin. Match the
connector to your radio (BNC for a Uniden [BC125AT](/reference/uniden-bc125at/), or an
adapter for an SMA scanner), mount it near the roof center, and keep the coax short.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01DY8FNAG?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For a permanent home setup, a fixed rooftop antenna beats any mag-mount — see the
[base scanner antenna](/reference/base-scanner-antenna/) page and the
[best scanner antenna guide](/best-scanner-antenna/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Ground plane](https://en.wikipedia.org/wiki/Ground_plane) — Wikipedia, for why a quarter-wave whip needs a conducting plane and how a vehicle roof supplies one for a mobile monopole.
