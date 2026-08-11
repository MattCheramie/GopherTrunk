---
slug: usb-extension-cable
title: USB extension cable
entry_type: hardware
category: rf-front-end
description: "An active, shielded USB extension cable places an SDR dongle at the window or antenna while the PC stays elsewhere — keeping the lossy RF run short and moving data over USB instead."
keywords: USB extension cable, active USB cable, USB repeater, shielded USB, SDR at window, RTL-SDR extension, ferrite USB, USB noise, place dongle near antenna
aka: [active USB cable, USB repeater cable, USB extender, shielded USB extension]
autolink: true
affiliate: true
product:
  name: "Active USB 2.0 extension cable (shielded, with signal booster)"
  brand: Generic
  category: Active USB extension cable
  lowPrice: "9"
  highPrice: "16"
  url: https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Active/repeater USB cable" }
  - { label: Signal, value: "Powered booster chip in-line" }
  - { label: Length, value: "16–32 ft (5–10 m) typical" }
  - { label: Shielding, value: "Foil/braid + ferrite recommended" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [coaxial-cable, coax-feedline, rtl-sdr, sma-connector, low-noise-amplifier, coax-pigtail]
cite_urls:
  - https://en.wikipedia.org/wiki/USB
faq:
  - q: "Which USB extension cable should I buy for an SDR?"
    a: "An active (repeater) shielded USB extension cable of 16–32 ft (around $12) is the pick. The 'active' part matters: a powered booster chip in the connector keeps the USB signal valid past the ~15 ft where a plain passive cable fails. It lets you put the SDR dongle right at a window or antenna and keep the PC across the room, so the lossy RF run stays short and the data travels over USB instead."
  - q: "Why extend USB instead of running longer coax?"
    a: "Because coax loss climbs steeply with frequency and USB loss does not. At the 700/800 MHz bands trunked systems use, ten feet of thin coax can cost several dB, while ten feet of active USB costs essentially nothing to the decoded signal. Moving the SDR to the antenna and sending digital samples down USB is the single cleanest way to beat feedline loss — 'short is better than good' for the RF run."
  - q: "Do passive USB extensions work for an SDR?"
    a: "Only for very short reaches. USB 2.0 is specified to about 5 m (16 ft) per cable, and cheap passive extensions often fail well before that, showing up as dropouts, sample overruns, or a dongle that will not enumerate. Use an active/repeater cable for anything past a few feet, and avoid stacking passive extensions."
  - q: "How do I stop a USB extension from adding noise?"
    a: "Use a well-shielded cable and clip a ferrite choke on each end — the ferrite suppresses common-mode current that would otherwise radiate USB hash into your antenna and raise the noise floor. Keep the USB run away from the coax and the antenna, and power a hungry SDR from a clean supply. A noisy or unshielded extension can undo the very sensitivity you moved the dongle to gain."
  - q: "Will a long USB cable underpower my dongle?"
    a: "It can. Voltage drop over a long thin cable can starve a power-hungry SDR (or its bias-tee LNA), causing instability. Pick an active cable with adequate conductor gauge, keep the total length reasonable, and if needed use a powered USB hub or a cable with an auxiliary power input at the far end."
---

An **active USB extension cable** places the SDR dongle where the signal is — at a window or
right below the antenna — while the PC stays across the room. It solves the same problem as
low-loss [coax](/reference/coax-feedline/) but from the other direction: instead of carrying
lossy RF a long way to the radio, you move the radio to the antenna and carry the *digital*
[IQ samples](/reference/iq-data/) the long way over USB, where distance costs almost nothing.
The word **active** is the key — a powered repeater chip in the connector keeps the USB
signal valid past the ~15 ft where a plain passive cable quits.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="An SDR dongle at a window feeding a short coax jumper to an antenna, with a long active USB cable running across the room to a PC — the RF run is short and the USB run is long." xmlns="http://www.w3.org/2000/svg">
  <line x1="70" y1="15" x2="70" y2="135" stroke="currentColor" stroke-width="2"/>
  <text x="70" y="12" font-size="8" fill="currentColor" text-anchor="middle">window</text>
  <line x1="90" y1="20" x2="90" y2="70" stroke="currentColor" stroke-width="2"/>
  <text x="98" y="30" font-size="9" fill="currentColor">antenna</text>
  <rect x="72" y="72" width="46" height="22" rx="3" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.3"/>
  <text x="95" y="87" font-size="9" fill="currentColor" text-anchor="middle">SDR</text>
  <path d="M90 70 L95 72" stroke="currentColor" stroke-width="1.5"/>
  <text x="130" y="64" font-size="8" fill="currentColor">short RF</text>
  <line x1="118" y1="83" x2="360" y2="83" stroke="currentColor" stroke-width="2" stroke-dasharray="5 3"/>
  <circle cx="239" cy="83" r="7" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="239" y="72" font-size="7" fill="currentColor" text-anchor="middle">booster</text>
  <text x="239" y="100" font-size="8" fill="currentColor" text-anchor="middle">long active USB</text>
  <rect x="360" y="66" width="70" height="36" rx="4" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="395" y="88" font-size="10" fill="currentColor" text-anchor="middle">PC</text>
</svg>
<figcaption>Keep the lossy RF run short at the window; send digital samples the long way over an active USB cable.</figcaption>
</figure>

## How it works

A USB 2.0 link is specified to roughly 5 m (16 ft) per cable, and cheap passive extensions
often fall short of even that — the signal degrades until the host sees dropouts, sample
**overruns**, or a dongle that never enumerates. An **active** (repeater) cable embeds a
small powered chip that re-clocks and re-drives the USB signal partway along, extending the
usable reach to 5–10 m and beyond with chained units. That is what lets the dongle sit at
the antenna while the computer stays at the desk.

The reason to bother is loss asymmetry. Coax [attenuation](/reference/attenuation/) climbs
steeply with frequency, so at 700/800 MHz a modest length of thin cable throws away several
dB before the signal is ever digitised. USB carries already-digitised samples, which do not
degrade with cable length the way an analog RF signal does. So the winning layout is a
**short** RF run — dongle to antenna via a [pigtail](/reference/coax-pigtail/) or short
[feedline](/reference/coax-feedline/) — and a **long** USB run back to the PC. It is the
concrete form of the rule "short is better than good" for the RF side.

## Watch the noise

The catch is that a USB cable is itself a noise source sitting near your antenna. Switching
noise and common-mode current on an unshielded or poorly-grounded extension can radiate USB
hash straight into the front end, raising the noise floor exactly where you are trying to
hear weak signals. Two cheap defenses matter:

- **Shielding.** Use a cable with real foil/braid shielding, not a bargain unshielded one.
- **Ferrites.** Clip a ferrite choke on each end (and route the USB away from the coax and
  the antenna) to suppress the common-mode current that does the radiating.

Also mind **power**: voltage drop over a long thin cable can starve a hungry SDR or its
[bias-tee LNA](/reference/bias-tee/), causing instability — pick an active cable with
adequate conductor gauge, or feed the far end from a powered hub.

## Relevance to GopherTrunk

GopherTrunk decodes the [IQ stream](/reference/iq-data/) the SDR delivers over USB and never
sees the antenna directly, so moving the dongle to the window and streaming samples back is
transparent to the software — it simply arrives with a better signal-to-noise ratio because
the RF never had to survive a long lossy cable. For a fixed listening post this is often the
single biggest, cheapest improvement to decode reliability on the UHF trunking bands, short
of a [mast-mounted LNA](/best-sdr-lna/). As always, the plumbing only governs *how cleanly*
clear traffic arrives — no cable makes [encrypted](/police-scanner-encryption/) traffic
decodable.

## Where to buy

Buy an **active (repeater), shielded USB extension** of 16–32 ft (around $12) and put the
SDR at the window on a short [pigtail](/reference/coax-pigtail/) to the antenna. Add a couple
of clip-on ferrites if you do not already have them. This keeps the RF run short — the whole
point — while the PC lives comfortably across the room.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For the alternative — running low-loss coax instead — see
[coax feedline](/reference/coax-feedline/) and the
[SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [USB](https://en.wikipedia.org/wiki/USB) — Wikipedia, for the USB 2.0 cable-length limit and the role of active/repeater cables in extending it.
