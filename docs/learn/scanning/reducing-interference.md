---
slug: reducing-interference
title: Reducing noise & interference
description: Why the modern city is so loud on the radio — where local electrical noise comes from, how to hunt down the sources in your own home, and the filters, ferrites, grounding, and habits that cut the racket so weak signals come through.
keywords: radio noise reduction, RFI interference, scanner noise floor, electrical noise radio, front-end overload, band-pass filter, ferrite choke, finding noise source, raise noise floor
level: intermediate
status: full
prereq:
  - station-setup
faq:
  - q: Why is my scanner so noisy in the city?
    a: Modern homes and cities are full of radio noise from switching power supplies, LED lighting, chargers, computers, networking gear, and countless other electronics, all raising the noise floor your receiver has to hear signals above. On top of that, strong nearby transmitters can overload a receiver's front end and create phantom interference. Most of this is local and can be reduced by finding the sources and filtering the strong signals.
  - q: What's the difference between noise and overload?
    a: "Noise raises the floor — a rising hiss from many small sources that buries weak signals. Overload is different: one or a few very strong signals push the receiver's front end past its limits, creating spurious signals and desensitisation across the band even where nothing is really transmitting. Noise is cured by finding and killing sources; overload is cured by attenuating or filtering the strong signal, not by turning up gain."
  - q: How do I find what's causing my noise?
    a: Hunt it down by elimination. Note the noise, then switch off circuits or unplug devices one at a time and watch the noise floor. When the hiss drops as something goes off, you've found a culprit. A portable receiver can also help you sniff along walls and outlets toward the loudest source. Once identified, you can filter it, move your antenna away from it, or replace the offending device.
---

# Reducing noise & interference

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The modern city is **loud on the radio** — switching supplies, LED lighting, chargers,
computers, and networking gear raise the **noise floor** that weak signals must rise
above. Separately, strong nearby transmitters can **[overload your
front end](/learn/rf-sdr/front-end-and-overload/)**, creating phantom signals across the
band. **Noise** is cured by **finding and killing sources**; **overload** by
**attenuating or filtering** the strong signal — never by adding gain. Ferrites, filters,
good grounding, and a well-placed [antenna](/learn/scanning/antennas-for-scanning/) are
your tools.
</div>

Your [station is built and verified](/learn/scanning/station-setup/). If weak signals
still won't come through cleanly, the culprit is usually not the receiver or the antenna
but the **noise** drowning them — and much of it comes from inside your own home. This
final lesson of the unit is about quieting the racket so the signals you fought to capture
actually surface.

## Noise floor vs. overload — two different enemies

Two distinct problems get lumped together as "interference," and they have opposite cures,
so separate them first:

- **A high noise floor.** Countless small sources add up to a rising hiss — the
  **[decibel](/learn/rf-sdr/antennas/) floor** your receiver must hear signals above. Raise
  the floor and weak signals vanish under it. The cure is to **find and eliminate
  sources**.
- **Front-end overload.** One or a few **very strong** signals push the receiver's front
  end past its linear range, producing spurious "phantom" signals and desensitising it
  across the band — even where nothing is transmitting. The cure is to **attenuate or
  filter the strong signal**, covered in depth in [front-end &
  overload](/learn/rf-sdr/front-end-and-overload/).

The critical mistake with overload is reaching for **more gain**: adding gain amplifies the
overloading signal too and makes it *worse*. When phantom signals appear across the band,
think "too much signal," not "too little" — and reduce, don't boost.

## Where the noise comes from

Almost every device in a modern home emits some radio noise, and a few are notorious:

- **Switching power supplies** — phone chargers, laptop bricks, cheap wall-warts — buzz
  across wide swaths of spectrum.
- **LED and fluorescent lighting**, and especially cheap LED drivers and dimmers.
- **Computers, monitors, and their cables**, and USB devices — including, ironically, the
  very computer running your SDR.
- **Networking gear** — routers, powerline (Ethernet-over-mains) adapters, which are
  spectacular noise sources.
- **Motors and appliances** — furnaces, fans, chargers, EV equipment, solar inverters.
- **The neighbours.** Noise doesn't respect property lines; a bad supply next door can
  raise your floor.

The unifying theme: this is mostly **local, human-made noise**, and because it's local,
you have real power to reduce it — unlike distant atmospheric or cosmic noise you simply
live with.

