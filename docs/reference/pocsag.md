---
slug: pocsag
title: POCSAG
entry_type: protocol
category: paging-data
description: "POCSAG (CCIR Radiopaging Code No. 1) is the classic asynchronous FSK paging protocol used worldwide for numeric and alphanumeric pager messages at 512, 1200, and 2400 bps."
keywords: POCSAG, paging, CCIR 584, ITU-R M.584, pager, FSK, capcode, numeric alphanumeric, DAPNET, fire EMS, batch codeword
aka: [POCSAG]
autolink: true
infobox:
  - { label: Type, value: One-way paging protocol }
  - { label: Standard, value: CCIR Radiopaging Code No. 1 (ITU-R M.584) }
  - { label: Modulation, value: 2-level FSK }
  - { label: Bit rates, value: 512 / 1200 / 2400 bps }
  - { label: Error correction, value: BCH(31,21) + parity }
  - { label: GopherTrunk support, value: Decoded }
see_also: [flex, ermes, frequency-shift-keying, bch-code, demodulation]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/POCSAG
  - https://www.itu.int/rec/R-REC-M.584/en
external:
  - { title: "GopherTrunk POCSAG decoder", url: /pocsag.html }
---

**POCSAG** (the **CCIR Radiopaging Code No. 1**) is the classic one-way **paging**
protocol used worldwide to deliver numeric and alphanumeric messages to pagers. It is
a simple asynchronous **2-level [FSK](/reference/frequency-shift-keying/)** scheme,
standardised internationally as ITU-R Recommendation M.584, and still in active use by
hospitals, fire/EMS, and industry.[^wiki][^itu] Its name comes from the **Post Office
Code Standardisation Advisory Group**, the British committee that defined it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A POCSAG transmission: preamble, sync codeword, then batches of address and message codewords sent to pagers." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="20" y="40" width="70" height="28" fill="none"/><rect x="90" y="40" width="50" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="140" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="220" y="40" width="80" height="28" fill="none"/><rect x="300" y="40" width="80" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="380" y="40" width="60" height="28" fill="none"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="55" y="58">preamble</text><text x="115" y="58">sync</text><text x="180" y="58">addr</text><text x="260" y="58">msg</text><text x="340" y="58">addr</text><text x="410" y="58">msg</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">2-FSK, one-way · 512/1200/2400 bps</text>
</svg>
<figcaption>POCSAG sends a preamble and sync, then batches of 32-bit address and message codewords, each pager waking for its own capcode.</figcaption>
</figure>

## Overview

POCSAG is deliberately austere, which is why it has outlived far more sophisticated
systems. A transmitter sends a long **preamble** (a 576-bit alternating pattern that
lets battery-saving pagers wake up and lock their bit clock), then a repeating
structure of **batches**. Each batch begins with a fixed 32-bit **sync codeword** and
contains eight **frames** of two 32-bit codewords each. A pager's identity (**capcode**
or RIC) selects which of the eight frame positions it must listen in, so most of the
time a pager's receiver can sleep and only power up for its slot — the trick that gives
pagers their famously long battery life. A codeword is either an **address codeword**
(marking that a message for a given capcode follows) or a **message codeword**
(carrying the numeric or alphanumeric payload).

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 2-FSK (±4.5 kHz typical deviation) |
| Bit rates | 512, 1200, 2400 bps (receiver auto-detects) |
| Codeword | 32 bits: 1 flag + 20 data + 10 BCH + 1 parity |
| Coding | [BCH(31,21)](/reference/bch-code/) + even parity |
| Batch | 1 sync codeword + 8 frames × 2 codewords |
| Content | Numeric (BCD), alphanumeric (7-bit), or tone-only |

## How it works

Every 32-bit codeword carries only 20 or 21 bits of usable payload; the rest is a
[BCH(31,21)](/reference/bch-code/) code plus an overall even-parity bit. BCH lets a
receiver **correct up to two bit errors and detect more** in each codeword, which is
what makes an uncoded, unacknowledged one-way link reliable enough for life-safety
paging — there is no return channel to request a retransmission, so the forward error
correction has to carry the whole burden. Numeric messages pack four bits per digit
(BCD); alphanumeric messages pack seven-bit characters across consecutive message
codewords. Because the three bit rates share the same codeword structure, a decoder
typically locks the preamble, measures the symbol period, and then reads the rest at
whichever of 512/1200/2400 bps it detected.

## History

POCSAG was developed by a British Post Office working group and adopted by the CCIR
(now ITU-R) in 1981 as Radiopaging Code No. 1, later maintained as Recommendation
M.584.[^itu] It rapidly displaced earlier, slower two-tone and five-tone paging and
became the dominant global paging code through the 1980s and 1990s. Higher-capacity
successors such as [FLEX](/reference/flex/) and the European
[ERMES](/reference/ermes/) system were built to handle traffic POCSAG's 512/1200 bps
rates could not, but POCSAG's simplicity kept it in service long after commercial
paging declined.

## Deployment

POCSAG remains widely used where a robust, infrastructure-light one-way alert matters:
hospital and clinical paging, fire and EMS dispatch, industrial and process-control
alerting, and utility SCADA notifications. The amateur-radio **DAPNET** network runs
POCSAG for operator paging. Because a POCSAG transmitter is little more than an FSK
modulator, it is also a common beginner target for SDR decoding.

## Decoding it with GopherTrunk

POCSAG is squarely in scope for GopherTrunk. The decoder FM-demodulates the channel,
recovers the 2-FSK symbol stream, synchronises on the preamble and sync codeword,
BCH-checks and error-corrects each codeword, and reassembles numeric and alphanumeric
messages by capcode — no proprietary vocoder or encryption stands in the way (though
some operators scramble content at the application layer, which GopherTrunk does not
attempt to unscramble). See the [POCSAG decoder](/pocsag.html) page.

## Sources

[^wiki]: [POCSAG](https://en.wikipedia.org/wiki/POCSAG) — Wikipedia, for the CCIR Radiopaging Code No. 1, its FSK bit rates, BCH coding, capcodes, and codeword/batch structure.
[^itu]: [Recommendation ITU-R M.584](https://www.itu.int/rec/R-REC-M.584/en) — ITU-R, the international standard for codes and formats for radiopaging that formalises POCSAG.
