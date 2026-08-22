---
slug: networking-your-board
title: Networking your board
description: Reach a headless board reliably — DHCP reservations vs static IPs, mDNS hostnames, Wi-Fi vs Ethernet trade-offs, and making a board's address survive reboots, router restarts, and years of uptime.
keywords: static ip raspberry pi, dhcp reservation, mdns hostname, wifi vs ethernet sbc, nmcli, ip address, headless networking, reliable network
level: intermediate
status: full
prereq:
  - first-boot-and-ssh
---

# Networking your board

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An appliance you can't reach is an appliance that's down — so its address must be
**boring and permanent**. Prefer **wired Ethernet** over Wi-Fi for anything
always-on. Give the board a stable address with a **DHCP reservation** at the
router (the low-maintenance winner) rather than a static IP configured on the
board, and reach it by **hostname** day-to-day. Check your links from both ends:
`ip addr` on the board, `ping` from the laptop. Set it up once, and the address
outlives reboots, router restarts, and your memory.
</div>

Unit 3 closes with the plumbing that everything later leans on: Unit 5 will
administer this board remotely for months, and Unit 6 serves a web console every
device in the house must find. Reliable reachability is a set-once decision — this
lesson makes it deliberately.

## Wi-Fi or Ethernet — is it actually a contest?

For a portable project, Wi-Fi is the point. For a fixed 24/7 appliance, **Ethernet
wins on every axis that matters**:

| | Wired Ethernet | Wi-Fi |
|---|---|---|
| **Reliability** | Link is up or the cable is visibly out | Interference, congestion, driver moods |
| **Configuration** | None — plug it in | Credentials that expire when you change routers |
| **Throughput/latency** | Consistent, full-duplex | Variable, shared airtime |
| **Radio hygiene** | Silent | A 2.4/5 GHz transmitter beside your SDR |
| **Placement freedom** | Needs a cable run | Anywhere |

That last row is Wi-Fi's one real card, and sometimes it wins — the attic corner
with the best antenna spot may have no jack. If you must use Wi-Fi for an
appliance, treat it as engineering: strong signal at the mounting spot (check with
the board *in place*), credentials configured at flash time, and know that the
board's transmitter sits centimetres from your SDR — one more noise source in the
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) story.

## Why do addresses wander, and how do you pin one down?

By default the router's **DHCP** hands the board a lease from a pool — and after an
outage or router swap, the board may come back at a *different* address, breaking
bookmarks and muscle memory. Two fixes:

**DHCP reservation (recommended).** In the router's admin page, pin the board's
hardware (MAC) address to a fixed IP. The board itself stays zero-config — it just
always receives, say, `192.168.1.50`. One place to manage all your fixed devices,
nothing on the board to migrate when you re-flash, and no risk of address
collisions.

**Static IP on the board.** Configure the address in the board's own network
settings (with NetworkManager: `nmcli con mod "Wired connection 1"
ipv4.addresses 192.168.1.50/24 ipv4.gateway 192.168.1.1 ipv4.dns 192.168.1.1
ipv4.method manual`). It works with no router cooperation — the right tool when
you don't control the router — but the settings live on the SD card (re-flash and
they're gone), and a typo'd or colliding address can knock the board off the
network *headlessly*, which is its own adventure.

> Rule of thumb: reservation at the router if you possibly can; static on the
> board only when you can't. Either way, **write the address down** — labels on
> the case are not a joke.

([IP addresses](/learn/networking/ip-addresses/) and
[DHCP &amp; local networks](/learn/networking/dhcp-and-local-networks/) cover the
underlying machinery.)

## What should you actually type day-to-day?

Names, not numbers. With the hostname set at flash time, `ssh matt@scanner.local`
and `http://scanner.local:8080` (Unit 6's console) work from most devices via
**mDNS**. Its limits: some older devices and VLAN-separated networks don't pass
mDNS — which is when the pinned IP earns its keep, or a hosts-file entry
(`192.168.1.50 scanner`) on your main machines, or a DNS entry if your router
offers local DNS. The pattern: **names for humans, one pinned IP underneath.**

## How do you check a link from both ends?

The two-minute diagnosis ritual, worth internalising before Unit 5 has you
debugging from afar. On the board:

```bash
$ ip addr show eth0        # does the interface hold the address you expect?
$ ip route                 # is there a default route (via the router)?
$ ping -c3 192.168.1.1     # can it reach the router?
$ ping -c3 deb.debian.org  # can it reach (and resolve) the internet?
```

From your laptop: `ping scanner.local`, then `ssh`. Each step that fails localises
the fault — interface, router, DNS, or the wider world.
([Testing connectivity](/learn/networking/testing-connectivity/) expands this
ladder.) An appliance habit worth copying: after any network change, reboot the
board and confirm it comes back reachable *on its own* — the reboot test is the
only proof the configuration is real.

## What about reaching it from outside the house?

Short answer for now: **don't** — no port-forwards to the board, nothing exposed
raw to the internet. The safe patterns (VPN into your LAN, or SSH tunnels) are
Unit 6's [Appliance networking &amp; access](/learn/embedded/appliance-networking/),
with the deeper treatment in the Networking module's
[Exposing a service safely](/learn/networking/exposing-a-service-safely/). Inside
the LAN, though, you're done: the board now has a permanent address and a name
every later lesson will rely on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the router always hands that board the same address, with nothing to maintain on the board itself." markdown="0">
  <p class="knowledge-check__q">Quick check: why is a DHCP reservation usually better than a static IP configured on the board?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Reservations make the network connection faster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The address is pinned at the router, so the board needs no local config that could be lost or collide</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Static IPs stop working after 30 days on home routers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An appliance's address must be **boring and permanent** — reachability is a
  set-once decision.
- **Ethernet beats Wi-Fi** for always-on boards: reliability, zero config, and no
  transmitter next to your SDR; Wi-Fi is for when the cable can't go there.
- Pin the address with a **DHCP reservation** at the router; static-on-board is the
  fallback when you don't control the router.
- Use **names day-to-day** (mDNS `scanner.local`, hosts entries) with one pinned IP
  underneath — and write the IP down.
- Diagnose from **both ends** (`ip addr`, `ip route`, `ping` outward; `ping`/`ssh`
  inward), and prove changes with a **reboot test**. Keep it **LAN-only** for now.

Next up: [GPIO basics](/learn/embedded/gpio-basics/).
