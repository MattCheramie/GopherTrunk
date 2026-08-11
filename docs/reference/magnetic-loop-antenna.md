---
slug: magnetic-loop-antenna
title: Magnetic loop antenna
entry_type: term
category: antennas
description: A magnetic loop is a small tuned loop resonated by a capacitor, giving a high-Q, low-noise, narrowband antenna favoured for portable and indoor HF reception.
keywords: magnetic loop antenna, small tuned loop, high Q loop, STL, resonant loop capacitor, low noise receive antenna, portable HF antenna, indoor antenna
aka: [small tuned loop, STL, mag loop]
autolink: true
affiliate: true
product:
  name: "MLA-30+ active receive magnetic loop antenna (0.5–30 MHz)"
  brand: MLA-30+
  category: HF/low-band receive magnetic loop antenna
  lowPrice: "32"
  highPrice: "45"
  url: https://www.amazon.com/dp/B095K89WND?tag=gophertrunk-20
infobox:
  - { label: Type, value: Small resonant tuned loop }
  - { label: Q, value: Very high (narrow band) }
  - { label: Strength, value: Low-noise, compact RX }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B095K89WND?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [loop-antenna, q-factor, antenna, resonance, low-noise-amplifier, dynamic-range]
cite_urls:
  - https://en.wikipedia.org/wiki/Loop_antenna
  - https://en.wikipedia.org/wiki/Q_factor
faq:
  - q: "Which magnetic loop antenna should I buy for SDR listening?"
    a: "For low-band and HF reception in a small space, the MLA-30+ active loop (around $38) is the popular budget pick: a broadband receive loop with a built-in low-noise amplifier that covers roughly 0.5–30 MHz, mounts on a balcony or rooftop, and stays quiet against local electric noise. It is a receive-only, active (amplified) loop, so it trades a tuned loop's razor selectivity for no-retune convenience. Note it is an HF/MW antenna, not a VHF/UHF scanning antenna."
  - q: "Is a magnetic loop useful for GopherTrunk trunking?"
    a: "Not directly. GopherTrunk decodes VHF/UHF land-mobile trunking (P25, DMR, NXDN, TETRA), where wavelengths are short and simple verticals work well, and a high-Q loop would need constant retuning. Mag loops shine at HF/MW shortwave listening on an SDR, where they act as a low-noise, compact antenna and a tracking preselector. For scanning, use a vertical, discone, or Yagi instead."
  - q: "Why use a magnetic loop instead of a long wire?"
    a: "Two reasons: size and noise. A tuned or active loop a fraction of a wavelength across fits on a balcony where a full-size wire cannot go, and because it responds to the magnetic field it stays quieter against the local electric noise that swamps HF in a city — often a bigger win than raw gain. A tuned loop's high Q also preselects, protecting the SDR's front-end dynamic range."
  - q: "Passive tuned loop or active (amplified) loop?"
    a: "A passive tuned loop like the classic small transmitting/receiving loop has the highest selectivity and the best preselection, but you must retune it every time you move in frequency. An active broadband loop such as the MLA-30+ drops the tuning and adds a low-noise amplifier, so it is plug-and-play across the whole HF band at the cost of some out-of-band rejection. For casual SDR listening the active loop is far more convenient."
---

A **magnetic loop antenna** (small tuned loop, or "mag loop") is a
[loop antenna](/reference/loop-antenna/) far smaller than a wavelength — typically an
eighth of a wavelength or less around — brought to [resonance](/reference/resonance/) by a
capacitor across a small gap in the loop.[^wiki] Tuning cancels the loop's large
inductive reactance, so a physically tiny antenna presents a usable match and its weak
radiation resistance is momentarily amplified by a very high [Q](/reference/q-factor/).
The payoff is a compact, sharply tuned, low-noise receiving antenna that works indoors and
on a balcony where a full-size wire is impossible — which is why the mag loop is a
favourite of apartment-bound and portable HF listeners.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A large main loop with a tuning capacitor across a gap at the top and a small coupling loop at the bottom feeding the coax, forming a tuned magnetic loop antenna." xmlns="http://www.w3.org/2000/svg">
  <path d="M230 150 A 70 70 0 1 1 250 150" fill="none" stroke="currentColor" stroke-width="3"/>
  <line x1="235" y1="34" x2="235" y2="20" stroke="currentColor" stroke-width="2"/>
  <line x1="255" y1="34" x2="255" y2="20" stroke="currentColor" stroke-width="2"/>
  <rect x="232" y="6" width="26" height="14" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="265" y="18" font-size="10" fill="currentColor">tuning capacitor</text>
  <circle cx="240" cy="150" r="18" fill="none" stroke="currentColor" stroke-width="2" stroke-opacity="0.7"/>
  <text x="262" y="155" font-size="9" fill="currentColor">coupling loop</text>
  <line x1="240" y1="168" x2="240" y2="180" stroke="currentColor" stroke-width="2"/>
  <text x="150" y="90" font-size="10" fill="currentColor">main loop</text>
