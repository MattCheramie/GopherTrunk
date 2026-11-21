---
title: "The Operator's Cookbook Vol. 2, Part 7: Transcription — Whisper on Every Call"
description: "A complete GopherTrunk transcription recipe: every finished recording is converted to 16 kHz mono WAV and posted to an OpenAI-compatible Whisper server, a local whisper.cpp on the same box or a hosted API, and the text is stored on the call, searchable in History, exported as CSV and usable as an alert keyword."
category: tutorials
keywords: scanner call transcription whisper, whisper.cpp scanner audio, speech to text police scanner, openai whisper radio transcription, faster-whisper scanner recordings, searchable scanner call log, transcribe p25 recordings, local whisper server sdr, keyword alerts scanner transcript, gophertrunk transcription
tags: [operator-cookbook, cookbook-volume-2, transcription, whisper, speech-to-text, call-log]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 7
---

*Part 7 of **The Operator's Cookbook, Volume 2**, a 14-part tutorial series
of complete, copy-paste GopherTrunk builds — one working rig per part —
built on what landed after Volume 1.
[Part 6]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
told you a call happened. This part tells you what was said. The
`transcription:` section sends every finished recording to a speech-to-text
server you choose and keeps the text with the call, so the History panel
of Volume 1's
[archival rig]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
becomes something you can search by word.*

> **TL;DR:** `transcription:` posts each finished recording to one
> OpenAI-shaped endpoint — multipart `file` and `model`, plus `language` and
> `prompt` when set, JSON `{"text": …}` back — which covers OpenAI, a
> whisper.cpp server at `/inference`, faster-whisper / Speaches and LocalAI.
> The default `upload_format: wav16k` converts the WAV or FLAC to 16 kHz
> mono 16-bit first, the one input every Whisper server takes. The text
> reaches the call row through `call_recordings`, matched by the recording's
> path, and a multi-over call's segments append. It shows in History's
> Transcript column and search, in the CSV export, and on the bus as
> `call.transcript`. GopherTrunk runs no model itself. One worker, a
> 128-deep queue, a 60 s timeout; counters at `GET /api/v1/transcription`.
> Status: synthetic-verified against a fake server only.

**Key takeaways**

- **One client covers four kinds of server.** They all take the same
  multipart request and answer with the same JSON, so moving from a hosted
  API to a box in your closet is a change of `url`.
- **The recording's path is the key.** The text is stored by matching that
  path to a call row; rename or move the file before its transcript arrives
  and the text has no row to land on.
- **Transcription sits after the recorder and cannot stall it.** It
  subscribes to finished recordings, queues 128 of them, and drops and
  counts beyond that.
- **Where the audio goes is one line, and it is your decision.** A cloud
  `url` uploads radio traffic to a third party; a loopback `url` keeps it
  on the box.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Endpoint | any server that takes the OpenAI transcription request | `transcription.url`, `api_key`, `model` |
| Hints | language and vocabulary for the model | `language`, `prompt` |
| What gets sent | filter by system, talkgroup, length and encryption | `systems`, `talkgroups`, `min_duration_ms`, `skip_encrypted` |
| Upload format | 16 kHz mono WAV, or the file as recorded | `upload_format` (`wav16k` or `original`) |
| Throughput | concurrent requests, and a bound on each | `workers`, `timeout` |
| Where the text shows | History's Transcript column and search; the CSV export | `/history`, `GET /api/v1/calls/history?format=csv` |
| Event | one per transcribed recording | `call.transcript` |
| Counters | sent, failed, skipped, dropped, mean latency | `GET /api/v1/transcription` and the Dashboard card |

## In this post

- **What you're building** — the path from a finished file to text on a call row.
- **The shopping list** — a server, and how to size it honestly.
- **The config** — a local whisper.cpp on the same box, every key verified.
- **Privacy — where the audio goes** — cloud against local, in plain terms.
- **First run — what healthy looks like** — the log lines, the counters, the search.
- **When it doesn't work** — symptom, cause, fix.

## What you're building

