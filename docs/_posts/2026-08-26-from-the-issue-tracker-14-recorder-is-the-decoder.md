---
title: "From the Issue Tracker, Part 14: The Recorder Is the Decoder — Perfect Recordings, Silent Speakers"
description: Live audio in the web UI did nothing while recordings came out flawless. Two plausible fixes shipped before a 44-byte curl probe exposed the real cause — only the recorder ever decoded vocoder frames to PCM, so every live path was silently skipped.
category: solution-postmortem
keywords: live audio, web ui audio, vocoder frames, pcm, audio publisher, wav header, curl probe, recorder, p25 audio silent, gophertrunk issue tracker
tags: [from-the-issue-tracker, audio, web-ui, vocoder, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 14
---

*Part 14 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 13]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
untangled a network-SDR handshake. This one stays entirely inside the box: a user
clicks "tap to enable audio" in the web UI and nothing happens — while the recordings
directory fills up with perfectly good WAV files. The audio pipeline looked healthy
precisely because its one healthy path was the one nobody was listening to.*

> **TL;DR:** In [#598](https://github.com/MattCheramie/GopherTrunk/issues/598), live
> audio in the web UI was silent while recordings were flawless. Two plausible
> theories were fixed first — Safari's insistence on HTTP Range requests (the user
> was on Chrome) and a grant-cache race in the publisher (real, but not it). The
> actual cause: digital calls arrive at the audio stage as raw vocoder frames, and
> the **recorder was the only component that decoded them to PCM**. The web stream,
> host speaker, and tone-out detector all speak PCM only, so digital audio never
> reached them at all. A `curl` probe that hung at exactly 44 bytes — a WAV header
> and nothing after it — was the measurement that cracked it. The lasting lesson is
> a coupling that survived the fix: live digital audio rides the recording path.

## Cheat sheet

| | |
|---|---|
| Symptom | "Tap to enable audio" does nothing; recordings perfect; dashboard shows live calls |
| Wrong theory 1 | Safari needs HTTP Range/`206 Partial Content` — real bug class, fix shipped, user was on Chrome |
| Wrong theory 2 | Publisher grant-cache race dropping `CallStart` under load — real, fixed, symptom survived |
| The probe | `curl -sN …/api/v1/audio/stream \| head -c 4000 \| wc -c` hangs at **44 bytes** — WAV header, zero PCM |
| The counters | `stream_subscribers: 1`, `stream_tracked_grants: 1`, `stream_drops_total: 0` — a listener and a call, nothing delivered |
| Real cause | Digital calls arrive as raw vocoder frames; only the recorder decodes them to PCM; every live consumer is PCM-only and silently skipped |
| Fix | The recording path's decoded PCM is fanned out to the web stream, host speaker, and tone-out detector |
| Lasting coupling | Live digital audio rides the recording path — a no-record talkgroup has no live audio |

## In this post

- **The symptom: perfect recordings, dead speakers** — why "recordings work" misled everyone.
- **Wrong theory one: Safari and Range requests** — a true bug class on the wrong browser.
- **Wrong theory two: the grant-cache race** — a real race, a real fix, and the diagnostics that outlived it.
- **The 44-byte probe** — measuring the stream with no browser in the loop.
- **Two kinds of audio, one privileged consumer** — the vocoder-frame vs PCM topology underneath it all.
- **The real cause: only the recorder decodes** — how the one healthy path hid the four broken ones.
- **The fix — and the coupling that stays** — live audio as a beneficiary of the recording path.
- **What we keep** — the durable lessons.

## The symptom: perfect recordings, dead speakers

The report was clean and maddening. The web UI's "tap to enable audio" button did
what it always does — armed the browser's audio context — and then nothing ever
played. Calls were visibly happening: the dashboard showed grants, the call log
grew, and every one of those calls landed on disk as a correct, audible WAV file.

That last fact is what made the bug expensive. "Recordings work fine" reads as "the
audio pipeline is healthy," and everyone involved read it that way. The UI even
showed "Audio backend 8.0 kHz," which reads as a live status. It isn't — it's a
static echo of configuration, and it says nothing about whether a single frame of
audio is actually flowing. So the search started at the far end of the pipe, in the
browser, where the symptom was visible.

## Wrong theory one: Safari and Range requests

