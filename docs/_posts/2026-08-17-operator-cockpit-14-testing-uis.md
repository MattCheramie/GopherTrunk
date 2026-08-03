---
title: "The Operator's Cockpit, Part 14: Testing Browser & Terminal UIs"
description: The finale — how GopherTrunk tests a browser and a terminal operator UI without a live radio, standing the whole cockpit up in memory — httptest API contract tests, a byte-for-byte WAV header pinned on both the Go server and the TypeScript client, Bubbletea model tests that feed synthetic messages into the reducer, and pure SSE-parser and audio-framer unit tests.
category: deep-dives
keywords: test ui without radio, httptest api contract, bubbletea model test, sse parser test, wav header byte for byte, vitest jsdom audio, spa fallback test, in memory server test, one contract two renderers, gophertrunk operator cockpit
tags: [operator-cockpit, testing, httptest, bubbletea, vitest, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 14
---

*Part 14 — the finale — of **The Operator's Cockpit**. Thirteen posts built two
front-ends over one daemon and one REST + SSE API: a React SPA baked into the Go
binary and a Bubbletea TUI, each a renderer over the same contract. This closing
post answers the question hanging over all of it — how do you *test* a browser and
a terminal UI for a radio, without a radio? — and shows that the answer is the
same discipline that made the whole series possible: stand the contract up in
memory and pin it from both sides.*

> **TL;DR:** Nothing here needs an SDR. **API contract tests** use `httptest` and
> a fake embedded filesystem to prove the server serves the SPA, falls back to
> `index.html` on client routes, and never shadows `/api/*`. The **audio stream**
> is verified byte-for-byte: a Go test asserts the exact 44-byte WAV header, and a
> TypeScript test rebuilds that *same* header to prove the browser framer's offsets
> match the server's. **TUI model tests** feed synthetic Bubbletea messages
> straight into the reducer and assert on the model and its rendered view — no
> network. **Pure unit tests** cover the SSE parser (string in, events out) and
> the audio framer (bytes in, samples out). One contract, two renderers, both
> tested against it in memory.

**Key takeaways**

- **The server is testable without a browser.** `httptest` + a fake `fs.FS` SPA
  exercise the real routing: root serves index, unknown client routes fall back,
  `/api/*` is never shadowed, and a no-embed build returns a helpful message.
- **The wire contract is pinned on both sides.** The audio WAV header is asserted
  in Go *and* reconstructed byte-for-byte in TypeScript, so server and client can't
  drift on the format that carries every live call.
- **The TUI is testable without a terminal.** Model tests feed messages into
  `Update` and read back `SharedState` and `View()` — the reducer is a pure
  function of messages, so a synthetic `pollActiveMsg` proves the dashboard renders.
- **Everything runs in memory.** No SDR, no network, no CGO — `go test ./...` and
  `vitest` exercise the genuine code paths anywhere a toolchain runs.

## Cheat sheet

| Test kind | What it proves | Where it lives |
|---|---|---|
| SPA routing | serve / fall back / don't shadow APIs | `internal/api/spa_test.go` |
| Audio header (Go) | exact 44-byte RIFF/WAVE layout | `internal/api/handlers_audio_stream_test.go` |
| Audio framer (TS) | client offsets match the server | `web/src/audio/streamPlayer.test.ts` |
| TUI model | messages → SharedState → View | `internal/tui/app_test.go` |
| SSE parser | text stream → decoded Events | `internal/tui/client/sse_test.go` |

## In this post

- **API contract tests** — the SPA served against a fake filesystem.
- **The two-sided WAV header** — the same bytes asserted in Go and TS.
- **TUI model tests** — driving the reducer with synthetic messages.
- **Pure parser tests** — SSE and the audio framer, in isolation.
- **The cockpit, delivered** — what the whole series added up to.

## API contract tests: the SPA without a browser

The web console is embedded in the Go binary, which means the *serving* of it is
Go code with a real contract: the root must return `index.html`, deep client
routes must fall back to it (so a hard refresh on `/scanner` works), static assets
must be served directly, and `/api/*` must never be shadowed by the SPA fallback.
All of that is testable with a tiny fake filesystem and `httptest`:

```go
// internal/api/spa_test.go (shape)
func fakeSPAFS() fstest.MapFS {
    return fstest.MapFS{
        "index.html":    {Data: []byte("<!doctype html>…spa-root…")},
        "assets/app.js": {Data: []byte("console.log('hi');")},
    }
}

func TestSPA_ClientRouteFallsBackToIndex(t *testing.T) {
    base, teardown := mkServer(t, ServerOptions{Bus: bus, WebAssets: fakeSPAFS()})
    defer teardown()
    resp, _ := http.Get(base + "/scanner")           // an SPA client route
    // …assert 200 and body contains "spa-root" — i.e. index.html was served
}

func TestSPA_APIRoutesNotShadowed(t *testing.T) {
    // …GET /api/v1/health must NOT return the SPA body — the API wins
}
```

The suite covers the full matrix: root serves index, an asset serves its real
bytes, a client route falls back, `/api/v1/health` is *not* shadowed, and — the
nice touch — a build with **no embedded SPA** returns a helpful message mentioning
`make dist` and `/api/v1/health` rather than a bare `404`, while still not shadowing
sub-paths. None of this needs a real bundle; a two-entry `fstest.MapFS` stands in
for the whole SPA because the test is about *routing*, not content.

## The two-sided WAV header

Here is the sharpest example of "one contract, both sides," and it's the one that
carries every live call the console plays. The audio stream (`GET
/api/v1/audio/stream`) begins with a 44-byte RIFF/WAVE header, then raw PCM. The
Go test pins that header byte-for-byte:

```go
// internal/api/handlers_audio_stream_test.go (shape)
func TestStreamingWAVHeader_Shape(t *testing.T) {
    h := streamingWAVHeader(8000)
    // len == 44; h[0:4]=="RIFF"; h[8:12]=="WAVE"; h[12:16]=="fmt "; h[36:40]=="data"
    // audioFormat(h[20:22])==1 (PCM); channels(h[22:24])==1; rate(h[24:28])==8000;
    // bitsPerSample(h[34:36])==16
}
```

And the *browser's* test reconstructs the identical header, explicitly to prove the
client framer reads it at the offsets the server writes:

```ts
// web/src/audio/streamPlayer.test.ts (shape)
// Build a canonical 44-byte RIFF/WAVE header byte-for-byte the way the daemon
// does in internal/api/handlers_audio_stream.go (streamingWAVHeader). Pinning
// the layout here is the whole point — the framer's offsets must match the
// server's.
function makeHeader(sampleRate: number): Uint8Array {
  // …RIFF/WAVE/fmt /data, PCM=1, mono=1, rate, 16-bit — the same bytes as Go
}

describe("parseWavHeader", () => {
  it("extracts the format from a canonical 8 kHz header", () => { /* … */ });
});
```

Read those two comments together and the design pops out: the format that couples
the daemon to the browser is asserted independently on each side, so neither can
drift without turning a test red. It's the same "pin the wire, test both ends" move
the [Recording, Composition & Streaming]({{ '/blog/series/recording-streaming/' | relative_url }})
finale used for its aggregator uploads — here applied to the live audio the cockpit
plays. The server-side stream test even stands up a real `AudioPublisher` on the
bus with a fake audio source and asserts the header-then-PCM shape end to end, all
in memory; a `503` test proves the endpoint declines cleanly when audio isn't wired.

<figure class="lab-figure">
<svg viewBox="0 0 660 196" width="660" height="196" role="img" aria-label="The audio WAV header is pinned on both sides of the wire. The Go server writes a 44-byte RIFF/WAVE header, asserted byte-for-byte by a Go test. The TypeScript client framer reads the header at fixed offsets, and a TypeScript test reconstructs the same 44 bytes to prove the offsets match. Neither side can drift without a failing test.">
  <rect x="20" y="70" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="95" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">Go server</text>
  <text x="95" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="8">streamingWAVHeader</text>
  <line x1="170" y1="93" x2="300" y2="93" stroke="currentColor"/><polygon points="300,89 310,93 300,97" fill="currentColor"/>
  <rect x="248" y="72" width="164" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="90" text-anchor="middle" fill="currentColor" font-size="10">44-byte WAV header</text>
  <text x="330" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">RIFF·WAVE·fmt·data + PCM</text>
  <line x1="412" y1="93" x2="490" y2="93" stroke="currentColor"/><polygon points="490,89 500,93 490,97" fill="currentColor"/>
  <rect x="500" y="70" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="575" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">TS client</text>
  <text x="575" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="8">parseWavHeader</text>
  <line x1="95" y1="116" x2="95" y2="150" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="95" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Go test asserts</text>
  <text x="95" y="178" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the exact bytes</text>
  <line x1="575" y1="116" x2="575" y2="150" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="575" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TS test rebuilds</text>
  <text x="575" y="178" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the same bytes</text>
  <text x="330" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one format, asserted independently on each side — neither can drift silently</text>
</svg>
<figcaption>The live-audio header is the coupling between daemon and browser. Pinning it on both sides turns a format mismatch into a failing test, not a silent glitch.</figcaption>
</figure>

## TUI model tests: driving the reducer

Testing a terminal UI sounds like it needs a terminal. It doesn't — because a
Bubbletea program is an Elm-model reducer, and a reducer is a *pure function of
messages*. A model test builds a `Model` pointed at an unreachable base URL,
fakes a window size, and then feeds synthetic messages straight into `Update`,
asserting on the resulting `SharedState` and even the rendered `View()`:

```go
// internal/tui/app_test.go (shape)
func newTestModel(t *testing.T) *Model {
    cli := client.New("http://example.invalid", time.Second, false) // never dialled
    m := New(cli, Options{NoColor: true})
    updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
    return updated.(*Model)
}

func TestPollActiveMsg_PopulatesShared(t *testing.T) {
    m := newTestModel(t)
    call := client.ActiveCallDTO{Talkgroup: &client.TalkgroupDTO{ID: 42, AlphaTag: "Dispatch"}}
    updated, _ := m.Update(pollActiveMsg{calls: []client.ActiveCallDTO{call}})
    m = updated.(*Model)
    if !strings.Contains(m.View(), "Dispatch") { t.Errorf("Dispatch not in dashboard") }
}
```

The base URL is `http://example.invalid` precisely because the network is never
touched — the test *is* the daemon, handing the reducer exactly the message a real
poll would produce. The same technique covers the rest of the reducer's behaviour:
`TestEventMsg_LandsInRingBuffers` proves an SSE `tone.alert` lands in both the event
and tone ring buffers, `TestSSEDownMsg_SchedulesReconnect` proves a dropped stream
yields a reconnect command, and `TestPanelSwitch_DigitAndTab` drives real `KeyMsg`
key presses and checks the active panel. Every branch of Part 12's reducer is a
message fed in and a state read back.

## Pure parser tests: SSE and the audio framer

Under the model tests sit the smallest, sharpest tests — pure functions with no
framework at all. The SSE parser is verified with string input and a channel of
decoded events:

```go
// internal/tui/client/sse_test.go (shape)
func TestParseSSE_BasicEvents(t *testing.T) {
    body := "event: cc.locked\ndata: {\"kind\":\"cc.locked\",…}\n\n" +
            "event: grant\ndata: {\"kind\":\"grant\",…}\n\n"
    ch := make(chan Event, 8)
    parseSSE(strings.NewReader(body), ch) // → 2 Events with the right kinds + times
}
```

Companion cases pin the spec edges: `TestParseSSE_IgnoresComments` (a `:` keep-alive
line produces no event) and `TestParseSSE_FlushesTrailingEventWithoutBlankLine` (a
server that closes mid-event still flushes what it has). On the browser side, the
audio module's unit tests do the same for the framer that turns the WAV byte stream
into `Float32Array` audio — `parseWavHeader`, `PcmFramer`, `int16ToFloat32` — each
a bytes-in, samples-out function tested under `vitest`'s jsdom environment with no
`AudioContext` required. These are the load-bearing conversions of the whole live-
audio path, and they're tested as plain functions, in milliseconds.

### Why all of it runs in memory

The through-line of every test in this post — and, quietly, of the whole project —
is that **nothing links a radio, a browser, or C.** The API tests use `httptest`
and `fstest.MapFS`; the audio tests use an in-memory publisher and a synthetic WAV;
the TUI tests feed messages to a pure reducer; the parser tests are string/byte
functions. So `go test ./...` and `vitest` exercise the *genuine* routing, framing,
reducing, and parsing code on any machine with a toolchain — the same pure-Go,
zero-CGO posture the [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }})
and Recording series lean on, carried all the way out to the operator surfaces.

