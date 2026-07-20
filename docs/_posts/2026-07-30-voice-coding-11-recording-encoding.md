---
title: "Voice Coding, Part 11: Recording & Encoding — WAV, MP3, Tone-Out & Hardware"
description: How GopherTrunk writes crash-safe WAVs, encodes MP3 in pure Go with a Xing/LAME header, detects paging tones with Goertzel, and offers an optional DVSI hardware vocoder behind a build tag.
category: deep-dives
keywords: pure go mp3 encoder, shine encoder, xing lame header, wav writer, goertzel tone detection, two-tone paging, quick call ii, dvsi hardware vocoder, ambe patent, gophertrunk voice coding
tags: [recorder, mp3, wav, toneout, dvsi, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 11
---

*Part 11 of **Voice Coding**. The PCM is decoded and conditioned; now it has to
land somewhere useful — a file that survives a crash, a stream a broadcast
aggregator will accept, a paging-tone alert, and (for operators who need it) a
hardware vocoder. This is the output stage.*

> **TL;DR:** The recorder writes **16-bit mono WAV** whose length fields are
> patched on close, so a daemon crash leaves a playable file. For streaming it
> encodes **MP3 in pure Go** (the fixed-point Shine encoder), keeping the
> zero-CGO single-binary guarantee — plus a hand-built **Xing/LAME "Info"
> header** so demuxers parse short clips. A **Goertzel** detector sniffs paging
> tones off the same PCM. And an optional **DVSI hardware vocoder** lives behind
> a `dvsi` build tag: default builds ship only the patent-surface-free packet
> framing.

**Key takeaways**

- **WAV is crash-safe by construction:** the RIFF and data length fields are
  written as zero placeholders and patched in `Close()`, so an interrupted
  recording is still readable.
- **MP3 is pure Go** via Shine, but Shine's fixed 128 kbps default is *illegal*
  at 8 kHz — GopherTrunk overrides the bitrate per sample-rate family and
  prepends a Xing/Info header so ffmpeg doesn't probe past EOF.
- **Tone-out uses Goertzel**, not an FFT — one cheap single-bin detector per
  target frequency, block-aligned, emitting `KindToneAlert`.
- **DVSI is opt-in and patent-scoped:** the hardware `Vocoder`, USB transport,
  and registration compile only under `-tags dvsi`; the default build carries
  just the wire-protocol framing.

## Cheat sheet

| Output | File / package | Notes |
|---|---|---|
| WAV | `voice/wav.go` | 16-bit mono; length patched on `Close` |
| MP3 | `voice/mp3/mp3.go` | Shine (pure Go); bitrate per rate family |
| Xing/Info frame | `voice/mp3/xing.go` | LAME-style CBR header for short clips |
| Tone-out | `voice/toneout/` | Goertzel; Quick Call II / single / DTMF |
| DVSI | `voice/dvsi/` (`-tags dvsi`) | AMBE-3003 over FTDI USB |

## In this post

- **WAV** — the crash-safe header trick.
- **MP3** — pure-Go encoding and the two bugs Shine hides at 8 kHz.
- **Tone-out** — Goertzel paging-tone detection off the PCM tap.
- **DVSI** — the hardware fallback, its build tag, and the patent line.

## WAV: readable even after a crash

A WAV file's RIFF header states the total file size and the `data` chunk states
the payload size — but you don't know either until you've finished writing. The
naive approach buffers everything or leaves the sizes wrong. GopherTrunk writes
**zero placeholders** up front and patches them in `Close()`:

```go
// internal/voice/wav.go (shape)
func (w *WavWriter) Close() error {
    w.closed = true
    return w.patchHeader() // seek to offset 4 (RIFF size) + 40 (data size), fill in
}
```

The payoff is stated in the file's own doc: *a daemon crash leaves a readable
(if length-zero) file behind rather than something most media players reject.*
Even if `Close` never runs, the header is structurally valid — and the reader
side (`ReadWAVSamples`) walks the chunk list tolerantly, clamping a truncated
final chunk to what's actually on disk. A recording is never all-or-nothing.

## MP3: pure Go, and the 8 kHz trap

Broadcast aggregators (Broadcastify Calls, RdioScanner, OpenMHz, Icecast) want
MP3, not WAV. GopherTrunk encodes it **in pure Go** by wrapping the fixed-point
Shine encoder — no `libmp3lame`, no `libshine`, so the daemon keeps its
zero-CGO single-binary guarantee. But digital-radio voice records at 8 kHz, and
Shine has two failure modes there that only surface at low sample rates.

First, **bitrate**. Shine hard-codes 128 kbps, which is not even a legal Layer-III
bitrate for the MPEG-2.5 family (8/11.025/12 kHz) — it writes an out-of-range
bitrate index that corrupts every frame header. Worse, Shine's single-granule
frames desync their bit-reservoir stuffing past ~3300 bits/frame, emitting
oversized frames no demuxer can follow. *This is why an 8 kHz call uploaded to
RdioScanner transcoded to silence.* GopherTrunk picks a legal bitrate per family:

```go
// internal/voice/mp3/mp3.go (shape)
func BitrateFor(sampleRate int) int {
    switch {
    case sampleRate >= 32000: return 128000 // MPEG-1: two granules/frame
    case sampleRate >= 16000: return 64000  // MPEG-2
    default:                  return 32000  // MPEG-2.5 (incl. 8 kHz)
    }
}
```

Second, a **mono stride bug**: Shine's `Write` advances its read cursor by two
frames' worth of samples per call (correct for interleaved stereo, wrong for
mono — it silently drops every other frame). GopherTrunk drives `Write` one
frame at a time so the stride stays correct and no audio is lost.

