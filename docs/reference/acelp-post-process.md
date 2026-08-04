---
slug: acelp-post-process
title: ACELP Post_Process
entry_type: term
category: voice-coding
description: "ACELP Post_Process is the ETSI EN 300 395-2 output stage that saturating-doubles every TETRA decoder PCM sample; without it GopherTrunk's audio sits 6 dB below the reference level."
keywords: ACELP Post_Process, output scaling, saturating multiply, 6 dB level, TETRA vocoder, PCM output, fixed-point saturation, ETSI EN 300 395-2, reference level
aka: [Post_Process, "output scaling stage", "x2 post-processing"]
autolink: true
infobox:
  - { label: Role, value: Output level scaling }
  - { label: Operation, value: Saturating ×2 per PCM sample }
  - { label: If omitted, value: Audio 6 dB below reference }
  - { label: Spec, value: ETSI EN 300 395-2 }
see_also: [acelp, etsi-g191-basic-operators, fixed-point-vs-floating-point, tetra-tchs-speech-coding, quantization, pulse-code-modulation, automatic-gain-control]
cite_urls:
  - https://en.wikipedia.org/wiki/Saturation_arithmetic
  - https://en.wikipedia.org/wiki/Pulse-code_modulation
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
---

**ACELP Post_Process** is the final output stage of the
[TETRA](/reference/tetra/) [ACELP](/reference/acelp/) decoder specified by ETSI EN 300 395-2: a
**saturating multiply-by-two** applied to every decoded [PCM](/reference/pulse-code-modulation/)
sample. It carries no spectral or excitation information — it exists solely to put the decoder's
output at the reference level. GopherTrunk applies it in the vocoder wrapper's `Decode` method,
and without it the rendered audio sits a full **6 dB below** the ETSI reference decoder.[^sat]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="Raw decoder samples pass through a saturating times-two stage; a sample already near full scale is clamped to the maximum rather than wrapping around, while a mid-scale sample is doubled cleanly, lifting the overall level by six decibels." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.4"><line x1="30" y1="70" x2="440" y2="70"/></g>
  <g stroke="currentColor" stroke-width="1.3" fill="none"><line x1="70" y1="70" x2="70" y2="52"/><line x1="120" y1="70" x2="120" y2="38"/><line x1="170" y1="70" x2="170" y2="60"/></g>
  <g stroke="currentColor" stroke-width="1.3" fill="none"><line x1="300" y1="70" x2="300" y2="34"/><line x1="350" y1="70" x2="350" y2="20"/><line x1="400" y1="70" x2="400" y2="50"/></g>
  <line x1="290" y1="20" x2="410" y2="20" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.7"/>
  <text x="350" y="16" text-anchor="middle" font-size="7" fill="currentColor">+full scale (clamp)</text>
  <text x="120" y="90" text-anchor="middle" font-size="8" fill="currentColor">raw decoder</text>
  <text x="350" y="90" text-anchor="middle" font-size="8" fill="currentColor">after saturating ×2</text>
  <path d="M200 55 L260 55" stroke="currentColor" stroke-width="1.2" fill="none" marker-end="url(#ppar)"/>
  <defs><marker id="ppar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="230" y="50" text-anchor="middle" font-size="7.5" fill="currentColor">×2</text>
</svg>
<figcaption>Post_Process doubles every output sample with saturation: mid-scale samples gain 6 dB, while any sample that would overshoot is clamped to full scale rather than wrapping.</figcaption>
</figure>

## What the stage does

The reference codec's internal synthesis produces PCM at half the intended playback level, so the
last thing `Decod_Tetra` does is double it. GopherTrunk's `vocoder.Decode` mirrors this exactly:
after the raw synthesis returns 240 int16 samples (30 ms at 8 kHz), it runs each sample through
`addOp(sfin, sfin)` — adding the sample to itself, which is a multiply-by-two using the
[G.191 saturating add](/reference/etsi-g191-basic-operators/). The result is one call's worth of
PCM at the reference level, ready for the downstream DC block,
[AGC](/reference/automatic-gain-control/), and playback.

Six decibels is a factor of two in amplitude, which is exactly what a ×2 buys — so an
implementation that renders otherwise-correct TETRA audio but sounds conspicuously quiet has
almost always dropped this stage. It is a small, easy-to-miss line that is nonetheless part of the
bit-exact contract with the reference decoder.

## Why saturation, not a plain multiply

Doubling a value that is already near the top of the int16 range would overflow. In ordinary
two's-complement arithmetic that overflow *wraps*: a large positive sample flips to a large
negative one, producing a loud click on every loud sample — the worst possible artifact on exactly
the samples a listener notices most. **Saturation arithmetic** avoids this by clamping instead of
wrapping: any product that exceeds +32767 pins to +32767, and any below −32768 pins to −32768.
The clamp introduces a little harmonic distortion on peaks, but that is inaudible next to the
full-scale sign inversion wrapping would cause. This is the same
[fixed-point](/reference/fixed-point-vs-floating-point/) discipline the entire ACELP decoder is
built on — every intermediate operation saturates rather than wraps, so the whole signal path
stays bounded and matches the reference bit for bit.

Using `addOp` rather than a bare `int16(sfin * 2)` is deliberate: `addOp` is the reference's
`add` operator, which sets the saturation clamp and the global overflow flag exactly as the C
codec does. Substituting a native multiply would silently reintroduce the wrap-around the standard
was written to prevent.

## Where it sits in the chain

Post_Process is the boundary between the ACELP codec proper and GopherTrunk's audio output path.
Upstream, the [TCH/S speech-frame decode](/reference/tetra-tchs-speech-coding/) and the ACELP
synthesis produce the raw PCM; Post_Process lifts it to reference level; downstream, generic audio
conditioning takes over. Placing the ×2 at the codec's own output — rather than folding it into a
later gain stage — keeps GopherTrunk's decode faithful to the reference at the exact point the
standard specifies, so the codec's output can be compared sample-for-sample against ETSI's before
any GopherTrunk-specific processing muddies the comparison.

## Relevance to SDR

For a scanner the practical payoff is consistent, correct loudness: TETRA calls decode at the same
level a reference-conformant radio would produce, so they sit sensibly alongside other protocols
in a multi-system scanner without a per-protocol volume fudge. Because the stage is a fixed part of
the reference algorithm, GopherTrunk verifies it as part of the ACELP decoder's end-to-end
conformance rather than tuning it by ear.

## Sources

[^sat]: [Saturation arithmetic](https://en.wikipedia.org/wiki/Saturation_arithmetic) — Wikipedia, on clamping versus wrap-around overflow in fixed-point audio arithmetic.
[^pcm]: [Pulse-code modulation](https://en.wikipedia.org/wiki/Pulse-code_modulation) — Wikipedia, on the int16 PCM representation the stage scales.
