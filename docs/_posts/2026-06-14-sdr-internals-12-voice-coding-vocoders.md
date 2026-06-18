---
title: "SDR in Pure Go, Part 12: Voice Coding — IMBE, AMBE+2 & MBE"
description: How GopherTrunk turns digital voice frames into audio with pure-Go IMBE and AMBE+2 vocoders sharing one MBE synthesis core, all behind a pluggable Vocoder registry.
category: deep-dives
tags: [sdr, go, vocoder, imbe, ambe, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 12
---

*Part 12 of **SDR Internals**. Digital voice isn't recorded audio — it's a model
of your voice, a few kilobits per second. This post covers the vocoders that
reconstruct speech and the registry that makes them pluggable.*

## In this post

- What a **vocoder** is and why digital radio can't work without one.
- GopherTrunk's pure-Go **IMBE** (P25 Phase 1) and **AMBE+2** (P25 Phase 2, DMR)
  decoders.
- The **shared MBE synthesis core** and the **`Vocoder` registry** — a plugin +
  DRY design.

## What a vocoder does

A [vocoder]({{ '/reference/vocoder/' | relative_url }}) compresses speech to a few
kilobits per second by modeling how speech is *produced* — pitch, voicing, and a
spectral envelope — rather than recording the waveform. The radio sends those
model parameters; the receiver resynthesizes speech from them. Every digital
voice mode depends on one. ([digital voice]({{ '/learn/rf-sdr/digital-voice/' | relative_url }}))

GopherTrunk implements two in pure Go:

- **IMBE** — 4.4 kbps,
  [P25 Phase 1]({{ '/reference/p25-phase-1/' | relative_url }}).
  ([reference]({{ '/reference/imbe/' | relative_url }}))
- **AMBE+2** — 2.4 kbps, P25 Phase 2, DMR, NXDN.
  ([reference]({{ '/reference/ambe-plus-2/' | relative_url }}))

Both belong to the
[Multi-Band Excitation]({{ '/reference/multi-band-excitation/' | relative_url }})
family — and that shared lineage is the key to the design.

## How GopherTrunk implements it in Go

IMBE and AMBE+2 differ in how they *unpack and error-correct* their on-air bits,
but they **synthesize** speech almost identically. So GopherTrunk factors the
synthesis into a shared `internal/voice/mbe` core, and the two codecs become thin
front-ends:

- `internal/voice/imbe` — unpacks 144-bit IMBE frames, runs
  [Golay]({{ '/reference/golay-code/' | relative_url }})/Hamming FEC,
  de-interleaves, and produces MBE parameters.
- `internal/voice/ambe2` — unpacks AMBE+2 frames, runs BPTC FEC and codebook
  lookups, and produces MBE parameters.
- `internal/voice/mbe` — takes those parameters and synthesizes 8 kHz PCM:
  voiced-harmonic generation, unvoiced FFT excitation with overlap-add, spectral
  enhancement, and per-frame AGC.

Every vocoder satisfies one interface:

```go
// internal/voice/vocoder.go (shape)
type Vocoder interface {
    Name() string
    FrameSize() int
    Decode(frame []byte) ([]int16, error)
    Reset()
    Close() error
}
```

…and registers itself with a factory registry, exactly like the SDR drivers from
[Part 2]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }}):

```go
var DefaultRegistry = NewRegistry() // name -> VocoderFactory
// imbe/register.go and ambe2/register.go call Register() at init()
```

## The design principle: plugin registry + shared-core DRY

Two principles combine. The **plugin registry** lets the engine pick a vocoder by
name without importing it. And **DRY** — don't repeat yourself — drives the shared
`mbe` core, so the genuinely-common synthesis math is written once.

### How that principle shaped the Go code

- **One synthesis engine, many front-ends.** Because IMBE and AMBE+2 both reduce
  to MBE parameters, the hard DSP — harmonic synthesis, overlap-add, enhancement —
  lives in a single tested package. A future Codec2 or hardware vocoder reuses it.
- **Per-call instances, no shared state.** Each active call gets its own
  `Vocoder` instance with its own `Reset()`-able state (cross-frame prediction
  needs memory), so two simultaneous calls never interfere.
- **Backends swap by name.** The protocol decoder asks the registry for "imbe" or
  "ambe2"; the planned DVSI hardware backend slots in behind the same interface
  under a build tag, with zero changes to the decoders.
- **A `NullVocoder` always exists.** Registering a silence vocoder means the
  pipeline always has a valid backend, so an unsupported codec degrades to silence
  instead of crashing.

## Where this goes next

Vocoders are a deep, fascinating topic — the source-filter model, how MBE splits
the spectrum into voiced and unvoiced bands, and the patent history that makes a
clean-room Go implementation valuable. A future **"Digital Voice in Go"** series
can dissect a single frame bit by bit. Next, we wire these audio samples into
recordings and live streams.

## FAQ

**Why are IMBE and AMBE+2 implemented separately if they share a core?**
Their bit-packing and FEC differ completely — only the *synthesis* is shared. The
split front-ends handle the on-air differences; the common `mbe` package handles
what's genuinely identical.

**Is implementing AMBE in Go a patent issue?**
Re-implementing a codec in a new language doesn't change its patent status.
GopherTrunk provides the decoders; whether a given codec is encumbered depends on
your jurisdiction. IMBE's patents have expired; AMBE+2's situation varies.

**Why 8 kHz mono PCM output?**
That's the native rate vocoders model speech at — 160 samples per 20 ms frame. It
matches telephone-quality voice and keeps recordings small.

## Series navigation

**Part 12 of 14** · ←
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
· Next →
[Part 13: Recording, composition & streaming]({{ '/blog/deep-dives/sdr-internals-13-recording-composition-streaming/' | relative_url }})
