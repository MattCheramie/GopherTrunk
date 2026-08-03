---
title: "Rebuilding GopherTrunk from Source: Update to the Latest Version with Git and GitHub"
description: A copy-paste guide to rebuilding GopherTrunk from source with the Git CLI and GitHub — clone the repo, check out the latest version or a specific branch, build the pure-Go binary, verify it, and keep an existing clone up to date.
keywords: rebuild gophertrunk from source, build gophertrunk, git clone, git checkout branch, go build, update gophertrunk, build from source linux, gophertrunk latest version, git cli, github, cross compile go
category: deep-dives
tags: [git, github, build-from-source, linux, go, getting-started]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
---

*GopherTrunk is a single, pure-Go static binary with no system-library
dependencies, which makes building it from source unusually painless. This guide
walks the whole loop — clone, pick a version, build, verify — and shows how to
grab a fix that hasn't shipped in a release yet by building a specific branch.*

## In this post

- **The fastest path** — three commands to go from nothing to a working binary.
- **How to pick a version** — `main`, a release tag, or an unmerged branch (this
  is how you test a fix before it lands in a release).
- **The full walkthrough** — prerequisites, cloning, building, verifying, and
  installing on your `PATH`.
- **Keeping a clone current** — the fetch / pull / rebuild loop.
- **Controls and optimizations** — version stamping, cross-compiling for another
  CPU, reproducible builds, and the Go build cache.

## Quick facts

If you already know your way around a terminal, this is the whole thing:

```bash
# 1. Get the code
git clone https://github.com/MattCheramie/GopherTrunk.git
cd GopherTrunk

# 2. Build the binary (output: ./bin/gophertrunk)
make build

# 3. Confirm it works
./bin/gophertrunk version
```

To build a **specific branch** instead of the default `main` — for example a
bug-fix that hasn't been released yet:

```bash
git fetch origin some-branch-name
git checkout some-branch-name
make build
```

> **Prerequisites:** Go 1.25+, `git`, and `make`. Nothing else — GopherTrunk is
> `CGO_ENABLED=0` pure Go, so there are no C libraries, DLLs, or
> `apt-get install` steps to chase.

## The short version

GopherTrunk lives in one Git repository on GitHub. Building it is the standard
Go loop, wrapped in a `Makefile`:

1. **Clone** the repository so you have a local copy with full history.
2. **Check out** the version you want — the tip of `main`, a tagged release like
   `v0.4.8`, or any branch.
3. **Build** with `make build`, which compiles `./cmd/gophertrunk` into
   `./bin/gophertrunk` with the version string stamped in.
4. **Verify** with `gophertrunk version` and `gophertrunk sdr list`.
5. Optionally **install** the binary onto your `PATH`.

Because the project is pure Go, the same source cross-compiles to Linux, macOS,
and Windows on any of those hosts without a cross-toolchain. The rest of this
post is the detailed version of those five steps, plus how to keep your clone
current and a few build knobs worth knowing.

<figure class="lab-figure">
<svg viewBox="0 0 660 132" width="660" height="132" role="img" aria-label="The five-step build pipeline: git clone the repository, git checkout the version you want (main, a tag, or a branch), make build to compile cmd/gophertrunk into bin/gophertrunk, verify with gophertrunk version and sdr list, and optionally install the binary onto your PATH.">
  <rect x="10" y="42" width="112" height="46" rx="6" fill="none" stroke="currentColor"/><text x="66" y="62" text-anchor="middle" fill="currentColor" font-size="10">clone</text><text x="66" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">git clone</text>
  <line x1="122" y1="65" x2="146" y2="65" stroke="currentColor"/><polygon points="146,61 156,65 146,69" fill="currentColor"/>
  <rect x="156" y="42" width="112" height="46" rx="6" fill="none" stroke="currentColor"/><text x="212" y="62" text-anchor="middle" fill="currentColor" font-size="10">checkout</text><text x="212" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">main · tag · branch</text>
  <line x1="268" y1="65" x2="292" y2="65" stroke="currentColor"/><polygon points="292,61 302,65 292,69" fill="currentColor"/>
  <rect x="302" y="42" width="112" height="46" rx="6" fill="none" stroke="var(--accent)"/><text x="358" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">make build</text><text x="358" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bin/gophertrunk</text>
  <line x1="414" y1="65" x2="438" y2="65" stroke="currentColor"/><polygon points="438,61 448,65 438,69" fill="currentColor"/>
  <rect x="448" y="42" width="112" height="46" rx="6" fill="none" stroke="currentColor"/><text x="504" y="62" text-anchor="middle" fill="currentColor" font-size="10">verify</text><text x="504" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">version · sdr list</text>
  <line x1="560" y1="65" x2="584" y2="65" stroke="currentColor"/><polygon points="584,61 594,65 584,69" fill="currentColor"/>
  <rect x="594" y="42" width="60" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/><text x="624" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9" transform="rotate(-90 624 66)">install PATH</text>
  <text x="358" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">compiles ./cmd/gophertrunk with version, commit &amp; build time stamped in</text>
