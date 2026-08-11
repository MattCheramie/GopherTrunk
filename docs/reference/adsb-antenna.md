---
slug: adsb-antenna
title: ADS-B antenna (1090 MHz)
entry_type: hardware
category: antennas
description: "An ADS-B antenna is a vertical tuned to 1090 MHz for tracking aircraft with an SDR; a resonant 1090 MHz antenna mounted high dramatically outperforms a stock dongle whip for flight tracking."
keywords: ADS-B antenna, 1090 MHz antenna, flight tracking antenna, aircraft tracking, FlightAware antenna, dump1090, Mode S antenna, 978 MHz UAT, SDR aviation antenna
aka: [1090 MHz antenna, flight tracking antenna, Mode S antenna]
autolink: true
affiliate: true
product:
  name: "FlightAware 1090/978 MHz ADS-B antenna (N-female, with cable)"
  brand: FlightAware
  category: 1090 MHz ADS-B flight-tracking antenna
  lowPrice: "55"
  highPrice: "75"
  url: https://www.amazon.com/dp/B0BK4N55FY?tag=gophertrunk-20
infobox:
  - { label: Type, value: 1090 MHz vertical (collinear) }
  - { label: Band, value: 1090 MHz ADS-B / 978 MHz UAT }
  - { label: Pattern, value: Omnidirectional, vertical }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0BK4N55FY?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [ads-b, mode-s, monopole-antenna, collinear-antenna, low-noise-amplifier, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast
  - https://en.wikipedia.org/wiki/Collinear_antenna_array
faq:
  - q: "Which antenna should I buy for ADS-B flight tracking?"
    a: "A purpose-built 1090 MHz antenna such as the FlightAware ADS-B antenna (around $65) is the standard pick: it is a gain vertical resonant right at 1090 MHz, so it dramatically outperforms the wideband whip that ships with an SDR dongle. Mount it as high and clear as you can and keep the coax short — at 1 GHz, cable loss is brutal, so a masthead-mounted antenna with a short low-loss feed (or an LNA at the antenna) is what pulls in distant aircraft."
  - q: "Do I need a special antenna, or will my dongle's whip work?"
    a: "A stock whip works for nearby aircraft, but a resonant 1090 MHz antenna is a large upgrade for range. ADS-B lives at 1090 MHz, far above the bands a generic telescopic whip is optimized for, and because the wavelength is only about 27 cm, a proper 1090 antenna is small, cheap, and easy to mount high. Height plus a resonant antenna plus short coax is the recipe for tracking planes 100–250 miles out."
  - q: "What is the difference between 1090 MHz and 978 MHz for ADS-B?"
    a: "1090 MHz (Mode S Extended Squitter) is the worldwide ADS-B frequency used by airliners and most aircraft. 978 MHz (UAT) is a US-only link used mainly by general aviation below 18,000 feet and also carries free weather and traffic uplinks (FIS-B/TIS-B). A 1090-tuned antenna is the priority; some dual-band antennas cover both, which is handy in the US if you also want UAT."
  - q: "Should I add a filter or LNA for ADS-B?"
    a: "Often yes. Strong nearby cellular and FM signals can desensitize a wideband SDR at 1090 MHz, so a 1090 MHz band-pass filter (and a low-noise amplifier at the antenna to overcome coax loss) is a common and effective addition. Mount the LNA at the antenna so its gain is applied before the cable loss, per the Friis budget. Many ADS-B 'pro' dongles bundle the filter in."
---

An **ADS-B antenna** is a vertical antenna tuned to **1090 MHz**, the frequency on which
aircraft broadcast their position, altitude, and identity via
[ADS-B](/reference/ads-b/) (Automatic Dependent Surveillance–Broadcast). Paired with an
[RTL-SDR](/reference/rtl-sdr/) and decoder software such as
[dump1090](/reference/dump1090/), it turns a cheap dongle into a live flight tracker. Like
any resonant antenna it hands the receiver far more signal at its design frequency than a
generic wideband whip, and because 1090 MHz has a wavelength of only about 27 cm, a proper
ADS-B antenna is small, cheap, and easy to mount high.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A short vertical 1090 MHz antenna on a mast with an aircraft above broadcasting to it, a coax running down to an SDR dongle, illustrating a 1090 megahertz flight-tracking setup." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="160" x2="150" y2="60" stroke="currentColor" stroke-width="2"/>
  <line x1="150" y1="60" x2="150" y2="30" stroke="currentColor" stroke-width="3"/>
  <text x="158" y="45" font-size="10" fill="currentColor">1090 MHz vertical</text>
  <path d="M320 40 l24 -8 l-6 12 z" fill="currentColor"/>
  <text x="330" y="26" font-size="9" fill="currentColor">aircraft</text>
  <path d="M312 44 q -80 10 -158 -12" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
  <text x="210" y="60" font-size="8" fill="currentColor">1090 MHz squitter</text>
  <path d="M150 60 C 130 90, 128 120, 138 158" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <rect x="120" y="158" width="46" height="18" rx="3" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="127" y="171" font-size="8" fill="currentColor">SDR dongle</text>
</svg>
<figcaption>Aircraft broadcast their position at 1090 MHz; a resonant 1090 MHz vertical mounted high feeds an SDR and decoder to track them out to a couple hundred miles.</figcaption>
</figure>

## How it works

ADS-B is an unencrypted, open broadcast: transponder-equipped aircraft transmit a **Mode S
Extended Squitter** at 1090 MHz roughly once a second, and anyone with a 1090 MHz receiver
can decode it — there is nothing to break into, unlike the
[encrypted](/police-scanner-encryption/) land-mobile traffic a scanner sometimes hits. The
antenna's whole job is to capture as much of that faint downlink as possible.

Most ADS-B antennas are **gain verticals** — a [collinear](/reference/collinear-antenna/)
stack of co-phased elements that squeezes the pattern toward the horizon, which is exactly
where distant aircraft sit relative to a rooftop antenna. They are omnidirectional in
azimuth, so they hear planes from every bearing, and they are cut sharply for the narrow
1090 MHz band, giving them a big edge over the broad, unresonant whip that ships with a
dongle. Some antennas are **dual-band 1090/978 MHz**, adding the US-only
[UAT](/reference/uat-978/) link used by general aviation and for FIS-B weather and TIS-B
traffic uplinks.

Two install facts dominate ADS-B range:

- **Height and line of sight.** At 1090 MHz reception is essentially line-of-sight to the
  [radio horizon](/reference/radio-horizon/), so every extra foot of antenna height and
  every obstruction removed adds coverage. A clear rooftop mount routinely reaches
  100–250 miles to high-altitude airliners.
- **Coax loss is severe.** At 1 GHz, cable [attenuation](/reference/attenuation/) is high,
  so a long thin feedline can waste most of the signal. Keep the [coax](/reference/coaxial-cable/)
  run short and thick, or — better — mount a [low-noise amplifier](/reference/low-noise-amplifier/)
  and a 1090 MHz band-pass [filter](/reference/rf-filter/) at the antenna so gain and
  selectivity are applied before the cable loss and before strong cellular/FM signals can
  desensitize the SDR.

## Relevance to SDR

ADS-B is one of the most popular first projects for an [SDR](/reference/software-defined-radio/):
an inexpensive [RTL-SDR](/reference/rtl-sdr/), a 1090 MHz antenna, and
[dump1090](/reference/dump1090/) or a similar decoder produce a live map of aircraft
overhead. It is a different application from GopherTrunk — GT decodes VHF/UHF land-mobile
**voice trunking**, not the 1090 MHz aircraft data link — but the hardware philosophy is
identical: a resonant antenna matched to the target frequency, mounted high with a short
low-loss feed, is what separates a toy setup from one that hears distant signals. Many
GopherTrunk users run an ADS-B receiver on the same bench or the same
[Raspberry Pi](/reference/raspberry-pi/), which is why the antenna is documented here.

## Where to buy

For flight tracking, a purpose-built **FlightAware 1090/978 MHz ADS-B antenna** (around
$65) is the standard upgrade over a dongle's stock whip: a gain vertical resonant right at
1090 MHz, with an N connector and a coax run for a masthead mount. Put it as high and clear
as you can, keep the feedline short, and add a 1090 MHz filter/LNA at the antenna if strong
local signals are desensitizing the receiver.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BK4N55FY?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For the decoding side, see [ADS-B](/reference/ads-b/), [Mode S](/reference/mode-s/), and
[dump1090](/reference/dump1090/); for the RF budget, the
[low-noise amplifier](/reference/low-noise-amplifier/) and
[coaxial cable](/reference/coaxial-cable/) pages.

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Automatic Dependent Surveillance–Broadcast](https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast) — Wikipedia, for the 1090 MHz Extended Squitter and 978 MHz UAT links and their open, unencrypted nature.
