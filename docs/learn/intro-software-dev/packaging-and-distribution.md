---
slug: packaging-and-distribution
title: Packaging & distributing your software
description: Turn your program into something others can install — single binaries vs interpreters, cross-compilation, semantic versioning, release channels, signing and docs.
keywords: software distribution, packaging, single binary, cross-compilation, semantic versioning, GitHub Releases, package managers, code signing, auto-update
level: advanced
status: full
faq:
  - q: "What is the easiest way to distribute software as a solo developer?"
    a: "Ship **prebuilt single binaries** through **GitHub Releases**. You cross-compile one self-contained executable per operating system, attach them to a tagged release, and the user downloads the file for their platform and runs it — no runtime to install, no dependencies to resolve. It costs you almost nothing per release (CI can build them) and removes the most common reason non-technical users give up."
  - q: "What is semantic versioning?"
    a: "**Semantic versioning** (semver) is the MAJOR.MINOR.PATCH numbering scheme. You bump **PATCH** for backward-compatible bug fixes, **MINOR** for backward-compatible new features, and **MAJOR** when you make a breaking change. It tells users at a glance whether an upgrade is safe, which builds trust and reduces support questions about what changed."
  - q: "Why do checksums and code signing matter?"
    a: "Both let users trust that the file they downloaded is really yours and was not tampered with. A **checksum** lets them verify the download is intact and unmodified; **code signing** attaches a verified identity so the operating system does not warn them off (or block them) when they run it. For software a stranger installs, this trust is the difference between \"just works\" and scary security warnings."
---
# Packaging & distributing your software

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Binaries beat dependencies** — a single file the user just runs. **Cross-compile and version** — build for every OS, number releases with semver. **Channels and trust** — GitHub Releases, package managers, checksums and signing.
</div>

A working program on your machine helps exactly one person: you. Distribution is the
step that turns it into something a stranger can install and run — and for a solo
developer it is often the part that decides whether anyone ever uses what you built.
This is the distribution-strategy lesson of the path. We cover how to package your
program so others can install it, how to build it for every operating system, how to
version and publish releases, the channels you can ship through, and the trust signals
that stop the user's machine from treating your software as a threat. The closer you
get the user's experience to "download one thing and run it," the more of them will
actually use your work.

## Binaries vs interpreters: what does the user need?

The first question is what the user must have on their machine *before* your program
will run. There are two broad answers, and they create very different experiences:

- **Compiled single binary.** The build process produces one self-contained
  executable with everything baked in. The user downloads it and runs it — nothing
  else to install. Languages like **Go** and **Rust** make this the default, and it is
  the gentlest possible install story.
- **Interpreter plus dependencies.** The program is source code that needs a runtime
  (Python, Node, a JVM) plus a set of libraries installed at the right versions. This
  is fine for other developers, but for a non-technical user it means installing the
  runtime, resolving dependencies, and matching versions — every step a chance to fail
  and an email to you.

```text
Single binary:        download file → run                       (1 step)
Interpreter + deps:   install runtime → install deps → run      (3+ steps, version-sensitive)
```

You can soften the interpreter story — bundlers and containers package the runtime
with the code — but if you have the choice and your audience is non-technical, the
single binary wins decisively. This is exactly why the
[choosing your stack](/learn/intro-software-dev/choosing-your-stack/) lesson treated
easy distribution as a language-selection factor, not an afterthought.

## Cross-compilation: one machine, every platform

Your users are on Linux, macOS and Windows; you develop on one of them. **Cross-
compilation** lets you produce executables for *all* of those targets from your single
development machine, without owning three computers.

Go's tooling is the poster child — you set a target OS and architecture and `go build`
emits a binary for it:

```text
build for linux/amd64   → myapp
build for darwin/arm64  → myapp        (Apple Silicon Macs)
build for windows/amd64 → myapp.exe
```

In practice you let **CI do this on every release** (see
[build, test and CI/CD](/learn/intro-software-dev/build-and-ci-cd/)): one tagged
commit triggers a job that compiles every platform variant and attaches them to the
release automatically. You never build them by hand. The result is that a user on any
common system finds a file that just runs on their machine, and you did nothing manual
to produce it.

## Versioning and release artifacts

When you publish a build, two things make it usable and trustworthy: a clear version
number and well-formed artifacts.

