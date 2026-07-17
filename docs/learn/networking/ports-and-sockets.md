---
slug: ports-and-sockets
title: Ports & sockets
description: One machine runs dozens of network services at once — a web server, SSH, DNS — all sharing a single IP address. Ports are how the machine keeps them apart, and a socket ties an address and port to a specific program. Learn what a port is, the well-known port numbers, what a socket really is, and what "listening" means.
keywords: port, socket, well-known ports, port 80, port 443, port 22, listening, ephemeral port, port number, TCP port, address already in use, networking ports
level: intermediate
status: full
prereq:
  - ip-addresses
faq:
  - q: "What is a port in networking?"
    a: "A **port** is a number, from 0 to 65535, that says which service on a machine a packet is meant for. One machine has a single IP address but runs many services — a web server, SSH, DNS — so the port number is the extra label that routes each packet to the right program. Think of the IP address as the building and the port as the numbered door."
  - q: "What is a socket?"
    a: "A **socket** ties an IP address and a port together with a protocol (TCP or UDP), naming one exact endpoint for a program to send or receive data. A single network **connection** is really a pair of sockets — one on the client and one on the server — and the two together identify the conversation uniquely."
  - q: "What are well-known ports?"
    a: "**Well-known ports** are the standard, reserved numbers below 1024 that common services listen on: **80** for HTTP, **443** for HTTPS, **22** for SSH, **53** for DNS, **25** for SMTP. Because everyone agrees on them, a browser knows to reach a website on port 443 without being told. Ports at the high end of the range are handed out temporarily to clients."
  - q: "What does 'address already in use' mean?"
    a: "It means a program tried to **listen** on a port that another program is already using. Only one program can listen on a given IP-and-port at a time, so the second one is refused. Either the port is genuinely taken by another service, or a previous run of your own program is still holding it. Find what's listening and stop it, or pick a different port."
---
# Ports & sockets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Your machine has one [IP address](/learn/networking/ip-addresses/) but runs many
services at once — a web server, SSH, a database. A **port** is the numbered door
that keeps them apart: every packet carries a port number saying **which service**
it's for. A **socket** ties an address and a port to one specific program, and a
**connection** is really just a pair of sockets talking. A server **listens** on a
port, waiting for connections — and only one program can hold a given port at a
time.
</div>

An IP address gets a packet to the right machine. But a single machine is running
dozens of network programs simultaneously, and they can't all read every packet.
Something has to say *which* program a packet belongs to. That something is the
**port number** — the second half of every network address, and the idea this
lesson is about.

## Why ports exist

An [IP address](/learn/networking/ip-addresses/) identifies a **machine**, not a
program. Your laptop might have a web server, an SSH server, and a DNS resolver all
running on the same address at the same moment. When a packet arrives, the machine
needs one more piece of information: **which of these is it for?**

That piece is the **port number**. Every packet carries a destination port
alongside the destination IP. The IP gets it to the machine; the port gets it to
the right program. The usual analogy: the **IP address is the building**, and the
**port is the numbered door or apartment** inside it. Same address, many doors, and
the number on the door tells the mail which one to knock on.

## Well-known ports

Port numbers run from **0 to 65535**. For this to work, clients and servers have to
**agree in advance** which number a service lives on — otherwise your browser
wouldn't know where to find a website. So the low numbers, **below 1024**, are
reserved as **well-known ports**: fixed, standard assignments that everyone honors.

| Port | Service | What it's for                    |
|-----:|---------|----------------------------------|
| 22   | SSH     | Secure remote login              |
| 25   | SMTP    | Sending email between servers    |
| 53   | DNS     | Turning names into IP addresses  |
| 80   | HTTP    | Unencrypted web traffic          |
| 443  | HTTPS   | Encrypted web traffic            |

Because these are standardized, a browser asked for `https://example.com` just
*knows* to connect on **port 443** without being told. The high end of the range is
different: when your machine opens a connection *out* to a server, it grabs a
temporary, unused high number for itself — an **ephemeral port**. Servers wait on
fixed well-known ports; clients borrow ephemeral ones for the duration of a call
and release them afterward.

## Sockets and connections

Put an IP address, a port, and a protocol (TCP or UDP) together and you have a
**socket** — one exact endpoint that a program can send from or receive on. A
socket isn't a connection by itself; it's the named point at one end.

A **connection** is a **pair of sockets**: the client's socket at one end and the
server's socket at the other. So a live conversation is fully identified by four
things — the client IP and port, and the server IP and port. This is why one server
on **port 443** can hold thousands of connections at once: each remote client has a
different address-and-port, so every socket pair is unique even though they all
share the server's end. When people say "a connection," *this pairing* is what
they really mean.

## Listening

Before a server can accept anything, it has to **listen** on a port — tell the
operating system "hand me any connection that arrives for port 80." From then on,
incoming connections to that port are delivered to that program.

The key rule: **only one program can listen on a given port at a time**. A port is
a single door, and two programs can't both claim to be behind it. If a second
program tries to listen on a port that's already taken, the operating system
refuses it — this is the familiar **"address already in use"** error. Usually it
means either another service genuinely owns that port, or an earlier run of your own
program is still holding it and hasn't let go. We'll come back to binding and
listening in detail in
[running a server](/learn/networking/running-a-server/).

## Seeing what's listening

You don't have to guess which ports are in use. Tools like `ss` (or the older
`netstat`) list every port your machine is currently **listening** on and which
program owns each one — the fast way to find the process squatting on the port you
wanted. The [inspecting your network](/learn/networking/inspecting-your-network/)
lesson covers these commands properly.

## Ties to firewalls

Ports are also the handle that access control grabs. Deciding what a machine will
let in largely comes down to **which ports are open** and which are shut: a
firewall might allow connections to port 443 while blocking everything else, so the
web service is reachable but nothing else is exposed. That's the subject of the
[firewalls](/learn/networking/firewalls/) lesson — but the reason it works at all is
the port number you've just met.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the port number is the extra label that routes each packet to the correct service on a machine." markdown="0">
  <p class="knowledge-check__q">Quick check: What lets a single machine run many network services at once?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Each service gets its own IP address</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ports — each service uses a different port number</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The firewall splits traffic between programs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An IP address finds a **machine**; a **port** number finds the right **service**
  on it — the building and the numbered door.
- Ports run **0 to 65535**. **Well-known ports** below 1024 are standardized (80
  HTTP, 443 HTTPS, 22 SSH, 53 DNS, 25 SMTP); clients borrow high **ephemeral** ports.
- A **socket** is an IP + port + protocol — one endpoint. A **connection** is a
  **pair of sockets**, which is what "a connection" really is.
- A server **listens** on a port for incoming connections, and only **one** program
  can listen on a given port — otherwise "address already in use."
- `ss`/`netstat` show what's listening, and controlling access is largely about
  which **ports are open**.

Next up: [NAT & private networks](/learn/networking/nat-and-private-networks/)
