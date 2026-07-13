---
slug: on-off-keying
title: On-off keying (OOK)
entry_type: technology
category: modulation
description: On-off keying (OOK) is the simplest amplitude-shift keying, switching a carrier fully on or off to send bits; used in key fobs, TPMS, and garage remotes.
keywords: OOK, on-off keying, ASK, amplitude shift keying, carrier present, RKE, key fob, TPMS, garage remote, ISM band
aka: [on-off keying, OOK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (binary ASK) }
  - { label: Idea, value: Carrier on = 1, off = 0 }
  - { label: Used by, value: RKE fobs, TPMS, garage remotes }
see_also: [amplitude-shift-keying, amplitude-modulation, remote-keyless-entry, tpms, carrier-wave, morse-code]
cite_urls:
  - https://en.wikipedia.org/wiki/On%E2%80%93off_keying
  - https://en.wikipedia.org/wiki/Amplitude-shift_keying
---

**On-off keying** (**OOK**) is the simplest form of [amplitude-shift
keying](/reference/amplitude-shift-keying/): a binary scheme that transmits data by
switching a [carrier](/reference/carrier-wave/) fully on to send a one and off to send a
zero.[^wiki] Because the presence or absence of RF energy carries the bit, OOK needs no
phase reference and only a trivial transmitter, which is why it dominates cheap
short-range devices in the ISM bands.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A bit stream of ones and zeros above a carrier that oscillates during ones and goes flat during zeros, showing on-off keying." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor" font-family="monospace"><text x="45" y="22">1</text><text x="145" y="22">0</text><text x="245" y="22">1</text><text x="345" y="22">1</text></g>
  <line x1="20" y1="80" x2="440" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M30 80 q5 -24 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0
           M230 80 q5 -24 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0
           M330 80 q5 -24 10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0 t10 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="130" y1="80" x2="230" y2="80" stroke="currentColor" stroke-width="1.5"/>
  <text x="20" y="122" font-size="9" fill="currentColor">carrier fully on for a 1, fully off (flat) for a 0</text>
</svg>
<figcaption>OOK toggles the carrier fully on or off; the receiver only has to detect whether energy is present.</figcaption>
</figure>

## How it works

An OOK transmitter gates the carrier with the data stream, typically by keying a simple
oscillator on and off. The receiver does not need to recover carrier phase or even
frequency precisely; it only measures envelope power and thresholds it. A basic
non-coherent detector rectifies and low-pass filters the signal — an "envelope
detector" — then compares the result against a decision level to recover each bit. This
makes OOK receivers as cheap as a superregenerative stage or a single detector diode,
and lets them run on a coin cell.

The trade-off is efficiency and robustness. Because the transmitter is silent during
zeros, average power depends on the data pattern, and a fixed threshold drifts as
signal strength changes. Practical OOK protocols therefore use **DC-balanced line
codes** — often [Manchester coding](/reference/manchester-coding/) or pulse-width
symbols — so that every bit contains a transition and the receiver can track a running
average for its slicing threshold. OOK is also spectrally wasteful and sensitive to
narrowband interference, since a burst of noise in an "off" interval can be mistaken for
an "on."

## Variants

OOK is the degenerate two-level case of [ASK](/reference/amplitude-shift-keying/); using
more amplitude levels yields multi-level ASK. Related pulse schemes encode data in
timing rather than a fixed on/off grid: **pulse-width** and **pulse-position** variants
are common in remote-control chips because they tolerate loose timing references. Morse
[continuous wave](/reference/continuous-wave/) telegraphy is historically the original
OOK — a hand key switching the transmitter on and off.

## Relevance to SDR

OOK is one of the first signals a newcomer decodes with a cheap receiver, because the
demodulation is so forgiving. A huge population of consumer devices in the 315 MHz,
433.92 MHz, 868 MHz, and 915 MHz ISM bands use it:
[remote keyless entry](/reference/remote-keyless-entry/) fobs, garage-door and gate
openers, [tire-pressure monitoring](/reference/tpms/) sensors (often mixing OOK with
FSK), wireless doorbells, weather stations, and many home-automation remotes. On a
waterfall these appear as short, bursty vertical stripes that blink on and off with the
data.

Decoding OOK in software usually means taking the magnitude of the [IQ](/reference/iq-data/)
stream, thresholding it, and measuring pulse widths to recover the underlying symbols —
the approach used by generic tools like the `rtl_433` family. GopherTrunk is a trunked
land-mobile scanner focused on P25, DMR, NXDN, and TETRA, so OOK telemetry devices fall
outside its decode chain; OOK is covered here as background for the amplitude-modulation
family that GopherTrunk's users frequently encounter on the same hardware.

## Sources

[^wiki]: [On–off keying](https://en.wikipedia.org/wiki/On%E2%80%93off_keying) — Wikipedia, for the definition of OOK as binary ASK and its use in low-cost ISM-band devices.
