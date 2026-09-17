---
title: "The Field Notebook, Part 10: Recorder Lines — Spans, Segments & Dead Keys"
description: How to read GopherTrunk's recorder log — call started and call ended with its reason field, the audio_pct that catches a recording shorter than its call span, the dead-key drops that leave nothing on disk, the per-transmission segment rolls, and the retention sweeper lines that tidy up behind them.
category: tutorials
keywords: recorder log gophertrunk, call ended reason, audio_pct short recording, dead key no file, per transmission recording segments, retention sweeper log, recording shorter than call span, vocoder frame yield, gophertrunk field notebook
tags: [field-notebook, logs, recorder, recording, dmr, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 10
---

*Part 10 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field measures,
what a healthy rig prints, and which deep dive to open when a line goes wrong.
[Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
read the capture lines that record the air. This part reads the lines the
recorder prints while it turns a followed call into a file: `recorder: call
started`, `recorder: call ended reason=…`, the DEBUG line that catches a
recording shorter than its call span, the dead-key drops that leave nothing
behind, and the retention sweeper that tidies up.*

> **TL;DR:** The recorder opens a call with `INF recorder: call started` (device,
> `wav`, `tg`, `provoice`, `vocoder`) and closes it with `INF recorder: call
> ended reason=…`. The `reason` is an `EndReason`: `released`/`normal` are clean
> ends, `timeout` is silent-from-start, and there is no separate "call decoded
> nothing" line — that shows as `DBG recorder: recording shorter than call span`
> with `audio_pct`, the frame yield (a 1.7 s recording of a 5.6 s over reads
> `audio_pct=31`). A dead-key leaves **no file**: `openSessionFiles` runs on the
> first write, and a vocoded call with zero voiced+unvoiced frames is deleted and
> publishes no `CallComplete` (`WRN recorder: no decodable speech`). A
> per-transmission call rolls one file per over, each a `CallComplete` stamped
> with `CallStartedAt` + `Segment`; the `/calls/{id}/audio` endpoint concatenates
> them. `INF retention: deleted …` lines sweep rows, log tables and files.

**Key takeaways**

- **`reason=` tells you how a call ended, not how well it decoded.** `released`
  and `normal` are clean; `timeout` means not one frame arrived — a decode
  failure, not a short transmission.
- **`audio_pct` is the frame-yield alarm.** A recording much shorter than its
  wall-clock span logs `audio_pct` with a frame count; a low value is upstream
  loss, diagnosable from one DEBUG line.
- **A dead-key leaves nothing — by design.** Files open lazily on the first
  write, and a vocoded call that decodes no real speech is deleted rather than
  filed as tiny-recording spam.
- **Per-transmission calls are many files, one call.** Each over is its own
  `CallComplete` keyed by `CallStartedAt` + `Segment`, so a 30 s call that once
  played only its first 4 s now reassembles from `call_recordings`.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Call-started line | opens the session; names device, tg, vocoder | `internal/voice/recorder.go` (`handleStart`) |
| Call-ended line | `reason=`, duration, wav path | `recorder.go` (`finalizeCall`) |
| Short-recording DEBUG | `audio_pct`, `frames` — the frame-yield alarm | `recorder.go` (`finalizeLocked`) |
| Dead-key suppression | delete a no-speech call, publish no `CallComplete` | `recorder.go` (`removeSessionFiles`, `logVoiceStats`) |
| Segment roll | one `CallComplete` per over, `CallStartedAt`+`Segment` | `recorder.go` (`handleSegment`), [Recording Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }}) |
| Retention sweep | age out rows, log tables, files | `internal/storage/retention.go` (`SweepOnce`) |
| Lagging-tap WARN | starved voice tap → short/gappy recordings | `internal/scanner/ccdecoder/voicetap.go` (issue #402) |

## In this post

- **What the recorder lines are telling you** — started, ended, `reason`.
- **What healthy looks like** — a clean call from grant to `released`.
- **When it doesn't** — `audio_pct`, dead keys, and `timeout`.
- **Segments — one call, many files** — the per-transmission rolls.
- **Retention lines** — what the sweeper deleted and why.

## What the recorder lines are telling you

The recorder turns each followed call into a WAV (and, for digital voice, a
`.raw` sidecar), bracketed by two INF lines. On the grant:

```
INF recorder: call started device=cc:same-carrier:2 wav=../recordings/Harbour-TETRA/61432/…wav tg=61432 provoice=false vocoder=tetra-acelp
```

`device` is the voice tap's serial, `wav` the resolved path (the file is not
opened yet), `tg` the granted talkgroup, `provoice` flags an EDACS ProVoice
call, and `vocoder` names the decoder mapped from the protocol. When the call
ends:

```
INF recorder: call ended device=cc:same-carrier:2 wav=../recordings/Harbour-TETRA/61432/…wav duration=6.32s reason=released
```

The field to read first is `reason`. It is an `EndReason`
(`internal/trunking/grant.go`) with precise string values: `released` is an
explicitly decoded teardown (the control channel announced the end — a TETRA
D-RELEASE); `normal` is a hangtime or carrier-drop end (frames decoded, then the
transmitter stopped — the usual P25 end, which sends no release); `timeout` is
the real failure — a call that **never delivered a single frame** (wrong demod
mode, gain too low, a stale grant). The rest are policy ends: `preempted`,
`lockout`, `encrypted`, `no-voice-sdr`, `manual`, `error`. `duration` is the
wall-clock span — *not* the audio length, a distinction the next section turns on.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A recording session's log lifecycle. A call started line opens a session with a resolved wav path but no file; the first decoded voice frame opens the file lazily. At the end, a clean call logs call ended reason released. Two failure branches leave the diagram: a call with poor frame yield logs recording shorter than call span with a low audio_pct, and a dead key with zero decodable speech logs no decodable speech, is deleted, and publishes no call complete.">
  <rect x="14" y="96" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="74" y="114" text-anchor="middle" fill="var(--accent)" font-size="10">call started</text>
  <text x="74" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">wav path, no file</text>
  <line x1="134" y1="117" x2="176" y2="117" stroke="currentColor"/><polygon points="176,113 186,117 176,121" fill="currentColor"/>
  <rect x="186" y="96" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="246" y="114" text-anchor="middle" fill="currentColor" font-size="10">first frame</text>
  <text x="246" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">file opened lazily</text>
  <line x1="306" y1="117" x2="348" y2="117" stroke="currentColor"/><polygon points="348,113 358,117 348,121" fill="currentColor"/>
  <rect x="358" y="70" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="433" y="86" text-anchor="middle" fill="var(--accent)" font-size="10">call ended</text>
  <text x="433" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reason=released — clean</text>
  <rect x="358" y="120" width="150" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="433" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">shorter than call span</text>
  <text x="433" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">audio_pct=31 — low yield</text>
  <line x1="306" y1="112" x2="356" y2="92" stroke="currentColor"/>
  <line x1="306" y1="122" x2="356" y2="138" stroke="var(--fg-muted)"/>
  <rect x="140" y="184" width="300" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="290" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dead key: zero voiced+unvoiced frames</text>
  <text x="290" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="8">WRN no decodable speech → deleted, no CallComplete</text>
  <line x1="74" y1="138" x2="180" y2="186" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="560" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">duration is wall-clock,</text>
  <text x="560" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">not audio length</text>
</svg>
<figcaption>The recorder's lifecycle: a call opens with a path but no file, opens the file on the first decoded frame, and ends either cleanly (reason=released) or with a low-yield or dead-key diagnostic — the latter leaving nothing on disk.</figcaption>
</figure>

## What healthy looks like

A clean call is two lines, audio matching duration:

```
INF recorder: call started device=cc:same-carrier:1 wav=../recordings/Metro-P25/9001/…wav tg=9001 provoice=false vocoder=imbe
INF recorder: call ended device=cc:same-carrier:1 wav=../recordings/Metro-P25/9001/…wav duration=4.86s reason=released
```

