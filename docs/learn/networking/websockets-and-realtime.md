---
slug: websockets-and-realtime
title: WebSockets & real-time connections
description: A plain-language look at real-time web connections — why plain HTTP request/response can't push updates, how WebSockets open a persistent two-way channel by upgrading an HTTP connection, the simpler alternatives of Server-Sent Events and long polling, and the trade-offs of keeping a connection open.
keywords: WebSocket, real-time, server push, server-sent events, long polling, persistent connection, streaming, live updates
level: advanced
status: full
prereq:
  - http
faq:
  - q: What is a WebSocket?
    a: "A WebSocket is a long-lived, two-way connection between a client and a server. It starts as an ordinary HTTP request that asks to \"upgrade\" the connection; once accepted, the connection stays open and either side can send messages at any time. That makes it well suited to chat, live dashboards, notifications, and streaming."
  - q: What is the difference between WebSockets and HTTP?
    a: "Plain HTTP is one-shot request/response: the client asks, the server answers, and the connection is done. A WebSocket keeps a single connection open so the server can push data whenever it has something new, without the client asking again. HTTP is ideal for fetching pages and data; WebSockets are for genuinely live, ongoing exchanges."
  - q: What are Server-Sent Events and long polling?
    a: "They are simpler ways to get updates without a full WebSocket. Server-Sent Events open a one-way stream from server to client over ordinary HTTP — good when only the server needs to push. Long polling has the client make a request the server holds open until it has news, then the client immediately asks again. Both are easier to deploy than WebSockets but do less."
  - q: When should I use a WebSocket instead of regular HTTP?
    a: Use a WebSocket when updates are frequent and can come from the server at any moment — live chat, a dashboard that changes second by second, or streaming audio. For occasional data you can fetch on demand, plain HTTP requests are simpler and cheaper. A persistent connection costs server resources, so reserve it for genuinely real-time needs.
---

# WebSockets & real-time connections

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Plain [HTTP](/learn/networking/http/) is one-shot **request/response**: the client
asks, the server answers, and the exchange is over. That's perfect for loading a
page, but live apps need the server to tell you the moment something changes. A
**WebSocket** solves this by opening a **persistent connection** — a single
long-lived channel where either side can send messages at any time, so the server
can **push** data without waiting to be asked.
</div>

Almost everything you've learned so far assumes the client asks and the server
answers. That model built the web, but it has one blind spot: the server can never
speak first. Real-time apps — chat, live dashboards, streaming — need exactly that,
and this lesson is about how they get it.

## The limit of request/response

In the [client-server](/learn/networking/clients-and-servers/) model, the
[HTTP](/learn/networking/http/) exchange only ever runs in one direction to begin
with: **the client always has to ask.** The server sits and waits, and no matter
how much has changed on its end, it cannot reach out to tell you. It can only
answer a request you've already sent.

That's fine when you're fetching a page or submitting a form — you ask, you get an
answer, you're done. But imagine a chat app built this way. New messages arrive on
the server, but your browser has no idea until it asks. So it asks again. And
again. This is **polling**: repeatedly firing off requests — "anything new? anything
new?" — just in case.

Polling works, but it's wasteful. Poll too slowly and updates feel laggy; poll too
quickly and you flood the server with mostly-empty requests. The real problem is
structural: request/response gives the server no way to speak first. To build
something genuinely live, we need a channel the server can talk over whenever it
likes.

## WebSockets

A **WebSocket** is that channel. It's a **persistent**, **full-duplex** connection
— "full-duplex" just meaning both sides can send at the same time, like a phone
call rather than a walkie-talkie.

The clever part is how it starts. A WebSocket begins life as an ordinary
[HTTP](/learn/networking/http/) request carrying a special header that asks to
**upgrade** the connection. If the server agrees, the two sides stop speaking HTTP
and keep the very same connection open as a WebSocket. From that moment on, there's
no more asking and answering — **either side can send a message any time**, for as
long as the connection lives.

That changes what's possible. The server can push a new chat message the instant it
arrives, update a dashboard the moment a number moves, or stream a continuous feed
of events — all without the client polling for any of it. Chat, live dashboards,
multiplayer games, notifications, and streaming are the classic cases, and they all
share the same shape: data that arrives unpredictably and needs to reach the user
right away.

## Simpler alternatives

A full two-way WebSocket is more than some apps need. Two lighter options cover
common cases.

**Server-Sent Events (SSE)** open a **one-way** stream from server to client over
an ordinary HTTP connection. The client makes a single request, and the server
keeps the response open, sending updates down it as they happen. There's no channel
back — the client still uses normal HTTP requests to send anything — but when *only
the server* needs to push (a news ticker, a progress feed, a notifications stream),
SSE gives you that with far less machinery than a WebSocket.

**Long polling** is simpler still, and older. The client makes a request, but
instead of answering immediately, the server **holds it open** until it actually
has something to say. When news arrives, the server responds; the client reads it
and immediately opens another request to wait again. It approximates a live feed
using nothing but standard requests, which makes it a reliable fallback where
WebSockets or SSE aren't available — at the cost of the overhead of constantly
reopening connections.

The rule of thumb: reach for **long polling** only as a fallback, **SSE** when the
push is one-way, and a **WebSocket** when both sides need to talk freely.

## Trade-offs

Real-time connections aren't free. A persistent connection **stays open**, so the
server holds resources for every connected client whether or not anything is being
sent — thousands of idle WebSockets still cost thousands of open connections.

They also need **reconnection handling**. Networks drop, laptops sleep, phones
switch from Wi-Fi to cellular — and unlike a one-shot request you can simply retry,
a dropped persistent connection has to be detected and re-established, often while
catching up on whatever was missed.

And they interact awkwardly with the infrastructure in between.
[Proxies and load balancers](/learn/networking/proxies-and-load-balancers/),
firewalls, and timeouts are mostly tuned for short HTTP requests, and a connection
that stays open for hours can trip them up unless everything along the path is
configured to allow it.

None of this is a reason to avoid WebSockets — it's a reason to **spend them
deliberately**. Keep persistent connections for genuinely live needs, and let plain
request/response handle everything that can wait to be asked.

## Where GopherTrunk fits

A scanner is a live thing — calls come and go second by second — and a web
interface that reflects that is a textbook real-time connection. When GopherTrunk's
interface shows call activity appearing the instant it happens, or streams decoded
audio to your browser, it isn't polling for updates: the server is **pushing** them
over an open channel, exactly the pattern this lesson describes. We'll see how that
fits into the wider picture of putting the scanner on your network in
[GopherTrunk on the network](/learn/networking/gophertrunk-on-the-network/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a WebSocket stays open so the server can push data any time." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a WebSocket give you that plain HTTP request/response doesn't?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A persistent, two-way connection where the server can push data any time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A faster way to load a single page</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A guarantee that requests never fail</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Plain **HTTP** is one-shot **request/response**: the client always asks, and the
  server can never speak first.
- Faking live updates by **polling** — asking over and over — is wasteful and laggy.
- A **WebSocket** opens a **persistent**, two-way channel by **upgrading** an HTTP
  connection; afterwards either side can send messages any time.
- **Server-Sent Events** give a one-way server-to-client stream, and **long polling**
  approximates live updates with held-open requests — simpler when that's all you need.
- Persistent connections cost resources, need reconnection handling, and can clash
  with proxies and firewalls — so keep them for genuinely real-time needs.
- GopherTrunk pushing live call activity or streaming audio to your browser is
  exactly this kind of real-time connection.

Next up: Module 4 covers connecting and securing networks — [firewalls & controlling access](/learn/networking/firewalls/)
