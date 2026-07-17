---
slug: inspecting-your-network
title: Inspecting your network
description: A plain-language guide to the everyday commands that show your network state — ip addr for interfaces and addresses, ip route for the default gateway, and ss (or netstat) for which programs are listening on which ports.
keywords: ip addr, ip route, ss, netstat, ifconfig, listening ports, network interface, hostname, default gateway, routing table, sockets
level: intermediate
status: full
prereq:
  - ip-addresses
faq:
  - q: How do I see my IP address on Linux?
    a: "Run ip addr to list every network interface and the addresses assigned to it — your address is the inet line under the interface you're using, such as eth0 or wlan0. For just the addresses without the detail, hostname -I prints them on one line. On macOS or older systems the older ifconfig command shows the same information."
  - q: How do I see which programs are listening on which ports?
    a: "Run ss -tulpn (or the older netstat -tulpn). It lists every TCP and UDP socket that is listening, the address and port it is bound to, and the program that owns it. This is the quickest way to confirm a server is actually running and bound where you expect."
  - q: How do I find my default gateway?
    a: "Run ip route and look for the line beginning with default — the address after via is your default gateway, the router the machine sends traffic to when the destination is not on the local network. The routing table below it lists the networks the machine can reach directly."
  - q: What is the difference between ss and netstat?
    a: "They do the same job — showing sockets and listening ports — but ss is the newer, faster tool that ships on modern Linux, while netstat is the older command many people still know by habit. The flags are almost identical, so ss -tulpn and netstat -tulpn read the same way."
---

# Inspecting your network

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three questions cover most of what you need to know about a machine's
networking, and a single command answers each. **`ip addr` / `ip route`**
show your addresses and how packets leave the machine; **`ss` (or the older
`netstat`)** shows **what's listening**. Together they answer *what's my
address, how do I get out, and what's running?* If addresses themselves are
new to you, start with [IP addresses & subnets](/learn/networking/ip-addresses/).
</div>

When something on the network isn't working, guessing is slow. A handful of
commands turn "it's broken somehow" into concrete facts: the machine has this
address, it sends outbound traffic to this gateway, and these programs are
listening on these ports. Learn the three below and you can inspect almost any
Linux box in under a minute.

## What's my address? — `ip addr`

The `ip addr` command lists every **network interface** on the machine and the
[IP addresses](/learn/networking/ip-addresses/) assigned to each one:

```
$ ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> ...
    inet 127.0.0.1/8 scope host lo
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> ...
    inet 192.168.1.42/24 brd 192.168.1.255 scope global eth0
```

Each numbered block is an interface: `lo` is the **loopback** (the machine
talking to itself at `127.0.0.1`), and `eth0` here is the wired connection with
address `192.168.1.42`. Wireless interfaces usually show up as `wlan0`. The
`inet` line is the one you want — that's the device's IPv4 address and its
subnet.

If you just want the addresses without the surrounding detail, `hostname -I`
prints them on a single line:

```
$ hostname -I
192.168.1.42
```

On macOS, or on older Linux systems, the classic `ifconfig` command shows the
same interface-and-address picture with slightly different formatting.

## How do I get out? — `ip route`

Knowing your address tells you who you are; the **routing table** tells you how
your traffic leaves the machine. `ip route` prints it:

```
$ ip route
default via 192.168.1.1 dev eth0
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.42
```

The important line is the one starting with `default`. Its address — here
`192.168.1.1` — is the **default gateway**, the router the machine hands a
packet to whenever the destination isn't on the local network. This is the
router that [hands out addresses](/learn/networking/dhcp-and-local-networks/) on
a typical home network.

The rest of the table lists networks the machine can reach **directly**. When
the machine has a packet to send, it checks the destination against these
routes: if the address is on a network listed here, it goes straight out that
interface; if not, it falls back to the default gateway. No default route
usually means no path to the wider internet.

## What's listening? — `ss`

The third question is which programs are accepting connections. `ss -tulpn`
answers it:

```
$ ss -tulpn
Netid State  Local Address:Port  Process
tcp   LISTEN 0.0.0.0:22          users:(("sshd",pid=712,fd=3))
tcp   LISTEN 127.0.0.1:5432      users:(("postgres",pid=980,fd=5))
udp   UNCONN 0.0.0.0:68          users:(("dhclient",pid=540,fd=6))
```

Those flags are worth memorising: `-t` TCP, `-u` UDP, `-l` listening only,
`-p` show the program, `-n` numeric (don't translate ports to names). The older
`netstat -tulpn` produces the same view on systems where `ss` isn't installed.

This is the essential check when you're asking *is my server actually up and
bound?* A program that isn't in this list isn't accepting connections, no
matter what its logs say. Each entry is a **listening
[socket](/learn/networking/ports-and-sockets/)** — an address-and-port the
kernel is holding open for a program. When you go on to
[run a server](/learn/networking/running-a-server/), this is the command that
confirms it came up.

## Reading the output

Once a socket line is in front of you, four fields tell the story. Take this
row:

```
tcp   LISTEN 0.0.0.0:22   users:(("sshd",pid=712,fd=3))
```

- **Protocol** — `tcp`. TCP for connection-based services, UDP for the rest.
- **Address** — `0.0.0.0`. The interface it's bound to (see below).
- **Port** — `22`. The [port number](/learn/networking/ports-and-sockets/)
  clients connect to — `22` is SSH.
- **Program** — `sshd`, process ID `712`. The process that owns the socket.

Read left to right and you have "the SSH daemon is listening for TCP
connections on port 22, on every interface." That one sentence is usually all
you need to confirm a service is where you expect it.

## Putting it together

Chain the three commands into a quick self-check whenever networking feels
broken:

1. **Do I have an IP?** `ip addr` — if the interface has no `inet` line, nothing
   else will work; the machine hasn't joined the network yet.
2. **Do I have a gateway?** `ip route` — no `default` route means local traffic
   may work but the wider internet won't.
3. **Is my service listening where I expect?** `ss -tulpn` — and check the
   **address**, not just the port.

That last point catches a very common mistake. A service bound to `127.0.0.1`
is reachable **only from the same machine**; one bound to `0.0.0.0` is reachable
from **any** interface, including other machines on the network. If a server
works locally but nothing else can reach it, this is the first thing to check —
a lesson that comes up again when you [run a server](/learn/networking/running-a-server/)
and when you [test connectivity](/learn/networking/testing-connectivity/) to it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — ss (or the older netstat) lists listening sockets and the programs that own them." markdown="0">
  <p class="knowledge-check__q">Quick check: which command shows which ports your machine is listening on and which programs own them?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ip route</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">ss (or netstat)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">hostname -I</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`ip addr`** lists your network interfaces and their IP addresses; **`hostname -I`** is the quick version, and **`ifconfig`** does the same on macOS or older systems.
- **`ip route`** shows the routing table; the **`default`** line is your gateway, the router used for anything not on the local network.
- **`ss -tulpn`** (or **`netstat -tulpn`**) shows which programs are listening on which ports — the check for "is my server actually up and bound?"
- Read a socket line by its four fields: **protocol, address, port, program**.
- A socket bound to **`0.0.0.0`** is reachable from any interface; one on **`127.0.0.1`** is reachable only from the same machine.
- The three commands together answer *what's my address, how do I get out, and what's running?*

Next up: [testing connectivity](/learn/networking/testing-connectivity/)