Any recording rig, plus a speech-to-text server, plus a `transcription:`
block that joins them. GopherTrunk never loads a model. It speaks one HTTP
request shape, the OpenAI audio-transcriptions one, and that is what
OpenAI, a local whisper.cpp server, faster-whisper / Speaches and LocalAI
all accept.

The path is short. When the recorder finishes a file it publishes the
completed-call event
([Recording, Composition & Streaming Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})),
after the loudness normalisation of
[Part 5]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }})
if that is on. The transcriber hears the event and applies its filters:
the system, the talkgroup, whether the call was encrypted, and whether it
lasted at least `min_duration_ms`. What passes is queued. A worker reads
the file, converts it to 16 kHz mono WAV, posts it and waits for the text.

The text then goes three places. It is stored on the call's row in the
SQLite call log
([Part 10 of the same series]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})),
found through the `call_recordings` table by the recording's path; a call
recorded as several per-over files gets each file's text appended. It is
published as a `call.transcript` event, which alert rules can match with
`keywords`. And it is logged.

Status, plainly: synthetic-verified. The client is tested against a local
fake that answers like an OpenAI or whisper.cpp server and records what it
was sent. The project's engineering notes record no run against a real
whisper.cpp or hosted endpoint, so treat the recipe below as unproven on
real hardware until your first transcript line appears.

<figure class="lab-figure">
<svg viewBox="0 0 680 214" width="680" height="214" role="img" aria-label="Transcription path: a finished recording passes filters for system, talkgroup, length and encryption into a 128-file queue that drops and counts when full; a worker converts the file to 16 kHz mono WAV and posts it to the server at transcription.url; the returned text goes to the call row by file path, to a call.transcript event and to the log">
  <rect x="8" y="58" width="100" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="58" y="84" text-anchor="middle" fill="currentColor" font-size="10">recording done</text>
  <text x="58" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">call.complete</text>
  <line x1="108" y1="94" x2="120" y2="94" stroke="currentColor"/>
  <rect x="120" y="58" width="112" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="176" y="78" text-anchor="middle" fill="currentColor" font-size="10">filters</text>
  <text x="176" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">system · talkgroup</text>
  <text x="176" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">length · encrypted</text>
  <line x1="232" y1="94" x2="244" y2="94" stroke="currentColor"/>
  <rect x="244" y="58" width="84" height="72" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="286" y="84" text-anchor="middle" fill="var(--accent)" font-size="10">queue</text>
  <text x="286" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">128 files</text>
  <line x1="286" y1="130" x2="286" y2="154" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="286" y="168" text-anchor="middle" fill="var(--accent)" font-size="9">full: drop + count</text>
  <text x="286" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the recorder never waits</text>
  <line x1="328" y1="94" x2="340" y2="94" stroke="currentColor"/>
  <rect x="340" y="58" width="104" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="392" y="84" text-anchor="middle" fill="currentColor" font-size="10">convert</text>
  <text x="392" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">16 kHz mono WAV</text>
  <line x1="444" y1="94" x2="456" y2="94" stroke="currentColor"/>
  <rect x="456" y="58" width="100" height="72" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="506" y="78" text-anchor="middle" fill="var(--accent)" font-size="10">POST to url</text>
  <text x="506" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">file + model</text>
  <text x="506" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">returns text</text>
  <line x1="556" y1="94" x2="568" y2="34" stroke="currentColor"/>
  <line x1="556" y1="94" x2="568" y2="94" stroke="currentColor"/>
  <line x1="556" y1="94" x2="568" y2="154" stroke="currentColor"/>
  <rect x="568" y="14" width="106" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="621" y="31" text-anchor="middle" fill="currentColor" font-size="10">call row</text>
  <text x="621" y="44" text-anchor="middle" fill="var(--fg-muted)" font-size="9">matched by file path</text>
  <rect x="568" y="74" width="106" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="621" y="91" text-anchor="middle" fill="currentColor" font-size="10">call.transcript</text>
  <text x="621" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bus event</text>
  <rect x="568" y="134" width="106" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="621" y="151" text-anchor="middle" fill="currentColor" font-size="10">log line</text>
  <text x="621" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="9">text, latency</text>
  <text x="340" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="9">GopherTrunk runs no model: the server at transcription.url does, on this box, on your LAN or in a cloud</text>
