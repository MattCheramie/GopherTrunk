---
title: "Voice Coding, Part 8: AMBE+2 FEC & the Knox Path"
description: Where AMBE+2 error correction actually lives, how GopherTrunk conceals bad frames with replay and adaptive smoothing, and what the vendor-specific knox tone path really is — and isn't.
category: deep-dives
keywords: ambe+2 fec, golay 23 12, dmr ambe fec, bad frame concealment, adaptive smoothing vocoder, knox tone, ambe dtmf, call alert tone, gophertrunk voice coding
tags: [ambe2, fec, error-concealment, dmr, tones, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 8
---

*Part 8 of **Voice Coding**. Part 7 unpacked AMBE+2's 49 information bits into
MBE parameters. This post is about what happens when those bits arrive
damaged, and about the one corner of the AMBE+2 tone map the public spec
leaves blank — the "knox" range — where GopherTrunk ships a mechanism but
deliberately no frequencies.*

> **TL;DR:** The forward-error-correction that protects AMBE+2 on the air —
> Golay(23,12) over DMR's 72-bit frame, trellis + Reed-Solomon on P25 Phase 2 —
> lives in the **protocol decoders**, not in `internal/voice/ambe2`. The
> vocoder receives 49 post-FEC bits. What it does with *residual* damage is
> **concealment**: replay the last good frame with progressive attenuation,
> and let the FEC's corrected-bit count drive adaptive spectral smoothing. The
> **knox** path is a runtime extension point for vendor-specific dual-tone
> frequencies the public AMBE+2 spec doesn't document — present as an API,
> absent as data.

**Key takeaways**

- **Error correction is upstream.** DMR's `ambefec.go` deinterleaves 72 on-air
  bits into C0–C3, runs Golay(23,12), descrambles C1, and emits the 49-bit
  payload. The vocoder never sees the parity bits.
- **The vocoder does concealment, not correction.** Two mechanisms: bad-frame
  *replay* (`MaxBadFrames = 6`, `BadFrameAttenuation = 0.7`) and *adaptive
  smoothing* driven by `SetFrameErrors`.
- Bad-frame replay is **largely defensive** — every valid voice `b0` resolves
  to a legal `L`, so `UnpackParams` only errors on a wrong-length input. The
  structure is retained for parity with IMBE and as a landing spot for future
  FEC signalling.
- **Knox tones are honest about their limits.** `b1 ∈ [144, 163]` is
  vendor-specific; GopherTrunk routes it to silence unless an operator
  registers `(freqA, freqB)` via `SetKnoxTone` / `RegisterPreset`.

## Cheat sheet

| Concept | Where it lives | Role |
|---|---|---|
| On-air FEC | `internal/radio/dmr/voice/ambefec.go` (P25 P2: `phase2`) | Golay(23,12) + descramble → 49-bit payload |
| `SetFrameErrors(n)` | `decoder.go` | feed the FEC corrected-bit count to smoothing |
| `MaxBadFrames = 6` | `mbe/agc.go` | consecutive replays before muting |
| `BadFrameAttenuation = 0.7` | `mbe/agc.go` | per-replay amplitude multiplier |
| `SetKnoxTone(b1, a, b)` | `knox.go` | register a vendor dual-tone pair |
| `RegisterPreset(p)` | `knox.go` | apply a named bundle of knox pairs |

## In this post

- **Where the FEC actually is** — the protocol/vocoder boundary.
- **Concealment #1: bad-frame replay** — and why it's mostly defensive.
- **Concealment #2: adaptive smoothing** — the corrected-bit count at work.
- **The knox path** — a mechanism without a spec, done honestly.

## Where the FEC actually is

"AMBE+2 FEC" is an ambiguous phrase, so let's pin it down. An AMBE+2 voice
frame on a DMR carrier is **72 bits**, of which only **49** are the vocoder
payload. The other 23 are error-control coding — a Golay(23,12) code over the
perceptually-critical bits, plus a C0-seeded descramble. That wrapping is
undone in the DMR radio layer, not the voice layer:

```go
// internal/radio/dmr/voice/ambefec.go (shape)
// Each 72-bit on-air AMBE+2 frame wraps 49 bits of vocoder payload in FEC.
// Deinterleave into C0..C3, run Golay(23,12) over C0 and C1, descramble C1
// with the C0-seeded PRNG, assemble the 49-bit ambe_d payload.
const (
    ambeOnAirBits = 72
    ambeInfoBits  = 49
)
```

P25 Phase 2 does the equivalent with its own machinery (trellis decode,
Reed-Solomon, deinterleave) in the `phase2` package. By the time bits reach
`internal/voice/ambe2`, the heavy lifting is done — the decoder is handed the
same 49 clean-ish information bits regardless of which protocol carried them.
That is exactly the separation that let one vocoder serve three protocols in
[Part 7]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="On-air 72-bit frames pass through the protocol decoder's Golay and descramble stages to yield 49-bit payloads, which the vocoder decodes while using the corrected-bit count for concealment">
  <rect x="6" y="56" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="66" y="76" text-anchor="middle" fill="currentColor" font-size="12">72 on-air bits</text>
  <text x="66" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="10">DMR AMBE frame</text>
  <line x1="126" y1="79" x2="170" y2="79" stroke="currentColor"/>
  <polygon points="170,75 180,79 170,83" fill="currentColor"/>
  <rect x="180" y="44" width="168" height="70" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="264" y="66" text-anchor="middle" fill="var(--accent)" font-size="12">protocol decoder</text>
  <text x="264" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Golay(23,12) · descramble</text>
  <text x="264" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">emits corrected-bit count</text>
  <line x1="348" y1="79" x2="392" y2="79" stroke="currentColor"/>
  <polygon points="392,75 402,79 392,83" fill="currentColor"/>
  <rect x="402" y="44" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="462" y="64" text-anchor="middle" fill="currentColor" font-size="12">49-bit payload</text>
  <text x="462" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ambe_d</text>
  <line x1="522" y1="67" x2="566" y2="67" stroke="currentColor"/>
  <polygon points="566,63 576,67 566,71" fill="currentColor"/>
  <rect x="576" y="44" width="98" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="625" y="64" text-anchor="middle" fill="currentColor" font-size="12">ambe2</text>
  <text x="625" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="10">decode</text>
  <line x1="348" y1="104" x2="576" y2="104" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <polygon points="572,100 582,104 572,108" fill="var(--accent)"/>
  <text x="470" y="120" text-anchor="middle" fill="var(--accent)" font-size="10">SetFrameErrors(correctedBits) → adaptive smoothing</text>
  <text x="340" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">correction is upstream; the vocoder gets clean bits plus a quality signal</text>
</svg>
<figcaption>The parity bits never reach the vocoder. What crosses the boundary is the payload and one number — how many bits the FEC had to fix — which the vocoder uses to conceal what the FEC couldn't.</figcaption>
</figure>

## Concealment #1: bad-frame replay

If a frame *does* fail to unpack, the decoder doesn't emit a click of silence.
It replays the last good frame's parameters with a per-frame amplitude taper,
so a brief glitch fades rather than punches:

```go
// internal/voice/ambe2/decoder.go (shape)
case err != nil && d.lastGoodParams.L > 0 && d.badFrameCount < mbe.MaxBadFrames:
    d.badFrameCount++
    atten := math.Pow(mbe.BadFrameAttenuation, float64(d.badFrameCount))
    repeatedM := d.lastGoodM
    for l := 1; l <= d.lastGoodParams.L; l++ {
        repeatedM[l] *= atten
    }
    d.synthFrame(d.lastGoodParams, &d.lastGoodLog2M, &repeatedM, pcm)
    d.applyOutput(pcm, out, true) // freeze AGC so the taper is audible
```

`BadFrameAttenuation = 0.7` means one bad frame plays at 70% of the previous
good frame; six in a row taper to `0.7⁶ ≈ 0.12`. After `MaxBadFrames = 6`
consecutive replays (~120 ms), the cache clears and the decoder emits silence —
long enough to hide a real FEC slip, short enough that an extended dropout
fades naturally instead of looping the same envelope. The AGC is *frozen*
(`freezeEnvelope = true`) during a replay so the deliberate attenuation isn't
immediately clawed back by gain — the listener actually hears the signal
degrade.

Here's the honest caveat, straight from the decoder's own doc comment: this
path is **largely defensive**. `AmbePlusLtable` guarantees every voice `b0`
resolves to a valid `L ∈ [9, 56]`, so `UnpackParams` only returns an error on a
wrong-length input — which `Decode` catches even earlier. The replay machinery
is retained to mirror the IMBE decoder's shape and to give a future
protocol-layer FEC signal (a "this frame is erased" flag) a place to land.

## Concealment #2: adaptive smoothing

The more active mechanism is the one wired to the FEC's own telemetry. When the
protocol layer FEC-decodes a frame, it knows how many bits it had to correct —
a direct proxy for channel quality. It hands that count to the vocoder *before*
the matching `Decode`:

```go
// internal/voice/ambe2/decoder.go (shape)
func (d *Decoder) SetFrameErrors(correctedBits int) { d.frameErrs = correctedBits }

func (d *Decoder) Decode(frame []byte) ([]int16, error) {
    correctedBits := d.frameErrs
    d.frameErrs = 0
    muteByER := d.smoother.UpdateErrorRate(correctedBits)
    // ... unpack + synthesize ...
    d.smoother.Smooth(&folded, &M, correctedBits) // cap spikes, reclaim voiced
    if muteByER {
        for l := 1; l <= folded.L; l++ { M[l] = 0 } // silence a hopeless frame
    }
}
```

On a clean channel (`correctedBits == 0`) the smoother is inert — the faithful
path is untouched. On a degrading channel it caps error-induced amplitude
spikes and reclaims obviously-voiced harmonics that a bit-flip flipped to
unvoiced, taming the "warble" you'd otherwise hear. And when the error rate
crosses a mute threshold, `muteByER` zeroes the amplitudes, which the voiced
generator's amplitude tilt fades out cleanly. This is the `voice.ErrorAware`
interface in action; the recorder only calls `SetFrameErrors` when the upstream
chain supplies a count (see
[Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})).

