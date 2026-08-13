---
title: "From the Issue Tracker, Part 15: The Silent MP3 — Three Encoder Bugs and a Test That Checked One Frame"
description: Rdio Scanner accepted every upload and played silence. Behind it sat three independent MP3 encoder bugs — an illegal bitrate corrupting headers via putBits(-1), bit-reservoir stuffing desync, and a mono stride that skipped half the audio — none caught by a test that only checked the first frame's sync word.
category: solution-postmortem
keywords: rdio scanner, mp3 encoder, shine, mpeg-2.5 bitrate, bit reservoir, xing header, mono stride, ffmpeg invalid data, upload silent audio, gophertrunk issue tracker
tags: [from-the-issue-tracker, audio, mp3, recording, integrations, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 15
---

*Part 15 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 14]({{ '/blog/solution-postmortem/from-the-issue-tracker-14-recorder-is-the-decoder/' | relative_url }})
found live audio starving because only the recorder decoded vocoder frames. This
one is the recorder's own output biting back: the WAV on disk is perfect, the MP3
made from it uploads successfully — and plays nothing.*

> **TL;DR:** In [#874](https://github.com/MattCheramie/GopherTrunk/issues/874),
> Rdio Scanner logged `newcall: success` for every GopherTrunk upload and stored
> silence. The MP3 encoder had **three independent bugs**: 128 kbps is not a legal
> Layer III bitrate for the MPEG-2.5 family that 8 kHz voice lands in, and the
> lookup's −1 sentinel was written into the header as four 1-bits, corrupting the
> version/layer fields of *every* frame; the bit-reservoir stuffing writer desynced
> on any frame budget over ~420 bytes; and mono input advanced a hard-coded stereo
> stride, silently discarding every other block of samples. A popular theory — the
> missing Xing header — was disproved with a ten-byte experiment before the real
> hunt started. The old unit test validated exactly one thing: the first frame's
> sync word. All three bugs sailed past it.

## Cheat sheet

| Bug | Mechanism | Who it hit | Why the old test missed it |
|---|---|---|---|
| Illegal bitrate → `putBits(uint32(-1), 4)` | 128 kbps has no MPEG-2.5 bitrate index; the −1 sentinel writes four 1-bits over the header, `ff e3 …` becomes `ff ff f8 c4` | every 8 kHz frame — the recorder's default voice rate | the test predated the 8 kHz configuration that triggers it |
| Bit-reservoir stuffing desync | `putBits(^uint32(0), 32)` stuffing loses sync once the frame budget exceeds ~420 bytes | 16 kHz at 128 kbps, silently, for years | desync accumulates in *later* frames; only frame one was checked |
| Mono stride | `Write` advances `samples_per_pass * 2` while encoding 576 per pass — every other block dropped | all mono input (i.e., everything the recorder produces) | framing stays valid; the audio is just missing |
| Not the bug: missing Xing header | folk-wisdom diagnosis for "plays in some players" | — | disproved in ten bytes before the hunt started |

## In this post

- **The symptom: success, then silence** — an upload pipeline that validates metadata, not media.
- **The theory that died in ten bytes** — retiring the Xing-header folk diagnosis cheaply.
- **Anatomy of an MP3 frame header** — the 32-bit layout, and how a demuxer walks frames by arithmetic.
- **Bug one: an illegal bitrate and `putBits(-1)`** — a sentinel written into the wire format.
- **Bug two: the reservoir desync above ~420 bytes** — a content-independent corruption nobody had hit yet.
- **Bug three: mono advances a stereo stride** — half the audio never encoded.
- **Why the test caught none of it** — one frame, one sync word, zero decodes.
- **The fix** — per-family bitrates, real mono, and a verification matrix.
- **What we keep** — the durable lessons.

## The symptom: success, then silence

The reporter wired GopherTrunk's call uploads into Rdio Scanner. Every call went
through: Rdio Scanner logged `newcall: success`, the call appeared with correct
metadata — talkgroup, timestamps, frequencies — and playback produced silence.

The source WAVs were fine. Trunk Recorder feeding the same Rdio Scanner instance
worked. So the finger pointed at the one artifact in between: the MP3 that
GopherTrunk transcodes for upload. `ffmpeg` 6.1.1 agreed, loudly:

```
[mp3 @ ...] Invalid data found when processing input
```

and, on other files, the more interesting pair:

```
Format mp3 detected only with low score of 1
Failed to read frame size: Could not seek to 42811
```

That seek target was past end-of-file. The demuxer wasn't complaining about a
missing index or metadata — it was walking frame headers, computing where the next
frame *should* start, and falling off the end of the file. That's a frame-level
corruption signature, and it's worth learning to read, because it immediately
demotes the most popular theory.

## The theory that died in ten bytes

The folk-wisdom diagnosis for "MP3 plays in some players but not others" is the
missing Xing/Info header — the metadata frame that VBR files carry so players can
seek. It's the first thing every search result suggests.

Ten bytes killed it. Prepending a minimal ID3v2 tag — which changes the file's
framing preamble the same way a metadata fix would — changed nothing about the
failure. The demuxer still mis-walked the frames. Whatever was wrong was wrong
*inside* the frames, not in front of them. (A Xing/Info header was still added
later, because it's correct to have one. It just wasn't the fix, and knowing that
early kept the investigation pointed at the frame encoder.)

## Anatomy of an MP3 frame header

To see why each bug produces its particular symptom, you need the 32 bits every
MP3 frame starts with. In order:

| Bits | Field | Notes |
|---|---|---|
| 11 | sync word | all ones |
| 2 | version | `11` = MPEG-1, `10` = MPEG-2, `00` = MPEG-2.5 |
| 2 | layer | `01` = Layer III |
| 1 | CRC protection | |
| 4 | **bitrate index** | table lookup; `1111` is explicitly invalid |
| 2 | sample-rate index | |
| 1 | padding | |
| 9 | private / channel mode / mode extension / copyright / original / emphasis | |

Two consequences matter here. First, the *legal bitrates are a per-version table*:
the 4-bit index means each MPEG version gets at most fourteen valid bitrates, and
the tables differ between the families. The version, in turn, is pinned by the
sample rate — 32/44.1/48 kHz is MPEG-1, 16/22.05/24 kHz is MPEG-2, and
**8/11.025/12 kHz is MPEG-2.5**, which is where scanner-voice WAVs at 8 kHz land.

Second, an MP3 file has no index: a demuxer finds frame *N+1* by arithmetic on
frame *N*'s header — for Layer III, frame length is derived from the bitrate and
sample rate those header fields declare (plus the padding bit). A corrupt header
therefore doesn't just break one frame; it breaks the *walk*. Compute a frame
length from garbage fields and the "next frame" lands mid-stream or — as in the
reporter's `Could not seek to 42811` — past the end of the file entirely. That is
the mechanical reading of the symptom: a seek-past-EOF from an MP3 demuxer means
a frame header lied about its own length.

## Bug one: an illegal bitrate and `putBits(-1)`

GopherTrunk's encoder is a Go port of Shine, the classic fixed-point MP3 encoder,
and it hard-coded 128 kbps. Here's the trap: MPEG audio's legal bitrate table
depends on which *family* the sample rate falls in, and 8 kHz voice — the bread and
butter of scanner audio — lands in MPEG-2.5 (8 / 11.025 / 12 kHz). **128 kbps is
not a legal Layer III bitrate in that family.**

So `findBitrateIndex` returned its not-found sentinel, −1. And then the header
writer did this with it:

```go
putBits(uint32(-1), 4)
```

`uint32(-1)` is all ones — `0xFFFFFFFF`. The header writer packs fields by ORing
bits into the output buffer, so those four 1-bits didn't just set the bitrate
index to the explicitly-invalid `1111`; they ORed back over the version and layer
fields that had already been written. A valid 8 kHz Layer III header starts
`ff e3` — eleven sync bits, version `00` (MPEG-2.5), layer `01` (Layer III). What
came out instead was `ff ff f8 c4`: a header claiming MPEG-1 with a *reserved*
layer code and an impossible length — garbage that occasionally still scores as
"maybe MP3" (hence `low score of 1`). Every single 8 kHz frame was malformed at
the source. Not corrupted in transit, not mis-tagged: born broken.

## Bug two: the reservoir desync above ~420 bytes

Fixing the bitrate exposed the second bug. At sample rates below 32 kHz, Layer III
uses single-granule frames, and Shine's port stuffed unused bit-reservoir space
with a loop of:

```go
putBits(^uint32(0), 32)
```

That stuffing writer desynced once the per-frame byte budget exceeded roughly 420
bytes — entirely content-independent, purely a function of bitrate versus sample
rate; the investigation reproduced it with tones, noise, quiet noise, and
speech-shaped signals alike, all emitting oversized frames no demuxer could
follow. The arithmetic says who was exposed: a single-granule frame carries 576
samples, so at 16 kHz a frame spans 36 ms, and 36 ms at 128 kbps is **576 bytes
per frame** — comfortably past the ~420-byte cliff. Which means it had been
silently corrupting a configuration nobody had complained about yet: 16 kHz audio
at the default 128 kbps. The 8 kHz case never got far enough to hit it (bug one
corrupted its frames first); the 16 kHz case hit it and nobody noticed, because…

## Bug three: mono advances a stereo stride

…the third bug was the reporter's own find, and it's the most elegant of the three.
`Encoder.Write` consumed input with a hard-coded stride:

```go
samples_per_pass * 2   // always two channels
```

while `samplesPerPass()` is *per channel* — 576 for MPEG-2.5. Feed it mono, and
each pass encodes 576 samples but advances the read pointer by 1152. Half the
audio — every other block — was simply never encoded. The reporter had even
discovered the accidental workaround before the root cause: duplicating the mono
channel into fake stereo made the stride arithmetic correct, at exactly twice the
file size. A workaround that works for a suspiciously clean reason is often the
best pointer to the actual bug.

## Why the test caught none of it

Three bugs, one encoder, years in the tree. The unit test that was supposed to
guard this code checked precisely one property: that the **first frame** of the
output began with a valid MP3 sync word.

| Bug | First frame's sync word |
|---|---|
| Illegal bitrate → `putBits(-1)` header corruption | corrupt — but the test predated the 8 kHz config it triggers on |
| Reservoir stuffing desync over ~420 bytes/frame | fine — desync accumulates in *later* frames |
| Mono stride skipping half the samples | fine — framing is valid, audio is just missing |

A decoder-free, single-frame, header-only check is a round-trip test degenerated to
its weakest form. The replacement tests decode the *entire* output and — critically
— do it both ways Rdio Scanner's stack does:

```
ffmpeg -i file.mp3 …          # seekable file path
cat file.mp3 | ffmpeg -i - …  # non-seekable stdin path
```

Rdio Scanner feeds ffmpeg over stdin, where the demuxer cannot seek and fails
differently than it does on a file. A file that passes one path can fail the other.

## The fix

The bitrate is now chosen per family — `BitrateFor` returns 32 kbps for MPEG-2.5,
64 kbps for MPEG-2, 128 kbps for MPEG-1 — so the index lookup cannot return −1 for
any rate the recorder produces. The policy is driven by the same arithmetic that
explains bug two: each single-granule family gets the highest standard bitrate
that keeps a frame comfortably under the stuffing writer's limit (32 kbps at
8 kHz is 288 bytes a frame), while the MPEG-1 rates keep their two-granule frames
and the 128 kbps default. Both chosen rates are well above transparent for 8 kHz
mono voice, and `configureBitrate` reprograms Shine's CBR bookkeeping — bitrate
index, slots per frame — to match, since the port derived those at construction
from the hard-coded default.

