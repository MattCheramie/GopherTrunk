---
slug: polling-vs-push
title: Polling vs push
description: Asking "anything new?" every second versus being told the moment something happens — polling, long polling, and push compared, with latency and cost trade-offs at the heart of real-time API design.
keywords: polling vs push, long polling, server push, real-time API, event-driven, polling interval, push notifications
level: beginner
status: full
faq:
  - q: What is polling in an API?
    a: Polling is when a client repeatedly asks the server whether anything has changed — for example, requesting the latest call log every few seconds. It works with nothing but plain request/response HTTP, which is its great strength, but it wastes requests when nothing is happening and adds delay (up to one full polling interval) when something is.
  - q: What is the difference between polling and webhooks?
    a: With polling, your program asks the server for news on a schedule. With a webhook, the roles reverse — the server sends an HTTP request to *your* endpoint the moment an event happens. Webhooks eliminate the constant asking, but require you to run something that can receive requests, which is a real operational cost.
  - q: What is long polling?
    a: "Long polling is a hybrid: the client asks, and the server holds the request open — not answering — until it actually has news (or a timeout passes). The client then immediately asks again. It delivers near-push latency using only ordinary HTTP, at the cost of the server juggling many open, idle requests."
---

# Polling vs push

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Request/response has a blind spot: **the server can never speak first**. A client
that needs to know about events can **poll** — ask on a schedule, paying with
wasted requests and up to one interval of **latency** — or the system can **push**:
keep a channel open (WebSockets, SSE) or call the client back (webhooks) so news
travels the moment it exists. **Long polling** sits between. The choice is a trade
of latency against cost and complexity, and it shapes everything in this unit.
</div>

A scanner is the perfect stage for this problem: calls start at unpredictable
moments, and "tell me when talkgroup 1201 keys up" is the whole product. This
lesson frames the polling-versus-push trade that the next four lessons — 
WebSockets, server-sent events, webhooks, and backpressure — each resolve
differently.

## The problem: news the client doesn't know to ask for

Everything in Unit 2 was client-initiated: no request, no response. But events —
a call starting, a system going quiet, a config change — happen on the *server's*
clock. In pure request/response, the freshest a client can be is *as fresh as its
last question*. So the client asks repeatedly:

```bash
# crontab-grade "real-time": ask every 5 seconds
while true; do
  curl -s http://scanner.local:8080/api/v1/calls?limit=1
  sleep 5
done
```

That's **polling**, and it genuinely works — it's the simplest thing that can
possibly function, built from nothing but GETs. Its two costs are structural:

- **Latency.** A call starting right after a poll waits a full interval to be
  noticed. Average delay: half the interval.
- **Waste.** On a quiet system, hundreds of requests an hour return "nothing
  new" — load on the server, noise in the logs, battery on mobile clients — and
  the *server* pays that cost multiplied by every polling client it has.

Tightening the interval trades one cost for the other. Polling at 100 ms is
low-latency and abusive; polling hourly is polite and stale. There is no interval
that's both.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 200" role="img" aria-label="Timeline comparison: polling sends many requests with mostly empty responses and delayed news; push sends one event immediately when it happens." xmlns="http://www.w3.org/2000/svg">
  <text x="10" y="20" font-size="13" fill="currentColor" font-weight="bold">Polling</text>
  <line x1="70" y1="40" x2="500" y2="40" stroke="currentColor" stroke-opacity="0.3" stroke-width="1"/>
  <g stroke="currentColor" stroke-width="1.5">
    <line x1="100" y1="30" x2="100" y2="50"/><line x1="170" y1="30" x2="170" y2="50"/>
    <line x1="240" y1="30" x2="240" y2="50"/><line x1="310" y1="30" x2="310" y2="50"/>
    <line x1="380" y1="30" x2="380" y2="50"/><line x1="450" y1="30" x2="450" y2="50"/>
  </g>
  <text x="100" y="66" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">empty</text>
  <text x="170" y="66" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">empty</text>
  <text x="240" y="66" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">empty</text>
  <text x="380" y="66" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">news!</text>
  <circle cx="330" cy="40" r="5" fill="currentColor"/>
  <text x="330" y="90" text-anchor="middle" font-size="11" fill="currentColor">event happens · noticed 50 units later</text>
  <text x="10" y="130" font-size="13" fill="currentColor" font-weight="bold">Push</text>
  <line x1="70" y1="150" x2="500" y2="150" stroke="currentColor" stroke-opacity="0.3" stroke-width="1"/>
  <circle cx="330" cy="150" r="5" fill="currentColor"/>
  <line x1="335" y1="150" x2="360" y2="150" stroke="currentColor" stroke-width="2"/>
  <path d="M360 150 l-8 -4 v8 z" fill="currentColor" transform="rotate(180 360 150)"/>
  <text x="330" y="180" text-anchor="middle" font-size="11" fill="currentColor">event happens · delivered immediately</text>
