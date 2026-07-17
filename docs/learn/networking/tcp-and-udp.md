---
slug: tcp-and-udp
title: TCP vs. UDP
description: The transport layer offers two ways to deliver data over a network — TCP, which guarantees every byte arrives in order, and UDP, which just fires packets and forgets. Learn the three-way handshake, why TCP is reliable and ordered, why UDP is fast and connectionless, and how to choose between them.
keywords: TCP, UDP, transport layer, three-way handshake, reliable, connectionless, packet loss, streaming, datagram, retransmission, ports, TCP vs UDP
level: intermediate
status: full
prereq:
  - protocols-and-layers
faq:
  - q: "What is the difference between TCP and UDP?"
    a: "**TCP** sets up a connection and guarantees delivery: it numbers bytes, acknowledges them, retransmits anything lost, and reorders what arrives out of sequence, so the app sees a clean, ordered stream. **UDP** does none of that — it just sends independent datagrams with no handshake and no guarantees, which makes it faster and lighter but leaves lost or out-of-order packets for the app to deal with."
  - q: "What is the TCP three-way handshake?"
    a: "It's the exchange that opens a TCP connection: the client sends a **SYN**, the server replies with a **SYN-ACK**, and the client answers with an **ACK**. After those three messages both sides agree the connection is open and can start sending data. UDP has no equivalent — it sends immediately, which is part of why it's faster to get going."
  - q: "Is TCP or UDP better?"
    a: "Neither — they trade off. **TCP** is better when every byte must arrive correctly and in order: web pages, file transfers, APIs, SSH. **UDP** is better when speed matters more than completeness and a late packet is worse than a lost one: live voice and video, online gaming, DNS lookups. The right choice depends on whether reliability or latency matters most for the job."
  - q: "Why does UDP not guarantee delivery?"
    a: "UDP is **connectionless** by design — it sends each datagram on its own with no handshake, no acknowledgements, and no retransmission. That's deliberate: skipping those steps removes latency and overhead, which is exactly what real-time media and quick lookups want. If an application needs reliability on top of UDP, it has to add that itself."
---

# TCP vs. UDP

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The [transport layer](/learn/networking/protocols-and-layers/) offers two ways to
deliver data, and they trade off against each other. **TCP (reliable, ordered)**
opens a connection with a **handshake**, then guarantees every byte arrives
complete and in order — at the cost of setup and overhead. **UDP (fast,
connectionless)** skips all of that and just fires datagrams off, lower-latency
and lighter but with no promise they arrive. One guarantees the whole stream; the
other fires and forgets.
</div>

You already know networking is split into
[layers](/learn/networking/protocols-and-layers/), and that the **transport
layer** manages the conversation between two programs. What that layer actually
gives you is a choice between two protocols with very different personalities.
Almost every network application picks one of them, and knowing why is the point
of this lesson.

## The transport layer's job

The [internet layer](/learn/networking/protocols-and-layers/) delivers
[packets](/learn/networking/packets-and-switching/) on a **best-effort** basis:
it does its best to get each one to the right machine, but packets can be lost,
duplicated, or arrive out of order, and nothing lower down promises otherwise.
That's rarely what a program wants to work with directly.

The transport layer's job is to turn that raw best-effort delivery into something
**usable**, and it offers two options with different trade-offs:

- **TCP** does the work to make delivery reliable and ordered.
- **UDP** stays out of the way and hands you the raw best-effort delivery, fast.

Both identify which program a packet is for using
[ports](/learn/networking/ports-and-sockets/) — a port number rides along on
every TCP or UDP packet so the machine knows which service to hand it to.

## TCP — reliable and ordered

**TCP (Transmission Control Protocol)** builds a **connection** before any data
flows. It opens with a **three-way handshake**:

1. The client sends a **SYN** ("I want to talk").
2. The server replies with a **SYN-ACK** ("go ahead, and I want to talk too").
3. The client answers with an **ACK** ("agreed").

