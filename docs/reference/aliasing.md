---
slug: aliasing
title: Aliasing
entry_type: term
category: sdr-dsp
description: Aliasing is the appearance of out-of-band energy at a false frequency when a signal is sampled too slowly for its bandwidth; anti-alias filtering and adequate sample rate prevent it.
keywords: aliasing, fold-back, anti-alias filter, undersampling, image, false signal, Nyquist
aka: [aliasing]
autolink: true
infobox:
  - { label: Type, value: Sampling artefact }
  - { label: Cause, value: Sampling below Nyquist for the bandwidth }
  - { label: Prevented by, value: Anti-alias filter + adequate rate }
see_also: [nyquist-theorem, sample-rate, decimation, digital-filter, oversampling, bandpass-sampling, image-frequency]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Aliasing
  - https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem
---

**Aliasing** is when energy outside the bandwidth a [sample rate](/reference/sample-rate/)
can represent gets **folded** back into the captured spectrum, appearing at a wrong
frequency — a phantom that looks like a real signal but sits nowhere near where its source
actually transmits.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A captured band between Nyquist edges with a real signal inside and an out-of-band signal folding back to a false position." xmlns="http://www.w3.org/2000/svg">
  <line x1="60" y1="105" x2="420" y2="105" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="110" y1="20" x2="110" y2="115" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <line x1="330" y1="20" x2="330" y2="115" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="220" y="125" text-anchor="middle" font-size="9" fill="currentColor">captured bandwidth</text>
  <path d="M160 105 L170 55 L180 105 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/><text x="170" y="48" text-anchor="middle" font-size="8" fill="currentColor">real</text>
  <path d="M360 105 L370 70 L380 105 Z" fill="none" stroke="currentColor" stroke-opacity="0.5"/><text x="372" y="63" text-anchor="middle" font-size="8" fill="currentColor">out of band</text>
  <path d="M255 105 L265 78 L275 105 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-dasharray="3 2"/><text x="265" y="71" text-anchor="middle" font-size="8" fill="currentColor">alias!</text>
  <path d="M368 73 q-50 -28 -100 0" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="2 2"/>
</svg>
<figcaption>Aliasing: energy beyond the captured bandwidth folds back to a false position inside it.</figcaption>
</figure>

## How it works

Sampling maps every input frequency to a value modulo the sample rate. Two tones separated
by an exact multiple of Fs produce identical sample sequences — they are *aliases* of one
another and become indistinguishable the instant they are digitised. Formally, the spectrum
repeats every Fs, so any component above the Nyquist frequency (Fs/2) mirrors, or "folds,"
back into the band below it. This is simply the [Nyquist theorem](/reference/nyquist-theorem/)
violated: the theorem guarantees a perfect reconstruction *only* if no energy exists above
the Nyquist frequency, and aliasing is exactly what goes wrong when that precondition fails.

The corruption is irreversible. Once a strong out-of-band station has folded on top of a
weak in-band one, no filter applied after sampling can pull them apart, because in the
digital domain they occupy the same bin. The defence therefore has to come *before* the
sampler: an analog anti-alias filter removes the offending energy while it can still be
separated. In the digital domain the same logic governs [decimation](/reference/decimation/),
which is why the rule is always **filter first, then drop samples** — decimating without
filtering re-introduces aliasing by lowering the effective Nyquist frequency under the
signal.

## Variants

- **Anti-alias (pre-sampling) aliasing** — the classic case, prevented by the analog filter
  ahead of the ADC.
- **Decimation aliasing** — created by lowering the rate in software without an adequate
  low-pass stage first.
- **Deliberate aliasing** — [bandpass sampling](/reference/bandpass-sampling/) uses folding
  on purpose to bring a high band down to baseband.
- **[Image](/reference/image-frequency/) responses** — the analog cousin: a mixer folds a
  band on the far side of the local oscillator into the IF, which is why superheterodyne
  front-ends need image rejection.

## In practice

[Oversampling](/reference/oversampling/) gives the anti-alias filter a wide transition band
to work with, making a gentle, cheap filter sufficient. On a waterfall an alias often gives
itself away by moving the *wrong* direction when you retune, or by mirroring across the
band edge as you change the sample rate — a real signal stays put in absolute frequency
while an alias shifts. This is a standard sanity check when a station appears where none
should be.

## Relevance to SDR

Recognising an alias prevents chasing signals that are not really where they appear.
GopherTrunk's front-end and decimation stages include the low-pass filtering that keeps real
control and voice channels safely inside the usable window; an operator who decimates a raw
capture with a third-party tool that skips the filter can inject aliases that break the
decode.

## Sources

[^wiki]: [Aliasing](https://en.wikipedia.org/wiki/Aliasing) — Wikipedia, on out-of-band energy folding to false frequencies when undersampled.
[^nyq]: [Nyquist–Shannon sampling theorem](https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem) — Wikipedia, on the band-limiting precondition whose violation causes aliasing.
