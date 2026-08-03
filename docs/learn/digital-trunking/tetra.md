---
slug: tetra
title: "TETRA"
description: TETRA explained — the ETSI EN 300 392 standard for European and global public safety, with four-slot TDMA in 25 kHz, π/4-DQPSK, ACELP voice, TMO and DMO.
keywords: TETRA, Terrestrial Trunked Radio, ETSI EN 300 392, four-slot TDMA, pi/4-DQPSK, ACELP vocoder, TMO, DMO, 25 kHz channel, public safety Europe, trunked mode, direct mode
level: intermediate
status: full
prereq:
  - digital-modulation-for-trunking
  - tdma-vs-fdma
faq:
  - q: What is TETRA?
    a: TETRA (Terrestrial Trunked Radio) is an ETSI digital trunking standard, defined in EN 300 392, used heavily by public safety, transport, and industry outside North America. It uses four-slot TDMA in a 25 kHz channel with π/4-DQPSK modulation and an ACELP voice codec. It is a complete trunked system with strong support for group calls, direct mode, and encryption.
  - q: Why do I rarely hear TETRA in the United States?
    a: TETRA was developed in Europe and its standard 25 kHz channels and frequency plan did not fit cleanly into US allocations or the FCC narrowbanding push toward 12.5 kHz and below. North American public safety standardised on P25 instead. TETRA dominates Europe, much of Asia, the Middle East, Africa, and Latin America, but it is uncommon on US airwaves.
  - q: How wide is a TETRA channel on the waterfall?
    a: A TETRA carrier occupies 25 kHz, noticeably wider than the 12.5 kHz of P25 Phase 1 or DMR. That extra width is a useful first clue when you spot one on the waterfall — a single fat 25 kHz block carrying four interleaved time slots rather than a narrow 12.5 kHz lane.
  - q: What is the difference between TMO and DMO?
    a: TMO (Trunked Mode Operation) is the normal mode where radios talk through the infrastructure — base stations and the switching network — using the control channel. DMO (Direct Mode Operation) lets two radios talk directly to each other without any infrastructure, like a walkie-talkie, for when units are out of network range.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which protocols GopherTrunk decodes and how completely.
  - title: Constellation
    url: /constellation.html
    note: watch the π/4-DQPSK symbol cloud when inspecting a TETRA carrier.
---

# TETRA

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**TETRA** (Terrestrial Trunked Radio) is the **ETSI EN 300 392** standard that dominates
**public safety and commercial radio outside North America** — across Europe, Asia, the
Middle East, and beyond. It packs **four TDMA slots into a 25 kHz channel** using
**π/4-DQPSK** at **18000 symbols/s** (36 kbps gross) with an **ACELP** voice codec.
Radios work in **TMO** (through the network) or **DMO** (radio-to-radio, no
infrastructure), and **encryption is routine**. On the waterfall it is **notably wider —
25 kHz** — than P25 or DMR, and you'll **rarely meet it in the US**, where P25 won out.
</div>

You met TETRA briefly in the [protocol landscape](/learn/rf-sdr/protocol-landscape/). Here
is the full picture: a mature, feature-rich trunked standard that most of the world's
emergency services run on, even though it barely registers on North American airwaves.

## Where TETRA came from

TETRA was designed by **ETSI**, the European Telecommunications Standards Institute, and
published as the **EN 300 392** family of specifications in the 1990s. The goal was a
single open digital standard for *professional mobile radio* — the kind of mission-critical
network police, fire, ambulance, railways, airports, and utilities depend on. It succeeded
spectacularly outside the US: national public-safety networks across Europe, large parts of
Asia, the Middle East, Africa, and Latin America are TETRA. Think of it as the
rough-equivalent role that [P25](/learn/digital-trunking/p25-phase-1/) plays in North America, filling the same
public-safety niche with a different rulebook.

## How the air interface works

TETRA's physical layer is its most distinctive trait. A carrier is **25 kHz wide** and
carries **four time slots** in a TDMA frame, so up to four calls (or signaling plus three
calls) share one frequency by taking turns — the same time-sharing idea you met in
[TDMA vs FDMA](/learn/digital-trunking/tdma-vs-fdma/), but with *four* slots rather than DMR's two.

- **Modulation:** **π/4-DQPSK** — a differential four-phase scheme. Each symbol carries two
  bits, and the constellation rotates by a quarter-turn between symbols, which keeps the
  envelope from passing through zero and eases the transmitter's amplifier. This is *not*
  the [4FSK](/learn/digital-trunking/digital-modulation-for-trunking/) of P25 and DMR; it's phase modulation, and on
  the [constellation](/constellation.html) it shows up as a ring of points rather than four
  frequency levels.
