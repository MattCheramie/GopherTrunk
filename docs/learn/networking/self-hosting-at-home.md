---
slug: self-hosting-at-home
title: Self-hosting on a home network
description: Running your own services at home gives control, privacy, and low cost — but adds uptime, addressing, and security work. Learn the pitfalls (dynamic IP, CGNAT) and why LAN-only reached over a VPN is the safest default.
keywords: self hosting, home server, home lab, dynamic IP, CGNAT, reverse proxy, LAN only, Raspberry Pi, static IP reservation, port forwarding, VPN access, home network security
level: intermediate
status: full
prereq:
  - running-a-server
faq:
  - q: "Is it safe to host a service from my home network?"
    a: "It's safe when you keep it on your **LAN** and reach it from outside over a **VPN** or an SSH tunnel — nothing is exposed to the public internet, so there is nothing for strangers to attack. It becomes risky the moment you forward a port and publish a service directly, because you then own every patch, password, and misconfiguration on a machine anyone can reach. Expose publicly only when you genuinely must, and do it properly."
  - q: "Why can't I reach my home server from the internet?"
    a: "Usually because of your **public IP address**. Residential ISPs hand out IPs that change (a **dynamic IP**), and many now place customers behind **CGNAT** (carrier-grade NAT), where you share one public address with other subscribers and cannot accept inbound connections at all. Port forwarding won't help behind CGNAT. A VPN that dials out from the home box, or a tunnel service, sidesteps the whole problem."
  - q: "What hardware do I need to self-host at home?"
    a: "Any always-on machine with enough power for your services. A **Raspberry Pi** or a repurposed old PC is ideal — cheap, low-power, and happy to run around the clock. Give it a **static or DHCP-reserved address** so it doesn't move on your network, and you have the foundation for a home lab."
  - q: "What is CGNAT and why does it matter for self-hosting?"
    a: "**CGNAT** (carrier-grade NAT) is when your ISP shares a single public IP among many customers instead of giving you your own. It saves the ISP address space, but it means inbound connections from the internet can't be routed to you — port forwarding on your router does nothing, because your router isn't the edge. If you're behind CGNAT, keep services LAN-only or use a VPN/tunnel that connects outward."
---

# Self-hosting on a home network

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Self-hosting** means running your own services on hardware you own at home —
you get control, privacy, and low cost. The big choice is **LAN-only vs. public**:
keep a service on your local network and it stays simple and safe; publish it to the
internet and you take on **the pitfalls** — uptime, a shifting public address, ISP
limits, and a permanent security job. Much of that work disappears the moment you
decide the service only needs to be reachable from inside your own network. Start from
[running a server](/learn/networking/running-a-server/) and add the home-network layer.
</div>

You already know how to run a server. Doing it *at home* changes almost nothing about
the software and almost everything about the network around it. This lesson is about
that network layer: how to place a box on your LAN, why the internet doesn't
automatically reach it, and how to decide whether it should reach it at all.

## Why self-host

People self-host for four reasons that all point the same way — **you own the box**:

- **Control** — you decide what runs, which version, and when it updates. No provider
  changes the rules on you.
- **Privacy** — your data lives on hardware in your home, not in someone else's cloud.
  For an SDR feed, logs, or family files, that matters.
- **Cost** — after the hardware, there's no monthly bill; you're using internet you
  already pay for.
- **Learning** — a home server teaches real skills at low stakes.

If you want the fuller case for owning the machine, the
[home servers](/learn/intro-hardware/home-servers/) lesson covers what people run and
why the cloud can't do some of it.

## The device

A home server is just **an always-on machine**. It doesn't need to be special: a
[Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) or a repurposed old PC
is ideal — cheap, quiet, low-power, and content to run day and night for pennies.

The one networking step that matters up front: give it a **static or DHCP-reserved
address** so it doesn't move. By default your router hands out addresses by
[DHCP](/learn/networking/dhcp-and-local-networks/), and a lease can change — which
breaks every bookmark, config, and firewall rule that pointed at the old one. Reserve
an address for the server (or set one statically) and it stays put. Everything else
you build assumes the box has a fixed home on the LAN.

