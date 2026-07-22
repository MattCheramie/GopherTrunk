---
title: "Recording, Composition & Streaming, Part 6: Segmentation, Naming & Raw Sidecars"
description: How a trunked call maps to the filesystem in GopherTrunk — the system/talkgroup directory scheme, timezone-aware timestamped filenames tagged with frequency, source and timeslot, per-transmission file rolls, and the .raw sidecar written for DMR, ProVoice and TETRA.
category: deep-dives
keywords: sdr recording filename scheme, trunk recorder directory layout, per transmission recording, dmr timeslot recording, raw vocoder sidecar, ambe frame capture, talkgroup directory, timezone aware timestamp, gophertrunk recorder, provoice tetra raw
tags: [recording, filesystem, dmr, sidecar, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 6
---

*Part 6 of **Recording, Composition & Streaming**, a 14-part deep dive into
GopherTrunk's output half.
[Part 5]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
was about one file's integrity; this post is about where that file lives and what
it's named. When our 3 p.m. dispatch on talkgroup 101 hits disk, its path encodes
the system, the talkgroup, the wall-clock instant, the RF frequency, the source
radio, and — on a slotted protocol — the timeslot. And for protocols whose voice
GopherTrunk can't render to a WAV, a `.raw` sidecar of on-air frames is written so
the call is never lost.*

> **TL;DR:** The recorder lays calls out Trunk-Recorder-style:
> `<OutDir>/<system>/<talkgroup>/<timestamp>_freq<Hz>_src<src>[_ts<slot>].wav`.
> The timestamp renders in the configured **display timezone**, the frequency and
> source tags make a shared voice channel legible, and the `_ts` tag keeps DMR's
> two concurrent slots from colliding. In **per-transmission** mode every over
> rolls to a fresh file. DMR voice, EDACS ProVoice, and TETRA voice always get a
> **`.raw` sidecar** — a flat concatenation of on-air vocoder frames — because
> their audio either has no in-process decoder or must survive for out-of-band
> tools regardless.

**Key takeaways**

- A call's directory is `<system>/<talkgroup>`, both **sanitized** for
  cross-OS-safe paths; the talkgroup folder is the alpha tag when known, else the
  decimal group ID.
- The basename is a **timezone-aware timestamp** plus `_freq`, `_src`, and
  optional `_ts` tags — enough to identify the physical channel and originator
  from the filename alone.
- **Per-transmission recording** rolls each over into its own file via a segment
  event; a DMR carrier running two slots produces two independent session streams.
- The **`.raw` sidecar** is a raw dump of vocoder frames with no surrounding
  metadata, written for DMR/ProVoice/TETRA so an external decoder can consume it.

## Cheat sheet

| Thing | What it does | Where in code |
|---|---|---|
| `directoryFor` | Builds `<OutDir>/<system>/<talkgroup>` | `internal/voice/recorder.go` |
| `basenameFor` | `<stamp>_freq<Hz>_src<src>[_ts<slot>]`, display-tz stamp | `internal/voice/recorder.go` |
| `sanitize` | Maps unsafe runes to `_` for cross-OS paths | `internal/voice/recorder.go` |
| `handleSegment` | Rolls the current over, parks a dormant session | `internal/voice/recorder.go` |
| `rawWanted` | Marks a session for a `.raw` sidecar | `internal/voice/recorder.go` |
| `dmrVoiceProtocol` / `tetraVoiceProtocol` | Force `.raw` even when `WriteRaw` is off | `internal/voice/recorder.go` |
| `writeRawFrame` | Appends verbatim frames to `.raw` | `internal/voice/recorder.go` |

## In this post

- **The directory scheme** — system and talkgroup, and how each is chosen.
- **The basename** — timestamp, timezone, and the `_freq`/`_src`/`_ts` tags.
- **Per-transmission rolls** — one over, one file, via `handleSegment`.
- **Raw sidecars** — who gets one, and why the format is deliberately bare.
- **DMR's two slots** — one carrier, two concurrent sessions, distinct files.

## The directory: system, then talkgroup

Every recording lands under a two-level tree beneath the recordings directory:
`<OutDir>/<system>/<talkgroup>`. `directoryFor` builds it from the call's grant:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) directoryFor(cs trunking.CallStart) string {
    system := sanitize(cs.Grant.System)
    if system == "" {
        system = "unknown-system"
    }
    tgDir := fmt.Sprintf("%d", cs.Grant.GroupID)
    if cs.Talkgroup != nil && cs.Talkgroup.AlphaTag != "" {
        tgDir = sanitize(cs.Talkgroup.AlphaTag)
    }
    return filepath.Join(r.outDir, system, tgDir)
}
```

Two choices are worth calling out. First, the talkgroup folder prefers the
human-readable **alpha tag** when the engine resolved one (`Talkgroup != nil` and
a non-empty `AlphaTag`), falling back to the raw decimal `GroupID` when it didn't.
So a configured system browses as `PD Dispatch/` and `Fire Tac 2/`, while an
unlabelled talkgroup on an unconfigured system still records cleanly as `1234/`.
Second, everything user-derived passes through `sanitize`, which maps anything
outside `[A-Za-z0-9._-]` to an underscore — a system named `Metro (North)` becomes
`Metro__North_`, safe on every filesystem GopherTrunk targets.

Crucially, `directoryFor` only *computes* the path. The directory is **not
created here** — it's made lazily in `openSessionFiles`, just before the first
file is opened, which is on the first write. A grant that's followed but never
yields audio (a dead-key, an aborted encrypted call, a tap parked off-frequency
while a hunt borrows the SDR) therefore leaves **no empty `<system>/<talkgroup>`
folder** behind. The tree only ever contains directories that hold a real
recording.

## The basename: a self-documenting filename

`basenameFor` builds the filename stem. It starts from a timestamp and appends
tags until the name uniquely and legibly identifies the call:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) basenameFor(cs trunking.CallStart) string {
    stamp := t.In(r.displayLoc).Format("20060102T150405Z0700")
    base := stamp
    if cs.Grant.FrequencyHz != 0 {
        base = fmt.Sprintf("%s_freq%d", base, cs.Grant.FrequencyHz)
    }
    base = fmt.Sprintf("%s_src%d", base, cs.Grant.SourceID)
    if cs.Grant.Timeslot != 0 {
        base = fmt.Sprintf("%s_ts%d", base, cs.Grant.Timeslot)
    }
    return base
}
```

