---
title: "Build in the Open, Part 3: Brainstorming Features with Claude & Writing the README as Your Roadmap"
description: How to research a domain and brainstorm features with Claude Code, then write a README that doubles as a living roadmap — plus what CLAUDE.md is for.
keywords: brainstorm features, claude code, readme as roadmap, project planning, CLAUDE.md, competitive research, prioritize features, github, living roadmap
category: tutorials
tags: [github, claude-code, planning, documentation, getting-started]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 3
---

> **TL;DR:** Use Claude Code as a research and brainstorming partner — ask it to
> map the domain, list existing tools and their gaps, and propose features — then
> *you* prioritize the list ruthlessly. Capture the result in a README that
> doubles as a living roadmap: a clear "what is this?", a status snapshot of
> what's done versus what's still a gap, and links to deeper planning docs. Add a
> `CLAUDE.md` so Claude knows your project's standing rules every session.

**Key takeaways**

- Claude Code is at its best early: domain research, competitive scans, and
  feature brainstorms in an afternoon instead of a week.
- Brainstorming is divergent; *prioritizing* is convergent — that judgment call
  stays yours. Sort into Must/Should/Could/Won't.
- A great README answers "what is this, why should I care, how do I run it, and
  what's the state of it" — fast.
- A README-as-roadmap includes a status snapshot (done vs. gaps) so the
  project's reality is always visible.
- `CLAUDE.md` is your project's standing instructions to Claude — write it once,
  benefit every session.

*This is Part 3 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **How to brainstorm features with Claude Code** — prompts for domain research
  and competitive scans.
- **Turning a brainstorm into a prioritized list** you can actually build.
- **What a great README contains** — and the order to put it in.
- **The README as a living roadmap** — done vs. gaps, always current.
- **What `CLAUDE.md` is** and why every project should have one.
- **How GopherTrunk does it**, as a concrete example you can copy.

## Brainstorming features with Claude Code

In [Part 1]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
you validated a problem; in
[Part 2]({{ '/blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/' | relative_url }})
you chose a stack. Now you need to know *what to build*, in what order. This is
one of the highest-leverage places to use Claude Code, because the work is mostly
reading and synthesis — exactly what it's fast at.

Treat it like a research assistant, not an oracle. Good prompts are specific and
ask for structure:

- **Map the domain.** *"I'm building a self-hosted X for Y users. Explain the
  landscape: the main concepts, the hard problems, and the terms I should know."*
- **Scan the competition.** *"List the existing tools that solve this, with each
  one's strengths, weaknesses, and what users complain about. Where are the
  gaps?"*
- **Brainstorm features.** *"Given that problem and those competitors, propose a
  feature list. Group features as table-stakes, differentiators, and
  nice-to-haves."*
- **Pressure-test scope.** *"For a first useful version I can ship in a few
  weeks, which of these are truly essential and which can wait?"*

Two habits make this trustworthy. First, **ask for sources or reasoning**, not
just conclusions, so you can sanity-check. Second, remember Claude is generating
candidates — the *judgment* about what's actually worth building is yours.

<figure class="lab-figure">
<svg viewBox="0 0 640 190" width="640" height="190" role="img" aria-label="An AI-assisted planning loop: your problem statement feeds Claude Code, which proposes a domain map, a competitor scan, and a candidate feature list; you prioritize those candidates into Must, Should, Could and Won't; and a refined prompt loops back to Claude. A CLAUDE.md file supplies standing rules to every pass.">
  <rect x="16" y="76" width="130" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="81" y="94" text-anchor="middle" fill="currentColor" font-size="10">problem</text>
  <text x="81" y="108" text-anchor="middle" fill="currentColor" font-size="10">statement</text>
  <line x1="146" y1="98" x2="182" y2="98" stroke="currentColor"/><polygon points="182,94 192,98 182,102" fill="currentColor"/>
  <rect x="192" y="70" width="170" height="56" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="277" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">Claude proposes</text>
  <text x="277" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">domain map · competitor scan</text>
  <text x="277" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">candidate feature list</text>
  <line x1="362" y1="98" x2="398" y2="98" stroke="currentColor"/><polygon points="398,94 408,98 398,102" fill="currentColor"/>
  <rect x="408" y="70" width="170" height="56" rx="6" fill="none" stroke="currentColor"/>
  <text x="493" y="90" text-anchor="middle" fill="currentColor" font-size="10">you prioritize</text>
  <text x="493" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Must · Should</text>
  <text x="493" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Could · Won't</text>
  <line x1="493" y1="126" x2="493" y2="158" stroke="currentColor"/>
  <line x1="493" y1="158" x2="277" y2="158" stroke="currentColor"/>
  <line x1="277" y1="158" x2="277" y2="126" stroke="currentColor"/><polygon points="273,134 277,124 281,134" fill="currentColor"/>
  <text x="385" y="153" text-anchor="middle" fill="var(--fg-muted)" font-size="8">refine the prompt, repeat</text>
  <rect x="192" y="16" width="170" height="26" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="277" y="33" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CLAUDE.md · standing rules</text>
  <line x1="277" y1="42" x2="277" y2="68" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="273,60 277,70 281,60" fill="var(--fg-muted)"/>
