---
title: "The Operator's Cockpit, Part 8: Constellation, Eye Diagram & Symbol Scope"
description: How one WebSocket symbol stream off the daemon feeds three browser DSP scopes — a complex constellation, a folded C4FM eye diagram, and a rolling symbol oscilloscope — with server-side offset tuning to dodge the DC spike and a soft-vs-dibit frame shape shared across all three canvases.
category: deep-dives
keywords: p25 constellation web, eye diagram sdr browser, symbol scope oscilloscope, cqpsk c4fm demod quality, websocket symbol stream, canvas dsp scope, offset tuning dc spike, soft decision dibits, op25 datascope, gophertrunk operator cockpit
tags: [operator-cockpit, dsp, p25, canvas, websocket, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 8
---

*Part 8 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from both a browser and a terminal. Part 7 put
the wideband spectrum and waterfall on a canvas. This post zooms all the way in
— past the FFT, past the channel filter — to the recovered **symbols** a P25
receiver is deciding on, and shows how three different DSP scopes (constellation,
eye, symbol) all read the *same* diagnostic stream off the daemon.*

> **TL;DR:** The daemon exposes one WebSocket, `GET /api/v1/diag/symbols`, that
> spins up a parallel P25 receiver on a chosen device + offset and streams a
> `SymbolFrame` per batch — pre-slicer **soft** waveform, complex **sym_i/sym_q**
> decision points, oversampled **eye_soft** samples, and sliced **dibits**, plus
> receiver-loop metrics. Three React panels subscribe to that one stream and
> render three views: a **constellation** scatter, a **folded eye**, and a
> **rolling symbol oscilloscope**. An `offset` query param mixes an off-centre
> channel to baseband *server-side* so its symbols clear the SDR's DC spike. The
> client reconnects with jittered backoff; the whole thing is a diagnostic tap,
> not a decode path.

**Key takeaways**

- **One stream, three scopes.** Constellation, Eye, and Symbol Scope are three
  renderers over a single `SymbolFrame` WebSocket — the same one-contract-many-
  renderers pattern the whole series turns on.
- **The frame carries every representation at once.** Soft waveform, complex
  decision points, oversampled eye samples, and sliced dibits ride together, so
  each panel picks the fields it needs and the demod runs once.
- **Offset tuning happens on the server.** The panel sends `offset` in Hz; the
  daemon mixes that off-centre channel down to baseband before channelizing, so
  a locked voice/control channel appears clear of the centre DC spike.
- **C4FM and CQPSK are genuinely different pictures.** CQPSK has a real complex
  constellation; C4FM is constant-envelope FM with a 4-level *eye*, not a
  constellation — the UI models that difference instead of faking it.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Symbol WS endpoint | streams `SymbolFrame` per batch | `internal/api/symbols.go` (`handleSymbolStream`) |
| `SymbolProvider` | daemon seam to the parallel receiver | `internal/api/symbols.go` |
| Stream client | connect / reconnect / decode frames | `web/src/api/symbols.ts` (`openSymbolStream`) |
| Constellation | complex scatter (CQPSK) / IQ ring (C4FM) | `web/src/panels/Constellation.tsx` |
| Eye diagram | fold `eye_soft` over the symbol period | `web/src/panels/EyeDiagram.tsx` |
| Symbol scope | rolling soft-waveform / dibit oscilloscope | `web/src/panels/SymbolScope.tsx` |
| Dibit wire fix | force a JSON number array, not base64 | `internal/api/symbols.go` (`DibitArray`) |

## In this post

- **The one symbol stream** — the WebSocket, its frame shape, and the DC-spike trick.
- **The daemon seam** — how `SymbolProvider` keeps the API free of DSP.
- **Three scopes over one frame** — what each panel pulls from the same batch.
- **C4FM vs CQPSK** — why the UI draws two different pictures on purpose.
- **The wire gotcha** — why dibits need a custom JSON marshaller.

## The one symbol stream

Every DSP scope in this post is fed by exactly one endpoint. The daemon runs a
*parallel* P25 receiver — separate from the live decoder — on whatever device
and channel offset the operator points it at, and streams the recovered symbols
out as JSON text frames over a WebSocket:

```go
// internal/api/symbols.go (shape)
// SymbolProvider is the daemon-side abstraction the symbol endpoint
// consumes; the API package never imports the DSP package.
type SymbolProvider interface {
    // proto selects the receiver ("p25-c4fm" / "p25-cqpsk"); offsetHz
    // tunes an off-centre channel down to baseband before channelizing.
    OpenSymbolStream(ctx context.Context, serial, proto string, offsetHz int32) (<-chan SymbolFrame, func(), error)
}

// handleSymbolStream answers WS /api/v1/diag/symbols?device=…&proto=…&offset=…
func (s *Server) handleSymbolStream(w http.ResponseWriter, r *http.Request) {
    // …validate device + proto (p25-c4fm | p25-cqpsk), parse offset
    frames, cleanup, err := s.symbols.OpenSymbolStream(ctx, serial, proto, offset)
    defer cleanup() // MUST run on disconnect — it tears the parallel receiver down
    // …ping every 30s; marshal each SymbolFrame to a text message
}
```

The important structural fact is that `handleSymbolStream` knows nothing about
DSP. It validates the query, opens a stream through the `SymbolProvider`
interface, and pumps frames — pinging every 30 seconds so a proxy can't idle the
socket, and calling `cleanup` on disconnect so the parallel receiver doesn't leak.
The DSP lives behind the seam, in `internal/scanner/symbolscope`; the API package
stays a transport.

The single most useful field in the request is `offset`. An SDR's own DDC leaks a
residual carrier at 0 Hz — the **DC spike** — that lands on top of anything tuned
to the centre of the band. So the panels don't ask you to centre-tune the channel
you care about; they send its offset in Hz and the daemon mixes it down to
baseband *before* the receiver ever sees it:

```go
// internal/api/symbols.go (shape)
offset := int32(parseIntQuery(q, "offset", 0, -30_000_000, 30_000_000))
// …clamped to the device Nyquist client-side so the mix can't just alias back
```

That is why all three panels share the same little offset-and-Hold control: with
Hold off, the offset follows the newest active call on the selected SDR (or rests
on the control channel); pinning Hold freezes it. The symbols of a locked channel
appear clear of the centre spike because the *server* moved them there.

### How that principle shaped the frame shape

A `SymbolFrame` is deliberately over-complete: it carries every representation of
the batch at once, so a single demod pass feeds three very different pictures.

```go
// internal/api/symbols.go (shape)
type SymbolFrame struct {
    SymbolRateHz float64    `json:"symbol_rate_hz"`
    Soft         []float32  `json:"soft"`     // pre-slicer soft waveform (empty on CQPSK)
    SymI         []float32  `json:"sym_i"`    // complex decision points — the true
    SymQ         []float32  `json:"sym_q"`    //   constellation (empty on C4FM)
    EyeSoft      []float32  `json:"eye_soft"` // oversampled matched-filter output (C4FM eye)
    EyeSPS       int        `json:"eye_sps"`  // samples per symbol; fold over this
    Dibits       DibitArray `json:"dibits"`   // sliced decisions (0..3 C4FM, 0..1 bits)
    IsBits       bool       `json:"is_bits"`
    // …CarrierOffsetHz, AGCLevel, ClockMu, CMAError — receiver-loop metrics
}
```

`Soft` is the analog waveform *before* the slicer makes a decision; `SymI`/`SymQ`
are the complex decision points sampled once per symbol (the true constellation);
`EyeSoft` is oversampled matched-filter output at `EyeSPS` samples per symbol for
folding into an eye; `Dibits` are the final hard decisions. When a field applies
it's aligned index-for-index with the dibits; when it doesn't it's empty. One
frame, four ways of looking at the same symbols.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="One symbol WebSocket stream fans out to three browser scopes. The daemon's parallel P25 receiver emits SymbolFrames carrying soft waveform, complex sym_i and sym_q decision points, oversampled eye_soft samples, and sliced dibits. The Constellation panel reads sym_i and sym_q, the Eye Diagram panel reads eye_soft folded over eye_sps, and the Symbol Scope panel reads the soft waveform and dibits.">
  <rect x="8" y="80" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="83" y="102" text-anchor="middle" fill="var(--accent)" font-size="11">daemon receiver</text>
  <text x="83" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WS /diag/symbols</text>
  <line x1="158" y1="106" x2="196" y2="106" stroke="currentColor"/><polygon points="196,102 206,106 196,110" fill="currentColor"/>
  <rect x="206" y="82" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="266" y="102" text-anchor="middle" fill="currentColor" font-size="11">SymbolFrame</text>
  <text x="266" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">soft · sym_i/q · eye · dibits</text>
  <line x1="326" y1="96" x2="372" y2="58" stroke="currentColor"/><polygon points="372,58 382,58 375,66" fill="currentColor"/>
  <line x1="326" y1="106" x2="372" y2="106" stroke="currentColor"/><polygon points="372,102 382,106 372,110" fill="currentColor"/>
  <line x1="326" y1="116" x2="372" y2="154" stroke="currentColor"/><polygon points="372,150 382,154 371,158" fill="currentColor"/>
  <rect x="382" y="38" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="457" y="55" text-anchor="middle" fill="var(--accent)" font-size="10">Constellation</text>
  <text x="457" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sym_i / sym_q</text>
  <rect x="382" y="86" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="457" y="103" text-anchor="middle" fill="var(--accent)" font-size="10">Eye diagram</text>
  <text x="457" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">eye_soft ÷ eye_sps</text>
  <rect x="382" y="134" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="457" y="151" text-anchor="middle" fill="var(--accent)" font-size="10">Symbol scope</text>
  <text x="457" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="8">soft + dibits</text>
  <text x="300" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one demod pass, one frame, three canvases — each panel reads only the fields it needs</text>
</svg>
<figcaption>The symbol WebSocket is the single source; the three DSP scopes are renderers over the same over-complete frame.</figcaption>
</figure>

## The stream client: connect, decode, reconnect

On the browser side, `openSymbolStream` is the shared connection scaffold all
three panels use — the same reconnect discipline the IQ and audio streams use
elsewhere in the app:

```ts
// web/src/api/symbols.ts (shape)
export function openSymbolStream(cfg, opts): SymbolStream {
  const connect = () => {
    const ws = new WebSocket(symbolWebSocketURL(cfg, opts)); // …?device=&proto=&offset=
    ws.onmessage = (ev) => {
      const frame = JSON.parse(ev.data) as SymbolFrame;
      if (frame && Array.isArray(frame.dibits)) opts.onFrame(frame);
    };
    const onDown = () => {
      setStatus("closed");
      const wait = jittered(backoff);           // 500ms → 30s, halved + jittered
      backoff = Math.min(backoff * 2, MAX_BACKOFF);
      reconnectTimer = window.setTimeout(connect, wait);
    };
    ws.onerror = onDown; ws.onclose = onDown;
  };
  connect();
  return { close() { /* …stop reconnects, close ws */ } };
}
```

A panel picks its receiver with a small `Mode` control that resolves through
`demodModeToProto`: **Auto** follows the modulation the selected SDR is actually
decoding (`device.p25_modulation`), falling back to C4FM when unknown; an explicit
choice is used verbatim. Because the receiver, the ideal-cluster markers, and the
tuning label all key off that one resolved proto, "Auto" just works — the scope
matches whatever the daemon is decoding without the operator thinking about it.

## Three scopes over one frame

Each panel keeps a rolling buffer and repaints a `<canvas>` on every frame — the
identical technique Part 7 used for the waterfall. What differs is *which* fields
they read and *how* they draw them.

**Constellation** is a 2D scatter. On CQPSK it pushes the complex decision points
straight in; on C4FM — which has no complex domain — it falls back to either the
raw IQ ring or the four soft levels on the real axis:

```tsx
// web/src/panels/Constellation.tsx (shape) — onFrame
const pts: IQPoint[] = [];
if (f.sym_i && f.sym_i.length > 0) {
  const n = Math.min(f.sym_i.length, f.sym_q?.length ?? 0);
  for (let k = 0; k < n; k++) pts.push({ i: f.sym_i[k], q: f.sym_q[k] });
} else if (f.soft && f.soft.length > 0) {
  for (const s of f.soft) pts.push({ i: s, q: 0 }); // C4FM soft levels on the real axis
}
pushPoints(pts); // additive-blended, age-faded, DC-blocked, auto-scaled
```

The render is deliberately OP25-flavoured: hollow amber rings mark the ideal
cluster centres (±45° diagonals for CQPSK, ±1/±3 on the real axis for C4FM soft
levels), points are drawn additively so dense clusters bloom toward cyan-white
while the noise floor stays dim, and the newest samples are brightest. A clean
CQPSK signal reads as four tight clusters; a closing eye smears them into an X.

**Eye diagram** reads only `eye_soft` + `eye_sps` and folds the oversampled
matched-filter output over the symbol period, overlaying the windows:

```tsx
// web/src/panels/EyeDiagram.tsx (shape) — onFrame, C4FM only
if (!f.eye_soft || f.eye_soft.length === 0) return;
const buf = eyeRef.current.concat(f.eye_soft);
eyeRef.current = buf.slice(Math.max(0, buf.length - WINDOW_SAMPLES));
setEye({ samples: eyeRef.current, sps: f.eye_sps || 0 });
```

A healthy C4FM channel shows four open horizontal bands with clear gaps at the
decision instant; a closed centre means symbol-timing or SNR trouble. The eye is a
property of the FM-discriminated baseband, so this panel is C4FM-only and pins its
proto to `p25-c4fm` — CQPSK's quality view is the constellation instead.

**Symbol scope** is a rolling oscilloscope of the soft waveform with the sliced
dibits overlaid, keeping the two tracks aligned only when a frame carried a full
soft track:

```tsx
// web/src/panels/SymbolScope.tsx (shape) — onFrame
const sb = softRef.current.concat(f.soft ?? []);
const db = dibitRef.current.concat(f.dibits ?? []);
const aligned = sb.length === db.length; // soft present ⇒ keep both; else dibit rows only
softRef.current = aligned ? sb.slice(-WINDOW_SYMBOLS) : [];
dibitRef.current = db.slice(-WINDOW_SYMBOLS);
```

Three panels, three canvases, one WebSocket — and because the demod runs once on
the daemon, opening all three at different offsets costs three parallel receivers,
not three copies of the UI's imagination. The operator docs walk each one as a
finished surface: the
[Constellation]({{ '/constellation.html' | relative_url }}),
[Eye diagram]({{ '/eye-diagram.html' | relative_url }}), and
[Symbol scope]({{ '/symbol-scope.html' | relative_url }}) pages.

## The wire gotcha: dibits aren't base64

There's one Go-specific trap worth calling out, because it silently broke the
scopes once. Go's `encoding/json` renders a `[]byte` (which is `[]uint8`) as a
**base64 string**, not a number array — and the web client expects `number[]`,
so it would quietly drop every dibit. The fix is a named type with a custom
marshaller:

```go
// internal/api/symbols.go (shape)
// DibitArray forces the number-array form on the wire; a nil/empty slice
// becomes [] so the field is always an array, never null.
type DibitArray []uint8

func (d DibitArray) MarshalJSON() ([]byte, error) {
    if len(d) == 0 { return []byte("[]"), nil }
    // …append '[', strconv.AppendUint each value comma-separated, append ']'
}
```

It's a two-line lesson that generalizes: any time a Go `[]byte` needs to reach a
JavaScript typed array as numbers, you marshal it yourself. The client even
guards on it — `openSymbolStream` only forwards a frame when `frame.dibits` is a
real `Array`, so a malformed payload is dropped rather than crashing a scope.

## Where this goes next

[Part 9]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }})
climbs back out of the DSP and onto a map: plotting P25 sites and position-bearing
emitters (APRS, AIS, ADS-B, DSC) on a shared Leaflet canvas, wired to the daemon's
locations and sites endpoints, with live marker patching as fixes arrive. After
the abstract complex plane, it's the most literally *situational* panel in the
cockpit.

