---
slug: afsk
title: AFSK
entry_type: technology
category: modulation
description: AFSK (audio frequency-shift keying) sends digital data as audio tones that then modulate a radio, classically the Bell 202 tones used by 1200 bps APRS packet.
keywords: AFSK, audio frequency shift keying, Bell 202, 1200 baud, APRS, packet radio, mark space tones
aka: [AFSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (audio FSK) }
  - { label: Classic form, value: Bell 202 (1200/2200 Hz) }
  - { label: Used by, value: APRS / packet radio }
see_also: [frequency-shift-keying, aprs, ax25, ffsk]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Frequency-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-shift_keying }
---

**AFSK** (audio frequency-shift keying) represents bits as **audio tones** that then
modulate a radio (usually FM). The classic case is the Bell 202 standard — 1200 Hz and
2200 Hz tones — carrying 1200 bps [APRS](/reference/aprs/) packet over
[AX.25](/reference/ax25/).

## How it works

Because the keying is at audio frequencies, AFSK can pass through an ordinary FM voice
channel. A receiver demodulates the FM to audio, then detects which tone is present per
bit.

## Relevance to SDR

GopherTrunk decodes AFSK as part of its APRS pipeline, detecting the mark/space tones
after FM demodulation.
