---
title: "From the Issue Tracker, Season 2, Part 10: The Re-Key That Dropped Both Overs — Drain Coordination Fenced by Call ID"
description: "How a conventional-DMR re-key within hangtime made GopherTrunk's recorder lose two recordings at once — the previous call's deferred finalize met the next call's start on the same serial, the replace path closed one session unfinalised and a call-blind drain signal finalised the other before its first frame — and the Grant.CallID fence that now keeps both overs."
category: solution-postmortem
keywords: dmr re-key recording lost, recorder drain coordination, device already has session replacing, finalizeDrainingBeforeReuse, NotifyDrainCompleteForCall, Grant.CallID fence, call complete missing history row, ipsc re-key within hangtime, TestRecorderRekeyDuringPendingDrainKeepsNextOver, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, dmr, recorder, composer, concurrency, go, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 10
---

*Part 10 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems that continues the
[first season]({{ '/blog/series/from-the-issue-tracker/' | relative_url }}):
one bug per part, with receipts.
[Part 9]({{ '/blog/solution-postmortem/issue-tracker-s2-09-sixteen-thousand-codewords/' | relative_url }})
made the wideband pump cheap again. This part is a recording that went
missing with the pump idle and the decoder correct: an IPSC radio keyed up
again inside hangtime, the engine granted the new over exactly as designed
— and the recorder lost both the over that was ending and the over that
was starting.*

> **TL;DR:** The 17 Sep "Fire2" material (600 s of 442.3875 MHz IPSC at
> 25 kS/s as FLAC, plus `debug.log`; idle-beacon repeater, cc 12, tg 11)
> held one re-key in five transmissions and **two recordings that never
> logged an end**, beside a lone `recorder: device already has session,
> replacing` WARN. The composer runs the recorder *drain-coordinated*:
> `handleEnd` defers a call's finalize until the voice chain signals its
> tail frames are written. A Voice-LC-Header re-key ends the previous call
> and grants the next in the same instant, so the previous `CallEnd` was
> still parked in `pendingFinalize` when the next `CallStart` arrived on
> the same serial. `handleStart`'s replace path closed the previous session
> **without finalising it** — no `CallComplete`, no sidecar, no history row
> — and the previous call's late, serial-keyed drain signal then finalised
> the *new* session before its first frame (files open lazily, so
> silently); every frame of the next over was dropped. Fix:
> `finalizeDrainingBeforeReuse` finalises the deferred call before the
> reuse, and every finalize/drain path is fenced by `Grant.CallID`
> (`callIDsDiffer`, `NotifyDrainCompleteForCall` threaded composer →
> `fanoutSink` → recorder, `sessionForWrite`). Pinned failing-first by
> `TestRecorderRekeyDuringPendingDrainKeepsNextOver`: old recorder, one
> `.raw` and one `CallComplete`; fixed, `[5 4]` frames and two.

**Key takeaways**

- **A deferred completion needs an identity, not just a key.** A drain
  signal keyed by device serial is ambiguous the moment the serial is
  reused; `Grant.CallID` is the identity the engine already stamps.
- **"Replace" is not "finalize".** Closing a session releases its files;
  finalising it publishes the call. The replace path did the first and
  skipped the second.
- **Lazy opens make the wrong finalize silent.** A session finalised before
  its first write has nothing to close, so the loss produced no error — only
  a missing row.
- **The control path's fix exposed the recording path's bug.** Granting the
  re-key was correct (Part 6 of DMR End to End); recording it was the next
  seam.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Drain coordination | `handleEnd` parks the `CallEnd` in `pendingFinalize` until the chain's drain signal; 3 s backstop | `internal/voice/recorder.go` (`EnableDrainCoordination`, `drainFinalizeTimeout`) |
