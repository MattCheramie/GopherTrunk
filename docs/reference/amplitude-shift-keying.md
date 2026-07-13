---
slug: amplitude-shift-keying
title: Amplitude-shift keying (ASK)
entry_type: technology
category: modulation
description: Amplitude-shift keying (ASK) is digital modulation that encodes bits in the carrier's amplitude; on-off keying is its simplest binary form, common in ISM-band devices.
keywords: ASK, amplitude shift keying, digital modulation, OOK, on-off keying, amplitude modulation, envelope detector, ISM band, multi-level ASK
aka: [amplitude-shift keying, ASK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Carrier amplitude (discrete levels) }
  - { label: Used by, value: RFID, remotes, optical links }
see_also: [on-off-keying, amplitude-modulation, quadrature-amplitude-modulation, carrier-wave, frequency-shift-keying, phase-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Amplitude-shift_keying
  - https://en.wikipedia.org/wiki/On%E2%80%93off_keying
---

**Amplitude-shift keying** (**ASK**) is digital [modulation](/reference/modulation/) that
represents each [symbol](/reference/symbol-rate/) by a discrete amplitude of the
[carrier](/reference/carrier-wave/), while its frequency and phase stay fixed.[^wiki] It
is the digital counterpart of analog [amplitude modulation](/reference/amplitude-modulation/):
where AM varies amplitude continuously with a message, ASK snaps it to a small set of
levels chosen by the data. The two-level case — carrier on or off — is
[on-off keying](/reference/on-off-keying/), by far the most common form.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A carrier whose envelope steps between three amplitude levels representing multi-level amplitude-shift keying." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="80" x2="440" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M30 80 q5 -8 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path d="M130 80 q5 -30 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path d="M230 80 q5 -18 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <path d="M330 80 q5 -30 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="20" y="122" font-size="9" fill="currentColor">each symbol picks a carrier amplitude (here low / high / mid / high)</text>
</svg>
<figcaption>ASK maps symbols to carrier amplitude levels; binary ASK (on/off) is on-off keying, and more levels pack more bits per symbol.</figcaption>
</figure>

## How it works

The transmitter multiplies the carrier by a data-dependent gain, producing bursts of
different amplitude. A receiver recovers the bits with an **envelope detector** — it
takes the magnitude of the signal and thresholds it — which needs no carrier-phase
recovery, so ASK demodulators are simple and cheap. With *M* amplitude levels, each
symbol carries log₂*M* bits; four levels give 2 bits per symbol, and so on.

The weakness of ASK is that amplitude is exactly what noise, fading, and gain drift
attack most directly. A fluctuating channel moves the received levels around, so the
decision thresholds must adapt, and multi-level ASK needs a good signal-to-noise ratio
to keep the levels distinguishable. Because the information rides on amplitude, ASK also
demands a **linear** transmit amplifier, unlike constant-envelope
[FSK](/reference/frequency-shift-keying/), which tolerates efficient saturated
amplifiers. These constraints keep pure ASK confined to short, benign links.

## Variants

Binary ASK is [OOK](/reference/on-off-keying/). Combining amplitude and phase into a
two-dimensional grid produces [quadrature amplitude modulation](/reference/quadrature-amplitude-modulation/),
which is really the joint amplitude/phase generalisation of ASK and
[PSK](/reference/phase-shift-keying/) and is what carries high data rates in Wi-Fi, DVB,
and cellular. So while pure multi-level ASK is rare on the air, its amplitude dimension
lives on inside QAM.

## Relevance to SDR

Most ASK you will meet in software radio is on-off keying in the license-free ISM bands:
RFID tags, remote controls, wireless sensors, and simple telemetry links. Amplitude
keying also appears in optical and infrared links, where turning a light source on and
off is the natural channel. On a waterfall, ASK bursts show up as amplitude-modulated
stripes whose brightness — not position — carries the data.

Demodulating ASK in an SDR is a matter of computing the [IQ](/reference/iq-data/)
magnitude and slicing it against an adaptive threshold; no [Costas
loop](/reference/costas-loop/) or symbol-phase tracking is required. GopherTrunk targets
trunked land-mobile systems that use frequency- and phase-keyed modulations, so it does
not decode ASK telemetry itself; ASK is documented here because it anchors the
amplitude branch of the modulation family and is a common first signal on the same
low-cost dongles GopherTrunk runs on.

## Sources

[^wiki]: [Amplitude-shift keying](https://en.wikipedia.org/wiki/Amplitude-shift_keying) — Wikipedia, for the definition, the on-off-keying special case, and multi-level ASK.
