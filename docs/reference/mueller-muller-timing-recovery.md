---
slug: mueller-muller-timing-recovery
title: Mueller–Müller timing recovery
entry_type: algorithm
category: synchronization
description: Mueller–Müller timing recovery is a decision-directed symbol-timing algorithm that estimates clock error from just one sample per symbol, making it efficient but carrier-phase sensitive.
keywords: Mueller-Muller, timing recovery, decision directed, symbol timing, timing error detector, TED, clock recovery, one sample per symbol
aka: [Mueller–Müller timing recovery, Mueller-Muller, M&M]
autolink: true
infobox:
  - { label: Type, value: Symbol-timing (TED) algorithm }
  - { label: Feature, value: One sample per symbol (decision-directed) }
  - { label: Use, value: AIS, APRS, paging demodulators }
see_also: [gardner-timing-recovery, early-late-gate, clock-recovery, symbol-rate, ais, aprs]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
related_reading:
  - { title: "SDR Internals, Part 7: Symbol timing & sync recovery", url: /blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Symbol_synchronization
  - https://ieeexplore.ieee.org/document/1092560
---

**Mueller–Müller timing recovery** is a *decision-directed* symbol-timing algorithm that
needs only **one sample per symbol**, making it the most computationally economical member
of the timing-recovery family.[^wiki] It reaches that low sample rate by using the
receiver's own symbol *decisions* — the sliced output — to build a timing-error estimate,
rather than oversampling the waveform.[^mm]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A symbol waveform sampled once per symbol at the strobe instants, with the current and previous decisions combined to form a timing error at a lower sample rate than Gardner." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 95 C 90 95 90 40 150 40 C 210 40 210 95 270 95 C 330 95 330 40 390 40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="150" cy="40" r="3.6"/><circle cx="270" cy="95" r="3.6"/></g>
  <line x1="150" y1="20" x2="150" y2="112" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <line x1="270" y1="20" x2="270" y2="112" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="150" y="16">strobe k-1</text><text x="270" y="16">strobe k</text><text x="150" y="126">dec[k-1]</text><text x="270" y="126">dec[k]</text></g>
  <text x="230" y="138" text-anchor="middle" font-size="8.5" fill="currentColor">e = dec[k-1]*x[k] - dec[k]*x[k-1]  (one sample/symbol)</text>
</svg>
<figcaption>Mueller–Müller timing recovery samples once per symbol and combines the current and previous decisions with the current and previous samples to form a timing error — half the sample rate of Gardner.</figcaption>
</figure>

## How it works

The Mueller–Müller **timing-error detector (TED)** cross-multiplies the current and previous
on-time samples with the previous and current *decisions*:

`e[k] = â[k−1] · x[k] − â[k] · x[k−1]`,

where `x[·]` are the received (strobe) samples and `â[·]` are the sliced symbol estimates.
When the strobe sits at the symbol centre, the pulse-shaped waveform's neighbour
contributions are symmetric and this difference averages to zero; a timing offset breaks
that symmetry and the detector produces a signed error. Because it consumes exactly one
sample per symbol, there is no midpoint sample to compute and no 2× oversampling to carry —
the loop filter and interpolator/NCO that follow are otherwise identical to any other
timing loop.

The economy comes with two conditions:

- **It is decision-directed, so it needs reliable decisions.** At start-up, before the loop
  has pulled in, decisions are noisy and the error is noisy with them; acquisition can be
  slower or need a coarse pre-lock. At usable SNR it settles cleanly.
- **It is carrier-phase sensitive.** Unlike Gardner, the M&M error depends on correctly
  sliced symbols, which requires the carrier to be reasonably de-rotated first. In practice
  the carrier loop and M&M timing loop are run together or the signal is a form (like FSK
  taken to a real soft-decision) where phase is not an issue.

## Variants and contrast

| Detector | Samples/symbol | Data-aided? | Carrier-phase sensitive? |
|---|---|---|---|
| [Gardner](/reference/gardner-timing-recovery/) | 2 | Non-data-aided | No |
| [Early-late gate](/reference/early-late-gate/) | ≥2 | Non-data-aided | Depends |
| Mueller–Müller | 1 | Decision-directed | Yes |

The trade is clear: [Gardner](/reference/gardner-timing-recovery/) and the
[early-late gate](/reference/early-late-gate/) buy carrier-phase independence with a second
sample per symbol, while Mueller–Müller trades that robustness for half the sample rate and
lower arithmetic cost. Choose M&M when the front end is already close in frequency and CPU
or sample budget is tight; choose Gardner when timing must lock before the carrier does.

## Relevance to SDR

The low sample rate makes Mueller–Müller attractive for lightweight and embedded
demodulators. GopherTrunk uses Mueller–Müller recovery in decoders such as
[AIS](/reference/ais/), [APRS](/reference/aprs/), and related signalling pipelines, where a
single sample per symbol keeps the [clock-recovery](/reference/clock-recovery/) stage cheap
while the framing/CRC layer catches the occasional slip. The detector is named for Kurt
Mueller and Markus Müller, who published it in 1976.

## Sources

[^wiki]: [Symbol synchronization](https://en.wikipedia.org/wiki/Symbol_synchronization) — Wikipedia, for decision-directed symbol-timing recovery such as the Mueller–Müller method and its one-sample-per-symbol operation.
[^mm]: [Timing recovery in digital synchronous data receivers](https://ieeexplore.ieee.org/document/1092560) — K. H. Mueller & M. Müller, *IEEE Trans. Communications*, 1976, the original decision-directed timing-recovery paper.
