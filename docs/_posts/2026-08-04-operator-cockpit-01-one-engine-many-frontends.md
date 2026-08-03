---
title: "The Operator's Cockpit, Part 1: One Engine, Many Front-Ends"
description: How one running GopherTrunk daemon exposes a single REST + Server-Sent-Events contract that both a React SPA in the browser and a Bubbletea terminal TUI render, kept honest by an api package that talks only to interfaces and never imports the SDR pool.
category: deep-dives
keywords: sdr operator console, rest sse api, react spa embedded go, bubbletea tui, one api two renderers, interface decoupled api, daemon front-end, live event stream, gophertrunk operator cockpit
tags: [operator-cockpit, api, sse, tui, web, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 1
---

*Part 1 of **The Operator's Cockpit**, a 14-part deep dive into how you actually
drive a running GopherTrunk. The engine decodes and records on its own; this
series is the surface a human uses to watch it, listen to it live, and change it
safely. The through-line is a single claim we test all the way down: there is
**one daemon** exposing **one REST + Server-Sent-Events contract**, and the
browser SPA and the terminal TUI are two renderers of that one contract. This
opener is the map of that whole journey.*

> **TL;DR:** A running GopherTrunk daemon is a **headless engine plus one HTTP
> surface**: read models over `GET /api/v1/*`, mutations over gated `POST`/`PATCH`,
> and a live push channel (SSE at `/api/v1/events`, its WebSocket twin at
> `/api/v1/events/ws`). The `internal/api` package talks only to **interfaces**
> — `EngineSnapshot`, `DevicesProvider`, `AudioController`, `SpectrumProvider` —
> so it never imports the SDR pool or the engine's concrete types. The React SPA
> (embedded in the binary) and the Bubbletea TUI are both just **clients** of
> that contract, which is why what you see in a browser and what you see over SSH
> are the same thing.

**Key takeaways**

- **One contract, two renderers.** The SPA and the TUI don't share code; they
  share the *API*. Every panel in either is a view over the same read models and
  the same event stream.
- **The api package is decoupled by interfaces.** `Server` holds `EngineSnapshot`,
  `DevicesProvider`, `AudioController`, `SpectrumProvider`, and a dozen more — the
  daemon injects real implementations, tests inject fakes, and `internal/api`
  imports no SDR driver.
- **Reads are open; writes are gated.** Every mutating route is wrapped in
  `s.gate(...)`, and `GET /api/v1/mutations` tells a client up front whether its
  credentials would pass, so a front-end can grey out write controls before
  trying them.
- **Optional subsystems are `nil`-able.** Audio, spectrum, hunt, bookmarks — each
  is an optional provider; when the daemon doesn't wire one, the route returns
  `503` instead of pretending. A front-end degrades a panel rather than crashing.

## Cheat sheet

| Concern | What it is | Where it lives |
|---|---|---|
| HTTP surface | routes, listener, graceful shutdown | `internal/api/server.go` (`Server`, `routes`) |
| Read models | JSON DTOs the panels render | `internal/api/handlers.go`, `api.go` |
| Live push | SSE fan-out of every bus event | `internal/api/sse.go` (`handleSSE`) |
| Live push (browser) | the same events over WebSocket | `internal/api/ws.go` (`handleWS`) |
| Mutation gate | auth wrapper on every write route | `internal/api/server.go` (`gate`), `auth.go` |
| Embedded SPA | the built Vite bundle, served at `/` | `web/embed.go`, `server.go` (`spaHandler`) |
| Terminal renderer | read-only Bubbletea view of the API | `internal/tui/app.go` (`Model`) |

## In this post

- **What "one engine, many front-ends" means** — and why it's an API boundary,
  not a UI toolkit.
- **The Server and its interfaces** — how `internal/api` stays free of the SDR
  pool.
- **Reads, mutations, and the live stream** — the three shapes of every route.
- **Two renderers** — the embedded React SPA and the Bubbletea TUI, side by side.
- **The map** — what the next thirteen posts build on top of this contract.

## What "one engine, many front-ends" means

Point a browser on your phone at a Raspberry Pi running GopherTrunk and you get a
full operator console: active calls, a spectrum waterfall, live audio, write-mode
config. SSH into the same Pi and run `gophertrunk tui` and you get *the same
information* in a full-screen terminal. These are not two applications that happen
to look alike. They are two **renderers of one contract**, and the contract is an
HTTP API.

That is the whole architectural bet of this series. The decode-and-record engine
has no idea a human is watching. It publishes events on an internal bus and holds
some read models in memory. The `internal/api` package wraps that state in an HTTP
surface — REST for pull, SSE/WebSocket for push — and *everything a human does*
goes through that surface. The browser doesn't get a privileged back door; neither
does the terminal. If a value isn't on the API, no front-end can show it, and the
moment it *is* on the API, both front-ends can.

Naming that boundary is what makes the rest of the series tractable. A bug where
the browser shows a stale call but the TUI is correct is a **client** bug. A bug
where *both* are wrong is a **server** bug. A feature that needs a new number on
screen is first an API change and only then a UI change. The seam tells you which
half of the codebase to open.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="One decode-and-record engine publishes to an internal event bus and read models; the api package wraps them in one REST plus Server-Sent-Events contract; the React SPA and the Bubbletea TUI are two renderers of that single contract">
  <rect x="8" y="84" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="104" text-anchor="middle" fill="currentColor" font-size="12">engine</text>
  <text x="68" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">decode + record</text>
  <line x1="128" y1="107" x2="152" y2="107" stroke="currentColor"/><polygon points="152,103 162,107 152,111" fill="currentColor"/>
  <rect x="162" y="70" width="140" height="74" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="232" y="94" text-anchor="middle" fill="var(--accent)" font-size="12">internal/api</text>
  <text x="232" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">read models + bus</text>
  <text x="232" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">REST · SSE · WS</text>
  <line x1="302" y1="90" x2="338" y2="60" stroke="currentColor"/><polygon points="333,58 344,55 338,66" fill="currentColor"/>
  <line x1="302" y1="124" x2="338" y2="154" stroke="currentColor"/><polygon points="338,148 344,159 333,156" fill="currentColor"/>
  <rect x="344" y="30" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="419" y="50" text-anchor="middle" fill="var(--accent)" font-size="12">React SPA</text>
  <text x="419" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">browser · embedded</text>
  <rect x="344" y="138" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="419" y="158" text-anchor="middle" fill="var(--accent)" font-size="12">Bubbletea TUI</text>
  <text x="419" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="9">terminal · over SSH</text>
  <text x="540" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same contract,</text>
  <text x="540" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="10">two renderers</text>
  <text x="340" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no front-end has a back door — if it isn't on the API, nothing can render it</text>
</svg>
<figcaption>The engine never renders. The api package is the only surface, and both front-ends are clients of it.</figcaption>
</figure>

## The Server and its interfaces

The heart of the contract is one struct, `api.Server`, and the striking thing
about it is what it *doesn't* depend on. It holds an `EngineSnapshot`, not a
`*trunking.Engine`. It holds a `DevicesProvider`, not an `*sdr.Pool`. Every
capability arrives as a small interface the daemon satisfies with a real object
and a test satisfies with a fake:

```go
// internal/api/server.go (shape)
// EngineSnapshot is the subset of trunking.Engine the API needs. Decoupling
// from the concrete type keeps the API testable with a fake engine.
type EngineSnapshot interface {
    ActiveCalls() []*trunking.ActiveCall
    ObservedCalls() []*trunking.ActiveCall
    IsKnownRadio(id uint32) bool
}

// DevicesProvider returns a snapshot of the SDR pool. The api package
// stays free of a hard dependency on internal/sdr's implementation.
type DevicesProvider interface {
    Snapshot() []sdr.SDRStatus
}

type Server struct {
    bus      *events.Bus       // the live push source
    engine   EngineSnapshot    // read models
    devices  DevicesProvider   // …audio, spectrum, hunt, scanner, bookmarks …
    webAssets fs.FS            // the embedded SPA (or nil)
    auth     *authState        // the mutation gate
    // …one field per optional subsystem, most nil-able
}
```

`ServerOptions` is the injection point: the daemon fills in the fields it has and
leaves the rest zero. That is not just tidiness. It is what lets `internal/api`
compile without importing a single SDR driver, and it is what makes the whole
surface testable end to end — the SPA-serving tests spin up a `Server` with a
`fstest.MapFS` for `WebAssets` and a fake bus, no radio in sight.

### How that principle shaped the Go code

- **The api package never imports the pool.** It imports `internal/events`,
  `internal/trunking`, and `internal/sdr` only for the `SDRStatus` DTO — never a
  driver. Every hardware-touching capability is behind a provider interface the
  daemon wires.
- **Optional means `nil`.** `audioPub`, `spectrum`, `hunt`, `bookmarks`, `diag`,
  `symbols` are all pointers or interfaces that may be nil; each handler checks
  and returns `503` when its provider is absent. A build without SDRs doesn't
  pretend to have a waterfall.
- **The listener owns the lifecycle, not the handlers.** `Run` binds the socket,
  records the bound address (so `":0"` works in tests), and gives streaming
  connections a 30-second drain window on shutdown so a clean restart doesn't
  guillotine an in-flight call's audio.

## Reads, mutations, and the live stream

Every route in `routes()` is one of three shapes, and knowing which is which is
most of understanding the API.

**Reads** are plain `GET`s returning a JSON DTO: `/api/v1/systems`,
`/api/v1/calls/active`, `/api/v1/devices`, `/api/v1/scanner`. They're open — a
read never needs a token — because a scanner's state is not a secret on the LAN it
lives on.

**Mutations** are `POST`/`PATCH`/`DELETE` wrapped in one tiny middleware:

```go
// internal/api/server.go (shape)
func (s *Server) gate(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if status, reason := s.auth.authorize(r); status != 0 {
            s.writeError(w, status, reason) // 401 / 403
            return
        }
        h(w, r)
    }
}
```

So `mux.HandleFunc("PATCH /api/v1/audio", s.gate(s.handleAudioPatch))` reads as
"changing audio is a gated write," and the pattern repeats for every write route —
end a call, edit a talkgroup, start a hunt, save config. Part 3 is entirely about
what `authorize` decides.

**The live stream** is the third shape, and it's the one that makes a *cockpit*
instead of a dashboard you refresh. `handleSSE` subscribes to the internal event
bus and re-emits every event as one JSON envelope down a long-lived
`text/event-stream`:

```go
// internal/api/sse.go (shape)
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
    // clear the server WriteTimeout — this connection is long-lived
    _ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
    w.Header().Set("Content-Type", "text/event-stream")
    sub := s.bus.Subscribe()
    defer sub.Close()
    for {
        select {
        case <-r.Context().Done():
            return
        case ev := <-sub.C:
            dto := eventToDTO(ev)                 // one JSON envelope per event
            fmt.Fprintf(w, "event: %s\n", ev.Kind)
            // …data: lines… then flush
        }
    }
}
```

The same events are also served as WebSocket frames at `/api/v1/events/ws`
(`handleWS`) — identical payload shape, different transport — because, as we'll
see in Part 4, a browser `EventSource` can't attach an `Authorization` header,
so the SPA prefers the WS twin while `curl` and the TUI happily use SSE.

## Two renderers

The **React SPA** is baked into the binary. `web/embed.go` does a
`//go:embed all:dist` of the built Vite bundle, and `server.go` serves it at `/`
with an SPA fallback so client-side routes resolve. That whole story — embed
versus standalone, the fallback handler, the empty-`dist` sentinel — is Part 2.

The **TUI** is a separate program that is, by its own package doc, "a read-only
operator view over the daemon's REST + SSE API." Its root model is pure API
plumbing:

```go
// internal/tui/app.go (shape) — Init kicks off the polling fan + SSE connect
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        cmdPollSystems(m.cli), cmdPollActive(m.cli), cmdPollScanner(m.cli),
        cmdPollHunt(m.cli), cmdPollAudio(m.cli), cmdPollDevices(m.cli),
        cmdMutationStatus(m.cli),  // asks the daemon: may I write?
        connectSSE(m.cli),         // long-lived push, same events as the browser
    )
}
```

That is the *same* shape the SPA uses: a fan of interval polls for read models
plus one long-lived event connection, both hitting the same routes the browser
does. The TUI even asks `GET /api/v1/mutations` before enabling its write keys —
exactly what the SPA's store does before ungreying a button. Two codebases, one
contract, and the contract is the only thing they agree on.

## Where this goes next

[Part 2]({{ '/blog/deep-dives/operator-cockpit-02-react-spa-in-a-go-binary/' | relative_url }})
opens up the first renderer: how the built React bundle is `go:embed`-ed into the
daemon, served at `/` with an SPA fallback so `/scanner` and `/settings` resolve
on the client, and how the same binary degrades gracefully to a helpful message
when it was built without `make web-build`. From there the series climbs the
contract: the connect screen and auth handshake (Part 3), the event stream into
React (Part 4), live audio (Parts 5–6), and the browser's DSP canvases —
spectrum, constellation, eye, symbol scope (Parts 7–8) — before the map,
write-mode config, and the reflect-driven form that renders in both front-ends.

## FAQ

**Do the SPA and the TUI share any code?**
No. They share the *API*, not code — one is TypeScript/React, the other is
Go/Bubbletea. Both are clients of the same REST + SSE contract, which is exactly
why they stay consistent: there is one source of truth, and it's the daemon.

**Why interfaces like `EngineSnapshot` instead of the real engine type?**
So `internal/api` never has to import the SDR pool or the concrete engine. The
daemon injects real implementations; tests inject fakes. The whole HTTP surface is
unit-testable with a canned bus and no hardware.

**Why both SSE and WebSocket for the same events?**
They carry the identical `EventDTO` payload. SSE is simplest for `curl` and the
TUI; the browser SPA prefers the WebSocket twin because an `EventSource` can't send
an `Authorization: Bearer` header. Part 4 walks that trade-off in detail.

**What happens to a route whose subsystem isn't wired?**
It returns `503`. Audio, spectrum, hunt, and bookmarks are optional providers;
when the daemon starts without one (no SDRs, audio off, no storage), the route
answers "not wired" and the front-end degrades that one panel instead of failing.

**Is the API authenticated?**
Reads are open; mutations go through `s.gate(...)`. The policy is `auto` /
`required` / `disabled`, and `GET /api/v1/mutations` lets a client learn up front
whether its credentials would pass. Part 3 is the whole auth story.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: A React SPA Inside a Go Binary]({{ '/blog/deep-dives/operator-cockpit-02-react-spa-in-a-go-binary/' | relative_url }})
</content>
</invoke>
