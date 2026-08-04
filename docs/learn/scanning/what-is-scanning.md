---
slug: what-is-scanning
title: What radio scanning is
description: Radio scanning is the hobby of monitoring live two-way radio traffic — police, fire, EMS, aviation, rail, utilities, and business — as it happens. Learn what a scanner does, why people listen, and what a modern scanning setup looks like today.
keywords: radio scanning, what is scanning, police scanner, monitoring radio traffic, two-way radio, scanner hobby, listening to the radio spectrum, trunk tracking scanner
level: beginner
status: full
faq:
  - q: What is radio scanning?
    a: "Radio scanning is monitoring live two-way radio traffic — the working communications of public-safety agencies, aviation, rail, utilities, and business — as it happens over the air. A scanner is a receiver that sweeps quickly through a list of frequencies, stops when it hears activity, lets you listen, then resumes sweeping. It is a receive-only hobby: you listen, you do not transmit."
  - q: Is scanning just listening to the police?
    a: Public-safety dispatch is the classic draw, but it is a small slice of what is on the air. Scanner listeners follow aircraft and air-traffic control, marine and rail operations, weather and emergency services, utility crews, event and business radios, and amateur repeaters. Much of that traffic is routine and open, and the hobby is as much about the breadth of the spectrum as any one service.
  - q: Do I need special equipment to start scanning?
    a: A modern setup is either a dedicated hardware scanner or a software-defined radio (SDR) connected to a computer running software like GopherTrunk. Either one, plus a decent antenna, is enough to start. The rest of this module walks through choosing the gear, putting up an antenna, and finding something to listen to.
---

# What radio scanning is

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Radio scanning** is the hobby of monitoring live two-way radio traffic — public
safety, aviation, rail, utilities, business — *as it happens*. A **scanner** is a
receiver that sweeps quickly through many frequencies, **stops when it hears
activity**, and lets you listen before moving on. It is a **receive-only** pursuit:
you listen, you never transmit. A modern setup is either a dedicated hardware
scanner or a **[software-defined radio](/learn/rf-sdr/what-is-sdr/)** driving software
like GopherTrunk — and everything downstream starts with a decent antenna.
</div>

This module is the operator's side of the hobby: not how digital radio works on the
inside, but how you actually sit down and *listen* to it. This first lesson answers
the simplest question — what is scanning, really? — and sketches what a working setup
looks like today. Later lessons build the station, find the systems, and follow the
calls.

## What a scanner actually does

Most of the radio spectrum is quiet most of the time. A given police dispatch
channel might carry a few seconds of traffic, then nothing for a minute. If you
parked a radio on one frequency you would spend most of your time listening to
silence. A **scanner** solves that by doing what its name says: it **sweeps** — steps
rapidly through a list of frequencies you have programmed, checking each one for a
signal.

When it finds a channel that is active, it **stops and lets you hear it**. As long as
someone is talking, the scanner holds on that frequency. When the transmission ends
and the channel goes quiet, the scanner resumes sweeping, looking for the next active
channel. The whole loop happens many times a second, so from your chair it feels like
one radio that is somehow always on whatever is busy right now.

That single behaviour — sweep, stop on activity, resume — is the heart of the hobby.
Everything else is refinement: which frequencies you load, how you group them, how you
decide what is worth stopping for, and how you follow the more complicated systems that
do not sit still on one channel.

## Why people listen

People come to scanning for very different reasons, and all of them are valid:

- **Situational awareness.** Hearing your local fire, EMS, and public-works traffic
  gives you a real-time picture of your town — a crash on the highway, a storm rolling
  in, a water main that just broke — often well before the news.
- **The technical hobby.** For many, the radios *are* the interest: understanding
  modulation, antennas, trunking, and digital protocols, and building a station that
  pulls in weak or distant systems.
- **A specific service.** Aviation enthusiasts follow air-traffic control; railfans
  follow train crews and dispatchers; storm-spotters follow weather nets; event-goers
  follow race or airshow operations.
- **Emergency preparedness.** When cell networks are down in a disaster, the two-way
  radio traffic of the responders is still on the air, and a scanner is a way to stay
  informed.

You do not have to pick one. Most listeners drift across several of these over time.

## Receive only — the one rule that defines the hobby

Scanning is **listening, not talking**. A scanner is a receiver: it has no
transmitter, and the whole hobby is built on quietly receiving signals that are
already travelling through the air around you. This is what keeps it legal in most
places and separates it from **amateur radio**, where you are licensed to transmit and
hold a two-way conversation.

That receive-only nature also shapes the ethics. You are a listener, not a
participant — you do not interfere, you do not act on what you hear in ways that cause
harm, and you respect that some traffic, while receivable, is not yours to
rebroadcast. We give that its own lesson in
[legal & ethical scanning](/learn/scanning/scanning-legal-and-ethical/), because the
line between *what you can receive* and *what you may do with it* matters.

## What a modern setup looks like

Twenty years ago a scanner was a single box with a whip antenna and a numeric keypad.
That box still exists and still works, but the modern hobby has two roads into it:

- **A dedicated hardware scanner** — a self-contained receiver you switch on, program,
  and listen to. It is the plug-and-play path: no computer required, purpose-built
  buttons, and trunk-tracking built in on the better models.
- **A software-defined radio (SDR)** — a small, cheap receiver that hands raw radio to
  a computer, where software like **GopherTrunk** does the tuning, decoding, and
  following. This path is more flexible and more capable, at the cost of a bit more
  setup.

We compare the two in depth in
[hardware scanners vs. SDR](/learn/scanning/scanners-vs-sdr/). Either way, the setup is
the same shape: a **receiver**, a **decent antenna** (the single biggest factor in what
you can hear), and — for anything trunked or digital — some brains to decode and follow
the system. GopherTrunk sits at the software end of that spectrum, turning an SDR into a
trunk-tracking scanner.

## Conventional and trunked, in one breath

You will meet two families of radio systems early. **Conventional** systems assign a
fixed frequency to a channel — dispatch is *here*, tactical is *there* — so you can park
on it and listen. **Trunked** systems pool a handful of frequencies and hand them out to
calls on demand, coordinated by a **[control channel](/learn/digital-trunking/the-control-channel/)**,
so a single conversation can hop from frequency to frequency. Following a trunked system
means letting your scanner read that control channel and chase the calls it grants — the
whole reason trunk-tracking scanners exist. The next few lessons unpack this; the
digital module covers the theory in
[conventional vs. trunked](/learn/digital-trunking/conventional-vs-trunked/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a scanner sweeps through frequencies and stops when it hears activity, then resumes." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a scanner do that a radio parked on one frequency does not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It transmits a reply so the dispatcher knows you're listening</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It sweeps through many frequencies and stops on whichever one is active</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It decrypts encrypted traffic automatically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Scanning** is monitoring live two-way radio traffic as it happens — public safety,
  aviation, rail, utilities, business, and more.
- A **scanner** sweeps through many frequencies, **stops on activity**, lets you
  listen, and resumes — that sweep-and-stop loop is the core of the hobby.
- People listen for **situational awareness**, the **technical challenge**, a
  **specific service**, or **preparedness** — often several at once.
- The hobby is **receive-only**: you listen, you never transmit, which is what keeps it
  legal and shapes its ethics.
- A modern setup is a **hardware scanner** or an **SDR plus software** like GopherTrunk,
  and always a **decent antenna**.
- Systems come in two families — **conventional** (fixed channels) and **trunked**
  (frequencies handed out on demand) — and following trunked systems is what the rest of
  the module builds toward.

Next up: [A short history of scanners](/learn/scanning/history-of-scanning/).
