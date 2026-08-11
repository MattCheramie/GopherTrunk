---
slug: antenna-mast
title: Antenna mast
entry_type: hardware
category: mounting
description: "An antenna mast is the pole that lifts a scanner antenna above rooflines and obstructions — the single biggest free upgrade to what your GopherTrunk SDR can hear."
keywords: antenna mast, push up mast, telescoping mast, antenna pole, mast pipe, fiberglass mast, TV antenna mast, scanner antenna mast, SDR antenna height
aka: [mast, push-up mast, antenna pole, mast pipe]
autolink: true
affiliate: true
product:
  name: "Easy Up 20' 9\" Telescoping Push-Up Antenna Mast"
  brand: Easy Up
  category: Telescoping antenna mast
  lowPrice: "69"
  highPrice: "99"
  url: https://www.amazon.com/dp/B019FVJKAA?tag=gophertrunk-20
infobox:
  - { label: Type, value: Telescoping push-up pole }
  - { label: Typical height, value: 20–30 ft }
  - { label: Material, value: Steel / aluminium / fiberglass }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B019FVJKAA?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [tripod-mount, eave-mount, chimney-mount, guy-wire-kit, discone-antenna, coaxial-cable]
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_masts_and_towers
faq:
  - q: "Which antenna mast should I buy for GopherTrunk scanning?"
    a: "For most rooftop or deck installs a telescoping push-up mast like the Easy Up 20' 9\" (around $80) is the right answer — it collapses to about 5 ft for shipping and storage, then extends to clear your roofline. Pair it with a wall bracket or tripod at the base; a single unguyed 20 ft push-up mast usually needs one support point plus a guy set above about 15 ft of exposed pole."
  - q: "How high should a scanner antenna mast be?"
    a: "Height is the single biggest free improvement to VHF/UHF reception, which is line-of-sight. Getting a discone or scanner antenna above the ridgeline and nearby obstructions matters far more than antenna gain. Twenty to thirty feet clears most single-storey rooflines; beyond that you are into guyed-mast or tower territory and should think hard about wind load and grounding."
  - q: "Steel, aluminium, or fiberglass mast?"
    a: "Galvanised steel push-up masts are cheapest and stiffest but heavy. Aluminium is lighter and rust-free but flexes more. Fiberglass is non-conductive, which is ideal directly around an antenna's feedpoint and avoids detuning, but it is more expensive and less rigid. For a plain vertical scanner antenna any of the three works; keep the top section metal-free only if the antenna maker calls for it."
  - q: "Do I need to guy a push-up mast?"
    a: "Any exposed mast taller than about 10–15 ft above its top support should be guyed. A guy wire kit at one or two levels stops the pole whipping in wind, which otherwise fatigues the base bracket and can bring the whole thing down. Guying is cheap insurance and it also steadies the antenna so the pattern stays put."
---

An **antenna mast** is the pole that raises a [scanner antenna](/best-scanner-antenna/)
above your roofline, trees, and neighbouring buildings. It is the most boring part of an
outdoor install and, dollar for dollar, the most important: because VHF and UHF reception is
essentially line-of-sight, **height buys you more signal than almost anything else you can
add** — more than [antenna gain](/reference/antenna-gain/), more than a bigger
[SDR](/best-sdr-for-gophertrunk/), often more than a mast-mounted
[LNA](/best-sdr-lna/). A mast is just plumbing, but it is the plumbing that decides how many
[trunking sites](/reference/trunking-site/) your [GopherTrunk](/downloads.html) receiver can
hear.

## What it is

A mast is a length of pipe — commonly 1 to 2 inches in outer diameter — that carries the
antenna at the top and clamps to a fixed support at the bottom. There are three common
forms:

- **Telescoping "push-up" masts.** Nested sections that ship collapsed (often to about 5 ft)
  and slide up to full height, locking with clamps or pins. The convenient choice for one
  person and a ladder; typical heights are 20–30 ft.
- **Fixed sections of mast pipe.** Straight galvanised or aluminium tube, sold by the foot or
  in 5–10 ft joints, coupled with swaged ends or [mast clamps](/reference/mounting-hardware/).
  Stiffer and cheaper per foot, but you assemble the height you want.
- **Fiberglass poles.** Non-conductive, so they will not detune an antenna near the feedpoint
  and are popular for the top few feet even when the rest of the run is metal.

Whatever the material, the mast does not stand on its own. It bolts to a
[wall or eave bracket](/reference/eave-mount/), a [rooftop tripod](/reference/tripod-mount/),
or a [chimney strap mount](/reference/chimney-mount/), and anything tall enough to catch the
wind gets a [guy wire kit](/reference/guy-wire-kit/) as well.

## How height helps

At the frequencies scanners live on — VHF around 150 MHz, UHF around 450 MHz, and the
700/800 MHz trunked public-safety bands — radio travels in near-straight lines. A hill, a
roof ridge, or a stand of trees between you and a
[control channel](/reference/trunked-radio/) attenuates it sharply. Raising the antenna does
two things: it lifts the antenna's take-off angle clear of nearby clutter, and it extends the
distance to the radio horizon (which grows with the square root of height). A discone sitting
in the attic might hear one weak site; the same discone on a 25 ft mast above the ridge can
hear several.

Height also cuts local noise. Household electronics, LED lighting, and switching supplies
radiate a broadband hash that is strongest near the ground. Getting the antenna up and away
from the house lowers that noise floor, which improves the signal-to-noise ratio the
[demodulator](/reference/automatic-gain-control/) sees just as surely as more signal would.

## Relevance to GopherTrunk

GopherTrunk decodes whatever the front end delivers and knows nothing about your mast — but
the mast is where most real-world decode reliability is won or lost. A marginal P25 or TETRA
site that produces garbled voice from an indoor whip will often lock cleanly once the antenna
clears the roofline, because the improvement is in raw carrier-to-noise, upstream of every
DSP trick. Before reaching for a [filter](/sdr-filters/) or a
[low-noise amplifier](/best-sdr-lna/) to chase a weak system, put the antenna higher; it is
cheaper and it fixes the actual problem more often. (No amount of height, of course, decodes
[AES-encrypted](/police-scanner-encryption/) talkgroups — nothing does.)

Mind the feedline, too. A taller mast means a longer run of
[coax](/reference/coaxial-cable/), and coax loss climbs with frequency, so use decent low-loss
cable up the mast and terminate it in weatherproof [N-type](/reference/n-type-connector/)
connectors rather than throwing away your new height in thin lossy jumper.

## Where to buy

For most base installs a **telescoping push-up mast** is the practical pick: the Easy Up
20' 9" (around $80) collapses to about 5 ft for shipping and storage, then extends to clear a
single-storey roofline. Bolt the base to a
[wall/eave bracket](/reference/eave-mount/) or a [tripod](/reference/tripod-mount/), and add a
[guy wire kit](/reference/guy-wire-kit/) once you are much past 15 ft of exposed pole.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B019FVJKAA?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For how the mast fits into a complete outdoor stack — bracket, grounding, arrestor, and
weatherproofing — see the
[antenna mast and mounting guide](/antenna-mast-and-mounting-guide/) and the
[outdoor base build](/gophertrunk-outdoor-base-build/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Radio masts and towers](https://en.wikipedia.org/wiki/Radio_masts_and_towers) — Wikipedia, on mast construction, guying, and why antenna height governs line-of-sight coverage.
