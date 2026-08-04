---
slug: talkgroups-and-scan-lists
title: Talkgroups, scan lists & priorities
description: Deciding what you actually hear on a busy trunked system — organising talkgroups into scan lists, locking out the traffic you don't want, and setting priorities so the calls you care about interrupt everything else.
keywords: talkgroups, scan list, priority scan, lockout, alpha tag, trunked scanning, talkgroup ID, favourites list, avoid channel, monitoring priorities
level: intermediate
status: full
prereq:
  - programming-a-trunked-system
faq:
  - q: What is a talkgroup?
    a: A talkgroup is a logical channel on a trunked system — a group of radios that talk to each other, like "County Fire Dispatch" or "PD Patrol North." It isn't a frequency; it's an ID the system uses to route a conversation to whichever voice channel is free. Following a trunked system means following talkgroups, not frequencies.
  - q: What's a scan list for?
    a: A scan list is your curated selection of which talkgroups the receiver should follow. A busy system can carry hundreds of talkgroups, most of which won't interest you, so a scan list lets you monitor just the ones you care about — and swap between different lists for different situations.
  - q: What does priority do?
    a: Priority lets an important talkgroup interrupt a lower-priority call in progress. Set your key dispatch channel as priority and, even while you're listening to routine chatter, the receiver breaks away the instant that priority talkgroup keys up — so you never miss the calls that matter most.
---

# Talkgroups, scan lists & priorities

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A busy trunked system carries far more traffic than you want to hear, so you **curate**. A
**talkgroup** is a logical channel — a group of radios, identified by an ID, not a frequency — and
following a trunked system means following talkgroups. **Scan lists** are your chosen selection of
them; **lockout** silences the ones you don't want; and **priority** lets a key talkgroup interrupt
whatever else is playing. Together these turn a firehose into exactly the calls you care about.
Build on [talkgroup basics](/learn/digital-trunking/talkgroups-ids-affiliation/) and your
[frequency records](/learn/scanning/frequency-records/).
</div>

Once a trunked system is [programmed](/learn/scanning/programming-a-trunked-system/) and locked,
you face a new problem: there's too much to hear. A metro system might carry hundreds of
talkgroups — every fire, police, public-works, transit, and school channel in a county — and
listening to all of them at once is just noise. This lesson is about *curation*: choosing what you
hear, silencing what you don't, and making sure the important calls always win.

## Talkgroups, not frequencies

The unit of a trunked system is the **talkgroup** — a set of radios that talk together, identified
by a numeric ID and usually a short **alpha tag** like "Fire Dispatch." It is *not* a frequency;
the system routes a talkgroup's calls to whichever voice channel is free at that moment. So when
you follow a trunked system, you follow **talkgroups**, and the receiver handles the frequency
hopping underneath.

This is why your [records](/learn/scanning/frequency-records/) for a trunked system are a list of
talkgroup IDs and tags rather than frequencies, and why the [talkgroup
basics](/learn/digital-trunking/talkgroups-ids-affiliation/) lesson matters here — the ID is the
handle you organise everything around.

## Scan lists — choosing what you follow

A **scan list** is your curated set of talkgroups to monitor. Instead of following every talkgroup
on the system, you pick the ones that interest you — say, county fire and the two police districts
near you — and group them into a list. The receiver then follows only those, ignoring the rest.

The real power is having **several lists** for different situations. A "daily" list of the channels
you always want, an "incident" list you switch to when something's happening, a "railfan" or
"aviation" list for a specific interest. Switching lists reshapes what you hear in one action,
which is far faster than editing talkgroups one at a time. Consistent alpha tags from your records
make these lists quick to build and easy to read while scanning.

## Lockout — silencing the noise

The mirror image of a scan list is **lockout** (sometimes "avoid"). When a talkgroup you don't care
about keeps interrupting — an automated data channel, a chatty maintenance group, a distant
talkgroup you can barely hear — you lock it out and the receiver skips it from then on. Lockout is
how you refine a list in the field: leave everything on at first, then lock out whatever annoys you
until only the good traffic remains.

It's worth distinguishing **temporary** from **permanent** lockout on radios that offer both — a
temporary lockout clears when you power-cycle, handy for a talkgroup that's busy today but usually
interesting, while a permanent one sticks.

## Priority — never missing the important call

Even within a good scan list, some talkgroups matter more than others. **Priority** solves this: a
talkgroup you mark as priority can **interrupt** a lower-priority call already in progress. Set your
main dispatch channel as priority, and while you're listening to routine chatter the receiver keeps
one ear on that priority talkgroup and breaks away the instant it keys up.

This is the feature that lets you monitor casually without missing the calls you actually tuned in
for. On a busy system it's the difference between hearing a dispatch as it happens and catching it
three calls later — if at all. Use it sparingly: mark only the handful of talkgroups you'd genuinely
drop everything for, or priority loses its meaning.

## Building a listening strategy

Put the three together and you have a strategy rather than a raw feed. Start broad — follow most of
the system to learn what's active and when. Lock out the dead weight as it reveals itself. Group the
keepers into purpose-built scan lists. Then flag the one or two must-hear talkgroups as priority.
The result is a setup tuned to *you*: quiet when nothing's happening, and loud with exactly the
right call when it is.

Every choice you make here also feeds back into your [records](/learn/scanning/frequency-records/) —
which talkgroups earned a place, which got locked out, which deserve priority — so the strategy
sharpens each time you use it. With the lists built, the next question is what actually happens on
the air when one of those talkgroups keys up.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a priority talkgroup interrupts a lower-priority call in progress, so you never miss the calls that matter most." markdown="0">
  <p class="knowledge-check__q">Quick check: you're listening to routine chatter but must not miss main dispatch. What do you set?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Lock out the routine chatter permanently</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Mark main dispatch as a priority talkgroup so it interrupts</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete all your other scan lists</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **talkgroup** is a logical channel — a group of radios identified by an ID, not a frequency —
  and trunked following means following talkgroups.
- **Scan lists** are your curated selection of talkgroups; keeping several for different situations
  lets you reshape what you hear in one action.
- **Lockout** silences talkgroups you don't want; use temporary lockout for the busy-today,
  interesting-usually ones.
- **Priority** lets a key talkgroup **interrupt** a lower-priority call, so you never miss what you
  tuned in for — use it sparingly.
- A good strategy is: follow broadly, lock out the dead weight, group the keepers, flag a few as
  priority — and feed the choices back into your records.

Next up: [following a call across the system](/learn/scanning/following-a-call/).
