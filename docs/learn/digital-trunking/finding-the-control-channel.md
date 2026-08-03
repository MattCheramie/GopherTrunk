---
slug: finding-the-control-channel
title: Finding the control channel
description: Locate the one frequency worth monitoring — use GopherTrunk's Hunt feature, online databases like RadioReference, and the telltale constant data burst, then add the system so GopherTrunk can follow it.
keywords: control channel, find control channel, RadioReference, GopherTrunk hunt, control channel data burst, add trunked system, config.yaml, control channel frequency, hunt SDR
level: intermediate
status: full
prereq:
  - the-control-channel
  - identifying-the-system
faq:
  - q: How do I find a trunked system's control channel?
    a: Three ways, often combined. Look it up in a database like RadioReference, which lists known control-channel frequencies for documented systems. Recognise it on the air by its constant data burst — a control channel never goes quiet. Or let GopherTrunk's Hunt feature sweep a band, detect candidate carriers, identify the protocol, and report which one is the control channel.
  - q: What does a control channel look like on the waterfall?
    a: A control channel carries data continuously, so on the waterfall it shows as a carrier that is always active — a solid, unbroken trace rather than the on-and-off bursts of a voice channel. That constant activity is its single most reliable visual signature. Voice channels light up only when a call is granted; the control channel never stops.
  - q: What is GopherTrunk's Hunt feature?
    a: Hunt is GopherTrunk's tool for discovering systems you don't have frequencies for. It sweeps a band or processes IQ captures, detects carriers above the noise floor, runs each through the protocol identifier and decoder, and builds a map of the system — identity, control channel, and observed talkgroups — which it can export or merge straight into your config so you can start scanning.
  - q: How do I add a system once I know its control channel?
    a: Add the control-channel frequency and system type to GopherTrunk's configuration. If you discovered it with Hunt, the tool can commit the discovery into config.yaml directly. Either way, once the control channel and protocol are configured, GopherTrunk locks the control channel and begins following grants across the system.
gophertrunk_links:
  - title: Hunt (discover systems)
    url: /hunt.html
    note: sweep a band or process captures to find control channels you don't have.
  - title: CC Activity
    url: /cc-activity.html
    note: confirm a control channel is locked by watching its chatter scroll live.
---

# Finding the control channel

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To follow a trunked system you need exactly one frequency: its **[control
channel](/learn/digital-trunking/the-control-channel/)**. Find it three ways — look it up in a **database** like
RadioReference, recognise its **constant data burst** on the
[waterfall](/learn/rf-sdr/fft-and-waterfall/) (a control channel *never* goes quiet),
or let GopherTrunk's **[Hunt](/hunt.html)** feature sweep a band, identify carriers, and
report which one is the control channel. Then **add the control-channel frequency and
system type** to your configuration — Hunt can even commit it for you — and GopherTrunk
locks on and starts following grants.
</div>

You've learned to [identify](/learn/digital-trunking/identifying-the-system/) what a system *is*. Now you need
the one frequency to point GopherTrunk at. Everything else — voice channels,
talkgroups, who's talking — flows from decoding the control channel, so finding it is
the whole job.

## Path 1 — look it up in a database

For documented systems, someone has already done the work. **RadioReference.com** is
the largest community database, listing system identities, sites, and — crucially —
**control-channel frequencies** for tens of thousands of systems. Search by county or
agency, note the listed control-channel (and alternate control-channel) frequencies and
the system type, and you have everything you need to configure GopherTrunk.

This is the easy path, and it's where you should start. The catch: **many systems are
undocumented**. A new county system, a recently rebanded site, or a private network may
not be in any database — which is where the other two paths come in.

## Path 2 — recognise it on the air