## FAQ

**Do the scopes decode anything, or just visualize?**
They visualize. `GET /api/v1/diag/symbols` runs a *parallel* diagnostic receiver;
it never touches the live decode path. Opening a scope can't disturb what the
daemon is recording — it just taps the same SDR through the broker pool.

**Why is the eye C4FM-only but the constellation works for both?**
Because they measure different things. The eye is the FM-discriminated baseband
folded over the symbol period, which only exists for constant-envelope C4FM.
CQPSK/LSM has a genuine complex constellation instead, so its quality view is the
scatter — the panels model that difference rather than forcing one view on both.

**What does the offset control actually do?**
It sends a frequency offset in Hz that the daemon mixes down to baseband
server-side before channelizing, so a channel that isn't at the SDR's centre
appears clear of the 0 Hz DC spike. With Hold off it follows the newest active
call; Hold pins it to a chosen (or control-channel) reference.

**Why three panels instead of one with tabs?**
Because they carry different persisted preferences and follow-logic, but they all
subscribe to the identical `SymbolFrame` stream — the "one contract, many
renderers" spine of this whole series. Splitting the render, not the data source,
keeps each panel simple.

**How does the client survive the daemon restarting?**
`openSymbolStream` reconnects with a jittered exponential backoff (500 ms up to
30 s), the same discipline the audio and IQ streams use. A restart shows as a
brief "connecting" pill, then the scope resumes when the socket comes back.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Live Spectrum & Waterfall in the Browser]({{ '/blog/deep-dives/operator-cockpit-07-spectrum-waterfall/' | relative_url }})
· Next →
[Part 9: The Map — Plotting Sites & Emitters]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }})