After those three messages both ends agree the connection is open. From there,
TCP does the heavy lifting: it **numbers every byte**, the receiver
**acknowledges** what it got, and anything that goes missing is
**retransmitted**. Packets that arrive out of order are **reordered** before the
app sees them. The result is that the application reads a clean, continuous
**stream** of bytes, exactly as they were sent, with the loss and reordering
hidden underneath.

That reliability isn't free. The handshake adds a round-trip of **setup** before
the first byte, and the acknowledgements and bookkeeping add **overhead** on
every message. For most uses that cost is well worth paying — which is why
**HTTP**, **SSH**, and **email** all run over TCP.

## UDP — fast and connectionless

**UDP (User Datagram Protocol)** takes the opposite approach: there's **no
handshake** and **no guarantees**. A program just sends independent
**datagrams** and they go out immediately. UDP doesn't number them, doesn't wait
for acknowledgements, and doesn't retransmit — it simply hands each datagram to
the internet layer and moves on.

The payoff is **lower latency** (no round-trip to set up, no waiting for lost
packets) and **lower overhead** (almost no per-packet bookkeeping). The cost is
that any **loss, duplication, or reordering is the application's problem** to
handle — or to ignore. For a lot of jobs, ignoring it is exactly right: a single
dropped frame of audio is better than pausing the whole stream to recover it.
That's why **DNS**, **live voice and video**, and **online gaming** favor UDP.

## Choosing

The decision comes down to what matters more for the job: **completeness** or
**speed**.

- When **reliability** matters — files, web pages, APIs — reach for **TCP**. A
  corrupted download or a web page missing half its bytes is useless, so it's
  worth waiting for retransmissions.
- When **speed** matters and a **late packet is worse than a lost one** —
  real-time media, live calls, games — reach for **UDP**. By the time a dropped
  audio packet could be resent, the moment it belonged to has already passed, so
  there's no point recovering it.

| | TCP | UDP |
| --- | --- | --- |
| Connection | Yes — three-way handshake | None — connectionless |
| Delivery | Guaranteed, retransmits losses | Best-effort, no guarantees |
| Ordering | Reordered into a clean stream | May arrive out of order |
| Overhead / latency | Higher | Lower |
| Good for | Web, files, SSH, email | Live media, gaming, DNS |

## Why it's relevant here

This trade-off shows up directly in a scanner setup. **Streaming audio** — like
the live audio a scanner produces — often favors **UDP-style delivery**: a
listener would rather drop the occasional packet and stay in sync with the live
call than have playback stall while a late packet is resent. Meanwhile a **web
control interface**, where you click buttons and expect every response to arrive
intact, runs over **TCP**.

You'll see both sides again: the live-update techniques in
[websockets & real-time](/learn/networking/websockets-and-realtime/) build on the
reliable-stream side, and putting these pieces together for a real deployment is
the subject of
[GopherTrunk on the network](/learn/networking/gophertrunk-on-the-network/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — TCP numbers, acknowledges, retransmits, and reorders so the app sees a clean, ordered stream." markdown="0">
  <p class="knowledge-check__q">Quick check: which protocol guarantees data arrives complete and in order?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">UDP</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">TCP</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Neither — the transport layer never guarantees delivery</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **transport layer** turns best-effort packets into something usable, and
  offers two protocols with different trade-offs. Both use **ports** to reach the
  right program.
- **TCP** opens a connection with a **three-way handshake** (SYN, SYN-ACK, ACK),
  then numbers, acknowledges, retransmits, and reorders — so the app sees a clean,
  ordered **stream**. Cost: setup and overhead.
- **UDP** is **connectionless**: no handshake, no guarantees, just datagrams.
  Lower latency and overhead, but losses are the app's problem.
- Choose **TCP** when every byte must arrive (web, files, APIs); choose **UDP**
  when a late packet is worse than a lost one (real-time media, gaming, DNS).
- In a scanner setup, streaming audio favors **UDP-style** delivery while the web
  control interface runs over **TCP**.

Next up: [HTTP — the web's protocol](/learn/networking/http/)
