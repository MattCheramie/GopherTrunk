---
layout: page
title: "Multi-Dongle & Wideband SDR Setup for GopherTrunk"
description: "How to run multiple SDRs with GopherTrunk — a pool of RTL-SDRs by serial and role, wideband Airspy channelizing several control channels, rtl_tcp/SoapySDR remotes, powered hubs, and overload."
keywords: multi-dongle SDR, wideband SDR, GopherTrunk multiple SDR, rtl_tcp, SoapySDR, channelize control channels, SDR pool, powered USB hub, Airspy channelizer, remote SDR
permalink: /multi-dongle-sdr-setup/
nav_group: Hardware
affiliate: true
faq:
  - q: "Can GopherTrunk use more than one SDR at once?"
    a: "Yes. GopherTrunk can drive a pool of SDRs simultaneously — locally over USB or remotely over rtl_tcp and SoapySDR — and assign each a role such as control-channel tracking, voice following, or wideband capture. That lets one instance cover multiple sites or split control and voice across dongles."
  - q: "How do I tell multiple identical dongles apart?"
    a: "Give each RTL-SDR a unique serial number with rtl_eeprom, then reference dongles by serial in GopherTrunk's configuration. Without unique serials the operating system can enumerate identical dongles in an unpredictable order, so setting serials is the first step in any multi-dongle build."
  - q: "Can one wideband Airspy replace several dongles?"
    a: "Often, yes. A wideband Airspy captures up to ~10 MHz at once, and GopherTrunk can channelize several control channels from that single capture in software. If a system's control channels fall within one Airspy's bandwidth, one wideband receiver can do the work of several narrow dongles."
  - q: "Do I need a powered USB hub for multiple SDRs?"
    a: "Yes. Several RTL-SDRs draw more current than an unpowered hub or a Raspberry Pi can supply, and they run warm. A quality powered (active) USB hub gives each dongle stable power and keeps them physically spaced so they run cooler and interfere less."
  - q: "Can GopherTrunk use SDRplay, LimeSDR, or a USRP in a pool?"
    a: "Only over the network. SDRplay RSP, LimeSDR, USRP, bladeRF, and PlutoSDR are supported through SoapySDR/SoapyRemote or rtl_tcp, not as direct pure-Go USB devices. RTL-SDR, HackRF, and Airspy are the direct-USB devices; everything else is mounted as a network source."
  - q: "Does sharing one antenna across dongles cause problems?"
    a: "It can. Splitting one antenna into several dongles adds loss and, more importantly, a strong signal in a shared wideband capture can overload the front end for every channel you extract from it. Filter strong local signals ahead of the split and watch gain on shared captures."
  - q: "Can I run SDRs on another machine?"
    a: "Yes. Run rtl_tcp or a SoapySDR server on a remote host — say a Raspberry Pi at the antenna — and point GopherTrunk at it over the network. You can mix local USB dongles and remote network sources in the same pool."
---

# Multi-Dongle & Wideband SDR Setup for GopherTrunk

**GopherTrunk can drive a whole pool of SDRs at once — locally over USB or remotely over
[rtl_tcp](/reference/rtl-tcp/) and [SoapySDR](/reference/soapysdr/) — and it can
channelize several control channels from a single wideband
[Airspy](/reference/airspy/) capture.** That means one instance can cover multiple sites,
split control and voice across dongles, or feed many decoders from one wideband front
end.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Pool of dongles:** assign each [RTL-SDR](/reference/rtl-sdr/) a **unique serial** and a
**role** (control / voice / wideband). **Wideband path:** one
[Airspy](/reference/airspy/) channelizes several control channels from one ~10 MHz
capture. **Remotes:** mount SDRs over [rtl_tcp](/reference/rtl-tcp/) or
[SoapySDR](/reference/soapysdr/) from another host. **Power:** use a **powered USB hub**.
**Watch overload** on shared captures. **SDRplay/Lime/USRP are network-only** via
SoapyRemote.
</div>

## Two ways to cover more than one channel

There are two distinct strategies, and many builds combine them:

1. **A pool of narrow dongles.** Each [RTL-SDR](/reference/rtl-sdr/) tunes its own
   frequency. Perfect when a system's sites or control channels are spread far across a
   band — assign a dongle per site, or one dongle to the control channel and another to
   follow voice.
