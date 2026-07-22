---
title: "SDR in Pure Go, Part 13: Recording, Composition & Streaming"
description: How GopherTrunk records calls to WAV, drives a per-call demodulation chain with the composer, and streams audio to Broadcastify and RdioScanner — all from optional, config-driven subsystems.
category: deep-dives
tags: [sdr, go, recording, streaming, broadcast, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 13
---

*Part 13 of **SDR Internals**. A call has been granted and the vocoder is
producing audio. This post is about turning that into files and live streams —
and the config-driven design that means you only pay for what you enable.*

## In this post

- The **composer** that runs a per-call demodulation chain.
- The **recorder** that writes crash-safe WAV files.
- **Outbound streaming** to Broadcastify, RdioScanner, OpenMHz, and Icecast.
- The **config-driven lazy initialization** principle behind every optional
  subsystem.

## What these subsystems do

When the engine from
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
starts a call, three things have to happen on the output side:

- **Composition** — open the voice device's IQ stream, run the right demod chain
  (FM passthrough or a protocol receiver → audio low-pass → optional AGC →
  resample → 16-bit PCM), and feed samples to the recorder.
- **Recording** — write those samples to a per-call WAV file, with a sidecar of
  raw frames for debugging if requested.
- **Streaming** — once a call's WAV is flushed, optionally encode it to MP3 and
  push it to public call networks.

These are the
[antenna-to-audio]({{ '/learn/rf-sdr/antenna-to-audio/' | relative_url }}) last mile.

## How GopherTrunk implements it in Go

The **composer** (`internal/voice/composer`) bridges call events to a demod
chain. Notably, it doesn't depend on the whole SDR or recorder packages — it
declares the *narrow* interfaces it actually needs:

```go
// internal/voice/composer — consumer-owned interfaces (shape)
type IQSource interface {
    StreamIQ(ctx context.Context) (<-chan []complex64, error)
    SampleRateHz() uint32
}

type PCMSink interface {
    WritePCM(deviceSerial string, samples []int16) error
}
```

The **recorder** (`internal/voice/recorder.go`) implements `PCMSink`, opening WAV
files and **patching the header length fields on close** — so a recording stays
valid even if the daemon is killed mid-call. It can split a call into per-
transmission segments, and for DMR's two slots it writes separate `_ts1`/`_ts2`
files.

The **broadcaster** (`internal/broadcast`) is a `Manager` that subscribes to
`KindCallComplete`, encodes the audio to MP3 with the pure-Go
`internal/voice/mp3` package, and fans it out to the configured backends
(`broadcastify`, `rdioscanner`, `openmhz`, `icecast`) with bounded
exponential-backoff retry.

## The design principle: config-driven lazy initialization

The unifying rule: **a subsystem exists only if its config section is present.**
No broadcast config → no `Manager`, no subscription, no MP3 encoder, no overhead.
The daemon constructs the dependency graph from configuration at startup.

### How that principle shaped the Go code

- **Optional means absent, not idle.** The daemon (`cmd/gophertrunk/daemon.go`)
  builds the broadcaster, paging receivers, APRS/AIS decoders, and the rest only
  when their YAML sections appear. Disabled features aren't flag-gated dead code —
  they're never instantiated, so they cost nothing.
- **Consumer-owned interfaces keep wiring loose.** The composer's `IQSource` /
  `PCMSink` / `Devices` interfaces describe only what it needs, so it's trivial to
  test with fakes and works equally with a real dongle or a virtual wideband
  tuner.
- **Subscribers, not callers.** The broadcaster reacts to `KindCallComplete` from
  the event bus rather than being called by the recorder — same observer pattern
  as
  [Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }}),
  so streaming never blocks recording.
- **Crash-safety by construction.** Patching WAV headers on close, and bounding
  upload retries with backoff, are small design choices that keep one slow network
  or one crash from corrupting the local record.

## Where this goes next

The recording and streaming path has plenty worth its own series — call
segmentation heuristics, the pure-Go MP3 encoder, and the quirks of each upload
backend's API. That series now exists:
[Recording, Composition & Streaming]({{ '/blog/series/recording-streaming/' | relative_url }})
traces one call from PCM to a Broadcastify upload across 14 parts. Next here, the
finale: how all of this is exposed, observed, and tested.

## FAQ

**Why patch the WAV header on close instead of writing it up front?**
You don't know a call's length until it ends. Writing placeholder header fields
and patching them on close means the file is correct when finished — and still
recoverable if the process dies mid-call.

**Does enabling streaming slow down decoding?**
No. The broadcaster is a separate event-bus subscriber with its own retry loop, so
a slow or failing upload can't back-pressure the decode or recording path.

**Can I record without uploading anywhere?**
Yes. Recording and streaming are independent, config-driven subsystems. Omit the
broadcast section and you get local WAV files with no outbound traffic.

## Series navigation

**Part 13 of 14** · ←
[Part 12]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }})
· Next →
[Part 14: APIs, testing & the pure-Go story]({{ '/blog/deep-dives/sdr-internals-14-apis-testing-pure-go/' | relative_url }})
