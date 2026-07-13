---
slug: frequency-hopping-spread-spectrum
title: Frequency-hopping spread spectrum (FHSS)
entry_type: algorithm
category: spread-spectrum
description: FHSS rapidly retunes the carrier across many channels on a pseudo-random schedule, spreading energy over a wide band for interference resistance and low probability of intercept; used by Bluetooth and military radios.
keywords: frequency-hopping spread spectrum, FHSS, frequency hopping, hop set, hopping pattern, Bluetooth, dwell time, LPI, anti-jam, SINCGARS
aka: [FHSS, frequency hopping]
autolink: true
infobox:
  - { label: Type, value: Spread-spectrum modulation }
  - { label: Spreads via, value: Pseudo-random carrier hopping }
  - { label: Used by, value: Bluetooth, military radios }
see_also: [direct-sequence-spread-spectrum, maximal-length-sequence, cdma, tdma, fdma, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-hopping_spread_spectrum
  - https://www.bluetooth.com/specifications/specs/core-specification/
---

**Frequency-hopping spread spectrum (FHSS)** repeatedly retunes the carrier across many
narrow channels following a pseudo-random **hop pattern** known to both ends, so the signal's
energy is spread across a wide band over time rather than all at once.[^wiki] Any given
transmission lingers on one channel for a short **dwell time** before jumping to the next, so
a narrowband interferer or fade only corrupts the hops that happen to land on it — the rest
get through, and forward error correction stitches the message back together.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A time-versus-frequency grid where the transmitted carrier occupies a different pseudo-random channel in each time slot, tracing a scattered hop pattern." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.6" fill="none" stroke-opacity="0.4">
    <path d="M40 20 V150 M40 150 H430"/>
  </g>
  <g fill="currentColor">
    <rect x="60" y="110" width="34" height="16"/>
    <rect x="100" y="40" width="34" height="16"/>
    <rect x="140" y="90" width="34" height="16"/>
    <rect x="180" y="60" width="34" height="16"/>
    <rect x="220" y="130" width="34" height="16"/>
    <rect x="260" y="30" width="34" height="16"/>
    <rect x="300" y="100" width="34" height="16"/>
    <rect x="340" y="50" width="34" height="16"/>
    <rect x="380" y="120" width="34" height="16"/>
  </g>
  <text x="235" y="165" text-anchor="middle" font-size="9" fill="currentColor">time (each slot = one dwell) →</text>
  <text x="18" y="90" text-anchor="middle" font-size="9" fill="currentColor" transform="rotate(-90 18 90)">frequency ↑</text>
</svg>
<figcaption>FHSS moves the carrier to a new pseudo-random channel every dwell; a jammer on one channel only ever hits a fraction of the hops.</figcaption>
</figure>

## How it works

A pseudo-random number generator — often driven by a
[maximal-length sequence](/reference/maximal-length-sequence/) or a keyed cipher — produces
the sequence of channel indices, the **hop set**. Transmitter and receiver run the same
generator from the same seed and stay time-synchronized, so the receiver's local oscillator
retunes in lockstep and always lands where the signal is. Symbols are usually carried by a
simple modulation ([FSK](/reference/frequency-shift-keying/) or
[GFSK](/reference/gfsk/)) within each hop channel.

Two regimes are distinguished by how the hop rate compares to the symbol rate:

- **Slow hopping** sends many symbols per hop (e.g. Bluetooth: 1600 hops/s, hundreds of
  symbols per dwell). Simpler, but a hit on one channel loses a whole block of symbols.
- **Fast hopping** changes frequency several times *per symbol*, giving strong anti-jam and
  frequency diversity at the cost of a fast, agile synthesizer.

Processing gain comes from the ratio of the total hopped bandwidth to the instantaneous
channel bandwidth. Unlike [DSSS](/reference/direct-sequence-spread-spectrum/), the
instantaneous signal is *narrowband* — it just never stays in one place — which makes FHSS
tolerant of the near-far problem and easy to build with conventional narrowband radios.

## In practice

Multiple FHSS users can share a band with *different* hop patterns; when two happen to
collide on the same channel in the same slot, both lose that hop and rely on FEC and
retransmission — a form of [CDMA](/reference/cdma/) by orthogonal hopping rather than
orthogonal codes. Adaptive schemes (Bluetooth's Adaptive Frequency Hopping) drop channels
that are persistently busy from the hop set.

## Relevance to SDR

**Bluetooth** and Bluetooth Low Energy are the most familiar FHSS systems; military radios
(SINCGARS, HAVE QUICK) use it for anti-jam and LPI, and some proprietary telemetry and
cordless systems hop as well. The land-mobile trunking protocols GopherTrunk targets do
**not** hop their traffic *channels* — P25, DMR, and TETRA assign fixed frequencies via a
[control channel](/reference/control-channel/), and a scanner follows those grants rather
than a hop pattern.

That distinction is the practical takeaway for a scanner operator: to receive a genuinely
frequency-hopping transmission you need the **hop set and hop timing**, not just a center
frequency — without the pattern and synchronization, each intercepted dwell is an isolated,
unintelligible fragment. GopherTrunk therefore does not attempt to follow hopped links; it
decodes the fixed-channel FDMA/TDMA systems where every voice and control channel sits at a
known frequency.

## Sources

[^wiki]: [Frequency-hopping spread spectrum](https://en.wikipedia.org/wiki/Frequency-hopping_spread_spectrum) — Wikipedia, for the hop-pattern mechanism, dwell time, and slow/fast hopping.
