---
slug: signal-to-noise-ratio
title: Signal-to-noise ratio (SNR)
entry_type: term
category: rf-fundamentals
description: Signal-to-noise ratio is the difference in decibels between a signal's power and the noise floor; digital modes require a minimum SNR to decode reliably.
keywords: SNR, signal to noise ratio, noise floor, dB, decode threshold, SINAD, Eb/N0, C/N
aka: [signal-to-noise ratio, SNR, S/N]
autolink: true
infobox:
  - { label: Symbol, value: SNR }
  - { label: Unit, value: Decibels (dB) }
  - { label: Formula, value: "signal (dBm) − noise floor (dBm)" }
see_also: [noise-floor, decibel, dbm, demodulation, carrier-to-noise-ratio, eb-n0, shannon-capacity, receiver-sensitivity]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Signal-to-noise_ratio
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Signal-to-noise ratio** (**SNR**) is the gap, in [decibels](/reference/decibel/),
between a signal's power and the [noise floor](/reference/noise-floor/).[^wiki] It is the
single best predictor of whether a signal will decode: enough SNR and the bits come out
clean; too little and errors overwhelm the
[demodulator](/reference/demodulation/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A spectrum with a noisy baseline labelled noise floor and a tall peak labelled signal, with the vertical gap between the peak and the floor labelled SNR in decibels." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 120 L70 124 L110 116 L150 122 L190 119 L240 121 L300 118 L360 121 L420 119" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <line x1="30" y1="120" x2="430" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/>
  <text x="350" y="135" font-size="10" fill="currentColor">noise floor</text>
  <path d="M210 120 L224 45 L238 120 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.5"/>
  <text x="224" y="38" text-anchor="middle" font-size="10" fill="currentColor">signal</text>
  <line x1="260" y1="45" x2="260" y2="120" stroke="currentColor"/>
  <text x="268" y="86" font-size="11" fill="currentColor">SNR</text>
  <text x="230" y="158" text-anchor="middle" font-size="9" fill="currentColor">frequency →</text>
</svg>
<figcaption>SNR is how far a signal rises above the noise floor; each digital mode needs a minimum SNR to decode.</figcaption>
</figure>

## How it works

SNR = signal level − noise-floor level, with both measured in [dBm](/reference/dbm/) so
the subtraction gives a value in dB. A signal at −85 dBm over a −105 dBm floor has 20 dB
of SNR. Because it is a *difference* of two dBm readings, any constant calibration offset
in the receiver cancels — which is why SNR is meaningful even on uncalibrated SDR
hardware that cannot report true absolute dBm.

The measurement depends on **bandwidth**. Noise power scales with the width over which
you integrate it (the [thermal noise](/reference/thermal-noise/) floor rises ~3 dB each
time bandwidth doubles), so an SNR figure is only well defined relative to a stated
bandwidth. Measuring noise in a wide span and signal in a narrow one inflates the number;
comparing SNR across systems means normalising to a common bandwidth. This is also why
narrowing the receive filter to the signal's occupied bandwidth improves SNR — it admits
less noise while keeping all the signal.

Each digital mode has a **threshold SNR** below which the demodulator's error rate climbs
steeply. The relationship is a "waterfall" curve: above threshold the
[bit error rate](/reference/bit-error-rate/) is negligible, and within a few dB below it
the link collapses. [Forward error correction](/reference/forward-error-correction/)
lowers the threshold by trading redundancy for robustness, letting a mode work at lower
SNR than its raw modulation would allow.

SNR has several close relatives worth distinguishing. **[C/N](/reference/carrier-to-noise-ratio/)**
(carrier-to-noise) measures the modulated carrier against noise before demodulation.
**[Eb/N0](/reference/eb-n0/)** normalises to energy per bit and noise density, making a
fair comparison across data rates and modulations. **SINAD** folds distortion in with
noise for analog voice quality. All are dB ratios of wanted to unwanted power, differing
in exactly what they count.

## In practice

SNR is the number to watch on a site survey. Typical thresholds: analog FM voice becomes
readable around 10–12 dB SNR; C4FM/CQPSK digital modes like P25 need roughly 15–20 dB at
the demodulator for a clean lock, with FEC providing some margin below that. A signal 3 dB
over threshold is usable but fragile; 10 dB over is comfortable.

The [Shannon–Hartley theorem](/reference/shannon-capacity/) sets the ultimate limit:
channel capacity grows with log₂(1 + SNR), so every doubling of SNR buys diminishing
returns in achievable bit rate. Practical systems pick a modulation and code that decode
reliably a few dB above their threshold at the expected SNR.

## Relevance to SDR

Improving SNR — a better [antenna](/reference/antenna/), a higher or clearer site,
correct [gain](/reference/automatic-gain-control/), a
[low-noise amplifier](/reference/low-noise-amplifier/), and matched filtering — is usually
what moves a marginal signal from un-decodable to clean. GopherTrunk reports per-channel
demodulator SNR and EVM so an operator can see, in decibels, exactly how much margin a
link has and whether a failed decode is an SNR problem or something else (front-end
overload, wrong tuning) entirely.

## Sources

[^wiki]: [Signal-to-noise ratio](https://en.wikipedia.org/wiki/Signal-to-noise_ratio) — Wikipedia, definition, bandwidth dependence, and significance of SNR.
[^shannon]: [Shannon–Hartley theorem](https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem) — Wikipedia, the capacity limit relating achievable bit rate to SNR and bandwidth.
