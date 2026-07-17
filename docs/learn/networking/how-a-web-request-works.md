---
slug: how-a-web-request-works
title: How a web request works, end to end
description: A plain-language walkthrough of what happens when you type a URL and press Enter — parsing the URL, a DNS lookup, opening a TCP connection, the TLS handshake, the HTTP request and response, and rendering the reply — the whole spine of a web request.
keywords: how a web request works, DNS, TCP, TLS, HTTP, URL, browser request, round trip, HTTPS, request response, web request steps, what happens when you type a URL
level: beginner
status: full
prereq:
  - clients-and-servers
faq:
  - q: What happens when you type a URL and press Enter?
    a: "Your browser parses the URL, does a DNS lookup to turn the hostname into an IP address, opens a TCP connection to that address, secures it with a TLS handshake for https, sends an HTTP request, and receives a response it then renders. In short: DNS, TCP, TLS, HTTP, response — a precise sequence that runs in a fraction of a second."
  - q: What is the first network step of a web request?
    a: "A DNS lookup. Before anything can connect, the human-friendly hostname in the URL has to be turned into a numeric IP address, because the network delivers data by address, not by name. Only once your browser has the IP can it open a connection."
  - q: What is the difference between HTTP and HTTPS?
    a: "HTTP is the request-and-response language the browser and server speak. HTTPS is the same language carried over a TLS-encrypted connection, so the traffic is private and the server's identity is verified by a certificate. HTTPS is HTTP after a TLS handshake has secured the pipe."
  - q: Why is it useful to know the steps of a web request?
    a: "Almost every connection problem is one of these steps failing — the name won't resolve, the connection won't open, the certificate is rejected, or the server returns an error. Knowing the sequence tells you exactly where to look instead of guessing."
---

# How a web request works, end to end

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every time you load a page, the same spine runs underneath:
**DNS → TCP → TLS → HTTP → response**. Typing a
[URL](/learn/networking/clients-and-servers/) and pressing Enter kicks off a
precise sequence — the browser turns the name into an address, opens a
connection, secures it, asks for the page, and renders the reply. This lesson
traces the **whole spine** in one pass; the rest of the path unpacks each step in
detail.
</div>

You've used this thousands of times without seeing it. Under the hood, a single
web request is a short, ordered chain of steps, and each one has to succeed
before the next can start. Learn the chain once and the whole path ahead — DNS,
ports, TCP, TLS, HTTP — stops feeling like a pile of unrelated acronyms and
starts looking like one journey.

## The one-sentence version

Here's the whole thing in a breath: **the browser turns the name into an
address, opens a connection, secures it, asks for the page, and renders the
reply.** Everything below is just that sentence, slowed down.

## Step by step

Say you type `https://example.com/about` and press Enter. Roughly this happens,
in order:

1. **Parse the URL.** The browser breaks the address into parts: the **scheme**
   (`https`, which says how to talk), the **host** (`example.com`, which says
   who to talk to), and the **path** (`/about`, which says what to ask for).
   These three pieces drive every step that follows.

2. **DNS lookup.** The host is a name, but the network delivers data by number,
   so the browser resolves `example.com` to an **IP address** through
   [DNS](/learn/networking/dns/) — the internet's directory. This is the first
   network step, and until it finishes the browser doesn't even know where to
   send anything.

3. **Open a TCP connection.** With an IP in hand, the browser opens a
   [TCP](/learn/networking/tcp-and-udp/) connection to that address on the
   right **port** (443 for https). TCP's short handshake sets up a reliable,
   ordered pipe between the two machines. The pair of (address, port) it
   connects to is a [socket](/learn/networking/ports-and-sockets/).

4. **TLS handshake.** Because the scheme is `https`, the browser and server
   negotiate encryption and the browser **verifies the server's certificate**
   before any real data flows — this is the
   [TLS](/learn/networking/tls-and-https/) handshake. It proves you're talking
   to the real `example.com` and scrambles everything from here on. (Plain
   `http` skips this step.)

5. **HTTP request.** Now the browser actually asks for the page. It sends an
   [HTTP](/learn/networking/http/) request — a line like `GET /about` plus a set
   of **headers** describing the browser, what formats it accepts, and any
   cookies. This is the "please send me this" message.

6. **Server responds.** The server answers with an HTTP **response**: a
   **status code** (`200 OK`, `404 Not Found`, and so on), its own **headers**,
   and the **body** — the actual HTML, image, or data you asked for.

7. **Render or use.** The browser reads the body and **draws the page**,
   fetching any extra pieces (styles, images, scripts) with more requests that
   repeat this same chain. If the caller is your own program rather than a
   browser, it simply **uses the data** it got back — the pattern behind
   [web APIs and REST](/learn/networking/web-apis-and-rest/).

## A simple diagram

A quick sketch of who talks to whom, in order:

```
  You type a URL
        │
        ▼
  ┌───────────┐   1. "what's the IP for example.com?"
  │  Browser  │ ─────────────────────────────▶ ┌─────┐
  │ (client)  │ ◀───────────────────────────── │ DNS │
  └───────────┘   2. "it's 93.184.x.x"          └─────┘
        │
        │  3. open TCP  ·  4. TLS handshake
        │  5. GET /about
        ▼
  ┌───────────┐                                 ┌─────────┐
  │  Browser  │ ─────────────────────────────▶  │ Server  │
  │           │ ◀───────────────────────────── │         │
  └───────────┘   6. 200 OK + page body         └─────────┘
        │
        ▼
  7. render the page
```

The same round trip repeats for every extra file the page needs.

## Why trace it

Almost every "why won't this connect?" is one of these exact steps failing. The
name won't resolve (DNS). The connection never opens (TCP, or a firewall on the
port). The certificate is rejected (TLS). The server answers with an error
status (HTTP). Because the steps run in a fixed order, the point where things
break tells you **where to look** — you stop guessing and start checking the
chain in sequence. That's precisely the mindset behind
[testing connectivity](/learn/networking/testing-connectivity/) later in this
path.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the name has to become an IP address before anything can connect." markdown="0">
  <p class="knowledge-check__q">Quick check: after you type a domain and press Enter, what's the FIRST network step?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The TLS handshake to encrypt the connection</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A DNS lookup to turn the name into an IP address</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Sending the HTTP GET request for the page</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A web request is a fixed chain: **DNS → TCP → TLS → HTTP → response**.
- The browser **parses the URL** into scheme, host, and path first.
- **DNS** turns the host name into an IP address — the first network step.
- **TCP** opens the connection; for https, **TLS** encrypts it and checks the
  certificate.
- The browser sends an **HTTP request** and gets back a status, headers, and a
  body, then **renders** it (or your code uses the data).
- The steps run in order, so a broken connection points to the exact step that
  failed.

Next up: Module 2 covers how machines find each other — [IP addresses & subnets](/learn/networking/ip-addresses/)
