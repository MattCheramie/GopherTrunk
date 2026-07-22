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

<figure class="lab-figure">
<svg viewBox="0 0 680 120" width="680" height="120" role="img" aria-label="The composer's per-call demodulation chain: a voice-device IQ stream passes through an FM or protocol receiver, an audio low-pass filter, an optional AGC stage, and a resampler, emerging as 16-bit PCM handed to the recorder.">
  <rect x="6" y="42" width="84" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="48" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">IQ stream</text>
  <text x="48" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">StreamIQ</text>
  <line x1="90" y1="64" x2="102" y2="64" stroke="currentColor"/><polygon points="102,60 112,64 102,68" fill="currentColor"/>
  <rect x="112" y="42" width="104" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="164" y="62" text-anchor="middle" fill="currentColor" font-size="10">FM / proto RX</text>
  <text x="164" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">demod</text>
  <line x1="216" y1="64" x2="228" y2="64" stroke="currentColor"/><polygon points="228,60 238,64 228,68" fill="currentColor"/>
  <rect x="238" y="42" width="90" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="283" y="66" text-anchor="middle" fill="currentColor" font-size="10">audio LPF</text>
  <line x1="328" y1="64" x2="340" y2="64" stroke="currentColor"/><polygon points="340,60 350,64 340,68" fill="currentColor"/>
  <rect x="350" y="42" width="94" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="397" y="62" text-anchor="middle" fill="currentColor" font-size="10">AGC</text>
  <text x="397" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">optional</text>
  <line x1="444" y1="64" x2="456" y2="64" stroke="currentColor"/><polygon points="456,60 466,64 456,68" fill="currentColor"/>
  <rect x="466" y="42" width="90" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="511" y="66" text-anchor="middle" fill="currentColor" font-size="10">resample</text>
  <line x1="556" y1="64" x2="568" y2="64" stroke="currentColor"/><polygon points="568,60 578,64 568,68" fill="currentColor"/>
  <rect x="578" y="42" width="96" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="626" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">16-bit PCM</text>
  <text x="626" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">to recorder</text>
</svg>
<figcaption>The composer (<code>internal/voice/composer</code>) runs this chain per call over consumer-owned <code>IQSource</code>/<code>PCMSink</code> interfaces, so it never imports the whole SDR or recorder packages.</figcaption>
</figure>

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

<figure class="lab-figure">
<svg viewBox="0 0 680 180" width="680" height="180" role="img" aria-label="Output fan-out on the streaming side: the recorder writes a WAV then publishes KindCallComplete; the broadcast Manager subscribes to it, encodes MP3 once with internal/voice/mp3, and pushes to four backends — Broadcastify, RdioScanner, OpenMHz and Icecast — with bounded exponential-backoff retry.">
  <rect x="8" y="68" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="88" text-anchor="middle" fill="currentColor" font-size="10">recorder</text>
  <text x="68" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="8">writes .wav</text>
  <text x="154" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="8">KindCallComplete</text>
  <line x1="128" y1="91" x2="178" y2="91" stroke="currentColor"/><polygon points="178,87 188,91 178,95" fill="currentColor"/>
  <rect x="188" y="64" width="140" height="54" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="258" y="86" text-anchor="middle" fill="var(--accent)" font-size="10">broadcast.Manager</text>
  <text x="258" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">subscribes complete</text>
  <line x1="328" y1="91" x2="360" y2="91" stroke="currentColor"/><polygon points="360,87 370,91 360,95" fill="currentColor"/>
  <rect x="370" y="68" width="108" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="424" y="88" text-anchor="middle" fill="currentColor" font-size="10">MP3 encode</text>
  <text x="424" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="8">voice/mp3</text>
  <line x1="478" y1="82" x2="540" y2="26" stroke="currentColor"/><polygon points="536,22 546,24 540,33" fill="currentColor"/>
  <line x1="478" y1="88" x2="540" y2="70" stroke="currentColor"/><polygon points="536,66 546,68 539,77" fill="currentColor"/>
  <line x1="478" y1="94" x2="540" y2="112" stroke="currentColor"/><polygon points="539,105 546,114 536,116" fill="currentColor"/>
  <line x1="478" y1="100" x2="540" y2="156" stroke="currentColor"/><polygon points="538,149 546,158 534,158" fill="currentColor"/>
  <rect x="548" y="12" width="126" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="611" y="30" text-anchor="middle" fill="currentColor" font-size="10">broadcastify</text>
  <rect x="548" y="56" width="126" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="611" y="74" text-anchor="middle" fill="currentColor" font-size="10">rdioscanner</text>
  <rect x="548" y="100" width="126" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="611" y="118" text-anchor="middle" fill="currentColor" font-size="10">openmhz</text>
  <rect x="548" y="144" width="126" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="611" y="162" text-anchor="middle" fill="currentColor" font-size="10">icecast</text>
</svg>
<figcaption>The broadcaster is a bus subscriber, not something the recorder calls: it waits for <code>KindCallComplete</code>, encodes MP3 once, and fans the file out to each configured backend with bounded retry.</figcaption>
</figure>

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
