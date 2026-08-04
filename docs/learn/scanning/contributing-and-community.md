---
slug: contributing-and-community
title: The community & contributing data
description: The scanning hobby runs on shared knowledge — the forums, the databases like RadioReference, the live feeds, and the open-source decoders — and this is how to give back the systems, talkgroups, and fixes you discover.
keywords: scanning community, contributing data, RadioReference submission, talkgroup submission, open source radio, giving back, scanner forums, community database, feed volunteer, GopherTrunk contribute
level: beginner
status: full
prereq:
  - radioreference-database
faq:
  - q: How can I contribute to the scanning community?
    a: Several ways, all valuable. Submit new or corrected system and talkgroup information to databases like RadioReference. Run a live feed so others can hear your area. Share findings and help newcomers on forums. And if you use open-source software like GopherTrunk, contribute captures, bug reports, protocol details, or code. The hobby is a commons — most of what you rely on was contributed by someone before you.
  - q: What kind of data is most useful to submit?
    a: The things a database can't know without a local listener — new control-channel frequencies, talkgroup IDs and their real-world purpose, aliases, and corrections when a system rebands or changes. You did the identifying work by listening; writing it back is what keeps the shared map accurate for the next person.
  - q: Do I need to be an expert to give back?
    a: No. Correcting one wrong talkgroup, confirming a frequency still works, answering a beginner's question, or filing a clear bug report all help. Contribution scales from a one-line correction to running a feed or writing a decoder. Everyone starts by consuming the commons; giving back begins the moment you've learned something worth passing on.
---

# The community & contributing data

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Almost everything you've relied on in this module — the
[databases](/learn/scanning/radioreference-database/), the
[feeds](/learn/scanning/audio-feeds-and-streaming/), the open-source decoders — was
**contributed by other listeners**. The hobby is a **commons**, and it stays accurate
only because people write back what they learn. You give back by **submitting system
and talkgroup data**, **running a feed**, **helping newcomers on forums**, and
**contributing to open-source projects** like GopherTrunk. Contribution scales from a
one-line correction to a whole decoder — and it starts the moment you've learned
something worth passing on.
</div>

Scanning looks like a solitary hobby — you and a receiver — but it runs almost
entirely on shared knowledge. The database that told you what's on the air, the feed
you listened to before you owned a scanner, the software that decodes a protocol you'd
never reverse-engineer alone: every one of those is the accumulated work of a
community. This lesson is about the other direction — how to give back, and why it
matters more than it looks.

## The hobby is a commons

Think about what got you this far. You looked up systems in
[RadioReference](/learn/scanning/radioreference-database/), which is filled in by
volunteers. You may have heard scanner audio on a
[feed](/learn/scanning/audio-feeds-and-streaming/) run by someone with a rooftop
antenna. You're decoding with software written and maintained in the open. None of
that is a product you bought — it's a **commons**, maintained by people who
contributed a little more than they took.

A commons only stays healthy if it's replenished. Databases go stale when systems
reband and nobody updates them; feeds go dark when the last volunteer in an area quits;
open-source decoders stagnate without new captures and fixes. The hobby's continued
existence isn't automatic — it's the sum of small contributions, and once you've
benefited from it, you're in a position to add to it.

## Contribute what only a listener knows

The most valuable thing you can give is exactly what you produced by
[identifying signals](/learn/scanning/identifying-unknown-signals/) and
[building your frequency records](/learn/scanning/frequency-records/): local knowledge
a remote database can't have. Specifically:

- **New systems and control channels** — a county system that isn't documented yet,
  or a [control channel you found by hunting](/learn/scanning/searching-and-discovery/).
- **Talkgroup IDs and their real-world purpose** — which talkgroup is dispatch, which
  is a tac channel, and the aliases that make a log readable.
- **Corrections** — a frequency that moved, a system that rebanded, an alias that was
  wrong. Corrections are as valuable as new data, because stale data quietly misleads
  everyone downstream.

Submitting this back to RadioReference (or a similar database) is often a simple form.
You already did the hard part by listening; writing it down for the next person is the
small step that keeps the shared map true.

## Run a feed, help newcomers

Beyond data, two of the most useful contributions are time and patience. **Running a
[feed](/learn/scanning/audio-feeds-and-streaming/)** puts your area on the air for
everyone who can't listen locally — sometimes the only public monitoring an entire
region has. And **helping newcomers** on forums and community sites is how the hobby
reproduces itself: the questions you struggled with a month ago are the ones someone
is asking today, and a clear answer saves them the time you lost. Neither requires
expertise — just a willingness to share what you now know.

## Contribute to open source

If you scan with open software like **GopherTrunk**, you can give back to the tool
itself, and this is where the scanning hobby meets software development. Useful
contributions come in many sizes:

- **Captures** — a recorded IQ file of a system that decodes poorly is often the
  single most useful thing a developer can receive, because it lets them reproduce and
  fix the exact problem you hit. (This is a recurring theme in GopherTrunk's own
  history — the real fix usually starts with the reporter's capture.)
- **Bug reports** — clear, specific, reproducible reports of what failed, on what
  system, with what symptom.
- **Protocol details** — documentation, field observations, or corrections about how a
  system behaves on the air.
- **Code** — fixes and features, if you're inclined and able.

You don't need to write a line of Go to help; a good capture and a clear report move a
decoder forward as much as a patch. The [project home](/) is where to start, and it's
the natural bridge to the software side of the hobby that the final lesson points you
toward.

## Start small, give back anyway

None of this asks for expertise. Correct one talkgroup. Confirm a frequency still
works. Answer one beginner's question. File one clear bug report with a capture
attached. Every person who maintains the commons you've been using started exactly
there — consuming it, then noticing they'd learned something worth passing on. The
moment you have is the moment to begin.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a recorded IQ capture of a poorly-decoding system lets a developer reproduce and fix the exact problem, making it one of the most useful contributions." markdown="0">
  <p class="knowledge-check__q">Quick check: what's often the single most useful thing you can give an open-source decoder project?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A screenshot of the software running correctly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A recorded IQ capture of a system that decodes poorly, so the problem can be reproduced</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A donation of a faster computer to the developers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The scanning hobby is a **commons** — databases, feeds, and open-source decoders,
  all maintained by contributing listeners — and it stays healthy only if replenished.
- Contribute the **local knowledge only a listener has**: new systems and control
  channels, talkgroup IDs and purposes, and **corrections** when systems change.
- **Run a feed** and **help newcomers** — time and patience are as valuable as data.
- For open software like **GopherTrunk**, **IQ captures** and clear **bug reports**
  are often the most useful contributions, no code required.
- Contribution **scales from a one-line correction to a decoder** — start small, and
  start as soon as you've learned something worth sharing.

Next up: [where to go next](/learn/scanning/where-to-go-next/).