Streaming audio into browsers has a genuinely notorious failure mode: Safari
requires HTTP Range support and `206 Partial Content` responses for media elements,
and a server that answers `200` with the whole stream plays silence on Apple
devices. That is a real bug class, the fix is worth having, and it shipped: the
stream endpoint learned to answer Safari's two characteristic requests — a small
size probe (`Range: bytes=0-1`) served from the WAV header, and the open-ended
`bytes=N-` request that streams live PCM with a matching `Content-Range`. The
frontend player was even rewritten from a bare `<audio>` element to a fetch + Web
Audio pipeline in the same round, and playback failures started getting logged
instead of dying in a swallowed promise rejection.

It was also irrelevant. The reporter came back with the disproof already run: they
were on Chrome on macOS, and their own `curl -sD -` showed the new `206 Partial
Content` with `Accept-Ranges: bytes` working exactly as designed — during an
active call, with the tap pressed, and still no sound. A theory can be true in
general and still be the wrong theory for the ticket in front of you — the fix
changed nothing for the user who filed the issue.

## Wrong theory two: the grant-cache race

Theory two moved server-side: the audio publisher tracks active grants so it can
label and route stream audio, and there was a plausible race in that cache. The
publisher only emitted PCM for a call once it had cached that call's grant — the
talkgroup and system context arriving via a `CallStart` on an internal event
subscription — and under load that subscription can drop an event. Drop the
`CallStart` and every PCM chunk for that call was silently discarded. It even fit
the reporter's environment: they were running *several* voice taps, and more
simultaneous calls means more event traffic and better odds of losing the one
event the stream needed. Also real, also fixed — the publisher stopped depending
on winning that race and now streams as soon as a call is decoding. Also not it.
The symptom survived both patches intact.

The round did leave one thing behind that turned out to matter more than the fix:
`GET /api/v1/audio` grew diagnostic counters — `stream_subscribers`,
`stream_drops_total`, `stream_tracked_grants` — added so the next theory could be
*measured* instead of argued. Two shipped fixes with zero user-visible change is
usually the tracker's way of saying you're patching downstream of the actual
break; the counters were about to prove exactly where the break was.

## The 44-byte probe

What finally cracked it was refusing to debug through the browser at all. The live
stream endpoint serves WAV over HTTP, so you can measure it with nothing but `curl`:

```
curl -sN http://host:port/api/v1/audio/stream | head -c 4000 | wc -c
```

If audio is flowing, that returns 4000 almost immediately. On the reporter's box it
hung at **44 bytes** — the exact size of a WAV header. The server had accepted the
subscriber, written the header, and then never sent a single sample. Paired with the
publisher's own counters:

```
curl -s http://host:port/api/v1/audio | \
  jq '{stream_subscribers, stream_drops_total, stream_tracked_grants}'
```

which showed `stream_subscribers: 1`, `stream_tracked_grants: 1`,
`stream_drops_total: 0` — a listener connected, a call tracked, nothing dropped.
The publisher wasn't failing to deliver audio. It was never handed any. The
reporter's one-line summary of their own output was the whole diagnosis:
"Looks like problem is that stream actually never happens."

## Two kinds of audio, one privileged consumer

To see why nothing upstream had ever handed the publisher audio, you have to look
at what "audio" means at the seam where a call's voice chain meets its consumers —
because there are two different currencies in circulation.

**Analog audio is PCM from birth.** A conventional or trunked FM voice channel is
demodulated straight to PCM samples; anything downstream can play it as-is.

**Digital audio is vocoder frames until someone decodes it.** P25 Phase 1 and 2,
DMR, and NXDN carry voice as compressed vocoder frames — IMBE, AMBE+2 — and a
frame is not sound; it's an instruction set for a synthesizer. In the composer,
that difference is two distinct sink interfaces: every voice chain can write PCM
through the common `PCMSink` (`WritePCM`), but the digital chains hand their raw
frames through a second, narrower interface (`rawFrameSink`, `WriteRawFrame`) that
the chain discovers with a type assertion on the sink. Exactly one component
implemented the raw-frame side and owned a vocoder: the recorder, which decodes
frames to PCM as part of writing the WAV.

| Consumer | Input it understands | Digital calls before the fix |
|---|---|---|
| Recorder | vocoder frames (decodes itself) | works — files perfect |
| Web stream publisher | PCM | silently skipped |
| Host speaker | PCM | silently skipped |
| Tone-out detector | PCM | silently skipped |

That table is the entire bug. The digital voice path forked at the sink: raw
frames went to the one consumer that could use them, and the PCM-only consumers
were never in the delivery at all.

## The real cause: only the recorder decodes

That reframing pointed straight at the seam. Digital protocols — P25 Phase 1 and 2,
DMR, NXDN — don't arrive at the audio stage as PCM. They arrive as raw vocoder
frames, compressed voice that has to be run through a vocoder decode before anything
can play it. And in the code as it stood, those frames were delivered to exactly one
consumer: the recorder, which decoded vocoder → PCM → WAV as part of writing files.