Nothing else fires: no `recording shorter than call span` DEBUG line (the audio
ran close to the 4.86 s span), no `no decodable speech` WARN, no lagging-tap
WARN. The vocoded WAV lands at 8 kHz (the recorder forces a vocoded call's
header rate to the vocoder's native 8 kHz), the `.raw` sidecar holds the on-air
frames for DMR/ProVoice/TETRA, and the path encodes system, talkgroup,
timestamp, frequency and source exactly as
[Recording Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
lays out. `reason=released` is the cleanest possible teardown.

### How this line shapes operator practice

- **Read `reason` before the audio.** `released`/`normal` mean a clean end;
  `timeout` means the tap never decoded a frame — a tuning or gain problem, not a
  short call.
- **Do not read `duration` as audio length.** It is wall-clock; a multi-over
  call drops the silence between overs, so `duration` exceeds the audio and that
  is normal.
- **Silence on the recorder is often correct.** A dead-keyed grant leaves no
  file and no `CallComplete` — the missing `call ended` file is by design, not a
  lost recording.

## When it doesn't look like that

Three unhealthy shapes cover almost every recorder complaint.

**A recording shorter than its call span.** When a digital call's audio runs
well under its wall-clock span, the recorder logs a DEBUG diagnostic with the
frame yield:

```
DBG recorder: recording shorter than call span device=cc:same-carrier:2 wav=…wav vocoder=tetra-acelp audio_seconds=1.7 wall_seconds=5.6 frames=58 audio_pct=31
```

`audio_pct` is the fraction of the call span that decoded to audio — here 31%,
1.7 s of a 5.6 s over from 58 surviving frames. That fingerprint is upstream
frame loss, not a short transmission: on TETRA it is bursts failing the class-2
CRC on a marginal same-carrier signal, or bursts mis-routed by the usage-marker
demux; the recorder concatenates only the survivors, so the audio plays choppy.
A very low `audio_pct` with a small `frames` count sends you to the decode chain
([Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})),
not the recorder. Rule out one thing — a companion WARN,

```
WRN ccdecoder: same-carrier voice tap dropped IQ to a lagging voice consumer — the followed call's decode was starved (expect short/gappy recordings) … dropped_chunks=5904 (issue #402)
```

says the tap was starved of IQ — a CPU problem, not RF. Its absence blames the
signal.

**A dead key.** A grant that keys and unkeys without a word leaves nothing on
disk. Files open lazily — `openSessionFiles` runs on the first write — so a call
with no audio never creates a WAV or even a talkgroup folder. A vocoded call that
*does* write frames but decodes no real speech (every frame idle, silent or bad —
voiced + unvoiced == 0) is deleted afterward and publishes no `CallComplete`,
with a WARN carrying the evidence:

```
WRN recorder: no decodable speech — likely dead-key/idle carrier or mistuned tap device=… wav=…wav vocoder=imbe frames=42 voiced=0 unvoiced=0 min_b0=0 max_b0=6 first_frame_hex=…
```

The `min_b0`/`max_b0` range separates a genuine dead-key (b_0 varying in the idle
corner) from a mistuned tap feeding zero frames (b_0 pinned at 0). This stops
per-transmission mode spamming the tree with tiny files.

**`reason=timeout`.** When `call ended` reads `reason=timeout`, not one frame
ever decoded. The boundary tracker tears a silent call down after its no-voice
window (twice the hangtime) rather than holding the tap for the engine's longer
watchdog. On air it means the wrong demod mode, gain too low, or a stale grant —
never a short-but-real call, which ends `normal`.

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `reason` | how the call ended | `released` / `normal` | `timeout` (never decoded), or unexpected `encrypted`/`error` |
| `duration` | wall-clock span of the call | ≈ the transmission length | far exceeds audio *and* audio_pct is low |
| `audio_pct` | fraction of the span that decoded | absent (audio ≈ span) | present and low (e.g. 31) — upstream frame loss |
| `frames` | decoded/raw frames in the call | tens to hundreds for a real over | single digits with a low audio_pct |
| `voiced`/`unvoiced` | real speech frames (IMBE stats) | both non-zero | both zero → dead-key/idle, file deleted |
| `dropped_chunks` (tap WARN) | IQ starved from the voice tap | 0 (no WARN) | non-zero — CPU starvation, not RF |

## Segments — one call, many files

A per-transmission call is many files but one call. In `voice_call_grouping:
transmission` the recorder finalizes each over into its own file and parks a
dormant session, so a talker change produces two `call ended`/`CallComplete`
pairs under the same call. This is where a real bug lived: a 30-second call once
played back as a 4-second "recording" because each segment's `CallComplete` was
stamped with the over's own start time, matching no call row, so the call log
kept the first file and orphaned the rest. The fix (v1.1.3) makes each
`CallComplete` carry the **call's** start (`CallStartedAt`) and a `Segment`
index; every file lands in a `call_recordings` table, and `GET
/api/v1/calls/{id}/audio` concatenates the segments into one WAV with the real
inter-over silence restored (capped at 1.5 s). So a multi-over call is several
`call ended` lines sharing a `wav` directory with rising timestamps — the
healthy shape, not a fault. The mechanics are
[Recording Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
and the fences that keep one over's frames out of the next are
[Recording Part 7]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }}).

