---
slug: feedlines-and-connectors
title: Feedlines, connectors & grounding
description: The unglamorous parts that quietly cost you signal — coax feedline loss that worsens with frequency and length, the SMA/BNC/N/PL-259 connectors that join it all, and how to ground an outdoor antenna safely against static and lightning.
keywords: coax feedline, coax loss, scanner connectors, SMA BNC N PL-259, feedline loss frequency, antenna grounding, lightning protection, low-loss coax, adapter loss
level: intermediate
status: full
prereq:
  - antennas-for-scanning
faq:
  - q: Does coax type really matter for receiving?
    a: Yes, more than people expect, and more at higher frequencies and longer runs. Cheap thin coax can lose a large fraction of your signal over a long run at UHF, undoing a good antenna. For a short desk jumper it barely matters; for a run up to a rooftop antenna, better low-loss coax is worth it. Keep runs as short as practical and use quality cable for the length you do need.
  - q: Why do connectors and adapters cost signal?
    a: Every connector and every adapter introduces a small loss and a small impedance discontinuity, and stacking several adapters multiplies the effect. A tower of SMA-to-BNC-to-N adapters can quietly eat a meaningful fraction of a decibel and add failure points. Use the right single connector where you can, keep adapter stacks short, and make sure each connection is clean and tight.
  - q: Do I need to ground an outdoor antenna?
    a: If your antenna is outdoors, especially up high, grounding is a safety matter, not an optional upgrade. A proper ground and a lightning arrestor bleed off static build-up and give a nearby strike a path to earth other than through your equipment and home. Follow local electrical codes and, if unsure, have it done properly — this is the one part of the hobby where getting it wrong is dangerous.
---

# Feedlines, connectors & grounding

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Between the [antenna](/learn/scanning/antennas-for-scanning/) and the receiver sits the
**feedline**, and it quietly costs you signal. **Coax loss grows with frequency and
length**, so keep runs **short** and use **low-loss cable** for the length you need.
Every **connector and adapter** (SMA, BNC, N, PL-259) adds a little loss — don't build
adapter towers. And any **outdoor antenna must be grounded**, with a lightning
arrestor, for **safety** — this is the one place in the hobby where getting it wrong is
dangerous, so follow local code.
</div>

The [last lesson](/learn/scanning/antennas-for-scanning/) got a good antenna up high. Now
the signal has to travel down to your receiver without leaking away — and the antenna has
to be safe up there. This lesson covers the plumbing: coax, connectors, and grounding.
None of it is glamorous, and all of it can undo the work you just did on the antenna if
you get it wrong.

## Coax loss: the tax on every metre

The cable that carries your signal from the antenna to the receiver is **coaxial cable —
coax** — and it is not lossless. Every metre absorbs a fraction of a
[decibel](/learn/rf-sdr/antennas/), and two things make that worse:

- **Higher frequency.** Coax loss climbs with frequency. The same cable that barely
  touches a VHF signal can eat a serious chunk of an 800 MHz one.
- **Longer runs.** Loss accumulates with length. A short jumper is nearly free; a long run
  up to a rooftop antenna is where it bites.

The two combine into the practical rule: **the higher your band and the longer your run,
the more the cable matters.** A long, thin, cheap coax at UHF can throw away much of the
signal your good antenna just captured — the classic way a strong antenna is quietly
wasted.

## Choosing and running coax

You do not need exotic cable, just the right cable for the job:

- **Keep runs short.** The cheapest loss reduction is fewer metres. Mount the receiver
  (or an SDR) as close to the antenna as practical, and don't coil up excess cable "just
  in case."
- **Match the cable to the length.** For a short desk jumper, thin flexible coax is fine.
  For a long rooftop run, step up to a proper **low-loss** coax — the extra cost buys back
  decibels you would otherwise lose every second.
- **Weatherproof outdoor connections.** Water in coax is ruinous. Seal outdoor connectors
  against the weather so the cable doesn't slowly fill and rot.