## The cockpit, delivered

Fourteen posts ago the pitch was simple: one daemon that decodes and records on its
own, and two front-ends that let a human watch it, listen to it live, and change it
safely — a React SPA baked into the binary and a Bubbletea TUI, both speaking the
*same* REST + SSE API. We built it end to end. Part 1 established the one contract;
Parts 2–6 built the live audio cockpit and its AudioWorklet ring buffer; Part 7's
spectrum and Part 8's constellations put the RF on a canvas; Part 9 mapped it;
Part 10 made writes safe; Part 11 packaged it as a phone app; Part 12 rebuilt the
whole thing in a terminal over the identical API; Part 13 generated the config
editor for both from one schema. And this post showed that every layer of it is
tested without a radio, with the wire contract pinned on both sides.

That is the payoff of *one API, two renderers*: a constellation you can read on a
phone at a hilltop and a dashboard you can read over SSH are the same daemon seen
two ways, and a single `httptest` server plus a `vitest` run can prove both of them
correct before an antenna is ever connected.

## FAQ

**How do you test a UI that needs a radio, without a radio?**
You test the layers under the pixels. The server's routing and audio contract are
`httptest`-driven; the TUI is a pure reducer fed synthetic messages; the SSE and
audio-framer conversions are plain functions. None of them need an SDR — they need
the same code paths the daemon runs, exercised in memory.

