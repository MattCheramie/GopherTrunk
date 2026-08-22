---
slug: websockets
title: WebSockets
description: One HTTP handshake upgrades to a persistent two-way pipe — how the WebSocket protocol works, its message framing, keepalives and reconnection, and when full duplex is worth the operational cost.
keywords: WebSocket protocol, WebSocket handshake, ws wss, full duplex, WebSocket upgrade, persistent connection, real-time web
level: intermediate
status: full
prereq:
  - polling-vs-push
  - anatomy-of-http
---

# WebSockets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **WebSocket** starts life as an ordinary HTTP request that asks to **Upgrade**
the connection; once the server agrees (`101 Switching Protocols`), the
request/response rules end and the socket becomes a **persistent, full-duplex
pipe** — either side sends **messages** whenever it likes. That buys the lowest
latency and true two-way traffic, and costs you what HTTP had given for free:
**connection lifecycle** — keepalives, detection of silent death, and **reconnect
logic** — is now your client's job.
</div>

WebSockets are the maximal answer to the push problem: not a workaround inside
request/response, but a negotiated exit from it. This lesson covers the
handshake, what flows afterwards, and the operational duties that come with
owning a long-lived connection — duties GopherTrunk's own web console learned the
hard way, as [Unit 6 recounts](/learn/apis/web-console-sockets/).

## The handshake: HTTP's polite exit

A WebSocket begins as a GET with two special headers:

```text
GET /diag/symbols HTTP/1.1
Host: scanner.local:8080
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

`101 Switching Protocols` is the last HTTP the connection ever speaks. From the
next byte on, the same TCP socket carries WebSocket **frames** in both
directions. Starting as HTTP is the protocol's cleverest feature: the connection
traverses the same ports (80/443), proxies, and TLS setup as the web itself —
`ws://` is the plain scheme, `wss://` the TLS one, and everything from the
[authentication lesson](/learn/apis/authentication-basics/) about TLS applies
doubly to a connection that lives for hours.

## After the upgrade: messages, not requests

The WebSocket protocol gives you **message boundaries** (a welcome upgrade from
raw TCP's endless byte stream — the problem
[message framing](/learn/apis/message-framing/) examines) and two payload types,
**text** and **binary**. And that's all. There are no methods, no paths, no
status codes, no headers — no *semantics*. What messages mean is entirely up to
your application: most designs invent a small JSON envelope, e.g.
`{"type": "spectrum", "bins": [...]}` from a scanner streaming display data, or
`{"type": "subscribe", "channel": "calls"}` from a client choosing feeds. You are
designing a tiny [protocol](/learn/apis/what-is-a-protocol/) of your own, with
all the contract duties that implies.

Full duplex is the differentiator: **both sides send, any time, without waiting**.
A spectrum panel receives thirty updates a second *while* sending pan/zoom
commands upstream on the same socket. If your data only flows one way —
server to client — you're paying for a capability you don't use, and the
[next lesson's](/learn/apis/server-sent-events/) simpler machinery deserves the
job.

## The new job: keeping the connection alive (and honest)

Here's what HTTP's one-shot model had been quietly doing for you: making
connection death *visible*. A request either answers or fails. A long-lived
socket, by contrast, can die **silently** — a NAT router times out the idle
mapping, a laptop sleeps, a cable drops — and neither side necessarily learns of
it until it next writes. Owning a WebSocket means owning three routines:

- **Keepalives.** The protocol has **ping/pong** frames; send pings periodically
  and treat a missing pong as a dead connection. This also keeps NAT mappings
  warm.
- **Timeouts.** A connection that hasn't produced expected traffic within some
  window is presumed dead — don't wait for TCP to notice on its own (it can take
  many minutes).
- **Reconnection with backoff.** Clients must expect to reconnect, waiting
  longer after each consecutive failure (1 s, 2 s, 4 s… capped), and — the subtle
  part — **reset that backoff only when the new connection proves itself by
  delivering data, not merely by opening.** A server that accepts the handshake
  and immediately closes will otherwise be hammered at full speed by every
  well-meaning client forever. This single sentence is the hard-won engineering
  lesson behind [GopherTrunk's console sockets](/learn/apis/web-console-sockets/).

> Rule of thumb: design every WebSocket client around the assumption that the
> connection *will* drop, silently, at the worst moment. Reconnection is the
> normal path, not the error path.

## Where WebSockets earn their keep

| Fit | Why |
|-----|-----|
| Live dashboards at high update rates | Sub-second latency, no per-update overhead |
| Chat, collaborative editing | Genuinely two-way, many small messages |
| Streaming telemetry (spectrum, symbols) | Binary frames carry dense data efficiently |
| Occasional notifications | **Poor fit** — an open socket per idle client is waste; consider SSE or polling |

The [networking-side view](/learn/networking/websockets-and-realtime/) of
WebSockets and the [browser-side view](/learn/web-dev/websockets-and-realtime/)
in the web-dev module complete the picture from below and above.

<div class="knowledge-check" data-quiz data-correct-msg="Right — resetting on data, not on handshake, is what stops a broken server from being hammered at full speed." markdown="0">
  <p class="knowledge-check__q">Quick check: a reconnecting WebSocket client should reset its backoff delay when…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">the socket's open event fires — the handshake succeeded</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">immediately before every reconnect attempt</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">the connection actually delivers data (or survives a grace period)</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A WebSocket is an HTTP request that **upgrades** (`101 Switching Protocols`)
  into a **persistent, full-duplex** message pipe over the same port and TLS.
- The protocol supplies **message framing** (text/binary) but **no semantics** —
  your message vocabulary is a contract you design and must honour.
- Full duplex suits **two-way, high-rate** traffic; one-way flows are usually
  better served by simpler transports.
- Long-lived sockets die **silently**: keepalive pings, liveness timeouts, and
  **reconnect with backoff** are the client's job.
- Reset backoff **only when data arrives**, never on a successful handshake.

Next up: [Server-sent events](/learn/apis/server-sent-events/).