Then there's the demuxer problem. Shine emits bare Layer-III frames with no
header frame. On a *short* clip (one radio call), ffmpeg scores the format
low-confidence and tries to confirm by seeking to a computed next-frame offset —
which, for the final frame, lands past EOF and is fatal ("Invalid data found").
LAME always writes an **Info** frame first precisely so demuxers read the length
instead of probing. GopherTrunk reproduces that frame in pure Go:

```go
// internal/voice/mp3/xing.go (shape)
// A silent CBR "Info" frame (vs "Xing" for VBR) advertising frames + bytes,
// reusing Shine's own 4-byte header so version/rate/mode always match.
func withInfoHeader(stream []byte) []byte
```

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A bare Shine MPEG stream of audio frames gets a silent Xing Info header frame prepended, advertising the total frame and byte counts so a demuxer reads the length instead of probing past end of file">
  <rect x="20" y="40" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="60" text-anchor="middle" fill="var(--accent)" font-size="12">Info frame</text>
  <text x="95" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="10">silent · frames+bytes</text>
  <rect x="176" y="40" width="96" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="224" y="66" text-anchor="middle" fill="currentColor" font-size="11">frame 1</text>
  <rect x="278" y="40" width="96" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="326" y="66" text-anchor="middle" fill="currentColor" font-size="11">frame 2</text>
  <rect x="380" y="40" width="60" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="410" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="11">…</text>
  <rect x="446" y="40" width="96" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="494" y="66" text-anchor="middle" fill="currentColor" font-size="11">frame N</text>
  <line x1="95" y1="86" x2="95" y2="112" stroke="var(--accent)"/>
  <polygon points="91,112 95,122 99,112" fill="var(--accent)"/>
  <text x="300" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the Info frame tells the demuxer the stream length — no probe past EOF on the last frame</text>
  <line x1="542" y1="63" x2="590" y2="63" stroke="currentColor"/>
  <polygon points="590,59 600,63 590,67" fill="currentColor"/>
  <text x="636" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="10">aggregator</text>
  <text x="636" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="10">upload</text>
</svg>
<figcaption>The prepended Info frame decodes to a blip of silence and carries the total frame/byte count, so short-clip demuxers read the length instead of seeking past the end of the file.</figcaption>
</figure>

## Tone-out: Goertzel, not FFT

Fire/EMS dispatch still uses audible paging tones — Motorola Quick Call II
(Two-Tone Sequential), single-tone, and DTMF. Detecting them doesn't need a
full spectrum; each profile cares about one or two specific frequencies, so
GopherTrunk uses the [Goertzel algorithm]({{ '/reference/goertzel-algorithm/' | relative_url }}) —
a single-bin power detector materially cheaper than an FFT:

```go
// internal/voice/toneout/goertzel.go (shape)
func (g *Goertzel) Process(sample int16) (float64, bool) {
    x := float64(sample) / 32768.0
    s0 := x + g.coeff*g.s1 - g.s2
    g.s2, g.s1 = g.s1, s0
    if g.count++; g.count < g.blockSize { return 0, false }
    mag2 := g.s1*g.s1 + g.s2*g.s2 - g.coeff*g.s1*g.s2
    g.Reset()
    return mag2 * g.normalize * 4, true // block-aligned magnitude
}
```

The `Detector` satisfies the same `composer.PCMSink` shape the recorder does
(`WritePCM(serial, samples)`), so the daemon simply fans the decoded PCM into it
alongside the recorder — no second decode. It keeps per-device state so
concurrent calls on different SDRs don't cross-contaminate match progress, runs
one Goertzel per unique target frequency across all profiles, and emits
`events.KindToneAlert` when a profile's tone sequence matches (default block
800 samples = 100 ms at 8 kHz). This is the [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }})'s
event bus again — a tone alert is just another published event a subscriber can
act on.

## DVSI: the hardware fallback

