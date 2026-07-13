---
slug: harmonics
title: Harmonics
entry_type: term
category: rf-fundamentals
description: Harmonics are signals at integer multiples of a fundamental frequency, produced by nonlinearity and a major source of spurious radiation controlled by filtering.
keywords: harmonics, harmonic distortion, second harmonic, third harmonic, fundamental frequency, integer multiple, nonlinearity, spurious radiation, harmonic filter, THD
aka: [harmonic, harmonic distortion, harmonic frequencies]
autolink: true
infobox:
  - { label: Type, value: Nonlinear distortion product }
  - { label: Relation, value: "fₙ = n·f₀ (n = 2, 3, 4 …)" }
  - { label: Controlled by, value: Low-pass / band-pass filtering }
see_also: [spurious-emissions, intermodulation, rf-filter, power-amplifier, occupied-bandwidth, mixer-rf]
cite_urls:
  - https://en.wikipedia.org/wiki/Harmonic
  - https://en.wikipedia.org/wiki/Total_harmonic_distortion
---

**Harmonics** are spectral components at integer multiples of a signal's fundamental
frequency *f₀* — the second harmonic sits at *2f₀*, the third at *3f₀*, and so on.[^wiki]
They arise whenever a device treats a sine wave nonlinearly, and in a transmitter they
are a leading source of unwanted [spurious emissions](/reference/spurious-emissions/)
that must be suppressed before the signal reaches the antenna.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A spectrum plot with a tall fundamental line at f-zero and progressively shorter lines at two-f, three-f and four-f, showing harmonics decreasing in amplitude." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="harmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="currentColor" font-size="10">
    <line x1="40" y1="20" x2="40" y2="140"/>
    <line x1="40" y1="140" x2="440" y2="140" marker-end="url(#harmar)"/>
    <text x="425" y="158">freq</text>
    <line x1="90" y1="40" x2="90" y2="140" stroke-width="2"/>
    <text x="72" y="34">f₀</text>
    <line x1="190" y1="80" x2="190" y2="140" stroke-width="2"/>
    <text x="178" y="74">2f₀</text>
    <line x1="290" y1="100" x2="290" y2="140" stroke-width="2"/>
    <text x="278" y="94">3f₀</text>
    <line x1="390" y1="118" x2="390" y2="140" stroke-width="2"/>
    <text x="378" y="112">4f₀</text>
    <text x="46" y="30" fill="currentColor">amp</text>
  </g>
</svg>
<figcaption>The fundamental at f₀ and its harmonics at integer multiples; each harmonic is normally weaker, but even a small one can violate emission limits.</figcaption>
</figure>

## How it works

Any perfectly linear device passes a pure sine wave through unchanged. Real components —
[power amplifiers](/reference/power-amplifier/) driven near saturation, diodes, and
[mixers](/reference/mixer-rf/) — have a transfer curve that bends. Mathematically, a
nonlinear response can be written as a power series *y = a₁x + a₂x² + a₃x³ + …*, and
feeding a sine *x = cos(2πf₀t)* into the squared and cubed terms generates energy at
*2f₀*, *3f₀*, and higher. The stronger the nonlinearity — the harder an amplifier is
driven into compression — the richer the harmonic content.

Two useful facts fall out of this. First, the square-law (even-order) terms create
even harmonics while the cubic (odd-order) terms create odd harmonics, so a symmetric,
push-pull stage naturally suppresses even harmonics. Second, harmonics always fall
*above* the fundamental at exact integer multiples, which makes them predictable and
therefore easy to filter — unlike [intermodulation](/reference/intermodulation/)
products, which land near the wanted signal.

**Total harmonic distortion (THD)** quantifies the effect as the ratio of combined
harmonic power to the fundamental, usually expressed as a percentage or in dB.[^thd]

## In practice

Because harmonics land at known frequencies, a transmitter suppresses them with a
**low-pass or band-pass filter** after the final amplifier. A 150 MHz VHF transmitter,
for example, uses a low-pass filter that passes 150 MHz but rejects the second harmonic
at 300 MHz by tens of dB. Class-C and switching amplifiers are efficient precisely
because they run nonlinearly, so they lean hardest on this output filtering; linear
classes (A, AB) generate fewer harmonics at the cost of efficiency.

Harmonics matter on the receive side too. A strong local FM broadcast can present a
harmonic that lands in a band you are trying to monitor, and the receiver's own
front-end and [mixer](/reference/mixer-rf/) generate harmonics of the local oscillator
that create image and spurious responses. Good [RF filtering](/reference/rf-filter/)
and avoiding front-end overload keep these in check.

## Relevance to SDR

Harmonic suppression is a regulatory requirement: transmitter type-approval rules set
hard limits on harmonic radiation, and exceeding them causes interference in unrelated
services. For land-mobile trunking systems (P25, DMR, TETRA), base-station and portable
transmitters carry cavity and low-pass filters specifically to hold harmonics below the
mandated [spurious-emission](/reference/spurious-emissions/) mask.

**GopherTrunk** is a receive-only decoder and does not transmit, so it produces no
harmonics of its own. Harmonics still matter to its users indirectly: cheap SDR
front-ends (RTL-SDR dongles and similar) have limited front-end selectivity, so a
strong out-of-band signal or its harmonic can overload the tuner and raise the effective
[noise floor](/reference/noise-floor/), degrading decode of the wanted control channel.
An external band-pass filter ahead of the SDR is the standard cure, and understanding
where harmonics fall helps a monitor diagnose a spur that is not actually a real
transmission but a harmonic of one.

## Sources

[^wiki]: [Harmonic](https://en.wikipedia.org/wiki/Harmonic) — Wikipedia, definition of harmonics as integer multiples of a fundamental frequency.
[^thd]: [Total harmonic distortion](https://en.wikipedia.org/wiki/Total_harmonic_distortion) — Wikipedia, the metric relating combined harmonic power to the fundamental.
