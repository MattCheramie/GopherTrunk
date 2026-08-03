---
layout: page
title: "How Police Scanners Work (2026 Explainer)"
description: "How police scanners actually work in 2026 — scanning, conventional vs trunked systems, control channels and talkgroups, analog vs digital P25/DMR/NXDN, simulcast, and why trunk-tracking matters."
keywords: how police scanners work, how do police scanners work, trunked radio explained, control channel, talkgroup, P25 scanner, digital scanner explained, trunk tracking scanner
permalink: /how-police-scanners-work/
nav_group: Hardware
affiliate: true
faq:
  - q: "How does a police scanner work?"
    a: "A scanner is a wideband radio receiver that rapidly steps through a list of frequencies (or follows a trunked system's control channel), stops when it detects transmitted audio, plays it, and resumes scanning. Digital scanners also decode the P25, DMR, or NXDN bitstream back into voice."
  - q: "What is a control channel on a scanner?"
    a: "On a trunked system, one frequency continuously broadcasts data telling every radio which frequency a given talkgroup's next transmission will use. A trunk-tracking scanner reads that control channel and jumps to the assigned voice frequency automatically, so you follow a conversation across many channels."
  - q: "What is a talkgroup?"
    a: "A talkgroup is a virtual channel on a trunked system — a numeric ID that groups users (say, a police district) together. You program talkgroups, not fixed frequencies, because the system assigns a different physical frequency to each call on the fly."
  - q: "Do I need a digital scanner or an analog one?"
    a: "It depends on your local agencies. If police, fire, or EMS use P25, DMR, or NXDN you need a digital scanner. If they are still on conventional FM you can use a cheap analog scanner. Check RadioReference for your county before buying."
  - q: "Why won't my old scanner pick up the police anymore?"
    a: "Almost always because the agency moved to a trunked and/or digital system your radio can't follow. An old analog non-trunking scanner can't track a P25 trunked system, and it certainly can't decode digital voice — you need a trunk-tracking digital scanner or an SDR running GopherTrunk."
  - q: "Can a scanner hear encrypted police?"
    a: "No. If a talkgroup is AES-encrypted, no consumer scanner and no software-defined radio can decode it. You will see the talkgroup activity but hear silence. That is a cryptographic and legal wall, not a hardware limitation you can pay your way around."
---

# How Police Scanners Work (2026 Explainer)

**A police scanner is a fast, wideband radio receiver that hunts across many
frequencies, stops on active ones, and — on modern systems — decodes a digital
bitstream back into voice.** Understanding the four moving parts (scanning,
trunking, the [control channel](/reference/control-channel/), and digital voice)
tells you exactly which scanner you need and why an old radio may have gone quiet.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A scanner rapidly steps through frequencies and stops on activity. **Conventional**
systems put each channel on a fixed frequency; **[trunked](/reference/trunked-radio/)**
systems share a pool of frequencies and use a **[control channel](/reference/control-channel/)**
to assign them per call. You follow **[talkgroups](/reference/talkgroup/)**, not
frequencies. **Digital** voice ([P25](/reference/project-25/),
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/)) needs a digital scanner.
**[Simulcast](/reference/simulcast/)** systems demand the best front ends. **Nothing
here decodes [encryption](/police-scanner-encryption/).**
</div>

## Step one: scanning

The name says it. A scanner holds a list of frequencies and **steps through them
many times per second**, checking each for a signal. When it finds one — squelch
opens — it **stops, plays the audio, and resumes** when the transmission ends.
That's the whole trick on a simple conventional radio: a receiver plus an automatic
channel-hopping loop, so you don't have to twist a dial waiting for your fire
department to key up.

- **Steps per second** determine how likely you are to catch a short transmission.
- **Squelch** decides when a channel counts as "active" versus noise.
- **Lockouts and priority** let you skip dead channels and jump to important ones.

This works perfectly for **conventional** systems — but most large agencies no
longer use them.

## Conventional vs trunked

The single most important concept in modern scanning is **trunking**.

| | Conventional | [Trunked](/reference/trunked-radio/) |
|---|---|---|
| **Channel = ** | A fixed frequency | A [talkgroup](/reference/talkgroup/) (an ID) |
| **How you tune** | Program the frequency | Program the talkgroup + system |
| **Frequency use** | One channel, one frequency, always | A shared pool, assigned per call |
| **Needs a scanner that…** | Just receives FM/digital | **Trunk-tracks** the [control channel](/reference/control-channel/) |
| **Typical users** | Small towns, some fire/EMS, ham, marine | Cities, counties, statewide public safety |

On a **conventional** system, the police dispatch channel is always, say, 154.400
MHz. Easy: program that frequency and listen.

On a **trunked** system, there is no fixed police frequency. The agency owns a
handful of frequencies (a "site") and a computer hands one out to each call for the
few seconds it lasts, then reclaims it. Dispatch might be on 851.0125 MHz for one
call and 852.4375 MHz thirty seconds later. A plain frequency scanner is lost — it
would catch fragments of unrelated conversations. This is the usual reason an old
scanner "stopped working": the agency went trunked.

## The control channel and talkgroups

