---
title: "Recording, Composition & Streaming, Part 5: WAV on Disk — Crash-Safety, the Tolerant Reader & Clean Cuts"
description: How GopherTrunk's recorder turns decoded PCM into a WAV that survives a crash — placeholder-then-patch RIFF headers, a chunk-walking reader that clamps a truncated final chunk, an end-of-call fade tail, and dead-key suppression that removes empty files.
category: deep-dives
keywords: wav file crash safety, riff header patch, tolerant wav reader, truncated chunk clamp, end of call fade, dead key suppression, empty recording removal, gophertrunk recorder, 16-bit pcm mono wav, sdr call recording
tags: [recording, wav, file-integrity, dsp, go, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 5
---

*Part 5 of **Recording, Composition & Streaming**, a 14-part deep dive into
GopherTrunk's output half. Parts 2–4 got a call's composed audio into a
[recording session]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
and started its WAV. This post is about the file itself: what the recorder does
so that our 3 p.m. dispatch on talkgroup 101 lands as a playable file even if the
daemon is killed mid-write — and what it decides NOT to keep. The
[Voice Coding]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }})
series already dissected the header-patch trick at the byte level; here we care
about the recorder's use of it and the finalize decisions layered on top.*

> **TL;DR:** `WavWriter` writes a 44-byte header with **zero placeholders** for
> the two length fields and patches them in `Close()`, so a crash leaves a file
> whose worst case is a readable, length-zero recording rather than one media
> players reject. `ReadWAVSamples` **walks the RIFF chunks** and clamps a
> truncated final `data` chunk to the bytes that actually exist, so even an
> un-patched file reads back. On finalize the recorder appends a short **linear
> fade** to zero (killing the end-of-call click), and **removes** a WAV that
> captured no real speech instead of leaving dead-key spam on disk.

**Key takeaways**

- The WAV is written **length-last**: two fields (RIFF size at byte 4, data size
  at byte 40) start as zeros and are back-patched on `Close()`. A crash before
  the patch costs the length fields, not the audio.
- The reader is **deliberately tolerant** — a chunk walk that accepts extra
  chunks before `data` and clamps a final chunk whose declared size runs past
  end-of-file, so an un-patched or truncated capture still yields samples.
- Finalize is where the recorder **shapes the cut**: `fadeTail` ramps the last
  decoded sample down to zero over ~10 ms so a short over doesn't end on a click.
- **Empty and dead-key calls are removed, not kept.** No PCM, or a vocoder that
  decoded zero voiced/unvoiced frames, means the WAV is deleted and no
  `CallComplete` is published — nothing to upload, no tiny-file spam.

## Cheat sheet

| Thing | What it does | Where in code |
|---|---|---|
| `WavWriter.writeHeader` | Emits 44 bytes with zeroed length fields | `internal/voice/wav.go` |
| `WavWriter.Close` → `patchHeader` | Seeks to byte 4 and byte 40, writes real lengths | `internal/voice/wav.go` |
| `WavWriter.DataBytes` | Payload byte count; readable after Close | `internal/voice/wav.go` |
| `ReadWAVSamples` | Chunk-walking, truncation-clamping tolerant reader | `internal/voice/wav.go` |
| `finalizeLocked` | Fade, empty check, CallComplete-or-nil decision | `internal/voice/recorder.go` |
| `fadeTail` | Linear ramp from last sample to zero | `internal/voice/recorder.go` |
| `removeSessionFiles` | Deletes WAV + `.raw` for a worthless call | `internal/voice/recorder.go` |

## In this post

- **Length-last WAV** — why the header is written with holes and patched on close.
- **The tolerant reader** — the chunk walk and the truncated-chunk clamp.
- **The fade tail** — smoothing an abrupt digital cut without touching the DSP.
- **Empty and dead-key removal** — the finalize decisions that keep the tree clean.
- **The audio-vs-wall-clock check** — a diagnostic that catches lost voice frames.

## Length-last, because you can't know the length yet

A canonical PCM WAV is a RIFF container: a 12-byte `RIFF … WAVE` preamble, a
`fmt ` chunk describing the format, and a `data` chunk holding the samples. Two
of those fields are byte counts that can't be known until the file is done — the
overall RIFF size and the `data` chunk size. A call is a live stream; the
recorder is appending samples for as long as the radio holds the channel. It has
no idea how long the file will be when it opens it.

`WavWriter` resolves this the standard way — **write the header with zeros in the
length slots, patch them when you close** — but the reason GopherTrunk leans on
it is durability. `writeHeader` lays down all 44 bytes up front with placeholders
for the two lengths:

```go
// internal/voice/wav.go (shape)
func (w *WavWriter) writeHeader() error {
    header := make([]byte, wavHeaderSize) // 44
    copy(header[0:4], "RIFF")
    // header[4:8]  RIFF size — patched in Close()
    copy(header[8:12], "WAVE")
    copy(header[12:16], "fmt ")
    binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
    binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM
    // …channels, sampleRate, byteRate, blockAlign, bitsPerSample…
    copy(header[36:40], "data")
    // header[40:44] data size — patched in Close()
    _, err := w.w.Write(header)
    return err
}
```

Every `WriteSamples` call appends little-endian `int16` payload and bumps
`bytesWritten`. Nothing seeks; the file grows monotonically. Only `Close()` goes
back and fills the holes:

```go
// internal/voice/wav.go (shape)
func (w *WavWriter) patchHeader() error {
    w.w.Seek(4, io.SeekStart)
    binary.Write(w.w, binary.LittleEndian, uint32(36+w.bytesWritten)) // RIFF size
    w.w.Seek(40, io.SeekStart)
    binary.Write(w.w, binary.LittleEndian, w.bytesWritten)            // data size
    w.w.Seek(0, io.SeekEnd)
    return nil
}
```

The payoff is the crash story. If the daemon is `SIGKILL`ed mid-call — OOM,
power loss, a panic elsewhere — the file on disk is a valid header (with zero
lengths) followed by however many samples were flushed. Byte 4 and byte 40 still
read zero, so a strict player sees a length-zero recording, but the container is
structurally intact and the samples are physically present. Contrast the naive
alternative, where you'd buffer the whole call and write the header once at the
end: a crash there loses *everything*. Length-last trades a strict-reader edge
case for never losing the audio you already captured. `TestWavHeaderShape` pins
the patched offsets — data size at `[40:44]`, RIFF size at `[4:8]` — and
`TestWavCloseTwice` confirms `Close()` is idempotent, which matters because the
recorder's session teardown can reach a writer more than once.

`Close()` is also nil-safe and closed-safe on purpose. A dormant session parked
between overs holds a nil writer; closing it must be a no-op, not a panic:

```go
// internal/voice/wav.go (shape)
func (w *WavWriter) Close() error {
    if w == nil || w.closed {
        return nil
    }
    w.closed = true
    if err := w.patchHeader(); err != nil {
        if w.closeFn != nil {
            _ = w.closeFn() // still release the fd; chain the error
        }
        return err
    }
    // …closeFn()…
}
```

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="Byte layout of the 44-byte WAV header. The RIFF chunk id occupies bytes 0 to 4, the RIFF size field at byte 4 is highlighted as patched-on-close, WAVE and fmt and format fields fill the middle, the data chunk id sits at byte 36, and the data size field at byte 40 is highlighted as patched-on-close, followed by the PCM payload starting at byte 44.">
  <rect x="8" y="40" width="70" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="43" y="60" text-anchor="middle" fill="currentColor" font-size="10">RIFF</text>
  <text x="43" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0..4</text>
  <rect x="78" y="40" width="80" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="118" y="57" text-anchor="middle" fill="var(--accent)" font-size="10">RIFF size</text>
  <text x="118" y="69" text-anchor="middle" fill="var(--accent)" font-size="8">byte 4 · patched</text>
  <rect x="158" y="40" width="60" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="188" y="60" text-anchor="middle" fill="currentColor" font-size="10">WAVE</text>
  <text x="188" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">8..12</text>
  <rect x="218" y="40" width="140" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="288" y="60" text-anchor="middle" fill="currentColor" font-size="10">fmt · PCM · rate</text>
  <text x="288" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">12..36</text>
  <rect x="358" y="40" width="66" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="391" y="60" text-anchor="middle" fill="currentColor" font-size="10">data</text>
  <text x="391" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">36..40</text>
  <rect x="424" y="40" width="82" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="465" y="57" text-anchor="middle" fill="var(--accent)" font-size="10">data size</text>
  <text x="465" y="69" text-anchor="middle" fill="var(--accent)" font-size="8">byte 40 · patched</text>
  <rect x="506" y="40" width="146" height="34" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="579" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="10">int16 PCM payload</text>
  <text x="579" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="8">byte 44 →</text>
  <text x="330" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">44-byte header — two fields written last</text>
  <text x="330" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">A crash before Close() leaves both accent fields at zero — the payload is still physically present.</text>
</svg>
<figcaption>The two highlighted fields are the only ones written after the audio. Everything else is fixed at open; a crash costs the lengths, never the samples.</figcaption>
</figure>

## The tolerant reader

Writing crash-safe files is only half the contract — something has to read them
back, including the ones a crash left un-patched. `ReadWAVSamples` is that reader,
and it is written to tolerate exactly the damage the writer's crash story
produces. GopherTrunk uses it in the normalize pass (measuring a finished WAV
before rewriting it) and anywhere a recording is re-opened, so its tolerance is
load-bearing, not decorative.

