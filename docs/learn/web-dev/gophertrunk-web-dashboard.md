---
slug: gophertrunk-web-dashboard
title: The GopherTrunk web dashboard
description: A worked example that ties the whole module together — how a live dashboard for a scanner is built, from the Go back end that serves decoded calls over a REST API and a real-time stream, to the browser that shows them the instant a radio keys up.
keywords: worked example, scanner dashboard, live dashboard, REST API, WebSocket, real-time, Go backend, event bus, full stack, GopherTrunk
level: advanced
status: full
prereq:
  - choosing-a-web-stack
faq:
  - q: "Why does a scanner even need a web dashboard?"
    a: "Because the interesting output — decoded calls, active talkgroups, which sites are up — arrives continuously and is far easier to watch in a browser than a terminal. A dashboard turns GopherTrunk's stream of events into a live, glanceable view anyone on the network can open, without touching the command line. It's the human-facing front end on top of the decoding engine."
  - q: "How does a call get from the radio to my screen?"
    a: "The [engine](/architecture.html) decodes a call and emits it on its internal event bus. The web back end subscribes to that bus, and pushes each new call to connected browsers over a real-time channel. The browser, which loaded recent history over the REST API when it opened, appends the call to the live table the moment it arrives — no refresh."
  - q: "Is this a heavy front-end framework app?"
    a: "No — and that's deliberate. The job is a live table of calls, so the stack is intentionally small: Go on the back end (the whole project is Go), a REST API for history, a real-time stream for live updates, and a light front end. It's a concrete example of the 'match the stack to the job' principle from the previous lesson."
---

# The GopherTrunk web dashboard

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the whole module in one worked example. GopherTrunk's
[engine](/architecture.html) decodes radio calls and emits them on an internal event
bus; a **Go back end** exposes them two ways — a **[REST API](/learn/web-dev/building-a-rest-api/)**
for history and a **[real-time stream](/learn/web-dev/websockets-and-realtime/)** for
live pushes — and a **light front end** renders them into a live table the instant a
radio keys up. It's a deliberately **small [stack](/learn/web-dev/choosing-a-web-stack/)
matched to the job**, and it exercises nearly every idea in this path. See the running
project at [gophertrunk.org](/).
</div>

Every idea in this module — [client and server](/learn/web-dev/client-server-web/),
[APIs](/learn/web-dev/building-a-rest-api/), [real-time](/learn/web-dev/websockets-and-realtime/),
[security](/learn/web-dev/web-security-essentials/), [deployment](/learn/web-dev/deploying-a-web-app/) —
has been a piece in isolation. This final lesson assembles them into one real thing:
the dashboard that shows what a scanner is hearing, live. It's a worked example, not
new theory, so read it as a walk-through of how the parts click together.

## The problem it solves

GopherTrunk is a headless [decoding engine](/architecture.html): it manages SDR
hardware, follows trunked systems, and decodes calls — but it has no screen. Its
output is a stream of events (a call started on this talkgroup, a site went active,
audio was recorded) arriving unpredictably and continuously. A terminal is a poor
place to watch that. The dashboard's job is to turn that live stream into a
**glanceable web view** anyone on the network can open in a browser — the human face
on top of the engine.

## The back end: engine to API

The engine already emits everything of interest on an internal **event bus** — one of
the sinks in the [architecture diagram](/architecture.html), alongside voice and
storage. The web back end, written in **Go** like the rest of the project, subscribes
to that bus and exposes it to browsers. It presents two surfaces, because a dashboard
needs both *history* and *what's happening now*:

- A **REST API** for state and history — recent calls, active talkgroups, system
  status — that a freshly opened page can [fetch](/learn/web-dev/fetching-data/) to
  populate itself.
- A **real-time stream** that pushes each new call as the engine emits it.

```http
GET /api/calls?limit=50     ->  200 OK   [ {call}, {call}, … ]   # history
GET /api/systems            ->  200 OK   [ {system, active} ]    # status
(live stream)               ->  {call}   pushed as each one is decoded
```

