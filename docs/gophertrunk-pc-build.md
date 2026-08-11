---
layout: page
title: "GopherTrunk PC Build — Complete Parts List for a Desktop or Laptop Scanner"
description: "The cheapest way to run GopherTrunk: a complete, priced parts list to turn the Windows, Mac, or Linux PC you already own into a P25/DMR/NXDN trunking scanner — dongle, antenna, adapter, and optional LNA or filter."
keywords: GopherTrunk PC build, RTL-SDR PC scanner, SDR laptop scanner, cheap SDR scanner build, GopherTrunk hardware list, RTL-SDR Windows setup, SDR desktop build, police scanner PC
permalink: /gophertrunk-pc-build/
nav_group: Hardware
affiliate: true
faq:
  - q: "What do I need to run GopherTrunk on my PC?"
    a: "Three things beyond the computer you already own: an SDR dongle (a ~$35 RTL-SDR Blog V4 or NESDR SMArt v5), an antenna (a ~$25 dipole kit to start), and usually an SMA adapter or short cable to join them. That is a complete GopherTrunk scanner for about $60–80, and the software is free."
  - q: "Does GopherTrunk run on Windows and Mac?"
    a: "Yes. GopherTrunk is a single Go binary that runs on Windows, macOS, and Linux. On Windows you bind the dongle's driver with Zadig once; on Mac and Linux it works out of the box. Any machine from the last decade with a free USB port has ample power to decode a control channel or two."
  - q: "How cheap can a GopherTrunk setup be?"
    a: "If you already own a computer, about $60–80: roughly $35 for a good dongle, $25 for a dipole antenna kit, and a few dollars for an adapter. Buying the dongle-plus-antenna bundle is the single cheapest route because it puts both key parts in one box."
  - q: "Do I need an LNA or a filter for a PC build?"
    a: "Usually not to start. Add a low-noise amplifier only if your control channel is weak and distant, and add a broadcast FM/AM notch filter only if a strong local station overloads the dongle — a common problem in cities. Buy the base kit first, then add these only if reception needs it."
  - q: "Can my laptop decode trunked radio in real time?"
    a: "Easily. Following one or two control channels plus voice is light work for any modern laptop — it uses a fraction of one core. You only need more compute for large multi-dongle pools or wideband channelizing, which is a different build."
  - q: "Will this PC build hear encrypted channels?"
    a: "No. No SDR or scanner can decode AES-encrypted talkgroups. A PC build hears everything transmitted in the clear, which in most areas is plenty of dispatch and nearly all fire and EMS. Check your local system before buying so you know what is clear."
---

# GopherTrunk PC Build

**The cheapest way into GopherTrunk is the computer already on your desk.** Add an
[SDR dongle](/best-sdr-for-gophertrunk/), an [antenna](/best-sdr-antenna/), and a small
[adapter](/sdr-cables-and-connectors/), and the Windows, Mac, or Linux machine you own
becomes a full [P25](/reference/project-25/)/DMR/NXDN/TETRA trunking scanner. The
[software is free](/downloads.html); this whole build is about **$60–80** in hardware.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Use a PC you own** (Windows/Mac/Linux, any machine from the last ~decade). **Buy:** a
[RTL-SDR](/reference/rtl-sdr/) dongle (~$35), a [dipole antenna kit](/best-sdr-antenna/)
(~$25), an [SMA adapter](/sdr-cables-and-connectors/) (~$13). **Cheapest route:** the
[dongle + antenna bundle](/reference/rtl-sdr/). **Optional:** an [LNA](/best-sdr-lna/) for
weak signals, a [notch filter](/sdr-filters/) for city overload. **Total ~$60–80.**
**No build decodes [encryption](/police-scanner-encryption/).**
</div>

## The complete parts list

| # | Item | Why | Pick | Approx $ |
|---|---|---|---|---|
| 1 | **Computer** | Runs GopherTrunk | A PC/laptop you already own | $0 |
| 2 | **SDR dongle** | Receives the radio | [RTL-SDR Blog V4](/reference/rtl-sdr/) or [NESDR SMArt v5](/reference/nesdr/) | ~$35 |
| 3 | **Antenna** | Actually hears signals | [Telescopic dipole kit](/reference/dipole-antenna/) | ~$25 |
| 4 | **SMA adapter kit** | Joins antenna to dongle | [16-piece SMA adapter kit](/reference/sma-adapter-kit/) | ~$13 |
| + | **LNA** (optional) | Boosts a weak, distant channel | [RTL-SDR Blog wideband LNA](/best-sdr-lna/) | ~$30 |
| + | **FM notch filter** (optional) | Fixes broadcast overload | [Broadcast FM block filter](/sdr-filters/) | ~$25 |
| + | **Active USB extension** (optional) | Antenna at the window, PC elsewhere | [Shielded active cable](/reference/usb-extension-cable/) | ~$15 |

**Running total (essentials): ~$73.** Buy the dongle-plus-antenna bundle and it drops
toward **~$60**. Add the optional parts only if your reception needs them.

