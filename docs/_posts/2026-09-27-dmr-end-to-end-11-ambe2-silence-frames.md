---
title: "DMR End to End, Part 11: AMBE+2 Voice & the Silence-Frame Bug"
description: "How a frame-by-frame diff against two mbelib lineages refuted the 'AMBE+2 2450 high-band deficit' theory, found the real DMR voice bug in the silence frames that reset the gain predictor 24 dB low at every utterance onset, and calibrated the unvoiced band and radio tilt against dsd-neo — measured, not tuned by ear."
category: deep-dives
keywords: ambe+2 3600x2450 decoder, dmr vocoder sounds bad, ambe silence frame b0 124, mbelib gamma predictor, unvoiced gain calibration, dsd-fme amb file, dsd-neo comparison, vocoder log spectral distance, gophertrunk dmr
tags: [dmr-end-to-end, dmr, ambe2, vocoder, calibration, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 11
---

*Part 11 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous calls, direct-mode handhelds, and decrypted
Enhanced Privacy voice.
[Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})
left a wideband tap producing clean superframes. This part follows them into
the vocoder — the AMBE+2 3600×2450 decoder in `internal/voice/ambe2` — and to
the "computer voice" report of
[#644](https://github.com/MattCheramie/GopherTrunk/issues/644). Two of
everything, once more: two mbelib lineages to diff against, two kinds of
frame the decoder must tell apart, and two calibrations that were never in
the vocoder core.*

> **TL;DR:** An earlier repo note blamed DMR's rough audio on a "high-band
> deficit in `unpackParams2450`". A frame-by-frame diff of the committed
> #644 clip (`testdata/dmr-voice.raw`, 378 frames) against szechyjs/mbelib
> 1.3.0 **and** mbelib-neo refuted it: Tl / Vl / L / w0 match to 1e-5 on all
> 201 voice frames. The defect was the other 177 — AMBE+2 **silence frames**
> (b0 124/125) that both references decode as an all-unvoiced fixed-model
> frame (w0 = 2π/32, L = 14, the frame's own ΔΓ / PRBA / HOC) while
> GopherTrunk emitted digital silence and **reset the gain predictor**, so
> every onset after a pause decoded at gamma = ΔΓ + 0 instead of
> ΔΓ + 0.5·γ_prev — frame 95: 4.37 vs 8.40, ≈ 24 dB low. Fixed to the
> reference behaviour, pinned by `params2450_silence_test.go`. Two
> calibrations followed, measured against dsd-neo: `ShapeUnvoicedSpectrumGain`
> with `DefaultUnvoicedGain` = 5.49 (unvoiced bands were 13–15 dB low) and a
> first-order `tilt_hz` = 450 high-pass. LSD to dsd-neo: DMR male
> 4.9 → 2.1 dB. Still open: AMBE 2450 male voiced harmonics −3..−6 dB.

**Key takeaways**

- **A refuted note is worth recording.** "The deficit is in the 2450 tables"
  was plausible from whole-file band fractions and wrong; a per-frame diff
  against an independent decoder settled it.
- **Silence frames are not silence.** b0 124/125 carries a gain delta and an
  envelope; the references synthesise it and carry the predictor through.
  Zeroing `prevGamma` instead made every onset 24 dB quiet.
- **Calibrate the band, not the ear.** Per-harmonic DFT, band fractions and
  log-spectral distance against dsd-neo are the metrics; the shipped WAVs hid
  all of it under enhance, normalize and warm.
- **Compare voice frames only.** The references disagree on the silence model
  itself (L = 14 vs 15), so whole-file spectra depend on which you pick.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| 2450 unpack | b0..b8 scattered layout → dmr* tables → PRBA/HOC inverse DCTs → Tl | `internal/voice/ambe2/params2450.go` (`unpackParams2450`), `tables2450.go` |
| Silence frame | b0 124/125 ⇒ w0 = 2π/32, L = 14, all-unvoiced, own ΔΓ / Tl; `SilenceFrame` flag | `params2450.go`, `params.go` (`Params.SilenceFrame`) |
| Gain predictor | gamma = ΔΓ + 0.5·prevGamma, carried through pauses | `decoder.go` (`prevGamma`, `foldGammaIntoTl`) |
| Reference pins | literal frames vs mbelib `cur_mp` values | `params2450_silence_test.go` (`TestDecodeDMRSampleGammaTracksReference`) |
| Unvoiced band level | per-band power = gain·Ml²/2; 5.49 = mbelib's 3-cosine level | `internal/voice/mbe/synth_unvoiced.go` (`ShapeUnvoicedSpectrumGain`, `DefaultUnvoicedGain`) |
| Radio tilt | first-order HPF, bilinear pre-warped, default 450 Hz | `mbe/enhancer.go` (`radioTilt`, `EnhancerConfig.TiltHz`) |
| Operator knobs | `recordings.unvoiced_gain`, `recordings.enhance.tilt_hz`, `recordings.mbe_files` | `config.example.yaml`, `docs/vocoders.md` |

## In this post

- **From superframe to 49 bits** — what reaches the vocoder, and which frames are not voice.
- **The note that was refuted** — the per-frame diff against two mbelib lineages.
- **Silence frames are not silence** — the predictor reset and the 24 dB onset.
- **Calibrating the unvoiced band and the tilt** — dsd-neo as the measured reference.
- **Checking your own frames** — `.amb` sidecars, DSD-FME's 8 kHz header, `decode`.

## From superframe to 49 bits

[Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})
sliced the voice superframe; each burst carries three 72-bit AMBE+2 frames,
and the Golay(23,12) + descramble stage in `internal/radio/dmr/voice`
reduces each to 49 bits
([Voice Coding Part 8]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }})
covers that boundary). The DMR variant is `ambe2-dmr`: the 3600×2450 rate
whose bit positions and codebooks (generated from mbelib's
`ambe3600x2450_const.h`) differ from the 3600×2400 default of
[Voice Coding Part 7]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }}).
Everything after `unpackParams2450` — the gamma fold, `mbe.PredictLog2Ml`,
§6.2 enhancement, synthesis — is shared with the base decoder and IMBE.

