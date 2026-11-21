---
title: "The Operator's Cookbook Vol. 2, Part 6: Alerts — Tell Me When It Matters"
description: "A complete GopherTrunk alerts recipe: rules that watch bus events (an emergency on a talkgroup, a tone-out match, one radio keying up, a control channel lost) and deliver through ntfy, MQTT, a local command and six other channel types, behind a bounded queue that a dead destination cannot turn into a stalled decoder."
category: tutorials
keywords: scanner push notifications phone, p25 emergency alert notification, ntfy scanner alerts, mqtt home assistant scanner, tone out alert home automation, radio id alert script, trunking scanner alert rules, discord scanner webhook alerts, sdr scanner notifications, gophertrunk alerts
tags: [operator-cookbook, cookbook-volume-2, alerts, notifications, mqtt, ntfy]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 6
---

*Part 6 of **The Operator's Cookbook, Volume 2**, a 14-part tutorial series
of complete, copy-paste GopherTrunk builds — one working rig per part —
built on what landed after Volume 1.
[Part 5]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }})
tuned how calls sound. This part decides which of them should interrupt
you. The `alerts:` section is the "tell me when" feature of a hardware
scanner, pointed at the places you actually look: a phone notification, a
home-automation bus, a script of your own. It extends the tone-out
detector of
[Volume 1 Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }}),
whose matches used to go no further than the log and the Tones panel.*

> **TL;DR:** `alerts:` has two lists. `channels` are destinations, nine
> types: `discord`, `slack`, `ntfy`, `pushover`, `telegram`, `gotify`,
> `webhook`, `exec`, `mqtt`. `rules` say which bus events (`on:`, 22 kinds,
> default `call.start`) under which conditions (`systems`, `talkgroups`,
> `radios`, `emergency`, `encrypted`, `tone_profiles`, `keywords`,
> `min_duration_ms`, `cooldown`) go to which channels. Deliveries run on two
> workers behind a 256-deep queue; a full queue drops and counts instead of
> blocking the event bus. `exec` splits its command on whitespace, with no
> shell, and gives the program the alert as JSON on stdin plus `GT_ALERT_*`
> variables; `mqtt` publishes QoS 0 to `<topic>/alerts/<rule>`. Prove a
> channel with `POST /api/v1/alerts/test/<channel>` and read the counters at
> `GET /api/v1/alerts`. Status: synthetic-verified against local fakes of
> each service; no field report yet.

**Key takeaways**

- **Rules and channels are separate lists joined by name.** One rule fans
  out to several channels and one channel serves many rules; a rule naming
  a channel that is not defined stops the daemon at start-up.
- **Every condition on a rule must hold, and not every event carries every
  field.** A tone-out match has no talkgroup, and a grant whose
  control-channel message carried no source has source 0.
- **A destination can fail without touching the decoder.** Two workers, a
  256-deep queue and a 10-second delivery timeout sit between the bus and
  the network; overflow is dropped and counted.
- **Success is silent at the default log level.** A delivered alert logs at
  debug only, so the proof is the test route and the counters.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Destinations | nine channel types; a rule names one or more | `alerts.channels[]` (`name`, `type`, then the fields the type needs) |
| What to watch | 22 event kinds; a call starting is the default | `alerts.rules[].on` |
| Filters | every one you list must match | `systems`, `talkgroups`, `radios`, `emergency`, `encrypted`, `tone_profiles`, `keywords`, `min_duration_ms` |
| Repeat suppression | one firing per system, kind and talkgroup (or tone profile) per window | `cooldown` |
| Wording | a Go text template per rule, a built-in line otherwise | `message` |
| Prove a channel | a synthetic alert that bypasses the rules | `POST /api/v1/alerts/test/<channel>` |
| Counters | sent, failed, last error, suppressions, recent firings | `GET /api/v1/alerts` and the Dashboard card |
| Overflow | drop and count, never block | `alerts: delivery queue full, dropping alert` |

## In this post

- **What you're building** — three alerts, and the queue between them and the decoder.
- **What a rule can watch** — the 22 event kinds and the nine channel types.
- **The config** — phone push on an emergency, MQTT on a tone-out, a script on one radio.
- **First run — what healthy looks like** — the start-up line, the test route, the counters.
- **When it doesn't work** — symptom, cause, fix.

## What you're building

Three alerts on top of any working rig:

1. A push to your phone when an **emergency** is declared on a dispatch
   [talkgroup]({{ '/reference/talkgroup/' | relative_url }}).
