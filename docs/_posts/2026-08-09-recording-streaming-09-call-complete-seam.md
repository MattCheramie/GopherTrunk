---
title: "Recording, Composition & Streaming, Part 9: CallComplete — The Seam to Everything Downstream"
description: How GopherTrunk's single CallComplete event decouples recording from streaming and persistence — what finalizeLocked builds, the identity and file metadata it carries, the per-call VoiceStats quality log emitted alongside it, and why the demod signal, EVM, and SNR figures ride the earlier CallEnd instead.
category: deep-dives
keywords: callcomplete event seam, finalizelocked callcomplete payload, decouple recording streaming persistence, per call voicestats quality log, callend evm snr signal dbfs, audiopath sample rate grant identity, gophertrunk event bus handoff, one event fan out subscribers
tags: [events, recording, streaming, architecture, go, audio]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 9
---

*Part 9 of **Recording, Composition & Streaming**. Our 3 p.m. dispatch on
talkgroup 101 is now a finished, loudness-normalized WAV on disk. The recorder is
done with it. Everything else the scanner does with that call — upload it, index
it, show it in a UI — hangs off one event the recorder publishes at exactly this
moment: `KindCallComplete`. This post is about that seam: what `finalizeLocked`
puts into the payload, what it pointedly leaves out, and why that single event is
the reason recording and streaming never have to know each other's types.*

> **TL;DR:** `CallComplete` is the one event that decouples recording from
> everything downstream. `finalizeLocked` builds it after it closes a call's WAV,
> and it carries just enough to act on a **finished file**: the grant identity,
> the resolved talkgroup, start/end timestamps, the end reason, and — the two
> fields the engine could never know — the on-disk `AudioPath` and its
> `SampleRate`. The per-call audio-quality picture (`VoiceStats`: pitch, AGC gain,
> clip %) is *logged* at finalize but not carried on the event, and the demod
> figures (`SignalDbFS`, `EVMPct`, `SNRDb`) ride the earlier `CallEnd` instead.
> Knowing which figure lives on which event is knowing this seam.

**Key takeaways**

- **One event, many subscribers.** The recorder publishes `CallComplete` and
  moves on; the broadcast Manager (and any future subscriber) reacts. No
  subsystem calls another — the seam is a struct on a bus, not a function call.
- **`CallComplete` carries the file; `CallEnd` carries the demod quality.** The
  engine emits `CallEnd` with `SignalDbFS` / `EVMPct` / `SNRDb` the instant the
  call tears down; the recorder emits `CallComplete` later, once a real
  `AudioPath` exists. The split is deliberate.
- **`finalizeLocked` is the sole constructor.** It's the only place a
  `*CallComplete` is built, and it returns `nil` — publishing nothing — for a
  call that captured no usable audio, so downstream never sees a phantom file.
- **`VoiceStats` is a log, not a payload.** The audio-layer quality summary is
  emitted to the operator's log at finalize; it isn't threaded onto the event,
  because its consumer is a human triaging "robotic" audio, not the uploader.

## Cheat sheet

| Thing | Type / function | Where | Role |
|---|---|---|---|
| The completion event | `trunking.CallComplete` | `internal/trunking/grant.go` | Payload of `KindCallComplete` — the downstream seam |
| Its builder | `finalizeLocked` | `internal/voice/recorder.go` | Closes the WAV, returns `*CallComplete` or `nil` |
| The demod figures | `CallEnd.SignalDbFS` / `EVMPct` / `SNRDb` | `internal/trunking/grant.go` | Ride `CallEnd`, feed the call log |
| Per-call quality log | `VoiceStats`, `logVoiceStats`, `voiceStatsFor` | `internal/voice/stats.go`, `recorder.go` | Audio-layer triage summary, logged not carried |
| Clip-warn threshold | `voiceClipWarnPct` | `internal/voice/recorder.go` | Escalates the quality log to WARN |
| Stats capability | `StatProvider`, `ErrorAware` | `internal/voice/stats.go` | Interfaces a vocoder opts into |

