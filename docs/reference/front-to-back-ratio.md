---
slug: front-to-back-ratio
title: Front-to-back ratio (F/B)
entry_type: term
category: antennas
description: Front-to-back ratio is the decibel difference between a directional antenna's forward main-lobe gain and its rearward radiation, measuring how well it rejects signals from behind.
keywords: front-to-back ratio, F/B, front-to-back, back lobe, directional antenna, Yagi, rejection, rear rejection, main lobe, dB
aka: [front-to-back ratio, F/B ratio]
autolink: true
infobox:
  - { label: Type, value: Directional antenna metric }
  - { label: Unit, value: dB }
  - { label: Measures, value: Forward gain vs rearward radiation }
see_also: [yagi-uda-antenna, radiation-pattern, antenna-gain, beamwidth, log-periodic-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Front-to-back_ratio
---

**Front-to-back ratio (F/B)** is the ratio, in [decibels](/reference/decibel/), between the
power a directional antenna radiates in its forward main-lobe direction and the power it
radiates directly to the rear.[^wiki] A high F/B means the antenna is nearly deaf behind it — it
favours signals coming from where it points and rejects those from the opposite bearing. It is a
key figure of merit for a [Yagi](/reference/yagi-uda-antenna/) or any beam antenna, read
straight off the [radiation pattern](/reference/radiation-pattern/) as the difference between the
main lobe and the back lobe.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A large forward main lobe pointing right and a much smaller back lobe pointing left from a central antenna, with the decibel difference between them labelled as the front-to-back ratio." xmlns="http://www.w3.org/2000/svg">
  <path d="M230 80 C 300 20, 430 40, 445 80 C 430 120, 300 140, 230 80 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <path d="M230 80 C 200 62, 165 63, 160 74 C 165 86, 200 88, 230 80 Z" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
  <circle cx="230" cy="80" r="3" fill="currentColor"/>
  <text x="360" y="78" font-size="10" fill="currentColor">front (0 dB)</text>
  <text x="120" y="60" font-size="9" fill="currentColor">back lobe</text>
  <text x="105" y="100" font-size="9" fill="currentColor">(−20 dB)</text>
  <text x="150" y="150" text-anchor="middle" font-size="9" fill="currentColor">F/B = front − back (dB)</text>
</svg>
<figcaption>Front-to-back ratio: the decibel gap between a directional antenna's forward lobe and its much weaker rearward lobe.</figcaption>
</figure>

## How it works

A directional antenna gets its shape by adding several elements whose radiated fields reinforce
forward and largely cancel to the rear. In a Yagi, a **reflector** element sits behind the driven
element; a **director** (or several) sits in front. Their spacings and lengths are tuned so that
re-radiated currents arrive in phase along the boom axis but out of phase toward the back,
suppressing the back lobe. The residual rearward radiation, compared with the forward peak, is
the front-to-back ratio.

Two subtleties matter:

- **F/B is not the same as gain.** Gain measures how much the forward lobe exceeds an isotropic
  reference; F/B measures forward-versus-rearward *for the same antenna*. An antenna can have
  high gain but mediocre F/B, or vice versa, and designs are often optimized for one at the
  expense of the other.
- **"Back" can mean a range, not a point.** Some datasheets quote the ratio to the single
  180° back direction; others quote the worst case over a rear angular window (sometimes called
  the *front-to-rear* ratio), which is the more honest number because a null exactly at 180°
  can flatter the figure.

Typical values run from about 10 dB for a small two-element beam to 20–30 dB for a well-designed
multi-element Yagi. F/B is also sharply frequency-dependent: an antenna optimized for maximum
F/B usually shows a deep, narrow rear null that drifts off the design frequency, so a broadband
[log-periodic](/reference/log-periodic-antenna/) trades some F/B for its wide bandwidth.

## Relevance to SDR

Front-to-back ratio is what lets a directional scanner antenna **reject interference from
behind**. If a strong local transmitter or a co-channel [simulcast](/reference/simulcast/) site
sits roughly opposite the distant system you want, aiming the beam forward puts the interferer in
the back lobe, improving the carrier-to-interference ratio by the F/B figure — often the
difference between a decode and a stream of errors. For direction finding, a clean rear null also
resolves the front/back ambiguity that a symmetric pattern would leave. **GopherTrunk** has no
awareness of the antenna's F/B, but a high-F/B beam raises the effective signal-to-interference
ratio at the SDR input, which is exactly what the demodulator and error-correction stages need to
lock a marginal [control channel](/reference/control-channel/).

## Sources

[^wiki]: [Front-to-back ratio](https://en.wikipedia.org/wiki/Front-to-back_ratio) — Wikipedia, for the decibel definition and its role in directional-antenna evaluation.
