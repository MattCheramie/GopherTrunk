---
slug: bit-rate-vs-baud
title: Bit rate vs baud
entry_type: term
category: modulation
description: "Bit rate vs baud distinguishes symbols per second (baud) from bits per second; multi-level modulation carries several bits per symbol, so bit rate exceeds baud."
keywords: bit rate, baud rate, symbols per second, bits per symbol, bit rate vs baud, symbol rate, gross bit rate, modulation order, QAM bits per symbol
aka: [bit rate vs baud, baud rate, symbol rate vs bit rate]
autolink: true
infobox:
  - { label: Symbol, value: "Rb (bit/s), Rs (baud)" }
  - { label: Unit, value: "bit/s vs symbol/s (baud)" }
  - { label: Relation, value: "Rb = Rs · log2(M)" }
see_also: [symbol-rate, quadrature-amplitude-modulation, phase-shift-keying, spectral-efficiency, bandwidth]
cite_urls:
  - https://en.wikipedia.org/wiki/Baud
  - https://en.wikipedia.org/wiki/Symbol_rate
---

**Bit rate versus baud** is the distinction between how many **symbols** a link sends each second
(the baud rate, or [symbol rate](/reference/symbol-rate/)) and how many **bits** it carries each second
(the bit rate).[^wiki] They are equal only when each symbol conveys exactly one bit; whenever a
modulation packs several bits into each symbol, the bit rate is a multiple of the baud rate — a
distinction people routinely blur by calling everything "baud."

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A timeline of four symbol slots: a two-level scheme carries one bit per slot for four bits total, while a four-level scheme carries two bits per slot for eight bits at the same baud." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="35" font-size="8" fill="currentColor">2 levels</text>
  <text x="20" y="45" font-size="8" fill="currentColor">1 bit/sym</text>
  <line x1="90" y1="55" x2="430" y2="55" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M90 55 V35 H175 V55 M175 55 V35 H260 V55 M260 55 V70 H345 V55 M345 55 V35 H430" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="132" y="30" text-anchor="middle" font-size="8" fill="currentColor">1</text><text x="217" y="30" text-anchor="middle" font-size="8" fill="currentColor">1</text><text x="302" y="82" text-anchor="middle" font-size="8" fill="currentColor">0</text><text x="387" y="30" text-anchor="middle" font-size="8" fill="currentColor">1</text>
  <text x="20" y="115" font-size="8" fill="currentColor">4 levels</text>
  <text x="20" y="125" font-size="8" fill="currentColor">2 bit/sym</text>
  <line x1="90" y1="135" x2="430" y2="135" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M90 135 V105 H175 V125 H260 V95 H345 V115 H430" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="132" y="100" text-anchor="middle" font-size="8" fill="currentColor">11</text><text x="217" y="120" text-anchor="middle" font-size="8" fill="currentColor">01</text><text x="302" y="90" text-anchor="middle" font-size="8" fill="currentColor">10</text><text x="387" y="110" text-anchor="middle" font-size="8" fill="currentColor">00</text>
  <text x="260" y="152" text-anchor="middle" font-size="8" fill="currentColor">same 4 symbols → 4 bits vs 8 bits</text>
</svg>
<figcaption>Both rows send four symbols in the same time (same baud); the four-level scheme carries two bits per symbol, doubling the bit rate.</figcaption>
</figure>

## How it works

A symbol is one signalling state held for one symbol period; the baud rate R_s is how many of those
states are sent per second, and it — with the [roll-off](/reference/roll-off-factor/) — sets the
occupied [bandwidth](/reference/bandwidth/). The bit rate depends additionally on how many distinct
symbols the alphabet has. With M possible symbols, each one selects among M choices and therefore
carries log₂(M) bits, giving the core relation **R_b = R_s · log₂(M)**. Binary schemes (M = 2, e.g.
BPSK, 2-FSK) send one bit per symbol, so bit rate equals baud. A four-level scheme (M = 4, e.g. QPSK or
4-FSK) sends 2 bits/symbol; 16-[QAM](/reference/quadrature-amplitude-modulation/) sends 4; 256-QAM sends
8. So a modem running 4800 baud with a four-level alphabet moves 9600 bit/s — same symbol rate,
same bandwidth, double the data.

The appeal of higher-order modulation is exactly this: pack more bits into each symbol and raise the bit
rate (and [spectral efficiency](/reference/spectral-efficiency/)) without widening the channel. The cost
is that the constellation points sit closer together, so a denser alphabet needs more
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) to keep the error rate down. Choosing M is
therefore a bandwidth-versus-robustness trade, and bandwidth is tied to baud, not to bit rate.

## Relevance to SDR

Keeping the two rates straight is essential when reasoning about any digital signal an SDR decodes.
GopherTrunk's timing recovery locks to the **symbol** (baud) rate, because that is what determines where
to sample; the **bit** rate then follows from the modulation order. P25 Phase 1 C4FM runs 4800 symbols/s
with four levels for a 9600 bit/s gross channel rate, while P25 Phase 2 uses a two-slot TDMA π/4-DQPSK
scheme at 6000 symbols/s carrying more bits per symbol — different baud and bit rates for the same family.
When someone reports a "9600 baud" system that is really 4800 baud, four-level, the confusion is precisely
this bit-rate-versus-baud mix-up, and it changes what symbol clock a decoder should hunt for.

The confusion has deep roots. The unit "baud" honors Émile Baudot, whose 1870s telegraph code sent
one symbol per signalling interval, so in that binary world baud and bits per second genuinely were the
same number — and the habit of using the words interchangeably stuck long after multi-level modems made
them diverge. Early voiceband modems made the split concrete and public: a "2400 baud" telephone modem
running four-level or higher modulation carried 4800, 9600, or more bits per second over the same 2400
symbols, and later modems held the symbol rate near the channel's Nyquist limit while stacking ever
denser constellations to push the bit rate up. The channel bandwidth never changed; only the bits per
symbol did.

## In practice

Because bandwidth scales with baud and not bit rate, engineers size filters and channelizers from the
symbol rate, then report throughput as a bit rate. A quick sanity check on any modulation is
R_b / R_s = log₂(M): if that ratio is not a clean power-of-two count of bits, either the order or one of
the rates has been misquoted.

The distinction also reframes what "faster" means. Raising the baud rate widens the signal and needs
more spectrum; raising the bits per symbol raises throughput within the *same* spectrum but demands more
[signal-to-noise ratio](/reference/signal-to-noise-ratio/). Modern systems push both levers at once and
adapt them to conditions — cellular and Wi-Fi links switch to a denser constellation (more bits/symbol)
when the channel is clean and fall back to a sparser, more robust one when it degrades, all while holding
the symbol rate fixed. Understanding which quantity a spec is quoting is therefore the difference between
predicting a signal's bandwidth (a baud question) and predicting its data throughput or its noise margin
(a bits-per-symbol question).

## Sources

[^wiki]: [Baud](https://en.wikipedia.org/wiki/Baud) — Wikipedia, for the symbol-versus-bit distinction; Symbol rate for the Rb = Rs·log2(M) relation.