Because the engine is [Go with typed channels](/architecture.html), bridging the event
bus to an HTTP handler and a stream is a small, natural amount of code — the back end
is mostly a thin translation layer from internal events to web responses.

## The front end: a live table

The browser side is deliberately light, because the job is a table that updates. When
the page loads it does two things, exactly the pairing from the
[WebSockets lesson](/learn/web-dev/websockets-and-realtime/): fetch recent history
once over the REST API to fill the table, then subscribe to the live stream so every
new call appends at the top the moment it's decoded.

```javascript
// 1. Load history once so the table isn't empty on open
const history = await (await fetch("/api/calls?limit=50")).json();
history.forEach(renderCall);

// 2. Subscribe to the live stream — new calls appear instantly, no refresh
const live = new EventSource("/api/stream");
live.onmessage = (e) => renderCall(JSON.parse(e.data));   // prepend to the table
```

Each call becomes a row — talkgroup, time, duration, system — built by updating the
[DOM](/learn/web-dev/the-dom/) as data arrives. There's no heavy
[framework](/learn/web-dev/frontend-frameworks/) here because the UI is simple enough
not to need one; the [stack](/learn/web-dev/choosing-a-web-stack/) was matched to the
job rather than to fashion.

## Where the module's pieces show up

The dashboard is a checklist of the whole path in one place:

- **[Client–server](/learn/web-dev/client-server-web/) & [REST](/learn/web-dev/building-a-rest-api/)** —
  the browser requests, the Go server responds with JSON.
- **[Real-time](/learn/web-dev/websockets-and-realtime/)** — live calls pushed, not
  polled, so the table is current to the second.
- **[The DOM](/learn/web-dev/the-dom/) & [fetching data](/learn/web-dev/fetching-data/)** —
  the front end reads JSON and updates the page in place.
- **[Security](/learn/web-dev/web-security-essentials/)** — call text is encoded, not
  injected as HTML; the API is read-only and access-controlled.
- **[Static site](/learn/web-dev/static-vs-dynamic/)** — this very docs site
  ([gophertrunk.org](/)) is the static counterpart, built by
  [Jekyll](/learn/web-dev/templating-and-static-sites/), while the dashboard is the
  dynamic one.
- **[Deploying](/learn/web-dev/deploying-a-web-app/) & [monitoring](/learn/web-dev/monitoring-and-analytics/)** —
  it ships behind a [TLS reverse proxy](/learn/deployment/reverse-proxies-and-tls/) and
  is watched like any running service.

Seeing them together is the point: a web app isn't any one of these ideas, it's all of
them cooperating around a clear job.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the page fetches history once over REST, then subscribes to a live stream so new calls appear instantly." markdown="0">
  <p class="knowledge-check__q">Quick check: how does the dashboard show both past calls and new ones the instant they happen?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It reloads the whole page every second to get fresh data</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It fetches recent history once over the REST API, then subscribes to a live stream for new calls</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The browser reads the SDR hardware directly over USB</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The dashboard is the **human-facing front end** on top of GopherTrunk's headless
  [decoding engine](/architecture.html), turning a live event stream into a glanceable
  web view.
- The **Go back end** subscribes to the engine's internal **event bus** and exposes it
  two ways — a **REST API** for history/status and a **real-time stream** for live
  pushes.
- The **light front end** fetches history once, then subscribes to the stream, so new
  calls append to a live table the instant they're decoded — no refresh.
- The **stack is small and deliberate** — Go throughout, a simple front end, no heavy
  framework — a direct application of matching the stack to the job.
- It exercises nearly the whole module at once: client–server, REST, real-time, the
  DOM, security, static vs. dynamic, and deployment.

Next up: keep the [glossary](/learn/web-dev/glossary/) handy, and take it live with [Containers &amp; Deployment](/learn/deployment/).