</svg>
<figcaption>Transcription hangs off the finished recording, behind its own queue, so a slow or dead server costs transcripts and never a recording.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| A recording rig | $0 | with `recordings.dir` and `storage.path` set: no file, nothing to send; no database, nowhere to keep the text |
| A Whisper server | $0 | whisper.cpp built from source on this box or another; or a hosted API billed by the minute of audio |
| CPU for it | varies | the one real cost; see below |

Sizing it honestly means admitting what is not measured. The project has
no benchmark table for any model on any hardware. The operator doc's
guidance is that a local whisper.cpp with the `base.en` or `small.en`
model keeps up with a busy system on a small box. Three facts make your
own answer easy to get:

- **The work scales with talk time, not wall time.** A system with two
  hours of talk a day gives the server two hours of audio a day.
- **The counters tell you your margin.** `mean_latency_ms` times `sent`,
  divided by 1000, divided by `audio_seconds`, is the fraction of real time
  your server needs. Under 1 with one worker, it keeps up on average.
- **The decoder is real time and the model is not.** On a small computer
  such as a [Raspberry Pi]({{ '/reference/raspberry-pi/' | relative_url }}),
  a model using every core competes with the DSP. If IQ-drop WARNs appear
  while a transcription runs, limit the server's threads or move it:
  `url` can point at any machine on your LAN. Part 14 sizes the decoder
  side of a Pi 4.

## The config

```yaml
storage:
  path: "../data/calls.db"       # transcripts are stored on call rows

recordings:
  dir: "../recordings"
  min_duration_ms: 500           # drop key-ups before they cost a request

transcription:
  enabled: true
  url: "http://127.0.0.1:8081/inference"    # whisper.cpp server, this box
  model: "whisper-1"             # ignored by whisper.cpp; hosted APIs need it
  language: "en"                 # empty = let the server detect
  prompt: "Engine 1, Medic 7, Main Street"  # your unit and street names
  systems: ["Metro-P25"]         # empty = every system
  talkgroups: [1001, 1002]       # empty = every talkgroup
  min_duration_ms: 1000
  workers: 1
  timeout: "60s"
  upload_format: wav16k          # wav16k (default) | original
  skip_encrypted: true
```

And the server it points at. These are whisper.cpp's own commands, not
GopherTrunk's; older builds name the binary `server`, so check its README
and `--help` for your version:

```bash
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
sh ./models/download-ggml-model.sh base.en
cmake -B build
cmake --build build -j --config Release
./build/bin/whisper-server -m models/ggml-base.en.bin --host 127.0.0.1 --port 8081
```

Decisions worth understanding.

**Port 8081, not 8080.** whisper.cpp's server listens on 8080 by default,
and so does GopherTrunk's own API (`api.http_addr: "127.0.0.1:8080"`). The
example in `config.example.yaml` shows `http://127.0.0.1:8080/inference`,
which is right when the server is the only thing on that port and wrong on
a box that also runs the daemon with defaults. Give one of them another
port.

**`wav16k` is the safe default.** Recordings are 8 kHz for digital voice
and may be FLAC. whisper.cpp accepts 16 kHz WAV and nothing else, so the
transcriber reads either container, resamples to 16 kHz mono 16-bit and
uploads that: 32 kB per second of audio. `original` uploads the file
as recorded, for a server that does its own conversion.

**Filter before you transcribe.** `systems` and `talkgroups` decide what
is worth a request. `min_duration_ms` (default 1000) skips squelch tails,
which a speech model turns into confident nonsense at the same cost as a
real call, and `recordings.min_duration_ms` drops those files before they
exist at all. `skip_encrypted` defaults to true because ciphertext
transcribes to nothing useful.

**One worker keeps the text in order.** A local server serialises requests
anyway and a hosted one rate-limits. There is a second reason: the
segments of a multi-over call are appended as their transcripts return, so
with several workers a short later over can land ahead of a long earlier
one.

