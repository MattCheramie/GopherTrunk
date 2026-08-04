---
slug: client-server-web
title: The client-server web
description: Every web interaction is one side asking and the other answering. This lesson explains the client-server model on the web — who the client and server are, what a request and response carry, why the model scales, and how it maps to the front end and back end you already know.
keywords: client server, request and response, web client, web server, client-server model, HTTP request, front end back end, stateless web, how the web works
level: beginner
status: full
prereq:
  - how-the-browser-works
faq:
  - q: What is the difference between a client and a server?
    a: "The **client** is the program that starts the conversation by making a request — on the web, almost always a browser. The **server** is the program that listens for requests and sends back a response. One asks, the other answers; the roles are defined by who initiates, not by the hardware."
  - q: Can one machine be both a client and a server?
    a: "Yes. The roles are about the direction of a given request, not fixed labels. A back-end server answering your browser might itself act as a **client** when it calls another service — say a payment provider — for data it needs to build its response."
  - q: Why is the client-server model so common on the web?
    a: "Because it centralises the parts that must be shared and trusted. One server can hold the authoritative data and rules and serve many clients at once, while each client only needs to know how to make a request. That separation is what lets millions of browsers safely use the same application."
---

# The client-server web

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every web interaction is a **request** and a **response**: a **client** asks, a
**server** answers. On the web the client is almost always the **browser** (the
front end) and the server is the machine running your **back end**. One server can
serve many clients at once, which is what makes the model scale. The web is also
**stateless** — each request stands alone — so anything to be remembered gets sent
again next time. This is the same rhythm under
[how a web request works](/learn/networking/how-a-web-request-works/), viewed from
the web developer's chair.
</div>

The first two lessons kept returning to one idea: the front end asks and the back
end answers. That idea has a name — the **client-server model** — and it is the
organising principle of the entire web. Once you can see any web interaction as a
client making a request and a server returning a response, a great deal of what
follows in this module is just detail hung on that frame.

## Who is the client, who is the server

The two roles are defined by **who starts the conversation**:

- The **client** initiates. It sends a request whenever it wants something — a
  page, an image, some data. On the web the client is overwhelmingly the
  **browser**, though a mobile app or a command-line tool can play the same role.
- The **server** waits and responds. It sits listening for requests, does whatever
  work each one needs, and sends back a response. Your **back end** is a server.

Notice the mapping to the last two lessons: the **front end is the client**, the
**back end is the server**. "Client-server," "front end / back end," and "browser /
server" are three ways of naming the same divide, each emphasising a different
angle — the role, the codebase, or the machine.

## Request and response

The unit of a client-server exchange is a **request** paired with a **response**.
The client sends a request that says, in effect, *what* it wants and *how*; the
server sends back a response with the result and some information about it.

A request typically carries:

- A **method** — the kind of action wanted, like *read this* or *submit this*.
- A **path** — which resource it's about, such as `/calls` or `/login`.
- **Headers** — metadata about the request (who's asking, what format is
  acceptable).
- Sometimes a **body** — data being sent, such as the contents of a form.

A response typically carries a **status** (did it work?), its own **headers**, and
usually a **body** — the HTML, the data, the image. This request/response format
is written in a protocol called **HTTP**, which the networking module covers in
[HTTP — the web's protocol](/learn/networking/http/). The whole low-level journey a
single request takes across the network — DNS, connection, and all — is walked
through in [how a web request works](/learn/networking/how-a-web-request-works/).

## Many clients, one server

The reason the web is built this way is that it **scales** and it **centralises
trust**. A single server can hold the one authoritative copy of the data and the
rules, and answer thousands of clients at once. Each client is simple — it only
needs to know how to make a request — while the hard, shared, sensitive parts live
in one place the operator controls.

That's also *why* the back end exists at all. You can't put the master database or
the "who is allowed to do what" rules on the user's device, because the user
controls that device and could change anything on it. Centralising those on a
server the operator runs is what makes a multi-user application possible and safe.
We build out that reasoning in
[what a back end does](/learn/web-dev/what-a-backend-does/).

## Roles can swap

The client and server labels describe a *single* request, not a permanent
identity. The same machine can be both, depending on the direction of a given
call.

Picture your back-end server handling a browser's request for a page. To build
that page it might need exchange-rate data, so it calls another company's API. In
*that* exchange your server is the **client** and the other company's is the
**server**. Web systems are full of these chains — browser to your server, your
server to a database, your server to a third-party API — each link a small
client-server conversation. Seeing the pattern repeat is more useful than
memorising which box is which.

## Stateless by default

One property shapes everything built on top of the model: HTTP is **stateless**.
The server handles each request on its own and, by default, remembers nothing
about the previous one. Two clicks a second apart arrive as complete strangers to
each other.

That keeps servers simple and lets any server in a pool handle any request, but it
raises an obvious problem: how does a site know you're still logged in on the next
click? The answer is that the client **resends** the proof every time — a cookie or
token travels with each request, reconstructing continuity out of many independent
exchanges. This is such a central idea that a later lesson,
[authentication & sessions](/learn/web-dev/authentication-and-sessions/), is
devoted to it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the client is whoever initiates the request; on the web that's usually the browser." markdown="0">
  <p class="knowledge-check__q">Quick check: in the client-server model, what makes something the client?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's the more powerful machine in the exchange</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It initiates the request — on the web, usually the browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's always the machine in the data centre</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **client-server model** is the web's organising principle: a **client**
  requests, a **server** responds.
- On the web the **client is the browser** (the front end) and the **server runs
  the back end** — three names for one divide.
- A **request** carries a method, path, headers, and sometimes a body; a
  **response** carries a status, headers, and usually a body — all over **HTTP**.
- One server can serve **many clients**, centralising the authoritative data and
  rules where the operator controls them.
- Roles can **swap** per request — a server calling another service is itself a
  client.
- HTTP is **stateless**, so continuity like being logged in is re-sent with every
  request.

Next up: [Anatomy of a modern web app](/learn/web-dev/anatomy-of-a-web-app/).
