---
slug: webhooks
title: Webhooks
description: The server calls you — HTTP callbacks that push events to your endpoint, and the delivery, retry, verification, and idempotence care every webhook needs on both sides.
keywords: webhooks, HTTP callback, webhook receiver, webhook retries, webhook signature verification, at-least-once delivery, idempotent webhook
level: intermediate
status: full
prereq:
  - polling-vs-push
  - clients-and-servers
---

# Webhooks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **webhook** flips the roles: you register a URL, and when an event happens the
service makes an HTTP **POST to you** — the server becomes a client, and your
program becomes a **server**. No connection stays open, which makes webhooks ideal
for **server-to-server** integration — but delivery over the open internet means
your receiver must answer **fast**, expect **retries and duplicates**
(**at-least-once** delivery ⇒ handle events **idempotently**), and **verify** that
each delivery really came from the sender.
</div>

WebSockets and SSE both hold a connection open. Webhooks take the third path: no
standing connection at all — just an agreement that *when something happens, the
service will send a request to a URL you chose*. It's how services integrate with
each other all over the industry, and it's one of the delivery options for
GopherTrunk's call events ([Unit 6](/learn/apis/live-events-and-webhooks/)).

## The role reversal

Everything from [clients and servers](/learn/apis/clients-and-servers/) still
applies — the roles just swap per direction. You tell the scanner daemon (via its
config or API): "when a call starts, POST to
`https://myserver.example/hooks/scanner`." Later, on its own initiative:

```text
POST /hooks/scanner HTTP/1.1
Host: myserver.example
Content-Type: application/json
X-Event-Type: call.start
X-Signature: sha256=7fd2a1...

{"system":"county-p25","talkgroup":1201,"label":"County Fire Dispatch","start":"2026-08-21T14:03:12Z"}
```

Your endpoint answers `200 OK` and does its thing — a push notification, a
database row, a light turning on. The elegance: between events, **nothing
exists** — no socket, no polling loop, no state but the registered URL. The
catch: you now operate an internet-reachable HTTP server, with everything that
implies ([exposing a service safely](/learn/networking/exposing-a-service-safely/)
is required reading before doing this from home).

## Delivery is best-effort — so it retries

The sender can't know your endpoint is healthy; it only knows whether the POST
got a 2xx. So every serious webhook sender follows the same discipline, and your
receiver must anticipate each piece of it:

- **Timeouts are short.** If your endpoint takes 30 s to respond, the sender
  gives up. Therefore: **acknowledge first, work later** — return `200` as soon
  as the event is durably queued, and do the slow work (transcoding, notifying,
  writing) afterwards.
- **Failures are retried**, usually with growing delays over minutes or hours.
  Good senders treat any non-2xx or timeout as "try again later."
- **Retries mean duplicates.** A delivery whose *response* got lost will be sent
  again — the event arrived twice. This is **at-least-once delivery**, and it's a
  law of the pattern, not a bug: the only alternatives are at-most-once (events
  silently lost) or exactly-once (impossible without coordination both ends
  rarely have).

> Rule of thumb: process webhook events **idempotently** — use the event's ID to
> make the second arrival of the same event a no-op. This is
> [idempotence](/learn/apis/methods-and-status-codes/) from Unit 2, now as a
> receiver's survival skill.

## Trust: anyone can POST to a URL

Your webhook endpoint is a public URL, and *anything* on the internet can send
requests to it. An unverified receiver will happily act on forged "call.start"
events from a script kiddie's laptop. The standard defence is a **signature**:
sender and receiver share a secret; the sender computes an HMAC of each payload
and ships it in a header (the `X-Signature` above); the receiver recomputes and
compares before trusting a byte of the body. Two details matter in practice:
sign the **raw body bytes** (re-serialized JSON may differ harmlessly but hash
differently), and use a constant-time comparison. TLS on your endpoint is table
stakes — it protects the payload in transit, but only the signature proves *who
sent it* (the [cybersecurity module](/learn/cybersecurity/hashing-and-integrity/)
covers the HMAC machinery itself).

## Webhooks vs the open-connection transports

| | Webhook | SSE / WebSocket |
|---|---------|-----------------|
| Receiver is | a public HTTP **server** | an outbound **client** |
| Standing state | none — just a registered URL | an open connection per client |
| Works behind NAT/firewall | **no** (must be reachable) | **yes** (outbound only) |
| Missed-event story | sender retries for you | resume/replay (SSE) or DIY |
| Best for | server-to-server integrations | browsers, dashboards, anything behind NAT |

That NAT row decides most real cases: a phone app or a browser tab can't
receive webhooks, and a cloud service integrating with another cloud service
shouldn't hold a million idle sockets. Pick the transport whose *receiver role*
your situation can actually play.

<div class="knowledge-check" data-quiz data-correct-msg="Right — at-least-once delivery makes duplicates normal, so receivers must deduplicate by event ID." markdown="0">
  <p class="knowledge-check__q">Quick check: your webhook receiver got the same <code>call.start</code> event twice. What does this most likely mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The sender has a bug — duplicates should never happen</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Someone is forging events to your endpoint</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Normal at-least-once behaviour — a retry after a lost acknowledgment; dedupe by event ID</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A webhook is an **HTTP callback**: you register a URL, the service **POSTs
  events to you** — roles reversed, no standing connection.
- Receivers must **acknowledge fast and work later**; senders time out quickly
  and **retry** failures.
- Delivery is **at-least-once**: duplicates are normal, so process events
  **idempotently**, keyed by event ID.
- **Verify signatures** (HMAC over the raw body) — a public URL will receive
  forgeries eventually; TLS alone doesn't prove the sender.
- Webhooks fit **server-to-server**; anything behind NAT or in a browser should
  use SSE or WebSockets instead.

Next up: [Streaming & backpressure](/learn/apis/streaming-and-backpressure/).
