---
slug: anatomy-of-a-web-app
title: Anatomy of a modern web app
description: The map of a web app and how its pieces fit — the front end in the browser, the back end on a server, the database that stores data, and the API that connects them — so the rest of the module has a place to hang each new idea.
keywords: web app architecture, front end back end database, API, three-tier, web application components, client server database, how a web app is structured, tiers of a web app
level: beginner
status: full
prereq:
  - client-server-web
faq:
  - q: What are the main parts of a web app?
    a: "Four, most of the time: a **front end** running in the browser, a **back end** running on a server, a **database** that stores data durably, and an **API** — the agreed set of requests — that lets the front end talk to the back end. Simple sites drop some of these; big ones split each into many pieces, but the roles stay the same."
  - q: What is an API in a web app?
    a: "An **API** (application programming interface) is the contract between the front end and the back end: a defined set of requests the front end can make and responses the back end promises to give. It's how the browser asks the server for data or tells it to do something, without either side knowing the other's internal details."
  - q: Does every web app need a database?
    a: "No. A purely static site has no database at all — it's just files. But any app that remembers things between visits or between users — accounts, posts, saved settings, decoded calls — needs somewhere durable to keep that data, and that's the database's job."
---

# Anatomy of a modern web app

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A typical web app has four parts: a **front end** in the browser, a **back end**
on a server, a **database** that stores data, and an **API** connecting the front
end to the back end. The front end handles what the user sees; the back end holds
the logic and secrets; the database remembers things between visits; the API is
the agreed set of requests between them. This is the map the rest of the module
fills in, and it slots straight onto the
[client-server](/learn/web-dev/client-server-web/) rhythm you already know.
</div>

You now have the rhythm — client asks, server answers. This lesson zooms out to
the whole machine and names its parts, so that every later lesson has an obvious
place to attach. Think of it as the floor plan: not much detail yet, but you'll
know which room you're standing in when we go deep on auth, or databases, or
real-time updates.

## The four parts

Most web apps, however large, are built from four kinds of thing:

- **Front end** — the code in the browser: HTML, CSS, and JavaScript. It draws the
  interface and handles interaction. It is the **client**.
- **Back end** — the code on a server. It runs the business logic, checks
  permissions, keeps secrets, and answers requests. It is the **server**.
- **Database** — where data lives durably: accounts, records, settings, decoded
  calls. It survives restarts and is shared across all users.
- **API** — the defined set of requests the front end makes to the back end, and
  the responses it gets back. It's the **contract** between the two halves.

The classic name for this layering is a **three-tier architecture**:
presentation (front end), application logic (back end), and data (database), with
the API stitching the first two together. You don't need the jargon, but you'll
see it, and it names a real and durable pattern.

## How a request flows through them

Follow one action — a user opening a page that lists live calls — and watch it
pass through all four parts:

1. The **front end** in the browser needs the list of calls, so it makes an
   **API** request to the back end — something like *GET /api/calls*.
2. The **back end** receives it, checks the user is allowed to see it, and asks the
   **database** for the current calls.
3. The **database** returns the rows; the back end shapes them into a tidy
   response (usually **JSON**) and sends it back across the API.
4. The **front end** receives the data and updates the page to show the calls — no
   full reload needed.

Every arrow there is a client-server exchange, and every box is one of the four
parts. This is the skeleton under an enormous range of apps, from a to-do list to
GopherTrunk's own scanner dashboard, which we dissect at the end of the module in
[the GopherTrunk web dashboard](/learn/web-dev/gophertrunk-web-dashboard/).

## The API is the seam

Of the four parts, the **API** is the one worth dwelling on, because it's the seam
that lets the two halves be built and changed independently.

The front end doesn't know *how* the back end stores calls or what language it's
written in; it only knows the **agreed requests** and the shape of the data that
comes back. The back end doesn't know whether the client is a browser, a phone
app, or a script; it just honours the contract. As long as both sides keep to the
API, either can be rebuilt without breaking the other.

```http
GET /api/calls HTTP/1.1
Host: dashboard.example
```

```json
[
  { "id": 41, "talkgroup": "Fire dispatch", "started": "12:04:31" },
  { "id": 42, "talkgroup": "Police tac 2",  "started": "12:05:02" }
]
```

That request-and-JSON pair *is* the API in miniature. When an API organises its
requests around resources and standard HTTP methods in a consistent style, you get
**REST**, the dominant convention on the web — covered here in
[building a REST API](/learn/web-dev/building-a-rest-api/) and, from the network
angle, in [web APIs & REST](/learn/networking/web-apis-and-rest/).

## Where the data lives

The **database** deserves its own callout because it answers a question the other
parts can't: how does the app remember anything?

The front end forgets everything when you close the tab. The back end, by itself,
forgets everything when it restarts. Durable memory — the state that must outlive a
single request, a single user, and a single server — lives in the **database**. It
holds the authoritative copy of everything the app knows, and the back end is the
only part allowed to touch it directly. We come back to this boundary in
[the back end & its database](/learn/web-dev/backend-and-database/), and the
[databases module](/learn/databases/) covers the data layer in its own right.

## Simple apps and big apps

Real apps stretch this map in both directions. A plain static site is *only* a
front end — files served as-is, no back end or database at all, which is exactly
how this learning site works. At the other extreme, a large product splits each
box into many: several front ends, dozens of back-end services, multiple
databases, and layers of caching and delivery in between.

But the four roles stay constant. Whether you're looking at a weekend project or a
platform serving millions, you can always ask: what's the front end, what's the
back end, where's the data, and what's the API between them? Answer those and
you've understood the shape of the system.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the API is the agreed contract of requests and responses between the front end and back end." markdown="0">
  <p class="knowledge-check__q">Quick check: what connects the front end to the back end?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The database, which the browser queries directly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The API — the agreed set of requests and responses between them</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — they run as one combined program</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Most web apps have four parts: a **front end** (browser), a **back end**
  (server), a **database** (durable data), and an **API** (the contract between
  front and back).
- This is the classic **three-tier** shape: presentation, logic, and data.
- A request flows front end → API → back end → database and back again, each arrow
  a client-server exchange.
- The **API** is the seam that lets the two halves change independently; organised
  in a consistent style it becomes **REST**.
- The **database** provides the durable memory the front end and back end lack on
  their own, and only the back end touches it directly.
- Simple apps drop parts (a static site is front end only); big apps multiply them,
  but the four roles stay the same.

Next up: [Static vs. dynamic sites](/learn/web-dev/static-vs-dynamic/).
