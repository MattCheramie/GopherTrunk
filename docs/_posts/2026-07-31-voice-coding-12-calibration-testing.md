---
title: "Voice Coding, Part 12: Calibrating & Testing Vocoders Against References"
description: How GopherTrunk proves its pure-Go IMBE and AMBE+2 decoders are correct — the voice-calibrate harness, golden-frame regression, streaming decode, and cross-correlation against DSD-FME and OP25.
category: deep-dives
keywords: vocoder calibration, voice-calibrate, golden frame regression, cross correlation decode, rms ratio db, reference decoder validation, dsd-fme op25, voice stats crest factor, gophertrunk voice coding
tags: [testing, calibration, vocoder, regression, go, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 12
---

*Part 12 of **Voice Coding**, the finale. Eleven posts built a pure-Go vocoder
stack from bits to conditioned audio. The obvious question about a
from-scratch codec is: how do you know it's *right*? This post is the answer —
the tooling that measures GopherTrunk's output against decoders the world
already trusts, and the series wrap-up.*

> **TL;DR:** GopherTrunk validates its pure-Go vocoders three ways. **Golden
> regression**: deterministic seeding makes decode byte-reproducible, so a
> fixed frame stream must always produce the same PCM. **Calibration**: the
> `voice-calibrate` harness decodes the same raw frames through GopherTrunk and
> reads a reference WAV from DSD-FME / OP25, then reports a loudness offset
> (`RMSRatioDb`) and a waveform-similarity score (`PeakXcorr`). **Field triage**:
> `VoiceStats` summarises a call's decode health without per-frame log spam. Pass
> = `|RMSRatioDb| < 3 dB` and `PeakXcorr > 0.85`.

**Key takeaways**

- Vocoders seed their noise source from a fixed default (`NewWithSeed(0)`), so
  two decoders produce **byte-identical** output for the same frames — the
  foundation of golden-frame regression.
- `calibrate.Compare` quantifies agreement with a reference decoder as two
  numbers: **RMS ratio** (loudness) and **peak cross-correlation** (shape),
  aligned over a ±25 ms lag window.
- `DecodeStream` turns any raw frame file into a WAV through the same registry
  the daemon uses — the offline `gophertrunk decode` path and the calibration
  fixture generator share one code path.
- `VoiceStats` reads **crest factor** and clip percentage to catch an over-hot
  AGC from a single per-call line instead of a log dump.

## Cheat sheet

| Tool | Entry point | Answers |
|---|---|---|
| Calibration | `calibrate.Compare(raw, refWav, name)` | "does it match the reference?" |
| Similarity math | `calibrate.CompareSamples(a, b)` | testable w/ synthetic PCM |
| Streaming decode | `voice.DecodeStream(in, name, out)` | "decode this frame file to WAV" |
| CLI | `voice-calibrate -raw … -ref-wav … -vocoder …` | one-off PASS/FAIL |
| Per-call health | `voice.VoiceStats` | crest factor, clip %, b₀ range |

## In this post

- **Golden-frame regression** — determinism as a test tool.
- **The calibration harness** — RMS ratio and cross-correlation.
- **The CLI** — `voice-calibrate`, thresholds, and streaming decode.
- **Field triage** — what `VoiceStats` catches that FEC counters can't.
- **Series wrap-up.**

## Golden-frame regression

The first line of defense is determinism. An MBE vocoder needs a noise source
for the §6.4 unvoiced excitation and §6.3 phase dispersion — but "random" would
make output untestable. So the constructors seed from a fixed default:

```go
// internal/voice/ambe2/decoder.go (shape)
func New() *Decoder { return NewWithSeed(0) } // fixed seed → reproducible output
```

Two decoders built with the same seed produce **byte-identical** PCM for the
same frame stream. That turns "does the decoder still work?" into a checkable
assertion: run a fixed frame stream through the decoder, compare the output to a
committed golden WAV, fail on any diff. It's the vocoder equivalent of the
[golden test vectors]({{ '/reference/golden-test-vectors/' | relative_url }}) the
rest of the codebase leans on — a change that alters decode by one sample is
caught immediately, not in the field.

## The calibration harness

Golden regression proves the decoder is *stable*. It doesn't prove it's
*correct* — a byte-reproducible decoder can be reproducibly wrong. For
correctness, GopherTrunk compares its output to decoders that are already
trusted: DSD-FME and OP25. Both are fed the *same* raw vocoder frames, and the
harness quantifies how close the two outputs are:

```go
// internal/voice/calibrate/calibrate.go (shape)
func Compare(rawPath, refWavPath, vocoderName string) (Result, error) {
    v, _ := voice.DefaultRegistry.New(vocoderName)
    var inTree []int16
    for i := 0; i < len(rawBytes); i += v.FrameSize() {
        s, _ := v.Decode(rawBytes[i : i+v.FrameSize()])
        inTree = append(inTree, s...)   // GopherTrunk's decode
    }
    refSamples, _, _ := readWav(refWavPath) // the reference decoder's decode
    return CompareSamples(inTree, refSamples), nil
}
```

`CompareSamples` returns two numbers that answer two different questions:

- **`RMSRatioDb`** — loudness offset in dB, positive when GopherTrunk is louder
  than the reference. It isolates *level*, which is exactly the knob the MBE
  AGC's `TargetPeak` tunes. A 4 dB offset means "turn the target down 4 dB," not
  "the decode is broken."
- **`PeakXcorr`** — normalised cross-correlation magnitude in `[0, 1]`, the
  *shape* agreement independent of level. Above 0.85 the two waveforms are
  recognisably the same speech; below 0.5 they're very likely not even the same
  source.

Because two decoders have different pipeline group delays, the harness searches
for the best alignment over `±MaxLagSamples` (200 samples ≈ ±25 ms at 8 kHz):

```go
// internal/voice/calibrate/calibrate.go (shape)
for lag := -maxLag; lag <= maxLag; lag++ {
    // dot product of x against y shifted by lag, normalised by energy
    v := math.Abs(dot / denom)
    if v > bestVal { bestVal, bestLag = v, lag }
}
```

Taking the *absolute* correlation is deliberate: a 180°-phase-inverted decoder
still scores `|xcorr| ≈ 1`, and the sign mismatch is caught separately by
`RMSRatioDb`, so a polarity flip can't sneak through as a "match."

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="The same raw vocoder frames are decoded by the pure-Go vocoder and by a reference decoder, and both PCM streams feed CompareSamples which outputs an RMS ratio, a peak cross-correlation, and a best-alignment lag">
  <rect x="12" y="66" width="128" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="76" y="86" text-anchor="middle" fill="currentColor" font-size="12">raw frames</text>
  <text x="76" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">.raw fixture</text>
  <line x1="140" y1="78" x2="182" y2="46" stroke="var(--fg-muted)"/>
  <polygon points="179,44 189,44 183,52" fill="var(--fg-muted)"/>
  <line x1="140" y1="98" x2="182" y2="130" stroke="var(--fg-muted)"/>
  <polygon points="183,124 189,132 179,132" fill="var(--fg-muted)"/>
  <rect x="190" y="22" width="170" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="275" y="42" text-anchor="middle" fill="var(--accent)" font-size="12">pure-Go vocoder</text>
  <text x="275" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="10">in-tree PCM</text>
  <rect x="190" y="112" width="170" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="275" y="132" text-anchor="middle" fill="currentColor" font-size="12">reference decoder</text>
  <text x="275" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="10">DSD-FME / OP25 WAV</text>
  <line x1="360" y1="43" x2="404" y2="76" stroke="currentColor"/>
  <polygon points="404,71 410,79 400,79" fill="currentColor"/>
  <line x1="360" y1="133" x2="404" y2="100" stroke="currentColor"/>
  <polygon points="400,97 410,97 404,105" fill="currentColor"/>
  <rect x="412" y="66" width="130" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="477" y="86" text-anchor="middle" fill="var(--accent)" font-size="12">CompareSamples</text>
  <text x="477" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">RMS · xcorr · lag</text>
  <line x1="542" y1="88" x2="586" y2="88" stroke="currentColor"/>
  <polygon points="586,84 596,88 586,92" fill="currentColor"/>
  <rect x="596" y="66" width="72" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="632" y="86" text-anchor="middle" fill="currentColor" font-size="11">PASS?</text>
  <text x="632" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">3 dB · 0.85</text>
</svg>
<figcaption>One raw frame source, two decoders, one comparison. Level and shape are scored separately so a fixable loudness offset is never confused with a broken decode.</figcaption>
</figure>

## The CLI and streaming decode

`voice-calibrate` is the operator-facing wrapper — no test-writing required:

```
voice-calibrate -raw call.raw -ref-wav dsdfme.wav -vocoder ambe2-dmr
```

It imports the imbe and ambe2 packages so their factories register, then prints
the `RMSRatioDb`, `PeakXcorr`, and lag, and applies the pass/fail thresholds
(`|RMSRatioDb| < 3 dB` and `PeakXcorr > 0.85`), exiting non-zero under `-strict`
so it slots into CI. Producing the in-tree side of that comparison — turning a
raw frame file into a WAV — is `DecodeStream`, the same path the offline
`gophertrunk decode` tool uses:

```go
// internal/voice/streamdecode.go (shape)
func DecodeStream(in io.Reader, vocoderName string, out io.WriteSeeker) (int, error)
// reads FrameSize()-aligned frames, decodes via the registry, writes a WAV;
// ErrPartialFrame if the input ends mid-frame (the WAV is still closed cleanly)
```

The `WithVocoder` variant lets a caller pass a decoder built with an explicit
seed or AGC config, so a calibration run can pin reproducibility exactly the way
the golden tests do. One decode path serves the daemon, the offline tool, and
the test harness — there's no separate "test decoder" to drift out of sync.

## Field triage: VoiceStats

Calibration needs a reference WAV. In the field you don't have one — you have a
tester saying "it sounds robotic." `VoiceStats` is the compact per-call summary
that turns that into a number without per-frame logging:

```go
// internal/voice/stats.go (shape)
type VoiceStats struct {
    Frames, Voiced, Unvoiced, Silent, Bad, Repeated int
    MeanF0Hz, MeanL, MeanVoicedFrac float64
    MeanAGCGain, MaxPreClipPeak, OutputRMS, CrestFactor float64
    ClipSamples, TotalSamples int
    MinB0, MaxB0 int
}
func (s VoiceStats) ClipPct() float64 { /* limited samples / total */ }
```

The single most useful field is `CrestFactor` (peak/RMS): natural speech sits
around 8–15, and a value near 3–4 with `ClipPct > 0` is the unmistakable
signature of an over-hot AGC slamming frames into the rail — exactly the bug
[Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
described, now diagnosable from one line. The `MinB0`/`MaxB0` range disambiguates
a genuine low-modulation dead-key from empty frames reaching the vocoder (a
mistuned tap pins b₀ at 0). It's exposed through the `StatProvider` interface, so
a vocoder that doesn't track stats simply produces no summary — the recorder and
`gophertrunk decode` type-assert for it.

## Series wrap-up

Twelve parts, one arc: bits to sound to trust.

- Parts **1–3** framed the vocoder problem and the Multi-Band Excitation model,
  then built the shared synthesis core.
- Parts **4–6** decoded IMBE for P25 Phase 1 — the FEC, deinterleave, and
  acquisition that turn a symbol stream into clean frames.
- Parts **7–8** crossed to AMBE+2, showing how one Go decoder serves three
  protocols with two rate tables, and where its error handling really lives.
- Parts **9–11** wired it into the system: the composer's per-protocol chains,
  the post-synthesis conditioning and loudness stack, and the recording /
  encoding / hardware output stage.
- Part **12** closed the loop, measuring pure-Go output against the decoders the
  community already trusts.

The through-line is a single principle stated a dozen ways: *be faithful, and be
honest about the limits.* The default decode path is byte-reproducible and
byte-faithful; every "sound-good" and every hardware escape hatch is opt-in and
labelled; and where the public spec goes dark — the knox tones — GopherTrunk
ships the mechanism and refuses to guess the numbers. A vocoder you can't verify
is a vocoder you can't trust, so the last thing the series built was the ruler.

Voice frames arrive from the
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }}) series;
calls come from the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series. If
you want the rest of the pipeline, those are where to go next.

