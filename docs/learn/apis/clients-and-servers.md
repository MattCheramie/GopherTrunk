---
slug: clients-and-servers
title: Clients and servers
description: Who asks and who answers — the request/response roles that shape almost every API. Learn what makes a program a client or a server, why one program can be both, and where peer-to-peer designs differ.
keywords: client server model, what is a client, what is a server, request response, client server architecture, peer to peer, API roles
level: beginner
status: full
---

# Clients and servers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
In the **client–server model**, the **client initiates** — it sends a request when
it wants something — and the **server waits and answers**, sitting at a known
address ready for anyone to ask. These are **roles, not machine types**: one
program is often a server to some parties and a client to others at the same
moment. Almost every API in this module assumes these roles; **peer-to-peer**
systems are the exception, where both sides may initiate.
</div>

Before dissecting any protocol, you need to know who's talking. This lesson
establishes the two roles nearly every API is built around — and why "server"
describes a job, not a big humming box in a data centre.

## Who asks, and who answers?

The split is simple and strict:

- The **client** decides *when* to communicate. It opens the connection, sends a
  request, and waits for the answer. When it has no questions, nothing happens.
- The **server** decides *nothing* about timing. It binds to a known address and
  port, listens, and answers whatever arrives, from whoever arrives, for as long as
  it runs.

That asymmetry is what makes the model scale: a server doesn't need to know its
clients exist until they speak, and a client only needs one thing — the server's
address. Your browser is a client; GopherTrunk's daemon is a server on your LAN;
`curl` is a client you drive by hand. The
[Networking module](/learn/networking/clients-and-servers/) covers how the
connection itself is made; here we care about what the roles mean for APIs.

## Roles, not machines

The most common beginner confusion is thinking "server" means hardware. It doesn't —
it means *the program that answers*. A ten-dollar single-board computer running the
GopherTrunk daemon is a server. Your powerful desktop running `curl` is a client.
The roles are per-conversation, and a single process routinely holds both:

- GopherTrunk's daemon is a **server** to the web console and to your scripts…
- …and simultaneously a **client** of a SoapyRemote SDR server on another machine,
  and a client of RadioReference when fetching system data.

When you later build a webhook receiver in
[Unit 3](/learn/apis/webhooks/), you'll flip roles yourself: your notification
script becomes a *server*, because the daemon needs somewhere to deliver events —
and that role reversal is exactly what makes webhooks interesting.

## Why do APIs love this model?

Request/response fits how most software needs to interact:

| Property | Why it helps an API |
|----------|---------------------|
| Client initiates | The server does zero work until asked — cheap to run, easy to scale |
| Server is stateless between requests (usually) | Any request can be handled on its own; servers can be restarted or multiplied |
| One known address | Clients need only a URL, not a map of the whole system |
| Clear failure story | No answer = the request failed; the client retries or reports |

The "usually stateless" point matters more than it looks: because a well-designed
server treats each request independently, you can restart it, load-balance it, or
replace its implementation without clients noticing — the contract survival idea
from [lesson 1](/learn/apis/what-is-an-api/) made practical.

## Where the model creaks

The client-initiates rule has a famous blind spot: **the server can never speak
first**. If a call starts on your scanner right now, the daemon *knows* — but a pure
request/response client won't find out until it next asks. Working around that
limitation (politely, without asking every 100 ms) is the whole of
[Unit 3](/learn/apis/polling-vs-push/), and it's where WebSockets, server-sent
events, and webhooks come from. Keep the limitation in mind; it explains most of
the real-time machinery you'll meet later.

## What about peer-to-peer?

Some systems drop the asymmetry entirely: in a **peer-to-peer** design, every
participant can both initiate and answer — BitTorrent peers exchange file pieces,
and two radios in TETRA direct mode talk to each other with no tower in between.
Peer-to-peer buys resilience (no single server to fail) at the cost of complexity:
every peer needs discovery, addressing, and server-grade robustness. It's worth
knowing the pattern exists, but the APIs you'll consume and build are
overwhelmingly client–server, and this module stays there.

<div class="knowledge-check" data-quiz data-correct-msg="Right — client and server are per-conversation roles, and one program can hold both at once." markdown="0">
  <p class="knowledge-check__q">Quick check: GopherTrunk's daemon answers requests from the web console while fetching samples from a remote SDR service. What is the daemon?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A server — daemons are always servers</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Both — a server to the console and a client of the SDR service</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A peer — because it talks to two other programs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **client initiates** requests; the **server waits at a known address** and
  answers — an asymmetry that makes the model simple and scalable.
- **Client and server are roles**, not kinds of machine, and one program often
  plays both at once.
- Servers that treat each request **independently** can be restarted or replaced
  without clients noticing.
- The model's blind spot — **the server can never speak first** — motivates all of
  the real-time push techniques in Unit 3.
- **Peer-to-peer** removes the asymmetry, at a cost; almost all APIs stay
  client–server.

Next up: [Data formats: JSON and friends](/learn/apis/data-formats/).