</svg>
<figcaption>The whole loop, five steps: clone the repo, check out the version you want, <code>make build</code> the binary, verify it runs and sees your SDR, then optionally install it on your <code>PATH</code>.</figcaption>
</figure>

> New to Git itself? This post assumes you can run `git clone` and
> `git checkout`. If those aren't second nature yet, the free
> [**Git & GitHub learning path**]({{ '/learn/git/' | relative_url }}) starts from
> "what is version control?" and builds up to branches, remotes, and pull
> requests.

## Prerequisites

You need three tools on your machine:

| Tool | Why | Check |
| --- | --- | --- |
| **Go 1.25 or newer** | Compiles the daemon. | `go version` |
| **Git** | Clones the repo and switches versions. | `git --version` |
| **make** | Runs the build recipe (optional — you can call `go build` directly). | `make --version` |

That is the entire list. GopherTrunk uses `purego` for every system call it makes
(USB, audio) and a pure-Go SQLite for its call log, so it builds with
`CGO_ENABLED=0` and ships as one
[static binary]({{ '/reference/static-binary/' | relative_url }}). There is no
`libusb`, no `librtlsdr`, no compiler toolchain to install.

<figure class="lab-figure">
<svg viewBox="0 0 620 196" width="620" height="196" role="img" aria-label="The build toolchain dependency stack, bottom to top: the Go 1.25-plus toolchain building with CGO_ENABLED=0 sits at the base; make sits above it wrapping a plain go build; git supplies the source at cmd/gophertrunk; and the top layer is the single static binary bin/gophertrunk with no C libraries — no libusb, librtlsdr, or compiler toolchain.">
  <rect x="120" y="12" width="380" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="310" y="30" text-anchor="middle" fill="var(--accent)" font-size="11">bin/gophertrunk — one static binary</text>
  <text x="310" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no C libs · no libusb · no librtlsdr</text>
  <line x1="310" y1="46" x2="310" y2="58" stroke="currentColor"/><polygon points="306,48 310,58 314,48" fill="currentColor"/>
  <rect x="120" y="60" width="380" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="79" text-anchor="middle" fill="currentColor" font-size="10">make build  →  wraps  go build ./cmd/gophertrunk</text>
  <line x1="310" y1="90" x2="310" y2="102" stroke="currentColor"/><polygon points="306,92 310,102 314,92" fill="currentColor"/>
  <rect x="120" y="104" width="380" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="123" text-anchor="middle" fill="currentColor" font-size="10">Go 1.25+ toolchain  ·  CGO_ENABLED=0</text>
  <line x1="310" y1="134" x2="310" y2="146" stroke="currentColor"/><polygon points="306,136 310,146 314,136" fill="currentColor"/>
  <rect x="120" y="148" width="380" height="30" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="310" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="10">git  ·  source at ./cmd/gophertrunk</text>
  <text x="60" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9" transform="rotate(-90 60 97)">the three required tools</text>
</svg>
<figcaption>The toolchain is a short stack: git fetches the source, the Go 1.25+ toolchain compiles it with <code>CGO_ENABLED=0</code>, <code>make</code> wraps <code>go build</code>, and the top of the stack is a single static binary with no C libraries underneath.</figcaption>
</figure>