Each piece earns its place:

- **The timestamp renders in the display timezone.** `displayLoc` comes from
  `display.timezone` (via `DisplayConfig.Location()`); it defaults to UTC when
  unset. Formatting in local wall-clock time means the filename matches the
  timestamps the rest of the UI and logs already show — no mental UTC conversion
  when you're scrubbing through a directory listing. The `Z0700` zone token keeps
  the stamp self-documenting and **filename-safe** (no colon): UTC formats as a
  literal `Z`, other zones as a numeric offset like `+1000`.
  `TestRecorderBasenameTimezone` locks this in.
- **`_freq<Hz>`** tags the RF voice-channel frequency. On a trunked system the
  voice frequencies are a shared pool — many talkgroups cycle through the same
  physical channels — so stamping the frequency on each file tells you which
  channel a recording came from. Omitted when the grant frequency is unknown (0).
  `TestRecorderFrequencyInFilename` covers it.
- **`_src<src>`** is the source radio ID (the originating subscriber unit),
  always present.
- **`_ts<slot>`** is appended only for slotted protocols (`Timeslot != 0`). This
  is the collision-breaker described below; `TestRecorderTimeslotInWavName`
  asserts it lands in the name.

The WAV path is then `filepath.Join(dir, base+".wav")`, and — for the protocols
below — the sidecar is `base+".raw"` in the same directory.

