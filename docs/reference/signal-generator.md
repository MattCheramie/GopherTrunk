---
slug: signal-generator
title: Signal generator
entry_type: hardware
category: test-equipment
description: "A signal generator is a bench instrument that produces a known RF signal — a pure carrier or a modulated waveform — as a reference source for testing receivers and RF hardware."
keywords: signal generator, RF signal generator, vector signal generator, VSG, arbitrary waveform generator, analog signal generator, CW source, reference source, test equipment
aka: [sig gen, RF source, VSG]
autolink: true
infobox:
  - { label: Type, value: RF source instrument }
  - { label: Produces, value: "CW / modulated RF" }
  - { label: Variants, value: "Analog / vector (VSG)" }
  - { label: Key specs, value: "Level accuracy, phase noise" }
  - { label: TX, value: "Yes (bench source, not a radio)" }
  - { label: Typical price, value: "$300 – $100,000+" }
see_also: [local-oscillator, phase-noise, spectrum-analyzer, dbm, frequency-stability, iq-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Signal_generator
  - https://www.keysight.com/us/en/assets/7018-01310/application-notes/5989-7833.pdf
---

**A signal generator** is a bench instrument that synthesizes a known, controlled RF
signal to serve as a reference stimulus[^wiki] — a pure carrier at a chosen frequency and
[power level](/reference/dbm/), optionally carrying a defined modulation. Where a
[spectrum analyzer](/reference/spectrum-analyzer/) *measures* signals, a signal generator
*creates* them, making it the essential source for testing receiver sensitivity,
calibrating gain, and exercising a decoder with a repeatable input.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Block diagram of a signal generator: a frequency synthesizer feeds a modulator, whose output passes through a level-control attenuator to the RF output connector, with an I and Q baseband input driving the modulator." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="90" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="65" y="76" font-size="9" fill="currentColor" text-anchor="middle">Synthesizer</text>
  <rect x="150" y="55" width="80" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="190" y="76" font-size="9" fill="currentColor" text-anchor="middle">Modulator</text>
  <rect x="270" y="55" width="90" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="315" y="72" font-size="8" fill="currentColor" text-anchor="middle">Level /</text>
  <text x="315" y="83" font-size="8" fill="currentColor" text-anchor="middle">attenuator</text>
  <circle cx="420" cy="72" r="9" fill="none" stroke="currentColor"/>
  <text x="420" y="105" font-size="8" fill="currentColor" text-anchor="middle">RF out</text>
  <line x1="110" y1="72" x2="150" y2="72" stroke="currentColor" marker-end="url(#sgar)"/>
  <line x1="230" y1="72" x2="270" y2="72" stroke="currentColor" marker-end="url(#sgar)"/>
  <line x1="360" y1="72" x2="411" y2="72" stroke="currentColor" marker-end="url(#sgar)"/>
  <text x="190" y="25" font-size="8" fill="currentColor" text-anchor="middle">I/Q baseband</text>
  <line x1="190" y1="30" x2="190" y2="55" stroke="currentColor" marker-end="url(#sgar)"/>
</svg>
<figcaption>A signal generator: a synthesizer sets the carrier frequency, a modulator imprints information, and a calibrated attenuator sets the exact output level.</figcaption>
</figure>

## Overview

The value of a signal generator lies in its **known, traceable outputs**. Frequency is
set by a precise [PLL](/reference/phase-locked-loop/) or
[DDS](/reference/dds-synthesizer/) synthesizer referenced to a stable
[TCXO](/reference/tcxo/) or [OCXO](/reference/ocxo/), so
[frequency stability](/reference/frequency-stability/) and low
[phase noise](/reference/phase-noise/) are headline specs. Output level is set by a
calibrated step attenuator, so a "−100 dBm" setting really is −100 dBm at the connector —
which is what lets you measure a [receiver's sensitivity](/reference/receiver-sensitivity/)
meaningfully.

## Variants

**Analog (RF) signal generators** produce a continuous-wave carrier and support basic
built-in modulation — AM, FM, phase, and pulse. They are the simplest and are ideal as a
clean [local-oscillator](/reference/local-oscillator/)-quality reference or for
sensitivity testing with a tone.

**Vector signal generators (VSGs)** add an
[I/Q modulator](/reference/iq-modulation/) fed by an internal arbitrary-waveform engine,
so they can synthesize essentially any digital modulation —
[QPSK](/reference/qpsk/), [C4FM](/reference/c4fm/),
[π/4-DQPSK](/reference/pi-4-dqpsk/), [OFDM](/reference/ofdm/), and full standard
waveforms such as LTE or Wi-Fi frames. A VSG can replay a captured
[IQ file](/reference/iq-data/), letting you generate a realistic P25 or DMR signal on the
bench with controlled level, noise, and impairments.

**Simpler sources** fill the low end: [DDS](/reference/dds-synthesizer/) modules and the
transmit side of an SDR like a [HackRF](/reference/hackrf/) or
[LimeSDR](/reference/limesdr/) can generate test signals, though without the calibrated
level accuracy and spectral purity of a bench instrument. A tracking generator paired
with a [spectrum analyzer](/reference/spectrum-analyzer/) or the
[tinySA](/reference/tinysa/) forms a low-cost scalar network-measurement setup.

## In practice

- **Sensitivity / SINAD testing** — inject a known low-level modulated signal and find the
  level at which the decoder or [SINAD](/reference/sinad/) meets a threshold.
- **Reference / LO substitution** — supply a clean carrier to stand in for a
  local oscillator when characterizing a mixer or filter.
- **Impairment injection** — a VSG can add controlled noise, frequency offset, or
  [phase noise](/reference/phase-noise/) to stress a demodulator's tracking loops.
- **Level discipline** — always mind the calibrated output range and never exceed a
  device-under-test's damage threshold; use external [attenuators](/reference/attenuator/)
  for very low levels below the generator's floor.

## Relevance to SDR

For SDR and trunking work, a signal generator — especially a vector one replaying a
captured IQ waveform — is how you exercise a decoder repeatably: feed a known-good P25 or
DMR signal at a controlled level to confirm the receive chain, then walk the level down to
measure sensitivity, or add impairments to probe where lock breaks. GopherTrunk itself is
a receiver and generates no RF; a bench signal generator (or the TX side of an SDR) is an
external aid used to validate the receive path rather than something GopherTrunk drives.

## Sources

[^wiki]: [Signal generator](https://en.wikipedia.org/wiki/Signal_generator) — Wikipedia, on analog and vector RF signal generators and their use as reference sources.
