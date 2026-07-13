---
slug: frequency-counter
title: Frequency counter
entry_type: hardware
category: test-equipment
description: "A frequency counter measures the frequency of a periodic signal by gated counting, with accuracy set by its timebase reference oscillator."
keywords: frequency counter, gated counting, reciprocal counter, timebase, gate time, frequency measurement, TCXO OCXO reference, resolution, RF test equipment
aka: [frequency counter, freq counter, counter-timer]
autolink: true
infobox:
  - { label: Type, value: RF measurement instrument }
  - { label: Measures, value: "Frequency (Hz) of a periodic signal" }
  - { label: Method, value: "Gated / reciprocal counting" }
  - { label: Accuracy set by, value: "Timebase (TCXO / OCXO / GPSDO)" }
  - { label: TX, value: "No (measures applied signal)" }
  - { label: Typical price, value: "$30 – $10,000+" }
see_also: [frequency-stability, gpsdo, ocxo, tcxo, ppm-frequency-correction, frequency]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_counter
  - https://en.wikipedia.org/wiki/Frequency_standard
---

**A frequency counter** measures the [frequency](/reference/frequency/) of a periodic
electrical signal by counting cycles over a precisely known interval.[^wiki] It is the
instrument you use to verify that a transmitter, oscillator, or reference is exactly on
frequency — and its own accuracy is set entirely by the quality of its internal
**timebase**, which is why [frequency-stability](/reference/frequency-stability/) is the
spec that matters most.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A frequency counter diagram: an input signal and a gate derived from a timebase reference both feed a digital counter, which divides the cycle count by the gate time to display a frequency." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="45" y="42" font-size="8" fill="currentColor" text-anchor="middle">Input</text>
  <path d="M20 62 h8 v-10 h8 v20 h8 v-10 h8" fill="none" stroke="currentColor"/>
  <rect x="150" y="42" width="90" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="195" y="63" font-size="8" fill="currentColor" text-anchor="middle">Gate / count</text>
  <line x1="60" y1="58" x2="150" y2="58" stroke="currentColor" marker-end="url(#fcar)"/>
  <rect x="150" y="100" width="90" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="195" y="119" font-size="8" fill="currentColor" text-anchor="middle">Timebase</text>
  <line x1="195" y1="100" x2="195" y2="76" stroke="currentColor" marker-end="url(#fcar)"/>
  <text x="90" y="118" font-size="7" fill="currentColor" text-anchor="middle">TCXO/OCXO/GPSDO</text>
  <line x1="150" y1="115" x2="120" y2="115" stroke="currentColor" stroke-opacity="0.5"/>
  <rect x="300" y="46" width="140" height="42" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor"/>
  <text x="370" y="72" font-size="12" fill="currentColor" text-anchor="middle">146.520 000 MHz</text>
  <line x1="240" y1="59" x2="300" y2="63" stroke="currentColor" marker-end="url(#fcar)"/>
</svg>
<figcaption>A frequency counter counts input cycles during a gate interval set by its timebase; the count divided by the gate time is the displayed frequency, so timebase accuracy bounds the whole measurement.</figcaption>
</figure>

## How it works

The classic **direct (gated) counter** opens a gate for a fixed, timebase-derived interval
— the *gate time*, say 1 second — and counts how many input cycles pass through. Cycles
divided by gate time is the frequency. Its resolution is one count per gate: a 1-second
gate resolves 1 Hz, and a longer gate resolves finer but takes longer. At low input
frequencies this is coarse, because few cycles occur per gate.

A **reciprocal counter** inverts the scheme: it measures the *period* by counting
high-speed timebase clock ticks over an integer number of input cycles, then takes the
reciprocal. This gives constant relative resolution — the same number of significant
digits — regardless of input frequency, so it resolves a 100 Hz signal as finely as a
100 MHz one for a given gate. Nearly all modern counters are reciprocal, often with
interpolation that adds several more digits.

## Accuracy and the timebase

A counter can never be more accurate than its **timebase reference**, because every
measurement is a ratio against it. The hierarchy runs:

- **TCXO** ([temperature-compensated crystal](/reference/tcxo/)) — a few
  [ppm](/reference/ppm-frequency-correction/); adequate for casual work.
- **OCXO** ([oven-controlled crystal](/reference/ocxo/)) — tens of ppb, far more stable
  over temperature and time.
- **GPSDO** ([GPS-disciplined oscillator](/reference/gpsdo/)) — locked to atomic time from
  the GNSS constellation, giving parts in 10¹¹ or better long-term. Many counters accept an
  external 10 MHz reference precisely so they can be slaved to a GPSDO.

Aside from the timebase, the dominant short-term error is **±1 count** quantization plus
trigger noise, both of which shrink with a longer gate or reciprocal interpolation.

## In practice

- Use the **longest gate you can tolerate** for the finest resolution on a stable signal.
- **Feed the counter a clean signal** at an appropriate level; a squaring/trigger stage
  needs a decent slew rate, and noise near the trigger point adds jitter.
- For real accuracy, **discipline the timebase to a 10 MHz GPSDO** and let an OCXO warm up
  and settle before trusting the last digits.

## Relevance to SDR

Every SDR carries a reference oscillator, and its accuracy directly offsets every tuned
frequency — the reason GopherTrunk and other SDR software expose a
[ppm frequency correction](/reference/ppm-frequency-correction/). A frequency counter, its
own timebase disciplined by a [GPSDO](/reference/gpsdo/), is the bench tool that lets you
*measure* an SDR's or transmitter's true frequency and derive that correction, or verify
that a control channel really sits where the system says it does. GopherTrunk does not
drive a counter; it estimates and corrects residual frequency offset in software from the
recovered carrier, and a bench counter is a complementary external reference rather than
part of the decode chain.

## Sources

[^wiki]: [Frequency counter](https://en.wikipedia.org/wiki/Frequency_counter) — Wikipedia, on gated and reciprocal frequency counting and timebase-limited accuracy.