| The reuse path | finalise a deferred previous call before a new session takes its serial | `finalizeDrainingBeforeReuse` |
| Call identity | `a != 0 && b != 0 && a != b`; zero matches anything (legacy / synthetic calls) | `callIDsDiffer` |
| Call-aware drain | ignores a drain for a call the serial no longer records; resets a stale `drained` flag | `NotifyDrainCompleteForCall`, `handleEnd` |
| Frame fence | a frame carrying another call's ID never lands in the open session | `sessionForWrite`, `WriteRawFrameForCall`, `WritePCMForCall` |
| Who stamps the ID | `VoicePool.Bind` puts a fresh `CallID` on the grant the `CallStart` carries | `internal/trunking/voicepool.go`, published from `engine.go` |
| Plumbing | composer's `handleEnd` → `fanoutSink.NotifyDrainCompleteForCall` → recorder | `composer/composer.go`, `cmd/gophertrunk/daemon.go` |
| Failing-first pin | re-key during a pending drain records both overs | `recorder_drain_test.go` (`TestRecorderRekeyDuringPendingDrainKeepsNextOver`) |

## In this post

- **The log with a hole in it** — one WARN, two missing ends.
- **Why the recorder waits** — drain coordination and the tail-frame race it closed.
- **The instant a re-key creates** — `CallEnd` and `CallStart` with no gap.
- **Two wrong moves on one serial** — replace without finalise; finalise the wrong session.
- **The fence** — `Grant.CallID` on every finalize and drain path.
- **The failing-first test, and what is still open** — `[5 4]` frames, two completes.

## The log with a hole in it

The 17 Sep Fire2 material came from the same IPSC tap as the deaf-heal
work ([DMR End to End Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }})):
600 s of 442.3875 MHz at 25 kS/s as FLAC, with `debug.log` for the same
window. The repeater is the idle-beacon kind — colour code 12, talkgroup
11, ~10 s beacon trains with 5–9 s gaps — and the commit that came out of
the capture fixed four independent defects. This part is the second.

The signature in the log was small. One `recorder: device already has
session, replacing` WARN — a line the recorder emits when a `CallStart`
arrives for a serial that still has an open session, documented in the
code as "Engine should have ended the prior call first, but be
defensive" — and, around it, two `recorder: call started` lines with no
matching `recorder: call ended`. Two recordings opened; neither completed;
the history panel showed neither. One re-key in five transmissions, two
recordings gone.

## Why the recorder waits

The recorder does not finalise a call the moment the engine publishes
`CallEnd`. It used to, and that produced a race: the composer's voice
chain is still writing the transmission's tail frames during teardown,
and a recorder that deleted the session on `CallEnd` dropped them onto a
closed file. `EnableDrainCoordination` — called once by the composer when
it is wired to this recorder — switches `handleEnd` into a deferred mode:

```go
// internal/voice/recorder.go (shape) — handleEnd, drain-coordinated
st := r.pendingFinalize[serial]
if st == nil { st = &pendingFinalize{}; r.pendingFinalize[serial] = st }
st.ce, st.haveCE = ce, true
if st.drained {              // the chain already signalled: finalize now
    delete(r.pendingFinalize, serial)
    r.finalizeCall(ce)
    return
}
if st.timer == nil {         // else wait for NotifyDrainComplete, bounded
    st.timer = time.AfterFunc(drainFinalizeTimeout, func() { r.finalizeOnDrainTimeout(serial) })
}
```

The composer's `handleEnd` cancels the chain, blocks on its `done`
channel until every tail frame is written, and only then calls
`NotifyDrainComplete` — for every `CallEnd`, chain or not, so the recorder
never waits for a signal that will not come. `drainFinalizeTimeout` = 3 s
is the backstop, and reaching it WARNs (`drain signal missing; finalizing
call on timeout`). This is the design
[Recording & Streaming Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
describes from the `CallComplete` side: finalize is the seam everything
downstream hangs off — the sidecar, the history row, normalisation,
upload — so it must run exactly once, after the last frame.

Both halves of the handshake were keyed by **device serial**. That is
enough as long as a serial carries one call at a time with a gap between
them. A re-key removes the gap.

## The instant a re-key creates

[DMR End to End Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }})
covered the control-side half of this story: a Voice LC Header for a
still-tracked call more than `headerRekeyDibits` (0.25 s) past the call's
anchor is a new transmission, so the Tier II state machine releases the
old call and grants the new one — `Rekeys` counts them — instead of
deduplicating the header against the call it was still tracking. That fix
is what made the re-key *grant* on 10 Sep. The engine's side of a grant on
a busy serial is to publish `CallEnd` for the previous call and
`CallStart` for the next in the same instant, and `VoicePool.Bind` stamps
each `CallStart`'s grant with a fresh `CallID` — the comment where
`internal/trunking/engine.go` publishes it says what it is for: "the voice chain +
recorder use [it] to fence cross-call audio bleed on a reused tap serial".

