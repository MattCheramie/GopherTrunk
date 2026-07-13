---
slug: reflection-coefficient
title: Reflection Coefficient (Γ)
entry_type: term
category: rf-fundamentals
description: The reflection coefficient Γ is the complex ratio of reflected to incident wave at an impedance boundary, Γ = (ZL − Z0)/(ZL + Z0), and underlies VSWR and return loss.
keywords: reflection coefficient, gamma, Γ, rho, impedance mismatch, ZL Z0, return loss, VSWR, S11
aka: [Γ, gamma, rho, voltage reflection coefficient]
autolink: true
infobox:
  - { label: Symbol, value: "Γ (or ρ)" }
  - { label: Range, value: "0 (matched) to 1 (total)" }
  - { label: Formula, value: "Γ = (ZL − Z0)/(ZL + Z0)" }
see_also: [impedance, return-loss, standing-wave-ratio, s-parameters, vector-network-analyzer, feedpoint-impedance]
cite_urls:
  - https://en.wikipedia.org/wiki/Reflection_coefficient
  - https://en.wikipedia.org/wiki/Reflections_of_signals_on_conducting_lines
---

**The reflection coefficient** (Γ, sometimes ρ) is the complex ratio of the wave
reflected from an impedance boundary to the wave incident on it.[^wiki] For a load
*ZL* terminating a line of characteristic [impedance](/reference/impedance/) *Z0*, it
is **Γ = (ZL − Z0) / (ZL + Z0)**. Its magnitude runs from 0 for a perfect match to 1
for total reflection, and its phase records where along the wave cycle the reflection
occurs. Γ is the single quantity from which
[return loss](/reference/return-loss/) and
[standing-wave ratio](/reference/standing-wave-ratio/) are derived.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="An incident wave travels down a transmission line to a load; part continues into the load and part reflects back as Gamma times the incident wave, with the formula Gamma equals ZL minus Z0 over ZL plus Z0." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="20" y1="60" x2="300" y2="60" stroke="currentColor" stroke-width="1.5"/>
  <line x1="20" y1="90" x2="300" y2="90" stroke="currentColor" stroke-width="1.5"/>
  <rect x="300" y="55" width="55" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="327" y="79" font-size="9" text-anchor="middle" fill="currentColor">ZL</text>
  <line x1="40" y1="45" x2="150" y2="45" stroke="currentColor" stroke-width="2" marker-end="url(#rcar)"/>
  <text x="60" y="40" font-size="9" fill="currentColor">incident</text>
  <line x1="150" y1="105" x2="40" y2="105" stroke="currentColor" stroke-width="2" marker-end="url(#rcar)"/>
  <text x="60" y="120" font-size="9" fill="currentColor">reflected = Γ · incident</text>
  <text x="150" y="145" font-size="11" text-anchor="middle" fill="currentColor">Γ = (ZL − Z0) / (ZL + Z0)</text>
  <text x="130" y="55" font-size="8" fill="currentColor">Z0</text>
</svg>
<figcaption>At a mismatched boundary part of the incident wave reflects; Γ is the complex ratio of that reflection to the incident wave, set entirely by ZL and Z0.</figcaption>
</figure>

## How it works

When a travelling wave reaches a boundary where the impedance changes, it cannot
simply continue unchanged — the boundary conditions on voltage and current must be
satisfied on both sides. The only way to satisfy them is for part of the wave to
reflect. Solving those conditions gives the defining formula
*Γ = (ZL − Z0)/(ZL + Z0)*, where both impedances may be complex, so Γ itself is
generally complex.

Three special cases build intuition:

- **Matched load, ZL = Z0.** The numerator is zero, so Γ = 0. Nothing reflects; all
  power enters the load. This is the design goal.
- **Open circuit, ZL → ∞.** Γ → +1. The wave reflects fully and in phase; voltage
  doubles at the open end.
- **Short circuit, ZL = 0.** Γ = −1. The wave reflects fully but inverted; voltage
  is forced to zero at the short.

For a passive load |Γ| never exceeds 1, because a passive termination cannot reflect
more power than it received. The fraction of *power* reflected is |Γ|², so a Γ
magnitude of 0.1 sends back only 1 % of the incident power even though it is a tenth
of the incident *voltage*.

## In practice

Γ is not just a load property; it varies with position along a lossless line,
rotating in phase by 720° per wavelength while keeping constant magnitude. That
rotation is exactly what a Smith chart plots, which is why impedance matching can be
read off as movement around the chart. On a two-port device the input reflection
coefficient looking into port 1 is the [S-parameter](/reference/s-parameters/)
*S11*, and a [vector network analyzer](/reference/vector-network-analyzer/) measures
Γ directly in both magnitude and phase across frequency.

Two derived numbers repackage |Γ| for convenience.
[Return loss](/reference/return-loss/) is −20·log₁₀|Γ| in decibels, and the
[standing-wave ratio](/reference/standing-wave-ratio/) is
(1 + |Γ|)/(1 − |Γ|). All three describe the same mismatch.

## Relevance to SDR

The reflection coefficient at an SDR's antenna port determines how much of the
captured signal actually crosses into the receiver rather than bouncing back up the
feedline. A [feedpoint impedance](/reference/feedpoint-impedance/) that drifts from
50 Ω as you tune across a band raises |Γ| and quietly costs signal, which for a
weak trunking control channel can be the difference between a lock and dropped
frames. On transmit-capable radios a high |Γ| also means power reflected back toward
the [power amplifier](/reference/power-amplifier/), a stress worth avoiding.

GopherTrunk works on the IQ samples produced after the front end, so it never sees Γ
directly — the reflection has already happened in the antenna and cabling by the time
samples reach the decoder. The concept still matters to operators because minimising
|Γ| with a resonant antenna and a clean feedline is the cheapest way to raise the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) the software depends on.

## Sources

[^wiki]: [Reflection coefficient](https://en.wikipedia.org/wiki/Reflection_coefficient) — Wikipedia, the Γ = (ZL − Z0)/(ZL + Z0) definition and its relation to VSWR and return loss.
