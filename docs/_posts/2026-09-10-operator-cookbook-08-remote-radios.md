---
title: "The Operator's Cookbook, Part 8: Radios Far Away — SoapyRemote, rtl_tcp & ka9q"
description: "A complete GopherTrunk recipe for putting the antenna where reception is and the decoder where the CPU is: sdr.soapy_remote, rtl_tcp and ka9q_radio config, the network bandwidth math per format, validated antenna-port selection, and why host_drops means your decoder stalled — not your network."
category: tutorials
keywords: sdr over network, rtl_tcp gophertrunk setup, soapyremote sdr server config, ka9q-radio multicast iq, remote sdr antenna attic, usrp remote streaming, sdr network bandwidth calculator, antenna port selection usrp, gophertrunk cookbook
tags: [operator-cookbook, soapyremote, rtl-tcp, ka9q, networking, config]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 8
---

*Part 8 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
filled one box with dongles; this part cuts the USB cable. The best antenna
spot — attic, roof, a shed on a hill — is rarely where you want a computer,
and the best decode box is rarely worth hauling up a ladder. GopherTrunk
mounts three kinds of network radio as first-class virtual tuners:
**rtl_tcp** for cheap dongles, **SoapyRemote** for everything up to a USRP,
and **ka9q-radio** for multicast channel feeds. Same pool, same decoders,
same web console — just with an Ethernet run where the coax used to be.*

> **TL;DR:** A Pi at the antenna runs `rtl_tcp` or `SoapySDRServer`; the
> rack box lists it under `sdr.rtl_tcp:` / `sdr.soapy_remote:` /
> `sdr.ka9q_radio:` and it joins the pool like local hardware — roles,
> voice, everything. Budget the wire first: 2.4 MS/s costs ~38 Mbit/s as
> rtl_tcp's 8-bit stream and ~77 Mbit/s as SoapyRemote CS16. USRP
> [antenna ports]({{ '/reference/soapyremote/' | relative_url }}) are set
> with `antenna: [RX2]` — validated against the device's port list and
> **read back** after setting, a hard-won behavior. And when
> `SDR overruns … host_drops` appears, the network is innocent: the
> *decode side* stopped draining. All three transports are plaintext —
> trusted networks or a tunnel only.

**Key takeaways**

- **A remote radio is a pool citizen, not a special case.** Each entry
  mounts as a virtual tuner with a serial and a role; every recipe in this
  series works unchanged on top of one.
- **Format sets the bill.** rtl_tcp is hardcoded 8-bit; SoapyRemote carries
  CS16 (default) or CF32 at 2× the bytes; ka9q-radio ships narrow
  per-channel streams that barely register. Do the multiplication before
  blaming the network.
- **Antenna selection is verified, not hoped.** The port name is checked
  against what the device advertises, set, and read back — because for a
  long time the "set antenna" call silently invoked the wrong remote
  function and nobody's config did anything.
- **`host_drops` points downstream.** The driver sheds samples only when
  the consumer stops draining its buffer — pair it with the ccdecoder's
  can't-keep-up WARN and fix CPU, not cables.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Cheap remote dongle | 8-bit rtl_tcp stream as a virtual tuner | `sdr.rtl_tcp[]` ([rtl_tcp]({{ '/reference/rtl-tcp/' | relative_url }})) |
| High-end remote radio | CS16/CF32, full tuning control | `sdr.soapy_remote[]` ([SoapySDR]({{ '/reference/soapysdr/' | relative_url }})) |
| Multicast channels | radiod RTP channels, one entry per SSRC | `sdr.ka9q_radio[]` ([ka9q-radio]({{ '/reference/ka9q-radio/' | relative_url }})) |
| Antenna port | per-channel RX port, validated + read back | `soapy_remote[].antenna: [RX2]` |
| Link tuning | MTU / flow-control window for fat or slow links | `stream_mtu`, `stream_window` |
| Exact USRP rates | make sample_rate an integer clock division | `master_clock_rate` ([USRP B210]({{ '/reference/usrp-b210/' | relative_url }})) |
| RPC forensics | log every control call + hex dump | `verbose_debug: true` (needs `log.level: debug`) |

## In this post

- **What you're building** — antenna in the attic, decoder in the rack.
- **The shopping list** — a Pi, a dongle, and honest Ethernet.
- **The network budget** — bytes per second, before anything else.
- **The config** — all three transports, every key verified.
- **First run — what healthy looks like** — three connected lines and a read-back.
- **When it doesn't work** — the heaviest troubleshooting table in the series.

## What you're building