In the 2450 frame, **b0 is not always a pitch**: 0..119 index `dmrW0table`,
120..123 mark an erasure, 126..127 a tone frame, and 124..125 an AMBE+2
*silence frame* — the subject of this part.

## The note that was refuted

The #644 report was that DMR voice "sounds awful — unnatural, computer
voice", and an early measurement seemed to pin it. Against the operator's
DSD-FME decode of the same `.amb` frames, GopherTrunk's `ambe2-dmr` output
had 4–10× less energy above 1 kHz. The base `ambe2` decoder shares the
synthesis and gave mbelib-like highs, so the note concluded the deficit was
"purely in `unpackParams2450`" and would need a frame-by-frame reference
before anyone touched the tables.

That diff was done, and it **refuted the note**. szechyjs/mbelib 1.3.0 and
arancormonk/mbelib-neo were built from source and made to dump `cur_mp` per
frame over the committed 378-frame clip; on every one of the **201 voice
frames** the 2450 unpack's Tl, Vl, L and w0 match mbelib to 1e-5 and the
amplitude prediction within ~1 dB per band.
`TestUnpack2450TlMatchesMbelibFrame0` pins a literal on-air frame
(`b0b9243e0bc180`, L = 37) against `mbe_decodeAmbe2450Parms`'s Tl — the
independent-reference vector the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
demands, since a round-trip could never catch table drift. Do not chase the
2450 tables. On voice frames only, GopherTrunk is *not* high-band deficient
against mbelib-neo — the whole-file fractions were confounded by the other
177 frames.

## Silence frames are not silence

Those 177 frames are AMBE+2 silence frames — 47 % of the clip, in runs of 44
and 63 (≈ 1.3 s) between utterances. Both references decode them as an
all-unvoiced frame with a fixed model — `w0 = 2π/32`, `L = 14` — carrying the
frame's **own** gain delta and PRBA/HOC envelope, then synthesise them like
any other: the background plays at its level and the cross-frame state
carries through. GopherTrunk short-circuited them to digital silence *and
reset the predictor*:

