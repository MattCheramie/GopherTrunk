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

## Cheat sheet

| | Bug one | Bug two |
|---|---|---|
| Symptom | activity detected, `recorder: call started wav=…`, directory stays empty | same empty directory, now with `composer: StreamIQ failed … rtlsdr: stream already active` on every call |
| Reachable when | always — every conventional call | only after bug one was fixed and the FM chain finally ran |
| Cause | composer's chain selection gated on decoding a digital protocol sync; `fm-conv` has nothing to decode | `runFMChain` opened a second `StreamIQ` on the device the scanner was already monitoring — single-consumer API, same device, same frequency |
| Why it never bit before | trunked digital systems always decode a sync | trunked voice records on a *separate retuned voice SDR*; conventional scanning has no monitor/record separation |
| Fix | `fm-conv` skips the digital gate and builds an FM chain directly | scanner becomes the broker's primary consumer; the FM chain **subscribes** to the IQ fan-out ([#1089](https://github.com/MattCheramie/GopherTrunk/issues/1089)) |
| Key tell | named WAV path + empty directory = no PCM ever arrived | `stream already active` on a device nothing else uses = self-contention |

## In this post

- **The symptom: activity without audio** — and the one log line that rules out the filesystem.
- **Bug one: the gate that only opened for digital** — a condition `fm-conv` could never meet.
- **Bug two: "stream already active"** — the collision that was unreachable until bug one died, and the topology that guarantees it.
- **The fix: one consumer, many subscribers** — the IQ fan-out broker.
- **Inside the broker** — bounded subscribers, zero-copy primary, and surviving USB reacquire.
- **The verification round** — WAV files, intelligible audio, and a new unrelated ticket.
- **What we keep** — the durable lessons.

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

No. Fixing bug one produced a new failure, 100% reproducible. The reporter came
back with a follow-up so precise it read like the postmortem's first draft:

```
synthetic call started  device=SDRV4-03 grant="scanner/fm-conv tg=2147483648 src=0 freq=146670000"
composer: StreamIQ failed  serial=SDRV4-03 err="rtlsdr: stream already active"
recorder: call started  device=SDRV4-03 wav=/opt/gophertrunk/recordings/scanner/2147483648/20260811_110603_2147483648.wav
synthetic call ended  device=SDRV4-03 reason=normal
```

Same named WAV path, same empty directory — but now with the cause printed one
line above it, on every call, on both configured channels. And the reporter had
already ruled out the obvious external explanation: the device was dedicated to
this scanner, with nothing else streaming from it. This was self-contention.

The freshly-unblocked FM chain (`runFMChain`) did what every voice chain had always
done: it opened its own IQ stream on the device it needed. And the device said no,
because the scanner already held the stream — it was, after all, *monitoring that
channel for activity*; continuous IQ from the device is how a squelch opening gets
detected and a "synthetic call" gets started at all. The device API is
single-consumer: one `StreamIQ`, one reader, by design — a second caller would
conflict at the USB/driver layer.

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

The reporter's working theory in that follow-up named all of this — the monitor
stream, the second `StreamIQ`, the trunked-versus-conventional separation — and
asked the exact right question: should the FM chain consume the scanner's existing
stream, or should the monitor yield the device for the duration of the call? The
answer was the first, via a piece of machinery the codebase already had.

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

## Inside the broker

The broker (`internal/sdr/iqtap`) predates this bug — it already fanned IQ out to
trunking-adjacent observers like the live spectrum view, paging decoders, the
rtl_tcp server, and signal-domain diagnostics. The #1089 fix promoted it from
"optional observers on the side" to the ownership answer for conventional
scanning. Its design constraints are worth spelling out, because each one is a
failure mode it prevents:

- **The primary keeps its contract.** The broker wraps the device and exposes the
  same interface; the primary consumer calls `StreamIQ` and gets one channel,
  exactly as before, with a single extra goroutine hop. Nothing about the scanner
  had to change to sit behind it.
- **Subscribers are bounded and non-blocking.** Each `Subscribe` gets its own
  bounded channel (default 16 chunks — about 60 ms at 2.048 MS/s), and delivery
  never blocks: a slow subscriber's chunks are dropped and *counted*, not queued
  behind. A wedged FM chain can never back-pressure the squelch monitor that
  detects the next call.