<figure class="lab-figure">
<svg viewBox="0 0 620 210" width="620" height="210" role="img" aria-label="A directory tree. The recordings root contains a system folder named Metro P25, which contains two talkgroup folders: PD Dispatch and Fire Tac 2. PD Dispatch holds one WAV file with a timestamp, frequency and source tag. Fire Tac 2 holds two files sharing a timestamp but distinguished by timeslot tags ts1 and ts2, each paired with a matching raw sidecar file.">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="10">recordings/</text>
  <line x1="26" y1="30" x2="26" y2="188" stroke="var(--fg-muted)"/>
  <rect x="40" y="34" width="120" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="100" y="50" text-anchor="middle" fill="var(--accent)" font-size="10">Metro_P25/</text>
  <line x1="52" y1="58" x2="52" y2="176" stroke="var(--fg-muted)"/>
  <rect x="66" y="66" width="120" height="22" rx="6" fill="none" stroke="currentColor"/>
  <text x="126" y="81" text-anchor="middle" fill="currentColor" font-size="10">PD_Dispatch/</text>
  <line x1="78" y1="88" x2="78" y2="104" stroke="var(--fg-muted)"/>
  <rect x="92" y="98" width="300" height="20" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="102" y="112" fill="var(--fg-muted)" font-size="8.5">20260722T150000-0500_freq851012500_src4021.wav</text>
  <rect x="66" y="126" width="120" height="22" rx="6" fill="none" stroke="currentColor"/>
  <text x="126" y="141" text-anchor="middle" fill="currentColor" font-size="10">Fire_Tac_2/</text>
  <line x1="78" y1="148" x2="78" y2="200" stroke="var(--fg-muted)"/>
  <rect x="92" y="152" width="330" height="18" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="100" y="165" fill="var(--accent)" font-size="8.5">…T151233-0500_freq851262500_src310_ts1.wav + .raw</text>
  <rect x="92" y="176" width="330" height="18" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="100" y="189" fill="var(--accent)" font-size="8.5">…T151233-0500_freq851262500_src512_ts2.wav + .raw</text>
  <text x="470" y="162" fill="var(--fg-muted)" font-size="9">same second,</text>
  <text x="470" y="176" fill="var(--fg-muted)" font-size="9">split by _ts</text>
</svg>
<figcaption>Two talkgroups under one system. Fire Tac 2 is a DMR carrier: two concurrent slots share a timestamp but are kept distinct by the <code>_ts1</code>/<code>_ts2</code> tag, each with its own <code>.raw</code> sidecar.</figcaption>
</figure>

## Per-transmission rolls

