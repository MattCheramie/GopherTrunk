---
slug: antenna-tuner
title: Antenna tuner (ATU)
entry_type: term
category: antennas
description: An antenna tuner is a matching network placed between transceiver and feedline that transforms an antenna's feedpoint impedance to the 50-ohm value the radio expects, lowering the standing-wave ratio.
keywords: antenna tuner, ATU, antenna tuning unit, matching network, impedance matching, transmatch, L-network, T-network, pi-network, SWR, 50 ohm
aka: [antenna tuner, ATU, antenna tuning unit, transmatch]
autolink: true
infobox:
  - { label: Type, value: Impedance-matching network }
  - { label: Does, value: Transforms load Z to ~50 Ω }
  - { label: Topologies, value: L, T, and Pi networks }
see_also: [standing-wave-ratio, feedpoint-impedance, impedance, balun, reflection-coefficient]
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_tuner
---

**An antenna tuner (ATU)**, also called a matching unit or transmatch, is a network of adjustable
inductors and capacitors inserted between a radio and its feedline that transforms the antenna's
[feedpoint impedance](/reference/feedpoint-impedance/) into the roughly 50 Ω the transceiver is
designed to drive.[^wiki] It does not "tune" the antenna itself — the antenna's resonance is
unchanged — but it presents the radio with a matched load, lowering the
[standing-wave ratio](/reference/standing-wave-ratio/) that the transmitter sees and letting it
deliver full power without folding back its output.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 155" role="img" aria-label="A transceiver connects through a matching network of a series inductor and two shunt capacitors to a feedline and antenna, transforming a mismatched load to fifty ohms." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="atuar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="60" height="45" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="50" y="82" text-anchor="middle" font-size="8.5" fill="currentColor">radio</text>
  <text x="50" y="118" text-anchor="middle" font-size="8" fill="currentColor">50 Ω</text>
  <line x1="80" y1="70" x2="130" y2="70" stroke="currentColor" stroke-width="1.4"/>
  <rect x="130" y="45" width="130" height="60" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="195" y="40" text-anchor="middle" font-size="8.5" fill="currentColor">matching network</text>
  <path d="M145 70 q8 -10 16 0 q8 10 16 0 q8 -10 16 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <line x1="210" y1="70" x2="245" y2="70" stroke="currentColor" stroke-width="1.3"/>
  <line x1="150" y1="70" x2="150" y2="98" stroke="currentColor" stroke-width="1.1"/>
  <line x1="144" y1="98" x2="156" y2="98" stroke="currentColor" stroke-width="1.4"/>
  <line x1="225" y1="70" x2="225" y2="98" stroke="currentColor" stroke-width="1.1"/>
  <line x1="219" y1="98" x2="231" y2="98" stroke="currentColor" stroke-width="1.4"/>
  <line x1="260" y1="70" x2="330" y2="70" stroke="currentColor" stroke-width="1.4" marker-end="url(#atuar)"/>
  <text x="300" y="62" text-anchor="middle" font-size="8" fill="currentColor">feedline</text>
  <line x1="345" y1="70" x2="380" y2="45" stroke="currentColor" stroke-width="1.8"/>
  <line x1="345" y1="70" x2="380" y2="95" stroke="currentColor" stroke-width="1.8"/>
  <text x="400" y="72" font-size="8.5" fill="currentColor">Z_ant</text>
</svg>
<figcaption>A tuner inserts adjustable reactances (here a T-network) between radio and antenna to present the radio a 50-ohm match.</figcaption>
</figure>

## How it works

Impedance matching is the business of cancelling reactance and transforming resistance. A tuner
does both with reactive components — inductors and capacitors store and release energy without
dissipating it, so (aside from small residual losses) they transform impedance rather than absorb
power. Adjusting them moves the load's [reflection coefficient](/reference/reflection-coefficient/)
to the centre of the Smith chart, i.e. to 50 Ω. Three network topologies are common:

- **L-network** — one series and one shunt element. The simplest; it can match any single load to
  50 Ω but with limited flexibility and no independent control of loaded Q.
- **T-network** — two series elements around a shunt element. The most common in commercial HF
  tuners; wide matching range, though it can pass through high-Q, lossy states if misadjusted.
- **Pi-network** — two shunt elements around a series element. Common in transmitter output
  stages and offers good harmonic attenuation.

Because reactances are frequency-dependent, a tuner match holds only near the frequency it was
set for; changing bands means re-tuning. Tuners are either adjusted manually by peaking for
minimum SWR, or done automatically by a microcontroller that steps relay-switched components until
an internal SWR bridge reads a minimum.

## In practice

A crucial caveat: a tuner **at the radio end** hides the mismatch from the transmitter but does
nothing to fix the standing wave on the feedline between the tuner and the antenna. Power still
sloshes back and forth on that section, and any feedline loss is magnified by the mismatch. The
tuner's job is to protect the transmitter and let it load up, not to make a poor antenna radiate
better. To actually reduce feedline loss, the match must be made **at the antenna**. Tuners also
frequently include a [balun](/reference/balun/) to feed a balanced antenna or open-wire line from
the unbalanced coax world, and their duty is real only on transmit — a mismatch matters far less
on receive.

## Relevance to SDR

For receive-only SDR work a tuner is rarely necessary: no power is being reflected into a power
amplifier, and a modest mismatch mostly costs a little signal level rather than causing damage.
Where a tuner can help is with a compromise or electrically short receiving antenna on the lower
bands, where matching its high reactance to 50 Ω recovers signal that would otherwise be lost to
mismatch, improving delivered signal-to-noise ratio. Some enthusiasts use a small
[preselector](/reference/rf-filter/) with a matching stage for the same reason. **GopherTrunk**
itself has no concept of matching — it consumes IQ samples — but a proper match at the front end
is one of the physical steps that maximizes the signal reaching the decoder, especially for weak
HF or low-VHF captures.

## Sources

[^wiki]: [Antenna tuner](https://en.wikipedia.org/wiki/Antenna_tuner) — Wikipedia, for the matching-network topologies and the distinction between tuning at the radio versus at the antenna.
