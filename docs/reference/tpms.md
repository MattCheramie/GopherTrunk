---
slug: tpms
title: TPMS (Tire Pressure Monitoring)
entry_type: protocol
category: wireless-data-iot
description: "TPMS sensors report tire pressure and temperature over short 315/433 MHz OOK or FSK bursts carrying a sensor ID; the frames are receivable with common SDR tools."
keywords: TPMS, tire pressure monitoring, 315 MHz, 433 MHz, OOK, ASK, FSK, Manchester, sensor ID, direct TPMS, rtl_433, wheel sensor
aka: [TPMS, "tire pressure monitoring system"]
autolink: true
infobox:
  - { label: Type, value: In-vehicle tire sensor telemetry }
  - { label: Bands, value: "315 MHz (US), 433.92 MHz (EU) ISM" }
  - { label: Modulation, value: OOK/ASK or FSK, often Manchester-coded }
  - { label: Payload, value: Sensor ID + pressure + temperature + flags }
  - { label: Access, value: Unslotted, sensor-initiated bursts }
  - { label: GopherTrunk support, value: Not decoded (receivable with other SDR tools) }
see_also: [on-off-keying, frequency-shift-keying, remote-keyless-entry, amplitude-shift-keying, software-defined-radio, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Tire-pressure_monitoring_system
  - https://en.wikipedia.org/wiki/ISM_radio_band
---

**TPMS** (tire-pressure monitoring system) is the short-range radio link by which
**direct** tire sensors report each wheel's pressure and temperature to the car.[^wiki] A
battery-powered sensor inside each tire wakes periodically (or on pressure change / rotation)
and transmits a brief burst on a sub-GHz ISM band — **315 MHz** in North America, **433.92
MHz** in Europe — carrying a unique sensor ID plus the measured values. Because these bursts
are simple OOK/FSK packets in the clear, they are among the easiest signals to receive with
a general-purpose [SDR](/reference/software-defined-radio/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Four tire sensors on a car each transmit a short radio burst to a receiver in the vehicle body; a magnified burst shows an on-off-keyed packet of ID, pressure, and temperature." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tp_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <circle cx="55" cy="40" r="10" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
    <circle cx="55" cy="110" r="10" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
    <circle cx="150" cy="40" r="10" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
    <circle cx="150" cy="110" r="10" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
    <rect x="90" y="60" width="30" height="30" fill="none" stroke="currentColor"/><text x="105" y="80" font-size="8">ECU</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-opacity="0.8">
    <line x1="64" y1="46" x2="88" y2="64" marker-end="url(#tp_ar)"/>
    <line x1="64" y1="104" x2="88" y2="86" marker-end="url(#tp_ar)"/>
    <line x1="141" y1="46" x2="121" y2="64" marker-end="url(#tp_ar)"/>
    <line x1="141" y1="104" x2="121" y2="86" marker-end="url(#tp_ar)"/>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <path d="M210 90 L230 90 L230 55 L245 55 L245 90 L260 90 L260 55 L268 55 L268 90 L285 90 L285 55 L300 55 L300 90 L320 90 L320 55 L330 55 L330 90 L350 90"/>
  </g>
  <text x="285" y="112" text-anchor="middle" font-size="8.5" fill="currentColor">OOK burst: preamble · ID · pressure · temperature · CRC</text>
</svg>
<figcaption>Each tire sensor sends a short OOK/FSK burst carrying its ID, pressure, and temperature to the vehicle's receiver.</figcaption>
</figure>

## Overview

Direct TPMS puts a small sensor module in each wheel, powered by a long-life battery. It
transmits infrequently to conserve power, so the receiver in the car body listens
continuously and matches incoming IDs to learned wheel positions. (Indirect TPMS, by
contrast, uses no radio at all — it infers low pressure from wheel-speed differences via
ABS sensors.)

## Technical characteristics

| Property | Value |
|----------|-------|
| Bands | 315 MHz (US), 433.92 MHz (EU) ISM |
| Modulation | [OOK](/reference/on-off-keying/)/ASK or [FSK](/reference/frequency-shift-keying/) |
| Line coding | Commonly Manchester or differential Manchester |
| Payload | 32-bit sensor ID, pressure, temperature, status/battery |
| Integrity | Checksum or CRC per protocol |
| Trigger | Periodic, motion, or a 125 kHz LF activation tone |

Formats are vendor-specific (Schrader, Continental, Pacific, and others), so decoders keep a
library of per-manufacturer framings.

## History

Direct TPMS spread after regulations mandated pressure monitoring — notably the US TREAD
Act, which required systems on new light vehicles from the mid-2000s, with the EU following
for new cars around 2014.[^ism] Those mandates made 315/433 MHz TPMS bursts one of the most
common signals on the road.

## Deployment

Essentially every modern passenger vehicle with direct TPMS emits these bursts, making them
a familiar sight for hobbyists surveying the 315/433 MHz ISM bands alongside
[remote keyless entry](/reference/remote-keyless-entry/) and other short-range devices.

## Decoding it with GopherTrunk

GopherTrunk does not decode TPMS — it targets trunked land-mobile voice, not vehicle
telemetry. TPMS is, however, genuinely *receivable with general SDR tools*: an
[RTL-SDR](/reference/rtl-sdr/) tuned to 315 or 433.92 MHz with a decoder such as `rtl_433`
will print sensor IDs, pressures, and temperatures directly, since the frames are typically
unencrypted. That capability lives in those tools rather than in GopherTrunk.

## Sources

[^wiki]: [Tire-pressure monitoring system](https://en.wikipedia.org/wiki/Tire-pressure_monitoring_system) — Wikipedia, for direct vs indirect TPMS, the sensor bursts, and payload contents.
[^ism]: [ISM radio band](https://en.wikipedia.org/wiki/ISM_radio_band) — Wikipedia, for the 315/433 MHz bands TPMS shares with other short-range devices.
