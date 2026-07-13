---
slug: bias-tee
title: Bias tee
entry_type: hardware
category: rf-front-end
description: A bias tee injects DC power onto the coax feeding an antenna-mounted device such as an LNA, while passing the RF signal through to the receiver.
keywords: bias tee, bias-T, DC injection, LNA power, phantom power, coax, mast-mounted amplifier, inductor capacitor network
aka: [bias tee, bias-T]
autolink: true
infobox:
  - { label: Type, value: RF/DC combining network }
  - { label: Function, value: DC power up the coax, RF through }
  - { label: Powers, value: Antenna-mounted LNA / active antenna }
see_also: [low-noise-amplifier, preamplifier, coaxial-cable, antenna, rtl-sdr, sma-connector]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bias_tee
  - https://en.wikipedia.org/wiki/Inductor
---

A **bias tee** is a small three-port network that injects **DC power onto the coax**
feeding an antenna-mounted device — typically a
[low-noise amplifier](/reference/low-noise-amplifier/) or active antenna — while passing the
RF signal through to the receiver unaffected.[^wiki] It lets a **single**
[coaxial cable](/reference/coaxial-cable/) carry both the faint received signal coming down
and the supply voltage going up, so a mast-mounted amplifier needs no separate power run.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A bias tee: an inductor feeds DC onto the coax while a series capacitor blocks DC from reaching the receiver, passing RF through." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="55" x2="150" y2="55" stroke="currentColor" stroke-width="1.4"/><text x="70" y="46" font-size="8" fill="currentColor">RF + DC (to LNA)</text>
  <path d="M150 55 q6 -8 12 0 q6 8 12 0 q6 -8 12 0 q6 8 12 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="198" y1="55" x2="420" y2="55" stroke="currentColor" stroke-width="1.4"/><text x="360" y="46" font-size="8" fill="currentColor">RF to RX</text>
  <text x="215" y="46" font-size="8" fill="currentColor">‖ (blocks DC)</text>
  <line x1="150" y1="55" x2="150" y2="90" stroke="currentColor" stroke-width="1.4"/>
  <path d="M143 90 h14 M145 94 h10 M147 98 h6" stroke="currentColor" stroke-width="1.2" fill="none"/>
  <text x="150" y="118" font-size="8" fill="currentColor" text-anchor="middle">DC supply (via inductor)</text>
  <circle cx="150" cy="55" r="3" fill="currentColor"/>
</svg>
<figcaption>A bias tee feeds DC up the coax through an inductor to power a mast-mounted LNA, while a series capacitor blocks that DC from the receiver and passes the RF.</figcaption>
</figure>

## How it works

The trick is two frequency-selective components. A **series capacitor** in the RF path
passes the high-frequency signal but blocks DC, so the supply voltage never reaches the
receiver's sensitive input. A **shunt inductor** (an RF choke) connects the DC supply to the
signal line: it presents a near-short to DC but a high impedance to RF, so the amplifier
gets its voltage while almost none of the RF leaks away into the power supply.[^ind]
Together they combine the two energy paths onto one conductor without either interfering
with the other. A good bias tee holds this trick flat across its whole frequency range;
outside that range the choke or cap stops behaving ideally and either RF is lost or DC
regulation suffers.

Two bias tees are usually involved end to end: one at the receiver *injects* the DC, and a
matching decoupling network inside the mast-mounted device *extracts* it to power the
amplifier while feeding the signal on.

## Variants

- **Built-in switchable bias tee** — many [RTL-SDR](/reference/rtl-sdr/) dongles,
  [Airspy](/reference/airspy/), and SDRplay units can enable a fixed 4.5–5 V feed in
  software. Convenient, but current-limited (often ~180 mA).
- **Inline / external bias tee** — a separate module with its own regulated supply, used
  when the LNA needs more current or a different voltage (9–15 V is common for higher-power
  amplifiers).
- **Wideband vs narrowband** — a lab bias tee may span DC–6 GHz; a cheap hobby one is
  specified only over a hobby band.

## In practice

Never enable a bias tee into a device that is not expecting DC — a passive antenna, a
[SAW filter](/reference/saw-filter/) with a DC-grounded input, or a splitter can short the
supply and, at worst, damage the SDR. Check the current rating against the LNA's draw, and
confirm the connector polarity. When it works, the payoff is real: the amplifier lives right
at the [antenna](/reference/antenna/), the coax loss that follows it is suppressed by the
LNA's gain (per the Friis budget), and only one cable runs down the mast.

## Relevance to SDR

The bias tee is the standard way SDR hobbyists power mast-mounted LNAs and active antennas
for ADS-B, satellite, and weak-signal work. GopherTrunk itself is purely a decoder and does
not toggle bias-tee hardware, but the practice it enables — a low-noise amplifier at the
antenna over a single feedline — directly improves the [SNR](/reference/signal-to-noise-ratio/)
of the [IQ](/reference/iq-data/) stream GopherTrunk decodes.

## Sources

[^wiki]: [Bias tee](https://en.wikipedia.org/wiki/Bias_tee) — Wikipedia, on the RF/DC combining network that powers antenna-mounted devices over the coax.
[^ind]: [Inductor](https://en.wikipedia.org/wiki/Inductor) — Wikipedia, on the RF choke behaviour (high impedance to RF, low to DC) that lets the shunt inductor feed DC without shorting the signal.