- **Mind the bends.** Sharp kinks damage coax and change its behaviour; keep bends gentle.

Think of it as a budget: you spent effort getting signal onto the antenna, and the coax is
where you can either preserve it or throw it away.

## Connectors you'll meet

The cable ends in **connectors**, and the scanning world uses a handful. You met these in
[Antennas 101](/learn/rf-sdr/antennas/); here they are with where each turns up:

| Connector | Where you'll see it |
|-----------|---------------------|
| SMA | Most RTL-SDR dongles and small antennas |
| MCX | Some SDR dongles |
| BNC | Handheld scanners, lab gear — quick twist-lock |
| N | Larger outdoor antennas and low-loss runs |
| PL-259 / SO-239 (UHF) | Common on scanner antennas and mounts |

The practical point is that your antenna and your receiver often **don't share a
connector**, so you will need an **adapter** — SMA-to-BNC, N-to-SMA, and so on.

## Every connection is a small tax

Each connector and each adapter introduces a **small loss** and a tiny impedance bump, and
— crucially — the effect **stacks**. A tower of three adapters chaining SMA to BNC to N to
PL-259 adds up to a measurable loss and, worse, several extra points that can loosen,
corrode, or fail. The habits that keep this in check:

- **Use the right single connector** where you can, rather than adapting through several.
- **Keep adapter stacks short** — one adapter is fine, a tower of four is asking for
  trouble.
- **Keep every connection clean and tight.** A loose or corroded connector is intermittent
  signal and maddening noise.

None of this is dramatic on its own, but combined with a long lossy coax it is exactly how
a good antenna ends up sounding mediocre.

## Grounding and lightning: the safety part

Everything above is about signal. **Grounding is about safety**, and it is not optional
for an outdoor antenna. A metal antenna up high does two dangerous things: it accumulates
**static charge** from the atmosphere, and it presents an attractive path for a nearby
**lightning** strike. Without a proper ground, both look for a route to earth — and that
route can be through your coax, your receiver, and your house wiring.

The essentials, done to **local electrical code**:

- **Ground the antenna mast and system** to a proper earth ground, so static bleeds off
  harmlessly instead of building up.
- **Fit a lightning arrestor** in the coax line where it enters the building, bonded to
  ground, to divert surge energy away from your equipment.
- **Disconnect during storms.** Even with protection, unplugging the antenna during an
  electrical storm is cheap insurance for your gear.
- **When in doubt, get it done properly.** This is the one area of the hobby where a
  mistake risks your equipment, your home, and your safety — not just a weak signal.

Grounding earns no better reception, but it is what makes an outdoor antenna a safe,
long-term part of your station rather than a liability. With the antenna up, fed, and
grounded, the next step is assembling it all into a working
[station](/learn/scanning/station-setup/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — coax loss grows with both frequency and length, so it bites hardest on long UHF/800 MHz runs." markdown="0">
  <p class="knowledge-check__q">Quick check: where does coax feedline loss hurt the most?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">On a short jumper at low VHF frequencies</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">On a long run at high frequencies like 800 MHz — loss grows with both length and frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's the same regardless of frequency or cable length</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Coax loss grows with frequency and length** — it bites hardest on long runs at high
  bands, and can quietly undo a good antenna.
- **Keep runs short**, use **low-loss coax** for the length you need, weatherproof outdoor
  connections, and avoid sharp bends.
- Learn the **connectors** (SMA, MCX, BNC, N, PL-259) and expect to need an **adapter** to
  join antenna to receiver.
- **Every connector and adapter adds a small loss that stacks** — use the right single
  connector, keep adapter towers short, keep connections clean and tight.
- **Ground any outdoor antenna** and fit a **lightning arrestor** for **safety** — follow
  local code, disconnect in storms, and get it done properly if unsure.

Next up: [Setting up your station](/learn/scanning/station-setup/).
