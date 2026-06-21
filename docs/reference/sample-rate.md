---
slug: sample-rate
title: Sample rate
entry_type: term
category: sdr-dsp
description: Sample rate is the number of samples per second an SDR produces; with IQ sampling it approximately equals the captured bandwidth, trading spectrum coverage against CPU and USB load.
keywords: sample rate, samples per second, capture bandwidth, IQ rate, decimation, CPU load
aka: [sample rate]
autolink: true
infobox:
  - { label: Symbol, value: Fs }
  - { label: Unit, value: Samples/second (Sa/s) }
  - { label: Rule, value: "captured bandwidth ≈ sample rate (IQ)" }
see_also: [bandwidth, nyquist-theorem, aliasing, decimation, analog-to-digital-converter]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Sampling_(signal_processing)
---

**Sample rate** is how many [IQ](/reference/iq-data/) samples per second an SDR produces.[^wiki]
With complex sampling, the **captured [bandwidth](/reference/bandwidth/) is approximately
equal to the sample rate** — an RTL-SDR at 2.4 MSa/s sees about 2.4 MHz at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sine wave with evenly spaced sample dots, captioned that captured bandwidth roughly equals the sample rate." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="55" x2="440" y2="55" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 55 C 80 15, 140 15, 200 55 S 320 95, 380 55 S 440 35, 440 55" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <g fill="currentColor"><circle cx="20" cy="55" r="3"/><circle cx="60" cy="28" r="3"/><circle cx="110" cy="28" r="3"/><circle cx="160" cy="44" r="3"/><circle cx="210" cy="58" r="3"/><circle cx="260" cy="78" r="3"/><circle cx="320" cy="86" r="3"/><circle cx="380" cy="55" r="3"/><circle cx="430" cy="40" r="3"/></g>
  <text x="20" y="105" font-size="10" fill="currentColor">captured bandwidth ≈ sample rate (IQ sampling)</text>
</svg>
<figcaption>Sample rate sets how often the SDR measures the signal — and how much spectrum it captures at once.</figcaption>
</figure>

## How it works

Higher rates capture more spectrum but produce more data, raising CPU and USB load (and
risking dropped samples). [Filtering and decimation](/reference/decimation/) narrow a wide
capture down to one channel.

## Relevance to SDR

For trunk-tracking, choose a rate that just covers the channels you follow — not the
widest possible span.

## Sources

[^wiki]: [Sampling (signal processing)](https://en.wikipedia.org/wiki/Sampling_(signal_processing)) — Wikipedia, on samples per second and the captured bandwidth they represent.