**Semantic versioning** (semver) numbers releases as `MAJOR.MINOR.PATCH`:

| Bump | When | Example |
| --- | --- | --- |
| PATCH | Backward-compatible bug fix | 1.4.2 → 1.4.3 |
| MINOR | Backward-compatible new feature | 1.4.3 → 1.5.0 |
| MAJOR | Breaking change | 1.5.0 → 2.0.0 |

A user reading `1.5.0 → 1.5.1` knows it is a safe update; `1.x → 2.0` warns them to
read the notes. That single convention prevents a lot of confusion and support
questions.

A **release** then bundles, for that version:

- the **binaries** for each OS/architecture,
- **checksums** so users can verify the downloads are intact,
- **release notes** describing what changed,
- and the **source** (automatic on most platforms).

Tag the commit with the version, and the release is a permanent, referenceable
snapshot anyone can come back to.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in semver you bump MAJOR for a breaking, backward-incompatible change." markdown="0">
  <p class="knowledge-check__q">Quick check: in semantic versioning, when do you increase the MAJOR number?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Every time you publish any new release at all</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">When you make a breaking, backward-incompatible change</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only when you fix a small bug</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Channels, trust, and updates

How do people find and install your release? You have several **distribution
channels**, and you can use more than one:

- **GitHub Releases** — the simplest and most solo-friendly: attach your binaries to a
  tagged release and link to it. Free, permanent, and where developers already look.
- **Package managers** — meet users where they live: **Homebrew** on macOS, **apt** on
  Linux, **winget** on Windows. More setup, but installs and updates become one
  command for the user.
- **Containers** — publish a container image for users who run that way (often
  developers and servers).
- **App stores** — the most polished install for consumer apps, but with review
  processes, fees, and overhead that are often too heavy for a solo project.

Two trust signals matter for software a stranger runs:

- **Checksums** prove the download is intact and unmodified.
- **Code signing** attaches a verified identity, so macOS and Windows do not warn the
  user away from (or outright block) an "unknown developer." It costs money and setup,
  but for non-technical audiences it removes a scary barrier.

Finally, consider **auto-update** so users do not get stranded on an old, buggy
version — even just a startup check that says "a newer release is available" goes a
long way. And whatever channel you choose, **install instructions and documentation**
are part of distribution: a clear "download this, run that" beats the most elegant
binary nobody can figure out how to start.

## How a solo SDR project ships: the GopherTrunk pattern

Bring it together with a concrete case. **GopherTrunk** is radio software aimed partly
at scanner hobbyists — people who want to listen, not to compile code. A heavyweight
distribution story would lose most of them at step one. So the strategy is exactly the
solo-friendly one this lesson describes:

- Written in a language that **cross-compiles to a single binary**, so there is no
  runtime for the user to install.
- **CI builds prebuilt binaries for every operating system** on each tagged release —
  Linux, macOS and Windows — with no manual building.
- Those binaries are published as versioned **releases**, and the
  [downloads](/downloads.html) page simply points the user to the right file for their
  system.
- Supporting docs — including a [hardware](/hardware.html) page on which radios and
  dongles work — close the gap between "downloaded it" and "it's running."

The payoff: a non-technical hobbyist visits the downloads page, grabs one file for
their OS, plugs in a supported dongle, and is scanning — no toolchain, no dependency
hunt, no terminal required. That is what good distribution buys you. The hard
engineering inside is real, but the *install* is one step, and that is what turns a
personal project into something a community can actually use.

## Recap

- **Single binaries beat interpreters** — one self-contained file the user just runs
  removes the most common reason people give up.
- **Cross-compile in CI** — build for Linux, macOS and Windows from one machine, on
  every release, with no manual effort.
- **Version with semver** — MAJOR.MINOR.PATCH tells users at a glance whether an
  upgrade is safe.
- **Ship through the right channels** — GitHub Releases is the solo default; package
  managers and containers reach more users.
- **Build trust** — checksums and code signing stop tampering fears and scary OS
  warnings.
- **Distribution includes docs** — clear install instructions and auto-update keep
  users from getting stuck or stranded.

Next up: cutting your first real release and keeping the project healthy over time in
[shipping v1 and maintaining solo](/learn/intro-software-dev/shipping-and-maintaining/).
