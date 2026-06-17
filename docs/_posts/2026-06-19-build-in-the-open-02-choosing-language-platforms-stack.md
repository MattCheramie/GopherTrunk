---
title: "Build in the Open, Part 2: Choosing Your Language, Platforms & Tech Stack"
description: How to choose a programming language and tech stack — let your constraints (platforms, performance, distribution, team skills) decide, with GopherTrunk as a worked example.
keywords: choosing a programming language, tech stack, picking a language, single binary, cross-compile, boring technology, distribution, claude code, github
category: tutorials
tags: [github, claude-code, architecture, planning, getting-started]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 2
---

> **TL;DR:** Don't pick a language because it's trendy — pick the one whose
> strengths line up with your *constraints*. Write down where your software has
> to run, how fast it has to be, how you'll ship it, and what you already know.
> Those four answers usually point at one or two obvious choices. Favor boring,
> proven tech, and treat **distribution** — how a stranger installs your thing —
> as a first-class design decision, not an afterthought.

**Key takeaways**

- Constraints pick the language, not taste: target platforms, performance,
  distribution, ecosystem, and the skills you already have.
- "Boring" (mature, well-documented, widely used) beats "exciting" for anything
  you intend to maintain.
- Distribution is a feature. A single static binary beats a tool that needs a
  runtime and three system libraries to install.
- Pick your datastore and frontend the same way — by constraint — and pin your
  toolchain versions so builds are reproducible and patched.

*This is Part 2 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **Why constraints, not preferences, should pick your language** — the five
  questions that do most of the deciding.
- **Why "boring" technology usually wins** for projects you plan to keep.
- **Treating distribution as a feature** — single binary vs. runtime
  dependencies.
- **Picking a frontend and a datastore** by the same constraint-first logic.
- **How GopherTrunk chose its stack**, as a concrete example you can adapt.

## Let your constraints pick the language

In [Part 1]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
you wrote a problem statement and scoped a first version. That work pays off
now, because a clear problem turns "what language should I use?" from a religious
debate into a checklist. Ask five questions and write down the answers:

1. **Where does it have to run?** A web service, a CLI on a teammate's laptop, a
   phone, a Raspberry Pi, all of the above? Cross-platform reach narrows the
   field fast.
2. **How fast does it have to be?** Most software is I/O-bound and "fast enough"
   in almost anything. A few projects genuinely need tight latency or raw
   throughput — be honest about which you are.
3. **How will people install it?** This is the one beginners forget, and it
   matters more than almost anything else. We'll come back to it.
4. **What does the ecosystem give you for free?** A mature library for your hard
   problem (parsing, crypto, ML, DSP) can save months. Check what exists *before*
   you commit.
5. **What do you (or your team) already know?** Shipping in a language you know
   beats stalling in one you're learning — unless learning it is the point.

Notice that "which language is best" isn't on the list. There's no best
language, only a best *fit* for a specific set of constraints. The same problem
can have different right answers for a solo hobbyist and a ten-person team.

## Why boring technology usually wins

Once a few candidates survive the constraint checklist, prefer the **boring**
one. "Boring" here is a compliment: it means mature, widely deployed, thoroughly
documented, and supported by a deep bench of libraries and answered questions.

Boring tech wins because the costs of software show up *after* the fun part.
You'll spend far more time debugging, upgrading, and onboarding than you spent
writing the first version. A popular, stable language means that when you hit a
weird error at 11pm, someone has already hit it, asked about it, and posted the
answer. A bleeding-edge stack means *you're* the one writing that post.

Keep your "innovation budget" for the part of the project that's actually
novel — your domain logic — and spend boring, proven choices everywhere else.

## Distribution is a feature, not an afterthought

Here's the constraint that quietly decides the most: **how does a stranger get
your software running?** Every step between "found it" and "it works" loses
users. Compare two ends of the spectrum:

- **A single self-contained binary.** Download one file, run it. No interpreter
  to install, no package manager, no "works on my machine" version skew. This is
  the gold standard for CLI tools and self-hosted services.
- **A runtime plus dependencies.** "Install Python 3.11, then `pip install`
  these twelve packages, then make sure `libfoo-dev` is present, then…" Every
  one of those is a place where a new user gives up.

If your project is something people self-host or run locally, distribution
should shape your language choice directly. Languages that compile to a single
static binary — Go, Rust, sometimes C/C++ or Zig — make "download and run" real.
Interpreted languages can get close with bundlers, but it's extra work you're
choosing to take on. Decide this *up front*, because retrofitting easy
distribution onto a stack that fights it is painful.

## Picking a frontend and a datastore

The same constraint-first logic applies to the rest of the stack.

**Frontend.** If your tool needs a UI, ask what *kind*. A terminal UI (TUI) is
perfect for developer tools and keeps everything in one binary. A browser-based
console reaches non-technical users on any device with no install. A native
desktop or mobile app buys you OS integration at the cost of platform-specific
builds. For a web console, the boring-and-proven default today is a component
framework (React, Vue, Svelte) with a fast build tool and a typed language —
because type safety catches whole classes of bugs before they ship.

