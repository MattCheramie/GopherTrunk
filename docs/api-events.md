---
layout: page
title: Event API & webhooks
description: The real-time telemetry surface — SSE/WebSocket event streams, the stable JSON event envelope and payload schema, and the per-call / per-grant webhook sinks
nav_group: Operate
---

# Event API & webhooks

GopherTrunk publishes every decode event on an internal bus and exposes it four
ways: a **Server-Sent Events** stream, a **WebSocket** stream, an optional
append-only **JSONL event log**, and outbound **webhook** POSTs. The three
stream/log surfaces all carry the *same* [event envelope](#event-envelope); the
webhooks are a separate, flatter push shape documented [below](#webhooks).

This page is the stable JSON contract downstream consumers (Grafana, Prometheus
exporters, dashboards) build against.

- [Transports](#transports)
- [Event envelope](#event-envelope)
- [Event kinds](#event-kinds)
- [Stable payload schema](#stable-payload-schema)
- [Webhooks](#webhooks)
- [Configuration](#configuration)
- [Reliability & caveats](#reliability--caveats)

## Transports

### Server-Sent Events — `GET /api/v1/events`

A long-lived `text/event-stream`. On connect the server sends one comment line
(`: gophertrunk events stream`) and then one SSE record per event:

```
event: call.start
data: {"kind":"call.start","timestamp":"2026-08-03T17:00:00Z","payload":{ ... }}

```

The `event:` line carries the [event kind](#event-kinds); browsers can dispatch
per kind with `addEventListener("grant", …)`. The `data:` payload is the JSON
[envelope](#event-envelope). A payload containing embedded newlines is split
across multiple `data:` lines per the SSE spec (reassemble by concatenation).

```
curl -N http://localhost:8080/api/v1/events
```

There is **no server-side filtering** — every subscriber receives every kind and
filters client-side. There is no periodic keepalive on SSE (only the initial
comment line); use the WebSocket transport if you need a ping.

### WebSocket — `GET /api/v1/events/ws`

The same stream as one JSON **text frame** per event (identical body to the SSE
`data:` payload). The connection is server→client only; any frames a client
sends are discarded. The server sends a WebSocket **ping every 30 s** so idle
connections and proxies stay open. The browser console uses this transport
(the `EventSource` API can't set headers).

```
websocat ws://localhost:8080/api/v1/events/ws
```

### Event log (JSONL)

When `log.event_log.enabled` is set, GopherTrunk mirrors **every** bus event to
an append-only JSONL file in the exact same [envelope](#event-envelope) the
streams emit — an offline/replayable copy of the live feed. See
[configuration](#configuration).

### Webhooks

Outbound `POST` sinks that push a completed call (or a decoded grant) to a URL
you configure, rather than a stream you subscribe to. See [Webhooks](#webhooks).

## Event envelope

Every streamed event (SSE `data:`, WS frame, and JSONL line) is this envelope:

| field | type | notes |
|---|---|---|
| `kind` | string | the event kind (equals the `event:` line on SSE) |
| `timestamp` | RFC3339 string | bus publish time |
| `payload` | object | kind-specific body — see [payload schema](#stable-payload-schema) |

`timestamp` and all `time.Time` payload fields serialize as RFC3339. Non-finite
floats (±Inf/NaN) are scrubbed from payloads before marshaling so a marginal
metric can't break the stream.

## Event kinds

Payloads come in **two tiers**:

1. **Stable DTOs** — ten kinds have a curated, versioned JSON payload defined in
   the API package. These are the contract you should build against; their field
   names will not change without notice. Documented in full
   [below](#stable-payload-schema).

2. **Passthrough kinds** — every other kind serializes its underlying internal
   payload struct directly. These are useful but **less stable**: their JSON
   tracks the internal type, and a few (e.g. `channel.power`, `decode.error`)
   have no JSON tags and therefore emit Go-capitalized field names. Treat them as
   best-effort. For a passthrough kind's current shape, read the field off a live
   stream rather than pinning it.

Stable kinds: `grant`, `call.start`, `call.end`, `call.encryption`,
`affiliation`, `registration`, `unit.request`, `patch`, `dmr.grant.observed`,
`dmr.bandplan.learned`.

Passthrough kinds include: `sdr.attached`, `sdr.detached`, `cc.locked`,
`cc.lost`, `call.complete`, `call.segment`, `call.source`, `call.release`,
`call.talker`, `grant.unserved`, `tone.alert`, `decode.error`, `error`,
`talker.alias`, `location`, `site.update`, `patch`, `audio.state`,
`channel.power`, `bookmark.created`, `bookmark.updated`, `bookmark.deleted`,
`pager.message`, `aprs.packet`, `ais.message`, `dsc.message`, `adsb.aircraft`,
`mdc1200.message`, `m17.linksetup`, `lora.frame`, `cchunt.progress`,
`cchunt.failed`, `hunt.progress`, `hunt.candidate`, `hunt.done`,
`unit.request`, `dmr.grant.observed`, `dmr.bandplan.learned`.

The in-call passthrough kinds `call.source`, `call.talker`, `call.release` and
`call.segment` now carry snake_case field names matching the grant DTO
(`system`, `group_id`, `source_id`, `frequency_hz` on `call.source`, `at`, …)
so activity feeds can render them with the same formatter as `grant`. They
remain passthrough (best-effort) kinds; earlier releases emitted Go-capitalized
field names for them.

## Stable payload schema

Field types are the JSON types the Go structs marshal to. `omitempty` fields are
absent (not zero) when unset.

### `grant`

Fires when the control channel decodes a voice/data channel grant.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `protocol` | string | |
| `group_id` | number | talkgroup — or the destination unit RID when `individual` is true |
| `source_id` | number | source RID as the grant carried it (0 if none; not voice-backfilled) |
| `frequency_hz` | number | |
| `channel_id` | number | *omitempty* |
| `channel_number` | number | *omitempty* |
| `rfss_id` | number | *omitempty* — P25 site identity |
| `site_id` | number | *omitempty* |
| `nac` | number | *omitempty* — P25 network access code; on Phase 2 the NSB colour code. Absent on non-P25 grants (read "absent" as "not a P25 grant", never as a P25 grant that lost its NAC) |
| `timeslot` | number | *omitempty* — 1-based (DMR Tier III) |
| `encrypted` | bool | *omitempty* |
| `emergency` | bool | *omitempty* |
| `data_call` | bool | *omitempty* |
| `individual` | bool | *omitempty* — `group_id` is a unit, not a talkgroup |
| `algorithm_id` | number | *omitempty* |
| `key_id` | number | *omitempty* |

### `call.start`

| field | type | notes |
|---|---|---|
| `grant` | object | a [`grant`](#grant) payload |
| `talkgroup` | object | *omitempty* — see [Talkgroup](#talkgroup-object) |
| `device_serial` | string | the SDR that carried the call |
| `started_at` | RFC3339 string | |

### `call.end`

| field | type | notes |
|---|---|---|
| `grant` | object | a [`grant`](#grant) payload |
| `talkgroup` | object | *omitempty* — see [Talkgroup](#talkgroup-object) |
| `device_serial` | string | |
| `started_at` | RFC3339 string | |
| `ended_at` | RFC3339 string | |
| `duration_ms` | number | call length in ms (`ended − started`); `0` when timestamps aren't sane (e.g. a watchdog teardown). Lets an SSE/WS-only consumer read duration without pairing back to `call.start` |
| `reason` | string | why the call ended |

> `call.end` is the stream's completion event. `call.complete` also exists but is
> a passthrough kind; prefer `call.end` for a stable duration/reason.

### `call.encryption`

Emitted when a call is observed to be encrypted.

| field | type | notes |
|---|---|---|
| `device_serial` | string | |
| `system` | string | *omitempty* |
| `protocol` | string | *omitempty* |
| `group_id` | number | *omitempty* |
| `algorithm_id` | number | |
| `key_id` | number | |
| `at` | RFC3339 string | |

### `affiliation`

P25 group affiliation response.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `protocol` | string | |
| `source_id` | number | affiliating unit |
| `group_id` | number | talkgroup |
| `announcement_group` | number | *omitempty* |
| `response` | string | `accepted` / `denied` / `refused` / `failed` |
| `rfss_id` | number | *omitempty* |
| `site_id` | number | *omitempty* |
| `nac` | number | *omitempty* |

### `registration`

P25 unit registration response.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `protocol` | string | |
| `source_id` | number | registering unit |
| `wacn` | number | |
| `system_id` | number | |
| `response` | string | `accepted` / `denied` / `refused` / `failed` |
| `rfss_id` | number | *omitempty* |
| `site_id` | number | *omitempty* |
| `nac` | number | *omitempty* |

### `unit.request`

Unit-to-unit (private call) request.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `protocol` | string | |
| `source_id` | number | |
| `target_id` | number | |
| `service_options` | number | *omitempty* |

### `patch`

Talkgroup patch / supergroup activation or cancellation.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `protocol` | string | |
| `super_group` | number | the patch supergroup |
| `members` | array of number | member talkgroups |
| `vendor` | string | *omitempty* |
| `add` | bool | `true` = activate, `false` = cancel |
| `at` | RFC3339 string | |

### `dmr.grant.observed`

A DMR Tier III channel grant observed on the control channel.

| field | type | notes |
|---|---|---|
| `system` | string | |
| `color_code` | number | |
| `lcn` | number | logical channel number |
| `timeslot` | number | raw CSBK value: `0` = TS1, `1` = TS2 |
| `group_id` | number | |
| `source_id` | number | |
| `cc_freq_hz` | number | control-channel frequency |
| `at` | RFC3339 string | |

### `dmr.bandplan.learned`

The learned DMR band plan (LCN → frequency mapping).

| field | type | notes |
|---|---|---|
| `system` | string | |
| `base_hz` | number | *omitempty* |
| `spacing_hz` | number | *omitempty* |
| `offset` | number | *omitempty* |
| `table` | array | *omitempty* — each `{ "lcn": number, "freq_hz": number }` |
| `num_pairs` | number | |
| `confidence` | number | |
| `residual_hz` | number | *omitempty* |

### Talkgroup object

Nested in `call.start` / `call.end` as `talkgroup` (from the talkgroup alias
table). Present only when the talkgroup is known.

| field | type | notes |
|---|---|---|
| `id` | number | |
| `alpha_tag` | string | |
| `description` | string | *omitempty* |
| `tag` | string | *omitempty* |
| `group` | string | *omitempty* |
| `mode` | string | *omitempty* |
| `priority` | number | *omitempty* |
| `lockout` | bool | *omitempty* |
| `scan` | bool | |
| `stream` | bool | |
| `record` | bool | |
| `mute` | bool | |
| `icon` | string | *omitempty* |
| `discovered` | bool | true if auto-discovered rather than from the alias file |

## Webhooks

Two independent outbound sinks. Both `POST` JSON, set `Content-Type:
application/json`, and set an `Authorization` header verbatim from `auth_header`
when configured. Neither is served over the SSE/WS stream.

### Per-call webhook

Fires **once per completed call** (after the recording is flushed). Configured
under `broadcast.webhook`. Retries up to **3 times** (4 attempts total) with
exponential backoff starting at 2 s; a non-2xx response counts as a failure.

Payload:

| field | type | notes |
|---|---|---|
| `event` | string | always `"call"` |
| `system` | string | |
| `protocol` | string | |
| `call_type` | string | `group` / `unit` / `data` |
| `talkgroup` | number | |
| `talkgroup_label` | string | *omitempty* |
| `source` | number | *omitempty* — source RID at end of call (voice-backfilled) |
| `frequency_hz` | number | |
| `channel_id` | number | *omitempty* |
| `rfss_id` | number | *omitempty* |
| `site_id` | number | *omitempty* |
| `nac` | number | *omitempty* |
| `timeslot` | number | *omitempty* |
| `encrypted` | bool | |
| `algorithm_id` | number | *omitempty* |
| `key_id` | number | *omitempty* |
| `emergency` | bool | |
| `patched_groups` | array of number | *omitempty* |
| `started_at` | RFC3339 string | in the configured display timezone |
| `ended_at` | RFC3339 string | |
| `duration_ms` | number | |
| `audio_filename` | string | *omitempty* |
| `audio_format` | string | *omitempty* — `mp3` when audio is embedded |
| `audio_base64` | string | *omitempty* — base64 MP3, only when `include_audio: true` |

### Per-grant webhook

Fires **once per decoded control-channel grant** — the push form of
`GET /api/v1/grants` and the `grant` stream event. Configured under
`broadcast.grant_webhook`. Uses a bounded internal queue (depth 256; overflow
drops the newest grant rather than blocking the decoder), retries up to **3
times** with 2 s exponential backoff, 30 s timeout per attempt.

The payload is the [`grant`](#grant) schema field-for-field, plus:

| field | type | notes |
|---|---|---|
| `event` | string | always `"grant"` |
| `at` | RFC3339 string | decode time, in the display timezone |

The `grant` event, `GET /api/v1/grants`, and this webhook share field names
deliberately. Its `source_id` is what the control channel saw at grant time
(never voice-backfilled) — that is the difference from the per-call webhook's
`source`.

## Configuration

The SSE and WebSocket streams run whenever the HTTP API is enabled — there is no
per-stream toggle.

```yaml
api:
  http_addr: "127.0.0.1:8080"   # enables the HTTP server (SSE + WS live here)
  cors:
    allowed_origins: ["https://dash.example.com"]  # also gates the WS Origin check
  # api.auth gates MUTATIONS only — it does NOT authenticate SSE/WS (see caveats)

log:
  event_log:
    enabled: true               # mirror every event to a JSONL file
    path: "/var/lib/gophertrunk/events.jsonl"
    max_size_mb: 16

broadcast:
  webhook:                       # per completed call
    - enabled: true
      name: "call-hook"
      url: "https://example.com/hooks/call"
      auth_header: "Bearer <token>"   # optional
      include_audio: false            # embed base64 MP3 when true
      systems: []                     # empty = all systems
  grant_webhook:                 # per decoded grant
    - enabled: true
      name: "grant-hook"
      url: "https://example.com/hooks/grant"
      auth_header: "Bearer <token>"
      systems: []
```

## Reliability & caveats

- **The event streams are unauthenticated.** `api.auth` gates mutating endpoints
  only; `GET /api/v1/events` and `/events/ws` are open to anyone who can reach
  the API. If you expose the API beyond localhost, front it with a reverse proxy
  that enforces auth/TLS. See [Hardening & operations](hardening.md).
- **No server-side filtering.** Every subscriber receives every kind; filter
  client-side (by the `kind` field or the SSE `event:` name). There are no
  `?system=` / `?kind=` query parameters.
- **Slow subscribers drop events.** Each subscriber has a bounded buffer; a
  client that can't keep up has events dropped rather than blocking the decoder,
  so a stream is not a guaranteed-delivery log. For a complete record use the
  JSONL event log or a webhook (which retries).
- **Two stability tiers.** Only the ten [stable DTOs](#stable-payload-schema) are
  a versioned contract. Passthrough kinds mirror internal structs and may change.
- **Timezones.** Envelope `timestamp` and stream DTO time fields are RFC3339 in
  UTC as Go marshals them; webhook `at` / `started_at` / `ended_at` are RFC3339
  rendered in the operator's configured display timezone.
