---
slug: evs-codec
title: Enhanced Voice Services (EVS)
entry_type: technology
category: voice-coding
description: "EVS (Enhanced Voice Services) is 3GPP's super-wideband speech and audio codec for LTE/5G VoLTE, spanning narrowband to full-band with an AMR-WB-compatible mode for backward interworking."
keywords: EVS, Enhanced Voice Services, 3GPP codec, super-wideband, full-band, VoLTE, VoNR, HD Voice Plus, AMR-WB IO, ACELP, MDCT, packet loss concealment, channel-aware mode
aka: [EVS, Enhanced Voice Services]
autolink: true
infobox:
  - { label: Type, value: 3GPP super-wideband speech/audio codec }
  - { label: Bandwidth, value: "NB / WB / SWB / FB (up to 20 kHz)" }
  - { label: Rates, value: "5.9–128 kbps, incl. AMR-WB IO" }
see_also: [amr, opus-codec, volte, acelp]
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Voice_Services
  - https://www.3gpp.org/technologies/evs
---

**EVS** (**Enhanced Voice Services**) is the 3GPP speech and audio codec introduced with LTE to
carry telephone calls at far higher quality than earlier cellular vocoders.[^wiki] It is the
codec behind "HD Voice+" and is used for [VoLTE](/reference/volte/) and 5G VoNR calls. EVS spans
narrowband all the way to *full-band* (up to 20 kHz audio), degrades gracefully under packet
loss, and includes an [AMR](/reference/amr/)-WB interoperable mode so it can hand calls off to
legacy networks without transcoding.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A bar showing audio bandwidth from narrowband through wideband and super-wideband to full-band, with EVS spanning the whole range while older AMR codecs cover only the lower portion." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <line x1="30" y1="70" x2="440" y2="70" stroke="currentColor" stroke-width="1"/>
    <g stroke="currentColor" stroke-width="1"><line x1="130" y1="66" x2="130" y2="74"/><line x1="230" y1="66" x2="230" y2="74"/><line x1="330" y1="66" x2="330" y2="74"/></g>
    <text x="80" y="86">NB 4k</text><text x="180" y="86">WB 8k</text><text x="280" y="86">SWB 16k</text><text x="385" y="86">FB 20k</text>
    <rect x="30" y="34" width="200" height="14" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="130" y="30">AMR-NB / AMR-WB</text>
    <rect x="30" y="50" width="410" height="14" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="235" y="61" font-size="8">EVS (NB → FB)</text>
  </g>
</svg>
<figcaption>EVS covers audio bandwidths from narrowband to full-band in one codec, extending well beyond the range that AMR reaches.</figcaption>
</figure>

## How it works

EVS is a switched codec: it chooses, per frame, between an
[ACELP](/reference/acelp/)-based coder for speech and an MDCT transform coder for music and
generic audio, so a single codec handles both cleanly. It supports four audio bandwidths —
narrowband, wideband, super-wideband, and full-band — at bitrates from 5.9 kbps up to 128 kbps,
with a variable-rate mode that can drop to comfort noise during silence. Backward compatibility
is built in through an **AMR-WB IO** (interoperable) mode that is bit-exact with AMR-WB, letting a
5G/LTE call cross into a 3G network without a lossy transcode.

Two robustness features set EVS apart. Its packet-loss concealment reconstructs missing frames
well enough that occasional loss is inaudible, and a **channel-aware mode** proactively embeds
partial redundancy of earlier frames so the decoder can recover a lost packet from a later one —
essentially source-integrated forward error correction tuned for VoIP jitter and loss.

## Relevance to SDR

Like AMR, EVS lives inside the cellular packet core (VoLTE/VoNR) rather than on open land-mobile
channels, so it is not something a trunking scanner decodes. It appears in this guide as the
current state of the art in *cellular* voice coding and as the natural comparison point to
open-source VoIP codecs such as [Opus](/reference/opus-codec/), which occupy a similar
speech-plus-audio, loss-resilient design space. GopherTrunk does not implement EVS; its decode
chain is land-mobile digital voice, whose vocoders are the MBE family and ACELP rather than EVS.

## Sources

[^wiki]: [Enhanced Voice Services](https://en.wikipedia.org/wiki/Enhanced_Voice_Services) — Wikipedia, on EVS bandwidths, bitrates, the ACELP/MDCT switch, AMR-WB IO, and channel-aware mode.
