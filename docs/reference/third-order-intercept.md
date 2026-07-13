---
slug: third-order-intercept
title: Third-order intercept (IP3)
entry_type: term
category: rf-metrics
description: The third-order intercept is the extrapolated power where a device's third-order intermodulation product would equal the wanted signal; it grades RF linearity.
keywords: third-order intercept, IP3, IIP3, OIP3, intercept point, intermodulation, IMD, linearity, two-tone test, TOI
aka: [IP3, IIP3, OIP3, TOI, third-order intercept point]
autolink: true
infobox:
  - { label: Symbol, value: "IP3 (IIP3 / OIP3)" }
  - { label: Unit, value: dBm }
  - { label: Grades, value: RF front-end linearity }
see_also: [intermodulation, 1-db-compression-point, spurious-free-dynamic-range, dynamic-range, low-noise-amplifier, mixer-rf]
cite_urls:
  - https://en.wikipedia.org/wiki/Third-order_intercept_point
  - https://en.wikipedia.org/wiki/Intermodulation
---

**Third-order intercept** (**IP3**, or **TOI**) is a theoretical power level — found
by extrapolation, never actually reached — at which a device's third-order
[intermodulation](/reference/intermodulation/) products would rise to equal the
wanted signal.[^wiki] It is the standard single-number grade of an amplifier's or
mixer's **linearity**: the higher the IP3, the more strong-signal punishment the
device tolerates before it manufactures troublesome spurs. IP3 directly sets a
receiver's [spurious-free dynamic range](/reference/spurious-free-dynamic-range/) and
is a key term in every strong-signal-handling specification.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 185" role="img" aria-label="A log-log plot of output power versus input power showing the fundamental output rising at slope one and the third-order product rising at slope three, both extrapolated to meet at the third-order intercept point, well above the region where gain compression actually occurs." xmlns="http://www.w3.org/2000/svg">
  <line x1="55" y1="18" x2="55" y2="160" stroke="currentColor" stroke-width="1.3"/>
  <line x1="55" y1="160" x2="440" y2="160" stroke="currentColor" stroke-width="1.3"/>
  <text x="48" y="16" text-anchor="end" font-size="9" fill="currentColor">P_out</text>
  <text x="438" y="176" text-anchor="end" font-size="9" fill="currentColor">P_in</text>
  <line x1="55" y1="150" x2="360" y2="30" stroke="currentColor" stroke-width="1.6"/>
  <text x="300" y="48" font-size="9" fill="currentColor">fundamental (slope 1)</text>
  <path d="M55 150 Q300 55 340 52" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/>
  <text x="215" y="78" font-size="8.5" fill="currentColor" fill-opacity="0.8">actual (compresses)</text>
  <line x1="120" y1="160" x2="360" y2="30" stroke="currentColor" stroke-width="1.4" stroke-dasharray="5 3"/>
  <text x="140" y="150" font-size="9" fill="currentColor">3rd-order (slope 3)</text>
  <circle cx="360" cy="30" r="4" fill="currentColor"/>
  <text x="366" y="27" font-size="10" fill="currentColor">IP3</text>
  <line x1="360" y1="30" x2="360" y2="160" stroke="currentColor" stroke-width="0.9" stroke-dasharray="2 2"/>
  <text x="360" y="174" text-anchor="middle" font-size="9" fill="currentColor">IIP3</text>
</svg>
<figcaption>Extrapolating the slope-1 fundamental and slope-3 intermod lines to their crossing gives IP3; the input-referred value is IIP3, the output-referred value OIP3. The intercept lies above where the device actually compresses.</figcaption>
</figure>

## How it works

A weak non-linearity in an amplifier or [mixer](/reference/mixer-rf/) can be written
as a power series; the cubic term is responsible for third-order products. Feed two
equal tones f₁ and f₂ and the cubic term produces spurs at 2f₁ − f₂ and 2f₂ − f₁,
close enough to the wanted signals that no filter removes them. Their growth rate is
the whole point:

- The **fundamental** output rises 1 dB per dB of input (slope 1).
- The **third-order product** rises 3 dB per dB of input (slope 3).

Two lines of different slope must eventually cross. Extend the measured slope-1 and
slope-3 lines and their intersection is the **third-order intercept point**. Referred
to the input it is **IIP3**; referred to the output, **OIP3**; they differ by the
device gain (OIP3 = IIP3 + G). The device never actually operates there — it
compresses and saturates long before — but the extrapolated intercept is a clean,
gain-independent linearity figure that lets you *predict* the spur level at any real
operating power:

**IMD3 level below each tone (dBc) = 2·(IIP3 − P_in)**

Back off the input 10 dB and the third-order spurs drop 30 dB — a 3:1 payoff that is
the engineer's main lever against intermod.

## Variants

- **IIP3 vs OIP3** — same intercept, referenced to input or output. Receivers usually
  quote **IIP3** (how strong an input they tolerate); power amplifiers often quote
  **OIP3**.
- **IP3 vs the [1 dB compression point](/reference/1-db-compression-point/)** — both
  measure linearity, but IP3 comes from a two-tone intermod test while P1dB comes
  from a single-tone gain test. As a rough rule of thumb IP3 sits about 10–15 dB
  above P1dB for many amplifiers, though it is always better to measure than assume.

## Relevance to SDR

IP3 is the first thing to check when picking an LNA or evaluating an SDR for a busy
RF environment. A [low-noise amplifier](/reference/low-noise-amplifier/) with a great
noise figure but poor IP3 will improve weak-signal reception in a quiet band yet make
things *worse* near strong transmitters, because it generates intermod that the bare
receiver would not. The best front-end choice balances low noise figure (bottom of
the dynamic range) against high IP3 (top of it). This is why serious scanner and
SDR users add high-IP3 amplifiers and, crucially, front-end filtering to strip strong
out-of-band signals before they can drive the non-linearity.

GopherTrunk is a decoder and cannot see or undo intermodulation once it is in the
samples — a 2f₁ − f₂ spur looks like a real carrier to any DSP. When decode quality
collapses only in the presence of strong nearby signals, IP3 (and the SFDR it sets)
is the limitation, addressed upstream with better front-end linearity, filtering, or
reduced gain, not in software.

## Sources

[^wiki]: [Third-order intercept point](https://en.wikipedia.org/wiki/Third-order_intercept_point) — Wikipedia, definition of IP3/IIP3/OIP3 and the slope-1/slope-3 extrapolation.
