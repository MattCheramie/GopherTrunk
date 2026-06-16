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
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Sampling (signal processing) (Wikipedia)", url: https://en.wikipedia.org/wiki/Sampling_(signal_processing) }
---

**Sample rate** is how many [IQ](/reference/iq-data/) samples per second an SDR produces.
With complex sampling, the **captured [bandwidth](/reference/bandwidth/) is approximately
equal to the sample rate** — an RTL-SDR at 2.4 MSa/s sees about 2.4 MHz at once.

## How it works

Higher rates capture more spectrum but produce more data, raising CPU and USB load (and
risking dropped samples). [Filtering and decimation](/reference/decimation/) narrow a wide
capture down to one channel.

## Relevance to SDR

For trunk-tracking, choose a rate that just covers the channels you follow — not the
widest possible span.