</svg>
<figcaption>The AI-assisted planning loop: Claude proposes candidates, you prioritize with MoSCoW, and a refined prompt loops back — with <code>CLAUDE.md</code> supplying standing rules to every pass.</figcaption>
</figure>

## From brainstorm to a prioritized list

A brainstorm gives you breadth; a project needs a spine. Converge the long list
into something buildable using the same **MoSCoW** lens from Part 1:

- **Must** — without these the thing doesn't solve the problem at all. This is
  your first milestone, and only this.
- **Should** — important but not launch-blocking. The near-term roadmap.
- **Could** — nice if they're cheap; otherwise later.
- **Won't (yet)** — explicitly out of scope. Writing these down is as valuable as
  the Musts, because it stops scope creep and answers "why didn't you build X?"
  before anyone asks.

The output isn't a Gantt chart — it's a short, honest list of what's in, what's
next, and what's deliberately not happening. That list is the raw material for
your README.

## What a great README contains

The README is the front door. Most visitors decide in seconds whether to keep
going, so lead with the answer. A strong README, roughly in order:

1. **What is this?** One or two sentences a newcomer understands — the problem it
   solves and who it's for. No jargon dump.
2. **Why does it exist / what's different?** The one-line reason to pick this
   over alternatives.
3. **Quick start.** The shortest path from "found it" to "it's running" — copy-
   pasteable commands.
4. **Features.** What it actually does, in plain language.
5. **Status.** How finished it is — what works, what's coming. (More on this
   next.)
6. **Build, contribute, license, links.** The logistics, near the bottom where
   people who need them will look.

Write for the person who has *never seen your project*, not for yourself.

## The README as a living roadmap

Here's the move that keeps a README honest: include a **status snapshot** — a
short section listing what's done and what's still a gap. This turns the README
from a static brochure into a *living roadmap*. It does three jobs at once:

- **Sets expectations.** Visitors immediately know whether the feature they need
  works today or is aspirational.
- **Builds trust.** Naming your gaps openly reads as confidence, not weakness.
- **Doubles as your plan.** The gap list *is* your near-term backlog.

Keep the README's snapshot short and link out to deeper planning documents — a
`roadmap.md` for near-term plans, a `status.md` for detailed per-feature state,
and a `CHANGELOG.md` for what's already shipped. The README summarizes; those
docs carry the detail. Update the snapshot whenever reality changes, and it stays
trustworthy.

<figure class="lab-figure">
<svg viewBox="0 0 660 168" width="660" height="168" role="img" aria-label="The flow from a prioritized feature list into a README that doubles as a roadmap: the list becomes the README's what-is-this, quick start, features, and status snapshot; the status snapshot links out to roadmap.md, status.md, and CHANGELOG.md; and a build step synthesizes the website landing page from the same README.">
  <rect x="12" y="60" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="72" y="80" text-anchor="middle" fill="currentColor" font-size="10">prioritized</text>
  <text x="72" y="94" text-anchor="middle" fill="currentColor" font-size="10">feature list</text>
  <line x1="132" y1="84" x2="164" y2="84" stroke="currentColor"/><polygon points="164,80 174,84 164,88" fill="currentColor"/>
  <rect x="174" y="46" width="150" height="76" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="249" y="64" text-anchor="middle" fill="var(--accent)" font-size="11">README.md</text>
  <text x="249" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">what is this? · quick start</text>
  <text x="249" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">features</text>
  <text x="249" y="110" text-anchor="middle" fill="currentColor" font-size="9">status snapshot</text>
  <line x1="324" y1="84" x2="356" y2="84" stroke="currentColor"/><polygon points="356,80 366,84 356,88" fill="currentColor"/>
  <rect x="366" y="40" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/><text x="441" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">docs/roadmap.md</text>
  <rect x="366" y="71" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/><text x="441" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">docs/status.md</text>
  <rect x="366" y="102" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/><text x="441" y="119" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CHANGELOG.md</text>
  <line x1="249" y1="122" x2="249" y2="140" stroke="currentColor"/><polygon points="245,132 249,142 253,132" fill="currentColor"/>
  <rect x="150" y="140" width="198" height="22" rx="5" fill="none" stroke="currentColor"/><text x="249" y="155" text-anchor="middle" fill="currentColor" font-size="9">pages.yml → website landing page</text>
