---
layout: page
title: Alerts & Notifications
description: Alert rules that watch the event bus (talkgroup key-ups, emergencies, encrypted calls, tone-outs, control-channel loss) and deliver to Discord, Slack, ntfy, Pushover, Telegram, Gotify, a webhook, a command or MQTT
nav_group: Reference
---

# Alerts & Notifications

GopherTrunk's `alerts:` section is the "tell me when…" feature of a
hardware scanner — Uniden's custom alerts, Whistler's alert LED, SDRTrunk's
alias actions — pointed at the places you actually look: a phone
notification, a chat channel, a home-automation bus. It has two halves:

- **Channels** are destinations: `discord`, `slack`, `ntfy`, `pushover`,
  `telegram`, `gotify`, a generic JSON `webhook`, a local `exec` command, or
  an `mqtt` broker.
- **Rules** say which events, under what conditions, go to which channels —
  with an optional cooldown and message template.

Rules and channels are decoupled, so one rule can fan out to several
channels and one channel can serve many rules.

```yaml
alerts:
  channels:
    - name: phone
      type: ntfy
      url: https://ntfy.sh/my-scanner-topic
      priority: 4
    - name: ops-discord
      type: discord
      url: https://discord.com/api/webhooks/…
  rules:
    - name: Fire dispatch
      talkgroups: [1001, 1002]
      cooldown: 30s
      channels: [phone, ops-discord]
    - name: Any emergency
      on: [call.start, grant]
      emergency: true
      channels: [phone]
```

## Events a rule can watch (`on:`)

| Kind | Fires when | Notable fields |
|---|---|---|
| `call.start` (default) | a voice SDR starts following a call | system, talkgroup, source, emergency, encrypted, frequency |
| `call.end` | the call is released | duration, end reason |
| `call.complete` | the recording is finished on disk — the only kind that can **attach audio** | audio path, duration |
| `grant` | a control-channel grant is decoded (followed or not) | as call.start |
| `tone.alert` | a `tone_out` profile matches | profile, alpha tag, tone frequencies |
| `cc.locked` / `cc.lost` | a control channel locks / drops | system, frequency |
| `affiliation` / `registration` | a radio joins a talkgroup / registers | source, talkgroup |
| `patch` | a patch / supergroup is announced | supergroup, members |
| `call.encryption` | a call's algorithm / key id is decoded mid-call | algorithm, key id |
| `talker.alias` | an over-the-air alias decodes | alias |
| `location` | a subscriber GPS fix decodes | latitude, longitude |

## Conditions

`systems`, `talkgroups` (a patched supergroup matches when any member is
listed), `radios` (source IDs), `emergency: true`, `encrypted: only |
exclude`, `tone_profiles` (tone.alert), `min_duration_ms` (call.end /
call.complete). `cooldown: 30s` suppresses repeat firings for the same
system + talkgroup (or tone profile) inside the window, so a busy dispatch
channel does not page you on every over while a different talkgroup still
gets through.

## Messages

Each delivery carries a short title ("Call", "EMERGENCY call",
"Tone-out", "Control channel LOST", …) and a one-line text. The built-in
text reads like `Metro P25 · TG FIRE DISP (1001) · from 70001 · 851.0125
MHz`. Override it per rule with a Go `text/template`:

{% raw %}
```yaml
message: "{{.System}} TG {{.TalkgroupAlpha}} ({{.Talkgroup}}) from {{.SourceLabel}}{{if .Emergency}} EMERGENCY{{end}}"
```
{% endraw %}

Fields: `.Kind .System .Protocol .Talkgroup .TalkgroupAlpha .TalkgroupLabel
.PatchedGroups .Source .SourceAlpha .SourceLabel .FrequencyHz .FrequencyMHz
.Emergency .Encrypted .Algorithm .KeyID .Individual .Timeslot
.DurationSeconds .EndReason .AudioPath .ToneProfile .ToneAlpha .ToneHz
.Alias .Latitude .Longitude .Device .At`.

## Channels

| Type | Fields | Notes |
|---|---|---|
| `discord` | `url` | incoming webhook; `attach_audio` uploads the recording as a file |
| `slack` | `url` | incoming webhook |
| `ntfy` | `url` (topic URL), `token` or `user`+`password`, `priority` 1–5 | tags: 🚨 emergency, 🚒 tone-out, ⚠ cc lost, 🔒 encrypted |
| `pushover` | `token` (application), `user` (user/group key), `priority` −2…2 | priority 2 sends retry/expire for the emergency mode |
| `telegram` | `token` (bot), `user` (chat id) | `attach_audio` uses sendAudio |
| `gotify` | `url` (server), `token` (application), `priority` 0–10 | |
| `webhook` | `url`, `token` (Bearer) | JSON body = the whole notification (`rule`, `title`, `text`, `event{…}`); with audio, multipart `alert` + `audio` |
| `exec` | `command` | argv split on whitespace (no shell); alert JSON on stdin, `GT_ALERT_RULE/TITLE/TEXT/KIND/SYSTEM/PROTOCOL/TALKGROUP/TALKGROUP_ALPHA/SOURCE/FREQUENCY_HZ/EMERGENCY/ENCRYPTED/AUDIO_PATH/TONE_PROFILE/AT` in the environment |
| `mqtt` | `url` (`tcp://host:1883`, `ssl://host:8883`), `user`, `password`, `topic` prefix | QoS 0; alerts on `<prefix>/alerts/<rule>`; `mirror_events: true` also publishes **every** bus event as JSON on `<prefix>/events/<kind>` — the trunk-recorder MQTT-plugin shape for Home Assistant / Node-RED |

`timeout` (default `10s`) bounds one delivery. Deliveries run on a small
worker pool behind a bounded queue: a dead destination can never stall the
decoder, and dropped alerts are counted.

## Checking it works

- `GET /api/v1/alerts` — compiled rules and channels with sent / failed /
  last-error counters, cooldown suppressions, and the most recent firings.
- `POST /api/v1/alerts/test/<channel>` — sends a synthetic "GopherTrunk
  test alert" through one channel right now (needs the API write gate).

## Relationship to `broadcast:`

`broadcast.webhook` posts one JSON object per **completed call** (optionally
with the MP3) to a fixed endpoint and `broadcast.grant_webhook` one per
grant — a firehose for your own database. Alerts are the opposite shape:
few, filtered, human-readable events to places people read. Use both.
