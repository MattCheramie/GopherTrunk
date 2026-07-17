---
slug: gophertrunk-on-the-network
title: GopherTrunk on the network
description: How to reach GopherTrunk's web interface across your network — the localhost-versus-0.0.0.0 binding question, browsing to it on your LAN, reaching it remotely and safely with a VPN or SSH tunnel, streaming live call activity, and a sensible self-hosted setup.
keywords: gophertrunk web interface, remote access, LAN, stream audio, reverse proxy, VPN, scanner network, secure access, self-hosting, SSH tunnel
level: intermediate
status: full
prereq:
  - running-a-server
  - exposing-a-service-safely
faq:
  - q: How do I access GopherTrunk from another computer on my network?
    a: "GopherTrunk's web interface is a server, so from another device on the same LAN you browse to the host machine's IP address and the interface's port. This only works if GopherTrunk is bound to a network-facing address (like 0.0.0.0) rather than localhost, and if the host's firewall allows the port. Check the host's own address and see the interface's page for the port it listens on."
  - q: Can I access GopherTrunk over the internet?
    a: "Yes, but don't expose the web interface raw to the internet. The safe ways are to VPN into your home network and reach GopherTrunk as if you were local, or to SSH-tunnel to it. If it genuinely must be public, put it behind a reverse proxy that adds TLS and authentication — never a scanner UI on a bare public port."
  - q: Why can't other devices reach GopherTrunk even though it's running?
    a: "Usually it's bound to localhost, so it only accepts connections from the host machine itself, or a firewall is blocking the port. Confirm what address and port it's listening on, bind it to a network-facing address if you want LAN access, and open the port in the host's firewall."
  - q: Is it safe to leave GopherTrunk running all the time?
    a: "On a home LAN, yes — a good pattern is a small always-on box like a Raspberry Pi, kept LAN-only or reachable only through a VPN, and monitored so you notice if it stops. The risk comes from exposing it to the wider internet without TLS and authentication, not from running it continuously at home."
gophertrunk_links:
  - title: Web interface
    url: /web.html
    note: what the browser UI offers and the port it listens on.
  - title: Architecture
    url: /architecture.html
    note: how the daemon, decoders, and web server fit together.
---

# GopherTrunk on the network

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
GopherTrunk's **web interface is a server** — you reach it in a browser, so
everything in this path applies to it. On your **LAN**, other devices browse to
the host's address and port; **remotely**, you reach it through a VPN or SSH
tunnel, never raw on the open internet. The rule is simple: **secure it** the way
you'd secure any service. This is the payoff — put the whole path to work and
reach GopherTrunk across your network, and from afar if you need to, without
exposing it carelessly. New here? Start with
[running a server](/learn/networking/running-a-server/).
</div>

You've worked through addressing, ports, firewalls, VPNs, and how to run and
expose a service. This lesson threads all of it together on a real service you
care about: your scanner. Nothing below is new networking — it's the path applied.

## GopherTrunk's web interface is a server

You reach GopherTrunk in a **browser**, which tells you everything about its
shape: it's a [client-server](/learn/networking/clients-and-servers/) setup, with
your browser as the client and GopherTrunk as the server. Every question you'd
ask of any server applies here.

The first of those is the **binding** question. A server can listen on
`localhost` — accepting connections only from the host machine itself — or on a
network-facing address like `0.0.0.0`, which accepts connections from other
devices on the network. That single choice decides whether the rest of your
house can reach the interface at all. See
[running a server](/learn/networking/running-a-server/) for the localhost-versus-
everywhere trade-off, and GopherTrunk's own [web interface page](/web.html) for
how it's configured.

## Reaching it on your LAN

From another device in the house — a laptop, a phone, a tablet — you reach
GopherTrunk by browsing to the **host machine's
[IP address](/learn/networking/ip-addresses/)** and the interface's
**[port](/learn/networking/ports-and-sockets/)**. That's the whole trick: an
address to find the machine, a port to find the service on it.

