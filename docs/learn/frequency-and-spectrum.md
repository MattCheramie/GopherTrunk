---
slug: frequency-and-spectrum
title: Frequency, bands & the spectrum
description: How the radio spectrum is measured and divided — Hz to GHz, the VLF through SHF bands, HF/VHF/UHF, and how regulators carve spectrum into band plans for broadcast, public safety, aviation, marine, and amateur radio.
keywords: radio spectrum, frequency bands, HF VHF UHF, band plan, spectrum allocation, kHz MHz GHz, public safety frequencies, where to scan
level: beginner
status: full
prereq:
  - radio-waves
faq:
  - q: What are the main radio frequency bands?
    a: The radio spectrum is split into bands by frequency. The common ones are VLF (3–30 kHz), LF (30–300 kHz), MF (300 kHz–3 MHz), HF (3–30 MHz), VHF (30–300 MHz), UHF (300 MHz–3 GHz), and SHF (3–30 GHz). Each has different propagation behaviour and is allocated to different uses. Most scanner and trunked-radio activity lives in VHF and UHF.
  - q: What frequencies do police and fire use?
    a: It varies by country and region, but public-safety radio commonly sits in VHF (around 150–174 MHz), UHF (around 450–470 MHz), and the 700/800 MHz bands, increasingly as digital trunked systems. There is no single answer — you look up the specific system in a database like RadioReference for your area.
  - q: What is a band plan?
    a: A band plan is the agreed division of a frequency band into channels and uses. Regulators (like the FCC in the US or Ofcom in the UK) and standards bodies publish band plans so broadcasters, public safety, aviation, marine, amateur, and other users do not interfere with each other. It tells you what kind of signal to expect at a given frequency.
  - q: Why does the band matter for what I can receive?
    a: Different bands propagate differently and need different antennas. HF can bounce off the ionosphere and travel worldwide; VHF and UHF are mostly line-of-sight. The band also tells you what is there — aircraft on VHF airband, marine on VHF marine, trunked public safety on UHF and 700/800 MHz — so knowing the band plan tells you where to point your SDR.
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: check your dongle covers the band your target systems live in.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: sweep a band to find active control channels.
---

# Frequency, bands & the spectrum

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The radio spectrum runs from a few kilohertz to hundreds of gigahertz, divided into
**bands** — VLF, LF, MF, **HF**, **VHF**, **UHF**, SHF — each with its own
propagation behaviour and uses. Regulators publish **band plans** that say what
lives where: broadcast, aviation, marine, public safety, amateur. Knowing the band
plan tells you *where to point your SDR* and *what to expect*. Most trunked-radio
scanning happens in **VHF, UHF, and the 700/800 MHz bands**.
</div>

In [lesson 1](/learn/radio-waves/) you learned that frequency is how fast a wave
cycles. This lesson zooms out: how the whole spectrum is measured, named, and carved
up — so you know which slice of it to go looking in.

## How is frequency measured?

Frequency is counted in **hertz (Hz)** — cycles per second — but radio frequencies
are large, so we lean on prefixes:

| Prefix | Hertz | Example |
|--------|-------|---------|
| kHz (kilo) | 1,000 | AM broadcast, navigation beacons |
| MHz (mega) | 1,000,000 | FM radio, airband, public safety |
| GHz (giga) | 1,000,000,000 | Wi-Fi, GPS, satellite, radar |

When you set GopherTrunk to **460.025 MHz**, you're naming a precise point on this
ruler. The spectrum is continuous, but in practice it's organised into the named
bands below.

## What are the radio bands?

Each band spans a decade of frequency (a ten-fold range) and behaves differently.
The names are an international convention:

