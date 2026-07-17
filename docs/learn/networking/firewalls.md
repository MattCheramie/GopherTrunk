---
slug: firewalls
title: Firewalls & controlling access
description: A plain-language guide to firewalls — how they allow or block traffic by port, address, and protocol, why default-deny is the safe policy, the difference between host and network firewalls, stateful connection tracking, open vs closed ports, and why NAT is not a real firewall.
keywords: firewall, default deny, open port, closed port, ufw, iptables, nftables, stateful firewall, allowlist, inbound traffic, attack surface, network security
level: intermediate
status: full
prereq:
  - ports-and-sockets
faq:
  - q: What does a firewall do?
    a: "A firewall inspects network traffic and decides whether to allow or block each packet based on rules — typically the port, the source or destination address, and the protocol. It is the gate that controls which of your listening ports the outside world is actually allowed to reach."
  - q: What is a default-deny firewall policy?
    a: "Default-deny means the firewall blocks all inbound traffic by default and only allows the specific ports a service genuinely needs. Instead of listing what to block, you list the short set of things to permit and everything else is refused. It is the safest starting point because every open port is attack surface."
  - q: Is NAT the same as a firewall?
    a: "No. NAT drops unsolicited inbound connections as a side effect of having no mapping for them, which looks a bit like protection, but it is not a security feature and can be bypassed. For real control over what reaches your machine you need an actual firewall with explicit rules."
  - q: What is a stateful firewall?
    a: "A stateful firewall remembers the connections your machine starts, so the replies to your own outbound traffic are allowed back automatically without a separate rule. You only have to write rules for new inbound connections, which keeps a default-deny policy simple to manage."
---

# Firewalls & controlling access

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **firewall** decides which network traffic is allowed to reach your machine and
which gets blocked, judged by [port](/learn/networking/ports-and-sockets/),
address, and protocol. The safe way to run one is **default-deny**: block
everything inbound and open only the ports you truly need. That makes the
difference between an **open port** (something is listening and permitted) and a
**closed port** (nothing gets through) a deliberate choice rather than an
accident. Every port you leave open is one more way in.
</div>

You can have a perfectly configured service and still get burned if the wrong
things can reach it across the network. A firewall is the layer that decides who
gets to knock on which door. Once you think of it as a gate with an explicit
guest list, the rest is straightforward.

## What a firewall does

A firewall sits between the network and your machine (or between two networks)
and inspects traffic as it passes. For each packet it asks a simple question:
does a rule allow this? It judges by **port**
([which service](/learn/networking/ports-and-sockets/) the traffic is aimed at),
by **address** (where the traffic is coming from or going to), and by **protocol**
(TCP, UDP, and so on). If a rule permits the packet it passes; otherwise it is
dropped.

In other words, a firewall is the gate on which of your ports the world can
actually reach. A service might be listening, but if the firewall doesn't allow
traffic to that port, nothing from outside gets in.

## Default-deny

The single most important idea in firewalls is **default-deny**: block all
inbound traffic by default, then open only the specific ports a service needs.

The alternative — allow everything and block the bad stuff — is a losing game,
because you can never enumerate every threat. Default-deny flips it around. You
write a short list of what to *permit* and everything else is refused
automatically. You only have to reason about the handful of things you actually
want reachable.

The reason this matters: **every open port is attack surface**. An open port is a
door someone can try. The fewer doors you leave open, the less there is to
attack, to misconfigure, or to forget about. Deny by default and the surface
stays as small as the list you deliberately allowed.

## Host vs network firewalls

Firewalls live in two places, and most setups use both.

A **host firewall** runs on the machine itself and controls what reaches that one
computer. On Linux this is tools like **ufw** (a friendly front-end), or the
lower-level **iptables** / **nftables** that do the actual filtering in the
kernel. A host firewall protects the individual machine even from other devices
on the same LAN.

A **network firewall** sits at the perimeter — usually built into your router —
and filters traffic entering or leaving the whole network before it reaches any
device. It guards everything behind it at once.

Both are commonly **stateful**: they track the connections your machine has
already started, so the **replies** to your own outbound traffic are recognised
and allowed back automatically. You don't write a rule for every response — you
only write rules for *new inbound* connections, and the firewall handles the
return path itself. That is what makes a strict default-deny policy livable.

## Open and closed ports

An **open** port means two things are true at once: a service is
[listening](/learn/networking/ports-and-sockets/) on it, and the firewall allows
traffic to reach it. Both are required. If you [run a
server](/learn/networking/running-a-server/) but the firewall blocks its port, it
stays unreachable no matter how healthy the service is — so a service you intend
to expose must have its port explicitly allowed.

A **closed** port is one nothing gets through — either nothing is listening, or
the firewall refuses the traffic. Closing ports you aren't using is free
security: it removes doors nobody needs. Every unused port you close is one less
piece of attack surface, which is default-deny applied port by port.

## Firewall vs NAT

It's tempting to think you're already protected because you sit behind
[NAT](/learn/networking/nat-and-private-networks/). NAT does drop unsolicited
inbound connections — but only as a *side effect* of having no mapping to send
them to, not because it decided the traffic was unwanted.

That is not the same as a firewall. NAT has no allow-list, makes no security
decision, and can be traversed in ways a real firewall would refuse. Treat NAT's
inbound-blocking as a happy accident, not a defence. For actual control over what
reaches your machines, rely on a **real firewall** with explicit rules.

<div class="knowledge-check" data-quiz data-correct-msg="Right — deny everything by default and open only the specific ports you need." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the safest default policy for a firewall?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Allow all inbound traffic and block known-bad ports</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Deny by default and allow only the ports you actually need</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Rely on NAT to drop unwanted connections</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## In practice

Before you expose anything to the network, make two decisions on purpose: exactly
**which ports** you're opening, and **to whom** — the whole internet, just your
LAN, or a single trusted address. That deliberate, minimal list is the heart of
running a service without getting burned, and it's the groundwork for [exposing a
service safely](/learn/networking/exposing-a-service-safely/) and the wider
[network security basics](/learn/networking/network-security-basics/) covered
later in this path.

## Recap

- A **firewall** allows or blocks traffic by **port, address, and protocol** — it
  gates which of your ports the world can reach.
- **Default-deny** is the safe policy: block all inbound, open only what a service
  truly needs, because **every open port is attack surface**.
- A **host firewall** (ufw, iptables/nftables) protects one machine; a **network
  firewall** protects everything behind the perimeter.
- **Stateful** firewalls track your own connections, so replies are allowed back
  without extra rules.
- An **open** port needs both a listener and an allow rule; closing unused ports
  shrinks exposure.
- **NAT is not a firewall** — it blocks unsolicited inbound only by accident, so
  rely on a real firewall for security.

Next up: [port forwarding & dynamic DNS](/learn/networking/port-forwarding-and-dynamic-dns/)
