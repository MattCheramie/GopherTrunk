---
slug: clients-and-servers
title: Clients, servers & peers
description: A plain-language look at the client-server model — how a client initiates a request and a server listens and responds, why a server is a role rather than special hardware, the request-response rhythm behind HTTP, and how peer-to-peer offers a decentralized alternative.
keywords: client server, request response, peer to peer, server, listen, service, P2P, client, client-server model, HTTP request, decentralized
level: beginner
status: full
prereq:
  - what-is-a-network
faq:
  - q: What is the difference between a client and a server?
    a: A client is the program that starts an exchange by sending a request; a server is the program that waits for requests and sends back a response. Your browser is a client; the machine hosting a website is a server. The client always initiates; the server answers.
  - q: What does a server actually do?
    a: A server waits — it listens for incoming connections and, when one arrives, handles the request and sends a response. That's the whole job. A server is a role a program plays, not necessarily a special or powerful machine.
  - q: What is peer-to-peer?
    a: "Peer-to-peer (P2P) is a model with no central server: every participant, called a peer, both requests and serves. Peers connect directly to each other to share the work. File sharing, some chat systems, and blockchains are built this way."
  - q: Can one computer be both a client and a server?
    a: Yes. Client and server are roles, not fixed identities. The same machine can serve web pages to others while also acting as a client when it fetches an update or calls another service. A program can even do both at once.
---

# Clients, servers & peers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Most conversations on a [network](/learn/networking/what-is-a-network/) follow the
**client / server** model: a **client** asks and a **server** answers. The basic
rhythm is **request-response** — the client sends a request, the server sends back
a response, and the client always starts the exchange. The main alternative is
**peer-to-peer**, where there's no central server and every participant both asks
and answers. Most of the internet is clients asking and servers answering, with
peer-to-peer as the decentralized alternative.
</div>

Almost every time two machines talk, one of them is asking for something and the
other is providing it. Naming those two roles — and seeing the one pattern they
follow — makes the rest of networking much easier to reason about.

## The client-server model

The most common way for programs to talk over a network is the **client-server
model**, and it comes down to two roles:

- A **client** initiates. It's the program that wants something and sends a
  **request** to get it — a browser loading a page, an app fetching your
  messages, a script calling an API.
- A **server** responds. It sits ready, receives the request, and sends back a
  **response** with whatever was asked for.

The key detail is direction: **the client starts every exchange.** A server
never reaches out first — it waits to be asked, then replies. If nothing sends a
request, a server simply sits idle. That one rule explains most of what follows.

## What a server does

A server's job sounds almost boring: it **waits**. More precisely, it **listens**
for incoming connections on a known [port](/learn/networking/ports-and-sockets/)
and, when a connection arrives, handles the request and sends a response. Listen,
accept, answer — over and over.

Two things trip beginners up here.

First, **a server is a role, not special hardware.** We picture servers as
humming racks in a data centre, and many are — but "server" just means a program
playing the answering role. A Raspberry Pi on your desk, or even the laptop
you're reading this on, becomes a server the moment it runs software that listens
for requests.

Second, **one machine can be both client and server.** These are roles, not fixed
identities. A computer can serve web pages to others while, in the same breath,
acting as a client to fetch a software update or call another service. The labels
describe what a program is doing in a given exchange, nothing more.

## Request and response

The heartbeat of the client-server model is **request-response**: a request goes
in, a response comes out. The client asks a specific question — "give me this
page", "save this record" — and the server sends back a specific answer, whether
that's the data requested or a message explaining why it couldn't.

The classic example is **HTTP**, the protocol behind the web. Your browser (the
client) sends an HTTP request for a page; the web server sends an HTTP response
carrying the page back. We'll unpack that exchange in detail in
[HTTP](/learn/networking/http/), and see how the same pattern powers APIs in
[web APIs & REST](/learn/networking/web-apis-and-rest/).

This request-then-response rhythm is so common that once you spot it, you'll
recognise it almost everywhere data moves across a network.

## Peer-to-peer

Not every system has a central server. In the **peer-to-peer** (P2P) model there's
no single machine everyone depends on. Instead, every participant is a **peer**
that both **requests and serves** — asking others for what it needs while
providing what it has in return. The peers connect directly to one another rather
than routing everything through a middle authority.

You've met P2P more often than you might think: file-sharing networks spread
downloads across many peers, some chat and calling systems connect people
directly, and blockchains keep a shared ledger with no central owner.

The trade-offs are real. Peer-to-peer avoids a single point of failure and can
scale as more peers join, since each newcomer brings capacity as well as demand.
But it's harder to coordinate, harder to secure, and harder to keep consistent
than a straightforward client-server setup, where one authoritative server calls
the shots. Most of the internet sticks with client-server for exactly that
reason; P2P shines when decentralization is worth the extra complexity.

## Where GopherTrunk fits

You're already using the client-server model to read this. GopherTrunk's **web
interface** is a **server**: it listens on your network, and your **browser** is
the **client** that sends requests and renders the responses. Every meter,
histogram, and control page you open is a request-response exchange.

That's also a preview of the end of this path. Running GopherTrunk's interface
means putting your own **service** on the network for other devices to reach — the
practical side of being a server. We'll turn that into a hands-on setup in
[running a server](/learn/networking/running-a-server/) and
[GopherTrunk on the network](/learn/networking/gophertrunk-on-the-network/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the client sends the request that begins every exchange." markdown="0">
  <p class="knowledge-check__q">Quick check: in the client-server model, who starts the exchange?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">The client (it sends the request)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The server (it reaches out first)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whichever machine is more powerful</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- In the **client-server** model, a **client** initiates a **request** and a
  **server** listens and sends a **response**.
- The **client always starts** the exchange; a server waits to be asked.
- A **server** is a **role**, not special hardware — and one machine can be both
  client and server.
- **Request-response** is the basic rhythm, and **HTTP** on the web is the classic
  example.
- **Peer-to-peer** drops the central server: peers connect directly and both
  request and serve, trading simplicity for decentralization.
- GopherTrunk's web interface is a **server** your **browser** talks to — a preview
  of putting your own service on the network.

Next up: [how a web request works, end to end](/learn/networking/how-a-web-request-works/)
