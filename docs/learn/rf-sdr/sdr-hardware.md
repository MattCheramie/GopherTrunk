---
slug: sdr-hardware
redirect_from: /learn/sdr-hardware/
title: SDR hardware — RTL-SDR, HackRF, Airspy
description: A practical SDR hardware buying guide — RTL-SDR, HackRF, Airspy R2/Mini and HF+ compared on frequency range, bandwidth, sensitivity, and price, plus remote backends, and how each maps to GopherTrunk's supported hardware.
keywords: SDR hardware, RTL-SDR, HackRF One, Airspy, Airspy HF+, best SDR for scanning, SDR comparison, rtl_tcp, SoapyRemote, which SDR to buy
level: beginner
status: full
prereq:
  - what-is-sdr
faq:
  - q: What is the best SDR for beginners?
    a: An RTL-SDR (a modern RTL2832U dongle, ideally a quality "v3/v4"-class unit) is the best starting point. It costs around $30, covers roughly 24 MHz to 1.7 GHz, captures about 2.4 MHz of bandwidth, and is enough to follow most VHF/UHF trunked systems with GopherTrunk. You can always step up later.
  - q: What's the difference between RTL-SDR, HackRF, and Airspy?
    a: RTL-SDR is the cheap, receive-only entry point with modest bandwidth. Airspy radios offer better sensitivity and wider bandwidth for more demanding reception, and the Airspy HF+ specialises in the lower (HF) bands. HackRF One is a wideband transceiver that can also transmit and covers a very large frequency range, but isn't needed for scanning. For trunk-tracking, RTL-SDR or Airspy are the usual picks.
  - q: How much SDR bandwidth do I need for trunked radio?
    a: Enough to cover the channels you follow. If a system's control and voice channels fit within about 2 MHz, an RTL-SDR is sufficient. If they're spread wider, an Airspy's larger bandwidth — or a second dongle covering another chunk — helps. GopherTrunk can drive a pool of radios for exactly this reason.
  - q: Can the SDR be on a different computer from GopherTrunk?
    a: Yes. Remote backends like rtl_tcp and SoapyRemote let the radio live on one machine (say, a Raspberry Pi at the antenna) and stream IQ to GopherTrunk on another. This keeps the dongle close to the antenna with a short coax run while you run the decoder elsewhere.
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: the authoritative list of GopherTrunk-supported radios and notes.
  - title: Downloads
    url: /downloads.html
    note: get GopherTrunk once you've picked a radio.
  - title: Get started
    url: /getting-started-setup.html
    note: go from dongle to first decoded call.
---

# SDR hardware — RTL-SDR, HackRF, Airspy

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
For trunk-tracking, an **RTL-SDR** (~$30, ~24 MHz–1.7 GHz, ~2.4 MHz bandwidth,
receive-only) is the best place to start and enough for most systems. **Airspy** adds
sensitivity and [bandwidth](/learn/rf-sdr/sample-rate-nyquist/); the **Airspy HF+** targets the
lower bands. **HackRF One** is a wideband transceiver (can transmit) but overkill for
scanning. The radio mostly sets your **frequency range** and **how much spectrum you
capture at once** — and GopherTrunk can drive a **pool** of radios, even
[remote](/learn/rf-sdr/what-is-sdr/) ones, at the same time.
</div>

[What is SDR?](/learn/rf-sdr/what-is-sdr/) made the case that the software does the clever part
and the hardware is almost interchangeable. "Almost" — there are real differences worth
knowing before you buy. This lesson keeps it practical.

## What actually matters

When comparing SDRs for trunk-tracking, three specs dominate:

- **Frequency range** — can it tune the [band](/learn/rf-sdr/frequency-and-spectrum/) your
  targets live in? (VHF/UHF/700-800 MHz for most trunked systems.)
- **Bandwidth** — how much spectrum it captures at once
  ([sample rate](/learn/rf-sdr/sample-rate-nyquist/)); more lets you follow channels spread
  further apart from one radio.
- **Sensitivity & dynamic range** — how well it hears weak signals without overloading
  on strong ones (tied to [gain](/learn/rf-sdr/gain-and-agc/) headroom).

Price and "receive vs. transmit" round it out. For *receiving* trunked voice, you never
need transmit.

## RTL-SDR — the cheap entry point

The **RTL-SDR** (an RTL2832U-based dongle, originally a TV tuner) is the reason this
hobby exploded. A good modern unit costs around **$30**, tunes roughly **24 MHz–1.7
GHz**, and captures about **2.4 MHz**. It's **receive-only** with modest dynamic range —
but it's more than enough to learn on and to follow most VHF/UHF trunked systems in
GopherTrunk. Buy a reputable one (quality varies); cheap no-name clones can drift and
overload.

## Airspy R2 / Mini and HF+

