---
slug: sample-rate
title: Sample rate
entry_type: term
category: sdr-dsp
description: Sample rate is the number of samples per second an SDR produces; with IQ sampling it approximately equals the captured bandwidth, trading spectrum coverage against CPU and USB load.
keywords: sample rate, samples per second, capture bandwidth, IQ rate, decimation, oversampling, CPU load
aka: [sample rate]
autolink: true
infobox:
  - { label: Symbol, value: Fs }
  - { label: Unit, value: Samples/second (Sa/s) }
  - { label: Rule, value: "captured bandwidth ≈ sample rate (IQ)" }
see_also: [bandwidth, nyquist-theorem, aliasing, decimation, oversampling, bandpass-sampling, analog-to-digital-converter]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Sampling_(signal_processing)
  - https://en.wikipedia.org/wiki/Nyquist_rate
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

The reason complex sampling captures a bandwidth equal to the rate — rather than *half* the
rate, as the [Nyquist theorem](/reference/nyquist-theorem/) says for a single real channel
— is that each sample carries two numbers (I and Q). The two channels resolve the sign of
the frequency offset, so the usable window runs from −Fs/2 to +Fs/2 around the tuned centre:
a full Fs wide. A real-only stream at the same rate would only span 0 to Fs/2.

Higher rates capture more spectrum but produce more data, raising CPU and USB load and
risking dropped samples on a bus that cannot keep up. Because trunked systems spread control
and voice channels across a band, the practical question is not "the widest possible span"
but "the narrowest span that still covers every channel I need to follow." Once a wide
capture is in hand, [filtering and decimation](/reference/decimation/) narrow it down to
one channel at a much lower rate for the demodulator.

## Variants

- **Oversampling** — running Fs well above the minimum. [Oversampling](/reference/oversampling/)
  eases the analog anti-alias filter's job and, after decimation, lowers in-band
  quantization noise.
- **Critical (Nyquist-rate) sampling** — Fs set just above twice the signal bandwidth;
  efficient but demands sharp filters.
- **[Bandpass sampling](/reference/bandpass-sampling/)** — deliberately sampling a
  high-frequency band *below* its carrier so it aliases down to baseband on purpose; used by
  some direct-sampling and undersampling receivers.

## In practice

SDR hardware only offers a discrete menu of rates set by its clock dividers — an RTL-SDR is
reliable around 2.4 and 2.56 MSa/s, an Airspy runs at 2.5, 6, or 10 MSa/s, and higher-end
radios reach tens of MSa/s. The rate also fixes the frequency resolution of an FFT display:
finer resolution needs either a longer transform or a lower rate. GopherTrunk's decode chain
is deliberately **rate-invariant**: it normalises whatever capture rate the radio delivers
down to the per-protocol channel rate (48 kHz for the 4800-baud C4FM family, 144 kHz for
TETRA) before demodulation, so the receiver behaves the same whether fed 2.5 or 10 MSa/s.

## Relevance to SDR

For trunk-tracking, choose a rate that just covers the channels you follow — not the widest
possible span — and let the [digital down-converter](/reference/digital-down-converter/)
pull each channel out. A rate mismatch, or a capture whose true problem is front-end noise
at a high native clock rather than the rate itself, is a common source of decode failures.

## Sources

[^wiki]: [Sampling (signal processing)](https://en.wikipedia.org/wiki/Sampling_(signal_processing)) — Wikipedia, on samples per second and the captured bandwidth they represent.
[^nyq]: [Nyquist rate](https://en.wikipedia.org/wiki/Nyquist_rate) — Wikipedia, on the minimum sampling rate for a given bandwidth and why complex sampling doubles the usable span.