2. An MQTT message to
   [Home Assistant]({{ '/reference/home-automation/' | relative_url }})
   when your station's tone-out matches, so the lights come on.
3. A script that runs when one particular
   [radio ID]({{ '/reference/radio-id/' | relative_url }}) keys up.

Everything GopherTrunk decodes is published on an in-process event bus
([Trunking Engine Part 2]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})).
The alert manager subscribes to it, normalises each event into one
protocol-neutral shape, and tests it against every rule. A match that is
not inside its cooldown is rendered into a title and a line of text and
queued once per channel. Two workers take jobs off that queue and make the
actual HTTP request, MQTT publish or command run.

That queue is the design. The bus never waits for a subscriber, and the
manager never waits for a destination: if a webhook hangs, its jobs occupy
a worker until the timeout, the queue fills, and further alerts are
dropped and counted while the decoder carries on. It is the same shape as
the grant webhook of
[Running It For Real Part 11]({{ '/blog/deep-dives/running-it-for-real-11-grant-webhooks/' | relative_url }}),
with the opposite purpose: that sink is a firehose of every grant for your
own database, this one is a few filtered events for a person.

Status, plainly: synthetic-verified. Rule matching, cooldowns, each HTTP
channel's payload, the command channel and the MQTT publisher are tested
against local fakes — an HTTP test server, a shell script, a minimal MQTT
3.1.1 broker. The project's engineering notes record no operator field
report for alerts yet, so the first real phone buzz is yours.

<figure class="lab-figure">
<svg viewBox="0 0 680 214" width="680" height="214" role="img" aria-label="Alert path: the event bus feeds rules that match on event kind and filters and apply a cooldown; matches enter a 256-job queue that drops and counts when full so the bus never waits; two workers with a ten-second timeout deliver to channels such as ntfy, MQTT and a local command">
  <rect x="8" y="58" width="112" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="64" y="78" text-anchor="middle" fill="currentColor" font-size="10">event bus</text>
  <text x="64" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">call.start · grant</text>
  <text x="64" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tone.alert · cc.lost</text>
  <line x1="120" y1="94" x2="136" y2="94" stroke="currentColor"/>
  <rect x="136" y="58" width="140" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="206" y="78" text-anchor="middle" fill="currentColor" font-size="10">rules</text>
  <text x="206" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">on: kinds + all filters</text>
  <text x="206" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cooldown per key</text>
  <line x1="276" y1="94" x2="292" y2="94" stroke="currentColor"/>
  <rect x="292" y="58" width="104" height="72" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="344" y="78" text-anchor="middle" fill="var(--accent)" font-size="10">queue</text>
  <text x="344" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">256 jobs</text>
  <text x="344" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one per channel</text>
  <line x1="344" y1="130" x2="344" y2="154" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="344" y="168" text-anchor="middle" fill="var(--accent)" font-size="9">full: drop + count</text>
  <text x="344" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the bus never waits</text>
  <line x1="396" y1="94" x2="412" y2="94" stroke="currentColor"/>
  <rect x="412" y="58" width="104" height="72" rx="4" fill="none" stroke="currentColor"/>
  <text x="464" y="78" text-anchor="middle" fill="currentColor" font-size="10">2 workers</text>
  <text x="464" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">timeout 10 s</text>
  <text x="464" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per delivery</text>
  <line x1="516" y1="94" x2="536" y2="36" stroke="currentColor"/>
  <line x1="516" y1="94" x2="536" y2="94" stroke="currentColor"/>
  <line x1="516" y1="94" x2="536" y2="152" stroke="currentColor"/>
  <rect x="536" y="16" width="136" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="604" y="33" text-anchor="middle" fill="currentColor" font-size="10">ntfy</text>
  <text x="604" y="46" text-anchor="middle" fill="var(--fg-muted)" font-size="9">push to a phone</text>
  <rect x="536" y="74" width="136" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="604" y="91" text-anchor="middle" fill="currentColor" font-size="10">mqtt</text>
  <text x="604" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">topic/alerts/rule, QoS 0</text>
  <rect x="536" y="132" width="136" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="604" y="149" text-anchor="middle" fill="currentColor" font-size="10">exec</text>
  <text x="604" y="162" text-anchor="middle" fill="var(--fg-muted)" font-size="9">JSON on stdin + env</text>
  <text x="340" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a dead destination holds a worker until its timeout; it cannot hold the decoder</text>