Trunking is coordinated by a **[control channel](/reference/control-channel/)** — one
frequency that does nothing but broadcast a constant stream of data. It announces,
in effect: *"[Talkgroup](/reference/talkgroup/) 1051 (Police District 3), your next
transmission is on frequency 852.4375."*

A **trunk-tracking scanner** listens to that control channel, reads the assignment,
and **instantly jumps to the voice frequency** to play the call — then hops back to
the control channel to await the next one. To you it sounds like one continuous
conversation; under the hood the scanner is chasing it across a dozen frequencies.

- **You program talkgroups, not frequencies.** Pick the districts/agencies you want
  by their numeric talkgroup IDs.
- **The scanner does the chasing.** It decodes the control channel and follows
  grants in real time.
- **One system, many talkgroups.** A single site carries police, fire, EMS, public
  works, and schools — you choose which to hear.

> **This is why "trunk tracking" is the feature that matters.** A radio that can't
> decode a [control channel](/reference/control-channel/) simply cannot follow a
> trunked agency, no matter how sensitive it is.

## Analog vs digital voice

Trunking decides *how channels are assigned*; the **modulation** decides *how the
voice itself is carried*.

- **Analog (FM).** The voice rides directly on the carrier, like an old FM walkie.
  Any scanner with the right band can hear it.
- **Digital.** The voice is sampled, compressed by a vocoder, and sent as a
  **bitstream**. A digital scanner must decode that bitstream back to audio. The
  common public-safety formats:
  - **[P25](/reference/project-25/)** — the dominant US public-safety standard.
    Phase I is FDMA; **[Phase II](/reference/p25-phase-2/)** is 2-slot TDMA and now
    very common in metros.
  - **[DMR](/reference/dmr/)** — widespread in business, some public safety, and ham.
  - **[NXDN](/reference/nxdn/)** — narrowband digital used by some agencies and utilities.

An analog-only scanner hears **none** of these digital modes — you'll see activity
but get only a digital "warble." If your agencies went digital, you need a scanner
(or [SDR](/reference/software-defined-radio/) + software) that decodes the specific
mode they use.

## Simulcast: the hard case

Big trunked systems often broadcast the **same signal from many towers at once** on
the same frequency to blanket a region. That's **[simulcast](/reference/simulcast/)**,
and where two towers' signals overlap they arrive slightly out of step and **distort
each other**. A cheap receiver garbles the audio; a great one recovers it.

- **Best-in-class hardware:** Uniden's **True I/Q** front end in the
  [SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/) is the
  gold standard on brutal simulcast.
- **Software approach:** [GopherTrunk](/downloads.html) applies a CMA equalizer that
  clears up many simulcast systems from a cheap dongle — see
  [scanner vs SDR](/police-scanner-vs-sdr/).

## Why you need trunk-tracking (and what to buy)

Put it together: to hear a modern agency you almost always need a scanner that is
**both** trunk-tracking **and** digital in the right mode.

| Your agencies are… | You need… | Example |
|---|---|---|
| Conventional analog | Any analog scanner | [BC125AT](/reference/uniden-bc125at/) (~$110) |
| Trunked, but analog voice | Trunk-tracking analog | [BCT15X](/reference/uniden-bearcat-bct15x/) |
| Trunked P25 (no simulcast) | Digital trunk-tracker | [BCD436HP](/reference/uniden-bcd436hp/) (~$520) |
| Trunked P25 **simulcast** | True I/Q digital | [SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/) |
| Any of the above + a PC | [SDR](/reference/rtl-sdr/) + software | [GopherTrunk](/downloads.html) (~$30 dongle) |

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check BCD436HP price on Amazon &rarr;</a>

Not sure what your county runs? That's the first step in the
[buying guide](/scanner-buying-guide/): look yourself up on
[RadioReference](https://www.radioreference.com/) before spending a dollar.

## The encryption wall

One honest caveat that no product can wish away: when an agency turns on **AES
encryption**, the [talkgroup](/reference/talkgroup/) still appears but the audio is
**silence**. **No consumer scanner and no software-defined radio — including
GopherTrunk — can decode it.** It is a cryptographic wall, and defeating it without
authorization is a federal offense (18 U.S.C. §2512). Any product claiming to
"unlock encrypted police" is a scam. Read
[police scanner encryption explained](/police-scanner-encryption/) and buy based on
what's **still in the clear** — which in most areas still includes plenty of
dispatch and nearly all fire/EMS.

## Bottom line

A police scanner works by **scanning** frequencies, **trunk-tracking** a
[control channel](/reference/control-channel/) to follow
[talkgroups](/reference/talkgroup/), and **decoding digital voice** where agencies
have gone [P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/).
Match those three abilities to what your county actually runs — start with
[digital vs analog](/digital-vs-analog-scanner/) and the
[buying guide](/scanner-buying-guide/) — and remember that
[simulcast](/reference/simulcast/) rewards a better front end while
[encryption](/police-scanner-encryption/) stops everyone equally. If you own a PC, a
[$30 dongle and GopherTrunk](/police-scanner-vs-sdr/) does all of this and records
every call.
