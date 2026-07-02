---
title: "RF Scope, Part 9: The Scene Cockpit — Live TUI & Web Console"
description: rfscope cockpit is a full-screen Bubbletea terminal dashboard and rfscope serve is a browser console — both render the same live Scene (header, hierarchy tree, channel I/O sparklines, top talkers, conversations, severity-colored expert info) over a capture or a live SDR, with a Crypto Lab hand-off built in.
category: tutorials
keywords: rfscope, cockpit, tui, bubbletea, web console, rfscope serve, live scene, sparkline, top talkers, conversations, expert info, crypto lab bridge, gophertrunk
tags: [rfscope, tui, web, getting-started, gophertrunk, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 9
---

*Part 9 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. You have
seen every analyzer produce data; now put them all on one screen, live.*

> **TL;DR:** `rfscope cockpit` (and `rfscope live -tui`) is a full-screen Bubbletea
> dashboard with panels for the scene header, protocol hierarchy, per-channel I/O
> sparklines, top talkers, conversations, and severity-colored expert info. Keys:
> `q`/`ctrl+c` quit, `space` freeze/resume, `e` export (`rfscope-scene-<unix>.json`),
> `?` help. Live mode re-analyzes on a `-refresh` interval (default 2s) over
> `-duration` seconds of IQ per pass. `rfscope serve` is the browser equivalent at
> `127.0.0.1:8098` — upload, analyze, view the same panels plus a Crypto Lab bridge
> card with a Download-frames button.

**Key takeaways**

- **One Scene, two front ends** — a terminal cockpit and a web console, both rendering
  the same six panels.
- **Live mode refreshes** every `-refresh` interval (2s default) over `-duration`
  seconds of fresh IQ; `space` freezes it so you can read.
- **`e` exports** the current Scene to timestamped JSON, on the spot.
- **The web console has a Crypto Lab card** — one click downloads the recovered `ks`
  frames file with the suggested next command.

## Cheat sheet

| Command / key | What it does |
|---|---|
| `rfscope cockpit -in cap.cfile` | Offline cockpit over a capture |
| `rfscope cockpit -serial <sdr> -freq Hz` | Live cockpit over an SDR |
| `rfscope live -tui …` | Same, via the live subcommand |
| `-refresh 2s` / `-duration 4` | Live re-analysis interval / IQ per pass |
| `q` / `ctrl+c` | Quit |
| `space` | Freeze / resume the live view |
| `e` | Export scene → `rfscope-scene-<unix>.json` |
| `?` | Toggle help |
| `rfscope serve -addr 127.0.0.1:8098 -open` | Browser console |

## In this post

- **The cockpit's six panels** and how they map to the analyzers.
- **Live vs offline** — the refresh loop and freeze.
- **Keybindings** — quit, freeze, export, help.
- **`rfscope serve`** — the web console and its flags.
- **The Crypto Lab bridge card.**

## Six panels, one Scene

`rfscope cockpit` is a self-contained Bubbletea program modeled on Signal Lab's TUI: a
worker goroutine runs the segmentation + analyzer pipeline and feeds finished Scenes
into the event loop, which redraws. The `View` stacks six panels top to bottom, each a
rendering of part of the Scene you already understand:

```go
// cmd/gophertrunk/rfscope_cockpit.go
b.WriteString(renderSceneHeader(m.scene) + "\n")
b.WriteString(renderHierarchyPanel(m.scene) + "\n")
b.WriteString(renderChannelsPanel(m.scene, panelWidth(m.width)) + "\n")
b.WriteString(renderTalkersPanel(m.scene) + "\n")
b.WriteString(renderConversationsPanel(m.scene) + "\n")
b.WriteString(renderExpertPanel(m.scene) + "\n")
```

- **Scene header** — center frequency, span, window, noise floor, and the counts:
  bursts, channels, emitters, anomalies. The one-line summary of the whole band.
- **Protocol hierarchy** — the Part 3 tree, indented by depth, each node with its burst
  count and time %.
- **Channels (I/O graph)** — one row per channel: frequency, dominant class, the
  `█`/`·` occupancy **sparkline** from Part 4, and duty %.
- **Top talkers** — the busiest emitters by airtime, each with a `hop×N` suffix when it
  is a frequency hopper.
- **Conversations** — each linked pair or hop sequence: kind, the emitter IDs joined
  with `↔`, and the correlation.
- **Expert info** — the Part 8 anomaly list, severity-colored: red for `alert`, yellow
  for `warn`, plain for `note`.

<figure class="lab-figure">
<svg viewBox="0 0 560 340" width="560" height="340" role="img" aria-label="Cockpit panel layout">
  <rect x="8" y="8" width="544" height="324" rx="6" fill="none" stroke="var(--border)"/>
  <rect x="20" y="20" width="520" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="30" y="39" fill="currentColor" font-size="11" font-family="monospace">center 453.400 MHz  span 2.400 MHz  |  128 bursts  5 ch  6 emitters  4 anomalies</text>
  <rect x="20" y="58" width="256" height="120" rx="4" fill="none" stroke="currentColor"/>
  <text x="30" y="76" fill="var(--accent)" font-size="11" font-family="monospace">Protocol hierarchy</text>
  <text x="30" y="96" fill="currentColor" font-size="10" font-family="monospace">c4fm            96b  41% time</text>
  <text x="40" y="112" fill="currentColor" font-size="10" font-family="monospace">12.5 kHz → p25p1</text>
  <text x="30" y="132" fill="currentColor" font-size="10" font-family="monospace">fsk             22b   4% time</text>
  <rect x="284" y="58" width="256" height="120" rx="4" fill="none" stroke="currentColor"/>
  <text x="294" y="76" fill="var(--accent)" font-size="11" font-family="monospace">Channels (I/O graph)</text>
  <text x="294" y="96" fill="currentColor" font-size="10" font-family="monospace">453.100 c4fm ████████████ 98%</text>
  <text x="294" y="112" fill="currentColor" font-size="10" font-family="monospace">453.325 c4fm ··██···██··· 18%</text>
  <text x="294" y="128" fill="currentColor" font-size="10" font-family="monospace">453.550 fsk  █···█···█··· 6%</text>
  <rect x="20" y="186" width="256" height="80" rx="4" fill="none" stroke="currentColor"/>
  <text x="30" y="204" fill="var(--accent)" font-size="11" font-family="monospace">Top talkers</text>
  <text x="30" y="222" fill="currentColor" font-size="10" font-family="monospace">#1 c4fm@453.100  12.4s</text>
  <text x="30" y="238" fill="currentColor" font-size="10" font-family="monospace">#5 digital hopper  2.1s hop×4</text>
  <rect x="284" y="186" width="256" height="80" rx="4" fill="none" stroke="currentColor"/>
  <text x="294" y="204" fill="var(--accent)" font-size="11" font-family="monospace">Conversations</text>
  <text x="294" y="222" fill="currentColor" font-size="10" font-family="monospace">co-active   #1↔#2 (0.71)</text>
  <text x="294" y="238" fill="currentColor" font-size="10" font-family="monospace">hop-sequence #5 (1.00)</text>
  <rect x="20" y="274" width="520" height="48" rx="4" fill="none" stroke="currentColor"/>
  <text x="30" y="292" fill="var(--accent)" font-size="11" font-family="monospace">Expert info</text>
  <text x="30" y="310" fill="currentColor" font-size="10" font-family="monospace">[warn] obfuscated 453.550 …   [note] hopper 453.550 …</text>
</svg>
<figcaption>The cockpit stacks the scene header, hierarchy, channel sparklines, top talkers, conversations, and severity-colored expert info into one live view.</figcaption>
</figure>

## Live vs offline

The cockpit runs in two modes. **Offline** (`-in <capture>`) analyzes the file once and
holds the result — a static, navigable Scene. **Live** (`-serial <sdr> -freq Hz`)
re-analyzes on a timer:

```go
// cmd/gophertrunk/rfscope_cockpit.go
duration := fs.Float64("duration", 4, "seconds of IQ per live refresh")
refresh  := fs.Duration("refresh", 2*time.Second, "live re-analysis interval")
```

Every `-refresh` interval (default **2 s**), a worker grabs the next `-duration` seconds
(default **4 s**) of live IQ, runs the full pipeline, and hands the new Scene back to be
drawn. So the cockpit is a rolling window on the band: hoppers blink across channels,
duty cycles rise and fall, and new anomalies appear as they happen. Ada leaves it
running on 453 MHz and watches Mercury's hopper row flicker to life each time it keys up.

## Keybindings

Four keys, all handled in `Update`:

```go
// cmd/gophertrunk/rfscope_cockpit.go
case "q", "ctrl+c": return m, tea.Quit
case " ": m.frozen = !m.frozen
case "?": m.showAll = !m.showAll
case "e": m.status = m.exportScene()
```

- **`q` / `ctrl+c`** — quit.
- **`space`** — freeze / resume. While frozen, the live loop keeps analyzing but stops
  swapping in new Scenes, so you can actually read a fast-moving band. The header shows
  `[FROZEN]`.
- **`e`** — export the current Scene to `rfscope-scene-<unix>.json` in the working
  directory (full indented JSON via the same `Write` path as `-out-format json`). Handy
  for grabbing a snapshot the moment something interesting appears.
- **`?`** — toggle the help overlay.

## How the refresh loop stays responsive

The cockpit has to do something hard: run a heavy DSP pipeline *and* stay responsive to
keystrokes. It solves this the idiomatic Bubbletea way — analysis runs in a **worker
goroutine**, and finished Scenes arrive as messages the event loop consumes:

```go
// cmd/gophertrunk/rfscope_cockpit.go
go func() {
    scene, in, err := rfscope.Segment(context.Background(), src, cfg)
    if err == nil {
        scene.Source = label
        err = rfscope.Run(context.Background(), scene, in, analyzers...)
    }
    select {
    case ch <- cockpitSceneMsg{scene: scene, err: err}:
    default:
    }
}()
```

The `Update` loop never blocks on analysis — it only reacts to the `cockpitSceneMsg`
that lands when a pass finishes, and to a `tea.Tick` that schedules the next pass at the
refresh interval. That is why `space` freezes instantly even while a full segmentation is
mid-flight: freeze just sets a flag that tells `Update` to ignore the next Scene, and the
keystroke is handled on the main loop, not behind the worker. It is also why an analysis
error shows up as a status line rather than a crash — the message carries the error and
the loop renders it. The render helpers themselves (`renderSceneHeader`,
`renderChannelsPanel`, and friends) are deliberately **pure functions** of a `*Scene`,
which is how they get unit-tested without a TTY: hand them a constructed Scene, assert on
the string.

For live mode there is one more subtlety worth knowing. `-duration` controls how much IQ
each pass analyzes and `-refresh` controls how often a pass starts. Set `-duration`
larger than `-refresh` and passes overlap in wall-clock time (more CPU, smoother rolling
window); set it smaller and there are gaps between analyzed windows. The defaults — 4 s of
IQ every 2 s — deliberately overlap, so the rolling view never has a blind spot between
refreshes.

## rfscope serve: the web console

Not everyone wants a terminal. `rfscope serve` runs a standalone browser console:

```text
gophertrunk rfscope serve -addr 127.0.0.1:8098 -open
```

Its flags:

| Flag | Default | Purpose |
|---|---|---|
| `-addr` | `127.0.0.1:8098` | Listen address (localhost by default) |
| `-open` | false | Open the console in the system browser once up |
| `-tmp-dir` | fresh temp dir | Where staged capture uploads land |
| `-max-upload` | 512 MiB | Max capture upload size in bytes (0 = default) |
| `-log-level` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `-log-format` | `text` | `text` \| `json` |

The workflow is **upload → analyze → view**: drop a capture in the browser, the server
segments and analyzes it, and the page renders the same six panels the cockpit does. It
binds to localhost by default, so it is a personal console, not a public service, unless
you deliberately change `-addr`. (Build the SPA first with `make rfscope-web-build`; the
default binary serves an API-only placeholder until then.)

## The Crypto Lab bridge card

The web console has one thing the cockpit does not: a **Crypto Lab bridge card**. When
the entropy analyzer recovered any unknown payloads, the card appears with a
**Download-frames** button that hands you the `ks` frames file — the same
`{label, iv, ct}` JSONL from Part 7 — alongside the suggested next command
(`cryptolab classify auto -in frames.jsonl`, and `ks reuse` when reuse was detected).
It is the one-click version of `-frames-out`: analyze a band in the browser, spot the
obfuscated emitter in the expert panel, and download its bytes straight into the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) workflow. For Mercury,
that button is the hand-off — one click and its frame is on its way to being broken.

