---
slug: wi-fi
title: Wi-Fi
entry_type: concept
category: hw-networking
description: Wi-Fi is the family of wireless local-area networking standards that let devices join a network over radio in the 2.4, 5, and 6 GHz bands instead of a cable, bridging clients onto a wired LAN through an access point.
keywords: Wi-Fi, IEEE 802.11, WLAN, wireless networking, 2.4 GHz, 5 GHz, 6 GHz, Wi-Fi 6, Wi-Fi 7, access point, generations
aka: [WiFi, IEEE 802.11, WLAN]
infobox:
  - { label: Type, value: Wireless LAN standard }
  - { label: Standard, value: IEEE 802.11 }
  - { label: Bands, value: 2.4, 5, 6 GHz }
  - { label: Reach, value: Tens of metres indoors }
  - { label: Bridges via, value: Wireless access point }
see_also: [wireless-access-point, wifi-80211, ethernet, bluetooth, network-interface-card, router, electromagnetic-spectrum]
cite_urls:
  - https://en.wikipedia.org/wiki/Wi-Fi
---

**Wi-Fi** is the family of wireless local-area networking standards that let devices join a network over radio, in the 2.4, 5, and 6 GHz bands, instead of a wired connection.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A Wi-Fi link: a wired access point radiates arcs of radio signal to a laptop and a phone over the air, and connects by an Ethernet cable to a switch that reaches the wider network. The clients reach the LAN wirelessly through the access point." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="206" y="24" width="48" height="26" rx="4" fill-opacity="0.16" stroke-width="1.3"/>
    <g stroke-width="1.3" fill="none" stroke-opacity="0.85">
      <path d="M198 40 a18 18 0 0 0 -18 18"/>
      <path d="M190 40 a26 26 0 0 0 -26 26"/>
      <path d="M262 40 a18 18 0 0 1 18 18"/>
      <path d="M270 40 a26 26 0 0 1 26 26"/>
    </g>
    <g stroke-width="1.3">
      <rect x="128" y="90" width="52" height="34" rx="3" fill-opacity="0.1"/>
      <rect x="286" y="94" width="26" height="38" rx="4" fill-opacity="0.1"/>
      <rect x="360" y="30" width="46" height="26" rx="3" fill-opacity="0.14"/>
    </g>
    <line x1="254" y1="37" x2="360" y2="43" stroke-width="1.3"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="230" y="41" font-weight="600" font-size="8">AP</text>
    <text x="154" y="110" font-size="8">laptop</text>
    <text x="299" y="118" font-size="8">phone</text>
    <text x="383" y="47" font-size="8">switch</text>
    <text x="312" y="40" font-size="7.5" fill-opacity="0.9">wired to LAN</text>
    <text x="230" y="142" fill-opacity="0.9">clients reach the LAN over the air through the access point</text>
  </g>
</svg>
<figcaption>A wireless access point radiates a shared radio channel that client devices — a laptop, a phone — associate with, while its wired uplink bridges that traffic onto the Ethernet LAN. The air link is convenient but shared, so all nearby clients compete for the same channel.</figcaption>
</figure>

## Overview

Built on the **IEEE 802.11** standards, Wi-Fi connects client devices to a [wireless access point](/reference/wireless-access-point/), which bridges them onto the wired [LAN](/reference/lan-and-wan/). A client *associates* with the access point and then shares its radio channel with every other client, taking turns to transmit — so raw link speed and real throughput can differ sharply when the air is busy.

Successive generations — now labelled Wi-Fi 4, 5, 6, and 7 — have raised throughput and efficiency using wider channels, more spatial streams (MIMO), and smarter modulation, and the newest add the roomy 6 GHz band. Real-world range and speed still depend heavily on interference, walls, and how many devices share the channel.

Because it uses *unlicensed* [spectrum](/reference/electromagnetic-spectrum/), Wi-Fi competes with [Bluetooth](/reference/bluetooth/), microwave ovens, and neighbouring networks for airtime, and has no exclusive claim to any frequency — coexistence, not isolation, is the design premise.

## Generations

Each generation added bandwidth and efficiency while staying backward compatible:

| Generation | 802.11 | Bands | Peak rate (order of) |
|------------|--------|-------|----------------------|
| Wi-Fi 4 | n | 2.4 / 5 GHz | Hundreds of Mb/s |
| Wi-Fi 5 | ac | 5 GHz | ~1 Gb/s |
| Wi-Fi 6 / 6E | ax | 2.4 / 5 / 6 GHz | Several Gb/s |
| Wi-Fi 7 | be | 2.4 / 5 / 6 GHz | Tens of Gb/s |

## Where it fits

Wi-Fi trades the raw stability of [Ethernet](/reference/ethernet/) for mobility and easy deployment, making it the default for laptops, phones, and IoT devices. A GopherTrunk node can report over Wi-Fi where running a cable is impractical, and pointing a phone or laptop at the daemon's web console over the house Wi-Fi is a natural way to check on it. But the 2.4 GHz band is busy and can desensitize a nearby receiver, so a wired link or careful placement is wiser when the antenna and the radio share a roof.

## Sources

[^wiki]: [Wi-Fi](https://en.wikipedia.org/wiki/Wi-Fi) — Wikipedia, on the IEEE 802.11 wireless networking family, its bands, and its generations.
