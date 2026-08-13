---
title: "From the Issue Tracker, Part 19: One Render Loop — A Blank UI, a Host-less URL, and React Error #185"
description: A blank web UI, a WebSocket URL missing its host, sockets closing before the handshake, and React's maximum-update-depth error all traced to a single unstable selector in an effect dependency array — plus a separate go:embed trap that shipped a binary whose web assets were one .gitkeep file.
category: solution-postmortem
keywords: react error 185, blank web ui, websocket url, unstable selector, useeffect dependency, reconnect backoff, go embed, web dist, gitkeep, gophertrunk issue tracker
tags: [from-the-issue-tracker, web-ui, react, websocket, go-embed, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 19
---

*Part 19 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 18]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})
peeled three layers off a "stalled" decoder. This one moves to the browser, where a
blank screen, a malformed WebSocket URL, and a React crash looked like three bugs —
and were one line: an object that was never the same object twice.*

> **TL;DR:** In [#290](https://github.com/MattCheramie/GopherTrunk/issues/290), the
> web UI went blank with React error #185 (maximum update depth exceeded), the
> console showed a WebSocket URL with no host — `ws:/api/v1/events/ws` — and
> sockets logged "closed before established." One root cause explained the crash
> *and* the socket churn: a store selector that returned a **new object on every
> call** sat in a `useEffect` dependency array, and opening the stream
> synchronously wrote status back into the store — an unbounded loop that opened
> and killed a socket per render. The malformed URL was a separate string-concat
> bug, a flappy reconnect backoff got redesigned, one plausible hypothesis was
> tested and discarded, and a fifth, unrelated finding earned its own heading: a
> `go:embed` binary whose entire web UI was one `.gitkeep` file.

## The symptom: everything at once

The report contained three symptoms that didn't obviously share a cause. The UI
rendered nothing — a blank page over a console full of React's minified error
#185, "maximum update depth exceeded." The network tab showed WebSocket connection
attempts to `ws:/api/v1/events/ws` — note the single slash and the missing host.
And the socket lifecycle log repeated "WebSocket is closed before the connection is
established," dozens of times a second.

Three symptoms invite three investigations. The economical move — and the lesson of
this issue — is to ask first whether one mechanism could emit all of them.

## The loop

It could. The chain has four links, each innocent alone:

1. `selectClientConfig`, a store selector, built and returned a **new object on
   every invocation** — same contents, fresh identity each call.
2. `App`'s WebSocket effect listed that selector's result in its `useEffect`
   dependency array. New identity ⇒ dependencies "changed" ⇒ effect re-runs:
   tear down the socket, open a new one.
3. Opening the stream **synchronously wrote connection status into the store**.
4. A store write re-renders; the re-render calls the selector; the selector mints
   a new object; go to 2.

That's the whole machine. React counts the nested update depth, hits its limit,
and throws #185 — the blank screen. And every trip around the loop opened a
WebSocket and destroyed it milliseconds later, long before the server could
complete the handshake — the "closed before established" spam. Two of the three
symptoms were one loop, observed from two vantage points.

The fix is the standard prescription, applied ruthlessly: stabilize the selector
(memoize so equal contents mean equal identity), and key the effect on the
**primitive** values it actually depends on — `serverURL` and `token`, strings that
compare by value — rather than any object that carries them. Effects keyed on
primitives cannot be fooled by identity churn.

## The URL with no host

Symptom three fell separately. The WebSocket URL was assembled by string
concatenation, with a case-sensitive regex deciding how to rewrite the page's
`http`/`https` origin into `ws`/`wss`. Feed that pipeline an input it didn't
anticipate and it emits `ws:/api/v1/events/ws` — scheme, one slash, no authority.
The rebuild threw the hand-rolled parsing away in favor of the platform's `URL`
API: construct from the page origin, swap the protocol, set the path. The browser's
own parser guarantees a well-formed result or throws where you can see it.

## The reconnect storm

Fixing the loop exposed a residual flaw in the reconnect logic that the loop had
been masking. On `onopen`, the client immediately reset its reconnect backoff to
500 ms. Sound reasonable? It means a socket that connects and then drops — a
flapping network, an overloaded proxy, a server mid-restart — retries at 500 ms
*forever*: each brief success re-arms the fast retry. That's a reconnect storm with
extra steps.

The redesigned policy treats "connected" and "healthy" as different states:

| Rule | Why |
|---|---|
| Exponential backoff with jitter | flapping clients spread out instead of synchronizing |
| Reset backoff only after 5 s of healthy connection | a connect that instantly drops doesn't re-arm fast retry |
| Null out handlers on `close()` | a torn-down socket can't fire stale callbacks into fresh state |
| Coalesce inbound frames over 100 ms | a burst of events becomes one render, not fifty |

## The hypothesis that didn't hold

One early theory deserves its honest mention: that the dashboard was crashing on a
null decoded system/site while the scanner was in a perpetual control-channel hunt
— a plausible story, since the reporter's daemon was indeed hunting. It was tested
and it didn't hold; the crash reproduced with fully populated state. Recording the
disproven theory in the issue is part of why the thread stayed navigable: nobody
re-proposed it.

## The other bug: a binary with no UI inside

The same issue surfaced one more finding, unrelated to React and worth its own
post-it on every Go developer's monitor. Some users saw `404 page not found` at
`/` — not a blank UI, no UI at all.

GopherTrunk serves its web UI from the binary via `//go:embed all:dist`, which
snapshots `web/dist/` **at Go compile time**. The `make build` target compiled the
Go binary but had no dependency on the frontend build. Run it in a fresh checkout —
or any tree where `web/dist/` hadn't been populated — and the embed dutifully
captured the only file present: `.gitkeep`. The binary built, shipped, and ran;
`HasAssets()` saw no `index.html` and returned false; the `GET /` route was never
registered; the router answered 404.

"The right binary running with the wrong embed inside it" is a uniquely quiet
failure because every observable artifact — version string, API, logs — is
correct. The fix: `make dist` orders the frontend build before the Go build, and
the asset-less case now serves an HTML 404 that says what's missing and how to
build it, instead of a bare router default.

## What we keep

- **One unstable identity can be a whole bug cluster.** A selector returning fresh
  objects, an effect depending on it, and a synchronous store write closed a loop
  that presented as a crash *and* a network storm. Key effects on primitives.
- **Symptoms that arrive together should be suspected together.** Asking "what one
  mechanism emits all three?" before dividing the work is step one of the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Build URLs with the `URL` API.** String concatenation plus a regex produced a
  host-less WebSocket URL; the platform parser can't.
- **Backoff resets on *health*, not on *connect*.** Resetting to 500 ms in
  `onopen` turns every flap into a storm. Require a sustained healthy interval.
- **`go:embed` snapshots compile time, not runtime.** A build graph that doesn't
  order the frontend before the embed ships a working binary with an empty UI —
  and an explicit "assets missing" page beats a router 404.
- **Write down the theories that failed.** The null-state hypothesis was tested,
  disproven, and recorded — the cheapest possible gift to the next reader of the
  thread.

## Series navigation

← [Part 18: the stall that wasn't]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})
