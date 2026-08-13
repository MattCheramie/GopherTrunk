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

## Cheat sheet

| Finding | Looked like | Actually | Fix |
|---|---|---|---|
| Blank UI + React #185 | render crash somewhere in a panel | unstable selector in an effect dep array + synchronous store write = unbounded loop | memoize the selector; key effects on primitive `serverURL`/`token` |
| `ws:/api/v1/events/ws` | backend/WS endpoint broken | string-concat URL builder with a case-sensitive regex | build with the platform `URL` API |
| "closed before established" spam | network problem | the same render loop opening and killing a socket per iteration | same fix as the loop |
| Residual ~500 ms reconnect storm | flaky network / proxy | `onopen` reset the backoff to the floor immediately | exponential backoff + jitter; reset only after 5 s healthy |
| `404 page not found` at `/` | missing route / broken build | `go:embed` snapshot contained only `.gitkeep` | `make dist` orders the frontend build before the Go build |

## In this post

- **The symptom: everything at once** — three symptoms, one report.
- **Proving the backend innocent** — health checks and a hand-opened WebSocket.
- **The loop** — four innocent links and one unbounded machine.
- **The URL with no host** — the string-concat bug that fell separately.
- **The reconnect storm** — the flaw the loop had been masking.
- **The hypothesis that didn't hold** — the null-state theory, tested and retired.
- **The other bug: a binary with no UI inside** — the `go:embed` trap.
- **What we keep** — the durable rules.

## The symptom: everything at once

The report contained three symptoms that didn't obviously share a cause. The UI
rendered nothing — a blank page over a console full of React's minified error
#185, "maximum update depth exceeded." The network tab showed WebSocket connection
attempts to `ws:/api/v1/events/ws` — note the single slash and the missing host.
And the socket lifecycle log repeated "WebSocket is closed before the connection is
established," dozens of times a second.

Three symptoms invite three investigations. The economical move — and the lesson of
this issue — is to ask first whether one mechanism could emit all of them.

## Proving the backend innocent

Before any frontend digging, the reporter did the isolation that kept the whole
thread tractable. From browser devtools, bypassing the SPA entirely:
`/api/v1/health` answered correctly, the metrics endpoint worked, and a manually
opened WebSocket to `ws://192.168.1.9:8080/api/v1/events/ws` connected immediately
and received well-formed frames:

```json
{"kind":"cchunt.progress", ...}
```

Transport functional, backend event pipeline functional, valid JSON arriving.
Whatever was wrong lived in the frontend's state and render handling — a
conclusion that survived every later twist in the thread. A second user
reproduced the blank screen on both Windows and Linux, ruling out anything
environment-specific. (One field detail that matters for every retest below: the
UI registers a service worker, so without a hard refresh or a storage clear you
can spend a round testing the *previous* bundle.)

## The loop

One mechanism could. The chain has four links, each innocent alone:

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

Two hardening details rode along. The first `connecting` status write is now
deferred to a microtask, so opening the event stream never writes to the store
synchronously from inside the effect that called it — link three of the chain is
severed independently of link one. And a regression test asserts the event stream
opens **exactly once** on mount, so any future re-introduction of identity churn
fails CI instead of a field session. A top-level error boundary went in at the
same time — later upgraded to auto-retry a caught render error (up to 3 attempts,
4 s apart, with the budget refreshed after a healthy stretch), so a transient
render fault degrades to a blip instead of a stranded blank page.

## The URL with no host

Symptom three fell separately. The WebSocket URL was assembled by string
concatenation, with a case-sensitive regex deciding how to rewrite the page's
`http`/`https` origin into `ws`/`wss`. Feed that pipeline an input it didn't
anticipate and it emits `ws:/api/v1/events/ws` — scheme, one slash, no authority.
The rebuild threw the hand-rolled parsing away in favor of the platform's `URL`
API: construct from the page origin, swap the protocol, set the path. The browser's
own parser guarantees a well-formed result or throws where you can see it. The
rebuilt path also handles uppercase schemes and falls back to the page's own
origin when the SPA is served by the daemon itself — and a malformed URL now
retries gracefully instead of throwing.

## The reconnect storm

Fixing the loop exposed a residual flaw in the reconnect logic that the loop had
been masking. On `onopen`, the client immediately reset its reconnect backoff to
500 ms. Sound reasonable? It means a socket that connects and then drops — a
flapping network, an overloaded proxy, a server mid-restart — retries at 500 ms
*forever*: each brief success re-arms the fast retry. That's a reconnect storm with
extra steps.

The reporter's daemon, meanwhile, was the perfect storm generator: it sat in
perpetual control-channel hunt (its decode problem was a separate issue), so the
backend emitted a steady drip of `cchunt.progress` events and nothing else, and
every sustained session gave the flapping socket endless chances to churn the
store. The UI would come up stable, then progressively degrade; the reporter
measured the console spam recurring "roughly every ~500 ms" — reading the backoff
floor from the outside. On the minified production build both failures surfaced as
the same bare `Minified React error #185`, which is part of why the original loop
and the residual storm were indistinguishable from the console.

