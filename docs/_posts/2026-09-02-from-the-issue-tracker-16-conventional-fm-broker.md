---
title: "From the Issue Tracker, Part 16: The Channel That Was Its Own Voice Channel — Conventional FM and the IQ Broker"
description: Conventional analog FM channels lit up with activity and recorded nothing. Fixing the composer's digital-only gate revealed a second bug hiding behind it — in conventional scanning the monitored channel is the recorded channel, and a single-consumer device API can't serve both.
category: solution-postmortem
keywords: conventional fm, analog scanning, composer, audio chain, stream already active, iq fan-out broker, wav path empty directory, single-consumer device, gophertrunk issue tracker
tags: [from-the-issue-tracker, analog, fm, scanner, architecture, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 16
---

*Part 16 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 15]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }})
was three bugs stacked inside one encoder. This one is two bugs stacked inside one
feature — where fixing the first was the only way to discover the second, and the
second was the more interesting of the pair.*

> **TL;DR:** In [#1075](https://github.com/MattCheramie/GopherTrunk/issues/1075),
> conventional analog FM channels detected activity, logged call starts, named WAV
> paths — and recorded nothing. Bug one: the composer refused to build *any* audio
> chain until it decoded a digital protocol sync, and `fm-conv` has nothing to
> decode by definition, so analog calls waited forever. Fixing that exposed bug
> two: the FM chain opened its own IQ stream on a device whose stream the scanner
> already held — `rtlsdr: stream already active`, 100% of the time — because in
> conventional scanning the monitored channel IS the recorded channel, on the same
> single-consumer device. The fix made the scanner the device's one true consumer
> behind an IQ fan-out broker everyone else subscribes to.

## The symptom: activity without audio

The reporter configured a handful of conventional analog FM channels — no trunking,
no digital protocol, just carrier-squelch voice — and the scanner did most of its
job convincingly. Channels showed activity when someone keyed up. The log announced
call starts. The recorder even printed the file it intended to write:

```
recorder: call started wav=/…/recordings/…/call-….wav
```

And the recordings directory stayed empty. Not short files, not corrupt files:
*no* files.

That log line deserves a paragraph, because it's the tell that routes the whole
investigation. `recorder: call started wav=<path>` announces an **intended** path.
The recorder defers file creation until the first PCM actually arrives — a
deliberate guard against writing empty WAVs for calls that never produce audio. So
"a named WAV path plus an empty directory" is not a filesystem problem, not a
permissions problem, not a disk-full problem. It means precisely: **no PCM ever
reached the recorder.** The hunt is upstream, in whatever was supposed to produce
audio.

One red herring cleared quickly: the reporter tried `mode: "nfm"` versus leaving
the mode empty. No difference — the config knob wasn't the lever.

## Bug one: the gate that only opened for digital

Upstream sat the composer, the component that assembles a call's audio chain when a
grant or channel activity arrives. Its chain selection had an unconditional gate:
before building anything, decode a digital protocol sync — P25, DMR, NXDN — and
pick the chain based on what answered.

For trunked digital systems that's exactly right. For `protocol: fm-conv` it's a
logical impossibility: a conventional analog FM channel has **nothing to decode by
definition**. There is no sync word, no frames, no protocol — just FM voice. So
every conventional call sat forever in "waiting to decode digital protocol," a
state it could not leave, while the recorder held an intended path and waited for
PCM that no chain existed to produce.

The fix for bug one is as plain as the bug: `fm-conv` skips the digital gate and
builds an FM demodulation chain directly. Ship it, watch the recordings appear?

## Bug two: "stream already active"

No. Fixing bug one produced a new failure, 100% reproducible:

```
rtlsdr: stream already active
```

The freshly-unblocked FM chain (`runFMChain`) did what every voice chain had always
done: it opened its own IQ stream on the device it needed. And the device said no,
because the scanner already held the stream — it was, after all, *monitoring that
channel for activity*. The device API is single-consumer: one `StreamIQ`, one
reader, by design.

Here's why this had never bitten anyone before, and it's the architectural heart of
the issue. On a trunked system, monitoring and recording are physically separate:
the scanner watches the control channel on one device, and when a call is granted,
a *voice* SDR retunes to a *different frequency* and opens its own stream. Monitor
device ≠ voice device; no contention, by construction.

Conventional scanning collapses that separation. The channel being monitored **is**
the channel being recorded — same frequency, same device, same stream. There is no
second device to hand the call to and no different frequency to retune to. The
single-consumer assumption wasn't wrong when it was written; it was wrong the day a
topology arrived in which monitor and consumer are the same physical stream.

| Topology | Who monitors | Who records | Contention? |
|---|---|---|---|
| Trunked digital | scanner, on the control channel | voice SDR, retuned to the grant | no — different device/frequency |
| Trunked analog voice | scanner, on the control channel | voice SDR, retuned | no — same reason |
| Conventional FM | scanner, on the channel | the same channel, same device | **yes — by construction** |

## The fix: one consumer, many subscribers

The resolution (landed as the follow-up,
[#1089](https://github.com/MattCheramie/GopherTrunk/issues/1089)) restructures who
owns the stream. The scanner becomes the device's single primary consumer, and in
front of the raw stream sits an **IQ fan-out broker**: the scanner feeds it, and
anyone else who needs that device's samples — the composer's FM chain included —
subscribes and receives copies. No second physical stream is ever opened; the
device API stays single-consumer; the broker turns one stream into as many logical
streams as there are interested parties.

That shape should look familiar if you've read the wideband posts: it's the same
move GopherTrunk makes when many DDC taps share one wideband dongle. The insight of
#1075 is that even a plain, narrowband, single-channel configuration needs it the
moment monitoring and consuming converge on one device.

## What we keep

- **"WAV path logged, directory empty" means no PCM ever arrived.** The recorder
  defers file creation until audio flows, so this specific pairing rules out the
  filesystem entirely and points upstream. Filed with its siblings in
  [audio pipeline tells]({{ '/reference/audio-pipeline-tells/' | relative_url }}).
- **A gate must have an answer for every protocol it can face.** "Decode a digital
  sync first" is unsatisfiable for a protocol with nothing to decode. Enumerate the
  cases; don't let one class of input wait forever on a condition it cannot meet.
- **Fixing bug one is sometimes the only way to find bug two.** The stream
  collision was unreachable while the composer's gate blocked every FM chain. Ship
  the first fix expecting the layer beneath it to speak up — the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  treats "new error after a fix" as progress, not regression.
- **Single-consumer APIs fail on topology, not on load.** The assumption held for
  years across every trunked configuration and failed instantly on the first
  topology where monitor and consumer are the same device at the same frequency.
  When you write "one consumer," ask which topology makes that consumer two.
- **Fan-out brokers are the general answer.** One physical stream, many
  subscribers, copies for everyone — the same pattern at 25 kHz that the wideband
  path uses at 10 MS/s.

## Series navigation

← [Part 15: the silent MP3]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }})
· Next → [Part 17: placeholder constants]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
