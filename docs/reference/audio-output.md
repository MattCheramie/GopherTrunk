---
slug: audio-output
title: Audio output (PortAudio/ALSA)
entry_type: concept
category: sdr-app-building
description: "Audio output in an SDR sends demodulated audio to the sound card by resampling it to the device's rate and feeding a callback buffer without starving it, which would cause underruns."
keywords: audio output, PortAudio, ALSA, sound card, resample to soundcard rate, audio buffer, underrun, xrun, callback, demodulated audio, 48 kHz, ring buffer
aka: ["sound output", "audio playback", "soundcard output"]
autolink: true
infobox:
  - { label: Type, value: "Output stage / device I/O" }
  - { label: Job, value: "Resample decoded audio to the card's rate" }
  - { label: Hazard, value: "Buffer underrun (xrun) → clicks/gaps" }
see_also: [demodulation, overruns-underruns, resampler, ring-buffer, sample-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/PortAudio
  - https://en.wikipedia.org/wiki/Advanced_Linux_Sound_Architecture
---

**Audio output** is the final stage of a receive chain that hands
[demodulated](/reference/demodulation/) audio to the computer's sound card so a
human can hear it. Its two jobs are to convert the audio to the sample rate the
device actually runs at — commonly 48 kHz — and to keep the device's playback
buffer continuously fed, because a sound card that runs out of samples produces an
audible gap or click called an **underrun**.[^pa] Cross-platform toolkits like
**PortAudio** and Linux's **ALSA** provide the interface between application code
and the audio hardware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Demodulated audio at one rate is resampled to the sound card rate, buffered in a ring buffer, and pulled by the audio callback that feeds the speaker; if the buffer empties, an underrun occurs." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="aoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="12" y="52" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="47" y="64">demod audio</text><text x="47" y="75" font-size="7">e.g. 8 kHz</text>
    <line x1="82" y1="66" x2="112" y2="66" stroke="currentColor" stroke-width="1.1" marker-end="url(#aoar)"/>
    <rect x="114" y="52" width="66" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="147" y="64">resample</text><text x="147" y="75" font-size="7">→ 48 kHz</text>
    <line x1="180" y1="66" x2="210" y2="66" stroke="currentColor" stroke-width="1.1" marker-end="url(#aoar)"/>
    <rect x="212" y="50" width="80" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="252" y="63">ring buffer</text><text x="252" y="75" font-size="7">producer/consumer</text>
    <line x1="292" y1="66" x2="322" y2="66" stroke="currentColor" stroke-width="1.1" marker-end="url(#aoar)"/>
    <rect x="324" y="50" width="70" height="32" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="359" y="63">audio</text><text x="359" y="75" font-size="7">callback</text>
    <line x1="394" y1="66" x2="424" y2="66" stroke="currentColor" stroke-width="1.1" marker-end="url(#aoar)"/>
    <text x="440" y="69" font-size="7">🔊</text>
    <text x="252" y="102" font-size="7">empty buffer → underrun (gap)</text>
  </g>
</svg>
<figcaption>Decoded audio is rate-converted, queued in a ring buffer, and pulled by the device callback; the buffer decouples a bursty producer from the card's steady demand.</figcaption>
</figure>

## How it works

Sound cards run on their own clock and demand a fixed number of samples at a fixed
rate. The application does not push audio whenever it likes; instead the audio
library invokes a **callback** (or drains a queue) at regular intervals, asking for
the next block of samples right now. The application's job is to always have that
block ready.

Two problems must be solved to keep the block ready:

- **Rate conversion.** The decoder rarely produces audio at the card's rate — a
  voice codec might output 8 kHz while the device wants 48 kHz — so a
  [resampler](/reference/resampler/) converts between them. The ratio must be exact
  and continuous; a wrong or drifting ratio slowly empties or overflows the buffer.
- **Buffering.** The decoder produces audio in bursts (a whole voice frame at once,
  then nothing), while the card consumes it in a steady trickle. A
  [ring buffer](/reference/ring-buffer/) sits between them: the decoder writes when
  it has data, the callback reads a fixed amount each time, and the buffer absorbs
  the mismatch.

If the callback asks for samples and the buffer is empty, that is an **underrun**
(ALSA calls it an *xrun*): the card plays silence or repeats stale data, producing
a click or dropout.[^alsa] The mirror problem, an **overrun**, occurs on the input
side when a producer outpaces a consumer — both are covered under
[overruns and underruns](/reference/overruns-underruns/).

## In practice

Buffer size is the central tuning knob and a direct latency-versus-safety trade. A
large buffer rarely underruns but adds delay between a transmission and hearing it;
a small buffer is responsive but underruns the moment the CPU is briefly busy. Real
systems pick the smallest buffer that survives normal scheduling jitter, often a
few tens of milliseconds.

Because the callback runs on a time-critical audio thread, the golden rule is to do
no slow or blocking work inside it — no file I/O, no lock that a slow thread holds,
no memory allocation that might stall. The callback should only copy ready samples
out of the ring buffer; all the heavy DSP happens on other threads. Sample-rate
clock drift between the SDR's clock and the sound card's clock is a subtle
long-run issue, handled by occasionally nudging the resampler ratio or dropping and
inserting samples so the buffer neither drains nor overflows over minutes of
playback.

PortAudio abstracts all of this across Windows, macOS, and Linux behind one
callback API; on Linux, ALSA is the kernel-level interface PortAudio (and other
layers like PulseAudio or PipeWire) sit on top of.

## Relevance to SDR

Audio output is the payoff stage of any voice-receiving SDR — a scanner, a ham
transceiver front-end, an air-band monitor — and getting the resample-and-buffer
plumbing right is the difference between clean speech and a stream of clicks. The
same producer/consumer, ring-buffer, callback pattern recurs throughout SDR at the
device boundary, so audio output is a compact, audible lesson in real-time I/O
discipline.

**GopherTrunk** decodes trunked voice, and its decode chain already normalizes to
fixed per-protocol channel rates, so feeding a sound card is a
[resample](/reference/resampler/)-to-48-kHz-plus-[ring-buffer](/reference/ring-buffer/)
problem of exactly this shape. GT is a pure-Go application and does not embed a
GNU Radio audio stack; it interfaces with the platform's audio through Go bindings
to the same underlying PortAudio/ALSA facilities, and applies the standard
discipline — keep the callback lightweight, size the buffer to survive scheduling
jitter, and correct slow clock drift — to avoid underruns during live monitoring.

## Sources

[^pa]: [PortAudio](https://en.wikipedia.org/wiki/PortAudio) — Wikipedia, on the cross-platform audio I/O library and its callback-driven streaming model.
[^alsa]: [Advanced Linux Sound Architecture](https://en.wikipedia.org/wiki/Advanced_Linux_Sound_Architecture) — Wikipedia, on the Linux kernel audio interface and buffer under/overrun (xrun) behavior.
