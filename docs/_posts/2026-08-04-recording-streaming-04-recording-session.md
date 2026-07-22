---
title: "Recording, Composition & Streaming, Part 4: The Recording Session"
description: How GopherTrunk's per-call recording session opens files lazily on the first write so dead-keys leave nothing on disk, carries identity forward through dormant post-segment parks, and runs decode-only when no recordings directory is configured.
category: deep-dives
keywords: lazy file open recorder, dead key no file, decode only mode, recording session state machine, dormant session park, gophertrunk recorder session, build session vocoder, empty outdir live audio
tags: [recording, session, state-machine, go, lazy-init, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 4
---

*Part 4 of **Recording, Composition & Streaming**, opening the black box the last
two posts leaned on: the `recordingSession`. [Part 3]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})
showed segments parking a session dormant; this post is the whole state machine
behind that. Our 3 p.m. dispatch on talkgroup 101 gets a session the instant its
grant lands — but not a file. If the transmitter keys and unkeys without a word,
we want nothing on disk. This is how the recorder threads that needle.*

> **TL;DR:** A `recordingSession` is **prepared** on `CallStart` but its files are
> **not opened** — `buildSession` resolves paths and builds the vocoder, and
> `openSessionFiles` runs only on the first actual write. A dead-key therefore
> leaves no WAV, no `.raw`, and no empty talkgroup folder. A post-segment
> **dormant** session carries the call's identity with no open files, reopening
> lazily on the next over. And an **empty `OutDir`** flips the recorder into
> **decode-only** mode: every call still builds a vocoder and feeds live audio, it
> just never writes a file.

**Key takeaways**

- **Files open lazily.** `buildSession` prepares paths and a vocoder; the WAV and
  `.raw` are created by `openSessionFiles` on the first sample or frame — so a
  call with no audio touches disk zero times.
- **The talkgroup directory is lazy too.** It's `MkdirAll`'d just before the first
  file, so a followed-but-silent grant leaves no empty `<system>/<tg>` folder.
- **Dormant sessions carry identity.** After a segment, the parked session holds
  only the `CallStart` and `CallID`; `sessionForWrite` rebuilds it under a fresh
  timestamp when the next write arrives.
- **Decode-only mode** (empty `OutDir`) decodes and fans live audio with no files —
  how live browser audio works without a configured recordings directory.

## Cheat sheet

| Thing | What it does | Where in code |
|---|---|---|
| `recordingSession` | Per-call state: files, vocoder, paths, `CallID` | `internal/voice/recorder.go` |
| `buildSession` | Prepare paths + vocoder; open **no** files | `internal/voice/recorder.go` |
| `openSessionFiles` | Lazily create the WAV / `.raw` on first write | `internal/voice/recorder.go` |
| `sessionForWrite` | Resolve/reopen the session for a write; `CallID` fence | `internal/voice/recorder.go` |
| `handleStart` | Gate the `CallStart`, then build + register a session | `internal/voice/recorder.go` |
| `decodeOnly` | Empty `OutDir` → decode + live audio, no files | `internal/voice/recorder.go` |

## In this post

- **The session struct** — what one call's state actually holds.
- **Prepare, don't open** — `buildSession` versus `openSessionFiles`.
- **The write path** — how `sessionForWrite` reopens a dormant session lazily.
- **The gates** — what `handleStart` checks before a session ever exists.
- **Decode-only mode** — the recorder as a pure live-audio decoder.

## One call's state

Everything the recorder knows about an in-flight call lives in one struct, keyed
by device serial in the recorder's `sessions` map:

```go
// internal/voice/recorder.go (shape)
type recordingSession struct {
    wav         *WavWriter   // nil until the first write (or in decode-only)
    wavPath     string       // resolved at buildSession; "" for a dormant park
    raw         *os.File     // the .raw sidecar, likewise lazy
    rawPath     string
    vocoder     Vocoder      // per-protocol; nil for analog
    vocoderName string
    sampleRate  uint32       // WAV header rate (8 kHz for vocoded calls)
    lastSample  int16        // for the end-of-call fade
    rawWanted   bool         // should a .raw be opened on first frame?
    startedAt   time.Time
    callID      uint64       // Grant.CallID — the cross-call fence key
    cs          trunking.CallStart // retained so a segment roll can reopen
}
```

The two fields that make the whole state machine work are `wav` and `cs`. A `wav`
of `nil` means "no file is open" — which is true in three distinct situations: a
freshly prepared session that hasn't been written yet, a dormant post-segment park,
and a decode-only session that will never open a file. `cs` (the originating
`CallStart`) is retained so a segment roll can open the *next* file with the same
grant and talkgroup under a new timestamp. The struct is small on purpose: it is
the identity of a call plus a few handles that may or may not be live.

## Prepare, don't open

`handleStart` builds a session; `buildSession` prepares it *without* touching disk.
This is the design's central move, and its doc comment is explicit that it "does
NOT open the files":

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) buildSession(cs trunking.CallStart, startedAt time.Time) *recordingSession {
    dir := r.directoryFor(cs)          // path only — NOT created here
    base := r.basenameFor(withStart(cs, startedAt))
    s := &recordingSession{startedAt: startedAt, cs: cs, callID: cs.Grant.CallID}

    // Instantiate the per-protocol vocoder if one is mapped, before the
    // WAV is opened (its header rate tracks the vocoder's native rate).
    if name, ok := r.vocoderForProtocol[cs.Grant.Protocol]; ok && name != "" {
        if v, err := DefaultRegistry.New(name); err == nil {
            // opt-in enhancement + startup squelch installed here
            s.vocoder, s.vocoderName = v, name
        }
    }
    s.sampleRate = r.sampleRate
    if s.vocoder != nil {
        s.sampleRate = pcmHzDefault // vocoder output is always 8 kHz
    }
    s.wavPath = filepath.Join(dir, base+".wav")
    if r.writeRaw || cs.Grant.ProVoice ||
        dmrVoiceProtocol(cs.Grant.Protocol) || tetraVoiceProtocol(cs.Grant.Protocol) {
        s.rawPath = filepath.Join(dir, base+".raw")
        s.rawWanted = true
    }
    // Files are NOT opened here — see openSessionFiles.
    return s
}
```

So after `buildSession`, a session has a vocoder, a chosen sample rate, and two
resolved paths — but nothing on disk. The talkgroup directory itself isn't even
created yet. That is deliberate: a grant that is *followed* but never yields audio
— a dead-key, an immediately-aborted encrypted call, a voice tap left
off-frequency while a hunt borrows the SDR — leaves neither a header-only WAV nor
an empty `<system>/<talkgroup>` folder behind. The WAV header rate is worth one
note: a *vocoded* call is forced to 8 kHz regardless of `recordings.sample_rate`,
because the vocoder's output is always 8 kHz and appending it under a different
header rate would play back garbled; `recordings.sample_rate` applies only to
analog PCM fed via `WritePCM`.

The actual disk work is quarantined in `openSessionFiles`, called from the write
path on the first sample or frame:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) openSessionFiles(s *recordingSession) error {
    if s.wav != nil {
        return nil // idempotent: already open
    }
    if dir := filepath.Dir(s.wavPath); dir != "" {
        if err := os.MkdirAll(dir, 0o755); err != nil { // lazy dir creation
            return err
        }
    }
    wav, err := NewWavFile(s.wavPath, s.sampleRate)
    if err != nil {
        if s.vocoder != nil { s.vocoder.Close(); s.vocoder = nil }
        return err
    }
    s.wav = wav
    if s.rawWanted && s.raw == nil {
        if raw, err := os.Create(s.rawPath); err == nil {
            s.raw = raw
        }
    }
    return nil
}
```

