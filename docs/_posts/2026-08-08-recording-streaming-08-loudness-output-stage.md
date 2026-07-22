---
title: "Recording, Composition & Streaming, Part 8: Loudness at the Output Stage"
description: Where and when GopherTrunk applies loudness leveling to a finished call — an atomic per-call rewrite of the on-disk WAV, an independent in-memory gain on the distributed MP3 copy, and an optional real-time AGC on the live browser stream — each a separate config switch, none re-deriving the BS.1770 math.
category: deep-dives
keywords: ebu r128 loudness normalization placement, bs.1770 per call leveling, atomic wav rewrite temp rename, in-memory mp3 gain broadcast, live loudness agc stream parity, apply_to recording distributed both, gophertrunk normalize config, output-stage audio leveling
tags: [loudness, normalization, recording, streaming, go, audio]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 8
---

*Part 8 of **Recording, Composition & Streaming**. Our 3 p.m. dispatch on
talkgroup 101 has been decoded, written to a crash-safe WAV, named, and
guarded against cross-call bleed (Parts 4–7). Before the recorder announces
the finished file to the uploader, one optional step can level its loudness so
it matches every other call in the feed. This post is about **placement**: not
how the loudness measurement works — the [Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }})
series owns that — but exactly **where** and **when** a gain is applied, and why
GopherTrunk offers three independent places to apply it.*

> **TL;DR:** Loudness leveling in GopherTrunk lives at **three separate seams**,
> each its own config switch. `AppliesToRecording()` rewrites the **on-disk WAV**
> in place — atomically, temp-file-plus-rename — so every downstream consumer
> inherits normalized audio. `AppliesToDistributed()` applies the gain **in
> memory** while encoding the outbound MP3, leaving the WAV pristine. And the
> optional `audio.live_loudness` sink runs a real-time envelope AGC on the
> **live browser stream** so what you hear now tracks the normalized files. The
> BS.1770/R128 math itself is a black box called `loudness.NormalizeGain` — this
> post is only about which of those three seams you enable.

**Key takeaways**

- **Leveling is placement, not math.** GopherTrunk calls one shared function,
  `loudness.NormalizeGain`, from three different sites. Choosing a site — WAV,
  MP3, live stream — is the entire configuration surface an operator touches.
- **The on-disk rewrite is atomic.** `rewriteWAV` writes a `.tmp` next to the
  file and `os.Rename`s it over the original, so a crash mid-normalize can never
  truncate a good recording.
- **The distributed MP3 gain is non-destructive.** `encodeNormalizedMP3` levels
  the samples in memory and encodes; the WAV on `AudioPath` is never touched. A
  call can therefore ship loud to Broadcastify while staying faithful on disk.
- **The live AGC is an approximation, by necessity.** R128 *integrated*
  loudness needs the whole call; the live stream doesn't have the whole call
  yet. So `liveLoudnessSink` runs an envelope-follower AGC that matches
  *perceived* loudness in real time rather than reproducing the offline result.

## Cheat sheet

| Seam | Enabled by | Applied where | Function | Touches the WAV? |
|---|---|---|---|---|
| On-disk WAV rewrite | `NormalizeConfig.AppliesToRecording()` | recorder, after finalize | `normalizeWAVFile` → `rewriteWAV` | Yes — atomic temp+rename |
| Distributed MP3 gain | `NormalizeConfig.AppliesToDistributed()` | broadcast Manager, at encode | `encodeNormalizedMP3` | No — in memory only |
| Live-stream loudness | `audio.live_loudness` | daemon, decoded-PCM tap | `newLiveLoudnessSink` (`liveLoudnessSink`) | No — real-time AGC |
| Voice enhancement | `recordings.enhance` | recorder, at **decode** | `SetVoiceEnhance` on the vocoder | (decode stage, not output) |
| The gain calculation | (shared) | all three call it | `loudness.NormalizeGain` | — |

## In this post

- **Three seams, one gain function** — the shape of the leveling surface.
- **The on-disk rewrite** — where it sits in the recorder, and why it's atomic.
- **The distributed copy** — an independent switch that leaves the WAV alone.
- **The live AGC** — keeping the browser stream matched to normalized files.
- **What isn't here** — the BS.1770 math and the decode-stage enhance chain.

## Three seams, one gain function