The finished rig is two boxes. At the antenna: a Raspberry Pi with the
radio on a short, low-loss coax run — physics happy. In the rack: the
GopherTrunk daemon from Parts 1–7, CPU and disk happy, with the remote
radio in its pool next to any local dongles. Decoders, voice pool,
recordings, web console — none of them know the IQ crossed a wire.

Which transport fits which build:

| Transport | Hardware | Stream | Best for |
|---|---|---|---|
| `rtl_tcp` | RTL-SDR dongles | 8-bit, fixed | the $35 attic dongle |
| `soapy_remote` | USRP, LimeSDR, Airspy, HackRF, bladeRF, SDRplay, RTL | CS16 / CF32, full control | serious remote front ends, diversity rigs |
| `ka9q_radio` | radiod + RX888/Airspy/RTL | per-channel RTP multicast | one shared receiver feeding many consumers on a LAN |

All three are **plaintext** with no authentication — keep them on a trusted
LAN or inside an SSH/WireGuard tunnel, the same posture logic as the
[API auth deep dive]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="The remote-radio topology: at the antenna site a mast feeds a radio attached to a Raspberry Pi running rtl_tcp or SoapySDRServer or radiod; an Ethernet link annotated with per-format bandwidth carries raw IQ to the rack box, where GopherTrunk's rtl_tcp, soapyremote and ka9q drivers mount each stream as a virtual tuner in the SDR pool feeding the usual decoders, recordings and web console">
  <rect x="8" y="20" width="200" height="190" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="108" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="10">antenna site (attic / roof / shed)</text>
  <rect x="24" y="52" width="70" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="59" y="72" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="59" y1="84" x2="59" y2="104" stroke="currentColor"/>
  <text x="110" y="98" fill="var(--fg-muted)" font-size="9">short coax</text>
  <rect x="24" y="104" width="70" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="59" y="124" text-anchor="middle" fill="currentColor" font-size="10">radio</text>
  <line x1="94" y1="120" x2="122" y2="120" stroke="currentColor"/>
  <rect x="122" y="96" width="74" height="70" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="159" y="114" text-anchor="middle" fill="var(--accent)" font-size="10">Pi</text>
  <text x="159" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">rtl_tcp /</text>
  <text x="159" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SoapySDRServer /</text>
  <text x="159" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="9">radiod</text>
  <line x1="208" y1="120" x2="330" y2="120" stroke="var(--accent)" stroke-width="2"/>
  <text x="269" y="108" text-anchor="middle" fill="var(--accent)" font-size="10">Ethernet</text>
  <text x="269" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">u8 ~38 Mbit/s</text>
  <text x="269" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CS16 ~77 Mbit/s</text>
  <text x="269" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="9">@ 2.4 MS/s</text>
  <rect x="330" y="20" width="340" height="190" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="500" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="10">rack box — GopherTrunk daemon</text>
  <rect x="346" y="52" width="140" height="110" rx="4" fill="none" stroke="currentColor"/>
  <text x="416" y="70" text-anchor="middle" fill="currentColor" font-size="10">network drivers</text>
  <text x="416" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">rtltcp: connected</text>
  <text x="416" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">soapyremote: connected</text>
  <text x="416" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ka9qradio: connected</text>
  <text x="416" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ virtual tuners, by serial</text>
  <line x1="486" y1="107" x2="516" y2="107" stroke="currentColor"/>
  <rect x="516" y="52" width="140" height="52" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="586" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">SDR pool (Part 7)</text>
  <text x="586" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">roles, watchdog, voice</text>
  <rect x="516" y="120" width="140" height="52" rx="4" fill="none" stroke="currentColor"/>
  <text x="586" y="140" text-anchor="middle" fill="currentColor" font-size="10">decoders → recordings</text>
  <text x="586" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ web console</text>
  <line x1="586" y1="104" x2="586" y2="120" stroke="currentColor"/>
  <text x="500" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">plaintext transports — trusted LAN or tunnel only</text>
</svg>
<figcaption>The coax run shrinks to inches and the network carries the IQ instead: each remote stream mounts as a virtual tuner in the same pool every other recipe uses.</figcaption>
</figure>

## The network budget

Do this arithmetic before anything else:
**bytes/s = sample_rate × bytes-per-complex-sample.**

| Stream | @ 2.4 MS/s | @ 6.25 MS/s (USRP) |
|---|---|---|
| rtl_tcp (u8, 2 B/sample) | ~4.8 MB/s ≈ 38 Mbit/s | — |
| SoapyRemote CS16 (4 B) | ~9.6 MB/s ≈ 77 Mbit/s | ~25 MB/s ≈ 200 Mbit/s |
| SoapyRemote CF32 (8 B) | ~19 MB/s ≈ 154 Mbit/s | ~50 MB/s ≈ 400 Mbit/s |
| ka9q-radio channel | kilobits–a few Mbit/s each | (per channel, not per band) |

