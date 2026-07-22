---
title: "Recording, Composition & Streaming, Part 3: Assembling a Call"
description: How GopherTrunk turns a burst of separate overs into one continuous call — hangtime, talkgroup gating and per-transmission splitting surfacing as segment events that roll the recorder's files, from the output side rather than the DSP.
category: deep-dives
keywords: call boundary hangtime, per transmission recording, callsegment event, voice call grouping conversation, transmission splitting scanner, file roll recorder, gophertrunk composition, talkgroup gating boundary
tags: [recording, composition, events, hangtime, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 3
---

*Part 3 of **Recording, Composition & Streaming** — the "composition" the series
is named for. [Part 2]({{ '/blog/deep-dives/recording-streaming-02-composition-contract/' | relative_url }})
carried audio across the seam one frame at a time. Now we ask what decides where a
call begins and ends. Our 3 p.m. dispatch on talkgroup 101 isn't one continuous
transmission — it's a dispatcher's over, a two-second gap, a unit's reply, another
gap. This post is about how those separate overs become one call (or a tidy run of
per-over files), and how a transmission boundary turns into a file roll on disk.*

> **TL;DR:** A voice chain feeds a shared **boundary tracker** two facts —
> "a voice frame decoded" and "a transmission ended." The tracker applies
> **hangtime** (end the call `VoiceHangtime` after the last frame) and
> **talkgroup gating** (drop, and eventually end on, a foreign talkgroup on a
> shared frequency). In per-transmission mode it publishes a **`KindCallSegment`**
> at each over boundary; the recorder finalizes the current file, emits its
> `CallComplete`, and parks a dormant session so the next over opens a fresh file.
> The `voice_call_grouping` config picks per-over or per-conversation files.

**Key takeaways**

- The unit of *composition* is the call; the unit of *transmission* is the over.
  `VoiceCallGrouping` decides whether one call is one file or one file per over.
- **Hangtime**, not a channel-release message, ends most calls. The tracker ends
  the call `VoiceHangtime` after the last matching voice frame.
- A **`CallSegment`** event is the composer telling the recorder to *roll the
  file* without ending the engine's call — a same-talkgroup re-key stays captured.
- On a segment the recorder **finalizes and parks a dormant session**: identity is
  carried forward, but no empty trailing file is created until real audio arrives.

## Cheat sheet

| Concept | What it does | Where in code |
|---|---|---|
| `boundaryTracker` | Per-call controller: hangtime, TG gating, split | `internal/voice/composer/boundary.go` |
| `onVoice(tg)` | Records a decoded frame; returns whether to write it | `internal/voice/composer/boundary.go` |
| `onTransmissionEnd()` | Emits a `KindCallSegment` in split mode | `internal/voice/composer/boundary.go` |
| `CallSegment` | Payload telling the recorder to roll the file | `internal/trunking/grant.go` |
| `handleSegment` | Recorder: finalize current file, park dormant session | `internal/voice/recorder.go` |
| `voice_call_grouping` | `"transmission"` (per over) vs `"conversation"` | `internal/config/config.go` |

## In this post

- **Overs vs calls** — the two units, and the config knob that maps between them.
- **The boundary tracker** — the three facts it turns into decisions (at the event level).
- **Hangtime** — why silence, not a message, ends most calls.
- **The segment roll** — how `CallSegment` becomes a new file without ending the call.
- **The dormant park** — how the recorder carries identity forward across a roll.

## Two clocks: the over and the call

A trunked conversation is a sequence of **overs** — each keying of a radio's PTT,
separated by short silences while people take turns. GopherTrunk has to decide how
those map onto files. There are two sensible answers and it supports both, via one
config field:

```go
// internal/config/config.go (shape)
// VoiceCallGrouping controls how voice recordings are split, for EVERY
// voice protocol. "transmission" (default) writes one file per over —
// the recording rolls at each end-of-transmission boundary.
// "conversation" keeps consecutive overs of the same talkgroup in one
// file, splitting only when a different talkgroup takes the (shared)
// frequency or the channel goes idle past VoiceHangtimeMs.
VoiceCallGrouping string `yaml:"voice_call_grouping"`
```

The daemon reduces this to a single boolean it hands the composer:

```go
// cmd/gophertrunk/daemon.go (shape)
SplitPerTransmission: cfg.Trunking.VoiceCallGrouping != "conversation",
```

In `"transmission"` mode (the default), every over becomes its own file with its
own timestamp — the Trunk-Recorder style, tidy for one-line-per-over browsing. In
`"conversation"` mode, consecutive overs of the same talkgroup accumulate into one
file, and the recording only splits when a *different* talkgroup seizes the shared
frequency or the channel goes quiet past hangtime. Same audio, different file
granularity. The choice is made once, at startup, and threaded through every voice
chain identically — the boundary logic is protocol-agnostic on purpose.

## The boundary tracker, from the outside

Every voice chain — FM, DMR, P25 Phase 1, P25 Phase 2 — shares one small
controller, the `boundaryTracker`. Its *internals* (the demod-quality accumulators,
the C4FM soft-symbol buffers, the SNR/EVM estimator) belong to
[Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
and we treat them as given. What matters here is its **event surface**: a chain
feeds it two facts and it turns them into three decisions.

```go
// internal/voice/composer/boundary.go (shape)
// onVoice records one decoded voice frame and returns whether its audio
// should be written. tg is the in-band talkgroup, or 0 when the frame
// carries none (P25 LDU2, FM) — then the previous match is inherited.
func (bt *boundaryTracker) onVoice(tg uint32) bool

// onTransmissionEnd marks an over boundary; rolls the file in split mode.
func (bt *boundaryTracker) onTransmissionEnd()

// run drives the hangtime timer + throttled Touch heartbeat until ctx ends.
func (bt *boundaryTracker) run(ctx context.Context)
```

The three decisions those produce are:

1. **Write or drop this frame.** `onVoice` returns a bool. When the grant carries
   a talkgroup and the frame's in-band talkgroup differs from it (and isn't a
   patched member), the frame is *foreign* — its audio is dropped, so a second
   talkgroup sharing the frequency can't append its speech to this call.
2. **End the call on a sustained foreign talkgroup.** A couple of consecutive
   frames of the *same* foreign talkgroup (`foreignRunToEnd`) end the call — the
   frequency has been taken over, and the engine will start the other talkgroup's
   call on its own tuner. A lone mis-decode is debounced away.
3. **Roll the file at an over boundary.** `onTransmissionEnd` publishes a segment
   (below) — but only in split mode, and only when audio was actually written
   since the last roll, so a run of terminators can't spawn empty files.

For our dispatch on talkgroup 101, `onVoice(101)` matches the grant and returns
true for every frame of the dispatcher's over. If a unit on talkgroup 250 briefly
keyed up on the same voice channel, those frames would return false and be dropped;
two in a row and the call ends cleanly.

## Hangtime: silence ends the call

The most important thing the tracker does is decide when a call is *over*. On many
systems there is no explicit channel-release message — P25 in particular rarely
sends one — so the call ends when the audio simply stops. That's **hangtime**:

```go
// internal/voice/composer/boundary.go (shape)
func (bt *boundaryTracker) run(ctx context.Context) {
    // …ticker…
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            if !bt.sawVoice.Load() {
                // Never decoded a matching frame: end on the no-voice
                // startup window so a phantom grant frees its tap.
                if time.Since(start) > bt.noVoiceTimeout {
                    bt.end(trunking.EndReasonTimeout)
                    return
                }
                continue
            }
            last := bt.lastVoiceNano.Load()
            // Keep the engine's LastHeardAt fresh, gated on progress.
            if bt.c.engine != nil && last != bt.lastTouchNano {
                bt.c.engine.Touch(bt.serial)
                bt.lastTouchNano = last
            }
            if time.Since(lastFrame) > bt.hangtime {
                bt.end(trunking.EndReasonNormal)
                return
            }
        }
    }
}
```

Two windows live here. Once voice *has* been decoding, the call ends
`VoiceHangtime` (default 3.5 s) after the **last matching frame** — tightly
bounding the recording to the actual transmission instead of waiting out the
engine's much longer call-timeout watchdog. If voice *never* decodes at all — a
stale control-channel grant, a mis-tuned tap — the call is torn down after a
short no-voice startup window with `EndReasonTimeout`, so a phantom grant frees
its voice SDR promptly rather than hanging idle. The distinction between the two
end reasons (`EndReasonNormal` vs `EndReasonTimeout`) is exactly the "radio
stopped transmitting" versus "we never decoded a thing" split the call log later
surfaces.

When the tracker decides to end, it calls `end(reason)` exactly once, which stamps
the measured signal and demod-quality figures onto the bound call and calls
`EngineHooks.EndCall`. The engine publishes `KindCallEnd`; the composer cancels
the chain. The recorder's response to that — finalize and publish `CallComplete`
— is Part 4's territory. What's new *here* is the third decision: rolling the file
mid-call, without ending it.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="A timeline of one call on talkgroup 101: a first over of decoded voice frames, a short silence gap shorter than hangtime, a second over, and a final silence. In per-transmission mode a CallSegment marker at the end of the first over rolls the recording to a new file. After the last over, once the silence exceeds the hangtime window, a CallEnd fires. The hangtime window is drawn as a bracket after the final over.">
  <line x1="20" y1="120" x2="640" y2="120" stroke="var(--fg-muted)"/>
  <text x="20" y="150" fill="var(--fg-muted)" font-size="9">t →</text>
  <rect x="40" y="96" width="150" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="115" y="112" text-anchor="middle" fill="var(--accent)" font-size="9">over 1 · tg 101</text>
  <text x="230" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gap &lt; hangtime</text>
  <rect x="290" y="96" width="150" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="365" y="112" text-anchor="middle" fill="var(--accent)" font-size="9">over 2 · tg 101</text>
  <line x1="190" y1="120" x2="190" y2="70" stroke="currentColor"/>
  <polygon points="186,74 190,66 194,74" fill="currentColor"/>
  <text x="190" y="60" text-anchor="middle" fill="currentColor" font-size="8">CallSegment</text>
  <text x="190" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="8">roll → new file</text>
  <text x="115" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="8">file A</text>
  <text x="365" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="8">file B</text>
  <line x1="440" y1="128" x2="560" y2="128" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <line x1="440" y1="124" x2="440" y2="132" stroke="var(--fg-muted)"/>
  <line x1="560" y1="124" x2="560" y2="132" stroke="var(--fg-muted)"/>
  <text x="500" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="8">VoiceHangtime</text>
  <line x1="560" y1="120" x2="560" y2="70" stroke="currentColor"/>
  <polygon points="556,74 560,66 564,74" fill="currentColor"/>
  <text x="560" y="60" text-anchor="middle" fill="currentColor" font-size="8">CallEnd</text>
  <text x="560" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reason: normal</text>
</svg>
<figcaption>Per-transmission mode: a segment marker rolls the recording to a new file at each over boundary, while the call itself ends only once silence outlasts the hangtime window.</figcaption>
</figure>

## The segment roll

When the split mode is on, an over boundary publishes a small event rather than
ending the call:

```go
// internal/voice/composer/boundary.go (shape)
func (bt *boundaryTracker) onTransmissionEnd() {
    if !bt.c.splitTx || !bt.voiceSinceRoll {
        return // conversation mode, or nothing written since last roll
    }
    bt.voiceSinceRoll = false
    bt.c.bus.Publish(events.Event{
        Kind: events.KindCallSegment,
        Payload: trunking.CallSegment{
            DeviceSerial: bt.serial,
            At:           time.Now(),
        },
    })
}
```

The payload is deliberately minimal — a device serial and the boundary instant:

```go
// internal/trunking/grant.go (shape)
// CallSegment is published at an end-of-transmission boundary when
// per-transmission recording is enabled, so the recorder closes the
// current file and starts a fresh one for the next over. At marks the
// boundary instant; the recorder uses it as the new segment's start
// timestamp.
type CallSegment struct {
    DeviceSerial string
    At           time.Time
}
```

The crucial property is what a segment is *not*: it is **not** a call end. The
engine's call keeps running — the voice SDR stays tuned, the `CallID` is unchanged
— so a same-talkgroup re-key that never triggers a fresh control-channel grant is
still captured on the same tuner. The segment only tells the recorder to close one
file and open the next. That's the difference the bus vocabulary from Part 1 buys:
`KindCallSegment` is a file-roll instruction, `KindCallEnd` is a call teardown, and
they travel independently.

## The dormant park

On the recorder side, a segment is handled by finalizing the current file and
parking a **dormant session**:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) handleSegment(seg trunking.CallSegment) {
    r.mu.Lock()
    s, ok := r.sessions[seg.DeviceSerial]
    if !ok || s.wav == nil {
        r.mu.Unlock()
        return // nothing open to roll (already dormant, or decode-only)
    }
    cc := r.finalizeLocked(s, seg.DeviceSerial, seg.At, trunking.EndReasonNormal)
    // Park a dormant session: keeps the call's identity so the next
    // write opens the next segment file, without creating an empty
    // trailing file if no further audio arrives.
    r.sessions[seg.DeviceSerial] = &recordingSession{cs: s.cs, callID: s.callID}
    r.mu.Unlock()
    if cc != nil {
        r.normalizeIfEnabled(cc.AudioPath)
        r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
    }
}
```

Three things happen, in order. First, `finalizeLocked` closes the current WAV
(and `.raw`), producing a `CallComplete` for the over that just finished — so each
over rolls out to the uploader as its own completed file. Second, the recorder
replaces the session in its map with a **dormant** one: a `recordingSession`
carrying only the originating `CallStart` and the `CallID`, with **no open files**.
Third, after releasing the lock, it publishes `CallComplete` for the finished
over.

The dormant park is what makes per-over splitting leave a clean recordings tree.
The parked session has no WAV path and no open handle; it exists only to carry the
call's identity forward. If the next over arrives, the first write reopens files
lazily under a fresh timestamp (the mechanism Part 4 details). If the call instead
ends during the silence — no further audio — the dormant session is simply closed
with nothing to finalize, and **no empty trailing file** is ever created. That is
why a chatty channel of short overs doesn't litter the disk with 44-byte WAVs
between transmissions.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="A state machine for one recording session across a segment roll: it starts active with an open WAV file, a CallSegment finalizes it and publishes CallComplete, moving to a dormant state with no open files, which carries only the call identity forward. From dormant, the next write reopens a fresh file back to active; alternatively a CallEnd during the silence closes the dormant session with nothing to finalize, reaching ended without leaving a trailing file.">
  <rect x="30" y="76" width="130" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="96" text-anchor="middle" fill="var(--accent)" font-size="10">active</text>
  <text x="95" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">WAV open, writing</text>
  <line x1="160" y1="98" x2="255" y2="98" stroke="currentColor"/><polygon points="255,94 265,98 255,102" fill="currentColor"/>
  <text x="212" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CallSegment</text>
  <text x="212" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">finalize + Complete</text>
  <rect x="265" y="76" width="140" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="335" y="96" text-anchor="middle" fill="currentColor" font-size="10">dormant</text>
  <text x="335" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no files, keeps id</text>
  <path d="M 335 76 C 300 30, 130 30, 95 74" fill="none" stroke="currentColor"/>
  <polygon points="99,68 93,76 104,74" fill="currentColor"/>
  <text x="215" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="8">next write → reopen fresh file</text>
  <line x1="405" y1="98" x2="500" y2="98" stroke="currentColor"/><polygon points="500,94 510,98 500,102" fill="currentColor"/>
  <text x="455" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CallEnd in silence</text>
  <text x="455" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">close, nothing to finalize</text>
  <rect x="510" y="76" width="120" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="570" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ended</text>
  <text x="570" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no trailing file</text>
</svg>
<figcaption>A session's life across a roll: a segment parks it dormant with its identity intact; the next write reopens a fresh file, while a call end during the silence closes it cleanly with no empty file left behind.</figcaption>
</figure>

## Where this goes next

[Part 4]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
opens up the `recordingSession` state machine that this post kept treating as a
black box: why files open **lazily** on the first write (so a dead-key leaves
nothing on disk), how the dormant park carries a call's identity across a segment
roll, and how an empty recordings directory turns the whole recorder into a
decode-only live-audio feed that still writes no files.

## FAQ

**What's the difference between a transmission and a call?**
A transmission (an "over") is one keying of a radio's PTT; a call is the whole
conversation. `voice_call_grouping` maps between them: `"transmission"` writes one
file per over, `"conversation"` keeps consecutive same-talkgroup overs in one file.
The boundary tracker detects overs; the config decides whether each one becomes a
separate file.

**Why does hangtime end the call instead of a channel-release message?**
Because many trunked systems — P25 especially — rarely send an explicit release.
The tracker ends the call `VoiceHangtime` (default 3.5 s) after the last decoded
voice frame, which bounds the recording tightly to the actual transmission rather
than waiting out the engine's much longer call-timeout watchdog.

**Does a `CallSegment` end the call?**
No. A segment tells the recorder to close the current file and open a fresh one,
but the engine's call keeps running — same tuner, same `CallID`. That's what lets a
same-talkgroup re-key that never re-grants stay captured. Only `CallEnd` tears the
call down.

**Why doesn't per-over splitting fill my disk with tiny files between overs?**
Because a segment parks a *dormant* session with no open files, and the next file
is opened lazily on the first write. If the call ends during the silence after an
over, the dormant session closes with nothing to finalize — so no empty trailing
WAV is ever created.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Composition Contract]({{ '/blog/deep-dives/recording-streaming-02-composition-contract/' | relative_url }})
· Next →
[Part 4: The Recording Session]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