</svg>
<figcaption>A capacitor resonates the small main loop; a coupling loop at the bottom transfers energy to the feed line.</figcaption>
</figure>

## How it works

A small loop is mostly inductance with a very small radiation resistance. Left alone it is
a hopeless match. Placing a capacitor across a break in the loop forms a series (or
parallel) resonant circuit: at one frequency the capacitor's reactance exactly cancels the
loop's, leaving only resistance, and a large circulating current builds up. That current is
what actually radiates or receives, so the tuned loop performs far better than an untuned
one of the same size.

The circulating current is large because the circuit [Q](/reference/q-factor/) is very
high — often several hundred. High Q brings the mag loop's characteristic virtues and
vices in one package:

- **Narrow bandwidth.** The antenna is only well matched over a few kilohertz to tens of
  kilohertz, so it must be **retuned** whenever you move more than a little in frequency.
  A remotely driven variable capacitor is standard.
- **Built-in preselection.** That same sharpness rejects out-of-band signals before they
  reach the receiver, easing intermodulation and protecting the SDR's front-end
  [dynamic range](/reference/dynamic-range/).
- **High voltages.** The resonant current develops kilovolts across the capacitor when
  transmitting, so mag loops need wide-spaced or vacuum capacitors — a receive-only loop
  is far more forgiving.

Like any small loop it keeps the **figure-eight pattern** and magnetic-field pickup, so it
stays quiet against local electric noise and can be rotated to null an interferer. A small
**coupling loop** (about a fifth the diameter) or a gamma/Faraday feed transfers energy
between the main loop and the coax without directly loading it.

## In practice

Receive-only mag loops are cheap to build and forgiving of construction, and are widely
sold as compact "wideband" active loops (a broadband loop plus a low-noise amplifier,
trading the tuned loop's selectivity for no-retune convenience). Transmitting mag loops
demand careful high-voltage construction and lose efficiency as they are made smaller, but
reward the effort with a genuinely portable HF antenna. Orientation matters: because pickup
is magnetic and the nulls are sharp, a few degrees of rotation can markedly change the
signal.

## Relevance to SDR

The magnetic loop is one of the best partners for a wideband SDR at HF. SDRs are exposed
to the entire band at once, so a strong shortwave broadcaster or local transmitter can
overload the receiver; the mag loop's high-Q [resonance](/reference/resonance/) acts as a
tracking preselector, knocking down everything but the wanted signal before it hits the
[ADC](/reference/analog-to-digital-converter/). Combined with its low noise and small
footprint, that makes it a standard indoor and field antenna for RTL-SDR and higher-end
receivers doing MW and shortwave work.

GopherTrunk decodes VHF/UHF land-mobile trunking, where wavelengths are short, verticals
are easy, and high-Q retuning would be a nuisance, so mag loops are not part of a
GopherTrunk station. The antenna is documented here as the practical, tuned member of the
[loop](/reference/loop-antenna/) family and a textbook example of trading bandwidth for
selectivity via Q.

## Where to buy

For low-band and HF reception in a small space, an active loop like the **MLA-30+**
(around $38) is the popular budget pick: a broadband receive loop with a built-in
[low-noise amplifier](/reference/low-noise-amplifier/) covering roughly 0.5–30 MHz, quiet
against local electric noise and small enough for a balcony. It is receive-only and
amplified, trading a tuned loop's razor selectivity for no-retune convenience.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B095K89WND?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

This is an **HF/MW listening** antenna, not a scanning antenna — GopherTrunk's VHF/UHF
trunking wants a vertical, [discone](/reference/discone-antenna/), or
[Yagi](/reference/yagi-uda-antenna/) instead. See the
[best SDR antenna guide](/best-sdr-antenna/) for the scanning line-up.

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Loop antenna](https://en.wikipedia.org/wiki/Loop_antenna) — Wikipedia, for the small tuned loop, capacitor resonance, high Q, and narrow bandwidth.
