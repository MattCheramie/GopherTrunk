---
slug: antennas
title: Antennas 101
description: Antenna basics for SDR — why length follows wavelength, resonance and bandwidth, vertical vs horizontal polarization, gain and directivity, SWR, feedline loss, and the SMA/BNC/N connectors that join an antenna to your software-defined radio.
keywords: SDR antenna, antenna basics, dipole, antenna length wavelength, polarization, antenna gain, SWR, coax loss, SMA BNC connector, discone antenna
level: beginner
status: full
prereq:
  - radio-waves
  - decibels
faq:
  - q: What size antenna do I need for a frequency?
    a: Antenna size follows wavelength. A common design, the half-wave dipole, is about half a wavelength long; a quarter-wave whip is a quarter. Since wavelength ≈ 300 ÷ frequency-in-MHz metres, a 150 MHz antenna's quarter-wave is about 0.5 m, and a 460 MHz one about 0.16 m. Match the antenna to the band you want and it will work far better than a random whip.
  - q: Does antenna placement matter more than the antenna itself?
    a: Often yes. VHF and UHF are line-of-sight, so height and a clear view of the horizon usually help more than a fancier antenna. Getting the antenna outside, up high, and away from obstructions and electrical noise typically beats any upgrade you could make at the radio.
  - q: What is SWR and why does it matter?
    a: SWR (standing wave ratio) measures how well your antenna is matched to your radio and feedline at a given frequency. A good match (low SWR, near 1:1) means most energy is transferred; a poor match reflects energy back. For receive-only SDR use it's less critical than for transmitting, but a badly matched antenna still costs you signal.
  - q: What connector does an RTL-SDR use?
    a: Most RTL-SDR dongles use an SMA or MCX connector. Many antennas and adapters use SMA, BNC, or N connectors. You'll often need a small adapter to join a given antenna to your dongle. Keep adapters and coax runs short and good-quality to minimise loss.
gophertrunk_links:
  - title: Hardware guide
    url: /hardware.html
    note: antenna and dongle pairing notes for GopherTrunk.
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: compare antennas by watching live signal level and SNR.
---

# Antennas 101

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The antenna sets the ceiling on everything downstream — no software can recover a
signal the antenna never caught. Antenna **size follows wavelength** (a quarter-wave
whip is *λ/4*), so match it to your band. **Polarization** (usually vertical for
scanning) should match the transmitter. **Gain** trades coverage for directivity.
**SWR** measures how well the antenna is matched, and **coax/connectors** quietly eat
[decibels](/learn/decibels/). For VHF/UHF, *placement and height usually beat a
fancier antenna.*
</div>

You've learned what a wave is and how power is measured. The antenna is where the
two meet: it turns waves in the air into the faint signal your [SDR](/learn/what-is-sdr/)
digitises. Get it wrong and everything after struggles.

## Why does antenna length follow wavelength?

An antenna works best when its size is a specific fraction of the **wavelength** of
the signal — because the wave's electric field has to "fit" the conductor to drive a
strong current. Two classic designs:

| Design | Length | 150 MHz | 460 MHz |
|--------|--------|---------|---------|
| Quarter-wave whip (λ/4) | a quarter wavelength | ~0.5 m | ~0.16 m |
| Half-wave dipole (λ/2) | half a wavelength | ~1.0 m | ~0.33 m |

Recall *λ ≈ 300 ÷ MHz* from [lesson 1](/learn/radio-waves/): plug in the frequency,
take a quarter or half. A whip cut for the band you care about will dramatically
out-perform a random "telescopic" antenna left at the wrong length.

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="A vertical dipole antenna: two rods, each a quarter wavelength, fed in the middle, radiating a doughnut-shaped pattern around it." xmlns="http://www.w3.org/2000/svg">
  <line x1="180" y1="20" x2="180" y2="68" stroke="currentColor" stroke-width="3"/>
  <line x1="180" y1="82" x2="180" y2="130" stroke="currentColor" stroke-width="3"/>
  <circle cx="180" cy="75" r="3" fill="currentColor"/>
  <text x="190" y="48" font-size="11" fill="currentColor">λ/4</text>
  <text x="190" y="112" font-size="11" fill="currentColor">λ/4</text>
  <ellipse cx="180" cy="75" rx="120" ry="30" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="4 3"/>
  <text x="250" y="75" font-size="11" fill="currentColor">radiation</text>
