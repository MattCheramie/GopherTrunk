---
slug: home-server
title: Home server & self-hosting
entry_type: concept
category: hw-servers
description: A home server is a computer you run at home to host services for yourself, such as file storage, a personal site, or media — the heart of self-hosting.
keywords: home server, self-hosting, NAS, self host, port forwarding, home lab, dynamic DNS
aka: [Home server, Self-hosting]
infobox:
  - { label: Type, value: Home-run server }
  - { label: Common uses, value: NAS, media, personal site }
  - { label: Hidden costs, value: Power, uptime, backups }
  - { label: Reached via, value: Port forwarding / DNS }
  - { label: Good first box, value: Raspberry Pi or old PC }
see_also: [server, raspberry-pi, single-board-computer, network-attached-storage, dedicated-server, cloud-computing]
related_lessons:
  - { title: "Home servers", url: /learn/intro-hardware/home-servers/ }
  - { title: "Combining tiers", url: /learn/intro-hardware/combining-tiers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Self-hosting_(web_services)
---

A **home server** is a computer you run at home to host services for yourself — file storage (a NAS), a personal site, or media you stream around the house.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A home network. A small always-on server inside the house serves files and media to a laptop, phone, and TV over the local network. A router with port forwarding connects the server to the internet cloud so it can be reached from outside." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <path d="M20 60 h230 v96 h-230 z M20 60 l115 -34 l115 34" fill-opacity="0.03" stroke-width="1.4"/>
    <text x="135" y="50" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">your home</text>
    <rect x="40" y="92" width="52" height="52" rx="4" fill-opacity="0.18" stroke-width="1.4"/>
    <text x="66" y="114" text-anchor="middle" font-size="8" stroke="none" font-weight="600">home</text>
    <text x="66" y="126" text-anchor="middle" font-size="8" stroke="none" font-weight="600">server</text>
    <g stroke-width="1.2" fill="none">
      <line x1="92" y1="104" x2="150" y2="82"/>
      <line x1="92" y1="118" x2="150" y2="118"/>
      <line x1="92" y1="132" x2="150" y2="150"/>
    </g>
    <g stroke-width="1.1" fill-opacity="0.14">
      <rect x="152" y="72" width="30" height="20" rx="2"/>
      <rect x="152" y="108" width="30" height="20" rx="2"/>
      <rect x="152" y="140" width="30" height="20" rx="2"/>
    </g>
    <text x="200" y="86" font-size="7.5" stroke="none" fill-opacity="0.85">laptop</text>
    <text x="200" y="122" font-size="7.5" stroke="none" fill-opacity="0.85">phone</text>
    <text x="200" y="154" font-size="7.5" stroke="none" fill-opacity="0.85">TV</text>
    <rect x="270" y="98" width="46" height="34" rx="3" fill-opacity="0.12" stroke-width="1.3"/>
    <text x="293" y="112" text-anchor="middle" font-size="7.5" stroke="none">router</text>
    <text x="293" y="124" text-anchor="middle" font-size="6.5" stroke="none" fill-opacity="0.8">port fwd</text>
    <line x1="92" y1="118" x2="270" y2="115" stroke-width="1" stroke-dasharray="3 2" fill="none"/>
    <line x1="316" y1="115" x2="372" y2="115" stroke-width="1.4" fill="none"/>
    <path d="M372 115 l-8 -3 v6 z" stroke-width="1"/>
    <path d="M392 96 a16 12 0 0 1 0 24 h-14 a13 13 0 0 1 0 -22 a17 12 0 0 1 32 -2 z" fill-opacity="0.08" stroke-width="1.3"/>
    <text x="404" y="116" text-anchor="middle" font-size="7.5" stroke="none">internet</text>
    <text x="230" y="174" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">you own the hardware &#183; you pay for power, uptime, and backups</text>
  </g>
</svg>
<figcaption>A home server sits on your own network serving files and media to household devices; a router with port forwarding lets you reach it from the internet — the whole appeal being control and privacy, at the cost of running the box yourself.</figcaption>
</figure>

## Overview

Self-hosting means running those services on hardware you own instead of paying a provider. The appeal is control and privacy; the costs nobody mentions up front are electricity, keeping the box online 24/7, and a real backup plan. To reach it from outside your house you typically set up port forwarding on your router — often paired with dynamic DNS — so requests find the [server](/reference/server/) even as your home IP address changes.

Home servers range from a single-board computer sipping a few watts to a repurposed desktop or a small rack in a closet ("home lab"). What they share is that *you* are the operator: there is no provider to page when a disk fills or a service crashes, which is both the point and the burden.

## Where it fits

Choosing the first box is mostly a power-versus-capability trade:

| Option | Power draw | Good for | Watch out for |
|--------|-----------|----------|---------------|
| [Raspberry Pi](/reference/raspberry-pi/) / SBC | ~5 W | File shares, small sites, sensors | Limited CPU & I/O |
| Old desktop PC | 40–100 W | Media, VMs, heavier apps | Electricity, noise |
| [NAS](/reference/network-attached-storage/) appliance | 10–30 W | Bulk storage with RAID | Less general-purpose |

A [single-board computer](/reference/single-board-computer/) or an old PC gathering dust makes a great starting point: cheap, low-power, and enough for file shares and small sites. In GopherTrunk terms a home machine can do both jobs at once — capture RF from an attached dongle *and* store and serve the decoded data, which a remote VPS cannot.

## Sources

[^wiki]: [Self-hosting (web services)](https://en.wikipedia.org/wiki/Self-hosting_(web_services)) — Wikipedia, on running services on hardware you own.
