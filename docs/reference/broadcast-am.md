---
slug: broadcast-am
title: Broadcast AM
entry_type: technology
category: broadcast
description: "Broadcast AM is amplitude-modulated radio on the medium-wave and long-wave bands, recoverable by simple envelope detection."
keywords: broadcast AM, AM radio, medium wave, MW band, long wave, LW band, envelope detection, amplitude modulation, 530-1710 kHz, AM broadcast band
aka: [AM radio, AM broadcast, medium wave, MW, long wave, LW]
autolink: true
infobox:
  - { label: Type, value: Analog broadcast modulation }
  - { label: Bands, value: "LW 148–283 kHz, MW 530–1710 kHz" }
  - { label: Idea, value: Carrier amplitude tracks the audio envelope }
  - { label: Detection, value: Envelope (diode) or synchronous }
see_also: [amplitude-modulation, sky-wave, subcarrier, modulation, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/AM_broadcasting
  - https://en.wikipedia.org/wiki/Medium_wave
---

**Broadcast AM** is the oldest mass radio service, carrying audio by
[amplitude modulation](/reference/amplitude-modulation/) of a carrier on the
long-wave (148–283 kHz) and medium-wave (roughly 530–1710 kHz) bands.[^wiki] The
audio rides directly on the carrier's amplitude, so the transmitted envelope is a
scaled copy of the modulating waveform — which means a receiver can recover the sound
with nothing more than a diode and a capacitor, the property that made AM the
foundation of early radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="An amplitude-modulated carrier whose envelope traces the audio waveform, with a dashed line marking the envelope that a diode detector follows." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 80 Q120 20 220 80 T420 80" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <path d="M20 80 Q120 140 220 80 T420 80" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <path d="M20 80 L26 62 L32 98 L38 60 L44 100 L50 56 L56 104 L62 54 L68 106 L74 55 L80 105 L86 58 L92 102 L98 62 L104 98 L110 68 L116 92 L122 72 L128 88 L134 74 L140 86 L146 74 L152 86 L158 70 L164 90 L170 64 L176 96 L182 58 L188 102 L194 54 L200 106 L206 53 L212 107 L218 54 L224 106 L230 58 L236 102 L242 63 L248 97 L254 69 L260 91 L266 73 L272 87 L278 74 L284 86 L290 72 L296 88 L302 67 L308 93 L314 61 L320 99 L326 57 L332 103 L338 55 L344 105 L350 56 L356 104 L362 60 L368 100 L374 66 L380 94 L386 72 L392 88 L398 76 L404 84 L410 78 L416 82 L420 80" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="240" y="150" text-anchor="middle" font-size="9" fill="currentColor">carrier amplitude follows the audio envelope (dashed)</text>
</svg>
<figcaption>In AM the carrier's envelope is a copy of the audio; a diode detector simply follows that envelope.</figcaption>
</figure>

## How it works

An AM transmitter multiplies a constant-frequency carrier by (1 + m·audio), where the
modulation index *m* stays below 1 to avoid over-modulation distortion. The result is a
carrier flanked by two mirror-image sidebands, each spanning the audio bandwidth; a
double-sideband AM signal is therefore about twice the audio bandwidth wide, and
broadcast channels are spaced 9 kHz (ITU regions) or 10 kHz (the Americas) apart,
limiting fidelity to a few kHz of audio.

Because the envelope directly represents the audio, the classic **envelope detector** —
a diode that rectifies the RF and a resistor–capacitor network that smooths it — recovers
the sound without any local oscillator or phase reference. This simplicity is AM's
defining virtue and its weakness: the same detector responds to any amplitude change,
so static crashes, fading, and electrical noise pass straight through. A
*synchronous* detector, which regenerates a phase-locked carrier and multiplies it back
against the signal, improves noise and fading performance at the cost of complexity.
Some AM stations add a low-level [subcarrier](/reference/subcarrier/) (AMSS or C-QUAM
stereo pilot tone) above the audio.

## Relevance to SDR

AM is a natural SDR demodulation exercise: take the magnitude of the complex baseband
[IQ](/reference/software-defined-radio/) samples and you have the envelope, no carrier
recovery required — the software equivalent of the diode. Long- and medium-wave AM also
showcase propagation: by day the [ground wave](/reference/ground-wave/) reaches perhaps
a few hundred kilometres, while after dark the [sky wave](/reference/sky-wave/) refracts
off the ionosphere and distant stations pile in, making MW DXing a popular pastime for
direct-sampling SDRs that reach down to the low kHz.

**GopherTrunk** targets VHF/UHF trunked land-mobile systems and does not decode
broadcast AM. The mode remains useful background: envelope detection is the conceptual
root of the amplitude-shift-keying and magnitude-based detectors that appear elsewhere
in radio, and the MW band is a demanding test of an SDR front end's dynamic range at
night when many strong signals coexist.

## Sources

[^wiki]: [AM broadcasting](https://en.wikipedia.org/wiki/AM_broadcasting) — Wikipedia, for the long-wave and medium-wave bands, amplitude modulation with double sidebands, 9/10 kHz channel spacing, and envelope detection.