A control channel has one unmistakable signature: **it carries data continuously and
never stops**. While voice channels light up only when a call is granted and fall silent
between calls, the control channel transmits its [signalling](/learn/digital-trunking/control-channel-signaling/)
stream every moment the system is up.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 180" role="img" aria-label="A waterfall sketch showing one carrier that is solid and unbroken labelled control channel, and several other carriers that flicker on and off labelled voice channels." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="20" y="20" font-weight="600">Waterfall — time runs downward</text>
    <!-- control channel: solid column -->
    <rect x="70" y="34" width="22" height="128" fill="currentColor" fill-opacity="0.5"/>
    <text x="81" y="176" text-anchor="middle" font-size="9">control</text>
    <!-- voice channels: broken columns -->
    <g fill="currentColor" fill-opacity="0.5">
      <rect x="180" y="40" width="22" height="34"/><rect x="180" y="120" width="22" height="28"/>
      <rect x="290" y="60" width="22" height="50"/>
      <rect x="400" y="34" width="22" height="20"/><rect x="400" y="90" width="22" height="44"/>
    </g>
    <text x="191" y="176" text-anchor="middle" font-size="9">voice 1</text>
    <text x="301" y="176" text-anchor="middle" font-size="9">voice 2</text>
    <text x="411" y="176" text-anchor="middle" font-size="9">voice 3</text>
  </g>
</svg>
<figcaption>The control channel is the one carrier that's always lit. Voice channels flicker on only while a call is granted — so the unbroken column is your control channel.</figcaption>
</figure>

On the [waterfall](/learn/rf-sdr/fft-and-waterfall/) that constant activity stands out as
a solid, unbroken column while the voice channels around it flicker. Tune to the
suspect carrier, watch its [constellation](/constellation.html), and if it locks into a
steady symbol stream that never pauses, you've almost certainly found the control
channel. The [finding-systems](/learn/rf-sdr/finding-systems/) lesson covers the
band-scanning side of this in more depth.

## Path 3 — let GopherTrunk Hunt for it

When you have no frequencies at all, GopherTrunk's **[Hunt](/hunt.html)** feature
automates the search. Point it at a band (or feed it IQ captures of a suspected control
channel) and it:

1. **Sweeps** the band with an FFT and **detects carriers** above the noise floor.
2. **Identifies** the protocol of each candidate across all of GopherTrunk's decoders.
3. **Decodes** the control channel and **accumulates** what it sees — the system's
   identity, the control-channel frequency, and every talkgroup observed in a grant.
4. **Exports** a system map, and can **merge the discovery straight into your
   config** so you can start scanning immediately.

Because Hunt doesn't assume the control channel is the loudest carrier, it works even
when the control channel sits off-centre and below a louder voice channel. This is the
tool that closes the loop on undocumented systems — and it builds the same map a
database lookup would have given you, straight from the air.

## Adding the system so GopherTrunk can follow it

Once you know the **control-channel frequency** and the **system type**, you add them to
GopherTrunk's configuration. If you found the system with Hunt, the tool can commit the
discovery into `config.yaml` for you; if you found it by hand, you enter the frequency
and protocol yourself. Either way, GopherTrunk then **locks the control channel** and
begins reading grants.

Confirm it worked in the **[CC Activity](/cc-activity.html)** panel: within seconds of a
good lock you should see affiliations, registrations, and voice grants scrolling past.
If the panel stays empty, the control channel either isn't locked or you've got the wrong
frequency — a problem we'll tackle in [troubleshooting](/learn/digital-trunking/troubleshooting-a-decode/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the control channel transmits data constantly, so it's the carrier that never goes quiet." markdown="0">
  <p class="knowledge-check__q">Quick check: on the waterfall, which carrier is most likely the control channel?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The one that lights up briefly every few seconds</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The one that's solid and never goes quiet</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The widest one on the band</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- You only need **one frequency** — the control channel — to follow a whole system.
- A **database** like RadioReference lists control channels for documented systems.
- On the air, the control channel is the carrier that **never goes quiet** — a solid
  column on the [waterfall](/learn/rf-sdr/fft-and-waterfall/).
- **[Hunt](/hunt.html)** sweeps, identifies, and maps undocumented systems automatically.
- Add the **control-channel frequency and system type** to config, then confirm the lock
  in **[CC Activity](/cc-activity.html)**.

Next, we'll trace the whole pipeline — from a locked control channel to recorded audio —
in [following a system end to end](/learn/digital-trunking/following-a-system-end-to-end/).