Every other audio consumer — the web stream publisher, the host speaker player, the
tone-out detector — understands PCM only. When a digital call's frames came through,
those consumers weren't handed garbage; they were **silently skipped**. No error, no
log line, no counter. The recorder was on a privileged path and nobody knew it,
because analog FM produces PCM directly and so analog audio worked everywhere. That
is exactly why the bug wore a browser-codec costume: the one case people spot-check
casually (analog) worked, and the failing case (digital) failed only off-disk. It
even explains the reporter's multi-tap instinct — those taps were carrying P25
digital calls, which is precisely the broken path.

## The fix — and the coupling that stays

The fix moved the decoded PCM to where the consumers are: the vocoder output that
the recording path produces is now fanned out to the live consumers as well, so the
web stream, speaker, and tone-out detector all receive the same PCM the WAV file
does — decoded once, duplicated nowhere. Live audio and the recording are now
bit-identical by construction, because they are the same decode.

But note what that fix *is*: it makes live audio a beneficiary of the recording
path, not an independent one. The recorder is still the decoder. Which leaves a
coupling every operator should know about, because it looks like a bug and isn't:

- a talkgroup configured **not to record** has no live audio,
- recording toggled off globally means no live audio for digital calls,
- calls skipped as encrypted are torn down before decode — no live audio.

If you want to *listen* to a digital talkgroup, it has to be recordable. That's the
lasting shape of [#598](https://github.com/MattCheramie/GopherTrunk/issues/598):
the symptom is gone, the architecture that produced it is documented instead of
hidden.

## What we keep

- **"Recordings work" does not mean "the audio pipeline works."** It means one
  consumer works — and that consumer may be on a privileged path. Check what the
  *other* consumers are actually fed.
- **Probe the stream, not the player.** `curl -sN … | head -c 4000 | wc -c` hanging
  at 44 bytes says "WAV header, no PCM" in one line, with no browser in the loop.
  This and the other stalled-audio measurements live in the Field Guide under
  [audio pipeline tells]({{ '/reference/audio-pipeline-tells/' | relative_url }}).
- **Silent skips are the enemy.** A consumer that can't use an input should count
  the fact somewhere. Three consumers were skipped for years without a single
  observable trace.
- **Two correct fixes that change nothing are a signal.** You're downstream of the
  break. Stop patching and start measuring — the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  starts there.
- **Ship diagnostics with every failed fix.** The counters added alongside wrong
  theory two (`stream_subscribers`, `stream_drops_total`) are what made round
  three a measurement instead of a fourth guess.
- **Know the coupling:** live digital audio rides the recording path. No-record
  talkgroups have no live audio, by construction.

## FAQ

**Why were the recordings perfect if the audio pipeline was broken?**
Because the recorder was the one consumer that received vocoder frames *and* owned
a vocoder to decode them. Its path — frames in, PCM out, WAV on disk — was
complete and healthy. Every other consumer needed PCM it was never given. The
pipeline wasn't broken so much as it only had one working lane.

**Does the fix decode every call twice — once for the file, once for the stream?**
No. The vocoder runs once, on the recording path, and the resulting PCM is fanned
out to the live consumers. That's why live audio is identical to the recording —
and why live audio depends on the call being recordable at all.

**Why not give the live consumers their own vocoder instead of coupling them to
the recorder?**
It would decouple the paths at the cost of running every digital call through the
vocoder twice and maintaining two decode points that must agree. The coupling was
judged acceptable, made explicit, and documented — the failure mode changed from
"silent skip nobody knew about" to "known rule operators can reason about."

**How do I verify live audio is actually flowing on my instance?**
During an active call, `curl -sN http://host:port/api/v1/audio/stream | head -c
4000 | wc -c` should return 4000 quickly — hanging at 44 means header-only. Then
`curl -s http://host:port/api/v1/audio` should show `stream_subscribers >= 1` with
your tab open. Those two checks isolate server-side delivery from browser-side
playback in under a minute.

**I set a talkgroup not to record and its live audio stopped. Is that a bug?**
No — it's the documented coupling. Digital live audio is produced by the recording
path's decode, so a call that will never be recorded is never decoded. Make the
talkgroup recordable to hear it live.

## Series navigation

**Part 14 of 22** · ←
[Part 13: The SoapyRemote Handshake — Three Wrong Root Causes and a Server That Says Nothing First]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
· Next →
[Part 15: The Silent MP3 — Three Encoder Bugs and a Test That Checked One Frame]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }})