**The path is the key.** The transcript is stored with a single `UPDATE`
that finds the call row by the recording's path. Nothing in GopherTrunk
moves a recording, but your own tooling might: a sync job, a tidy-up
script, an `exec` alert from Part 6. Anything that renames or moves the
file before the text arrives leaves the worker either unable to read it or
with no row to attach the text to. Move files after they are transcribed,
or leave them where the recorder put them and let `retention` age them
out.

## Privacy — where the audio goes

A hosted `url` sends every matching recording to a company outside your
control. That is other people's radio traffic, and it can include names,
addresses and medical details. The file name goes with it (by default the
date, time and talkgroup), and so does your `prompt`. What the provider
keeps, and for how long, is its policy and not GopherTrunk's.

A loopback or LAN `url` keeps the audio on machines you own. That is the
reason this recipe is built around a local server.

Either way: send only what you need (`systems`, `talkgroups`), leave
`skip_encrypted` on, and check what your local law says about recording,
transcribing and passing on radio communications — receiving a signal and
redistributing its content are different things
([scanner law, in plain English]({{ '/police-scanner-legal/' | relative_url }})).
Stored transcripts deserve the same care as recordings. They live on the
call row, so `retention.call_log_days` removes them; `files_days` removes
only the audio, and the text outlives it if the two differ. An `api_key`
is a secret in a plain-text file: keep the API on loopback, as Part 6
explains.

## First run — what healthy looks like

Prove the server alone first, with the 16 kHz sample that ships in its
repository:

```bash
curl -s http://127.0.0.1:8081/inference -F file=@samples/jfk.wav -F response_format=json
```

Then start the daemon:

```bash
gophertrunk run -config config.yaml
```

```text
INF transcription enabled url=http://127.0.0.1:8081/inference model=whisper-1
INF recorder: call ended device=... wav=... duration=... reason=...
INF transcribe: transcript system=Metro-P25 tg=1001 segment=... latency=... text="..."
```

The third line is the whole feature: one per transcribed file, with the
round-trip time and the first 120 characters of text. The counters:

```bash
curl -s http://127.0.0.1:8080/api/v1/transcription
```

```text
{"configured":true,"url":"http://127.0.0.1:8081/inference","model":"whisper-1","language":"en",
 "queued":...,"dropped":0,"sent":...,"failed":0,"skipped":...,
 "last_error":"","last_text":"...","audio_seconds":...,"mean_latency_ms":...}
```

`skipped` counts what the filters excluded and `dropped` what the full
queue refused. In the web console, History (`/history`) has a Transcript
column showing the first 80 characters, the full text in the call's detail
view, and a search box that matches transcript text alongside alias,
system, talkgroup and radio ID. The Dashboard's "Alerts & transcription"
card shows the same counters and the last transcript. From a shell:

```bash
curl -s "http://127.0.0.1:8080/api/v1/calls/history?q=engine&limit=5"
curl -s -o calls.csv "http://127.0.0.1:8080/api/v1/calls/history?format=csv"
```

The first returns the matching call rows, each with a `transcript` field;
the second is the same history as CSV with a `transcript` column.

## When it doesn't work

The three lines worth searching the log for:

```text
WRN transcribe: request failed path=... err=...
WRN transcribe: queue full, dropping recording path=...
WRN transcribe: store failed path=... err=...
```

| Symptom | Likely cause | Fix |
|---|---|---|
| No start-up line about transcription | the section is commented out or disabled | `enabled: true` and a `url` beginning with http:// or https:// |
| Enabled, but nothing is ever queued | decode-only daemon: with no recordings directory there is no file to send | Set `recordings.dir` |
| The first WARN with a refused connection | the server is not running, or is on another port | Start it; repeat the curl test against the same `url` |
| The first WARN with HTTP 405 or 404 | the URL points at GopherTrunk's own API, which shares whisper.cpp's default port | Move the server to another port and update `url` |
| The server rejects the audio | the file was uploaded as recorded: 8 kHz, or FLAC | `upload_format: wav16k` |
| Skipped climbs and sent stays at zero | a filter excludes everything: a system name that matches no system, a talkgroup list, encrypted calls, recordings under the minimum length | Loosen `systems`, `talkgroups` or `min_duration_ms` one at a time |
| The second WARN, and dropped climbs | the server is slower than the talk | A smaller model, more threads, another machine, or a narrower `talkgroups` list |
| Transcript lines in the log, none in History | the path matched no call row: the file was renamed or moved, or there is no call log | Stop moving recordings before they are transcribed; set `storage.path` |
| A long call's text is out of order | several workers; segments append as they return | `workers: 1` |
| Timeouts on long recordings | the per-request bound is shorter than the server needs | Raise `timeout`, or use a smaller model |

