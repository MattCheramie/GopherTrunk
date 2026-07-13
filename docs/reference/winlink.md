---
slug: winlink
title: Winlink
entry_type: technology
category: amateur-digital
description: "Winlink is a global radio email system that relays messages between ham stations and the internet over HF and VHF data modes such as Pactor, VARA, and ARDOP."
keywords: Winlink, radio email, Winlink 2000, WL2K, Pactor, VARA, ARDOP, RMS gateway, emergency communications, amateur radio email
aka: [Winlink, Winlink 2000, WL2K]
autolink: true
infobox:
  - { label: Type, value: Radio email network }
  - { label: Idea, value: Store-and-forward email over HF/VHF data modes }
  - { label: Examples, value: Pactor, VARA HF/FM, ARDOP, packet gateways }
see_also: [pactor, vara, ax25, packet-radio, ofdm]
cite_urls:
  - https://en.wikipedia.org/wiki/Winlink
---

**Winlink** (formally **Winlink 2000** / **WL2K**) is a worldwide **radio email** system
that lets a licensed amateur send and receive standard email — with attachments — through a
radio link when the internet is unavailable. A user's client connects over the air to a
**Radio Message Server (RMS) gateway**, which is itself internet-connected; the gateway
relays the message into and out of the global email system.[^wiki] The over-the-air hop can
use several data modes — most commonly [Pactor](/reference/pactor/) and
[VARA](/reference/vara/) on HF, or [AX.25](/reference/ax25/) packet on VHF — so Winlink is
best understood as the *network and message layer* riding on top of whichever modem the
conditions allow.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A Winlink client sends email over an HF or VHF data mode to an RMS gateway, which forwards it through the Common Message Servers to the internet email system." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="45" width="80" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor"/>
  <text x="60" y="64" font-size="8.5" fill="currentColor" text-anchor="middle">ham client</text>
  <line x1="100" y1="60" x2="165" y2="60" stroke="currentColor" marker-end="url(#wlar)"/>
  <text x="132" y="52" font-size="7.5" fill="currentColor" text-anchor="middle">Pactor/VARA</text>
  <text x="132" y="74" font-size="7.5" fill="currentColor" text-anchor="middle">RF link</text>
  <rect x="165" y="45" width="90" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor"/>
  <text x="210" y="64" font-size="8.5" fill="currentColor" text-anchor="middle">RMS gateway</text>
  <line x1="255" y1="60" x2="320" y2="60" stroke="currentColor" marker-end="url(#wlar)"/>
  <rect x="320" y="45" width="120" height="30" fill="none" stroke="currentColor"/>
  <text x="380" y="60" font-size="8" fill="currentColor" text-anchor="middle">Common Message</text>
  <text x="380" y="70" font-size="8" fill="currentColor" text-anchor="middle">Servers → internet</text>
</svg>
<figcaption>Winlink is store-and-forward: a client reaches an RMS gateway over an HF/VHF data mode, and redundant Common Message Servers bridge the traffic to ordinary email.</figcaption>
</figure>

## How it works

Winlink is a **store-and-forward** system with a hub-and-spoke topology. Redundant central
**Common Message Servers (CMS)** hold each user's mailbox; a mesh of volunteer **RMS
gateways** provides the RF on-ramps. To send mail, a client (Winlink Express is the standard
one) composes messages offline, then opens a radio session to a reachable gateway, exchanges
compressed and error-controlled data, and disconnects — the gateway forwards to the CMS, and
from there to the wider email system. Incoming mail is queued until the user next connects.

Two features make it practical over poor links. First, messages are **compressed and
protocol-framed** (the B2F/FBB protocol) so a short on-air session moves a lot of text.
Second, the **ARQ data modes** underneath — Pactor's HF ARQ, the OFDM-based
[VARA](/reference/vara/) modems, or the open ARDOP mode — automatically request
retransmission of any block that fails its checksum, so email arrives intact even across a
fading HF path. On VHF/UHF, plain 1200/9600-baud [packet](/reference/packet-radio/) over
AX.25 fills the same role at local range.

## Relevance to SDR

Winlink matters in the wider RF world chiefly for **emergency and off-grid communications**:
served agencies, sailors, and expeditions use it to pass formatted email when infrastructure
is down. On a waterfall its component modes are recognisable — Pactor's structured bursts, or
VARA's [OFDM](/reference/ofdm/) block of subcarriers — and monitoring/decoding them means
decoding the *underlying modem*, since the email payload is compressed and, for Pactor,
carried by a proprietary protocol.

**GopherTrunk does not decode Winlink or its data modes.** Pactor is proprietary, VARA is a
closed OFDM modem, and none of them are digital land-mobile trunking, which is GopherTrunk's
focus. Winlink is documented here as context for the amateur HF/VHF data ecosystem that
surrounds the signals GopherTrunk *does* target, not as a supported protocol.

## Sources

[^wiki]: [Winlink](https://en.wikipedia.org/wiki/Winlink) — Wikipedia, for the Winlink 2000 architecture, RMS/CMS gateways, supported data modes, and emergency-communications role.
