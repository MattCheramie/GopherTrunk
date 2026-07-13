---
slug: group-delay
title: Group Delay
entry_type: term
category: rf-fundamentals
description: Group delay is the frequency-dependent time a filter imposes on a signal's envelope; variation across the band is phase distortion that smears symbols and raises ISI.
keywords: group delay, group delay variation, phase distortion, envelope delay, linear phase, group delay flatness, dispersion, ISI, filter phase
aka: [group delay, envelope delay, group-delay variation]
autolink: true
infobox:
  - { label: Type, value: Frequency-dependent delay }
  - { label: Formula, value: "τ_g = −dφ/dω" }
  - { label: Ideal, value: Flat (constant) across the band }
see_also: [phase, root-raised-cosine-filter, intersymbol-interference, fir-filter, iir-filter, error-vector-magnitude]
cite_urls:
  - https://en.wikipedia.org/wiki/Group_delay_and_phase_delay
---

**Group delay** is the time by which a system delays the *envelope* of a narrow band of
frequencies, defined as the negative derivative of [phase](/reference/phase/) with
respect to angular frequency, *τ_g = −dφ/dω*.[^wiki] When group delay is not constant
across a signal's occupied band, different frequency components arrive at slightly
different times — a form of phase distortion that spreads each symbol into its
neighbours and raises [intersymbol interference](/reference/intersymbol-interference/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A plot of group delay versus frequency: a flat line across the passband labelled ideal, and a curved line that rises sharply near the band edges labelled real filter, over a shaded passband region." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" font-size="10">
    <line x1="45" y1="20" x2="45" y2="140" marker-end="url(#gdar)"/>
    <line x1="45" y1="140" x2="440" y2="140" marker-end="url(#gdar)"/>
    <rect x="150" y="20" width="180" height="120" fill="currentColor" fill-opacity="0.12" stroke="none"/>
    <line x1="120" y1="95" x2="360" y2="95" stroke-dasharray="4 3"/>
    <path d="M120 60 Q150 92 200 95 T280 95 Q330 92 360 60" stroke-width="1.7"/>
  </g>
  <g fill="currentColor" font-size="10" stroke="none">
    <text x="8" y="30">τ_g</text>
    <text x="410" y="158">freq</text>
    <text x="205" y="88">ideal (flat)</text>
    <text x="120" y="52">rise at band edges</text>
    <text x="190" y="16">passband</text>
  </g>
</svg>
<figcaption>An ideal filter has constant group delay (flat) across the passband; a real filter's delay curves upward near the band edges, delaying edge frequencies more than mid-band and distorting the waveform.</figcaption>
</figure>

## How it works

A signal is only passed undistorted in phase if the system has **linear phase** — phase
that changes proportionally with frequency. The derivative of a straight line is a
constant, so linear phase means *constant group delay*: every frequency component is held
up by the same amount and the waveform shape is preserved, merely shifted in time.

When phase deviates from a straight line, group delay varies with frequency and the
system is **dispersive**. Higher-frequency components of a pulse may be delayed more (or
less) than lower ones, so the reassembled pulse spreads and rings. It is the *variation*
in group delay across the used band — the **group-delay flatness** — that causes trouble,
not the absolute delay: a constant 10 µs delay harms nothing, but 2 µs of variation
across a channel can close an eye diagram.

Sharp filters are the usual culprit. A steep [IIR filter](/reference/iir-filter/) has
strongly non-flat group delay near its cutoff, worst right at the band edges where the
phase bends most. A symmetric [FIR filter](/reference/fir-filter/) can be made *exactly*
linear-phase — its group delay is a flat constant of half the filter length — which is a
major reason FIR structures are favoured in communications where waveform fidelity
matters.

## In practice

Because group-delay variation and amplitude ripple both degrade a channel, systems
either design for flatness or correct for it. A matched
[root-raised-cosine](/reference/root-raised-cosine-filter/) pulse-shaping pair is chosen
to give an overall linear-phase, controlled-[ISI](/reference/intersymbol-interference/)
response at the sampling instants. Where the channel itself adds dispersion, an
[adaptive equalizer](/reference/adaptive-filter/) in the receiver estimates and inverts
the combined amplitude and group-delay distortion. Group-delay flatness is a standard
line item on filter and cable datasheets, and excess variation shows up directly as a
raised [error vector magnitude](/reference/error-vector-magnitude/).

## Relevance to SDR

Group-delay flatness matters most for wideband and high-order-modulation signals, where
a distorted phase response smears symbols and closes the eye. TETRA, P25, and DMR all
rely on well-behaved, near-linear-phase transmit and receive filtering to keep their
constellations clean; sharp analog IF filters in older receivers were a classic source of
group-delay distortion that digital FIR filtering now avoids.

**GopherTrunk** performs its channel filtering and decimation digitally, and its
pulse-shaping and channelizing stages use FIR structures whose linear phase gives
constant group delay by construction — so the decode chain does not itself introduce
phase dispersion. The concept remains useful for diagnosis: if a captured signal decodes
poorly with an otherwise healthy SNR, group-delay distortion introduced *before* the SDR
(in a cheap external filter or a long marginal feedline) is one candidate the concept
helps rule in or out.

## Sources

[^wiki]: [Group delay and phase delay](https://en.wikipedia.org/wiki/Group_delay_and_phase_delay) — Wikipedia, definitions and the linear-phase / constant-delay relationship.
