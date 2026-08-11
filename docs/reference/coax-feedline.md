---
slug: coax-feedline
title: Coax feedline
entry_type: hardware
category: rf-front-end
description: "A coax feedline is the low-loss outdoor cable run — RG8X, LMR-240, or LMR-400 with connectors — that carries signal from a rooftop antenna down to your SDR without bleeding it away."
keywords: coax feedline, low loss coax, LMR-400, LMR-240, RG8X, outdoor antenna cable, feedline run, mast to radio cable, N connector feedline, rooftop antenna coax
aka: [feedline, antenna feedline, coax run, low-loss coax]
autolink: true
affiliate: true
product:
  name: "LMR-240/RG8X coax jumper, PL-259 both ends (made-to-length)"
  brand: ABR Industries
  category: Low-loss outdoor coax feedline
  lowPrice: "25"
  highPrice: "60"
  url: https://www.amazon.com/dp/B07F23HK4C?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Low-loss feedline run" }
  - { label: Cable, value: "RG8X / LMR-240 / LMR-400" }
  - { label: Ends, value: "PL-259, or N for UHF" }
  - { label: Use, value: "Rooftop antenna to radio" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07F23HK4C?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [coaxial-cable, n-type-connector, uhf-connector-pl259, coax-pigtail, usb-extension-cable, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Coaxial_cable
faq:
  - q: "Which coax feedline should I buy for a rooftop SDR antenna?"
    a: "Buy a made-to-length low-loss run — RG8X or LMR-240 for moderate distances, LMR-400 for long runs or the 700/800 MHz bands — with connectors already fitted. A factory-terminated 25 ft LMR-240/RG8X jumper (around $30) covers a typical run and spares you soldering PL-259s. For the UHF trunking bands, order it with N connectors rather than PL-259, which gets lossy above ~300 MHz. Do not use thin RG58/RG316 for an outdoor feedline."
  - q: "How much cable loss is too much?"
    a: "Six dB of feedline loss throws away three-quarters of your signal power before it reaches the SDR, and loss climbs steeply with frequency — a cable that is fine at VHF can be a poor choice at 800 MHz. Keep the run as short as you can, use thick low-loss cable, and if the run must be long, put a mast-mounted LNA at the antenna so its gain is applied before the loss."
  - q: "RG8X, LMR-240, or LMR-400 for a feedline?"
    a: "RG8X and LMR-240 are a good middle ground — noticeably lower loss than RG58, still flexible enough to route easily, fine for runs up to roughly 50 ft at VHF/UHF. LMR-400 is thicker, stiffer, and lowest-loss, the choice for long masthead runs or the 700/800 MHz P25 bands where every dB counts. Match the cable to the distance and frequency rather than overbuying stiff cable for a short bench run."
  - q: "Should my feedline use PL-259 or N connectors?"
    a: "For HF/VHF, PL-259/SO-239 is fine and common. For the UHF land-mobile trunking bands, prefer N: it is weatherproof and holds a clean 50 Ω match where the non-constant-impedance UHF connector adds reflection and loss above a few hundred MHz. Whichever you choose, weatherproof the outdoor joint — water in coax raises loss permanently."
  - q: "Feedline or USB extension — which is better?"
    a: "If you can, avoid a long RF run altogether: put the SDR at the window on a short pigtail and send the digital samples back over an active USB extension, since USB length costs almost nothing while coax length costs signal. Use a real low-loss feedline when the antenna genuinely must be far from any PC — a rooftop or mast install — and then keep it short, thick, and sealed."
---

A **coax feedline** is the outdoor cable run that carries signal from a rooftop or
mast-mounted [antenna](/reference/antenna/) down to your receiver. Unlike a bench
[pigtail](/reference/coax-pigtail/), a feedline covers real distance, so its one job is to
deliver that distance **with as little loss as possible** — which means thick, low-loss
[coax](/reference/coaxial-cable/) (RG8X, LMR-240, or LMR-400) with proper connectors, not
the thin RG58 or RG316 that bleeds signal away over a long run. This page is about *buying
and specifying* a feedline; for the physics of why cable loss climbs with frequency, see
the [coaxial cable](/reference/coaxial-cable/) reference.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A rooftop antenna at the top of a mast connected by a thick low-loss feedline running down to an SDR at the bottom, with a note that thin cable would lose signal over the same run." xmlns="http://www.w3.org/2000/svg">
  <line x1="90" y1="20" x2="90" y2="60" stroke="currentColor" stroke-width="2.5"/>
  <line x1="72" y1="24" x2="108" y2="24" stroke="currentColor" stroke-width="2"/>
  <text x="118" y="28" font-size="9" fill="currentColor">rooftop antenna</text>
  <line x1="90" y1="60" x2="90" y2="140" stroke="currentColor" stroke-width="4"/>
  <text x="100" y="105" font-size="9" fill="currentColor">thick low-loss coax</text>
  <line x1="60" y1="145" x2="400" y2="145" stroke="currentColor" stroke-width="1.4"/>
  <text x="60" y="160" font-size="8" fill="currentColor">roof line</text>
  <rect x="66" y="140" width="48" height="22" rx="3" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.3"/>
  <text x="90" y="155" font-size="9" fill="currentColor" text-anchor="middle">SDR</text>
  <g font-size="9" fill="currentColor"><text x="300" y="90" text-anchor="middle">every dB here</text><text x="300" y="104" text-anchor="middle">is gone before</text><text x="300" y="118" text-anchor="middle">the SDR sees it</text></g>
</svg>
<figcaption>A feedline's job is distance with minimal loss — thick low-loss coax and weatherproof connectors, sized to the run.</figcaption>
</figure>

## Choosing the cable

The trade is loss versus flexibility, and both are set by the cable's thickness and
construction:

- **RG8X / LMR-240** — a practical middle ground: noticeably lower loss than RG58, still
  flexible enough to route easily. Good for runs up to roughly 50 ft at VHF/UHF.
- **LMR-400 / RG213** — thick, stiffer, and lowest-loss; the choice for long masthead runs
  or the 700/800 MHz P25 bands where every dB counts.
- **RG58 / RG316** — thin patch-lead cable. Fine for a short jumper, **wrong** for an
  outdoor feedline: it throws away too much signal over distance, especially at UHF.

Because [attenuation](/reference/attenuation/) grows with frequency, match the cable to both
the *distance* and the *band*. A 25 ft RG8X run that is unremarkable at VHF becomes worth
upgrading to LMR-400 at 800 MHz. And buy it **made to length with connectors already
fitted** unless you enjoy soldering [PL-259s](/reference/uhf-connector-pl259/) — a
factory-terminated jumper is weatherproof and correctly matched out of the bag.

## Connectors and weatherproofing

For the UHF land-mobile trunking bands, order the feedline with
[N connectors](/reference/n-type-connector/) rather than PL-259: N is weatherproof and holds
a clean 50 Ω match, while the non-constant-impedance UHF connector adds reflection and loss
above a few hundred MHz. For HF/VHF, PL-259/SO-239 is fine and common. Either way, the last
short hop into the dongle steps down to [SMA](/reference/sma-connector/) with a
[pigtail](/reference/coax-pigtail/) or [adapter](/reference/sma-adapter-kit/).

Whatever connector you use, **weatherproof every outdoor joint** with self-amalgamating tape
or a proper boot. Water in coax raises loss permanently and corrodes the connectors — the
slow death of many rooftop antennas.

## Relevance to GopherTrunk

GopherTrunk decodes whatever the SDR captures and has no view of the cable, so the feedline
is precisely where a good rooftop antenna's signal is most often lost before digitisation. A
long or low-grade run quietly undoes the antenna: 6 dB of feedline loss discards
three-quarters of the signal power. Two levers fix it — keep the run **short and thick**, or
mount a [low-noise amplifier](/best-sdr-lna/) at the antenna (fed via a
[bias tee](/reference/bias-tee/)) so its gain lands *before* the cable loss. Better still,
where the antenna can be near a PC, skip the long RF run entirely: put the SDR at the window
and send samples back over a [USB extension](/reference/usb-extension-cable/), since USB
length costs nothing while coax length costs signal. None of this changes what is decodable
— [encrypted](/police-scanner-encryption/) traffic stays encrypted — but it sets the
signal-to-noise every clear decode depends on.

## Where to buy

For a real rooftop or mast run, buy a **made-to-length low-loss feedline** with connectors
fitted — a factory-terminated **LMR-240/RG8X** jumper (around $30 for 25 ft) covers a
typical run, stepping up to **LMR-400** for long runs or the 700/800 MHz bands. Order the
length you actually need and, for UHF, choose the **N-connector** variant over PL-259.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07F23HK4C?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For the loss physics behind the cable choice, see [coaxial cable](/reference/coaxial-cable/);
for the whole RF chain, the [SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Coaxial cable](https://en.wikipedia.org/wiki/Coaxial_cable) — Wikipedia, on coax construction and the frequency- and length-dependent loss that drives feedline choice.
