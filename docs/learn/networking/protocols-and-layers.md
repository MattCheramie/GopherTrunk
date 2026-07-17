---
slug: protocols-and-layers
title: Protocols & layers
description: A plain-language guide to network protocols and layers — what a protocol is, why networking is split into layers, the four practical TCP/IP layers, how the OSI model compares, and how encapsulation wraps data as it moves down the stack.
keywords: protocol, network layers, TCP/IP model, OSI model, encapsulation, application layer, transport layer, link layer, internet layer, network stack
level: beginner
status: full
prereq:
  - packets-and-switching
faq:
  - q: What is a network protocol?
    a: A protocol is an agreed set of rules that two machines follow so they can communicate. It defines how messages are formatted, what order they're sent in, and what each side does in response. For it to work, both ends must speak the same protocol — one machine sending in a format the other doesn't understand gets nowhere.
  - q: What is the TCP/IP model?
    a: "The TCP/IP model describes networking as four practical layers stacked on top of each other: link, internet, transport, and application. Each layer handles one job and relies on the layer below it. It's the model the real internet is built on, which is why it's the one worth learning first."
  - q: What is the difference between the TCP/IP model and the OSI model?
    a: "They describe the same thing at different levels of detail. The TCP/IP model has four layers and maps directly onto how the internet actually works. The OSI model splits the same ground into seven layers and is mostly used as a teaching and reference framework — you'll hear people say 'layer 7' meaning the application layer, for example."
  - q: What is encapsulation in networking?
    a: "Encapsulation is how the layers cooperate: each layer wraps the data from the layer above with its own header, like putting an envelope inside a bigger envelope. As data moves down the stack it gains a header at each layer; the receiving machine unwraps them in reverse order, each layer reading and removing its own header."
---

# Protocols & layers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **protocol** is an agreed set of rules two machines follow to talk to each
other — both ends have to speak the same one. Networking is far too big for a
single protocol, so it's split into **layers**, each handling one job and
relying on the layer below. The **TCP/IP model** names four practical layers:
link, internet, transport, and application. Data moves down the stack by
**encapsulation** — each layer wraps the layer above in its own header, like
envelopes inside envelopes. Splitting the work this way lets each part work, and
change, independently.
</div>

You already know data crosses a network as
[packets](/learn/networking/packets-and-switching/). But something has to decide
what goes in each packet, how machines agree on its meaning, and who handles
which part of the journey. That's what protocols and layers are for, and they're
the mental model the rest of this path hangs on.

## What a protocol is

A **protocol** is simply an agreed set of rules that two machines follow in order
to communicate. It spells out the details they'd otherwise disagree on: how a
message is formatted, what order things are sent in, how the other side should
reply, and what to do when something goes wrong.

The key word is *agreed*. A protocol only works if **both ends speak the same
one**. If your machine sends a request in a format the server doesn't recognise,
nothing happens — the same way two people need a shared language to hold a
conversation. Human protocols work like this too: a phone call has an
understood ritual of "hello", take turns, then "goodbye". Network protocols are
that idea written down precisely enough for machines to follow it exactly.

## Why layers

Here's the problem: everything that has to happen for one machine to talk to
another is far too much for a single protocol to handle. Getting bits across a
cable, finding a route to a distant machine, recovering lost packets, and
formatting a web page are wildly different jobs.

So networking is **split into layers**. Each layer takes responsibility for one
job and trusts the layer below it to handle everything beneath. The layer that
formats a web request doesn't care whether the connection is Wi-Fi or fibre —
that's a lower layer's problem. The layer moving bits over the wire doesn't care
whether they're part of an email or a video — that's an upper layer's problem.

This separation is the whole trick. Because each layer only has to talk to the
ones directly above and below it, you can **change one layer without touching the
others**. Swap Wi-Fi for ethernet and every app keeps working. Invent a new app
protocol and it runs over the existing internet unchanged.

## The four practical layers (TCP/IP)

The model the real internet is built on is the **TCP/IP model**, and it has four
layers. From the bottom up:

- **Link** — moving data across one physical hop, like Ethernet or Wi-Fi.
  This layer handles local delivery between directly connected devices.
- **Internet** — getting packets across networks to the right machine, using
  **IP** for addressing and routing. (We'll cover
  [IP addresses](/learn/networking/ip-addresses/) soon.)
- **Transport** — managing the conversation between two programs: setting up
  connections and, where needed, making delivery reliable. This is **TCP** and
  **UDP**. (Coming up in [TCP vs. UDP](/learn/networking/tcp-and-udp/).)
- **Application** — the protocols programs actually use, like **HTTP** for the
  web, **DNS** for names, and **SSH** for remote logins.

A short way to hold it:

| Layer | Its job | Example protocols |
| --- | --- | --- |
| Application | What the program wants to say | HTTP, DNS, SSH |
| Transport | Connections and reliability between programs | TCP, UDP |
| Internet | Addressing and routing across networks | IP |
| Link | Local delivery over one physical hop | Ethernet, Wi-Fi |

You'll also hear about the **OSI model**, an older seven-layer description of the
same ground. It splits things more finely and is mostly used as a teaching and
reference framework — when someone says "layer 7" they mean the application
layer. For understanding the internet as it really works, the four TCP/IP layers
are enough.

## Encapsulation

How do the layers actually cooperate on a single packet? Through
**encapsulation**. As data heads down the stack, **each layer wraps the data from
the layer above with its own header** — a small block of control information for
that layer.

Picture envelopes inside envelopes. Your web request (application) is placed
inside a transport envelope with a TCP header, which goes inside an internet
envelope with an IP header, which goes inside a link envelope with an Ethernet
header. Each header carries exactly what that layer needs — a port, an IP
address, a hardware address — and nothing more.

The receiving machine does the reverse. It **unwraps the envelopes in order**:
the link layer reads and removes its header, hands what's left up to the internet
layer, which does the same, and so on until the original web request arrives at
the application. Each layer only ever reads its own header and ignores the rest.

## Why it matters to you

Most of the time you work at the **application layer** — you make an HTTP
request, look up a name, open an SSH session — and you **trust the lower layers**
to carry it. That trust is the point: layering lets you build without thinking
about routing or cables.

But when something breaks, knowing the stack tells you **where to look**. Can't
resolve a name? That's DNS, an application-layer problem. Connection times out
but the name resolves? Suspect the transport or internet layer. Nothing reaches
the machine at all? Check the link. Working *down* the layers is exactly how the
command-line tools in
[testing connectivity](/learn/networking/testing-connectivity/) diagnose a
connection that won't come up.

<div class="knowledge-check" data-quiz data-correct-msg="Right — IP handles addressing and routing at the internet layer." markdown="0">
  <p class="knowledge-check__q">Quick check: which layer is IP (addressing and routing) part of?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The link layer</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The internet (network) layer</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The application layer</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **protocol** is an agreed set of rules two machines follow to communicate; both ends must speak the same one.
- Networking is too big for one protocol, so it's split into **layers**, each doing one job and relying on the layer below.
- The **TCP/IP model** has four layers: **link** (local delivery), **internet** (IP, addressing and routing), **transport** (TCP/UDP, connections and reliability), and **application** (HTTP, DNS, SSH).
- The **OSI model** describes the same ground in seven layers and is used mainly as a reference.
- **Encapsulation** wraps data in a header at each layer on the way down, like envelopes inside envelopes; the receiver unwraps them in reverse.
- You usually work at the **application layer** and trust the rest — but knowing the stack tells you where to look when something breaks.

Next up: [clients, servers & peers](/learn/networking/clients-and-servers/)