</svg>
<figcaption>The bounded queue is the whole point: rules decide what matters, workers deliver it, and the event bus is never made to wait for a network.</figcaption>
</figure>

## What a rule can watch

`on:` takes any of the 22 kinds below; leave it out and the rule watches
`call.start`.

| Kind (`on:`) | Fires when |
|---|---|
| `call.start`, `call.end`, `call.complete` | a voice SDR starts following a call; the call is released; its recording is finished on disk |
| `grant` | a control-channel grant is decoded, followed or not |
| `tone.alert` | a tone-out profile matches |
| `cc.locked`, `cc.lost` | a control channel locks or drops |
| `affiliation`, `registration` | a radio joins a talkgroup or registers |
| `patch` | a patch or supergroup is announced |
| `call.encryption` | a call's algorithm and key ID are decoded mid-call |
| `talker.alias`, `location` | an over-the-air alias or a GPS fix decodes |
| `unit.status`, `unit.message`, `call.alert`, `unit.ack`, `unit.queued`, `unit.deny`, `unit.function`, `unit.monitor` | P25 unit signalling on the control channel: status, short message, page, the three responses, radio check or inhibit, remote monitor |
| `call.transcript` | the transcription backend of Part 7 returns text |

Each channel type takes the fields its service needs:

- `discord`, `slack` — `url`, the incoming webhook.
- `ntfy` — `url` (the topic URL), `token` or `user` and `password`,
  `priority` 1 to 5.
- `pushover` — `token` (application), `user` (user key), `priority` −2 to 2.
- `telegram` — `token` (bot), `user` (chat ID).
- `gotify` — `url` (server), `token` (application), `priority` 0 to 10.
- `webhook` — `url`, and `token`, sent as a Bearer header.
- `exec` — `command`.
- `mqtt` — `url`, `user`, `password`, `topic`, `mirror_events`.

All take `timeout` (default 10 s). The full reference is the
[alerts page]({{ '/alerts.html' | relative_url }}).

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| A working rig | $0 | a trunked system for the emergency rule; the `tone_out` profile from Volume 1 Part 6 for the second |
| ntfy app | $0 | on the public server a topic name is the only secret: make it long and random, or self-host |
| MQTT broker | $0 | Mosquitto; a Home Assistant install usually has one already |

## The config

{% raw %}
```yaml
alerts:
  channels:
    - name: phone
      type: ntfy
      url: "https://ntfy.sh/CHANGE-ME-to-a-long-random-topic"
      priority: 5
    - name: home-assistant
      type: mqtt
      url: "tcp://192.168.1.10:1883"
      user: "gophertrunk"
      password: "CHANGE-ME"
      topic: "gophertrunk"
    - name: radio-script
      type: exec
      command: "/usr/local/bin/gt-radio-alert.sh"
      timeout: "5s"
  rules:
    - name: "dispatch-emergency"
      on: [grant]
      systems: ["Metro-P25"]
      talkgroups: [1001, 1002]
      emergency: true
      cooldown: "2m"
      channels: [phone]
      message: "EMERGENCY {{.System}} TG {{.TalkgroupLabel}}{{if .Source}} from {{.SourceLabel}}{{end}}"
    - name: "station-1-tone-out"
      on: [tone.alert]
      tone_profiles: ["station-1-engine"]
      channels: [home-assistant, phone]
    - name: "watch-radio-70001"
      on: [call.start]
      radios: [70001]
      cooldown: "5m"
      channels: [radio-script]
```
{% endraw %}

**Recipe 1 — phone push on an emergency.** `on: [grant]` fires on the
control channel's grant whether or not a voice SDR was free to follow it,
which is what you want for an
[emergency]({{ '/reference/emergency-call/' | relative_url }}). A grant
rule sees every decoded grant, so it gets a `cooldown`: one push per
talkgroup per two minutes. `systems` must equal the `name` of the system
under `trunking.systems` (case does not matter). On a conventional scan
list there is no control channel, so use `call.start` there. The `message`
template is optional; without it the text is a built-in line of the same
kind, and on ntfy an emergency also carries the rotating-light tag.

**Recipe 2 — MQTT on a tone-out.** The rule names the `tone_out` profile,
not a talkgroup: a tone alert (the detector is
[Beyond Voice Part 5]({{ '/blog/deep-dives/beyond-voice-05-two-tone-tone-out/' | relative_url }}))
carries the profile, its alpha tag and the matched tone frequencies, and
nothing about who was talking. The alert is
published as JSON to `gophertrunk/alerts/station-1-tone-out` — the topic
prefix, then `alerts`, then the rule name with spaces and slashes replaced
by underscores, which is why these rule names have none. In Home Assistant
an MQTT trigger on that topic is the whole integration. Messages are QoS 0
and not retained.