## Hunt the source down

You cannot filter what you can't find, so the highest-value skill is **locating** the
noise, by elimination:

1. **Watch the floor.** Note the noise level on a quiet frequency or the
   [waterfall](/learn/rf-sdr/finding-systems/).
2. **Kill power in blocks.** Switch off circuits at the breaker, or unplug devices one at a
   time, and watch the floor. When the hiss **drops** as something goes off, that device or
   circuit is a culprit.
3. **Sniff it out.** A portable receiver (or an SDR on a laptop) tuned to the noise can be
   walked around the house — the signal gets louder toward the source, letting you home in
   on the offending outlet or gadget.
4. **Confirm and decide.** Once identified, you can replace the device, filter it, move
   your antenna away from it, or simply switch it off while listening.

This detective work, done once, often buys back more weak-signal reception than any
equipment purchase.

## Filters, ferrites, and grounding

Once you know your enemies, a handful of tools cut them down:

- **Ferrite chokes.** Clamp-on ferrite beads around a power lead, USB cable, or coax choke
  off common-mode noise riding on the cable — cheap, easy, and effective against the noise
  many devices radiate along their wires.
- **Band-pass and notch filters.** A **band-pass filter** passes only the range you care
  about and rejects everything else — the standard defence against **overload** from strong
  out-of-band transmitters (broadcast FM, pagers, cellular). A **notch filter** kills one
  specific offending frequency. On the SDR side these are often the single biggest fix for
  a receiver being swamped.
- **Attenuation.** Deliberately reducing signal level with an attenuator can pull an
  overloaded front end back into its linear range — the counter-intuitive fix where *less*
  signal means *more* usable reception.
- **Good grounding and clean power.** A properly
  [grounded](/learn/scanning/feedlines-and-connectors/) antenna system and clean power
  supplies keep noise from being injected in the first place; a cheap supply feeding your
  SDR can be its own worst source.

## Habits that keep it quiet

Beyond hardware, a few habits keep the floor low for good:

- **Get the antenna away from the noise.** Distance and height do double duty — an
  [outdoor, elevated antenna](/learn/scanning/antennas-for-scanning/) escapes the household
  noise cloud as well as seeing further. This alone is often the biggest single
  improvement.
- **Isolate the SDR from its computer.** A short USB extension and a ferrite move the
  dongle out of the computer's noise field.
- **Retest after changes.** Noise sources come and go — a new charger, a neighbour's new
  gadget — so re-run the hunt when a once-quiet band gets loud.
- **Don't chase the impossible.** Some floor is unavoidable; aim to get weak signals *above*
  it, not to reach a perfect silence that doesn't exist in a real environment.

Quieting your station is what lets everything in the previous lessons pay off — a good
antenna, a short feedline, and a capable receiver only shine once the racket is down. With
the station built and the noise tamed, you're ready to point it at the airwaves and find
something to listen to, starting with the community's shared map of what's out there.

<div class="knowledge-check" data-quiz data-correct-msg="Right — overload comes from too much signal, so adding gain makes it worse; you attenuate or filter instead." markdown="0">
  <p class="knowledge-check__q">Quick check: phantom signals appear across the band whenever a strong local transmitter is active. What's the fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Turn up the gain to lift the real signals above the phantoms</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Attenuate or band-pass filter the strong signal — overload is too much signal, so reduce, don't boost</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Add a longer antenna to capture more of the weak signals</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Separate the two enemies: a **high noise floor** (many small sources) versus **front-end
  overload** (a few very strong signals) — they have **opposite cures**.
- **Never add gain to fix overload** — it amplifies the offender; attenuate or filter
  instead.
- Most noise is **local and human-made** — switching supplies, LED lighting, computers,
  networking gear — so you can actually reduce it.
- **Hunt the source by elimination**: watch the floor, kill power in blocks, sniff with a
  portable receiver, then filter, move, or replace it.
- Tools: **ferrite chokes**, **band-pass/notch filters**, **attenuation**, and **clean
  power and grounding**.
- Habits: get the **antenna away from the noise**, **isolate the SDR from its computer**,
  and **retest** when a quiet band gets loud.

Next up: [RadioReference &amp; system databases](/learn/scanning/radioreference-database/).
