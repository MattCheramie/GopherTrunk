---
slug: glossary
title: Glossary of networking terms
description: Plain-language definitions of networking terms — packet, protocol, IP, subnet, DNS, port, NAT, TCP, UDP, HTTP, TLS, firewall, VPN, proxy, and more — each cross-linked to the lesson that explains it.
keywords: networking glossary, TCP IP terms, DNS, port, NAT, HTTP, TLS, firewall, VPN, proxy, packet, subnet, socket
level: beginner
status: full
lesson_standalone: true
---

# Glossary of networking terms

Every term used across the [Networking & the Internet](/learn/networking/) path,
defined in plain language and linked to the lesson where it's explained in full.
Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word.
Terms are grouped by theme, roughly in the order the path introduces them.

## How networks work

**Network** — Devices (nodes) connected by links so they can share data. See
[What is a network?](/learn/networking/what-is-a-network/)

**LAN / WAN** — A Local Area Network (home or office) versus a Wide Area Network
spanning distance; the internet interconnects countless networks. See
[What is a network?](/learn/networking/what-is-a-network/)

**Packet** — A small chunk of data with a header (addresses, sequence) wrapped
around a payload; data is split into packets to travel the network. See
[Packets & switching](/learn/networking/packets-and-switching/)

**Router / switch** — A switch moves packets within a local network; a router moves
them between networks, forwarding each toward its destination. See
[Packets & switching](/learn/networking/packets-and-switching/)

**Protocol** — An agreed set of rules two machines follow to communicate; both ends
must speak the same one. See [Protocols & layers](/learn/networking/protocols-and-layers/)

**Layers (TCP/IP model)** — Networking split into stacked layers — link, internet,
transport, application — each with its own job and protocols. See
[Protocols & layers](/learn/networking/protocols-and-layers/)

**Client / server** — The client initiates a request; the server listens and
responds. See [Clients, servers & peers](/learn/networking/clients-and-servers/)

**Peer-to-peer** — A model with no central server, where peers connect directly and
both request and serve. See [Clients, servers & peers](/learn/networking/clients-and-servers/)

## Addressing

**IP address** — A number identifying a device on a network so packets can be routed
to it. See [IP addresses & subnets](/learn/networking/ip-addresses/)

**IPv4 / IPv6** — The older 32-bit addressing (dotted quad, ~4 billion addresses)
and its vastly larger successor. See [IP addresses & subnets](/learn/networking/ip-addresses/)

**Subnet / CIDR** — A block of addresses; CIDR notation like /24 marks how many bits
are the network part versus the host part. See [IP addresses & subnets](/learn/networking/ip-addresses/)

**Public vs. private IP** — Publicly routable addresses versus private ranges
(10.x, 172.16–31.x, 192.168.x) reused inside home and office networks. See
[NAT & private networks](/learn/networking/nat-and-private-networks/)

**DNS** — The internet's phone book, translating names like example.com into IP
addresses. See [DNS — names to addresses](/learn/networking/dns/)

**DNS record** — An entry in DNS: A (name→IPv4), AAAA (→IPv6), CNAME (alias), MX
(mail), TXT (verification). See [DNS — names to addresses](/learn/networking/dns/)

**Port** — A numbered door on a machine so it can run many network services at once
(80 = HTTP, 443 = HTTPS, 22 = SSH). See [Ports & sockets](/learn/networking/ports-and-sockets/)

**Socket** — An address + port + protocol that ties a connection to a specific
program; a connection is a pair of sockets. See [Ports & sockets](/learn/networking/ports-and-sockets/)

**NAT (network address translation)** — How a router maps many private addresses
onto one public IP, so all your devices share it. See
[NAT & private networks](/learn/networking/nat-and-private-networks/)

**DHCP** — The protocol that automatically leases a joining device its IP address,
gateway, and DNS settings. See [DHCP & joining a local network](/learn/networking/dhcp-and-local-networks/)

**Default gateway** — The router a device sends traffic to when the destination is
outside its own subnet. See [DHCP & joining a local network](/learn/networking/dhcp-and-local-networks/)

## Protocols

**TCP** — A reliable, ordered, connection-based transport protocol (with a
three-way handshake) used by the web, SSH, and email. See
[TCP vs. UDP](/learn/networking/tcp-and-udp/)

**UDP** — A fast, connectionless transport protocol with no delivery guarantees,
used for DNS, voice, and video. See [TCP vs. UDP](/learn/networking/tcp-and-udp/)

