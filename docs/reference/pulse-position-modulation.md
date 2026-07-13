---
slug: pulse-position-modulation
title: Pulse-position modulation (PPM)
entry_type: technology
category: modulation
description: Pulse-position modulation (PPM) encodes data in the timing of a pulse within a slot rather than its amplitude; it is power-efficient and used in UAT 978 and IR remotes.
keywords: PPM, pulse position modulation, pulse timing, time slot, power efficient, UAT 978, ADS-B, infrared remote, optical, radio control
aka: [pulse-position modulation, PPM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (time-based) }
  - { label: Varies, value: Pulse position within a slot }
  - { label: Used by, value: UAT 978, IR remotes, optical }
see_also: [pulse-code-modulation, pulse-amplitude-modulation, uat-978, ads-b, pulse-width-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Pulse-position_modulation
  - https://en.wikipedia.org/wiki/Universal_Access_Transceiver
---

**Pulse-position modulation** (**PPM**) encodes information in the **timing of a pulse
within a slot**, not in its amplitude or width.[^wiki] Each symbol period is divided into
possible positions, and the transmitter places one narrow pulse in the slot chosen by the
data. Because every symbol uses the same short, full-amplitude pulse, PPM is
**power-efficient** and tolerant of amplitude fading — attractive for optical, infrared,
and impulsive-RF links.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Three symbol slots each divided into four positions, with a single pulse in a different position per slot, illustrating pulse-position modulation." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="440" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-opacity="0.25" stroke-dasharray="3 3"><line x1="150" y1="30" x2="150" y2="110"/><line x1="290" y1="30" x2="290" y2="110"/></g>
  <g stroke="currentColor" stroke-opacity="0.15"><line x1="55" y1="30" x2="55" y2="110"/><line x1="90" y1="30" x2="90" y2="110"/><line x1="125" y1="30" x2="125" y2="110"/><line x1="185" y1="30" x2="185" y2="110"/><line x1="220" y1="30" x2="220" y2="110"/><line x1="255" y1="30" x2="255" y2="110"/><line x1="325" y1="30" x2="325" y2="110"/><line x1="360" y1="30" x2="360" y2="110"/><line x1="395" y1="30" x2="395" y2="110"/></g>
  <g fill="currentColor"><rect x="88" y="45" width="6" height="65"/><rect x="183" y="45" width="6" height="65"/><rect x="358" y="45" width="6" height="65"/></g>
  <g font-size="9" fill="currentColor"><text x="60" y="128">slot 0 -> pos 2</text><text x="200" y="128">slot 1 -> pos 0</text><text x="330" y="128">slot 2 -> pos 3</text></g>
</svg>
<figcaption>PPM sends one pulse per slot; which of the possible positions it occupies carries the data, so all the energy is in timing rather than amplitude.</figcaption>
</figure>

## How it works

The symbol interval is split into *M* time positions, and a single pulse is emitted in
the position that represents the log₂*M* data bits. The receiver only has to detect
*where* in the slot the pulse arrived, so it needs accurate timing but not amplitude
calibration — noise and fading that would corrupt an [amplitude](/reference/pulse-amplitude-modulation/)
scheme leave a correctly placed pulse recoverable. This makes M-ary PPM a close cousin of
orthogonal signalling: like M-ary FSK, it trades bandwidth for power efficiency as M
grows, which is why deep-space optical links favour high-order PPM.

The catch is timing sensitivity. Multipath spreads a pulse across slot boundaries, and
clock jitter between transmitter and receiver directly maps to position error, so PPM
systems spend effort on synchronisation. Simpler *on-off pulse-position* variants used in
radio-control and telemetry gear encode data in the *gap* between pulses, a looser scheme
that undemanding decoders can read with a single timer.

## Relevance to SDR

PPM appears across several domains a software-radio user encounters. In aviation, the
**[UAT 978](/reference/uat-978/)** datalink — the 978 MHz [ADS-B](/reference/ads-b/) and
flight-information channel used in the US — carries its bits with a form of pulse-position
signalling on top of continuous-phase FSK framing. Consumer **infrared remote controls**
use PPM-style gap coding, and many optical and free-space links choose PPM for its
power efficiency. On a time capture, PPM is recognisable as uniform-height pulses that
slide back and forth within regular slots.

GopherTrunk is a land-mobile trunking decoder (P25, DMR, NXDN, TETRA) and does not
demodulate UAT or IR remotes, so PPM is outside its decode chain. It is documented here
to complete the pulse-modulation family alongside PAM and PCM and because UAT/ADS-B are
frequent neighbours on the same SDR hardware.

## Sources

[^wiki]: [Pulse-position modulation](https://en.wikipedia.org/wiki/Pulse-position_modulation) — Wikipedia, for the position-in-slot definition, the power-efficiency trade-off, and its use in optical and IR links.