`TestRecorderNoAudioLeavesNoFiles` and `TestRecorderNoAudioLeavesNoDir` in
`recorder_idle_test.go` pin this behaviour: a `CallStart` with no following write
produces no files *and* no directory.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="A state machine for a recording session: CallStart builds a prepared session that has a vocoder and resolved paths but no open files. On the first write, openSessionFiles lazily creates the WAV and dot-raw and the session becomes files-open. A CallSegment finalizes and parks it dormant, from which the next write reopens fresh files back to files-open. A CallEnd from either the prepared or dormant state, with no audio ever written, closes to ended leaving nothing on disk; a CallEnd from files-open closes and publishes CallComplete.">
  <rect x="20" y="76" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="80" y="96" text-anchor="middle" fill="currentColor" font-size="10">prepared</text>
  <text x="80" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">paths+vocoder</text>
  <text x="80" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CallStart</text>
  <line x1="140" y1="99" x2="228" y2="99" stroke="var(--accent)"/><polygon points="228,95 238,99 228,103" fill="var(--accent)"/>
  <text x="184" y="92" text-anchor="middle" fill="var(--accent)" font-size="8">first write</text>
  <text x="184" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">openSessionFiles</text>
  <rect x="238" y="76" width="130" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="303" y="96" text-anchor="middle" fill="var(--accent)" font-size="10">files-open</text>
  <text x="303" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">WAV + .raw</text>
  <line x1="368" y1="99" x2="452" y2="99" stroke="currentColor"/><polygon points="452,95 462,99 452,103" fill="currentColor"/>
  <text x="410" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CallSegment</text>
  <rect x="462" y="76" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="522" y="96" text-anchor="middle" fill="currentColor" font-size="10">dormant</text>
  <text x="522" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">keeps identity</text>
  <path d="M 462 84 C 400 44, 360 44, 303 74" fill="none" stroke="currentColor"/>
  <polygon points="307,68 301,76 312,74" fill="currentColor"/>
  <text x="383" y="46" text-anchor="middle" fill="var(--fg-muted)" font-size="8">next write → reopen</text>
  <line x1="80" y1="122" x2="80" y2="164" stroke="var(--fg-muted)"/><polygon points="76,158 80,168 84,158" fill="var(--fg-muted)"/>
  <line x1="522" y1="122" x2="522" y2="164" stroke="var(--fg-muted)"/><polygon points="518,158 522,168 526,158" fill="var(--fg-muted)"/>
  <rect x="250" y="152" width="120" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="310" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ended · no file</text>
  <line x1="140" y1="164" x2="248" y2="167" stroke="var(--fg-muted)"/>
  <line x1="462" y1="164" x2="372" y2="167" stroke="var(--fg-muted)"/>
  <text x="310" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CallEnd with no audio written</text>
</svg>
<figcaption>The session state machine: prepared on CallStart, files opened lazily on the first write, parked dormant across a segment, and — when no audio ever arrives — ended without ever touching disk.</figcaption>
</figure>

## The write path reopens lazily

