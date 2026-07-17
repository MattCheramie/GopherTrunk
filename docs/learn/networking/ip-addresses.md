---
slug: ip-addresses
title: IP addresses & subnets
description: A plain-language guide to IP addressing — what an IP address is, how IPv4 and IPv6 differ, the split between public and private addresses, and what a subnet and CIDR notation like /24 actually mean.
keywords: IP address, IPv4, IPv6, subnet, CIDR, private IP, public IP, network mask, dotted quad, address space
level: intermediate
status: full
prereq:
  - what-is-a-network
faq:
  - q: What is an IP address?
    a: "An IP address is a number that identifies a device on a network so data can be routed to it. Just as a postal address tells the mail where to go, an IP address tells the network which device should receive a packet. Every device that talks on a network has one."
  - q: What is the difference between IPv4 and IPv6?
    a: "IPv4 uses 32-bit addresses written as four numbers like 192.0.2.10, which allows about four billion addresses. IPv6 uses much longer 128-bit addresses to provide a vastly larger space, because the world ran short of IPv4 addresses. The two coexist, and most devices speak both."
  - q: What is a private IP address?
    a: "A private IP address comes from a reserved range — 10.0.0.0/8, 172.16.0.0/12, or 192.168.0.0/16 — that is meant for use inside home and office networks. These addresses are reused on countless separate networks and are not reachable directly from the internet, unlike globally routable public addresses."
  - q: What does a /24 mean in an IP address?
    a: "The /24 is CIDR notation: it says the first 24 bits of the address are the network part and the rest identify hosts within it. For 192.168.1.0/24 that leaves the last number free, so the block covers 192.168.1.0 through 192.168.1.255 — a single small network of addresses."
---

# IP addresses & subnets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **IP address** is a number that identifies a device on a network so packets
can be delivered to it. There are two versions in use: **IPv4**, written as four
numbers like `192.0.2.10`, and the much larger **IPv6**. Addresses split into
**public** ones, reachable across the internet, and **private** ones reused
inside home and office networks. A **subnet** is a block of addresses, and
**CIDR** notation like `/24` says how much of the address is the network part.
New to networks? Start with [what is a network?](/learn/networking/what-is-a-network/).
</div>

Once you know a network is just [nodes and links sharing
data](/learn/networking/what-is-a-network/), the next question is how any one
device gets found among millions of others. The answer is addressing, and it's
the foundation the rest of this module builds on.

## What an IP address is

An **IP address** is a number that identifies a device on a network so that data
can be routed to the right place. It plays the same role a postal address plays
for a letter: when the network breaks your data into
[packets](/learn/networking/packets-and-switching/), each packet carries a
destination address, and the routers along the way use it to move the packet
closer to that device.

Every device that takes part in a network — your laptop, your phone, a server,
the router in your hallway — has at least one IP address. Without it, there would
be no way to say *where* a packet should go, and delivery would be impossible.

## IPv4 and IPv6

There are two versions of IP addressing in everyday use.

**IPv4** is the original and still the most common. An IPv4 address is 32 bits,
written as four numbers separated by dots — a **dotted quad** — where each number
ranges from 0 to 255, like `192.0.2.10`. That format allows about **four billion**
distinct addresses. When IPv4 was designed that seemed like plenty, but with
billions of phones, computers, and connected gadgets, the world ran out.

**IPv6** exists to solve that shortage. An IPv6 address is 128 bits, written as
groups of hexadecimal digits separated by colons, and the space it provides is
staggeringly larger — enough that running out is no longer a practical concern.
The addresses are longer and less friendly to read, which is one reason we lean
on names (covered in the [DNS](/learn/networking/dns/) lesson) rather than typing
them by hand.

The two **coexist**. You don't have to choose one; most modern devices and
networks speak both at once, using IPv6 where it's available and IPv4 where it
isn't. For learning the concepts, IPv4's dotted quad is the easier picture to
hold in your head, so the examples here use it.

## Public vs. private addresses

Not every IP address is meant to be reachable from anywhere.

A **public** address is **globally routable** — it's unique across the whole
internet, and packets from anywhere can be directed to it. The address of a
public website or a mail server is a public address.

A **private** address comes from one of three reserved ranges set aside for use
inside local networks:

- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`

These ranges are **reused** on countless separate home and office networks at the
same time — the `192.168.1.x` addresses on your network and on your neighbour's
are completely independent. Because they aren't unique across the internet,
private addresses **aren't reachable directly** from the outside. Something has to
translate between the private addresses inside a network and the single public
address the network shows to the world; that trick is
[NAT](/learn/networking/nat-and-private-networks/), and it gets its own lesson.

## Subnets and CIDR

A **subnet** is simply a block of addresses that belong to the same local
network. Splitting the whole address space into subnets is how large networks
stay organised and how routers know which addresses live where.

To describe a subnet compactly we use **CIDR notation**: an address followed by a
slash and a number, like `/24`. That number says **how many bits at the front of
the address are the network part** — shared by every device in the subnet — while
the remaining bits are the **host part** that identifies individual devices.

Take `192.168.1.0/24`. The `/24` means the first 24 bits (the `192.168.1`
portion) are fixed as the network part, leaving the last 8 bits free for hosts.
So this subnet covers the addresses `192.168.1.0` through `192.168.1.255` — a
single small network of 256 addresses, of which the usable ones sit in between. A
smaller number after the slash, like `/16`, fixes fewer bits and so covers a
larger block; a larger number covers a smaller one. That's as far into the math
as you need to go for now: **fewer network bits means a bigger block**.

## How a device gets one

A device can get its IP address in one of two ways.

A **static** address is set **by hand** — you configure the device with a
specific address and it keeps it. Servers and some fixed equipment are often
given static addresses so they're always found in the same place.

A **dynamic** address is **handed out automatically** when the device joins the
network, so you don't have to configure anything. This is what happens the moment
your phone connects to Wi-Fi — a service quietly assigns it an address from the
local subnet. That service is DHCP, and the
[DHCP & joining a local network](/learn/networking/dhcp-and-local-networks/)
lesson walks through exactly how it works.

<div class="knowledge-check" data-quiz data-correct-msg="Right — 192.168.x.x is one of the reserved private ranges." markdown="0">
  <p class="knowledge-check__q">Quick check: which of these is a private address?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">198.51.100.7</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">192.168.1.10</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">203.0.113.20</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **IP address** is a number that identifies a device so packets can be routed to it.
- **IPv4** is a dotted quad like `192.0.2.10` with about four billion addresses; **IPv6** is far larger, and the two coexist.
- **Public** addresses are globally routable; **private** ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) are reused inside local networks and aren't reachable directly.
- A **subnet** is a block of addresses; **CIDR** like `/24` says how many bits are the network part vs. the host part.
- `192.168.1.0/24` covers `192.168.1.0`–`192.168.1.255` — one small network.
- A device gets its address either **statically** (by hand) or **dynamically** (handed out automatically).

Next up: [DNS — names to addresses](/learn/networking/dns/)