Every leveling site in GopherTrunk ends up calling the same function. It lives
in `internal/dsp/loudness` and the [Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
post documents its internals; from the output half's point of view it is a black
box with one signature:

```go
// internal/dsp/loudness (shape) — documented in Voice Coding Part 10
func NormalizeGain(samples []float64, sampleRate int, p NormalizeParams) (gain float64, ok bool)
```

It measures a whole call's integrated loudness, returns the single linear gain
that moves it toward the target (true-peak limited), and reports `ok == false`
when the audio is too short or quiet to measure. **Pure gain, no compression** —
the within-call dynamics the vocoder AGC produced are preserved. The knobs are
carried by `NormalizeConfig` (target LUFS, true-peak ceiling, max boost); the
`WithDefaults` path fills any zero field with the package defaults (−16 LUFS,
−1.5 dBTP, ±12 dB).

The interesting part is that there are **three** callers, and which ones fire is
decided entirely by config. The YAML `NormalizeConfig` carries an `ApplyTo`
string, and two predicate methods read it:

```go
// internal/config/config.go (shape)
func (n NormalizeConfig) AppliesToRecording() bool {
    return n.Enabled && (n.ApplyTo == "" || n.ApplyTo == "recording" || n.ApplyTo == "both")
}
func (n NormalizeConfig) AppliesToDistributed() bool {
    return n.Enabled && (n.ApplyTo == "distributed" || n.ApplyTo == "both")
}
```

`apply_to: recording` (the default when normalize is enabled) rewrites the WAV.
`apply_to: distributed` leaves the WAV pristine and levels only the outbound
MP3. `apply_to: both` does both. These are **not mutually exclusive settings on
a dial** — they are two independent booleans, and the third seam (the live AGC)
is a completely separate config key. An operator can archive faithful WAVs while
shipping loud MP3s, or normalize the archive while keeping distribution
verbatim, or level everything, or nothing.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="One finished call fans into three independent loudness switches: AppliesToRecording rewrites the on-disk WAV atomically, AppliesToDistributed applies an in-memory gain to the outbound MP3 while leaving the WAV pristine, and audio.live_loudness runs a real-time AGC on the live browser stream. All three call the same loudness.NormalizeGain function.">
  <rect x="8" y="90" width="120" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="68" y="110" text-anchor="middle" fill="var(--accent)" font-size="11">finished call</text>
  <text x="68" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WAV on AudioPath</text>
  <line x1="128" y1="100" x2="176" y2="34" stroke="currentColor"/><polygon points="172,30 182,32 176,41" fill="currentColor"/>
  <line x1="128" y1="113" x2="176" y2="110" stroke="currentColor"/><polygon points="176,106 186,110 176,114" fill="currentColor"/>
  <line x1="128" y1="126" x2="176" y2="186" stroke="currentColor"/><polygon points="176,178 184,188 172,188" fill="currentColor"/>
  <rect x="186" y="16" width="210" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="291" y="33" text-anchor="middle" fill="var(--accent)" font-size="10">AppliesToRecording()</text>
  <text x="291" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="8">rewrite WAV — atomic temp+rename</text>
  <rect x="186" y="90" width="210" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="291" y="107" text-anchor="middle" fill="currentColor" font-size="10">AppliesToDistributed()</text>
  <text x="291" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">in-memory gain — WAV pristine</text>
  <rect x="186" y="166" width="210" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="291" y="183" text-anchor="middle" fill="currentColor" font-size="10">audio.live_loudness</text>
  <text x="291" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="8">real-time AGC — live stream</text>
  <line x1="396" y1="36" x2="470" y2="105" stroke="var(--fg-muted)"/><polygon points="466,99 474,109 461,107" fill="var(--fg-muted)"/>
  <line x1="396" y1="110" x2="470" y2="110" stroke="var(--fg-muted)"/><polygon points="470,106 480,110 470,114" fill="var(--fg-muted)"/>
  <line x1="396" y1="186" x2="470" y2="116" stroke="var(--fg-muted)"/><polygon points="466,113 474,111 470,122" fill="var(--fg-muted)"/>
  <rect x="480" y="92" width="170" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="565" y="108" text-anchor="middle" fill="var(--accent)" font-size="10">loudness.NormalizeGain</text>
  <text x="565" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shared — Voice Coding Part 10</text>
</svg>
<figcaption>Three independent config switches, one shared gain calculation. Enabling one seam says nothing about the others; the WAV, the MP3, and the live stream are leveled — or not — on their own.</figcaption>
</figure>

## The on-disk rewrite

The first seam lives inside the recorder, at the exact point a call becomes a
finished file. When `finalizeLocked` closes a session's WAV and returns a
`*CallComplete` (the subject of [Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})),
the recorder does one thing before it publishes the completion event:

