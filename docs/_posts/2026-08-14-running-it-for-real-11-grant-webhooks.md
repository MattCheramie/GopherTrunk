---
title: "Running It For Real, Part 11: Grant Webhooks & External Integrations"
description: Firing an HTTP webhook the instant a control-channel grant is decoded — the push form of GET /api/v1/grants for home automation, dashboards, and alerting, with a bounded queue that drops before it ever stalls the decoder, a shared JSON schema, and per-grant retry/backoff.
category: deep-dives
keywords: grant webhook, control channel grant, http webhook sdr, home automation trunking, push notifications scanner, bounded queue drop, json event schema, external integrations, gophertrunk running it for real
tags: [running-it-for-real, broadcast, webhook, integrations, operations, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 11
---

*Part 11 of **Running It For Real**. The four audio backends deliver a finished
call — minutes after the fact, once the voice decode has landed. But some
integrations can't wait that long: a dashboard that lights a talkgroup the instant
it keys up, a home-automation rule that flips a light when your agency dispatches,
an alerter that pages you on an emergency grant. Those want the *grant* — the
control-channel event that says "talkgroup T is now on frequency F" — pushed the
moment it's decoded, before any audio exists. On the laptop you'd poll an endpoint.
On a 24/7 service you want a push that survives a wedged consumer without ever
touching the decoder behind it. That's the grant webhook.*

> **TL;DR:** `GrantWebhook` POSTs one JSON object per decoded control-channel grant
> to your endpoint as it lands — the **push form of `GET /api/v1/grants`**, same
> schema, plus an `"event": "grant"` discriminator and a decode timestamp. It
> subscribes to the bus at construction (so grants decoded before `Run` starts
> aren't lost), delivers on a single worker with retry/backoff, and — the load-bearing
> property — sits behind a **bounded 256-deep queue that drops the overflow rather
> than blocking the event loop and the decoder behind it**. It carries no audio: a
> grant is a control-channel fact, not a recording, and the source RID is reported
> exactly as decoded (never backfilled), which makes it distinct from the
> completed-call webhook.

**Key takeaways**

- **Fires at call setup, not call end.** The grant webhook pushes the instant the
  control channel decodes a grant — seconds before the completed-call webhook, which
  waits for the voice decode.
- **One schema across three surfaces.** The payload mirrors the `GrantDTO` that
  `GET /api/v1/grants` and the `KindGrant` SSE stream already publish, so a consumer
  reads one shape whether it snapshots, subscribes, or receives a push.
- **A wedged consumer can't hurt the decoder.** The bounded queue drops overflow
  grants and increments a counter; it never applies backpressure to the event loop.
- **Source RID is reported, not invented.** A grant whose TSBK carried `src=0` is
  POSTed with `source_id=0` — the feed reports what the control channel actually saw,
  unlike the completed-call webhook whose source depends on a separate voice decode.

## Cheat sheet

| Concern | Where it lives | Operator lever |
|---|---|---|
| Config | `GrantWebhookConfig{URL, AuthHeader, Systems}` | `broadcast.grant_webhook[]` |
| Subscription | `NewGrantWebhook` (subscribes at construction) | — |
| Delivery loop | `worker` → `sendWithRetry` → `post` | `MaxRetries`, `RetryBase` |
| Bounded queue | `jobs` chan, depth `grantWebhookQueueDepth` = 256 | (drops on overflow) |
| Auth | `AuthHeader` → verbatim `Authorization` header | `auth_header: "Bearer …"` |
| Schema | `grantWebhookPayload` (mirrors `api.GrantDTO`) | — |
| Health | `Stats()` → `queued`/`dropped`/`sent`/`failed` | `GET /api/v1/broadcast` |

## In this post

- **Grant vs call** — two webhooks, two moments, two truths about the source.
- **The schema** — one shape shared with the REST snapshot and the SSE stream.
- **The bounded queue** — the rule that keeps a bad consumer off the decoder.
- **Delivery** — subscribe-at-construction, single worker, retry/backoff.
- **Operating it** — auth, filtering, and the counters that flag a dead endpoint.

## Grant vs call: two webhooks, two moments

GopherTrunk has two JSON webhooks and it's worth being precise about the
difference, because they answer different questions. The **completed-call
webhook** (covered in [Recording, Composition &
Streaming]({{ '/blog/series/recording-streaming/' | relative_url }}) Part 14)
fires when a call *finishes* — it can carry the audio, base64-encoded, opt-in. The
**grant webhook** fires when the control channel *decodes a grant* — at call setup,
before a single voice frame exists. If you're driving a live board or a
low-latency alert, you want the grant; if you're archiving finished audio to a
custom sink, you want the call.

That timing difference produces a subtle but real semantic difference in the
source radio ID, and the code comments it deliberately:

```go
// internal/broadcast/grantwebhook.go (shape)
// Grants are surfaced exactly as decoded off the control channel: the source
// RID rides the grant at call-setup time for every protocol, and a grant whose
// TSBK carried src=0 is POSTed with source_id=0 (never backfilled), so the feed
// reports what the control channel actually saw — the way SDRtrunk's grant log
// does. That makes it distinct from the completed-call webhook, whose source
// RID depends on a separate voice-side decode landing.
```

So the grant webhook is *honest about the control channel*: `source_id=0` means
"the grant didn't carry a source," not "we don't know yet." The completed-call
webhook, by contrast, may fill the source from the voice-side decode. Neither is
wrong — they're reporting different facts at different moments, and knowing which
you're consuming keeps a dashboard from mislabelling a call.

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="A timeline of one call. At call setup the control channel decodes a grant and the grant webhook fires immediately with source-id as decoded. Voice frames then flow for the duration of the call. At call end the completed-call webhook fires with the finished recording and optional audio. The grant webhook is early and audio-free; the call webhook is late and can carry audio.">
  <line x1="40" y1="80" x2="620" y2="80" stroke="var(--fg-muted)"/>
  <polygon points="620,76 630,80 620,84" fill="var(--fg-muted)"/>
  <line x1="120" y1="70" x2="120" y2="90" stroke="var(--accent)"/>
  <text x="120" y="60" text-anchor="middle" fill="var(--accent)" font-size="10">grant decoded</text>
  <rect x="60" y="98" width="120" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="120" y="117" text-anchor="middle" fill="var(--accent)" font-size="9">grant webhook →</text>
  <text x="330" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">voice frames flow (seconds to minutes)</text>
  <line x1="200" y1="80" x2="470" y2="80" stroke="currentColor" stroke-dasharray="3 3"/>
  <line x1="520" y1="70" x2="520" y2="90" stroke="currentColor"/>
  <text x="520" y="60" text-anchor="middle" fill="currentColor" font-size="10">call ends</text>
  <rect x="470" y="98" width="140" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="540" y="113" text-anchor="middle" fill="currentColor" font-size="9">call webhook →</text>
  <text x="540" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+ optional audio</text>
</svg>
<figcaption>The grant webhook fires at call setup with the control channel's view; the completed-call webhook fires at call end with the finished recording. Same call, two moments, two truths.</figcaption>
</figure>

## One schema, three surfaces

The grant payload isn't a bespoke shape — it's the same `GrantDTO` that the REST
snapshot and the live SSE stream already publish, plus a discriminator and a
timestamp. That's a deliberate integration property: a consumer that already
parses `GET /api/v1/grants` needs no new code to consume the push:

```go
// internal/broadcast/grantwebhook.go (shape)
type grantWebhookPayload struct {
    Event       string `json:"event"` // always "grant"
    System      string `json:"system"`
    Protocol    string `json:"protocol"`
    GroupID     uint32 `json:"group_id"`
    SourceID    uint32 `json:"source_id"`
    FrequencyHz uint32 `json:"frequency_hz"`
    ChannelID   uint8  `json:"channel_id,omitempty"`
    RFSSID      uint8  `json:"rfss_id,omitempty"`
    SiteID      uint8  `json:"site_id,omitempty"`
    NAC         uint16 `json:"nac,omitempty"`
    Encrypted   bool   `json:"encrypted,omitempty"`
    Emergency   bool   `json:"emergency,omitempty"`
    DataCall    bool   `json:"data_call,omitempty"`
    Individual  bool   `json:"individual,omitempty"`
    AlgorithmID uint8  `json:"algorithm_id,omitempty"`
    KeyID       uint16 `json:"key_id,omitempty"`
    At          string `json:"at"` // RFC3339 decode time
}
```

Two fields carry most of the automation value. `Emergency` is `true` when the
grant flagged an emergency — the natural trigger for a page or an alert.
`Encrypted` (with `AlgorithmID` / `KeyID`) lets a dashboard mark a call it can hear
about but can't hear. The P25 site-identity fields (`nac`, `rfss_id`, `site_id`)
are `omitempty`, so their *presence* means "this value is known" — a consumer
treats a missing `nac` as "not a P25 grant," not "NAC zero." And `at` is an
offset-bearing RFC3339 in the operator's configured timezone, so a grant's
timestamp is unambiguous even across zones — the same timezone helper the audio
webhooks and the call log use, so one call reads with one wall-clock everywhere.

## The bounded queue: the rule that protects the decoder

This is the part that makes a grant webhook safe on a 24/7 daemon rather than a
liability. A busy control channel emits **thousands of grants an hour**. If your
webhook endpoint wedges — the receiving service restarts, its disk fills, the
network partitions — a naive design would block on the POST, and that backpressure
would climb straight back through the event loop into the decoder. So the grant
webhook is built to fail *forward*:

```go
// internal/broadcast/grantwebhook.go (shape)
func (w *GrantWebhook) enqueue(g trunking.Grant) {
    if !w.filter.Accepts(g.System) {
        return
    }
    p := grantWebhookPayloadFrom(g, at, w.loc)
    select {
    case w.jobs <- p: // room in the 256-deep queue
        w.mu.Lock(); w.queued++; w.mu.Unlock()
    default: // queue full — a wedged endpoint — drop the OLDEST discipline
        w.mu.Lock(); w.dropped++; w.mu.Unlock()
        w.log.Warn("broadcast/grantwebhook: queue full, dropping grant",
            "name", w.name, "system", g.System, "tg", g.GroupID)
    }
}
```

The `default` branch is the whole point: a non-blocking send onto a bounded
`jobs` channel means a full queue **drops the grant and moves on** instead of
stalling. This is the exact same discipline the call Manager uses — bound the
backlog, shed load under pressure, never let an outbound sink apply backpressure
to decode. The queue is 256 deep, which absorbs a brief endpoint hiccup without
dropping anything; past that, a chronically dead endpoint costs you dropped grants
(counted, logged) but *never* a missed control-channel decode or a stalled
recorder.

### How that principle shaped the Go code

- **Subscribe at construction, not at `Run`.** `NewGrantWebhook` calls
  `Bus.Subscribe()` and starts the worker before returning, so grants decoded in the
  window between wiring the sink and calling `Run` are queued, not lost. Start-up
  ordering can't drop the first grants of a session.
- **One worker, serial delivery.** A single goroutine drains `jobs` and calls
  `sendWithRetry`, so grants are delivered in order and a slow endpoint can't spawn
  unbounded in-flight POSTs — the queue is the only buffer, and it's bounded.
- **Drop is a counter, not a panic.** Overflow increments `dropped` and warns;
  there's no error path back to the caller, because the caller is the event loop and
  it must never block.

## Delivery: subscribe, retry, back off

Once a grant is on the queue, delivery is the same retry discipline you saw for the
audio backends, scaled to a single worker. `sendWithRetry` attempts the POST up to
`maxRetries+1` times (default 4) with exponential backoff (default 2s, doubling),
each attempt bounded by a 30-second timeout:

```go
// internal/broadcast/grantwebhook.go (shape)
func (w *GrantWebhook) sendWithRetry(p grantWebhookPayload) {
    backoff := w.retryBase
    for attempt := 0; attempt <= w.maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        err := w.post(ctx, p)
        cancel()
        if err == nil {
            w.mu.Lock(); w.sent++; w.mu.Unlock()
            return
        }
        // …warn, then sleep backoff and double it
    }
    w.mu.Lock(); w.failed++; w.mu.Unlock()
    w.log.Error("broadcast/grantwebhook: giving up on grant", ...)
}
```

There's a design tension worth naming here. The worker is *single* and delivery is
*serial*, so while `sendWithRetry` is backing off (2s + 4s + 8s = up to 14 seconds
for one wedged grant), it isn't draining the queue. That's intentional: it couples
"the endpoint is slow" to "the queue fills," which trips the bounded-queue drop —
so a dead endpoint self-limits to dropped grants rather than an ever-growing
backlog of doomed retries. The retry budget handles a *flaky* endpoint; the bounded
queue handles a *dead* one. Together they mean the grant webhook needs no operator
intervention when a consumer misbehaves.

## Operating it: auth, filtering, and health

The config surface is small. `url` is required; `auth_header` is sent verbatim as
the `Authorization` header — so `auth_header: "Bearer <token>"` authenticates to
your endpoint, and (as with every credential in this series) it belongs in an
environment file the systemd unit references, not inline in `config.yaml`. The
`systems` list scopes the feed to named trunked systems; empty pushes every
system. A feed with `enabled: false` is parsed but skipped, so you can stage a
webhook before turning it on.

Health reads through the same door as the audio backends —
`GET /api/v1/broadcast`, which returns `{"enabled": false}` when nothing is
configured so a UI renders a stable shape — and the grant webhook contributes its
own four counters:

```go
// internal/broadcast/grantwebhook.go (shape)
type GrantWebhookStats struct {
    Name    string `json:"name"`
    Queued  int    `json:"queued"`
    Dropped int    `json:"dropped"` // overflow — a wedged endpoint
    Sent    int    `json:"sent"`
    Failed  int    `json:"failed"`  // exhausted retries
}
```

The operating read: `sent` climbing with your grant rate is healthy. A rising
`dropped` means the endpoint is chronically slow and the queue is overflowing — the
consumer can't keep up. A rising `failed` means individual POSTs are exhausting
their retries — usually a wrong `auth_header` or a 4xx from the endpoint. As
everywhere in this series, these are numbers you *alert on*, because a grant
webhook that stops firing is otherwise completely silent — the decoder keeps
decoding, the counters just stop moving.

## Where this goes next

We've now built the whole outbound surface — audio to four aggregators, grants to
a webhook — on top of a daemon that so far runs wherever you launch it. The last
three posts make *where it runs* robust.
[Part 12]({{ '/blog/deep-dives/running-it-for-real-12-docker-usb/' | relative_url }})
containerizes the daemon and passes an RTL-SDR through to it — the udev rules, the
`--device` mapping, and the USB-reset dance that separate a container that sees a
dongle from one that just thinks it should. Cross-reference the
[Containers & Deployment]({{ '/learn/deployment/' | relative_url }}) module for
first principles.

## FAQ

**When exactly does the grant webhook fire relative to the call?**
At call setup — the instant the control channel decodes the grant, before any voice
frame is processed and seconds before the completed-call webhook. If you want the
earliest possible signal that a talkgroup keyed up (for a live board or a
low-latency alert), it's the grant webhook, not the call webhook.

**Why does a grant sometimes have `source_id: 0`?**
Because the grant genuinely carried no source RID. The feed reports the control
channel exactly as decoded and never backfills, so `source_id: 0` means "not
present in the grant," not "unknown, check back later." The completed-call webhook may
fill the source from the voice decode instead — a different fact at a different
moment.

**What happens to grants if my endpoint goes down?**
Individual POSTs retry up to four times with exponential backoff. If the endpoint
stays down, the single delivery worker stalls on backoff, the 256-deep queue fills,
and further grants are dropped (counted in `dropped`, logged as warnings) — never
queued unbounded and never allowed to stall the decoder. When the endpoint recovers,
delivery resumes; the dropped grants are gone, by design.

**Can I consume the same data without a webhook?**
Yes — that's the point of the shared schema. `GET /api/v1/grants` gives the snapshot
and the `KindGrant` SSE stream gives a live feed, both in the same `GrantDTO` shape
plus the webhook's `event`/`at` fields. The webhook is the push option for consumers
that can't poll or subscribe.

**Does the grant webhook carry audio?**
No. A grant is a control-channel event with no recording attached — audio doesn't
exist yet at call setup. If you need the audio, use the completed-call webhook with
`include_audio: true`, which base64-embeds the finished MP3.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Broadcast Backends II — RdioScanner, OpenMHz, Icecast]({{ '/blog/deep-dives/running-it-for-real-10-broadcast-backends-ii/' | relative_url }})
· Next →
[Part 12: Docker & RTL-SDR USB Pass-Through]({{ '/blog/deep-dives/running-it-for-real-12-docker-usb/' | relative_url }})
