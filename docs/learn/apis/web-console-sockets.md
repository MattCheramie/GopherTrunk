---
slug: web-console-sockets
title: The web console's sockets
description: "Spectrum and symbol panels ride WebSockets — and a real reconnect-backoff bug in GopherTrunk's console is a masterclass in real-time client design: reject before upgrade, reset backoff only on data, share the reconnect logic."
keywords: WebSocket reconnect, reconnect backoff bug, spectrum WebSocket, real-time client design, backoff reset, stale client, WebSocket lifecycle
level: advanced
status: full
prereq:
  - websockets
  - rate-limiting-and-quotas
gophertrunk_links:
  - title: Web console guide
    url: /web.html
    note: the console whose live panels — spectrum, symbols, plots — are the WebSocket clients this lesson dissects.
---

# The web console's sockets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The console's live panels — spectrum, symbol constellations — ride
**WebSockets**, streaming display frames many times a second. A real bug in
their reconnect path is this lesson's core: clients **reset their backoff on a
successful handshake**, so a server that accepted the upgrade and then closed
made the maximum-backoff cap **dead code** — one stale tab hammered the daemon
**2–4 times a second, forever**. The fixes generalise to every real-time
client: **validate before you upgrade** (fail the handshake, not the session),
**reset backoff only when data arrives**, and **centralise reconnect logic**
instead of copy-pasting it per panel.
</div>

This is the unit's engineering-postmortem lesson. The transport theory is
[Unit 3's](/learn/apis/websockets/); what a production system adds is the
failure story — and GopherTrunk's console once shipped a subtle, instructive
version of the classic real-time client bug. No blame, all lessons: this is
what the discipline looks like when reality tests it.

## The setup: streaming panels

Open the console's spectrum panel and it dials a WebSocket; the daemon builds
a DSP chain for that view and streams display frames — spectrum bins, symbol
constellations — several times a second. Request/response could never keep up;
this is squarely the [WebSocket quadrant](/learn/apis/websockets/): high-rate,
continuous, server-to-browser. Each socket names which SDR device it wants, and
therein lay the trap.

## The bug, in three stacking mistakes

**Mistake 1: accept first, validate second.** The handler upgraded the
connection *before* resolving which device the client asked for. Ask for a
device that doesn't exist — say, a stale tab from before a hardware change —
and you got a *successful* handshake followed immediately by an error close.
From the client's perspective: connection succeeded, then died.

**Mistake 2: reset backoff on `onopen`.** The reconnect logic did the
textbook exponential backoff — but reset the delay to minimum in the
open handler, reasoning "we connected; things are healthy again." Combined
with mistake 1, every doomed attempt *looked* healthy for a few
milliseconds — so the backoff reset, every time, and the loop ran at its
minimum delay. The `MAX_BACKOFF` constant sat in the code, syntactically
present, **semantically unreachable**.

**Mistake 3: per-panel copies.** The reconnect logic was copy-pasted across
the console's socket clients — several mount points — so the flaw existed in
every copy, multiplied by open panels. One copy even bound both `onerror` and
`onclose` to schedule reconnects while remembering only one timer handle —
two timers per failure, one of them unstoppable.

Result: one stale browser tab, left open across a daemon restart with
different hardware, reconnecting **2–4 times a second, indefinitely**, across
multiple panels — a well-intentioned client performing a slow-motion
denial-of-service, exactly the un-backed-off retry loop the
[rate-limiting lesson](/learn/apis/rate-limiting-and-quotas/) warned about.
Nothing was "down"; every individual piece worked as written.

## The fixes — each one a general principle

1. **Reject before the upgrade.** The handler now resolves the device *first*
   and answers an unknown one with a plain HTTP **404** — no upgrade, no
   phantom success. Principle: **fail requests at the cheapest, most honest
   layer.** A failed handshake is unambiguous to every client and lets HTTP's
   error machinery ([status codes](/learn/apis/methods-and-status-codes/),
   proxies, logs) do its job; accept-then-close launders an error into a
   success-shaped event.
2. **Reset backoff only when a frame arrives** (or the socket survives a
   grace period). Principle: **a handshake is a hello, not a heartbeat** — 
   proof of health is *the thing you connected for actually flowing*, the
   same only-data-counts rule the [WebSocket lesson](/learn/apis/websockets/)
   stated and this bug proved. With it, a permanently-failing endpoint walks
   politely up to maximum backoff and stays there.
3. **One reconnecting-socket implementation, shared.** The logic moved into a
   single module all panels use. Principle: lifecycle code — reconnect,
   backoff, liveness — is **infrastructure, not per-feature boilerplate**;
   copies drift, and each copy is another place the subtle bug survives.
4. **Re-validate assumptions on reconnect.** The stale tab was stale because
   it kept asking for a device that no longer existed; the client now
   reconciles its device choice against a fresh list instead of assuming
   forever. Principle: **a reconnect is a new world** — re-fetch what you
   believed, don't replay it.

> Rule of thumb for every real-time client you'll ever write: *reject early,
> reset on data, centralise the lifecycle, re-validate on reconnect.* Test the
> failure path specifically — connect to something that always fails and watch
> the backoff actually grow.

## Why this lesson sits here

Because it's the whole module in one bug. A [contract](/learn/apis/api-contracts/)
question (what does a successful handshake *promise*? — less than the client
assumed), a [protocol-layering](/learn/apis/websockets/) question (which layer
should reject — HTTP or the socket session?), and a
[citizenship](/learn/apis/rate-limiting-and-quotas/) question (what does your
client do to a struggling server?) — all hiding in twenty lines of reconnect
code that looked, to every reviewer, completely reasonable. Real-time clients
fail in the failure paths; that's where the engineering lives. The web-dev
module's [realtime lesson](/learn/web-dev/websockets-and-realtime/) shows the
browser-side machinery; you now know what to demand of it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — accept-then-close reads as success to the client, defeating backoff; a pre-upgrade 404 is an honest, unambiguous failure." markdown="0">
  <p class="knowledge-check__q">Quick check: why was upgrading the WebSocket before validating the requested device the root enabler of the reconnect storm?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Upgrading is expensive, so early upgrades overloaded the daemon's CPU</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The successful handshake made every doomed attempt look healthy — so clients reset their backoff each time and retried at full speed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">WebSockets can't be closed by the server once upgraded</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The console's live panels are **WebSocket clients** streaming display frames
  — the right transport, with the standard lifecycle burdens.
- The storm needed three stacked mistakes: **accept-then-close** (error
  laundered into success), **backoff reset on handshake** (cap became dead
  code), and **copy-pasted lifecycle logic** (flaw everywhere at once).
- The fixes are universal: **reject before upgrade (HTTP 404)**, **reset
  backoff only on data**, **one shared reconnect implementation**,
  **re-validate state on every reconnect**.
- **Test the failure path** — point the client at a hostile endpoint and watch
  the backoff genuinely grow.
- Real-time clients are judged by their **failure behaviour**; the happy path
  is the easy 10%.

Next up: [Build your own client](/learn/apis/building-your-own-client/).