> **On Linux and want to actually receive with an SDR after building?** USB
> device permissions are a separate, one-time setup step. See
> [Installing on Linux]({{ '/install-linux/' | relative_url }}) for the udev rule
> and `modprobe` blacklist that let GopherTrunk claim your dongle without
> `sudo`.

If you don't have Go yet, grab it from [go.dev/dl](https://go.dev/dl/) (or your
package manager), and see [Installing Git]({{ '/learn/git/installing-git/' | relative_url }})
if you need Git.

## 1. Get the code

Clone the repository over HTTPS:

```bash
git clone https://github.com/MattCheramie/GopherTrunk.git
cd GopherTrunk
```

This downloads the full project and its history into a `GopherTrunk/` directory
and leaves you on the default branch, `main`. If you have SSH keys set up with
GitHub, you can use the SSH URL instead
(`git@github.com:MattCheramie/GopherTrunk.git`); see
[Authentication]({{ '/learn/git/authentication/' | relative_url }}) for the
difference.

## 2. Choose your version

A fresh clone leaves you on `main` — the latest development tip. Most of the time
that's exactly what you want. But there are three things you can check out, and
the difference matters:

**The latest development code (`main`).** Already where a fresh clone lands.
Newest features and fixes, but it moves between releases:

```bash
git checkout main
git pull origin main      # fast-forward to the newest commits
```

**A tagged release** — a frozen, named snapshot like `v0.4.8`. Pick this when you
want the exact code behind a published version:

```bash
git fetch --tags
git checkout v0.4.8
```

**A specific branch** — this is the important one. When a fix or feature is still
being worked on, it lives on its own branch and is **not** in `main` or any
release yet. Building that branch is how you test the change early:

```bash
git fetch origin the-branch-name
git checkout the-branch-name
```

> If a maintainer points you at a branch to try a fix — say
> `claude/some-fix-branch` — those last two commands are the whole story: fetch
> it, check it out, then build. When the fix later merges to `main` and ships in
> a release, you switch back with `git checkout main`.

Not sure what you're on? `git status` shows the current branch, and
`git log --oneline -5` shows the latest commits. The
[Branches]({{ '/learn/git/branches/' | relative_url }}) lesson covers this in
depth.

## 3. Build it

From the repository root:

```bash
make build
```

That compiles `./cmd/gophertrunk` and writes the binary to `./bin/gophertrunk`,
stamping in the version, commit, and build time so `gophertrunk version` can
report exactly what you built. Under the hood it's a plain `go build` — if you'd
rather not use `make`:

```bash
go build -o bin/gophertrunk ./cmd/gophertrunk
```

**Want the web console too?** `make build` produces the daemon and the terminal
UI, but the in-browser operator console is a separate bundle. To get a single
binary that also serves the web UI at `/`, use:

```bash
make dist
```

`make dist` builds the web front-ends first and embeds them, then builds the Go
binary. It needs Node.js/`npm`; plain `make build` does not.

## 4. Verify

Confirm the binary runs and reports the version you expect:

```bash
./bin/gophertrunk version
```

If you built a branch with a custom label, you'll see it here — handy for
confirming you're running the right code. Then check that GopherTrunk can see
your hardware:

```bash
./bin/gophertrunk sdr list
```

That enumerates attached SDR devices. (On Linux, if nothing shows up, revisit the
USB permissions in [Installing on Linux]({{ '/install-linux/' | relative_url }}).)
For a guided first run, the [Launcher]({{ '/launcher.html' | relative_url }}) and the
TUI walkthrough take it from here.

## 5. Install on your PATH

`./bin/gophertrunk` works fine in place. To run it as just `gophertrunk` from
anywhere, copy it onto your `PATH`:

```bash
sudo install -m 0755 bin/gophertrunk /usr/local/bin/gophertrunk
gophertrunk version
```

To update later, rebuild and re-run that `install` command — it overwrites the
old binary.

## Updating an existing clone

Already have the repository cloned from a previous build? You don't re-clone —
you pull the new commits and rebuild:

```bash
cd GopherTrunk
git checkout main          # make sure you're on the branch you want
git pull origin main       # fetch + merge the latest commits
make build                 # rebuild with the new code
./bin/gophertrunk version  # confirm the new version
```

If `git pull` complains about local changes, you either have edits in your
working tree or you're on a different branch than you think. `git status` tells
you which; [Status and diffs]({{ '/learn/git/status-and-diffs/' | relative_url }})
explains how to read it, and [Stashing]({{ '/learn/git/stashing/' | relative_url }})
shows how to shelve local edits before pulling.

## Controls and optimizations

A few build knobs worth knowing once the basics work:

**Stamp a custom version label.** Pass `ldflags` to mark a build — useful when
you're testing a branch and want `gophertrunk version` to say so:

```bash
go build -trimpath \
  -ldflags "-X github.com/MattCheramie/GopherTrunk/internal/version.Version=v0.4.8-mytest" \
  -o bin/gophertrunk ./cmd/gophertrunk
```

**`-trimpath` for reproducible builds.** It strips absolute filesystem paths from
the binary so the same source produces the same bytes regardless of where you
built it. The official release pipeline uses it; you can too.

**Cross-compile for another CPU or OS.** Because the build is `CGO_ENABLED=0`,
you can target any platform from your current one just by setting `GOOS` and
`GOARCH` — no cross-toolchain required:

```bash
# A Linux arm64 binary (e.g. for a Raspberry Pi) built on an amd64 laptop
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -o bin/gophertrunk-arm64 ./cmd/gophertrunk
```

The repo ships a `make cross-build` target that produces binaries for every
common OS/arch pair at once.

**The build cache.** Go caches compiled packages, so the *first* build of a fresh
clone is the slow one (a minute or two); subsequent rebuilds after a `git pull`
only recompile what changed and finish in seconds. There's nothing to configure —
just don't be alarmed by the first build's time.

**Clean rebuild.** If a build ever behaves strangely, `make clean` removes
`bin/` and `dist/`, and `go clean -cache` empties the compile cache for a
from-scratch rebuild.

## Troubleshooting

**`go: command not found`** — Go isn't installed or isn't on your `PATH`. Install
it from [go.dev/dl](https://go.dev/dl/) and reopen your terminal.

**`go.mod requires go >= 1.25`** — your Go is too old. Check with `go version`
and upgrade.

**`make: command not found`** — install `make` (`sudo apt install make`,
`sudo dnf install make`, or Xcode command-line tools on macOS), or skip it and
run the `go build` command directly.

**`git pull` says "Already up to date" but you expected changes** — you're
probably on the wrong branch, or the fix is on a branch you haven't fetched. Run
`git branch` to see where you are and `git fetch origin the-branch-name` to pull
a specific branch down.

**The binary builds but `sdr list` is empty** — that's a USB permissions issue,
not a build problem. See [Installing on Linux]({{ '/install-linux/' | relative_url }}).

## FAQ

**Do I need a compiler or system libraries to build GopherTrunk?**
No. GopherTrunk is pure Go (`CGO_ENABLED=0`) and links no C libraries. You need
Go, Git, and optionally `make` — nothing else.

**How do I get a fix that isn't in a release yet?**
Build the branch it's on. `git fetch origin <branch>`, `git checkout <branch>`,
then `make build`. When the fix merges to `main` and ships in a release, switch
back with `git checkout main`.

**What's the difference between `make build` and `make dist`?**
`make build` produces the daemon and terminal UI binary. `make dist` additionally
builds and embeds the in-browser web console, so the single binary serves the web
UI at `/`. `make dist` needs Node.js; `make build` does not.

**Can I build a Windows or macOS binary on Linux (or vice versa)?**
Yes. Set `GOOS` and `GOARCH` on a normal `go build` — no cross-toolchain needed,
because the build uses no cgo.

**How do I update a binary I already built?**
`cd` into your clone, `git pull origin main`, `make build`, and re-run the
`install` step if you copied it onto your `PATH`.

**I'm new to Git — where do I start?**
The free [Git & GitHub learning path]({{ '/learn/git/' | relative_url }}) takes
you from your first commit through branching, remotes, and pull requests.
