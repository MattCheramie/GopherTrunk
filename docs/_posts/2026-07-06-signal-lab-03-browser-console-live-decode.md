---
title: "Signal Lab, Part 3: The Browser Console — siglab serve and Live Decode"
description: Signal Lab's browser console, gophertrunk siglab serve — a fully offline web app at 127.0.0.1:8099 with Captures, Synthesize, Identify, Wideband, and Compare tabs, upload-configure-run workflow, and a live SSE decode event stream.
category: tutorials
keywords: siglab serve, signal lab console, offline web console, sse event stream, iq upload, siglab tabs, max-upload, tmp-dir, live decode, browser sdr analysis
tags: [siglab, sdr, web-console, getting-started, iq, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 3
---

*Part 3 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. The terminal app is fast; the browser console is
where you actually *see* a signal.*

> **TL;DR:** `gophertrunk siglab serve` starts a fully offline web console at
> `http://127.0.0.1:8099/`. It bundles the entire RF-analysis UI into the binary
> — no Node.js at runtime, no CDN — and organizes work into five tabs: Captures,
> Synthesize, Identify, Wideband, and Compare. The core loop is upload →
> configure the engine → run with a live SSE event stream → read the dashboard.
> Live *capture* from an SDR only exists when the console is served by a daemon
> with a radio; the standalone server is purely offline analysis.

**Key takeaways**

- **One command, a local web app.** `siglab serve` binds `127.0.0.1:8099` and
  serves the whole console from the binary — nothing fetched from the internet.
- **Five tabs** cover the workbench: Captures, Synthesize, Identify, Wideband,
  Compare.
- **The loop is upload → configure → run → dashboard**, with decode events
  streaming live over Server-Sent Events.
- **`-open`, `-addr`, `-tmp-dir`, `-max-upload`** shape where it listens and how
  big an upload it accepts.
- **Live SDR capture is daemon-only.** The standalone server analyzes files; it
  doesn't touch hardware.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `gophertrunk siglab serve` | Serve the console on `127.0.0.1:8099` |
| `-open` | Open it in the system browser once it's up |
| `-addr host:port` | Change the listen address (e.g. `0.0.0.0:9000`) |
| `-tmp-dir <dir>` | Directory for staged capture uploads |
| `-max-upload <bytes>` | Max upload size (`0` = default 512 MiB) |
| `-verbose-errors` | Full error chain + stack on failure |

## In this post

- **Starting the console** — the one command and its flags.
- **The five tabs** — what each surface is for.
- **The core loop** — upload, configure, run, read.
- **The live event stream** — what SSE gives you that a batch run doesn't.
- **Where live capture fits** — and why the standalone server stays offline.

## Starting the console

The whole thing is one subcommand:

```bash
gophertrunk siglab serve              # serve on 127.0.0.1:8099
gophertrunk siglab serve -open        # ...and open it in your browser
gophertrunk siglab serve -addr 0.0.0.0:9000
```

It prints its address to stderr and waits:

```text
siglab serve: listening on http://127.0.0.1:8099/
```

The server is deliberately minimal — it runs the API server with *only* the
Signal Lab subsystem wired in, no SDR, no daemon, no storage:

```go
// cmd/gophertrunk/siglab_serve.go
opts := api.ServerOptions{
    Addr: *addr,
    Siglab: api.SiglabOptions{
        Enabled:        true,
        TempDir:        *tmpDir,
        MaxUploadBytes: *maxUpload,
    },
}
```

Everything the browser needs — the SPA, the fonts, the analysis code — is
embedded in the binary, so the console works on an air-gapped machine. On
Windows the installer drops a **Signal Lab console** shortcut in the Start Menu
that runs exactly this command, so operators who never touch a shell still get
the full workbench.

The flags are few and practical. `-addr` moves the listener (bind `0.0.0.0` to
reach it from another machine — mind that it's unauthenticated, so keep it on
trusted networks). `-open` launches your browser once the server is up.
`-tmp-dir` chooses where uploaded captures are staged (default: a fresh temp
dir). `-max-upload` caps upload size in bytes, where `0` means the default of
**512 MiB** — raise it if you routinely analyze multi-gigabyte wideband files.

## The five tabs

The console organizes the workbench into five tabs, each a front-end onto part
of the engine you've already met or will meet soon:

| Tab | What it's for | Covered in |
|---|---|---|
| **Captures** | Upload a file, configure the engine, run a decode, read the dashboard | this post |
| **Synthesize** | Generate an ideal capture and inject impairments | Part 6 |
| **Identify** | Rank protocol candidates for an unknown capture | Part 8 |
| **Wideband** | Survey a wide capture for carriers and name them | Part 8 |
| **Compare** | Overlay spectra and diff metrics across captures | Parts 6 & 10 |

<figure class="lab-figure">
<svg viewBox="0 0 660 260" width="660" height="260" role="img" aria-label="Annotated layout of the Signal Lab browser console">
  <rect x="8" y="8" width="644" height="244" rx="8" fill="none" stroke="var(--border)"/>
  <text x="24" y="34" fill="var(--accent)" font-size="13" font-weight="bold">Signal Lab</text>
  <!-- tab bar -->
  <rect x="120" y="18" width="90" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="165" y="34" text-anchor="middle" fill="var(--accent)" font-size="11">Captures</text>
  <text x="245" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="11">Synthesize</text>
  <text x="330" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="11">Identify</text>
  <text x="410" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="11">Wideband</text>
  <text x="485" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="11">Compare</text>
  <line x1="8" y1="52" x2="652" y2="52" stroke="var(--border)"/>
  <!-- left config panel -->
  <rect x="24" y="68" width="180" height="168" rx="6" fill="none" stroke="currentColor"/>
  <text x="114" y="90" text-anchor="middle" fill="currentColor" font-size="11">configure engine</text>
  <text x="114" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="10">protocol · rate</text>
  <text x="114" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="10">format · tune</text>
  <text x="114" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="10">auto-tune · conj</text>
  <text x="114" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="10">IQ-correct · P25 knobs</text>
  <rect x="44" y="188" width="140" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="114" y="208" text-anchor="middle" fill="var(--accent)" font-size="12">▶ Run</text>
  <!-- right result panel -->
  <rect x="224" y="68" width="404" height="90" rx="6" fill="none" stroke="currentColor"/>
  <text x="426" y="90" text-anchor="middle" fill="currentColor" font-size="11">live event stream (SSE)</text>
  <text x="240" y="112" fill="var(--fg-muted)" font-size="10" font-family="monospace">0.42s  lock       @ -12500 Hz</text>
  <text x="240" y="128" fill="var(--fg-muted)" font-size="10" font-family="monospace">0.55s  grant      TG=101</text>
  <text x="240" y="144" fill="var(--fg-muted)" font-size="10" font-family="monospace">1.03s  grant      TG=204</text>
  <rect x="224" y="168" width="404" height="68" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="426" y="192" text-anchor="middle" fill="var(--accent)" font-size="11">dashboard</text>
  <text x="426" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="10">EVM 7.4%  ·  SNR≈19.7 dB  ·  constellation  ·  PSD</text>
</svg>
<figcaption>The Captures tab: a configuration panel on the left, a live event stream and results dashboard on the right, with the tab bar across the top.</figcaption>
</figure>

## The core loop

A typical session on the **Captures** tab is four steps:

1. **Upload** a capture (or hop to **Synthesize** and generate one). The file is
   staged into the temp directory `-tmp-dir` points at, subject to the
   `-max-upload` limit.
2. **Configure the engine.** Choose the protocol, set the sample rate, and toggle
   the same knobs the CLI exposes plus a few deeper ones: tune, auto-tune,
   conjugate (mirror I/Q), IQ-correct, and the P25 deep knobs. These map onto the
   same engine `Config` the TUI builds — the console is a richer front panel on
   identical machinery.
3. **Run.** The decode runs and streams events back live (next section).
4. **Read the dashboard**, then optionally **Compare** against another capture or
   a synthesized ideal, and **Export** the structured result. The console offers
   the engine's JSON / JSONL / YAML / CSV serializers — the same output the
   terminal app writes when you press `e`.

Because it's the same engine, the 2% baud-deviation rule from Part 2 applies
here verbatim: if the run reports a large baud deviation, fix the sample rate in
the config panel and re-run before trusting anything else.

## The live event stream

The console's signature feature is that a run isn't a black box that returns an
answer — it **narrates**. Decode events stream to the browser over Server-Sent
Events (SSE) as they happen: the moment of lock with its residual frequency, each
grant, stage transitions, errors. For a five-second capture that's over in a
blink, but for a long recording it's the difference between staring at a spinner
and watching a control channel wake up, grant a call, and settle.

The same event records feed the terminal app's live log; the console just renders
them in a scrolling panel next to the results. When something goes wrong
mid-decode, you see *where* in the capture it happened rather than only a final
"not locked" — which is often the clue that turns a mysterious failure into an
obvious one (a burst of errors right at a known interference spike, say).

## Where live capture fits

One boundary matters. The **standalone** `siglab serve` is purely offline: it
analyzes uploaded and synthesized files and never touches hardware. The
live-**capture** surface — record a fixed-length raw-IQ file from a tuner, then
analyze it immediately and download the `.cfile` — only exists when the console
is served by a **running daemon with an SDR**.

That's not an accident of packaging; it's the architecture. The daemon mounts the
identical `/api/v1/siglab/*` routes the standalone server does:

> The running daemon mounts the same `/api/v1/siglab/*` routes, so if you already
> have a daemon up you can reach the console (and the live-capture surface) there
> instead of starting a separate server.

So there are two ways to reach the same console. Start `siglab serve` for
offline, file-only analysis on any machine; or open the console on a daemon
that's attached to a radio to *also* get live capture. Either way the analysis
half — every tab in this post — behaves identically, because it's the same
bundled UI over the same engine.

## Where this goes next

Now that you can drive the console, the next four parts fill its visualization
tabs with meaning. [Part 4]({{ '/blog/tutorials/signal-lab-04-constellations-and-eye-diagrams/' | relative_url }})
opens the constellation and eye diagram — the plots that show *modulation
quality* directly — so the EVM and SNR numbers from Part 2 get a picture to go
with them. The [SigLab docs]({{ '/siglab.html' | relative_url }}) page lists every
serve flag if you need the reference.

## FAQ

**Does the console need internet or Node.js?**
No. The entire SPA and its analysis code are embedded in the binary. It runs on
an air-gapped machine; nothing is fetched from a CDN at runtime.

**Can I capture live from an SDR in the console?**
Only when the console is served by a running daemon that has an SDR attached. The
standalone `siglab serve` is offline analysis of uploaded and synthesized files
only.

**My capture upload was rejected as too large. What do I raise?**
`-max-upload`, in bytes. The default is 512 MiB (`0` selects that default). For
multi-gigabyte wideband files, pass a larger value.

**How do I reach the console from another machine?**
Bind a non-loopback address with `-addr 0.0.0.0:9000`. It's unauthenticated, so
only do this on a trusted network.

## Series navigation

**Part 3 of 10** · ←[Part 2]({{ '/blog/tutorials/signal-lab-02-reading-the-dashboard/' | relative_url }}) · Next →
[Part 4: Constellations & Eye Diagrams]({{ '/blog/tutorials/signal-lab-04-constellations-and-eye-diagrams/' | relative_url }})
