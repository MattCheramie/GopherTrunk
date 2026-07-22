---
title: "Recording, Composition & Streaming, Part 7: Correctness Guards — Cross-Call Fencing, Encryption Skip & Runtime Toggles"
description: The subtle safety machinery in GopherTrunk's recorder and audio publisher — a CallID fence that stops a reused voice-tap serial bleeding one call's tail into the next, mid-stream encryption abort-and-delete, grant backfill for late-resolving IDs, and runtime record on/off that never drops an in-flight call.
category: deep-dives
keywords: cross call audio bleed, callid fence, voice tap serial reuse, mid call encryption abort, encrypted call skip, grant backfill, runtime recording toggle, talkgroup record gate, gophertrunk recorder guards, live audio fencing
tags: [recording, correctness, encryption, concurrency, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 7
---

*Part 7 of **Recording, Composition & Streaming**, a 14-part deep dive into
GopherTrunk's output half. Parts 4–6 built the recording session, the crash-safe
WAV, and the on-disk naming. This post is about the guards that keep them correct
under adversarial timing: a voice tap whose serial gets reused before the old
call drained, a call that turns out to be encrypted halfway through, IDs that
resolve late, and an operator flipping recording off mid-conversation. None of
these is visible in a happy-path recording of our 3 p.m. dispatch — they're the
machinery that makes sure the file you get is *that* call's audio and nothing
else's.*

> **TL;DR:** Wideband voice taps reuse a small pool of device serials, so a frame
> from a just-ended call can arrive after the next call has claimed the same
> serial. GopherTrunk fences this with the **CallID**: `sessionForWrite` rejects a
> frame whose `callID` doesn't match the open session, and the live
> `AudioPublisher` applies the **same fence** on `WritePCMForCall`/`WriteRawFrameForCall`.
> When `skip_encrypted` is set and a call is discovered encrypted mid-stream, the
> recorder **closes and deletes** the partial and publishes no `CallComplete`.
> Late-resolving RID/ALG/KID are **backfilled** onto the session's grant so the
> eventual `CallComplete` is accurate, and recording can be toggled at runtime
> **without truncating** in-flight calls.

**Key takeaways**

- The **CallID fence** is enforced in two independent places — the recorder's
  `sessionForWrite` (the WAV/`.raw` side) and the audio publisher's `writePCM`/
  `writeRaw` (the live side) — because a reused serial can bleed audio into
  *either* sink.
- **Encryption is handled at two moments**: a grant that already signals
  encryption never opens a file; encryption discovered mid-call closes and
  `os.Remove`s the partial and suppresses `CallComplete`.
- **Backfill** mirrors late-resolved source ID and encryption facts onto the
  session's stored grant, so a Phase 2 compressed grant that resolves on the
  traffic channel doesn't broadcast as `encrypted=false`.
- **Runtime toggles are graceful**: disabling recording drops *new* `CallStart`s
  but lets in-flight sessions finish; the per-talkgroup `Record` flag gates files
  without silencing live audio.

## Cheat sheet

| Guard | What it stops | Where in code |
|---|---|---|
| CallID fence (recorder) | Stale frame → wrong WAV/`.raw` | `sessionForWrite` (`internal/voice/recorder.go`) |
| CallID fence (live) | Stale frame → wrong live stream | `writePCM`/`writeRaw` (`internal/api/audio_publisher.go`) |
| Mid-call encryption abort | Encrypted partial reaching uploads | `handleEncryptionUpdate` (`internal/voice/recorder.go`) |
| Grant backfill | Inaccurate `CallComplete` metadata | `backfillSessionGrant` (`internal/voice/recorder.go`) |
| Runtime record gate | Losing the head of a mid-call disable | `SetRecordingEnabled` / `recordDisabled` |
| Talkgroup `Record` gate | Writing files for no-record talkgroups | `handleStart` (`internal/voice/recorder.go`) |

## In this post

- **The reused-serial problem** — why a valid frame can belong to the wrong call.
- **The CallID fence**, enforced twice: on disk and on the live stream.
- **Encryption**, at grant time and mid-stream — abort, delete, suppress.
- **Backfill** — making the finished-call metadata match what the engine learned late.
- **Runtime toggles** — record on/off and per-talkgroup gating without collateral.

## The reused-serial problem

A recording session is keyed by **device serial** — which voice SDR (or wideband
voice tap) is following the call. That key is stable and cheap, but it has a sharp
edge: wideband taps draw from a **fixed pool** of serials. When one call ends and
the next grant reuses the same tap, the recorder's `sessions[serial]` is
re-pointed at the new call — while the *previous* call's decode chain may still be
draining a few final frames. Those frames are valid audio; they just belong to a
call that's over. Delivered naively, they'd be appended to the new call's WAV, and
fanned to live subscribers **filtered on the new call's talkgroup** — one call's
tail bleeding into another's stream, mislabelled.

The engine already carries the thing needed to tell them apart: `Grant.CallID`, a
process-monotonic identifier the voice pool assigns per call. The guard is to
stamp each session with its `CallID` and reject any frame that arrives claiming a
different one.

## The CallID fence, on disk

Every write path funnels through `sessionForWrite`, which does the matching under
the recorder's lock:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) sessionForWrite(serial string, callID uint64) *recordingSession {
    r.mu.Lock()
    defer r.mu.Unlock()
    s, ok := r.sessions[serial]
    if !ok {
        return nil
    }
    if callID != 0 && s.callID != 0 && callID != s.callID {
        return nil // stale frame from the call that previously held this serial
    }
    // …lazily open files / return the live session…
}
```

The rule is deliberately permissive at the edges: a **zero on either side
matches**. A zero incoming `callID` (an analog or synthetic call that doesn't
stamp one) or a zero session `callID` preserves the old behaviour for callers that
never opted in. Only when *both* are non-zero and they differ is the frame
rejected — and it's rejected **without reopening a dormant session**, so a
mismatched frame can't even spawn an empty file. Digital voice chains that know
their call's identity call `WriteRawFrameForCall(serial, callID, …)` in preference
to the plain form; the fence is invisible to everyone else.
`TestRecorderCallIDFenceDropsStaleFrames` reproduces the reuse and asserts the
stale frame lands nowhere.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="A timeline of one voice-tap serial reused across two calls. Call A holds the serial and is stamped CallID 1; near its end the tap is reassigned to Call B, stamped CallID 2, so the session map now points at Call B. A few of Call A's final decoded frames, still stamped CallID 1, arrive after the flip; the fence compares their CallID 1 against the session's CallID 2, they mismatch, and the frames are dropped instead of being written to Call B's file or stream.">
  <line x1="30" y1="60" x2="630" y2="60" stroke="currentColor"/>
  <text x="30" y="48" fill="var(--fg-muted)" font-size="9">tap serial "voice-0"</text>
  <line x1="150" y1="52" x2="150" y2="150" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="150" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">serial reassigned</text>
  <rect x="40" y="66" width="200" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="140" y="83" text-anchor="middle" fill="currentColor" font-size="10">Call A · CallID 1</text>
  <rect x="150" y="100" width="300" height="26" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="300" y="117" text-anchor="middle" fill="var(--accent)" font-size="10">Call B · CallID 2 (session now)</text>
  <circle cx="200" cy="60" r="4" fill="currentColor"/>
  <circle cx="230" cy="60" r="4" fill="currentColor"/>
  <text x="215" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="8">A's draining frames</text>
  <text x="215" y="44" text-anchor="middle" fill="var(--fg-muted)" font-size="8">(still CallID 1)</text>
  <line x1="200" y1="64" x2="230" y2="98" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <line x1="230" y1="64" x2="255" y2="98" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="470" y="96" width="170" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="555" y="112" text-anchor="middle" fill="currentColor" font-size="9">fence: 1 ≠ 2 → drop</text>
  <text x="555" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no WAV, no live stream</text>
  <line x1="450" y1="113" x2="470" y2="113" stroke="currentColor"/><polygon points="470,109 480,113 470,117" fill="currentColor"/>
  <text x="335" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="9">A zero CallID on either side matches — the fence only fires when both are known and differ.</text>
</svg>
<figcaption>The reused serial points the session at Call B, but Call A's late frames still carry CallID 1. The fence compares against the session's CallID 2 and drops them.</figcaption>
</figure>

## The same fence, on the live stream

The WAV isn't the only sink a stale frame can poison. The live
[audio publisher]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})
fans decoded PCM (and, for opted-in subscribers, raw frames) to gRPC/HTTP
listeners — and it filters by talkgroup. A stale frame fanned out here would be
labelled with, and leaked to subscribers filtered on, the *new* call's talkgroup.
So the publisher enforces a **matching fence**, keyed the same way, against the
grant map its own bus subscription maintains:

```go
// internal/api/audio_publisher.go (shape)
func (p *AudioPublisher) writePCM(deviceSerial string, callID uint64, samples []int16) error {
    p.mu.RLock()
    defer p.mu.RUnlock()
    grant, haveGrant := p.grants[deviceSerial]
    // The serial's live grant has moved to a different call — drop the old tail.
    if haveGrant && callID != 0 && grant.CallID != 0 && callID != grant.CallID {
        return nil
    }
    // …fan to matching subscribers…
}
```

`writeRaw` carries the identical check. The recorder hands its session's CallID to
whichever sink implements the call-aware interface (`DecodedPCMCallSink` /
`RawFrameCallSink`); a sink that doesn't still gets the plain `WritePCM`. The two
fences are **independent implementations of one invariant** because the two sinks
learn "the call has moved on" through different channels — the recorder from its
`sessions` map, the publisher from its bus-fed `grants` map — and either could
still hold the old identity for a beat.
`TestAudioPublisher_WritePCMForCallFencesStaleCall` and its raw twin drive exactly
the reuse the recorder test does, one layer out.

## Encryption, at two moments

GopherTrunk can be told (`recordings.skip_encrypted`) to refuse encrypted calls.
Encryption is known at two very different times, so the guard has two halves.

**At grant time**, if the control-channel grant already flags encryption,
`handleStart` never opens a session — the file simply never exists:

```go
// internal/voice/recorder.go (shape)
if r.skipEncrypted && cs.Grant.Encrypted {
    r.log.Debug("recorder: skipping encrypted call", /*…*/)
    return
}
```

**Mid-stream** is the harder case. P25 Phase 1 carries its Encryption Sync in
LDU2, and a Phase 2 compressed grant only resolves encryption on the traffic
channel — so a call can be *recording clear-looking audio* before it announces
it's encrypted. When that arrives (a `KindCallEncryption` or `KindCallSourceUpdate`
event whose flag resolves to encrypted), `handleEncryptionUpdate` aborts: it drops
the session, closes the open files, and **deletes them from disk**, publishing no
`CallComplete` so no partial ever reaches the upload feeds:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) handleEncryptionUpdate(deviceSerial string, encrypted bool) {
    if !r.skipEncrypted || !encrypted {
        return
    }
    s, ok := r.sessions[deviceSerial]
    if !ok {
        return
    }
    delete(r.sessions, deviceSerial)
    if s.wav == nil {
        return // dormant post-segment park — no open files
    }
    _ = s.close()
    for _, p := range []string{s.wavPath, s.rawPath} {
        if p != "" {
            _ = os.Remove(p) // tolerate os.IsNotExist
        }
    }
    // …no CallComplete published…
}
```