- **The primary path stays zero-copy; subscribers get copies.** Secondaries
  receive freshly-allocated slices they can hold or mutate without corrupting the
  primary or each other — the classic shared-buffer aliasing bug is designed out
  rather than policed.
- **Subscriptions survive device replacement.** A subscriber channel stays alive
  across `StreamIQ` sessions, and after a USB disconnect the daemon points the
  broker at the reacquired handle (`SetInner`) — the FM chain doesn't need to know
  the physical device bounced mid-call.
- **A small handoff buffer protects the driver.** A two-chunk handoff between the
  fan-out goroutine and the primary keeps a momentarily-busy consumer from
  stalling the drain of the driver's bounded ring — which would otherwise force
  whole-chunk IQ drops at the USB layer.

Answering the reporter's architectural question directly: `runFMChain` now
consumes the scanner's existing stream via a broker subscription. The alternative
— having the monitor release the device for each call and reacquire after — would
have deafened the scanner to every *other* channel for the duration of every call.

## The verification round

The collision reproduced faithfully in a unit test (a fake single-consumer device
that rejects a second stream; the fix delivers IQ to both consumers concurrently),
but a green synthetic test isn't proof of on-air audio — so the issue stayed open
until the reporter ran the fix against the same two channels, 146.670 MHz NFM and
442.100 MHz NFM with CTCSS:

```
synthetic call started  device=SDRV4-03 grant="scanner/fm-conv tg=2147483648 src=0 freq=146670000"
recorder: call started  device=SDRV4-03 wav=…/20260813_095829_2147483648.wav
synthetic call ended    device=SDRV4-03 reason=normal
recorder: call ended    device=SDRV4-03 wav=…/20260813_095829_2147483648.wav duration=6.844s reason=normal
broadcast: call streamed  backend=iaxs01-Rdio system=scanner tg=2147483648 attempt=1
```

No `StreamIQ failed`, a real file at the named path, intelligible audio on
playback, and the call flowing on into the broadcast backend. The same test round
also surfaced a squelch-hangtime debounce problem — recordings occasionally
running past the end of a transmission — which the reporter filed as its own
ticket rather than folding into this one: unrelated mechanism, separate issue,
exactly how a tracker stays navigable.

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

## FAQ

**Why did the log say a WAV was started if no file was ever created?**
`recorder: call started wav=<path>` names the path the recorder *intends* to use;
the file is only created when the first PCM arrives, so that calls which never
produce audio don't litter the disk with empty WAVs. The pairing of a named path
with an empty directory is therefore diagnostic gold: it localizes the failure to
"upstream of the recorder" in one glance.

**Nothing else was using the device — how could its stream be "already active"?**
Because the scanner itself was the other user. Conventional scanning requires a
continuous IQ stream just to detect a squelch opening, so by the time a call
started, the monitor already owned the device's single `StreamIQ`. The collision
was GopherTrunk contending with itself — which is why it was 100% reproducible
rather than a race.

**Why not just have the monitor release the device while a call records?**
On a multi-channel conventional setup, releasing the stream for the duration of a
call would blind the scanner to every other channel until the call ended — turning
one active channel into a blackout of the rest. The broker keeps the monitor
running and gives the voice chain a copy, so detection and recording proceed
concurrently on one physical stream.

**Does the broker add risk to the monitoring path it sits in front of?**
It's designed not to: the primary path is zero-copy with a single goroutine hop,
subscribers get bounded channels with non-blocking delivery, and a slow or wedged
subscriber shows up as a drop counter rather than back-pressure. The scanner reads
exactly the stream it always read.

**Was the two-bug structure avoidable?**
Only by testing the conventional path end to end before shipping it — the second
bug was unreachable while the first blocked every FM chain, so no amount of
staring at bug one's fix would have revealed it. The durable habit is to treat a
*new* error after a fix as the next layer speaking up, and budget for it.

## Series navigation

**Part 16 of 22** · ←
[Part 15: The Silent MP3 — Three Encoder Bugs and a Test That Checked One Frame]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }})
· Next →
[Part 17: Placeholder Constants — The TETRA Sync That Never Existed]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
