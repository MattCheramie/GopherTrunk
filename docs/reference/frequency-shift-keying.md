---
slug: frequency-shift-keying
title: Frequency-shift keying (FSK)
entry_type: technology
category: modulation
description: Frequency-shift keying (FSK) is digital modulation that switches a carrier between discrete frequencies; four-level FSK (4FSK) underlies P25 C4FM, DMR, and NXDN.
keywords: FSK, frequency shift keying, 2FSK, 4FSK, C4FM, digital modulation, mark space, CPFSK, MSK, GFSK, deviation
aka: [frequency-shift keying, FSK, 4FSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Carrier frequency (discrete) }
  - { label: Used by, value: P25, DMR, NXDN, paging }
see_also: [phase-shift-keying, quadrature-amplitude-modulation, c4fm, gmsk, afsk, ffsk, symbol-rate, four-fsk, minimum-shift-keying, continuous-phase-modulation, fm-deviation, m-ary-fsk]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
  - https://en.wikipedia.org/wiki/Continuous-phase_frequency-shift_keying
---

**Frequency-shift keying** (**FSK**) is digital [modulation](/reference/modulation/)
that switches a [carrier](/reference/carrier-wave/) between a fixed set of frequencies,
one per [symbol](/reference/symbol-rate/).[^wiki] It is the digital descendant of
[FM](/reference/frequency-modulation/): where FM sweeps the carrier continuously, FSK
hops it among discrete tones. Two frequencies gives binary 2FSK; four gives
[4FSK](/reference/four-fsk/), the workhorse of digital land-mobile radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A bit stream above, and below it a carrier that switches between a low frequency for zeros and a high frequency for ones." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor" font-family="monospace"><text x="40" y="24">1</text><text x="140" y="24">0</text><text x="240" y="24">1</text><text x="340" y="24">0</text></g>
  <path d="M30 80 q6 -26 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0
            M130 80 q12 -26 24 0 t24 0 t24 0 t24 0
            M230 80 q6 -26 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0
            M330 80 q12 -26 24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="30" y="115" font-size="9" fill="currentColor">each bit picks a frequency (here 2FSK; 4FSK uses four)</text>
</svg>
<figcaption>FSK switches the carrier between set frequencies; four-level 4FSK underlies P25 C4FM and DMR.</figcaption>
</figure>

## How it works

Each symbol selects one frequency offset (a **deviation**) from the carrier centre. In
2FSK a "space" and a "mark" tone carry one bit each. In 4FSK four evenly spaced
deviations — for P25 at ±600 Hz and ±1800 Hz — each stand for a two-bit **dibit**, so the
scheme carries 2 bits per symbol and doubles throughput for the same symbol rate. Because
the transmitted waveform is always a constant-amplitude sinusoid whose only property that
changes is frequency, FSK has a **constant envelope**: it can be amplified by cheap,
efficient, saturated (Class C) power amplifiers without distortion, which is exactly why
portable and mobile radios favour it over amplitude-varying schemes like
[QAM](/reference/quadrature-amplitude-modulation/) that demand a linear amplifier.

The quality-defining subtlety is *phase continuity*. Naively switching between two free
oscillators produces phase jumps at each symbol boundary, which splatter energy into
adjacent channels. Practical radio FSK is **continuous-phase FSK** (CPFSK): the same
oscillator's frequency is steered so the phase never jumps, giving a far more compact
spectrum. Pushing this further with pulse shaping yields the
[continuous-phase modulation](/reference/continuous-phase-modulation/) family —
[GMSK](/reference/gmsk/), [MSK](/reference/minimum-shift-keying/), and P25's
[C4FM](/reference/c4fm/) — which is why those schemes are best understood as filtered
FSK rather than as something separate.

## Variants

- **2FSK / M-ary FSK** — binary FSK is simplest; [M-ary FSK](/reference/m-ary-fsk/)
  uses 2, 4, 8, or more tones, trading bandwidth for bits per symbol.
- **[4FSK](/reference/four-fsk/)** — four levels, 2 bits/symbol; the physical layer of
  [DMR](/reference/dmr/), [NXDN](/reference/nxdn/), and P25 Phase 1 C4FM.
- **[GFSK](/reference/gfsk/)** — Gaussian-filtered FSK used by Bluetooth and many ISM
  radios.
- **[MSK](/reference/minimum-shift-keying/) / [GMSK](/reference/gmsk/)** — the minimum-
  deviation, phase-continuous limit; GMSK carries GSM and [AIS](/reference/ais/).
- **Audio FSK** — [AFSK](/reference/afsk/) and coherent [FFSK](/reference/ffsk/) key
  audio tones that then FM-modulate an ordinary voice radio.

## Relevance to SDR

An FSK demodulator recovers the instantaneous frequency of the complex
[IQ](/reference/iq-data/) baseband — the same phase-differentiation an
[FM](/reference/frequency-modulation/) discriminator does — then slices the result at
the symbol instants into levels. The recovered symbol levels appear as three stacked
openings on a [symbol scope / eye diagram](/reference/eye-diagram/) and as clusters on a
[constellation](/reference/constellation-diagram/). GopherTrunk's core decode chain is
built on exactly this: it demodulates the 4FSK/C4FM physical layer of P25 Phase 1, DMR,
and NXDN, recovering dibits from the four deviation levels before framing and vocoder
decode.

## Sources

[^wiki]: [Frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying) — Wikipedia, for the definition and the 2FSK/4FSK variants.
[^cpfsk]: [Continuous-phase frequency-shift keying](https://en.wikipedia.org/wiki/Continuous-phase_frequency-shift_keying) — Wikipedia, for phase continuity and its effect on the transmitted spectrum.
