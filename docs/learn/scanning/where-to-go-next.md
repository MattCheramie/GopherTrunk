---
slug: where-to-go-next
title: Where to go next
description: Where the scanning hobby leads once you can follow a system — decoding data modes, digging into the protocols underneath, or writing your own decoder — and the GopherTrunk.org modules that take you each of those directions.
keywords: next steps scanning, learn radio protocols, decoding data modes, write a decoder, DSP for radio, digital trunking module, RF software development, scanning to development, radio deep dive, GopherTrunk learning path
level: beginner
status: full
prereq:
  - a-worked-monitoring-setup
faq:
  - q: I can follow a system now — what's the next challenge?
    a: Three directions branch from here. Go deeper into how trunked systems work on the inside with the Digital & Trunked Radio module. Learn the signal processing that turns raw samples into symbols with the DSP module. Or head toward building the tools yourself with the RF Software Development path. Which you pick depends on whether protocols, signals, or software pulls at you most.
  - q: Do I need to code to go further?
    a: Not to go deeper into the radio side — the digital-trunking and DSP material is about understanding, not programming. Coding only becomes the point if you choose the software-development direction, where the goal is building and modifying decoders like GopherTrunk. Plenty of the hobby's most knowledgeable people never write a line of code.
  - q: What ties all of this together?
    a: GopherTrunk. Everything you monitored in this module — control channels, grants, voice decoding, logging — is something you can now open up, understand more deeply, and eventually shape. The other modules explain the layers beneath the scanner, and the glossary keeps the vocabulary handy as you go.
---

# Where to go next

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
You can now build a station, find and identify systems, follow trunked calls, and log
and record them with a scanner or [GopherTrunk](/learn/scanning/gophertrunk-as-a-scanner/).
From here the hobby branches **three ways**: deeper into **how systems work**
([Digital & Trunked Radio](/learn/digital-trunking/)), down into **the signal
processing** underneath ([DSP](/learn/dsp/)), or outward into **building the tools
yourself** ([RF Software Development](/learn/paths/rf-software-dev/)). Pick the one
that pulls at you — protocols, signals, or software — and keep the
[glossary](/learn/scanning/glossary/) handy as you go.
</div>

You've come the whole way: from what scanning *is*, through building a listening
station, finding and identifying systems, following trunked calls in the field, and
turning it all into a logged, recorded, always-on monitoring post. That's a real,
complete skill — you can sit down at a receiver anywhere and make sense of the traffic
above you. This last lesson is about what comes after "I can follow a system," and
where each direction leads on GopherTrunk.org.

## You've closed the loop

Take stock of what you can do now. You understand
[conventional versus trunked](/learn/scanning/conventional-vs-trunked-recap/), you can
[build a station](/learn/scanning/station-setup/) with a decent
[antenna](/learn/scanning/antennas-for-scanning/), you can
[find and identify](/learn/scanning/identifying-unknown-signals/) systems and
[follow a call](/learn/scanning/following-a-call/) across one, and you can
[log, record](/learn/scanning/logging-and-recording/), and
[alert](/learn/scanning/alerting-on-calls/) on what matters — running unattended if you
like. The scanner is no longer a mystery box; it's a tool you drive. From a solid base
like that, "deeper" can go in a few different directions.

## Deeper into how systems work

If following calls left you curious *why* the control channel says what it says, the
[Digital & Trunked Radio](/learn/digital-trunking/) module is your next stop. This
module was the operator's view — point it here, follow that call. That one is the
inside view: how the control channel signals, how grants and affiliations actually
work, how P25, DMR, NXDN, and TETRA differ under the hood, and how a system behaves
across sites. It answers the "but how does it *know*?" questions this module kept
deferring.

## Down into the signal

If the part that fascinated you was the leap from a raw carrier to clean symbols —
how a smear of radio becomes bits — then the [DSP](/learn/dsp/) module goes there. It's
the signal-processing layer beneath every decoder: filtering, down-conversion,
demodulation, symbol timing, the maths that pulls a signal out of noise. It's the
difference between using a decoder and understanding what it's doing when a marginal
signal locks — or doesn't. This is where "why won't this system decode?" becomes a
question you can answer from first principles.

## Outward into building the tools

And if what you really want is to *shape* the software — fix a decoder, add a
protocol, build your own scanner — the
[RF Software Development path](/learn/paths/rf-software-dev/) is the road from listener
to builder. It braids the radio knowledge with the programming to turn samples into
working software, using GopherTrunk as the worked example throughout. The
[trunked-radio path](/learn/paths/trunked-radio/) is the companion route focused on the
systems themselves. This is where the hobby becomes engineering, and where the
[captures and bug reports](/learn/scanning/contributing-and-community/) you learned to
contribute turn into code.

## It all runs through GopherTrunk

Whichever direction you take, GopherTrunk is the thread. As a
[scanner](/learn/scanning/gophertrunk-as-a-scanner/) it's what you monitored with; as
open software it's something you can open up, understand layer by layer, and
eventually change. The [project home](/) and the
[architecture overview](/architecture.html) are the map of those layers, and the other
modules explain each one in turn. You started this module curious about what's on the
air; you can finish it able to hear it, follow it, record it — and, if you want, build
the very tools that decode it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the DSP module covers the signal processing beneath every decoder: filtering, demodulation, and symbol timing that turn raw samples into bits." markdown="0">
  <p class="knowledge-check__q">Quick check: which direction goes deepest into how raw radio samples become clean symbols?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The Digital &amp; Trunked Radio module</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The DSP module — filtering, demodulation, and symbol timing</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The community & contributing lesson</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- You've closed the loop: **build a station, find and identify systems, follow calls,
  and log, record, and alert** — unattended if you like.
- Go **deeper into how systems work** with
  [Digital & Trunked Radio](/learn/digital-trunking/) — the inside view of control
  channels, grants, and protocols.
- Go **down into the signal** with [DSP](/learn/dsp/) — the filtering, demodulation,
  and timing beneath every decoder.
- Go **outward into building tools** with the
  [RF Software Development path](/learn/paths/rf-software-dev/) — listener to builder,
  with GopherTrunk as the worked example.
- **GopherTrunk is the thread** through all of it — a scanner you used and open
  software you can now understand and shape.

Next up: keep the [glossary](/learn/scanning/glossary/) handy, and dive deeper with [Digital &amp; Trunked Radio](/learn/digital-trunking/) or the [RF Software Development path](/learn/paths/rf-software-dev/).