**Why assert the WAV header in both Go and TypeScript?**
Because it's the format coupling the daemon to the browser. Asserting the exact 44
bytes in Go and reconstructing them in TypeScript means server and client are held
to the same layout independently — a mismatch turns a test red instead of producing
a silent audio glitch.

**Do the TUI tests open a terminal or a socket?**
Neither. A Bubbletea model is a reducer, so a test builds a `Model` (pointed at an
unreachable URL), feeds it the exact messages a poll or key press would produce, and
reads back `SharedState` and `View()`. It's a pure function of messages.

**What does the fake SPA filesystem prove?**
Routing, not content. A two-entry `fstest.MapFS` lets the tests verify the real
serving contract — root serves index, client routes fall back, assets serve
directly, `/api/*` is never shadowed, and a no-embed build returns a helpful
message — without building the real bundle.

**Can I run all of this on a machine with no SDR and no CGO?**
Yes — that's the point. `go test ./...` and `vitest` build and run the genuine
routing, framing, reducing, and parsing code with nothing linked but the standard
toolchain, on any platform.

## Series navigation

**Part 14 of 14** · ←
[Part 13: The Reflect-Driven Config Form]({{ '/blog/deep-dives/operator-cockpit-13-reflect-driven-config-form/' | relative_url }})
· This is the finale — back to the [series index]({{ '/blog/series/operator-cockpit/' | relative_url }}).

*Where to next? You can drive the daemon from anywhere now — the last step is keeping it up 24/7, which is exactly what [Running It For Real]({{ '/blog/series/running-it-for-real/' | relative_url }}) is about.*
