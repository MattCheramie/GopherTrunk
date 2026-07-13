---
slug: cqpsk
title: CQPSK
entry_type: technology
category: modulation
description: "CQPSK (compatible QPSK) is the linear phase-modulation counterpart to C4FM used on P25, producing the same symbols so one demodulator can handle both transmit paths."
keywords: CQPSK, compatible QPSK, LSM, linear simulcast modulation, P25, phase modulation, simulcast, root raised cosine, linear amplifier
aka: [CQPSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (phase) }
  - { label: Related to, value: C4FM (same symbols) }
  - { label: Used by, value: P25 (linear/simulcast path) }
see_also: [phase-shift-keying, c4fm, project-25, constellation-diagram, qpsk, pi-4-dqpsk, simulcast, iq-modulation, root-raised-cosine-filter]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Phase-shift_keying
---

**CQPSK** (compatible QPSK, also linear simulcast modulation, LSM) is the linear
[phase-modulation](/reference/phase-shift-keying/) counterpart to
[C4FM](/reference/c4fm/) used on [P25](/reference/project-25/). It produces the **same
symbol stream** as C4FM so a single demodulator can receive either.[^p25] Where C4FM is a
constant-envelope FSK signal, CQPSK is an amplitude-and-phase-varying linear signal — yet
by design they arrive at the receiver as the same four-level symbol sequence.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 210" role="img" aria-label="A QPSK constellation with four points and arcs showing the linear phase transitions between them." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="270" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="20" x2="150" y2="190" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M205 55 A 78 78 0 0 1 205 155" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
  <g fill="currentColor"><circle cx="205" cy="55" r="5"/><circle cx="95" cy="55" r="5"/><circle cx="95" cy="155" r="5"/><circle cx="205" cy="155" r="5"/></g>
  <text x="150" y="205" text-anchor="middle" font-size="9" fill="currentColor">linear phase transitions (compatible with C4FM detection)</text>
</svg>
<figcaption>CQPSK conveys the same symbols as C4FM on a linear (phase) path, so one receiver design handles both.</figcaption>
</figure>

## How it works

CQPSK maps the same 4800-baud [dibit](/reference/dibit/) stream that C4FM carries onto
carrier *phase* rather than frequency, using [IQ modulation](/reference/iq-modulation/)
with [root-raised-cosine](/reference/root-raised-cosine-filter/) pulse shaping. The clever
part is that P25 chose the two schemes so that C4FM's frequency trajectory and CQPSK's
phase trajectory are mathematically linked — frequency is the derivative of phase — with
the result that a receiver's symbol slicer sees the identical four levels either way.
This is why a P25 radio never has to know in advance which modulation the far end used;
it recovers symbols and the physical modulation is transparent above the slicer.

The price CQPSK pays for its phase-linear shaping is that it needs a **linear** power
amplifier: the signal's envelope dips between constellation points, and a saturated
Class-C amplifier would clip those dips and regrow the spectrum. Subscriber portables,
which prize battery efficiency, therefore transmit C4FM, while infrastructure transmitters
that can afford a linear amplifier often use CQPSK/LSM.

## Variants

CQPSK is closely related to the [π/4-DQPSK](/reference/pi-4-dqpsk/) family used elsewhere
in land-mobile radio; the shared idea is a shaped [QPSK](/reference/qpsk/) whose trajectory
avoids abrupt phase jumps. The reason CQPSK exists at all is
[simulcast](/reference/simulcast/): when several transmitters on the same frequency cover
a wide area, their signals overlap at the receiver. A linear modulation superimposes
those overlapping signals more gracefully than a constant-envelope FSK signal would,
producing fewer distortion products in the combined waveform — hence the name *linear
simulcast modulation*. Decoders often benefit from an LSM-aware equaliser or a slightly
different symbol-timing strategy when a P25 system is running CQPSK simulcast.

## Relevance to SDR

A P25 receiver can demodulate both C4FM and CQPSK, and on the
[constellation](/reference/constellation-diagram/) the recovered symbols look alike once
locked. GopherTrunk decodes the P25 Phase 1 symbol stream regardless of which transmit
path produced it, though heavily distorted simulcast CQPSK is one of the harder
real-world cases — multipath and transmitter timing spread the constellation, and the
[GopherTrunk DSP notes](/reference/simulcast/) treat such captures as
front-end/channel-limited rather than a defect in the steady-state demodulator.

## Sources

[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the CQPSK/LSM linear path and its compatibility with C4FM on P25.
[^psk]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, for the linear QPSK basis and its linear-amplifier requirement.
