---
slug: websockets-and-realtime
title: WebSockets & real-time updates
description: When request-and-response isn't enough — pushing live data to the browser over WebSockets and server-sent events, the way a live scanner dashboard streams decoded calls the instant they happen.
keywords: WebSocket, real-time, server push, server-sent events, SSE, polling, long polling, live updates, bidirectional, streaming, scanner dashboard
level: intermediate
status: full
prereq:
  - client-server-web
faq:
  - q: "Why can't a normal HTTP request just wait for new data?"
    a: "It can — that's **long polling** — but it's a workaround. Plain HTTP is client-initiated: the browser asks and the server answers, then the connection closes. The server has no way to speak first. For a steady stream of updates you'd be opening request after request. A **WebSocket** keeps one connection open in both directions so the server can push the instant something happens."
  - q: "When should I use server-sent events instead of WebSockets?"
    a: "Use **server-sent events (SSE)** when data only flows *one way* — server to browser — like a live feed or notifications. SSE is simpler, runs over ordinary HTTP, and reconnects automatically. Reach for a **WebSocket** when you also need the browser to send messages back over the same live channel, such as a chat or an interactive control."
  - q: "Do real-time connections replace my REST API?"
    a: "No — they complement it. You still load the initial page and historical data over your [REST API](/learn/web-dev/building-a-rest-api/); the real-time channel carries only the *live changes* on top. A dashboard typically fetches the current state once, then subscribes to a stream for everything new."
---

# WebSockets & real-time updates

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Ordinary HTTP is **client-initiated** — the browser asks, the server answers — so it
can't *push*. For live data you either **poll** repeatedly (simple, wasteful), open
a one-way **server-sent events** stream, or open a **WebSocket**: a single
long-lived, **bidirectional** connection either side can send over at any time. A
[live scanner dashboard](/learn/web-dev/gophertrunk-web-dashboard/) is the perfect
case — decoded calls must appear the instant they're received, not on the next
refresh. The deeper networking mechanics live in
[WebSockets & real-time](/learn/networking/websockets-and-realtime/).
</div>

Everything so far in this module has followed one shape: the browser makes a
[request](/learn/web-dev/client-server-web/), the server sends a response, done.
That shape is a poor fit for anything *live* — a chat, a stock ticker, a
notification, or a scanner dashboard where a call can key up at any moment. This
lesson covers the ways a server pushes data to the browser, and when each fits.

## The problem: HTTP can't push

Plain HTTP is **client-initiated**. The server never speaks first; it only answers
what it's asked. That's fine for loading a page, but it means a server that *has*
new data — a call just decoded, a message just arrived — has no way to tell the
browser. The browser has to come asking.

For live features that leaves a fundamental gap: how does fresh data reach a page
that's just sitting there? The three answers below each close the gap differently,
trading simplicity for immediacy.

## Polling: asking over and over

The simplest approach needs no new technology. The browser just **asks
repeatedly** — every few seconds, "anything new?" — using ordinary
[fetch requests](/learn/web-dev/fetching-data/):

```javascript
setInterval(async () => {
  const res = await fetch("/api/calls/latest");
  render(await res.json());
}, 3000);   // ask every 3 seconds
```

**Polling** is easy and works everywhere, but it's a blunt instrument. Poll too
slowly and updates feel laggy; poll too fast and you flood the server with mostly
empty responses. **Long polling** improves on it — the server holds the request open
until it *has* something to say, then answers — but you're still reopening
connections. Polling is a fine default for data that changes slowly; it strains
under a genuine live stream.

## WebSockets: a two-way open line

A **WebSocket** replaces all that with a single **persistent, bidirectional**
connection. It starts as a normal HTTP request that asks to "upgrade"; once the
server agrees, the same connection stays open and **either side can send a message
at any time** with no new request. The server pushes the instant it has something.

```javascript
const socket = new WebSocket("wss://example.org/live");

socket.onmessage = (event) => {
  const call = JSON.parse(event.data);
  addCallToDashboard(call);   // arrives the moment the server sends it
};

socket.send(JSON.stringify({ subscribe: "talkgroup-42" }));  // browser can talk back
```

Note `wss://` — the secure, TLS-wrapped form, the WebSocket equivalent of HTTPS, and
what you should always use in production. Because the channel is two-way, WebSockets
suit anything interactive: chat, collaborative editing, games, or a dashboard where
the browser both receives updates *and* sends controls. The tradeoff is that a
long-lived connection is more to manage — reconnection, scaling many open sockets —
than a stateless request. The
[networking lesson on WebSockets](/learn/networking/websockets-and-realtime/) covers
the protocol handshake and framing underneath.

## Server-sent events: one-way and simpler

Between polling and full WebSockets sits **server-sent events (SSE)**: a single
long-lived HTTP response the server streams updates down, **one way only**
(server → browser). The browser subscribes with `EventSource` and the browser
handles reconnection for you:

```javascript
const stream = new EventSource("/api/calls/stream");
stream.onmessage = (event) => addCallToDashboard(JSON.parse(event.data));
```

If your data only flows *outward* — a feed, notifications, live decoded calls with
no controls to send back — SSE is often the better choice: it's simpler, rides
ordinary HTTP, and reconnects automatically. Choose a WebSocket when you need the
browser to send over the same live channel too.

## Framing the live scanner dashboard

This is exactly the shape of GopherTrunk's
[web dashboard](/learn/web-dev/gophertrunk-web-dashboard/). Decoded calls arrive
whenever radios key up — unpredictably, and potentially many per second on a busy
system. Polling would either miss the timing or hammer the server; a **push**
channel lets a call appear on screen the moment the decoder emits it. The dashboard
loads its current state over the [REST API](/learn/web-dev/building-a-rest-api/),
then subscribes to a live stream for everything new — the standard pairing of a
one-time fetch with an ongoing push, which the worked-example lesson builds out in
full.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a WebSocket keeps one connection open that either side can send over, so the server pushes the instant it has data." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a WebSocket different from repeatedly polling with fetch?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It downloads the whole page again each time, but faster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It keeps one connection open that either side can send over, so the server can push instantly</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It lets the browser read files directly from the server's disk</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Plain HTTP is **client-initiated** and can't push, so a server with fresh data has
  no way to reach an idle page on its own.
- **Polling** (and long polling) asks repeatedly — simple and universal, but wasteful
  and laggy for a real live stream.
- A **WebSocket** is a single **persistent, bidirectional** connection either side
  can send over at any time; use `wss://` for the secure form.
- **Server-sent events** stream updates **one way** (server → browser) over ordinary
  HTTP, simpler than WebSockets when the browser doesn't need to talk back.
- Real-time channels **complement** your REST API: load initial state once, then
  subscribe for live changes — exactly how a live scanner dashboard shows decoded
  calls as they happen.

Next up: [web security essentials](/learn/web-dev/web-security-essentials/).
