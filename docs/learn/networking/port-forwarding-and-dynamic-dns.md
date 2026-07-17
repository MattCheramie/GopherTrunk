---
slug: port-forwarding-and-dynamic-dns
title: Port forwarding & dynamic DNS
description: How to reach a service running at home from the outside world — why NAT blocks inbound connections, how port forwarding opens a path to one device, how dynamic DNS follows your changing public IP, the exposure risk you take on, and safer alternatives like VPNs and reverse tunnels.
keywords: port forwarding, dynamic DNS, DDNS, home server, expose service, router, NAT traversal, reverse tunnel, public IP, inbound connection
level: intermediate
status: full
prereq:
  - nat-and-private-networks
faq:
  - q: What is port forwarding?
    a: Port forwarding is a rule on your router that sends traffic arriving on a chosen public port to a specific device and port inside your home network. Without it, NAT has no way to know which internal device an unsolicited inbound connection is meant for, so it drops the traffic. The rule creates that mapping — public port X on the router forwards to a private address and port on one device.
  - q: What is dynamic DNS?
    a: Dynamic DNS (DDNS) keeps a fixed hostname pointed at your home's changing public IP address. Most home ISPs hand out an IP that changes over time, so a client on the DDNS hostname's device updates the DNS record whenever the address changes. You (and anyone you allow) can then reach your service by name instead of chasing a number that keeps moving.
  - q: Is port forwarding safe?
    a: Port forwarding removes NAT's protection for one service and exposes it directly to the entire internet, including automated scanners and bots that probe every address constantly. It is only as safe as the service behind it — that service must be secured with TLS, real authentication, current updates, and ideally a firewall. If you can avoid opening an inbound port at all, a VPN or reverse tunnel is safer.
  - q: Do I need a static IP to run a home server?
    a: No. A static IP is convenient but most home connections don't include one. Dynamic DNS solves the same problem for free or cheaply by tracking your changing address behind a stable hostname, so you rarely need to pay your ISP for a static IP just to reach a home service by name.
---

# Port forwarding & dynamic DNS

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To reach a service running at home from outside, you need two things.
**Port forwarding** maps a port on your router to one device inside your
[private network](/learn/networking/nat-and-private-networks/), so an inbound
connection knows where to land. **Dynamic DNS** gives that connection a stable
name that follows your ISP's changing public IP. Both together let the outside
world find your service — but a forwarded port is an **exposure risk**: it puts
that service in front of the whole internet, so it must be secured or, better,
avoided with a VPN or tunnel.
</div>

Running something at home you want to reach from elsewhere — a game server, a
file share, a camera, GopherTrunk's web interface — sounds like it should just
work. It doesn't, and the reason is the same NAT that quietly protects your
network every day.

## The problem

[NAT](/learn/networking/nat-and-private-networks/) lets many devices share one
public IP address, and as a side effect it hides them. When a packet arrives at
your router unprompted, NAT has no matching outbound connection to tie it to —
so it has no idea which internal device the packet was meant for, and it drops
it. That is exactly what keeps the internet from poking at your laptop.

The catch is that "the internet can't reach in unprompted" is also true when
*you* want it to reach in. To accept an inbound connection, you must explicitly
open a path through the router and tell it where that traffic should go.

## Port forwarding

Port forwarding is that explicit path. You add a rule in the router's
configuration that reads, in effect: *traffic arriving on public port X →
forward to device 192.168.1.50, port Y*. From then on, the router stops dropping
those inbound packets and hands them to the device you named.

A few pieces have to line up:

- **A public port** — the port on the router's internet-facing side that
  outsiders connect to.
- **A target device and port** — the private
  [address](/learn/networking/ip-addresses/) and
  [port](/learn/networking/ports-and-sockets/) inside your network that should
  receive it.

One requirement is easy to miss: the target device should have a **fixed or
reserved** address. Home devices normally get their address from
[DHCP](/learn/networking/dhcp-and-local-networks/), which can hand out a
different one after a reboot. If that happens, your forwarding rule still points
at the old address and silently stops working. Reserve the device's address (a
DHCP reservation, or a static setting) so the rule keeps pointing at the right
machine.

## Dynamic DNS

Port forwarding gets a connection from your router to the right device. But how
does an outsider reach your router in the first place? By its **public IP
address** — and that is the next problem.

Most home ISPs hand out a *dynamic* public IP: it changes every so often, on a
reconnect, a lease renewal, or the ISP's own schedule. Anything you told people
to connect to yesterday may point nowhere today.

**Dynamic DNS (DDNS)** solves this. A DDNS service gives you a stable hostname —
something like `myhouse.example-ddns.net` — and a small client running on your
network watches your public IP. Whenever the address changes, the client updates
the [DNS](/learn/networking/dns/) record so the hostname always resolves to your
current IP. You, and anyone you allow, reach the service by name and never have
to know the number underneath.

## The risk you just took on

Here is the part people skip, and it matters most. NAT was protecting that
service by default. The moment you forward a port, you remove that protection
for it: the service is now exposed to the **entire internet**, not just to you.

That internet is full of automated **scanners and bots** that probe every
address and every common port continuously, looking for anything that answers.
Within minutes of opening a port, something will find it. So a forwarded service
**must** be secured before it is worth doing:

- **Encryption** — [TLS](/learn/networking/tls-and-https/) so the traffic can't
  be read or tampered with in transit.
- **Authentication** — real credentials, not defaults, so answering the door
  isn't the same as letting someone in.
- **Updates** — the exposed software kept patched against known holes.
- **A firewall** — to limit what is reachable and from where.

Read [exposing a service safely](/learn/networking/exposing-a-service-safely/),
[network security basics](/learn/networking/network-security-basics/), and
[firewalls](/learn/networking/firewalls/) before you leave a port open to the
world.

## Safer alternatives

Often the cleanest answer is to **not open an inbound port at all**. Several
approaches give you remote access without exposing a listening service to
everyone:

- **VPN in.** Connect to your home network over a
  [VPN](/learn/networking/vpns/) and reach services as if you were on the local
  network. Only the VPN itself is exposed, and it's built to be.
- **Outbound reverse tunnel.** Have a device at home make an *outbound*
  connection to a server you control and forward traffic back through it — for
  example [SSH remote forwarding](/learn/networking/ssh-tunnels-and-transfers/).
  Nothing inbound is opened on your router at all.
- **A tunneling service.** A third-party service establishes the outbound
  connection for you and gives you a public URL, again with no forwarded port.

The common thread: the connection originates *from inside*, so NAT's default
"drop unsolicited inbound" stays in force and there's no open port for scanners
to find.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the forwarding rule maps a public port to that device's private address and port." markdown="0">
  <p class="knowledge-check__q">Quick check: what lets an outside connection reach a specific device behind your home router?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Dynamic DNS — it points a hostname at your IP</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Port forwarding — a rule mapping a public port to that device's private address and port</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">NAT — it hides internal devices from the internet</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- NAT blocks unsolicited inbound traffic, so reaching a home service from
  outside takes deliberate setup.
- **Port forwarding** maps a public port on the router to one device's private
  address and port.
- Give that target a **fixed/reserved** address so the rule keeps working after
  a reboot.
- **Dynamic DNS** keeps a stable hostname pointed at your changing public IP.
- A forwarded port **exposes that service to the whole internet** — secure it
  with TLS, authentication, updates, and a firewall.
- Prefer a **VPN or reverse tunnel** so you never open an inbound port at all.

Next up: [VPNs & tunnels](/learn/networking/vpns/)
