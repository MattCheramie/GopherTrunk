---
layout: page
title: "TETRA Scanner: What Works (and GopherTrunk)"
description: "Almost no US consumer scanner decodes TETRA voice — but GopherTrunk does, from a $30 SDR dongle. An honest guide to TETRA, why hardware scanners fall short, and the free software route that works."
keywords: TETRA scanner, TETRA voice decoder, decode TETRA, TETRA SDR, GopherTrunk TETRA, TETRA police scanner, TEA encryption, TETRA trunked radio
permalink: /tetra-scanner/
nav_group: Hardware
affiliate: false
faq:
  - q: "Can a consumer scanner decode TETRA voice?"
    a: "Almost none can. The Uniden and Whistler scanners sold for P25, DMR, and NXDN do not decode TETRA voice. TETRA uses π/4-DQPSK modulation and an ACELP vocoder that mainstream US scanners were never built to handle. The reliable way to hear unencrypted TETRA voice is an SDR running GopherTrunk."
  - q: "How does GopherTrunk decode TETRA voice?"
    a: "GopherTrunk decodes TETRA end-to-end in software: it demodulates the traffic burst, decodes the TCH/S traffic channel, and runs a clean-room ACELP vocoder validated against the ETSI EN 300 395-2 reference codec to produce PCM audio. It runs on a PC with a ~$30 RTL-SDR or a wider-band SDR."
  - q: "Is TETRA used in the United States?"
    a: "Rarely. TETRA is mostly a European and Asian standard, used by public safety, transport, utilities, airports, and critical infrastructure. In the US it appears mainly at some private and critical-infrastructure sites. If you are in Europe, the Middle East, or Asia-Pacific, TETRA is likely the dominant local system."
  - q: "Can GopherTrunk decode encrypted TETRA?"
    a: "No. TETRA's TEA air-interface encryption cannot be decoded by GopherTrunk or any scanner — it is a cryptographic and legal wall. GopherTrunk decodes unencrypted TETRA voice only. Many TETRA networks are encrypted, so check your local system before expecting audio."
  - q: "What hardware do I need to decode TETRA with GopherTrunk?"
    a: "A PC and an SDR. A ~$30 RTL-SDR covers a single TETRA carrier; a wider-band SDR (Airspy and similar) lets you capture more of the network at once. GopherTrunk normalizes to the TETRA channel rate internally, so decoding is rate-invariant across supported SDRs."
  - q: "Why can't my P25/DMR scanner just add TETRA?"
    a: "TETRA is a different air interface — different modulation (π/4-DQPSK), framing, and vocoder (ACELP, not AMBE). Adding it is not a firmware toggle on a P25/DMR scanner; the hardware and DSP were not designed for it. Software radios like GopherTrunk implement the whole chain in code, which is why they can."
---

# TETRA Scanner: What Works (and GopherTrunk)

**If you want to hear TETRA voice, a normal police scanner will not do it — GopherTrunk will, from a $30 SDR dongle.** This is the honest headline. The Uniden and Whistler scanners sold for [P25](/reference/project-25/), [DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) do **not** decode [TETRA](/reference/tetra/) voice, and no US consumer scanner meaningfully does. The route that works is a software-defined radio running GopherTrunk.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Consumer scanners:** essentially **none** decode TETRA voice. **What works:** a $30 [RTL-SDR](/reference/rtl-sdr/) + [GopherTrunk](/downloads.html), which decodes [TETRA](/reference/tetra/) voice end-to-end in software. **Where TETRA lives:** mostly Europe/Asia and critical infrastructure, rarely the US. **Encryption:** [TEA](/reference/tetra-tea/)-encrypted TETRA cannot be decoded by anything — GopherTrunk hears unencrypted TETRA only.
</div>

## The honest truth about TETRA scanners

TETRA (Terrestrial Trunked Radio) is an ETSI digital standard built on **π/4-DQPSK** modulation and an **ACELP** vocoder — a completely different air interface from the P25/DMR/NXDN family that US scanners target.

