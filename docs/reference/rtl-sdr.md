---
slug: rtl-sdr
title: RTL-SDR
entry_type: hardware
category: hardware
description: RTL-SDR is a family of low-cost USB software-defined radio receivers built on the RTL2832U chip — repurposed from DVB-T TV tuners — covering roughly 24 MHz to 1.7 GHz. A full history, the hardware explained, every major tuner and dongle variant compared, and where to buy the ones still in production.
keywords: RTL-SDR, RTL2832U, cheap SDR, DVB-T dongle, R820T2, R828D, R828S, E4000, RTL-SDR Blog V3, RTL-SDR Blog V4, V4 Lite, NooElec NESDR, FlightAware Pro Stick, 24 MHz 1.7 GHz, receive only, which RTL-SDR to buy
aka: [RTL-SDR, RTL SDR, RTL2832U dongle]
autolink: true
infobox:
  - { label: Type, value: USB SDR receiver }
  - { label: Bridge chip, value: RTL2832U }
  - { label: ADC, value: 8-bit }
  - { label: First SDR use, value: 2012 }
  - { label: Range, value: ~24 MHz – 1.7 GHz }
  - { label: Bandwidth, value: ~2.4 MHz usable (3.2 max) }
  - { label: TX, value: No (receive only) }
  - { label: Typical price, value: $15 – $40 }
