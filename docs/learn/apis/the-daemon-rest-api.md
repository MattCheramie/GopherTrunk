---
slug: the-daemon-rest-api
title: GopherTrunk's REST API
description: The daemon's HTTP API in practice — talkgroups, radio IDs, and call history under /api/v1/, why the web console is just another client, and what a real API teaches that examples can't.
keywords: GopherTrunk API, scanner REST API, talkgroups API, call history API, radio IDs, daemon HTTP API, dogfooding an API
level: intermediate
status: full
prereq:
  - rest-fundamentals
  - urls-queries-and-bodies
faq:
  - q: Does GopherTrunk have an API?
    a: Yes. The daemon serves an HTTP REST API under /api/v1/ covering its core records — talkgroups, radio IDs, and call history — alongside live event streams and audio interfaces. The web console you see in the browser is built on the same API, so anything the console displays, a script of yours can fetch too.
  - q: What can I build against a scanner daemon's API?
    a: Anything that consumes its records or events — a phone notification when a priority talkgroup goes active, a nightly report of the busiest systems, a dashboard widget of recent calls, an exporter feeding a database. The REST API covers lookups and history; for reacting the moment something happens, the event stream in the next lesson is the right tool.
  - q: Do I need an API key to use a local daemon's API?
    a: On a private LAN a local daemon is commonly run without API authentication, with the network boundary as the control — the trade-off discussed in this module's authentication lesson. Treat that as a deliberate decision to revisit the moment the daemon is reachable from anywhere you don't fully trust, and never expose it to the internet unauthenticated.
gophertrunk_links:
  - title: Web console guide
    url: /web.html
    note: the console is the API's first client — every panel maps to endpoints described here.
  - title: API & events reference
    url: /api-events.html
    note: the daemon's documented API and event surface, the authoritative companion to this unit.
---

# GopherTrunk's REST API

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Unit 6 studies a working system. GopherTrunk's daemon serves a **REST API under
`/api/v1/`** — **talkgroups**, **radio IDs**, **call history** — and its **web
console is just another client** of the same contract, which means everything
you learned in Units 1–5 becomes hands-on here. The API's design choices —
plural nouns, query-string filters, a version namespace, labels applied via
PATCH — are the module's conventions **operating in production**, on a box you
can run yourself.
</div>

Everything so far used invented examples. From here, the module studies the
real thing: the interfaces of an actual daemon you can install, curl, and build
against. This lesson tours the REST surface; the next three cover its
[events](/learn/apis/live-events-and-webhooks/),
[audio streaming](/learn/apis/streaming-audio-with-grpc/), and
[console sockets](/learn/apis/web-console-sockets/) — then
[you build a client](/learn/apis/building-your-own-client/).

## The shape of the surface

A trunking scanner's domain model maps to resources exactly as
[REST fundamentals](/learn/apis/rest-fundamentals/) taught — the daemon's
nouns are the hobby's nouns:

| Resource | What it holds |
|----------|---------------|
| **Talkgroups** | The virtual channels of each trunked system — IDs, labels, priorities |
| **Radio IDs** | Individual radios seen on the air — IDs, aliases, last activity |
| **Call history** | The record of decoded calls — when, which talkgroup, which unit, how long |
| **Systems** | The configured trunked systems being tracked |

All under `/api/v1/` — that `v1` being the
[versioning lesson's](/learn/apis/api-versioning/) contract namespace in the
wild. Reads are GETs with query-string filters; the interesting write is
labelling: when you name a talkgroup or radio in the console, a **PATCH**
carries the new label, and the daemon layers your operator-applied name over
whatever alias files were imported — the *operator's most recent explicit act
wins*. That's a contract-level design decision of exactly the kind
[Unit 5](/learn/apis/designing-a-good-api/) discussed: the API models how
operators think (my label sticks), not how storage works.

## Reading the records with curl

The [anatomy](/learn/apis/anatomy-of-http/) habits transfer directly:

```bash
# What talkgroups does the daemon know?
curl -s http://scanner.local:8080/api/v1/talkgroups | jq .

# Recent calls on one talkgroup — filters as query params
curl -s "http://scanner.local:8080/api/v1/calls?talkgroup=1201&limit=20" | jq .

# Name a radio you keep hearing
curl -s -X PATCH http://scanner.local:8080/api/v1/rids/70233 \
  -H "Content-Type: application/json" \
  -d '{"alias": "Engine 3 portable"}'
```

(`jq` pretty-prints and queries JSON — worth installing today.) Notice you're
exercising the full Unit 2 stack: methods carrying intent, paths carrying
identity, query strings carrying filters, JSON bodies carrying payloads. When
something surprises you, `curl -v` shows the raw truth, and the responses are
plain enough to read in a terminal — the
[text-protocol dividend](/learn/apis/text-vs-binary-protocols/) paying out.

## The console is a client — and why that matters

Open the web console's History panel with your browser's developer tools on the
Network tab: you'll watch it issue the same `/api/v1/` requests you just typed.
The console holds **no private door** into the daemon — it authenticates,
filters, and pages through the public contract like any script of yours.

This pattern is called **dogfooding** an API, and it's worth adopting in your
own designs because of what it guarantees: the API *must* be complete (the
console needs everything users see), *stays* tested (every console session
exercises it), and treats third-party clients as first-class (they use the very
same door). It also hands you a discovery technique for any dogfooded product,
GopherTrunk included: **when you wonder how to fetch something, watch the
official client fetch it.** The Network tab is documentation that cannot drift.

One cautionary tale from this very codebase keeps you honest about client-side
care: the console's history panel once read the wrong envelope key from the
call-history response — the code expected one field name, the daemon sent
another — and a `?? []` fallback quietly turned every response into "no calls
found." The handler, filters, and SQL were all correct; the *client's parsing
assumption* was the bug, and it read as "search is broken" for as long as it
lived. Contract precision cuts both ways — which is why
[pinning shapes in tests](/learn/apis/testing-an-api/) applies to clients too.

## What the REST surface deliberately isn't

Notice what's absent: no "tell me the instant a call starts" endpoint.
REST-over-HTTP answers questions you ask; for a scanner, the most valuable
information is *unasked* — calls beginning **now**. You could poll
`/api/v1/calls` in a loop, but [Unit 3](/learn/apis/polling-vs-push/) taught
you what that costs. The daemon's answer is a separate, push-shaped surface —
server-sent events and webhooks — which is precisely the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a dogfooded console guarantees the public API is complete and continuously exercised, and makes the Network tab live documentation." markdown="0">
  <p class="knowledge-check__q">Quick check: what does "the web console is just another API client" guarantee about the daemon's API?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That the API is faster than the console's internal paths</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">That the public contract is complete and constantly exercised — anything the console shows, your script can fetch the same way</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That the API requires no authentication, since the console doesn't log in</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The daemon's REST surface lives under **`/api/v1/`**: **talkgroups, radio
  IDs, call history, systems** — the hobby's nouns as resources.
- Reads are **filtered GETs**; naming things is a **PATCH**, with operator
  labels layered to win over imported files — contract design mirroring how
  operators think.
- The **web console is a client of the same API** — dogfooding keeps the
  contract complete, and the browser's **Network tab** is undriftable
  documentation.
- A real client-side envelope bug shows contract precision matters **on both
  ends** — pin shapes in tests.
- REST answers questions you ask; **calls starting now** need the push surface
  in the next lesson.

Next up: [Live events & webhooks](/learn/apis/live-events-and-webhooks/).
