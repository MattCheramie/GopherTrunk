---
slug: dpmr-dstar-ysf
title: "dPMR, D-STAR & System Fusion"
description: dPMR, D-STAR, and Yaesu System Fusion explained — the lighter digital modes you meet on the air, mostly not trunked, with narrowband 4FSK, GMSK, and C4FM voice.
keywords: dPMR, D-STAR, System Fusion, C4FM, GMSK, 4FSK FDMA, 6.25 kHz, amateur radio digital, AMS, DV voice, M17, narrowband digital voice
level: beginner
status: full
prereq:
  - nxdn
  - voice-to-bits-vocoders
faq:
  - q: What is dPMR?
    a: dPMR (digital Private Mobile Radio) is an ETSI narrowband standard using 6.25 kHz FDMA with 4FSK modulation. It is very close to NXDN in footprint and is used mostly for conventional, light commercial radio. It is not usually a trunked system, but you'll meet it on business and utility channels.
  - q: What is D-STAR?
    a: D-STAR is a digital amateur-radio mode developed with Icom. It uses GMSK modulation at 4800 bps carrying a single voice stream (DV) plus slow data. It is a ham mode rather than a trunked public-safety system, popular for repeaters and internet-linked networks.
  - q: What is Yaesu System Fusion?
    a: System Fusion is Yaesu's amateur digital mode, built on C4FM 4FSK. Its Automatic Mode Select (AMS) feature lets a repeater handle both digital and analog signals automatically, so a Fusion repeater can pass either. Like D-STAR, it is an amateur mode, not a trunked system.
  - q: Are these modes trunked?
    a: Mostly no. dPMR, D-STAR, and System Fusion are predominantly conventional — fixed channels and repeaters rather than control-channel trunking. You'll still meet them on the air, and GopherTrunk can decode their voice even though there's usually no control channel to follow.
gophertrunk_links:
  - title: M17
    url: /m17.html
    note: GopherTrunk's support for the open M17 digital voice mode.
  - title: Vocoders
    url: /vocoders.html
    note: how GopherTrunk decodes the voice these modes carry.
---

# dPMR, D-STAR & System Fusion

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A round-up of the **lighter digital modes** you'll meet that are mostly **not trunked**.
**dPMR** is an **ETSI 6.25 kHz 4FSK FDMA** standard, close to NXDN, used for light
commercial radio. **D-STAR** is an **amateur** mode (Icom) using **GMSK at 4800 bps** with a
single **DV** voice stream plus slow data. **Yaesu System Fusion** is amateur **C4FM** that
can **mix digital and analog** via **AMS**. These are mostly **conventional**, but you'll
still hear them — and **GopherTrunk can decode their voice**. The open **[M17](/m17.html)**
mode rounds out the picture.

</div>

The systems earlier in this module were full trunked networks. These last few are different:
they're mostly **conventional** digital modes — fixed channels and repeaters — that you'll
nonetheless run across on the air. None usually has a [control channel](/learn/digital-trunking/the-control-channel/)
to follow, but each carries digital voice GopherTrunk can decode.

## dPMR — NXDN's close cousin

**dPMR** (digital Private Mobile Radio) is an **ETSI** standard that uses **6.25 kHz FDMA**
with **4FSK** modulation. If that sounds almost identical to [NXDN](/learn/digital-trunking/nxdn/), it is — the two
share the same narrowband 4FSK footprint and are easy to confuse on the
[waterfall](/learn/rf-sdr/fft-and-waterfall/). They're separate standards, but their on-air
fingerprints are nearly the same width and shape.

dPMR is used mostly for **conventional, light commercial** radio — small business fleets and
similar — rather than big trunked networks. Treat it as the European sibling of NXDN that you
identify the same way: very narrow, 4FSK, confirmed by the decoder.

## D-STAR — the amateur GMSK mode

**D-STAR** (Digital Smart Technologies for Amateur Radio) is a **ham-radio** mode developed
with **Icom**. Its traits:

- **GMSK** modulation — Gaussian Minimum Shift Keying — at **4800 bps**. This is a different
  scheme from the 4FSK of NXDN/dPMR and the C4FM of Fusion.
- A **single voice stream** (called **DV**, digital voice) plus a **slow data** channel
  riding alongside for things like callsigns and short messages.