Every write — `WritePCM` and the `WriteRawFrame` family — first resolves the
session through `sessionForWrite`, which is where the lazy reopen lives:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) sessionForWrite(serial string, callID uint64) *recordingSession {
    r.mu.Lock()
    defer r.mu.Unlock()
    s, ok := r.sessions[serial]
    if !ok {
        return nil // no session (the composer can race ahead of CallStart)
    }
    if callID != 0 && s.callID != 0 && callID != s.callID {
        return nil // cross-call fence: a stale frame from a reused tap serial
    }
    if r.decodeOnly {
        return s // no files ever; the vocoder is enough to fan live audio
    }
    if s.wav == nil {
        if s.wavPath == "" {
            // A dormant park carries only cs+callID: build a fresh session
            // under a new timestamp before opening its files.
            ns := r.buildSession(s.cs, time.Now().UTC())
            if ns == nil { return nil }
            r.sessions[serial] = ns
            s = ns
        }
        if err := r.openSessionFiles(s); err != nil {
            delete(r.sessions, serial) // don't retry on every frame
            return nil
        }
    }
    return s
}
```

This one function reconciles all three `wav == nil` cases. A **prepared** session
(has a `wavPath`, no files) just needs `openSessionFiles`. A **dormant** park (no
`wavPath` at all) is first rebuilt via `buildSession` under a fresh `time.Now()`
timestamp — that's how the second over of a per-transmission call gets a new
filename while keeping the same grant and `CallID`. A **decode-only** session is
handed straight back with `wav == nil`, because it will never open a file. And the
`CallID` fence sits right at the top: a frame whose call identity doesn't match the
open session is rejected *without* reopening a dormant session, so a mismatched
frame can't even spawn an empty file. `TestRecorderCallIDFenceDropsStaleFrames`
covers that path. (The dormant `recordingSession` can also just be closed with no
open files — `TestDormantSessionCloseNoPanic` and `TestWavWriterNilClose` in
`recordingsession_close_test.go` guard against the nil-writer panic that would
otherwise lurk there.)

## The gates before a session exists

`handleStart` is the decision that decides whether a session is built at all. It
runs a short sequence of gates, each of which can drop the `CallStart` before any
session is registered:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) handleStart(cs trunking.CallStart) {
    if r.recordDisabled.Load() {
        return // operator toggled recording off at runtime
    }
    if !r.decodeOnly && cs.Talkgroup != nil && !cs.Talkgroup.Record {
        return // talkgroup record=false: follow + play live, write nothing
    }
    if r.skipEncrypted && cs.Grant.Encrypted {
        return // opted out of recording encrypted calls
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    // …replace any stale session on this serial…
    s := r.buildSession(cs, cs.StartedAt)
    if s == nil {
        return
    }
    r.sessions[cs.DeviceSerial] = s
}
```

The ordering has one subtlety worth calling out. The `record=false` gate is
skipped when `decodeOnly` is set — a decode-only recorder writes nothing to disk
*regardless*, so if it also dropped `record=false` talkgroups here, those
talkgroups would be silent on the *live* feed too. The whole point of decode-only
mode is live audio; dropping the call at the gate would defeat it. The
`recordDisabled` gate, by contrast, applies to everyone and is the runtime "stop
laying down WAVs" switch — and it deliberately leaves in-flight sessions alone, so
flipping it mid-conversation doesn't truncate the head of a call.

## Decode-only mode

The `decodeOnly` flag is set once, in `NewRecorder`, purely from whether an output
directory was configured:

```go
// internal/voice/recorder.go (shape)
func NewRecorder(opts RecorderOptions) (*Recorder, error) {
    // An empty OutDir runs the recorder in decode-only mode: it still
    // decodes each call and feeds the live-audio tap, it just never
    // writes files. This is how live browser audio works without a
    // configured recordings dir.
    decodeOnly := opts.OutDir == ""
    if !decodeOnly {
        os.MkdirAll(opts.OutDir, 0o755)
    }
    // …vocoder map, normalize/enhance defaults, displayLoc…
}
```

