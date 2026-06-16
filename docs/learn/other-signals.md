---
slug: other-signals
title: Other signals you'll meet
description: Beyond trunked voice — the other signals an SDR and GopherTrunk can decode, including POCSAG and FLEX paging, AIS ship tracking, ADS-B aircraft tracking, APRS amateur position reports, and signalling like DSC and MDC1200.
keywords: POCSAG, FLEX, paging, AIS, ADS-B, APRS, AX.25, DSC, MDC1200, SDR decode, non-voice signals, data modes
level: beginner
status: full
prereq:
  - what-is-sdr
faq:
  - q: What else can an SDR decode besides voice?
    a: A lot. Common data signals include POCSAG and FLEX pagers, AIS ship-position reports, ADS-B aircraft transponders, APRS amateur position and telemetry, marine DSC calling, and signalling like MDC1200. Because the SDR just provides IQ samples, the same hardware can decode any of these given the right software — and GopherTrunk supports many of them.
  - q: What is ADS-B?
    a: ADS-B (Automatic Dependent Surveillance–Broadcast) is a system where aircraft continuously broadcast their identity, position, altitude, and velocity on 1090 MHz. An SDR can receive these and plot live aircraft on a map. GopherTrunk can decode ADS-B alongside its trunked-radio work.
  - q: What is AIS?
    a: AIS (Automatic Identification System) is the maritime equivalent of ADS-B — ships broadcast their identity, position, course, and speed on VHF marine frequencies (around 162 MHz). An SDR can receive and plot them, giving a live picture of nearby vessel traffic.
  - q: What is APRS?
    a: APRS (Automatic Packet Reporting System) is an amateur-radio data mode, usually on 144.39 MHz in North America, carrying position reports, weather, messages, and telemetry over AX.25 packet. An SDR can decode APRS to see station locations and data.
gophertrunk_links:
  - title: POCSAG paging
    url: /pocsag.html
    note: decode pager messages with GopherTrunk.
  - title: AIS
    url: /ais.html
    note: track ships on VHF marine.
  - title: ADS-B
    url: /adsb.html
    note: plot aircraft on 1090 MHz.
  - title: APRS / AX.25
    url: /aprs.html
    note: amateur position and packet data.
---

# Other signals you'll meet

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Trunked voice is only part of what's on the air. Because an [SDR](/learn/what-is-sdr/)
just delivers [IQ samples](/learn/iq-data/), the *same* hardware can decode many other
signals with the right software — and GopherTrunk supports a range of them: **POCSAG/FLEX
paging**, **AIS** ship tracking, **ADS-B** aircraft tracking, **APRS** amateur position
data, plus signalling like **DSC** and **MDC1200**. Most are **data, not voice**, and
many can be plotted on a map. They're a great way to see results even where voice systems
are quiet or encrypted.
</div>

You've followed the path toward trunked digital voice — but the spectrum is full of other
useful, decodable signals. This lesson is a quick tour, each with a pointer to
GopherTrunk's reference page, so you know what else your dongle can do.

## Paging (POCSAG / FLEX)

Pagers are far from dead — hospitals, fire services, and industry still use them.
**POCSAG** and the faster **FLEX** are [FSK](/learn/digital-modulation/) paging protocols
carrying short text and numeric messages. They're easy to decode and a satisfying first
data mode. GopherTrunk decodes [POCSAG](/pocsag.html); you'll see messages stream in as
they're transmitted.

## AIS — tracking ships

**AIS (Automatic Identification System)** is how ships broadcast their **identity,
position, course, and speed** on VHF marine frequencies (around 161.975 / 162.025 MHz).
Decode it and you get a **live map of vessel traffic** near any coast or waterway.
GopherTrunk's [AIS](/ais.html) support turns these bursts into plotted ships.

## ADS-B — tracking aircraft

**ADS-B (Automatic Dependent Surveillance–Broadcast)** is the aviation counterpart:
aircraft continuously transmit **position, altitude, velocity, and identity** on
**1090 MHz**. It's one of the most popular SDR applications — plot live aircraft overhead
in real time. See [ADS-B](/adsb.html) for GopherTrunk's decoder (note 1090 MHz is within
RTL-SDR range, though a tuned antenna helps).

## APRS and AX.25

**APRS (Automatic Packet Reporting System)** is an amateur-radio data network — usually
**144.39 MHz** in North America — carrying **position reports, weather, telemetry, and
messages** over **AX.25** packet. Decoding it shows you nearby ham stations, weather
sensors, and mobile trackers. GopherTrunk handles [APRS / AX.25](/aprs.html).

## DSC, MDC1200, and more

Smaller signalling systems are everywhere:

- **DSC (Digital Selective Calling)** — marine distress and calling on VHF/HF.
- **MDC1200** — a Motorola signalling format that sends a unit ID as a quick data burst
  at the start/end of an analog transmission (PTT IDs).

These add identity and control data to otherwise analog systems, and GopherTrunk can
surface them.

## Why this matters for your learning

Every one of these uses the **exact concepts** from this path — a
[carrier](/learn/signal-anatomy/) carrying [modulated](/learn/digital-modulation/) data,
captured as [IQ](/learn/iq-data/), pulled out by [filtering](/learn/filtering-decimation/),
and [demodulated and decoded](/learn/demodulation-pipeline/). Learning trunked voice gave
you a toolkit that *generalises*. And practically, these data modes are a great way to
**see immediate results** — a map filling with ships or planes — even when local voice
systems are quiet or [encrypted](/learn/encryption/).

| Signal | Roughly where | What you get |
|--------|---------------|--------------|
| POCSAG / FLEX | VHF/UHF paging allocations | Text/numeric pages |
| AIS | ~162 MHz | Ship positions |
| ADS-B | 1090 MHz | Aircraft positions |
| APRS | 144.39 MHz (NA) | Ham positions, weather |
| DSC | VHF/HF marine | Marine calling/distress |
| MDC1200 | On analog voice channels | Radio unit IDs |

<div class="knowledge-check" data-quiz data-correct-msg="Right — ADS-B aircraft transponders broadcast on 1090 MHz." markdown="0">
  <p class="knowledge-check__q">Quick check: on which frequency do aircraft broadcast ADS-B?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">144.39 MHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">162 MHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">1090 MHz</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- One SDR decodes far more than voice — the difference is **software**, not hardware.
- **POCSAG/FLEX** (paging), **AIS** (ships), **ADS-B** (aircraft), **APRS** (ham data),
  plus **DSC** and **MDC1200** signalling.
- Many are **mappable data modes** — quick, visible wins.
- They all reuse the same path concepts, so your skills **generalise**.

That completes Module 5. The final module puts everything to work — starting with finding
systems to monitor.