- Strong support for **internet-linked** repeater networks, which made it popular for
  long-distance amateur contacts.

It is an amateur mode, so you won't find public-safety trunking here — just ham operators on
repeaters and simplex.

## Yaesu System Fusion — C4FM that mixes modes

**Yaesu System Fusion** is Yaesu's amateur digital mode, built on **C4FM** — the same
four-level FSK family you met in [digital modulation for
trunking](/learn/digital-trunking/digital-modulation-for-trunking/). Its standout feature is **AMS**, *Automatic Mode
Select*: a Fusion repeater can **automatically handle both digital and analog** signals,
switching as needed. That lets a single repeater serve digital Fusion users and traditional
analog FM users without anyone choosing a mode by hand — a pragmatic bridge for clubs moving
to digital gradually.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 160" role="img" aria-label="Three light digital modes side by side: dPMR using 4FSK FDMA, D-STAR using GMSK, and System Fusion using C4FM with automatic analog and digital handling." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="90" y="20" text-anchor="middle" font-weight="600">dPMR</text>
    <rect x="30" y="32" width="120" height="60" rx="6" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.2"/>
    <text x="90" y="56" text-anchor="middle" font-size="9">4FSK FDMA</text>
    <text x="90" y="72" text-anchor="middle" font-size="9">6.25 kHz</text>
    <text x="90" y="88" text-anchor="middle" font-size="8">light commercial</text>
    <text x="260" y="20" text-anchor="middle" font-weight="600">D-STAR</text>
    <rect x="200" y="32" width="120" height="60" rx="6" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.2"/>
    <text x="260" y="56" text-anchor="middle" font-size="9">GMSK 4800 bps</text>
    <text x="260" y="72" text-anchor="middle" font-size="9">DV + slow data</text>
    <text x="260" y="88" text-anchor="middle" font-size="8">amateur</text>
    <text x="430" y="20" text-anchor="middle" font-weight="600">System Fusion</text>
    <rect x="370" y="32" width="120" height="60" rx="6" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.2"/>
    <text x="430" y="56" text-anchor="middle" font-size="9">C4FM</text>
    <text x="430" y="72" text-anchor="middle" font-size="9">AMS: digital + analog</text>
    <text x="430" y="88" text-anchor="middle" font-size="8">amateur</text>
    <text x="260" y="125" text-anchor="middle" font-size="9">All mostly conventional — no control channel to follow.</text>
  </g>
</svg>
<figcaption>Three lighter digital modes: dPMR (narrowband 4FSK), D-STAR (GMSK), and System Fusion (C4FM with automatic analog/digital handling) — mostly conventional, not trunked.</figcaption>
</figure>

## Not trunked, but still on the air

The common thread is that none of these is normally a **trunked** system. There's usually no
control channel granting voice channels — they're [conventional](/learn/digital-trunking/conventional-vs-trunked/)
modes on fixed frequencies and repeaters. You meet them not by hunting a control channel but
by tuning the channel directly. The good news is that **GopherTrunk can decode the voice**
they carry once you're on the right frequency, even though there's no trunking to track.

One more worth knowing: **[M17](/m17.html)** is a fully **open**, community-developed digital
voice mode — free of the patent-encumbered [vocoders](/learn/rf-sdr/vocoders/) the others rely
on. GopherTrunk supports it, and it's a refreshing contrast to the proprietary codecs
elsewhere in this path.

## Recap

- **dPMR** is an **ETSI 6.25 kHz 4FSK FDMA** standard, nearly a twin of NXDN, for light
  commercial use.
- **D-STAR** is an **amateur** mode using **GMSK at 4800 bps** with a single **DV** voice
  stream plus slow data.
- **Yaesu System Fusion** is amateur **C4FM**, and its **AMS** feature mixes digital and
  analog on one repeater.
- These are mostly **conventional** — no control channel — but **GopherTrunk decodes their
  voice**.
- The open **M17** mode is a patent-free alternative worth knowing.

That wraps the survey of other digital systems. The next module gets hands-on, starting with
[identifying the system](/learn/digital-trunking/identifying-the-system/) you're actually looking at.

You can also revisit the broader catalogue of non-trunked transmissions in [other
signals](/learn/rf-sdr/other-signals/).
