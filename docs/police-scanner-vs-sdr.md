---
layout: page
title: "Police Scanner vs SDR + GopherTrunk: An Honest Comparison"
description: "Should you buy a police scanner or run a free SDR with GopherTrunk? An honest, side-by-side comparison of cost, portability, recording, simulcast, and encryption — with clear recommendations for each."
keywords: police scanner vs SDR, scanner vs software defined radio, GopherTrunk vs scanner, RTL-SDR police scanner, free police scanner software, SDRTrunk alternative, digital scanner alternative
permalink: /police-scanner-vs-sdr/
nav_group: Hardware
faq:
  - q: "Is an SDR better than a police scanner?"
    a: "It depends on what you value. A software-defined radio with GopherTrunk is free (after a ~$30 dongle), records and timestamps every call, and follows unlimited talkgroups — but needs a PC and setup. A dedicated scanner is turnkey, portable, and battery-powered. Neither can decode encryption."
  - q: "Can GopherTrunk decode P25 like a Uniden scanner?"
    a: "Yes. GopherTrunk decodes P25 Phase I and II, DMR, NXDN, and TETRA from an RTL-SDR, HackRF, or Airspy — the same digital modes a Uniden SDS or Whistler TRX decodes. It also has a software equalizer that handles many simulcast systems."
  - q: "How much does it cost to replace a scanner with an SDR?"
    a: "About $30 for an RTL-SDR dongle, plus a computer you already own. GopherTrunk itself is free and open source. Compare that to $380–$800 for a digital trunking scanner."
  - q: "What can a scanner do that GopherTrunk can't?"
    a: "Run on batteries in your pocket or car with no computer, work out of the box in minutes, and — on the Uniden SDS series — squeeze slightly more out of the worst simulcast signals with a purpose-built front end."
  - q: "Can an SDR decode encrypted police that a scanner can't?"
    a: "No. AES encryption blocks scanners and SDRs equally. GopherTrunk is a receiver; it decodes clear and scrambled traffic but never keyed AES. Anyone claiming an SDR defeats encryption is wrong."
---

# Police Scanner vs SDR + GopherTrunk

**A dedicated scanner and a software-defined radio running GopherTrunk decode the
same digital police, fire, and EMS traffic — the difference is convenience versus
cost and data.** This page is the honest version: where the hardware genuinely
wins, and where free software does.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Same decoding.** Both handle [P25](/reference/project-25/),
[DMR](/reference/dmr/), and [NXDN](/reference/nxdn/). **Scanner wins on**
turnkey setup, pocket/car portability, batteries, and the best
[simulcast](/reference/simulcast/) front end (Uniden SDS). **GopherTrunk wins on**
price (~$30 dongle, free software), **recording/logging every call**, unlimited
simultaneous talkgroups, a remote **web console**, and exportable data. **Neither**
decodes [encryption](/police-scanner-encryption/). **Try GopherTrunk first** if you
own a PC — it costs almost nothing.
</div>

## The honest scorecard

| | Dedicated scanner | SDR + [GopherTrunk](/downloads.html) |
|---|---|---|
| **Price** | $380–$800 (digital) | ~$30 dongle, **free** software |
| **Setup** | Minutes (ZIP code) | Config file / guided setup |
| **Portability** | **Pocket/car, battery** | Needs a PC (or SBC) |
| **Digital modes** | P25/DMR/NXDN | P25/DMR/NXDN/**TETRA** |
| **Simulcast** | **Best on Uniden SDS** | Software CMA equalizer |
| **Record every call** | Limited / no | **Yes, timestamped** |
| **Simultaneous talkgroups** | A few | **Unlimited** |
| **Remote access** | No | **Web console** |
| **Data export / search** | No | **Yes (logs, DB)** |
| **Encryption (AES)** | Can't | Can't |
| **Runs with no computer** | **Yes** | No |

## Where a dedicated scanner genuinely wins

We build free software, and we'll still tell you the truth: **buy the scanner** if
any of these matter.

- **No computer, no fuss.** A [BCD436HP](/reference/uniden-bcd436hp/) programs from
  your ZIP in minutes and just works. GopherTrunk needs a PC and a little setup.
- **Portability and battery.** You can't put a laptop in your shirt pocket or clip
  it to a belt at a race. A handheld scanner goes anywhere.
- **In the car.** A base/mobile scanner is designed for a 12 V vehicle install with
  a proper speaker; an SDR-in-a-car means a laptop or single-board computer.
- **The hardest simulcast.** Uniden's [True I/Q](/reference/uniden-sds200/) front
  end still edges out a cheap [RTL-SDR](/reference/rtl-sdr/) on the worst
  [simulcast](/reference/simulcast/) systems. (An [Airspy](/reference/airspy/)
  narrows that gap.)
- **FCC-certified, warrantied, supported.** A finished product with a manual and a
  support line.

## Where GopherTrunk wins

- **Cost.** A ~$30 [RTL-SDR](/reference/rtl-sdr/) plus free, open-source
  [GopherTrunk](/downloads.html) versus $380–$800. If you have a spare PC, the
  barrier is one dongle.
- **It records everything.** Every call, timestamped, searchable — replay what you
  missed, keep an archive, correlate talkgroups. Scanners are built to listen live,
  not to log.
- **Unlimited scale.** Follow many [talkgroups](/reference/talkgroup/) and multiple
  [sites](/reference/trunking-site/)/systems at once, limited by your dongle and CPU,
  not a firmware channel cap.
- **Remote and headless.** Run it on a Raspberry Pi in the attic by the antenna and
  reach the [web console](/web.html) from your phone.
- **Open and extensible.** Inspect the [DSP](/reference/software-defined-radio/),
  add [decoders](/status.html), export data. It even decodes
  [TETRA](/reference/tetra/) voice, which most US scanners don't.
- **Learn how it works.** The whole [digital-trunking](/learn/digital-trunking/)
  pipeline is documented, not a black box.

## The one thing neither can do

**Decode encryption.** When an agency enables AES-256, the
[talkgroup](/reference/talkgroup/) shows up but the audio is silence — for the
$800 scanner and the $30 SDR alike. This is cryptographic reality and, without
authorization, a federal offense to defeat. See
[police scanner encryption explained](/police-scanner-encryption/). Choose your
tool based on the traffic that's **still in the clear** — which, in most areas,
still includes plenty of dispatch and nearly all fire/EMS.

## Which should you pick?

- **Buy a scanner** if you want to listen in the car, on a belt, or with zero
  computer involvement — start at [best police scanners](/best-police-scanners/).
- **Use GopherTrunk** if you own a PC, want every call recorded and searchable, want
  to monitor many systems at once, or just want to spend $30 instead of $600.
  [Download it](/downloads.html) and pair it with an
  [RTL-SDR](/reference/rtl-sdr/).
- **Do both.** Many hobbyists keep a handheld for the field and a GopherTrunk box at
  home logging 24/7. They complement each other.

## Bottom line

If portability or a no-computer experience matters, buy the scanner — the Uniden
SDS series is worth it on simulcast. If you have a computer and care about cost,
recording, and scale, a **[$30 dongle and GopherTrunk](/downloads.html)** does the
core job for free and then some. Everyone hits the same
[encryption](/police-scanner-encryption/) wall, so decide on the features you'll
actually use.