By default a call is one continuous recording. But in **transmission** grouping
mode the composer publishes a `KindCallSegment` at each end-of-over boundary, and
the recorder rolls to a new file. `handleSegment` finalizes the current file and
parks a dormant session that carries the call's identity forward:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) handleSegment(seg trunking.CallSegment) {
    s, ok := r.sessions[seg.DeviceSerial]
    if !ok || s.wav == nil {
        return // nothing open to roll
    }
    cc := r.finalizeLocked(s, seg.DeviceSerial, seg.At, trunking.EndReasonNormal)
    // Park a dormant session: same cs + callID, no open files yet.
    r.sessions[seg.DeviceSerial] = &recordingSession{cs: s.cs, callID: s.callID}
    // …publish CallComplete for the finished over…
}
```

The parked session has `wav == nil` and no paths. The next write on that serial
finds it dormant and opens a **fresh file under a new timestamp** — reusing the
same grant and talkgroup but with `basenameFor` stamping the roll's own start
time. This is why segmentation never leaves a trailing empty file: the next over's
file isn't created until audio actually arrives for it. Each finished over is a
complete, closed, `CallComplete`-published recording in its own right.
`TestRecorderSegmentRollsToNewFile` walks exactly this: one call, a segment event,
two distinct files.

## The raw sidecar

For some protocols the WAV isn't enough — or isn't possible. The `.raw` sidecar
is a **flat concatenation of on-air vocoder frames**, written verbatim with no
surrounding metadata, so an operator can bring their own decoder (external libmbe,
DVSI hardware, DSD-FME) and consume the file directly. `buildSession` decides
whether a call wants one:

```go
// internal/voice/recorder.go (shape)
if r.writeRaw || cs.Grant.ProVoice ||
    dmrVoiceProtocol(cs.Grant.Protocol) ||
    tetraVoiceProtocol(cs.Grant.Protocol) {
    s.rawPath = filepath.Join(dir, base+".raw")
    s.rawWanted = true
}
```

Four ways to earn a sidecar, and the reasons differ:

- **`recordings.write_raw` is on** (`r.writeRaw`) — the operator wants raw frames
  for every digital call, alongside the decoded WAV.
- **EDACS ProVoice** (`cs.Grant.ProVoice`) — the ProVoice vocoder is patent- and
  trade-secret-encumbered, so GopherTrunk ships **no** built-in decoder. The
  sidecar is the *only* capture of the call; without it, a ProVoice grant would
  produce nothing usable.
- **DMR voice** (`dmr-tier1/2/3`) — DMR *does* have an in-process AMBE+2 decoder,
  so it gets a playable WAV, but it *also* always gets a sidecar so the on-air
  AMBE frames remain available for out-of-band tools.
- **TETRA voice** — no in-process vocoder yet (TCH/S FEC + ACELP are follow-ups),
  so like ProVoice the `.raw` of full-slot traffic frames is the only capture.

`rawWanted` is recorded at session-build time, but — like the WAV — the file is
opened **lazily** in `openSessionFiles`, so a call that never emits a frame leaves
no 0-byte `.raw` behind. Once open, every `writeRawFrame` appends the frame bytes
directly to it:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) writeRawFrame(deviceSerial string, callID uint64, frame []byte, /*…*/) error {
    s := r.sessionForWrite(deviceSerial, callID)
    if s == nil {
        return nil
    }
    if s.raw != nil {
        if _, err := s.raw.Write(frame); err != nil {
            return err
        }
    }
    // …then fan to the live raw tap, and decode into the WAV if a vocoder exists…
}
```

The sidecar and the WAV are independent outputs of the same frame stream: the raw
write happens first and unconditionally (for any open session with a sidecar),
then the frame is decoded into PCM for the WAV if a vocoder was instantiated.
`TestRecorderRawFrameSidecar`, `TestRecorderProVoiceForcesRawSidecar`, and
`TestRecorderNonProVoiceSkipsRawWhenDisabled` pin the who-gets-one matrix.

<figure class="lab-figure">
<svg viewBox="0 0 640 200" width="640" height="200" role="img" aria-label="A single DMR carrier feeds two concurrent recording sessions, one per timeslot. Timeslot 1's frames flow to a session that writes both a ts1 WAV and a ts1 raw sidecar; timeslot 2's frames flow to a separate session that writes a ts2 WAV and a ts2 raw sidecar. The two sessions are keyed by distinct device serials and never share a file.">
  <rect x="16" y="80" width="110" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="71" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">DMR carrier</text>
  <text x="71" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TS1 + TS2 slots</text>
  <line x1="126" y1="94" x2="176" y2="58" stroke="currentColor"/><polygon points="172,54 182,56 176,64" fill="currentColor"/>
  <line x1="126" y1="106" x2="176" y2="146" stroke="currentColor"/><polygon points="176,140 182,150 170,149" fill="currentColor"/>
  <rect x="182" y="36" width="130" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="247" y="55" text-anchor="middle" fill="currentColor" font-size="10">session (serial A)</text>
  <text x="247" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="8">callID 1 · ts1</text>
  <rect x="182" y="126" width="130" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="247" y="145" text-anchor="middle" fill="currentColor" font-size="10">session (serial B)</text>
  <text x="247" y="159" text-anchor="middle" fill="var(--fg-muted)" font-size="8">callID 2 · ts2</text>
  <line x1="312" y1="50" x2="356" y2="42" stroke="currentColor"/><polygon points="356,38 366,42 355,46" fill="currentColor"/>
  <line x1="312" y1="66" x2="356" y2="76" stroke="currentColor"/><polygon points="355,72 366,76 354,80" fill="currentColor"/>
  <rect x="366" y="30" width="150" height="22" rx="5" fill="none" stroke="var(--accent)"/><text x="441" y="45" text-anchor="middle" fill="var(--accent)" font-size="9">…_ts1.wav</text>
  <rect x="366" y="60" width="150" height="22" rx="5" fill="none" stroke="var(--fg-muted)"/><text x="441" y="75" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…_ts1.raw</text>
  <line x1="312" y1="140" x2="356" y2="132" stroke="currentColor"/><polygon points="356,128 366,132 355,136" fill="currentColor"/>
  <line x1="312" y1="156" x2="356" y2="166" stroke="currentColor"/><polygon points="355,162 366,166 354,170" fill="currentColor"/>
  <rect x="366" y="120" width="150" height="22" rx="5" fill="none" stroke="var(--accent)"/><text x="441" y="135" text-anchor="middle" fill="var(--accent)" font-size="9">…_ts2.wav</text>
  <rect x="366" y="150" width="150" height="22" rx="5" fill="none" stroke="var(--fg-muted)"/><text x="441" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…_ts2.raw</text>
  <text x="558" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">distinct files</text>
  <text x="558" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per slot</text>