</svg>
<figcaption><strong>Polling</strong> pays for many empty answers and still learns the news late; <strong>push</strong> sends one message, at exactly the right moment.</figcaption>
</figure>

## Push: let the server speak

**Push** inverts the flow: when the event happens, the *server* initiates
delivery. The catch is that "the server initiates" isn't something bare
request/response can express, so every push mechanism is a workaround with its
own shape:

| Mechanism | How the server gets a voice | Lesson |
|-----------|------------------------------|--------|
| **Long polling** | Client asks; server *withholds the answer* until there's news | (below) |
| **Server-sent events** | Client opens a response the server never finishes, streaming events into it | [SSE](/learn/apis/server-sent-events/) |
| **WebSockets** | One handshake upgrades the connection to a permanent two-way pipe | [WebSockets](/learn/apis/websockets/) |
| **Webhooks** | The server becomes a *client* of you — it POSTs to your URL | [Webhooks](/learn/apis/webhooks/) |

Note what they share: someone is now maintaining state *between* requests — an
open connection, or a registered callback URL. That's the price of push, and it's
why polling never fully dies: stateless, cache-friendly, firewall-proof asking is
sometimes worth the staleness.

## Long polling: the clever compromise

**Long polling** deserves its own moment because it delivers push-like latency
with only HTTP: the client asks "anything new?", and the server simply *doesn't
answer* until there is something (or ~30 s passes) — then the client re-asks
immediately. News flows within milliseconds of happening, yet every hop is an
ordinary request/response. The cost lands on the server, which now holds many
open, idle requests. It was the web's main real-time trick before WebSockets and
still shines where proxies or old infrastructure choke on fancier transports.

## Choosing, in practice

> Rule of thumb: poll when staleness is cheap (a dashboard refreshing a daily
> stat), push when moments matter (a call notification), and prefer the simplest
> push that fits — SSE before WebSockets if the flow is one-way.

And a preview of the unit's hardest lesson: push creates a new failure mode.
When the producer can emit events faster than a consumer drains them, something
must buffer, block, or drop — that's
[streaming & backpressure](/learn/apis/streaming-and-backpressure/), and no push
system escapes it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — on average you wait half the polling interval before noticing an event." markdown="0">
  <p class="knowledge-check__q">Quick check: a client polls every 10 seconds. On average, how stale is its knowledge of a new event?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Zero — polling always sees events immediately</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">About 5 seconds — half the interval, and up to the full 10</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Exactly 10 seconds, every time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Request/response means **the server can never speak first** — the structural
  gap all real-time techniques fill.
- **Polling** is simple and stateless but trades **latency** (≈ half the
  interval) against **waste**, with no interval that fixes both.
- **Push** delivers events the moment they happen, at the price of state between
  requests — an open connection or a registered callback.
- **Long polling** fakes push over plain HTTP by withholding the answer until
  there's news.
- Choose the **simplest mechanism that meets the latency need** — and remember
  every push system inherits the **backpressure** problem.

Next up: [WebSockets](/learn/apis/websockets/).
