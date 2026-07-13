---
slug: manchester-coding
title: Manchester coding
entry_type: algorithm
category: modulation
description: Manchester coding is a line code that encodes each bit as a mid-bit voltage transition, making it self-clocking and DC-balanced; used in POCSAG paging, 10BASE-T Ethernet, RFID, and telemetry.
keywords: Manchester coding, Manchester encoding, line code, self-clocking, DC balance, mid-bit transition, biphase, POCSAG, 10BASE-T Ethernet, RFID, telemetry, clock recovery
aka: [Manchester code, Manchester encoding, biphase-L]
autolink: true
infobox:
  - { label: Type, value: Line code (biphase) }
  - { label: Property, value: Self-clocking, DC-balanced }
  - { label: Used by, value: POCSAG, 10BASE-T, RFID }
see_also: [pocsag, nrzi, differential-decoding, bit-rate-vs-baud, clock-recovery, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Manchester_code
  - https://en.wikipedia.org/wiki/Line_code
---

**Manchester coding** is a line code in which each data bit is represented not by a steady
level but by a **transition in the middle of the bit period** — the direction of that
mid-bit edge carries the bit value.[^wiki] This makes the code **self-clocking** (a
transition in every bit lets the receiver recover timing) and **DC-balanced** (equal time
high and low), two properties that make it robust over channels that cannot carry DC or a
separate clock, such as paging and early Ethernet.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A data bit sequence and its Manchester-coded waveform, where each bit contains a mid-bit transition: a rising edge encodes one value and a falling edge the other, guaranteeing a transition every bit period." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="20" y="30" text-anchor="start" font-size="8">bits</text>
    <text x="70" y="30">1</text><text x="130" y="30">0</text><text x="190" y="30">1</text><text x="250" y="30">1</text><text x="310" y="30">0</text>
    <g stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="2 3"><line x1="40" y1="45" x2="40" y2="110"/><line x1="100" y1="45" x2="100" y2="110"/><line x1="160" y1="45" x2="160" y2="110"/><line x1="220" y1="45" x2="220" y2="110"/><line x1="280" y1="45" x2="280" y2="110"/><line x1="340" y1="45" x2="340" y2="110"/></g>
  </g>
  <g stroke="currentColor" stroke-width="1.6" fill="none">
    <path d="M40 100 L70 100 L70 60 L100 60 L100 60 L130 60 L130 100 L160 100 L160 60 L190 60 L190 100 L220 100 L220 60 L250 60 L250 100 L280 100 L280 60 L310 60 L310 100 L340 100"/>
  </g>
  <text x="190" y="124" text-anchor="middle" font-size="8" fill="currentColor">a transition sits in the centre of every bit — the receiver clocks off it</text>
</svg>
<figcaption>In Manchester coding every bit carries a mid-bit edge, so timing is embedded in the data and the average level stays at zero.</figcaption>
</figure>

## How it works

Manchester coding maps each bit onto a **two-level, half-bit-each** symbol, equivalent to
XORing the data bit with a clock running at the bit rate:

- **The mid-bit transition is the bit.** In the common IEEE 802.3 convention a **rising**
  mid-bit edge encodes a 1 and a **falling** edge encodes a 0 (the original G. E. Thomas
  convention is the opposite). Whatever edges are needed at bit boundaries to set up the next
  mid-bit transition are "housekeeping" and carry no data.
- **Self-clocking.** Because there is guaranteed to be an edge in the centre of every bit,
  the receiver's [clock-recovery](/reference/clock-recovery/) loop always has an event to
  lock onto — there can be no long run of unchanging level to lose sync on, unlike plain NRZ.
- **DC balance.** Each bit spends exactly half its time high and half low, so the running
  average is zero. That lets the signal pass through transformer- or capacitor-coupled links
  and AC-coupled receivers without baseline drift.
- **The cost is bandwidth.** Guaranteeing a mid-bit edge doubles the maximum signalling
  rate, so Manchester needs roughly twice the bandwidth of an
  [NRZI](/reference/nrzi/)-style code carrying the same data rate. That trade — spectral
  efficiency for robust timing and DC balance — is the whole design decision.

A receiver decodes by sampling the two half-bit levels (or detecting the direction of the
central edge); ambiguity from an inverted signal is handled by a known preamble or by pairing
Manchester with [differential decoding](/reference/differential-decoding/).

Because every bit is squeezed into two half-bit signalling elements, the line's symbol rate is
twice the data rate — the [bandwidth cost](/reference/bit-rate-vs-baud/) that pays for the
embedded clock and DC balance.

## Variants

Several closely related biphase codes trade off the same properties differently:

- **Differential Manchester (biphase-M/S).** Instead of an absolute rising-versus-falling
  convention, the bit is encoded by whether there *is* a transition at the bit boundary, while the
  mandatory mid-bit edge is used only for clocking. Like [NRZI](/reference/nrzi/) this makes the
  code immune to a whole-signal inversion, at the cost of the simple "edge direction = value"
  read-out. It is the physical coding of Token Ring and some fieldbus links.
- **Thomas vs. IEEE 802.3 convention.** The two mainstream conventions assign the rising and
  falling mid-bit edges to opposite bit values, so a decoder must know which one a transmitter
  uses (or resolve it from a known preamble) to avoid inverting every bit.
- **Biphase mark/space (FM0/FM1).** Related self-clocking codes used in RFID (EPC Gen2), audio
  time-code (SMPTE/LTC), and aviation ARINC buses, chosen for the same AC-coupling robustness.

Because Manchester needs no separate clock line and tolerates AC coupling, it appears wherever
a simple, robust, self-synchronising bitstream is wanted: **POCSAG** and other paging bursts,
**10BASE-T** and 10-Mbit Ethernet, near-field/RFID tag links, IR remote and consumer-IR
protocols, and countless low-rate telemetry and sensor radios. It is favoured in bursty,
preamble-then-data formats where fast, reliable clock acquisition matters more than squeezing
out bandwidth.

## Relevance to SDR

For a scanner the most relevant case is [POCSAG](/reference/pocsag/) paging, where the
2-level FSK baseband is essentially a Manchester-style self-clocking stream a decoder must
bit-sync and slice. Recognising Manchester's guaranteed mid-bit transition tells the decoder
where the symbol clock is and how to reject an accidental level inversion. GopherTrunk decodes
POCSAG and similar self-clocked paging/telemetry formats; understanding the line code explains
how bit timing is recovered from the [demodulated](/reference/demodulation/)
[FSK](/reference/frequency-shift-keying/) baseband before framing.

## Sources

[^wiki]: [Manchester code](https://en.wikipedia.org/wiki/Manchester_code) — Wikipedia, for the mid-bit-transition encoding, self-clocking and DC-balance properties, and conventions.
