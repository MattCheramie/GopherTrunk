---
slug: following-a-call
title: Following a call across the system
description: What happens in the seconds after a radio keys up on a trunked system — how the control channel grants a voice channel, how your receiver jumps to it, plays the call, and returns, and why calls sometimes clip at the start or hop mid-conversation.
keywords: following a call, trunked call, voice grant, channel grant, control channel to voice, call sequence, trunk tracking, missed call start, late entry, GopherTrunk following
level: intermediate
status: full
prereq:
  - talkgroups-and-scan-lists
faq:
  - q: What happens when someone keys up on a trunked system?
    a: The radio asks the system for a channel, the control channel grants one and announces it, and every radio in that talkgroup — and your receiver — tunes to the assigned voice channel to hear the call. When the talker unkeys, the channel is released back to the pool. The whole grant-and-tune happens in a fraction of a second.
  - q: Why do I sometimes miss the first word of a call?
    a: Because your receiver has to read the grant on the control channel and retune to the voice channel before it can play audio, and that takes a moment. On most systems it's quick enough to catch the whole call, but a slow retune, a weak control channel, or a call you join late can clip the opening syllable.
  - q: Why does a call sometimes jump to a different frequency mid-conversation?
    a: On some systems a long call can be reassigned to a different voice channel, and the control channel announces the move. A receiver following the control channel reads the update and hops along automatically. If it doesn't hop cleanly, you hear the call cut off — usually a sign the control channel decode is marginal.
---

# Following a call across the system

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A trunked call is a **fast handshake**. A radio keys up, the
[control channel](/learn/digital-trunking/the-control-channel/) **grants** a voice channel and
announces it, and your receiver **jumps** to that frequency, plays the call, and returns to the
control channel to wait for the next one — all in a fraction of a second. Understanding this
sequence explains the quirks: why the **first word** sometimes clips, why a call can **hop**
mid-conversation, and why everything depends on a clean control-channel decode. The full protocol
view is in [anatomy of a call](/learn/digital-trunking/anatomy-of-a-call/).
</div>

You've built your [scan lists](/learn/scanning/talkgroups-and-scan-lists/); now watch what happens
when one of those talkgroups actually keys up. A trunked call is over in seconds, but a lot happens
in those seconds, and knowing the sequence turns puzzling behaviour — clipped starts, sudden hops,
calls you almost catch — into something you can read and, when needed, fix. This lesson walks the
call from key-up to release from the listener's chair.

## The sequence, step by step

When a user presses the transmit key on a trunked radio, a quick exchange unfolds:

1. **Request.** The radio sends a channel request to the system over the control channel — "this
   talkgroup wants to talk."
2. **Grant.** The system picks a free voice channel and the control channel **announces the grant**:
   this talkgroup, on that frequency.
3. **Tune.** Every radio in the talkgroup — and your receiver — reads the grant and **retunes** to
   the assigned voice channel.
4. **Talk.** The voice rides on that channel until the user unkeys.
5. **Release.** The channel returns to the shared pool, free for the next call, and the receiver
   goes back to watching the control channel.

The [anatomy of a call](/learn/digital-trunking/anatomy-of-a-call/) lesson covers the protocol-level
detail; what matters here is that your receiver is a passive listener riding along on the same grant
the real radios follow.

## What your receiver is doing

While all this happens, your receiver's job is to **read the control channel continuously** and act
on grants for talkgroups in your active [scan list](/learn/scanning/talkgroups-and-scan-lists/). When
a matching grant appears, it tunes to the voice channel, plays the audio, and — the moment the call
ends — snaps back to the control channel so it doesn't miss the next grant.

This is why the control channel is everything: the receiver is only ever *following instructions* it
reads there. A perfect voice channel is useless if the receiver never saw the grant that pointed to
it. Everything you hear is downstream of a clean control-channel decode.

## Why the first word sometimes clips

Between the grant and the audio there's a small delay: the receiver must decode the grant and retune
before it can play anything. On most systems that's fast enough to catch the whole call, but not
always. A **weak or marginal control channel** slows the decode; a **slow retune** on some hardware
adds a beat; and if you **join a call already in progress** (late entry), you naturally miss what
came before you locked on.

A consistently clipped first syllable usually points at the control-channel decode being marginal
rather than a fault in the audio path — which is exactly the kind of distinction the
[when decoding fails](/learn/scanning/when-decoding-fails/) lesson helps you make.

## Why a call can hop mid-conversation

Sometimes a call jumps to a different voice channel partway through. On some systems a long
transmission can be **reassigned** — the system moves it to another channel and the control channel
announces the change. A receiver following the control channel reads that update and **hops along**
automatically, so you hear the call continue seamlessly.

When the hop *isn't* seamless — the call cuts off where it should have continued — it's a strong
hint that the control-channel decode dropped the update. The audio was fine; the instruction to
follow it went missing. Again, the health of the control channel decides whether following works.

## The priority interrupt in action

If you've set a [priority](/learn/scanning/talkgroups-and-scan-lists/) talkgroup, the following
behaviour changes in one important way: while you're listening to a routine call, the receiver keeps
watching the control channel for a **grant on your priority talkgroup**, and when one appears it
**breaks away** from the current call to follow the priority one. This is the whole sequence running
in service of your priorities — the receiver constantly reading grants and choosing which to follow
based on the rules you set.

Seen this way, "following a call" and "priority" are the same machinery: read the control channel,
match grants against your lists, and jump. Master that mental model and the receiver's behaviour
stops being mysterious.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the receiver reads the grant on the control channel and retunes to the assigned voice channel; that small delay can clip the opening syllable." markdown="0">
  <p class="knowledge-check__q">Quick check: why might you miss the very first word of a trunked call?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The talker always pauses before speaking</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The receiver must read the grant and retune to the voice channel first, which takes a moment</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Trunked systems mute the first second of every call</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A trunked call is a fast handshake: **request → grant → tune → talk → release**.
- The [control channel](/learn/digital-trunking/the-control-channel/) **announces the grant**; your
  receiver reads it, **jumps** to the assigned voice channel, plays the call, and returns.
- Everything you hear is **downstream of the control-channel decode** — a perfect voice channel is
  useless if the grant was missed.
- The **first word can clip** because of the decode-and-retune delay, a marginal control channel, or
  late entry.
- A call can **hop** to a new channel mid-conversation; the receiver follows if it reads the update
  cleanly.
- **Priority** is the same machinery — read grants, match your lists, and jump to what matters most.

Next up: [multisite, simulcast &amp; roaming](/learn/scanning/multisite-and-roaming/).
