---
title: "Recording, Composition & Streaming, Part 1: The Output Half"
description: How GopherTrunk turns a decoded call into files, feeds, and uploads — four independent, config-gated subsystems that subscribe to a handful of bus events instead of calling each other, so recording, persistence, live audio, and streaming never block one another.
category: deep-dives
keywords: sdr call recording architecture, event bus subscriber, call start end complete events, config driven lazy init, recorder call log broadcast, audio publisher, gophertrunk recording composition streaming, pure go scanner output
tags: [recording, streaming, architecture, events, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 1
---

*Part 1 of **Recording, Composition & Streaming**, a 14-part deep dive into
GopherTrunk's output half — everything that happens after the vocoders in the
[Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }}) series produce
PCM. This opener is the map: the four subsystems that turn a decoded call into a
file, a database row, a live feed, and an upload, and the one design rule that
lets them coexist without ever calling each other. We'll follow a single call —
call it the 3 p.m. dispatch on talkgroup 101 — from the moment its audio exists
to the moment it lands in a Broadcastify feed, across the whole series.*

> **TL;DR:** Once a call is decoded, four **independent, optional** subsystems
> react to it — the **recorder** (WAV on disk), the **call log** (a SQLite row),
> the **audio publisher** (live browser/gRPC audio), and the **broadcast
> manager** (uploads to aggregators). None of them calls the others; they all
> **subscribe** to the same four bus events — `KindCallStart`, `KindCallSegment`,
> `KindCallEnd`, `KindCallComplete`. The unifying rule is **config-driven lazy
> init**: a subsystem exists only if its YAML section is present, so a disabled
> feature isn't flag-gated dead code — it is never constructed and costs nothing.

**Key takeaways**

- The output half is **four subscribers on one bus**, not a call chain. Slow
  uploads can't back-pressure recording; a missing database can't stop live audio.
- Four events carry a call's whole life: **start**, **segment** (a
  transmission boundary), **end**, and **complete** (the WAV is flushed and
  closed). Each subsystem cares about a different subset.
- **`KindCallEnd` ≠ `KindCallComplete`.** End fires the instant the engine tears
  the call down; complete fires later, once the recorder has a finished file to
  hand to the uploader. That gap is the seam the whole series hangs on.
- **Optional means absent, not idle.** No broadcast section → no manager, no MP3
  encoder, no goroutines. The daemon builds the dependency graph from config.

## Cheat sheet

| Subsystem | Constructed by | Subscribes to | Produces |
|---|---|---|---|
| Recorder | `voice.NewRecorder` (`internal/voice/recorder.go`) | start · segment · end · encryption | `.wav` (+ `.raw`), then `KindCallComplete` |
| Call log | `storage.NewCallLog` (`internal/storage/calllog.go`) | start · end | a `call_log` row |
| Audio publisher | `api.NewAudioPublisher` (`internal/api/audio_publisher.go`) | decoded PCM tap | live HTTP/gRPC audio |
| Broadcast manager | `buildBroadcastManager` (`cmd/gophertrunk/broadcast.go`) | `KindCallComplete` | uploads to aggregators |
| Retention | `storage.NewRetention` (`internal/storage/retention.go`) | timer, not events | deletes old rows/files |

## In this post

- **The four events** that describe a call's life, and why there are four.
- **The four subscribers**, and which events each one wants.
- **Why a bus** instead of the recorder calling the uploader directly.
- **Config-driven lazy init** — the rule that makes every subsystem optional.
- **The thread we'll pull** — one call, PCM to Broadcastify, over 14 posts.

## Where a call arrives

By the time this series begins, the hard part is done. The
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) has
turned a control-channel grant into a call and retuned a voice SDR to its
frequency; the [Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }})
vocoders are turning that channel's IQ into 8 kHz PCM. What we have, in other
words, is a stream of audio samples and some metadata about whose call it is. The
output half's job is to do something durable and useful with them.

There are four things a scanner operator might want from a call, and GopherTrunk
treats each as a separate concern:

- **Keep it** — write a playable audio file that survives a crash.
- **Index it** — record who/what/when so the call is searchable later.
- **Hear it now** — stream the audio live to a browser or the host speakers.
- **Share it** — upload it to Broadcastify, RdioScanner, OpenMHz, or an Icecast feed.

The tempting design is a straight line: recorder writes the file, then calls the
indexer, then calls the uploader. GopherTrunk deliberately does not do that.
Instead, all four hang off the event bus introduced in
[Trunking Engine Part 2]({{ '/blog/series/trunking-engine/' | relative_url }}).

## The four events of a call's life

The engine and the composer publish a small, fixed vocabulary. Four kinds
describe a call from cradle to grave:

```go
// internal/events/bus.go (shape)
const (
    KindCallStart    Kind = "call.start"    // voice SDR retuned; audio incoming
    KindCallSegment  Kind = "call.segment"  // a transmission boundary mid-call
    KindCallEnd      Kind = "call.end"      // engine tore the call down
    KindCallComplete Kind = "call.complete" // recorder flushed & closed the WAV
)
```

The payloads are equally small, and they live with the engine in
`internal/trunking/grant.go`. `CallStart` says a device has been pointed at a
call; `CallEnd` says it's over and carries the quality figures the composer
measured; `CallComplete` adds the one thing the engine can't know — the path to
the finished file:

```go
// internal/trunking/grant.go (shape)
type CallComplete struct {
    Grant        Grant
    Talkgroup    *TalkGroup
    DeviceSerial string
    StartedAt    time.Time
    EndedAt      time.Time
    Reason       EndReason
    AudioPath    string // the .wav the recorder wrote
    SampleRate   uint32 // its PCM rate in Hz
}
```

The reason there are **two** end-of-call events is the single most important
thing in this post. `KindCallEnd` fires the instant the engine decides the call
is over — the hangtime expired, or a higher-priority grant preempted the voice
SDR. At that moment there is no finished file yet: the recorder still has to
flush its buffer, patch the WAV header, and close the handle. Only when that's
done does the recorder publish `KindCallComplete` with a real `AudioPath`. Any
subsystem that needs the *file* (the uploader) waits for complete; any subsystem
that only needs the *metadata* (the call log) can act on end. Collapsing the two
would mean either uploading a half-written file or blocking the engine on disk
I/O. Keeping them separate is what lets the engine move on immediately.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="One decoded call publishes four events on the bus; four independent subsystems each subscribe to the subset they need: the recorder to start, segment and end; the call log to start and end; the audio publisher to the live PCM tap; and the broadcast manager to call-complete only.">
  <rect x="8" y="80" width="120" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="68" y="100" text-anchor="middle" fill="var(--accent)" font-size="11">decoded call</text>
  <text x="68" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">PCM + metadata</text>
  <line x1="128" y1="103" x2="160" y2="103" stroke="currentColor"/><polygon points="160,99 170,103 160,107" fill="currentColor"/>
  <rect x="170" y="72" width="90" height="62" rx="6" fill="none" stroke="currentColor"/>
  <text x="215" y="97" text-anchor="middle" fill="currentColor" font-size="11">event bus</text>
  <text x="215" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">start · segment</text>
  <text x="215" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">end · complete</text>
  <line x1="260" y1="90" x2="300" y2="34" stroke="currentColor"/><polygon points="296,30 306,32 300,41" fill="currentColor"/>
  <line x1="260" y1="98" x2="300" y2="74" stroke="currentColor"/><polygon points="296,70 306,74 299,82" fill="currentColor"/>
  <line x1="260" y1="110" x2="300" y2="118" stroke="currentColor"/><polygon points="299,113 306,120 296,122" fill="currentColor"/>
  <line x1="260" y1="120" x2="300" y2="162" stroke="currentColor"/><polygon points="298,155 306,164 294,164" fill="currentColor"/>
  <rect x="306" y="16" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="381" y="31" text-anchor="middle" fill="currentColor" font-size="10">recorder</text>
  <text x="381" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="8">start·segment·end</text>
  <rect x="306" y="60" width="150" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="381" y="78" text-anchor="middle" fill="currentColor" font-size="10">call log · start·end</text>
  <rect x="306" y="104" width="150" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="381" y="122" text-anchor="middle" fill="currentColor" font-size="10">audio publisher · live</text>
  <rect x="306" y="148" width="150" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="381" y="163" text-anchor="middle" fill="var(--accent)" font-size="10">broadcast manager</text>
  <text x="381" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="8">call.complete only</text>
  <line x1="456" y1="163" x2="500" y2="163" stroke="currentColor"/><polygon points="500,159 510,163 500,167" fill="currentColor"/>
  <rect x="510" y="148" width="140" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="580" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Broadcastify · RdioScanner</text>
  <text x="330" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9"></text>
</svg>
<figcaption>Four subsystems, one bus. Each subscribes only to the events it needs; the broadcast manager waits for <code>call.complete</code> because it needs the finished file, not just the metadata.</figcaption>
</figure>

## The four subscribers

Each subsystem is a long-lived object that the daemon constructs at startup and
wires to the bus. None of them imports the others.

- **The recorder** (`voice.NewRecorder`) is the busiest. It subscribes to
  `KindCallStart` to open a session, `KindCallSegment` to roll to a new file at a
  transmission boundary, and `KindCallEnd` to finalize. It is also the *lone
  decoder*: the same decode that fills the WAV feeds the live tap, so audio is
  decoded once and fanned out (Part 2). When it closes a file it publishes
  `KindCallComplete`.
