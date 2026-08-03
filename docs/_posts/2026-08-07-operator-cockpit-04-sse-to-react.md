---
title: "The Operator's Cockpit, Part 4: The Event Stream — SSE to React"
description: How GopherTrunk fans every internal bus event to clients as one JSON envelope — Server-Sent Events for the terminal and curl, a WebSocket twin for the browser that can't set an auth header — with jittered reconnect backoff, frame coalescing, and interval poll hooks that only re-render on real change.
category: deep-dives
keywords: server sent events react, websocket event stream, sse vs websocket auth header, reconnect backoff jitter, event envelope dto, poll hook change detection, live scanner events, gophertrunk operator cockpit
tags: [operator-cockpit, sse, web, react, api]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 4
---

*Part 4 of **The Operator's Cockpit**. Parts 1–3 got a paired, authenticated
client talking to one daemon. This post is how that client stays live: the daemon
fans every internal bus event out as one JSON envelope, over Server-Sent Events
for the terminal and a WebSocket twin for the browser, and React folds the batch
into a store without thrashing the DOM.*

> **TL;DR:** `handleSSE` subscribes to the internal event bus and re-emits each
> event as one `EventDTO` down a long-lived `text/event-stream`; `eventToDTO` maps
> known payloads to JSON-friendly shapes and scrubs non-finite floats so a stray
> `NaN` can't kill the stream. The browser can't attach an auth header to an
> `EventSource`, so the SPA uses the **WebSocket twin** at `/api/v1/events/ws`
> instead — same `EventDTO` per frame — through `openEventStream`, which adds
> jittered reconnect backoff, a 100 ms coalescing window, and teardown guards.
> Slower state that isn't event-driven rides `useDataPoll`, which only invokes
> `onData` when the serialized snapshot actually changed.

**Key takeaways**

- **One envelope, two transports.** SSE and WebSocket both carry the identical
  `EventDTO`. The browser prefers WS purely because `EventSource` can't send
  `Authorization: Bearer`.
- **The ingest boundary is defensive.** The daemon scrubs non-finite floats before
  marshaling; the client re-validates each frame's shape before it reaches a
  panel. A malformed frame is dropped, not fatal.
- **Reconnect is jittered and stability-gated.** Backoff doubles to 30 s with
  equal jitter, and only resets after a connection has *held* for 5 s — a flapping
  socket doesn't storm the floor.
- **Polls only fire on change.** `useDataPoll` serializes each snapshot and skips
  `onData` when it's identical, killing the poll-induced re-render flicker every
  panel used to hand-roll.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| SSE fan-out | bus event → `text/event-stream` | `internal/api/sse.go` (`handleSSE`) |
| Envelope map | payload → JSON DTO, scrub `NaN`/`Inf` | `internal/api/sse.go` (`eventToDTO`) |
| WS twin | same events, browser-friendly transport | `internal/api/ws.go` (`handleWS`) |
| Browser stream | reconnect, coalesce, teardown guards | `web/src/api/events.ts` (`openEventStream`) |
| Poll hook | interval fetch, change-detect, stale flag | `web/src/hooks/useDataPoll.ts` |
| Active-calls poll | keep live-call snapshot fresh anywhere | `web/src/hooks/useActiveCallsPoll.ts` |

## In this post

- **The fan-out** — one bus subscription per client, one envelope per event.
- **Why the browser picks WebSocket** — the header `EventSource` can't send.
- **Reconnect done right** — jitter, a stability gate, and frame coalescing.
- **Polls that don't flicker** — change detection in `useDataPoll`.
- **Push plus pull** — how the two update paths divide the work.

## The fan-out

Everything the engine does — a call starts, a grant lands, encryption metadata
arrives, an SDR changes state — is an event on the internal bus. The live cockpit
is, at bottom, a bus subscriber that happens to live across an HTTP connection.
`handleSSE` is that bridge: one subscription per connected client, one JSON
envelope per event, held open until the client leaves:

```go
// internal/api/sse.go (shape)
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
    _ = http.NewResponseController(w).SetWriteDeadline(time.Time{}) // long-lived
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("X-Accel-Buffering", "no") // don't let nginx buffer it
    sub := s.bus.Subscribe()
    defer sub.Close()
    for {
        select {
        case <-r.Context().Done():
            return
        case ev := <-sub.C:
            dto := eventToDTO(ev)
            payload, _ := json.Marshal(dto)
            fmt.Fprintf(w, "event: %s\n", sanitizeForHeader(string(ev.Kind)))
            for _, line := range strings.Split(string(payload), "\n") {
                fmt.Fprintf(w, "data: %s\n", line) // SSE reframes per line
            }
            fmt.Fprint(w, "\n")
            flusher.Flush()
        }
    }
}
```

`eventToDTO` is where a raw bus payload becomes something a JSON client can
consume. It switches on the concrete payload type — `CallStart`, `Grant`,
`CallEncryption`, `Affiliation`, and the rest — mapping each to a tagged DTO, and
passing anything it doesn't recognise through untouched. The last line before it
returns is the one that has saved the stream more than once:

```go
// internal/api/sse.go (shape) — eventToDTO tail
// Strip non-finite floats (±Inf/NaN) from the chosen payload so a stray
// metric from a marginal carrier can't fail json.Marshal and tear down the
// live WS/SSE stream (issue #648). Operates on a copy; the bus payload and
// on-disk sinks keep their original values.
dto.Payload = scrubNonFinite(dto.Payload)
```

A single `NaN` in a demod-quality field would make `json.Marshal` error, and one
failed marshal on a shared stream would drop it for that client. Scrubbing a copy
at the ingest boundary makes one bad number a non-event.

## Why the browser picks WebSocket

Server-Sent Events is the natural fit here — it's a one-way text stream, exactly
what an event feed is — and the terminal TUI and `curl` use it directly. But the
browser SPA uses the *WebSocket* twin at `/api/v1/events/ws`, and the reason is a
single missing capability, spelled out at the top of the client:

```ts
// web/src/api/events.ts (shape)
// WebSocket is used rather than SSE because browsers cannot attach the
// Authorization header to an EventSource; the WS upgrade carries the same
// payload shape (one JSON EventDTO per frame).
```

An auth-gated daemon needs the bearer token on the event connection, and the
`EventSource` API has no way to set request headers. WebSocket does (via the
upgrade), and — crucially — it delivers the *identical* `EventDTO` per frame, so
the client code downstream doesn't care which transport carried it. Same
envelope, different pipe: the daemon serves both from the same bus so no logic
diverges between "what the terminal sees" and "what the browser sees."

## Reconnect done right

A live connection *will* drop — the daemon restarts, wifi hiccups, a proxy reaps an
idle socket. `openEventStream` is mostly the machinery for surviving that
gracefully, and it gets three things right that a naive reconnect loop gets wrong.

**Jittered backoff.** Delay doubles from 500 ms to a 30 s ceiling, but each wait is
*equal-jittered* — half the base plus a random half — so many clients reconnecting
after a daemon restart don't synchronise into a thundering herd, and a single
client never collapses toward a zero-delay busy loop:

```ts
// web/src/api/events.ts (shape)
const jittered = (base: number) => base / 2 + Math.random() * (base / 2);

ws.onopen = () => {
  setStatus("open");
  // Only reset the backoff once the socket has HELD for STABLE_MS. A socket
  // that opens then drops immediately keeps backing off instead of storming.
  stableTimer = setTimeout(() => { backoff = INITIAL_BACKOFF; }, STABLE_MS);
};
```

**A stability gate.** The subtle bug is resetting the backoff on `onopen` alone: a
flapping connection that opens and dies every 200 ms would reset to the floor each
time and hammer the daemon. So the reset is deferred until the connection has
*survived* `STABLE_MS` (5 s). Flap, and the backoff keeps climbing.

**Frame coalescing.** Events arrive in bursts — a busy control channel emits many
per second. Delivering each one as its own store write would trigger a render per
frame. Instead incoming frames accumulate for a 100 ms `FLUSH_MS` window and land
as one batch:

```ts
// web/src/api/events.ts (shape)
ws.onmessage = (msg) => {
  if (closed) return;
  const parsed = JSON.parse(msg.data);
  if (!isEventDTO(parsed)) return; // re-validate shape at the ingest boundary
  pending.push(parsed);
  scheduleFlush();                 // one onEvents(batch) per 100 ms
};
```

And the whole thing is teardown-safe: `close()` sets a `closed` flag, clears every
timer, and *nulls every socket handler* so an in-flight socket that opens or closes
after the React effect tore down can't call back into a store that's gone. The
React consumer runs `openEventStream` in an effect and returns its `close` as the
cleanup — mount opens the stream, unmount closes it, and no late event leaks.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="The internal event bus feeds one client subscription that emits identical EventDTO envelopes over Server-Sent Events to the terminal and over a WebSocket twin to the browser, which coalesces a burst of frames into one store batch per hundred milliseconds">
  <rect x="8" y="74" width="110" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="63" y="94" text-anchor="middle" fill="var(--accent)" font-size="11">event bus</text>
  <text x="63" y="109" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CallStart · Grant …</text>
  <line x1="118" y1="96" x2="150" y2="96" stroke="currentColor"/><polygon points="150,92 160,96 150,100" fill="currentColor"/>
  <rect x="160" y="72" width="130" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="225" y="92" text-anchor="middle" fill="currentColor" font-size="11">eventToDTO</text>
  <text x="225" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">map + scrub NaN</text>
  <line x1="290" y1="86" x2="326" y2="52" stroke="currentColor"/><polygon points="321,50 332,47 326,58" fill="currentColor"/>
  <line x1="290" y1="106" x2="326" y2="140" stroke="currentColor"/><polygon points="326,134 332,145 321,142" fill="currentColor"/>
  <rect x="332" y="28" width="140" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="402" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="11">SSE /events</text>
  <text x="402" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TUI · curl</text>
  <rect x="332" y="118" width="140" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="402" y="137" text-anchor="middle" fill="var(--accent)" font-size="11">WS /events/ws</text>
  <text x="402" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">browser (auth header)</text>
  <line x1="472" y1="141" x2="504" y2="141" stroke="currentColor"/><polygon points="504,137 514,141 504,145" fill="currentColor"/>
  <rect x="514" y="118" width="130" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="579" y="137" text-anchor="middle" fill="var(--accent)" font-size="11">store batch</text>
  <text x="579" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">coalesced 100 ms</text>
  <text x="330" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="10">identical EventDTO on both transports — the only difference is which pipe can carry the bearer token</text>