## The picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Cheapest route: buy the bundle</span>
<h3>RTL-SDR Blog V4 + dipole kit</h3>
<p class="pick-card__price">around $50</p>
<p>The dongle <em>and</em> a proper tunable dipole antenna in one box — items 2 and 3 of the list together, for less than buying them apart. The simplest way to start a PC build.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD7558GT?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 + antenna kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/best-rtl-sdr/">dongle guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Always in stock</span>
<h3>NooElec NESDR SMArt v5</h3>
<p class="pick-card__price">around $35</p>
<p>0.5 ppm TCXO, aluminium heatsink case, R820T2/R860 tuner. The mainstream "just buy this" dongle — rock-steady control-channel lock and reliable stock. Pair with a dipole.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/nesdr/">NESDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Bridges any mismatch</span>
<h3>16-piece SMA adapter kit</h3>
<p class="pick-card__price">around $13</p>
<p>SMA to/from BNC, UHF, N, and F in both genders. Covers almost any dongle-to-antenna connector you will hit — the one thing to buy if you are unsure what your antenna ends in.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Adapter kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/sma-adapter-kit/">adapter details</a> · <a href="/sdr-cables-and-connectors/">cable guide</a></p>
</div>
</div>

## Why these parts

**The dongle** is the receiver. A good [RTL-SDR](/reference/rtl-sdr/) — the
[RTL-SDR Blog V4](https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20) or a
[NESDR SMArt v5](https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20) — is the sweet
spot at about $35. The one part that matters is a **TCXO**: a temperature-compensated
oscillator that keeps the dongle from drifting off the control channel as it warms up.
Skip the no-name $15 sticks — the drift will cost you the lock. Full rundown:
[best RTL-SDR for scanning](/best-rtl-sdr/).

**The antenna matters more than the dongle.** The stub bundled with cheap sticks hears
strong locals and little else. A [telescopic dipole kit](https://www.amazon.com/dp/B075445JDF?tag=gophertrunk-20)
(~$25) you can tune to your band is a big upgrade you can set on a windowsill today, and
it upgrades cleanly to an outdoor antenna later. See [best SDR antenna](/best-sdr-antenna/).

**The adapter** exists because most SDRs use an [SMA](/reference/sma-adapter-kit/) jack
while antennas often end in BNC, UHF/PL-259, or N. A cheap
[adapter kit](https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20) bridges whatever you
have; a short [pigtail or extension](/sdr-cables-and-connectors/) lets you place the
antenna where the signal is.

## Optional upgrades — add only if needed

- **[LNA (low-noise amplifier)](/best-sdr-lna/)** — if your control channel is weak and
  distant, a [wideband LNA](https://www.amazon.com/dp/B07G14Q6XX?tag=gophertrunk-20) (~$30)
  lifts it out of the noise. Best mounted at the antenna and powered up the coax by the
  dongle's [bias tee](/reference/bias-tee/). Do not overdo gain — too much *causes* overload.
- **[Broadcast FM notch filter](/sdr-filters/)** — in a city a strong FM or AM station can
  desensitize the dongle. A [88–108 MHz block filter](https://www.amazon.com/dp/B01LE9LRPM?tag=gophertrunk-20)
  (~$25) restores weak-signal reception.
- **[Active USB extension](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20)** (~$15) —
  put the dongle at the window with the antenna and the PC wherever is convenient, without a
  lossy long coax run.

## Put it together

1. Screw the antenna (via any needed [adapter](/reference/sma-adapter-kit/)) onto the SDR.
2. Plug the SDR into a USB port.
3. [Download GopherTrunk](/downloads.html) and follow the
   [hardware setup guide](/hardware.html). On **Windows**, bind the driver with
   [Zadig](/reference/zadig/) first; on Mac/Linux it just works.
4. Enter your system's control channel — look it up on
   [RadioReference](https://www.radioreference.com/) — and start decoding.

Want it always on instead of tied to your desk? The next step up is a dedicated
[Raspberry Pi build](/gophertrunk-sbc-build/). Ready for real range? Move the antenna
outdoors with the [outdoor base build](/gophertrunk-outdoor-base-build/).

## Where to buy

The cheapest complete route is the
[**RTL-SDR Blog V4 + dipole antenna kit**](https://www.amazon.com/dp/B0CD7558GT?tag=gophertrunk-20)
(~$50) plus an [SMA adapter kit](https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20)
(~$13) — under $65 for the whole radio chain, software free.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD7558GT?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost to
you. It never changes what we recommend.*

## Bottom line

A **computer you own + a ~$35 [dongle](/best-rtl-sdr/) + a ~$25
[antenna](/best-sdr-antenna/) + a few-dollar [adapter](/sdr-cables-and-connectors/)** is the
entire shopping list for a GopherTrunk PC build — about **$60–80**, or less with the bundle.
Add an [LNA](/best-sdr-lna/) or [filter](/sdr-filters/) only if reception needs it. Everything
hits the same [encryption](/police-scanner-encryption/) wall, so buy for the traffic that is
in the clear. Compare the other setups on the [build-lists hub](/gophertrunk-build-lists/).