see_also: [rtl2832u, r820t-tuner, hackrf, airspy, airspy-hf-plus, upconverter, bias-tee, zadig, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
  - { title: "What is software-defined radio?", url: /learn/rf-sdr/what-is-sdr/ }
  - { title: "Gain, AGC & avoiding overload", url: /learn/rf-sdr/gain-and-agc/ }
related_reading:
  - { title: "RF Front End, Part 7: RTL-SDR / RTL2832U bring-up", url: /blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/ }
  - { title: "RF Front End, Part 8: RTL-SDR R82xx & the Blog V4", url: /blog/deep-dives/rf-front-end-08-rtlsdr-r82xx-blog-v4/ }
external:
  - { title: "GopherTrunk hardware guide", url: /hardware.html }
  - { title: "rtl-sdr.com — the community reference", url: https://www.rtl-sdr.com/about-rtl-sdr/ }
  - { title: "Osmocom rtl-sdr project wiki", url: https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr }
cite_urls:
  - https://www.rtl-sdr.com/about-rtl-sdr/
  - https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr
  - https://www.rtl-sdr.com/rtl-sdr-blog-v4-dongle-initial-release/
  - https://www.rtl-sdr.com/rtl-sdr-blog-v4-end-of-line/
  - https://www.rtl-sdr.com/comparisons-r820t-r820t2-rtl-sdr-tuners/
  - https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR
---

**RTL-SDR** is a family of inexpensive USB [software-defined radio](/reference/software-defined-radio/)
receivers built around the [RTL2832U](/reference/rtl2832u/) chip — originally a DVB-T
digital-TV tuner that hobbyists discovered could stream raw [IQ](/reference/iq-data/)
samples straight to a computer.[^wiki] At roughly **$15–$40**, tuning about **24 MHz–1.7
GHz**, and capturing about **2.4 MHz** of [bandwidth](/reference/bandwidth/), it is the
radio that turned SDR from a lab curiosity into a mainstream hobby — and the baseline
GopherTrunk is built to run on.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for RTL-SDR (~24 MHz–1.7 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="113" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">RTL-SDR (~24 MHz–1.7 GHz) coverage</text>
</svg>
<figcaption>RTL-SDR covers most VHF/UHF scanning at low cost — receive only.</figcaption>
</figure>

## A short history

The RTL-SDR was never designed as an SDR. Millions of cheap USB dongles shipped as
**DVB-T** (European digital TV) and DAB/FM receivers built on Realtek's **RTL2832U**
demodulator. Around March 2010 **Eric Fry** noticed these chips exposed a raw sample
mode; in February 2012 **Antti Palosaari** confirmed the RTL2832U could be told to dump
unsigned raw I/Q samples over USB instead of decoding TV internally.[^about] The
[Osmocom](https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr) project — notably **Steve
Markgraf** — turned that discovery into the first open-source `librtlsdr` driver, and a
$20 TV dongle suddenly became a wideband radio receiver.[^osmo]

That accident is the whole story of the hobby's explosion. Software like SDR#, GQRX and
dump1090 appeared within months, the community site **rtl-sdr.com** became the reference,
and vendors such as **NooElec** and **RTL-SDR Blog** began shipping dongles purpose-built
for SDR — better clocks, shielding, connectors and HF reception — rather than repurposed
TV tuners. GopherTrunk stands on that lineage: the same `librtlsdr`-style raw-IQ interface
is what a modern trunking decoder still speaks to.

## What it actually is

An RTL-SDR is really **two chips on a stick**:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 96" role="img" aria-label="Block diagram: antenna into a tuner chip, into the RTL2832U demodulator and analog-to-digital converter, out over USB to a computer." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="34" y="52">antenna</text>
    <rect x="86" y="30" width="96" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="134" y="44">tuner</text><text x="134" y="56" font-size="7">R820T2 / R828D…</text>
    <rect x="214" y="30" width="118" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="273" y="44">RTL2832U</text><text x="273" y="56" font-size="7">8-bit ADC + USB</text>
    <text x="430" y="52">computer (IQ)</text>
  </g>
  <g stroke="currentColor" stroke-opacity="0.6">
    <line x1="60" y1="47" x2="84" y2="47"/><line x1="182" y1="47" x2="212" y2="47"/><line x1="332" y1="47" x2="372" y2="47"/>
  </g>
</svg>
<figcaption>A tuner chip mixes the target frequency down to baseband; the RTL2832U digitises it and ships raw IQ over USB.</figcaption>
</figure>

- A **tuner chip** (Rafael Micro [R820T/R820T2](/reference/r820t-tuner/), R828D, Elonics
  E4000, Fitipower FC001x…) mixes the frequency you want down toward baseband. The tuner
  is what sets the **frequency range**.
- The **[RTL2832U](/reference/rtl2832u/)** — the USB "bridge" — contains an **8-bit
  analog-to-digital converter** and the DSP that, in raw mode, streams unsigned
  [IQ](/reference/iq-data/) samples to the host. It caps the **sample rate** at ~3.2 MS/s
  (about **2.4 MS/s** is reliable), and its 8-bit ADC is what limits dynamic range.

Everything else — the local oscillator's accuracy, shielding, HF reception, connectors —
is a matter of *how well the board around those two chips is built*, which is exactly what
separates a $12 generic dongle from a $35 purpose-built one. RTL-SDRs are **receive-only**:
there is no transmit path. Below ~24 MHz the tuner can't reach, so **HF** needs either a
[direct-sampling](/reference/rtl2832u/) mode (with Nyquist folding around 14.4 MHz) or an
[upconverter](/reference/upconverter/) — the trick the RTL-SDR Blog V4 builds in.

## Tuner chips (the silicon variants)

The tuner defines a dongle's reach and, largely, its sensitivity. A handful of chips
account for almost every RTL-SDR ever sold:

| Tuner | Range | Notes | Status |
|-------|-------|-------|--------|
| Elonics **E4000** | ~54–2200 MHz (gap ~1100–1250) | Widest reach; higher noise; premium/used only | Discontinued |
| Rafael **R820T** | ~24–1766 MHz | The original SDR-era workhorse | Superseded by R820T2 |
| Rafael **R820T2** | ~24–1766 MHz | Best all-round sensitivity; the de-facto standard | Now sold as R860 |
| Rafael **R860** | ~24–1766 MHz | Current-production R820T2 equivalent | In production |
| Rafael **R828D** | ~24–1766 MHz | 3 inputs → enables Blog V4 upconverter + notch filters | End-of-line (stockpile) |
| Rafael **R828S** | ~24–1766 MHz | 2 inputs; used in the Blog V4 Lite | Limited (stockpile) |
| Fitipower **FC0013** | ~22–1100 MHz | Common on early/generic dongles | Legacy |
| Fitipower **FC0012** | ~22–948 MHz | Lowest reach; oldest generic sticks | Legacy |

The **R820T2 / R860** is the chip to want unless you specifically need the E4000's extra
top-end reach. It has better sensitivity than the E4000 across the frequencies most
scanners care about, a smaller DC spike, and is cheap — though the tuner needs a moment
to PLL-lock (and some airflow) above ~1.5 GHz.[^r820t]

## Dongle variants (the products)

### Generic / no-name DVB-T dongles

The cheapest option (~$12–15): an RTL2832U paired with an R820T2 (or an older FC0013).
They work, and they're fine to learn on. Their **limitations** are the give-away — no
temperature-compensated oscillator, so they drift tens of kHz as they warm up; weak
shielding and ESD protection; and front ends that overload easily near strong
transmitters. Quality is a lottery.

### NooElec NESDR family

Purpose-built SDRs with a **0.5 ppm TCXO**, aluminium enclosures and proper connectors:

- **NESDR SMArt v5** — R820T2/R860, 100 kHz–1.75 GHz (direct-sampling HF). The mainstream
  "buy this" NooElec; markedly better HF/VHF SNR and tuning stability than a generic stick.
- **NESDR Nano 3** — the same TCXO and tuner in a tiny body for embedded/portable use.
- **NESDR SMArTee v2** — adds an **always-on 4.5 V [bias tee](/reference/bias-tee/)** to
  power an inline LNA or active antenna without hardware mods.
- **NESDR SMArt XTR / SMArTee XTR** — E4000-based, trading some sensitivity for extended
  tuning up toward ~2.2 GHz.

### RTL-SDR Blog lineup

The reference commercial line, and what most guides mean by "a good RTL-SDR":

- **V1 / V2** — early boards; superseded. Used-market only.
- **V3** — R820T2, **1 ppm TCXO**, HF **direct sampling**, SMA connector, software-
  switchable bias tee, expansion pads and a metal case. For years *the* recommended
  dongle. Its HF has the classic Nyquist fold around 14.4 MHz.
- **V4** — R828D, keeping the 1 ppm TCXO and SMA but replacing direct sampling with a
  **built-in HF upconverter** (true, continuous ~500 kHz–1.766 GHz with no folding),
  plus **notch filtering** for broadcast AM/FM and DAB and lower phase noise and heat.[^v4]
  Trade-offs: ~2–3 dB less sensitivity on some bands from the extra filtering, and it needs
  the RTL-SDR Blog driver branch (now upstreamed to Osmocom). **As of 2026 the V4 is
  end-of-line** — the R828D is out of production and the stockpile is exhausted, so
  remaining stock is the last of it.[^eol]
- **V4 Lite (V4L)** — the 2026 stopgap: the V4 architecture (upconverter kept) on the
  **R828S**, which has only two inputs, so the V4's extra VHF/UHF filtering is dropped —
  a small **sensitivity gain** in exchange. Also a limited edition (R828S is likewise
  stockpiled) and requires an updated driver to detect it.[^eol] A **V5** is only in early
  scoping and is not expected before 2027.

### ADS-B-optimised: FlightAware Pro Stick / Pro Stick Plus

Single-purpose 1090 MHz receivers with a built-in ~19 dB LNA; the **Plus** adds a 1090 MHz
SAW bandpass [filter](/reference/saw-filter/). Excellent for aircraft tracking, but the
integrated amp is easy to overload with strong out-of-band signals, so they're a poor
choice as a general scanner.

## Capabilities & limitations at a glance

| Variant | Tuner | HF | TCXO | Bias tee | Key limitation | Still sold? |
|---------|-------|----|------|----------|----------------|-------------|
| Generic dongle | R820T2 / FC0013 | Fold (if any) | ✗ | ✗ | Drift, overload, no shielding | Yes (varies) |
| NESDR SMArt v5 | R820T2/R860 | Direct-sample | ✓ 0.5 ppm | ✗ | 8-bit dynamic range | Yes |
| NESDR SMArTee v2 | R820T2 | Direct-sample | ✓ 0.5 ppm | ✓ (always on) | 8-bit dynamic range | Yes |
| NESDR SMArt XTR | E4000 | ✗ | ✓ 0.5 ppm | ✗ | Higher noise than R820T2 | Yes |
| RTL-SDR Blog V3 | R820T2 | Direct-sample (fold ~14.4 MHz) | ✓ 1 ppm | ✓ (switchable) | HF folding; 8-bit | Limited |
| RTL-SDR Blog V4 | R828D | **Upconverter** (no fold) | ✓ 1 ppm | ✓ (switchable) | EOL; ~2–3 dB filter loss | End-of-line |
| RTL-SDR Blog V4L | R828S | Upconverter | ✓ 1 ppm | ✓ | Limited edition; needs driver update | Emerging |
| FlightAware Pro Stick+ | R820T2 | ✗ | ✗ | ✗ | 1090 MHz only; amp overloads | Yes |

Every RTL-SDR shares the same hard ceilings: **receive-only**, an **8-bit ADC** (modest
dynamic range — set [gain](/learn/rf-sdr/gain-and-agc/) carefully to avoid overload), and
~**2.4 MHz** of usable bandwidth. Those are properties of the RTL2832U, not of any one
board.

## Where to buy

For variants **still in production or stock**, the links below go to Amazon (a direct
product page first, then a search fallback that always resolves to current listings).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases.* Prices, stock and
listings change often — the search links track the live catalogue.

| Variant | Tuner | Amazon (direct · search) |
|---------|-------|--------------------------|
| **RTL-SDR Blog V4** (dongle) | R828D | <a href="https://www.amazon.com/dp/B0CD745394?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=RTL-SDR+Blog+V4+dongle&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **RTL-SDR Blog V4** (+ dipole kit) | R828D | <a href="https://www.amazon.com/dp/B0CD7558GT?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=RTL-SDR+Blog+V4+dipole+kit&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **RTL-SDR Blog V4 Lite** | R828S | <a href="https://www.amazon.com/s?k=RTL-SDR+Blog+V4L+Lite&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **NooElec NESDR SMArt v5** | R820T2/R860 | <a href="https://www.amazon.com/dp/B01HA642SW?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=NooElec+NESDR+SMArt+v5&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **NESDR SMArt v5** (3-antenna bundle) | R820T2/R860 | <a href="https://www.amazon.com/dp/B01GDN1T4S?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=NooElec+NESDR+SMArt+v5+bundle&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **NESDR SMArTee v2** (bias tee) | R820T2 | <a href="https://www.amazon.com/dp/B079C3FHPG?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=NooElec+NESDR+SMArTee+v2&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **NESDR Nano 3** (tiny) | R820T2 | <a href="https://www.amazon.com/dp/B073JZ8CC2?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=NooElec+NESDR+Nano+3&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **NESDR SMArt XTR** (extended range) | E4000 | <a href="https://www.amazon.com/dp/B06Y1HKLHY?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=NooElec+NESDR+SMArt+XTR+E4000&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| **FlightAware Pro Stick Plus** (ADS-B) | R820T2 | <a href="https://www.amazon.com/dp/B01M7REJJW?tag=GopherTrunk-20" rel="sponsored nofollow noopener">product</a> · <a href="https://www.amazon.com/s?k=FlightAware+Pro+Stick+Plus&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |
| Generic RTL2832U dongle | R820T2 | <a href="https://www.amazon.com/s?k=RTL-SDR+RTL2832U+R820T2+dongle&tag=GopherTrunk-20" rel="sponsored nofollow noopener">search</a> |

Older variants — the **RTL-SDR Blog V1/V2** and generic **E4000/FC001x** sticks — are
effectively discontinued and only turn up on the used market, so they aren't linked here.

## Relevance to GopherTrunk

The RTL-SDR is the ideal entry point and the baseline GopherTrunk targets — one dongle is
enough to follow most VHF/UHF trunked systems, and GopherTrunk can drive a **pool** of
them (locally or over [rtl_tcp](/reference/rtl-tcp/) / [SoapySDR](/reference/soapysdr/)) to
cover channels spread across a band. On Windows you'll bind the driver with
[Zadig](/reference/zadig/) first. For a first radio, an R820T2/R860-class unit (NESDR SMArt
v5, or a Blog V3/V4 while stock lasts) is the sweet spot. Need **HF**? Use a V4's built-in
upconverter, add an external [upconverter](/reference/upconverter/), or step up to an
[Airspy HF+](/reference/airspy-hf-plus/). See the [hardware guide](/hardware.html) for the
authoritative, GopherTrunk-specific list.

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR) — Wikipedia, on the DVB-T-dongle origins and capabilities of RTL-SDR.
[^about]: [About RTL-SDR](https://www.rtl-sdr.com/about-rtl-sdr/) — rtl-sdr.com, on Eric Fry, Antti Palosaari and the discovery of the raw-IQ mode.
[^osmo]: [rtl-sdr project wiki](https://osmocom.org/projects/rtl-sdr/wiki/rtl-sdr) — Osmocom, on the first open-source `librtlsdr` driver (Steve Markgraf).
[^r820t]: [Comparisons between the R820T and R820T2 tuners](https://www.rtl-sdr.com/comparisons-r820t-r820t2-rtl-sdr-tuners/) — rtl-sdr.com, on tuner sensitivity and behaviour.
[^v4]: [RTL-SDR Blog V4 Dongle Initial Release](https://www.rtl-sdr.com/rtl-sdr-blog-v4-dongle-initial-release/) — rtl-sdr.com, on the R828D, built-in upconverter and notch filtering.
[^eol]: [RTL-SDR Blog V4 End Of Line](https://www.rtl-sdr.com/rtl-sdr-blog-v4-end-of-line/) — rtl-sdr.com, on the V4's discontinuation, the V4 Lite (R828S) and V5 scoping.