| Band | Range | Wavelength | Typical uses |
|------|-------|------------|--------------|
| VLF | 3–30 kHz | 100–10 km | Submarine comms, time signals |
| LF | 30–300 kHz | 10–1 km | Longwave broadcast, navigation |
| MF | 300 kHz–3 MHz | 1 km–100 m | AM broadcast, marine |
| **HF** | 3–30 MHz | 100–10 m | Shortwave, amateur, long-distance |
| **VHF** | 30–300 MHz | 10–1 m | FM radio, airband, marine, public safety |
| **UHF** | 300 MHz–3 GHz | 1 m–10 cm | TV, cellular, public-safety trunking, GPS |
| SHF | 3–30 GHz | 10–1 cm | Wi-Fi, satellite, radar |

Notice the wavelength column — it's just *λ ≈ 300 ÷ MHz* from the last lesson. It
shrinks as you climb, which is why a UHF antenna is a handspan while an HF antenna
can be metres long.

## Why does the band change how a signal behaves?

The band sets the *propagation* — how far and by what path a signal travels (the
[next lesson](/learn/propagation/) goes deeper):

- **HF** waves can refract off the ionosphere and skip thousands of kilometres —
  that's how shortwave reaches across oceans.
- **VHF and UHF** are essentially **line-of-sight**: they travel to the radio
  horizon and don't bend far around the Earth. Local public-safety, airband, and
  marine all live here because they're meant to cover a region, not the globe.
- **SHF** is even more line-of-sight and easily blocked, but carries huge bandwidth.

For GopherTrunk the practical upshot: the trunked systems you'll follow are regional,
so they live in the line-of-sight VHF/UHF/700/800 MHz range.

## What is a band plan, and why should I care?

Within each band, regulators and standards bodies divide the spectrum into
**channels and allocations** — a **band plan**. It's the master map that keeps
broadcasters, aircraft, ships, police, and hams from stepping on each other.

A rough tour of where common things live (exact allocations vary by country):

| Roughly | What you'll find |
|---------|------------------|
| 88–108 MHz | FM broadcast radio |
| 108–137 MHz | Aircraft (VHF airband, AM) |
| 144–148 MHz | Amateur 2-metre (APRS, repeaters) |
| 156–162 MHz | Marine VHF (and AIS at 161.975 / 162.025) |
| 150–174 MHz | VHF public safety / business |
| 420–450 MHz | Amateur 70-cm |
| 450–470 MHz | UHF public safety / business |
| 700 / 800 MHz | Digital trunked public safety (P25, etc.) |
| 1090 MHz | Aircraft ADS-B transponders |

You don't memorise this — you look up specific systems in a database. But knowing the
*shape* of the band plan means that when you see a signal on the
[waterfall](/learn/fft-and-waterfall/), you already have a guess about what it is.

## How does this map to my SDR?

Every SDR has a **frequency range** it can tune. A basic RTL-SDR covers roughly
24 MHz–1.7 GHz — enough for nearly all VHF/UHF trunked scanning, airband, marine, and
ADS-B, but *not* the HF shortwave bands (you'd want an Airspy HF+ or an upconverter
for those). Matching your hardware's range to the band your targets live in is the
first hardware decision — see the [SDR hardware lesson](/learn/sdr-hardware/) and the
[Hardware guide](/hardware.html).

<div class="knowledge-check" data-quiz data-correct-msg="Right — VHF/UHF are line-of-sight, ideal for regional systems." markdown="0">
  <p class="knowledge-check__q">Quick check: a regional police trunked system is most likely found in which bands?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">VLF and LF</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">VHF, UHF, and 700/800 MHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">SHF only</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Frequency is measured in **Hz**, scaled by kHz / MHz / GHz.
- The spectrum is divided into **bands** (VLF…SHF); **VHF and UHF** carry most
  scanner traffic.
- The band sets **propagation** — HF can skip worldwide; VHF/UHF are line-of-sight.
- A **band plan** tells you what lives where, so you know where to look.
- Match your SDR's tuning range to the band your targets are in.

Next: why two antennas at the same frequency can perform completely differently.