- **The call log** (`storage.NewCallLog`) subscribes to `KindCallStart` (INSERT a
  row with a null end time) and `KindCallEnd` (UPDATE it with duration, end
  reason, and the quality figures). It never touches the audio (Part 10).
- **The audio publisher** (`api.NewAudioPublisher`) isn't event-driven in the
  same way — it receives decoded PCM through a sink the recorder feeds, and fans
  it to any connected HTTP or gRPC listener (Part 12).
- **The broadcast manager** (`buildBroadcastManager`) subscribes to
  `KindCallComplete` alone, reads the finished WAV, encodes MP3 once, and pushes
  to each configured backend with bounded retry (Parts 13–14).

The daemon holds them as plain fields and builds each behind a config check:

```go
// cmd/gophertrunk/daemon.go (shape)
type daemon struct {
    recorder  *voice.Recorder
    audioPub  *api.AudioPublisher
    callLog   *storage.CallLog   // nil unless a database is configured
    retention *storage.Retention // nil unless a retention window is set
    // …broadcast manager built and Run() only when a feed is enabled
}
```

## Why a bus, not a call chain

Making these four subscribers rather than a pipeline buys three properties that a
straight call chain can't.

First, **isolation of failure.** A Broadcastify upload that hangs for thirty
seconds on a bad network is a problem contained entirely inside the broadcast
manager's worker goroutine. Because it reacts to an event rather than being
called by the recorder, it can never back-pressure the decode or the file write.
The recorder published `KindCallComplete` and moved on; whether the upload
succeeds, retries, or is dropped is the manager's business.

Second, **independent lifecycles.** You can run the recorder with no database,
the database with no recordings directory (decode-only, for live audio), or the
broadcaster with neither — each is wired only if its config asks for it. There is
no "recording implies logging implies uploading" coupling to unpick.

Third, **testability.** Each subsystem can be exercised by publishing synthetic
events onto a bus and asserting on its output — no need to stand up the whole
daemon. That's the same observer discipline the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series
leans on, extended to the output side.

### How that principle shaped the Go code

- **The recorder never imports `internal/broadcast`.** The seam between them is a
  `KindCallComplete` event carrying a file path — a string and some metadata, not
  a Go call. Either side can be tested, replaced, or disabled without touching
  the other.
- **One decode, many sinks.** Rather than the recorder and the live-audio path
  each decoding the channel, the recorder decodes once and writes to a
  `fanoutSink` — a slice of PCM sinks. Recording and live listening share a
  single vocoder pass (Part 2).