**Airspy R2** and the smaller **Airspy Mini** offer better front ends, higher
sensitivity, and **wider bandwidth** (R2 up to ~10 MHz) than an RTL-SDR — useful when a
system's channels are spread across a band, or in tough RF environments. The **Airspy
HF+** is a different beast, optimised for the **lower bands (HF and low VHF)** with
excellent dynamic range — the one to reach for if shortwave or low-band is your goal
(an RTL-SDR can't reach HF without a direct-sampling mode or upconverter).

## HackRF One

**HackRF One** is a wideband **transceiver**: a huge range (~1 MHz–6 GHz) and the
ability to *transmit*. That breadth is great for experimentation, but for *receiving*
trunked voice it's overkill, has only 8-bit sampling (less dynamic range than Airspy),
and transmit is irrelevant to scanning. Worth it if you have broader SDR ambitions;
not the obvious trunk-tracking pick.

## Comparison at a glance

| Radio | Range | Bandwidth | TX? | Best for |
|-------|-------|-----------|-----|----------|
| RTL-SDR | ~24 MHz–1.7 GHz | ~2.4 MHz | No | Starting out, most trunked systems |
| Airspy Mini/R2 | ~24 MHz–1.8 GHz | up to ~6–10 MHz | No | Wider spans, weak signals |
| Airspy HF+ | HF–low VHF | ~0.66 MHz | No | HF / low-band reception |
| HackRF One | ~1 MHz–6 GHz | up to ~20 MHz | Yes | Wideband experimentation |

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 170" role="img" aria-label="Frequency coverage bars for four SDRs against a log frequency axis from 1 MHz to 6 GHz. A shaded band marks the VHF/UHF/700-800 MHz trunked-radio range." xmlns="http://www.w3.org/2000/svg">
  <rect x="206" y="20" width="150" height="120" fill="currentColor" fill-opacity="0.08"/>
  <text x="281" y="16" text-anchor="middle" font-size="9" fill="currentColor">trunked-radio range</text>
  <g font-size="9" fill="currentColor">
    <text x="10" y="44">RTL-SDR</text>
    <rect x="120" y="36" width="250" height="12" rx="3" fill="currentColor" fill-opacity="0.4"/>
    <text x="10" y="74">Airspy R2</text>
    <rect x="120" y="66" width="255" height="12" rx="3" fill="currentColor" fill-opacity="0.4"/>
    <text x="10" y="104">Airspy HF+</text>
    <rect x="60" y="96" width="80" height="12" rx="3" fill="currentColor" fill-opacity="0.4"/>
    <text x="10" y="134">HackRF</text>
    <rect x="95" y="126" width="420" height="12" rx="3" fill="currentColor" fill-opacity="0.4"/>
  </g>
  <line x1="40" y1="150" x2="520" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="60" y="162">1 MHz</text><text x="206" y="162">30 MHz</text><text x="356" y="162">1 GHz</text><text x="515" y="162">6 GHz</text>
  </g>
</svg>
<figcaption>Coverage at a glance (frequency axis is roughly logarithmic). Most radios comfortably cover the trunked-radio range; the HF+ trades reach for low-band excellence, and HackRF spans the most.</figcaption>
</figure>

## Remote backends

The radio doesn't have to sit next to the computer running GopherTrunk. **rtl_tcp** and
**SoapyRemote** stream [IQ](/learn/rf-sdr/iq-data/) over the network, so you can put a dongle on
a **Raspberry Pi right at the antenna** (short coax = less [loss](/learn/rf-sdr/antennas/)) and
decode on a beefier machine elsewhere. GopherTrunk treats these like any other radio.

## Matching hardware to your targets

1. **Identify your systems' bands** (see [finding systems](/learn/rf-sdr/finding-systems/)).
2. **Check coverage** — an RTL-SDR covers nearly all VHF/UHF trunked work.
3. **Check span** — if your channels fit in ~2 MHz, RTL-SDR is fine; wider may want
   Airspy or a second dongle.
4. **Going to HF?** Choose Airspy HF+.
5. **Start cheap.** An RTL-SDR teaches you everything; upgrade only when you hit a real
   limit.

The [Hardware guide](/hardware.html) is the authoritative, GopherTrunk-specific list of
supported radios and gotchas.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an RTL-SDR covers VHF/UHF and is the ideal starting radio." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to follow a local 800 MHz P25 system on a budget. Best first radio?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">An RTL-SDR</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An Airspy HF+ (HF-focused)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing under $500 will work</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The radio mainly sets **frequency range** and **capture bandwidth**.
- **RTL-SDR** is the ideal, cheap starting point for VHF/UHF trunk-tracking.
- **Airspy** adds sensitivity/bandwidth; **HF+** for the low bands; **HackRF** for wideband/TX.
- **Remote backends** let the dongle live at the antenna.
- Match the radio to your targets' band and span — and start cheap.

That completes Module 3. Next module digs into the DSP that turns IQ into bits —
starting with the FFT and the waterfall.
