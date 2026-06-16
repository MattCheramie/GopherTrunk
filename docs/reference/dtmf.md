---
slug: dtmf
title: DTMF
entry_type: term
category: algorithms
description: DTMF (dual-tone multi-frequency) signalling encodes each keypad digit as the sum of two simultaneous tones — one from a low group and one from a high group.
keywords: DTMF, dual-tone multi-frequency, touch-tone, Goertzel, signalling tones
aka: [DTMF, touch-tone, "dual-tone multi-frequency"]
autolink: true
see_also: [fast-fourier-transform, mdc1200]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/signal-anatomy/ }
external:
  - { title: "DTMF (Wikipedia)", url: https://en.wikipedia.org/wiki/Dual-tone_multi-frequency_signaling }
---

**DTMF** (**dual-tone multi-frequency**, "touch-tone") encodes each key as the **sum of
two tones** — one from a low-frequency group (rows) and one from a high-frequency group
(columns). Detecting which two tones are present identifies the digit. It appears as
in-band signalling on some radio systems.

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

## Overview

DTMF tones are detected with narrow band-pass filters or the **Goertzel algorithm** (an
efficient single-frequency [DFT](/reference/fast-fourier-transform/)). GopherTrunk
synthesises DTMF among other call-progress tones in its audio path.
