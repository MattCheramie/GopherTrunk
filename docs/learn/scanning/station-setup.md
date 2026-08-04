---
slug: station-setup
title: Setting up your station
description: Assembling the pieces into a working monitoring setup — receiver, antenna, feedline, and (for SDR) a computer running software like GopherTrunk — whether it's a scanner on the desk or a rooftop antenna feeding an always-on post. A practical setup walkthrough.
keywords: scanner station setup, SDR station, monitoring setup, GopherTrunk setup, receiver antenna computer, home scanning station, desk scanner, rooftop antenna setup
level: intermediate
status: full
prereq:
  - feedlines-and-connectors
faq:
  - q: What do I actually need to set up a scanning station?
    a: At minimum a receiver, an antenna, and the feedline joining them. A hardware scanner adds nothing else — antenna in, switch on. An SDR setup also needs a computer (even a small board) running software like GopherTrunk to do the tuning and decoding. From there, sound output, power, and a place to put it all turn the pieces into a station you actually use.
  - q: Where should I put the receiver relative to the antenna?
    a: As close as practical, to keep the feedline short and its loss low. For an SDR that often means mounting the dongle near the antenna and running USB or network to the computer, rather than a long coax run down to the desk. For a hardware scanner, a short good-quality coax to a well-placed antenna is the goal. Short feedline, well-placed antenna is the recurring theme.
  - q: Can I run a scanning station unattended?
    a: Yes, and an SDR running GopherTrunk is well suited to it — it can decode, log, and record around the clock on a small always-on computer. A basic always-on post needs reliable power, stable software, and somewhere to store recordings. This lesson gets the pieces working together; building a hardened 24/7 post is its own later topic.
---

# Setting up your station

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A station is just the pieces of the last three lessons **connected and working
together**: a **receiver**, an **[antenna](/learn/scanning/antennas-for-scanning/)**, the
**[feedline](/learn/scanning/feedlines-and-connectors/)** between them, and — for SDR — a
**computer running software** like GopherTrunk. Keep the **feedline short** by putting the
receiver near the antenna, get the **power and audio** sorted, and give it a stable home.
The same recipe scales from a **scanner on the desk** to a **rooftop antenna feeding an
always-on post**.
</div>

You have [chosen a receiver](/learn/scanning/choosing-a-scanner/), got an
[antenna](/learn/scanning/antennas-for-scanning/) up high, and understand the
[feedline and grounding](/learn/scanning/feedlines-and-connectors/). This lesson assembles
those parts into a working **station** — something you switch on and listen to. The
components are the same whichever road you took; only the amount of assembly differs.

## The pieces, and how they connect

Every scanning station, from the simplest to the most elaborate, is the same short chain:

1. **Antenna** — captures the signal (the ceiling on everything).
2. **Feedline** — carries it down, losing as little as possible.
3. **Receiver** — a hardware scanner, or an SDR that digitizes the spectrum.
4. **(SDR only) Computer + software** — GopherTrunk or similar, doing the tuning,
   decoding, and trunk-tracking.
5. **Output** — a speaker or headphones, plus logging and recording if you want them.

A **hardware scanner** collapses steps 3–5 into one box: antenna in, switch on, listen. An
**SDR** setup keeps them separate — the dongle is the receiver, and a computer does the
rest — which is what gives it its flexibility and what asks a little more of you to set up.

## The desk setup

The simplest station lives on a desk. For a **hardware scanner**, that is genuinely: plug
the antenna in, apply power, and program it. The only real decisions are where the antenna
goes (as [high and outdoor as you can manage](/learn/scanning/antennas-for-scanning/)) and
keeping the feedline decent.

For an **SDR on the desk**, the chain is antenna → coax → SDR dongle → USB → computer →
GopherTrunk. Two practical notes make it painless:

- **Mind USB noise and heat.** SDR dongles run warm and are sensitive to USB electrical
  noise; a short USB extension to move the dongle away from the computer often *reduces*
  interference and improves what you hear.
- **Keep the coax short.** The same feedline logic applies — a long coax to a desk in the
  basement is loss you don't need.

