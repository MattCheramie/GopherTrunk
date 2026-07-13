---
slug: dbm
title: dBm
entry_type: term
category: rf-fundamentals
description: dBm is power expressed in decibels relative to one milliwatt, giving an absolute signal-strength figure; received radio signals are negative dBm values.
keywords: dBm, decibel milliwatt, absolute power, received signal strength, RSSI, dBW
aka: [dBm]
autolink: true
infobox:
  - { label: Type, value: Absolute power unit }
  - { label: Reference, value: 1 milliwatt }
  - { label: Examples, value: "0 dBm = 1 mW; −80 dBm ≈ solid signal" }
see_also: [decibel, dbfs, noise-floor, signal-to-noise-ratio, receiver-sensitivity, link-budget, thermal-noise]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/DBm
  - https://en.wikipedia.org/wiki/Received_signal_strength_indication
---

**dBm** is power expressed in [decibels](/reference/decibel/) relative to **one
milliwatt**, making it an *absolute* measure of signal strength rather than a mere
ratio.[^wiki] The formula is *P(dBm) = 10·log₁₀(P/1 mW)*, so 0 dBm equals 1 mW, +30 dBm
is 1 watt, and −30 dBm is one microwatt.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A dBm scale showing reference points from +30 dBm (1 watt) down to -120 dBm, with received signals in the negative range near the noise floor." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.5"/>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="80" y1="54" x2="80" y2="66" stroke="currentColor"/><text x="80" y="80">+30</text><text x="80" y="44">1 W</text>
    <line x1="170" y1="54" x2="170" y2="66" stroke="currentColor"/><text x="170" y="80">0</text><text x="170" y="44">1 mW</text>
    <line x1="280" y1="54" x2="280" y2="66" stroke="currentColor"/><text x="280" y="80">-80</text><text x="280" y="44">strong RX</text>
    <line x1="400" y1="54" x2="400" y2="66" stroke="currentColor"/><text x="400" y="80">-120 dBm</text><text x="400" y="44">in the noise</text>
  </g>
  <text x="240" y="110" text-anchor="middle" font-size="9" fill="currentColor">each 10 dB step is a factor of ten in power</text>
</svg>
<figcaption>dBm is absolute power referenced to 1 mW. Received signals are negative; closer to zero is stronger.</figcaption>
</figure>

## How it works

Because dBm carries a fixed reference (1 mW), a single dBm figure names a real power
level, while a plain [decibel](/reference/decibel/) figure only names a ratio. The two
combine naturally: adding a gain in dB to a level in dBm gives a new level in dBm. A
+20 dB amplifier turns a −80 dBm input into a −60 dBm output; the units bookkeep
themselves.

Received radio signals are tiny fractions of a milliwatt, so they land as **negative**
dBm values, and the one closer to zero is stronger. −70 dBm beats −90 dBm by 20 dB,
which is 100× in power. This ordering trips up newcomers who read "−90" as bigger than
"−70"; in dBm, less negative always means more power.

A few landmarks build intuition. A handheld transmitter is +30 to +37 dBm (1–5 W). A
usable off-air signal at a scanner might be −60 to −90 dBm. The
[thermal noise](/reference/thermal-noise/) floor in a narrow channel sits around
−120 dBm, and a good receiver can pull signals out just a few dB above it. That span —
roughly +37 dBm to −127 dBm — is more than 16 orders of magnitude in raw power, which is
exactly why the logarithmic unit exists.

dBm is a **power** unit, so it uses the 10·log₁₀ form. It should not be confused with
[dBFS](/reference/dbfs/), which references digital full scale and lives entirely inside
the converter, with no fixed relationship to dBm until the analog gain of the front end
is known.

## In practice

Most receivers report a proxy for dBm called RSSI (received signal strength
indicator).[^rssi] True dBm requires a calibrated front end; many SDR dongles instead
give an *uncalibrated* number that tracks relative changes but is offset from absolute
power by an unknown amount. That is usually fine — what matters for decoding is the gap
between signal and [noise floor](/reference/noise-floor/), the
[SNR](/reference/signal-to-noise-ratio/), which is a difference of two dBm-like readings
and cancels a constant offset.

dBm is the working currency of a [link budget](/reference/link-budget/): start at the
transmitter's dBm, add and subtract gains and losses in dB, and compare the arriving
dBm against the [receiver sensitivity](/reference/receiver-sensitivity/) threshold, also
quoted in dBm.

## Relevance to SDR

Receiver meters, waterfall scales, and site-survey tools report signal and
[noise-floor](/reference/noise-floor/) levels in dBm (or an RSSI proxy). Their difference
is the [SNR](/reference/signal-to-noise-ratio/) that determines whether a channel
decodes. GopherTrunk works primarily with SNR and demod-quality figures rather than
absolute dBm, because most supported SDR hardware is not power-calibrated — but the
mental model of "signal in dBm, noise in dBm, decode if the gap is wide enough" is
exactly right.

## Sources

[^wiki]: [dBm](https://en.wikipedia.org/wiki/DBm) — Wikipedia, power referenced to one milliwatt and the resulting negative values for received signals.
[^rssi]: [Received signal strength indication](https://en.wikipedia.org/wiki/Received_signal_strength_indication) — Wikipedia, how receivers report signal strength and why RSSI is often uncalibrated relative to true dBm.