## Retention lines

Behind all of this the retention sweeper runs on its interval and logs what it
removed — three INF lines, each gated on a non-zero count:

```
INF retention: deleted call rows count=214
INF retention: deleted log rows table=pager_log count=1801
INF retention: deleted recordings count=96
```

The first ages out call-log rows past `retention.call_log_days`; the second
sweeps the decoder-log tables (pager, APRS, DSC, aircraft, MDC1200, FleetSync,
M17, location) past `retention.log_days`, one line per table; the third deletes
recording files older than `retention.files_days`. The file sweep only touches
recording artifacts — never a config or talkgroup CSV parked in the tree — and a
failed delete logs `WRN retention: rm failed` and keeps going. If the recordings
dir fills up with no `deleted recordings` line, the sweep is disabled
(`files_days: 0`) or the interval hasn't elapsed — the full policy is
[Recording Part 11]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }}).

| Symptom | Likely cause | Fix / read |
|---|---|---|
| WAV much shorter than the call; `audio_pct` low | upstream frame loss (marginal CRC, demux) | Improve signal/CPU; [Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}) |
| `WRN … no decodable speech`, no file | dead-key/idle carrier, or a mistuned tap (`b_0` pinned at 0) | Check the tap frequency; a real dead-key is expected and correct |
| `call ended reason=timeout` | tap never decoded a frame — wrong demod / gain / stale grant | Fix tuning or gain; `timeout` is a decode failure, not a short call |
| 30 s call plays only ~4 s | pre-v1.1.3 segment keying (fixed) | Play via `/calls/{id}/audio`, which concatenates segments — [Recording Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }}) |
| `dropped_chunks=N` in a tap WARN | voice tap starved of IQ (CPU) | Lower `sdr.sample_rate` or CPU load (issue #402) |
| Recordings dir keeps growing | file sweep disabled or not yet run | Set `retention.files_days`; check the interval — [Recording Part 11]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }}) |

## Where this goes next

The recorder's `audio_pct` and dead-key lines point *upstream* when a call
decodes poorly — to the composer's voice chain.
[Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})
reads those lines: `voice follow started`, `speech_frames` and `tch_frames`, the
`bfi_count` bad-frame counter, the DMO seed lines, and the `undecoded_drops` /
`concurrency_suppressed` counters that look alarming but work as designed.

## FAQ

**What does reason=timeout mean on a GopherTrunk call ended line?**
It means the call was reaped without ever decoding a single voice frame — the
silent-from-start failure. On air that is a wrong demod mode, gain too low, or a
stale control-channel grant, not a short transmission. A real but brief call ends
`reason=normal` or `released`; only a never-decoded call ends `timeout`.

**Why is my recording shorter than the call's duration?**
Because `duration` is wall-clock and the WAV is only the frames that decoded.
The `recording shorter than call span` DEBUG line prints `audio_pct` (the frame
yield) and `frames`; a low value is upstream loss — marginal CRC, a starved tap,
or demux mis-routing — diagnosable from that one line, not the recorder itself.

**Why did a call leave no recording at all?**
A dead-key or idle carrier decodes no real speech, so the recorder deletes the
file and publishes no `CallComplete` (with a `no decodable speech` WARN carrying
the `b_0` range). Files also open lazily on the first write, so a followed grant
that yields nothing leaves neither a WAV nor an empty talkgroup folder.

**Why does one call produce several recording files?**
In per-transmission grouping the recorder rolls a fresh file at each over
boundary, so each over is its own `CallComplete` stamped with the call's start
(`CallStartedAt`) and a segment index. They reassemble via `GET
/api/v1/calls/{id}/audio`, which concatenates the segments with the real
inter-over silence restored.

**What do the retention lines mean?**
`retention: deleted call rows`, `deleted log rows` (one per decoder-log table)
and `deleted recordings` report each sweep's counts, gated on non-zero, aging
out data past `call_log_days`, `log_days` and `files_days`. A failed file delete
logs `retention: rm failed` and the sweep continues rather than aborting.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Capture Lines — Aligning a Capture to the Log]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
· Next →
[Part 11: Voice-Chain Lines — Frames, Colours & Teardown Reasons]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})
