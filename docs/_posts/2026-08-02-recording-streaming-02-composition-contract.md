---
title: "Recording, Composition & Streaming, Part 2: The Composition Contract"
description: How composed audio crosses from GopherTrunk's demod composer into its recorder through three consumer-owned interfaces, why the recorder is the one place a call is decoded, and how the PCM path and the raw-vocoder-frame path differ.
category: deep-dives
keywords: consumer owned interface go, dependency inversion audio pipeline, pcm sink raw frame sink, vocoder decode fanout, recorder lone decoder, iqsource pcmsink enginehooks, digital voice raw frame, gophertrunk composer recorder contract
tags: [recording, architecture, interfaces, go, vocoder, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 2
---

*Part 2 of **Recording, Composition & Streaming**, tracing the seam between the
demod composer and the recorder. In [Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})
we followed our 3 p.m. dispatch on talkgroup 101 through four bus-subscribing
subsystems. Now we zoom into the one hop that carries the call's actual audio:
how the composer hands samples to the recorder without either package importing
the other's concrete types, and why the recorder — not the composer — is the
single place a digital call gets decoded.*

> **TL;DR:** The composer knows the recorder only through three tiny,
> **consumer-declared** interfaces — `IQSource`, `PCMSink`, `EngineHooks`. It
> never imports `voice.Recorder`. Analog FM chains demodulate to PCM in the
> composer and push it via `WritePCM`; **digital** chains emit undecoded vocoder
> frames via `WriteRawFrame`, and the **recorder** decodes them. That makes the
> recorder the *lone decoder*: one vocoder pass fills the WAV and fans the same
> PCM to every live listener, so a call is never decoded twice.

**Key takeaways**

- The composer package depends on **interfaces it defines itself**
  (`PCMSink` is `WritePCM(serial, samples)` and nothing more), not on the
  recorder. Dependency arrows point *into* the composer, not out of it.
- **Two paths cross the seam.** Analog FM → real PCM via `WritePCM`. Digital
  (P25/DMR/…) → raw IMBE/AMBE frames via `WriteRawFrame`, which the recorder
  decodes with a per-call vocoder.
- The recorder is the **lone decoder** for digital voice. `writeRawFrame`
  decodes once, then fans out: `.raw` sidecar, live raw tap, WAV, and the
  decoded-PCM live tap all come from that single pass.
- The daemon glues the two packages together with a `fanoutSink` — one object
  that satisfies `PCMSink` for the composer while multiplexing to the recorder,
  the host player, and the tone-out detector.

## Cheat sheet

| Thing | What it is | Where in code |
|---|---|---|
| `IQSource` | The slice of an SDR the composer needs (`StreamIQ` + `SampleRateHz`) | `internal/voice/composer/composer.go` |
| `PCMSink` | `WritePCM(serial, []int16)` — the composer's whole view of the recorder | `internal/voice/composer/composer.go` |
| `EngineHooks` | `Touch` / `EndCall` / `UpdateSignal` / `UpdateDemod` back to the engine | `internal/voice/composer/composer.go` |
| `WritePCM` | Recorder entry for analog PCM; appends to the open WAV | `internal/voice/recorder.go` |
| `WriteRawFrame` | Recorder entry for a digital vocoder frame; decode-and-fan | `internal/voice/recorder.go` |
| `DecodedPCMSink` / `RawFrameSink` | Live taps the recorder feeds after decode | `internal/voice/recorder.go` |
| `fanoutSink` | Daemon glue: one `PCMSink` that multiplexes to many | `cmd/gophertrunk/daemon.go` |

## In this post

- **Consumer-owned interfaces** — why `PCMSink` lives in the composer, not the recorder.
- **The two crossings** — the analog PCM path versus the digital raw-frame path.
- **The lone decoder** — how `writeRawFrame` decodes once and fans out four ways.
- **The daemon glue** — `fanoutSink`, `playerSink`, and how the wiring is assembled.
- Where the *shape* of a call (overs, hangtime, segments) comes from — the teaser for Part 3.

## Who imports whom

The interesting fact about the composer package is what it does **not** import.
It builds per-call demod chains that end by handing audio to the recorder, yet it
has no compile-time knowledge of `voice.Recorder` at all. Instead it declares the
narrow slice of behaviour it needs and accepts anything that satisfies it. This
is the classic Go move — **interfaces belong to the consumer** — and the composer
uses it three times.

