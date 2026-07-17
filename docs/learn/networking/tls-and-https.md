---
slug: tls-and-https
title: TLS & HTTPS
description: What TLS and HTTPS actually do — encrypting web traffic, proving a server's identity, how the handshake works, what certificates and certificate authorities are, what the padlock means (and doesn't), and free certs from Let's Encrypt.
keywords: TLS, HTTPS, SSL, certificate, certificate authority, encryption in transit, handshake, Let's Encrypt, chain of trust, padlock
level: intermediate
status: full
prereq:
  - http
faq:
  - q: What is the difference between HTTP and HTTPS?
    a: "HTTPS is HTTP over TLS. Ordinary HTTP sends requests and replies in the clear, so anyone between you and the server can read or alter them. HTTPS wraps that same HTTP traffic in a TLS-encrypted channel and verifies the server's identity first, which is what the browser padlock represents."
  - q: What is a TLS certificate?
    a: "A certificate is a small signed file that binds a domain name to a public key. It is signed by a certificate authority your device already trusts, so your browser can check that the server presenting it really controls that domain. If the signature or name doesn't match, the browser warns you."
  - q: Is TLS the same as SSL?
    a: "In everyday use, yes. SSL was the original protocol; TLS is its modern successor and the two names are still used interchangeably. When people say SSL certificate they almost always mean a TLS certificate. The old SSL versions are obsolete and disabled — everything today is really TLS."
  - q: Does the padlock mean a website is safe?
    a: "No. The padlock means the connection is encrypted and the server's identity was verified — nothing more. A scam site can hold a valid certificate for its own domain and show a padlock. TLS protects the traffic in transit; it does not vouch for who is running the site or whether they are honest."
---

# TLS & HTTPS

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**TLS** wraps ordinary [HTTP](/learn/networking/http/) in encryption and proves the
server is who it claims to be. HTTP-over-TLS is **HTTPS** — that's the padlock in your
address bar. The proof comes from a **certificate**: a signed file binding a domain
name to a public key, vouched for by a **certificate authority (CA)** your device
already trusts. The padlock means *encrypted and identity-verified* — it does **not**
mean the site itself is safe or honest.
</div>

Plain [HTTP](/learn/networking/http/) was designed in an era of trust: requests and
replies travel in the clear, and any device along the path can read or quietly change
them. TLS closes that gap. It's the layer that turned the web from something you'd never
type a password into to something you bank on — literally.

## What TLS adds

TLS (Transport Layer Security) sits *underneath* HTTP and adds three guarantees before
a single web request is sent:

- **Confidentiality** — the traffic is **encrypted**, so anyone in between (your ISP, a
  café's Wi-Fi, a router along the way) sees scrambled bytes, not your passwords or
  page content.
- **Integrity** — the data is **tamper-evident**. If someone flips a bit in transit,
  the receiver detects it and rejects the message rather than acting on altered data.
- **Authentication** — the server **proves who it is**, so you know you're really
  talking to your bank and not an impostor that intercepted the connection.

HTTP with all three wrapped around it is **HTTPS**. It's the same HTTP you already
know — same methods, same status codes — just carried inside a secure channel.

## The handshake (in brief)

Before any HTTP flows, the client and server run a short **handshake** to set up the
secure channel. Conceptually, it goes like this:

1. The client says hello and offers the TLS versions and cipher options it supports.
2. The server picks the parameters both sides can use and **presents its certificate**.
3. The client checks that certificate (more on that below), and the two sides use public-key
   cryptography to **establish a shared secret key** that no eavesdropper can derive.
4. From then on, everything — every request and response — is **encrypted** with that
   shared key.

You don't have to know the math. The point is that by the time your browser sends
`GET /`, it has already confirmed the server's identity and agreed on keys, so the
request never travels in the clear.

## Certificates and certificate authorities

The identity proof rests on a **certificate**. A certificate binds a **domain name**
(like `example.com`) to a **public key**, and it's **signed** by a **certificate
authority (CA)**. Your operating system and browser ship with a built-in list of CAs
they trust.

When a server presents its certificate, your device checks the CA's signature against
that trusted list. This is the **chain of trust**: you trust the CA, the CA vouches for
the domain, so you can trust that this really is that domain's server. If the name
doesn't match, the certificate has expired, or no trusted CA signed it, the browser
throws a warning instead of connecting.

Certificates used to be a paid, fiddly chore. Today **[Let's Encrypt](https://letsencrypt.org/)**
issues them **free** and automatically, which is a big reason nearly the whole web is
now HTTPS.

## The padlock — and its limits

The padlock icon is worth reading precisely. It means two things and only two things:
the connection is **encrypted**, and the server's **identity was verified** against a
trusted CA.

It does **not** mean:

- **The site is trustworthy or safe.** A phishing site can obtain a perfectly valid
  certificate for *its own* domain and show a padlock. TLS secures the pipe; it says
  nothing about who's on the other end or their intentions. Always check the domain
  name itself, not just the lock.
- **Nothing leaks.** TLS hides the *content* of your traffic, but not the fact that you
  connected to a given server — some metadata is still visible, including the server's
  address and, unless you use [encrypted DNS](/learn/networking/dns/), the domain names
  you look up.

Encrypted and verified is a strong guarantee. It is not the same as *good*.

## Why it matters to you

If you expose anything over a network — a web UI, an API, a dashboard — it should be
served over **HTTPS**, not plain HTTP. Sending credentials or session data in the clear
is exactly the gap TLS was built to close.

In practice you rarely wire TLS into your application directly. A
**[reverse proxy](/learn/networking/proxies-and-load-balancers/)** usually **terminates
TLS** in front of your service: it holds the certificate, decrypts incoming HTTPS, and
forwards plain HTTP to your app on a private network. That keeps certificate handling in
one place and is the standard pattern when
[exposing a service safely](/learn/networking/exposing-a-service-safely/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the certificate proves identity, not safety." markdown="0">
  <p class="knowledge-check__q">Quick check: besides encrypting traffic, what does a valid TLS certificate prove?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That the website is safe and trustworthy</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">That the server is who it claims to be (its identity)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That your connection is fast</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **TLS** encrypts web traffic and verifies the server's identity; **HTTPS** is
  [HTTP](/learn/networking/http/) carried over TLS.
- It adds three things: **confidentiality**, **integrity**, and **authentication**.
- A short **handshake** verifies the certificate and establishes shared keys before any
  HTTP flows.
- A **certificate** binds a domain to a public key and is signed by a **CA** your device
  already trusts — the chain of trust; **Let's Encrypt** makes certs free.
- The **padlock** means *encrypted and identity-verified*, **not** that the site is safe.
- Expose services over HTTPS; a **reverse proxy** commonly terminates TLS for you.

Next up: [web APIs & REST](/learn/networking/web-apis-and-rest/)
