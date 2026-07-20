---
title: "Voice Coding, Part 1: What a Vocoder Is & the Pluggable Registry"
description: How GopherTrunk turns 2400-bit-per-second radio data back into speech — what a vocoder actually models, and the Go Vocoder interface + registry that lets IMBE and AMBE+2 plug in behind one seam.
category: deep-dives
keywords: what is a vocoder, imbe vs ambe plus 2, p25 voice decoder, digital voice codec, vocoder interface go, strategy pattern registry, mbe multi band excitation, gophertrunk vocoder
tags: [voice, vocoder, imbe, ambe, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 1
---

*Part 1 of **Voice Coding**, a 12-part deep dive into how GopherTrunk turns a
handful of bits per frame back into human speech. The
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
series hands us decoded voice frames; the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series
hands us the calls those frames belong to. This series is about the last mile —
the codec — and it starts with the idea that makes the whole thing possible.*

> **TL;DR:** A vocoder does not compress *recorded audio*. It transmits a small
> **model of how the voice was produced** — a pitch, a set of voiced/unvoiced
> decisions, and a spectral envelope — a few dozen numbers every 20 ms, and
> **re-synthesizes** speech from them at the far end. That is how P25 fits voice
> into 4.4 kbit/s (IMBE) or 2.4 kbit/s (AMBE+2). GopherTrunk hides every codec
> behind one `Vocoder` interface and a name-keyed `Registry`, so the decode path
> asks for `"imbe"` or `"ambe2"` by string and never learns which one it got.

**Key takeaways**

- A vocoder is a **speech production model**, not an audio compressor — it sends
  parameters and rebuilds the waveform, which is why it survives at bitrates a
  waveform codec (MP3, Opus) can't touch.
- GopherTrunk's `Vocoder` is a five-method Go interface. IMBE, AMBE+2, a future
  DVSI hardware backend, and a silent `NullVocoder` all satisfy it.
- A process-global `Registry` maps a **name** to a **factory**; drivers register
  from `init()`, callers resolve by the string that arrives on a `CallStart`.
- IMBE (4.4 kbps, P25 Phase 1) and AMBE+2 (2.4 kbps, P25 Phase 2 / DMR / NXDN)
  are different bit-level codecs that **synthesize from the same MBE core** —
  the design spine this whole series unpacks.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| The interface | `Vocoder` (`internal/voice/vocoder.go`) | five methods every codec implements: `Name`/`FrameSize`/`Decode`/`Reset`/`Close` |
| The registry | `Registry` (`vocoder.go`) | name → `VocoderFactory`; process-global `DefaultRegistry` |
| Registration | `init()` in each subpackage | `DefaultRegistry.Register("imbe", …)` — drivers self-register |
| Resolution | `Registry.New(name)` | one fresh instance per call, so per-call state never bleeds |
| The safe default | `NullVocoder` (`vocoder.go`) | emits silence; touches no patented algorithm |
| The shared core | `internal/voice/mbe` | Multi-Band Excitation synthesis both real codecs drive |

## In this post

- **What a vocoder models** — production, not recording, and why that buys the
  bitrate.
- **The interface** — the five methods, and why `Decode` is one frame in / 160
  samples out.
- **The registry** — strategy pattern at the subsystem scale, and how a name
  becomes a decoder.
- **IMBE vs AMBE+2** — two codecs, one synthesis core, and the map of the
  series.

## A vocoder models the voice, it doesn't record it

Start with the number that should feel impossible. An AMBE+2 voice frame is
**2400 bits per second**. A single second of CD audio is roughly 1.4 *million*
bits. The vocoder is throwing away 99.8% of the data and you can still recognise
your neighbour's voice on the other end. No waveform compressor — not MP3, not
Opus — gets close to that ratio on speech and stays intelligible.

The trick is that a vocoder does not send the sound. It sends a **description of
the vocal tract that made the sound**. Human speech is produced by a source (the
vocal folds buzzing at a pitch, or turbulent air hissing) shaped by a filter
(the throat, mouth, and nose forming an ever-changing resonant envelope). A
vocoder measures those two things every 20 ms and transmits *them*:

- a **fundamental frequency** — the pitch the vocal folds are buzzing at;
- a set of **voiced / unvoiced decisions** — for each frequency band, is this a
  buzz (a vowel) or a hiss (an `f`, an `s`)?
- a **spectral envelope** — how loud each harmonic of that pitch is, i.e. the
  shape the mouth is making.

The far end never receives your voice. It receives that parameter set and
**re-grows a voice** from it — summing sinusoids for the buzzy bands and filtering
noise for the hissy ones. It sounds a little synthetic because it *is* synthetic;
that's the "vocoder" timbre. The word is a portmanteau of **voice** and **coder**,
and it dates to 1930s telephone research for exactly this reason.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A waveform codec transmits the sound wave itself, while a vocoder transmits a small set of speech-production parameters — pitch, voicing, and spectral envelope — and re-synthesizes the waveform at the receiver">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="11">waveform codec (MP3 / Opus): send the sound</text>
  <path d="M20 55 q10 -22 20 0 q10 22 20 0 q10 -22 20 0 q10 22 20 0 q10 -22 20 0 q10 22 20 0 q10 -18 20 0" fill="none" stroke="var(--fg-muted)"/>
  <line x1="200" y1="55" x2="250" y2="55" stroke="var(--fg-muted)"/>
  <polygon points="250,51 260,55 250,59" fill="var(--fg-muted)"/>
  <text x="330" y="59" text-anchor="middle" fill="var(--fg-muted)" font-size="11">~kilobits, big</text>

  <line x1="20" y1="82" x2="660" y2="82" stroke="currentColor" stroke-dasharray="3 3" opacity="0.4"/>

  <text x="20" y="108" fill="var(--accent)" font-size="11">vocoder (IMBE / AMBE+2): send the recipe</text>
  <rect x="20" y="120" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="138" text-anchor="middle" fill="var(--accent)" font-size="10">pitch ω₀</text>
  <text x="95" y="151" text-anchor="middle" fill="var(--accent)" font-size="10">voiced / unvoiced</text>
  <text x="95" y="164" text-anchor="middle" fill="var(--accent)" font-size="10">spectral envelope</text>
  <line x1="170" y1="146" x2="230" y2="146" stroke="currentColor"/>
  <polygon points="230,142 240,146 230,150" fill="currentColor"/>
  <rect x="240" y="128" width="120" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="300" y="150" text-anchor="middle" fill="currentColor" font-size="11">synthesize</text>
  <line x1="360" y1="146" x2="410" y2="146" stroke="currentColor"/>
  <polygon points="410,142 420,146 410,150" fill="currentColor"/>
  <path d="M430 146 q8 -18 16 0 q8 18 16 0 q8 -18 16 0 q8 18 16 0 q8 -18 16 0 q8 18 16 0" fill="none" stroke="currentColor"/>
  <text x="600" y="150" text-anchor="middle" fill="var(--accent)" font-size="11">~2.4 kbps, tiny</text>
</svg>
<figcaption>A waveform codec ships a compressed copy of the sound. A vocoder ships the parameters of the voice that produced it and rebuilds the sound locally — the reason speech survives at a few kilobits per second.</figcaption>
</figure>

If you want the from-scratch tour of that idea, the Field Guide entry on the
[vocoder]({{ '/reference/vocoder/' | relative_url }}) and the
[digital voice]({{ '/learn/rf-sdr/digital-voice/' | relative_url }}) lesson both
approach it from the RF side. This series takes the codec apart in Go.

## The interface: one frame in, 160 samples out

Every decoder GopherTrunk can use satisfies one small interface. Here it is,
faithful to the source:

```go
// internal/voice/vocoder.go
type Vocoder interface {
    Name() string
    FrameSize() int                   // input bytes per frame
    Decode(frame []byte) ([]int16, error)
    Reset()
    Close() error
}
```

`Decode` is the whole job: hand it **one compressed frame** and it returns
**16-bit PCM at 8 kHz mono** — one frame is 20 ms, which is exactly **160
samples**, for both IMBE and AMBE+2. That 20 ms / 160-sample cadence is a
constant of the MBE family, not an accident of one codec.

The other four methods are the plumbing that makes a *stateful* codec safe to use
as a black box. Most vocoders carry memory between frames — the previous frame's
pitch and phase feed the next frame's synthesis (Part 3 lives on that memory) —
so `Reset()` exists to wipe it on a re-sync, and `Close()` releases anything the
implementation holds. The doc comment is blunt about the consequence: decoders
that keep internal state are **not safe for concurrent calls on the same
instance**. That single sentence drives the registry design below.

## The registry: a name becomes a decoder

GopherTrunk never news-up a codec directly. It asks a registry for one by name:

```go
// internal/voice/vocoder.go (shape)
type Registry struct {
    mu sync.RWMutex
    v  map[string]VocoderFactory
}

// A factory builds a FRESH instance per call, so per-call state never leaks.
type VocoderFactory func() (Vocoder, error)

var DefaultRegistry = NewRegistry() // process-global; init() populates it

func (r *Registry) New(name string) (Vocoder, error) {
    r.mu.RLock(); f, ok := r.v[name]; r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("voice: unknown vocoder %q", name)
    }
    return f()
}
```

This is the **strategy pattern**, applied at the scale of a whole subsystem. The
recorder that opens a `.wav` file for a call knows only that a `CallStart` event
named a vocoder — `"imbe"`, `"ambe2"`, `"null"` — and calls
`DefaultRegistry.New(name)`. It gets back *a* `Vocoder` and drives it. It has no
import of the IMBE package, no `switch` on codec type, no idea whether a decode
is 4.4 kbps Golay-protected bits or a 2.4 kbps DMR burst.

Two design decisions in that small struct are worth naming:

- **Factories, not instances.** The map stores `func() (Vocoder, error)`, not a
  `Vocoder`. Because a stateful codec is unsafe to share across concurrent calls,
  the registry mints a *new* decoder per call. Two talkgroups decoding in
  parallel each get their own pitch/phase memory; nothing bleeds.
- **Self-registration from `init()`.** Each codec subpackage registers itself:
  the IMBE package does `DefaultRegistry.Register("imbe", …)` in its `init()`.
  Link the package into the build and the name appears; leave it out (say, a
  patent-restricted build) and it silently doesn't. `voice.go` registers only
  the always-safe `NullVocoder` — a decoder that emits 160 samples of silence
  and touches no patented algorithm — as the fallback when no real codec is
  linked or a `CallStart` names one that isn't registered
  (`ErrNoVocoder`).

### How that principle shaped the Go code

The payoff of the interface-plus-registry seam is that **codecs are additive**.
When the pure-Go AMBE+2 decoder landed, nothing in the recorder, the call log, or
the trunking engine changed — the new package registered `"ambe2"` from its
`init()` and the existing `Registry.New(name)` call resolved it. The same is true
for the future DVSI hardware backend: it becomes one more `Register` call and one
more `Vocoder` implementation, not a rewrite of the decode path.

It also keeps the licensing story honest. IMBE's US patents have **expired**;
AMBE+2 carries **active patents** in some jurisdictions, and re-implementing it in
Go doesn't change that posture. Because every codec is behind the same seam, a
build that omits AMBE+2 doesn't fail — it just resolves `"ambe2"` to
`ErrNoVocoder` and the call records silence through `NullVocoder`. The policy
lives at the registry boundary, not smeared through the pipeline.

## IMBE vs AMBE+2: two codecs, one core

The two codecs this series decodes look different on the wire and share a spine.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="IMBE and AMBE+2 have separate bit-level frame formats and quantization tables but both produce an MBE parameter set that feeds one shared synthesis core, which outputs 8 kHz PCM">
  <rect x="20" y="20" width="200" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="120" y="42" text-anchor="middle" fill="currentColor" font-size="12">IMBE 4400</text>
  <text x="120" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="10">4.4 kbps · P25 Phase 1</text>
  <text x="120" y="71" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Golay/Hamming · scrambler</text>

  <rect x="20" y="128" width="200" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="120" y="150" text-anchor="middle" fill="currentColor" font-size="12">AMBE+2 2400</text>
  <text x="120" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="10">2.4 kbps · P25 P2 / DMR / NXDN</text>
  <text x="120" y="179" text-anchor="middle" fill="var(--fg-muted)" font-size="10">two-stage VQ · Golay+Knox</text>

  <line x1="220" y1="49" x2="300" y2="90" stroke="currentColor"/>
  <polygon points="298,85 306,93 294,94" fill="currentColor"/>
  <line x1="220" y1="157" x2="300" y2="118" stroke="currentColor"/>
  <polygon points="294,114 306,115 298,123" fill="currentColor"/>

  <rect x="306" y="80" width="180" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="396" y="100" text-anchor="middle" fill="var(--accent)" font-size="12">mbe.Params</text>
  <text x="396" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ω₀ · Vl[1..L] · Tl[1..L]</text>

  <line x1="486" y1="104" x2="540" y2="104" stroke="var(--accent)"/>
  <polygon points="540,100 550,104 540,108" fill="var(--accent)"/>
  <rect x="550" y="80" width="110" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="605" y="100" text-anchor="middle" fill="var(--accent)" font-size="11">MBE synth</text>
  <text x="605" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="10">8 kHz PCM</text>
</svg>
<figcaption>Different bit budgets, tables, and codebooks feed one shared MBE parameter set and one synthesis core. The codec-specific work is unpacking bits into parameters; the speech-growing is common.</figcaption>
</figure>

| | IMBE 4400 | AMBE+2 2400 |
|---|---|---|
| Bitrate | 4.4 kbps | 2.4 kbps (+1.15 kbps FEC in DMR) |
| Systems | P25 Phase 1 (LDU1/LDU2) | P25 Phase 2, DMR, NXDN |
| Parameter coding | PRBA gain blocks + HOC DCT coefficients | two-stage vector quantization codebooks |
| FEC on frame | Golay(23,12) + Hamming(15,11) | Golay + "Knox" convolutional |
| Patent status | expired | active in some jurisdictions |
| GopherTrunk pkg | `internal/voice/imbe` | `internal/voice/ambe2` |

Both decoders do the same final thing: build an `mbe.Params` — a fundamental
frequency `W0`, per-harmonic voicing decisions `Vl[1..L]`, and spectral
log-amplitude residuals `Tl[1..L]` — and feed it through the shared
[Multi-Band Excitation]({{ '/reference/multi-band-excitation/' | relative_url }})
synthesis core in `internal/voice/mbe`. That shared core, kept DRY behind the
codec-specific bit unpacking, is the design spine of the whole series.

## Where this goes next
[Part 2]({{ '/blog/deep-dives/voice-coding-02-the-mbe-model/' | relative_url }})
opens up that shared core: the MBE parameter set itself — what a fundamental
frequency, a per-band voicing decision, and a spectral envelope *are* as Go
structs, and why both codecs converge on `mbe.Header` + `mbe.Params`. From there
Part 3 synthesizes speech from those parameters, and Parts 4–8 reverse-engineer
the two codecs' bit formats that produce them — IMBE first (Parts 4–6), then
AMBE+2 (Parts 7–8).

## FAQ

**What is a vocoder, in one sentence?**
A vocoder is a speech codec that transmits a compact *model* of how the voice was
produced — pitch, voicing, and spectral envelope — instead of the sound itself,
and re-synthesizes speech from that model at the receiver, which is how it reaches
2.4–4.4 kbit/s.

**Why can't I just use MP3 or Opus for P25 voice?**
Those are waveform codecs — they compress the sound wave, and at 2.4 kbps they'd
be unintelligible. A vocoder wins at that bitrate precisely because it doesn't
send a waveform; it sends production parameters and grows a new waveform locally.

**What's the difference between IMBE and AMBE+2?**
IMBE (4.4 kbps) is P25 Phase 1; AMBE+2 (2.4 kbps) is P25 Phase 2, DMR, and NXDN.
They use different bit formats, quantization, and FEC, but both decode to the same
MBE parameter set and share GopherTrunk's synthesis core.

**Why does GopherTrunk build a new vocoder per call instead of reusing one?**
Because vocoders carry state between frames (previous pitch and phase), and that
state isn't safe to share across concurrent calls. The registry stores factories,
not instances, so every call gets a fresh decoder with its own memory.

**What is the NullVocoder for?**
It's the always-safe fallback: it produces silence and touches no patented
algorithm. If a build omits a codec, or a `CallStart` names one that isn't
registered, the recorder falls back to it so the call still records (silently)
instead of erroring out.

## Series navigation

**Part 1 of 12** · Next →
[Part 2: The MBE Model]({{ '/blog/deep-dives/voice-coding-02-the-mbe-model/' | relative_url }})
