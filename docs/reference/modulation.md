---
slug: modulation
title: Modulation
entry_type: term
category: rf-fundamentals
description: Modulation is the process of varying a carrier wave's amplitude, frequency, or phase to encode information for transmission over radio.
keywords: modulation, AM FM PSK FSK, carrier, encoding information, digital modulation, symbol, constellation
infobox:
  - { label: Type, value: Signal-processing concept }
  - { label: Varies, value: Amplitude, frequency, or phase }
  - { label: Families, value: Analog (AM/FM/SSB), digital (FSK/PSK/QAM) }
see_also: [carrier-wave, amplitude-modulation, frequency-modulation, phase-shift-keying, frequency-shift-keying, quadrature-amplitude-modulation, spectral-efficiency]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Modulation
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
---

**Modulation** is the process of varying a property of a
[carrier wave](/reference/carrier-wave/) — its [amplitude](/reference/amplitude/),
[frequency](/reference/frequency/), or [phase](/reference/phase/) — in step with a
message so that information can travel over radio.[^wiki] It is the bridge between a
baseband signal (audio, data) and the radio-frequency carrier that can actually be
radiated, and the reverse process at the receiver is
[demodulation](/reference/demodulation/). Choosing a modulation scheme is a trade among
data rate, bandwidth, power, and robustness.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A message wave, then an AM version whose height follows the message, then an FM version whose cycle spacing follows the message." xmlns="http://www.w3.org/2000/svg">
  <text x="6" y="22" font-size="10" fill="currentColor">message</text>
  <path d="M70 20 Q120 0 170 20 T270 20 T370 20 T440 20" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="6" y="78" font-size="10" fill="currentColor">AM</text>
  <path d="M70 75 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="6" y="140" font-size="10" fill="currentColor">FM</text>
  <path d="M70 137 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0 q4 -16 8 0 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0 q4 -16 8 0 q4 -16 8 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
</svg>
<figcaption>Modulation encodes a message by varying the carrier — its amplitude (AM), frequency (FM), or phase.</figcaption>
</figure>

## How it works

There are only three things about a sinusoidal carrier you can change, and every
modulation scheme is a way of changing one or more of them:

- **Amplitude.** [AM](/reference/amplitude-modulation/) writes the message into the
  carrier's envelope. [SSB](/reference/single-sideband/) is a bandwidth- and
  power-efficient variant that transmits one sideband only.
- **Frequency.** [FM](/reference/frequency-modulation/) shifts the carrier frequency
  with the message; its constant envelope makes it resistant to amplitude noise, the
  reason it sounds cleaner than AM.
- **Phase.** Phase modulation nudges the carrier's timing; it is closely related to FM
  and underlies most digital schemes.

Analog modulation varies a property continuously. Digital modulation instead switches
the carrier among a finite set of **symbols**, each standing for one or more bits:
[FSK](/reference/frequency-shift-keying/) toggles between discrete frequencies,
[PSK](/reference/phase-shift-keying/) between discrete phases, and
[QAM](/reference/quadrature-amplitude-modulation/) among combinations of amplitude and
phase. The symbols are naturally drawn on a
[constellation diagram](/reference/constellation-diagram/), the map of the IQ plane the
receiver uses to decide which symbol arrived. Packing more bits per symbol (more
constellation points) raises [spectral efficiency](/reference/spectral-efficiency/) but
shrinks the spacing between points, so it needs more signal-to-noise to keep them
distinct — the fundamental power-versus-rate trade that
[Shannon capacity](/reference/shannon-capacity/) bounds.

## In practice

Modulation is chosen to fit the channel and the job. Low, robust orders survive weak or
fading links: land-mobile digital voice uses [C4FM](/reference/c4fm/) /
four-level FSK (DMR, NXDN, P25 Phase 1's alternate) and
[π/4-DQPSK](/reference/pi-4-dqpsk/) precisely because they hold up at modest SNR.
[Pulse shaping](/reference/pulse-shaping/) with a
[root-raised-cosine filter](/reference/root-raised-cosine-filter/) keeps each symbol from
smearing into its neighbours and contains the
[occupied bandwidth](/reference/occupied-bandwidth/). High-throughput systems (LTE, Wi-Fi,
DVB) climb to 16-, 64-, or 256-QAM over [OFDM](/reference/ofdm/) subcarriers when the
channel is good, and adapt downward when it degrades.

## Relevance to SDR

Recognising a signal's modulation and applying the matching
[demodulator](/reference/demodulation/) is the heart of decoding. GopherTrunk's chain
identifies the trunking waveform, then runs the appropriate symbol recovery: four-level
FSK slicing for DMR/NXDN, π/4-DQPSK carrier-and-symbol tracking for P25. The three
carrier properties reappear as the axes of the [IQ](/reference/iq-data/) plane, so once
the samples are in software, demodulation is a matter of measuring amplitude, frequency,
and phase and mapping them back to symbols and bits.

## Sources

[^wiki]: [Modulation](https://en.wikipedia.org/wiki/Modulation) — Wikipedia, overview of analog and digital modulation methods.
[^qam]: [Quadrature amplitude modulation](https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation) — Wikipedia, on jointly modulating amplitude and phase and the rate/robustness trade of higher-order constellations.