So the recorder sees, with no gap: `CallEnd(A)` — parked in
`pendingFinalize["serial"]`, waiting for A's chain to drain — then
`CallStart(B)` on the same serial, while A's session is still open and
A's drain signal is still in flight.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two timelines of one re-key on one serial: CallEnd A parked, CallStart B, A's drain arriving late, B's frames, CallEnd B. Old lane: B's start closes A unfinalised, A's serial-keyed drain finalises B's empty session, B's frames are dropped. New lane: B's start finalises A with A's own CallEnd, A's drain carries call id 1 and is ignored against session id 2, B's frames are written and B completes.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">re-key on one serial: CallEnd(A) parked · CallStart(B) · drain(A) late · B frames · CallEnd(B)</text>
  <g font-size="8" fill="var(--fg-muted)" text-anchor="middle">
    <text x="90" y="36">CallEnd(A) parked</text>
    <text x="220" y="36">CallStart(B)</text>
    <text x="350" y="36">drain(A) arrives</text>
    <text x="480" y="36">B frames</text>
    <text x="610" y="36">CallEnd(B)+drain</text>
  </g>
  <line x1="40" y1="44" x2="660" y2="44" stroke="var(--fg-muted)"/>
  <g stroke="var(--fg-muted)"><line x1="90" y1="40" x2="90" y2="48"/><line x1="220" y1="40" x2="220" y2="48"/><line x1="350" y1="40" x2="350" y2="48"/><line x1="480" y1="40" x2="480" y2="48"/><line x1="610" y1="40" x2="610" y2="48"/></g>
  <text x="34" y="92" text-anchor="end" fill="var(--fg-muted)" font-size="8">old</text>
  <rect x="40" y="62" width="620" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <g font-size="8" text-anchor="middle" fill="currentColor">
    <text x="90" y="80">waits for drain</text>
    <text x="220" y="80">"replacing":</text><text x="220" y="92">close A, NO finalize</text>
    <text x="350" y="80">serial-keyed:</text><text x="350" y="92">finalizes B (empty)</text>
    <text x="480" y="80">no session</text><text x="480" y="92">→ dropped</text>
    <text x="610" y="80">nothing to end</text>
  </g>
  <text x="350" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">A: no CallComplete · B: no frames, no CallComplete — the lone WARN, two missing ends</text>
  <text x="34" y="172" text-anchor="end" fill="var(--accent)" font-size="8">new</text>
  <rect x="40" y="142" width="620" height="60" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <g font-size="8" text-anchor="middle" fill="var(--accent)">
    <text x="90" y="160">waits for drain</text>
    <text x="220" y="160">finalizeDrainingBeforeReuse</text><text x="220" y="172">→ A completes (id 1)</text>
    <text x="350" y="160">id 1 ≠ session id 2</text><text x="350" y="172">→ ignored</text>
    <text x="480" y="160">id 2 = session</text><text x="480" y="172">→ written</text>
    <text x="610" y="160">id 2 → B completes</text>
  </g>
  <text x="350" y="194" text-anchor="middle" fill="var(--accent)" font-size="8" font-weight="bold">two .raw files, two CallComplete events</text>
  <text x="340" y="236" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TestRecorderRekeyDuringPendingDrainKeepsNextOver: frames per .raw = [5 4]; old recorder = one file, one complete</text>