The redesigned policy treats "connected" and "healthy" as different states:

| Rule | Why |
|---|---|
| Exponential backoff with jitter | flapping clients spread out instead of synchronizing |
| Reset backoff only after 5 s of healthy connection | a connect that instantly drops doesn't re-arm fast retry |
| Null out handlers on `close()` | a torn-down socket can't fire stale callbacks into fresh state |
| Coalesce inbound frames over 100 ms | a burst of events becomes one render, not fifty |
| Shape-validate frames at the ingest boundary | a malformed frame is dropped at the door, not thrown mid-render |

## The hypothesis that didn't hold

One early theory deserves its honest mention: that the dashboard was crashing on a
null decoded system/site while the scanner was in a perpetual control-channel hunt
— a plausible story, since the reporter's daemon was indeed hunting. It was tested
and it didn't hold; the crash reproduced with fully populated state. The disproof
had two concrete legs: the Scanner panel already null-guards every nested access,
and the daemon drops *events* — not connections — for slow clients, so a hunting
daemon can't starve the UI into a crash. A regression test now mounts the Scanner
mid-hunt with no lock, pinning the disproof in CI. Recording the disproven theory
in the issue is part of why the thread stayed navigable: nobody re-proposed it.

## The other bug: a binary with no UI inside

The same issue surfaced one more finding, unrelated to React and worth its own
post-it on every Go developer's monitor. Some users saw `404 page not found` at
`/` — not a blank UI, no UI at all.

The reporter's workaround is what made the diagnosis so clean. They rebuilt the
SPA with `make web-build` — the build succeeded and populated `web/dist/` — yet
the daemon still answered 404 at `/`. So they served the very same `web/dist`
with `python3 -m http.server 8090`, pointed it at the daemon, and everything
worked: dashboard up, both devices listed, live events streaming. Assets fine,
API fine. The only thing wrong was the *binary's copy* of the assets.

GopherTrunk serves its web UI from the binary via `//go:embed all:dist`, which
snapshots `web/dist/` **at Go compile time**. The `make build` target compiled the
Go binary but had no dependency on the frontend build. Run it in a fresh checkout —
or any tree where `web/dist/` hadn't been populated *before the binary was built* —
and the embed dutifully captured the only file present: `.gitkeep`. The binary
built, shipped, and ran; `HasAssets()` saw no `index.html` and returned false; the
`GET /` route was never registered; the stdlib router answered its blank 404.

"The right binary running with the wrong embed inside it" is a uniquely quiet
failure because every observable artifact — version string, API, logs — is
correct. The fix: a new `make dist` target chains the frontend build before the Go
build and became the operator-facing target, with `cross-build`, `release-dry-run`,
and `make run` gaining the same dependency — while plain `make build` deliberately
stays Go-only for fast backend iteration. And the asset-less case now serves an
HTML 404 that says what's missing and how to build it (`make dist`), instead of a
bare router default; the REST and WebSocket APIs keep working either way.

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

## FAQ

**How was the backend ruled out so early?**
By hand, from browser devtools: `/api/v1/health` and the metrics endpoint
answered, and a manually opened WebSocket to the events endpoint connected
instantly and received valid `cchunt.progress` frames. When the raw transport
works outside the app, the bug is inside the app. That one check saved the thread
from a backend detour it never needed.

**Was the crash related to the daemon being stuck hunting for a control channel?**
Indirectly. The render loop itself needed no particular backend behavior — a
single synchronous status write closed it. But the field reproduction rode on the
hunt: a daemon that never locks emits a continuous stream of progress events and
gives a flapping socket endless chances to churn the store, which is what dragged
the residual backoff flaw into view after the primary loop was fixed.

**What does React error #185 actually say?**
"Maximum update depth exceeded" — React detected more nested state updates than
its limit allows and aborted rendering rather than hang the tab. In a minified
production build it surfaces as just the error number, which is why two different
mechanisms (the render loop, then the reconnect storm) looked identical from the
console.

**Why did the daemon 404 when `web/dist` existed right there on disk?**
Because `//go:embed` copies files into the binary at compile time. The reporter's
`web/dist` was built *after* the binary was; the binary's embedded snapshot still
contained only `.gitkeep`. Rebuilding via `make dist` — which orders the frontend
build before the Go build — was the entire fix.

**How do I make sure I'm retesting the new bundle and not a cached one?**
The UI registers a service worker, so an ordinary reload can serve the previous
bundle. Hard-refresh or clear the site's storage after every rebuild — retest
rounds in this very issue depended on it.

## Series navigation

**Part 19 of 22** · ←
[Part 18: The Stall That Wasn't — A Dongle Off the Bus and an Opcode Off the Books]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})
· Next →
[Part 20: The Self-Consistent Trap — Round-Trip Tests That Validate Their Own Bugs]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
