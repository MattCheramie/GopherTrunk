---
slug: crest-factor-papr
title: Crest Factor & PAPR
entry_type: term
category: rf-fundamentals
description: Crest factor and PAPR measure how far a waveform's peaks exceed its average power, forcing amplifier backoff — a defining challenge of OFDM and multi-carrier signals.
keywords: crest factor, PAPR, peak-to-average power ratio, peak to average, amplifier backoff, OFDM PAPR, envelope, headroom, clipping, power efficiency
aka: [crest factor, PAPR, peak-to-average power ratio, peak-to-average ratio]
autolink: true
infobox:
  - { label: Type, value: Peak-to-average amplitude ratio }
  - { label: PAPR, value: "10·log10(P_peak / P_avg) dB" }
  - { label: Drives, value: Amplifier backoff & headroom }
see_also: [ofdm, power-amplifier, 1-db-compression-point, quadrature-amplitude-modulation, root-raised-cosine-filter, error-vector-magnitude]
cite_urls:
  - https://en.wikipedia.org/wiki/Crest_factor
  - https://en.wikipedia.org/wiki/PAPR
---

**Crest factor** is the ratio of a waveform's peak amplitude to its RMS value, and
**peak-to-average power ratio (PAPR)** is the same idea in power terms — how far the
instantaneous peaks rise above the average.[^wiki] PAPR is normally quoted in dB as
*10·log₁₀(P_peak / P_avg)*, and it matters because a
[power amplifier](/reference/power-amplifier/) must stay linear all the way up to the
peaks, forcing it to run "backed off" from its most efficient operating point.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A fluctuating waveform envelope whose average level is low but which has occasional tall peaks reaching near a clipping ceiling, illustrating high peak-to-average power ratio." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" font-size="10">
    <line x1="40" y1="20" x2="40" y2="140" stroke-width="1"/>
    <line x1="40" y1="140" x2="440" y2="140" stroke-width="1"/>
    <line x1="40" y1="35" x2="440" y2="35" stroke-dasharray="4 3" stroke-opacity="0.7"/>
    <line x1="40" y1="100" x2="440" y2="100" stroke-dasharray="2 3" stroke-opacity="0.6"/>
    <path d="M40 100 Q70 95 90 98 T140 92 Q160 40 175 45 Q190 95 210 98 T270 90 Q300 30 315 33 Q330 96 360 99 T420 95" stroke-width="1.6"/>
  </g>
  <g fill="currentColor" font-size="10" stroke="none">
    <text x="300" y="31">clipping / P_peak</text>
    <text x="300" y="96">P_avg</text>
    <text x="42" y="155">time →</text>
  </g>
</svg>
<figcaption>A high-PAPR signal spends most time near its average but throws occasional tall peaks toward the clipping ceiling; the amplifier must be linear all the way up to them.</figcaption>
</figure>

## How it works

A constant-envelope signal — an FM carrier, or a [GMSK](/reference/gmsk/) waveform — has
a PAPR near 0 dB: its amplitude never changes, so an amplifier can be run right up into
saturation for maximum efficiency without distorting it. As soon as a modulation varies
its amplitude, peaks appear. Filtered [QAM](/reference/quadrature-amplitude-modulation/)
and [root-raised-cosine](/reference/root-raised-cosine-filter/) pulse shaping add a few
dB of PAPR because the filter overshoots between symbols.

The extreme case is **[OFDM](/reference/ofdm/)**, where many subcarriers are summed. When
a large number of independent [subcarriers](/reference/subcarrier/) happen to align in
phase, their voltages add coherently and produce a brief peak far above the average —
PAPR of 10–13 dB is typical for a wideband OFDM signal. Statistically these alignments
are rare, but the amplifier must handle them without clipping, or the resulting
distortion regrows [spurious](/reference/spurious-emissions/) energy into adjacent
channels and raises the [error vector magnitude](/reference/error-vector-magnitude/).

## In practice

The design response is **backoff**: operate the amplifier several dB below its
[1-dB compression point](/reference/1-db-compression-point/) so even the peaks stay
linear. Backoff wastes efficiency — a power amplifier backed off 8 dB for an OFDM signal
may run at well under 20 % efficiency, draining batteries and generating heat. Several
techniques fight back: **clipping and filtering** shaves the rare peaks at a small cost
in distortion; **tone reservation** and **selective mapping** reshape the OFDM symbol to
lower its peaks; and **DPD (digital predistortion)** inverts the amplifier's curve so it
can run closer to saturation. The choice trades power efficiency against complexity and
signal cleanliness.

Crest factor is distinct from [duty cycle](/reference/duty-cycle/): PAPR describes the
amplitude spread *while transmitting*, whereas duty cycle describes the fraction of time
transmitting at all.

## Relevance to SDR

PAPR shapes real air interfaces. Single-carrier land-mobile systems (P25
[C4FM](/reference/c4fm/), DMR [four-FSK](/reference/four-fsk/)) deliberately use
near-constant-envelope modulation so portable radios can use efficient saturated
amplifiers and preserve battery life. The high-PAPR penalty of OFDM is exactly why LTE
uses OFDM on the downlink (base stations can afford linear, backed-off amplifiers) but
switches to lower-PAPR SC-FDMA on the uplink to spare the handset. TETRA's
[π/4-DQPSK](/reference/pi-4-dqpsk/) is chosen partly to keep its envelope variation
modest.

**GopherTrunk** is a receiver, so it never amplifies a transmit signal and has no PAPR
budget of its own. On the receive side the same statistics still matter: a high-PAPR
signal needs adequate ADC headroom so its peaks are not clipped, which is why
[dBFS](/reference/dbfs/) gain staging and avoiding front-end overload are part of getting
a clean decode.

## Sources

[^wiki]: [Crest factor](https://en.wikipedia.org/wiki/Crest_factor) — Wikipedia, definition of peak-to-RMS ratio and its power form.
[^papr]: [PAPR](https://en.wikipedia.org/wiki/PAPR) — Wikipedia, peak-to-average power ratio and its significance for OFDM amplifiers.
