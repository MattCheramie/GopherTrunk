---
slug: dhcp-and-local-networks
title: DHCP & joining a local network
description: When a device joins a network it's handed everything it needs to communicate — an IP address, a default gateway, DNS servers, and a subnet mask — automatically. Learn what DHCP does, why the default gateway is your exit to other networks, how the subnet mask decides local vs remote, and when to use a static IP instead.
keywords: DHCP, default gateway, subnet mask, static IP, local network, router, lease, IP assignment, DHCP reservation, dynamic addressing, DNS server, joining a network
level: beginner
status: full
prereq:
  - ip-addresses
faq:
  - q: "What is DHCP?"
    a: "**DHCP** (the Dynamic Host Configuration Protocol) is how a device gets its network settings automatically. When you join a network, the device asks for an address, and a **DHCP server** — usually your router — leases it an **IP address** for a while and hands back the other settings it needs: the **default gateway**, **DNS servers**, and the **subnet mask**. No manual typing required."
  - q: "What is a default gateway?"
    a: "The **default gateway** is the address your device sends packets to when they're bound for a machine outside your local network — in a home it's your router. Traffic for your own subnet goes directly; everything else goes to the gateway, which forwards it on toward its destination. It's your network's exit door to the rest of the internet."
  - q: "What is a subnet mask?"
    a: "A **subnet mask** tells a device which IP addresses are on its **local network** and which are remote. The device compares a destination against the mask: if it falls inside the local subnet it talks directly, and if it falls outside it sends the packet to the **default gateway** instead. It's how a device sorts local traffic from everything else."
  - q: "Should I use a static IP or DHCP?"
    a: "**DHCP** is the right default — it's automatic and hands out addresses as devices come and go. But a device that others need to reach at a known address, like a server or anything you port-forward to, is easier to manage with a fixed address. Use a **static IP** or, cleaner, a **DHCP reservation** so the router always leases that device the same address."
---
# DHCP & joining a local network

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a device joins a network, it's handed everything it needs to communicate —
automatically. **DHCP** (usually your router) leases it an
[IP address](/learn/networking/ip-addresses/) and tells it the rest: the
**default gateway** to reach other networks, the DNS servers, and the subnet
mask. This is **dynamic** addressing, and it's why plugging in "just works." The
exception is anything that needs a fixed address others can rely on — there you
choose a **static** address instead.
</div>

You've never had to type in an IP address to get online, and that's not an
accident. Behind every "connected" notification is a short automatic conversation
that sets your device up completely. Once you see what's being handed over, a lot
of network settings stop looking mysterious.

## What happens when you connect

Join a Wi-Fi network or plug in an ethernet cable, and within a couple of seconds
your device has an [IP address](/learn/networking/ip-addresses/), a **default
gateway**, one or more **DNS servers**, and a **subnet mask** — all without you
touching anything. Those four pieces are the minimum a device needs to send a
packet anywhere: an identity, an exit route, a way to look up names, and a way to
tell local from remote.

The system that arranges all of it is **DHCP**.

## DHCP

**DHCP** stands for the **Dynamic Host Configuration Protocol**. The name is a
mouthful, but the idea is simple: instead of every device being configured by
hand, the device **asks** and the network **answers**.

When your device joins, it broadcasts a request essentially saying "I'm new here —
does anyone have an address for me?" A **DHCP server** — in a home network that's
almost always built into your router — replies with an available **IP address**
and hands over the other settings at the same time. The address comes as a
**lease**: the device gets to use it for a set period, and renews the lease to
keep it. When a device leaves and the lease expires, that address can be handed to
someone else.

That's the whole trick. No two devices are told to use the same address, and none
of it requires you to know a single number.

## The default gateway

Your device can talk directly to other machines on its own local network, but the
internet is billions of machines on *other* networks. The **default gateway** is
how it reaches them.

The gateway is your network's **exit door** — in a home, it's your router. When
your device has a packet to send, it checks whether the destination is on the
[local network](/learn/networking/packets-and-switching/). If it is, the packet
goes straight there. If it isn't, the device hands the packet to the default
gateway, which **forwards it on** toward the wider internet, hop by hop. Every
request to a website or a server you don't host yourself leaves through the
gateway.

Because that forwarding is exactly the boundary between your private addresses and
the public internet, it's also where address translation happens — see
[NAT & private networks](/learn/networking/nat-and-private-networks/).

## Subnet mask

How does a device *know* whether a destination is local or remote? That's the job
of the **subnet mask**.

The mask splits an IP address into two parts: the bit that identifies your
**local network** (the subnet) and the bit that identifies a device on it. When
your device wants to send a packet, it compares the destination against its own
address using the mask. If the network part matches, the destination is **local** —
talk to it directly. If it doesn't match, the destination is **remote** — send the
packet to the **default gateway** instead.

So the subnet mask is the quiet rule behind every send decision: local traffic
goes direct, everything else goes out the gateway. Subnets are covered more fully
in [IP addresses](/learn/networking/ip-addresses/).

## Static vs dynamic addresses

DHCP's automatic, **dynamic** addressing is exactly what you want for laptops,
phones, and anything that comes and goes — you never think about it. The catch is
that a dynamically assigned address can **change** the next time a device
reconnects or its lease expires.

For most devices that's fine. But some devices need to be found at a **known,
unchanging address**: a server, a network printer, or anything you'll
**port-forward** to from outside. If its address kept shifting, everything
pointing at it would break. For those you want a fixed address — either a
**static IP** configured on the device, or, cleaner, a **DHCP reservation** where
the router always leases that same device the same address.

You'll rely on this directly later, when you set up
[port forwarding & dynamic DNS](/learn/networking/port-forwarding-and-dynamic-dns/)
and [run a server](/learn/networking/running-a-server/) of your own.

<div class="knowledge-check" data-quiz data-correct-msg="Right — DHCP, usually running on your router, leases the address automatically." markdown="0">
  <p class="knowledge-check__q">Quick check: what hands your laptop an IP address the moment it joins the Wi-Fi?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You type it in by hand each time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">DHCP (usually your router)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The DNS server picks one at random</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Joining a network hands your device everything it needs **automatically**.
- **DHCP** — usually your router — **leases** an IP address and the other settings.
- The **default gateway** is your exit to every other network; non-local packets go there.
- The **subnet mask** decides which addresses are **local** (talk direct) vs **remote** (via the gateway).
- **Dynamic** addressing suits most devices; servers and port-forward targets want a **static IP** or **DHCP reservation**.

Next up: Module 3 turns to the protocols you build on — [TCP vs. UDP](/learn/networking/tcp-and-udp/)
