---
slug: live-events-and-webhooks
title: Live events & webhooks
description: GopherTrunk's push surface in practice — server-sent events carry every call and system event to subscribers, webhooks deliver them to your endpoints, and choosing between them per consumer.
keywords: GopherTrunk events, scanner SSE, call events, webhook notifications, live call feed, event stream, real-time scanner API
level: intermediate
status: full
prereq:
  - server-sent-events
  - webhooks
  - the-daemon-rest-api
gophertrunk_links:
  - title: API & events reference
    url: /api-events.html
    note: the authoritative list of event types and delivery options this lesson tours.
---

# Live events & webhooks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The daemon's push surface delivers **every call and system event as it
happens**, two ways: an **SSE stream** any subscriber can attach to (the web
console's live panels ride it) and **webhooks** that POST events to endpoints
you configure. It's Unit 3 deployed: **SSE for listeners you run** (dashboards,
scripts behind NAT), **webhooks for services that must be told** — and the
receiving disciplines (typed events, tolerant parsing, idempotent handling,
liveness awareness) are exactly the ones those lessons taught.
</div>

A scanner's soul is real-time: the REST API tells you what *happened*, but
"talkgroup 1201 keyed up *right now*" is why the box exists. This lesson tours
how the daemon pushes its life outward, and how the transport theory from
Unit 3 maps onto a system you can subscribe to tonight.

## One event flow, two deliveries

Inside the daemon, decoders publish onto an internal event bus — call starts
and ends, system lock and loss, device health. At the edge, that one flow
leaves through two doors, and the [Unit 3 trade-offs](/learn/apis/polling-vs-push/)
decide which door fits which consumer:

| | SSE stream | Webhooks |
|---|-----------|----------|
| You are | an outbound **subscriber** | a registered **receiver** |
| Works behind NAT | yes — outbound connection | no — must be reachable |
| Missed events | reconnect + resume semantics | sender retries |
| Best for | dashboards, scripts, anything long-running you operate | notifying services: chat alerts, home automation, log collectors |

Same events, different delivery contracts. A monitoring script on your laptop
wants the stream; a notification into your chat server wants the webhook.

## Riding the stream

Attaching is one command — the [SSE lesson's](/learn/apis/server-sent-events/)
`curl -N` habit, pointed at a live daemon:

```bash
curl -N http://scanner.local:8080/api/v1/events
```

```text
event: call.start
data: {"system":"county-p25","talkgroup":1201,"label":"County Fire Dispatch","freq_hz":460412500}

event: call.end
data: {"system":"county-p25","talkgroup":1201,"duration_seconds":8.4,"recording":"/api/v1/calls/48214/audio"}

event: system.status
data: {"system":"county-p25","control_channel":"locked"}
```

Leave it running while your scanner works and the module's abstractions become
concrete: those `event:` type names are the **vocabulary of a contract** —
consult the daemon's [event reference](/api-events.html) for the full set —
and everything Unit 1 said about contracts applies to them. Two consumer
disciplines matter especially here:

- **Parse tolerantly, dispatch on type.** Handle the event types you know,
  skip the ones you don't — additive evolution will add types, and a client
  that crashes on novelty breaks on every daemon upgrade.
- **Notice the cross-references.** The `call.end` event carries a *URL* into
  the REST API for the recording — the push surface and the
  [pull surface](/learn/apis/the-daemon-rest-api/) are one system, events
  telling you *when* and REST telling you *everything else*. This
  events-carry-pointers pattern keeps events small and consumers free to
  fetch only what they need.

## Receiving webhooks

Configure a webhook and the daemon becomes the client, POSTing each event to
your URL. Everything from the [webhooks lesson](/learn/apis/webhooks/) now
applies *to you*, concretely: answer fast (queue, then process — your handler
is on the daemon's clock, and a slow endpoint risks missed deliveries);
process **idempotently** (retries mean the same call event can arrive twice —
dedupe on the event's identity, or your "one call" chat alert fires two
messages); and treat the endpoint as the **public server it is** — reachable
URL, TLS, and verification that deliveries really come from your daemon,
especially the moment the receiver lives outside your LAN
([exposing a service safely](/learn/networking/exposing-a-service-safely/)
remains the checklist).

A worked shape, end to end: *call-start alert for a priority talkgroup* — 
webhook fires → your receiver checks the talkgroup against a watch list →
pushes a phone notification with the label from the event. Total code:
a page. That page is essentially [the final lesson's](/learn/apis/building-your-own-client/)
warm-up.

## The liveness lesson

One more discipline, learned honestly by this very system: **a silent stream
is ambiguous** — no events can mean "quiet airwaves" or "dead connection," and
a scanner's stream is *usually* quiet. Consumers that matter must resolve the
ambiguity actively: watch for the connection closing, apply the
[reconnect-with-backoff etiquette](/learn/apis/rate-limiting-and-quotas/), and
treat "how long since *any* event?" as a health signal with a threshold —
the same signal-liveness thinking the daemon itself applies to its radio-side
decoding. Push transports deliver events; *staying subscribed* is the
client's job, and the [next lesson](/learn/apis/web-console-sockets/) shows
how subtle that job can get.

<div class="knowledge-check" data-quiz data-correct-msg="Right — outbound SSE traverses NAT like any client connection; a webhook receiver must be a reachable server." markdown="0">
  <p class="knowledge-check__q">Quick check: your alert script runs on a laptop behind home NAT with no port forwarding. Which delivery fits?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Webhooks — the daemon will find the laptop when events occur</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The SSE stream — the laptop connects outbound and listens, no reachability required</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Neither — NAT rules out real-time delivery entirely</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- One internal event flow, **two deliveries**: **SSE** for subscribers you run
  (NAT-friendly, resume semantics), **webhooks** for services that must be
  told (sender retries, receiver must be reachable).
- Event **type names are contract vocabulary** — dispatch on type, tolerate
  the unknown, and expect additive growth.
- Events **carry REST pointers** (like the recording URL): push says *when*,
  pull says *everything else*.
- Webhook receivers: **answer fast, dedupe retries, verify and secure** the
  public endpoint you now operate.
- **Silence is ambiguous** — real consumers track liveness and reconnect with
  backoff.

Next up: [Streaming audio with gRPC](/learn/apis/streaming-audio-with-grpc/).
