---
slug: what-is-a-network
title: What is a network?
description: A plain-language introduction to computer networks — what nodes and links are, how a LAN differs from a WAN, why the internet is a network of networks, and how devices find each other by address and name.
keywords: computer network, LAN, WAN, internet, nodes, network of networks, router, connectivity, ISP, links
level: beginner
status: full
faq:
  - q: What is a computer network?
    a: A computer network is a set of devices connected together so they can share data. The devices are called nodes and the connections between them are called links. Any two or more computers that can exchange information over a wire or radio link form a network.
  - q: What is the difference between a LAN and a WAN?
    a: "A LAN (Local Area Network) covers a small area like a home or office and is usually owned by one person or organisation. A WAN (Wide Area Network) spans long distances and ties many separate networks together. Your home network is a LAN; the wider connections that reach across cities and countries are WANs."
  - q: Is the internet just one big network?
    a: "Not exactly — the internet is a network of networks. Countless independent networks, run by different companies and organisations, connect to each other through routers and internet service providers. No single entity owns all of it; it works because everyone agrees to interconnect."
  - q: How do devices on a network find each other?
    a: "Every device has an address so data can be delivered to the right place, much like a postal address. Because addresses are hard for people to remember, we also use names — a naming system translates a friendly name into the numeric address behind it."
---

# What is a network?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **network** is nothing more than devices connected together so they can share
data. A small one in your home or office is a **LAN** (Local Area Network); one
that stretches across long distances is a **WAN** (Wide Area Network). The
internet is the biggest network of all — really a **network of networks**, where
countless independent networks interconnect into one global system. Nobody owns
the whole thing; it works because everyone agrees to connect.
</div>

Before you can reason about web requests, servers, or security, it helps to be
crystal clear on the thing underneath all of it: the network. The idea is
simpler than the jargon suggests, and once it clicks the rest of this path falls
into place quickly.

## Nodes and links

A network has two ingredients. The first is **nodes** — the devices that take
part. Your laptop, your phone, a server in a data centre, and the little
**router** in your hallway are all nodes. Anything that can send or receive data
counts.

The second is **links** — the connections between nodes. A link might be an
ethernet cable, a Wi-Fi radio signal, or a strand of fibre-optic glass. The
physical form doesn't matter much at this level; what matters is that a link
carries data from one node to another.

Put them together and you have the whole definition: **a network is nodes plus
links, sharing data.** Two phones swapping a photo over Wi-Fi are a network. So
is a company with thousands of machines. The scale changes; the idea doesn't.

## LAN vs. WAN

Networks are often described by how much ground they cover.

A **LAN**, or Local Area Network, is small and local — the devices in your home,
a single office, or one school building. The nodes are close together, the links
are fast, and usually one person or organisation owns and controls the whole
thing. When your phone and laptop are on the same Wi-Fi and can see each other,
that's your LAN.

A **WAN**, or Wide Area Network, spans distance — across a city, a country, or
the planet. WANs stitch many separate LANs together over long-haul links. You
don't own a WAN; you connect to one. The neat mental picture: your LAN is your
own little world, and WANs are the roads that link your world to everyone
else's.

## The internet is a network of networks

Here's the part that surprises people. The internet is not one giant network
owned by one company. It's a **network of networks** — countless independent
networks that each agree to interconnect.

Your home LAN connects to your internet service provider's (ISP's) network. That
ISP connects to other ISPs and to large carrier networks. Businesses, universities,
and cloud providers all run their own networks and plug into the same web of
connections. The devices that pass data between these separate networks are
**routers**, and the whole tangle is held together by a shared agreement to
speak the same language and forward each other's traffic.

Because it's built this way, **no one owns the internet**. There's no central
switch to flip. It keeps working because interconnection is in everyone's
interest, and because the rules for connecting are open and agreed upon.

## Finding each other

If a network is going to deliver data to the right place, every device needs an
**address** — a unique label, much like a postal address, that says where data
should go. On the internet these are [IP addresses](/learn/networking/ip-addresses/).

But numeric addresses are awkward for people to remember, so we also use
**names**. When you type a friendly name like a website's, a naming system
quietly looks up the numeric address hiding behind it. That system is
[DNS](/learn/networking/dns/), and we'll come back to both ideas in detail later
in this path.

## Why developers care

Almost no useful program lives entirely on one machine anymore. An app fetches
data from a server, a game syncs with other players, a script calls an API in the
cloud — nearly every program you write ends up **talking to another machine over
a network**.

That means understanding the network isn't a networking-specialist topic; it's
part of understanding how your software actually connects to the world. When you
know how data moves from node to node, debugging a request that won't go through
stops being guesswork. We'll trace exactly that journey in
[how a web request works](/learn/networking/how-a-web-request-works/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the internet is many independent networks agreeing to interconnect." markdown="0">
  <p class="knowledge-check__q">Quick check: what is "the internet", in one phrase?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">One giant computer owned by a single company</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A network of networks connected together</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A single very long cable between every device</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **network** is just devices connected together to share data.
- The devices are **nodes**; the connections between them are **links**.
- A **LAN** is small and local; a **WAN** spans long distances and links networks together.
- The **internet** is a **network of networks** — independent networks interconnected by routers and ISPs, owned by no one.
- Devices find each other by **address**, and people use **names** that map to those addresses.
- Almost every program talks to another machine, so the network is core to how software connects.

Next up: [packets & switching](/learn/networking/packets-and-switching/)
