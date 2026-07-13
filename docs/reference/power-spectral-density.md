---
slug: power-spectral-density
title: Power spectral density (PSD)
entry_type: term
category: sdr-dsp
description: "Power spectral density describes how a signal's power is distributed across frequency, expressed in watts per hertz (or dB/Hz), and estimated from sampled data."
keywords: power spectral density, PSD, spectral density, power per hertz, dB/Hz, Welch method, periodogram, spectrum estimation, noise density, dBm/Hz
aka: [PSD, spectral density, power density spectrum]
autolink: true
infobox:
  - { label: Symbol, value: "S(f)" }
  - { label: Unit, value: W/Hz (often dBm/Hz) }
  - { label: Estimated by, value: Periodogram / Welch }
see_also: [welch-method, spectrogram, fast-fourier-transform, noise-floor, window-function, discrete-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectral_density
---

**Power spectral density (PSD)** describes how the power of a signal is distributed across
frequency, expressed in watts per hertz — or, on the logarithmic scale radios use, in dB/Hz.[^wiki]
It answers "how much power sits in each slice of the spectrum?", and its integral over a frequency
band gives the total power in that band. Because it is a *density*, the PSD lets you compare the
noise or signal content of channels of different widths on equal footing, which is why it, rather
than raw FFT magnitude, is the honest way to report a spectrum.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A power spectral density curve versus frequency showing a signal peak rising above a flat noise-floor density, with power per hertz on the vertical axis." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="445" y2="135" stroke="currentColor" stroke-width="1.1"/>
  <line x1="40" y1="25" x2="40" y2="135" stroke="currentColor" stroke-width="1.1"/>
  <text x="30" y="30" font-size="8" fill="currentColor" text-anchor="end">dB/Hz</text>
  <text x="242" y="152" font-size="8.5" fill="currentColor" text-anchor="middle">frequency →</text>
  <path d="M45 110 L120 108 L170 106 L210 104 L235 40 L260 104 L300 107 L360 105 L440 108" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="45" y1="112" x2="440" y2="112" stroke="currentColor" stroke-width="0.9" stroke-dasharray="4 3"/>
  <text x="80" y="106" font-size="8" fill="currentColor">noise-floor density</text>
  <text x="235" y="33" font-size="8" fill="currentColor" text-anchor="middle">signal</text>
</svg>
<figcaption>A PSD estimate: the flat baseline is the noise-power density and the peak is a signal; integrating the curve over any band gives the power in that band.</figcaption>
</figure>

## How it works

Formally the PSD is the Fourier transform of the signal's autocorrelation function
(the Wiener–Khinchin theorem), defined for random/stationary signals whose Fourier transform does
not otherwise exist. In practice it is *estimated* from a finite record of samples. The simplest
estimate is the **periodogram**: take the [DFT](/reference/discrete-fourier-transform/) of the
data (via the [FFT](/reference/fast-fourier-transform/)), square the magnitude of each bin, and
scale by the record length and sample rate. That single-shot estimate is unbiased but very noisy —
its variance does not shrink as you take more samples in one long transform, so the trace looks
grassy no matter how much data you feed it.

Two tools tame it, at the cost of frequency resolution:

- **Averaging** — split the record into segments, compute a periodogram of each, and average
  them. Averaging K independent segments cuts the variance by about K, smoothing the estimate.
  The [Welch method](/reference/welch-method/) formalizes this with overlapping, windowed
  segments.
- **Windowing** — multiplying each segment by a [window function](/reference/window-function/)
  before the FFT suppresses spectral leakage, so a strong signal does not smear its power across
  neighboring bins and bury a weak one. The window changes the effective bin bandwidth, which must
  be accounted for so the density scale stays correct.

The essential scaling detail is the **resolution bandwidth (RBW)** — the effective width of one
bin. Because PSD is power *per hertz*, a proper estimate divides each bin's power by its RBW; that
normalization is exactly what makes two spectra taken with different FFT sizes or sample rates
comparable, and it is why a noise floor should be quoted as, say, −150 dBm/Hz rather than a bare
dBm number that depends on the analyzer settings.

## In practice

Radio engineers live on PSD-derived quantities: receiver noise floors and noise figures are
densities; occupied bandwidth and adjacent-channel power are integrals of the PSD; and the
carrier-to-noise ratio is a ratio of a signal's integrated power to the noise density times the
bandwidth. A spectrum analyzer's "noise marker" reports dBm/Hz precisely so results are
independent of its RBW setting.

A recurring confusion is between the PSD of noise and the PSD of a discrete tone. A continuous
noise process genuinely has a density — its power spreads over frequency, so halving the RBW halves
the noise power in each bin, and the noise floor on a display *drops* as you narrow the RBW. A pure
sinusoid, by contrast, carries all its power in an infinitesimal bandwidth, so its bin value does
*not* change with RBW; it sits at a fixed level while the noise around it sinks. This is exactly
why narrowing the resolution bandwidth digs weak carriers out of noise, and why "processing gain"
in an FFT is really just the RBW reduction that comes from a longer transform. Reporting a signal
as a power (dBm) and noise as a density (dBm/Hz) keeps the two straight.

## Relevance to SDR

Every SDR spectrum view is a PSD estimate under the hood — the [waterfall](/reference/waterfall-display/)
and [spectrogram](/reference/spectrogram/) displays in SDR#, GQRX, and SDRangel stack Welch-style
periodograms over time. For a trunking scanner, a PSD sweep is how a user finds control-channel
carriers and gauges whether a channel rises enough above the
[noise floor](/reference/noise-floor/) to decode. GopherTrunk computes FFT-based power estimates to
locate and monitor channel activity across its tuned bandwidth; that channel-power measurement is a
PSD estimate applied to the decision of which channels are carrying traffic.

## Sources

[^wiki]: [Spectral density](https://en.wikipedia.org/wiki/Spectral_density) — Wikipedia, on power spectral density, its units, and estimation from sampled data.
