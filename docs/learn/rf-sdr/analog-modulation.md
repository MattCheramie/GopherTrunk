---
slug: analog-modulation
redirect_from: /learn/analog-modulation/
title: Analog modulation — AM, FM, SSB
description: The three classic analog modulations explained — amplitude modulation (AM), frequency modulation (FM), and single sideband (SSB). How each puts a voice on a carrier, how they sound and look on a waterfall, and where you still hear them today.
keywords: analog modulation, AM, FM, SSB, single sideband, amplitude modulation, frequency modulation, deviation, airband AM, narrowband FM
level: beginner
status: full
prereq:
  - radio-waves
  - signal-anatomy
faq:
  - q: What is the difference between AM and FM?
    a: AM (amplitude modulation) varies the carrier's strength to carry the audio, keeping its frequency fixed. FM (frequency modulation) varies the carrier's frequency instead, keeping its strength fixed. FM is more resistant to amplitude noise (static), which is why it sounds cleaner for voice and music; AM is simpler and still used where many signals share a band, like aviation.
  - q: Why does aviation still use AM?
    a: Aircraft VHF airband uses AM partly for historical reasons and partly because of a useful property — when two stations transmit at once, AM lets you hear both (a "heterodyne"), so a controller notices a collision. With FM, the stronger signal would simply capture the channel and mask the weaker one.
  - q: What is SSB and why is it efficient?
    a: SSB (single sideband) is a refined form of AM that removes the carrier and one of the two redundant sidebands, transmitting only what's needed. That puts all the power into the information and uses about half the bandwidth, so SSB carries far over HF with modest power. The trade-off is that the receiver must be tuned precisely or the voice sounds distorted.
  - q: How can I tell AM, FM, and SSB apart on a waterfall?
    a: AM shows a central carrier spike with symmetric sidebands. FM is a wider block whose width depends on deviation. SSB is an offset blob of energy with no carrier spike, sitting to one side of where the carrier would be. With practice each has a recognisable footprint.
gophertrunk_links:
  - title: Plots (signal scopes)
    url: /plots.html
    note: see AM/FM/SSB footprints on the live spectrum.
  - title: ADS-B
    url: /adsb.html
    note: aviation's data side, alongside the AM airband voice.
---

# Analog modulation — AM, FM, SSB

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Analog modulation puts a voice on a [carrier](/learn/rf-sdr/signal-anatomy/) by varying one
of its properties. **AM** varies the carrier's **amplitude** (simple, used for
aviation and shortwave). **FM** varies its **frequency** (noise-resistant, used for
broadcast and most analog two-way voice). **SSB** strips AM down to a single sideband
with no carrier — half the bandwidth, all the power on the information, the backbone of
long-distance HF voice. The same three properties — amplitude, frequency, phase —
return as the basis of [digital modulation](/learn/rf-sdr/digital-modulation/).
</div>

Digital modes are built on ideas first used in analog radio, so it pays to meet the
classics. All three change the carrier in step with the audio; they differ in *which*
property they change.

## Amplitude modulation (AM)

In **AM**, the audio rides on the carrier's **amplitude** (strength) while its
frequency stays put. Louder audio → bigger swings in carrier height. It's the oldest
and simplest scheme — easy to build and easy to demodulate.

Its weakness is that noise and interference are *also* amplitude variations, so AM
picks up static readily. Yet it survives where simplicity or a special property
matters: **shortwave broadcast** and **aviation VHF airband**. Aviation keeps AM
partly because if two aircraft transmit at once you hear both (a beat tone), alerting
the controller — whereas FM would let the stronger one silently mask the other.

## Frequency modulation (FM)

In **FM**, the audio varies the carrier's **frequency** while amplitude stays
constant. The amount the frequency swings is the **deviation** — wider deviation
carries louder/richer audio but uses more bandwidth.

Because the information is in frequency, not amplitude, FM can **ignore amplitude
noise** — a receiver just tracks the frequency wiggle and discards strength changes.
That's why FM sounds clean and is everywhere: **FM broadcast** (wide deviation,
~200 kHz wide) and **narrowband FM** two-way voice (~12.5 kHz), the analog cousin of
the digital voice systems GopherTrunk decodes. FM also has a *capture effect*: the
strongest signal on a channel takes over, suppressing weaker ones.