Suppressing `CallComplete` is the load-bearing part: as
[Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})
established, the broadcast manager only acts on `CallComplete`. No event, no
upload — the encrypted partial can't leak even though it briefly existed on disk.
`TestRecorderAbortsOnEncryptionSync` and
`TestRecorderAbortsMidCallEncryptedSourceUpdate` cover both trigger events;
`TestRecorderRecordsEncryptedWhenSkipDisabled` confirms it's opt-in.

<figure class="lab-figure">
<svg viewBox="0 0 620 190" width="620" height="190" role="img" aria-label="A decision flow for a call discovered encrypted mid-stream. An in-progress recording receives an encryption update; if skip_encrypted is off or the call is clear, recording continues normally. Otherwise the session is dropped from the map, its open WAV and raw files are closed and removed with os.Remove, and no CallComplete is published, so downstream upload feeds never see the partial.">
  <rect x="20" y="80" width="120" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="80" y="98" text-anchor="middle" fill="currentColor" font-size="10">recording…</text>
  <text x="80" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">encryption update</text>
  <line x1="140" y1="100" x2="180" y2="100" stroke="currentColor"/><polygon points="180,96 190,100 180,104" fill="currentColor"/>
  <polygon points="190,100 250,74 310,100 250,126" fill="none" stroke="currentColor"/>
  <text x="250" y="96" text-anchor="middle" fill="currentColor" font-size="9">skip &amp;&amp;</text>
  <text x="250" y="108" text-anchor="middle" fill="currentColor" font-size="9">encrypted?</text>
  <line x1="250" y1="126" x2="250" y2="160" stroke="currentColor"/><polygon points="246,160 250,170 254,160" fill="currentColor"/>
  <text x="285" y="150" fill="var(--fg-muted)" font-size="9">no</text>
  <rect x="160" y="168" width="180" height="20" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="250" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">keep recording (unchanged)</text>
  <line x1="310" y1="100" x2="352" y2="100" stroke="var(--accent)"/><polygon points="352,96 362,100 352,104" fill="var(--accent)"/>
  <text x="332" y="92" fill="var(--accent)" font-size="9">yes</text>
  <rect x="362" y="42" width="240" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="482" y="58" text-anchor="middle" fill="var(--accent)" font-size="9">delete session from map</text>
  <rect x="362" y="72" width="240" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="482" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">close + os.Remove(wav, raw)</text>
  <rect x="362" y="102" width="240" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="482" y="118" text-anchor="middle" fill="var(--accent)" font-size="9">suppress CallComplete</text>
  <line x1="482" y1="66" x2="482" y2="72" stroke="var(--accent)"/>
  <line x1="482" y1="96" x2="482" y2="102" stroke="var(--accent)"/>
  <text x="482" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ upload feeds never see the partial</text>
