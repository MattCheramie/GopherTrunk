---
title: "Running It For Real, Part 9: Broadcast Backends I — Broadcastify"
description: Operating a Broadcastify Calls feed on a 24/7 daemon — where the API key and system ID live, how the two-leg upload retries with exponential backoff, how a rejection surfaces, and the counters that tell you a feed went quiet before your listeners do.
category: deep-dives
keywords: broadcastify calls feed, broadcastify api key, sdr call upload, retry backoff streaming, feed monitoring, broadcast credentials, upload failure handling, gophertrunk broadcast, gophertrunk running it for real
tags: [running-it-for-real, broadcast, operations, http, monitoring, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 9
---

*Part 9 of **Running It For Real**. On the laptop, "streaming to Broadcastify"
means you pasted a key, saw one call land, and called it done. On a 24/7 daemon
it means something else entirely: the key has to survive a config audit without
leaking, the upload has to ride out a flaky network without blocking the decoder,
and — the part nobody demos — you have to **notice** when the feed goes quiet at
2 a.m. This post is Broadcastify Calls from the operator's chair. The wire format
itself we already dissected in [Recording, Composition &
Streaming]({{ '/blog/series/recording-streaming/' | relative_url }}) (Parts
13–14); here we run it.*

> **TL;DR:** A Broadcastify Calls feed needs two secrets — an `api_key` and a
> numeric `system_id` — and the operating concerns are all downstream of that.
> Each upload is a two-leg exchange (metadata POST → one-time URL → PUT), and a
> failure on **either** leg returns an error that the broadcast Manager retries
> with exponential backoff (default 3 retries, 2s base, doubling). A rejected
> metadata response (`api_key` wrong, `system_id` unknown) is a hard error the
> retries can't fix, so it burns the budget and increments a `failed` counter.
> The single most important operational habit is watching `sent` vs `failed` per
> backend — a feed that stops streaming is silent unless you're reading those.

**Key takeaways**

- **Two secrets, kept out of `config.yaml` if you can.** `api_key` and `system_id`
  identify the feed; the key is the sensitive one. Prefer an environment file the
  systemd unit pulls in over committing it inline.
- **Retries are the Manager's job, not the backend's.** `broadcastifyBackend.Send`
  returns an error and stays stateless; `Manager.sendWithRetry` owns the backoff,
  the per-call timeout, and the give-up decision.
- **A wrong key looks the same as a wedged network — until the budget runs out.**
  Both fail `Send`; the difference is a rejection never succeeds on retry, so it's
  the `failed` counter, not the transient warnings, that names a real outage.
- **Monitor the counters, not the logs.** `Stats()` exposes `sent` and `failed`
  per backend; a healthy feed's `sent` climbs and `failed` stays flat. Alert on
  the delta, because nothing else tells you the feed died.

## Cheat sheet

| Concern | Where it lives | Operator lever |
|---|---|---|
| Credentials | `BroadcastifyConfig{APIKey, SystemID}` | `broadcast.broadcastify[].api_key` / `system_id` |
| System filter | `systemFilter` (`newSystemFilter`) | `systems: [ ... ]` (empty = all) |
| Two-leg upload | `Send` → `requestUploadURL` → `putAudio` | (internal) |
| Rejection parse | `parseBroadcastifyUploadURL` | reads the `"0 <url>"` success shape |
| Retry / backoff | `Manager.sendWithRetry` | `broadcast.workers`, retry base/count |
| Duration gate | `Manager.dispatch` (`MinDuration`) | `broadcast.min_duration_ms` |
| Health | `Manager.Stats()` → `sent`/`failed` | `GET /api/v1/broadcast` |

## In this post

- **The two secrets** — what `api_key` and `system_id` are, and where to keep them.
- **What one upload actually costs** — two legs, and why either can fail.
- **Who retries** — the Manager owns backoff; the backend stays stateless.
- **How a bad key surfaces** — rejection vs transient, and why the difference matters.
- **How you notice** — the counters that turn a silent feed into an alert.

## The two secrets

A Broadcastify Calls feed is identified by exactly two values, and the backend
refuses to construct without both:

```go
// internal/broadcast/broadcastify.go (shape)
func NewBroadcastify(cfg BroadcastifyConfig, hc *http.Client) (Backend, error) {
    if cfg.APIKey == "" {
        return nil, errors.New("broadcast/broadcastify: api_key is required")
    }
    if cfg.SystemID == 0 {
        return nil, errors.New("broadcast/broadcastify: system_id is required")
    }
    // …name defaults to "broadcastify"; endpoint defaults to the production API
}
```

The `system_id` is the numeric Broadcastify Calls system you're feeding — public,
harmless, the thing you'd put in a dashboard. The `api_key` is the secret: anyone
with it can upload to your feed. On a laptop demo you paste it straight into
`config.yaml` and move on. On a hardened deployment that's the wrong instinct —
`config.yaml` gets read by the config-audit endpoints, checked into more places
than you'd like, and copied when you clone a box. The
[hardening guide]({{ '/hardening.html' | relative_url }})'s pattern for the API
token applies here too: keep the secret in an environment file the systemd unit
references (`EnvironmentFile=-/etc/gophertrunk/env`) rather than inline. Note that
`/api/v1/runtime` is deliberately sanitised and never echoes the key — but that's
one endpoint's discipline, not a reason to leave the secret in a widely-read file.

The optional `systems` filter is the other lever worth knowing: an empty list
streams every trunked system the daemon decodes, while a named list restricts the
feed. That's a `systemFilter` built by `newSystemFilter`, checked by
`Accepts(system)` before the call is ever enqueued — so scoping a feed to one
system is free, not a per-call cost.

## What one upload actually costs

Every other audio backend POSTs the MP3 once. Broadcastify splits it in two,
because the audio doesn't go to the API host — the metadata POST returns a
one-time URL (usually object storage) that the MP3 is `PUT` to directly. We
walked that handshake byte-by-byte in [Recording, Composition &
Streaming]({{ '/blog/series/recording-streaming/' | relative_url }}) Part 14; the
operational point is that **there are two independent failure points**, and
`Send` treats either as a reason to fail the whole upload:

```go
// internal/broadcast/broadcastify.go (shape)
func (b *broadcastifyBackend) Send(ctx context.Context, c *Call) error {
    audio, err := c.MP3()
    if err != nil {
        return fmt.Errorf("%s: encode mp3: %w", b.name, err)
    }
    uploadURL, err := b.requestUploadURL(ctx, c) // leg 1: metadata POST
    if err != nil {
        return err
    }
    return b.putAudio(ctx, uploadURL, audio)      // leg 2: PUT to the one-time URL
}
```

Leg one is a form-encoded POST carrying the `apiKey`, `systemId`, the talkgroup,
the source RID, the frequency, and the Unix start time. A non-200 here — or a body
that doesn't parse as the `"0 <url>"` success shape — returns an error. Leg two is
a plain `PUT` of the MP3 bytes; a non-2xx there returns an error too. From the
Manager's perspective both look identical: `Send` returned non-nil, retry.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="A call flows from the CallComplete event into the broadcast Manager, which gates it on stream policy and minimum duration, encodes it to MP3 once, and hands it to a worker. The worker calls the Broadcastify backend's Send, which does the metadata POST and audio PUT. On failure the worker retries with exponential backoff up to the retry budget, then increments the failed counter.">
  <rect x="6" y="66" width="104" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="58" y="86" text-anchor="middle" fill="currentColor" font-size="10">CallComplete</text>
  <text x="58" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">on the bus</text>
  <line x1="110" y1="88" x2="134" y2="88" stroke="currentColor"/><polygon points="134,84 144,88 134,92" fill="currentColor"/>
  <rect x="144" y="54" width="120" height="68" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="204" y="76" text-anchor="middle" fill="var(--accent)" font-size="11">Manager</text>
  <text x="204" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">gate + encode</text>
  <text x="204" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sendWithRetry</text>
  <line x1="264" y1="88" x2="288" y2="88" stroke="currentColor"/><polygon points="288,84 298,88 288,92" fill="currentColor"/>
  <rect x="298" y="54" width="126" height="68" rx="6" fill="none" stroke="currentColor"/>
  <text x="361" y="74" text-anchor="middle" fill="currentColor" font-size="11">Send</text>
  <text x="361" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">POST metadata</text>
  <text x="361" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">PUT audio</text>
  <line x1="424" y1="88" x2="448" y2="88" stroke="currentColor"/><polygon points="448,84 458,88 448,92" fill="currentColor"/>
  <rect x="458" y="54" width="100" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="508" y="73" text-anchor="middle" fill="var(--accent)" font-size="10">sent++</text>
  <rect x="458" y="92" width="100" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="508" y="111" text-anchor="middle" fill="var(--fg-muted)" font-size="10">failed++</text>
  <path d="M 361 122 Q 361 150 300 150 Q 240 150 240 128" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="236,128 240,120 244,128" fill="var(--fg-muted)"/>
  <text x="300" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">error → backoff → retry (2s, 4s, 8s), then give up</text>
</svg>
<figcaption>A completed call is gated, encoded once, and handed to a worker. The backend just tries once and reports; the Manager owns the retry loop and the two counters that record the outcome.</figcaption>
</figure>

## Who retries: the Manager, not the backend

`broadcastifyBackend.Send` is deliberately stateless — it tries once and reports.
All the resilience lives one layer up in `Manager.sendWithRetry`, which is shared
across every backend:

```go
// internal/broadcast/manager.go (shape)
func (m *Manager) sendWithRetry(b Backend, call *Call) {
    backoff := m.retryBase // default 2s
    for attempt := 0; attempt <= m.maxRetries; attempt++ { // default 3 → 4 tries
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        err := b.Send(ctx, call)
        cancel()
        if err == nil {
            m.mu.Lock(); m.sent[b.Name()]++; m.mu.Unlock()
            return
        }
        m.log.Warn("broadcast: upload failed", "backend", b.Name(),
            "attempt", attempt+1, "of", m.maxRetries+1, "err", err)
        if attempt < m.maxRetries {
            time.Sleep(backoff); backoff *= 2 // 2s, 4s, 8s
        }
    }
    m.mu.Lock(); m.failed[b.Name()]++; m.mu.Unlock()
    m.log.Error("broadcast: giving up on call", "backend", b.Name())
}
```

Three operational properties fall out of this. First, each attempt gets its own
**60-second timeout** — a wedged endpoint can't hang a worker forever. Second, the
backoff **doubles** (2s, 4s, 8s), so a brief network blip is absorbed but a dead
endpoint doesn't get hammered. Third, the retry budget is bounded: after
`maxRetries+1` failures the call is dropped and `failed` ticks. The daemon does
**not** queue that call for later — a live feed values freshness over
completeness, and an unbounded retry queue is how a wedged backend takes down the
recorder behind it.

### How that principle shaped the Go code

- **The queue is bounded and drops, not blocks.** `Manager.dispatch` does a
  non-blocking send onto a depth-64 `jobs` channel; when it's full — a backend is
  wedged — the call is dropped and `dropped` increments, rather than stalling the
  event loop and the recorder. Backpressure never propagates backward into decode.
- **Workers are concurrent and per-backend serial.** The default two workers each
  loop over every backend for a call, so a slow Broadcastify upload doesn't hold up
  an RdioScanner one for a *different* call — but within a call, backends run in
  sequence.
- **A duration gate runs before any of this.** `broadcast.min_duration_ms` drops
  squelch-crackle and failed-decode fragments up front in `dispatch`, so the retry
  budget is never spent on a 200 ms non-call.

## How a bad key surfaces

Here's the operationally important subtlety: **a wrong `api_key` and a wedged
network fail the exact same way** — `Send` returns non-nil, the Manager retries.
The difference only reveals itself over time. A network blip succeeds on retry two
or three and never touches `failed`. A rejected key can *never* succeed, because
the metadata POST returns a rejection every single attempt:

```go
// internal/broadcast/broadcastify.go (shape)
func parseBroadcastifyUploadURL(body string) (string, error) {
    fields := strings.Fields(strings.TrimSpace(body))
    if fields[0] == "0" && len(fields) >= 2 {
        return fields[1], nil // "0 <url>" — success
    }
    // anything else — including an auth rejection — is an error
    return "", fmt.Errorf("broadcastify: metadata response rejected: %s", body)
}
```

So the signature of a bad credential is a run of four identical warnings followed
by one `broadcast: giving up on call` error, per call, forever — and a `failed`
counter that climbs in lockstep with your call rate. The signature of a flaky
network is scattered warnings with `failed` staying flat. Reading the *rejection
message* off the warning line (the API echoes why it refused) usually tells you
which: an auth string points at the key, an unknown-system string at the
`system_id`.

## How you notice a feed went quiet

This is the part the laptop demo never teaches. A Broadcastify feed that dies does
so silently — no crash, no listener complaint until much later. The only reliable
signal is the counters, exposed per backend and surfaced over HTTP:

```go
// internal/broadcast/manager.go (shape) — the numbers to alert on
type Stats struct {
    Queued   int            `json:"queued"`
    Dropped  int            `json:"dropped"`
    Sent     map[string]int `json:"sent"`   // per backend name
    Failed   map[string]int `json:"failed"` // per backend name
    Backends []string       `json:"backends"`
}
```

`GET /api/v1/broadcast` returns `{"enabled": true, "stats": ...}` (or
`{"enabled": false}` when no feed is configured, so a UI renders a stable shape).
The operating rule is simple: on a healthy feed `sent["broadcastify"]` climbs with
your call volume and `failed` stays near zero. Alert on the **rate of `failed`**
and on `sent` **going flat** while calls are still being decoded — the first
catches a bad key or dead endpoint, the second catches a subtler stall. `Dropped`
is the third number to watch: a rising `dropped` means the queue is full, i.e. a
backend is chronically slow and the daemon is shedding calls to protect the
recorder. None of these show up in a glance at `journalctl` — you have to be
reading the counters. That's the difference between a demo and a feed you can
trust for six months.

## Where this goes next

Broadcastify is one of four audio destinations, and the other three each bend the
operating story differently.
[Part 10]({{ '/blog/deep-dives/running-it-for-real-10-broadcast-backends-ii/' | relative_url }})
takes RdioScanner and OpenMHz — same Manager, same retry loop, different
credentials and endpoints — and then Icecast, which is a live stream rather than an
upload and therefore drops instead of retrying. The counters you learned here read
the same across all of them.

## FAQ

**Where should the Broadcastify api_key actually live?**
Out of `config.yaml` if you can. Put it in an environment file the systemd unit
pulls in with `EnvironmentFile=-/etc/gophertrunk/env`, the same pattern the hardening
guide uses for the API bearer token. The `system_id` is public and fine inline; the
key is the secret worth isolating from config-audit surfaces.

**How many times does a failed upload retry, and for how long?**
Four attempts total by default (the initial try plus `maxRetries` = 3), with
exponential backoff of 2s, 4s, 8s between them, each attempt bounded by a 60-second
timeout. After that the call is dropped and the `failed` counter increments. Failed
calls are not re-queued — a live feed prioritises freshness.

**My feed just stopped and there's no error — what happened?**
Look at `GET /api/v1/broadcast`. If `sent["broadcastify"]` has gone flat while calls
are still decoding, the feed stalled; if `failed` is climbing, you have a bad key or
a dead endpoint (read the rejection message in the warning log). A silent flatline is
exactly why you monitor the counters instead of the logs.

**Does a short transmission get uploaded?**
Only if it clears `broadcast.min_duration_ms`. The Manager's dispatch gate drops
calls shorter than that up front — squelch crackle and failed decodes — before the
retry machinery ever sees them, so the budget isn't spent on non-calls.

**Can I stream only some systems to Broadcastify?**
Yes. Set `systems: [ ... ]` on the feed to a named list; empty streams every system.
The filter is checked before the call is enqueued, so scoping is free. Individual
talkgroups also opt out of *all* feeds with `stream: false`.

## Series navigation

**Part 9 of 14** · ←
[Part 8: The Opt-In Feature Matrix]({{ '/blog/deep-dives/running-it-for-real-08-opt-in-feature-matrix/' | relative_url }})
· Next →
[Part 10: Broadcast Backends II — RdioScanner, OpenMHz, Icecast]({{ '/blog/deep-dives/running-it-for-real-10-broadcast-backends-ii/' | relative_url }})
