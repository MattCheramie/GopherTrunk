---
slug: airspy-hf-plus
title: Airspy HF+
entry_type: hardware
category: sdr-devices
description: "Airspy HF+ is a software-defined radio optimised for the HF and low-VHF bands, with very high dynamic range for receiving shortwave and weak low-band signals."
keywords: Airspy HF+, Airspy HF plus, HF+ Discovery, HF SDR, shortwave receiver, high dynamic range, polyphase harmonic rejection, low VHF, 18-bit
aka: [Airspy HF+, Airspy HF plus, HF+ Discovery]
autolink: true
affiliate: true
product:
  name: "Airspy HF+ Discovery"
  brand: Airspy
  category: Software-defined radio
  lowPrice: "155"
  highPrice: "185"
  url: https://www.amazon.com/s?k=Airspy+HF+Discovery&tag=gophertrunk-20
infobox:
  - { label: Type, value: HF / low-VHF SDR receiver }
  - { label: Vendor, value: Airspy }
  - { label: Strength, value: Very high dynamic range on low bands }
  - { label: Range, value: "~9 kHz – 31 MHz, 60–260 MHz" }
  - { label: Bandwidth, value: up to ~768 kHz }
  - { label: TX, value: No (receive only) }
  - { label: Price, value: around $170 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Airspy+HF+Discovery&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [airspy, rtl-sdr, sdrplay-rsp1a, funcube-dongle, upconverter, ionospheric-propagation, frequency-bands, dynamic-range]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://airspy.com/airspy-hf-discovery/
faq:
  - q: "Is the Airspy HF+ Discovery the right SDR for GopherTrunk?"
    a: "Only if the low bands are your target. GopherTrunk decodes land-mobile trunking (P25/DMR/NXDN/TETRA) that lives in VHF/UHF, where an Airspy R2 or an RTL-SDR is the right radio. The HF+ Discovery is the pick when you also want shortwave, amateur SSB/CW or marine HF — its dynamic range is the best in class on a crowded low band. GopherTrunk can drive it as a receiver, but its wideband-channelizer strengths don't apply to the HF+'s narrow captures."
  - q: "Why choose the HF+ Discovery over an RTL-SDR with an upconverter for HF?"
    a: "Dynamic range. On HF the challenge is not sensitivity but surviving dozens of strong broadcasters at once. The HF+ Discovery's polyphase harmonic-rejection mixer and high-resolution ADC hear a weak signal next to a broadcaster tens of dB louder, where an 8-bit RTL-SDR plus a lossy upconverter or direct-sampling hack folds under overload."
  - q: "How wide a capture does the HF+ Discovery give?"
    a: "Up to about 768 kHz — deliberately narrow. That is all a shortwave, marine or amateur channel needs, and keeping the window small is exactly what lets the ADC deliver its very high dynamic range. It will not give you the 10 MHz-wide slice an Airspy R2 does."
  - q: "Can it decode encrypted transmissions?"
    a: "No. The HF+ Discovery is receive-only and GopherTrunk is a receive-only decoder — no SDR or scanner can decode AES-encrypted traffic."
---

**Airspy HF+** is a [software-defined radio](/reference/software-defined-radio/)
optimised for the **HF and low-VHF** [bands](/reference/frequency-bands/), with very high
dynamic range for receiving shortwave and weak low-band signals.[^wiki] Where the VHF/UHF
[Airspy R2](/reference/airspy/) chases wide bandwidth, the HF+ chases the opposite goal:
a narrow, exceptionally clean window on a crowded low band where a single strong
broadcaster can otherwise desensitise the whole receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A frequency coverage bar showing the Airspy HF+ covering HF up to about 31 megahertz and a separate low-VHF segment from 60 to 260 megahertz, on an axis from 0 to about 300 megahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="78" x2="430" y2="78" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="94">0</text><text x="163" y="94">100 MHz</text><text x="296" y="94">200 MHz</text><text x="430" y="94">300 MHz</text></g>
  <rect x="30" y="48" width="42" height="20" rx="3" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.3"/>
  <text x="51" y="42" text-anchor="middle" font-size="7.5" fill="currentColor">HF</text>
  <rect x="110" y="48" width="266" height="20" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="243" y="42" text-anchor="middle" font-size="7.5" fill="currentColor">low VHF (60–260 MHz)</text>
  <text x="230" y="118" text-anchor="middle" font-size="9" fill="currentColor">Two coverage windows; a narrow, very clean capture in each</text>
</svg>
<figcaption>The Airspy HF+ covers HF (to ~31 MHz) and a low-VHF window (60–260 MHz), trading wide bandwidth for exceptional dynamic range.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+HF+Discovery&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The HF / low-VHF dynamic-range champion.** The Airspy HF+ Discovery covers HF (to
~31 MHz) and a low-VHF window (60–260 MHz) with a narrow (~768 kHz), exceptionally
clean capture — its polyphase harmonic-rejection mixer and high-resolution ADC survive
a crowded shortwave band where cheaper receivers fold under overload. **Pick it if you
want shortwave or ham HF too**, not primarily for scanning:
[GopherTrunk](/downloads.html)'s trunking targets ([P25](/reference/project-25/),
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/)) live in VHF/UHF, where an
[Airspy R2](/reference/airspy/) or an [RTL-SDR](/reference/rtl-sdr/) is the right radio.
**Receive-only, ~$170.** Like every receiver it can't decode
[AES encryption](/police-scanner-encryption/). See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
</div>