</svg>
<figcaption>Mid-call encryption: when opted in, the session is dropped, its files are closed and removed, and no <code>CallComplete</code> is published — so the broadcast manager, which acts only on that event, never touches the partial.</figcaption>
</figure>

## Backfill: making the metadata match

Deleting the partial is one response to late-resolved encryption. But when the
operator *isn't* skipping encrypted calls — or when the late fact is a source
radio ID rather than encryption — the call keeps recording, and the concern flips:
the eventual `CallComplete` must carry the truth. That `CallComplete` is built
from the session's *stored* grant (a snapshot taken at `CallStart`), so
`backfillSessionGrant` mirrors the engine's late discoveries onto it:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) backfillSessionGrant(serial string, encrypted bool, algID uint8, keyID uint16, sourceID uint32) {
    s, ok := r.sessions[serial]
    if !ok {
        return
    }
    if encrypted {
        s.cs.Grant.Encrypted = true
        if algID != 0 { s.cs.Grant.AlgorithmID = algID }
        if keyID != 0 { s.cs.Grant.KeyID = keyID }
    }
    if sourceID != 0 {
        s.cs.Grant.SourceID = sourceID
    }
}
```

The update rules are conservative on purpose. Encryption is **sticky-true** — set,
never cleared here — so a later clear-looking update can't un-mark a call the
engine already saw encrypted. A **zero source is ignored**, so a subsequent
clear-source update can't erase a RID that was already resolved. Without this, a
Phase 2 compressed grant whose encryption resolves only on the traffic channel
would broadcast as `encrypted=false` even though the call log — fed from the
engine-backfilled `CallEnd` grant — correctly shows it encrypted (the exact split
of issue #897). `TestRecorderBackfillsMidCallEncryptionIntoCallComplete` pins the
two records back into agreement.

## Runtime toggles without collateral

The last guards are about operator actions mid-stream not damaging calls in
flight.

**Runtime record on/off.** `SetRecordingEnabled(false)` flips an atomic
`recordDisabled` that `handleStart` checks: subsequent `CallStart`s are dropped
silently, laying down no files. But **in-flight sessions are left alone** — they
finish naturally on `CallEnd` — so flipping the switch mid-conversation doesn't
truncate the call that's already recording. The gate stops *new* work; it never
severs *existing* work. `TestRecorderGateBlocksNewSessions` covers the block;
the atomic makes the read lock-free on the hot path.

**Per-talkgroup gating.** Independently, a talkgroup flagged `Record = false` is
followed and played live but written to no file. The check in `handleStart` is
careful about a subtlety: it must *not* drop the call outright, or the talkgroup
would go silent live too. Instead it skips only file creation, and only when the
recorder is actually persisting (`!r.decodeOnly`):

```go
// internal/voice/recorder.go (shape)
if !r.decodeOnly && cs.Talkgroup != nil && !cs.Talkgroup.Record {
    return // follow + play live, but write no files
}
```

A decode-only recorder writes nothing regardless, so it deliberately falls through
here — dropping the call would silence a talkgroup that's supposed to be audible.
`TestRecorderSkipsRecordFalseTalkgroup` locks the file-skip; the live path stays
open. Two toggles, one principle: change what's recorded going forward, never
damage what's already in flight or what's meant to stay audible.

## Where this goes next

[Part 8]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})
moves from correctness to polish: where and when the optional loudness
normalization runs — after the WAV is finalized and *before* `CallComplete` is
published, so every downstream consumer reads leveled audio — and why that seam,
not the DSP, is where GopherTrunk places it.

## FAQ

**What is the CallID fence and why is it needed?**
Wideband voice taps reuse a small pool of device serials, so a frame from a
just-ended call can arrive after the next call has claimed the same serial.
Each session and grant is stamped with the call's monotonic `CallID`; a frame
whose CallID doesn't match the current session (both non-zero and different) is
dropped, so one call's draining tail can't be written to — or streamed as — the
next call's audio.

**Why is the fence implemented in two places?**
Because a stale frame can poison two independent sinks: the recorder's WAV/`.raw`
files and the live audio publisher's stream. The recorder learns a call has ended
through its `sessions` map; the publisher learns through its own bus-fed `grants`
map. Each enforces the same CallID check against its own view, so neither sink
mislabels a reused-serial frame.

**What happens if a call is found to be encrypted after recording starts?**
With `skip_encrypted` on, the recorder closes and `os.Remove`s the in-progress WAV
and `.raw`, drops the session, and publishes no `CallComplete`. Because the
broadcast manager acts only on `CallComplete`, the encrypted partial that briefly
existed on disk never reaches any upload feed.

**Can I turn recording off without losing the call that's currently recording?**
Yes. The runtime record toggle only blocks *new* `CallStart` events; sessions
already open finish normally on `CallEnd`. Likewise a talkgroup marked
`Record = false` is still followed and played live — only its file writing is
skipped — so gating never silences live audio.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Segmentation, Naming & Raw Sidecars]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
· Next →
[Part 8: The Loudness Output Stage]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})