```go
// internal/voice/composer/composer.go (shape)

// The subset of an SDR device the composer consumes.
type IQSource interface {
    StreamIQ(ctx context.Context) (<-chan []complex64, error)
    SampleRateHz() uint32
}

// The subset of voice.Recorder we touch. WritePCM matches it exactly.
type PCMSink interface {
    WritePCM(deviceSerial string, samples []int16) error
}

// The engine calls the chain uses to keep the engine in sync.
type EngineHooks interface {
    Touch(deviceSerial string)
    EndCall(deviceSerial string, reason trunking.EndReason) bool
    UpdateSignal(deviceSerial string, dbfs float64)
    UpdateDemod(deviceSerial string, evmPct, snrDB float64)
}
```

`PCMSink` is the whole of the composer's view of the recorder: a single method
that takes a device serial and a slice of `int16`. The doc comment is blunt about
it — "`Recorder.WritePCM` matches this signature exactly" — but the point is that
the composer doesn't *know* that. It knows `PCMSink`. The recorder could be
swapped for a test double, a null sink, or a `fanoutSink` (below) and the composer
would neither notice nor need recompiling.

The `Options` struct the daemon fills in is entirely interface-typed on these
seams:

```go
// internal/voice/composer/composer.go (shape)
type Options struct {
    Bus     *events.Bus
    Devices Devices     // resolves an IQSource by serial
    Sink    PCMSink     // typically *voice.Recorder, via a fanoutSink
    Engine  EngineHooks // typically *trunking.Engine
    // …IQSampleRate, PCMSampleRate, VoiceHangtime, SplitPerTransmission, DSP knobs
}
```

`Devices` is a second consumer interface — `FindBySerial(serial) IQSource` — so
even the SDR pool reaches the composer through an abstraction. The concrete
`sdr.Pool`, the concrete `trunking.Engine`, and the concrete `voice.Recorder` all
live in other packages; the composer holds only their shadows. That is why the
whole demod pipeline can be exercised in a unit test with an in-memory IQ channel
and a map-backed `Devices` — no SDR, no daemon.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="Dependency-inversion diagram: the composer package sits in the centre and declares three interfaces it owns — IQSource, PCMSink and EngineHooks. The SDR pool, the recorder and the trunking engine live in other packages and satisfy those interfaces, so every dependency arrow points inward toward the composer rather than the composer importing any concrete type.">
  <rect x="230" y="70" width="200" height="70" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="94" text-anchor="middle" fill="var(--accent)" font-size="11">composer package</text>
  <text x="330" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">declares IQSource · PCMSink</text>
  <text x="330" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">EngineHooks · Devices</text>
  <rect x="20" y="24" width="160" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="100" y="44" text-anchor="middle" fill="currentColor" font-size="10">sdr.Pool</text>
  <rect x="20" y="90" width="160" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="100" y="105" text-anchor="middle" fill="currentColor" font-size="10">voice.Recorder</text>
  <text x="100" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">satisfies PCMSink</text>
  <rect x="20" y="152" width="160" height="34" rx="5" fill="none" stroke="currentColor"/>
  <text x="100" y="172" text-anchor="middle" fill="currentColor" font-size="10">trunking.Engine</text>
  <line x1="180" y1="41" x2="228" y2="88" stroke="currentColor"/><polygon points="222,82 230,90 219,90" fill="currentColor"/>
  <line x1="180" y1="107" x2="228" y2="105" stroke="currentColor"/><polygon points="221,101 230,105 221,109" fill="currentColor"/>
  <line x1="180" y1="169" x2="228" y2="122" stroke="currentColor"/><polygon points="219,120 230,120 224,128" fill="currentColor"/>
  <text x="205" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">implements ↗</text>
  <rect x="470" y="86" width="170" height="38" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="555" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">composer imports</text>
  <text x="555" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NO concrete type ↑</text>
  <line x1="470" y1="105" x2="432" y2="105" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
</svg>
<figcaption>Dependency inversion at the composition seam: concrete packages satisfy interfaces the composer owns, so every arrow points inward and the composer imports no recorder type.</figcaption>
</figure>

## Two ways audio crosses the seam

Once a chain is running, how audio reaches the recorder depends on whether the
protocol is analog or digital — and the difference is the reason there are two
recorder entry points.

