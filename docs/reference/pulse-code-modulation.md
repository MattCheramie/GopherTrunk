---
slug: pulse-code-modulation
title: Pulse-code modulation (PCM)
entry_type: technology
category: modulation
description: Pulse-code modulation (PCM) represents an analog signal as a stream of quantized digital samples taken at a fixed rate; it is the basis of digital audio, telephony, and SDR baseband.
keywords: PCM, pulse code modulation, sampling, quantization, digital audio, telephony, G.711, ADC, bit depth, sample rate, linear PCM
aka: [pulse-code modulation, PCM, linear PCM, LPCM]
autolink: true
infobox:
  - { label: Type, value: Digital representation of analog }
  - { label: Steps, value: Sample, quantize, encode }
  - { label: Used by, value: Digital audio, telephony, SDR baseband }
see_also: [quantization, analog-to-digital-converter, pulse-amplitude-modulation, digital-to-analog-converter, sample-rate, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/Pulse-code_modulation
  - https://en.wikipedia.org/wiki/Nyquist%E2%80%93Shannon_sampling_theorem
---

**Pulse-code modulation** (**PCM**) is the standard method for representing an analog
signal digitally: sample it at a fixed rate, [quantize](/reference/quantization/) each
sample to the nearest of a finite set of levels, and encode that level as a binary
number.[^wiki] The result is a stream of digital *codes* — hence the name — that a
[digital-to-analog converter](/reference/digital-to-analog-converter/) can turn back into
a faithful copy of the original. PCM is the foundation of digital audio, telephony, and
the baseband samples every software radio processes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A smooth analog wave overlaid with vertical sample lines whose heights snap to horizontal quantization levels, each labelled with a binary code, illustrating PCM." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.2"><line x1="40" y1="30" x2="440" y2="30"/><line x1="40" y1="55" x2="440" y2="55"/><line x1="40" y1="80" x2="440" y2="80"/><line x1="40" y1="105" x2="440" y2="105"/><line x1="40" y1="130" x2="440" y2="130"/></g>
  <path d="M40 105 C 110 20, 170 20, 230 80 S 360 150, 440 60" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-width="1.4"/>
  <g fill="currentColor"><rect x="70" y="55" width="5" height="75"/><rect x="130" y="30" width="5" height="100"/><rect x="190" y="30" width="5" height="100"/><rect x="250" y="80" width="5" height="50"/><rect x="310" y="105" width="5" height="25"/><rect x="370" y="105" width="5" height="25"/><rect x="420" y="55" width="5" height="75"/></g>
  <g font-size="8" fill="currentColor" font-family="monospace"><text x="60" y="148">10</text><text x="122" y="148">11</text><text x="182" y="148">11</text><text x="244" y="148">01</text><text x="304" y="148">00</text><text x="364" y="148">00</text><text x="414" y="148">10</text></g>
</svg>
<figcaption>PCM samples the analog wave at regular instants and snaps each sample to a quantization level encoded as a binary code — the digital form of the signal.</figcaption>
</figure>

## How it works

Three steps define PCM. **Sampling** measures the signal at a fixed rate, which by the
[Nyquist theorem](/reference/nyquist-theorem/) must exceed twice the highest frequency
present to avoid aliasing. **[Quantization](/reference/quantization/)** rounds each sample
to one of 2ᴺ levels set by the **bit depth** N, introducing a small, bounded quantization
error whose ratio to full scale gives roughly 6 dB of dynamic range per bit. **Encoding**
writes each level as an N-bit word. An [analog-to-digital converter](/reference/analog-to-digital-converter/)
performs the sample-and-quantize stages in hardware; a DAC reverses them on playback.

Because it stores exact numbers, PCM is lossless apart from the deliberate quantization
step, and it can be copied, filtered, and processed without further degradation — the
central advantage of digital over analog. The trade-offs are bit rate (sample rate ×
bits × channels) and the need for anti-alias and reconstruction filters. **Linear PCM**
uses uniform level spacing; **companded** variants such as telephony's G.711 μ-law and
A-law space levels logarithmically to give quiet sounds more resolution at a fixed bit
rate.

## Relevance to SDR

PCM is the native currency of software radio. The [ADC](/reference/analog-to-digital-converter/)
in an SDR front end delivers PCM samples — usually as [IQ](/reference/iq-data/) pairs,
which are simply two PCM streams for the in-phase and quadrature components — and every
downstream filter, mixer, and demodulator operates on those numbers. Recovered audio is
likewise PCM before it reaches the speaker or a codec. Standard sample formats (16-bit
integer, 8-bit unsigned from RTL-SDR dongles, 32-bit float) are all PCM encodings.

GopherTrunk consumes PCM/IQ sample streams from its SDR hardware and captures, runs its
DSP and demodulators on them, and outputs decoded PCM voice. So while PCM is not a
land-mobile *modulation* like the others in this family, it is the underlying digital
representation on which GopherTrunk's entire signal path — and all of SDR — is built.

## Sources

[^wiki]: [Pulse-code modulation](https://en.wikipedia.org/wiki/Pulse-code_modulation) — Wikipedia, for the sample/quantize/encode definition, bit depth and dynamic range, and companded μ-law/A-law variants.
