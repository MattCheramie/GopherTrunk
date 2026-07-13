---
slug: energy-detection
title: Energy detection
entry_type: algorithm
category: estimation-array
description: Energy detection is a radiometer that measures the energy in a band and compares it to a threshold to decide whether a signal is present, the simplest blind spectrum-sensing method.
keywords: energy detection, radiometer, spectrum sensing, cognitive radio, signal detection, squelch, occupancy detection, threshold, blind detection, noise floor
aka: [energy detection, radiometry, radiometer detector]
autolink: true
infobox:
  - { label: Type, value: Blind signal detector }
  - { label: Decides, value: Signal present vs absent }
  - { label: Needs, value: No signal structure knowledge }
see_also: [cfar-detection, noise-floor, signal-to-noise-ratio, matched-filter, welch-method]
cite_urls:
  - https://en.wikipedia.org/wiki/Detection_theory
  - https://ieeexplore.ieee.org/document/1451780
---

**Energy detection** decides whether a signal occupies a band by measuring the total energy
(or power) in that band over an observation window and comparing it to a threshold: above
the threshold means "signal present", below means "noise only".[^wiki] It is the classic
**radiometer**, and its defining virtue is that it needs no knowledge of the signal's
modulation, timing, or structure — it is a *blind* detector that works on anything with more
power than the background.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A signal passes through a bandpass filter, is squared, integrated over a window to form an energy statistic, and compared to a threshold to declare signal present or absent." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="45" width="70" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="59">bandpass</text><text x="65" y="71">filter</text>
    <rect x="130" y="45" width="52" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="156" y="66">(·)²</text>
    <rect x="212" y="45" width="70" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="247" y="59">∫ over</text><text x="247" y="71">window</text>
    <rect x="312" y="45" width="90" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="357" y="59">E ≷ λ ?</text><text x="357" y="71">decide</text>
    <line x1="10" y1="62" x2="29" y2="62" stroke="currentColor" stroke-width="1.1"/><text x="16" y="54">x(t)</text>
    <line x1="100" y1="62" x2="129" y2="62" stroke="currentColor" stroke-width="1.1" marker-end="url(#edar)"/>
    <line x1="182" y1="62" x2="211" y2="62" stroke="currentColor" stroke-width="1.1" marker-end="url(#edar)"/>
    <line x1="282" y1="62" x2="311" y2="62" stroke="currentColor" stroke-width="1.1" marker-end="url(#edar)"/>
    <line x1="402" y1="62" x2="430" y2="62" stroke="currentColor" stroke-width="1.1" marker-end="url(#edar)"/><text x="418" y="54">H₁/H₀</text>
    <text x="230" y="112" text-anchor="middle">threshold λ set from the noise floor and the target false-alarm rate</text>
  </g>
  <defs><marker id="edar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Energy detection filters to the band of interest, squares and integrates to form an energy statistic, then thresholds it to choose between "signal present" and "noise only".</figcaption>
</figure>

## How it works

The detector forms a test statistic `E = Σ |x[n]|²` — the accumulated squared magnitude of
the (band-limited) samples over an observation window of `N` points. Under the
noise-only hypothesis `H₀` this statistic is a scaled chi-square variable centred on the
noise power; under `H₁` it is shifted upward by the signal energy. Pick a threshold `λ`:
the [noise floor](/reference/noise-floor/) and `N` fix the false-alarm probability `P_fa`
(how often `E` crosses `λ` on noise alone), and the [SNR](/reference/signal-to-noise-ratio/)
and `N` then fix the detection probability `P_d`. Longer integration (larger `N`) trades
latency for sensitivity — you can dig a weak signal out by averaging longer, up to the
limits set by noise-power uncertainty.

## In practice

Energy detection's weakness is the flip side of its blindness: it cannot tell signal from
noise-like interference, and it is only as good as its knowledge of the noise power. If the
true [noise floor](/reference/noise-floor/) is uncertain by even a fraction of a dB, there
is an **SNR wall** below which no amount of integration makes the decision reliable, because
the signal energy is indistinguishable from a slightly-higher noise estimate. Making the
threshold adaptive — estimating the local noise from neighbouring bins, exactly the
[CFAR](/reference/cfar-detection/) idea — is the standard fix. Where the signal *is* known,
a [matched filter](/reference/matched-filter/) or cyclostationary detector beats energy
detection by exploiting that structure, at the cost of needing it.

## Relevance to SDR

Energy detection is the backbone of spectrum sensing in cognitive radio (is this channel
free to use?), spectrum-occupancy monitoring, and simple presence/squelch decisions, and it
is a natural companion to a [Welch-averaged](/reference/welch-method/) power spectrum.
**GopherTrunk** relies on power/quality thresholding to decide whether a control or voice
channel carries a usable signal — an energy-detection-style decision in spirit — while its
actual symbol recovery uses structured, matched-filter-based demodulation rather than a bare
radiometer. Energy detection is the simplest member of the detection-theory family that also
includes [CFAR](/reference/cfar-detection/) and matched filtering.

## Sources

[^wiki]: [Detection theory](https://en.wikipedia.org/wiki/Detection_theory) — Wikipedia, on hypothesis testing and the energy/radiometer detector for signal presence.