Wired gigabit [Ethernet]({{ '/reference/ethernet/' | relative_url }})
carries all of it; Wi-Fi carries the rtl_tcp row on a good day and nothing
above it reliably. CS16 is the right SoapyRemote default — CF32 doubles the
bill for dynamic range a trunking decode doesn't need. And the
[sample rate]({{ '/reference/sample-rate/' | relative_url }}) is *your*
knob: a remote USRP needn't stream its whole capability.

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| Raspberry Pi (or any SBC/old laptop) | ~$50 | runs the server end at the antenna |
| Radio | ~$35 and up | whatever the recipe you're remoting called for |
| Ethernet run / PoE | ~$20 | wired; a PoE splitter powers the Pi over the same cable |

## The config

One entry per transport, to show all three (use the ones you need):

```yaml
sdr:
  sample_rate: 2_400_000

  rtl_tcp:
    - addr: "192.168.1.50:1234"
      serial: "antenna-pi"
      role: control
      ppm: 0
      gain: "auto"
      connect_timeout_ms: 3000

  soapy_remote:
    - addr: "192.168.1.60:55132"   # bare host gets :55132 appended
      driver: "uhd"
      serial: "usrp-roof"
      role: control
      format: "CS16"               # or CF32 at 2x the bandwidth
      antenna: [RX2]               # validated against the device, read back
      master_clock_rate: 0         # see note below for exact rates
      stream_mtu: 0                # raise (e.g. 8192) on jumbo-frame links
      stream_window: 0             # raise on high-latency links
      gain: "auto"
      connect_timeout_ms: 3000

  ka9q_radio:
    - addr: "hf.local"             # radiod status group (mDNS or 239.x.x.x:5006)
      ssrc: 162550                 # the channel's RTP SSRC
      serial: "ka9q-wx"
      role: control
      connect_timeout_ms: 3000
```

Notes that earn their place. **`antenna:` is the real key, not `args`.** An
`antenna=` inside the `args` kwargs only reaches device construction and
never selects the per-channel port — GopherTrunk rejects it there. Port
names are device-specific (a B210 has `TX/RX` and `RX2`; a TwinRX has `RX1`
and `RX2`), so a config moved between rigs *should* fail loudly.
**`master_clock_rate`** exists because a USRP only streams integer
divisions of its clock: a B210 set to `61_440_000` makes a 6.144 MS/s
`sample_rate` exact instead of letting UHD coerce to a nearby rate.
**ka9q entries consume existing radiod channels** — configure the channel
in `radiod@.conf` first; GopherTrunk discovers its multicast group, rate
and encoding from the status stream.

## First run — what healthy looks like

Each transport announces itself once the stream is up:

```
INF rtltcp: connected addr=192.168.1.50:1234 tuner=R820T gain_count=29
INF soapyremote: connected addr=192.168.1.60:55132 format=CS16 proto=tcp diversity=none
INF soapyremote: rx antenna set addr=192.168.1.60:55132 channel=0 antenna=RX2
INF ka9qradio: connected status=239.1.2.3:5006 data=239.1.2.4:5004 ssrc=162550 samprate=48000
```

The third line deserves a pause. For a long time the "set antenna" RPC used
the **wrong opcode** — the SoapyRemote wire carries no schema, so the call
silently invoked a *different* remote function whose first arguments
happened to fit. Every `antennas:` config did nothing, and the only trace
was a cryptic `~SoapyRPCUnpacker: Unconsumed payload bytes 9` on the
server. The unit tests couldn't catch it — the fake test server switched on
the same wrong constant, the classic
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).
Today the driver validates your port name against the device's advertised
list, sets it, **reads it back**, and logs what the device itself reports —
the full saga is in
[From the Issue Tracker Part 18]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }}).
When you see `rx antenna set`, it happened.

From here, everything is Parts 1–7: control locks, grants, calls,
recordings. One more line matters on remote rigs specifically — the
overrun WARN:

```
WRN soapyremote: SDR overruns — the host can't keep up with the configured sample rate, so samples are being dropped and decoded audio will glitch. Lower sdr.sample_rate or reduce the channel/tap count. addr=... device_overflows=0 host_drops=412
```

