---
slug: wavelength
title: Wavelength
entry_type: term
category: rf-fundamentals
description: Wavelength is the physical distance a wave travels in one cycle; for radio it is inversely proportional to frequency and sets antenna dimensions.
keywords: wavelength, lambda, antenna length, frequency relation, quarter wave, propagation
aka: [wavelength]
autolink: true
infobox:
  - { label: Symbol, value: λ (lambda) }
  - { label: Unit, value: Metres }
  - { label: Rule of thumb, value: "λ (m) ≈ 300 / f (MHz)" }
see_also: [frequency, radio-wave, antenna, dipole-antenna, electromagnetic-spectrum, radio-propagation]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Wavelength
  - https://en.wikipedia.org/wiki/Dipole_antenna
---

**Wavelength** (λ) is the physical distance a wave covers in one complete cycle — the
gap between one crest and the next.[^wiki] For a [radio wave](/reference/radio-wave/) it
is inversely proportional to [frequency](/reference/frequency/): higher frequency means
shorter wavelength. Wavelength is the spatial twin of frequency's temporal count, and it
is what physically sizes an [antenna](/reference/antenna/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A sine wave with the distance of one full cycle marked as the wavelength, and the rule of thumb lambda equals 300 divided by frequency in megahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="75" x2="440" y2="75" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 75 C 60 15, 120 15, 160 75 S 260 135, 300 75 S 400 15, 440 75" fill="none" stroke="currentColor" stroke-width="2.2"/>
  <line x1="40" y1="30" x2="180" y2="30" stroke="currentColor" marker-start="url(#wls)" marker-end="url(#wle)"/>
  <text x="110" y="24" text-anchor="middle" font-size="12" fill="currentColor">λ = one cycle</text>
  <text x="20" y="118" font-size="11" fill="currentColor">λ (m) ≈ 300 ÷ frequency (MHz)</text>
  <defs>
    <marker id="wls" markerWidth="8" markerHeight="8" refX="2" refY="3" orient="auto"><path d="M6 0 L0 3 L6 6 z" fill="currentColor"/></marker>
    <marker id="wle" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
</svg>
<figcaption>Wavelength is the physical length of one cycle — inversely proportional to frequency.</figcaption>
</figure>

## How it works

Since radio travels at the speed of light *c*, wavelength and frequency are tied by
*λ = c / f*. Plugging in *c* ≈ 3×10⁸ m/s gives a handy shortcut:
**λ (metres) ≈ 300 ÷ frequency (MHz)**. So a 150 MHz VHF signal is about 2 m long, a
460 MHz UHF signal about 0.65 m, and a 1 GHz signal about 0.3 m. As you climb the
[spectrum](/reference/electromagnetic-spectrum/), wavelengths shrink from kilometres at
the low-frequency end to millimetres at the top of the microwave range — the origin of
the term "millimetre wave."

The number matters because most radio structures are built to a fraction of a
wavelength. An efficient antenna is resonant when its length is a simple fraction of λ:
a half-wave [dipole](/reference/dipole-antenna/) is λ/2, and a quarter-wave whip is λ/4
over a ground plane. That is why a 2 m band antenna is roughly a metre tall while a
Wi-Fi antenna at 2.4 GHz (λ ≈ 12.5 cm) fits inside a laptop lid. Wavelength also sets
the scale of diffraction and reflection: waves bend readily around objects that are
small compared to λ but cast sharp shadows behind objects much larger than λ.

## In practice

- **Antenna sizing.** Knowing the target frequency, an operator computes λ and cuts or
  buys an antenna to a resonant fraction of it. Get this wrong and the antenna presents
  a poor [impedance](/reference/impedance/) match, reflecting power instead of radiating
  or capturing it.
- **Propagation feel.** Long-wavelength HF signals hug the ground and refract off the
  ionosphere; short-wavelength UHF and microwave signals behave more like light, giving
  line-of-sight coverage and pronounced [multipath](/reference/multipath-propagation/).
- **Physical clues.** The size of a repeater's antenna or a radar dish is a quick
  visual estimate of its operating wavelength, and therefore its band.

## Relevance to SDR

Wavelength does not appear directly in an SDR's sample stream — the software works in
frequency and time — but it governs the hardware in front of the converter. Choosing the
right antenna for a target [band](/reference/frequency-bands/) is a wavelength
calculation, and an antenna mismatched to the wavelength you are listening on will starve
the receiver of signal no matter how good the DSP is. For the VHF/UHF land-mobile bands
GopherTrunk targets, wavelengths run from about 2 m down to 30 cm, which is why
discone and vertical whip antennas of that scale are the usual choice.

## Sources

[^wiki]: [Wavelength](https://en.wikipedia.org/wiki/Wavelength) — Wikipedia, on the spatial period of a wave and its inverse relation to frequency.
[^dipole]: [Dipole antenna](https://en.wikipedia.org/wiki/Dipole_antenna) — Wikipedia, on how antenna resonance is set by fractions of the operating wavelength.