**Recipe 3 — a script on one radio.** `radios` matches the source of the
grant that opened the call. A reply from another radio inside the same
call raises no new `call.start`; a rule on `call.complete` matches the
last radio heard on each finished recording instead, after the fact. The
command line is split on whitespace and run directly, so pipes, redirects
and quoting belong inside the script. Save this as the `command` path and
make it executable:

```bash
#!/bin/sh
cat > /var/tmp/gt-last-alert.json    # stdin: the alert as JSON
printf '%s %s TG %s radio %s\n' "$GT_ALERT_AT" "$GT_ALERT_SYSTEM" \
  "$GT_ALERT_TALKGROUP" "$GT_ALERT_SOURCE" >> /var/tmp/gt-radio.log
```

The variables are `GT_ALERT_RULE`, `TITLE`, `TEXT`, `KIND`, `SYSTEM`,
`PROTOCOL`, `TALKGROUP`, `TALKGROUP_ALPHA`, `SOURCE`, `FREQUENCY_HZ`,
`EMERGENCY`, `ENCRYPTED`, `AUDIO_PATH`, `TONE_PROFILE` and `AT`, each with
the `GT_ALERT_` prefix. The JSON on stdin is the whole notification:
`rule`, `title`, `text`, `at` and an `event` object with every normalised
field, including `source_alpha` when the radio has an alias. A non-zero
exit is a failed delivery, reported with the first 200 characters of the
program's output.

## First run — what healthy looks like

```bash
gophertrunk run -config config.yaml
```

```text
INF alert rules enabled rules=3 channels=3
```

No such line means no rule is enabled. Now prove each channel without
waiting for an emergency:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/alerts/test/phone
curl -s -X POST http://127.0.0.1:8080/api/v1/alerts/test/home-assistant
curl -s -X POST http://127.0.0.1:8080/api/v1/alerts/test/radio-script
curl -s http://127.0.0.1:8080/api/v1/alerts
```

```text
{"channel":"phone","ok":true}
```

The phone shows "GopherTrunk test alert" and a line saying the channel is
reachable. The test bypasses the rules and the queue, and it is a write
route: from loopback it needs nothing under the default `api.auth.mode`,
from elsewhere it needs the bearer token
([Running It For Real Part 2]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})).
The web Dashboard has the same thing as a button: an "Alerts &
transcription" card with a Test button per channel and the sent and failed
counts. To watch the broker side:

```bash
mosquitto_sub -h 192.168.1.10 -u gophertrunk -P 'CHANGE-ME' -t 'gophertrunk/alerts/#' -v
```

The test arrives on `gophertrunk/alerts/test`. The status route returns
this shape (trimmed and reordered):

```text
{"configured":true,
 "channels":[{"name":"phone","type":"ntfy","sent":1,"failed":0}, ...],
 "rules":[{"name":"dispatch-emergency","channels":["phone"],"fired":0,"cooldown_suppressed":0}, ...],
 "matched":0,"queued":0,"dropped":0,"recent":[]}