</svg>
<figcaption>A vertical half-wave dipole. It's most sensitive broadside (around it), least off the ends — which is why a vertical antenna favours signals on the horizon.</figcaption>
</figure>

## What is resonance and bandwidth?

An antenna is **resonant** at the frequency its length is tuned for, where it's most
efficient. Move away from that frequency and efficiency drops. The span over which it
stays "good enough" is its **bandwidth**. A simple whip is fairly narrow; **broadband
antennas** like a *discone* trade a little peak efficiency for coverage across a huge
range — handy for scanning many bands with one antenna.

## What is polarization, and why match it?

**Polarization** is the orientation of the wave's electric field — set by how the
transmitting antenna is mounted. A vertical antenna radiates **vertically polarized**
waves; a horizontal one, horizontal. If your receive antenna's polarization doesn't
match the transmitter's, you can lose a lot of signal (theoretically ~20 dB for a
full mismatch).

Most land-mobile and public-safety radio is **vertical**, so a vertical scanner
antenna is the safe default. FM broadcast is often horizontal or circular.

## What do gain and directivity mean?

Antenna **gain** (in dBi) doesn't create energy — it *focuses* it. A high-gain
antenna squeezes its pattern into a narrower shape, giving more signal in the favoured
direction at the cost of coverage elsewhere:

- An **omnidirectional** vertical hears all compass directions roughly equally —
  ideal when systems surround you.
- A **directional** antenna (Yagi) adds gain toward where it's pointed — great for
  pulling in one distant system, useless for everything off to the side.

For general trunk-tracking, omnidirectional is usually right. Reach for directional
only when chasing a specific weak, distant system.

## What is SWR and feedline loss?

**SWR (standing wave ratio)** measures how well the antenna is matched to your radio
and coax at a frequency. A good match (near 1:1) transfers most energy; a poor match
reflects some back. For receive-only SDR it matters less than for transmitting, but a
badly matched antenna still costs you signal.

Then there's the cable. Every metre of **coax** and every **connector/adapter** eats
a fraction of a [decibel](/learn/decibels/) — more at higher frequencies. Keep runs
short, use decent coax, and minimise adapter stacks. A long, lossy cable can quietly
undo a good antenna.

## Connectors you'll meet

| Connector | Where |
|-----------|-------|
| SMA | Most RTL-SDR dongles, small antennas |
| MCX | Some RTL-SDR dongles |
| BNC | Scanners, lab gear (quick twist-lock) |
| N | Larger outdoor antennas, low loss |

You'll often need a small **adapter** (e.g. SMA-to-BNC) to join a given antenna to
your dongle. See the [Hardware guide](/hardware.html) for specifics.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — a quarter-wave at 150 MHz is roughly 0.5 m." markdown="0">
  <p class="knowledge-check__q">Quick check: roughly how long is a quarter-wave whip for 150 MHz?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">About 5 cm</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">About 50 cm</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">About 5 m</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Antenna **size follows wavelength**; cut it for your band (λ/4 or λ/2).
- **Resonance** sets the best frequency; broadband antennas (discone) cover more.
- Match **polarization** to the transmitter — vertical for most scanning.
- **Gain** focuses the pattern; omnidirectional for general use, directional to chase.
- Mind **SWR** and **coax/connector loss** — and remember placement and height often
  matter most.

Next: where you put that antenna — how signals actually travel from transmitter to you.
