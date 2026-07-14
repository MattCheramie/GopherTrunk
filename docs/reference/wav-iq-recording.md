---
slug: wav-iq-recording
title: WAV IQ recording
entry_type: term
category: sdr-data-streaming
description: "A WAV IQ recording stores baseband IQ in a RIFF/WAVE file as a two-channel PCM stream — I in the left channel, Q in the right — with the sample rate in the header."
keywords: WAV IQ, RIFF WAVE IQ, SDR# recording, SDR++ WAV, two-channel IQ, stereo IQ recording, baseband WAV, auxi chunk, center frequency metadata
aka: [WAV IQ, RIFF IQ recording, stereo IQ WAV]
autolink: true
infobox:
  - { label: Type, value: IQ recording in a RIFF/WAVE container }
  - { label: Layout, value: "2-channel PCM: I = left, Q = right" }
  - { label: Carries, value: Sample rate (header); centre freq (auxi) }
see_also: [iq-file-format, sample-format, sdr-sharp, sdrangel, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/WAV
  - https://airspy.com/directory/
---

A **WAV IQ recording** stores baseband [IQ data](/reference/iq-data/) inside an ordinary RIFF/WAVE
audio container by treating the two quadrature channels as a **stereo pair** — I in the left
channel, Q in the right.[^wav] It is the recording format of SDR# (SDRSharp), SDR++, SDRangel and
similar desktop receivers, and its one real advantage over a bare
[IQ file](/reference/iq-file-format/) is that the WAV header carries the sample rate, so the file is
partly self-describing.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A WAV IQ file has a RIFF/WAVE header holding the sample rate and bit depth, an optional auxi chunk with the centre frequency, then interleaved left-I right-Q PCM samples." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="24" y="46" width="70" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="59" y="60">RIFF/WAVE</text><text x="59" y="72" font-size="7">rate, bits</text>
    <rect x="98" y="46" width="66" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="131" y="60">auxi</text><text x="131" y="72" font-size="7">centre freq</text>
    <rect x="168" y="46" width="52" height="34" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/><text x="194" y="66">L=I₀</text>
    <rect x="220" y="46" width="52" height="34" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="246" y="66">R=Q₀</text>
    <rect x="272" y="46" width="52" height="34" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/><text x="298" y="66">L=I₁</text>
    <rect x="324" y="46" width="52" height="34" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="350" y="66">R=Q₁</text>
    <text x="410" y="66">· · ·</text>
    <text x="240" y="102" font-size="8">stereo PCM frames = interleaved IQ</text>
  </g>
</svg>
<figcaption>A WAV IQ file wraps interleaved left-I / right-Q PCM samples in a RIFF header that carries the sample rate, with centre frequency in an optional auxi chunk.</figcaption>
</figure>

## How it works

A WAVE file is a sequence of chunks. The `fmt ` chunk declares the number of channels, the
[sample rate](/reference/sample-rate/), and the bit depth; the `data` chunk holds the PCM frames.
For an IQ recording the receiver writes **two channels** — a WAV "frame" is therefore one I value
followed by one Q value — so at the byte level the payload is identical to an
[interleaved](/reference/interleaved-iq/) stereo recording. The bit depth is usually 16-bit signed
PCM, giving values in ±32768 that DSP divides by 32768 to normalise to ±1.0, though 8-bit and
32-bit-float WAVs also occur.

Because plain WAVE has no field for RF centre frequency, SDR# introduced an **`auxi`** chunk that
records the tuned frequency and a timestamp. Programs that understand `auxi` can restore the absolute
frequency scale on playback; programs that do not simply skip the unknown chunk and still read the
audio, which is the beauty of the chunked container. A practical limitation inherited from the WAVE
format is the 32-bit size field: a classic RIFF file tops out near 4 GB, which at multi-MS/s IQ rates
is only minutes of recording, so long captures use bare IQ files or extended-WAV variants instead.

## Relevance to SDR

WAV IQ is the format a newcomer meets first, because the mainstream Windows receivers record it by
default and it opens in an audio editor for a quick look. Its self-describing sample rate makes it
friendlier than a raw [sample-format](/reference/sample-format/)-ambiguous blob, and the `auxi`
centre-frequency convention means a shared recording can be retuned correctly by the next person.
The trade-offs are the size ceiling and that not every tool honours `auxi`.

GopherTrunk reads WAV IQ recordings through its offline engine's `-format wav` decoder (also
accepted as `sw16`/`s16`). It parses the RIFF/WAVE header, takes the sample rate from the header —
overriding any `-sample-rate` flag — strips the 44-byte header, and decodes the two-channel 16-bit
PCM as I-then-Q normalised by 32768. This is the same layout GopherTrunk's own IQ writer emits and
that SDRtrunk and SDR++ produce, so a WAV captured in one of those tools replays through
GopherTrunk's production receiver pipeline unchanged. Because a baseband WAV is already channelised
to one signal, GopherTrunk treats it as pre-tuned: auto-tune is rejected for WAV input and a residual
offset is corrected with `-tune-hz` instead.

## In practice

The gotchas are practical ones. First, confirm the bit depth: most SDR software writes 16-bit signed
PCM, but some emits 8-bit or 32-bit float, and a decoder that assumes the wrong width reads garbage.
Second, mind the size ceiling — a wideband recording hits the classic 4 GB RIFF limit in minutes, so
WAV suits narrowband, already-channelised captures far better than raw wideband dumps, for which a bare
[IQ file](/reference/iq-file-format/) is the right container. Third, remember that the absolute frequency
scale only survives if both the recorder and the reader honour the `auxi` chunk; without it the file
still decodes, but at an unknown centre, so the tuned frequency is worth noting in the filename as a
belt-and-braces habit alongside the header's sample rate.

## Sources

[^wav]: [WAV](https://en.wikipedia.org/wiki/WAV) — Wikipedia, on the RIFF/WAVE chunked container, its fmt/data chunks, PCM channel interleaving, and the ~4 GB size limit that constrains long IQ captures.