## In this post

- **Why the seam exists** — recording and streaming that never import each other.
- **What `finalizeLocked` builds** — the two fields only the recorder knows.
- **The two-event split** — file metadata on complete, demod quality on end.
- **The `VoiceStats` quality log** — why it's logged and not carried.
- **The nil case** — the calls that finish but complete nothing.

## Why the seam exists

Part 1 of this series made the argument in the abstract: the output half is four
independent subscribers on one bus, not a call chain. `CallComplete` is where
that argument becomes concrete. The recorder produces a file; the broadcast
Manager consumes it; and the *only* thing connecting them is a struct carrying a
path. The recorder does not import `internal/broadcast`; the broadcast package
does not import the recorder. Either can be tested, replaced, or disabled without
touching the other, because the seam between them is data.

That matters most when a downstream step is slow or absent. A Broadcastify upload
that stalls for thirty seconds is contained entirely inside the Manager's worker
goroutine — the recorder published `CallComplete` and returned. If no broadcast
section is configured, the Manager doesn't exist and the event simply has one
fewer subscriber. Recording never blocks on, or even knows about, what happens
after the file is written.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="The recorder's finalizeLocked closes a WAV and publishes one KindCallComplete event onto the bus, which fans out to independent subscribers: the broadcast Manager, which reads AudioPath to encode and upload, and any other future subscriber. The recorder does not call any of them directly.">
  <rect x="8" y="76" width="130" height="50" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="73" y="97" text-anchor="middle" fill="var(--accent)" font-size="11">finalizeLocked</text>
  <text x="73" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">close WAV → *CallComplete</text>
  <line x1="138" y1="101" x2="176" y2="101" stroke="currentColor"/><polygon points="176,97 186,101 176,105" fill="currentColor"/>
  <rect x="186" y="80" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="246" y="98" text-anchor="middle" fill="currentColor" font-size="11">event bus</text>
  <text x="246" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">KindCallComplete</text>
  <line x1="306" y1="92" x2="352" y2="48" stroke="currentColor"/><polygon points="348,44 358,46 352,55" fill="currentColor"/>
  <line x1="306" y1="104" x2="352" y2="104" stroke="currentColor"/><polygon points="352,100 362,104 352,108" fill="currentColor"/>
  <line x1="306" y1="114" x2="352" y2="158" stroke="currentColor"/><polygon points="350,151 358,161 346,160" fill="currentColor"/>
  <rect x="358" y="30" width="180" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="448" y="46" text-anchor="middle" fill="var(--accent)" font-size="10">broadcast Manager</text>
  <text x="448" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reads AudioPath → MP3 → upload</text>
  <rect x="358" y="86" width="180" height="36" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="448" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">any future subscriber</text>
  <text x="448" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">same event, no recorder change</text>
  <rect x="358" y="140" width="180" height="36" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="448" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="10">(disabled → absent)</text>
  <text x="448" y="169" text-anchor="middle" fill="var(--fg-muted)" font-size="8">one fewer subscriber</text>
  <line x1="538" y1="49" x2="590" y2="49" stroke="currentColor"/><polygon points="590,45 600,49 590,53" fill="currentColor"/>
  <rect x="600" y="34" width="52" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="626" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">feeds</text>
</svg>
<figcaption>One <code>CallComplete</code> fans out. The recorder publishes and returns; each subscriber reads the fields it needs. Adding or removing a consumer never touches the recorder.</figcaption>
</figure>

## What finalizeLocked builds

`finalizeLocked` is the sole constructor of a `*CallComplete`. It runs under the
recorder's lock at the moment a session's WAV is closed — on a normal call end
(`handleEnd`) or at a per-transmission segment boundary (`handleSegment`). Its job
is to turn a live `recordingSession` into either a completion payload or a
decision to publish nothing.

When it does build one, the payload is small and deliberately file-centric:

```go
// internal/trunking/grant.go (shape)
type CallComplete struct {
    Grant        Grant      // full grant identity: system, protocol, TG, source, freq…
    Talkgroup    *TalkGroup // resolved label, or nil if unknown
    DeviceSerial string     // which voice SDR followed the call
    StartedAt    time.Time
    EndedAt      time.Time
    Reason       EndReason
    AudioPath    string     // the .wav the recorder wrote — only the recorder knows this
    SampleRate   uint32     // its PCM rate in Hz
}
```

The two fields that justify the event's existence are `AudioPath` and
`SampleRate`. Everything else — the grant, the talkgroup, the timestamps, the
reason — the engine already had at `CallEnd`. What the engine could *never* know
is where the finished file landed on disk and at what rate it was written, because
those are decided inside the recorder (see [Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
on naming and [Part 5]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
on the WAV rate). `CallComplete` exists precisely to carry those two facts to the
consumer that needs a real file, the uploader.

The build itself is the tail of `finalizeLocked`:

```go
// internal/voice/recorder.go (shape)
return &trunking.CallComplete{
    Grant:        s.cs.Grant,
    Talkgroup:    s.cs.Talkgroup,
    DeviceSerial: serial,
    StartedAt:    s.startedAt,
    EndedAt:      endedAt,
    Reason:       reason,
    AudioPath:    s.wavPath,
    SampleRate:   s.sampleRate,
}
```

Note it copies the grant off the *session* (`s.cs.Grant`), not off the incoming
`CallEnd`. That's what lets the recorder's in-call backfills — a source ID or
encryption flag recovered on the traffic channel and stitched onto `s.cs.Grant`
mid-call — reach the uploader with the identity the call actually turned out to
have, rather than the grant-time snapshot.

## The two-event split

The single most useful thing to internalize about this seam is which quality
figures live on which event — because they are split, on purpose, across two
events that fire at different times.

The demod-quality figures ride **`CallEnd`**, the event the engine publishes the
instant the call tears down:

```go
// internal/trunking/grant.go (shape)
type CallEnd struct {
    Grant        Grant
    Talkgroup    *TalkGroup
    DeviceSerial string
    StartedAt    time.Time
    EndedAt      time.Time
    Reason       EndReason
    SignalDbFS   *float64 // mean received channel power (dBFS), RSSI-style
    EVMPct       *float64 // RMS error-vector magnitude (%) over the settled decode
    SNRDb        *float64 // estimated symbol SNR (dB)
}
```

All three are pointers because they're optional — the composer measures them over
the settled decode (currently only the P25 Phase 1 chains feed the demod taps
that populate `EVMPct` / `SNRDb`), and they're `nil` on any call ended by the
watchdog, a preemption, or shutdown. Critically, `SignalDbFS` is a channel-power
figure and *not* SNR/EVM — they answer different questions and the struct keeps
them separate.

These figures do **not** appear on `CallComplete`. The reason is the same one
that motivated splitting `CallEnd` from `CallComplete` in the first place
([Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})):
the demod quality is known *immediately*, before any file is finished, and its
consumer — the call log — acts on `CallEnd` so it can persist that quality the
moment the call ends. Making the uploader wait on `CallComplete` for a *file* it
needs, while letting the logger act on `CallEnd` for *metadata* it already has, is
the whole point of having two events. Duplicating the demod figures onto
`CallComplete` would blur that line for no consumer that needs them there.