</svg>
<figcaption>One DMR carrier, two concurrent calls. Each slot is a separate session keyed by its own device serial and CallID, producing its own <code>_ts</code>-tagged WAV plus a matching <code>.raw</code> sidecar.</figcaption>
</figure>

## Why DMR's two slots don't collide

DMR Tier III runs **two concurrent calls on one physical carrier** — TS1 and TS2.
If those two calls happen to share a talkgroup (same directory) and start in the
same wall-clock second, their `stamp_freq_src` basenames would be identical and
one would clobber the other. The `_ts<slot>` tag exists precisely to break that
tie: each slot's file is distinct and self-labelling. And because each slot is
followed by its own voice tap with its own device serial, the recorder already
holds them as two independent sessions — the naming tag is the on-disk half of a
separation the session map enforces in memory. (That per-serial identity, and the
CallID that fences one slot's frames from the other's, is the subject of
[Part 7]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }}).)

## Where this goes next

[Part 7]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }})
turns to the subtle safety machinery that keeps all this honest: the CallID fence
that stops a reused voice-tap serial from bleeding one call's tail into the next,
mid-stream encryption abort-and-delete, grant backfill for late-resolving IDs, and
runtime record on/off that never drops an in-flight call.

## FAQ

**How is a recording's filename structured?**
`<OutDir>/<system>/<talkgroup>/<timestamp>_freq<Hz>_src<src>[_ts<slot>].wav`. The
timestamp is the call's start rendered in the configured display timezone (a
filename-safe `Z0700` stamp), `_freq` is the RF voice-channel frequency, `_src` is
the originating radio ID, and `_ts` is appended only for slotted protocols like
DMR. The talkgroup folder is the alpha tag when known, else the decimal group ID.

**What is the `.raw` sidecar and which calls get one?**
It's a flat, metadata-free concatenation of on-air vocoder frames written next to
the WAV, so you can decode it with external tools. It's written when
`recordings.write_raw` is on, and always for EDACS ProVoice, DMR voice, and TETRA
voice — protocols whose audio either has no in-process decoder (ProVoice, TETRA)
or must remain available as raw AMBE regardless (DMR).

**Does per-transmission recording create a separate file per over?**
Yes. In transmission grouping the composer emits a `KindCallSegment` at each
end-of-over boundary; the recorder finalizes the current file, publishes its
`CallComplete`, and parks a dormant session that opens a fresh file — under a new
timestamp — when the next over's audio arrives. No trailing empty file is left if
the call ends first.

**Why do DMR TS1 and TS2 recordings not overwrite each other?**
The `_ts<slot>` tag makes each slot's filename unique even when both calls share a
talkgroup and start in the same second, and internally the two slots are separate
recording sessions keyed by distinct voice-tap serials. The naming tag is the
on-disk expression of a separation the recorder already maintains in memory.

## Series navigation

**Part 6 of 14** · ←
[Part 5: WAV on Disk]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
· Next →
[Part 7: Correctness Guards]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }})