**HTTP** — The request/response protocol of the web — methods, headers, and status
codes. See [HTTP — the web's protocol](/learn/networking/http/)

**Status code** — HTTP's result number by class: 2xx success, 3xx redirect, 4xx
client error, 5xx server error. See [HTTP — the web's protocol](/learn/networking/http/)

**TLS / HTTPS** — TLS encrypts and authenticates a connection; HTTPS is HTTP over
TLS — the padlock. See [TLS & HTTPS](/learn/networking/tls-and-https/)

**Certificate / CA** — A certificate binds a domain to a public key and is signed by
a Certificate Authority your device trusts. See [TLS & HTTPS](/learn/networking/tls-and-https/)

**Web API / REST** — Programs talking over HTTP; REST addresses resources by URL and
uses HTTP methods as verbs, usually with JSON. See [Web APIs & REST](/learn/networking/web-apis-and-rest/)

**WebSocket** — A persistent, two-way connection (upgraded from HTTP) that lets the
server push data any time. See [WebSockets & real-time connections](/learn/networking/websockets-and-realtime/)

## Connecting & securing

**Firewall** — A control that allows or blocks traffic by port, address, and
protocol; safest set to deny-by-default. See [Firewalls & controlling access](/learn/networking/firewalls/)

**Port forwarding** — A router rule mapping a public port to a device's private
address and port, so outside traffic can reach it. See
[Port forwarding & dynamic DNS](/learn/networking/port-forwarding-and-dynamic-dns/)

**Dynamic DNS** — A service that keeps a hostname pointed at your changing home IP
address. See [Port forwarding & dynamic DNS](/learn/networking/port-forwarding-and-dynamic-dns/)

**VPN** — An encrypted tunnel to a VPN server, hiding your traffic from the local
network and ISP — but not true anonymity. See [VPNs & tunnels](/learn/networking/vpns/)

**Proxy / reverse proxy** — A middleman that forwards requests; a reverse proxy sits
in front of servers to terminate TLS, route, and hide backends. See
[Proxies, reverse proxies & load balancers](/learn/networking/proxies-and-load-balancers/)

**Load balancer** — Spreads incoming traffic across several backend servers and
skips unhealthy ones. See [Proxies, reverse proxies & load balancers](/learn/networking/proxies-and-load-balancers/)

**Man-in-the-middle** — An attacker positioned to read or alter traffic between two
parties; encryption defends against it. See [Network security basics](/learn/networking/network-security-basics/)

## Command-line tools

**ping** — Tests whether a host is reachable and how long a round trip takes. See
[Testing connectivity](/learn/networking/testing-connectivity/)

**traceroute** — Shows the sequence of hops between you and a destination. See
[Testing connectivity](/learn/networking/testing-connectivity/)

**dig / nslookup** — Queries DNS to see what address (and records) a name resolves
to. See [Testing connectivity](/learn/networking/testing-connectivity/)

**ss / netstat** — Lists network connections and which ports your machine is
listening on. See [Inspecting your network](/learn/networking/inspecting-your-network/)

**curl** — Sends HTTP requests from the shell to test an API or a site and read the
response. See [Making requests with curl](/learn/networking/curl-and-http-tools/)

**SSH tunnel** — Carrying another service's traffic securely inside an SSH
connection (ssh -L / -R / -D). See [SSH tunnels & transfers](/learn/networking/ssh-tunnels-and-transfers/)

**tcpdump / Wireshark** — Tools that capture and inspect the actual packets on a
network for debugging. See [Packet capture basics](/learn/networking/packet-capture-basics/)

## Running a service

**Bind / listen** — A server binds to an address and port and listens there for
incoming connections. See [Running a server — binding & reachability](/learn/networking/running-a-server/)

**localhost vs. 0.0.0.0** — Binding to localhost (127.0.0.1) allows only the same
machine; 0.0.0.0 allows every network interface. See
[Running a server — binding & reachability](/learn/networking/running-a-server/)

**Reverse proxy (TLS termination)** — The standard front door for a self-hosted web
service — it handles HTTPS and forwards to the app. See
[Exposing a service safely](/learn/networking/exposing-a-service-safely/)

**Self-hosting** — Running your own services on hardware you control, often on a home
network. See [Self-hosting on a home network](/learn/networking/self-hosting-at-home/)
