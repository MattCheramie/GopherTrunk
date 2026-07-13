---
slug: nxdn
title: NXDN
entry_type: protocol
category: land-mobile-trunking
description: NXDN is a narrowband digital land-mobile radio standard by Kenwood and Icom, using 4FSK in 6.25 or 12.5 kHz channels with the AMBE+2 vocoder, in conventional and trunked forms.
keywords: NXDN, NEXEDGE, IDAS, narrowband digital, 6.25 kHz, Kenwood, Icom, 4FSK, C4FM, AMBE+2, RAN, FDMA trunking
aka: [NXDN, NEXEDGE, IDAS]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Developers, value: Kenwood & Icom }
  - { label: Access, value: FDMA (conventional or trunked) }
  - { label: Channel spacing, value: 6.25 kHz or 12.5 kHz }
  - { label: Modulation, value: 4FSK (2400/4800 baud) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [dpmr, ran-nxdn, four-fsk, frequency-shift-keying, ambe-plus-2, fdma, trunked-radio, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://www.nxdn-forum.com/
---

**NXDN** is a **narrowband** digital land-mobile radio standard jointly developed by
**Kenwood** (marketed as NEXEDGE) and **Icom** (marketed as IDAS). It carries voice and
data using [4FSK](/reference/four-fsk/) modulation in channels as narrow as **6.25 kHz**
(or a 12.5 kHz variant) and encodes speech with the
[AMBE+2](/reference/ambe-plus-2/) [vocoder](/reference/vocoder/).[^wiki] It exists in
both conventional (single-repeater) and trunked forms, the latter coordinated by a
digital [control channel](/reference/control-channel/) — the trunking layer is sometimes
referred to by its **RAN** (Radio Access Number) signalling, covered in
[RAN / NXDN trunking](/reference/ran-nxdn/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Narrow 6.25 kHz FDMA channels for NXDN, one call per channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_nxdn)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (6.25 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_nxdn" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>NXDN packs calls into very narrow 6.25 kHz (or 12.5 kHz) FDMA channels — one conversation per RF channel.</figcaption>
</figure>

## Overview

NXDN was designed around **spectrum efficiency**. Its 6.25 kHz channel is half the
width of a typical [DMR](/reference/dmr/) or [P25 Phase 1](/reference/p25-phase-1/)
channel, letting a licensee fit twice as many talk paths into the same slice of band —
an appealing property as regulators pushed VHF/UHF land mobile toward narrowbanding. It
is a pure [FDMA](/reference/fdma/) design: each conversation occupies its own RF channel
rather than a time slot, which keeps radios and repeaters simple compared with the
two-slot [TDMA](/reference/tdma/) approach DMR uses to reach the same efficiency. The
6.25 kHz mode runs at 2400 baud (about 4800 bit/s gross with two bits per symbol); the
wider 12.5 kHz "very-wide" mode doubles this to 4800 baud. Beyond voice, NXDN carries
short data messages, GPS location, text, and paging, and supports both clear and
scrambled/encrypted traffic.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Channel spacing | 6.25 kHz (2400 baud) or 12.5 kHz (4800 baud) |
| Modulation | [4FSK](/reference/four-fsk/) (a filtered FSK closely related to [C4FM](/reference/c4fm/)) |
| Symbol rate | 2400 sym/s (6.25 kHz) or 4800 sym/s (12.5 kHz) |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |
| Error control | Convolutional / trellis coding with CRC |
| Trunking | Optional, via digital control channel ([RAN](/reference/ran-nxdn/) signalling) |
| Voice/data | Digital voice, short data, GPS, text, paging |

## History

NXDN emerged in the late 2000s when Kenwood and Icom agreed on a **common narrowband
digital air interface**, published as the NXDN Technical Specifications through the NXDN
Forum.[^wiki][^forum] The two vendors ship interoperable products under their own brand
names — Kenwood's **NEXEDGE** and Icom's **IDAS** — which is why the same on-air
protocol appears under three names. The standard was positioned as a low-cost,
spectrum-efficient competitor to DMR and P25 for commercial users, arriving alongside
the broader industry move to 6.25 kHz-equivalent efficiency mandated by regulators such
as the U.S. FCC.

## Deployment

NXDN is common in **business, transport, utility, and campus fleets** worldwide, and it
has a following among licensed amateur and public-service users who value its narrow
footprint. It sees particular use where regulators reward true 6.25 kHz channelisation
or where an operator wants digital voice without the timing complexity of TDMA.
Networked NXDN systems can link repeaters over IP, and the trunked variant supports
multi-site operation with talkgroups, radio IDs, and roaming — the same feature
vocabulary as other [trunked-radio](/reference/trunked-radio/) systems, just carried in
NXDN's compact frames.

## Decoding it with GopherTrunk

GopherTrunk demodulates the NXDN 4FSK air interface, recovers the frame structure, and
**follows trunked NXDN control channels** to track [channel grants](/reference/channel-grant/)
onto voice channels. Because the modulation is a filtered 4-level FSK, it shares much of
the receiver front end (symbol timing, C4FM-style discriminator, frame sync) with the
[DMR](/reference/dmr/) and [P25](/reference/p25-phase-1/) decoders. Clear and scrambled
NXDN voice can be decoded where an AMBE+2 vocoder is available; keyed encryption is not
recoverable, consistent with GopherTrunk's receiver-only, no-key-cracking design. See
the [Status](/status.html) page for the current state of NXDN conventional and trunked
support.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, for the Kenwood/Icom narrowband 4FSK air interface, the 6.25/12.5 kHz channel widths, and the AMBE+2 vocoder.
[^forum]: [NXDN Forum](https://www.nxdn-forum.com/) — the industry body that publishes the NXDN Technical Specifications, for the common air interface, RAN trunking signalling, and NEXEDGE/IDAS branding.
