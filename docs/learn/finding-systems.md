---
slug: finding-systems
title: Finding & identifying systems
description: How to find trunked radio systems near you — using online databases like RadioReference, scanning a band for control channels on the waterfall, using GopherTrunk's Hunt feature, identifying a system, and adding it to your configuration.
keywords: find trunked system, RadioReference, find control channel, scan for systems, GopherTrunk Hunt, identify radio system, add system config, what can I receive
level: intermediate
status: full
prereq:
  - what-is-trunking
  - fft-and-waterfall
faq:
  - q: How do I find trunked radio systems near me?
    a: Start with an online database such as RadioReference, which lists known systems, their control-channel frequencies, talkgroups, and types by location. Cross-check by looking at the waterfall for a steady control-channel signal in the listed band. If a system isn't in a database, you can scan the band yourself or use GopherTrunk's Hunt feature to discover active control channels.
  - q: What is a control channel frequency and where do I get it?
    a: A trunked system's control channel is the always-on data frequency that coordinates it. You get it from a database listing for the system, or by finding the steady, fixed-width digital signal on the waterfall in the system's band. GopherTrunk needs at least one working control-channel frequency to start following a system.
  - q: What does GopherTrunk's Hunt feature do?
    a: Hunt sweeps a frequency range looking for active control channels, helping you discover systems you don't already have frequencies for. It surfaces candidate control channels so you can identify and add them, which is especially useful for unknown or undocumented systems in your area.
  - q: How do I add a found system to GopherTrunk?
    a: Once you have a system's control-channel frequency (and ideally its type and talkgroup list), you add it to your GopherTrunk configuration, or import details from CSV or a database export. GopherTrunk then locks the control channel, reads the system parameters, and starts following calls.
gophertrunk_links:
  - title: Hunt (discover systems)
    url: /hunt.html
    note: sweep a band to find control channels automatically.
  - title: Import (PDF / CSV)
    url: /import.html
    note: bring in system and talkgroup data in bulk.
  - title: Bookmarks
    url: /bookmarks.html
    note: save frequencies and systems you've found.
---

# Finding & identifying systems

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To monitor a [trunked system](/learn/what-is-trunking/) you first need its **control-
channel frequency**. The fastest route is an **online database** (like RadioReference)
that lists systems, control channels, talkgroups, and types by location. Cross-check on
the **[waterfall](/learn/fft-and-waterfall/)** by spotting the steady, always-on control
channel. For unknown systems, **GopherTrunk's [Hunt](/hunt.html)** sweeps a band to
discover active control channels. Then **add or [import](/import.html)** the system and
GopherTrunk locks on and follows calls.
</div>

Module 6 puts the whole path to work. The very first practical task is finding something
to listen to — specifically, a system's [control channel](/learn/what-is-trunking/), the
key that unlocks everything else.

## Online databases — start here

The easiest way to find systems is to look them up. **RadioReference** (and similar
regional databases) catalogue known systems by location, listing:

- the **system type** ([P25, DMR, etc.](/learn/protocol-landscape/)),
- the **control-channel frequencies**,
- the **talkgroups** and what they're used for,
- and often **radio IDs** and notes.

For most populated areas, your local systems are already documented. Grab the
control-channel frequency and system details, and you're 90% of the way there. This
beats blind scanning almost every time.

## Scanning a band for control channels

When a system *isn't* documented, or you want to confirm one, look on the
[waterfall](/learn/fft-and-waterfall/). In the [band](/learn/frequency-and-spectrum/) the
system uses, hunt for the tell-tale **control-channel footprint**:

- **continuous** — it's always transmitting, unlike voice channels that flicker on/off;
- **fixed width** — a narrow, consistent digital signal;
- **steady strength** — it doesn't come and go.

That always-on stripe among intermittent voice bursts is your candidate.

## Using GopherTrunk Hunt

Scanning by eye is slow. **[Hunt](/hunt.html)** automates it: point it at a frequency
range and it sweeps for active control channels, surfacing candidates for you to
identify and add. This is the tool for **unknown or undocumented systems** — it turns
"is there anything here?" into a list of real signals to investigate.

## Identifying a system from its control channel

Once you've locked a candidate control channel, the **signalling identifies the system**.
GopherTrunk reads the control-channel data to determine the protocol and system
parameters (system ID, etc.), so you can confirm what you've found and match it against a
database entry. The [protocol landscape](/learn/protocol-landscape/) lesson helps you
interpret what you're seeing.

## Adding a system to your configuration

With a confirmed control channel (and ideally its talkgroup list), you bring it into
GopherTrunk:

1. **Add** the system and its control-channel frequency to your configuration.
2. **[Import](/import.html)** talkgroups and details in bulk from CSV or a database
   export, rather than typing them.
3. **[Bookmark](/bookmarks.html)** frequencies and systems you've found for later.

GopherTrunk then locks the control channel, reads its [running
commentary](/learn/what-is-trunking/), and begins following calls — exactly the
[antenna-to-audio](/learn/antenna-to-audio/) flow, now pointed at a real system you
found. If the control channel won't hold a clean lock, the next two lessons —
[tuning with scopes](/learn/tuning-with-scopes/) and
[calibration & troubleshooting](/learn/calibration-troubleshooting/) — are where you go.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the control channel is the always-on data signal you need first." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the single most important frequency to find for a trunked system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Any one voice channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The control-channel frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The FM broadcast band</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- You need a system's **control-channel frequency** above all.
- **Databases** (RadioReference) are the fastest source of systems, control channels, and
  talkgroups.
- Confirm on the **waterfall** by the control channel's **steady, always-on** footprint.
- **Hunt** discovers undocumented systems automatically.
- **Add or import** the system, then GopherTrunk locks on and follows calls.

Next: when a control channel is marginal, how to read the scopes and dial in a clean lock.
