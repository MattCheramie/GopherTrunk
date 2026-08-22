---
slug: server-sent-events
title: Server-sent events
description: A one-way event stream over plain HTTP — the SSE wire format, EventSource's built-in reconnection with Last-Event-ID, and why SSE is enough for most live-update needs.
keywords: server-sent events, SSE, EventSource, text/event-stream, one-way streaming, live updates, SSE vs WebSocket
level: intermediate
status: full
prereq:
  - polling-vs-push
  - anatomy-of-http
---

# Server-sent events

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Server-sent events (SSE)** turn one ordinary HTTP response into an **endless
one-way event stream**: the server sets `Content-Type: text/event-stream` and
simply never finishes the body, appending a small text-formatted **event** each
time something happens. Because it *is* HTTP, everything HTTP-shaped — TLS,
auth headers, proxies, curl — just works, and the browser's `EventSource` API
adds **automatic reconnection** with **`Last-Event-ID`** resume for free. When
data flows only server → client, SSE is usually the right tool before WebSockets.
</div>

If the WebSocket is a negotiated exit from HTTP, SSE is the opposite trick:
staying *inside* HTTP and just refusing to hang up. It's the mechanism behind
GopherTrunk's live event feed ([Unit 6](/learn/apis/live-events-and-webhooks/)),
and it's simpler than almost anyone expects.

## One response that never ends

An SSE session is a plain GET whose response streams forever:

```text
GET /api/v1/events HTTP/1.1
Host: scanner.local:8080
Accept: text/event-stream

HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache

event: call.start
id: 48214
data: {"system":"county-p25","talkgroup":1201,"label":"County Fire Dispatch"}

event: call.end
id: 48215
data: {"system":"county-p25","talkgroup":1201,"duration_seconds":8.4}

```

The wire format is that friendly: each event is a few `field: value` lines —
`event:` (a type name), `id:` (a resumption marker), `data:` (the payload, very
often JSON) — terminated by a **blank line**. You can watch a live stream with
nothing but curl:

```bash
curl -N http://scanner.local:8080/api/v1/events
```

(`-N` disables buffering so events print as they arrive.) That debuggability is a
genuine advantage: the same `curl -v` habits from
[anatomy of an HTTP request](/learn/apis/anatomy-of-http/) work unchanged on a
real-time feed.

## Reconnection is built in — and done right

In the browser, three lines subscribe:

```text
const es = new EventSource("/api/v1/events");
es.addEventListener("call.start", (e) => notify(JSON.parse(e.data)));
```

When the connection drops — and it will — `EventSource` reconnects
automatically, waiting the server-suggested `retry:` interval. Better, it sends
the last `id:` it saw in a **`Last-Event-ID`** request header, so a server that
keeps a short replay buffer can resend whatever the client missed during the
gap. **Gap-free delivery across disconnects** is the hardest part of any
real-time client, and SSE is the only browser transport that ships a standard
answer to it. With WebSockets you build id-tracking and resume yourself, or
silently lose events during every blip.

## SSE vs WebSockets vs polling

| | Polling | SSE | WebSocket |
|---|---------|-----|-----------|
| Direction | client asks | server → client | both ways |
| Latency | half the interval | immediate | immediate |
| Transport | plain HTTP | plain HTTP (one long response) | upgraded socket |
| Reconnect/resume | n/a (stateless) | **built in** (`Last-Event-ID`) | build it yourself |
| Debug with curl | yes | **yes** | no |
| Binary payloads | yes | text only (base64/JSON in practice) | native |

> Rule of thumb: if the client only ever *listens*, use SSE and keep HTTP's
> simplicity; reach for [WebSockets](/learn/apis/websockets/) only when the
> client must also *talk* on the same channel at low latency, or must ship
> binary frames.

The client can still talk, of course — it just uses ordinary HTTP requests
alongside the stream. "SSE for events down, REST for commands up" is a
well-worn architecture, and precisely how GopherTrunk's console pairs its event
feed with the [REST API](/learn/apis/the-daemon-rest-api/).

## The fine print

SSE's limits are real but narrow. Payloads are **text** (fine for JSON; binary
data needs encoding — or the WebSocket next door). Old HTTP/1.1 browsers capped
connections per host, which mattered when every tab opened its own stream;
HTTP/2 multiplexing dissolved this. And an "infinite" response can upset
middleware that buffers aggressively — proxies must be told not to
(`Cache-Control: no-cache`, disabled response buffering), a deployment detail
worth remembering when a stream works locally and stalls behind a reverse
proxy. Finally, the server holds one open connection per subscriber, so the
[backpressure question](/learn/apis/streaming-and-backpressure/) — what happens
when a subscriber stops reading? — applies to SSE exactly as to every other
push transport.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the reconnecting client presents Last-Event-ID, and the server replays what was missed." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes it possible for an SSE client to receive events it missed while disconnected?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">The <code>Last-Event-ID</code> header on reconnect, paired with a server-side replay buffer</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">TCP retransmission delivers the lost events automatically</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — missed events are always gone with any push transport</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- SSE is **one HTTP response that never ends**, streaming text
  **events** (`event:` / `id:` / `data:` + blank line) as they happen.
- Because it's plain HTTP, **TLS, auth, proxies, and curl** all work unchanged —
  a live feed you can debug with `curl -N`.
- `EventSource` gives **automatic reconnection**, and **`Last-Event-ID`** enables
  gap-free resume — the feature WebSocket clients must build by hand.
- **One-way flows fit SSE**; pair it with REST for the upstream direction, and
  step up to WebSockets only for two-way or binary traffic.
- An endless response needs **buffering-aware deployment**, and subscribers who
  stop reading raise the **backpressure** question like any push system.

Next up: [Webhooks](/learn/apis/webhooks/).
