---
slug: what-a-backend-does
title: What a back end does
description: The work a web app can't trust to the browser — business logic, the database, authentication, secrets, and talking to other services — and why every serious web app needs a server behind the front end doing the things the client can't.
keywords: back end, server, business logic, database, authentication, secrets, API keys, why a backend, client cannot be trusted, server-side, trusted environment
level: beginner
status: full
prereq:
  - anatomy-of-a-web-app
faq:
  - q: Why can't everything just run in the browser?
    a: "Because the browser is a place you don't control and can't trust. Users can read all its code, change it, and forge its requests, so any rule enforced only in the browser can be bypassed. Data that must be shared, secrets that must stay hidden, and logic that must be trusted all belong on a server you control — the back end."
  - q: What is the difference between the front end and the back end?
    a: "The front end is the part that runs in the user's browser — the HTML, CSS, and JavaScript they can see and interact with. The back end runs on a server you control, out of the user's reach, holding the business logic, the database, secrets, and authentication. The two talk over HTTP, usually through an API."
  - q: Does a static site have a back end?
    a: "Not necessarily. A purely static site is just files served as-is, with no server-side logic — this site is largely like that. You need a back end when the app must store shared data, keep secrets, enforce rules, or do work that can't be trusted to the client."
---

# What a back end does

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **back end** is the part of a web app that runs on a **server you control**, out of
the user's reach. It exists because the **browser can't be trusted**: users can read,
change, and forge everything the front end does. So the work that must be **trusted or
hidden** lives on the server — **business logic** and rules, the **database** of shared
data, **authentication** (who the user is), and **secrets** like [API keys](/learn/building-ai/your-first-api-call/)
and database passwords. The front end and back end talk over
[HTTP](/learn/networking/http/), usually through a
[REST API](/learn/web-dev/building-a-rest-api/).
</div>

The [anatomy lesson](/learn/web-dev/anatomy-of-a-web-app/) named the pieces of a web
app; this unit is the server side of it. Before the how — [servers](/learn/web-dev/http-servers-and-routing/),
[APIs](/learn/web-dev/building-a-rest-api/), [databases](/learn/web-dev/backend-and-database/)
— comes the *why*: why a browser full of capable JavaScript still isn't enough, and what
specific jobs a server has to do. The answer comes down to one idea — **the browser is a
place you don't control** — and everything a back end does follows from it.

## The browser can't be trusted

Everything on the front end runs on the **user's** machine, in **their** browser. That
means they can:

- **Read all of it** — every line of your JavaScript, every URL it calls, is right there
  in the developer tools.
- **Change it** — edit the running code, tweak values, disable your checks.
- **Forge requests** — send any HTTP request they like to your server, with any data,
  bypassing your UI entirely.

So any rule you enforce *only* in the browser is a suggestion, not a guarantee. Hide the
"delete" button and someone can still send the delete request by hand; validate a price
in JavaScript and someone can submit a different one. This isn't paranoia — it's the
security model of the web. Anything that has to be **true no matter what the user does**
must be enforced somewhere the user can't reach: **a server you control.** That server is
the back end, and it's why it exists at all.

## Business logic and rules

The first job of the back end is to be the **authority** on how the app behaves — the
**business logic** and rules that must hold regardless of what any client sends. Can this
user delete this record? Is this booking still available? Does this input actually make
sense? The server decides, because only the server's decision can be trusted.

The front end may (and should) check the same things too, but only for a **fast, friendly
experience** — instant feedback, not enforcement. The rule is: **validate on the client
for UX, enforce on the server for real.** A price, a permission, a limit — if it matters,
the server is the one that gets the final say, every request, no exceptions. This is the
same principle the [forms lesson](/learn/web-dev/frontend-frameworks/) hinted at from the
input side: never trust what arrives from the browser without re-checking it on the
server.

## Data that has to be shared and durable

The browser's memory lasts as long as the tab. Anything that must **outlive a session**
or be **shared between users** needs to live somewhere central — a **database** on the
back end. The list of decoded calls, user accounts, saved settings, an audit log: the
server owns that data, reads and writes it, and serves it to whichever client asks (and
is allowed).

```
Browser (front end)                Server (back end)
  fetch("/api/calls")  ───────────▶  read the database
                       ◀───────────  return JSON
```

This is the durable, shared source of truth. Two users see the same calls because both
are reading the one database behind the server, not separate copies in their own
browsers. How the server talks to that database — safely — is its own lesson,
[the back end &amp; its database](/learn/web-dev/backend-and-database/), and the data layer
itself is the [Databases module](/learn/databases/).

## Secrets stay on the server

Some values must **never reach the browser**: your database password, a payment
provider's secret key, the [API keys](/learn/building-ai/your-first-api-call/) for
services you call. If any of those shipped to the client, every user would have them —
and could spend, read, or break things on your account. Because the front end is fully
visible, **there is no such thing as a hidden secret in the browser.**

So secrets live only on the server. When the app needs to, say, call a paid third-party
API, the **browser asks your back end**, and your back end makes the real call using the
secret key the user never sees. The server acts as a trusted middleman precisely so the
secret stays on the trusted side. This is the same "keep the key out of client code" rule
from [your first API call](/learn/building-ai/your-first-api-call/), applied at the whole
app level.

## Knowing who the user is

Because [HTTP is stateless](/learn/networking/http/), the server treats every request as
a stranger unless something proves who's asking. Establishing and checking that identity
— **authentication** — is a core back-end job: verify a login, issue a session or token,
and check it on each request before doing anything sensitive. And once it knows *who* you
are, it decides *what you're allowed to do* — authorization. Both must happen on the
server, for the same reason as everything else here: the browser can claim to be anyone,
so only the server's verification counts. This is a whole unit on its own, starting with
[authentication &amp; sessions](/learn/web-dev/authentication-and-sessions/).

## Talking to the outside world

Finally, the back end is where the app **integrates with other systems** the browser
shouldn't touch directly: sending email, charging a card, calling internal services,
running a scheduled job. These need secrets, need to be trusted, or simply don't belong
in a user's tab. The pattern is consistent — the front end makes a simple request to
*your* server, and your server does the sensitive, trusted work behind it. In
GopherTrunk's own [dashboard](/architecture.html), the back end is what reads the live
decode pipeline and exposes it as an API the browser can safely consume — the front end
never touches the radio directly.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the browser is fully visible and editable by the user, so secrets and trusted rules must live on a server you control." markdown="0">
  <p class="knowledge-check__q">Quick check: why can't a secret API key be kept safe in front-end JavaScript?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Front-end code runs too slowly to protect a key in time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The browser is fully visible to the user, so any key shipped to it can be read</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Browsers refuse to store strings longer than a few characters</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **back end** runs on a **server you control** because the **browser can't be
  trusted** — users can read, change, and forge everything the front end does.
- **Business logic and rules** must be **enforced on the server**; client-side checks are
  for UX only, never for security.
- **Shared, durable data** lives in a **database** on the back end, giving every user one
  source of truth.
- **Secrets** — database passwords, API keys — stay on the server and never ship to the
  browser; the server acts as a trusted middleman.
- **Authentication and authorization** happen on the server, since the browser can claim
  to be anyone.
- The back end also **integrates with other systems** — email, payments, internal
  services — that don't belong in a user's tab.

Next up: [HTTP servers &amp; routing](/learn/web-dev/http-servers-and-routing/).
