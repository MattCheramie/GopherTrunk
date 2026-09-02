---
title: "The Operator's Cookbook, Part 9: Sharing the Feed — Broadcastify, OpenMHz & Friends"
description: Turn a working GopherTrunk rig into a public feed — copy-paste broadcast config for Broadcastify Calls, RdioScanner, OpenMHz, Icecast and webhooks, the MP3 transcode and loudness settings that make uploads sound right, and a troubleshooting table for uploads that fail or vanish.
category: tutorials
keywords: broadcastify calls upload sdr, rdioscanner upload gophertrunk, openmhz api upload, icecast scanner feed, police scanner streaming setup, sdr call upload mp3, trunking call webhook, scanner feed loudness normalization, gophertrunk cookbook
tags: [operator-cookbook, broadcastify, openmhz, rdioscanner, icecast, streaming]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 9
---

*Part 9 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
put the radios far away and piped their IQ home over the network. This part
points the pipe the other direction: your rig has been quietly recording clean
calls for eight parts, and now other people get to hear them. One `broadcast:`
block turns the box into an upload station for Broadcastify Calls, RdioScanner,
OpenMHz and an Icecast live mount — all at once, from the same recordings —
plus JSON webhooks for anything you want to build yourself.*

> **TL;DR:** Everything outbound lives under one `broadcast:` key in
> `config.yaml`, verified against `config.example.yaml`: `broadcastify:`,
> `rdioscanner:`, `openmhz:`, `icecast:`, `webhook:` and `grant_webhook:` are
> each a **list of feeds**, and every completed call fans out to every feed
> that accepts its system. GopherTrunk transcodes each call to MP3 in pure Go
> (8 kHz mono lands in the MPEG-2.5 family at 32 kbps) and retries uploads
> with exponential backoff. The healthy log line is
> `broadcast: call streamed backend=… system=… tg=… attempt=1`; the two you
> troubleshoot are `broadcast: upload failed` and
> `broadcast: upload queue full, dropping call`. Loudness for the *distributed*
> copy is `recordings.normalize.apply_to: distributed` — the on-disk file stays
> untouched.

**Key takeaways**

- **One rig, many feeds, zero extra hardware.** The broadcast manager is a
  fan-out: each backend is a list entry, each entry filters by `systems:`, and
  a call uploads to every feed that wants it. Running Broadcastify *and*
  OpenMHz *and* your own webhook is three blocks, not three rigs.
- **Per-call and live are different animals.** Broadcastify Calls, RdioScanner
  and OpenMHz take one MP3 per completed call; Icecast is a continuous source
  connection GopherTrunk keeps fed with calls back-to-back, padded with
  silence between them.
- **Uploads fail loudly and drop safely.** Every send gets retries with
  backoff and a 60-second per-attempt timeout; a wedged backend fills a queue
  that *drops calls rather than stall the recorder*. Your recordings are never
  hostage to someone else's server.
- **API keys are credentials — treat them like it.** Every backend key sits in
  `config.yaml`, which is exactly why that file must never land in a public
  repo or a support paste. The hygiene section below is short and
  non-negotiable.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Global gates | drop short calls, size the upload pool | `broadcast.min_duration_ms`, `broadcast.workers` |
| Broadcastify Calls | metadata POST → one-time URL → MP3 PUT | `broadcast.broadcastify[]`: `api_key`, `system_id` |
| RdioScanner | multipart POST to your own instance | `broadcast.rdioscanner[]`: `url`, `api_key`, `system_id` |
| OpenMHz | multipart POST to api.openmhz.com | `broadcast.openmhz[]`: `api_key`, `short_name` |
| Icecast / ShoutCast | live source mount, calls + silence | `broadcast.icecast[]`: `host`, `port`, `mount`, `password` |
| Webhooks | per-call JSON, per-grant JSON | `broadcast.webhook[]`, `broadcast.grant_webhook[]` |
| Loudness of the MP3 | normalize the distributed copy only | `recordings.normalize.apply_to: distributed`, [loudness deep dive]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }}) |

## In this post

- **What you're building** — one rig feeding aggregators, a live stream, and your own endpoints.
- **The shopping list** — accounts and keys, not hardware.
- **The config** — every backend block, every key verified.
- **First run — what healthy looks like** — the upload log lines that matter.
- **When it doesn't work** — symptom → cause → fix for silent and missing uploads.
- **Variations** — webhooks-only rigs, encrypted-call policy, multi-system routing.

## What you're building