Mono is now driven through `Write` one frame at a time, so the stride is correct
by construction — true mono output, no dropped audio, and none of the dual-mono
size doubling of the workaround. A Xing/Info header is prepended in pure Go with
real frame and byte counts, matching what LAME and Trunk Recorder write, so short
clips probe cleanly. And the Icecast streaming path now paces output at the *real*
encoded bitrate instead of the old hard-coded assumption.

Verification matched the failure surface rather than the code: every one of the
nine supported sample rates, decoded end to end with the reporter's exact ffmpeg
version (6.1.1), through both the seekable file path and the non-seekable stdin
path, plus the full AAC/M4A transcode Rdio Scanner performs after ingest. A small
offline tool (`cmd/encodetest`) ships so any operator can run their own WAV
through the encoder and both ffmpeg paths without standing up an upload pipeline.
The reporter's confirmation closed the loop: uploads play, with good quality.

## What we keep

- **"Upload accepted" validates metadata, not media.** Rdio Scanner's
  `newcall: success` never looked inside the audio. The only acceptance test that
  counts is decoding the payload end to end.
- **A seek-past-EOF error is a frame-walk error.** `Could not seek to 42811` means
  the demuxer computed a next-frame offset from a corrupt header — look inside the
  frames, not at the container metadata. More of these mappings live in
  [audio pipeline tells]({{ '/reference/audio-pipeline-tells/' | relative_url }}).
