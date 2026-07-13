---
slug: e4000-tuner
title: E4000 tuner
entry_type: hardware
category: sdr-devices
description: "E4000 is Elonics' wideband CMOS tuner chip used in early RTL-SDR dongles, prized for extended reach but with a coverage gap and now discontinued."
keywords: E4000, Elonics E4000, tuner chip, RTL-SDR tuner, wideband CMOS tuner, coverage gap, extended range, R820T2 alternative, direct conversion tuner
aka: [E4000, Elonics E4000, E4k]
autolink: true
infobox:
  - { label: Type, value: RF tuner chip (CMOS) }
  - { label: Vendor, value: Elonics }
  - { label: Architecture, value: Zero-IF / direct conversion }
  - { label: Range, value: "~52 – 2200 MHz (gap ~1100–1250)" }
  - { label: Used in, value: RTL-SDR / NESDR XTR dongles }
  - { label: TX, value: No (receive tuner) }
  - { label: Status, value: Discontinued }
see_also: [rtl2832u, r820t-tuner, rtl-sdr, nesdr, direct-conversion-receiver, local-oscillator]
cite_urls:
  - https://en.wikipedia.org/wiki/RTL-SDR
  - https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr
---

**E4000** is a wideband CMOS **RF tuner chip** made by the Scottish company **Elonics**,
and one of the two tuners — alongside the Rafael Micro [R820T2](/reference/r820t-tuner/) —
that defined the early [RTL-SDR](/reference/rtl-sdr/) era.[^wiki] A tuner sits between the
antenna and the [RTL2832U](/reference/rtl2832u/) demodulator, mixing the frequency you
want down toward baseband; the E4000 was valued for reaching **higher** in frequency than
any other RTL-SDR tuner, at the cost of a gap in its coverage and more noise.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the Elonics E4000 tuner, roughly 52 to 2200 megahertz with a gap around 1100 to 1250 megahertz, on an axis from 0 to about 2.4 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="197" y="86">1 GHz</text><text x="363" y="86">2 GHz</text><text x="430" y="86">2.4</text></g>
  <rect x="39" y="40" width="145" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <rect x="238" y="40" width="127" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="211" y="55" text-anchor="middle" font-size="7" fill="currentColor">gap</text>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">E4000 coverage (~52–2200 MHz, with a gap)</text>
</svg>
<figcaption>The E4000 tunes higher than most RTL-SDR tuners but has a dead zone around 1100–1250 MHz where its synthesizer cannot lock.</figcaption>
</figure>

## Overview

The E4000 is a **zero-IF (direct-conversion)** tuner: it mixes the target frequency
directly to a complex baseband using an on-chip
[local oscillator](/reference/local-oscillator/), which the RTL2832U then digitises. Its
distinguishing spec is reach — roughly **52 MHz up to about 2200 MHz** — noticeably
higher than the ~1766 MHz ceiling of the R820T2. That extra top end made E4000 dongles
the tool of choice for anyone needing the 1.8–2.2 GHz range (some aeronautical, pager,
and early cellular monitoring) that the newer R820T2 simply cannot see.

## The coverage gap and the noise trade-off

Two caveats came with the reach. First, the E4000 has a **coverage gap** — a band roughly
**1100–1250 MHz** where its frequency synthesizer cannot lock, leaving a dead zone in the
middle of its otherwise wide range. Second, across the frequencies most scanning cares
about (VHF/UHF), the E4000 is generally **noisier and less sensitive** than the
[R820T2](/reference/r820t-tuner/), and it has a larger DC spike at the tuning centre
typical of direct-conversion designs. So for everyday reception the R820T2 usually wins;
the E4000 was the specialist's choice for its ceiling, not its floor.

## Discontinuation

The E4000 became scarce when **Elonics ceased trading around 2012**, ending production of
the chip just as the RTL-SDR hobby was taking off. Existing stock was consumed, prices on
the used and stockpiled market rose, and the R820T2 (later re-released as the R860)
settled in as the de-facto standard tuner because it was cheaper, more sensitive across
the popular bands, and still in production. Today an E4000-based dongle is a
premium/legacy item bought specifically for the extended top-end coverage — Nooelec's
[NESDR](/reference/nesdr/) SMArt XTR line is the main way to still get one new.

## Relevance to GopherTrunk

To GopherTrunk the tuner is essentially invisible: the [RTL2832U](/reference/rtl2832u/)
presents the same raw-IQ interface regardless of whether an E4000 or an R820T2 sits in
front of it, so GopherTrunk drives an E4000 dongle exactly like any other
[RTL-SDR](/reference/rtl-sdr/). The tuner choice matters only for **what frequencies you
can reach and how cleanly**: an E4000 buys coverage above ~1.77 GHz that an R820T2 lacks,
but for the VHF/UHF land-mobile trunking GopherTrunk decodes, an R820T2/R860 unit is the
more sensitive and more available pick. Mind the ~1100–1250 MHz gap if a target sits
there.

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/RTL-SDR) — Wikipedia, on RTL-SDR tuner chips including the Elonics E4000 and its coverage.
[^osmo]: [rtl-sdr project wiki](https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr) — Osmocom, documenting supported tuners, the E4000 range and its gap.
