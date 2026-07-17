---
slug: running-a-server
title: Running a server — binding & reachability
description: A server that's running isn't automatically a server you can reach. Learn what it means to bind to an address and port, why binding to localhost blocks the network while 0.0.0.0 opens it, the checklist that separates "running" from "reachable," privileged ports below 1024, and how to test a service from the inside out.
keywords: bind, localhost, 0.0.0.0, listen, reachability, port, server, interface, bind address, 127.0.0.1, privileged ports, works on my machine
level: intermediate
status: full
prereq:
  - ports-and-sockets
faq:
  - q: "What does it mean for a server to bind to an address?"
    a: "**Binding** is when a server claims a specific interface address and port from the operating system and starts **listening** there. The bind address decides *which* network interfaces can reach the service: bind to **127.0.0.1** and only the same machine can connect; bind to **0.0.0.0** and every interface — including the network — can. A running process that hasn't bound where you expect is the usual reason a service is up but unreachable."
  - q: "What's the difference between localhost and 0.0.0.0?"
    a: "**localhost** (127.0.0.1) is the loopback address — traffic to it never leaves the machine, so a server bound there is reachable *only* from that same computer. **0.0.0.0** is a wildcard meaning \"all interfaces,\" so a server bound to it accepts connections from the network too. Binding to localhost is the number-one reason a service \"works on my machine but not from my phone.\""
  - q: "Why is my server running but not reachable?"
    a: "Running and reachable are different things. For a remote client to connect, **all** of these must be true: the process is up, it's bound to the right **address:port** (0.0.0.0, not 127.0.0.1, for network access), a **firewall** permits that port, and you're using the correct IP or hostname to reach it. A break in any one of them looks the same from outside — nothing connects."
  - q: "Why do I need root to use port 80?"
    a: "Ports below **1024** are **privileged** — the operating system only lets a process bind them with root or a specially granted capability. This is a safety rule so ordinary programs can't impersonate standard services like HTTP on 80 or HTTPS on 443. Ports at or above 1024 need no special privilege, which is why development servers often default to something like 8080."
---

# Running a server — binding & reachability

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A server has to **bind** to an interface address and a port before it can accept
connections — and the address you bind to decides who can reach it.
**localhost (127.0.0.1)** means *this machine only*; **0.0.0.0** means *every
interface, network included*. The big lesson: **running ≠ reachable**. A process
can be up and healthy yet unreachable, and the gap is almost always the bind
address or a firewall. Start from [ports & sockets](/learn/networking/ports-and-sockets/)
and this all clicks into place.
</div>

You started the server. The logs say it's listening. And yet the connection from
your phone just hangs. Nothing is broken in the way you'd expect — the program is
running fine. The problem is that **running a server and being able to reach it are
two separate things**, and the space between them is where most "why won't it
connect?" afternoons disappear. This lesson maps that space.

## Binding to an address and port

Before a server can accept anything, it **binds**: it asks the operating system to
reserve one specific interface address and one [port](/learn/networking/ports-and-sockets/),
then **listens** there for incoming connections. That pair — an address and a port —
is the exact endpoint clients connect to.

The port half is familiar. The address half is the part people skip over, and it's
the one that decides **reachability**. A machine has several interface addresses at
once: the loopback (127.0.0.1), its LAN address (say 192.168.1.50), maybe others.
When a server binds, it picks *which* of those it will answer on — and that choice
quietly controls who is allowed to connect.

You can see exactly what a server bound to with `ss`:

```
$ ss -ltn
State    Recv-Q   Send-Q   Local Address:Port
LISTEN   0        128          127.0.0.1:8080
LISTEN   0        128            0.0.0.0:443
```

The `Local Address` column is the bind address. Here one service is bound to
`127.0.0.1:8080` (local only) and another to `0.0.0.0:443` (all interfaces). The
[inspecting your network](/learn/networking/inspecting-your-network/) lesson covers
`ss` and its friends in full.

## localhost vs 0.0.0.0 — the classic gotcha

This is the single most common tripwire in all of self-hosting, so it's worth being
precise:

- **Bind to `127.0.0.1` (localhost)** — the loopback interface. Traffic to it never
  leaves the machine. The service is reachable **only from the same computer**.
  Perfect for a database you don't want exposed, or while developing on your laptop.
- **Bind to `0.0.0.0`** — a wildcard meaning "**all** interfaces." The service
  answers on the LAN address, so **other machines on the network can reach it**.

Same running program, same port — but bind to localhost and your phone gets
nothing, bind to 0.0.0.0 and it connects. This is *the* reason behind the classic
lament: **"it works on my machine but not from my phone."** The server was up the
whole time; it just wasn't listening on an address the phone could route to.

```
# Reachable only from this machine:
myserver --host 127.0.0.1 --port 8080

# Reachable from the network:
myserver --host 0.0.0.0 --port 8080
```

## Running is not reachable

Treat "reachable" as a checklist, not a single switch. For a remote client to
connect, **all** of these have to be true at once:

1. **The process is up** — the server is actually running and hasn't crashed.
2. **It's bound to the right address:port** — `0.0.0.0`, not `127.0.0.1`, if you
   want network access. Confirm with `ss -ltn`.
3. **A firewall allows the port** — the host (or router) must permit inbound traffic
   to it. See [firewalls](/learn/networking/firewalls/).
4. **You're using the right IP or name to reach it** — the client has to aim at the
   server's real network address, not `localhost` from the wrong machine.

The cruel part: a break in **any one** of these looks identical from the outside —
the connection just hangs or is refused. That's why guessing rarely works and
testing step by step (below) does. When a connection won't come up, walk this list
in order; [testing connectivity](/learn/networking/testing-connectivity/) gives you
the tools for each rung.

## Privileged ports

There's one more thing the operating system enforces when you bind: **ports below
1024 are privileged**. A process can only claim them with **root** or a specially
granted capability. Try to bind port 80 as an ordinary user and you'll get a
permission error before the server ever starts.

The reason is trust. The well-known low ports — 80 for HTTP, 443 for HTTPS, 22 for
SSH — are ones clients connect to expecting a *standard* service, so the OS won't let
just any program squat on them. Ports **at or above 1024** carry no such
restriction, which is why development servers so often default to 8080, 3000, or
5000 — no privilege needed. If you're on Linux, this ties straight into the file and
process permissions the [Linux path](/learn/linux-cli/) covers: binding a low port is
just another action that needs elevated rights.

## Test from inside out

When something won't connect, don't test the whole chain at once — isolate where it
breaks by moving outward one hop at a time:

1. **On the box itself.** `curl localhost:PORT`. If this fails, the server isn't
   really up or isn't bound where you think — nothing else matters yet.

   ```
   $ curl http://localhost:8080
   ```

2. **From another machine on the LAN.** Use the server's LAN address. If step 1
   worked but this doesn't, you're bound to `127.0.0.1` or a firewall is blocking
   the port.

   ```
   $ curl http://192.168.1.50:8080
   ```

3. **From outside the network.** If the LAN works but the outside world can't reach
   it, the issue is at the router/NAT or an external firewall — not the server.

Each rung that passes eliminates a whole class of causes below it, so you find the
break instead of guessing at it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — localhost only answers on the same machine; bind to 0.0.0.0 to reach it from the network." markdown="0">
  <p class="knowledge-check__q">Quick check: Your web app works on the same machine but not from your phone on the same Wi-Fi. Most likely cause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">It's bound to localhost/127.0.0.1 instead of 0.0.0.0</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The server crashed and isn't running</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The port number is above 1024</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A server **binds** to an interface address and a port, then **listens** — and the
  bind address decides *who* can reach it.
- **127.0.0.1 / localhost** = this machine only; **0.0.0.0** = all interfaces,
  reachable from the network.
- **Running ≠ reachable.** A service can be up yet unreachable; the gap is usually
  the bind address or a firewall.
- Reachability is a **checklist**: process up, bound to the right address:port,
  firewall open, and the client aiming at the right IP or name.
- Ports **below 1024** are **privileged** and need root; higher ports don't.
- Debug by testing **inside out** — `localhost` on the box, then the LAN, then
  outside — to isolate where it breaks.

Next up: [exposing a service safely](/learn/networking/exposing-a-service-safely/)
