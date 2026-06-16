---
slug: propagation
title: How signals travel
description: RF propagation for beginners — line-of-sight and the radio horizon, how terrain and buildings attenuate signals, multipath and fading, how propagation changes with frequency, and practical antenna-placement tips for a scanner setup.
keywords: RF propagation, line of sight, radio horizon, multipath, fading, signal attenuation, antenna placement, VHF UHF propagation, scanner reception tips
level: beginner
status: full
prereq:
  - radio-waves
  - antennas
faq:
  - q: Why can't I receive a station that isn't far away?
    a: At VHF and UHF, radio is mostly line-of-sight. Hills, buildings, and even dense trees between you and the transmitter block or weaken the signal. A station only a few kilometres away can be unreachable if terrain is in the way, while a more distant one on a hilltop comes in clearly. Height and a clear path matter more than raw distance.
  - q: What is the radio horizon?
    a: The radio horizon is the farthest point a line-of-sight signal reaches before the curvature of the Earth blocks it. It's slightly farther than the visual horizon because the atmosphere bends radio waves a little. Raising the antenna extends the radio horizon, which is why height helps so much for VHF/UHF reception.
  - q: What is multipath?
    a: Multipath is when a signal reaches your antenna by several paths at once — directly and via reflections off buildings, terrain, or vehicles. The copies arrive slightly out of step and can reinforce or cancel each other, causing fading and, for digital signals, decoding errors. Moving the antenna a little can change multipath dramatically.
  - q: How do I improve reception?
    a: Get the antenna higher and outdoors with a clear view toward the systems you want, match its polarization (usually vertical), keep coax short, and move it away from sources of electrical noise like computers, chargers, and LED lighting. For VHF/UHF, placement and height usually beat buying a better radio.
gophertrunk_links:
  - title: Tuning (receiver meters)
    url: /tuning.html
    note: watch SNR change live as you reposition the antenna.
  - title: Hunt (discover systems)
    url: /hunt.html
    note: find which systems actually reach your location.
---

# How signals travel

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
At the VHF/UHF frequencies most scanning uses, radio is essentially
**line-of-sight** — it travels to the **radio horizon** and stops, blocked by hills,
buildings, and the Earth's curve. Reflections create **multipath**, which causes
**fading** and decoding errors. Lower frequencies travel and penetrate better; higher
ones are blocked more easily. The single biggest lever you control is **antenna
placement**: height, a clear path, matched polarization, and distance from noise
usually beat any radio upgrade.
</div>

You've got an [antenna](/learn/antennas/) sized for your band. Where you put it, and
what's between it and the transmitter, decides whether a system comes in clean or not
at all. This lesson explains why.

## What is line-of-sight and the radio horizon?

At VHF and UHF, radio waves travel in nearly straight lines — **line-of-sight**. They
don't bend far around the Earth, so they reach a limit called the **radio horizon**,
where the planet's curvature gets in the way. It's a bit beyond the *visual* horizon
because the atmosphere refracts radio slightly.

The key consequence: **height extends range.** Raising either antenna pushes the
horizon out. A transmitter on a tall tower or hilltop reaches far; a receiver up high
with a clear view hears far. This is why repeaters live on towers and why getting your
antenna up and outside helps so much.

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 150" role="img" aria-label="A curved Earth with a tall transmitter tower and a receiver. A straight line of sight reaches over the curve to the horizon; an obstacle blocks a low path." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 140 Q210 90 410 140" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.5"/>
  <line x1="70" y1="120" x2="70" y2="60" stroke="currentColor" stroke-width="2"/>
  <text x="50" y="52" font-size="10" fill="currentColor">TX</text>
  <line x1="350" y1="118" x2="350" y2="88" stroke="currentColor" stroke-width="2"/>
  <text x="338" y="80" font-size="10" fill="currentColor">RX</text>
  <line x1="70" y1="60" x2="350" y2="88" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="180" y="62" font-size="10" fill="currentColor">line of sight</text>
  <rect x="205" y="100" width="14" height="20" fill="currentColor" fill-opacity="0.3"/>
  <text x="200" y="135" font-size="9" fill="currentColor">obstacle</text>
</svg>
<figcaption>Line-of-sight reaches to the radio horizon; height extends it. An obstacle in the path blocks or weakens the signal.</figcaption>
</figure>

## How do terrain and buildings weaken signals?

Anything in the path **attenuates** (weakens) the signal. Hills and large buildings
can block it entirely (a "shadow"). Walls, foliage, and even rain take their toll —
more so at higher frequencies. This is why a station only a few kilometres away can be
unreachable if a ridge or tower block sits between you, while a more distant hilltop
system comes in fine. **Distance is only part of the story; the path matters more.**

## What are multipath and fading?

A signal rarely takes just one route to your antenna. It also bounces off buildings,
terrain, and vehicles, so several copies arrive — slightly delayed relative to each
other. That's **multipath**. Because the copies are out of step, they can add up or
partly cancel, making the signal level **fade** up and down, sometimes as things
(like traffic) move.

For analog you hear this as flutter; for **digital** signals multipath smears the
[symbols](/learn/digital-modulation/) and can push decoding past its error limit.
Often the fix is simply to **move the antenna** a short distance — even tens of
centimetres can swap a deep fade for a strong spot.

## How does propagation change with frequency?

As a rule of thumb, lower frequencies travel and penetrate better:

| Band | Behaviour |
|------|-----------|
| HF | Can refract off the ionosphere — worldwide "skip" |
| VHF | Line-of-sight, bends slightly, decent building penetration |
| UHF | Line-of-sight, more easily blocked, but good in urban multipath |
| SHF | Very line-of-sight, blocked by almost anything |

The trunked systems GopherTrunk follows sit in VHF/UHF/700-800 MHz, so think
**line-of-sight with multipath** — height and a clear path are your friends.

## Practical placement tips

- **Go up and out.** Higher and outdoors beats low and indoors almost every time.
- **Clear the path** toward the systems you want; aim around big obstructions.
- **Match polarization** — vertical for most land-mobile (see [antennas](/learn/antennas/)).
- **Keep coax short** to limit loss.
- **Escape noise.** Computers, USB hubs, chargers, and LED/CFL lighting raise your
  [noise floor](/learn/decibels/); distance from them improves SNR more than you'd
  expect.

Watch the effect live in GopherTrunk's [tuning meters](/tuning.html): reposition the
antenna and see SNR rise or fall in real time.

<div class="knowledge-check" data-quiz data-correct-msg="Right — at VHF/UHF a clear, high path matters more than raw distance." markdown="0">
  <p class="knowledge-check__q">Quick check: a system 3 km away won't decode, but one 30 km away on a hilltop does. The likeliest reason?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The far system transmits on a lower frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Terrain/buildings block the near one; the far one has line-of-sight</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Closer signals are always weaker</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- VHF/UHF radio is **line-of-sight**, limited by the **radio horizon**; height extends it.
- **Terrain and buildings** attenuate or block signals — path beats distance.
- **Multipath** causes **fading** and digital errors; moving the antenna often helps.
- Lower frequencies travel/penetrate better; the trunked bands are line-of-sight.
- **Placement** — height, clear path, polarization, low noise — is your biggest lever.

That wraps Module 1. Next module: how information is actually put onto these waves,
starting with the anatomy of a signal on screen.
