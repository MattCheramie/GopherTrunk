---
slug: programming-a-trunked-system
title: Programming a trunked system
description: How to get a scanner or SDR to follow a trunked system — entering the control channel and the handful of system parameters that let the receiver read grants and jump to voice channels for you, and what each field actually does.
keywords: program trunked system, scanner trunking, control channel, system type, site frequencies, P25 programming, talkgroup import, GopherTrunk config, trunk tracking, system parameters
level: intermediate
status: full
prereq:
  - frequency-records
faq:
  - q: What do I actually enter to follow a trunked system?
    a: Far less than you'd expect. The core is the system type and at least one control-channel frequency; from there the receiver reads everything else — which voice channel a call is on, who's talking — off the control channel itself. You add talkgroups and site frequencies to shape and improve what you hear, but the control channel is the one thing you can't skip.
  - q: Why don't I program the voice frequencies?
    a: Because on a trunked system the voice channel changes call to call — the system assigns whichever channel is free. You can't park on a fixed voice frequency the way you would on a conventional channel. Instead you give the receiver the control channel, and it reads each call's assigned voice frequency in real time and jumps there for you.
  - q: Do I need every site's frequencies?
    a: Only the sites you can hear and care about. Each site has its own control channel, so to follow a particular site you program that site's control channel. For a single location you often need just one; for a wide-area system you might add several so the receiver can pick the site you're in range of.
---

# Programming a trunked system

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Following a trunked system takes surprisingly little: the **system type** and at least one
**control-channel frequency**. From there the receiver reads each call's assigned voice channel
off the [control channel](/learn/digital-trunking/the-control-channel/) and jumps there for you —
you never program the voice frequencies, because they **change call to call**. You add
**talkgroups** to decide what you hear and extra **site** control channels to follow the site
you're in range of. Get the system type and control channel right and the system comes alive; get
them wrong and nothing decodes.
</div>

Conventional scanning is simple: enter a frequency, park on it, listen. Trunked systems don't work
that way — the conversation hops across a pool of shared voice channels, and a fixed frequency
would catch only fragments. Programming a trunked system means giving the receiver the *one*
channel that coordinates everything, plus a little context, and letting it do the following. This
lesson covers exactly what you enter and why each field matters.

## Why you can't just enter a frequency

On a trunked system there is no permanent "fire dispatch frequency." When someone keys up, the
system grants the call whichever voice channel happens to be free, and the next call from the same
group might land somewhere else entirely. Park on one voice frequency and you'll hear scattered,
disconnected snippets — a bit of one conversation, silence, a bit of another.

The fix is to follow the **control channel** instead. It carries the signalling that assigns every
call to its voice channel, so a receiver reading it knows, in real time, where each conversation
is and who it belongs to. That's why trunk programming centres on the control channel, not the
voice frequencies — a point the [conventional-vs-trunked](/learn/scanning/conventional-vs-trunked-recap/)
recap first raised, now made concrete.

## The parameters that matter

Strip trunk programming to essentials and it's a short list:

- **System type** — P25, DMR, or whichever protocol the system uses. This is the single most
  important field, because it selects the decoder; the wrong type means nothing locks. You worked
  this out when you [identified the system](/learn/digital-trunking/identifying-the-system/).
- **Control-channel frequency** — at least one, ideally with alternates. This is the frequency the
  receiver locks onto and reads. Everything else flows from it.
- **Site frequencies** — for multi-site systems, each site has its own control channel; you add
  the ones you can hear.
- **Talkgroups** — the list of conversations, so the receiver can label calls and you can decide
  which to follow (the [next lesson](/learn/scanning/talkgroups-and-scan-lists/)).

Notice what's *not* there: voice frequencies, because the system assigns those dynamically and the
receiver reads them live.

## Where the numbers come from

You rarely enter these by hand from scratch. The [databases](/learn/scanning/radioreference-database/)
hand you a ready-made profile — system type, every site's control channel, and the full talkgroup
list — which you import wholesale. Your own [frequency records](/learn/scanning/frequency-records/)
fill the gaps and correct anything stale. And if a system isn't documented anywhere, the
[finding the control channel](/learn/digital-trunking/finding-the-control-channel/) lesson shows
how to discover its control channel on the air and confirm the type before you program it.

Whichever way you get them, double-check the system type and control-channel frequency above all
else — those two carry the whole decode.

## What the receiver does with it

Once programmed, the receiver **locks the control channel** and starts reading its stream. When a
radio keys up, the control channel announces a **grant** — this talkgroup, on that voice channel —
and the receiver jumps to the assigned frequency to play the audio, then returns to the control
channel to await the next grant. All of this happens in a fraction of a second, and it's the
subject of the [following a call](/learn/scanning/following-a-call/) lesson.

In GopherTrunk you confirm the lock the same way you would for any system: watch the control
channel's chatter start scrolling. If it stays empty, the system type or control-channel frequency
is wrong — the first things to re-check, and the top of the
[when decoding fails](/learn/scanning/when-decoding-fails/) checklist.

## Getting it wrong — and right

The two mistakes that stop a trunked system decoding are almost always the same two fields.
**Wrong system type** picks the wrong decoder, so even a perfect signal won't lock. **Wrong
control channel** — an inactive alternate, a voice frequency mistaken for the control channel, a
typo — gives you a carrier but no data. Because both are configuration, both are quick to fix once
you suspect them: reconfirm the type, then try each listed control channel in turn. Get those two
right and the rest of trunk following mostly takes care of itself.

<div class="knowledge-check" data-quiz data-correct-msg="Right — on a trunked system the voice channel is assigned per call, so you program the control channel and the receiver reads each call's voice frequency live." markdown="0">
  <p class="knowledge-check__q">Quick check: why don't you program the voice-channel frequencies when following a trunked system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Voice channels are always encrypted and can't be tuned</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The system assigns a voice channel per call, and the receiver reads it off the control channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Voice channels use a different antenna than the control channel</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A trunked call hops across shared voice channels, so you **can't park on a fixed frequency** —
  you follow the **control channel** instead.
- The two fields that carry the decode are the **system type** (selects the decoder) and a
  **control-channel frequency**.
- You add **site** control channels and **talkgroups** to shape what you follow, but never the
  voice frequencies — those are assigned per call.
- The numbers come from **databases**, your own **records**, or on-air **discovery**; double-check
  type and control channel above all.
- Once locked, the receiver reads **grants** and jumps to each call's voice channel for you; an
  empty control channel means the type or frequency is wrong.

Next up: [talkgroups, scan lists &amp; priorities](/learn/scanning/talkgroups-and-scan-lists/).
