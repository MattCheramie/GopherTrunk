---
title: "Running It For Real, Part 10: Broadcast Backends II — RdioScanner, OpenMHz, Icecast"
description: Operating the other three call destinations on a 24/7 daemon — RdioScanner and OpenMHz credentials and endpoints riding the shared retry loop, and why Icecast is the odd one out that drops instead of retrying, reconnects on its own, and needs a source password kept off disk.
category: deep-dives
keywords: rdioscanner call upload, openmhz feed, icecast source stream, shoutcast sdr, source password, best effort streaming, feed monitoring, backoff reconnect, gophertrunk running it for real
tags: [running-it-for-real, broadcast, icecast, operations, monitoring, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 10
---

*Part 10 of **Running It For Real**. Part 9 ran a Broadcastify feed; the other
three destinations share its Manager and its counters but each bends the
operating story. RdioScanner and OpenMHz are more uploads — different
credentials, different endpoints, the same retry loop. Icecast is a different
animal: a persistent live stream, not a per-call upload, which means it drops
instead of retrying, reconnects itself across network blips, and holds a source
password that must never end up in a config-audit dump. On a laptop you'd never
tell them apart. Running them for real, the differences are the whole job. The
wire formats live in [Recording, Composition &
Streaming]({{ '/blog/series/recording-streaming/' | relative_url }}) (Parts
13–14); here we operate them.*

> **TL;DR:** **RdioScanner** wants a base `url` (the backend appends
> `/api/call-upload`), an `api_key`, and a numeric `system_id`; **OpenMHz** wants
> an `api_key` and a `short_name` that becomes the upload path segment. Both are
> single POSTs riding the exact `Manager.sendWithRetry` loop from Part 9 — same
> backoff, same `sent`/`failed` counters. **Icecast** breaks the mould: it holds a
> background source connection open, `Send` just appends to a bounded queue and
> returns `nil` even when it drops (so no pointless retries), and the connection
> reconnects on a fixed 5-second backoff after any blip. Its `password` is a live
> secret; keep it out of `config.yaml`.

**Key takeaways**

- **RdioScanner's URL is a base, not the endpoint.** You configure the server
  root; the backend appends `/api/call-upload`. Getting this wrong is the most
  common "why is nothing uploading" misconfiguration.
- **OpenMHz identifies the system by a path segment.** The `short_name` isn't
  metadata — it's baked into the upload URL, so a typo is a 404, not a rejection.
- **Icecast drops; it never retries.** `Send` returns `nil` even on a full queue,
  because retrying live audio makes no sense — so its health shows up as dropped
  calls and reconnect log lines, not a `failed` counter.
- **The source password is a live credential.** Unlike an upload key checked per
  request, it authenticates a long-lived socket; treat it like the Broadcastify
  key and keep it in an environment file.

## Cheat sheet

| Backend | Required config | Endpoint shape | Failure mode |
|---|---|---|---|
| RdioScanner | `url`, `api_key`, `system_id` | `<url>/api/call-upload` | retry via Manager, then `failed++` |
| OpenMHz | `api_key`, `short_name` | `api.openmhz.com/<short_name>/upload` | retry via Manager, then `failed++` |
| Icecast | `host`, `port`, `password` (+ `mount`) | `SOURCE <mount>` socket | drop call / reconnect; never `failed++` |
| Shared retry | — | `Manager.sendWithRetry` | 2s→4s→8s, 4 tries |
| Health | — | `Manager.Stats()` | `sent` / `failed` / `dropped` |

## In this post

- **RdioScanner** — the base-URL trap and the credentials it checks.
- **OpenMHz** — why the short name is load-bearing in the URL.
- **Icecast** — a live stream that drops instead of retrying.
- **Reconnect** — how the source socket survives a network blip unattended.
- **Monitoring three shapes** — the counters read the same; the meaning differs.

## RdioScanner: the base-URL that isn't the endpoint

RdioScanner is a single `multipart/form-data` POST — the field vocabulary
(`key`, `system`, `dateTime`, `talkgroup`, `source`, `frequency`, `audio`) we
covered in the streaming series. The operating gotcha is the URL. You configure
the server *root*, and the backend appends the API path itself:

```go
// internal/broadcast/rdioscanner.go (shape)
func NewRdioScanner(cfg RdioScannerConfig, hc *http.Client) (Backend, error) {
    if cfg.URL == "" {
        return nil, errors.New("broadcast/rdioscanner: url is required")
    }
    if cfg.APIKey == "" {
        return nil, errors.New("broadcast/rdioscanner: api_key is required")
    }
    if cfg.SystemID == 0 {
        return nil, errors.New("broadcast/rdioscanner: system_id is required")
    }
    return &rdioScannerBackend{
        endpoint: strings.TrimRight(cfg.URL, "/") + "/api/call-upload",
        // …apiKey, systemID, loc
    }, nil
}
```

So `url: https://scanner.example` is correct and
`url: https://scanner.example/api/call-upload` is *wrong* — you'd get
`.../api/call-upload/api/call-upload`, a 404, and a feed that "authenticates fine"
but never lands a call. Because a 404 comes back as a non-2xx, `Send` returns an
error, and — exactly like Part 9 — the Manager retries four times and then ticks
`failed["rdioscanner"]`. The `dateTime` field is stamped with the operator's
configured display timezone (an offset-bearing RFC3339), so a call reads with one
consistent wall-clock across RdioScanner, the webhook, and the call log. Three
credentials, one URL that has to be a root: that's the whole RdioScanner
operating surface.

## OpenMHz: the short name is in the URL

OpenMHz needs only two secrets — an `api_key` and a `short_name` — but the
`short_name` is doing more work than it looks. It isn't a metadata field; it's a
**path segment in the upload URL**:

```go
// internal/broadcast/openmhz.go (shape)
func NewOpenMHz(cfg OpenMHzConfig, hc *http.Client) (Backend, error) {
    if cfg.APIKey == "" {
        return nil, errors.New("broadcast/openmhz: api_key is required")
    }
    if cfg.ShortName == "" {
        return nil, errors.New("broadcast/openmhz: short_name is required")
    }
    return &openMHzBackend{
        endpoint: strings.TrimRight(base, "/") + "/" + cfg.ShortName + "/upload",
        // …base defaults to https://api.openmhz.com
    }, nil
}
```

The operational consequence: a wrong `short_name` doesn't produce an auth
rejection — it produces a **404 or a wrong-system upload**, because the request
went to a different URL entirely. So when an OpenMHz feed's `failed` counter
climbs but the `api_key` is right, suspect the short name before the key. The
upload body carries `source_list` and `patch_list` JSON arrays (OpenMHz expects
them even for a single granting unit); those are a wire-format detail, covered in
the streaming series, that you never touch operationally. Same Manager, same retry
loop, same counters — the only new operating fact is that OpenMHz encodes system
identity in the path, so path mistakes masquerade as system problems.

## Icecast: a live stream, not an upload

Here the operating model changes completely. RdioScanner and OpenMHz *upload a
finished call*; Icecast maintains a *continuous live stream* a listener tunes into
at any moment. That single difference drives every operational property. There is
no per-call request — a background goroutine holds the source connection open, and
`Send` merely appends the call's MP3 onto a bounded queue:

```go
// internal/broadcast/icecast.go (shape)
func (b *icecastBackend) Send(_ context.Context, c *Call) error {
    audio, err := c.MP3()
    if err != nil {
        return fmt.Errorf("%s: encode mp3: %w", b.name, err)
    }
    b.mu.Lock()
    defer b.mu.Unlock()
    if len(b.queue)+len(audio) > b.maxQueue {
        b.log.Warn("broadcast: icecast queue full, dropping call",
            "system", c.System, "tg", c.Talkgroup)
        return nil // best-effort: drop, DON'T fail — no pointless retry
    }
    b.queue = append(b.queue, audio...)
    return nil
}
```

Read that `return nil` carefully — it's the crux of operating Icecast. **A dropped
call returns success**, because returning an error would trigger the Manager's
retry loop, and retrying a chunk of live audio you're already three seconds past
is meaningless. So Icecast's health *never* shows up in the `failed` counter. Its
failure signal is different: the `broadcast: icecast queue full, dropping call`
warning, which means the source connection has stalled and the queue (capped at
about two minutes of audio) has backed up. You monitor Icecast by watching for
those drop warnings and the reconnect lines below — not by watching `failed`,
which for Icecast stays zero even when the stream is down.

<figure class="lab-figure">
<svg viewBox="0 0 660 180" width="660" height="180" role="img" aria-label="Two operating models side by side. On the left, upload backends: Send does one POST, and on error the Manager retries with backoff and then increments the failed counter. On the right, the Icecast backend: Send appends to a bounded queue and always returns nil; a full queue drops the call with a warning; a separate background goroutine holds the source connection and reconnects on a five-second backoff after any failure.">
  <rect x="10" y="30" width="300" height="130" rx="8" fill="none" stroke="var(--fg-muted)"/>
  <text x="160" y="48" text-anchor="middle" fill="currentColor" font-size="11">upload: RdioScanner / OpenMHz</text>
  <rect x="34" y="64" width="90" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="79" y="85" text-anchor="middle" fill="currentColor" font-size="10">Send → POST</text>
  <line x1="124" y1="81" x2="150" y2="81" stroke="currentColor"/><polygon points="150,77 160,81 150,85" fill="currentColor"/>
  <rect x="160" y="64" width="120" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="220" y="80" text-anchor="middle" fill="var(--accent)" font-size="10">retry w/ backoff</text>
  <text x="220" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">then failed++</text>
  <text x="160" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">error propagates; freshness sacrificed for a retry</text>
  <rect x="350" y="30" width="300" height="130" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="500" y="48" text-anchor="middle" fill="var(--accent)" font-size="11">live stream: Icecast</text>
  <rect x="372" y="64" width="88" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="416" y="82" text-anchor="middle" fill="currentColor" font-size="10">Send → queue</text>
  <text x="416" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="8">returns nil</text>
  <line x1="460" y1="81" x2="486" y2="81" stroke="var(--fg-muted)"/><polygon points="486,77 496,81 486,85" fill="var(--fg-muted)"/>
  <rect x="496" y="56" width="130" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="561" y="75" text-anchor="middle" fill="var(--fg-muted)" font-size="9">full → drop + warn</text>
  <rect x="496" y="94" width="130" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="561" y="110" text-anchor="middle" fill="var(--accent)" font-size="9">source goroutine</text>
  <text x="561" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reconnect @ 5s</text>
  <text x="500" y="148" text-anchor="middle" fill="var(--fg-muted)" font-size="9">never fails a call; health = drops + reconnect logs</text>
</svg>
<figcaption>The two operating models diverge at the first failure: an upload retries and eventually counts a failure; a live stream drops the call and keeps the socket alive, reconnecting on its own.</figcaption>
</figure>

### How that principle shaped the Go code

- **Best-effort is encoded in the return type, not a comment.** `Send` returning
  `nil` on a drop is what tells the Manager "don't retry" — the operating policy
  lives in the signature, so the shared retry loop needs no special case for
  Icecast.
- **The queue is bounded, so a stalled stream can't OOM the daemon.** `maxQueue`
  is `bytesPerSec * 120` — about two minutes — and past it the oldest audio is
  simply not accepted. A wedged Icecast server degrades to dropped calls, never to
  unbounded memory growth.
- **Close is cooperative.** `Manager.Close` type-asserts each backend for a
  `Close() error` and calls it; Icecast's cancels the source goroutine and waits
  (bounded) for it to unwind, so a config reload doesn't leak a socket.

## Reconnect: surviving a blip unattended

The reason Icecast can be a 24/7 destination and not a babysitting chore is that
its source connection heals itself. The `run` loop dials, handshakes, paces
audio, and on *any* failure waits a fixed backoff and starts over:

```go
// internal/broadcast/icecast.go (shape)
func (b *icecastBackend) run(ctx context.Context) {
    defer close(b.done)
    for {
        if ctx.Err() != nil {
            return
        }
        if err := b.runStream(ctx); err != nil {
            b.log.Warn("broadcast: icecast source disconnected",
                "backend", b.name, "err", err)
        }
        select {
        case <-ctx.Done():
            return
        case <-time.After(icecastReconnect): // fixed 5s
        }
    }
}
```

Note the backoff is *fixed* at 5 seconds, not exponential like the upload retry —
because a live stream wants to be back up fast, and a source socket that's down for
a minute means a minute of dead air for every listener. The trade-off is
deliberate: uploads back off to avoid hammering a struggling API; the stream
reconnects promptly to minimise the listener's silence. Between calls the pacer
keeps the socket alive by feeding pre-encoded silence (the streaming series covers
that mechanism), so a quiet system doesn't get its source timed out. Operationally,
a healthy Icecast feed logs one `icecast source connected` line at startup and then
goes quiet; a run of `source disconnected` warnings every ~5 seconds is your signal
the server or network is down.

## Monitoring three shapes with one set of counters

All four backends report through the same `Manager.Stats()` and the same
`GET /api/v1/broadcast`, but the numbers *mean* different things per backend, and
knowing that is the operating skill:

- **RdioScanner / OpenMHz** behave like Broadcastify: `sent` climbs, `failed`
  should stay flat. A climbing `failed` is a bad credential, a wrong base URL
  (RdioScanner), or a wrong short name (OpenMHz).
- **Icecast** never touches `failed`. Its `sent` increments on every successful
  queue append (which is almost always), so `sent` climbing tells you calls are
  *arriving* at the backend, not that they *reached listeners*. Its real health is
  in the log: drop warnings and reconnect lines.
- **`dropped`** is Manager-wide, not per backend — it counts calls shed because the
  Manager's own job queue was full, i.e. some backend is chronically slow and the
  daemon is protecting the recorder. A rising `dropped` with Icecast configured
  usually means the Icecast server is the slow one.

The practical alerting rule: watch `failed` rate for the upload backends, watch for
Icecast drop/reconnect log patterns, and watch Manager `dropped` as the
whole-subsystem backpressure signal. Three shapes, one dashboard.

## Where this goes next

The four audio destinations all deliver a *finished call*.
[Part 11]({{ '/blog/deep-dives/running-it-for-real-11-grant-webhooks/' | relative_url }})
turns to a different kind of outbound feed: the grant webhook, which fires a JSON
POST the instant a control-channel *grant* is decoded — before any audio exists —
for home automation, dashboards, and alerting. Same retry discipline, same
bounded-queue-drops-under-load posture you've now seen three times, but a
control-channel event rather than a recording.

## FAQ

**My RdioScanner feed authenticates but never uploads — why?**
Almost always the `url`. Configure the server *root* (e.g. `https://scanner.example`);
the backend appends `/api/call-upload` itself. If you included the path, you get a
doubled `/api/call-upload/api/call-upload` 404 that retries and then increments
`failed`. Strip the path from your config.

**OpenMHz `failed` is climbing but my api_key is correct — what else?**
Check the `short_name`. It's a path segment in the upload URL, not a metadata field,
so a wrong short name sends the POST to the wrong (or nonexistent) URL — a 404 or a
wrong-system upload rather than an auth error. Fix the short name before touching the
key.

**Why doesn't Icecast show failures in the counters?**
Because it's a live stream, not an upload. `Send` returns `nil` even when it drops a
call, so the Manager never retries and never counts a failure — retrying live audio
you're already past is meaningless. Icecast health lives in the log: `queue full,
dropping call` warnings and `source disconnected`/`connected` reconnect lines.

**Where does the Icecast source password go?**
Treat it like the Broadcastify key. It authenticates a long-lived source socket, so
it's a live credential — keep it in an environment file the systemd unit references
rather than inline in `config.yaml`, and out of any config-audit surface.

**How fast does an Icecast feed recover from a network blip?**
The source goroutine reconnects on a fixed 5-second backoff (not exponential),
re-running the dial and handshake until the stream is back. A brief blip costs a few
seconds of dead air; a sustained outage logs a `source disconnected` warning roughly
every 5 seconds until the server returns.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Broadcast Backends I — Broadcastify]({{ '/blog/deep-dives/running-it-for-real-09-broadcast-backends-i/' | relative_url }})
· Next →
[Part 11: Grant Webhooks & External Integrations]({{ '/blog/deep-dives/running-it-for-real-11-grant-webhooks/' | relative_url }})
