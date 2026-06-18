---
title: "Build in the Open, Part 1: Picking What to Build — How Pros Decide"
description: How experienced developers choose what software to build — validating demand, scoping a first version, and avoiding the build-trap, with GopherTrunk as a worked example.
keywords: what to build, choosing a software project, MVP scope, validate an idea, side project ideas, open source project, claude code, github
category: tutorials
tags: [github, claude-code, product, planning, getting-started]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 1
---

> **TL;DR:** Don't start with code — start with a problem you understand well
> enough to be annoyed by. Pick something narrow enough to ship a first useful
> version in weeks, validate that at least a few other people share the pain,
> and write the idea down as a one-sentence problem statement before you open an
> editor. The best first project is one *you* will actually use.

**Key takeaways**

- Build to scratch a real itch — your own, ideally — not to chase a market you
  can't see.
- Validate demand cheaply (search, forums, existing tools) *before* committing.
- Scope ruthlessly to a "walking skeleton": the smallest end-to-end slice that
  does one real thing.
- Write a problem statement and success criteria first; let those drive every
  later decision.

*This is Part 1 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **How professionals decide what to build** — and why "a cool idea" is the
  wrong starting point.
- **Validating demand** without writing a line of code.
- **Scoping a first version** down to something you can actually finish.
- **Writing it down**: the problem statement and success criteria that anchor
  the whole project.
- **How GopherTrunk got picked**, as a concrete example you can copy.

## Start with a problem, not a product

The most common way side projects die is that they begin as a *solution*
looking for a problem. Someone learns a shiny framework, decides to "build an
app with it," and three weekends later abandons a half-built thing nobody —
including the author — needed.

Experienced builders invert that. They start from a **problem they personally
feel**: a task that's tedious, a tool that's missing, a workflow that's broken.
This is the classic "scratch your own itch" rule, and it works because it solves
the hardest problem in software up front — *motivation and judgment*. When you
are the user, you don't need a focus group to tell you whether a feature is
good. You already know.

A useful litmus test: **can you state the problem in one sentence, without
mentioning your solution?** "I want to build a React dashboard" is a solution.
"I can't see all my home-lab services' health in one place" is a problem. Lead
with the second kind.

## Validate that the problem is real (and shared)

Scratching your own itch gets you a user of one. Before you sink months in, do a
few hours of cheap validation to confirm other people feel the same pain — that
turns a throwaway script into a project worth maintaining.

You don't need surveys or a landing page. You need to read:

- **Search the obvious terms.** If nobody is asking about it anywhere, either
  you've found a gap or the need isn't real. Figure out which.
- **Find where the niche hangs out** — a subreddit, a Discord, a forum, a tag on
  Stack Overflow — and read what people complain about.
- **Inventory the existing tools.** If alternatives exist, that's *good*: it
  proves demand. Your job is then to be meaningfully different (faster, simpler,
  free, cross-platform, open source), not first.
- **Look for "I wish X could…" comments.** Unmet wishes next to an existing tool
  are the richest seam of project ideas there is.

This is also the perfect place to use Claude Code as a research partner: ask it
to summarize the landscape, list existing tools and their trade-offs, and
surface the recurring complaints. You'll cover in an afternoon what used to take
a week — we'll go deep on research-with-Claude in
[Part 3]({{ '/blog/tutorials/build-in-the-open-03-brainstorm-with-claude-readme-roadmap/' | relative_url }}).

## Scope to a walking skeleton

Once the problem is real, the failure mode shifts from "wrong idea" to "too big
to finish." The fix is to scope your first version to a **walking skeleton**:
the thinnest possible slice that runs end-to-end and does *one* genuinely useful
thing.

A good way to find that slice is **MoSCoW**: sort every feature you can imagine
into *Must / Should / Could / Won't*. Your first milestone is **only the
Musts** — and you should be ruthless about what counts. Everything else is a
roadmap, not a v1.

The walking skeleton matters because it forces every layer of the system to
exist, even if each is minimal. That flushes out the hard integration problems
early, when they're cheap to fix, instead of after you've polished a feature
nobody can reach yet.

## Write it down before you write code

Two short artifacts will save you from months of drift. Write them *before* the
first commit:

1. **A problem statement.** One or two sentences: who has the problem, what the
   problem is, and why existing options fall short.
2. **Success criteria.** How you'll know the first version worked — concrete and
   checkable, e.g. "decodes one real signal end-to-end on my hardware."

These become the seed of your README (Part 3) and the yardstick for every "should
I build this?" decision later. When a tempting feature shows up, you hold it
against the problem statement. If it doesn't serve that, it waits.

## How GopherTrunk got picked

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) is a digital-trunking
radio scanner — software that decodes and records voice calls from
software-defined radio (SDR) hardware. Here's how it maps onto everything above:

- **A real, personal itch.** Existing scanner software worked, but installing it
  meant wrestling with C libraries (`librtlsdr`, `libusb`) and platform-specific
  build pain. The problem statement was essentially: *"There's no SDR trunking
  scanner that installs as a single, dependency-free binary on any OS."*
- **Validated against existing tools.** The space already had mature
  competitors — proof the demand was real. GopherTrunk didn't need to be first;
  it needed to be **different**, and its difference was concrete: *pure Go,
  `CGO_ENABLED=0`, one ~10 MB static binary* that cross-compiles to Linux,
  macOS, and Windows with `go build`.
- **A walking skeleton, not the whole radio.** The project supports 11+
  protocols today (P25, DMR, NXDN, TETRA…), but the first useful slice was
  narrow: get **one** protocol's control channel to "light up" end-to-end —
  hardware → DSP → decode → event. Everything else was a roadmap item.
- **Written down as a living roadmap.** That scope lives in the README's "What
  is this?" and "Status snapshot" sections to this day, listing what's done and
  what's still a gap — exactly the problem-statement-plus-success-criteria idea,
  grown up.

You don't need to be building a radio for this to transfer. Swap "SDR scanner"
for "budgeting CLI," "team status dashboard," or "static-site generator" — the
moves are identical: real itch, cheap validation, ruthless scope, write it down.

## FAQ

**How do I know if my idea is too small?**
It probably isn't. Beginners almost always over-scope, not under-scope. If you
can describe the whole first version in a paragraph and build it in a few weeks,
that's a feature, not a bug — ship it and learn from real use.

**What if a tool that does this already exists?**
That's a positive signal: it means the problem is real and someone proved people
will use a solution. Compete on a clear, specific difference — simpler, faster,
free, open source, cross-platform — rather than trying to do everything the
incumbent does.

**Should I use Claude Code this early, before there's any code?**
Yes. The idea and validation stage is one of the highest-leverage places to use
it: market scans, competitor summaries, brainstorming feature lists, and
pressure-testing your problem statement. Code comes later.

**Do I need to pick the programming language now?**
Not yet — but soon. Once the problem and scope are clear, your constraints
(platforms, performance, distribution) will point at a language. That's
[Part 2]({{ '/blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/' | relative_url }}).

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Choosing Your Language, Platforms & Tech Stack]({{ '/blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/' | relative_url }})
