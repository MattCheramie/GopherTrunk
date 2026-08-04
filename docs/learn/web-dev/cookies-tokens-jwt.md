---
slug: cookies-tokens-jwt
title: Cookies, tokens & JWTs
description: How the browser remembers you between stateless requests — session cookies, bearer tokens, and JSON Web Tokens (JWTs) — with the security flags and tradeoffs that decide which to reach for.
keywords: cookie, session cookie, HttpOnly, Secure, SameSite, bearer token, JWT, JSON Web Token, access token, refresh token, authorization header
level: intermediate
status: full
prereq:
  - authentication-and-sessions
faq:
  - q: "What's the difference between a cookie and a token?"
    a: "A **cookie** is a small named value the browser stores and *automatically* attaches to matching requests — you don't write code to send it. A **token** (like a JWT) is a string your code stores and sends *deliberately*, usually in an `Authorization` header. Cookies are convenient for browser apps; tokens are common for APIs and mobile clients. They're not mutually exclusive — a token can even be delivered inside a cookie."
  - q: "Are JWTs encrypted?"
    a: "No — a standard **JWT is signed, not encrypted**. Anyone holding it can read its contents (the payload is just base64-encoded JSON), but the signature lets the server detect tampering. So never put secrets in a JWT payload, and always send it over HTTPS so it can't be read or stolen in transit."
  - q: "Why can't I just log the user out by deleting their JWT?"
    a: "Because a self-contained JWT is valid until it **expires**, no matter what the server does — the server keeps no record of it to delete. That's the tradeoff for being stateless. Real systems keep JWT lifetimes short and pair them with a **refresh token** or a revocation list when instant logout matters."
---

# Cookies, tokens & JWTs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The [session](/learn/web-dev/authentication-and-sessions/) identifier has to ride on
every request — the three common vehicles are **cookies**, **bearer tokens**, and
**JWTs**. A **cookie** is stored by the browser and sent automatically; harden it
with the **HttpOnly**, **Secure**, and **SameSite** flags. A **bearer token** is
sent deliberately in an `Authorization` header, common for APIs. A **JWT** is a
*signed, not encrypted* token that carries its own claims so the server can trust it
without a lookup — powerful, but hard to revoke before it **expires**. Which you
pick trades convenience against control.
</div>

The previous lesson ended with a session identified by a token the browser sends
back on every request. This lesson is about that token: what form it takes, how it
travels, and how to keep it from being stolen. Get this wrong and a stolen token
means a stolen account — so the security flags here matter as much as the mechanics.

## Cookies: the browser's memory

A **cookie** is a small named value the server asks the browser to store, using a
`Set-Cookie` response header. From then on the browser **automatically** includes it
on every request to that site — you write no code to send it. That automatic
behaviour is what makes cookies the classic home for a session ID.

```http
Set-Cookie: session=abc123; HttpOnly; Secure; SameSite=Lax; Max-Age=3600
```

Three flags on that header do the security heavy lifting:

- **HttpOnly** — JavaScript can't read the cookie. This blunts
  [cross-site scripting](/learn/web-dev/web-security-essentials/) from stealing the
  session, because a script that runs on your page still can't see the value.
- **Secure** — the cookie is only sent over HTTPS, so it never crosses the network
  in the clear. (See [TLS & HTTPS](/learn/networking/tls-and-https/).)
- **SameSite** — controls whether the cookie rides along on requests coming *from
  other sites*. `Lax` or `Strict` is a primary defence against
  [CSRF](/learn/web-dev/web-security-essentials/), which abuses the browser's
  automatic cookie-sending.

A session cookie holds an **opaque ID**; the real state lives on the server. That
makes logout trivial — the server forgets the session and the ID is worthless.

## Tokens: identity you send on purpose

The other model hands the client a **token** — a string it stores itself and sends
*explicitly*, usually as a **bearer token** in an HTTP header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Nothing is automatic here: your [front-end code](/learn/web-dev/fetching-data/)
attaches the header on each request. That extra work buys flexibility — the same
API can serve a browser app, a mobile app, and other servers, none of which share
the browser's cookie machinery. Because tokens aren't sent automatically, they
sidestep CSRF; the cost is that *you* must store them somewhere safe, and browser
storage readable by JavaScript is exposed to XSS.

## JWTs: a token that carries its own claims

A **JSON Web Token (JWT)** is a specific, popular token format. It packs a small
JSON payload — the **claims**, such as the user ID and an expiry time — and **signs**
it. The three dot-separated parts are a header, the payload, and a signature:

```
header.payload.signature   ->   eyJ...  .  eyJ...  .  SflKx...
```

The signature is the point. The server signs the token with a secret at login; on
each later request it re-checks the signature. If it matches, the claims are
trustworthy and **the server needs no database lookup** to know who you are — the
identity travels *inside* the token. That statelessness is the JWT's whole appeal,
especially across many servers.

Two things trip people up, both worth memorising:

- **A JWT is signed, not encrypted.** The payload is readable by anyone holding the
  token. Never put a secret in it, and always send it over HTTPS.
- **A self-contained JWT is hard to revoke.** Because the server stores nothing, a
  leaked token is valid until it **expires** — deleting it server-side isn't
  possible. The fix is short lifetimes plus a **refresh token** (a longer-lived
  credential used to mint new short access tokens), or a server-side revocation list
  that trades away some of the statelessness.

## Choosing between them

There's no universal winner — match the tool to the client:

| | Session cookie | Bearer token / JWT |
|---|---|---|
| **Sent** | Automatically by the browser | Deliberately, in a header |
| **State** | Server-side (opaque ID) | Often self-contained (JWT) |
| **Revoke / logout** | Easy — drop the session | Hard for stateless JWTs |
| **Main risk** | CSRF (mitigate with SameSite) | Theft via XSS if JS-readable |
| **Good fit** | Classic server-rendered web apps | APIs, mobile, cross-service |

A common, sound default for a browser app is **session cookies with HttpOnly,
Secure, and SameSite set**. Reach for tokens when non-browser clients or many
independent services are in the picture. Whatever you choose, the token is the keys
to the account — protect it as such.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a JWT is signed for integrity but readable by anyone, so never put secrets in the payload." markdown="0">
  <p class="knowledge-check__q">Quick check: is it safe to store a user's secret data inside a JWT payload?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — the signature encrypts the payload so no one can read it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">No — a JWT is signed, not encrypted; the payload is readable by anyone holding it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — JWTs are always sent over HTTPS so the contents are hidden</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The session token has to travel on every request; the three vehicles are
  **cookies**, **bearer tokens**, and **JWTs**.
- A **cookie** is stored and sent **automatically** by the browser; harden it with
  **HttpOnly** (no JS access), **Secure** (HTTPS only), and **SameSite** (limits
  cross-site sending).
- A **bearer token** is sent **deliberately** in an `Authorization` header, which
  suits APIs and non-browser clients and avoids CSRF.
- A **JWT** carries **signed** claims so the server can trust it without a lookup —
  but it's readable (**not encrypted**) and **hard to revoke** before it **expires**.
- Match the tool to the client: cookies for classic web apps, tokens for APIs and
  cross-service; either way, guard the token like the account it unlocks.

Next up: [WebSockets & real-time updates](/learn/web-dev/websockets-and-realtime/).