- **Metadata and audio travel separately.** The call log acts on `KindCallEnd`
  (metadata is ready immediately); the uploader waits for `KindCallComplete` (the
  file isn't). Splitting the events lets each consumer act as early as it safely can.

## Config-driven lazy init

The rule that ties the output half together: **a subsystem exists only if its
config section is present.** This is not a feature flag that skips a code path at
runtime — it's the absence of an object.

```go
// cmd/gophertrunk/broadcast.go (shape)
func buildBroadcastManager(cfg BroadcastConfig, /* … */) (*broadcast.Manager, error) {
    if !cfg.anyBackendEnabled() {
        return nil, nil // no feed configured → no Manager at all
    }
    // …build backends, construct Manager, which subscribes to the bus
}
```

The daemon does the same for the call log (built only when a database path is
configured) and for retention (constructed only when at least one of
`retention.call_log_days`, `retention.log_days`, or `retention.files_days` is
set). A GopherTrunk instance streaming live audio with no recordings, no
database, and no uploads allocates none of the recording, persistence, or
broadcast machinery. Disabled features aren't idle — they're not there.

<figure class="lab-figure">
<svg viewBox="0 0 640 200" width="640" height="200" role="img" aria-label="A vertical mapping from YAML config sections to the subsystems each one instantiates: the recordings section builds the recorder, the database section builds the call log, the retention section builds the retention sweeper, the api section builds the audio publisher, and the broadcast section builds the broadcast manager. Absent sections build nothing.">
  <text x="120" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">config.yaml section</text>
  <text x="470" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">instantiated subsystem</text>
  <rect x="24" y="36" width="180" height="26" rx="5" fill="none" stroke="currentColor"/><text x="114" y="53" text-anchor="middle" fill="currentColor" font-size="11">recordings:</text>
  <line x1="204" y1="49" x2="356" y2="49" stroke="currentColor"/><polygon points="356,45 366,49 356,53" fill="currentColor"/>
  <rect x="366" y="36" width="250" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="491" y="53" text-anchor="middle" fill="var(--accent)" font-size="11">voice.Recorder</text>
  <rect x="24" y="70" width="180" height="26" rx="5" fill="none" stroke="currentColor"/><text x="114" y="87" text-anchor="middle" fill="currentColor" font-size="11">database:</text>
  <line x1="204" y1="83" x2="356" y2="83" stroke="currentColor"/><polygon points="356,79 366,83 356,87" fill="currentColor"/>
  <rect x="366" y="70" width="250" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="491" y="87" text-anchor="middle" fill="var(--accent)" font-size="11">storage.CallLog</text>
  <rect x="24" y="104" width="180" height="26" rx="5" fill="none" stroke="currentColor"/><text x="114" y="121" text-anchor="middle" fill="currentColor" font-size="11">retention:</text>
  <line x1="204" y1="117" x2="356" y2="117" stroke="currentColor"/><polygon points="356,113 366,117 356,121" fill="currentColor"/>
  <rect x="366" y="104" width="250" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="491" y="121" text-anchor="middle" fill="var(--accent)" font-size="11">storage.Retention</text>
  <rect x="24" y="138" width="180" height="26" rx="5" fill="none" stroke="currentColor"/><text x="114" y="155" text-anchor="middle" fill="currentColor" font-size="11">broadcast:</text>
  <line x1="204" y1="151" x2="356" y2="151" stroke="currentColor"/><polygon points="356,147 366,151 356,155" fill="currentColor"/>
  <rect x="366" y="138" width="250" height="26" rx="5" fill="none" stroke="var(--accent)"/><text x="491" y="155" text-anchor="middle" fill="var(--accent)" font-size="11">broadcast.Manager</text>
  <rect x="24" y="172" width="180" height="22" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><text x="114" y="187" text-anchor="middle" fill="var(--fg-muted)" font-size="10">(section absent)</text>
  <line x1="204" y1="183" x2="356" y2="183" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <rect x="366" y="172" width="250" height="22" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><text x="491" y="187" text-anchor="middle" fill="var(--fg-muted)" font-size="10">nothing built — zero cost</text>
</svg>
<figcaption>The daemon builds the output-half dependency graph from configuration. Each subsystem's presence is decided once, at startup, by whether its YAML section exists.</figcaption>
</figure>

## The thread we'll pull

Over the next thirteen posts we follow one call end to end. Part 2 is the
**contract** that carries its composed audio into the recorder without the two
packages knowing each other's types. Part 3 is **composition** proper — how a
burst of overs becomes one call, and how a transmission boundary becomes a
`KindCallSegment`. Parts 4–7 are the **recorder**: session lifecycle, the
crash-safe WAV, file naming and raw sidecars, and the subtle guards that stop one
call's audio from bleeding into the next. Part 8 places **loudness** at the right
seam. Part 9 is the `CallComplete` **handoff**. Parts 10–11 are **persistence and
retention**. Part 12 is **live listening**. Parts 13–14 are the **broadcast
manager** and the five aggregator **backends** — the last step, where our 3 p.m.
dispatch finally becomes a row in a Broadcastify feed.

## Where this goes next

[Part 2]({{ '/blog/deep-dives/recording-streaming-02-composition-contract/' | relative_url }})
opens the seam between composition and recording: the narrow, consumer-owned
interfaces (`IQSource`, `PCMSink`, `RawFrameSink`) that let the composer hand off
audio without importing the recorder — and why the recorder, not the composer, is
the one place a call is decoded.

## FAQ

**What's the difference between `KindCallEnd` and `KindCallComplete`?**
`KindCallEnd` fires the instant the engine tears the call down — before any file
is finished. `KindCallComplete` fires later, once the recorder has flushed and
closed the WAV, and it carries the on-disk `AudioPath`. Subsystems that need the
file (the uploader) wait for complete; subsystems that need only metadata (the
call log) act on end.

**Why don't the recorder, logger, and uploader just call each other?**
So a slow or failing step can't block the others. They're independent bus
subscribers: the recorder publishes an event and moves on, and a thirty-second
upload stall stays contained in the broadcast manager's own goroutine, never
back-pressuring decode or disk writes.

**Can I record without uploading, or stream live without recording?**
Yes. Every subsystem is independent and config-gated. Omit the `broadcast:`
section for local-only recording; run with an empty recordings directory for
decode-only live audio with no files on disk.

**Does enabling a feature I don't use cost anything?**
No. "Optional means absent, not idle" — a subsystem with no config section is
never constructed, so it allocates no goroutines, buffers, or encoders.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Composition Contract]({{ '/blog/deep-dives/recording-streaming-02-composition-contract/' | relative_url }})