## The knox path

AMBE+2 tone frames (`b0 ∈ {0x7E, 0x7F}`) carry a `b1` index that names the
tone. Most of the map is well-defined and shared across every open decoder:
`b1 ∈ [5, 122]` is a single tone at `b1·31.25 Hz`; `b1 ∈ [128, 143]` is a DTMF
key from the ITU-T Q.23 4×4 matrix (697/770/852/941 Hz rows × 1209/1336/1477/1633
Hz columns). Those GopherTrunk synthesizes directly. Then there's the gap:
`b1 ∈ [144, 163]` — the **knox** / call-alert range — whose frequencies are
vendor-specific (Motorola Trbo, Hytera, and generic implementations differ) and
which the public AMBE+2 spec simply does not document.

<figure class="lab-figure">
<svg viewBox="0 0 660 178" width="660" height="178" role="img" aria-label="A decision map for AMBE+2 tone-frame b1 indices routing single tones, DTMF, registered knox tones, and unregistered knox tones to their synthesis or silence outcomes">
  <rect x="14" y="20" width="150" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="35" text-anchor="middle" fill="currentColor" font-size="11">b1 ∈ [5, 122]</text>
  <text x="89" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="9">single tone</text>
  <rect x="14" y="62" width="150" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="77" text-anchor="middle" fill="currentColor" font-size="11">b1 ∈ [128, 143]</text>
  <text x="89" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DTMF (Q.23)</text>
  <rect x="14" y="104" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="89" y="122" text-anchor="middle" fill="var(--accent)" font-size="11">b1 ∈ [144, 163]</text>
  <text x="89" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">knox / call-alert</text>
  <text x="89" y="149" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(vendor-specific)</text>
  <line x1="164" y1="37" x2="360" y2="60" stroke="var(--fg-muted)"/>
  <line x1="164" y1="79" x2="360" y2="66" stroke="var(--fg-muted)"/>
  <polygon points="357,55 367,60 356,64" fill="var(--fg-muted)"/>
  <rect x="368" y="46" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="443" y="67" text-anchor="middle" fill="currentColor" font-size="11">synthesize tone</text>
  <line x1="164" y1="128" x2="360" y2="120" stroke="var(--accent)"/>
  <polygon points="357,115 367,120 357,124" fill="var(--accent)"/>
  <rect x="368" y="104" width="150" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="443" y="119" text-anchor="middle" fill="var(--accent)" font-size="10">KnoxTone(b1) set?</text>
  <text x="443" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SetKnoxTone / preset</text>
  <line x1="518" y1="115" x2="600" y2="70" stroke="currentColor"/>
  <polygon points="598,71 606,64 602,74" fill="currentColor"/>
  <text x="590" y="58" text-anchor="middle" fill="currentColor" font-size="10">yes → dual-tone</text>
  <line x1="518" y1="128" x2="600" y2="150" stroke="var(--fg-muted)"/>
  <polygon points="598,145 606,152 596,154" fill="var(--fg-muted)"/>
  <text x="592" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no → silence</text>
