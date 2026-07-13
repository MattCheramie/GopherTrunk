---
slug: dtmf
title: DTMF
entry_type: term
category: modulation
description: "DTMF (dual-tone multi-frequency) signalling encodes each keypad digit as the sum of two simultaneous tones — one from a low group and one from a high group."
keywords: DTMF, dual-tone multi-frequency, touch-tone, Goertzel, signalling tones, Q.23, Q.24, in-band signalling, twist, keypad tones
aka: [DTMF, touch-tone, "dual-tone multi-frequency"]
autolink: true
see_also: [goertzel-algorithm, fast-fourier-transform, discrete-fourier-transform, mdc1200, ctcss]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/rf-sdr/signal-anatomy/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Dual-tone_multi-frequency_signaling
  - https://en.wikipedia.org/wiki/Goertzel_algorithm
---

**DTMF** (**dual-tone multi-frequency**, "touch-tone") encodes each key as the **sum of
two tones** — one from a low-frequency group (rows) and one from a high-frequency group
(columns).[^wiki] Detecting which two tones are present identifies the digit. Originally
developed by Bell Labs to replace pulse dialling, it now appears widely as in-band signalling
on telephone and radio systems, including some land-mobile radios.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 200" role="img" aria-label="A telephone keypad grid with a low-frequency tone per row and a high-frequency tone per column." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="40" y="55">697</text><text x="40" y="90">770</text><text x="40" y="125">852</text><text x="40" y="160">941</text>
    <text x="90" y="30">1209</text><text x="140" y="30">1336</text><text x="190" y="30">1477</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="70" y="40" width="40" height="28"/><rect x="120" y="40" width="40" height="28"/><rect x="170" y="40" width="40" height="28"/>
    <rect x="70" y="75" width="40" height="28"/><rect x="120" y="75" width="40" height="28"/><rect x="170" y="75" width="40" height="28"/>
    <rect x="70" y="110" width="40" height="28"/><rect x="120" y="110" width="40" height="28"/><rect x="170" y="110" width="40" height="28"/>
    <rect x="70" y="145" width="40" height="28"/><rect x="120" y="145" width="40" height="28"/><rect x="170" y="145" width="40" height="28"/></g>
  <g font-size="11" fill="currentColor" text-anchor="middle"><text x="90" y="59">1</text><text x="140" y="59">2</text><text x="190" y="59">3</text><text x="90" y="94">4</text><text x="140" y="94">5</text><text x="190" y="94">6</text><text x="90" y="129">7</text><text x="140" y="129">8</text><text x="190" y="129">9</text><text x="90" y="164">*</text><text x="140" y="164">0</text><text x="190" y="164">#</text></g>
</svg>
<figcaption>Each DTMF key sounds one row tone plus one column tone; a detector identifies the pair.</figcaption>
</figure>

## How it works

The scheme uses two mutually prime groups of four tones each: the **low group** at 697, 770, 852
and 941 Hz (the four keypad rows) and the **high group** at 1209, 1336, 1477 and 1633 Hz (the
columns; the fourth column A/B/C/D is rare outside military and amateur gear). Pressing a key sums
exactly one tone from each group, giving 4 × 4 = 16 unique pairs — the twelve familiar keys plus
four extras. The frequencies were chosen so that no tone is a harmonic of another and no sum or
difference of two tones lands on a third, which lets a detector reject voice and music that might
accidentally contain a single matching tone.

A valid digit must satisfy several checks beyond "two tones present":

- **Exactly one tone from each group**, with all others well below threshold, so that two
  simultaneous key presses (talk-off) are rejected.
- **Twist within limits.** The power difference between the high-group and low-group tone (the
  "twist") must stay inside a bounded range; a large imbalance usually means the pair is an
  artefact of speech rather than a real key.
- **Minimum duration.** The tone pair must persist for a minimum on-time (tens of milliseconds)
  to be accepted, guarding against transient tone-like sounds.

Detection is a bank of narrow band-pass filters or, far more efficiently, the
[Goertzel algorithm](/reference/goertzel-algorithm/) — a recursive filter that computes a single
[DFT](/reference/discrete-fourier-transform/) bin's energy at each of the eight target
frequencies without the cost of a full [FFT](/reference/fast-fourier-transform/). The eight
Goertzel outputs are compared to find the strongest tone in each group and to run the twist and
single-tone-per-group checks.

## In practice

DTMF appears far beyond the telephone keypad: interactive voice-response menus, remote control of
answering machines and PBXs, and — relevant to scanning — in-band signalling on some analogue and
mixed land-mobile radio systems, where sequences of DTMF digits carry selective-call (Selcall)
addresses, remote base commands, or repeater control codes. It coexists with sub-audible tone
squelch schemes like [CTCSS](/reference/ctcss/) and with data signalling formats such as
[MDC-1200](/reference/mdc1200/), but unlike those it is fully in the voice audio band and audible
as the familiar dial tones. The frequencies and detection tolerances are standardised in ITU-T
Recommendations Q.23/Q.24.

## Relevance to SDR

For a scanner, DTMF decoding runs on the demodulated audio of an analogue channel: a small
Goertzel bank watches the eight tone frequencies and reports digits as they are keyed, which can
reveal Selcall addressing or control activity on a repeater. The same math runs in reverse to
*synthesise* DTMF. GopherTrunk generates DTMF among the call-progress and signalling tones in its
audio path; the low computational cost of the Goertzel approach makes continuous DTMF monitoring
cheap enough to run on every active audio stream.

## Sources

[^wiki]: [Dual-tone multi-frequency signaling](https://en.wikipedia.org/wiki/Dual-tone_multi-frequency_signaling) — Wikipedia, for the row/column tone-pair encoding of each keypad digit and the standard frequencies.
[^goertzel]: [Goertzel algorithm](https://en.wikipedia.org/wiki/Goertzel_algorithm) — Wikipedia, for the efficient single-frequency detection used to identify the tone pair.
