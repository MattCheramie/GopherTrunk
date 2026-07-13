---
slug: pulse-shaping
title: Pulse shaping
entry_type: term
category: modulation
description: "Pulse shaping filters each transmitted symbol so the signal occupies less bandwidth and avoids inter-symbol interference — commonly with a root-raised-cosine filter."
keywords: pulse shaping, root-raised-cosine, inter-symbol interference, spectral splatter, Nyquist filter, roll-off factor, raised cosine, matched filter, band-limiting
aka: ["pulse shaping"]
autolink: true
see_also: [root-raised-cosine-filter, roll-off-factor, intersymbol-interference, matched-filter, symbol-rate, bandwidth]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Pulse_shaping
  - https://en.wikipedia.org/wiki/Raised-cosine_filter
---

**Pulse shaping** is filtering each transmitted symbol's pulse so the signal occupies
**less bandwidth** and successive symbols don't smear into one another
([inter-symbol interference](/reference/intersymbol-interference/)).[^wiki] Sharp
rectangular pulses spray energy into adjacent channels; a shaped pulse keeps it contained.
It is a mandatory step in almost every band-limited digital radio, because the two goals —
a narrow spectrum and no ISI — are in tension and pulse shaping is how they are reconciled.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A rectangular pulse with a wide spectrum versus a shaped pulse with a narrow spectrum." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 90 V50 H80 V90" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5"/>
  <text x="55" y="106" text-anchor="middle" font-size="8" fill="currentColor">rectangular</text>
  <path d="M150 90 Q 185 90 195 55 Q 205 35 215 55 Q 225 90 260 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="205" y="106" text-anchor="middle" font-size="8" fill="currentColor">shaped</text>
  <line x1="300" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M300 88 L350 88 L360 78 L370 88 L420 88" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <path d="M300 88 C 340 88 340 50 360 30 C 380 50 380 88 420 88" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" stroke-dasharray="3 2"/>
  <text x="360" y="106" text-anchor="middle" font-size="8" fill="currentColor">narrow vs wide spectrum</text>
</svg>
<figcaption>Shaping each symbol pulse (often with a root-raised-cosine filter) contains the signal's spectrum.</figcaption>
</figure>

## How it works

A digital symbol stream is a sequence of impulses, one per symbol. Sent as-is — or as flat
rectangular pulses — the signal's spectrum is a *sinc* with slowly decaying sidelobes that leak
far into neighbouring channels, causing "spectral splatter." The naive fix is to lowpass-filter
hard, but any filter that narrows the spectrum also spreads each symbol's pulse in time, so it now
overlaps its neighbours at their sampling instants and corrupts them: that is ISI. Pulse shaping
resolves the dilemma by choosing a pulse whose spectrum is compact **and** whose time-domain shape
has **zeros at every other symbol's sampling instant**.

That second property is the **Nyquist ISI criterion**. A pulse satisfying it is non-zero at its
own symbol time but exactly zero at all the neighbouring symbol times, so however much its tails
ring, they contribute nothing when each symbol is sampled at the right instant — the eye stays
open. The universal family that meets the criterion with a tunable bandwidth is the **raised-cosine**
pulse, whose excess bandwidth beyond the theoretical minimum is set by the
[roll-off factor](/reference/roll-off-factor/) β (0 = sharpest spectrum, hardest to build;
higher β = gentler, wider, more robust to timing error).

## Variants

In practice the raised-cosine response is **split** between transmitter and receiver: each applies
a [root-raised-cosine (RRC)](/reference/root-raised-cosine-filter/) filter, and the cascade of the
two — the transmit shaping filter followed by the receive
[matched filter](/reference/matched-filter/) — multiplies to the full raised-cosine response.
That arrangement simultaneously satisfies the Nyquist criterion (zero ISI) *and* maximises
signal-to-noise ratio at the sampling instant, which a single filter at either end cannot do.
Other shapes exist — Gaussian pulses (used by [GFSK](/reference/gfsk/) and
[GMSK](/reference/gmsk/), which deliberately accept a little ISI for an even tighter spectrum),
and simple sinc or rectangular pulses in undemanding links — but RRC dominates linear PSK/QAM
systems.

## In practice

Real systems specify the shaping filter exactly so that transmitter and receiver agree.
[P25](/reference/p25-phase-1/) C4FM uses a shaping filter with a raised-cosine-like response;
[TETRA](/reference/tetra/)'s [π/4-DQPSK](/reference/pi-4-dqpsk/) uses RRC with roll-off 0.35; many
cellular and satellite links quote a specific β (0.2–0.35) as part of the air interface. The
chosen roll-off is a direct spectrum-versus-robustness decision: tighter roll-off packs channels
closer but leaves less timing margin in the [eye diagram](/reference/eye-diagram/).

## Relevance to SDR

A software receiver must apply the *matching* receive filter — usually the same RRC the standard
specifies — to see a clean [constellation](/reference/constellation-diagram/) and open eye;
skipping it or using the wrong roll-off leaves residual ISI that smears the symbols and raises the
error rate. GopherTrunk's demod chain includes the receive-side matched pulse-shaping filter for
the linear modulations it handles, sized to the protocol's [symbol rate](/reference/symbol-rate/)
so that the combined transmit-plus-receive response meets the zero-ISI Nyquist condition at the
slicer.

## Sources

[^wiki]: [Pulse shaping](https://en.wikipedia.org/wiki/Pulse_shaping) — Wikipedia, for the bandwidth-limiting and inter-symbol-interference rationale.
[^rc]: [Raised-cosine filter](https://en.wikipedia.org/wiki/Raised-cosine_filter) — Wikipedia, for the Nyquist ISI criterion, the roll-off parameter, and the split RRC arrangement.
