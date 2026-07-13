---
slug: duty-cycle
title: Duty Cycle
entry_type: term
category: rf-fundamentals
description: Duty cycle is the fraction of time a transmitter is actually keyed, setting the ratio of average to peak power that governs heating and thermal design.
keywords: duty cycle, transmit duty cycle, average power, peak power, thermal design, keying, TDMA duty cycle, continuous carrier, intermittent transmission
aka: [duty cycle, duty ratio, transmit duty cycle]
autolink: true
infobox:
  - { label: Type, value: Time-domain ratio (0–100%) }
  - { label: Formula, value: "D = t_on / (t_on + t_off)" }
  - { label: Sets, value: "Average power = D × peak power" }
see_also: [power-amplifier, crest-factor-papr, tdma, cooling-and-thermals, erp-eirp, pulse-width-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Duty_cycle
---

**Duty cycle** is the fraction of a repeating cycle during which a transmitter is
actively emitting — *D = t_on / (t_on + t_off)*, expressed as a percentage.[^wiki] It
directly sets the ratio of **average power to peak power**, which in turn governs how
much heat a [power amplifier](/reference/power-amplifier/) must dissipate and therefore
how the radio is cooled and rated.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two pulse trains: the upper has narrow on-pulses with wide gaps for low duty cycle, the lower has wide on-pulses for high duty cycle, both over the same time axis." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dutyar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" font-size="10">
    <path d="M40 45 h30 v-25 h20 v25 h60 v-25 h20 v25 h60 v-25 h20 v25 h60" stroke-width="1.6"/>
    <path d="M40 120 h20 v-25 h70 v25 h20 v-25 h70 v25 h20 v-25 h70 v25 h30" stroke-width="1.6"/>
  </g>
  <g fill="currentColor" font-size="10" stroke="none">
    <text x="360" y="32">low duty (~25%)</text>
    <text x="360" y="107">high duty (~75%)</text>
    <text x="40" y="142">time →</text>
  </g>
</svg>
<figcaption>Same peak amplitude, different duty cycle: the low-duty train transmits briefly with long gaps; the high-duty train stays keyed most of the time and dissipates far more average power.</figcaption>
</figure>

## How it works

Peak (envelope) power is set by how hard the amplifier is driven during a transmit
burst; **average power** is that peak scaled by the duty cycle. A 50 W transmitter keyed
10 % of the time delivers only 5 W average. Because heating in the final stage tracks
*average* dissipated power, a low duty cycle lets a small amplifier and heatsink handle
bursts that would overheat them if sustained continuously.

Duty cycle spans a wide range in practice:

- **Continuous / 100 %** — analog FM broadcast, an unmodulated
  [carrier wave](/reference/carrier-wave/), and data modes that hold the transmitter
  keyed for the whole message.
- **High** — a busy repeater or a control channel that transmits almost continuously.
- **Low / bursty** — push-to-talk voice (keyed only while someone speaks) and
  [TDMA](/reference/tdma/) systems where each subscriber transmits in assigned time
  slots and is off between them.

Duty cycle is about *time on versus off*; it is distinct from
[crest factor / PAPR](/reference/crest-factor-papr/), which describes the amplitude
statistics *within* a transmission. A signal can have 100 % duty cycle yet a high PAPR,
or a low duty cycle yet a nearly constant envelope while keyed.

## In practice

Component and radio datasheets quote power ratings against a stated duty cycle. A
[dummy load](/reference/dummy-load/) rated "100 W continuous, 300 W intermittent"
survives higher peaks only if the average — after applying the duty cycle — stays within
its thermal limit. Amateur and commercial data modes that approach 100 % duty (FT8,
RTTY, digital voice) commonly force operators to reduce output power below the rig's SSB
rating so the finals do not overheat. Antenna and feedline power handling, and the
sizing of cooling, all flow from the same average-power calculation.

## Relevance to SDR

Duty cycle explains a lot about how trunked and time-division systems behave on the air.
A [TDMA](/reference/tdma/) protocol like P25 Phase 2 or DMR runs each logical channel at
roughly 50 % duty because two conversations share one RF carrier in alternating slots;
the transmitter's average power — and its battery drain in a portable — falls
accordingly. A [control channel](/reference/control-channel/), by contrast, runs at very
high duty because it must broadcast signalling continuously for subscribers to find it.

**GopherTrunk** is a receiver and never keys a transmitter, so it has no duty-cycle
rating of its own. The concept is still useful for interpreting captures: the bursty,
slotted structure of a TDMA signal in a [spectrogram](/reference/spectrogram/) is a
direct picture of its duty cycle, and recognising a near-continuous versus a slotted
emission helps identify which protocol and channel type is present before decoding.

## Sources

[^wiki]: [Duty cycle](https://en.wikipedia.org/wiki/Duty_cycle) — Wikipedia, definition of the on-time fraction and its relation to average power.