<figure class="lab-figure">
<svg viewBox="0 0 660 250" width="660" height="250" role="img" aria-label="An annotated split of a call's two end-of-life events. CallEnd fires first at teardown and carries Grant, timestamps, Reason, and the optional demod figures SignalDbFS, EVMPct and SNRDb, read by the call log. CallComplete fires later once the WAV is closed and carries Grant, Talkgroup, timestamps, Reason, plus AudioPath and SampleRate, read by the broadcast Manager. VoiceStats is logged at finalize but rides neither event.">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="10">at teardown →</text>
  <rect x="20" y="34" width="280" height="120" rx="6" fill="none" stroke="currentColor"/>
  <text x="34" y="52" fill="currentColor" font-size="11">CallEnd</text>
  <text x="34" y="70" fill="var(--fg-muted)" font-size="9">Grant · StartedAt · EndedAt · Reason</text>
  <text x="34" y="88" fill="var(--accent)" font-size="9">SignalDbFS *float64  — channel power</text>
  <text x="34" y="103" fill="var(--accent)" font-size="9">EVMPct *float64  — error-vector %</text>
  <text x="34" y="118" fill="var(--accent)" font-size="9">SNRDb *float64  — symbol SNR</text>
  <text x="34" y="140" fill="var(--fg-muted)" font-size="9">→ read by the call log (Part 10)</text>
  <text x="360" y="24" fill="var(--fg-muted)" font-size="10">after WAV closed →</text>
  <rect x="360" y="34" width="280" height="120" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="374" y="52" fill="var(--accent)" font-size="11">CallComplete</text>
  <text x="374" y="70" fill="var(--fg-muted)" font-size="9">Grant · Talkgroup · timestamps · Reason</text>
  <text x="374" y="88" fill="var(--accent)" font-size="9">AudioPath string  — where the .wav is</text>
  <text x="374" y="103" fill="var(--accent)" font-size="9">SampleRate uint32  — its PCM rate</text>
  <text x="374" y="125" fill="var(--fg-muted)" font-size="9">(only the recorder knows these two)</text>
  <text x="374" y="142" fill="var(--fg-muted)" font-size="9">→ read by the broadcast Manager (Part 13)</text>
  <line x1="300" y1="94" x2="360" y2="94" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="330" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="8">later</text>
  <rect x="120" y="184" width="420" height="52" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="330" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="10">VoiceStats — pitch · AGC gain · clip%</text>
  <text x="330" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="9">logged at finalize (logVoiceStats), carried on NEITHER event</text>
</svg>
<figcaption>Which figure lives where. Demod quality rides <code>CallEnd</code> for the logger; file metadata rides <code>CallComplete</code> for the uploader; the audio-layer <code>VoiceStats</code> summary is written to the log and travels on no event at all.</figcaption>
</figure>

## The VoiceStats quality log

There is a third body of quality information about our call, and it goes to a
third place: the operator's log. Where `CallEnd`'s figures describe the *channel*
(how clean the RF and the symbols were), `VoiceStats` describes the *audio* — the
layer the FEC counters can't see:

```go
// internal/voice/stats.go (shape)
type VoiceStats struct {
    Frames, Voiced, Unvoiced, Silent, Bad, Repeated int // frame-class counts
    MeanF0Hz, MeanL, MeanVoicedFrac                 float64 // pitch / spectral
    MeanAGCGain, MinAGCGain, MaxAGCGain             float64 // AGC behaviour
    MaxPreClipPeak, OutputRMS, CrestFactor          float64 // amplitude health
    ClipSamples, TotalSamples                       int
    // …b_0 range + FirstFrameHex for dead-key diagnostics…
}

func (s VoiceStats) ClipPct() float64 { /* 100 * ClipSamples / TotalSamples */ }
```

`finalizeLocked` calls `logVoiceStats` for every call that finalizes. It obtains
the stats through `voiceStatsFor`, which type-asserts the session's vocoder to
the optional `StatProvider` interface — currently only the pure-Go IMBE decoder
implements it, so a vocoder that doesn't track stats simply produces no summary,
and the recorder carries on. When stats exist, the line reports pitch, harmonic
count, AGC gain range, peak, RMS, crest factor, and clip percentage. It's logged
at DEBUG normally, but escalates to WARN when the output is clipping past
`voiceClipWarnPct` (0.5%) — the signature of an over-hot vocoder AGC slamming
speech into the int16 rail:

```go
// internal/voice/recorder.go (shape)
const voiceClipWarnPct = 0.5

if vs.ClipPct() > voiceClipWarnPct {
    r.log.Warn("recorder: voice audio quality — output clipping (vocoder gain too hot)", args...)
    return
}
r.log.Debug("recorder: voice audio quality", args...)
```