How wide is an FM signal? A handy estimate (Carson's rule) is *bandwidth ≈ 2 ×
(deviation + top audio frequency)*. Narrowband two-way FM with ~2.5 kHz deviation and
3 kHz audio works out to *2 × (2.5 + 3) = 11 kHz* — which is why it fits a 12.5 kHz
[channel](/learn/rf-sdr/frequency-and-spectrum/). Wide-deviation FM broadcast (~75 kHz
deviation, 15 kHz audio) needs *2 × (75 + 15) = 180 kHz* — hence the ~200 kHz channels.
More deviation buys better noise immunity at the cost of bandwidth.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 170" role="img" aria-label="Three rows. A message wave on top. AM below it shown as a wave whose height follows the message. FM shown as a wave whose spacing tightens and loosens with the message." xmlns="http://www.w3.org/2000/svg">
  <text x="6" y="22" font-size="10" fill="currentColor">message</text>
  <path d="M70 20 Q120 0 170 20 T270 20 T370 20" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="6" y="78" font-size="10" fill="currentColor">AM</text>
  <path d="M70 75 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="6" y="138" font-size="10" fill="currentColor">FM</text>
  <path d="M70 135 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0 q4 -16 8 0 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
</svg>
<figcaption>AM follows the message in the wave's <em>height</em>; FM follows it in the wave's <em>spacing</em> (frequency). FM's amplitude stays constant, which is why it shrugs off static.</figcaption>
</figure>

## Single sideband (SSB)

A plain AM signal is wasteful: it sends a carrier (no information) plus *two* mirror-
image sidebands (the same information twice). **SSB** throws away the carrier and one
sideband, transmitting just **one sideband** — either upper (USB) or lower (LSB).

The payoff is big: about **half the bandwidth** and **all the power on the
information**, so a modest SSB transmitter reaches across continents on
[HF](/learn/rf-sdr/frequency-and-spectrum/). That's why amateurs and long-distance services
favour it. The cost is precision — without a carrier to lock onto, the receiver must
be tuned exactly, or voices sound like Donald Duck.

## How they look and sound

| Mode | On the waterfall | Sounds like |
|------|------------------|-------------|
| AM | Central carrier spike + symmetric sidebands | Voice with audible static |
| FM | A wider block (width follows deviation) | Clean voice, hiss when no signal |
| SSB | Offset blob, no carrier spike | Clear voice only when tuned exactly |

Recognising these footprints on the [spectrum and waterfall](/learn/rf-sdr/signal-anatomy/)
is a practical skill — it tells you what you've found before you decode it.

## Where analog still lives

Despite the move to digital, analog is far from gone: **FM broadcast**, **aviation AM
airband**, **marine VHF FM**, **amateur SSB on HF**, **weather radio**, and plenty of
legacy two-way **narrowband FM**. Many of these aren't GopherTrunk's trunked-voice
specialty, but understanding them makes the [digital](/learn/rf-sdr/digital-modulation/)
versions — which vary the very same carrier properties — far easier to grasp.

<div class="knowledge-check" data-quiz data-correct-msg="Right — FM puts the information in frequency, so amplitude noise is ignored." markdown="0">
  <p class="knowledge-check__q">Quick check: why does FM resist static better than AM?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses less bandwidth</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its information is in frequency, not amplitude, so amplitude noise is discarded</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It transmits with more power</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **AM** varies amplitude — simple, static-prone, used for aviation and shortwave.
- **FM** varies frequency — noise-resistant, used for broadcast and analog two-way.
- **SSB** is AM with the carrier and one sideband removed — half the bandwidth, ideal
  for long-distance HF.
- Each has a recognisable **waterfall footprint** and sound.
- The same amplitude/frequency/phase ideas power digital modulation next.

Next: the bridge to digital — how symbol rate (baud) relates to bitrate.
