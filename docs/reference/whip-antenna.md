---
slug: whip-antenna
title: Whip antenna
entry_type: term
category: antennas
description: A whip antenna is a flexible single-rod monopole, usually a quarter or half wavelength, used on vehicles and handhelds for omnidirectional vertical coverage.
keywords: whip antenna, flexible whip, telescopic antenna, monopole, mobile antenna, rubber duck, quarter-wave whip, vehicle antenna
aka: [whip, flexible whip, telescopic whip, rubber duck]
autolink: true
affiliate: true
product:
  name: "Nagoya NA-771 telescopic whip antenna (SMA-Female)"
  brand: Nagoya
  category: Portable whip antenna
  lowPrice: "15"
  highPrice: "21"
  url: https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20
infobox:
  - { label: Type, value: Flexible monopole }
  - { label: Length, value: ~λ/4 (often shortened/loaded) }
  - { label: Pattern, value: Omnidirectional, vertical }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [monopole-antenna, ground-plane-antenna, dipole-antenna, rtl-sdr, polarization]
cite_urls:
  - https://en.wikipedia.org/wiki/Whip_antenna
  - https://en.wikipedia.org/wiki/Monopole_antenna
faq:
  - q: "Which whip antenna should I buy for a handheld SDR?"
    a: "For portable SDR and scanner use the Nagoya NA-771 (around $18) is a popular telescopic/whip upgrade. Make sure you get the SMA-Female version to mate with most SDR dongles, whose antenna port is an SMA jack — check the connector before ordering, since the NA-771 also ships in SMA-Male and BNC variants."
  - q: "Will a Nagoya whip beat the antenna that came with my dongle?"
    a: "For portable use, usually yes — a full-length telescopic whip you can extend to a quarter wave for the band hears better than a short, poorly grounded rubber duck. The biggest single gain still comes from a real ground plane; a magnetic-mount ground plane or a couple of counterpoise wires under any whip adds several dB."
  - q: "How long should I extend a telescopic whip?"
    a: "To about a quarter wavelength for your target frequency: length in cm ≈ 7125 / f(MHz). A collapsed whip sits far off resonance and hears poorly, so extend it to match the band you are scanning."
  - q: "Does a whip need a ground plane?"
    a: "A quarter-wave whip works against a ground plane — on a handheld the body, your hand, and the coax shield form an imperfect one, which is why rubber ducks are lossy. A half-wave whip (like many NA-771-class antennas at some bands) needs no ground plane and keeps a stable pattern on a poor mount."
---

A **whip antenna** is a [monopole](/reference/monopole-antenna/) built from a single
flexible or telescopic rod, most often a quarter [wavelength](/reference/wavelength/)
long.[^wiki] The name captures its behaviour: a thin, springy conductor that bends and
whips back when disturbed, which is exactly what a vehicle or handheld antenna needs to
survive motion, wind, and knocks. Whips are the most widely deployed antennas in the
world, found on cars, HTs, portable scanners, and every USB SDR dongle.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A flexible vertical whip rod curving slightly at the top, mounted on a base that uses a car roof or dongle body as its ground plane." xmlns="http://www.w3.org/2000/svg">
  <path d="M170 150 C 170 100, 172 70, 182 38" fill="none" stroke="currentColor" stroke-width="3"/>
  <text x="190" y="70" font-size="10" fill="currentColor">flexible λ/4 rod</text>
  <rect x="160" y="150" width="20" height="12" fill="currentColor" opacity="0.7"/>
  <line x1="120" y1="164" x2="220" y2="164" stroke="currentColor" stroke-width="2"/>
  <text x="225" y="168" font-size="9" fill="currentColor">vehicle body / dongle = ground</text>
  <line x1="170" y1="162" x2="170" y2="182" stroke="currentColor" stroke-width="1.5"/>
  <text x="176" y="180" font-size="8" fill="currentColor">feed</text>
  <ellipse cx="176" cy="95" rx="70" ry="18" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
</svg>
<figcaption>A whip is a flexible quarter-wave monopole; the mount surface — a car roof or the SDR's own body — acts as its ground plane.</figcaption>
</figure>

## How it works

Electrically a whip is a [monopole](/reference/monopole-antenna/): a quarter-wave
element that works against a ground plane, which is usually the metal surface it is
mounted on. On a car the roof or trunk lid is the plane; on a handheld or dongle the
body, the operator's hand, and the coax shield form an imperfect but usable ground. Like
any monopole it is vertically [polarized](/reference/polarization/), omnidirectional in
azimuth, and radiates toward the horizon with a null overhead.

Many whips are physically shorter than a true quarter wave. A **loaded whip** inserts an
inductor — a base coil, a centre coil, or a continuous helical winding (the "rubber
duck") — to make an electrically short rod resonate. Loading buys size at a cost:
shortened, loaded whips have lower radiation resistance, narrower
[bandwidth](/reference/bandwidth/), and less [gain](/reference/antenna-gain/) than a
full-length element. A **half-wave whip** goes the other way, using a longer rod with an
end-matching network; it needs no ground plane and keeps a stable pattern on a poor
mount, which is why many aftermarket mobile antennas are 5/8-wave or collinear whips for
extra gain toward the horizon.

Because the ground on a handheld is small and unpredictable, a rubber-duck whip is a
compromise antenna — convenient but lossy. Swapping it for a full quarter-wave element
with a real ground plane routinely adds several dB of usable signal.

## Relevance to SDR

The telescopic or "rubber duck" whip bundled with an [RTL-SDR](/reference/rtl-sdr/) is a
whip antenna, and it is almost always the weakest link in a scanning setup. Its short,
poorly grounded element makes it broad but insensitive. For VHF/UHF trunking, extending a
telescopic whip to a true quarter wave for the band, or replacing it with a
[ground-plane antenna](/reference/ground-plane-antenna/), gives the biggest single
improvement in signal-to-noise available to a GopherTrunk user.

GopherTrunk itself is a receive-only decoder and works with any antenna; the whip's
vertical [polarization](/reference/polarization/) happens to match land-mobile P25, DMR,
and NXDN signals, so a well-mounted vertical whip already suits the traffic GT targets.

## In practice

- **Extend telescopics to λ/4.** Length in cm ≈ 7125 / f(MHz); a collapsed whip is far
  off resonance.
- **Rubber ducks trade size for loss.** Fine for close-in monitoring, poor for weak
  distant sites.
- **Give it a ground.** A magnetic-mount ground plane or a couple of counterpoise wires
  dramatically improves a handheld whip.

## Where to buy

For portable SDR and scanner use, the **Nagoya NA-771** (around $18) is a common
telescopic/whip upgrade over the tiny rubber duck bundled with a dongle. Get the
**SMA-Female** version to mate with most SDR antenna ports, and extend it toward a
quarter wave for the band you are scanning.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For dipoles, discones, and ground-plane options that outperform a handheld whip at a
fixed location, see the [best SDR antenna guide](/best-sdr-antenna/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Whip antenna](https://en.wikipedia.org/wiki/Whip_antenna) — Wikipedia, for the flexible monopole construction, loaded/rubber-duck variants, and mobile mounting as ground plane.