### How this recipe shapes operator practice

- **Treat the recording's path as a primary key.** Until the transcript is
  stored, the file's name and place are how the text finds its call.
- **Read the counters before the transcripts.** Skipped, dropped and
  failed tell you whether the pipeline is healthy; the text only tells you
  about one call.
- **Decide where the audio goes on purpose.** The default should be a
  machine you own.

## Variations

- **A hosted API.** Change two lines and add one. The privacy section
  applies in full.

```yaml
transcription:
  enabled: true
  url: "https://api.openai.com/v1/audio/transcriptions"
  api_key: "sk-CHANGE-ME"        # sent as a Bearer header
  model: "whisper-1"
```

- **Other local servers.** Only the `url` differs:

| Server | `url` |
|---|---|
| whisper.cpp server | `http://host:8080/inference` |
| faster-whisper-server / Speaches | `http://host:8000/v1/audio/transcriptions` |
| LocalAI | `http://host:8080/v1/audio/transcriptions` |

- **Alert on a phrase.** A rule on `call.transcript` with `keywords`
  matches case-insensitively anywhere in the text, and `attach_audio`
  sends the recording along:

```yaml
alerts:
  channels:
    - name: ops-discord
      type: discord
      url: "https://discord.com/api/webhooks/CHANGE-ME"
  rules:
    - name: "working-fire"
      on: [call.transcript]
      keywords: ["structure fire", "working fire"]
      attach_audio: true
      channels: [ops-discord]
```

- **Your own consumer.** `call.transcript` carries `system`, `group_id`,
  `source_id`, `segment`, `audio_path` and `text`, on the event stream
  described in the [events reference]({{ '/api-events.html' | relative_url }})
  and, with Part 6's `mirror_events`, on MQTT.

The full key reference is the
[transcription page]({{ '/transcription.html' | relative_url }}).

## Where this goes next

Transcription skips encrypted calls because ciphertext says nothing. If
the keys are yours — your own radios, a system you administer —
[Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})
configures `encryption_keys` so those calls decode to speech in the first
place, and shows how to check a key offline before trusting it live.

## FAQ

**Does GopherTrunk run Whisper itself?**
No. It runs no model. It converts each finished recording to 16 kHz mono
WAV and posts it to the HTTP endpoint in `transcription.url`, which can be
a whisper.cpp server on the same machine, another box on your LAN, or a
hosted API.

**How do I transcribe scanner calls without sending audio to the cloud?**
Run a Whisper server locally, such as whisper.cpp's, and point
`transcription.url` at it on loopback, for example
`http://127.0.0.1:8081/inference`. The audio then never leaves the
machine. Use a port other than 8080 if the GopherTrunk API is on its
default.

**Why are my transcripts missing from the History panel?**
The text is attached to a call by matching the recording's file path, so a
recording renamed or moved before its transcript returned has no row to
receive it. The other causes are an unset `storage.path` and filters that
skip the call; `GET /api/v1/transcription` shows which.

**Can I get an alert when a specific phrase is said?**
Yes. An alert rule with `on: [call.transcript]` and a `keywords` list
fires when the transcript contains one of the phrases, case-insensitively,
and can attach the recording on channels that carry files.

**Is transcription verified against a real Whisper server?**
Not in the project's notes. The client is synthetic-verified against a
local fake that answers like an OpenAI or whisper.cpp server; no run
against a real endpoint is recorded yet.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Alerts — Tell Me When It Matters]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
· Next →
[Part 8: Your Own Keys — Decrypting What You're Authorised to Hear]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})