- **No mainstream US scanner decodes TETRA voice.** The Uniden SDS/BCD series and Whistler TRX line — excellent at P25, DMR, and NXDN — simply do not implement TETRA. Do not buy one expecting TETRA audio.
- **TETRA is mostly not a US standard.** It is dominant across **Europe, the Middle East, and Asia-Pacific** for public safety, rail and transit, airports, utilities, and other critical infrastructure. In the US it shows up mainly at private and critical-infrastructure sites.
- **Software radio is the working path.** Because TETRA's whole chain — demod, framing, vocoder — can be implemented in code, an SDR running the right software decodes it where dedicated scanners cannot.

> **Buying warning.** If a listing claims a handheld police scanner "does TETRA," treat it with heavy skepticism. The realistic way to decode unencrypted TETRA voice today is an SDR + software.

## GopherTrunk decodes TETRA voice — the standout

This is what sets GopherTrunk apart from the scanner shelf: it decodes **TETRA voice end-to-end**, entirely in software.

- **Full signal chain in code.** GopherTrunk demodulates the traffic burst, decodes the **TCH/S** traffic channel, and runs a **clean-room ACELP vocoder** to produce PCM audio you can play and record.
- **Validated against the ETSI reference.** The vocoder is conformance-tested against the **ETSI EN 300 395-2 reference C codec** — the same standard TETRA infrastructure is built to — so the decoded audio is faithful, not a guess.
- **Runs on cheap hardware.** A **~$30 [RTL-SDR](/reference/rtl-sdr/)** captures a single TETRA carrier; a wider-band SDR captures more of the network at once. GopherTrunk normalizes to the TETRA channel rate internally, so decoding behaves the same across supported SDRs.
- **Records everything.** Like every mode GopherTrunk handles, TETRA calls are logged and timestamped to disk, and follow the [control channel](/reference/control-channel/) across the [trunked](/reference/trunked-radio/) network.

## How the options really compare

| Option | Decodes TETRA voice? | Cost | Portable / turnkey | Notes |
|---|---|---|---|---|
| Uniden SDS / BCD scanners | **No** | ~$380–$650 | Yes | Great for P25/DMR/NXDN, not TETRA |
| Whistler TRX-1 / TRX-2 | **No** | ~$550–$600 | Yes | DMR/NXDN included; no TETRA |
| **RTL-SDR + [GopherTrunk](/downloads.html)** | **Yes (unencrypted)** | **~$30** | No (needs PC) | Software ACELP vocoder, records calls |
| Wider-band SDR + GopherTrunk | **Yes (unencrypted)** | ~$150+ | No (needs PC) | Capture more of the network at once |

If your primary interest is P25, DMR, or NXDN rather than TETRA, a dedicated scanner is still a fine choice — see the [best police scanners](/best-police-scanners/) guide. For TETRA specifically, the SDR path is the answer.

## The TEA encryption wall

Many real TETRA networks run **[TEA](/reference/tetra-tea/)** air-interface encryption. Be clear-eyed about it:

- **No scanner and no SDR — including GopherTrunk — decodes TEA-encrypted TETRA.** It is a cryptographic and legal wall, the same principle as [AES on P25/DMR](/police-scanner-encryption/).
- **GopherTrunk decodes unencrypted TETRA voice only.** Where a network (or a given talkgroup) is in the clear, you get audio; where it is TEA-encrypted, no tool recovers it.
- **Check your local system first.** Look up the network before expecting voice — encryption status decides whether any of this produces sound.

## Bottom line

For **TETRA voice**, forget the scanner shelf: essentially **no US consumer scanner decodes it**. The working route is a **$30 [RTL-SDR](/reference/rtl-sdr/) + free [GopherTrunk](/downloads.html)**, which decodes TETRA voice end-to-end through a clean-room ACELP vocoder validated against the ETSI reference codec — and records every call. TETRA is mostly a European/Asian and critical-infrastructure standard, so confirm it's what you actually have. And remember the wall: **[TEA](/reference/tetra-tea/)-encrypted TETRA cannot be decoded by anything** — GopherTrunk hears unencrypted TETRA only. If your area is really P25/DMR/NXDN, start with the [police scanner vs SDR](/police-scanner-vs-sdr/) comparison instead.
