---
slug: packets-and-switching
title: Packets & switching
description: Networks don't send data as one long stream — they chop it into small packets that travel independently and are reassembled at the far end. Learn what a packet is, the header-and-payload structure, how switches and routers move packets, and why delivery is best-effort.
keywords: packet, packet switching, header, payload, router, switch, best-effort, routing, MAC address, reassembly, out of order, network
level: beginner
status: full
prereq:
  - what-is-a-network
faq:
  - q: "What is a packet in networking?"
    a: "A **packet** is a small, self-contained chunk of data. Instead of sending a file or message as one long stream, the network chops it into many packets, each wrapped in a **header** that carries addresses and sequence information around the **payload** (the actual data). The packets travel across the network and are reassembled in the right order at the far end."
  - q: "What is the difference between a switch and a router?"
    a: "A **switch** moves packets within a single local network, forwarding them by hardware (MAC) address. A **router** moves packets between different networks, forwarding them by IP address. Roughly: switches handle traffic inside your network, routers handle traffic that needs to leave it."
  - q: "What does best-effort delivery mean?"
    a: "**Best-effort** means the network tries to deliver every packet but makes no guarantee. Packets can be lost, duplicated, delayed, or arrive out of order. The network itself doesn't fix this — a higher layer such as TCP detects the problems and cleans them up by re-sending and reordering."
  - q: "Why is data split into packets instead of sent all at once?"
    a: "Splitting data into packets lets many conversations share the same links at once, lets each packet be routed independently along whatever path is free, and lets a single lost piece be re-sent without repeating the whole transfer. One giant blob would monopolise a link and have to restart from scratch on any error."
---
# Packets & switching

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Networks don't send your data as one long stream. They chop it into small
**packets**, each a **payload** (the real data) wrapped in a **header** (the
addresses and sequence numbers that steer it). Packets travel **independently** —
**routing** picks a path for each one — and are reassembled in order at the far
end. Delivery is **best-effort**: the network tries but doesn't promise, so
packets can be lost or reordered, and a higher layer sorts that out. If you're new
to the idea of machines talking at all, start with
[what is a network?](/learn/networking/what-is-a-network/).
</div>

Once you have two machines connected, the obvious question is *how* the data
actually gets from one to the other. The answer underpins almost everything else
on the internet, and it's beautifully simple: break the data into pieces, address
each piece, and let the network carry them one hop at a time.

## Why packets

Imagine sending a large file as a single, unbroken stream of bytes down a wire.
While it's in flight, that one transfer would **hog the link** — nobody else's
data could pass until yours finished. And if a single byte were corrupted
somewhere in the middle, you'd have no easy way to fix just that byte; you'd have
to start the whole thing over.

Splitting the data into small **packets** solves all of this at once:

- **Many conversations share one link.** Because packets are small and
  interleaved, your download, a video call, and someone else's web page can all
  travel over the same cable, taking turns fraction-of-a-second by
  fraction-of-a-second. No single transfer monopolises the wire.
- **Each packet is routed independently.** The network decides a path for every
  packet on its own. If one route is busy or breaks, later packets can take
  another — no need to reserve a fixed line end to end.
- **Losses are cheap to fix.** If one packet goes missing, only that packet has to
  be re-sent, not the entire file.

This approach is called **packet switching**, and it's the reason the internet
scales to billions of simultaneous conversations.

## Anatomy of a packet

Every packet has two parts: a **header** wrapped around a **payload**.

- The **payload** is the actual data you're sending — a slice of a file, a piece
  of a web page, a fragment of audio.
- The **header** is the control information the network needs to deliver it:
  **where it's going** (a destination address), **where it came from** (a source
  address), and often a **sequence number** so the pieces can be put back in order
  at the other end.

The classic analogy is an **envelope around a letter**. The letter inside is the
payload — the part you care about. The envelope is the header: it carries the
to-and-from addresses that let the postal system move it, even though it says
nothing about the message itself. The network reads the "envelope," never needing
to open the "letter" to do its job.

## Switches and routers

Packets don't leap straight to their destination. They're passed along by devices
that read each header and forward the packet one step closer. Two kinds do most of
this work:

- A **switch** moves packets **within a single local network** — the machines in
  your home or office. It forwards by **hardware address**, also called a **MAC
  address**, which is baked into each network adapter. A switch essentially learns
  which device is on which of its ports and shuttles packets between them.
- A **router** moves packets **between networks** — for example, from your home
  network out toward a server on the internet. It forwards by **IP address**, the
  logical address we'll meet in [IP addresses](/learn/networking/ip-addresses/).
  Routers are the junctions that stitch millions of separate networks into one
  internet.

Each device is just one **hop**. A packet crossing the internet may pass through a
switch on your side, your home router, a handful of routers owned by internet
providers, and a switch on the destination side — each one reading the header and
forwarding the packet closer to where it's headed.

## Best-effort delivery

Here's the part that surprises newcomers: the network makes **no promise** that a
packet arrives. This is called **best-effort delivery** — every device tries its
best, but there's no guarantee. In practice, packets can be:

- **lost** — dropped when a link is congested or a device is overwhelmed,
- **duplicated** — accidentally sent twice, or
- **delivered out of order** — because independently routed packets can take
  different paths and overtake each other.

The network itself doesn't fix any of this. Instead, a **higher layer** does the
cleanup. **TCP**, which we'll cover in [TCP & UDP](/learn/networking/tcp-and-udp/),
watches the sequence numbers, asks for anything missing to be re-sent, drops
duplicates, and reassembles the pieces in order — turning best-effort packets into
a reliable, ordered stream. This split of jobs is exactly why networks are built
in layers, the subject of the next lesson.

## An analogy

Picture mailing a long message as a stack of **numbered postcards**. You write the
message across, say, forty cards, number them 1 to 40, and drop them in the
mailbox. The postal system doesn't keep them together: different cards may take
different routes, some travel faster than others, and they can land on your
friend's doormat in any order. But because every card is numbered, your friend can
lay them out 1 to 40 and read the message exactly as you wrote it. If card 17
never shows up, they only need you to re-send that one card — not all forty.

That's packet switching in miniature: small addressed pieces, sent independently,
reassembled by their sequence numbers at the far end.

<div class="knowledge-check" data-quiz data-correct-msg="Right — small independent packets let links be shared, let each piece be routed on its own, and let a single loss be re-sent." markdown="0">
  <p class="knowledge-check__q">Quick check: Why is data split into packets instead of sent as one big stream?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because networks can only transmit one byte at a time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So links can be shared, packets routed independently, and losses re-sent</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To encrypt the data so nobody can read it in transit</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Networks split data into small **packets** that travel independently and are
  reassembled at the far end — **packet switching**.
- Packets let many conversations **share links**, get **routed independently**, and
  make lost pieces cheap to **re-send**.
- Each packet is a **header** (addresses, sequence info) wrapped around a
  **payload** (the real data) — like an envelope around a letter.
- **Switches** move packets within a local network by MAC address; **routers** move
  them between networks by IP address; each **hop** forwards the packet closer.
- Delivery is **best-effort** — packets can be lost, duplicated, or reordered — and
  a higher layer like **TCP** cleans that up.

Next up: [protocols & layers](/learn/networking/protocols-and-layers/)