```go
// internal/voice/recorder.go (shape)
cc := r.finalizeLocked(s, ce.DeviceSerial, ce.EndedAt, ce.Reason)
r.mu.Unlock()
if cc != nil {
    r.normalizeIfEnabled(cc.AudioPath) // rewrite the WAV before anyone sees it
    r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
}
```

Two placement decisions are baked into those three lines. First, normalization
runs **after `r.mu` is released** — whole-file measure-and-rewrite I/O must not
block other recording sessions, so `normalizeIfEnabled` is deliberately outside
the recorder's lock. Second, it runs **before the `KindCallComplete` publish**,
so by the time the uploader, the web player, or the MP3 encoder ever touches the
file, the audio is already at target. There is no window in which a downstream
consumer reads the un-normalized version.

`normalizeIfEnabled` is a thin guard — it checks the switch and never lets a
failure drop a call:

```go
// internal/voice/recorder.go (shape)
func (r *Recorder) normalizeIfEnabled(wavPath string) {
    if !r.normalize.Enabled || wavPath == "" {
        return
    }
    if err := normalizeWAVFile(wavPath, r.normalize); err != nil {
        r.log.Warn("recorder: loudness normalize failed", "wav", wavPath, "err", err)
    }
}
```

`normalizeWAVFile` reads the samples, converts to float, asks
`loudness.NormalizeGain` for the gain, applies it, and — critically — calls
`rewriteWAV` to put the result back. It returns `nil` without touching the file
when the gain isn't measurable (silent or too short). The rewrite is where the
crash-safety lives:

```go
// internal/voice/normalize.go (shape)
func rewriteWAV(path string, samples []int16, sampleRate uint32) error {
    tmp := path + ".tmp"
    w, err := NewWavFile(tmp, sampleRate)
    // …WriteSamples, Close…
    return os.Rename(tmp, path) // atomic replace over the original
}
```

Write to a sibling temp file, close it, then `os.Rename` it over the original.
On any POSIX filesystem the rename is atomic, so a crash mid-write leaves either
the old good WAV or the new good WAV — never a truncated one. This is the same
discipline the crash-safe WAV writer used in [Part 5]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }}),
applied a second time to the rewrite.

Because this seam edits the file every consumer reads, its effect is **global**:
web playback, the MP3 encode, and every upload backend all inherit the
normalized audio for free. That is exactly why the default `apply_to` is
`recording` — one rewrite, and the whole output half is consistent.

There is also an exported entry point, `NormalizeWAVFile`, that the offline
tooling (`gophertrunk decode -normalize`) calls; it wraps the same internal path
with `withDefaults()` applied first, since the recorder backfills defaults at
construction but a one-shot CLI invocation has not.

## The distributed copy

The second seam lives in a completely different package —
`internal/broadcast` — and it never writes to disk. When the broadcast Manager
turns a `CallComplete` into an outbound `Call`, it stamps the call with its own
`NormalizeConfig`. The MP3 is encoded lazily, at most once, the first time any
backend asks for it:

```go
// internal/broadcast/broadcast.go (shape)
func (c *Call) MP3() ([]byte, error) {
    // …once-only guard…
    if c.normalize.Enabled {
        c.mp3Data, c.mp3Err = encodeNormalizedMP3(c.AudioPath, c.normalize.Params)
    } else {
        c.mp3Data, _, c.mp3Err = mp3.EncodeWAVFile(c.AudioPath)
    }
    return c.mp3Data, c.mp3Err
}
```

`encodeNormalizedMP3` reads the WAV, converts to float, applies
`loudness.NormalizeGain` **in memory**, and hands the leveled samples to the MP3
encoder. The comment on the function says it plainly: it applies the gain
"leaving the file untouched." The on-disk WAV on `AudioPath` stays exactly as
the recorder wrote it. The gain lives only in the byte slice that becomes the
MP3.