**Datastore.** Don't reach for a heavyweight database server because "real apps
have one." For a single-node tool or a self-hosted service, an embedded database
that lives in one file — SQLite being the canonical choice — means *zero* setup
for your users and keeps the single-binary dream alive. Scale to a client-server
database (Postgres, MySQL) only when concurrency, multi-host access, or data
size actually demands it.

## Pin your toolchain

One more habit that separates hobby projects from maintainable ones: **pin the
version of your compiler/runtime and dependencies.** "Latest" is a moving target
that makes builds non-reproducible and silently pulls in — or misses — security
patches. Pin to a specific version, document why, and bump it deliberately when
a fix or feature you need lands. Your future self and your CI both thank you.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) is a digital-trunking
SDR scanner, and every stack decision traces straight back to a constraint.

- **Language: pure Go, with `CGO_ENABLED=0`.** The problem statement from Part 1
  was essentially "there's no SDR scanner that installs as a single,
  dependency-free binary on any OS." Go delivers exactly that. By building with
  CGO disabled, the project has **no C dependencies at build or runtime** — no
  `librtlsdr`, `libhackrf`, `libusb`, `libasound2`, or `libmp3lame` — the very
  libraries that make competing tools painful to install. The result ships as a
  single **~10 MB static binary** that cross-compiles to Linux, macOS, and
  Windows (including Apple Silicon) with `go build`. Distribution drove the
  language, exactly as it should.

- **Datastore: SQLite via `modernc.org/sqlite`.** GopherTrunk logs calls, pager
  messages, vessels, and more — it needs a database. But the usual Go SQLite
  driver requires CGO, which would have blown up the no-C-dependencies
  constraint. The fix is `modernc.org/sqlite`, a **pure-Go** SQLite that needs no
  CGO. One embedded file, zero setup for users, single binary preserved. You can
  see it pinned in [`go.mod`](https://github.com/MattCheramie/GopherTrunk/blob/main/go.mod)
  alongside the rest of the dependencies.

- **Frontend: React 18 + Vite + TypeScript + Tailwind.** The web operator console
  needs to reach non-technical users in any browser, so it's a standard, boring,
  proven stack: React 18 for components, Vite for a fast build, TypeScript for
  type safety, Tailwind for styling. It compiles to a **prebuilt static bundle
  shipped alongside the daemon** — so there's no Node.js at runtime, and the
  single-binary distribution story still holds. (There's also a Bubbletea TUI
  cockpit for terminal users — same "one binary, no install" philosophy.)

- **Toolchain pinned to Go 1.25.11.** [`go.mod`](https://github.com/MattCheramie/GopherTrunk/blob/main/go.mod)
  contains a `toolchain go1.25.11` directive with a comment explaining exactly
  why: it closes a list of stdlib CVEs that `govulncheck` flagged against earlier
  1.25.x releases (an `html/template` XSS, `crypto/tls` issues, `crypto/x509`
  chain-building bugs, and more). CI's `setup-go` is pinned to the same version
  so the toolchain download doesn't repeat on every step. That's pinning done
  right — specific version, documented reason, security-driven.

Swap the domain and the moves are identical. A Python data tool might pick
Postgres because it genuinely needs concurrent writers; a game might pick a
native stack for GPU access; a docs site might pick a static-site generator for
zero-runtime hosting. The discipline is the same: name your constraints, then
let them choose.

## FAQ

**How do I choose a programming language for a new project?**
List your hard constraints first — target platforms, performance needs, how
you'll distribute it, what libraries you need, and what you already know — then
pick the language whose strengths match. For anything you'll maintain, prefer a
mature, widely-used language over a trendy one.

**When should I use a single static binary instead of a runtime?**
Whenever real people other than you will install your software — especially CLI
tools and self-hosted services. A single binary removes the most common reason
new users bounce: a broken or fiddly install. Choose a language that compiles to
one (Go, Rust) if distribution matters.

**What is a "boring" tech stack, and why is it good?**
Boring means mature, stable, widely adopted, and well-documented. It's good
because most of a project's cost is maintenance, debugging, and onboarding — all
of which are far cheaper when the answer to your problem is already on the
internet. Save novelty for your actual domain logic.

**Do I need a database server like Postgres for my project?**
Usually not at first. For a single-node tool or self-hosted service, an embedded
database like SQLite needs zero setup and keeps your install a single file. Move
to a client-server database only when concurrency, multi-host access, or scale
genuinely require it.

## Series navigation

**Part 2 of 14** · ←
[Part 1: Picking What to Build]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
· Next →
[Part 3: Brainstorming Features with Claude & Writing the README as Your Roadmap]({{ '/blog/tutorials/build-in-the-open-03-brainstorm-with-claude-readme-roadmap/' | relative_url }})
