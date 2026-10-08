---
title: "From the Issue Tracker, Season 2, Part 12: CGO_ENABLED=0 Is Not Static — purego, libasound and the Termux Build"
description: "Why GopherTrunk's 'single static binary' claim was false on Linux — the ALSA player dlopens libasound through purego, which links libdl.so.2 and the glibc loader even with cgo off — and how issue #1230's Android request produced the nolibasound tag, scripts/check-static.sh, the gophertrunk power sweep logger, and three phones' worth of results."
category: solution-postmortem
keywords: cgo_enabled=0 static binary, purego libdl dynamic link, go static binary android termux, nolibasound build tag, readelf needed check, rtl_power alternative rtl_tcp, gophertrunk power sweep, termux rtl_tcp driver, bionic no libdl, gophertrunk from the issue tracker s2
tags: [from-the-issue-tracker-s2, build, linux, android, audio, go, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 12
---

*Part 12 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems on GopherTrunk bugs that fought back.
[Part 11]({{ '/blog/solution-postmortem/issue-tracker-s2-11-phantom-neighbours/' | relative_url }})
traced phantom TETRA neighbour sites to the seams in MAC fragment
reassembly. This part leaves the decoders for the build: a request for an
rtl_power-style sweep over rtl_tcp
([#1230](https://github.com/MattCheramie/GopherTrunk/issues/1230)) turned
into a question nobody had asked — whether the Linux binary that every
install page called "static" could start on a phone — and the answer was
in `readelf -d`.*

> **TL;DR:** `CGO_ENABLED=0` does not mean a static binary on Linux. The
> live-audio player (`internal/voice/player/alsa_linux.go`) dlopens
> `libasound.so.2` through `github.com/ebitengine/purego`, and purego links
> the glibc loader plus `libdl.so.2` with cgo off: a default build's
> dynamic section reads `NEEDED libdl.so.2`, `libpthread.so.0`,
> `libc.so.6` and requests `/lib64/ld-linux-x86-64.so.2`. Android's Bionic
> has neither the loader nor libdl, so the regular Linux build cannot start
> under Termux, and the release notes' "single static binary" claim was
> false. Fix: `-tags nolibasound` (`alsa_static_linux.go`) routes audio to
> the direct `/dev/snd` ioctl backend and drops purego; `scripts/check-static.sh`
> fails the build on any `(NEEDED)` entry; `make termux-build` cross-compiles
> `termux-arm64` and `termux-armv7`; CI's `termux-static` job runs all of
> it. The same PR added `gophertrunk power` (`internal/powersweep`), an
> rtl_power-compatible CSV sweep that works over rtl_tcp, pinned end to
> end by `TestPowerSweepOverRTLTCP`. On air: three phones report
> "statically linked"; one runs; two die at init with `SIGSYS` on
> `faccessat2` — a separate, still-open problem.

**Key takeaways**

- **"No cgo" is a claim about the compiler, not the binary.** A pure-Go
  dependency can still emit a dynamic section; only `readelf -d` on the
  artefact answers the question.
- **Measure the artefact in CI.** `check-static.sh` is a regression test
  for a build property; the next purego or cgo import that creeps into
  `cmd/gophertrunk`'s graph fails the `termux-static` job instead of a
  phone.
- **One fix, one new problem.** The linking defect is verified gone on
  three devices by `file`; what two of them hit next is a syscall the
  platform refuses, which no linker setting touches.
- **The feature request was the easy half.** `gophertrunk power` is a
  plan of abutting FFT hops with a settle discard; the rtl_tcp path is
  exercised against a fake server that records every tune.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Default Linux audio | dlopen `libasound.so.2` via purego (five `snd_pcm_*` calls) | `internal/voice/player/alsa_linux.go` (`//go:build linux && !nolibasound`) |
| Static audio wiring | tag routes every device to the direct-ioctl backend | `alsa_static_linux.go` (`//go:build linux && nolibasound`), `ioctl_linux.go` (`newIoctlALSABackend`) |
| Static gate | `readelf -d`; any `(NEEDED)` fails | `scripts/check-static.sh` |
| Build target | arm64 + armv7 (`GOARM=7`), `-tags nolibasound`, then the gate | `Makefile` (`termux-build`, `TERMUX_ARCHES`) |
| CI | `make termux-build` + vet/test of the player under the tag | `.github/workflows/ci.yml` (`termux-static`) |
| Release assets | `gophertrunk-<ver>-termux-{arm64,armv7}.tar.gz` | `.github/workflows/release.yml`, `linux-build.yml` |
| Sweep logger | rtl_power CSV over local USB or rtl_tcp | `internal/powersweep`, `cmd/gophertrunk/power.go` (`runPower`) |
| rtl_tcp pin | fake server, one tune per hop, loudest bin at the tone | `cmd/gophertrunk/power_test.go` (`TestPowerSweepOverRTLTCP`) |

## In this post

- **A question about rtl_power** — how #1230 became a question about linking.
- **What `readelf -d` said** — the dynamic section of a cgo-free build.
- **The tag, the gate and the target** — `nolibasound`, `check-static.sh`, `termux-build`.
- **`gophertrunk power`** — hops, bins, settle time and the rtl_tcp pin.
- **Three phones** — what "statically linked" proved and what it did not.

## A question about rtl_power

The issue opened on 30 Sep as a novice question, in the reporter's own
framing: could GopherTrunk act as a simple power scanner like `rtl_power`,
writing a CSV — but over `rtl_tcp`, because the RTL2832U was not on the
same machine. The first answer was honest: no such logger existed, run
`rtl_power` where the dongle is, or use `soapy_power`. The reply narrowed
the problem to something more interesting. The dongle was on an Android
phone, served by an rtl_tcp driver app, and the scripts were to run in
Termux on a Samsung S5 or S5 Neo — "it can be tested when there is a ARMV7
or ARMV8 compiled version".

GopherTrunk had no ARM Android build, but it had something that looked
like one: Linux ARM builds, cross-compiled with `CGO_ENABLED=0`, described
on every install page as a single static binary with "no glibc version
drama". Termux runs ordinary Linux ELF executables, and the reporter's
driver app already served the dongle on a local port. If the claim were
true the arm64 tarball would have been the answer. It was not true, and the
reason has nothing to do with SDR code.

The live-audio player's Linux backend talks to ALSA without cgo by loading
`libasound.so.2` at runtime through purego and binding five functions —
`snd_pcm_open`, `snd_pcm_set_params`, `snd_pcm_writei`, `snd_pcm_drain`,
`snd_pcm_close`, plus `snd_pcm_recover` — a design chosen so the daemon
needs no `libasound2-dev` at build time and degrades to a null backend
where the library is missing. That design is sound on a glibc desktop.
What it costs is a dynamic section.

## What `readelf -d` said

The measurement is one command on a default build, run for this post:

```text
$ CGO_ENABLED=0 go build -o gt-default ./cmd/gophertrunk
$ readelf -d gt-default | grep NEEDED
 0x0000000000000001 (NEEDED)             Shared library: [libdl.so.2]
 0x0000000000000001 (NEEDED)             Shared library: [libpthread.so.0]
 0x0000000000000001 (NEEDED)             Shared library: [libc.so.6]
$ readelf -l gt-default | grep interpreter
      [Requesting program interpreter: /lib64/ld-linux-x86-64.so.2]
```

purego's `Dlopen` has to reach the system loader, and on Linux it does so
by linking against glibc's `libdl` and declaring the glibc program
interpreter — even though not one line of C was compiled. On any glibc
distribution this is invisible: the loader and `libdl.so.2` are always
present, and the binary behaves exactly like a static one would. Android
uses Bionic, which has **neither** `/lib64/ld-linux*.so.2` nor
`libdl.so.2` at those paths, so the kernel cannot even start the image.
The README's "Zero CGO, single static binary" was, on Linux, a statement
about the toolchain that had been read as a statement about the output.

The import itself is small and careful, which is part of why nobody
looked. `loadALSA` dlopens the SONAME `libasound.so.2` once — the
ABI-stable symlink, never the dev package's `libasound.so` — with
`RTLD_NOW|RTLD_GLOBAL`, binds the six functions through
`purego.RegisterLibFunc` under a `recover` so a stripped-down library
degrades instead of panicking, and a failed dlopen falls back to the
ioctl backend so distroless images keep audio. None of that runs at link
time. The `NEEDED` entries exist whether or not a sound card is ever
opened, and whether or not `audio.enabled` is true: the mere presence of
the import in the build graph is what sets them.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Two build pipelines. Top: the default build with CGO_ENABLED=0 includes alsa_linux.go, which imports purego, and the ELF declares NEEDED libdl.so.2, libpthread.so.0 and libc.so.6 plus the glibc interpreter, so it does not start on Bionic. Bottom: the nolibasound build includes alsa_static_linux.go, routes audio to the /dev/snd ioctl backend, has no dynamic section, passes check-static.sh and starts under Termux.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">same source, same CGO_ENABLED=0 — two different ELF files</text>
  <text x="20" y="48" fill="var(--fg-muted)" font-size="9">default</text>
  <rect x="80" y="34" width="110" height="22" fill="none" stroke="currentColor"/>
  <text x="135" y="48" text-anchor="middle" fill="currentColor" font-size="8">go build</text>
  <line x1="190" y1="45" x2="216" y2="45" stroke="var(--fg-muted)"/>
  <rect x="216" y="34" width="130" height="22" fill="none" stroke="currentColor"/>
  <text x="281" y="48" text-anchor="middle" fill="currentColor" font-size="8">alsa_linux.go → purego</text>
  <line x1="346" y1="45" x2="372" y2="45" stroke="var(--fg-muted)"/>
  <rect x="372" y="28" width="170" height="34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="457" y="41" text-anchor="middle" fill="var(--fg-muted)" font-size="8">NEEDED libdl.so.2 · libpthread.so.0 · libc.so.6</text>
  <text x="457" y="54" text-anchor="middle" fill="var(--fg-muted)" font-size="8">interp /lib64/ld-linux-x86-64.so.2</text>
  <line x1="542" y1="45" x2="568" y2="45" stroke="var(--fg-muted)"/>
  <text x="614" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Bionic: no loader</text>
  <text x="20" y="118" fill="var(--accent)" font-size="9">nolibasound</text>
  <rect x="80" y="104" width="110" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="135" y="118" text-anchor="middle" fill="var(--accent)" font-size="8">-tags nolibasound</text>
  <line x1="190" y1="115" x2="216" y2="115" stroke="var(--fg-muted)"/>
  <rect x="216" y="104" width="130" height="22" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="281" y="118" text-anchor="middle" fill="var(--accent)" font-size="8">alsa_static_linux.go → ioctl</text>
  <line x1="346" y1="115" x2="372" y2="115" stroke="var(--fg-muted)"/>
  <rect x="372" y="98" width="170" height="34" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="457" y="111" text-anchor="middle" fill="var(--accent)" font-size="8">no dynamic section</text>
  <text x="457" y="124" text-anchor="middle" fill="var(--accent)" font-size="8">check-static.sh: statically linked</text>
  <line x1="542" y1="115" x2="568" y2="115" stroke="var(--fg-muted)"/>
  <text x="614" y="118" text-anchor="middle" fill="var(--accent)" font-size="8">Termux: starts</text>
  <line x1="40" y1="156" x2="640" y2="156" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="340" y="176" text-anchor="middle" fill="currentColor" font-size="8">CI termux-static: make termux-build (arm64 + armv7) → check-static.sh → go vet / go test -tags nolibasound</text>
  <text x="340" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="8">a new purego or cgo import reintroduces (NEEDED) and fails the job</text>
</svg>
<figcaption>The compiler flag is identical in both rows; the ELF's dynamic section is what differs, and only the artefact can be checked for it.</figcaption>
</figure>

## The tag, the gate and the target

The fix is three small pieces that land in PR #1241 beside the sweep
logger. First, a build tag. `alsa_linux.go` now carries
`//go:build linux && !nolibasound`, and a sibling file carries the inverse
and re-points the backend factory:

```go
// internal/voice/player/alsa_static_linux.go
//go:build linux && nolibasound

func init() {
    defaultBackendFactory = func(cfg Config) (Backend, error) {
        spec := ""
        if strings.HasPrefix(cfg.Device, "ioctl:") {
            spec = strings.TrimPrefix(cfg.Device, "ioctl:")
        }
        return newIoctlALSABackend(cfg, spec)
    }
}
```

The ioctl backend already existed — it opens `/dev/snd/pcmC{card}D{device}p`
and drives `SNDRV_PCM_IOCTL_*` through `syscall.Syscall` with no library
at all — as the fallback for distroless images that ship the kernel sound
subsystem without `libasound2`. Under the tag it becomes the only path.
Where `/dev/snd` is unreachable, as in Termux without root, the backend
fails to open and the Player runs without sound; recordings, the web
console and the CLI tools are unaffected.

Second, the gate. `scripts/check-static.sh` is twenty lines of shell that
walk `readelf -d` and fail on any `(NEEDED)` entry, printing the offending
lines and the interpreter so the next person sees *which* import did it:

```sh
# scripts/check-static.sh (shape)
if readelf -d "$bin" 2>/dev/null | grep -q '(NEEDED)'; then
    echo "check-static: $bin is dynamically linked:" >&2
    readelf -d "$bin" | grep '(NEEDED)\|interpreter' >&2 || true
    status=1
```

Third, the target. `make termux-build` cross-compiles
`CGO_ENABLED=0 GOOS=linux GOARCH=arm64` and `GOARCH=arm GOARM=7` with
`-tags nolibasound`, then runs the gate on each artefact; `release.yml`
and `linux-build.yml` do the same and ship the two tarballs. CI's
`termux-static` job runs the target plus `go vet` and `go test` of the
player package under the tag, on every push. Measured here, the tagged
build passes: `check-static: … is statically linked`. The armv7 build
exists because many phones from before about 2017 run a 32-bit userland
on a 64-bit CPU and report `armv8l`; the
[Termux guide]({{ '/termux.html' | relative_url }}) keys the download on
`uname -m`.

Two platform limits are documented rather than fixed: a static Go binary
resolves DNS from `/etc/resolv.conf`, which Android does not have, so the
rtl_tcp address must be an IP; and there is no live audio on Android at
all. That pure-Go-by-default stance is the same one
[RF Front End Part 4]({{ '/blog/deep-dives/rf-front-end-04-usb-without-libusb/' | relative_url }})
took for USB — the difference is that `purego` keeps the stance in source
while quietly giving it up in the linker.

## `gophertrunk power`

The request itself is `internal/powersweep`: an rtl_power-style logger
that steps a tunable IQ source across `-f lower:upper:bin_size` (with
`k`/`M`/`G` suffixes, as `rtl_power`'s `atofs`), averages an FFT power
spectrum at each step, and writes `rtl_power`'s CSV layout —
`date, time, Hz low, Hz high, Hz step, samples, dB, dB, …` — so
`heatmap.py` and spreadsheet imports read it unchanged. The sweep is laid
out in whole bins: a hop keeps `HopBins` bins around the FFT's centre
(the `1−crop` edges where the tuner's anti-alias filter rolls off are
dropped, default `-crop 0.25`), and consecutive hops abut exactly, so bin
`j` of hop `h` sits at `Low + (h·HopBins + j)·BinHz` with no gap or
overlap. `MaxFFTSize = 1 << 18` bounds how fine a bin may be (~9 Hz at
2.4 MS/s).

Over rtl_tcp there is one detail a local dongle never shows. The server
keeps streaming while the retune command crosses the network, so samples
already queued in its buffers and the socket still belong to the old
frequency. `runPower` therefore discards `powerSettleRTLTCP` = 250 ms
after each retune (50 ms locally, `-settle` overrides both); the
operator-visible symptom of too little is a strong signal appearing one
hop away from where it should be. `TestPowerSweepOverRTLTCP` pins the
whole path against a fake rtl_tcp server streaming a tone at
100.333 MHz: the real `rtltcp` driver, a `100.0M:100.6M:2k` plan, the
settling IQ source and the CSV writer, asserting that the server saw
exactly one tune per hop and that the CSV's loudest bin sits at the
carrier. The package's own tests cover the plan tiling
(`TestNewPlanTilesTheRange`), the tone recovery
(`TestSweepFindsToneAtItsFrequency`) and the column layout
(`TestWriteCSVMatchesRTLPowerLayout`). The full reference is the
[power sweep page]({{ '/power-sweep.html' | relative_url }}).

## Three phones

v1.2.3 shipped with both tarballs. The reporter tested on 5 Oct,
and the thread is the verification record. On every device `file`
reports `statically linked` — the linking defect that motivated the work
is gone on real hardware, which is more than the CI gate can say. What
happened next differs by phone:

- **Samsung S5 Neo** (`armv8l`, kernel 3.10.108): `./gophertrunk -v`
  prints `v1.2.3 (sha=d77e555, …)` and `-h` prints the full usage,
  `power` included.
- **Samsung S5** (`armv7l`, kernel 3.4.113) and **CMF Phone 1**
  (`aarch64`, kernel 6.1.78, Android 14, the arm64 build): both die before
  `main` with `SIGSYS: bad system call`. The trace is identical on both —
  `syscall.faccessat2` (number `0x1b7`) called from `os/exec.LookPath`
  inside `github.com/atotto/clipboard.init.0`.

That second failure is not a linking problem and no build tag touches it.
`go mod why` traces the package to
`internal/configtui → charmbracelet/bubbles/textinput → atotto/clipboard`,
whose package `init` calls `os/exec.LookPath` — a `$PATH` search — and the
search reaches a syscall the platform refuses. The thread stops there at
the time of writing; nothing in the repository addresses it yet, so the
honest status is **static linking verified on three devices, start-up
verified on one**. The `docs/termux.md` status note — checked for static
linking only, not yet run on a phone — predates those results and is now
too cautious about the first and silent about the second.

### How the build shaped the Go code

- **The tag selects a file, not a branch.** `alsa_linux.go` and
  `alsa_static_linux.go` are mutually exclusive by build constraint, so
  the purego import cannot survive into a tagged build by accident.
- **The gate checks the artefact, not the flags.** `check-static.sh`
  reads the ELF; a green `CGO_ENABLED=0` line in a log proves nothing.
- **Settle time is a named constant with a reason.** `powerSettleLocal`
  and `powerSettleRTLTCP` carry the queue-draining explanation in their
  comment, and `-settle` exposes it.
- **The rtl_tcp test records tunes.** The fake server's `tuned()` list is
  compared to the plan's hops, so a hop that silently re-used stale
  samples fails on count, not on a loudness heuristic.

## Where this goes next

From the linker back to the scanner. A conventional-scanner operator
reported that CTCSS channels opened on noise even at a squelch of −1 dBFS,
and that the gate behaved as "Tone only, not CSQ AND Tone"
([#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239)). The
squelch was fine; what it measured was the whole 2.4 MHz span.
[Part 13]({{ '/blog/solution-postmortem/issue-tracker-s2-13-whole-span-squelch/' | relative_url }})
builds the per-channel power meter and the per-channel gain that shipped
with it.

## FAQ

**Why is a Go binary built with CGO_ENABLED=0 dynamically linked?**
Because a pure-Go dependency can still declare a dynamic section at link time.
GopherTrunk's ALSA player loads `libasound.so.2` through
`github.com/ebitengine/purego`, whose `Dlopen` links glibc's `libdl.so.2`
and the `/lib64/ld-linux*` interpreter. `readelf -d` on the binary shows
`NEEDED` entries for `libdl.so.2`, `libpthread.so.0` and `libc.so.6`.

**What does the nolibasound build tag do?**
It excludes `internal/voice/player/alsa_linux.go` (the purego backend) and
includes `alsa_static_linux.go`, which routes every audio device to the
direct `/dev/snd` ioctl backend. The result has no dynamic section, which
`scripts/check-static.sh` verifies. It is how the `termux-arm64` and
`termux-armv7` release tarballs are built.

**Can GopherTrunk replace rtl_power over rtl_tcp?**
`gophertrunk power -rtltcp host:port -f 88M:108M:10k -i 30 -out fm.csv`
sweeps a remote dongle and writes rtl_power's CSV layout, with `-i`, `-e`
and `-1` as in rtl_power. It discards 250 ms after each retune over
rtl_tcp because the server keeps streaming old-frequency samples; raise
`-settle` if a signal lands one hop off.

**Does the Termux build run on Android?**
Partly, as of the #1230 thread. `file` reports every build statically
linked, and a Samsung S5 Neo runs `-v` and `-h`. A Samsung S5 and a CMF
Phone 1 crash at init with `SIGSYS` on `faccessat2` from
`atotto/clipboard`'s package `init` via `os/exec.LookPath` — a separate
problem no linker setting addresses, still open.

**Why does Termux need an rtl_tcp driver app at all?**
A phone does not hand its USB dongle to a Termux process. An rtl_tcp
driver app owns the USB device and serves it on a local port (by default
`127.0.0.1:1234`), and GopherTrunk connects to that port exactly as it
would to a remote `sdr.rtl_tcp` entry. Use the numeric address: a static
binary cannot resolve host names without `/etc/resolv.conf`.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Phantom Neighbours — Spliced Broadcasts and the Seams in TETRA MAC Reassembly]({{ '/blog/solution-postmortem/issue-tracker-s2-11-phantom-neighbours/' | relative_url }})
· Next →
[Part 13: The Whole-Span Squelch — A Scanner Gated by Every Carrier in 2.4 MHz]({{ '/blog/solution-postmortem/issue-tracker-s2-13-whole-span-squelch/' | relative_url }})
