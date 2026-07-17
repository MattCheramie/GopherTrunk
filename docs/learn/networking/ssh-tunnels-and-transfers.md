---
slug: ssh-tunnels-and-transfers
title: SSH tunnels & transfers
description: Use SSH for more than a shell — forward a port with ssh -L, expose a local service with ssh -R, run a quick SOCKS proxy with ssh -D, and copy files with scp and rsync. How secure tunnels reach a remote service without opening it to the internet.
keywords: ssh tunnel, port forwarding, ssh -L, ssh -R, ssh -D, SOCKS proxy, scp, rsync, secure tunnel, local forwarding, remote forwarding, ssh port forward
level: advanced
status: full
prereq:
  - ports-and-sockets
faq:
  - q: "What does ssh -L do?"
    a: "ssh -L sets up local port forwarding: it opens a port on your own machine and tunnels anything sent to it, over the encrypted SSH connection, to a destination reachable from the remote host. It's how you reach a database or admin panel that only listens on the remote's localhost — the service appears on your laptop as if it were local."
  - q: "What is the difference between ssh -L and ssh -R?"
    a: "Both forward a port over SSH, but in opposite directions. -L (local) brings a remote service to a port on your machine. -R (remote) does the reverse: it exposes one of your local services on a port of the remote machine, which is handy for reaching something behind NAT without configuring port forwarding on your router."
  - q: "How do I make a quick SOCKS proxy with SSH?"
    a: "Run ssh -D 1080 user@host. This opens a SOCKS proxy on your machine at port 1080 and routes any traffic you point at it — such as your browser — out through the SSH server. It's a fast way to browse as if you were on the remote network, with no extra software to install."
  - q: "Should I use scp or rsync to copy files over SSH?"
    a: "Use scp for a quick one-off copy of a file or two — the syntax mirrors cp. Use rsync to sync whole directories: it transfers only the parts that changed, so updating a large folder is fast, and it can resume and mirror. Both ride the same SSH connection, so both are encrypted."
---

# SSH tunnels & transfers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
SSH can carry more than a shell — it can tunnel a **[port](/learn/networking/ports-and-sockets/)**
securely, so you reach a remote service without opening it to the world.
**`ssh -L`** brings a remote service to your machine, **`ssh -R`** exposes a
local one on the remote side, and **`ssh -D`** spins up a quick **SOCKS proxy**.
Everything rides one encrypted **secure tunnel**, so there's no extra inbound
port to expose. And the same connection moves files: **scp** for one-offs,
**rsync** for efficient directory syncs.
</div>

You already know SSH as a way to get a shell on another machine — see the Linux
path's [SSH & working remotely](/learn/linux-cli/ssh-and-remote/) lesson. But the
encrypted connection SSH builds can carry *any* TCP traffic, not just your
keystrokes. That turns SSH into a general-purpose tunnel: a way to reach a service
on the far end (or let the far end reach one on yours) without exposing that
service to the open internet.

## Local forwarding (-L)

**Local forwarding** brings a remote service to your machine. You open a port on
your own computer, and SSH tunnels anything sent to it across to a destination the
remote host can reach:

```bash
$ ssh -L 8080:localhost:80 alice@192.168.1.50
```

Read the `-L` argument as three parts: `8080` is the port that opens on *your*
localhost, and `localhost:80` is the destination *as seen from the remote host*.
So this makes the remote's port 80 appear on your `localhost:8080` — point your
browser at `http://localhost:8080` and you're talking to the remote's web server.

This is perfect for reaching a database or admin panel that only listens on the
remote's **localhost** — a service deliberately bound so it's *not* reachable from
the network. You don't open it up; you tunnel into it. The service still thinks
it's only talking to local connections, because from its point of view it is.

## Remote forwarding (-R)

**Remote forwarding** is the mirror image: it exposes one of *your* local services
on the remote side. You open a port on the remote machine, and connections to it
tunnel back to a service on your end:

```bash
$ ssh -R 9000:localhost:3000 alice@192.168.1.50
```

Now anyone who can reach `localhost:9000` on the remote host is really talking to
`localhost:3000` on your machine. This is useful when your machine sits **behind
NAT** — a home router that gives you no public address — and you want a service on
it reachable from elsewhere, without configuring
[port forwarding](/learn/networking/port-forwarding-and-dynamic-dns/) on the router.
The outbound SSH connection does the work that an inbound port rule would
otherwise have to.

## Dynamic (-D)

**Dynamic forwarding** doesn't tunnel one fixed destination — it stands up a quick
**SOCKS proxy** and routes whatever you point at it out through the SSH server:

```bash
$ ssh -D 1080 alice@192.168.1.50
```

Set your browser's SOCKS proxy to `localhost:1080` and every page it loads now
exits from the remote machine's network. It's a fast, install-nothing way to
browse as if you were sitting on the far end — handy for reaching something that's
only visible from inside the remote network.

## Why tunnels beat opening ports

The common thread is that everything rides the **encrypted SSH connection** you
already trust. To reach a remote service the naive way, you'd bind it to the
network and open an inbound [port](/learn/networking/ports-and-sockets/) — another
door to the internet, another thing to secure, another target for scanners. A
tunnel avoids that entirely: the service stays bound to localhost, and you reach
it through SSH's single, authenticated, encrypted channel. No extra inbound port,
no new attack surface.

For occasional access this is lighter and safer than
[exposing a service](/learn/networking/exposing-a-service-safely/) directly. When
you need *many* services or always-on access between whole networks, a
[VPN](/learn/networking/vpns/) is the heavier, more permanent tool for the job — an
SSH tunnel is the quick, per-service answer.

## Copying files

The same SSH connection also moves files, so you don't need a separate protocol.
Use **scp** ("secure copy") for a one-off transfer — the syntax mirrors `cp`, with
`user@host:` in front of the remote path:

```bash
$ scp capture.cfile alice@192.168.1.50:/home/alice/    # local  -> remote
$ scp alice@192.168.1.50:/var/log/app.log ./           # remote -> local
```

Use **rsync** to sync whole directories efficiently. It compares the two sides and
transfers only the parts that changed, so re-running it to update a large folder is
fast:

```bash
$ rsync -av ./captures/ alice@192.168.1.50:/home/alice/captures/
```

Reach for scp when copying a file or two, and rsync when keeping directories in
step over a slow or repeated transfer.

<div class="knowledge-check" data-quiz data-correct-msg="Right — ssh -L opens a local port and tunnels it to the remote's localhost service." markdown="0">
  <p class="knowledge-check__q">Quick check: a database on a remote server only listens on that server's localhost. How do you reach it from your laptop securely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Rebind the database to listen on the public network</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">SSH local port forwarding (ssh -L)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Open the database's port on the server's firewall</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- SSH tunnels carry any TCP traffic over its encrypted connection — not just a
  shell.
- **`ssh -L`** (local) brings a remote service to a port on *your* machine; great
  for a database or panel bound to the remote's localhost.
- **`ssh -R`** (remote) exposes one of *your* services on the remote side — useful
  behind NAT without router port forwarding.
- **`ssh -D`** opens a quick **SOCKS proxy** that routes your browser through the
  SSH server.
- Tunnels avoid opening an extra inbound port; a **VPN** is the heavier alternative
  for always-on, network-wide access.
- Move files on the same connection: **scp** for one-offs, **rsync** for efficient
  directory syncs.

Next up: [packet capture basics](/learn/networking/packet-capture-basics/)