</svg>
<figcaption>Single tones and DTMF are fully specified and always synthesized. Knox indices synthesize a dual-tone only when an operator has registered the vendor frequencies; otherwise they route to silence.</figcaption>
</figure>

GopherTrunk's answer is a **runtime extension layer**: the mechanism ships, the
numbers don't.

```go
// internal/voice/ambe2/knox.go (shape)
const (KnoxIndexLow = 144; KnoxIndexHigh = 163)

// Register a vendor-specific dual-tone pair; (0,0) clears it back to silence.
func SetKnoxTone(b1 int, freqA, freqB float64) error
func KnoxTone(b1 int) (float64, float64, bool) // false if unset → silence

// A named bundle of pairs, applied via SetKnoxTone, for operators with a
// curated per-vendor table sourced from DSDcc / DSD-FME / a service manual.
func RegisterPreset(p KnoxPreset) error
```

The table is guarded by an `RWMutex`: `SetKnoxTone` takes the write lock,
the decoder's tone-frame branch reads once per matching frame (~50 ns, lost in
the noise of the ~5 µs synthesis). When a pair is registered, a knox frame
synthesizes through the exact same summed-sinewave `synthDualTone` path DTMF
uses, with phase carried across frames so a held tone is click-free. When it
isn't, the frame falls through to silence. No guessed frequencies ship in the
tree — a claim about a tone we can't verify is worse than an honest silence.