</svg>
<figcaption>The prioritized list becomes a README that doubles as a roadmap; its status snapshot links out to <code>roadmap.md</code>, <code>status.md</code>, and <code>CHANGELOG.md</code>, and the same README is synthesized into the website landing page.</figcaption>
</figure>

## What CLAUDE.md is

A `CLAUDE.md` file in your repo root is your project's **standing instructions to
Claude Code** — it's read automatically at the start of a session and gives
Claude the context a new contributor would need: how to build and test, coding
conventions, architectural rules, what *not* to touch, and where things live.

Think of it as onboarding docs that an AI actually reads every time. Instead of
re-explaining "we use Conventional Commits" or "always run the linter before
committing" in each session, you write it once in `CLAUDE.md`. Good entries are
concrete and imperative: *"Run `make test` before committing,"* *"Never edit
generated files in `proto/`,"* *"Match the existing table-driven test style."*
You can generate a first draft with the `/init` command and then refine it as the
project's conventions solidify.

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk)'s
[README](https://github.com/MattCheramie/GopherTrunk/blob/main/README.md) is a
textbook README-as-roadmap.

- **"What is this?" up top.** The README opens with a `## What is this?` section
  that explains, in plain language, that GopherTrunk is a software-defined-radio
  scanner that follows trunked-radio voice calls and decodes them to audio — what
  hardware it runs on, and its defining difference (no C dependencies, single
  ~10 MB static binary). A newcomer gets the whole pitch in one paragraph.

- **A real status snapshot.** Further down, a `## Status snapshot` section states
  plainly what runs end-to-end today, then a **"Remaining gaps:"** list naming
  exactly what *isn't* finished — for example, which digital-voice composer chains
  don't yet decode to PCM, and which subsystems still need on-air validation.
  That honesty is the living-roadmap principle in action.

- **Links out to deeper planning docs.** The snapshot ends by pointing to the
  detailed planning trio: long-form per-protocol state in
  [`docs/status.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/status.md),
  near-term plans in
  [`docs/roadmap.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/roadmap.md),
  and shipped work in
  [`CHANGELOG.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CHANGELOG.md).
  README summarizes; those carry the detail.

- **The README is literally the website.** This is the payoff. GopherTrunk's
  GitHub Pages site at gophertrunk.org doesn't maintain a separate landing page —
  the `pages.yml` workflow **synthesizes the landing page from `README.md` at
  build time**, rewriting asset and cross-link paths so they resolve on the site.
  Because the README *is* the homepage, keeping it accurate isn't busywork; it's
  the one source of truth for the front door of both the repo and the site.

Whatever you're building, the pattern transfers: research with Claude, prioritize
yourself, and write a README whose status snapshot keeps it honest as the project
grows.

## FAQ

**How do I use Claude Code to brainstorm features?**
Give it your problem statement and ask for structured output: a domain map, a
competitive scan of existing tools and their gaps, and a feature list grouped
into table-stakes, differentiators, and nice-to-haves. Then *you* prioritize —
Claude generates candidates, but the call on what's worth building is yours.

**When should I write the README — before or after the code?**
Start it early, even before much code exists. A README draft forces you to state
what the project is and what its first version includes, and that clarity guides
the build. Then keep it current — especially the status snapshot — as you ship.

**What is CLAUDE.md and do I need one?**
`CLAUDE.md` is a file in your repo that Claude Code reads automatically each
session — your project's standing instructions: how to build and test,
conventions, and what not to touch. You don't strictly need one, but it makes
every session more accurate and saves you re-explaining the same rules. Run
`/init` to generate a starting draft.

**What should a README status snapshot include?**
A short list of what works end-to-end today and an explicit list of remaining
gaps, plus links to deeper planning docs (roadmap, status, changelog). Keep it
brief in the README and update it when reality changes so it stays trustworthy.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Choosing Your Language, Platforms & Tech Stack]({{ '/blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/' | relative_url }})
· Next →
[Part 4: Git & GitHub Fundamentals (Using the Web Interface)]({{ '/blog/tutorials/build-in-the-open-04-git-github-fundamentals-web-interface/' | relative_url }})