```go
// internal/voice/ambe2/params2450.go (shape)
silenceFrame := b0 == 124 || b0 == 125
if b0 >= 120 && !silenceFrame { /* erasure / tone → Silent */ }
if silenceFrame {
    f0, L = 1.0/32, 14 // mbelib fixed model: w0 = 2π/32, L = 14, all unvoiced
} else {
    f0, L = dmrW0table[b0], int(dmrLtable[b0])
}
// …b2 → p.DeltaGamma; PRBA/HOC → Tl, exactly as for voice
```

The gain is the cross-frame quantity AMBE+2 has and IMBE lacks:
`gamma = p.DeltaGamma + 0.5*d.prevGamma`. Zeroing `prevGamma` on every
silence frame meant the first voice frame after every pause decoded with
`gamma = ΔΓ + 0` — frame 95 at 4.37 against the references' 8.40, frame 308
at 5.78 against 10.03: **≈ 24 dB low**, on every onset, with the pauses
hard-gated between. That is the "computer voice", and none of it was in the
tables.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A frame timeline through a DMR utterance, a run of AMBE+2 silence frames, and the next utterance onset. The top row shows the gain predictor gamma in the reference decoders continuing through the silence frames, each one contributing its own gain delta plus half the previous gamma, so the onset frame lands at 8.40. The bottom row shows the old GopherTrunk decoder emitting digital silence for the same frames and resetting the predictor to zero, so the onset frame lands at 4.37, about 24 decibels low.">
  <text x="340" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">gamma = ΔΓ + 0.5·γ_prev across a pause (frames 75…95 of the #644 clip)</text>
  <text x="120" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="8">voice</text>
  <text x="340" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="8">silence frames · b0 = 124 / 125 · 20-frame run</text>
  <text x="580" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="8">onset (frame 95)</text>
  <rect x="40" y="44" width="160" height="14" fill="none" stroke="currentColor"/>
  <rect x="210" y="44" width="260" height="14" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <rect x="480" y="44" width="160" height="14" fill="none" stroke="currentColor"/>
  <text x="26" y="96" text-anchor="end" fill="currentColor" font-size="8">refs</text>
  <path d="M40 92 L200 88 L230 94 L270 100 L310 104 L350 106 L390 104 L430 100 L470 96 L500 82 L560 78 L640 80" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="340" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">w0 = 2π/32, L = 14, all unvoiced — own ΔΓ, own envelope, synthesised</text>
  <text x="600" y="72" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">γ = 8.40</text>
  <text x="26" y="176" text-anchor="end" fill="var(--accent)" font-size="8">old GT</text>
  <path d="M40 172 L200 168" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="210" y1="200" x2="470" y2="200" stroke="var(--accent)" stroke-width="1.5" stroke-dasharray="2 3"/>
  <text x="340" y="192" text-anchor="middle" fill="var(--accent)" font-size="8">digital silence · prevGamma = 0</text>
  <path d="M480 200 L500 164 L560 160 L640 162" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="600" y="150" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">γ = 4.37 (ΔΓ + 0)</text>
  <line x1="655" y1="80" x2="655" y2="162" stroke="var(--fg-muted)"/>
  <text x="660" y="124" fill="var(--fg-muted)" font-size="8">≈ 24 dB</text>
  <text x="340" y="238" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every onset after every pause — 47 % of this clip is silence frames</text>
</svg>
<figcaption>The references carry the predictor through a silence run; the old decoder zeroed it, so each utterance began four log2 units below the reference and rose from there.</figcaption>
</figure>

The fix is the reference behaviour: `Params.SilenceFrame` marks the frame,
the unpack builds the fixed model with the frame's own `DeltaGamma` and Tl,
and the decoder synthesises it through the normal path, predictor intact.
`TestUnpack2450SilenceFrameIsUnvoicedNoiseFrame` pins the literal
frame 19 (`f84902a06c3d80`) and fails against the old `Params{Silent: true}`;
`TestDecodeDMRSampleGammaTracksReference` checks `prevGamma` at nine named
frames against mbelib's `cur_mp->gamma` to 2e-3, and mbelib-neo agrees.