Read the counters, not the vibes. **`host_drops` means the decode side
stalled**: the driver sheds the oldest queued chunk only when its consumer
stops draining a ~400 ms buffer, so a climbing `host_drops` says something
*downstream* got slow — check for the companion
`ccdecoder: decode can't keep up with real time` WARN, which confirms CPU
rather than network. `device_overflows` climbing instead points at the
remote host or the wire. When this fires, ask what you recently made
slower.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Daemon starts cleanly, but no radio and no error | remote-only config on an old build — the pool gate once required a *local* device list, so a config with only `soapy_remote`/`ka9q_radio` registered no driver at all | Fixed (pinned by `TestRemoteOnlySDRConfigsRegisterTheirDriver`); on current builds grep for the `connected` line — absent with no error means the block isn't under `sdr:` where you think |
| No `connected` line, then timeouts | server not running / wrong port / firewall | `SoapySDRServer --bind` on the radio host, rtl_tcp on 1234; test with `nc host port`; raise `connect_timeout_ms` on slow links |
| `channel 0 antenna "RX9" is not a port on this device (available: RX1, RX2)` | port name from a different radio model | Use a name from the error's own list — this failing loudly is the feature |
| `antenna set to "RX2" but device reports "RX1"` | server-side driver ignored the set | Read-back caught it; check the remote SoapySDR module/version for that hardware |
| Audio glitches, `host_drops` climbing, `device_overflows` ~0 | decode box CPU exhausted — consumer stalled | Shed taps/systems, lower `sdr.sample_rate`, bigger box. The network is innocent on this signature |
| Glitches with `device_overflows` climbing | remote host or link can't sustain the rate | Check the budget table; wired link; lower rate; on jumbo/high-latency links raise `stream_mtu` / `stream_window` |
| USRP runs at a slightly different rate than configured | UHD coerced to an integer clock division | Set `master_clock_rate` so `sample_rate` divides it exactly |
| `~SoapyRPCUnpacker: Unconsumed payload bytes N` on the server | a mis-shaped RPC — the wire has no schema | `verbose_debug: true` + `log.level: debug` logs every call with a hex dump; file it with that output |
| ka9q entry mounts but decodes nothing | wrong `ssrc`, or channel not in raw IQ mode | Match the SSRC from `radiod@.conf`; the channel needs `output_channels = 2` — the `channel is not 2-channel IQ` WARN is the tell |
| Works on the bench, dies over Wi-Fi | the budget table | Wire it. Raw IQ and Wi-Fi are enemies at trunking rates |

### How this recipe shapes operator practice

- **Budget the wire like you budget the feedline.** Format × rate is the
  network's version of
  [coax loss]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})
  — decided before any software runs.
- **Trust read-backs over acknowledgements.** The antenna lesson
  generalizes: a setting the device *reports* is real; a setting that
  merely didn't error may be fiction.
- **Blame by counter, not by topology.** "It's remote, so it's the network"
  is exactly what `host_drops` exists to refute.

## Where this goes next

The rig now hears everything from anywhere. [Part
9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
turns it outward: Broadcastify Calls, RdioScanner, OpenMHz, Icecast and
webhooks — the broadcast config that shares your calls with the world, each
backend's quirks, and the key hygiene that keeps your API secrets out of
your config file.

## FAQ

**Can GopherTrunk use an SDR on another computer?**
Yes, three ways: `sdr.rtl_tcp` for RTL-SDR dongles behind rtl_tcp,
`sdr.soapy_remote` for anything SoapySDR serves, and `sdr.ka9q_radio` for
radiod multicast channels. Each mounts as a virtual tuner with a serial and
role, indistinguishable to the decoders from local USB hardware.

**How much network bandwidth does a remote SDR need?**
Sample rate times bytes per sample: at 2.4 MS/s, rtl_tcp's 8-bit stream is
~38 Mbit/s and SoapyRemote CS16 is ~77 Mbit/s; a USRP at 6.25 MS/s in CS16
is ~200 Mbit/s. Wired gigabit handles all of it; Wi-Fi doesn't.

**What does the host_drops counter mean in the SDR overruns warning?**
That GopherTrunk's own decode side stopped draining the driver's buffer —
the oldest queued IQ was shed to keep the stream live. It signals CPU or a
stalled consumer on the decode box, not a network problem;
`device_overflows` is the counter that points at the remote end.

**How do I select the RX antenna port on a remote USRP?**
Set `antenna: [RX2]` (or `[RX1, RX2]` per channel under diversity) in the
`soapy_remote` entry — not `antenna=` inside `args`, which is rejected. The
name is validated against the ports the device advertises and read back
after setting, so a wrong name fails at startup instead of silently doing
nothing.

**Is rtl_tcp or SoapyRemote traffic encrypted?**
No — both are plaintext with no authentication, as is ka9q-radio's
multicast. Keep them on a trusted LAN, or carry them through an SSH or
WireGuard tunnel; never expose the ports to the internet.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Many Systems, One Box — The SDR Pool]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
· Next →
[Part 9: Sharing the Feed — Broadcastify, OpenMHz & Friends]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