Everything so far is pure Go. But AMBE+2 decode is patent-encumbered in some
jurisdictions, and some operators are required to use a vendor-blessed decoder.
For them, GopherTrunk offers the **DVSI USB-3000 / AMBE-3003** hardware vocoder —
and scopes it carefully with a build tag.

```go
// internal/voice/dvsi/dvsi_enabled.go   //go:build dvsi   — Vocoder, Transport, USB, init()
// internal/voice/dvsi/dvsi_disabled.go  //go:build !dvsi  — nothing but the doc note
```

Under `-tags dvsi`, the package registers `"dvsi"` in the vocoder registry
pointing at a real AMBE-3003 chip (FTDI FT2232H, VID `0x0403` / PID `0x6010`);
if no matching USB device enumerates, `Open` returns `ErrNoDevice` and the
recorder falls back to the operator's configured pure-Go vocoder. Default builds
(`go build`, `go test`) compile **only** the patent-surface-free packet framing —
the `VocoderName` constant, the AMBE-3003 wire protocol, and the docs — so
nothing pulls in the DVSI codepath unless an operator opts in.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="The default build ships the pure-Go AMBE+2 decoder and DVSI packet framing only, while the dvsi build tag additionally links the hardware vocoder, USB transport, and registry registration">
  <rect x="20" y="24" width="280" height="128" rx="8" fill="none" stroke="currentColor"/>
  <text x="160" y="44" text-anchor="middle" fill="currentColor" font-size="12">default build</text>
  <rect x="40" y="56" width="240" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="160" y="75" text-anchor="middle" fill="var(--accent)" font-size="11">pure-Go ambe2 decoder</text>
  <rect x="40" y="92" width="240" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="160" y="111" text-anchor="middle" fill="var(--fg-muted)" font-size="11">DVSI packet framing (no decode)</text>
  <text x="160" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="10">patent-surface-free</text>
  <rect x="330" y="24" width="310" height="128" rx="8" fill="none" stroke="var(--accent)" stroke-dasharray="5 3"/>
  <text x="485" y="44" text-anchor="middle" fill="var(--accent)" font-size="12">-tags dvsi adds</text>
  <rect x="350" y="56" width="270" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="485" y="73" text-anchor="middle" fill="currentColor" font-size="11">hardware Vocoder + USB transport</text>
  <rect x="350" y="88" width="270" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="485" y="105" text-anchor="middle" fill="currentColor" font-size="11">registry: "dvsi" → Open(AMBE-3003)</text>
  <text x="485" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no chip → ErrNoDevice → fall back to pure-Go</text>
</svg>
<figcaption>The default binary never links the hardware decode path. The build tag adds the vocoder, transport, and registration; a missing chip degrades gracefully back to the pure-Go decoder.</figcaption>
</figure>

Both backends share the same `FrameBytes = 7` contract and the same
`voice.Vocoder` interface, so selecting DVSI is a name in the recorder's
protocol→vocoder map (Part 9) — the composer and recorder are unchanged. The
pure-Go path produces real audio under the license posture documented in
`docs/vocoders.md`; the DVSI path exists to outsource the patent surface to
silicon where policy demands it.

## Where this goes next

Output done, the series closes on trust. [Part 12]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }})
covers how GopherTrunk *proves* the pure-Go vocoders are correct — the
`voice-calibrate` harness, golden-frame regression, and cross-correlation
against reference decoders like DSD-FME and OP25.

## FAQ

**Why does GopherTrunk write its own MP3 encoder?**
To keep the zero-CGO single-binary guarantee — no `libmp3lame`/`libshine` at
build or runtime. It wraps the pure-Go fixed-point Shine encoder and fixes
Shine's low-sample-rate bugs (illegal 128 kbps default, mono frame-stride,
missing Xing header) so 8 kHz voice encodes to a file aggregators accept.

**What is the Xing/Info header for?**
Short MP3 clips (single radio calls) trip ffmpeg's demuxer, which probes past
EOF looking for the next frame and fails. A LAME-style Info frame advertises the
stream's frame/byte count so the demuxer reads the length instead of probing.
GopherTrunk prepends one in pure Go.

**Why Goertzel instead of an FFT for tone detection?**
Because each paging profile only cares about one or two frequencies. A Goertzel
single-bin detector is much cheaper than a full FFT and runs directly on the
PCM tap the recorder already produces, block-aligned, emitting `KindToneAlert`
on a match.

**Do I need special hardware to decode AMBE+2?**
No — the default pure-Go decoder produces real audio. The DVSI hardware backend
is an opt-in (`-tags dvsi`) for operators in jurisdictions that require a
vendor-blessed decoder; without the chip the recorder falls back to the pure-Go
vocoder automatically.

## Series navigation

**Part 11 of 12** · ←
[Part 10: Enhancement & Loudness]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
· Next →
[Part 12: Calibrating & Testing Vocoders]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }})