**Analog FM** (Motorola Type II, EDACS clear, LTR, MPT-1327, conventional FM)
carries voice as plain narrowband FM. The composer's `runFMChain` does the whole
job: low-pass the IQ, decimate, quadrature-demodulate, convert to `int16`, and
call `WritePCM`. By the time audio crosses the seam it is finished PCM — the
recorder just appends it to the WAV. There is nothing left to decode.

**Digital** protocols (P25 Phase 1/2, DMR, TETRA, …) are the opposite. Their
chains do the DSP — carrier recovery, symbol timing, FEC — and produce not audio
but **vocoder frames**: 88-bit IMBE codewords for P25 Phase 1, AMBE+2 for DMR and
Phase 2, and so on. Those chains never call `WritePCM`. They call `WriteRawFrame`
with the raw codec bytes, and the *recorder* runs the vocoder. (How each chain is
selected and clocked — C4FM vs CQPSK, the FIR decimation, the boundary tracker —
is the subject of [Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }});
here we care only about which recorder method the chain calls and what the
recorder does with it.)

The recorder exposes a small family of raw-frame entry points, differing only in
how much side information the chain can supply:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) WriteRawFrame(serial string, frame []byte) error
func (r *Recorder) WriteRawFrameWithErrors(serial string, frame []byte, correctedBits int) error
func (r *Recorder) WriteRawFrameForCall(serial string, callID uint64, frame []byte, correctedBits int) error
```

All three funnel into one unexported `writeRawFrame`. `WriteRawFrameWithErrors`
adds the channel-FEC corrected-bit count, which the pure-Go IMBE decoder folds
into its adaptive smoothing. `WriteRawFrameForCall` adds the `Grant.CallID`, which
lets the recorder **fence a stale frame** from a call that previously held a
reused voice-tap serial — the cross-call audio-bleed guard we'll return to in
Part 7. A P25 Phase 1 chain that knows all of this calls the richest variant; a
simple test harness calls plain `WriteRawFrame`.

## The lone decoder

Here is the design decision the whole series leans on: **a digital call is decoded
in exactly one place.** Not once for the recording and again for live audio — once,
in the recorder, with the result fanned to every consumer. The `writeRawFrame` hot
path is where that fan-out happens.

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) writeRawFrame(serial string, callID uint64, frame []byte,
    correctedBits int, haveErrs bool) error {

    s := r.sessionForWrite(serial, callID) // nil ⇒ no session (or CallID fenced)
    if s == nil {
        return nil
    }
    if s.raw != nil {
        s.raw.Write(frame) // (1) verbatim to the .raw sidecar
    }
    if r.rawTap != nil {   // (2) verbatim to the live raw tap (gRPC include_raw)
        // prefers RawFrameCallSink (CallID-fenced) over plain WriteRawFrame
    }
    if s.vocoder != nil {
        if haveErrs { /* hand correctedBits to an ErrorAware vocoder */ }
        samples, err := s.vocoder.Decode(frame) // the one decode
        if err != nil { return nil }            // log + drop from PCM, keep sidecar
        if s.wav != nil {
            s.wav.WriteSamples(samples)          // (3) into the WAV
        }
        if r.decodedTap != nil {                 // (4) to the live decoded-PCM tap
            // prefers DecodedPCMCallSink (CallID-fenced) over plain WritePCM
        }
    }
    return nil
}
```

One frame in, four things out. The raw bytes go to disk **and** to the raw live
tap *before* any decode, because they exist even for protocols with no in-process
vocoder (ProVoice, or an encrypted call) — the raw sidecar is often the only
capture of such a call. Then, if a vocoder exists for the protocol, the single
`Decode` call produces PCM that is written to the WAV *and* handed to the decoded
live tap. Recording and live listening therefore share one vocoder pass; a
decode error drops the frame from PCM but keeps the sidecar intact.

The two live taps are themselves consumer-owned interfaces, declared in the
recorder for the same reason the composer declares `PCMSink` — to avoid an import
cycle back to the packages that consume the audio:

```go
// internal/voice/recorder.go (shape)
type DecodedPCMSink interface {
    WritePCM(deviceSerial string, samples []int16) error
}
type RawFrameSink interface {
    WriteRawFrame(deviceSerial, vocoder string, frame []byte) error
}

func (r *Recorder) SetDecodedPCMSink(s DecodedPCMSink) { r.decodedTap = s }
func (r *Recorder) SetRawFrameSink(s RawFrameSink)     { r.rawTap = s }
```

