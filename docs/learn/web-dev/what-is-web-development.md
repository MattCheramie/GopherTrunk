---
slug: what-is-web-development
title: What web development is
description: A plain-language map of web development — the front end you see in the browser, the back end that runs on a server, and full-stack work that spans both — plus what actually happens in the seconds between typing a URL and seeing a page.
keywords: web development, front end, back end, full stack, browser, server, HTML CSS JavaScript, what happens when you visit a website, web app, client and server
level: beginner
status: full
faq:
  - q: What is the difference between front end and back end?
    a: "The **front end** is everything that runs in the browser and that a user sees and touches — the HTML, CSS, and JavaScript that make up the page. The **back end** is the code that runs on a server the user never sees: it stores data, enforces rules, and answers requests. A **full-stack** developer works across both."
  - q: Do I need to know how to code to understand web development?
    a: "No. This lesson and the next few build the mental model with no code at all. You'll meet HTML, CSS, and JavaScript later in the module, but the ideas — client and server, request and response, front end and back end — come first and stand on their own."
  - q: Is web development the same as building a website?
    a: "A simple website is one point on a wide spectrum. Web development ranges from a single static page served as a file, up to a full **web app** with logins, live data, and a database behind it. The tools and ideas scale across that whole range, which is why one module can cover both."
---

# What web development is

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Web development is building the software that runs on the web. It splits into a
**front end** — the part that runs in your **browser** and that you see and click
— and a **back end** — the part that runs on a **server** you never see and that
stores data and enforces rules. Someone who works across both is **full-stack**.
Underneath every page is one simple rhythm: the browser sends a **request** and a
server sends back a **response**. This lesson draws the map; the rest of the
module fills it in.
</div>

The web is where most software meets its users. When you check a bank balance,
book a flight, or watch a scanner dashboard update in real time, you are using a
web application — and someone built it. This module is a working developer's tour
of how that gets done. It is not a tutorial for one framework; it is the set of
durable ideas that outlast any framework, starting here with the plainest
question of all: what *is* web development?

## The front end and the back end

Every web app has two halves, and almost everything in web development hangs off
the difference between them.

The **front end** is the part that lives in your **browser**. It is the page you
look at — the text, images, buttons, forms, and animations — and the code that
makes them respond when you click or type. Front-end work is built from three
languages you'll meet soon: **HTML** for structure, **CSS** for appearance, and
**JavaScript** for behaviour. Because it runs on the user's own device, the front
end is sometimes called the **client** side.

The **back end** is the part the user never sees. It runs on a **server** — a
computer, usually in a data centre, that waits for requests and answers them. The
back end holds the data (your account, your saved items, everyone else's too),
enforces the rules (who is allowed to do what), and does the work that can't
safely happen on a stranger's device. It's written in languages like Go, Python,
JavaScript, Ruby, or Java, and it talks to a **database** behind it.

A developer who works on the browser side is a **front-end developer**; one who
works on the server side is a **back-end developer**; and one comfortable across
both is **full-stack**. These are roles and emphases, not walls — the ideas in
this module are worth knowing wherever you sit.

## Client and server

The reason for the split is a single, load-bearing idea: the web is a
conversation between a **client** and a **server**.

- The **client** is the program making the request — almost always a browser, but
  it could be a mobile app or a script.
- The **server** is the program that receives the request and sends back a
  response.

You ask; it answers. That request-and-response rhythm is the heartbeat of the
whole web, and we give it a lesson of its own in
[the client-server web](/learn/web-dev/client-server-web/). For now, hold onto the
shape: the front end is the client, the back end is the server, and everything
that happens between them is one side asking and the other answering.

## What happens when you visit a page

Put the pieces together by following one ordinary action — typing an address and
hitting Enter. In the second or two before the page appears, a surprising amount
happens:

1. The browser figures out **which server** the address belongs to and opens a
   connection to it.
2. It sends a **request** for the page.
3. The server runs its **back-end** code — perhaps looking things up in a database
   — and sends back a **response**: the HTML for the page, plus its CSS and
   JavaScript.
4. The browser **renders** that HTML and CSS into the visual page, and runs the
   JavaScript to make it interactive.
5. As you click around, the front end may send **more requests** in the
   background for fresh data, updating the page without a full reload.

Every step there is a lesson later in this module. [How the browser
works](/learn/web-dev/how-the-browser-works/) unpacks step 4; the plumbing of
steps 1–3 is the domain of the networking module, especially
[how a web request works](/learn/networking/how-a-web-request-works/) and
[HTTP](/learn/networking/http/), the protocol the request and response are
written in.

## A spectrum, not a single thing

"A website" and "a web app" sit at different points on one spectrum. At the
simple end, a page can be a plain **static** file the server hands over
unchanged — this very learning site is built that way. At the rich end, a page is
assembled **dynamically** for each visitor, with logins, live data, and a database
doing real work behind it. Most of what you use daily lives somewhere in between.

The good news is that the same core ideas carry across the whole range. A static
page and a sprawling web app both come down to a client requesting and a server
responding; they differ in how much happens on each side, not in the shape of the
conversation. We map that spectrum directly in
[static vs. dynamic sites](/learn/web-dev/static-vs-dynamic/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the back end runs on a server the user never sees, holding data and rules; the front end is the browser side." markdown="0">
  <p class="knowledge-check__q">Quick check: which part of a web app runs on a server the user never directly sees?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The front end — the HTML and CSS in the browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The back end — the code that stores data and enforces rules</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Neither — a web app runs entirely in the browser</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Web development** is building the software that runs on the web, from a single
  page to a full application.
- The **front end** runs in the **browser** (the client) and is what the user sees
  and interacts with; the **back end** runs on a **server** and holds the data and
  rules.
- A **full-stack** developer works across both; the roles are emphases, not walls.
- The web's underlying rhythm is **request and response** — the client asks, the
  server answers.
- Visiting a page sets off a chain: resolve the server, send a request, run
  back-end code, return a response, and **render** it in the browser.
- Sites range from **static** files to **dynamic** apps, but the same client-server
  ideas carry across the whole spectrum.

Next up: [How the browser works](/learn/web-dev/how-the-browser-works/).