## FAQ

**How do you know the pure-Go decode is correct, not just consistent?**
Consistency comes from deterministic seeding (golden regression). Correctness
comes from `voice-calibrate`: decoding the same raw frames through GopherTrunk
and through a reference decoder (DSD-FME / OP25), then scoring loudness
(`RMSRatioDb`) and waveform shape (`PeakXcorr`) separately.

**What are the pass thresholds?**
`|RMSRatioDb| < 3 dB` and `PeakXcorr > 0.85`. Above 0.85 cross-correlation the
two decoders are recognisably decoding the same speech; the 3 dB loudness
tolerance is tuned out via the AGC's `TargetPeak`.

**What does a bad crest factor tell me?**
Natural speech has a crest factor around 8–15. A value near 3–4 with a non-zero
clip percentage means the AGC is over-driving the output into the limiter — the
"robotic" signature — diagnosable from the per-call `VoiceStats` line without
enabling per-frame logs.

**Can I decode a raw frame file without the daemon?**
Yes. `voice.DecodeStream` (and the `gophertrunk decode` tool built on it) reads a
raw vocoder frame file, decodes it through the named vocoder from the registry,
and writes a WAV — the same code path the daemon and the calibration harness use.

## Series navigation

**Part 12 of 12** · ←
[Part 11: Recording & Encoding]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }})
