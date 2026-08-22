---
slug: appliance-networking
title: Appliance networking & access
description: Reach the GopherTrunk web console from any device in the house, keep the daemon private on the LAN, and — if you must reach it remotely — do it through a VPN or SSH tunnel, never a raw port-forward.
keywords: gophertrunk web console access, lan only service, never expose to internet, vpn home network, ssh tunnel, port forward risk, wireguard, appliance security
level: intermediate
status: full
prereq:
  - networking-your-board
  - installing-gophertrunk-on-a-pi
---

# Appliance networking & access

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The finished appliance serves its **web console to the whole LAN** — every
laptop, phone, and tablet in the house, via the board's pinned name/address. The
cardinal rule: **never expose the console raw to the internet.** No router
port-forward, ever — a home-appliance web service isn't hardened for a hostile
internet, and scanners find forwarded ports in hours. Remote access done right
rides an **encrypted, authenticated tunnel you already trust**: a **VPN** into
your home network (the comfortable way) or an **SSH tunnel** (the zero-new-parts
way). Same console, safe road.
</div>

The scanner works; now the house gets access — and the internet, pointedly,
does not. This lesson finishes the networking story Unit 3 started: consumption
on the LAN, and the two safe patterns for listening from anywhere.

## How does the household reach the console?

Groundwork already laid, this is the harvest. The daemon listens on its
configured port; anyone on your network opens:

```text
http://scanner.local:8080     ← mDNS name, most devices
http://192.168.1.50:8080      ← the pinned IP, works from everything
```

The [DHCP reservation](/learn/embedded/networking-your-board/) makes the address
permanent; bookmark it on the household's devices (phone browsers happily add it
to home screens, making the scanner feel like an app). This one-server,
many-clients shape is the standard pattern for every home service —
[clients &amp; servers](/learn/networking/clients-and-servers/) if you want the
theory. Live listening, call history, and system health all travel through this
one HTTP port, so "can I browse to it?" is also your first health probe.

## Why is a port-forward the wrong kind of easy?

The tempting move: forward router port 8080 to the Pi, listen from work. Resist
it, permanently. What that click actually does: your console joins the public
internet, where **automated scanners sweep the entire address space
continuously** — discovery in hours, not months, obscurity irrelevant. And what
they find is a hobby daemon's web interface: built for a *trusted home network*,
not hardened, penetration-tested, and patch-raced like internet-edge software
must be. Any flaw in it — or in the pre-auth surface of anything else exposed —
hands over a machine *inside* your network
([Users, permissions &amp; updates](/learn/embedded/users-and-updates/)'s
nightmare, delivered). The asymmetry is total: seconds to set up, one CVE from
disaster, and a better option exists that costs fifteen minutes once.

> Rule of thumb: nothing on the appliance should be reachable from the internet
> — not the console, not SSH. Remote access means *you tunnel in*, never
> *services face out*.

## What's the right way to listen from anywhere?

Both safe patterns share one idea: **one hardened, encrypted, authenticated
front door**, with the console staying LAN-only behind it.

**A VPN into your home network** — the comfortable way. Modern VPNs (WireGuard
at the base; overlay tools built on it make setup nearly one-click) put your
phone or laptop *virtually on the LAN*: keys exchanged once, then
`scanner.local` works from a beach the same as from the sofa — console, SSH,
everything, through one cryptographically authenticated tunnel. Many home
routers can host the VPN themselves. Setup lives in the Networking module
([VPNs](/learn/networking/vpns/) and
[exposing a service safely](/learn/networking/exposing-a-service-safely/)); the
design point here: the *only* thing facing the internet is a mature,
purpose-hardened VPN endpoint.

**An SSH tunnel** — the zero-new-parts way. SSH can carry port traffic
alongside your shell:

```bash
$ ssh -L 8080:localhost:8080 matt@home-gateway
# then browse http://localhost:8080 — the console, through the tunnel
```

Your browser talks to your own machine; SSH ferries it, encrypted, to the
appliance's port. Perfect for occasional use since you run it on demand — with
the caveat that it presumes an SSH path *to* your network, which itself should
be a VPN or a deliberately hardened, key-only bastion — not a naive
port-forward of SSH to the Pi, which merely swaps which service faces the
scanners. ([SSH tunnels &amp; transfers](/learn/networking/ssh-tunnels-and-transfers/)
covers the plumbing.)

Choose by usage: listening daily from your phone → VPN; checking occasionally
from a laptop → tunnel. Both keep the promise: the appliance faces only the LAN.

## What about sharing beyond your household?

Sharing *audio* with the world is a different problem with a purpose-built
answer: **feed streaming**, where your appliance *pushes* audio out to a public
platform (Broadcastify and kin) and strangers listen there — nothing inbound,
nothing exposed. That road is the [Scanning module's](/learn/scanning/audio-feeds-and-streaming/);
the boundary to keep crisp is *console for the household, feeds for the
public*, and legal/etiquette considerations
([legal &amp; ethical scanning](/learn/scanning/scanning-legal-and-ethical/))
come with it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — tunnel in through a VPN or SSH; never let the console itself face the internet." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to hear your scanner from a hotel. What's the safe design?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Forward port 8080 on the router — it's obscure enough that nobody will find it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">VPN into your home network (or open an SSH tunnel) and reach the LAN-only console through it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Put the console on the internet but change the port number to something unusual</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The console serves the **whole LAN** via the pinned name/address — bookmark
  `http://scanner.local:8080` everywhere; it doubles as a health probe.
- **Never port-forward the console (or SSH) to the internet**: scanners find it
  in hours, and home-service software isn't built for that fight.
- Remote access = **tunnel in**: a **VPN** (daily use, phone-friendly) or an
  **SSH tunnel** (occasional, zero new parts) — one hardened front door,
  console LAN-only behind it.
- Public sharing is **outbound feed streaming**, not inbound exposure — a
  different tool for a different audience.
- The finished posture: appliance invisible to the internet, effortless for
  the household, reachable by you from anywhere — through doors built for it.

Next up: [A complete appliance build](/learn/embedded/a-complete-appliance-build/).