### The problem we hit here

The tempting shortcut is to hardcode *some* vendor's knox frequencies as a
default. We didn't, and the reason is the same discipline the rest of the codec
follows: the AMBE+2 tone map for `[144, 163]` isn't in the public spec, and the
three vendors we know of disagree. Shipping one vendor's numbers as a default
would mean confidently synthesizing the *wrong* tone on two-thirds of systems —
an inaccurate technical claim baked into audio. Better to expose the seam and
let an operator with a real reference fill it in.

## Where this goes next

[Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
climbs up a layer to the composer — the component that owns the per-protocol
chains feeding these vocoders, decides which vocoder a grant needs, and wires
`SetFrameErrors` from the FEC layer into the decoder.

## FAQ

**Does the AMBE+2 vocoder correct bit errors?**
No. Forward-error-correction (Golay(23,12) + descramble on DMR, trellis + RS on
P25 Phase 2) runs in the protocol decoders and yields a 49-bit payload. The
vocoder does *concealment* on what's left: bad-frame replay and adaptive
smoothing.

**What happens on a run of unrecoverable frames?**
The decoder replays the last good frame with `0.7ⁿ` attenuation for up to
`MaxBadFrames = 6` frames (~120 ms), then clears its cache and emits silence.
The AGC is frozen during replay so the fade is audible rather than pumped back
up.

**What is a knox tone?**
A vendor-specific dual-tone (call-alert) signalled by AMBE+2 tone indices
`b1 ∈ [144, 163]`. The public spec doesn't document the frequencies, so
GopherTrunk synthesizes them only when an operator registers a `(freqA, freqB)`
pair via `SetKnoxTone` or `RegisterPreset`; otherwise the frame is silence.

**Why not ship default knox frequencies?**
Because different vendors use different frequencies for the same index, and the
values aren't publicly specified. Shipping one vendor's numbers as a default
would synthesize the wrong tone on other systems — an inaccurate claim in
audio. The mechanism ships; the numbers are the operator's to supply.

## Series navigation

**Part 8 of 12** · ←
[Part 7: AMBE+2 — One Decoder, Two Rates]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }})
· Next →
[Part 9: The Composer]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
