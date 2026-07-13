---
slug: j-pole-antenna
title: J-pole antenna
entry_type: term
category: antennas
description: A J-pole is an end-fed half-wave vertical matched by a quarter-wave parallel stub, giving a groundplane-free omnidirectional antenna shaped like the letter J.
keywords: j-pole antenna, j pole, end-fed half wave, matching stub, quarter-wave stub, slim jim, groundplane-free vertical, twin-lead antenna
aka: [j-pole, j antenna, end-fed half-wave with stub]
autolink: true
infobox:
  - { label: Type, value: End-fed half-wave vertical }
  - { label: Matching, value: λ/4 parallel (J) stub }
  - { label: Pattern, value: Omnidirectional, vertical }
see_also: [dipole-antenna, monopole-antenna, ground-plane-antenna, standing-wave-ratio, feedpoint-impedance]
cite_urls:
  - https://en.wikipedia.org/wiki/J-pole_antenna
  - https://en.wikipedia.org/wiki/Zeppelin_antenna
---

A **J-pole antenna** is a vertical, omnidirectional [antenna](/reference/antenna/)
consisting of an **end-fed half-wave** radiator matched to the feedline by a
**quarter-wave parallel stub** at its base — the two pieces together forming the shape of
the letter J.[^wiki] Its defining virtue is that it needs **no ground plane** or
[radials](/reference/radials-counterpoise/): the stub does the impedance matching that a
ground plane would otherwise provide, making it a rugged, self-contained vertical popular
for VHF/UHF base and portable use.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A J-pole: a tall half-wave radiator on the left rising from a short quarter-wave matching stub of two parallel conductors, with the coax tapped a short way up the stub." xmlns="http://www.w3.org/2000/svg">
  <line x1="170" y1="20" x2="170" y2="150" stroke="currentColor" stroke-width="3"/>
  <text x="178" y="70" font-size="10" fill="currentColor">λ/2 radiator</text>
  <line x1="210" y1="95" x2="210" y2="170" stroke="currentColor" stroke-width="3"/>
  <text x="218" y="130" font-size="9" fill="currentColor">λ/4 stub</text>
  <line x1="170" y1="150" x2="210" y2="150" stroke="currentColor" stroke-width="2"/>
  <line x1="170" y1="170" x2="210" y2="170" stroke="currentColor" stroke-width="2"/>
  <text x="150" y="188" font-size="8" fill="currentColor">shorted base</text>
  <circle cx="170" cy="128" r="3" fill="currentColor"/>
  <circle cx="210" cy="128" r="3" fill="currentColor"/>
  <line x1="170" y1="128" x2="120" y2="128" stroke="currentColor" stroke-width="1.5"/>
  <text x="70" y="124" font-size="8" fill="currentColor">coax tap</text>
  <ellipse cx="170" cy="80" rx="90" ry="16" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3"/>
  <text x="270" y="80" font-size="9" fill="currentColor">omni pattern</text>
</svg>
<figcaption>A J-pole feeds a half-wave radiator through a shorted quarter-wave stub; the coax taps up the stub where the impedance matches 50 ohms.</figcaption>
</figure>

## How it works

A half-wave radiator is a fine antenna, but fed at its **end** it presents a very high
impedance — thousands of ohms — hopelessly mismatched to 50 Ω coax. The J-pole solves this
with a **quarter-wave matching stub**: two parallel conductors shorted at the bottom,
which act as a quarter-wave parallel-line transformer. At the shorted base the impedance is
low; it rises smoothly toward the top. Somewhere a short distance up the stub there is a
point whose impedance equals 50 Ω, and that is where the coax is tapped — one conductor to
each side of the stub. From there the stub steps the low tap impedance up to the high
feedpoint impedance of the half-wave radiator, matching it without any ground plane.

Because the radiator is a full half wavelength worked against the stub rather than a
quarter-wave [monopole](/reference/monopole-antenna/) worked against a ground plane, the
J-pole is electrically complete on its own. The [radiation pattern](/reference/radiation-pattern/)
is essentially that of a vertical half-wave [dipole](/reference/dipole-antenna/):
omnidirectional in azimuth, vertically [polarized](/reference/polarization/), with a
low take-off angle favouring the horizon. Tap position and stub spacing are adjusted for
lowest [SWR](/reference/standing-wave-ratio/).

The best-known relative is the **Slim Jim**, a folded variant built from a single length
of 300 Ω twin-lead ("J Integrated Match"), which packs the same idea into a flat, rollable
ribbon — a favourite portable antenna. The J-pole descends historically from the
end-fed **Zeppelin** ("Zepp") antenna trailed from airships.

## Relevance to SDR

For an SDR listener the J-pole is an easy, cheap, ground-plane-free vertical that beats a
stock [whip](/reference/whip-antenna/) and rivals a [ground-plane](/reference/ground-plane-antenna/)
for fixed VHF/UHF monitoring. Built from copper pipe or twin-lead, it hangs from a single
support with nothing radial sticking out, which is handy indoors, in an attic, or clamped
to a mast. Its vertical polarization matches land-mobile P25, DMR, and NXDN signals, and
its omnidirectional pattern hears all [trunking sites](/reference/trunking-site/) at once.

GopherTrunk is a receive-only decoder and needs no special antenna; a well-tuned J-pole
simply raises the signal-to-noise reaching the front end. Like any single-band resonant
vertical it is narrower in coverage than a [discone](/reference/discone-antenna/), so cut
it for the band you care about most.

## Sources

[^wiki]: [J-pole antenna](https://en.wikipedia.org/wiki/J-pole_antenna) — Wikipedia, for the end-fed half-wave radiator, quarter-wave matching stub, coax tap point, and the Slim Jim variant.
