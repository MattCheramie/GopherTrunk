---
slug: ambe-plus-2-tones
title: AMBE+2 tones
entry_type: term
category: voice-coding
description: AMBE+2 tone frames are a special frame class, flagged by the b0 index, that carry single tones, DTMF pairs, or vendor-specific Knox and call-alert dual-tones instead of speech parameters.
keywords: AMBE+2 tones, tone frame, b0 0x7E 0x7F, DTMF ITU-T Q.23, Knox tones, call-alert dual-tone, tone index, vendor tones, single tone dual tone
aka: [AMBE+2 tone frame, "Knox tones", "call-alert tones"]
autolink: true
infobox:
  - { label: Flagged by, value: "b0 & 0x7E == 0x7E" }
  - { label: Single tones, value: "index 5–122" }
  - { label: DTMF pairs, value: "index 128–143 (Q.23)" }
  - { label: Knox / alert, value: "index 144–163 (undocumented)" }
see_also: [ambe-plus-2, ambe-plus-2-codebooks, dtmf, ambe-plus-2-fec, imbe-parameter-quantization]
cite_urls:
  - https://en.wikipedia.org/wiki/Dual-tone_multi-frequency_signaling
  - https://github.com/szechyjs/mbelib
---

**AMBE+2 tones** are a special class of [AMBE+2](/reference/ambe-plus-2/) frame that carry
signalling tones instead of speech. When the fundamental index `b_0` lands in its top range, the
frame is not a voice frame at all: the remaining bits index a single tone, a
[DTMF](/reference/dtmf/) pair, or a vendor-specific "Knox" or call-alert dual-tone, and the
decoder synthesises a sum of sinewaves rather than running the harmonic model.[^dtmf] Tone frames
let a digital-voice system carry in-band signalling — keypad digits, alert tones — over the same
2400 bps channel as speech.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 132" role="img" aria-label="A number line of AMBE+2 tone index values: indices five through one hundred twenty-two are single tones, one hundred twenty-eight through one hundred forty-three are DTMF dual-tone pairs, one hundred forty-four through one hundred sixty-three are vendor Knox and call-alert dual-tones, and all other values map to silence." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="448" y2="70" stroke="currentColor" stroke-width="1.1"/>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="40" y="56" width="150" height="28" fill="currentColor" fill-opacity="0.14"/>
    <rect x="250" y="56" width="70" height="28" fill="currentColor" fill-opacity="0.22"/>
    <rect x="320" y="56" width="88" height="28" fill="currentColor" fill-opacity="0.30"/>
  </g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="115" y="73">single tones</text>
    <text x="115" y="49">5–122</text>
    <text x="285" y="73">DTMF</text>
    <text x="285" y="49">128–143</text>
    <text x="364" y="73">Knox / alert</text>
    <text x="364" y="49">144–163</text>
  </g>
  <text x="231" y="108" text-anchor="middle" font-size="8" fill="currentColor">b0 &amp; 0x7E == 0x7E flags a tone frame; out-of-range index → silence</text>
</svg>
<figcaption>A tone frame's index selects a single tone (5–122), a DTMF pair (128–143), or a vendor Knox / call-alert dual-tone (144–163); any other value is treated as silence.</figcaption>
</figure>

## The tone-frame indicator

A frame is a tone frame when `b_0 & 0x7E == 0x7E` — that is, when `b_0` is 0x7E (126) or 0x7F
(127). In that case the ordinary voice-parameter layout is abandoned: `b_1` becomes an 8-bit tone
index and `b_2` an 8-bit amplitude, both drawn from a different bit-scatter pattern than a voice
frame uses. The top three bits of the tone index come from three small lookup tables
(`toneT5`/`toneT6`/`toneT7`) driven by three of the frame's bits; mbelib annotates these bits as
partly *verified* against captured frames and partly *derived* from the DTMF tone-index mapping.
The remaining index bits are read directly from the frame, and the amplitude index `b_2` sets how
loud the synthesised tone plays. Because tone frames recur every 20 ms for the duration of a tone,
the decoder carries the sinewave phase forward from one frame to the next so a held tone or DTMF
digit sounds continuous rather than clicking at each frame boundary.

## Index ranges

The tone index `b_1` partitions into well-defined ranges. GopherTrunk validates it and routes an
out-of-range index to silence rather than voicing noise:

| Index range | Meaning | Frequencies |
|-------------|---------|-------------|
| 5 – 122 | single tones | published |
| 128 – 143 | DTMF dual-tones | ITU-T Q.23 |
| 144 – 163 | Knox / call-alert dual-tones | vendor-specific, undocumented |
| all others | silence | — |

DTMF pairs decode against the standard [ITU-T Q.23](/reference/dtmf/) frequency assignments, so
they are fully determined by the index. The single-tone range is likewise published.

## Knox and call-alert tones

The 144–163 range is the awkward one. These "Knox" or call-alert dual-tones are vendor-specific:
Motorola TRBO, Hytera, and generic AMBE+2 implementations each pick slightly different
`(frequency_a, frequency_b)` pairs for the same indices, and the **public AMBE+2 specification does
not document the frequencies at all**. Without a reference table the decoder cannot know what to
synthesise, so GopherTrunk routes these indices through silence by default and exposes a runtime
override: `SetKnoxTone` (and a named-`KnoxPreset` bundle over it) lets an operator or an in-tree
test register the `(freqA, freqB)` pairs for a specific vendor, sourced from open receivers like
DSD-FME or a service manual and cited in a comment. Once registered, a Knox frame synthesises the
same summed-sinewave dual-tone the decoder produces for DTMF, with phase continuity across
consecutive tone frames.

## Relevance to SDR

Tone frames are woven through real AMBE+2 traffic — a DMR or P25 Phase 2 call may open with a
call-alert tone or carry DTMF overdial mid-conversation — so a decoder that treated every frame as
voice would render these as garbage. GopherTrunk's tone branch short-circuits the harmonic
synthesizer, validates the index, and either emits the correct dual-tone or falls back to a clean
silence path. The Knox override is the pragmatic answer to an under-specified corner of the
standard: rather than guess undocumented frequencies, it stays silent until an operator supplies a
citation-backed table, keeping the default decode honest.

## Sources

[^dtmf]: [Dual-tone multi-frequency signaling](https://en.wikipedia.org/wiki/Dual-tone_multi-frequency_signaling) — Wikipedia, on the DTMF frequency pairs (ITU-T Q.23) that AMBE+2 dual-tone frames reproduce.