## The pitfalls nobody mentions

The cloud quietly handled a lot of things for you. At home, they become yours:

- **Uptime and power** — the server is down when your power or internet is down. There's
  no data center's redundant feed behind it; a summer storm is now your outage.
- **A changing public IP** — residential ISPs hand out addresses that change over time
  (a **dynamic IP**), so "the address of my house" isn't a fixed thing you can point
  people at. [Dynamic DNS](/learn/networking/port-forwarding-and-dynamic-dns/) papers
  over this by tracking the current address under a name.
- **ISP limits and CGNAT** — many ISPs now place you behind **CGNAT** (carrier-grade
  NAT), sharing one public IP across many customers. Behind CGNAT you *cannot* accept
  inbound connections at all — port forwarding on your router does nothing, because your
  router is no longer the edge of the network. Some ISP terms also forbid running public
  servers from a residential line.
- **The security burden** — anything you expose to the internet is a target from the
  first minute, and keeping it safe (patches, passwords, TLS, monitoring) is now an
  ongoing job with no one else on call.

None of these are dealbreakers. But notice that **almost every one of them only bites
when you expose the service publicly**. Which leads to the most important decision.

## LAN-only vs. internet-exposed

The safest default is the simplest: **keep the service on your LAN** and don't publish
it to the internet at all. A dashboard, a media library, or a GopherTrunk box that only
*you* use has no reason to be reachable by strangers.

When you're away from home and still want in, don't open a hole in your router — reach
back *into* your network instead:

- **A VPN** — run a [VPN](/learn/networking/vpns/) endpoint on your network and dial in.
  Once connected, your phone or laptop behaves as if it's on the home LAN, and every
  service is reachable exactly as it is at home. Nothing is exposed publicly.
- **An SSH tunnel** — for a single service, an
  [SSH tunnel](/learn/networking/ssh-tunnels-and-transfers/) forwards one port over an
  encrypted connection you already trust.

Both of these connect *outward* or authenticate first, so they work even behind CGNAT
and leave nothing open to scan. Only expose a service **publicly** when you genuinely
must — something the wider world needs to reach — and when you do,
[do it properly](/learn/networking/exposing-a-service-safely/) rather than forwarding a
raw port and hoping.

## A sensible home setup

A dependable home lab doesn't take much. Build it in this order:

1. **A reserved IP** for the server, so it never moves.
2. **A reverse proxy with TLS** in front of anything web-facing, so services sit behind
   one hardened entry point speaking HTTPS rather than each exposing its own port. The
   [exposing a service safely](/learn/networking/exposing-a-service-safely/) lesson walks
   through this.
3. **Backups** — self-hosting means *you* are the backup plan; a failed disk shouldn't
   cost you the data.
4. **Monitoring** — a simple check that tells you when the box or a service is down,
   before you find out the hard way.

Start small. One reserved address and one service you actually use will teach you more
than a rack of half-configured containers.

<div class="knowledge-check" data-quiz data-correct-msg="Right — LAN-only plus a VPN gives you full access with nothing exposed to the public internet." markdown="0">
  <p class="knowledge-check__q">Quick check: the safest way to reach a home service that only you use is...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Forward the service's port on your router so it's reachable from anywhere</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Keep it LAN-only and connect in over a VPN, rather than exposing it publicly</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Put it on a dynamic-DNS name so the public address always resolves</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Self-hosting** runs your own services on hardware you own — control, privacy, low
  cost, and real learning.
- The device is just an **always-on machine** (a Raspberry Pi or old PC); give it a
  **static or DHCP-reserved address** so it stays put.
- **The pitfalls** — power/uptime, a changing public IP, ISP limits and **CGNAT**, and a
  permanent security job — mostly only bite when you expose a service publicly.
- **LAN-only is the safe default**: reach your services from outside over a **VPN** or an
  SSH tunnel instead of opening ports.
- **Expose publicly only when you must**, and then use a reverse proxy with TLS, backups,
  and monitoring.

Next up: [GopherTrunk on the network](/learn/networking/gophertrunk-on-the-network/)