The finished rig is any working build from Parts 1–8 with its outputs made
public: every completed call is transcoded to MP3 and pushed to the call
aggregators you've signed up for, a live Icecast mount plays your system like
a classic scanner feed, and a JSON webhook mirrors call metadata into whatever
you run — a Discord relay, a Grafana pipeline, a database. The architecture is
covered in depth on the deep-dive side
([the broadcast manager]({{ '/blog/deep-dives/recording-streaming-13-broadcast-manager/' | relative_url }})
and [the aggregator backends]({{ '/blog/deep-dives/recording-streaming-14-aggregator-backends/' | relative_url }}));
this recipe is the operator's view of the same machinery.

Two design facts shape everything below. First, the manager subscribes to
*completed calls* — the recorder finishes a file, then upload workers
(`broadcast.workers`, default 2) fan it out, so a slow aggregator can never
back up into the decode path. Second, each feed carries an optional
`systems:` list matching your `trunking.systems[].name` values, so a
multi-system rig from [Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
can send the county P25 to Broadcastify and keep the business DMR private.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Signal chain of the sharing rig: the recorder emits a completed call which enters the broadcast manager, is filtered by minimum duration and per-feed system lists, transcoded once to a 32 kilobit MP3 with optional loudness normalization, then fanned out by upload workers to Broadcastify Calls, RdioScanner, OpenMHz, an Icecast live mount padded with silence, and a JSON webhook; a separate grant webhook path runs straight from the control-channel decoder without audio">
  <rect x="10" y="96" width="104" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="62" y="112" text-anchor="middle" fill="currentColor" font-size="10">recorder</text>
  <text x="62" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">completed call</text>
  <line x1="114" y1="115" x2="148" y2="115" stroke="currentColor"/>
  <rect x="148" y="60" width="180" height="110" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="238" y="78" text-anchor="middle" fill="var(--accent)" font-size="10">broadcast manager</text>
  <text x="238" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">min_duration_ms · systems filter</text>
  <text x="238" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tg stream: false honored</text>
  <text x="238" y="132" text-anchor="middle" fill="currentColor" font-size="10">MP3 encode (32 kbps)</text>
  <text x="238" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="9">± loudness normalize (distributed)</text>
  <text x="238" y="162" text-anchor="middle" fill="var(--fg-muted)" font-size="9">workers · retry w/ backoff</text>
  <line x1="328" y1="90" x2="392" y2="40" stroke="currentColor"/>
  <line x1="328" y1="105" x2="392" y2="86" stroke="currentColor"/>
  <line x1="328" y1="120" x2="392" y2="132" stroke="currentColor"/>
  <line x1="328" y1="140" x2="392" y2="178" stroke="currentColor"/>
  <rect x="392" y="24" width="272" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="528" y="44" text-anchor="middle" fill="currentColor" font-size="10">Broadcastify Calls · RdioScanner · OpenMHz</text>
  <rect x="392" y="70" width="272" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="528" y="84" text-anchor="middle" fill="currentColor" font-size="10">Icecast live mount</text>
  <text x="528" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">calls back-to-back, silence between</text>
  <rect x="392" y="116" width="272" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="528" y="136" text-anchor="middle" fill="currentColor" font-size="10">webhook: per-call JSON (± base64 MP3)</text>
  <rect x="392" y="162" width="272" height="32" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="528" y="176" text-anchor="middle" fill="var(--accent)" font-size="10">grant_webhook: per-grant JSON, no audio</text>
  <text x="528" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fed by the CC decoder, not the recorder</text>
  <rect x="10" y="176" width="104" height="38" rx="4" fill="none" stroke="var(--fg-muted)"/>
  <text x="62" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="10">CC decoder</text>
  <text x="62" y="205" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every grant</text>
  <line x1="114" y1="195" x2="392" y2="180" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="340" y="238" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one recording, encoded once, delivered everywhere — and dropped, never blocking, when a backend wedges</text>
</svg>
<figcaption>The fan-out: per-call MP3s to the aggregators and webhooks, a paced live stream to Icecast, and grants pushed straight off the control channel.</figcaption>
</figure>

## The shopping list

No hardware this time — the shopping is accounts and keys:

| Item | Cost | What you need from it |
|---|---|---|
| Broadcastify Calls account | free to provide a feed | a Calls **API key** and your node's **system ID** |
| OpenMHz system | free | an **API key** and your system's **short_name** |
| RdioScanner instance | free, self-hosted | its **URL** and an **API key** you mint in its admin |
| Icecast server | free, self-hosted (or a host you have a source login on) | host, port, **mount**, source password |
| Your own endpoint | whatever you build | a URL that accepts JSON POSTs |

Pick any subset — data-only rigs skip straight to the webhook variation. And
a word before signing up anywhere: streaming public-safety audio is subject
to local law and the aggregators' own rules; that homework is yours.

## The config

Append to a working rig from any earlier part. Every key below exists in
`config.example.yaml`:

```yaml
broadcast:
  min_duration_ms: 1500     # skip squelch crackle & failed decodes
  workers: 0                # 0 = default upload pool (2)

  broadcastify:
    - enabled: true
      name: "bcfy-metro"
      api_key: "YOUR_BROADCASTIFY_CALLS_API_KEY"
      system_id: 12345
      systems: ["Metro-P25"]        # omit to stream every system

  rdioscanner:
    - enabled: true
      name: "local-rdio"
      url: "https://scanner.example.org"
      api_key: "YOUR_RDIOSCANNER_API_KEY"
      system_id: 1

  openmhz:
    - enabled: true
      name: "openmhz-metro"
      api_key: "YOUR_OPENMHZ_API_KEY"
      short_name: "metro911"

  icecast:
    - enabled: true
      name: "live-feed"
      host: "stream.example.org"
      port: 8000
      mount: "/gophertrunk"
      username: "source"
      password: "YOUR_ICECAST_SOURCE_PASSWORD"
      stream_name: "GopherTrunk Live"

  webhook:
    - enabled: true
      name: "analytics"
      url: "https://example.org/gophertrunk/calls"
      auth_header: "Bearer YOUR_TOKEN"
      include_audio: false          # true embeds the base64 MP3

  grant_webhook:
    - enabled: true
      name: "grant-log"
      url: "https://example.org/gophertrunk/grants"
      auth_header: "Bearer YOUR_TOKEN"

recordings:
  normalize:
    enabled: true
    apply_to: distributed   # normalize the MP3 only; disk file untouched
    target_lufs: -16.0
    true_peak_dbtp: -1.5
    max_boost_db: 12.0
```

Decisions worth explaining. **`min_duration_ms`** is your feed-quality knob:
sub-second files are almost always dead keys, and aggregators judge feeds by
signal-to-noise in the human sense too. **`apply_to: distributed`** runs the
EBU R128 loudness normalize on the in-memory MP3 while leaving your archive
faithful — the right split for a rig that's also the Part 10 archival build;
[the loudness deep dive]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})
explains the −16 LUFS target. **`grant_webhook`** is not a duplicate of
`webhook`: it fires per control-channel *grant* the moment it decodes, carries
no audio, and exists so consumers can retire a separate grant-log dependency
(issue [#915](https://github.com/MattCheramie/GopherTrunk/issues/915)).
The per-call webhook payload carries system, talkgroup, source RID, call
type, frequency, P25 site identity, encryption and emergency flags, and
start/stop timestamps.

Two quieter filters also apply: a talkgroup with `stream: false` in your
talkgroup CSV is excluded from **all** feeds (Part 13 covers the roster
columns), and `recordings.skip_encrypted: true` suppresses the completed-call
backends for encrypted calls — keep it `false` if your webhook consumer wants
encrypted-call metadata (issue
[#897](https://github.com/MattCheramie/GopherTrunk/issues/897)).

### Key hygiene

Every string above that says `YOUR_…` is a credential. Three rules: keep
`config.yaml` out of version control and out of support pastes (the daemon's
diagnostics never print it, but *you* might); on a shared box, make it
readable only by the daemon's user — the systemd recipe in
[Part 11]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }})
installs it mode `0640`; and when a key does leak, rotate it at the provider
first and in the file second. The API bearer token has a `token_file`
indirection for exactly this reason
([auth posture]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }}));
broadcast keys live inline, so the file itself is the secret.