This is the ideal setup to *learn* on. Everything is in reach, easy to change, and quick
to troubleshoot. Get comfortable here before you commit an antenna to the roof.

## The remote-antenna setup

To hear well you want the antenna outdoors and high, but that puts it far from your desk —
and a long coax run [loses signal](/learn/scanning/feedlines-and-connectors/), worst of all
at UHF. The elegant answer, especially for SDR, is to **move the receiver to the antenna
instead of the antenna to the receiver**:

- Mount the **SDR near the antenna** (in a weatherproof enclosure) so the lossy coax run is
  as short as centimetres, then carry the signal the rest of the way as **USB or over the
  network** — which, unlike RF in coax, doesn't degrade with distance.
- A small always-on computer — a Raspberry Pi or mini-PC — up near the antenna running
  GopherTrunk can decode on the spot and stream results to wherever you actually sit.

This is the shape of a serious home station: antenna high and outside, receiver right
behind it, short feedline, and the intelligence delivered to you over a cable that doesn't
care about distance. It is also the natural seed of an
[always-on monitoring post](/learn/scanning/building-a-monitoring-post/).

## Power, audio, and the boring essentials

A station is only as reliable as its dull supporting parts:

- **Power.** Give the receiver and computer clean, stable power. Cheap switching supplies
  can inject [noise](/learn/scanning/reducing-interference/) straight into your reception —
  a common, maddening, and avoidable problem.
- **Audio.** Decide how you'll listen: the scanner's own speaker, powered speakers, or
  headphones on the computer. For SDR, check the software is routed to the output you
  expect.
- **Storage.** If you'll log and record — and an SDR makes that easy — make sure there's
  somewhere for the files to go. A busy trunked system produces a surprising amount of
  audio.
- **A stable home.** Somewhere the gear can sit undisturbed, ventilated, and cabled
  tidily. Nothing kills a station faster than a knocked antenna lead or an overheating
  dongle.

## Verify it end to end

Before you call it done, confirm the whole chain actually works, in order:

1. **Signal in.** Does the receiver see the band — noise floor, and real carriers on the
   [waterfall](/learn/rf-sdr/finding-systems/) for an SDR, or activity on a known-busy
   channel for a scanner?
2. **Decode.** Point it at a known local system and confirm it locks and produces audio.
   For a trunked system, confirm it reads the
   [control channel](/learn/digital-trunking/the-control-channel/) and follows a call.
3. **Output and logging.** Confirm you can hear it, and that recordings and logs land where
   you expect.

If a step fails, you've localised the problem to that link in the chain rather than staring
at the whole station. That habit — test the chain link by link — is the single most useful
troubleshooting skill you can build, and it carries straight into
[reducing interference](/learn/scanning/reducing-interference/), where the enemy is the
noise your own setup and city inject into that chain.

<div class="knowledge-check" data-quiz data-correct-msg="Right — putting the receiver near the antenna keeps the lossy RF feedline short, then USB or network carries the signal the rest of the way without loss." markdown="0">
  <p class="knowledge-check__q">Quick check: your antenna is on the roof and your desk is in the basement. What's the low-loss way to connect them?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Run a long coax all the way down to the desk — cable length doesn't matter for receiving</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Mount the receiver near the antenna to keep the coax short, then carry the signal down over USB or the network</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Use a higher-gain antenna to overcome the long coax loss</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A station is the same short chain every time: **antenna → feedline → receiver → (SDR)
  computer + software → output**.
- A **hardware scanner** collapses the receiver-and-brains into one box; an **SDR** keeps
  them separate, driven by software like GopherTrunk.
- The **desk setup** is ideal to learn on; keep coax short and mind **USB noise and heat**
  on an SDR.
- For a remote antenna, **move the receiver to the antenna** and carry the signal the rest
  of the way over **USB or the network**, which doesn't degrade with distance.
- Don't neglect **clean power, audio routing, storage, and a stable home** — the boring
  parts make or break reliability.
- **Verify the chain link by link** — signal, decode, output — so any fault is localised.

Next up: [Reducing noise &amp; interference](/learn/scanning/reducing-interference/).