This is what makes the two switches genuinely independent. `apply_to:
distributed` gives you a faithful archive and a loud feed. It exists because the
two audiences want different things: an operator's local archive should be
byte-faithful to what came off the air, while Broadcastify listeners want every
call at a consistent volume regardless of how hot the transmitter was.

The wiring that connects config to this seam is in `cmd/gophertrunk/broadcast.go`.
`buildBroadcastManager` takes the daemon's `config.NormalizeConfig` and, only
when `AppliesToDistributed()` is true, translates it into the broadcast
package's own `NormalizeConfig` with defaults resolved:

```go
// cmd/gophertrunk/broadcast.go (shape)
var normalize broadcast.NormalizeConfig
if normCfg.AppliesToDistributed() {
    normalize = broadcast.NormalizeConfig{
        Enabled: true,
        Params:  loudness.NormalizeParams{ /* target, true-peak, max-boost */ }.WithDefaults(),
    }
}
return broadcast.NewManager(broadcast.Options{ /* …, */ Normalize: normalize })
```

If the operator chose `apply_to: recording`, `AppliesToDistributed()` is false,
`normalize` stays its zero value, and the Manager encodes the WAV verbatim —
because the WAV was *already* normalized on disk by the first seam. The two
predicates make sure the gain is applied exactly once for any given `apply_to`
choice, never twice.

## The live AGC

The third seam is the odd one out, and its constraints are worth stating
because they explain why it can't just reuse the offline path. The live browser
stream needs to sound as loud as the normalized recordings **while the call is
still in progress** — but R128 *integrated* loudness is a whole-call
measurement, and mid-call there is no whole call to measure. So the live path
uses a different tool: a real-time envelope-follower AGC.

The daemon builds it only when `audio.live_loudness` is set, wrapping the
decoded-PCM tap that feeds the network stream:

```go
// cmd/gophertrunk/daemon.go (shape)
var liveStreamSink composer.PCMSink = d.audioPub
if cfg.Audio.LiveLoudness {
    lls := newLiveLoudnessSink(d.audioPub, 8000, d.bus, log)
    d.liveLoudness = lls
    liveStreamSink = lls // AGC now sits in front of the publisher
}
```

`liveLoudnessSink` is a `composer.PCMSink` decorator: it levels each block of
decoded digital PCM with a per-serial `dsp.AudioAGC`, then forwards to the real
publisher. It subscribes to the bus so it can `Reset()` a serial's envelope at
`KindCallStart` (a loud prior call shouldn't under-amplify the next call's
onset) and drop the entry at `KindCallEnd`. Deliberately, it wraps **only the
digital decoded tap** — the on-disk WAV is untouched (the recorder does its own
per-call R128), and analog FM live audio is untouched (the composer already
shapes its loudness upstream), so nothing is double-processed.

The result isn't bit-identical to the offline rewrite, and the code says so: the
envelope AGC "is the real-time approximation that matches what the listener
actually perceives." Perceptual parity, not numerical identity, is the goal — a
live listener toggling between the stream and a just-recorded file shouldn't
reach for the volume knob.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="A timeline of one call showing where each leveling seam acts: the live-loudness AGC levels the decoded PCM in real time as the call streams; the recorder finalizes and, if AppliesToRecording, rewrites the WAV atomically before publishing CallComplete; and only afterward, on demand, the broadcast Manager applies its in-memory gain while encoding the MP3 if AppliesToDistributed.">
  <line x1="20" y1="40" x2="640" y2="40" stroke="currentColor"/><polygon points="640,36 650,40 640,44" fill="currentColor"/>
  <text x="20" y="26" fill="var(--fg-muted)" font-size="9">call in progress</text>
  <text x="330" y="26" fill="var(--fg-muted)" font-size="9">call ends</text>
  <text x="560" y="26" fill="var(--fg-muted)" font-size="9">on upload</text>
  <line x1="330" y1="30" x2="330" y2="50" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <rect x="24" y="60" width="280" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="164" y="77" text-anchor="middle" fill="currentColor" font-size="10">live-loudness AGC (real time)</text>
  <text x="164" y="89" text-anchor="middle" fill="var(--fg-muted)" font-size="8">levels decoded PCM as it streams</text>
  <rect x="340" y="60" width="180" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="430" y="77" text-anchor="middle" fill="var(--accent)" font-size="10">finalizeLocked</text>
  <text x="430" y="89" text-anchor="middle" fill="var(--fg-muted)" font-size="8">close WAV → *CallComplete</text>
  <rect x="340" y="108" width="180" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="430" y="125" text-anchor="middle" fill="var(--accent)" font-size="10">normalizeIfEnabled</text>
  <text x="430" y="137" text-anchor="middle" fill="var(--fg-muted)" font-size="8">atomic WAV rewrite (recording)</text>
  <rect x="340" y="156" width="180" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="430" y="175" text-anchor="middle" fill="currentColor" font-size="10">publish CallComplete</text>
  <line x1="430" y1="94" x2="430" y2="108" stroke="var(--accent)"/><polygon points="426,108 430,116 434,108" fill="var(--accent)"/>
  <line x1="430" y1="142" x2="430" y2="156" stroke="currentColor"/><polygon points="426,156 430,164 434,156" fill="currentColor"/>
  <rect x="536" y="108" width="112" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="592" y="125" text-anchor="middle" fill="currentColor" font-size="9">Call.MP3()</text>
  <text x="592" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="8">in-memory gain</text>
  <text x="592" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="8">(distributed)</text>
  <line x1="520" y1="171" x2="580" y2="150" stroke="var(--fg-muted)" stroke-dasharray="3 3"/><polygon points="576,145 584,149 573,153" fill="var(--fg-muted)"/>
