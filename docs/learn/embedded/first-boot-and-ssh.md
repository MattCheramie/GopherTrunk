---
slug: first-boot-and-ssh
title: First boot & SSH
description: Find a freshly booted headless board on your network — by mDNS name, router table, or scan — and log in over SSH. The moment a single-board computer becomes a tiny server.
keywords: find raspberry pi on network, ssh into raspberry pi, mdns, hostname.local, ping, nmap, arp, first login, headless login
level: beginner
status: full
prereq:
  - flashing-an-os-image
---

# First boot & SSH

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A headless board announces itself only on the network. Find it three ways, in
order of niceness: **mDNS** (`ping scanner.local` — the hostname you set while
flashing), your **router's device list**, or a **network scan** (`nmap`). Then
**`ssh you@scanner.local`** puts a real shell on the board — from that moment it is
a tiny server and the monitor question never comes up again. First-visit rituals:
accept the **host key**, confirm the OS sees the world (`uname -a`, `df -h`,
`ip addr`), and note the board's **IP address** for the day mDNS lets you down.
</div>

The card is flashed and the board is powered, its status LED flickering
meaningfully at nobody. This lesson is the payoff moment of headless computing:
finding the machine and logging in — and a small toolbox for the day it doesn't
answer.

## How do you find a board you can't see?

Your board asked the router for an address the moment it joined the network (that's
**DHCP** — the [Networking module](/learn/networking/dhcp-and-local-networks/) has
the full story). Three ways to find where it landed:

**1. mDNS — the friendly way.** Modern boards advertise `hostname.local` via
multicast DNS. If you set the hostname `scanner` while flashing:

```bash
$ ping scanner.local
PING scanner.local (192.168.1.47) 56(84) bytes of data.
64 bytes from 192.168.1.47: icmp_seq=1 ttl=64 time=1.32 ms
```

A reply means the board is up *and* you've learned its IP. mDNS works out of the box
on macOS and most Linux; Windows has supported it in recent versions.

**2. The router's table.** Your router's admin page lists DHCP clients by hostname —
look for `scanner` and read its address.

**3. A network scan.** The heavy tool when the first two fail:

```bash
$ nmap -sn 192.168.1.0/24        # ping-sweep the whole LAN
```

Compare the results with the board unplugged and plugged in; the new address is your
board. (Scan only networks you own —
[Inspecting your network](/learn/networking/inspecting-your-network/) covers these
tools properly.)

## What does SSH actually give you?

**SSH** (Secure Shell) opens an encrypted channel to the board and hands you a shell
— the identical experience to sitting at its keyboard, minus the keyboard:

```bash
$ ssh matt@scanner.local
The authenticity of host 'scanner.local (192.168.1.47)' can't be established.
ED25519 key fingerprint is SHA256:kkq7...
Are you sure you want to continue connecting (yes/no/[fingerprint])? yes
matt@scanner.local's password:
matt@scanner:~ $
```

That first-connection question is the **host key** ritual: the board proves its
identity with a key generated on first boot, your machine remembers it, and any
*future* mismatch warning means "this may not be the machine you think" — worth
respecting. The [Linux CLI module](/learn/linux-cli/ssh-and-remote/) covers SSH in
depth; [Remote administration](/learn/embedded/remote-administration/) will upgrade
you from passwords to keys.

Everything you do on the board from now on — installing GopherTrunk, editing
configs, reading logs — happens through this window. The prompt changing to
`matt@scanner` is the visible sign of *where your commands now run*; misreading it
is how people reboot the wrong machine.

## What should you check on the first visit?

A five-command health tour confirms the flash went well:

```bash
$ uname -a          # kernel, and the architecture (aarch64 = 64-bit ARM)
$ df -h /           # did the filesystem expand to fill the card?
$ free -h           # RAM as expected?
$ ip addr           # interfaces and addresses
$ vcgencmd measure_temp   # Pi-only: a sane idle temperature (~40-55 °C)
```

Two follow-ups worth doing immediately: **note the IP address** somewhere safe (if
mDNS ever fails, you'll want it), and run your first update
(`sudo apt update && sudo apt full-upgrade`) — the image you flashed was built weeks
ago; the [next lesson](/learn/embedded/users-and-updates/) makes this a habit.

## What if it never appears?

The headless-troubleshooting ladder, cheapest first:

- **Wait longer.** First boot does one-time setup; give it three minutes.
- **Check the LEDs.** No power LED → supply/cable. On a Pi, a rhythmic *blink
  pattern* on the activity LED is the firmware signalling a boot error (pattern
  meanings are in the board docs) — usually card or image.
- **Check the link lights** on the Ethernet jack — no lights, no network; reseat the
  cable, try another port.
- **Re-check the flash settings.** Pull the card, open its boot partition on your
  PC, and confirm SSH/Wi-Fi settings landed
  ([Flashing an OS image](/learn/embedded/flashing-an-os-image/)).
- **Re-flash with verification** — the definitive reset of all software doubt.

> Rule of thumb: when a fresh board misbehaves, suspect in this order — power,
> card, network, image. Actual dead boards are rare; bad cables are not.

<div class="knowledge-check" data-quiz data-correct-msg="Right — mDNS advertises the hostname you set, so hostname.local finds the board with no monitor." markdown="0">
  <p class="knowledge-check__q">Quick check: the friendliest first way to find a freshly booted headless board is…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">attaching a monitor briefly to read its IP address off the screen</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">pinging the hostname.local name it advertises over mDNS</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">power-cycling the router until the board gets address .100</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A headless board is found on the **network**: **mDNS** (`hostname.local`), the
  **router's DHCP table**, or an **nmap sweep** — in that order.
- **SSH** gives you the board's real shell over an encrypted channel; the first-visit
  **host key** prompt is the board introducing itself.
- Run the **five-command health tour** (`uname`, `df`, `free`, `ip addr`,
  `measure_temp`) and note the **IP address** for the day names fail.
- A missing board is diagnosed cheapest-first: **power, card, network, image** —
  and the LEDs are the board's only local voice.
- From here on, everything in this module happens **through SSH**.

Next up: [Users, permissions &amp; updates](/learn/embedded/users-and-updates/).