The design point for this series: `VoiceStats` is **logged, not carried**. It
rides neither `CallEnd` nor `CallComplete`, because its consumer is a human
triaging a "robotic" or "too loud" field report — not a subsystem making a
routing decision. Threading it onto an event would put diagnostic detail on the
wire that no automated subscriber reads. (The related interface, `ErrorAware`,
lets the same decoder *consume* the per-frame FEC corrected-bit count from the
recorder before it decodes — the input side of the same quality story.)

## The nil case

`finalizeLocked` doesn't always build a `CallComplete`. It returns `nil` — and
the caller publishes nothing — whenever there is no real file to hand
downstream. Two cases matter. A vocoder-decoded call whose every frame was
idle, silent, or bad (`vs.Voiced + vs.Unvoiced == 0`) is a dead-key or idle
carrier: `finalizeLocked` removes the empty outputs and returns `nil`. And a
call that decoded no PCM at all (`dataBytes == 0`) returns `nil` too — though a
digital call keeps its `.raw` sidecar as the only capture even when the WAV is
empty.

This is the seam's quiet correctness guarantee: because the *only* constructor of
a `CallComplete` refuses to build one for a call with no usable audio, no
downstream subscriber ever receives a completion event pointing at a file that
isn't worth uploading. The uploader never has to defend against a phantom path;
if it got a `CallComplete`, there is a real, non-empty WAV at `AudioPath`. The
publish sites make this explicit — they null-check before touching the bus:

```go
// internal/voice/recorder.go (shape)
cc := r.finalizeLocked(s, ce.DeviceSerial, ce.EndedAt, ce.Reason)
r.mu.Unlock()
if cc != nil {
    r.normalizeIfEnabled(cc.AudioPath) // Part 8 — level before anyone reads it
    r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
}
```

That `cc != nil` guard is the same at both publish sites (`handleEnd` and
`handleSegment`), and it's the reason the rest of the output half can treat a
`CallComplete` as a promise: the file exists, it's finished, and — after the
loudness step from [Part 8]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }}) —
it's ready to read.

## Where this goes next

[Part 10]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})
follows the *other* branch of the split we drew here: the call log, which acts on
`CallStart` and `CallEnd` — not `CallComplete` — to write a searchable SQLite row
carrying exactly the demod figures (`SignalDbFS`, `EVMPct`, `SNRDb`) this post
kept off the completion event. It's the persistence half of the same two-event
design.

## FAQ

**What's the difference between `CallEnd` and `CallComplete` again?**
`CallEnd` fires the instant the engine tears the call down and carries the demod
figures (`SignalDbFS`, `EVMPct`, `SNRDb`) — no file yet. `CallComplete` fires
later, once the recorder has closed the WAV, and carries the on-disk `AudioPath`
and `SampleRate`. Consumers needing the file wait for complete; consumers needing
only metadata act on end.

**Which quality numbers are on `CallComplete`?**
None of the demod figures. `CallComplete` is file-centric: grant identity,
talkgroup, timestamps, reason, `AudioPath`, and `SampleRate`. The demod quality
rides `CallEnd`, and the audio-layer `VoiceStats` is written to the log rather
than carried on any event.

**Why isn't `VoiceStats` attached to the completion event?**
Its consumer is a human triaging audio quality, not a subsystem making a routing
decision. `logVoiceStats` emits it at finalize — DEBUG normally, WARN when the
output clips past 0.5% — so operators can diagnose "robotic" or over-loud calls
without diffing the WAV, and no event has to carry diagnostic detail nothing reads.

**Does every finished call publish a `CallComplete`?**
No. `finalizeLocked` returns `nil` — and nothing is published — for a dead-key or
idle-only call (it deletes the empty outputs) and for a call that decoded no PCM.
So a `CallComplete` is a guarantee that a real, non-empty file exists at
`AudioPath`, and downstream never receives a phantom path.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Loudness at the Output Stage]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})
· Next →
[Part 10: The Call Log in SQLite]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})