Both are wired **once**, at daemon construction, before `Run` starts — so the hot
path reads `r.decodedTap` and `r.rawTap` without locking. Each has a call-aware
sibling (`DecodedPCMCallSink.WritePCMForCall`, `RawFrameCallSink.WriteRawFrameForCall`)
that carries the `CallID`; `writeRawFrame` prefers the call-aware form via a type
assertion so the live stream is fenced by the same identity the WAV uses.

Why does this matter enough to build a whole discipline around? Because the
composer's digital chains emit **only** raw frames — they never produce PCM. If
the recorder weren't the one to decode them, the live-audio path would be silent
for every digital call while the recordings played back fine. That was a real
bug (issue #598): decoded audio reached the WAV but never reached the
`WritePCM`-only live sinks, because nothing decoded on their behalf. Making the
recorder the lone decoder and fanning its output is what fixed it.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="One raw vocoder frame enters the recorder's writeRawFrame method and fans four ways: verbatim to the dot-raw sidecar on disk, verbatim to the live raw tap, and — after a single vocoder Decode pass — as PCM into the WAV file and to the live decoded-PCM tap. The single Decode box is highlighted to show recording and live listening share one decode.">
  <rect x="10" y="92" width="120" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="70" y="110" text-anchor="middle" fill="var(--accent)" font-size="10">raw frame</text>
  <text x="70" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">IMBE / AMBE bytes</text>
  <line x1="130" y1="110" x2="168" y2="110" stroke="currentColor"/><polygon points="168,106 178,110 168,114" fill="currentColor"/>
  <rect x="178" y="86" width="110" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="233" y="106" text-anchor="middle" fill="currentColor" font-size="10">writeRawFrame</text>
  <text x="233" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fan-out</text>
  <line x1="288" y1="98" x2="470" y2="26" stroke="currentColor"/><polygon points="464,22 474,24 468,33" fill="currentColor"/>
  <rect x="474" y="12" width="176" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="562" y="30" text-anchor="middle" fill="currentColor" font-size="9">.raw sidecar (verbatim)</text>
  <line x1="288" y1="104" x2="470" y2="70" stroke="currentColor"/><polygon points="464,66 474,68 467,77" fill="currentColor"/>
  <rect x="474" y="54" width="176" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="562" y="72" text-anchor="middle" fill="currentColor" font-size="9">live raw tap (gRPC)</text>
  <line x1="288" y1="118" x2="326" y2="130" stroke="var(--accent)"/><polygon points="320,126 330,131 319,134" fill="var(--accent)"/>
  <rect x="330" y="112" width="120" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="390" y="130" text-anchor="middle" fill="var(--accent)" font-size="10">vocoder.Decode</text>
  <text x="390" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="8">one pass → PCM</text>
  <line x1="450" y1="126" x2="470" y2="112" stroke="currentColor"/><polygon points="464,108 474,110 468,118" fill="currentColor"/>
  <rect x="474" y="96" width="176" height="28" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="562" y="114" text-anchor="middle" fill="var(--accent)" font-size="9">WAV (WriteSamples)</text>
  <line x1="450" y1="140" x2="470" y2="150" stroke="currentColor"/><polygon points="464,146 474,151 463,154" fill="currentColor"/>
  <rect x="474" y="138" width="176" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="562" y="156" text-anchor="middle" fill="currentColor" font-size="9">decoded-PCM live tap</text>
  <text x="330" y="196" fill="var(--fg-muted)" font-size="9">Analog FM skips all this: runFMChain calls WritePCM with finished PCM directly.</text>
</svg>
<figcaption>The decode-and-fan hot path: one digital frame becomes the sidecar, the raw tap, the WAV, and the decoded live tap — with a single vocoder pass shared by the recording and the live listeners.</figcaption>
</figure>

## The daemon glue

The composer wants one `PCMSink`. The daemon has several things that want the
audio — the recorder, the host speaker player, the tone-out detector. It reconciles
the two with a tiny adapter, `fanoutSink`, which *is* a slice of `PCMSink` and
satisfies `PCMSink` itself:

```go
// cmd/gophertrunk/daemon.go (shape)
type fanoutSink []composer.PCMSink

func (f fanoutSink) WritePCM(serial string, samples []int16) error {
    for _, s := range f {
        _ = s.WritePCM(serial, samples)
    }
    return nil
}
```

That is the whole trick behind "one composer, many consumers." The composer holds
a `fanoutSink` as its `Sink` and calls `WritePCM` once; the fan-out loops. But
`fanoutSink` also implements the richer shapes — `WritePCMForCall` (CallID-fenced
for the reused voice taps), and crucially the raw-frame methods `WriteRawFrame`,
`WriteRawFrameWithErrors`, and `WriteRawFrameForCall`. Each uses a type assertion
to forward only to the contained sinks that understand that shape: raw frames
reach the recorder (which understands them) and are silently skipped for the
player and tone-out (which don't).

> ⚠ That raw-frame forwarding is not optional politeness. Before it existed
> (issue #356), the digital chains' `c.sink.(rawFrameSink)` assertion failed
> against a `fanoutSink`, so every IMBE/AMBE frame was dropped before reaching
> disk — while the activity counter still ticked, producing healthy-looking call
> logs next to 0-byte `.raw` files and 44-byte header-only WAVs. The daemon glue
> must implement every shape the chains might assert, or the fence and the frames
> both go silently inert.

The other adapter, `playerSink`, wraps the host audio player and adds a
per-talkgroup mute check before writing to the speakers — a call for a muted
talkgroup is still recorded and streamed, just not played aloud:

```go
// cmd/gophertrunk/daemon.go (shape)
type playerSink struct {
    p      *player.Player
    engine *trunking.Engine
}
func (s playerSink) WritePCM(serial string, samples []int16) error {
    if s.engine != nil {
        if tg := s.engine.TalkgroupForDevice(serial); tg != nil && tg.Mute {
            return nil // muted for the speakers only
        }
    }
    return s.p.WritePCM(serial, samples)
}
```

So for our 3 p.m. dispatch: if it's a P25 call, the composer's Phase 1 chain
demodulates the IQ into IMBE frames and calls `WriteRawFrameForCall`. The daemon's
`fanoutSink` forwards each frame to the recorder, which decodes it once, writes
PCM to talkgroup 101's WAV, and simultaneously feeds the browser stream and the
host player. If it were an analog Motorola call instead, the composer would have
FM-demodulated to PCM itself and called `WritePCM` — same fan-out, no vocoder.

## Where this goes next

[Part 3]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})
answers a question this post skipped: audio crosses the seam frame by frame, but
what decides where one *call* — one continuous recording — begins and ends? That's
the "composition" in the series title: how a burst of overs separated by silence
becomes one file (or many), how hangtime and talkgroup gating draw the
boundaries, and how a transmission boundary becomes the `KindCallSegment` event
that rolls the recorder to a new file.

## FAQ

**Why does the composer define `PCMSink` instead of importing `voice.Recorder`?**
Because the interface belongs to the consumer. The composer needs exactly one
method — `WritePCM(serial, samples)` — so it declares that and accepts anything
satisfying it. This keeps the composer free of a hard dependency on the recorder,
avoids an import cycle, and lets tests drive the demod pipeline with an in-memory
sink instead of a real recorder.

**What's the difference between `WritePCM` and `WriteRawFrame`?**
`WritePCM` takes finished PCM and appends it to the WAV — analog FM chains produce
audio directly and use it. `WriteRawFrame` takes an undecoded vocoder frame
(IMBE/AMBE) from a digital chain; the recorder runs the vocoder to turn it into
PCM. Analog calls never touch `WriteRawFrame`; digital calls never call `WritePCM`
from the composer.

**Why is the recorder the "lone decoder" rather than decoding in the composer?**
So a digital call is decoded exactly once. `writeRawFrame` runs the vocoder a
single time and fans the resulting PCM to both the WAV and every live listener. If
the composer decoded instead, the live `WritePCM`-only sinks would never see
digital audio (they don't handle raw frames) — the exact silence bug #598 fixed by
centralising decode in the recorder.

**What does `fanoutSink` do that a plain slice couldn't?**
It implements every sink shape the composer's chains might assert for —
`WritePCM`, `WritePCMForCall`, and the three raw-frame methods — and forwards each
call only to the contained sinks that understand it. A missing method there
doesn't fail loudly; the chain's type assertion just falls back and silently drops
frames, which is how issue #356 produced empty recordings.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The Output Half]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})
· Next →
[Part 3: Assembling a Call]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})
