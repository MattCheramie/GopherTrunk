---
slug: decibel
title: Decibel (dB)
entry_type: term
category: rf-fundamentals
description: The decibel is a logarithmic unit expressing the ratio between two power levels; in radio it makes gains, losses, and huge dynamic ranges easy to add and compare.
keywords: decibel, dB, logarithmic, ratio, gain, loss, power, bel, dBm, dBFS
aka: [decibel, dB]
autolink: true
infobox:
  - { label: Type, value: Logarithmic ratio unit }
  - { label: Formula, value: "dB = 10·log10(P1/P2)" }
  - { label: Rules, value: "+3 dB ≈ ×2, +10 dB = ×10" }
see_also: [dbm, dbfs, signal-to-noise-ratio, noise-floor, path-loss, link-budget, dynamic-range]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Decibel
  - https://www.nist.gov/pml/special-publication-811
---

The **decibel** (**dB**) is a logarithmic unit expressing the ratio between two power
levels: *dB = 10·log₁₀(P₁/P₂)*.[^wiki] Radio relies on it because signal powers span an
enormous range — from watts at a transmitter to attowatts at a receiver — and because in
the log domain the multiplications of gain and loss along a chain collapse into simple
**addition**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A ladder showing that each +10 dB step multiplies power by ten and +3 dB roughly doubles it, next to a signal chain where amplifier gains and cable losses add up in decibels." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="10" fill="currentColor">
    <line x1="60" y1="16" x2="60" y2="124" stroke="currentColor" stroke-opacity="0.4"/>
    <text x="70" y="26">+30 dB = ×1000</text>
    <text x="70" y="54">+20 dB = ×100</text>
    <text x="70" y="82">+10 dB = ×10</text>
    <text x="70" y="110">+3 dB ≈ ×2</text>
    <line x1="55" y1="22" x2="65" y2="22" stroke="currentColor"/>
    <line x1="55" y1="50" x2="65" y2="50" stroke="currentColor"/>
    <line x1="55" y1="78" x2="65" y2="78" stroke="currentColor"/>
    <line x1="55" y1="106" x2="65" y2="106" stroke="currentColor"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="250" y1="70" x2="440" y2="70" stroke="currentColor" marker-end="url(#dbar)"/>
    <text x="270" y="60">+12</text><text x="315" y="60">−3</text><text x="360" y="60">+20</text>
    <text x="270" y="86">LNA</text><text x="315" y="86">coax</text><text x="360" y="86">amp</text>
    <text x="345" y="112" font-size="10">net = +29 dB</text>
  </g>
</svg>
<figcaption>Decibels are logarithmic: every +10 dB is ten times the power, and a cascade of gains and losses is just a running sum.</figcaption>
</figure>

## How it works

The decibel is one tenth of a *bel*, a unit named for Alexander Graham Bell and rooted
in early telephone loss measurements. The factor of ten in the definition converts bels
to the more convenient decibel.[^nist] Because the logarithm of a product is the sum of the
logarithms, a chain of components that multiply power — an antenna, cable, amplifier,
filter — can be tallied by **adding** their dB figures instead of multiplying raw ratios.

Two anchors cover most mental arithmetic: **+3 dB ≈ double the power**, and **+10 dB =
ten times**. From these you can reconstruct almost any value: +6 dB is ×4, +20 dB is
×100, −3 dB is half. Negative decibels mean the ratio is less than one, i.e. a loss.

A crucial subtlety is **power versus amplitude (field) quantities**. The 10·log₁₀ form
applies to power. When the quantity is a voltage, current, or field strength, power goes
as the *square* of amplitude, so the same ratio is written *dB = 20·log₁₀(V₁/V₂)*. The
factor of 20 is not a different decibel — it falls straight out of log(V²) = 2·log(V).
Getting this wrong is a common source of 2× errors: a voltage that doubles is +6 dB, not
+3 dB.

The decibel is inherently **relative** — it needs a reference. Attach a reference and it
becomes an absolute unit: [dBm](/reference/dbm/) references one milliwatt,
[dBFS](/reference/dbfs/) references digital full scale, dBW references one watt, and dBc
references a carrier. The suffix names the reference; the "dB" part is always the same
logarithm.

## In practice

Almost every RF budget is kept in decibels. A [link budget](/reference/link-budget/)
starts with transmit power in dBm, adds antenna gains, subtracts
[path loss](/reference/path-loss/) and cable [attenuation](/reference/attenuation/), and
lands on a received power that is compared against
[receiver sensitivity](/reference/receiver-sensitivity/) — all by adding and subtracting.
Working the same sum with raw watt ratios would mean multiplying numbers that differ by
twelve orders of magnitude.

Decibels also let a single meter span that range. A spectrum display marked in dB shows
a −120 dBm noise floor and a −20 dBm strong signal on one screen; a linear scale could
not resolve both. The same logic drives figures like
[dynamic range](/reference/dynamic-range/) and
[signal-to-noise ratio](/reference/signal-to-noise-ratio/), which are naturally
expressed as differences in dB.

## Relevance to SDR

Every level in an SDR toolchain is a decibel quantity. Absolute RF power is
[dBm](/reference/dbm/); digital headroom is [dBFS](/reference/dbfs/);
[antenna gain](/reference/antenna-gain/), [path loss](/reference/path-loss/),
[noise figure](/reference/noise-figure/), and [SNR](/reference/signal-to-noise-ratio/)
are all reported in dB. GopherTrunk surfaces demodulator SNR and error metrics in
decibels precisely because they add cleanly along the receive chain and compress a huge
range into readable numbers.

## Sources

[^wiki]: [Decibel](https://en.wikipedia.org/wiki/Decibel) — Wikipedia, definition of the logarithmic ratio unit, the power/field distinction, and reference suffixes.
[^nist]: [NIST Special Publication 811](https://www.nist.gov/pml/special-publication-811) — U.S. National Institute of Standards and Technology, guide to SI units covering correct use of the decibel and logarithmic quantities.