</svg>
<figcaption>The same five events in both lanes. The old recorder answered "which call?" with "whichever is on this serial"; the new one answers with the CallID the engine stamped.</figcaption>
</figure>

## Two wrong moves on one serial

The first wrong move was in `handleStart`. Finding a session already open
for the serial, it took the defensive path: WARN, `close()` the old
session, delete it from the map, build the new one. `close()` releases
files and the vocoder; it does not run `finalizeLocked`, which is where
the `CallComplete` event, the metadata sidecar and the history row are
produced. A's session was closed; A's parked `CallEnd` was still in
`pendingFinalize`, now pointing at a session that no longer existed; A was
never completed. The `replacing` WARN in the field log is this move.

The second was in `NotifyDrainComplete`. When A's chain finished draining
— after B's `CallStart`, since the composer blocks on the chain's `done`
before signalling — the signal arrived keyed by serial, found the parked
state (A's `CallEnd`), and finalised. But `finalizeCall` looks up the
session by serial too, and the session on that serial was now **B's**. B
was finalised with A's `CallEnd` before B's first frame, and silently: a
session opens its files lazily on the first write (`sessionForWrite`), so
an unwritten session had nothing to close and nothing to publish — it was
simply deleted. Every later B frame reached `writeRawFrame`, found no
session, and returned `nil` (by design: a frame for a call the recorder is
not recording is not a fault). B's own `CallEnd` found nothing. Two
recordings gone, one WARN, zero errors.

The pattern is the one
[Season 1 Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
named: the control pipeline and the recording pipeline both model a call,
and the 10 Sep fix moved the control model (a re-key is two calls) without
the recording model following (a serial is one call at a time). The
recorder was not wrong about anything it knew; it did not know which call
a signal belonged to.

## The fence

The fix has two parts, and the second is the general one.
`finalizeDrainingBeforeReuse` runs at the top of `handleStart`'s session
path: if a `CallEnd` is parked for this serial, stop its timer, remove it,
and finalise that call **now**, with its own `CallEnd`, before the new
session takes the serial:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) finalizeDrainingBeforeReuse(cs trunking.CallStart) {
    r.mu.Lock()
    st := r.pendingFinalize[cs.DeviceSerial]
    if st == nil || !st.haveCE { r.mu.Unlock(); return }
    if st.timer != nil { st.timer.Stop() }
    delete(r.pendingFinalize, cs.DeviceSerial)
    ce := st.ce
    r.mu.Unlock()
    r.log.Debug("recorder: device re-keyed while its previous call was draining — finalizing that call now",
        "device", cs.DeviceSerial, "prev_call_id", ce.Grant.CallID, "call_id", cs.Grant.CallID)
    r.finalizeCall(ce)
}

// callIDsDiffer reports whether two Grant.CallIDs name different calls. A zero
// on either side is "unknown" and matches anything.
func callIDsDiffer(a, b uint64) bool { return a != 0 && b != 0 && a != b }
```

A's tail frames may still arrive after that; they carry A's `CallID` and
`sessionForWrite(serial, callID)` refuses to land them in B's session.

Then every finalize and drain path checks the identity.
`NotifyDrainCompleteForCall(serial, callID)` is `NotifyDrainComplete` plus
the `Grant.CallID` of the call whose chain drained: a signal that finds an
open session for a *different* call is the previous call's, arriving after
a re-key reused the serial — already finalised by
`finalizeDrainingBeforeReuse` — and is ignored; one whose ID differs from
a parked `CallEnd`'s is likewise dropped. `handleEnd` resets a stale `drained` flag left by a
different call so the newer call's own drain is still awaited. And
`finalizeCall` itself refuses a `CallEnd` naming a call the session no
longer belongs to, logging `ignoring call end for a call this device no
longer records`. The zero rule in `callIDsDiffer` keeps every legacy and
synthetic caller — un-stamped grants, older tests — on the old behaviour.

The plumbing threads the ID from the one place that knows it. The
composer's `handleEnd` type-asserts its drain coordinator for the
call-aware method and passes `ce.Grant.CallID`; the daemon's `fanoutSink`
forwards to each sink by the same assertion, falling back to the plain
form for sinks that only know the serial.

## The failing-first test, and what is still open

`TestRecorderRekeyDuringPendingDrainKeepsNextOver` in
`internal/voice/recorder_drain_test.go` replays the field log's shape with
a registered `loudVocoder` (11-byte frames, raw sidecars on). Over A
(`CallID` 1) writes five frames through `WriteRawFrameForCall`, then
`handleEnd` parks its `CallEnd`. Over B (`CallID` 2, `StartedAt` + 2 s)
starts on the same serial — the re-key — and the test asserts B has a
session. Then A's drain lands, late: `NotifyDrainCompleteForCall(serial,
1)`, and the test asserts B *still* has a session ("the previous call's
drain signal finalized the next over's session (the bug)"). B writes four
frames, ends, drains. The assertions: `rawFrameCounts` over the output
directory is `[5 4]` — two transmissions, both recorded — and the bus
carried two `KindCallComplete` events. Against the old recorder it writes
one `.raw` and publishes one `CallComplete`.

What the test does not cover is the live rig. The whole commit was
diagnosed from the 600 s capture and the log, and the #764/#771 rule
holds: the operator's next live run on a build with these fixes is what
confirms that a re-key records both overs. The sign it has, in `debug.log`: the `replacing` WARN replaced by the
Debug line `device re-keyed while its previous call was draining —
finalizing that call now`, with `prev_call_id` and `call_id` one apart.

## Where this goes next

The recorder now knows which call a signal belongs to. The next part is a
TETRA decoder that did not know which *broadcast* a fragment belonged to:
phantom neighbour cells — a 1.5 GHz site, a 402 MHz one — that turned out
to be two transmissions of a rotating D-NWRK-BROADCAST spliced together
across lost blocks.
[Part 11]({{ '/blog/solution-postmortem/issue-tracker-s2-11-phantom-neighbours/' | relative_url }})
follows the seams in MAC fragment reassembly.

## FAQ

**Why did a DMR re-key make GopherTrunk lose two recordings?**
The previous call's `CallEnd` was parked waiting for its chain to drain
when the re-key's `CallStart` arrived on the same serial. `handleStart`
closed the old session without finalising it (no `CallComplete`), and the
old call's serial-keyed drain signal then finalised the new session before
its first frame, so every frame of the next over was dropped.

**What does "recorder: device already has session, replacing" mean?**
A `CallStart` arrived for a device serial whose previous session is still
open. Before 17 Sep that path closed the old session unfinalised. Now
`finalizeDrainingBeforeReuse` first finalises a parked `CallEnd` for the
serial, logging `device re-keyed while its previous call was draining —
finalizing that call now` with `prev_call_id` / `call_id`.

**What is Grant.CallID used for in the recorder?**
It is the per-call identity `VoicePool.Bind` stamps on each grant.
`callIDsDiffer` compares it on every finalize and drain path:
`NotifyDrainCompleteForCall` ignores a drain for another call,
`finalizeCall` refuses a `CallEnd` for a call the serial no longer
records, and `sessionForWrite` refuses frames carrying another call's ID.
A zero ID matches anything, preserving legacy behaviour.

**Is the re-key recording fix verified on air?**
Not yet. It was diagnosed from the 17 Sep 600 s Fire2 capture and
`debug.log` and pinned by `TestRecorderRekeyDuringPendingDrainKeepsNextOver`,
which fails against the old recorder. The operator's next live run on a
build with the fix — a re-key producing two completed recordings — is the
on-air confirmation the #764/#771 rule requires.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Sixteen Thousand Codewords per Slot — The AACH Decoder Behind 'Overruns at 200 kS/s']({{ '/blog/solution-postmortem/issue-tracker-s2-09-sixteen-thousand-codewords/' | relative_url }})
· Next →
[Part 11: Phantom Neighbours — Spliced Broadcasts and the Seams in TETRA MAC Reassembly]({{ '/blog/solution-postmortem/issue-tracker-s2-11-phantom-neighbours/' | relative_url }})
