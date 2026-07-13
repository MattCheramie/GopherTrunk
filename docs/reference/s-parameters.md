---
slug: s-parameters
title: S-parameters (Scattering Parameters)
entry_type: term
category: rf-fundamentals
description: S-parameters describe how an RF network scatters incident waves into reflected and transmitted ones; S11 is input match and S21 is forward gain, measured on a VNA.
keywords: s-parameters, scattering parameters, S11, S21, S12, S22, two-port network, VNA, return loss, insertion loss, gain
aka: [scattering parameters, S-matrix]
autolink: true
infobox:
  - { label: Symbol, value: "Sij" }
  - { label: Form, value: "complex (magnitude & phase)" }
  - { label: Measured with, value: "vector network analyzer" }
see_also: [reflection-coefficient, return-loss, vector-network-analyzer, impedance, low-noise-amplifier, rf-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Scattering_parameters
  - https://en.wikipedia.org/wiki/Two-port_network
---

**S-parameters** (scattering parameters) describe a linear RF network entirely by how
it scatters incident waves at its ports into reflected and transmitted waves.[^wiki]
Each *Sij* is a complex number — magnitude and phase — giving the wave emerging at
port *i* per unit wave injected at port *j*, with all other ports terminated in the
reference [impedance](/reference/impedance/). For the ubiquitous two-port device,
**S11** is the input match, **S21** the forward transmission (gain or loss), **S12**
the reverse transmission (isolation), and **S22** the output match.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A two-port network box with port 1 on the left and port 2 on the right; incoming and outgoing wave arrows are labelled with S11 as the input reflection, S21 as forward transmission, S12 as reverse transmission, and S22 as output reflection." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="spar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="160" y="50" width="140" height="70" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="230" y="90" font-size="10" text-anchor="middle" fill="currentColor">2-port network</text>
  <text x="150" y="45" font-size="9" text-anchor="end" fill="currentColor">port 1</text>
  <text x="310" y="45" font-size="9" fill="currentColor">port 2</text>
  <line x1="60" y1="70" x2="158" y2="70" stroke="currentColor" stroke-width="2" marker-end="url(#spar)"/>
  <text x="70" y="64" font-size="8" fill="currentColor">a1 in</text>
  <line x1="158" y1="95" x2="60" y2="95" stroke="currentColor" stroke-width="2" marker-end="url(#spar)"/>
  <text x="70" y="110" font-size="8" fill="currentColor">S11·a1 reflected</text>
  <line x1="302" y1="70" x2="400" y2="70" stroke="currentColor" stroke-width="2" marker-end="url(#spar)"/>
  <text x="330" y="64" font-size="8" fill="currentColor">S21·a1 out</text>
  <line x1="400" y1="95" x2="302" y2="95" stroke="currentColor" stroke-width="2" marker-end="url(#spar)"/>
  <text x="330" y="110" font-size="8" fill="currentColor">a2 in</text>
  <text x="140" y="150" font-size="9" fill="currentColor">S11 in-match · S21 gain · S12 isolation · S22 out-match</text>
</svg>
<figcaption>A two-port's behaviour is captured by four scattering parameters: S11 and S22 are the port reflections, S21 the forward transmission, and S12 the reverse leakage.</figcaption>
</figure>

## How it works

At radio frequencies you cannot easily hold a port open or shorted to measure the
voltages and currents that classic impedance (Z) or admittance (Y) parameters
require — the reactances of test leads dominate and open/short standards misbehave.
S-parameters sidestep this by working with *travelling waves* and terminating every
port in a well-defined reference impedance, usually 50 Ω. Each port has an incoming
wave *a* and an outgoing wave *b*, and the network is fully described by the matrix
equation *b = S·a*. For a two-port that expands to:

- *b1 = S11·a1 + S12·a2*
- *b2 = S21·a1 + S22·a2*

To measure one parameter you drive a single port and terminate the rest, so their
incoming waves are zero. With port 2 terminated, *S11 = b1/a1* is just the input
[reflection coefficient](/reference/reflection-coefficient/), and *S21 = b2/a1* is
the forward transmission. Swap the drive to the other port for *S22* and *S12*.

## In practice

Because each parameter is complex, S-data carries both how much a signal is
attenuated or amplified *and* the phase shift it undergoes — essential for cascading
stages, designing matching networks, and building filters. Common derived readings:

- |S11| and |S22| in dB give the [return loss](/reference/return-loss/) at each port.
- |S21| in dB gives insertion loss for a passive part or gain for an amplifier.
- |S12| in dB gives reverse isolation; a large negative value means the device is
  well unilateral (little leakage backward).

S-parameters are conventionally exported as **Touchstone** (`.s2p`) files listing
each parameter's magnitude and phase across frequency, which circuit simulators read
directly. They are defined for linear, time-invariant behaviour, so they characterise
small-signal operation; strongly nonlinear effects like compression and
intermodulation need additional measurements.

## Relevance to SDR

The building blocks of an SDR front end are specified in S-parameters: a
[low-noise amplifier](/reference/low-noise-amplifier/) is quoted by its S21 gain and
S11 input match, an [RF filter](/reference/rf-filter/) by its S21 passband shape and
stopband rejection, and connectors and cables by their return and insertion loss.
Designers measure all of these on a [vector network
analyzer](/reference/vector-network-analyzer/), which sweeps a known incident wave
and ratios the reflected and transmitted waves to recover each *Sij* in magnitude and
phase.

GopherTrunk is a software decoder that operates after the analog chain, so it neither
produces nor consumes S-parameters — they belong to the hardware layer that delivers
IQ samples. The concept is still worth knowing for GopherTrunk users choosing an
antenna, filter, or amplifier, because those parts' datasheets speak in S-parameters,
and a good S11/S21 profile at the frequency of interest is what ultimately hands the
decoder clean samples.

## Sources

[^wiki]: [Scattering parameters](https://en.wikipedia.org/wiki/Scattering_parameters) — Wikipedia, the S-matrix definition, two-port S11/S21/S12/S22 meanings, and reference-impedance conventions.
