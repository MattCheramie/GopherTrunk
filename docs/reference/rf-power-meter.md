---
slug: rf-power-meter
title: RF power meter
entry_type: hardware
category: test-equipment
description: "An RF power meter measures the absolute power of a radio-frequency signal, reporting it in dBm or watts using a diode, thermocouple, or thermistor sensor."
keywords: RF power meter, power sensor, dBm measurement, diode detector, thermocouple sensor, thermistor bolometer, true average power, peak power, RF test equipment
aka: [RF power meter, power meter, power sensor]
autolink: true
infobox:
  - { label: Type, value: RF measurement instrument }
  - { label: Measures, value: "Absolute power (dBm / W)" }
  - { label: Sensor types, value: "Diode / thermocouple / thermistor" }
  - { label: Key spec, value: "Frequency range, dynamic range" }
  - { label: TX, value: "No (measures applied power)" }
  - { label: Typical price, value: "$30 – $10,000+" }
see_also: [dbm, decibel, power-amplifier, dummy-load, attenuator, crest-factor-papr]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_measurement
  - https://www.keysight.com/us/en/assets/7018-01310/application-notes/5988-9213.pdf
---

**An RF power meter** measures the absolute power of a radio-frequency signal and reports
it in [dBm](/reference/dbm/) or watts.[^wiki] Where a
[spectrum analyzer](/reference/spectrum-analyzer/) shows how power is distributed *across*
frequency, a power meter answers a simpler, more accurate question: *how much total power
is in this signal?* — the measurement you need to set a transmitter's output, verify an
amplifier's gain in [decibels](/reference/decibel/), or characterize a
[dummy load](/reference/dummy-load/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An RF power meter block diagram: an RF input connector feeds a sensor head containing a diode, thermocouple, or thermistor, which drives a meter reading in dBm." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="45" cy="70" r="9" fill="none" stroke="currentColor"/>
  <text x="45" y="98" font-size="8" fill="currentColor" text-anchor="middle">RF in</text>
  <rect x="110" y="50" width="110" height="42" rx="4" fill="none" stroke="currentColor"/>
  <text x="165" y="68" font-size="8" fill="currentColor" text-anchor="middle">Sensor head</text>
  <text x="165" y="82" font-size="8" fill="currentColor" text-anchor="middle">(detector)</text>
  <line x1="54" y1="70" x2="110" y2="70" stroke="currentColor" marker-end="url(#pmar)"/>
  <rect x="280" y="45" width="140" height="52" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor"/>
  <text x="350" y="68" font-size="12" fill="currentColor" text-anchor="middle">+37.0 dBm</text>
  <text x="350" y="84" font-size="8" fill="currentColor" text-anchor="middle">(≈ 5 W)</text>
  <line x1="220" y1="71" x2="280" y2="71" stroke="currentColor" marker-end="url(#pmar)"/>
</svg>
<figcaption>An RF power meter: a calibrated sensor head converts applied RF power to a DC signal, which the meter displays as an absolute reading in dBm (and watts).</figcaption>
</figure>

## How it works

A power meter is a **sensor head plus a display unit**. The sensor converts RF power to a
proportional DC voltage; the meter applies calibration factors and shows the result. Three
sensor technologies dominate:

- **Thermocouple / thermistor (thermal) sensors** heat a load with the RF and measure the
  temperature rise. They respond to *true average power regardless of waveform*, are very
  accurate, and are the reference standard — but they are slower and less sensitive.
- **Diode detectors** rectify the RF. In their square-law region they, too, read true
  power; driven harder they approach peak detection. Diode sensors are fast and sensitive
  (reading to −70 dBm or lower), which makes them the basis of most inexpensive and
  wideband meters, but they need correction to stay accurate on high-
  [crest-factor](/reference/crest-factor-papr/) modulated signals.

Modern **wideband/peak power sensors** digitize the detected envelope, so they report
average, peak, and even a full power-versus-time trace for burst
[TDMA](/reference/tdma/) waveforms — important because a simple average reading understates
the peak power of a bursty or high-PAPR signal.

## In practice

- **Never exceed the sensor's damage level.** Sensors are low-power devices; measure a
  transmitter through a calibrated [attenuator](/reference/attenuator/) or a directional
  coupler, not by connecting the sensor straight to a PA.
- **Match the sensor to the waveform.** Use a true-average (thermal) or corrected wideband
  sensor for modulated signals; a plain diode reading can err on high-PAPR waveforms.
- **Mind frequency range and cal factors.** Each sensor is calibrated per frequency;
  applying the right cal factor is what turns a raw reading into an accurate dBm value.
- **Relative vs. absolute.** For a quick gain check you need only a consistent relative
  reading; for compliance or link budgets you need traceable absolute accuracy.

## Relevance to SDR

RF power meters live on the *transmit* side, so they matter less to receive-only SDR
scanning than a [spectrum analyzer](/reference/spectrum-analyzer/) or
[VNA](/reference/vector-network-analyzer/) do — an SDR reports level in relative
[dBFS](/reference/dbfs/), not calibrated dBm. Where a power meter does help the SDR
enthusiast is on the bench: setting the output of a
[signal generator](/reference/signal-generator/) or the transmit level of a
[HackRF](/reference/hackrf/)/[LimeSDR](/reference/limesdr/), verifying amplifier gain, and
confirming a [dummy load](/reference/dummy-load/) is dissipating the expected power.
GopherTrunk is a receiver and neither measures nor generates absolute power; a power meter
is a general RF-bench aid, not part of its decode path.

## Sources

[^wiki]: [Power measurement](https://en.wikipedia.org/wiki/Power_measurement) — Wikipedia, on RF/microwave power measurement and thermal versus diode sensors.
