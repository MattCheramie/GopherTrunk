---
slug: erp-eirp
title: ERP & EIRP
entry_type: term
category: rf-fundamentals
description: ERP and EIRP express a transmitter's effective radiated power relative to a reference antenna — a dipole for ERP, an ideal isotropic radiator for EIRP.
keywords: ERP, EIRP, effective radiated power, effective isotropic radiated power, antenna gain, dBi, dBd, transmitter power, radiated power
aka: [ERP, EIRP, effective radiated power, effective isotropic radiated power]
autolink: true
infobox:
  - { label: Type, value: Radiated-power measure }
  - { label: Formula, value: "EIRP(dBm) = P_tx − losses + G_ant(dBi)" }
  - { label: Reference, value: "ERP → dipole (dBd); EIRP → isotropic (dBi)" }
see_also: [antenna-gain, decibel, dbm, free-space-path-loss, link-budget, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Effective_radiated_power
  - https://en.wikipedia.org/wiki/Equivalent_isotropically_radiated_power
---

**ERP** (effective radiated power) and **EIRP** (effective isotropic radiated power) state
how much power a transmitter *appears* to radiate in its strongest direction, folding the
transmitter output, feedline loss, and [antenna gain](/reference/antenna-gain/) into a single
number.[^wiki] The two differ only in their reference antenna: **ERP** is referenced to a
half-wave [dipole](/reference/dipole-antenna/) (gain in **dBd**), while **EIRP** is referenced
to an ideal isotropic radiator (gain in **dBi**). Because a lossless dipole already has 2.15 dBi
of gain over isotropic, **EIRP = ERP + 2.15 dB** for the same system.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A transmitter feeds a lossy cable into a gain antenna; output power minus cable loss plus antenna gain equals EIRP, the power an isotropic source would need to match the peak radiated field." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ee-ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="10" fill="currentColor" stroke="none">
    <rect x="20" y="60" width="70" height="34" fill="none" stroke="currentColor"/>
    <text x="30" y="80">TX</text>
    <text x="30" y="91">+40 dBm</text>
    <line x1="90" y1="77" x2="150" y2="77" stroke="currentColor" marker-end="url(#ee-ar)"/>
    <text x="96" y="70">−2 dB cable</text>
    <rect x="150" y="60" width="70" height="34" fill="none" stroke="currentColor"/>
    <text x="160" y="80">Antenna</text>
    <text x="160" y="91">+9 dBi</text>
    <line x1="220" y1="77" x2="290" y2="77" stroke="currentColor" marker-end="url(#ee-ar)"/>
    <text x="300" y="74">EIRP =</text>
    <text x="300" y="88">+47 dBm</text>
    <path d="M300 100 q 60 20 130 0" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="150" y="130" font-style="italic">EIRP = 40 − 2 + 9 = 47 dBm  (ERP = 44.85 dBm)</text>
    <text x="20" y="150" fill-opacity="0.7">Peak main-lobe power an isotropic source would need to match this system.</text>
  </g>
</svg>
<figcaption>EIRP sums transmitter power, feedline loss, and antenna gain into one figure; ERP is the same quantity referenced to a dipole (2.15 dB lower).</figcaption>
</figure>

## How it works

Both quantities are built by adding [decibels](/reference/decibel/) along the transmit chain.
Start with the power delivered by the transmitter (say +40 [dBm](/reference/dbm/)), subtract the
loss of the feedline and connectors, then add the [antenna gain](/reference/antenna-gain/):

- **EIRP (dBm) = P_tx − L_feed + G (dBi)**
- **ERP (dBm) = P_tx − L_feed + G (dBd)**, with **G(dBd) = G(dBi) − 2.15**

The key idea is that antenna gain is not amplification — a passive antenna radiates no more total
power than it receives. Gain describes *directivity*: energy is concentrated into a narrower beam,
so along the main lobe the field is stronger than an omnidirectional reference would produce. ERP
and EIRP capture exactly that peak-direction equivalent power. Away from the main lobe the radiated
power is lower, which is why a high-EIRP directional link can be invisible off to the side.

Because the reference antennas differ by a fixed 2.15 dB, the two scales are trivially converted.
Broadcasters and land-mobile regulators traditionally quote **ERP** (dipole reference is natural
for VHF/UHF whip and dipole systems); satellite, microwave, and radar engineers quote **EIRP**
(isotropic reference is cleaner for [link budgets](/reference/link-budget/) and antenna-theory math).

## In practice

Regulators cap **EIRP or ERP**, not raw transmitter power, because what matters for interference and
exposure is the field actually radiated. A Wi-Fi rule of "36 dBm EIRP" lets you trade a bigger
antenna against a smaller amplifier as long as the product stays under the limit. A broadcast license
of "50 kW ERP" fixes the effective coverage regardless of how much of that comes from transmitter
power versus antenna gain. This is also why EIRP is the natural starting point for a
[link budget](/reference/link-budget/): it is precisely the term you plug in before subtracting
[free-space path loss](/reference/free-space-path-loss/) on the way to the receiver.

## Relevance to SDR

ERP and EIRP are transmit-side concepts, so a receive-only [SDR](/reference/software-defined-radio/)
never radiates them — but they govern the *signals it hears*. The EIRP of a trunking
[control channel](/reference/control-channel/), a broadcast tower, an [ADS-B](/reference/ads-b/)
transponder, or a satellite downlink sets how strong that signal arrives after path loss, and hence
whether it clears your [noise floor](/reference/noise-floor/) and decodes. Estimating a distant site's
EIRP (published ERP for licensed land-mobile and broadcast transmitters is often on record) lets you
sanity-check expected receive levels and coverage.

**GopherTrunk** is a receiver and does not transmit, so it has no ERP/EIRP of its own. The concept
still matters for planning: a site whose EIRP and distance imply a weak arriving signal will need a
better antenna, [low-noise amplifier](/reference/low-noise-amplifier/), or [antenna gain](/reference/antenna-gain/)
on the receive side to decode reliably.

## Sources

[^wiki]: [Effective radiated power](https://en.wikipedia.org/wiki/Effective_radiated_power) — Wikipedia, definitions of ERP and EIRP, dipole vs isotropic references, and the 2.15 dB relationship.
