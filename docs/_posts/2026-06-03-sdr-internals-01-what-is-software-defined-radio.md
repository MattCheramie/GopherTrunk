---
title: "SDR in Pure Go, Part 1: What Is Software-Defined Radio?"
description: A pure-Go tour of software-defined radio — what SDR is, the GopherTrunk signal pipeline from RF to audio, and the layered architecture behind a single static binary.
category: deep-dives
tags: [sdr, go, dsp, architecture, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 1
---

*This is the opening post of **SDR Internals**, a 14-part series that walks the
entire software-defined-radio pipeline behind GopherTrunk — one component per
post — and explains the software-design principle behind each piece and how
that principle shaped the Go code.*

## In this post

- **What software-defined radio is** and why it moves radio from soldered
  hardware into software you can change.
- **The GopherTrunk pipeline**: RF → IQ samples → DSP → symbols → protocol
  decode → events → audio.
- **The design principle for the whole engine**: a *layered architecture* with
  a strict one-way dependency direction, built as a single pure-Go
  (`CGO_ENABLED=0`) static binary.
- **A map of the series** so you can jump to the component you care about.

## What is software-defined radio?

A traditional radio is a chain of fixed analog parts — a mixer here, a filter
there, a demodulator soldered to a board. Each part does exactly one job at one
frequency. **Software-defined radio (SDR)** replaces that fixed chain with one
generic move: digitize the radio spectrum as early as possible, then do every
remaining step — tuning, filtering, demodulation, decoding — in software.

An SDR dongle hands your computer a torrent of **IQ samples**: complex numbers
that capture both the amplitude and the phase of the radio signal in a slice of
spectrum. Once the signal is just numbers, "build a radio" becomes "write a
program." That is the whole premise of GopherTrunk: a P25, DMR, TETRA, NXDN and
multi-protocol trunking scanner where every block — from the USB driver to the
voice codec — is Go code.

> New to the fundamentals? See the reference entries on
> [software-defined radio]({{ '/reference/software-defined-radio/' | relative_url }})
> and [IQ data]({{ '/reference/iq-data/' | relative_url }}), or the learn-path
> lesson [What is software-defined radio?]({{ '/learn/what-is-sdr/' | relative_url }}).

## The GopherTrunk pipeline

Every signal takes the same journey. Each stage is its own Go package, and each
becomes its own post in this series:

```
RF  →  IQ samples  →  DSP            →  symbols  →  protocol  →  events  →  audio
       (internal/sdr)  (internal/dsp)            (internal/radio) (trunking) (voice)
```

1. **`internal/sdr`** — pure-Go USB drivers for RTL-SDR, HackRF, and Airspy
   produce a stream of `[]complex64` IQ chunks.
2. **`internal/dsp`** — filters, oscillators, channelizers, demodulators, and
   timing-recovery loops turn wideband IQ into clean symbol streams.
3. **`internal/radio`** — per-protocol decoders (P25, DMR, NXDN, TETRA, …) turn
   symbols into framed, error-corrected control messages.
4. **`internal/trunking`** + **`internal/events`** — an engine consumes those
   messages and publishes domain events (a call started, a channel was granted).
5. **`internal/voice`** — vocoders and a recorder turn voice frames into WAV
   audio; **`internal/api`** serves it to clients.

## The design principle: a strict, layered architecture

The single most important architectural decision in GopherTrunk is that
**dependencies only ever point one way**: the SDR layer knows nothing about DSP,
DSP knows nothing about protocols, protocols know nothing about the trunking
engine, and the engine knows nothing about the API, storage, or UI that consume
its output.

### How that principle shaped the Go code

That one-way rule shows up concretely in three Go idioms you'll see throughout
the series:

- **Typed channels as the seams between layers.** A device exposes
  `<-chan []complex64`; a DSP stage consumes one channel and produces another.
  Layers are wired together by channels, not by calling into each other.
- **Interfaces owned by the consumer.** Where one layer *does* need a
  capability from another, it declares a small interface describing only what it
  needs — so the dependency is on a contract, not a concrete type. This is
  classic Go "accept interfaces, return structs."
- **An event bus instead of back-references.** The engine never calls the API,
  storage, or broadcaster. It publishes events to `internal/events`, and those
  subsystems subscribe. The core stays testable in isolation (more in
  [Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})).

And the principle that ties it all together: **pure Go, no CGO.** Every block —
USB transport, DSP, FEC math, even the voice codecs — is implemented in Go with
`CGO_ENABLED=0`. There is no `librtlsdr`, no `libusb`, no `libmp3lame`. The
payoff is one statically linked binary that cross-compiles to Linux, macOS, and
Windows with `go build`, and a codebase where you can read the radio from
antenna to audio without leaving the language.

## The series map

| Part | Component | Design principle |
|---|---|---|
| 1 | What is SDR? (this post) | Layered architecture |
| [2]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }}) | SDR devices & the driver registry | Registry / dependency inversion |
| [3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }}) | The SDR pool & streaming concurrency | Pipes-and-filters + CSP |
| [4]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }}) | DSP foundations: filters, NCO, AGC | Stateful streaming + zero-alloc reuse |
| [5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }}) | Tuning & channelization | Strategy pattern |
| [6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }}) | Demodulation (FM, C4FM, GFSK, …) | Single responsibility |
| [7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}) | Symbol timing & sync recovery | Feedback state machines |
| [8]({{ '/blog/deep-dives/sdr-internals-08-equalization-diversity-fft/' | relative_url }}) | Equalization, diversity & FFT | Decorator + interface segregation |
| [9]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }}) | Framing & forward error correction | Pure functions & table-driven code |
| [10]({{ '/blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/' | relative_url }}) | Protocol decoders as state machines | Adapter + uniform contract |
| [11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }}) | Trunking engine & event bus | Observer / event-driven |
| [12]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }}) | Voice coding: IMBE, AMBE+2, MBE | Plugin registry + shared core |
| [13]({{ '/blog/deep-dives/sdr-internals-13-recording-composition-streaming/' | relative_url }}) | Recording, composition & streaming | Config-driven lazy init |
| [14]({{ '/blog/deep-dives/sdr-internals-14-apis-testing-pure-go/' | relative_url }}) | APIs, testing & the pure-Go story | Ports & adapters + testability |

Each post is an overview — enough to understand the component, the Go that
implements it, and the principle behind it. Where a topic deserves more, it will
get its own dedicated deep-dive series later. You can always find every part on
the [SDR Internals series index]({{ '/blog/series/sdr-internals/' | relative_url }}).

## FAQ

**Is software-defined radio just a USB dongle?**
The dongle is only the digitizer. SDR is the idea that everything *after*
sampling — tuning, filtering, demodulating, decoding — happens in software. The
dongle produces IQ samples; the radio is the program that processes them.

**Why build an SDR stack in Go instead of C or C++?**
Go gives you memory safety, first-class concurrency (goroutines and channels map
beautifully onto a streaming DSP pipeline), and trivial cross-compilation. With
`CGO_ENABLED=0` the whole thing ships as one static binary, with no shared
libraries to install on the target machine.

**Do I have to read the series in order?**
No. Part 1 is the map; each later part stands alone. But the pipeline flows in
order, so reading top-to-bottom mirrors the path a signal actually takes.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: SDR devices & the driver registry]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }})
