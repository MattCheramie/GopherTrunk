---
slug: audio-pipeline-tells
title: Audio-pipeline tells
entry_type: term
category: fn-diagnostics
description: "Audio-pipeline tells are the specific observations that localize a silent-audio fault in GopherTrunk: the 44-byte stream, the empty recording directory behind a logged WAV path, and MP3s whose bitrate is illegal for their sample rate."
keywords: audio, silent audio, wav header, 44 bytes, live stream, recorder, vocoder frames, pcm, mp3 bitrate, mpeg-2.5, rdio scanner, ffmpeg, curl probe
aka: [silent-audio diagnosis, 44-byte stream tell]
infobox:
  - { label: Type, value: Diagnostic catalog }
  - { label: Applies to, value: "Live audio, recordings, and uploads" }
  - { label: Key fact, value: The recorder is the only vocoder-frame decoder }
  - { label: Signature tell, value: A WAV stream that stalls at exactly 44 bytes }
see_also: [diagnostic-playbook, signal-signatures, vocoder, imbe, talkgroup, sample-rate]
related_reading:
  - { title: "From the Issue Tracker, Part 14: The Recorder Is the Decoder — Perfect Recordings, Silent Speakers", url: /blog/solution-postmortem/from-the-issue-tracker-14-recorder-is-the-decoder/ }
  - { title: "From the Issue Tracker, Part 15: The Silent MP3 — Three Encoder Bugs and a Test That Checked One Frame", url: /blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/ }
  - { title: "From the Issue Tracker, Part 16: The Channel That Was Its Own Voice Channel — Conventional FM and the IQ Broker", url: /blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/598
  - https://github.com/MattCheramie/GopherTrunk/issues/1075
  - https://github.com/MattCheramie/GopherTrunk/issues/874
---

**Audio-pipeline tells** are the handful of concrete observations that localize a
"no audio" fault in GopherTrunk to a specific stage. Silent audio is a uniquely
misleading symptom class: the decode chain can be perfectly healthy while the
audio never reaches the listener
([#598](https://github.com/MattCheramie/GopherTrunk/issues/598)), a recording can
be announced and never written
([#1075](https://github.com/MattCheramie/GopherTrunk/issues/1075)), and an
uploaded file can be accepted by the receiving system yet be unplayable
([#874](https://github.com/MattCheramie/GopherTrunk/issues/874)). Each tell below
distinguishes "the audio was never produced" from "the audio was produced and
lost."

| Symptom | Looks like | Actually | Check |
|---|---|---|---|
| Live audio silent, recordings fine | Browser/codec problem | The recorder is the only vocoder-frame decoder; live paths never got PCM | The two `curl` probes below |
| Audio stream stalls after exactly 44 bytes | Network or player issue | 44 bytes is the WAV header — no PCM ever followed it | `curl -sN …/audio/stream \| head -c 4000 \| wc -c` |
| `recorder: call started wav=…` logged, directory empty | Filesystem/permissions problem | The log names an *intended* path; the file is only created when PCM arrives | Look upstream for why no PCM was produced |
| Upload accepted, stored MP3 silent/rejected | Uploader or metadata bug | Malformed MP3 frames — e.g. a bitrate illegal for the MPEG family | Both `ffmpeg -i file` *and* `cat file \| ffmpeg -i -` |

## The 44-byte stream

A WAV stream begins with a 44-byte header. If
`curl -sN http://host:port/api/v1/audio/stream | head -c 4000 | wc -c` hangs at
**44**, the server sent the header and then no PCM at all — the publisher is
connected but starved. Pair it with the counters:

```
curl -s http://host:port/api/v1/audio | \
  jq '{stream_subscribers, stream_drops_total, stream_tracked_grants}'
```

In [#598](https://github.com/MattCheramie/GopherTrunk/issues/598) this showed
`stream_subscribers=1`, a tracked grant, and zero drops — "the publisher has a
listener and a call but was never handed audio" — which eliminated two earlier
misdiagnoses (a Safari range-request theory and a grant-cache race, both
plausible, both shipped, neither the cause). Note the UI's "Audio backend 8.0 kHz"
label is a static config echo and says nothing about whether frames are flowing.

## The recorder is the decoder

The root cause behind [#598](https://github.com/MattCheramie/GopherTrunk/issues/598):
digital protocols (P25, DMR, NXDN) arrive at the audio stage as raw
[vocoder](/reference/vocoder/) frames, and the **recorder was the only component
that decoded them to PCM**. The web stream, host-speaker player, and tone-out
detector all consume PCM and were silently skipped. Analog FM produces PCM
directly, so it worked — which is exactly what made the bug look like a browser
problem. Two durable consequences:

- "Recordings work fine" does **not** mean the audio pipeline is healthy; the
  recorder sat on a privileged path.
- Because live digital audio rides the recording path, a
  [talkgroup](/reference/talkgroup/) set not to record, recording toggled off, or
  a skipped encrypted call has **no live audio** by construction.

## A logged WAV path is a promise, not a file

`recorder: call started wav=<path>` announces an *intended* destination; the file
itself is created only when the recorder receives PCM. A named path plus an empty
directory therefore means "no PCM ever arrived" — not a filesystem or permissions
problem. In [#1075](https://github.com/MattCheramie/GopherTrunk/issues/1075) this
tell pointed past the recorder to two upstream bugs on conventional analog FM:
the composer gated every audio chain on decoding a *digital* protocol sync
(`fm-conv` has nothing to decode by definition), and fixing that exposed a second
fault — the FM chain opened its own IQ stream and collided with the scanner's
monitor stream on the same single-consumer device (`stream already active`),
because in conventional scanning the monitored channel and the recorded channel
are the same device at the same frequency.

## MP3 bitrate legality

The MP3 tell from [#874](https://github.com/MattCheramie/GopherTrunk/issues/874):
an upload target logging success while the stored audio is silent or rejected. The
primary cause was an **illegal bitrate for the MPEG family**. Layer III defines a
different bitrate table per family, and 128 kbps — the encoder's hard-coded
default — is not a legal rate for MPEG-2.5, the family that covers the 8 kHz
sample rate of voice recordings. The lookup returned −1 and writing that value
corrupted the frame headers of *every* frame at source. The resulting policy:

| MPEG family | Sample rates | Bitrate used |
|---|---|---|
| MPEG-2.5 | 8 / 11.025 / 12 kHz | 32 kbps |
| MPEG-2 | 16 / 22.05 / 24 kHz | 64 kbps |
| MPEG-1 | 32 / 44.1 / 48 kHz | 128 kbps |

Verification must exercise **both** ffmpeg input paths: `ffmpeg -i file` (seekable)
*and* `cat file | ffmpeg -i -` (stdin, non-seekable) — Rdio Scanner decodes via
the stdin path, which fails differently. Two further lessons from the same
thread: a "missing Xing header" theory was disproved by a small experiment
(prepending a tag changed nothing — the failure was frame-level), and the original
unit test validated only the *first* frame's sync word, so a mid-file
bit-reservoir desync shipped undetected.

## Provenance

- [#598](https://github.com/MattCheramie/GopherTrunk/issues/598) — silent live audio; the 44-byte probe, the publisher counters, and the recorder-is-the-decoder coupling.
- [#1075](https://github.com/MattCheramie/GopherTrunk/issues/1075) — conventional FM produced no audio; the intended-WAV-path tell and the monitor/record stream collision.
- [#874](https://github.com/MattCheramie/GopherTrunk/issues/874) — Rdio Scanner uploads stored silent MP3s; illegal MPEG-2.5 bitrate and the dual ffmpeg verification.
