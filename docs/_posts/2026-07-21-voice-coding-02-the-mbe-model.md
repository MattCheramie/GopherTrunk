---
title: "Voice Coding, Part 2: The MBE Model — Pitch, Voicing & Spectral Envelope"
description: The Multi-Band Excitation parameter set at the heart of IMBE and AMBE+2 — fundamental frequency, per-band voiced/unvoiced decisions, and spectral amplitudes, as the Go structs both codecs synthesize from.
category: deep-dives
keywords: multi band excitation, mbe model, fundamental frequency pitch, voiced unvoiced decision, spectral amplitudes, imbe ambe shared core, mbe params go struct, gophertrunk voice
tags: [voice, mbe, imbe, ambe, dsp, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 2
---

*Part 2 of **Voice Coding**. Part 1 established that a vocoder ships a model of
the voice, not the sound, and that IMBE and AMBE+2 both synthesize from a shared
core. This post is that core's data model: the Multi-Band Excitation parameter
set, and the two Go structs — `mbe.Header` and `mbe.Params` — that every codec in
GopherTrunk fills in and every synthesis primitive reads.*

> **TL;DR:** Multi-Band Excitation describes 20 ms of speech with three things: a
> **fundamental frequency** `W0` (the pitch), a **per-harmonic voicing decision**
> `Vl[1..L]` (buzz or hiss, band by band), and a set of **spectral log-amplitude
> residuals** `Tl[1..L]` (how loud each harmonic is). GopherTrunk pins that to two
> structs in `internal/voice/mbe`. IMBE and AMBE+2 differ only in how they unpack
> bits into those fields; from there down, the synthesis code is shared and
> codec-blind.

**Key takeaways**

- MBE splits the speech spectrum into **bands around each pitch harmonic** and
  makes an independent voiced/unvoiced call **per band** — the "multi-band" idea,
  and the thing that makes it sound better than single-decision predecessors.
- `mbe.Header` is `{W0, L, Silent}`: pitch in radians/sample, harmonic count, and
  a silence flag. `L` ranges 9..56 and is *derived* from the pitch.
- `mbe.Params` adds `Vl[1..L]` (voicing) and `Tl[1..L]` (spectral residuals),
  both **1-indexed** per TIA-102.BABA — index 0 is deliberately unused.
- `Tl` is the residual **before** cross-frame prediction. The prediction that
  turns it into real amplitudes needs the previous frame — so it lives in the
  synthesizer (Part 3), not the parameter struct.

## Cheat sheet

| Field | Type | Meaning |
|---|---|---|
| `W0` | `float64` | fundamental frequency ω₀ in **radians/sample** |
| `L` | `int` | number of harmonics, 9..56 (derived from `W0`) |
| `Silent` | `bool` | frame is an explicit silence indicator |
| `Vl[1..L]` | `[MaxL+1]int` | per-harmonic voicing: `1` = voiced (buzz), `0` = unvoiced (hiss) |
| `Tl[1..L]` | `[MaxL+1]float64` | spectral log-amplitude residuals, **pre-prediction** |
| `MaxL` | `const = 56` | upper bound; arrays sized `[MaxL+1]` so `[1..L]` is addressable |
| `SamplesPerFrame` | `const = 160` | PCM samples one synthesis call produces (8 kHz × 20 ms) |

## In this post

- **The source-filter idea** — why speech decomposes into a pitch and an
  envelope, and where "multi-band" comes in.
- **The header** — pitch, harmonic count, silence, and why `L` is derived.
- **The parameter set** — voicing decisions and spectral residuals, and the
  1-indexing convention.
- **The shared seam** — how one struct keeps two codecs DRY.

## Speech as a source and a filter

Every model in this series rests on one observation about how humans make sound.
Speech is a **source** driving a **filter**. The source is either the vocal folds
opening and closing at a pitch — a buzz rich in harmonics — or turbulent air
rushing past a constriction — a hiss with no pitch at all. The filter is the vocal
tract: the throat, tongue, and lips forming a resonant chamber whose shape
sculpts the source into a recognisable vowel or consonant.

Vowels are voiced: fold buzz shaped by the tract. Fricatives like `s` and `f` are
unvoiced: pure hiss. But most speech is neither purely one nor the other — a `z`
is a buzz *and* a hiss at once, voiced in the low frequencies and turbulent up
high. That's the insight that names the model. **Multi-Band Excitation** doesn't
make one voiced/unvoiced decision for the whole frame; it splits the spectrum into
bands, one around each harmonic of the pitch, and decides **band by band** whether
that slice of spectrum is a buzz or a hiss.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A voice spectrum split into harmonic bands at multiples of the fundamental frequency; low bands are marked voiced and high bands unvoiced, illustrating the per-band voicing decision of Multi-Band Excitation">
  <line x1="40" y1="170" x2="660" y2="170" stroke="currentColor"/>
  <text x="350" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">frequency →  harmonics at l · ω₀  (l = 1, 2, 3, … L)</text>
  <line x1="40" y1="170" x2="40" y2="40" stroke="currentColor"/>
  <text x="30" y="40" text-anchor="end" fill="var(--fg-muted)" font-size="10">|amp|</text>
  <!-- voiced harmonics: tall lines -->
  <g stroke="var(--accent)" stroke-width="3">
    <line x1="90" y1="170" x2="90" y2="70"/>
    <line x1="150" y1="170" x2="150" y2="55"/>
    <line x1="210" y1="170" x2="210" y2="85"/>
    <line x1="270" y1="170" x2="270" y2="105"/>
  </g>
  <text x="180" y="34" text-anchor="middle" fill="var(--accent)" font-size="11">voiced bands (buzz)</text>
  <!-- unvoiced harmonics: hatched/noisy -->
  <g stroke="var(--fg-muted)" stroke-width="2" opacity="0.8">
    <line x1="330" y1="170" x2="330" y2="120"/>
    <line x1="360" y1="170" x2="360" y2="135"/>
    <line x1="390" y1="170" x2="390" y2="118"/>
    <line x1="420" y1="170" x2="420" y2="140"/>
    <line x1="450" y1="170" x2="450" y2="125"/>
    <line x1="480" y1="170" x2="480" y2="145"/>
    <line x1="510" y1="170" x2="510" y2="128"/>
    <line x1="540" y1="170" x2="540" y2="150"/>
    <line x1="570" y1="170" x2="570" y2="132"/>
    <line x1="600" y1="170" x2="600" y2="148"/>
  </g>
  <text x="470" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="11">unvoiced bands (hiss)</text>
  <!-- envelope curve over the harmonics -->
  <path d="M90 70 Q140 45 150 55 Q200 75 210 85 Q260 100 270 105 Q400 120 480 135 Q560 140 600 148" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="150" y="120" text-anchor="middle" fill="currentColor" font-size="10">spectral envelope (Ml)</text>
</svg>
<figcaption>MBE places one harmonic at each multiple of the pitch ω₀, decides voiced-or-unvoiced per band, and describes the whole thing with a spectral envelope of amplitudes. A single frame can be voiced low and unvoiced high — that's what a voiced fricative looks like.</figcaption>
</figure>

So a frame boils down to: the pitch (which sets *where* the harmonics sit), a
voiced/unvoiced flag *per* harmonic, and an amplitude *per* harmonic. That is
exactly what the two structs below hold.

## The header: pitch, count, silence

```go
// internal/voice/mbe/frame.go
type Header struct {
    W0     float64 // fundamental frequency in radians/sample
    L      int     // number of harmonics (IMBE 9..56; AMBE+2 similar)
    Silent bool    // true when the frame is an explicit silence indicator
}
```

Three fields, and two of them repay a closer look.

**`W0` is in radians per sample, not hertz.** The synthesis math works in
discrete-time angular frequency, so a 150 Hz pitch at 8 kHz is
`2π · 150 / 8000 ≈ 0.118` rad/sample. Keeping it in radians means the harmonic
generator can write `l · W0` for the l-th harmonic's frequency with no unit
conversion in the inner loop.

**`L` is derived, not independent.** The number of harmonics is set by the pitch:
a low, slow pitch packs more harmonics under the ~4 kHz voice band than a high one
does, so `L` and `W0` are two views of the same fact. In IMBE, `L` runs from 9
(high pitch, few harmonics) to 56 (low pitch, many). That upper bound is a package
constant with a comment worth quoting, because it explains the array sizes you'll
see everywhere in this series:

```go
// internal/voice/mbe/frame.go
const (
    SamplesPerFrame = 160 // 8 kHz × 20 ms, both IMBE and AMBE+2
    PCMSampleRate   = 8000
    MaxL            = 56  // arrays sized [MaxL+1] so [1..MaxL] is addressable
)
```

`Silent` is the escape hatch: some frames are an explicit "this is silence"
indicator rather than a set of parameters, and every synthesis primitive
short-circuits on it. On the IMBE side that flag is set when the fundamental-
frequency codeword `b_0` lands in a reserved silence window — a detail we'll meet
in Part 4.

## The parameter set: voicing and spectral residuals

The header is what comes "straight off" the pitch codeword. The full frame adds
the two per-harmonic arrays:

```go
// internal/voice/mbe/params.go
type Params struct {
    Header
    Vl [MaxL + 1]int     // Vl[1..L] voicing decisions (0=unvoiced, 1=voiced)
    Tl [MaxL + 1]float64 // Tl[1..L] spectral log-amplitude residuals
}
```

`Vl[l]` is the per-band voicing decision from the figure above: `1` if harmonic
`l` is a buzz, `0` if it's a hiss. Part 3's synthesizer routes voiced harmonics to
a sinusoid generator and unvoiced ones to a noise generator on exactly this flag.

`Tl[l]` is the spectral shape — but note the field name says **residual**, not
amplitude, and the comment is emphatic about why:

> *`Tl` is the residual before the inter-frame log-amplitude prediction (eq.
> 75-77); the prediction reads previous-frame state from `SynthState`.*

MBE doesn't transmit each harmonic's absolute loudness every frame — that would
waste bits, because a voice's spectral envelope changes slowly. Instead it
transmits the *difference* from what you'd predict by interpolating the previous
frame's envelope. `Tl` is that difference. Turning it into the real linear
amplitudes `Ml[1..L]` requires the previous frame, which is why that step lives in
the stateful synthesizer and not in this stateless struct. Part 3 is where `Tl`
becomes `Ml`.

**Everything is 1-indexed.** `Vl[0]` and `Tl[0]` are unused; the valid range is
`[1..L]`. This mirrors the TIA-102.BABA specification and the reference decoders
exactly, and the arrays are sized `[MaxL+1] = [57]` precisely so `[1..56]` is
addressable without off-by-one gymnastics. It looks un-Go-ish, and it is — but
matching the spec's indexing means every equation transcribes verbatim, and this
team weighs faithful transcription over idiomatic slicing.

## The shared seam: one struct, two codecs

Here's the design spine made concrete. Both codecs construct an `mbe.Params` and
hand it to the same pipeline. From the package doc:

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="Both the IMBE and AMBE+2 decoders perform codec-specific bit unpacking, then converge on a shared mbe.Params struct that feeds the common synthesis pipeline of PredictLog2Ml, AmplitudesFromLog2Ml, EnhanceAmplitudes, SynthVoiced, and SynthUnvoiced">
  <rect x="16" y="30" width="150" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="91" y="48" text-anchor="middle" fill="currentColor" font-size="11">IMBE unpack</text>
  <text x="91" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Gm · Cik · b_0</text>
  <rect x="16" y="100" width="150" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="91" y="118" text-anchor="middle" fill="currentColor" font-size="11">AMBE+2 unpack</text>
  <text x="91" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">two-stage VQ</text>

  <line x1="166" y1="50" x2="220" y2="78" stroke="currentColor"/><polygon points="218,73 226,80 214,82" fill="currentColor"/>
  <line x1="166" y1="120" x2="220" y2="92" stroke="currentColor"/><polygon points="214,88 226,90 218,97" fill="currentColor"/>

  <rect x="226" y="65" width="130" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="291" y="83" text-anchor="middle" fill="var(--accent)" font-size="12">mbe.Params</text>
  <text x="291" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">W0 · Vl · Tl</text>

  <line x1="356" y1="85" x2="400" y2="85" stroke="var(--accent)"/><polygon points="400,81 410,85 400,89" fill="var(--accent)"/>
  <rect x="410" y="55" width="254" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="537" y="76" text-anchor="middle" fill="var(--accent)" font-size="11">shared synthesis pipeline</text>
  <text x="537" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">PredictLog2Ml → Amplitudes → Enhance</text>
  <text x="537" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ SynthVoiced + SynthUnvoiced</text>
</svg>
<figcaption>The codec-specific work is unpacking bits into an mbe.Params. Everything after the struct is shared, codec-blind Go — the DRY seam this series is built around.</figcaption>
</figure>

The IMBE package has its own richer `Params` type (with `Gm` PRBA gain blocks and
`Cik` DCT coefficients), and it projects down to the shared shape with a tiny
`MBE()` method that drops the codec-specific intermediates and keeps only
`{W0, L, Silent}` + `Vl` + `Tl`. AMBE+2 does the same from its two-stage VQ
codebook indices. The shared package never sees a Golay bit or a codebook index —
it consumes `mbe.Params` and nothing else.

### Why the shared core is the right factoring

You could imagine writing IMBE and AMBE+2 as two independent decoders, each with
its own synthesis. It would work, and it would be a maintenance trap: the voiced
harmonic generator, the unvoiced noise excitation, the cross-frame prediction, the
overlap-add smoothing — that's the *hard*, DSP-heavy, artifact-prone code, and
it's **identical** for both codecs because they both target the same MBE speech
model. Factoring it behind `mbe.Params` means a fix to the synthesis (like the
female-voice prediction-gain fix we'll see in Part 3) improves *both* codecs at
once, and the per-codec packages stay small — they're just bit-unpackers that end
at a struct.

## Where this goes next

[Part 3]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})
takes an `mbe.Params` and grows 160 samples of speech from it: summed sinusoids
for the voiced bands, filtered noise for the unvoiced bands, and the inter-frame
smoothing that stops frame boundaries from clicking. That's where `Tl` finally
becomes real amplitude and where `W0` drives real oscillators. After that, Parts
4–6 reverse-engineer how IMBE fills this struct in from 88 bits, and Parts 7–8 do
the same for AMBE+2.

