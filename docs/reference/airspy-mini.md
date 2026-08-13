---
slug: airspy-mini
title: Airspy Mini
entry_type: hardware
category: sdr-devices
description: "The Airspy Mini is a 12-bit VHF/UHF software-defined radio in a dongle form factor — most of the Airspy R2's quality up to ~6 MS/s, cheaper and more portable, driven natively by GopherTrunk."
keywords: Airspy Mini, Airspy Mini SDR, 12-bit SDR dongle, VHF UHF receiver, 6 MSPS, R820T2, portable SDR, high performance dongle
aka: [Airspy Mini]
autolink: true
affiliate: true
product:
  name: "Airspy Mini"
  brand: Airspy
  category: Software-defined radio
  lowPrice: "99"
  highPrice: "130"
  url: https://www.amazon.com/s?k=Airspy+Mini+SDR&tag=gophertrunk-20
infobox:
  - { label: Type, value: VHF/UHF SDR dongle }
  - { label: Vendor/Chip, value: "Airspy; R820T2 tuner + LPC4370" }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: ~24 MHz – 1.7 GHz }
  - { label: Bandwidth, value: up to ~6 MS/s }
  - { label: TX, value: No (receive only) }
  - { label: Price, value: around $99–130 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Airspy+Mini+SDR&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [airspy, airspy-r2, airspy-hf-plus-discovery, airspy-rate-selection, rtl-sdr, sdrplay-rsp1b, software-defined-radio, dynamic-range]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://airspy.com/airspy-mini/
faq:
  - q: "Airspy Mini or Airspy R2?"
    a: "Same 12-bit architecture and the same R820T2 front end. The Mini tops out around ~6 MS/s in a compact, cheaper dongle; the R2 captures up to ~10 MS/s and adds a clock output and a 4.5 V bias tee. Choose the Mini for most of the quality at a lower price and better portability; the R2 for the widest capture and inline LNA power."
  - q: "Is the Airspy Mini worth it over an RTL-SDR for GopherTrunk?"
    a: "For tough or busy RF, yes. The Mini's 12-bit ADC carries roughly 72 dB of dynamic range against an RTL-SDR's ~48 dB, and its ~6 MS/s capture is wider than an RTL-SDR's usable ~2.4 MHz — enough to channelize a couple of control channels at once. On a clean single-site system a $30 RTL-SDR is still the cheapest thing that works."
  - q: "Does GopherTrunk drive the Airspy Mini natively?"
    a: "Yes — GopherTrunk talks to the Mini directly over USB with a pure-Go backend, no libairspy and no SoapySDR. It behaves like the R2, just with a narrower maximum capture."
  - q: "Can the Airspy Mini decode encrypted channels?"
    a: "No. It's a receiver, and GopherTrunk is receive-only. It decodes clear P25/DMR/NXDN/TETRA but no radio or scanner can decode AES-encrypted transmissions."
---

**The Airspy Mini** is a high-performance VHF/UHF
[software-defined radio](/reference/software-defined-radio/) in a dongle form factor: the
same **12-bit** [ADC](/reference/analog-to-digital-converter/) and
[R820T2](/reference/r820t-tuner/) front end as the [Airspy R2](/reference/airspy-r2/),
capturing up to about **6 MS/s** in a smaller, cheaper package.[^wiki] It gives you most of
the R2's [dynamic range](/reference/dynamic-range/) and [bandwidth](/reference/bandwidth/)
advantage over an [RTL-SDR](/reference/rtl-sdr/), and GopherTrunk drives it natively over USB.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the Airspy Mini (~24 MHz–1.7 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="112" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">Airspy Mini (~24 MHz–1.7 GHz) coverage</text>
</svg>
<figcaption>The Mini covers VHF/UHF with a 12-bit converter and up to ~6 MS/s of capture.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+Mini+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Most of the [R2](/reference/airspy-r2/)'s quality, cheaper and smaller.** The Airspy Mini's
**12-bit** ADC (~72 dB [dynamic range](/reference/dynamic-range/) vs an RTL-SDR's ~48 dB) and
**~6 MS/s** capture decode weak and busy
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) channels an
8-bit dongle stumbles on, and channelize a couple of
[control channels](/reference/control-channel/) at once. **No clock output or bias tee** — for
those and the full ~10 MS/s, step up to the [R2](/reference/airspy-r2/).
**Receive-only, ~$99–130.** Airspy is distributor-sold, so Amazon stock is intermittent — the
button tracks live listings. Like every receiver it can't decode
[AES encryption](/police-scanner-encryption/). New here? See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
</div>

## Overview

The Mini is the portable, lower-cost member of the [Airspy](/reference/airspy/) family. It is
built on the same architecture as the [R2](/reference/airspy-r2/), so its per-sample quality is
identical; what it trades is capture width (~6 MS/s vs ~10 MS/s) and the R2's clock output and
4.5 V [bias tee](/reference/bias-tee/). That makes it the pick when you want the Airspy front
end — 12 bits, oversampling process gain, clean [dynamic range](/reference/dynamic-range/) — in
a small dongle for a single system or a modest two-channel channelize, without paying for
wideband multi-site capture you won't use. For the lower bands, the
[Airspy HF+ Discovery](/reference/airspy-hf-plus-discovery/) is the specialised choice.

## How it works

Like the R2, the Mini shares the RTL-SDR's Rafael Micro [R820T2](/reference/r820t-tuner/) tuner
but replaces the back end: instead of the [RTL2832U](/reference/rtl2832u/)'s 8-bit ADC, it
digitises the IF with a **12-bit** [ADC](/reference/analog-to-digital-converter/) driven by an
NXP LPC4370, oversampling and converting real-to-complex on the way out.[^airspy] Two effects
follow:

- **More bits** — roughly **72 dB** of theoretical [dynamic range](/reference/dynamic-range/)
  against the RTL2832U's ~48 dB, the headroom that keeps a weak signal alive next to a strong
  one.
- **Process gain** — averaging many high-rate samples into each output sample adds effective
  bits, so the delivered stream is quieter than the raw ADC alone.

The Mini is **receive-only** and tops out around ~6 MS/s. It is otherwise the R2's equal; an
[SDRplay RSP1B](/reference/sdrplay-rsp1b/) is the closest 14-bit alternative in the same price
class.

## Relevance to GopherTrunk

GopherTrunk drives the Mini natively over USB with its pure-Go backend — no `libairspy`, no
SoapySDR — exactly as it drives the R2. The Mini suits a GopherTrunk node that wants better
sensitivity and [dynamic range](/reference/dynamic-range/) than an
[RTL-SDR](/reference/rtl-sdr/) but doesn't need the R2's full wideband multi-site capture: a
single busy [P25](/reference/project-25/)/[DMR](/reference/dmr/) system, a congested band, or a
couple of nearby [control channels](/reference/control-channel/) out of one capture. It remains
a receiver only, so it decodes clear and scrambled traffic, never keyed encryption.

## Choosing a sample rate

Like the R2, the Mini is a **real-sampling** device: the firmware streams bare ADC samples
and the host converts them to complex baseband
([#454](https://github.com/MattCheramie/GopherTrunk/issues/454)), and its firmware rate
table is expressed in **IQ output rates**, not raw ADC rates — the distinction behind an
R2 half-rate driver regression ([#851](https://github.com/MattCheramie/GopherTrunk/issues/851)).
On the R2 the lower native rate proved measurably *cleaner* than the higher one, because
it is an FPGA decimation of the same ADC stream
([#764](https://github.com/MattCheramie/GopherTrunk/issues/764),
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771)) — so on any Airspy,
prefer the lowest native rate that covers what you need. Details and the diagnostic recipe
are in [Airspy sample-rate selection](/reference/airspy-rate-selection/).

## Where to buy

Airspy is sold through its own distributor network, so Amazon stock comes and goes — the button
is a tagged search that always resolves to current listings. Get the **Mini** for most of the
Airspy quality at a lower price and better portability; step up to the
[R2](/reference/airspy-r2/) if you need the full ~10 MS/s wideband capture, clock output, or
bias tee.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+Mini+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

Deciding between radios? See [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/), compare
against the [RTL-SDR](/reference/rtl-sdr/), or, for shortwave/low-VHF, the
[Airspy HF+ Discovery](/reference/airspy-hf-plus-discovery/). Then grab GopherTrunk from the
[downloads page](/downloads.html).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
[^airspy]: [Airspy Mini](https://airspy.com/airspy-mini/) — Airspy, on the Mini's R820T2 front end, 12-bit oversampling architecture, and up-to-6 MS/s capture.