- **Disprove the folk theory cheaply first.** Ten bytes of ID3v2 retired the Xing
  theory in minutes and saved the detour. The
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  calls this buying information at the lowest price.
- **Sentinels must not be writable.** `putBits(uint32(-1), 4)` is a type-system
  escape hatch; a not-found result that can flow into an encoder is a bug waiting
  for the right sample rate.
- **Test the whole output, on every input path.** First-frame checks catch nothing
  that accumulates; file-path checks catch nothing that only stdin exposes.

## FAQ

**Why is 128 kbps illegal at 8 kHz when it's the most common MP3 bitrate there is?**
Because the legal bitrates are a per-version table selected by a 4-bit index, and
the version is pinned by the sample rate. 8 kHz forces MPEG-2.5, whose Layer III
table has no 128 kbps entry — the lookup has nowhere to point. The familiar
128 kbps default belongs to the MPEG-1 rates (32/44.1/48 kHz) that music files
use; scanner voice lives in a different family with different rules.

**How did three bugs coexist in one encoder without anyone noticing?**
They shadowed each other. Bug one corrupted every 8 kHz frame, so 8 kHz output
never got far enough to exhibit bug two; bug two only bit at 16 kHz, a rate nobody
was uploading; and bug three removed audio without breaking framing, so nothing
mechanical ever flagged it. The single guard test checked one sync word in one
frame — a property all three bugs left intact in at least one configuration.