Two design choices make it robust. First, it **walks the RIFF chunks** rather
than assuming the canonical fixed layout. It steps chunk-by-chunk from byte 12,
reading each `id`+`size`, and only acts on `fmt ` and `data` — so an extra chunk
some other encoder slipped in before `data` is skipped, not fatal. Second, and
this is the one that matters for our crash: when a chunk's declared size runs
past the end of the file, it **clamps to what's actually present** instead of
indexing out of bounds:

```go
// internal/voice/wav.go (shape)
for pos := 12; pos+8 <= len(b); {
    id := string(b[pos : pos+4])
    size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
    body := pos + 8
    if size < 0 || body+size > len(b) {
        // Truncated final chunk (e.g. a crash before patchHeader):
        // clamp to the bytes that actually exist.
        size = len(b) - body
    }
    switch id {
    case "fmt ":
        // …parse format, channels, sampleRate, bits…
    case "data":
        dataStart, dataLen = body, size
    }
    pos = body + size + (size & 1) // chunks are word-aligned
}
```

That clamp is the reader's half of the length-last bargain. A crashed file has a
`data` size field of zero — but the *bytes are there*, past the header. Because
the walk clamps `size` to `len(b) - body`, the reader recovers every sample that
was flushed, regardless of what the (un-patched, zero) length field claims. The
`size < 0` guard catches a garbage length that overflows `int`; the `size & 1`
step keeps the walk word-aligned across padded chunks.

The tolerance has limits, and they are deliberate: after the walk it insists on
16-bit mono PCM (`channels == 1`, `bits == 16`, `format == 1`) — the recorder's
own output contract — and errors out otherwise. It will forgive a truncated file
it wrote; it will not silently misread a stereo or float WAV as if it were the
format it expects. Tolerant about damage, strict about type.

## Shaping the cut: the fade tail

Everything above is about *not losing* audio. `finalizeLocked` is where the
recorder decides how the audio *ends*. A decoded digital over that stops
mid-waveform leaves a non-zero final sample, and a hard jump from that value to
silence is a step discontinuity — an audible click or scratch on playback, most
obvious on short overs where the tail is a big fraction of the call.

The fix is a short linear ramp appended at finalize, the digital counterpart of
the squelch-tail fade the analog FM chain already applies in the composer (owned
by [Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})).
The recorder tracks `lastSample` — the most recent decoded PCM value — and on
close builds a ~10 ms ramp from it down to zero:

```go
// internal/voice/recorder.go (shape)
func fadeTail(from int16, n int) []int16 {
    if from == 0 || n <= 0 {
        return nil // already at silence, or nothing to ramp
    }
    out := make([]int16, n)
    start := float64(from)
    for i := range out {
        out[i] = int16(start * (1.0 - float64(i)/float64(n)))
    }
    return out
}
```

Finalize sizes `n` as `sampleRate/100` (≈10 ms, floored at 8 samples), writes the
ramp to the WAV, and — importantly — fans the same tail to the live decoded-PCM
sink so the browser stream ends as cleanly as the file does. The guards keep it
honest: it runs only for vocoder-decoded calls (`s.vocoder != nil`) with real
audio (`dataBytes > 0`), and a dead-key call has `lastSample == 0`, so `fadeTail`
returns `nil` and appends nothing. `TestRecorderFadesDigitalTailToZero` and
`TestFadeTail` pin the ramp's first sample to `from` (seamless continuation) and
its shape down to zero.

<figure class="lab-figure">
<svg viewBox="0 0 660 180" width="660" height="180" role="img" aria-label="Timeline of a short over. Decoded PCM samples run flat then stop abruptly at a non-zero level; the recorder appends a roughly ten millisecond linear fade ramp from that last sample value down to zero, so the file ends at silence instead of a step discontinuity that would click on playback.">
  <line x1="30" y1="150" x2="640" y2="150" stroke="var(--fg-muted)"/>
  <line x1="30" y1="30" x2="30" y2="150" stroke="var(--fg-muted)"/>
  <text x="20" y="34" text-anchor="end" fill="var(--fg-muted)" font-size="8">+</text>
  <text x="20" y="92" text-anchor="end" fill="var(--fg-muted)" font-size="8">0</text>
  <polyline points="30,92 70,70 110,105 150,72 190,108 230,74 270,104 310,70 350,100 390,78 430,96" fill="none" stroke="currentColor"/>
  <text x="230" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="9">decoded PCM (the over)</text>
  <circle cx="430" cy="96" r="3" fill="var(--accent)"/>
  <text x="430" y="88" text-anchor="middle" fill="var(--accent)" font-size="8">lastSample</text>
  <line x1="430" y1="96" x2="520" y2="92" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="475" y="120" text-anchor="middle" fill="var(--accent)" font-size="9">~10 ms fade → 0</text>
  <line x1="430" y1="150" x2="430" y2="60" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="520" y1="92" x2="640" y2="92" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="580" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">silence</text>
  <text x="335" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Without the ramp, the jump from lastSample to zero is a step discontinuity — an audible click.</text>