2. **One wideband receiver, channelized.** A single [Airspy](/reference/airspy/) grabs up
   to ~10 MHz in one capture and GopherTrunk **channelizes** it — extracting several
   control channels from that one stream in software. Ideal when the channels you want
   fall inside one wideband window.

| Approach | Best when | Hardware |
|---|---|---|
| Dongle pool | Sites/channels spread across a band | Several [RTL-SDRs](/reference/rtl-sdr/) + [powered hub](/reference/rtl-tcp/) |
| Wideband channelize | Channels within ~10 MHz | One [Airspy R2](/reference/airspy/) |
| Hybrid | Big multi-site systems | Airspy wideband + dongles for the outliers |

## Serials and roles

The first practical step with multiple dongles is making them addressable.

> **Set unique serials first.** Fresh [RTL-SDRs](/reference/rtl-sdr/) often share the
> same default serial, so the OS enumerates them in an unpredictable order. Use
> `rtl_eeprom` to give each a unique serial (e.g. `00000001`, `00000002`), then refer to
> each dongle by serial in GopherTrunk. Now "the control dongle" is always the same
> physical stick.

With serials set, you assign **roles**: which dongle tracks a
[control channel](/reference/project-25/), which follows voice grants, which does
wideband capture. GopherTrunk coordinates the pool so grants read on the control dongle
are followed on a voice dongle without missing calls.

## Remote SDRs over the network

Dongles do not have to hang off the same machine. Run **[rtl_tcp](/reference/rtl-tcp/)**
or a **[SoapySDR](/reference/soapysdr/)** server on another host — commonly a
[Raspberry Pi at the antenna](/raspberry-pi-sdr-scanner/) — and point GopherTrunk at it
over your LAN. You can freely **mix local USB dongles and remote network sources** in one
pool, keeping the noisy USB and the long coax runs out at the antenna while the decoder
runs wherever is convenient.

> **SDRplay, LimeSDR, USRP, bladeRF, and PlutoSDR are network-only.** GopherTrunk speaks
> pure-Go USB directly to the [RTL-SDR](/reference/rtl-sdr/) family, HackRF, and
> [Airspy](/reference/airspy/). Everything else is mounted through
> [SoapySDR](/reference/soapysdr/)/SoapyRemote or [rtl_tcp](/reference/rtl-tcp/) as a
> network device — plan on running a Soapy server for those.

## Power, hubs, and heat

Several dongles are a real electrical and thermal load:

- **Use a powered (active) USB hub.** Multiple [RTL-SDRs](/reference/rtl-sdr/) draw more
  current than an unpowered hub — or a bare [Raspberry Pi](/raspberry-pi-sdr-scanner/) —
  can safely source. A good powered hub gives each dongle clean, stable power.
- **Space them out.** Dongles run warm and can couple noise into each other when stacked.
  A hub with spacing, or short
  [USB extensions](https://www.amazon.com/dp/B00BWK9VZ2?tag=gophertrunk-20), keeps them
  cool and quiet.
- **Mind total current.** Add a bias-tee-powered [LNA](/best-sdr-lna/) and the draw grows
  again — size the supply accordingly.

## Overload on shared captures

The catch with wideband and shared-antenna setups is **front-end overload**, and it bites
harder here than with a single dongle:

> **A strong local signal poisons every channel in a shared capture.** When you split one
> antenna across dongles, or channelize many signals out of one wideband
> [Airspy](/reference/airspy/) stream, an overloading broadcast FM, pager, or AM
> transmitter degrades *all* of them at once. Put a
> [broadcast notch or bandpass filter](/sdr-filters/) ahead of the split, keep gain
> conservative on wideband captures, and check the noise floor before blaming the decoder.

## Bottom line

GopherTrunk scales from one dongle to a coordinated **pool** — by serial and role, over
USB or [rtl_tcp](/reference/rtl-tcp/)/[SoapySDR](/reference/soapysdr/) — and can
**channelize a wideband [Airspy](/reference/airspy/)** to cover several
[control channels](/reference/project-25/) from one capture. Set unique serials, feed the
dongles from a powered hub, keep [SDRplay/Lime/USRP](/reference/soapysdr/) in mind as
network-only devices, and [filter](/sdr-filters/) strong locals before any split. Plan
the whole system in [best SDR for P25 trunking](/best-sdr-for-p25-trunking/) and wire it
up from the [hardware guide](/hardware.html).
