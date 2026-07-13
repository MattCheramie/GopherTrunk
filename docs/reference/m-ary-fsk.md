---
slug: m-ary-fsk
title: M-ary FSK
entry_type: technology
category: modulation
description: M-ary FSK (MFSK) is frequency-shift keying with M tones, encoding log2 M bits per symbol; it trades bandwidth for power efficiency and underlies 4FSK and HF data modes.
keywords: M-ary FSK, MFSK, multi-tone FSK, orthogonal signalling, tones, 4FSK, 8FSK, power efficiency, bandwidth, HF data modes
aka: [M-ary FSK, MFSK, multi-tone FSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Carries, value: log2 M bits per symbol (M tones) }
  - { label: Trade-off, value: More bandwidth for less power }
see_also: [frequency-shift-keying, four-fsk, phase-shift-keying, quadrature-amplitude-modulation, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Multiple_frequency-shift_keying
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
---

**M-ary FSK** (**MFSK**) generalises [frequency-shift
keying](/reference/frequency-shift-keying/) to **M distinct tones**, so each transmitted
[symbol](/reference/symbol-rate/) selects one of M frequencies and carries **log₂M
bits**.[^wiki] Binary FSK is the M = 2 case; [4FSK](/reference/four-fsk/) is M = 4 (2
bits per symbol); systems with 8, 16, or more tones push further. Unusually among
modulations, MFSK gets *more* power-efficient as M grows — at the cost of proportionally
more bandwidth.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Eight stacked frequency tones on the vertical axis with a symbol path selecting one tone per time slot, illustrating M-ary FSK with M equal to eight." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.25"><line x1="70" y1="20" x2="440" y2="20"/><line x1="70" y1="35" x2="440" y2="35"/><line x1="70" y1="50" x2="440" y2="50"/><line x1="70" y1="65" x2="440" y2="65"/><line x1="70" y1="80" x2="440" y2="80"/><line x1="70" y1="95" x2="440" y2="95"/><line x1="70" y1="110" x2="440" y2="110"/><line x1="70" y1="125" x2="440" y2="125"/></g>
  <text x="20" y="24" font-size="9" fill="currentColor">f7</text><text x="20" y="128" font-size="9" fill="currentColor">f0</text>
  <g fill="currentColor"><rect x="90" y="62" width="40" height="6"/><rect x="150" y="17" width="40" height="6"/><rect x="210" y="107" width="40" height="6"/><rect x="270" y="47" width="40" height="6"/><rect x="330" y="92" width="40" height="6"/><rect x="390" y="32" width="40" height="6"/></g>
  <text x="70" y="145" font-size="9" fill="currentColor">one of M tones per symbol; log2(M) bits each (M = 8 here)</text>
</svg>
<figcaption>M-ary FSK picks one of M orthogonal tones per symbol; more tones mean more bits per symbol and, with orthogonal spacing, better power efficiency.</figcaption>
</figure>

## How it works

The transmitter assigns each group of log₂M bits to one of M frequencies and emits that
tone for a symbol interval. If the tones are spaced so they are mutually **orthogonal** —
separated by an integer multiple of the symbol rate — a receiver can detect them
non-coherently with a bank of matched filters or an FFT, picking whichever tone has the
most energy. No carrier-phase recovery is needed, which makes MFSK robust on fading and
Doppler-affected channels.

The efficiency story is the reverse of most schemes. As M increases, the tones become an
ever-better approximation of orthogonal signalling, and the energy per bit needed to hit
a given error rate *falls*, approaching the ultimate Shannon limit for large M. The price
is bandwidth: doubling the bits per symbol by doubling M multiplies the number of tones,
so the occupied spectrum grows roughly linearly with M while the data rate grows only
logarithmically. MFSK therefore suits power-limited, bandwidth-rich channels — the
opposite regime from spectrally efficient [QAM](/reference/quadrature-amplitude-modulation/)
or high-order [PSK](/reference/phase-shift-keying/).

## Variants

Land-mobile radio uses the low-M, continuous-phase end of the family: 2FSK and
[4FSK](/reference/four-fsk/) shaped for a constant envelope and narrow channel. HF and
weak-signal amateur data modes sit at the high-M, orthogonal end, where dozens of tones
buy resilience against ionospheric fading. Coherent MFSK and continuous-phase MFSK
(CPFSK) are refinements that trade a little of the non-coherent simplicity for better
spectrum or distance.

## Relevance to SDR

Because MFSK is easy to detect with an FFT, it is a natural fit for software radio and
appears widely: 4FSK voice systems (DMR, NXDN, P25 C4FM), multi-tone HF keyboard modes,
paging, and various telemetry links. On a spectrogram, high-M MFSK is unmistakable — a
staircase of narrow tones hopping between fixed frequencies.

GopherTrunk works at the low-M end: its C4FM/4FSK demodulator recovers the dibit stream
for P25, DMR, and NXDN. The broader M-ary FSK theory is documented here to place that
4-level modulation within the larger orthogonal-signalling family and to explain why
FSK-based systems tolerate cheap, efficient transmitters.

## Sources

[^wiki]: [Multiple frequency-shift keying](https://en.wikipedia.org/wiki/Multiple_frequency-shift_keying) — Wikipedia, for the M-tone definition, orthogonal spacing, and the bandwidth-versus-power trade-off.
