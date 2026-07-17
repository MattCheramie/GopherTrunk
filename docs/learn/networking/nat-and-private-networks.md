---
slug: nat-and-private-networks
title: NAT & private networks
description: A plain-language guide to Network Address Translation — why private IP ranges exist, how your home router shares one public IP across every device, why outbound connections just work but inbound ones need setup, and how NAT relates to privacy, carrier-grade NAT, and IPv6.
keywords: NAT, network address translation, private IP, public IP, home router, port mapping, IPv4 exhaustion, carrier-grade NAT, IPv6, port forwarding
level: intermediate
status: full
prereq:
  - ip-addresses
faq:
  - q: What is NAT (Network Address Translation)?
    a: "NAT is what your home router does to let many devices share one public IP address. It rewrites the source address (and port) of outgoing packets to the router's single public IP, remembers the mapping, and reverses it for the replies coming back. To the wider internet, all your devices appear to come from one address."
  - q: What is the difference between a private and a public IP address?
    a: "A private IP address comes from a reserved range that any network can reuse internally — it is only meaningful inside your own LAN and is not routable on the open internet. A public IP address is globally unique and reachable from anywhere. Your devices have private addresses; your router holds the one public address the internet sees."
  - q: Why can my devices reach the internet but the internet cannot reach them?
    a: "Because a NAT mapping only exists after one of your devices sends a packet out. Outbound traffic creates the mapping the reply needs, so it flows freely. An unsolicited connection from the internet has no matching entry, so the router has nowhere to send it — which is why hosting a service at home needs port forwarding."
  - q: Does NAT protect my network like a firewall?
    a: "Partly, and by accident. NAT hides your internal devices behind one address and drops unsolicited inbound traffic because there is no mapping for it, which behaves a bit like a wall. But it is a side effect, not a security feature — NAT is not a real firewall and should not be relied on as one."
---

# NAT & private networks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
There aren't enough public addresses for every device, so your network uses
**private vs public IP** addresses: each device gets a **private** address that
only means something on your LAN, while your router holds **one shared public
address** the internet actually sees. **NAT** (Network Address Translation) is
the trick that makes this work — your router translates all your devices'
private addresses onto that single public IP. It's also why incoming
connections need extra setup: nothing outside knows how to reach a specific
device behind the shared address. If [IP addresses](/learn/networking/ip-addresses/)
are new to you, start there first.
</div>

Nearly every home and office network runs on this arrangement, usually without
anyone noticing. Once you see what NAT is doing, a lot of otherwise confusing
behaviour — why hosting is fiddly, why your devices all "share" an address —
suddenly makes sense.

## The problem

The original internet addressing scheme, IPv4, has room for about four billion
addresses. That sounded endless in the 1980s. It isn't: there are far more
phones, laptops, servers, and gadgets than there are IPv4 addresses to go
around. This shortage is called **IPv4 exhaustion**.

The fix was to stop giving every device a globally unique address. Instead,
certain [IP address](/learn/networking/ip-addresses/) ranges were set aside as
**private** — reserved for use inside local networks and never routed on the
open internet. Because they're never used publicly, every network on Earth can
reuse the same private ranges internally without colliding. Your home and your
neighbour's home can both hand out the same private addresses; they never meet
on the public internet, so it doesn't matter.

That solves the shortage, but it raises a question: if your devices only have
private addresses that the internet won't route, how do they reach the internet
at all? That's where NAT comes in.

## What NAT does

**NAT** — Network Address Translation — lives in your router, the one device
that sits between your private network and your ISP. It holds the single
**public IP** address that the outside world sees.

When one of your devices sends a packet out to the internet, the router:

1. **Rewrites the source address** on the packet, replacing the device's
   private address with the router's own public IP (and usually swapping the
   port number too, so it can tell connections apart).
2. **Remembers the mapping** — which internal device and port this outbound
   connection belongs to — in a translation table.
3. **Reverses the translation** when the reply comes back, looking up the table
   and rewriting the packet's destination back to the right private address, so
   it reaches the device that asked for it.

The result: **many devices share one public address**. To a web server out on
the internet, every request from your house looks like it came from the same
single IP, even though a dozen devices are behind it. The router quietly keeps
everyone's conversations straight.

## Outbound is easy, inbound is not

Here's the asymmetry that trips people up. Your devices can reach **out** to
anything on the internet freely — every outbound connection creates a mapping
in the router's table, and the reply follows that mapping straight home.

But the internet **can't reach in** to your devices unprompted. Say someone on
the internet tries to open a connection to your network out of the blue. It
arrives at your router's public IP with no matching entry in the translation
table — the router has no idea which internal device it's meant for, so it
simply drops it. There's no mapping until one of your devices makes one by
reaching out first.

That's exactly why running a service at home — a game server, a website, or
GopherTrunk's web interface reachable from outside — takes extra work. You have
to create the mapping yourself, deliberately, so inbound connections have
somewhere to go. That deliberate mapping is **port forwarding**, and it gets a
lesson of its own later in this path.

## A side effect: some privacy and safety

Because unsolicited inbound traffic has nowhere to go, NAT ends up doing
something it was never designed for: it **hides your internal devices** behind
one address and **blocks connections you didn't initiate**. An attacker
scanning the internet sees only your router's public IP, not the individual
machines behind it, and can't just connect to them.

This behaves a little like a wall around your network, and it's a genuine
benefit — but be careful how much you lean on it. NAT is **not a real
firewall**. It blocks unsolicited inbound traffic as a side effect of how
translation works, not because it's inspecting or deciding anything about
safety. A true firewall makes deliberate allow/deny decisions about traffic;
we'll cover firewalls properly later in this path. Treat NAT's protection as a
happy accident, not your security plan.

## Carrier-grade NAT & IPv6

Two wrinkles are worth knowing about.

Some ISPs don't even give you a public IP of your own. They put you behind
**carrier-grade NAT (CGNAT)** — a second layer of NAT at the ISP, where many
customers share a public address. Your router NATs your devices, and then the
ISP NATs your router again. Everything outbound still works, but hosting a
service becomes much harder or impossible, because you don't control the outer
mapping.

The longer-term answer is **IPv6**, a newer addressing scheme with so many
addresses that every device can have its own globally unique one again. With
IPv6 there's no shortage to work around, so NAT isn't needed to conserve
addresses — which changes the whole picture, restoring direct addressability
(though firewalls still decide what's actually allowed in).

<div class="knowledge-check" data-quiz data-correct-msg="Right — NAT rewrites private addresses onto the router's one public IP and tracks the mappings." markdown="0">
  <p class="knowledge-check__q">Quick check: how can all your home devices share one public IP address?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Each device is assigned its own public IP by the ISP</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">NAT translates their private addresses onto the router's single public IP</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The devices take turns using the internet one at a time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- IPv4 doesn't have enough addresses for every device, so networks use
  **private** address ranges internally and reuse them freely.
- Your router holds **one shared public IP**; your devices hold private
  addresses only meaningful on your LAN.
- **NAT** rewrites the source address (and port) of outgoing packets onto that
  public IP, remembers the mapping, and reverses it for the replies.
- **Outbound is easy, inbound is not**: replies follow existing mappings, but
  unsolicited inbound traffic has none — which is why hosting needs port
  forwarding.
- NAT incidentally **hides devices and blocks unsolicited traffic**, but it is
  **not a real firewall**.
- **Carrier-grade NAT** adds a second layer that complicates hosting; **IPv6**
  gives every device a real address and changes the picture.

Next up: [DHCP & joining a local network](/learn/networking/dhcp-and-local-networks/)
