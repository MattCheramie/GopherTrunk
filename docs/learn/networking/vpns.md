---
slug: vpns
title: VPNs & tunnels
description: A plain-language guide to VPNs — what an encrypted tunnel actually is, what a VPN hides from your ISP and local network and what it doesn't, remote-access versus commercial privacy VPNs, and the tech behind them like WireGuard and OpenVPN.
keywords: VPN, tunnel, WireGuard, OpenVPN, remote access, encryption, privacy, exit node
level: intermediate
status: full
prereq:
  - ip-addresses
faq:
  - q: What does a VPN actually do?
    a: "A VPN wraps your network traffic in an encrypted tunnel to a VPN server. Your local network and ISP can no longer read the contents or destinations of that traffic — they just see an encrypted connection to the server. From the server, your traffic continues to the internet, so websites see the server's address instead of yours."
  - q: Does a VPN make me anonymous?
    a: "No. A VPN hides your traffic from your ISP and local network, but the VPN provider can see everything your ISP used to, and the sites you visit still see the traffic arriving from the server. You are moving trust from your ISP to the provider, not becoming anonymous — and any account you log into still identifies you."
  - q: What is the difference between a remote-access VPN and a privacy VPN?
    a: "A remote-access VPN lets you reach a private network — a company LAN or your home network — as if you were physically there, which is a safer way to reach home services than opening ports. A commercial privacy VPN instead routes your general internet traffic through a provider's server to shift trust away from your ISP and change your apparent location."
  - q: Is WireGuard or OpenVPN better?
    a: "Both build an encrypted tunnel and both work well. WireGuard is newer, with a small, simple codebase that is fast and easy to configure, which is why it is often the default choice today. OpenVPN is older and more established, with broad support and flexible options. For a self-hosted tunnel into your home LAN, WireGuard is usually the easier place to start."
---

# VPNs & tunnels

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **VPN** wraps your network traffic in an **encrypted tunnel** to a VPN server;
from there your traffic continues to the internet, so it appears to come from the
server's address. That **hides** the contents and destinations of your traffic
from your local network and ISP — they see only the tunnel — and changes your
apparent location. What it **doesn't** do is make you anonymous: you're moving
trust from your ISP to the VPN provider, and logged-in accounts still identify
you. New to addressing? Start with [IP addresses & subnets](/learn/networking/ip-addresses/).
</div>

"Get a VPN" is common advice, and it's often good advice — but a VPN is
frequently sold as something it isn't. Once you can picture the tunnel and where
it begins and ends, it's easy to reason about exactly what a VPN protects and
what it leaves exposed.

## What a VPN actually is

A **VPN** — virtual private network — is an **encrypted tunnel** between your
device and a VPN server. Your normal network traffic, instead of going straight
out to the internet, travels **inside** that tunnel to the server first. At the
server the traffic leaves the tunnel and continues to wherever it was headed,
carrying the server's [IP address](/learn/networking/ip-addresses/) rather than
yours.

The word "tunnel" is a good picture: anything travelling inside it is wrapped in
[encryption](/learn/networking/tls-and-https/), so whoever is standing outside
the tunnel can see that it exists but not what's moving through it. The server is
the far end where your traffic emerges — often called the **exit node**, because
that's the point from which the wider internet sees it.

## What it hides

Because your traffic is encrypted from your device all the way to the server,
the parties **between** you and the server can no longer see inside it. Your
**local network** and your **ISP** used to see every site you connected to and,
where the connection wasn't already encrypted, the contents too. With a VPN, all
they see is an encrypted tunnel to one server. They know *that* you're using a
VPN, but not *what* you're doing through it.

The other thing that changes is your **apparent address**. Because your traffic
exits from the server, the sites and services you reach see the **server's**
IP and location, not yours. That's what lets a VPN make you appear to be
somewhere you're not — a common reason people reach for one.

## What it does NOT do

This is where VPNs are oversold, so be clear-eyed about it. A VPN is **not
anonymity**.

You haven't removed the party that can see your traffic — you've **changed who it
is**. Everything your ISP used to see, the **VPN provider** now sees instead,
because your traffic emerges unwrapped at their server. You are **moving trust**
from your ISP to the provider, and that's only a win if you trust the provider
more. A provider's promise not to keep logs is exactly that: a promise.

The destination end doesn't go blind either. The **website or service** you reach
still receives your traffic — it simply arrives from the server's address. And
the moment you **log into an account**, that account identifies you regardless of
which address the traffic came from. A VPN changes the path your traffic takes;
it doesn't erase who you are.

## Two common uses

VPNs get reached for in two quite different situations.

A **remote-access VPN** lets you join a private network from afar and use it as
if you were **physically there**. Connect to your company's VPN and you can reach
internal servers on its LAN; connect to one running on your home network and your
laptop behaves as though it's on the home [subnet](/learn/networking/ip-addresses/),
able to reach devices that are otherwise invisible from the internet. For home
services this is a far **safer** approach than
[forwarding ports](/learn/networking/port-forwarding-and-dynamic-dns/) — instead
of poking a hole straight to a service, you expose only the encrypted tunnel and
reach everything through it.

A commercial **"privacy" VPN** is the other use: it routes your general internet
traffic through a provider's server to **shift trust** away from your ISP and
**change your apparent location**. That's the kind advertised everywhere, and the
"what it doesn't do" section above is really about this use.

## The tech

Two pieces of software dominate.

**WireGuard** is the modern choice: a small, simple codebase that's fast and
straightforward to configure, which is why it's increasingly the default.
**OpenVPN** is the older, well-established option, with broad support and a lot
of flexibility. Both build the same thing — an encrypted tunnel — so the choice
is mostly about simplicity versus familiarity.

A concrete example ties it together: a **self-hosted WireGuard** server running
on your home LAN. From anywhere, your phone or laptop dials the tunnel, lands on
the home network, and can reach a scanner's web interface or a file share as if
it were sitting on the couch — no service exposed to the open internet. That's a
building block the [running your own service at home](/learn/networking/self-hosting-at-home/)
lesson leans on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a VPN hides your traffic from the local network and ISP, but it isn't anonymity." markdown="0">
  <p class="knowledge-check__q">Quick check: a VPN primarily provides…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Complete anonymity that no one can trace</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An encrypted tunnel that hides your traffic from the local network/ISP — not total anonymity</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Faster internet by removing your ISP from the path</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **VPN** is an **encrypted tunnel** from your device to a VPN server; your traffic exits from that server's address.
- It **hides** the contents and destinations of your traffic from your **local network and ISP**, who see only the tunnel, and changes your **apparent location**.
- It is **not anonymity**: the provider can see what your ISP used to, destinations still receive your traffic, and logged-in accounts still identify you.
- You're **moving trust** from your ISP to the VPN provider — worthwhile only if you trust the provider more.
- **Remote-access** VPNs reach a private LAN as if you were local (safer than port forwarding); **privacy** VPNs shift trust and location.
- **WireGuard** (modern, simple, fast) and **OpenVPN** are the common tools; a self-hosted WireGuard into your home LAN is a tidy example.

Next up: [proxies, reverse proxies & load balancers](/learn/networking/proxies-and-load-balancers/)