</svg>
<figcaption>The fade is appended at finalize, not computed in the DSP: the last decoded sample is ramped to zero over ~10 ms on both the WAV and the live stream.</figcaption>
</figure>

## What the recorder refuses to keep

The last set of finalize decisions is about *suppression* — the calls whose files
should never survive. Two cases produce no `CallComplete` and, in the worse case,
delete what was written.

**Empty files.** If `dataBytes == 0` — no PCM was ever decoded into the WAV —
there is nothing to stream. `finalizeLocked` returns `nil` (no `CallComplete`, so
no upload). A digital call keeps its `.raw` sidecar as the only capture; an analog
call simply produced nothing. Because the recorder opens files lazily on the
first write (covered in
[Part 4]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})),
a truly silent grant usually never creates a file at all —
`TestRecorderNoAudioLeavesNoFiles` and `TestRecorderNoAudioLeavesNoDir` confirm
it leaves not even an empty talkgroup directory.

**Dead-key / idle carriers.** Subtler is a call that *did* write PCM but is
worthless: a vocoder that decoded frames, all of which were idle, silent, or bad —
`Voiced + Unvoiced == 0`. The WAV holds only muted silence. In per-transmission
mode these are the dominant source of tiny-file spam. When the session's vocoder
implements `StatProvider` (the pure-Go IMBE decoder) and reports zero voiced and
unvoiced frames, `removeSessionFiles` deletes both the WAV and the `.raw`
sidecar, and finalize returns `nil`:

```go
// internal/voice/recorder.go (shape)
if haveStats && vs.Voiced+vs.Unvoiced == 0 {
    r.removeSessionFiles(s) // dead-key: delete WAV + .raw
    return nil              // no CallComplete
}
if dataBytes == 0 {
    return nil // empty: nothing to stream, keep any .raw
}
return &trunking.CallComplete{ /* Grant, AudioPath, SampleRate, … */ }
```

Alongside these, finalize runs a **diagnostic** that changes nothing on disk but
catches a real bug class: the audio-vs-wall-clock check. For a decoded call it
compares the WAV's audio seconds (`dataBytes / (sampleRate * 2)`) against the
call's wall-clock span. When the audio is under 75% of the wall time on a call
longer than a second, it logs at DEBUG — the self-evident "16 s call, 9 s
recording" symptom that means voice frames were lost upstream (talkgroup gating,
undecoded windows) rather than a genuinely short over. It's DEBUG, not WARN,
because a legitimate multi-over call drops the silent gaps between overs; this is
an investigation aid, not an alarm.

## Where this goes next

[Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
zooms out from one file to the filesystem: how a call maps to a
`<system>/<talkgroup>/<timestamp>_freq_src` path, how per-transmission recording
rolls each over into its own file, and why DMR, ProVoice, and TETRA always get a
`.raw` sidecar written alongside — the on-air frames that must survive even when
no playable WAV can be produced.

## FAQ

**What happens to a WAV if GopherTrunk crashes mid-call?**
The file on disk is a valid 44-byte header followed by every sample that was
flushed, with the two length fields (RIFF size at byte 4, data size at byte 40)
still zero because `Close()` never ran to patch them. GopherTrunk's own
`ReadWAVSamples` clamps the truncated `data` chunk and reads every recovered
sample; a strict third-party player may see a length-zero file but the audio is
physically intact.

**Why patch the header on close instead of writing it correctly up front?**
Because the recorder can't know the call's length until the call ends — it's a
live stream. Writing the header last (with zeroed length placeholders) means a
crash costs only the length fields, whereas buffering the whole call to write one
correct header would lose the entire recording on a crash.

**Why does a short over need a fade at the end?**
A decoded over that stops mid-waveform leaves a non-zero final sample; jumping
straight to silence is a step discontinuity that clicks on playback.
`fadeTail` appends a ~10 ms linear ramp from that last sample to zero, on both the
WAV and the live stream. A silent/dead-key call has a zero last sample, so no
ramp is added.

**Why does the recorder delete some recordings instead of keeping them?**
A call that decoded no voiced or unvoiced frames (a dead-key or idle carrier)
produced only muted silence — worthless audio and, in per-transmission mode, the
main source of tiny-file spam. The recorder removes the WAV and `.raw` and
publishes no `CallComplete`, so nothing downstream tries to upload an empty call.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The Recording Session]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
· Next →
[Part 6: Segmentation, Naming & Raw Sidecars]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