Two things have to be true for it to work. First, GopherTrunk must actually be
**listening** on a network-facing address — you can confirm what's bound and
where with the tools from
[inspecting your network](/learn/networking/inspecting-your-network/). Second, the
host's [firewall](/learn/networking/firewalls/) must **allow** that port; a
service that's listening but firewalled off looks, from another device, exactly
like one that isn't running. Check both and LAN access falls into place.

## Reaching it remotely, safely

Wanting the interface from outside the house is natural — and the wrong way to
get it is to point the internet straight at it. **Do not expose a scanner UI raw
to the internet.** It's a service you didn't harden for the open web, and a bare
public port invites exactly the trouble
[exposing a service safely](/learn/networking/exposing-a-service-safely/) warns
about.

The safe approaches, in order of preference:

- **VPN into your home network** ([VPNs](/learn/networking/vpns/)). You land on
  your own LAN as if you were sitting on the couch, and reach GopherTrunk by its
  local address — nothing is exposed publicly at all.
- **SSH-tunnel to it**
  ([SSH tunnels & transfers](/learn/networking/ssh-tunnels-and-transfers/)).
  Forward the interface's port over an encrypted SSH connection and browse to it
  locally; quick, and needs nothing beyond SSH access to the host.
- **Only if it truly must be public**, put it behind a **reverse proxy** that
  adds **TLS** and **authentication**
  ([exposing a service safely](/learn/networking/exposing-a-service-safely/)) —
  never the raw UI on a public port.

## Streaming and live data

GopherTrunk isn't just static pages. **Live call activity** and **audio** to
other tools are **real-time, long-lived** connections — the browser or client
stays connected and data flows continuously, rather than one request-and-done.
That's the model behind
[WebSockets & real-time](/learn/networking/websockets-and-realtime/), and it
rides on the transports from [TCP & UDP](/learn/networking/tcp-and-udp/). Knowing
a stream is a persistent connection explains why it behaves differently from a
plain page load when you're debugging or planning what to expose.

## A sensible setup

Pull it together into a setup you can actually live with:

- Run GopherTrunk on a small always-on box — a **Raspberry Pi** is a classic fit
  ([self-hosting at home](/learn/networking/self-hosting-at-home/)).
- Keep it **LAN-only**, or reachable only through a **VPN**, so the wider
  internet never touches it directly.
- **Monitor** it — a scanner that quietly stopped decoding is worse than one
  you know is down. GopherTrunk's [architecture](/architecture.html) shows the
  moving parts worth watching.

That's the whole path doing useful work at once: addressing to find it, ports and
firewalls to reach it, a VPN or tunnel to reach it safely, and self-hosting sense
to keep it running.

## Where to go next

You can now reason about a networked service end to end — where it binds, how
other devices reach it, how to get to it from afar without exposing it, and how
its live streams behave. That's the same reasoning you'd apply to **debug** a
connection that won't open or **secure** one that shouldn't be public. GopherTrunk
was the worked example; the skill is general.

<div class="knowledge-check" data-quiz data-correct-msg="Right — reach it through a VPN or SSH tunnel; don't put the UI on the open internet." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to use GopherTrunk's web UI from your laptop elsewhere, safely. Best approach?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Forward the port on your router so the UI is reachable from any browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">VPN into your home network (or SSH-tunnel) — don't expose the UI raw to the internet</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turn off the host firewall so nothing blocks the connection</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk's **web interface is a server**; the **localhost-versus-0.0.0.0**
  binding decides whether other devices can reach it.
- On your **LAN**, browse to the **host's IP and the interface's port**; confirm
  what's listening and that the firewall allows it.
- **Remotely**, use a **VPN** or **SSH tunnel** — do not expose the scanner UI
  raw to the internet; a reverse proxy with **TLS and auth** only if it must be
  public.
- **Live call activity and audio** are real-time, long-lived connections
  (WebSockets over TCP/UDP), not plain page loads.
- A **sensible setup** is an always-on Pi, kept LAN-only or VPN-reachable, and
  monitored — the payoff of the whole path.

Next up: keep the [glossary](/learn/networking/glossary/) handy — and to run and harden the box your service lives on, the [Linux & the Command Line](/learn/linux-cli/) path covers the rest.