In this mode the recorder is a pure decoder. `buildSession` still constructs the
per-protocol vocoder, so `writeRawFrame` can decode each frame; `sessionForWrite`
hands back the session with `wav == nil`; the decode result skips the WAV write but
still fans to the decoded-PCM live tap (Part 2's fan-out). The result is that a
GopherTrunk instance with no `recordings:` directory configured *still* streams
live digital audio to the browser — it decodes every call, it just persists none.
This is the recorder's contribution to the "optional means absent, not idle"
principle from Part 1: with no `OutDir`, there is no persistence, but the live
path is fully alive. `TestRecorderSuppressesAllIdleRecording` and the idle-suite
neighbours in `recorder_idle_test.go` validate the "still decodes, writes nothing"
contract.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="A decision flow for a CallStart entering the recorder: it first checks recordDisabled — if recording is off at runtime, the call is dropped. Otherwise, if the talkgroup is flagged record=false and the recorder is not in decode-only mode, it drops. Otherwise, if skipEncrypted is set and the grant is encrypted, it drops. Passing all gates, it builds a session. A separate branch shows that when OutDir is empty the recorder is in decode-only mode, which builds a vocoder and feeds live audio but writes no files.">
  <rect x="20" y="90" width="110" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="75" y="111" text-anchor="middle" fill="var(--accent)" font-size="10">CallStart</text>
  <line x1="130" y1="107" x2="158" y2="107" stroke="currentColor"/><polygon points="158,103 168,107 158,111" fill="currentColor"/>
  <rect x="168" y="88" width="96" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="216" y="104" text-anchor="middle" fill="currentColor" font-size="9">recordDisabled?</text>
  <text x="216" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">record=false? enc?</text>
  <line x1="216" y1="126" x2="216" y2="168" stroke="var(--fg-muted)"/><polygon points="212,162 216,172 220,162" fill="var(--fg-muted)"/>
  <text x="216" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">yes → drop, no session</text>
  <line x1="264" y1="107" x2="330" y2="107" stroke="currentColor"/><polygon points="330,103 340,107 330,111" fill="currentColor"/>
  <text x="300" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pass</text>
  <rect x="340" y="88" width="110" height="38" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="395" y="104" text-anchor="middle" fill="var(--accent)" font-size="9">buildSession</text>
  <text x="395" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">vocoder + paths</text>
  <line x1="450" y1="107" x2="500" y2="107" stroke="currentColor"/><polygon points="500,103 510,107 500,111" fill="currentColor"/>
  <rect x="510" y="72" width="130" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="575" y="91" text-anchor="middle" fill="currentColor" font-size="9">OutDir set → files</text>
  <rect x="510" y="112" width="130" height="34" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="575" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="9">OutDir empty →</text>
  <text x="575" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">decode-only, live only</text>
  <line x1="450" y1="120" x2="508" y2="128" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
</svg>
<figcaption>From CallStart to a session: three gates can drop the call before any state exists; passing them builds a session whose files open only when OutDir is set — an empty OutDir yields a decode-only, live-audio-only recorder.</figcaption>
</figure>

## Where this goes next

[Part 5]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
follows the file that `openSessionFiles` finally creates: the crash-safe WAV. Once
a session opens its `WavWriter`, how does GopherTrunk keep the on-disk file valid
even if the daemon dies mid-call — the streaming header, the length-field patch on
close, and why a half-written recording is still playable.

## FAQ

**Why open files lazily instead of on `CallStart`?**
Because a grant that yields no audio — a dead-key, an aborted encrypted call, a tap
left off-frequency — should leave nothing behind. `buildSession` resolves paths and
builds the vocoder but opens no files; `openSessionFiles` runs on the first write.
No audio, no WAV, no `.raw`, and not even an empty talkgroup directory.

**What is a dormant session?**
After a per-transmission segment roll, the recorder parks a session that holds only
the originating `CallStart` and `CallID` with no open files. It carries the call's
identity forward so the next over reopens a fresh file under a new timestamp — and
if the call ends first, it closes cleanly without leaving an empty trailing file.

**What does decode-only mode do?**
When `OutDir` is empty, the recorder builds each call's vocoder and decodes every
frame but never opens a file. The decoded PCM still fans to the live-audio tap, so
browser and host audio work with no recordings directory configured. It's how you
run GopherTrunk as a pure live scanner with nothing written to disk.

**Does disabling recording at runtime truncate the call I'm hearing?**
No. The `recordDisabled` gate stops *new* sessions from opening files, but leaves
in-flight sessions alone — they finish naturally on `CallEnd`. Flipping the switch
mid-conversation won't cut the head off the call already being recorded.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Assembling a Call]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})
· Next →
[Part 5: The WAV on Disk]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