**Was the missing Xing header ever part of the problem?**
No. The ten-byte ID3v2 experiment showed the failure was frame-level, and the
fixed encoder's output decodes fine with or without it. One was still added —
with real frame and byte counts, matching LAME's — because probing tools score
short files more confidently with it. Correct to have; irrelevant to the bug.

**Why test through stdin as well as a file path?**
Because Rdio Scanner pipes uploads into ffmpeg over stdin, where the demuxer
cannot seek. A seekable demuxer can scan around damage that a streaming demuxer
must fail on, so the same file genuinely can pass `ffmpeg -i file` and fail
`cat file | ffmpeg -i -`. Verifying only the path you don't ship is how encoder
bugs live for years.

**What's the quickest way to check my own instance's MP3s?**
`go run ./cmd/encodetest /path/to/call.wav /tmp/test.mp3`, then decode
`/tmp/test.mp3` both ways shown above. If both decodes succeed at the right
duration, the encoder side is healthy and any remaining silence is downstream.

## Series navigation

**Part 15 of 22** · ←
[Part 14: The Recorder Is the Decoder — Perfect Recordings, Silent Speakers]({{ '/blog/solution-postmortem/from-the-issue-tracker-14-recorder-is-the-decoder/' | relative_url }})
· Next →
[Part 16: The Channel That Was Its Own Voice Channel — Conventional FM and the IQ Broker]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