## FAQ

**What does "multi-band" mean in Multi-Band Excitation?**
It means the voiced/unvoiced decision is made **per frequency band** — one band
around each pitch harmonic — rather than once for the whole frame. That lets a
single frame be voiced in the low frequencies and unvoiced up high, which is what
real voiced fricatives (`z`, `v`) actually are.

**Why is the pitch stored in radians per sample instead of hertz?**
Because the synthesis math is discrete-time. Storing ω₀ in radians/sample lets the
harmonic generator write `l · W0` directly for the l-th harmonic with no
per-sample hertz-to-radians conversion.

**Why are the MBE arrays 1-indexed with an unused element 0?**
To transcribe the TIA-102.BABA equations and the reference decoders verbatim. The
spec indexes harmonics from 1 to L, so the Go arrays are sized `[MaxL+1]` and
`[0]` is deliberately left unused — faithfulness beats idiom here.

**Why is `Tl` called a residual instead of an amplitude?**
Because MBE transmits the *difference* between each harmonic's log-amplitude and a
prediction interpolated from the previous frame's envelope, to save bits. Turning
`Tl` into the real linear amplitude needs the previous frame, so that step lives
in the stateful synthesizer, not the parameter struct.

**Do IMBE and AMBE+2 really share this struct?**
Yes. Each has its own codec-specific `Params` with extra fields, but both project
down to `mbe.Params` (`W0`, `Vl`, `Tl`) via an `MBE()` method, and the entire
synthesis pipeline consumes only that shared shape.

## Series navigation

**Part 2 of 12** · ←
[Part 1: What a Vocoder Is]({{ '/blog/deep-dives/voice-coding-01-what-is-a-vocoder/' | relative_url }})
· Next →
[Part 3: MBE Synthesis]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})