## Where this goes next

The cockpit and console run the pipeline with sensible defaults. The final part pulls
the covers off: [Part
10]({{ '/blog/tutorials/rf-scope-10-advanced-tuning-pipelines/' | relative_url }}) covers
running a *subset* of analyzers, tuning segmentation for weird bands, the machine
output formats for scripting, and the reverse-engineering loop from `-frames-out` into
Crypto Lab — with a sidebar on how RF Scope's downconverter relates to GopherTrunk's DSP
paths.

## FAQ

**What's the difference between `cockpit` and `live -tui`?**
None functionally — `live -tui` just delegates to the cockpit with the same flags. Use
whichever reads better in your command.

**Does freeze stop the radio?**
No. In live mode the worker keeps analyzing on the refresh tick; freeze only stops the
display from swapping in new Scenes, so you can read the current one. Un-freeze and the
next pass shows through.

**Is `rfscope serve` safe to expose?**
It binds to `127.0.0.1:8098` by default — a personal, local console. Only change `-addr`
if you understand the exposure, and mind `-max-upload` for untrusted uploads.

**Where does `e` write the export?**
To `rfscope-scene-<unix-timestamp>.json` in the current working directory, as full
indented JSON — the same serialization as `-out-format json -out …`.

## Series navigation

**Part 9 of 10** · ←
[Part 8: Expert Info]({{ '/blog/tutorials/rf-scope-08-expert-info-anomalies/' | relative_url }})
· Next →
[Part 10: Advanced Tuning & Pipelines]({{ '/blog/tutorials/rf-scope-10-advanced-tuning-pipelines/' | relative_url }})
