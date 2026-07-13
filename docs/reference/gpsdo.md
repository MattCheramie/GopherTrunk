---
slug: gpsdo
title: GPSDO (GPS-disciplined oscillator)
entry_type: hardware
category: rf-front-end
description: "A GPSDO steers a local OCXO to the atomic-clock timing broadcast by GPS satellites, giving long-term frequency accuracy near 1e-12 — the reference standard for coherent SDR."
keywords: GPSDO, GPS-disciplined oscillator, disciplined oscillator, 10 MHz reference, PPS, one pulse per second, frequency standard, atomic accuracy, coherent SDR
aka: [GPSDO, "GPS-disciplined oscillator", "GPS disciplined oscillator"]
autolink: true
infobox:
  - { label: Type, value: "Disciplined reference oscillator" }
  - { label: Accuracy, value: "~1e-11 to 1e-12 long-term" }
  - { label: Inputs, value: "GNSS antenna + PPS" }
  - { label: Outputs, value: "10 MHz + 1 PPS typical" }
  - { label: TX, value: "N/A (reference)" }
see_also: [ocxo, gps-gnss, frequency-stability, tcxo, local-oscillator]
cite_urls:
  - https://en.wikipedia.org/wiki/GPS_disciplined_oscillator
---

A **GPSDO** (GPS-disciplined oscillator) is a local oscillator — usually an
[OCXO](/reference/ocxo/) — whose long-term frequency is continuously **steered to the
atomic-clock timing** carried by [GPS](/reference/gps-gnss/) satellites.[^wiki] It fuses two
strengths: the clean short-term stability of a good crystal oven and the near-perfect
long-term accuracy of the GPS time base, reaching fractional accuracies around **1e-11 to
1e-12** over a day. That makes it the practical gold-standard [reference](/reference/local-oscillator/)
for SDR work that must be exactly on frequency and, when several radios share it, phase
coherent.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A GPS antenna feeds a receiver that produces a one-pulse-per-second signal; a control loop compares it to an OCXO and steers the OCXO, outputting a disciplined 10 MHz and PPS reference." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gpsdoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 25 l14 20 l-28 0 z" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="40" y1="45" x2="40" y2="70" stroke="currentColor" stroke-width="1.6" marker-end="url(#gpsdoar)"/>
  <text x="40" y="18" font-size="8" fill="currentColor" text-anchor="middle">GNSS</text>
  <rect x="15" y="70" width="70" height="34" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="50" y="91" font-size="8" fill="currentColor" text-anchor="middle">GPS rx</text>
  <line x1="85" y1="87" x2="150" y2="87" stroke="currentColor" stroke-width="1.6" marker-end="url(#gpsdoar)"/>
  <text x="117" y="80" font-size="8" fill="currentColor" text-anchor="middle">1 PPS</text>
  <rect x="150" y="60" width="80" height="54" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="190" y="84" font-size="8" fill="currentColor" text-anchor="middle">phase</text>
  <text x="190" y="96" font-size="8" fill="currentColor" text-anchor="middle">compare</text>
  <rect x="150" y="130" width="80" height="34" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="190" y="151" font-size="8" fill="currentColor" text-anchor="middle">OCXO</text>
  <line x1="190" y1="130" x2="190" y2="114" stroke="currentColor" stroke-width="1.6" marker-end="url(#gpsdoar)"/>
  <path d="M230 90 q40 0 40 30 v10 q0 25 -40 25 h-2" fill="none" stroke="currentColor" stroke-width="1.4" marker-end="url(#gpsdoar)"/>
  <text x="285" y="120" font-size="8" fill="currentColor">steer</text>
  <line x1="230" y1="147" x2="360" y2="147" stroke="currentColor" stroke-width="1.6" marker-end="url(#gpsdoar)"/>
  <rect x="360" y="128" width="90" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="405" y="152" font-size="8.5" fill="currentColor" text-anchor="middle">10 MHz + PPS</text>
</svg>
<figcaption>A GPSDO compares its OCXO against the satellite-derived 1 PPS and slowly steers the oven so long-term frequency tracks GPS atomic time.</figcaption>
</figure>

## Overview

GPS satellites carry atomic clocks and broadcast time so precisely that a ground receiver can
recover a **one-pulse-per-second (1 PPS)** tick whose long-term rate is essentially exact.
The problem is that GPS timing is noisy second-to-second and can drop out entirely. A crystal
[OCXO](/reference/ocxo/) is the opposite: beautifully quiet over seconds but slowly aging over
days. A GPSDO marries them, using a slow control loop so each covers the other's weakness.

## How it works

- A GPS receiver produces a 1 PPS output locked to satellite atomic time.
- A phase/frequency comparator measures how far the local OCXO has drifted against that PPS,
  accumulated over a long time constant (seconds to many minutes).
- The loop applies a small steering voltage to the OCXO, nudging it back on track. Because
  the loop is slow, it corrects the OCXO's long-term aging while the OCXO's own short-term
  stability filters out the GPS jitter.
- **Holdover**: if GPS is lost, the loop freezes its last correction and the OCXO coasts,
  drifting only slowly until lock returns.

## In practice

A typical GPSDO outputs a disciplined **10 MHz** sine and a 1 PPS tick, both used to clock and
synchronise instruments and radios. It needs a sky-view GNSS [antenna](/reference/gps-gnss/)
and a few minutes (sometimes longer) to acquire satellites and settle its loop. The
short-term [frequency stability](/reference/frequency-stability/) is set by the OCXO, so a
better oven still matters; GPS supplies the long-term truth.

## Relevance to SDR

A GPSDO is what a serious SDR station uses when frequency error must be negligible and,
especially, when **multiple receivers must share one reference** — coherent direction finding,
passive radar, or multi-band monitoring where all radios must agree on frequency and phase.
SDRs with a 10 MHz external-reference input lock straight to it. GopherTrunk decodes the
samples the front end delivers and does not drive the oscillator, but a GPSDO removes reference
drift as a variable entirely: the control channel sits exactly where the system map says it
should, and nothing the software's automatic frequency correction does has to compensate for a
wandering clock. Below a GPSDO, an [OCXO](/reference/ocxo/) or [TCXO](/reference/tcxo/) covers
less demanding stations.

## Sources

[^wiki]: [GPS disciplined oscillator](https://en.wikipedia.org/wiki/GPS_disciplined_oscillator) — Wikipedia, on disciplining an OCXO to GPS 1 PPS, long-term accuracy, holdover, and 10 MHz/PPS outputs.