- **Symbol rate:** **18000 symbols/s**, giving a **36 kbps** gross channel rate (2 bits per
  symbol). Split four ways, each slot gets a useful sub-rate for voice and signaling.
- **Voice codec:** **ACELP** (Algebraic Code-Excited Linear Prediction), a different family
  from the [AMBE/IMBE vocoders](/learn/rf-sdr/vocoders/) used by P25 and DMR. ACELP is the same
  general technique behind many cellphone codecs of the era.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="A single 25 kHz TETRA carrier shown as a wide block split into four time slots labelled 1 through 4, with a note that one slot carries control and the others carry calls." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor">
    <text x="260" y="22" text-anchor="middle" font-weight="600">One TETRA carrier — 25 kHz, four TDMA slots</text>
    <rect x="40" y="50" width="440" height="50" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.5"/>
    <line x1="150" y1="50" x2="150" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
    <line x1="260" y1="50" x2="260" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
    <line x1="370" y1="50" x2="370" y2="100" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="95" y="80" text-anchor="middle" font-size="10">Slot 1</text>
    <text x="205" y="80" text-anchor="middle" font-size="10">Slot 2</text>
    <text x="315" y="80" text-anchor="middle" font-size="10">Slot 3</text>
    <text x="425" y="80" text-anchor="middle" font-size="10">Slot 4</text>
    <text x="95" y="94" text-anchor="middle" font-size="8">control</text>
    <text x="205" y="94" text-anchor="middle" font-size="8">call</text>
    <text x="315" y="94" text-anchor="middle" font-size="8">call</text>
    <text x="425" y="94" text-anchor="middle" font-size="8">call</text>
    <text x="40" y="125" font-size="9">25 kHz — wider than P25/DMR's 12.5 kHz</text>
    <text x="260" y="150" text-anchor="middle" font-size="10">π/4-DQPSK at 18000 symbols/s → 36 kbps gross, shared four ways</text>
  </g>
</svg>
<figcaption>A TETRA carrier is one fat 25 kHz block carrying four interleaved time slots — wider than the narrow channels P25 and DMR use, which is your first visual clue.</figcaption>
</figure>

## TMO and DMO

TETRA radios operate in two modes:

- **TMO — Trunked Mode Operation.** The normal case: radios register with and talk through
  the **infrastructure** (base stations linked to a switching and management network),
  coordinated by a [control channel](/learn/digital-trunking/the-control-channel/). This gives you all the trunking
  benefits — group calls, individual calls, priority, area-wide coverage.
- **DMO — Direct Mode Operation.** Radios talk **directly to one another** with no
  infrastructure at all, like a simplex walkie-talkie. Crews use DMO when they're out of
  network coverage or want a local tactical channel that doesn't load the network.

The distinction matters for monitoring: a DMO conversation never touches a control channel,
so there's no central signaling to follow — you'd watch the simplex frequency directly.

## Why it's wide, and why that helps you

The headline practical fact is **bandwidth**. A TETRA carrier is **25 kHz**, double the
12.5 kHz of P25 Phase 1 or a DMR channel. On the [waterfall](/learn/rf-sdr/fft-and-waterfall/)
that fat block stands out, and combined with the *phase*-modulated constellation (a ring,
not four FSK levels) it's a reasonable bet you're looking at TETRA rather than a 4FSK
system — especially if you're outside North America.

## Encryption is common

Unlike many systems where clear voice is the default, **encryption is routine on TETRA**.
The standard defines both **air-interface encryption** (scrambling the whole radio link,
including signaling) and **end-to-end encryption** for the most sensitive users. Where it's
enabled, you'll see a healthy control channel and active traffic slots but get no
intelligible audio — the same wall you met in [encryption and
authentication](/learn/digital-trunking/encryption-and-authentication/). That's expected on a lot of TETRA networks.

<div class="knowledge-check" data-quiz data-correct-msg="Right — TETRA uses 25 kHz channels with four TDMA slots and π/4-DQPSK." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a TETRA carrier stand out on the waterfall compared with P25 or DMR?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It is much narrower, around 6.25 kHz</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It is wider — a 25 kHz block</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses 4FSK with four frequency levels</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **TETRA** is the **ETSI EN 300 392** trunked standard that dominates public safety and
  commercial radio **outside North America**.
- It uses **four-slot TDMA in a 25 kHz channel** with **π/4-DQPSK** at **18000 symbols/s**
  (36 kbps) and the **ACELP** voice codec.
- Radios run in **TMO** (through the network) or **DMO** (direct, no infrastructure).
- It is **wider** than P25/DMR — a 25 kHz block — and uses *phase* rather than frequency
  modulation, both handy identification clues.
- **Encryption is common**, so don't be surprised by active-but-silent TETRA traffic.

Next we cross back to a narrowband North-American favourite and its global cousins: [NXDN](/learn/digital-trunking/nxdn/).