</svg>
<figcaption>One bus, one envelope, two transports. The browser reaches for WebSocket only because it can send the auth header the SSE EventSource cannot.</figcaption>
</figure>

## Polls that don't flicker

Not everything is event-driven. Some state — the systems list, the scanner status,
active calls — is a snapshot you *pull* on an interval. The trap there is that a
poll that returns identical data still triggers a store write and a re-render, so a
table flickers every two seconds for no reason. `useDataPoll` closes that trap by
comparing each snapshot against the last and only delivering real changes:

```ts
// web/src/hooks/useDataPoll.ts (shape)
const run = useCallback(async () => {
  try {
    const data = await fetcherRef.current();
    const serialized = JSON.stringify(data);
    if (serialized !== lastSerialized.current) {
      lastSerialized.current = serialized;
      onDataRef.current(data); // only on a real change
    }
    setError(null); setStale(false); setLastUpdated(Date.now());
  } catch (e) {
    setError(e instanceof Error ? e.message : "request failed");
    setStale(lastSerialized.current !== null); // "stale" only if we had good data
  } finally {
    setLoading(false);
  }
}, []);
```

The hook also carries the bookkeeping every panel used to reinvent: a `loading`
flag until the first response, a `stale` flag when a fetch fails *after* we had
good data (so the UI can dim rather than blank), and a `resetKey` — the daemon base
URL — that clears the change-detection cache when you switch servers, so the new
daemon's data always lands even if it happens to serialize identically. A thin
wrapper, `useActiveCallsPoll`, uses it to keep the shared active-calls snapshot
fresh from *any* panel, so a spectrum view opened directly doesn't sit on a call
that ended long ago.

## Push plus pull

The two paths divide cleanly. **Push** (the event stream) carries the things you
must see the instant they happen — a call starting, a grant, an alert — and the
100 ms coalescing keeps a burst from thrashing React. **Pull** (`useDataPoll`)
carries slower, whole-snapshot state where a 2-second refresh is fine and change
detection keeps it quiet. Both feed the *same* Zustand store the panels render
from, and both key off the same `ClientConfig` so switching daemons swaps every
panel's data at once. The store is the single live picture; SSE/WS and the poll
hooks are just two ways to keep it current — which is exactly the "one snapshot,
many writers" shape the TUI's `SharedState` uses one language over.

## Where this goes next

[Part 5]({{ '/blog/deep-dives/operator-cockpit-05-live-audio-cockpit/' | relative_url }})
turns from *events about* audio to the audio itself: streaming live PCM to the
browser as a continuous WAV, an AudioWorklet ring buffer that emits silence on
underrun instead of glitching, and the gapless-playback machinery that makes a
scanner you can actually *listen to* over the network. It's the same "long-lived
HTTP connection carrying a live feed" idea as this post, one layer closer to the
speaker — and it leans on the `fetch`-with-header pattern for the same auth reason
the WebSocket twin exists.

## FAQ

**Why does the daemon offer both SSE and WebSocket for events?**
They carry the identical `EventDTO`. SSE is the simplest fit for `curl` and the
terminal TUI; the browser needs WebSocket because an `EventSource` can't attach the
`Authorization: Bearer` header an auth-gated daemon requires.

**What stops one bad event from killing the stream?**
Two guards. The daemon scrubs non-finite floats (`NaN`/`Inf`) from a copy of each
payload before marshaling, so a stray metric can't fail `json.Marshal`; the client
re-validates each frame's shape and silently drops a malformed one.

**Why coalesce frames instead of delivering each immediately?**
A busy control channel emits many events per second. Batching them over a 100 ms
window means one store write and one render pass per burst instead of dozens —
smooth UI, same data.

**Why does reconnect wait to reset its backoff?**
To survive a flapping connection. Resetting on `onopen` alone lets a socket that
opens and dies repeatedly hammer the daemon at the backoff floor; requiring the
connection to hold for 5 s first keeps the backoff climbing until it's genuinely
stable.

**When is a poll used instead of an event?**
For whole-snapshot state that isn't naturally event-shaped — systems, scanner
status, active calls. `useDataPoll` refetches on an interval but only re-renders on
a real change, and flags data `stale` when a refresh fails after prior success.

## Series navigation

**Part 4 of 14** · ←
[Part 3: The Connect Screen & Auth Handshake]({{ '/blog/deep-dives/operator-cockpit-03-connect-screen-auth/' | relative_url }})
· Next →
[Part 5: The Live Audio Cockpit]({{ '/blog/deep-dives/operator-cockpit-05-live-audio-cockpit/' | relative_url }})
</content>
