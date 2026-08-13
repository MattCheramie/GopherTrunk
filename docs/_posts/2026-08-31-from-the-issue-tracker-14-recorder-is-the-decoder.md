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
devices. That is a real bug class, the fix is worth having, and it shipped.

It was also irrelevant. The reporter was on Chrome. A theory can be true in general
and still be the wrong theory for the ticket in front of you — the fix changed
nothing for the user who filed the issue.

## Wrong theory two: the grant-cache race

Theory two moved server-side: the audio publisher tracks active grants so it can
label and route stream audio, and there was a plausible race in that cache — a call
could start before the publisher had the grant, leaving audio unattributed. Also
real, also fixed, also not it. The symptom survived both patches intact.

Two shipped fixes with zero user-visible change is usually the tracker's way of
saying you're patching downstream of the actual break.

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
The publisher wasn't failing to deliver audio. It was never handed any.

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
casually (analog) worked, and the failing case (digital) failed only off-disk.

| Consumer | Input it understands | Digital calls before the fix |
|---|---|---|
| Recorder | vocoder frames (decodes itself) | works — files perfect |
| Web stream publisher | PCM | silently skipped |
| Host speaker | PCM | silently skipped |
| Tone-out detector | PCM | silently skipped |

## The fix — and the coupling that stays

The fix moved the decoded PCM to where the consumers are: the vocoder output that
the recording path produces is now fanned out to the live consumers as well, so the
web stream, speaker, and tone-out detector all receive the same PCM the WAV file
does.

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
- **Know the coupling:** live digital audio rides the recording path. No-record
  talkgroups have no live audio, by construction.

## Series navigation

← [Part 13: the SoapyRemote handshake]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
· Next → [Part 15: the silent MP3]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }})
