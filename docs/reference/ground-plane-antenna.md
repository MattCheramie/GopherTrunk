---
slug: ground-plane-antenna
title: Ground-plane antenna
entry_type: term
category: antennas
description: A ground-plane antenna is a quarter-wave monopole given an artificial ground of radials, forming a self-contained omnidirectional vertical for VHF/UHF work.
keywords: ground plane antenna, ground-plane, quarter-wave vertical, radials, monopole, counterpoise, GPA, omnidirectional vertical
aka: [ground plane antenna, ground-plane vertical, GPA]
autolink: true
affiliate: true
product:
  name: "Diamond X50NA dual-band VHF/UHF base vertical antenna"
  brand: Diamond
  category: VHF/UHF base vertical antenna
  lowPrice: "79"
  highPrice: "99"
  url: https://www.amazon.com/dp/B0FNQDTR34?tag=gophertrunk-20
infobox:
  - { label: Type, value: Monopole with artificial ground }
  - { label: Element, value: λ/4 vertical + 3–4 radials }
  - { label: Pattern, value: Omnidirectional, vertically polarized }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0FNQDTR34?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [monopole-antenna, radials-counterpoise, collinear-antenna, discone-antenna, whip-antenna, base-scanner-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Ground_plane
  - https://en.wikipedia.org/wiki/Monopole_antenna
faq:
  - q: "Which ground-plane or base vertical antenna should I buy for scanning?"
    a: "For a fixed VHF/UHF scanning station the Diamond X50NA (around $85) is a solid, self-contained base vertical: an elevated gain vertical with its own decoupling section, so it hears every trunking site at once without needing a rooftop full of radials. It is vertically polarized to match land-mobile P25, DMR, and NXDN traffic. If you want the widest possible frequency coverage instead of band-optimized gain, a discone or the outdoor pick on the base scanner antenna page is the alternative."
  - q: "Ground plane versus discone for a scanner?"
    a: "A ground-plane or gain vertical is optimized for one band (or a dual band) and gives more signal there; a discone trades gain for sheer bandwidth and covers a decade of frequency from one feedline. If you mostly watch one region's VHF or UHF trunking, a base vertical hears better; if you sweep everything from air band to 800 MHz, a discone is more flexible."
  - q: "Do I still need to lay out radials with a commercial base vertical?"
    a: "No. A packaged base vertical like the X50NA supplies its own artificial ground — a decoupling sleeve or an integrated radial set at the base — so you just clamp it to a mast. A home-brew quarter-wave ground plane, by contrast, needs three or four radials about a quarter wavelength long to work against."
  - q: "How high should I mount a ground-plane antenna?"
    a: "As high and clear as practical. A ground-plane vertical is omnidirectional and radiates toward the horizon, so height and an unobstructed sky view matter more than aiming. Keep the feedline short and low-loss — a long thin coax run can throw away the gain you just paid for."
---

A **ground-plane antenna** is a [monopole](/reference/monopole-antenna/) — a
quarter-[wavelength](/reference/wavelength/) vertical element — mounted over an
**artificial ground** made of a few horizontal or sloping
[radials](/reference/radials-counterpoise/) instead of a solid metal sheet.[^wiki] The
radials supply the electrical image the monopole needs, so the whole assembly is a
self-contained, omnidirectional vertical that does not depend on being near real earth
or a vehicle body. It is one of the most common fixed-station antennas for scanning and
amateur VHF/UHF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A vertical quarter-wave element at a central feedpoint with four radial wires sloping downward from the base, forming an artificial ground plane." xmlns="http://www.w3.org/2000/svg">
  <line x1="230" y1="110" x2="230" y2="30" stroke="currentColor" stroke-width="3"/>
  <text x="238" y="70" font-size="10" fill="currentColor">λ/4 element</text>
  <circle cx="230" cy="112" r="3.5" fill="currentColor"/>
  <text x="240" y="128" font-size="9" fill="currentColor">feedpoint</text>
  <line x1="230" y1="112" x2="120" y2="160" stroke="currentColor" stroke-width="2"/>
  <line x1="230" y1="112" x2="340" y2="160" stroke="currentColor" stroke-width="2"/>
  <line x1="230" y1="112" x2="170" y2="175" stroke="currentColor" stroke-width="2" stroke-opacity="0.7"/>
  <line x1="230" y1="112" x2="300" y2="175" stroke="currentColor" stroke-width="2" stroke-opacity="0.7"/>
  <text x="95" y="172" font-size="9" fill="currentColor">radials (≈λ/4)</text>
  <line x1="230" y1="112" x2="230" y2="135" stroke="currentColor" stroke-width="1.5"/>
  <text x="238" y="150" font-size="8" fill="currentColor">coax</text>
</svg>
<figcaption>A ground-plane antenna is a quarter-wave vertical fed against three or four radials that stand in for a solid ground plane.</figcaption>
</figure>

## How it works

The driven element is an ordinary quarter-wave [monopole](/reference/monopole-antenna/):
it needs a conducting surface beneath it to mirror the element into an effective
half-wave [dipole](/reference/dipole-antenna/). A solid sheet works, but so does a
"skeleton" of a few [radial](/reference/radials-counterpoise/) wires, each about a
quarter wavelength long. Three or four radials are enough to approximate the current
distribution of a continuous plane, keeping the antenna light and wind-transparent while
still supplying the return path.

Radial geometry sets the feedpoint impedance. With the radials **horizontal**, an ideal
ground-plane presents roughly **37 Ω** — the monopole value — a poor match to 50 Ω coax.
Sloping the radials **downward at about 45°** raises the feedpoint impedance to near
**50 Ω**, giving a low [SWR](/reference/standing-wave-ratio/) directly on standard coax
without a matching network. Drooping the radials also lifts the main lobe slightly and
lowers the take-off angle, favouring distant signals near the horizon.

The result is an omnidirectional azimuth [pattern](/reference/radiation-pattern/), a null
overhead, and vertical [polarization](/reference/polarization/) — matched to the vertical
land-mobile signals a scanner listens to. Modest [gain](/reference/antenna-gain/) over a
dipole comes from concentrating radiation into the upper half-space.

## Relevance to SDR

For fixed SDR scanning of VHF/UHF trunked systems, a ground-plane antenna is often the
best value: it is broadband enough to cover a whole band segment, omnidirectional so it
hears all sites at once, and vertically polarized to match the target signals. Many
commercial "discone-lite" and dedicated scanner verticals are ground-plane designs.

GopherTrunk decodes whatever the front end delivers and cares only about
signal-to-noise. Because a ground-plane antenna can be built for pennies from wire and a
coax connector, and mounted high with a clear horizon, it is a common first upgrade over
the stock telescopic [whip](/reference/whip-antenna/) that ships with an
[RTL-SDR](/reference/rtl-sdr/) dongle.

## In practice

- **Cut for the band centre.** Element ≈ 7125 / f(MHz) in cm; radials a touch longer than
  the element.
- **Slope radials for match.** Horizontal ≈ 37 Ω; ~45° droop ≈ 50 Ω for a clean match.
- **More radials, diminishing returns.** Four beats three; beyond about four the gain is
  marginal for an elevated ground-plane.

## Where to buy

For a fixed VHF/UHF scanning station, a commercial base vertical like the **Diamond
X50NA** (around $85) is the easy upgrade over a stock [whip](/reference/whip-antenna/):
an elevated, self-contained gain vertical with its own decoupling section, so it mounts
to a mast with no radial field to lay out and hears every
[trunking site](/reference/trunking-site/) at once. Its vertical
[polarization](/reference/polarization/) matches land-mobile traffic, and no antenna —
this one included — can recover [AES-encrypted](/police-scanner-encryption/) audio; the
antenna only decides how well the signal reaches the SDR.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0FNQDTR34?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For a wideband alternative, a higher-gain [collinear](/reference/collinear-antenna/), or
the single outdoor pick, see the [base scanner antenna](/reference/base-scanner-antenna/)
page and the [best scanner antenna guide](/best-scanner-antenna/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Ground plane](https://en.wikipedia.org/wiki/Ground_plane) — Wikipedia, for the radial ground-plane antenna, its ~37 Ω horizontal feedpoint, and the 45° droop that raises it toward 50 Ω.
