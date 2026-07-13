---
slug: tetrapol
title: TETRAPOL
entry_type: protocol
category: land-mobile-trunking
description: TETRAPOL is an FDMA digital professional mobile radio standard using GMSK modulation, used by many European police and public-safety networks distinct from TETRA.
keywords: TETRAPOL, PMR, FDMA, GMSK, digital trunked radio, public safety, RTP-based vocoder, EADS, Airbus, Rubis Acropol
aka: [TETRAPOL]
autolink: true
infobox:
  - { label: Type, value: Digital trunked PMR (public safety) }
  - { label: Standards body, value: "TETRAPOL Forum / PAS" }
  - { label: Introduced, value: "1990s" }
  - { label: Access, value: FDMA (one carrier per channel) }
  - { label: Channel spacing, value: 10 / 12.5 kHz }
  - { label: Modulation, value: GMSK (8000 bps) }
  - { label: Vocoder, value: "RPCELP (analysis-by-synthesis)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [tetra, gmsk, fdma, continuous-phase-modulation, trunked-radio, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/TETRAPOL
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
---

**TETRAPOL** is a digital **professional mobile radio** (PMR) standard for trunked
public-safety networks that, unlike [TETRA](/reference/tetra/), uses
**[FDMA](/reference/fdma/)** — one radio carrier per channel — and
**[GMSK](/reference/gmsk/)** modulation. It is a distinct, competing technology to TETRA,
widely deployed by police, gendarmerie, and security services across Europe and beyond.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="TETRAPOL FDMA: several narrow GMSK carriers stacked side by side in frequency, each a separate channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="440" y2="105" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#tp_ar)"/>
  <text x="235" y="128" text-anchor="middle" font-size="9" fill="currentColor">frequency → · one GMSK carrier per 10–12.5 kHz channel</text>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="45" y="35" width="55" height="60" fill="currentColor" fill-opacity="0.22"/>
    <rect x="110" y="45" width="55" height="50" fill="currentColor" fill-opacity="0.14"/>
    <rect x="175" y="30" width="55" height="65" fill="currentColor" fill-opacity="0.22"/>
    <rect x="240" y="50" width="55" height="45" fill="currentColor" fill-opacity="0.14"/>
    <rect x="305" y="38" width="55" height="57" fill="currentColor" fill-opacity="0.22"/>
    <rect x="370" y="48" width="55" height="47" fill="currentColor" fill-opacity="0.14"/>
  </g>
  <text x="235" y="20" text-anchor="middle" font-size="8.5" fill="currentColor">FDMA — channels separated in frequency, not time</text>
  <defs><marker id="tp_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>TETRAPOL is FDMA: each user channel is its own narrow GMSK carrier, contrasting with TETRA's four-slot TDMA.</figcaption>
</figure>

## Overview

TETRAPOL grew out of a French system (Matra Communication, later EADS/Airbus) and became
the second major digital public-safety standard in Europe alongside TETRA. Its defining
architectural choice is FDMA rather than TDMA: every logical channel occupies a separate
narrowband carrier. This keeps the transmitter design simple and gives good
range and building penetration at the cost of the per-channel spectral packing that
TDMA can achieve. Networks are trunked, with dedicated control channels signalling
call setup, registration, and encryption.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | FDMA (one carrier per channel) |
| Channel | 10 kHz (some 12.5 kHz) |
| Modulation | GMSK at 8000 bps |
| Bit shaping | Gaussian-filtered [continuous-phase](/reference/continuous-phase-modulation/) |
| Vocoder | RPCELP analysis-by-synthesis |
| Encryption | Native end-to-end and air-interface options |
| Bands | Typically 70–520 MHz, deployment-dependent |

GMSK is a constant-envelope, continuous-phase scheme, which lets TETRAPOL radios use
efficient non-linear power amplifiers and gives the signal a compact spectrum — the same
modulation family used by GSM and many other PMR systems.

## History

Development began in the late 1980s in France, with the first large networks — the
gendarmerie's **Rubis** and the police **Acropol** systems — deployed through the 1990s.
The TETRAPOL Forum published the Publicly Available Specification (PAS) to encourage
wider adoption. The technology spread to public-safety users in Spain, Switzerland,
Czechia, Mexico, and other countries, positioning itself as the FDMA alternative to
ETSI's TDMA-based TETRA.

## Deployment

TETRAPOL remains in service in numerous national and regional police, gendarmerie, and
emergency networks, particularly in France, Spain, and parts of central and eastern
Europe. Because most of this traffic is encrypted for operational security, it is heard
but rarely intelligible to third parties.

## Decoding it with GopherTrunk

TETRAPOL is **not decoded** by GopherTrunk. Its FDMA/GMSK physical layer differs from the
C4FM and π/4-DQPSK families GopherTrunk targets, and its proprietary vocoder and
pervasive encryption place operational traffic out of reach. GopherTrunk decodes clear
and known-key signals only; TETRAPOL is documented here for identification and context
rather than as a supported decode target. For a supported European public-safety
standard, see [TETRA](/reference/tetra/).

## Sources

[^wiki]: [TETRAPOL](https://en.wikipedia.org/wiki/TETRAPOL) — Wikipedia, for the FDMA architecture, GMSK modulation, public-safety deployments, and its distinction from TETRA.
