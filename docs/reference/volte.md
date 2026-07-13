---
slug: volte
title: VoLTE (Voice over LTE)
entry_type: technology
category: cellular
description: VoLTE carries phone calls as IP packets over the LTE data bearer using an IMS core and the AMR / AMR-WB codecs, replacing circuit-switched fallback with all-IP voice.
keywords: VoLTE, Voice over LTE, IMS, IP Multimedia Subsystem, AMR, AMR-WB, HD Voice, SIP, QCI-1 bearer, VoNR, circuit-switched fallback, ViLTE
aka: [VoLTE, Voice over LTE]
autolink: true
infobox:
  - { label: Type, value: Packet-voice service over 4G }
  - { label: Idea, value: Carry calls as IP over the LTE data bearer via IMS }
  - { label: Codecs, value: "AMR-NB, AMR-WB (HD Voice), EVS" }
see_also: [lte, amr, evs-codec, emergency-call, 5g-nr, base-station-enodeb-gnodeb]
cite_urls:
  - https://en.wikipedia.org/wiki/Voice_over_LTE
  - https://www.gsma.com/futurenetworks/technology/volte/
---

**VoLTE (Voice over LTE)** is the technology that carries telephone calls as IP packets
over the [LTE](/reference/lte/) data connection, using an IP Multimedia Subsystem (IMS)
core and the [AMR](/reference/amr/) family of speech codecs — including wideband
AMR-WB, marketed as "HD Voice."[^wiki] Because early LTE networks were data-only, VoLTE
was what let a 4G phone place a call without dropping back to a legacy 2G/3G
circuit-switched network.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A VoLTE call path: a handset encodes speech with the AMR-WB codec into IP packets carried on a prioritised LTE bearer through the eNodeB to an IMS core using SIP signalling, then to the far-end phone." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="voar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="60" height="34" fill="currentColor" fill-opacity="0.14"/><text x="50" y="66">phone + AMR</text>
    <rect x="130" y="45" width="60" height="34" fill="none"/><text x="160" y="63">eNodeB</text>
    <rect x="240" y="45" width="70" height="34" fill="currentColor" fill-opacity="0.14"/><text x="275" y="63">IMS core</text><text x="275" y="74" font-size="7">SIP</text>
    <rect x="360" y="45" width="60" height="34" fill="none"/><text x="390" y="66">far phone</text>
  </g>
  <g stroke="currentColor" stroke-width="1" marker-end="url(#voar)">
    <line x1="80" y1="62" x2="128" y2="62"/><line x1="190" y1="62" x2="238" y2="62"/><line x1="310" y1="62" x2="358" y2="62"/>
  </g>
  <text x="230" y="110" text-anchor="middle" font-size="8" fill="currentColor">voice as RTP/IP on a prioritised QCI-1 bearer — no circuit switch</text>
</svg>
<figcaption>VoLTE encodes speech with AMR, packetises it as IP on a prioritised LTE bearer, and routes it through an IMS core using SIP signalling.</figcaption>
</figure>

## How it works

VoLTE treats a phone call as just another IP flow, but a **specially prioritised** one.
Signalling — call setup, ringing, teardown — uses SIP through the operator's **IMS (IP
Multimedia Subsystem)**, the standardised control core for packet-based
telephony. The speech itself is encoded by an [AMR](/reference/amr/) codec (narrowband
AMR-NB, wideband AMR-WB for HD Voice, or the later EVS codec), packetised as RTP, and
carried on a **dedicated bearer** with a guaranteed bit rate and low latency (QCI-1 in
LTE's quality-of-service scheme). That guaranteed bearer is what keeps a packet call
sounding like a circuit call: it protects the voice packets from the bursty best-effort
data sharing the same [LTE](/reference/lte/) carrier.

Before VoLTE, LTE phones used **Circuit-Switched Fallback (CSFB)**, momentarily dropping
to 2G/3G to make or take a call. VoLTE removes that fallback, keeping the handset on 4G,
which cuts call-setup time, frees the older networks, and enables wideband audio. The
same architecture extends directly to **VoNR (Voice over NR)** on
[5G NR](/reference/5g-nr/), and to Wi-Fi Calling, which reuses the identical IMS core
over any IP access.

## Relevance to SDR

VoLTE rides entirely inside the encrypted LTE user plane, so from a receiver's point of
view it is indistinguishable from other LTE traffic — there is no separate over-the-air
voice channel to tune the way there is on an analog FM repeater or a
[P25](/reference/p25-phase-1/) system. VoLTE is relevant to SDR mainly as the reason
modern cellular voice left the analyzable, narrowband circuit-switched world entirely:
the [AMR](/reference/amr/) and EVS vocoders it uses are close cousins of the
CELP-family coders in land-mobile digital voice, but they are wrapped in IMS signalling
and network-layer encryption.

**GopherTrunk does not decode VoLTE.** It is a land-mobile trunking scanner, not a
cellular interceptor, and cellular voice is both out of scope for its narrowband air
interfaces and protected by carrier-grade encryption. VoLTE is documented here to
explain how 4G/5G telephony works and why it does not appear on a scanner.

## Sources

[^wiki]: [Voice over LTE](https://en.wikipedia.org/wiki/Voice_over_LTE) — Wikipedia, for the IMS-based architecture, AMR/AMR-WB codecs, and the move away from circuit-switched fallback.
[^gsma]: [VoLTE](https://www.gsma.com/futurenetworks/technology/volte/) — GSMA, for the industry profile of VoLTE and HD Voice deployment.
