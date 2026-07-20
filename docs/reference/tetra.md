---
slug: tetra
title: TETRA
entry_type: protocol
category: land-mobile-trunking
description: TETRA (Terrestrial Trunked Radio) is an ETSI digital trunked-radio standard using four-slot TDMA and π/4-DQPSK, widely used by public safety and transport outside North America.
keywords: TETRA, Terrestrial Trunked Radio, ETSI, four-slot TDMA, pi/4-DQPSK, ACELP, public safety Europe, TETRA 2, TEDS, direct mode
aka: [TETRA, Terrestrial Trunked Radio, MCCH]
autolink: true
infobox:
  - { label: Type, value: Digital trunked radio }
  - { label: Standards body, value: ETSI }
  - { label: Introduced, value: "1995" }
  - { label: Access, value: TDMA (4 slots) }
  - { label: Channel spacing, value: 25 kHz }
  - { label: Modulation, value: π/4-DQPSK }
  - { label: Vocoder, value: ACELP }
  - { label: GopherTrunk support, value: See Status }
see_also: [tetrapol, pi-4-dqpsk, tdma, acelp, trunked-radio, phase-shift-keying, control-channel, etsi]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://www.etsi.org/technologies/tetra
---

**TETRA** (**Terrestrial Trunked Radio**) is an [ETSI](/reference/etsi/) digital
[trunked-radio](/reference/trunked-radio/) standard built for public-safety and
professional users, especially across Europe and much of the world outside North America.
It divides each **25 kHz** carrier into **four [TDMA](/reference/tdma/) timeslots** and
modulates them with **[π/4-DQPSK](/reference/pi-4-dqpsk/)**
([phase-shift keying](/reference/phase-shift-keying/)), encoding speech with an
[ACELP](/reference/acelp/) [vocoder](/reference/vocoder/).[^wiki] It is a complete system
standard rather than merely an air interface, defining call control, gateways, security,
and interworking.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="Four TDMA slots in a single 25 kHz TETRA carrier, repeating over time." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="360" y2="105" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#ta_tetra)"/>
  <text x="195" y="128" text-anchor="middle" font-size="9" fill="currentColor">time → · one 25 kHz carrier, 4 slots</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="40" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="80" y="40" width="40" height="50" fill="none"/><rect x="120" y="40" width="40" height="50" fill="currentColor" fill-opacity="0.12"/><rect x="160" y="40" width="40" height="50" fill="none" stroke-dasharray="3 2"/><rect x="200" y="40" width="40" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="240" y="40" width="40" height="50" fill="none"/><rect x="280" y="40" width="40" height="50" fill="currentColor" fill-opacity="0.12"/><rect x="320" y="40" width="40" height="50" fill="none" stroke-dasharray="3 2"/></g><g font-size="9" fill="currentColor" text-anchor="middle"><text x="60" y="69">1</text><text x="100" y="69">2</text><text x="140" y="69">3</text><text x="180" y="69">4</text><text x="260" y="69">1</text><text x="300" y="69">2</text></g>
  <defs><marker id="ta_tetra" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>TETRA uses four-slot TDMA on a 25 kHz carrier — four independent talk paths per RF channel — for high capacity.</figcaption>
</figure>

## Overview

TETRA is a **complete system standard**, not just a modulation scheme. It specifies group
and individual calls, fast call setup (often quoted under 300 ms), direct mode (DMO) for
radio-to-radio operation off the network, packet data, and a rich security suite with
authentication, air-interface encryption, and end-to-end encryption options. Splitting a
25 kHz carrier into four timeslots gives four simultaneous talk paths per channel — the
same spectral-efficiency goal as narrowband FDMA systems, but reached through time
division. One slot on a carrier typically acts as the **main control channel (MCCH)**,
continuously broadcasting system information and handling registration, affiliation, and
[channel grants](/reference/channel-grant/), while the others carry traffic. The choice of
[π/4-DQPSK](/reference/pi-4-dqpsk/) — a differential QPSK whose phase transitions avoid
the origin — keeps the signal envelope well-behaved for efficient amplification while
carrying 2 bits per symbol at 18 kbit/s gross.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [TDMA](/reference/tdma/), 4 slots per carrier |
| Channel spacing | 25 kHz |
| Modulation | [π/4-DQPSK](/reference/pi-4-dqpsk/), 18 kbit/s gross |
| Symbol rate | 18 000 sym/s (36 kbit/s raw before differential mapping) |
| Vocoder | [ACELP](/reference/acelp/) (~4.567 kbit/s speech) |
| Control | Continuous main control channel (MCCH) on one slot |
| Modes | Trunked (V+D), Direct Mode (DMO), packet data |
| Data evolution | TEDS (TETRA Enhanced Data Service) in "TETRA 2" |

## History

TETRA was **standardised by ETSI from the mid-1990s** (first specifications published
around 1995) as a pan-European digital replacement for the fragmented analogue and
proprietary trunked systems then serving emergency services.[^wiki][^etsi] It was designed
deliberately as an open, interoperable standard so agencies from different countries and
vendors could share infrastructure and roam across networks. A later evolution, sometimes
called **TETRA 2**, added the higher-throughput **TEDS** data service and additional
modulation options while preserving backward compatibility with the original voice-and-data
air interface. TETRA competes in its home market with the French **[Tetrapol](/reference/tetrapol/)**
system, a different, FDMA-based public-safety standard.

## Deployment

TETRA underpins **national public-safety networks** across Europe, the Middle East, Asia,
Africa, and Latin America, along with transport (airports, metros, railways), utilities,
and military/government users. It is essentially absent from North America, where
[P25](/reference/project-25/) dominates public safety for regulatory and historical
reasons. Because it is a mature standard with a large installed base, many countries are
now weighing long-term migration of mission-critical voice toward broadband
[LTE](/reference/lte/)/5G push-to-talk, though TETRA remains the workhorse for guaranteed,
infrastructure-independent group communications.

## Decoding it with GopherTrunk

TETRA uses a distinct physical layer — [π/4-DQPSK](/reference/pi-4-dqpsk/) at 18 kbit/s on
a 25 kHz carrier with four TDMA slots — that differs from the C4FM/4FSK family GopherTrunk
handles for P25/DMR/NXDN, and its speech uses the [ACELP](/reference/acelp/) vocoder rather
than an AMBE variant. Following a TETRA system means demodulating DQPSK, recovering the
slot/frame timing, and parsing the MCCH signalling to track calls onto traffic slots. Note
that GopherTrunk's downconversion path normalises TETRA to a **144 kHz** channel rate
(versus 48 kHz for the 4800-baud C4FM family), reflecting its wider symbol rate. Air-interface
and end-to-end encryption ([TETRA TEA](/reference/tetra-tea/)) are out of scope for a
receiver-only decoder. Consult the [Status](/status.html) page for GopherTrunk's current
TETRA coverage.

## Sources

[^wiki]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, for the ETSI TETRA standard, its four-slot TDMA air interface, π/4-DQPSK modulation, and the ACELP vocoder.
[^etsi]: [ETSI — TETRA](https://www.etsi.org/technologies/tetra) — ETSI, the standards body, for the TETRA specifications, the main control channel, direct mode, and the TEDS data evolution.
