---
title: "Running It For Real, Part 6: The Diagnostics Reporter"
description: How GopherTrunk prepends a host/SDR snapshot to every error surface — the boxed boot banner, the memoized Collector that enumerates dongles at most once, the cheap-versus-costly split in SysInfo, the full error-chain verbose trace, and the non-TTY-safe reporter.
category: deep-dives
keywords: diagnostic banner, error reporter, sysinfo snapshot, dongle enumeration, error chain unwrap, verbose trace, non-interactive terminal, sdr diagnostics, gophertrunk running it for real
tags: [running-it-for-real, diagnostics, observability, ops, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 6
---

*Part 6 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Part 5 kept the running timeline alive;
this post is about the *first* thing anyone sees when something goes wrong — the
error itself. On a laptop you'd just re-run with more logging. As a service, every
error is a message to a future you (or a support thread) who doesn't have the box
in front of them, so GopherTrunk attaches the host and radio context to the error
automatically — no "can you run this diagnostic command?" round-trip.*

> **TL;DR:** GopherTrunk prepends a **diagnostic banner** to every error surface —
> CLI, daemon log, API, web — giving whoever's triaging the macro context (build
> version, host OS/kernel, CPUs/RAM, and the SDR dongles in play) without asking
> the operator anything. The banner is rendered from a `SysInfo` snapshot split
> into **cheap, side-effect-free fields** (gathered every time) and **costly
> dongle enumeration** (walks libusb, memoized by a `Collector` so it runs at most
> once and never races a live pool's USB claim). A `Reporter` ties it together:
> banner, concise message, then — verbose flag set — the full `%w` error chain and
> goroutine stack, or on an interactive TTY an *offer* to show it, or on a
> non-TTY a printed hint so service managers and pipes never hang on a prompt.

**Key takeaways**

- **Errors carry their own context.** Every failure surface leads with version,
  host, and dongle state, so the first message already answers "what was the
  environment?" — the question that otherwise starts every support thread.
- **Enumeration is costly and it knows it.** Walking USB has side effects and can
  race a running daemon's device claim, so it's memoized behind a `Collector` and,
  in the daemon, *seeded from the live pool snapshot* instead of re-enumerating.
- **The verbose trace is the whole causal chain.** GopherTrunk wraps with
  `fmt.Errorf("…: %w", err)` everywhere, so `Chain` unwraps every layer
  (descending into `errors.Join`) into a numbered outermost-first trace plus the
  goroutine stack.
- **The reporter never hangs a service.** Interactive terminals get a `[y/N]`
  offer for the full trace; non-TTYs (systemd, Docker, pipes) get a one-line hint
  instead of a prompt nothing will ever answer.

## Cheat sheet

| Piece | Role | Where it lives |
|---|---|---|
| Boxed banner | version/host/dongles at the top of an error | `internal/diag/banner.go` (`FormatBanner`) |
| Plain banner | same fields, no box-drawing (JSON/logs) | `internal/diag/banner.go` (`FormatBannerPlain`) |
| Cheap snapshot | version, OS, CPUs, RAM — no USB | `internal/diag/sysinfo.go` (`CollectSysInfo`) |
| Kernel/RAM | uname + sysinfo on Linux | `internal/diag/sysinfo_linux.go` (`kernelInfo`) |
| Dongle enumerate | costly libusb walk, memoized | `internal/diag/dongles.go` (`Collector`) |
| Error chain | unwrap every `%w` layer + join | `internal/diag/trace.go` (`Chain`, `VerboseTrace`) |
| Reporter | banner + message + trace/offer/hint | `internal/diag/reporter.go` (`Reporter`) |

## In this post

- **Why errors carry context** — the banner and the round-trip it removes.
- **Cheap versus costly** — the `SysInfo` split and the memoized `Collector`.
- **The full causal chain** — unwrapping `%w` into a real trace.
- **Never hanging a service** — TTY-aware reporting.

## Why errors carry context

Think about the last time you triaged someone else's failure. The first three
questions are always the same: what version, what OS, what hardware? For an SDR
scanner add a fourth — what dongles, on what tuner? Every one of those is
answerable from the machine at the moment of failure, and every one of them
otherwise becomes a back-and-forth that adds a day to the fix. So GopherTrunk
answers all four *before* the error message, on every surface:

```
┌─ GopherTrunk diagnostics ──────────────────────────────
│ version : v1.2.3 (sha=a1b2c3d, built=2026-08-01)
│ os      : linux/amd64  (Linux 6.18.5)
│ host    : scanner-pi  cpus=4  go=go1.24  mem=612/3906 MB
│ dongles : 2 detected
│   - rtlsdr[0] serial=00000001 tuner=R820T2 product=Blog V4
│   - rtlsdr[1] serial=00000002 tuner=R820T2
└─────────────────────────────────────────────────────────
config: open /etc/gophertrunk/config.yaml: no such file or directory
```

The banner comes in two renderings from one field set. `FormatBanner` draws the
box for a terminal; `FormatBannerPlain` drops the box-drawing characters for
embedding in a JSON error envelope, a structured log, or a non-UTF-8 terminal.
Both honour `GOPHERTRUNK_QUIET_BANNER` so CI logs stay clean, and both *omit*
empty or zero fields rather than printing a broken line — a platform that can't
supply a kernel string or a RAM figure simply doesn't show that piece. The whole
`diag` package deliberately depends only on `internal/version` and `internal/sdr`
so it can be imported from the CLI, the API, and the daemon without an import
cycle. The banner is context that travels with the failure — that's the entire
point.

## Cheap versus costly

The single most important design decision in this package is which fields are free
and which are expensive, because getting it wrong means either a slow error path
or a dangerous one. `SysInfo` is split accordingly. The cheap half is gathered
unconditionally:

```go
// internal/diag/sysinfo.go (shape) — CollectSysInfo gathers only the free fields
func CollectSysInfo() SysInfo {
    host, _ := os.Hostname()
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    kernel, memTotal := kernelInfo() // uname + sysinfo on Linux
    return SysInfo{
        Version: version.String(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
        KernelOS: kernel, Hostname: host, NumCPU: runtime.NumCPU(),
        GoVersion: runtime.Version(), MemTotalMB: memTotal,
        MemInUseMB: ms.Alloc / (1024 * 1024),
        // Dongles deliberately NOT populated here — that's the costly part.
    }
}
```

Dongle enumeration is the costly, side-effecting half, and the code is blunt about
it: it walks libusb for every registered driver, so it must not run on every error
and must **never** run concurrently with a daemon that already holds the USB claim
— doing so could race the running pool's device grab. That's what the `Collector`
is for:

```go
// internal/diag/dongles.go (shape) — Collector memoizes the costly enumeration
type Collector struct {
    once      sync.Once
    info      SysInfo
    enumerate func() ([]DongleInfo, []string)
}

// CLI path: enumerate lazily the first time a banner actually renders.
func NewCollector() *Collector { return &Collector{enumerate: CollectDongles} }

// Daemon path: seed from the live pool snapshot so the error path NEVER
// re-enumerates USB and never races the pool's claim.
func NewCollectorWithDongles(dongles []DongleInfo, errs []string) *Collector { … }
```

<figure class="lab-figure">
<svg viewBox="0 0 660 170" width="660" height="170" role="img" aria-label="An error triggers a report; the reporter asks a Collector for SysInfo; the Collector memoizes with sync.Once; on the CLI path it enumerates USB lazily, on the daemon path it returns the pre-seeded pool snapshot without touching USB">
  <rect x="8" y="66" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="56" y="84" text-anchor="middle" fill="currentColor" font-size="11">error</text>
  <text x="56" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">any surface</text>
  <line x1="104" y1="88" x2="132" y2="88" stroke="currentColor"/><polygon points="132,84 142,88 132,92" fill="currentColor"/>
  <rect x="142" y="56" width="128" height="64" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="206" y="80" text-anchor="middle" fill="var(--accent)" font-size="11">Collector</text>
  <text x="206" y="95" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sync.Once</text>
  <text x="206" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">memoized SysInfo</text>
  <line x1="270" y1="76" x2="310" y2="60" stroke="currentColor"/><polygon points="310,56 320,58 312,65" fill="currentColor"/>
  <line x1="270" y1="100" x2="310" y2="116" stroke="currentColor"/><polygon points="310,111 320,118 312,122" fill="currentColor"/>
  <rect x="320" y="40" width="150" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="395" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="10">CLI: enumerate USB</text>
  <text x="395" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">lazy, once, costly</text>
  <rect x="320" y="100" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="395" y="118" text-anchor="middle" fill="var(--accent)" font-size="10">daemon: pool snapshot</text>
  <text x="395" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no USB, no race</text>
  <line x1="470" y1="88" x2="500" y2="88" stroke="currentColor"/><polygon points="500,84 510,88 500,92" fill="currentColor"/>
  <rect x="510" y="66" width="140" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="580" y="84" text-anchor="middle" fill="currentColor" font-size="11">banner</text>
  <text x="580" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">version/host/dongles</text>
  <text x="330" y="162" text-anchor="middle" fill="var(--fg-muted)" font-size="10">GOPHERTRUNK_DIAG_NO_ENUMERATE=1 skips the USB walk entirely for hostile stacks</text>
</svg>
<figcaption>The same banner, two enumeration paths. The CLI enumerates once, lazily, only when a banner actually renders; the daemon injects its pool snapshot so the error path never touches USB.</figcaption>
</figure>

The daemon builds its collector with `NewCollectorWithDongles`, feeding in the
`DongleInfo` it already has from `sdr.Pool.Snapshot()` — so a banner rendered
during a running daemon shows the real, claimed dongles without a second USB walk.
The CLI, which has no live pool, uses `NewCollector` and enumerates lazily *the
first time* a banner actually renders, memoized by `sync.Once` so a command that
fails twice pays for it once. And `GOPHERTRUNK_DIAG_NO_ENUMERATE=1` skips the walk
entirely for CI or a host with a slow/hostile USB stack. Cheap by default, costly
only when it's safe and worth it.

### How that principle shaped the Go code

- **`DongleInfo` is independent of `sdr.Info`.** The banner's per-device struct is
  its own type, so the daemon can populate it from a pool snapshot without
  re-enumerating and without the diag package importing the full SDR surface.
- **Zero fields self-omit.** `bannerLines` skips empty strings and zero counts, so
  every platform renders a clean banner — no `mem=0 MB` or blank `kernel:` lines.
- **The banner is a value transform.** `FormatBanner(si SysInfo) string` takes a
  snapshot and returns text — no I/O, no globals — so the same `SysInfo` renders
  boxed for a TTY and plain for a JSON envelope from one source of truth.
- **Version is stamped in.** `version.String()` carries the git SHA and build time
  into the top line, so an error report pins the exact binary.

## The full causal chain

A terse one-line error is fine when you're standing at the terminal and can
reproduce it. It's frustrating in a support thread, because the outermost message
("config: load failed") has thrown away the *cause*. GopherTrunk wraps with
`fmt.Errorf("…: %w", err)` throughout and never attaches stack frames to errors,
which means the whole causal chain is recoverable by unwrapping:

```go
// internal/diag/trace.go (shape)
// Chain unwraps the %w chain outermost-first and descends into errors.Join.
func Chain(err error) []string { … }

// VerboseTrace renders that chain plus the captured goroutine stack.
func VerboseTrace(err error, stack []byte) string {
    // error chain (outermost first):
    //   [0] config: load /etc/gophertrunk/config.yaml
    //   [1] open /etc/gophertrunk/config.yaml: no such file or directory
    // goroutine stack: …
}
```

`Chain` walks `errors.Unwrap` linearly and recurses into the `Unwrap() []error`
shape that `errors.Join` produces, so a multi-error join (say, several systems
each failing to load) shows every branch. `VerboseTrace` numbers the layers
outermost-first and appends the goroutine stack captured *at the report site*, so
the trace reflects where the error surfaced, not where the reporter happens to
live. The terse message stays the default; the full chain is one flag away.

## Never hanging a service

The last piece is the `Reporter`, and its cleverest behaviour is knowing when it's
*not* talking to a human. It prints the banner, the concise message, and then it
has to decide how to offer the verbose trace — and getting that wrong hangs a
service:

```go
// internal/diag/reporter.go (shape) — Report
switch {
case r.Verbose:
    fmt.Fprintln(out, VerboseTrace(err, stack)) // flag/env/config set: just print it
case r.isInteractive():
    r.prompt(out, err, stack)                   // TTY: offer "[y/N]"
default:
    fmt.Fprintln(out, verboseHint)              // non-TTY: print how to get it, never prompt
}
```

On an interactive terminal — both stdin and stdout are character devices — it
offers `Show full diagnostic trace? [y/N]:` and defaults to No. But under systemd,
in a Docker container, or at the end of a pipe, there's no one to answer that
prompt, and blocking on it would wedge the service. So `isInteractive` checks that
*both* streams are TTYs, and when they're not, the reporter prints a one-line hint
instead — *"set diagnostics.verbose_errors: true, GOPHERTRUNK_VERBOSE_ERRORS=1, or
pass -verbose-errors for the full trace"* — and moves on. It's a small thing that
is exactly the difference between a CLI that's friendly to a human and one that's
safe for an init system. The same `IsInteractive` check is shared with the
launcher so both agree on what "interactive" means.

## Where this goes next

The banner tells you what dongles the daemon *sees*. The next question is whether
each one is actually usable — bound to the right kernel driver, on a USB path that
can sustain the sample rate, at a gain that isn't deaf. [Part 7]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})
is `sdr doctor` and the config preflight: the read-only checks that catch a bad
dongle, a wrong driver, or an unwritable directory *before* the daemon commits to
a run and it costs you a missed call. It cross-links the
[RF Front End]({{ '/blog/series/rf-front-end/' | relative_url }}) series, where the
radio-layer diagnostics live in depth.

## FAQ

**Where does the banner show up?**
Every error surface — the CLI, the daemon log, the HTTP/gRPC API error envelopes,
and the web UI. It's the same `SysInfo`, rendered boxed for terminals and plain for
JSON/logs, so the environment context travels with the failure everywhere.

**Won't enumerating dongles on an error interfere with a running daemon?**
That's exactly why the daemon seeds its `Collector` from the live pool snapshot
via `NewCollectorWithDongles` — the error path never re-walks USB and never races
the pool's device claim. Only the CLI, which has no live pool, enumerates, and it's
memoized to run at most once.

**How do I get the full error, not just the one-liner?**
Set `diagnostics.verbose_errors: true`, export `GOPHERTRUNK_VERBOSE_ERRORS=1`, or
pass `-verbose-errors`. On an interactive terminal you can also just answer `y` to
the offer. The output is the full `%w` chain plus the goroutine stack.

**Why doesn't it prompt me under systemd?**
Because nothing would answer, and blocking on an unanswerable prompt would hang the
service. The reporter detects a non-TTY (neither stdin nor stdout is a character
device) and prints a hint instead of prompting.

**Can I silence the banner in CI?**
Yes — `GOPHERTRUNK_QUIET_BANNER` suppresses both banner renderings, and
`GOPHERTRUNK_DIAG_NO_ENUMERATE=1` additionally skips the USB walk for hosts with a
slow or hostile USB stack.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Structured Logs — Event, Message & Power]({{ '/blog/deep-dives/running-it-for-real-05-structured-logs/' | relative_url }})
· Next →
[Part 7: SDR Doctor & Preflight]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})