One thing the diff did **not** settle: the references disagree on the silence
model itself — mbelib-neo/JMBE uses L = 15 and a different w0; mbelib and
DSD-FME's lwvmobile fork use 2π/32 and L = 14 — and GopherTrunk follows the
DSD-FME lineage the reporter compares against. Whole-file band fractions
depend on that, so spectral comparisons must be made **on voice frames
only**.

## Calibrating the unvoiced band and the tilt

With the voice frames bit-identical, the remaining "sounds awful" was
measured, not debugged: a per-harmonic DFT of GopherTrunk's and dsd-neo's
decodes of the same 10 Sep `.imb`/`.amb` pairs. f₀ was identical and voiced
IMBE harmonics matched within ±1 dB above 1.5 kHz. Two things did not, and
neither was in the vocoder core.

**Unvoiced bands were 6 dB (IMBE) to 13–15 dB (AMBE+2 2450) low.** mbelib
synthesises an unvoiced harmonic as three random-phase cosines ≈ 1.353·Ml
each — about 2.745·Ml² per band, 5.49× the Ml²/2 a voiced harmonic carries.
GopherTrunk's §6.4 FFT-noise path scaled each bin by Ml. The fix normalises
per band:

```go
// internal/voice/mbe/synth_unvoiced.go (shape)
const LegacyUnvoicedGain  = -1   // old per-bin Ml scaling, kept for goldens
const DefaultUnvoicedGain = 5.49 // mbelib's 3-cosine level ≈ 2.745·Ml²
// gain ≥ 0: each unvoiced band's expected power is gain·Ml²/2, whatever
// its FFT-bin span; gain < 0 reproduces legacy byte-for-byte.
func ShapeUnvoicedSpectrumGain(spec []complex128, p Params, M *[57]float64, gain float64)
```

`recordings.unvoiced_gain` exposes it: `0` ⇒ the 5.49 default, `1` = the
equal-power spec reading, `< 0` = legacy. The raw decoders start legacy so
their goldens hold; the recorder and `gophertrunk decode` set the default,
and `decode -legacy-synthesis` opts out for A/B.

**Below ~1.5 kHz GopherTrunk carried a smooth low-frequency excess** — +4 dB
at 500 Hz rising to +16 at 125 — the shape of a first-order high-pass, which
is what dsd-neo runs by default (`use_hpf_d=1`) and what a handset does.
`radioTilt` is that filter, bilinear-transformed with a pre-warped corner so
the −3 dB point lands on fc at 8 kHz (the naive RC recurrence is ~1.2 dB
off). Fitting voiced harmonics alone gives ~750 Hz; with the unvoiced level
also calibrated, the corner minimising log-spectral distance to dsd-neo
across all four pairs is ~450 Hz, so `tilt_hz: 450` is the default.

The verdict is the number, not the ear: LSD to dsd-neo over 300–3400 Hz,
P25 male 8.3 → 1.5 dB, DMR male 4.9 → 2.1 dB. What remains **open** is real:
AMBE 2450 *male voiced* harmonics sit −3..−6 dB below dsd-neo above 1 kHz
(IMBE doesn't show it; the DMR female goes the other way), and that one is in
the 2450 amplitude path — it needs the mbelib-neo per-frame diff. The
operator's `warm_dmr_audio: true` plus `enhance` LPF only made it *worse* —
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
has the PCM chain these knobs sit in.

## Checking your own frames

The method that settled this is available to any operator, and
[Voice Coding Part 12]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }})
built the harness around it. `recordings.mbe_files: true` writes DSD-FME's
cookie-headed container next to each recording (`.amb` for DMR, `.imb` for
P25 Phase 1), and DSD-FME decodes it directly. One quirk: DSD-FME's `-w`
writer stamps an **8 kHz header on 12 kHz synthesis**, so the file plays
~1.5× slow. dsd-neo writes a correct 8 kHz WAV and is the reference the
calibration was measured against:

```sh
for f in *.amb; do ./dsd-neo -o null -w "${f%.amb}_dsd.wav" -fs -r "$f"; done
gophertrunk decode -in call.raw -out call_gt.wav -vocoder ambe2-dmr   # recorder defaults
gophertrunk decode -in call.raw -out call_raw.wav -vocoder ambe2-dmr -legacy-synthesis
```

Then compare with band fractions, centroid or LSD — on voice frames — never
by ear alone. The "gt-shipped" WAVs that started the report had `enhance`,
loudness normalize and `warm_dmr_audio` stacked on top and hid every effect
above.

### How the diff shaped the Go code

- **Silence is a flag, not a short-circuit.** `Params.SilenceFrame` rides
  the normal voice path; only erasure and tone frames set `Silent`.
- **Reference literals over round-trips.** `params2450_silence_test.go`
  carries hex frames and `cur_mp` values from two independent decoders — the
  only shape that catches table or predictor drift.
- **Calibration lives in the caller.** `LegacyUnvoicedGain` keeps the raw
  decoders' goldens byte-identical; the recorder and `decode` apply
  `DefaultUnvoicedGain` and the tilt per call.
- **Every knob landed in `config.example.yaml`** — `unvoiced_gain` and
  `enhance.tilt_hz` with their measured defaults.

## Where this goes next

Clear voice now decodes as the references do. The next part is voice that
was never meant to be clear:
[Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})
follows DMR Enhanced Privacy from the PI header through RC4 keyed with
key‖MI, the 256 dropped keystream bytes, and the embedded IV that names the
*next* superframe — and why a Golay histogram was the first instrument, not
a cipher theory.

## FAQ

**Why did GopherTrunk's DMR audio sound like a "computer voice"?**
Not the vocoder tables: on the #644 clip every voice frame's parameters
match mbelib to 1e-5. The cause was AMBE+2 silence frames (b0 124/125),
which the decoder rendered as digital silence while resetting the gain
predictor, so each utterance after a pause started ≈ 24 dB low. They now
decode as the references do.

**What is an AMBE+2 silence frame?**
A 3600×2450 frame whose b0 is 124 or 125. mbelib and mbelib-neo both decode
it as an all-unvoiced frame with a fixed model (w0 = 2π/32, L = 14 in the
DSD-FME lineage) and the frame's own ΔΓ / PRBA / HOC bits, then synthesise it
— the background the radio encoded plays at its transmitted level.

**Is the "2450 high-band deficit" real?**
No. That earlier note was refuted by a frame-by-frame diff against
szechyjs/mbelib 1.3.0 and mbelib-neo: `unpackParams2450` is bit-identical on
every voice frame; the whole-file fractions that suggested it were confounded
by 47 % silence frames. What remains open is a −3..−6 dB deficit in male
voiced harmonics above 1 kHz on AMBE 2450 specifically.

**What does `recordings.unvoiced_gain` do?**
It sets the unvoiced band power relative to a voiced harmonic of the same
amplitude. `0` selects the default 5.49 — mbelib's three-cosine level, 6–15 dB
above GopherTrunk's old scaling; `1` is the spec's equal-power reading; a
negative value restores legacy. The recorder and `decode` apply it per call.

**How do I compare GopherTrunk's DMR decode against DSD-FME?**
Set `recordings.mbe_files: true` for a `.amb` sidecar and decode it with
DSD-FME or dsd-neo. DSD-FME's `-w` stamps an 8 kHz header on 12 kHz audio, so
relabel or resample first. Decode the `.raw` with `gophertrunk decode
-vocoder ambe2-dmr` and compare band fractions or LSD on voice frames.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Wideband DMR — Bin Edges & the Deaf Heal]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})
· Next →
[Part 12: Enhanced Privacy — RC4 & the IV That Names the Next Superframe]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }})