```

When the pager tones go out, the detector's own line appears, and the rule
fires behind it:

```text
INF toneout: profile matched profile=station-1-engine device=00000001 tones=[1042.2 1297.4]
```

The delivered text reads
`tone-out matched: Station 1 Engine · 1042.2 / 1297.4 Hz · [00000001]`.
Nothing more is logged at `info` unless a delivery fails.

## When it doesn't work

The two lines worth searching the log for:

```text
WRN alerts: delivery failed rule=... channel=... type=... err=...
WRN alerts: delivery queue full, dropping alert rule=... channel=...
```

| Symptom | Likely cause | Fix |
|---|---|---|
| The daemon will not start after the section was added | validation: a rule names an undefined channel, an unknown event kind, or a template that does not parse | The error names the entry, for example `alerts.rules[0] (dispatch-emergency): channel "fone" is not defined under alerts.channels` |
| A rule fires on everything | a misspelled filter key; the loader ignores unknown keys instead of rejecting them | Check spelling against this post: `talkgroups`, `radios`, `tone_profiles` are plural |
| A rule never fires, and the matched counter stays at zero | the conditions are ANDed and one cannot hold: a talkgroup filter on a tone alert (it has none, and a system only if its profile sets one), an emergency filter on routine traffic, a radio filter where the grant carried no source | Remove filters until `matched` moves in `GET /api/v1/alerts`, then add them back one at a time |
| Matched climbs, nothing arrives | the cooldown, or a failing channel | Read `cooldown_suppressed` per rule, and `failed` and `last_error` per channel |
| Two notifications per call | two kinds are listed and the cooldown is kept per kind | Use `on: [grant]` or `on: [call.start]`, not both |
| The first WARN above with an HTTP status | the service refused: wrong token, protected topic, revoked webhook | The error carries the service's own first line; fix the credential and use the test route |
| The first WARN above for the broker | unreachable, or it refused the login | The error says which: `mqtt: connect ...` or `mqtt: broker refused connection (return code 4: bad user name or password)` |
| The script works by hand but fails as a channel | there is no shell: no pipes, redirects, tilde or quotes in the command | Put them inside the script and make `command` its path; raise `timeout` if it is slow |
| Event topics never appear with mirroring on | the manager starts only when at least one rule is enabled | Add a rule; `mirror_events` on a channel alone starts nothing |

### How this recipe shapes operator practice

- **Alert on little, and give everything a cooldown.** A phone that buzzes
  on every over is switched off within a day.
- **Test the channel before you trust the rule.** The test route separates
  "the destination is broken" from "the rule never matched".
- **Treat the section as secrets.** Webhook URLs, tokens and passwords sit
  in the config in plain text, and the config builder's read route
  (`GET /api/v1/config/file`) redacts decryption keys and nothing else.
  Keep `api.http_addr` on loopback, or put a
  [reverse proxy]({{ '/blog/deep-dives/running-it-for-real-03-tls-reverse-proxy/' | relative_url }})
  with its own login in front.

## Variations

- **The recording, not just the news.** A rule with `on: [call.complete]`,
  `attach_audio: true` and a `min_duration_ms` sends the finished file to a
  `discord`, `telegram` or `webhook` channel.
- **The rig itself.** `on: [cc.lost]` to your phone tells you the control
  channel dropped before you notice the silence.
- **Encryption appearing.** `encrypted: only` on a `call.start` rule, or
  `on: [call.encryption]` for the algorithm and key ID.
- **A scan-list channel.** Filter on the channel's `talkgroup_id` with
  `on: [call.start]`. A conventional call's grant carries no source radio,
  so `radios` cannot match there.
- **A phrase.** `on: [call.transcript]` with `keywords` fires on what was
  said; that needs
  [Part 7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }}).
- **Everything, for a dashboard.** `mirror_events: true` on the MQTT channel
  also publishes every bus event to `gophertrunk/events/<kind>`, including
  kinds no rule can watch, such as the MDC1200 and pager messages of
  [Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
  and [Part 2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }}).
  The mirror publishes from the manager's own loop, so a broker that hangs
  can make the manager miss events; it still cannot block the decoder.

## Where this goes next

An alert tells you a call happened.
[Part 7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }})
tells you what was said: every finished recording goes to a Whisper server
you choose, the text lands on the call row, and the keyword rule above
gets something to match.

## FAQ

**How do I get a phone notification when a talkgroup keys up?**
Define an `ntfy` (or `pushover`, `telegram`, `gotify`) channel under
`alerts.channels` and a rule under `alerts.rules` with `talkgroups` listing
the IDs and `channels` naming it. Add a `cooldown` so a busy talkgroup
notifies once per window.

**Can a GopherTrunk alert run a script?**
Yes. An `exec` channel runs the program in `command` once per alert, with
the alert as JSON on stdin and the main fields in `GT_ALERT_*` environment
variables. The command is split on whitespace with no shell, so anything
beyond a path and plain arguments belongs inside the script.

**Will a dead webhook or MQTT broker slow down decoding?**
No. Deliveries run on two workers behind a 256-deep queue, each bounded by
the channel's `timeout`. When the queue is full the alert is dropped and
counted; the event bus and the decoder are never made to wait.

**Is the alerts subsystem verified on air?**
It is synthetic-verified: rules, cooldowns and each channel type are
tested against local stand-ins for the services. The project's notes
record no operator field report yet.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Tuning the Sound — Voice Profiles, Loudness and FLAC]({{ '/blog/tutorials/cookbook-volume-2-05-tuning-the-sound/' | relative_url }})
· Next →
[Part 7: Transcription — Whisper on Every Call]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }})