## First run — what healthy looks like

Restart the daemon and key up the system (or wait). Per completed call, per
feed, the line you want is:

```
INF broadcast: call streamed backend=bcfy-metro system=Metro-P25 tg=9001 attempt=1
```

`attempt=1` means it worked first try. The Icecast source announces itself
once at startup and again after any reconnect:

```
INF broadcast: icecast source connected backend=live-feed mount=/gophertrunk
```

Then verify at the far end: your call appears on the Broadcastify Calls
node / OpenMHz system page within seconds, RdioScanner shows it in its own
UI, and the Icecast mount plays your calls with silence between them —
that's the pacing design, not a fault. A transient failure looks like this
and is self-healing:

```
WRN broadcast: upload failed backend=openmhz-metro system=Metro-P25 tg=9001 attempt=1 of 4 err=...
INF broadcast: call streamed backend=openmhz-metro system=Metro-P25 tg=9001 attempt=2
```

Only `ERR broadcast: giving up on call` means a call was lost to that feed —
and even then only to that feed; the recording on disk and every other
backend are unaffected.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| `broadcast: upload failed … HTTP 401/403` on every call | wrong or revoked API key | Re-paste the key from the provider's dashboard; check you didn't swap two feeds' keys |
| `broadcastify: metadata response rejected` | wrong `system_id`, or the talkgroup isn't in Broadcastify's DB for that node | Broadcastify validates metadata *before* issuing the upload URL — fix the node config on their side, then the ID here |
| Uploads succeed but play as silence | historically, MP3 encoder bugs — three of them | Fixed and regression-pinned; the forensic story is [the silent-MP3 postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-15-silent-mp3/' | relative_url }}). If you see it on a current build, file an issue with the paired WAV |
| Some talkgroups never appear on any feed | `stream: false` in the talkgroup CSV, or the call is shorter than `min_duration_ms` | Check the roster column and the duration gate before suspecting the backend |
| One system uploads, another doesn't | `systems:` filter mismatch | The list must match `trunking.systems[].name` **exactly** — it's a name, not a pattern |
| `broadcast: upload queue full, dropping call` | a backend is wedged or the network is down, and the queue chose dropping over stalling | Find which backend logs `upload failed`, fix or disable it; raise `workers` only if all backends are merely slow |
| Encrypted calls missing from your webhook | `recordings.skip_encrypted: true` ends the call before the recorder lifecycle completes | Set it `false` to deliver them (issue [#897](https://github.com/MattCheramie/GopherTrunk/issues/897)) |
| `broadcast: icecast source disconnected` repeating | wrong mount/password, or the server dropped the source | The backend reconnects on a fixed backoff forever; fix the credentials server-side and watch for `source connected` |

One habit transfers from the decode side of this series: **read the log line
before theorizing.** Every failure above names its backend and its attempt
count — the difference between "OpenMHz rejected the key" and "my ISP dropped
for a minute" is right there in `attempt=N of M`.

## Variations

- **Data-only rig.** Enable only `webhook` and `grant_webhook`: full call and
  grant metadata to your own systems, zero audio leaves the box. The
  [grant-webhook deep dive]({{ '/blog/deep-dives/running-it-for-real-11-grant-webhooks/' | relative_url }})
  shows consumer patterns.
- **Split routing.** Multiple entries per backend are legal — two
  Broadcastify feeds with different `system_id`s and disjoint `systems:`
  lists route a two-county rig to two nodes from one process.
- **Louder everywhere.** `apply_to: both` normalizes the archive *and* the
  MP3; `recording` normalizes only disk. Distributed-only is the recommended
  start because it's reversible.
- **Live-first rig.** Icecast alone, `min_duration_ms: 0`, and you've built a
  classic streaming scanner; add
  [`audio.enabled: true`]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})
  if the box should also play to local speakers.

## Where this goes next

Everything you just shared also lands on your own disk, and disks fill.
[Part 10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
builds the archival rig: first-class FLAC recordings at half the size, IQ
taps that archive the *radio* signal too, the SQLite call log, and the
retention math that says how many days actually fit.

## FAQ

**How do I stream GopherTrunk calls to Broadcastify?**
Sign up to provide a Broadcastify Calls feed, take the Calls API key and
system ID they assign, and paste both into a `broadcast.broadcastify` entry
as above. GopherTrunk does the two-step Calls protocol itself — a metadata
POST that returns a one-time upload URL, then an MP3 PUT — so there's no
external uploader script to run.

**What bitrate and format are the uploads?**
Per-call MP3, encoded in pure Go from the finished recording. At the
vocoder-native 8 kHz mono that recordings use, the encoder selects the
MPEG-2.5 family at 32 kbps CBR — small enough that a full day of busy-system
calls is tens of megabytes of upload.

**Can I upload the same system to Broadcastify and OpenMHz at once?**
Yes — that's the default behavior. Every enabled feed whose `systems:` list
(or absence of one) accepts the call gets its own upload, from one shared
MP3 encode. Backends fail independently: an OpenMHz outage never costs you a
Broadcastify call.

**Why do my uploads sound quieter than other feeds?**
Faithful decode is conservative, and aggregator players do no gain riding.
Enable `recordings.normalize` with `apply_to: distributed` for −16 LUFS
loudness-normalized MP3s while keeping the on-disk archive untouched — or
`recordings.enhance` from Part 1's table if you want the whole chain louder.

**Does the Icecast feed go silent between calls?**
It plays encoded silence, which is correct source behavior — the mount stays
up, listeners stay connected, and the next call starts immediately. If the
mount itself drops, look for `icecast source disconnected` lines; the backend
redials on a fixed backoff until the server takes the source again.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Radios Far Away — SoapyRemote, rtl_tcp & ka9q]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
· Next →
[Part 10: The Archival Rig — FLAC, Retention & the Call Log]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
