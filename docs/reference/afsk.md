---
slug: afsk
title: AFSK
entry_type: technology
category: modulation
description: AFSK (audio frequency-shift keying) sends digital data as audio tones that then modulate a radio, classically the Bell 202 tones used by 1200 bps APRS packet.
keywords: AFSK, audio frequency shift keying, Bell 202, 1200 baud, APRS, packet radio, mark space tones, Bell 103, AX.25, TNC
aka: [AFSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (audio FSK) }
  - { label: Classic form, value: Bell 202 (1200/2200 Hz) }
  - { label: Used by, value: APRS / packet radio }
see_also: [frequency-shift-keying, aprs, ax25, ffsk, packet-radio, kiss-tnc, direwolf, nrzi]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
  - https://en.wikipedia.org/wiki/Bell_202_modem
---

**AFSK** (audio frequency-shift keying) represents bits as **audio tones** that then
modulate a radio (usually FM).[^wiki] The classic case is the Bell 202 standard — 1200 Hz
and 2200 Hz tones — carrying 1200 bps [APRS](/reference/aprs/) packet over
[AX.25](/reference/ax25/). Because the modulation lives entirely in the audio band, AFSK
turns any ordinary voice transceiver into a data radio without touching its RF section.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An audio waveform alternating between a low-pitch tone for one bit value and a high-pitch tone for the other." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q15 -26 30 0 t30 0 t30 0
            M110 60 q8 -26 16 0 t16 0 t16 0 t16 0 t16 0 t10 0
            M230 60 q15 -26 30 0 t30 0 t30 0
            M320 60 q8 -26 16 0 t16 0 t16 0 t16 0 t16 0 t16 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <g font-size="9" fill="currentColor"><text x="55" y="100">space tone</text><text x="150" y="100">mark tone</text></g>
</svg>
<figcaption>AFSK sends data as two audio tones over an FM channel — the scheme behind APRS (Bell 202).</figcaption>
</figure>

## How it works

Instead of shifting the *RF carrier* between frequencies the way plain
[FSK](/reference/frequency-shift-keying/) does, AFSK shifts an *audio-frequency* tone
between a mark and a space pitch and feeds that audio into an ordinary FM (or SSB)
transmitter's microphone input. The RF end is oblivious — it just transmits whatever
audio it is given — so a stock handheld or mobile becomes a modem. In the Bell 202 scheme
a mark is 1200 Hz and a space is 2200 Hz, sent at 1200 baud. The bits are usually
[NRZI](/reference/nrzi/)-encoded (a *change* of tone denotes a zero, no change a one),
which keeps the receiver's clock recovery fed with transitions during long runs of the
same bit and makes the link insensitive to which tone is "high."

A receiver FM-demodulates the RF to recover the audio, then decides which tone is present
in each bit period — historically with a pair of bandpass filters and an envelope
comparator, or a phase-locked loop, and in software with a correlation or Goertzel
detector at the two tone frequencies. The recovered bits are framed as
[AX.25](/reference/ax25/) packets by a terminal node controller
([TNC](/reference/kiss-tnc/)) or its software equivalent.

## Variants

The Bell 202 1200 bps form dominates VHF [packet radio](/reference/packet-radio/) and
APRS, but AFSK spans a family. Bell 103 (300 baud, used on HF) shifts by only ±100 Hz.
[FFSK](/reference/ffsk/) is the *coherent* cousin, where the tone frequencies are exact
integer multiples of the bit rate so each bit holds a whole number of cycles — cleaner to
detect, used for [MDC1200](/reference/mdc1200/) and [DSC](/reference/dsc/). Higher-rate
packet abandons audio tones for direct [GFSK](/reference/gfsk/) at 9600 baud (the G3RUH
modem), which keys the transmitter's modulator directly rather than through the audio
stage.

## Relevance to SDR

AFSK is a natural fit for software decoding: an SDR FM-demodulates the channel, then a
tone detector recovers mark/space and the AX.25 framer extracts packets — exactly what
[Dire Wolf](/reference/direwolf/) does as a soundcard TNC. GopherTrunk decodes AFSK as
part of its APRS pipeline, detecting the Bell 202 mark/space tones after FM demodulation
and passing the framed AX.25 to its position/telemetry parser.

## Sources

[^wiki]: [Frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying) — Wikipedia, for audio FSK and the mark/space tone concept.
[^bell202]: [Bell 202 modem](https://en.wikipedia.org/wiki/Bell_202_modem) — Wikipedia, for the 1200/2200 Hz tones and 1200 baud standard used by APRS.
