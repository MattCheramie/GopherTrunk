---
slug: searching-and-discovery
title: Search, scan & discovery
description: How to find what the databases don't list — running a frequency search across a band, spotting active channels on a hardware scanner or an SDR waterfall, and close-call techniques for catching the transmitters right next to you.
keywords: frequency search, scanner search mode, close call, signal stalker, SDR waterfall discovery, band scan, finding unlisted frequencies, active channel, service search, spectrum sweep
level: intermediate
status: full
prereq:
  - band-plans
faq:
  - q: What's the difference between scanning and searching?
    a: Scanning steps through a list of frequencies you already know and stops on activity. Searching sweeps a continuous range of frequency looking for any transmission at all, whether or not you have it listed. Scanning tells you when your known channels are busy; searching is how you discover channels you never had.
  - q: What is close call or signal stalker?
    a: It's a feature on many hardware scanners that instantly detects and tunes a strong nearby transmitter without you knowing its frequency in advance. Because a close transmitter overwhelms the receiver, the scanner can lock onto it in a fraction of a second — ideal for catching a radio being used right next to you.
  - q: Is an SDR better than a scanner for discovery?
    a: For discovery, an SDR's waterfall is a real advantage — it shows a whole band at once, so you see every active carrier as a visible trace instead of stepping through frequencies one at a time. A hardware scanner searches faster per channel but sees only where it's tuned this instant. Many hobbyists use both.
---

# Search, scan & discovery

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When the [databases](/learn/scanning/radioreference-database/) come up empty, you go find the
signals yourself. **Searching** sweeps a continuous range of frequency for any activity — unlike
**scanning**, which only steps through channels you already know. On an **SDR** the
[waterfall](/learn/rf-sdr/fft-and-waterfall/) shows a whole band at once so active carriers jump
out visually; on a **hardware scanner**, search mode and **close-call** features hunt for you.
Discovery is where the [band plan](/learn/scanning/band-plans/) pays off — you search the right
neighbourhood, then confirm what you caught.
</div>

Databases document the known. But new systems appear, business licences go unlogged, and the
transmitter in the truck next to you at a traffic light is on *some* frequency nobody has ever
written down. Discovery is the craft of finding those signals — and it's some of the most
satisfying work in the hobby, because what you catch is genuinely yours to identify. This lesson
covers the three main ways to hunt: searching a band, watching a waterfall, and close-call
detection.

## Scanning vs. searching

The two words get used loosely, but the distinction matters. **Scanning** steps through a list
of frequencies you already have and stops when one is active — it tells you *which of your known
channels is busy right now*. **Searching** sweeps a continuous span of frequency, say 450 to 460
MHz, listening for *any* transmission, listed or not. Scanning works a list; searching works a
range.

Discovery is almost entirely a searching activity. You point a search at a band where the
[band plan](/learn/scanning/band-plans/) says interesting traffic lives, let it sweep, and note
every frequency it stops on. Over a few sessions you build a picture of what's actually on the
air in your area — including plenty the databases never captured.

## Searching on a hardware scanner

Most scanners have a **search mode** where you set a lower and upper frequency and the radio steps
across the range, pausing on any active channel. You can define **service searches** (pre-set
ranges for public safety, air, marine, and so on) or your own custom limits. When the scanner
stops, note the frequency, the mode, and anything you can tell about the traffic.

Search mode has a rhythm to learn. Set the step size to match the band's channel spacing so you
land on real channel centres, not between them. Expect false stops on noise and intermittent
carriers, and expect to sit through dead air — searching rewards patience. The payoff is a list
of live frequencies you can then look up or identify.

## Discovery on an SDR waterfall

An SDR changes the game because it shows a whole slice of spectrum **at once**. On the
[waterfall](/learn/rf-sdr/fft-and-waterfall/), every active carrier appears as a visible vertical
trace, so instead of stepping through frequencies blind, you *see* where the activity is. A busy
band lights up like a city skyline, and you simply click the traces that interest you.

This visual search is faster for surveying a band and far better at catching brief or bursty
transmissions, because a signal that keys up anywhere in view leaves a mark you can scroll back
to. It's also how you spot the tell-tale **solid, unbroken column** of a
[control channel](/learn/digital-trunking/the-control-channel/) — a signature that's obvious on
a waterfall and invisible on a channel-stepping scanner. The
[finding-systems](/learn/rf-sdr/finding-systems/) lesson goes deeper on the SDR discovery
workflow.

## Close call — catching what's right next to you

One of the most fun tricks in the hobby is **close call** (branded "Signal Stalker" on some
scanners): the radio instantly detects and tunes a **strong nearby transmitter** without you
knowing its frequency at all. Because a transmitter a few metres away floods the receiver, the
scanner can identify its frequency in a fraction of a second and jump straight to it.

It's ideal for catching a security guard's handheld, a store's radios, an event's crew, or the
service truck at the next pump — anything transmitting close by. The catch is in the name: it
only works on *close, strong* signals, so it's a proximity tool, not a general search. Used well,
it turns "I wonder what they're on" into an answer in seconds.

## Turning a catch into knowledge

Finding a signal is only half the job — next you have to figure out **what it is**. Note
everything at the moment of the catch: the exact frequency, the mode (AM or FM), how wide it is,
whether it's voice or data, and what the [band plan](/learn/scanning/band-plans/) suggests for
that neighbourhood. That bundle of clues feeds straight into the
[identifying an unknown signal](/learn/scanning/identifying-unknown-signals/) lesson, and once
you've named it, into your [frequency records](/learn/scanning/frequency-records/) so the catch
isn't lost.

<div class="knowledge-check" data-quiz data-correct-msg="Right — searching sweeps a continuous range for any activity, which is how you find frequencies no database listed." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to find transmitters that aren't in any database. Which do you use?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Scan mode, stepping through your saved channel list</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Search mode, sweeping a continuous range of frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A priority scan on your favourite talkgroup</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Scanning** works a list of known channels; **searching** sweeps a continuous range for *any*
  activity — discovery is a searching job.
- On a **hardware scanner**, search mode and service searches step across a band; mind the step
  size and expect patience.
- On an **SDR**, the [waterfall](/learn/rf-sdr/fft-and-waterfall/) shows a whole band at once, so
  active carriers — and control channels — are visible at a glance.
- **Close call** instantly tunes a strong *nearby* transmitter without knowing its frequency —
  perfect for what's right next to you.
- Note the frequency, mode, and width at the moment of a catch, then **identify** it and log it.

Next up: [identifying an unknown signal](/learn/scanning/identifying-unknown-signals/).
