---
slug: sitor
title: SITOR
entry_type: protocol
category: aviation-marine
description: SITOR is an HF maritime radiotelex mode with error correction, built on the CCIR 476 code; SITOR-A uses ARQ handshaking and SITOR-B uses one-way FEC, as in NAVTEX.
keywords: SITOR, simplex teleprinting over radio, radiotelex, CCIR 476, SITOR-A, SITOR-B, ARQ, FEC, NAVTEX, AMTOR, HF maritime
aka: [SITOR, Simplex teleprinting over radio, NAVTEX mode, AMTOR]
autolink: true
infobox:
  - { label: Full name, value: Simplex Teleprinting Over Radio }
  - { label: Code, value: CCIR 476 (7-bit, 4:3 ratio) }
  - { label: Modes, value: "SITOR-A (ARQ), SITOR-B (FEC)" }
see_also: [navtex, rtty, forward-error-correction, marine-vhf, dsc, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/SITOR
  - https://en.wikipedia.org/wiki/Telex#Radioteletype
---

**SITOR** — **Simplex Teleprinting Over Radio** — is an HF maritime radiotelex mode that
adds **error correction** to teleprinter traffic over shortwave.[^wiki] Where plain
[RTTY](/reference/rtty/) sends characters with no way to catch a corrupted bit, SITOR is
built on the 7-bit **CCIR 476** code, a *constant-ratio* code in which every valid
character has exactly **four mark bits and three space bits** — so any bit flipped in
transit breaks the 4:3 ratio and is instantly flagged as an error. It is transmitted as
[frequency-shift keying](/reference/frequency-shift-keying/) on HF.[^telex]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="SITOR-A on the left shows two stations exchanging request and acknowledge handshake arrows; SITOR-B on the right shows one transmitter broadcasting repeated time-diverse blocks one way." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="115" y="18">SITOR-A (ARQ)</text>
    <text x="345" y="18">SITOR-B (FEC)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <rect x="35" y="70" width="22" height="42" rx="3"/>
    <rect x="175" y="70" width="22" height="42" rx="3"/>
    <path d="M58 80 L173 80" marker-end="url(#st_arrow)"/>
    <path d="M173 102 L58 102" marker-end="url(#st_arrow)"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="115" y="76">block →</text>
    <text x="115" y="116">← acknowledge / repeat</text>
    <text x="115" y="150" font-size="8">two-way handshake</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <rect x="290" y="70" width="22" height="42" rx="3"/>
    <path d="M313 78 L430 78" marker-end="url(#st_arrow)"/>
    <path d="M313 91 L430 91" marker-end="url(#st_arrow)"/>
    <path d="M313 104 L430 104" marker-end="url(#st_arrow)"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="372" y="128">same block sent twice,</text>
    <text x="372" y="140">time-spaced (diversity)</text>
    <text x="372" y="162" font-size="8">one-way broadcast</text>
  </g>
  <defs><marker id="st_arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SITOR-A handshakes between two stations and re-requests bad blocks; SITOR-B broadcasts one way, repeating each block time-spaced so a receiver can recover it.</figcaption>
</figure>

## How it works

The foundation is the constant-ratio **CCIR 476** alphabet. By fixing the number of marks
and spaces in every symbol, the code turns simple bit errors into detectable violations:
a receiver that gets a character with the wrong mark/space count knows it is corrupt
without needing a checksum. SITOR then uses that detection differently in its two modes.

**SITOR-A** is an **ARQ** (Automatic Repeat reQuest) mode for a two-way link between a
specific pair of stations. The sender transmits short blocks of characters and pauses; the
receiver checks each block and sends back a brief acknowledgement, and any block that fails
the ratio test is **re-requested** and sent again. The two stations alternate in a tight
handshake, so throughput drops on a poor channel but the delivered text is essentially
error-free. This is the mode used for point-to-point radiotelex traffic.

**SITOR-B** is a **[forward-error-correction](/reference/forward-error-correction/)** mode
for **one-way broadcast**, where there is no return path to request repeats. Instead of
asking again, the transmitter sends **each character twice, separated in time** — a form of
time diversity. A receiver that gets a corrupt copy of a character can fall back on the
delayed second copy, so short fades and noise bursts rarely take out both. Because it needs
no acknowledgement, one transmitter can serve any number of listeners at once.

## Family and relatives

SITOR spawned a family. **[NAVTEX](/reference/navtex/)**, the international maritime safety
broadcast service on **518 kHz**, is simply SITOR-B carrying navigational warnings, weather,
and search-and-rescue notices to any ship in range. **AMTOR** (Amateur Teleprinting Over
Radio) is the amateur-radio adaptation of SITOR, using the same CCIR 476 code and the same
ARQ/FEC pair of modes. All of them descend from the commercial and maritime radiotelex
networks that carried teleprinter traffic over HF before satellite links, and they remain
in service for the robustness that constant-ratio coding gives on a noisy shortwave channel.

## Decoding it with GopherTrunk

SITOR is an HF, narrowband FSK mode, well outside GopherTrunk's VHF/UHF land-mobile
trunking focus, so GopherTrunk does not decode it. It is documented here because an SDR
listener sweeping the marine HF bands will encounter its distinctive sound — the
chirping, paused "chirp-chirp-pause" of SITOR-A's handshake, or the steady stream of
SITOR-B/NAVTEX — and because decoding it needs only a general-coverage receiver and a
soundcard decoder such as those that read NAVTEX on 518 kHz. Recognising SITOR's two modes
by ear (handshaking pauses versus continuous broadcast) tells you immediately whether you
are hearing a two-station radiotelex link or a one-way safety broadcast.

## Sources

[^wiki]: [SITOR](https://en.wikipedia.org/wiki/SITOR) — Wikipedia, for the CCIR 476 constant-ratio code, the SITOR-A ARQ and SITOR-B FEC modes, and the NAVTEX/AMTOR relationships.
[^telex]: [Radioteletype](https://en.wikipedia.org/wiki/Telex#Radioteletype) — Wikipedia, for HF radiotelex context and how error-corrected teleprinter modes work over shortwave.