</svg>
<figcaption>The three seams act at different times. The live AGC runs while the call streams; the WAV rewrite sits between <code>finalizeLocked</code> and the <code>CallComplete</code> publish; the MP3 gain runs only later, on demand, when a backend asks for the encoded copy.</figcaption>
</figure>

## What isn't here

Two things this post deliberately does **not** cover.

The **BS.1770/R128 measurement and the AGC control law** belong to
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }}).
`loudness.NormalizeGain` and `dsp.AudioAGC` are black boxes here; how they gate,
window, and true-peak-limit is the codec series' subject. This post is only about
which of the three call sites you switch on.

The **voice enhancement chain** (`recordings.enhance`, toggled live by
`SetVoiceEnhance`) is a related but distinct feature that lives at a different
stage. Enhancement — band-limit, warmth shelf, louder AGC — is installed on the
per-call vocoder at **decode** time, so it shapes the WAV and the live fan-out
identically because both come from that one decode. Loudness normalization, by
contrast, is an **output-stage** operation applied to already-decoded audio. If
you find yourself asking "does this shape the sound before or after the file
exists?", enhancement is before (at decode) and the three seams above are after
(at output). They compose cleanly precisely because they sit at different stages.

## Where this goes next

[Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
opens the event those first two seams bracket: `CallComplete`. We'll see exactly
how `finalizeLocked` builds it, what identity and metadata it carries, and how it
becomes the single decoupling seam between recording and everything downstream —
the point where the loudness-normalized WAV we just produced is handed off to the
rest of the world.

## FAQ

**Does enabling normalization overwrite my original recordings?**
Only if `apply_to` includes `recording` (the default when normalize is on). That
path rewrites the WAV in place — atomically, via a temp file and rename, so a
crash never truncates it. Choose `apply_to: distributed` to keep the on-disk WAV
byte-faithful and level only the outbound MP3.

**Can I keep faithful archives but still send loud audio to Broadcastify?**
Yes — that's exactly `apply_to: distributed`. The recorder leaves the WAV
untouched, and `encodeNormalizedMP3` applies the gain in memory while encoding
the outbound copy, so the archive and the feed diverge on purpose.

**Why doesn't the live stream sound bit-identical to the recorded file?**
Integrated R128 loudness needs the whole call, which the live path doesn't have
yet. The `audio.live_loudness` sink uses a real-time envelope AGC instead — a
perceptual approximation that matches what a listener hears, not a reproduction
of the offline measurement.

**Where in the pipeline is loudness applied?**
For the WAV, between `finalizeLocked` and the `CallComplete` publish, so every
downstream reader sees normalized audio. For the MP3, later and on demand, inside
`Call.MP3()` when a backend requests the encoded bytes. For the live stream,
continuously, as decoded PCM passes through the sink on its way to the publisher.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Correctness Guards]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }})
· Next →
[Part 9: CallComplete — The Seam to Everything Downstream]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