## How it works

HF is a hostile band for a receiver: hundreds of powerful shortwave, broadcast-AM and
amateur signals arrive at once, and the challenge is not sensitivity but **not being
overloaded** by the strongest of them. The HF+ is built around that problem rather than
around raw bandwidth:

- A **polyphase harmonic-rejection mixer** and a very selective front-end tuner suppress
  the images and harmonics that a simpler mixer would fold into the passband, so nearby
  strong signals do not create spurious responses.
- A high-resolution sigma-delta [ADC](/reference/analog-to-digital-converter/) and heavy
  oversampling deliver very high **[dynamic range](/reference/dynamic-range/)** — the
  headroom to hear a weak signal sitting next to a broadcaster tens of dB louder.
- The delivered capture is deliberately **narrow** (up to ~768 kHz for the HF+
  Discovery), which is all a shortwave, marine or amateur channel needs, and keeps the
  in-band energy — and thus the demand on the ADC — small.[^hfp]

The tradeoff is explicit: the HF+ will not give you the 10 MHz-wide slice an Airspy R2
does, because on HF that width would let in more strong signals than any front end could
handle cleanly. It is **receive-only**.

## Variants

- **Airspy HF+** — the original, covering HF and a low-VHF window (60–260 MHz).
- **Airspy HF+ Discovery** — a smaller, refined revision with an improved preselector
  and even better close-in dynamic range; the current mainstream HF+ model.

Both target the same job. Against the alternatives, a basic [RTL-SDR](/reference/rtl-sdr/)
needs an [upconverter](/reference/upconverter/) or a lossy direct-sampling hack to reach
HF at all and has far less dynamic range; an [SDRplay RSP1A](/reference/sdrplay-rsp1a/)
is a wideband 14-bit rival that covers HF continuously through GHz but is generally
beaten by the HF+ on pure close-in HF dynamic range; and the older
[FUNcube Dongle](/reference/funcube-dongle/) is a narrowband receiver aimed more at
satellite work.

## Relevance to SDR

The HF+ is the receiver of choice when **HF or the low VHF band is your target** —
shortwave broadcast, amateur SSB/CW, marine HF, [ionospheric](/reference/ionospheric-propagation/)
propagation study, or low-band utility monitoring — precisely because its dynamic range
survives the crowded HF environment where cheaper receivers fold under overload.

For GopherTrunk this is more of a complementary tool than a core one: the land-mobile
trunking systems GopherTrunk decodes (P25, DMR, NXDN, TETRA) live in VHF/UHF, where an
[Airspy R2](/reference/airspy/) or an RTL-SDR is the right radio. GopherTrunk can drive
the HF+ as a receiver, but its wideband-channelizer strengths don't apply to the HF+'s
narrow captures. Choose the HF+ when the low bands themselves are what you want to hear.

## Where to buy

The current mainstream model is the **Airspy HF+ Discovery**. Airspy sells through its
own distributor network, so Amazon stock is intermittent — the button below is a tagged
search that resolves to the live listings rather than a single page that may be out of
stock.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+HF+Discovery&tag=gophertrunk-20" rel="nofollow sponsored noopener">Airspy HF+ Discovery on Amazon &rarr;</a>

If your target is really VHF/UHF trunking rather than the low bands, an
[Airspy R2](/reference/airspy/) or an [RTL-SDR](/reference/rtl-sdr/) is the better buy —
see [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and, if you're comparing
against a handheld, [police scanner vs SDR](/police-scanner-vs-sdr/). Then get the
software from the [downloads page](/downloads.html).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on the Airspy HF+ and HF-optimised, high-dynamic-range SDR receivers.
[^hfp]: [Airspy HF+ Discovery](https://airspy.com/airspy-hf-discovery/) — Airspy, on the HF+ architecture, polyphase harmonic-rejection mixer, coverage and dynamic-range design.
