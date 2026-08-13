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

`uint32(-1)` is all ones. Four 1-bits ORed into the header field *after* the
version and layer bits had been written turned a valid `ff e3 …` sync sequence into
`ff ff f8 c4` — garbage that occasionally still scores as "maybe MP3" (hence
`low score of 1`). Every single 8 kHz frame was malformed at the source. Not
corrupted in transit, not mis-tagged: born broken.

## Bug two: the reservoir desync above ~420 bytes

Fixing the bitrate exposed the second bug. At sample rates below 32 kHz, Layer III
uses single-granule frames, and Shine's port stuffed unused bit-reservoir space
with a loop of:

```go
putBits(^uint32(0), 32)
```

That stuffing writer desynced once the per-frame byte budget exceeded roughly 420
bytes — entirely content-independent, purely a function of bitrate versus sample
rate. Which means it had been silently corrupting a configuration nobody had
complained about yet: 16 kHz audio at 128 kbps. The 8 kHz case never got far enough
to hit it; the 16 kHz case hit it and nobody noticed, because…

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
any rate the recorder produces. The stuffing writer and the mono stride were fixed
outright, and the Icecast streaming path now paces output at the *real* encoded
bitrate instead of the old hard-coded assumption. Uploads play.

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

## Series navigation

← [Part 14: the recorder is the decoder]({{ '/blog/solution-postmortem/from-the-issue-tracker-14-recorder-is-the-decoder/' | relative_url }})
· Next → [Part 16: conventional FM and the IQ broker]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
