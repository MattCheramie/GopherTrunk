---
slug: remote-keyless-entry
title: Remote keyless entry (RKE)
entry_type: protocol
category: wireless-data-iot
description: "Remote keyless entry is the 315/433 MHz OOK key-fob link that locks and unlocks vehicles, using a rolling code so each captured transmission cannot simply be replayed."
keywords: remote keyless entry, RKE, key fob, 315 MHz, 433 MHz, OOK, ASK, rolling code, KeeLoq, hopping code, PWM, Manchester, replay attack
aka: [RKE, "remote keyless entry", "key fob"]
autolink: true
infobox:
  - { label: Type, value: Vehicle key-fob command link }
  - { label: Bands, value: "315 MHz (US), 433.92 MHz (EU) ISM" }
  - { label: Modulation, value: OOK/ASK (some FSK), PWM/Manchester coded }
  - { label: Security, value: Rolling (hopping) code — e.g. KeeLoq }
  - { label: Direction, value: One-way fob → vehicle (RKE) }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [on-off-keying, rolling-code, tpms, frequency-shift-keying, amplitude-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Remote_keyless_system
  - https://en.wikipedia.org/wiki/Rolling_code
---

**Remote keyless entry** (**RKE**) is the short-range radio link from a car **key fob** to
the vehicle that locks, unlocks, and opens the trunk at the press of a button.[^wiki] It is
a one-way transmission in a sub-GHz ISM band — **315 MHz** in North America, **433.92 MHz**
in Europe — using simple on-off keying, but with a crucial security twist: instead of
sending a fixed code, each press sends a different [rolling code](/reference/rolling-code/),
so an attacker who records one transmission cannot simply replay it later.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A key fob sends an on-off-keyed packet to a car; the packet contains a fixed serial plus a rolling code counter that advances each press, so a replayed capture is rejected." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rk_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="55" width="46" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/><text x="53" y="112">fob</text>
    <rect x="384" y="55" width="46" height="40" fill="none" stroke="currentColor"/><text x="407" y="112">car</text>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <path d="M120 78 L138 78 L138 50 L152 50 L152 78 L165 78 L165 50 L173 50 L173 78 L190 78 L190 50 L205 50 L205 78 L225 78 L225 50 L235 50 L235 78 L255 78 L255 50 L270 50 L270 78 L290 78 L290 50 L300 50 L300 78 L318 78"/>
  </g>
  <line x1="325" y1="70" x2="382" y2="70" stroke="currentColor" stroke-opacity="0.8" marker-end="url(#rk_ar)"/>
  <text x="222" y="100" text-anchor="middle" font-size="8.5" fill="currentColor">OOK packet: serial · button · rolling counter</text>
  <text x="222" y="128" text-anchor="middle" font-size="8.5" fill="currentColor">counter advances every press → replay rejected</text>
</svg>
<figcaption>An RKE fob sends an OOK packet holding its serial and a rolling counter that increments each press; the car accepts only codes ahead of the last one it saw.</figcaption>
</figure>

## Overview

Pressing an RKE button transmits a short packet containing the fob's serial number, the
button pressed, and a counter (or cryptographically hopping) value. The vehicle keeps the
last accepted counter and accepts only codes within a forward window of it, so an old,
recorded packet is stale and ignored. Classic implementations use Microchip's **KeeLoq**
hopping-code cipher; newer systems use stronger keyed algorithms.

## Technical characteristics

| Property | Value |
|----------|-------|
| Bands | 315 MHz (US), 433.92 MHz (EU) ISM |
| Modulation | [OOK](/reference/on-off-keying/)/[ASK](/reference/amplitude-shift-keying/); some [FSK](/reference/frequency-shift-keying/) |
| Line coding | PWM or Manchester |
| Security | [Rolling code](/reference/rolling-code/) (e.g. KeeLoq) |
| Direction | One-way fob → vehicle |
| Payload | Fob serial + button flags + rolling counter |

RKE is distinct from *passive* keyless entry (PKE) and immobilizer transponders, which add a
bidirectional LF challenge-response so the car unlocks or starts just by proximity.

## History

Remote keyless entry appeared on cars in the 1980s and became near-universal in the 1990s.
Early systems sent fixed codes and were trivially replayable; the rolling-code approach —
popularized by KeeLoq — was adopted to defeat simple record-and-replay attacks.[^roll]
Researchers have since shown weaknesses in some implementations, driving a move to stronger
ciphers.

## Deployment

RKE fobs are ubiquitous on passenger vehicles and share the 315/433 MHz ISM bands with
[TPMS](/reference/tpms/) sensors, garage remotes, and other short-range devices — a busy
slice of spectrum for anyone surveying it.

## Decoding it with GopherTrunk

RKE is out of scope for GopherTrunk, which decodes trunked land-mobile voice, not key-fob
control packets. The raw OOK bursts can be *observed* with a general
[SDR](/reference/software-defined-radio/) and generic 433 MHz decoders, but the rolling code
means a captured packet is not reusable, and GopherTrunk implements neither the fob framings
nor any of this. It is included here for context on the sub-GHz device signals a scanner
operator encounters, not as a target GopherTrunk decodes.

## Sources

[^wiki]: [Remote keyless system](https://en.wikipedia.org/wiki/Remote_keyless_system) — Wikipedia, for the RKE fob link, bands, and relationship to passive keyless entry.
[^roll]: [Rolling code](https://en.wikipedia.org/wiki/Rolling_code) — Wikipedia, for the hopping-code scheme (KeeLoq) that defeats replay of captured transmissions.
