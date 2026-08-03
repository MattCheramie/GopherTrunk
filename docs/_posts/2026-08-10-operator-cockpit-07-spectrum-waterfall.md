---
title: "The Operator's Cockpit, Part 7: Live Spectrum & Waterfall in the Browser"
description: How GopherTrunk streams FFT frames from the SDR broker over a WebSocket and paints a scrolling waterfall plus analyzer trace on a canvas — a per-request FFT producer, a rate-clamped frame protocol, ImageData row rendering, and an FFT-shifted x-to-frequency mapping that powers hover readout and click-to-tune.
category: deep-dives
keywords: live spectrum waterfall browser, fft frame websocket stream, canvas waterfall imagedata, click to tune spectrum, sdr spectrum panel react, dbfs colormap palette, x to frequency mapping, gophertrunk operator cockpit
tags: [operator-cockpit, spectrum, web, react, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 7
---

*Part 7 of **The Operator's Cockpit**. The audio feed of Parts 5–6 was one kind of
live stream; this is another — FFT frames instead of PCM. The daemon streams a
power spectrum of an SDR's passband, and the browser paints it as a scrolling
waterfall you can hover to read a frequency and click to retune, all over the same
API surface.*

> **TL;DR:** `GET /api/v1/spectrum/devices` lists streamable SDRs;
> `WS /api/v1/spectrum/stream?device=…&bins=…&fps=…` opens a per-request FFT
> producer on the daemon's iqtap broker and pushes one JSON `SpectrumFrame`
> (`center_hz`, `sample_rate_hz`, `bins[]` in dBFS) per tick, rate- and
> size-clamped server-side. The `Spectrum.tsx` panel opens that stream through the
> same reconnecting-WebSocket pattern as the event feed, keeps a ring of recent
> rows, and paints two canvases: a live analyzer trace and a `ImageData` waterfall.
> One shared **FFT-shifted x→frequency mapping** drives the hover readout and
> click-to-tune, and a click `POST`s the new centre through the gated tune route.

**Key takeaways**

- **FFT lives on the daemon.** A per-request producer computes frames off the
  shared iqtap broker; the browser only renders. `bins` must be a power of two and
  is clamped `64..16384`; `fps` is clamped `1..30`.
- **One frame is one waterfall row.** New frames prepend to a fixed-length ring;
  `renderWaterfall` writes a full `ImageData` in one pass, newest row on top.
- **A single mapping powers hover and tune.** `xRatioToHz` maps a canvas x back
  through the FFT-shifted bin layout, so the hover frequency and the click-to-tune
  target always agree.
- **Click-to-tune is a gated mutation.** A click resolves to a centre frequency and
  `POST`s `/api/v1/spectrum/devices/{serial}/tune`, wrapped in the same `s.gate`
  auth as every other write.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Device list | streamable SDRs + P25 mode + CC hint | `internal/api/spectrum.go` (`SpectrumDevice`) |
| Frame stream | per-request FFT producer over WS | `internal/api/spectrum.go` (`handleSpectrumStream`) |
| Tune route | gated click-to-tune write | `internal/api/spectrum.go` (`handleSpectrumTune`) |
| Stream client | reconnecting WS, one frame per message | `web/src/api/spectrum.ts` (`openSpectrumStream`) |
| Panel | canvas waterfall + analyzer + hover | `web/src/panels/Spectrum.tsx` |
| x→Hz mapping | shared by hover readout + tune | `Spectrum.tsx` (`xRatioToHz`) |

## In this post

- **The device list** — what a streamable SDR advertises.
- **The frame protocol** — one clamped producer, one JSON frame per tick.
- **Painting the waterfall** — a ring of rows and one `ImageData` write.
- **One mapping, two features** — hover readout and click-to-tune.
- **Tuning is a gated write** — the click that changes the radio.

## The device list

Before there's a waterfall there's a picker. `GET /api/v1/spectrum/devices`
returns the SDRs the daemon can stream, each described by a small DTO that carries
more than an identifier — it carries hints the panels use to default sensibly:

```go
// internal/api/spectrum.go (shape)
type SpectrumDevice struct {
    Serial           string `json:"serial"`
    Role             string `json:"role"`          // "control" | "voice" | …
    CenterHz         uint32 `json:"center_hz"`
    SampleRateHz     uint32 `json:"sample_rate_hz"`
    // The configured P25 demod mode of the system this SDR decodes, so the
    // symbol/constellation panels auto-pick a receiver without asking.
    P25Modulation    string `json:"p25_modulation,omitempty"`
    // The control-channel frequency inside this SDR's passband — the Plots
    // panels rest their view here so they default off the useless DC spike
    // onto a decodable channel (#557).
    ControlChannelHz uint32 `json:"control_channel_hz,omitempty"`
}
```

`ControlChannelHz` is a small idea with a big payoff: a spectrum view that opens
centred on the SDR's DC spike shows the operator nothing useful, so the daemon
tells the client where the decodable channel actually is. The client-side
`defaultSymbolDevice` complements it by preferring the `control`-role SDR, so a
plot opened during active decoding lands on the dongle that's carrying signal
rather than an idle aux one (issue #402).

## The frame protocol

The FFT itself runs on the daemon. `handleSpectrumStream` upgrades the request to a
WebSocket, opens a *per-request* producer against the SDR (via the
`SpectrumProvider` interface, so `internal/api` never imports the broker), and
pushes one JSON `SpectrumFrame` per tick until the client leaves. The knobs are
clamped server-side so a client can't ask for something abusive:

```go
// internal/api/spectrum.go (shape) — handleSpectrumStream
bins := parseIntQuery(q, "bins", 4096, 64, 16384) // clamp size
if bins&(bins-1) != 0 {
    s.writeError(w, http.StatusBadRequest, "bins must be a power of two")
    return
}
fps := parseFloatQuery(q, "fps", 10, 1, 30)       // clamp rate
frames, cleanup, err := s.spectrum.OpenStream(ctx, serial, bins, fps)
if err != nil { /* WS close frame */ return }
defer cleanup()

ping := time.NewTicker(30 * time.Second)          // keep idle proxies from reaping
for {
    select {
    case <-ctx.Done(): return
    case <-ping.C: /* WriteMessage(Ping) */
    case f, ok := <-frames:
        if !ok { return }
        body, _ := json.Marshal(f)                // {ts_ns, center_hz, sample_rate_hz, bins[]}
        conn.WriteMessage(websocket.TextMessage, body)
    }
}
```

Each frame is a snapshot of the passband: `center_hz`, `sample_rate_hz`, and a
`bins` array of dBFS values. The browser's `openSpectrumStream` consumes it with
the *same* reconnecting-WebSocket shape as the event feed in Part 4 — jittered
backoff to a 30 s ceiling, teardown-safe — just with `onFrame` instead of
`onEvents`. It's the third long-lived stream in the series, and the pattern is
deliberately identical so there's one way to think about "a live connection that
might drop."

## Painting the waterfall

On the client, each arriving frame does two things: it prepends a row to a
fixed-length ring, and it triggers a render of both canvases. The ring is just an
array capped at `HISTORY_ROWS`:

```tsx
// web/src/panels/Spectrum.tsx (shape) — onFrame
onFrame: (f) => {
  setLatest(f);
  latestRef.current = f;
  const row = new Float32Array(f.bins);
  rowsRef.current = [row, ...rowsRef.current.slice(0, HISTORY_ROWS - 1)]; // newest first
  renderAnalyzer(analyzerRef.current, f);            // top: live trace
  renderWaterfall(canvasRef.current, rowsRef.current, f, bookmarksRef.current); // bottom: history
},
```

`renderWaterfall` is where the performance choice lives. Rather than drawing
pixel-by-pixel with canvas calls, it builds one `ImageData` for the whole canvas
and writes it in a single `putImageData` — each row's dBFS bins are nearest-neighbor
resampled to the canvas width and colour-mapped through a fixed 5-stop palette:

```tsx
// web/src/panels/Spectrum.tsx (shape) — dbToColor, a fixed [-100,0] dBFS ramp
function dbToColor(db: number): [number, number, number] {
  if (db <= DB_FLOOR) return [0, 0, 0];               // ≤-100 dBFS → black
  if (db >= DB_CEIL)  return [255, 0, 0];             //   0 dBFS → red
  const t = (db - DB_FLOOR) / (DB_CEIL - DB_FLOOR);   // black→blue→cyan→yellow→red
  // …piecewise interpolation across the ramp…
}
```

The analyzer canvas above it draws the *same* newest frame as a filled line trace
with a 20 dB grid, sharing the waterfall's full-width x mapping so a peak in the
trace lines up vertically with its streak in the waterfall below. Bookmark markers
are painted as cyan ticks along the top edge — but only for bookmarks whose
frequency actually falls inside the visible band, so an out-of-band bookmark simply
doesn't clutter this view.

## One mapping, two features

The clever, reused bit is a single function that maps a horizontal canvas position
back to a frequency, accounting for the FFT layout where the leftmost pixel is
`center − sampleRate/2` and the rightmost is `center + sampleRate/2`:

```tsx
// web/src/panels/Spectrum.tsx (shape)
function xRatioToHz(frame: SpectrumFrame, xRatio: number): number {
  const sampleRate = frame.sample_rate_hz;
  return frame.center_hz - sampleRate / 2 + sampleRate * xRatio;
}
```

Both interactive features route through it. **Hover** takes the cursor's x-ratio,
computes the frequency, and reads the dBFS of the underlying bin from the newest
row — so the readout shows "446.0125 MHz · −62.3 dBFS" under the pointer.
**Click-to-tune** takes the same x-ratio, resolves the same frequency, and sends
it to the daemon. Because both use one mapping, the number you *see* on hover is
exactly the number you *tune to* on click — there's no drift between the readout
and the action.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="The daemon runs a per-request FFT producer on the SDR broker and streams JSON spectrum frames over a WebSocket; the panel prepends each frame to a row ring and paints an analyzer trace and an ImageData waterfall, and a shared x-to-frequency mapping drives both the hover readout and the gated click-to-tune write">
  <rect x="8" y="80" width="112" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="64" y="100" text-anchor="middle" fill="currentColor" font-size="11">iqtap broker</text>
  <text x="64" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">FFT producer</text>
  <line x1="120" y1="104" x2="152" y2="104" stroke="currentColor"/><polygon points="152,100 162,104 152,108" fill="currentColor"/>
  <rect x="162" y="80" width="126" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="100" text-anchor="middle" fill="var(--accent)" font-size="11">WS stream</text>
  <text x="225" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SpectrumFrame · clamped</text>
  <line x1="288" y1="104" x2="320" y2="104" stroke="currentColor"/><polygon points="320,100 330,104 320,108" fill="currentColor"/>
  <rect x="330" y="72" width="120" height="64" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="390" y="92" text-anchor="middle" fill="var(--accent)" font-size="11">row ring</text>
  <text x="390" y="107" text-anchor="middle" fill="var(--fg-muted)" font-size="9">newest first</text>
  <text x="390" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ImageData paint</text>
  <line x1="450" y1="90" x2="486" y2="60" stroke="currentColor"/><polygon points="481,58 492,55 486,66" fill="currentColor"/>
  <line x1="450" y1="118" x2="486" y2="150" stroke="currentColor"/><polygon points="486,144 492,155 481,152" fill="currentColor"/>
  <rect x="492" y="34" width="160" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="572" y="53" text-anchor="middle" fill="currentColor" font-size="11">hover readout</text>
  <text x="572" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="9">xRatioToHz + dBFS</text>
  <rect x="492" y="134" width="160" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="572" y="153" text-anchor="middle" fill="var(--accent)" font-size="11">click → tune</text>
  <text x="572" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">POST /tune · gated</text>
  <text x="330" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one x→frequency mapping feeds both, so the frequency you read is exactly the one you tune to</text>
</svg>
<figcaption>FFT on the daemon, canvas on the client; a single x→frequency mapping keeps the readout and the tune target in lockstep.</figcaption>
</figure>

## Tuning is a gated write

A click on either canvas is a *mutation of the radio*, so it goes through the same
auth gate as every other write. The handler is guarded by `s.gate`, checks the
serial and centre, and programs the SDR through the provider:

```go
// internal/api/spectrum.go (shape) — handleSpectrumTune (registered via s.gate)
if body.CenterHz == 0 {
    s.writeError(w, http.StatusBadRequest, "center_hz is required and must be > 0")
    return
}
if err := s.spectrum.Tune(serial, body.CenterHz); err != nil {
    s.writeError(w, http.StatusBadRequest, err.Error())
    return
}
w.WriteHeader(http.StatusNoContent)
```

The client's `tuneSpectrumDevice` attaches the bearer token and `POST`s the rounded
frequency; the panel surfaces any failure as an inline banner. Note the handler's
own doc steers *external* clients toward the rigctld TCP server (standard Hamlib)
against the same broker — the HTTP tune route is specifically the web panel's
click-to-tune surface, not a general control API. It's a fitting place to end the
first DSP canvas: the waterfall isn't just a picture, it's an input device, and
the click that moves the radio rides the identical REST + auth contract we started
the series on. For the finished view, see the
[Spectrum]({{ '/spectrum.html' | relative_url }}) operator docs.

## Where this goes next

[Part 8]({{ '/blog/deep-dives/operator-cockpit-08-constellation-eye-symbol/' | relative_url }})
stays in the DSP canvases but drops from the passband to the *symbol* domain: the
constellation, eye diagram, and symbol scope. Those stream decimated IQ and
recovered symbols rather than FFT bins — over `WS /api/v1/diag/iq` and
`/api/v1/diag/symbols` — and render the demodulator's inner life, where the P25
mode hint from this post's `SpectrumDevice` finally gets used to auto-pick a
receiver. Same streaming pattern, same canvas discipline, one layer deeper into the
signal.

## FAQ

**Where does the FFT actually run?**
On the daemon. `handleSpectrumStream` opens a per-request producer against the
shared iqtap broker and streams finished dBFS frames; the browser only renders. The
`SpectrumProvider` interface keeps `internal/api` free of any broker dependency.

**Why must `bins` be a power of two?**
It's the FFT size, and the transform requires it. The server clamps the value to
`64..16384` and rejects non-powers-of-two with a 400, so a client can't request a
degenerate or oversized transform.

**How does hover always match click-to-tune?**
Both call the same `xRatioToHz`, which maps a canvas x-position back through the
FFT-shifted bin layout. The frequency shown under the cursor is by construction the
frequency a click tunes to.

**Why draw the waterfall as one `ImageData`?**
Performance. Building the full row buffer and writing it with a single
`putImageData` is far cheaper than per-pixel canvas calls at 15 frames per second
across thousands of bins.

**Is retuning from the browser authenticated?**
Yes. The tune handler is wrapped in `s.gate`, so it obeys the same auth policy as
every mutation; the client sends the bearer token with the `POST`. External
automation should use the rigctld/Hamlib server instead of this panel-specific
route.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Client-Side Resampling]({{ '/blog/deep-dives/operator-cockpit-06-client-side-resampling/' | relative_url }})
· Next →
[Part 8: Constellation, Eye Diagram & Symbol Scope]({{ '/blog/deep-dives/operator-cockpit-08-constellation-eye-symbol/' | relative_url }})
</content>
