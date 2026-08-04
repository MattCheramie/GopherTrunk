---
slug: javascript-in-the-browser
title: JavaScript in the browser
description: JavaScript is the language that makes web pages interactive. This lesson covers what JavaScript does in the browser, how it responds to events like clicks and typing, the single-threaded event loop, and the crucial distinction between code that runs in the browser and code that runs on the server.
keywords: JavaScript, browser scripting, events, event listener, client-side JavaScript, DOM scripting, event loop, interactivity, client vs server code, async
level: beginner
status: full
prereq:
  - css-and-layout
faq:
  - q: What does JavaScript do in the browser?
    a: "JavaScript adds **behaviour** to a page. It responds to what the user does — clicks, typing, scrolling — by changing the page, fetching new data from a server, validating input, and running logic. Where HTML is structure and CSS is appearance, JavaScript is the interactivity that makes a page feel like an application rather than a document."
  - q: Is JavaScript in the browser the same as Node.js on the server?
    a: "It's the same *language*, but a different environment. In the **browser**, JavaScript can touch the page (the DOM) and respond to user events, but not the file system. On a **server** with Node.js it can read files and talk to a database, but there's no page to manipulate. Same syntax, different powers, decided by where the code runs."
  - q: How does JavaScript respond to a click?
    a: "You attach an **event listener** — a function the browser calls when a particular event happens on a particular element. When the user clicks the button, the browser runs your function. This event-driven style is the core of how interactive pages work: the code waits for things to happen and reacts."
---

# JavaScript in the browser

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**JavaScript** is the third of the front-end trio — the language that makes a page
**interactive**. It runs inside the browser's JavaScript engine, responds to
**events** like clicks and typing through **event listeners**, and can change the
page, validate input, and fetch fresh data. It's **single-threaded** and
**event-driven**: the code reacts to things as they happen. The most important
distinction to hold is **where code runs** — the same language behaves very
differently in the **browser** than on the **server**. This builds on
[HTML](/learn/web-dev/html-structure/) and [CSS](/learn/web-dev/css-and-layout/) to
complete the trio.
</div>

HTML gave us structure and CSS gave us style, but both are static — the page just
sits there. **JavaScript** is what makes it *do* things: respond to a click, check
a form as you type, pull in new data, update part of the page without a reload.
It's the third and final language of the browser, and it's what turns a document
into an application. This lesson is about what it does and, crucially, *where* it
runs.

## What JavaScript adds

HTML and CSS describe a page; JavaScript **changes it while it's running**. With
JavaScript a page can:

- **React to the user** — run code when someone clicks, types, scrolls, or submits.
- **Change the page** — add, remove, or update elements on the fly (the next
  lesson, [the DOM](/learn/web-dev/the-dom/), is entirely about this).
- **Fetch data** — request fresh information from a server in the background and
  show it without a full reload ([fetching data](/learn/web-dev/fetching-data/)).
- **Validate and compute** — check input, do calculations, manage state in the page.

That's the leap from a *document* to an *app*. A static article needs no
JavaScript; a live dashboard that updates as calls come in is JavaScript from top
to bottom.

## Events and listeners

The browser runs JavaScript in an **event-driven** style. Rather than a script
that runs start to finish and stops, browser JavaScript mostly sets up **listeners
** — functions the browser calls later, when something happens.

```js
const button = document.querySelector("#refresh");

button.addEventListener("click", () => {
  console.log("Refresh clicked!");
  // ...update the page, fetch new data, etc.
});
```

That code finds the refresh button and says "when it's clicked, run this
function." The function doesn't run now; it runs *whenever the user clicks*, however
many times. Almost all interactivity is built this way: attach listeners for the
events you care about — `click`, `input`, `submit`, `keydown` — and put your
response inside. The page becomes a set of reactions waiting to fire.

## Single-threaded and asynchronous

Browser JavaScript runs on a **single thread** — it does one thing at a time.
That raises a question: if it can only do one thing at once, how does a page stay
responsive while waiting seconds for a server to reply?

The answer is that slow work is **asynchronous**. When you fetch data, JavaScript
doesn't freeze waiting for it — it kicks off the request and carries on, and the
browser runs your "when it arrives" code later, once the response is back. This is
managed by the **event loop**, which juggles pending events and callbacks so the
one thread is never blocked waiting.

```js
// Kick off a request; keep running; handle the result when it arrives.
fetch("/api/calls")
  .then((response) => response.json())
  .then((data) => console.log("Got calls:", data));

console.log("This line runs before the data arrives.");
```

You'll meet `fetch` and async data properly in
[fetching data from the front end](/learn/web-dev/fetching-data/). The takeaway
here is the *shape*: start something slow, don't wait, react when it's done. That's
how one thread keeps a page smooth.

## Where the code runs

This is the single most important idea in the lesson, and a frequent source of
confusion: **the same JavaScript language runs in two very different places.**

- In the **browser** (client-side), JavaScript can touch the **page** — read and
  change the DOM, respond to user events — but it *cannot* read the user's files or
  reach a database directly. It runs on the user's device, so it's untrusted and
  sandboxed.
- On a **server** (with a runtime like Node.js), JavaScript has *no page* to
  manipulate, but it *can* read files, talk to a database, and hold secrets. It
  runs on the operator's machine.

Same syntax, different powers — decided entirely by **where it runs**. This maps
straight onto the client-server split from earlier: browser JavaScript is the
front end, server JavaScript is a back end. Confusing the two leads to real bugs
and real security holes — for instance, trusting the browser to enforce a rule it
can't, since the user controls their own device.

## Client-side means untrusted

That last point deserves emphasis. Because browser JavaScript runs on the user's
machine, the user can read it, change it, or skip it entirely. So client-side code
is for **convenience and experience** — instant feedback, a smoother interface —
never the final word on anything that matters.

The classic example is form validation. Checking input in the browser gives a fast,
friendly "that email looks wrong" *before* the user submits — but the **server must
check again**, because a determined user can bypass the browser's check completely.
Anything security-relevant — permissions, prices, whether an action is allowed —
belongs on the server. We return to this in
[forms & user input](/learn/web-dev/forms-and-user-input/) and, from the attacker's
side, in [web application attacks](/learn/cybersecurity/web-application-attacks/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — browser JavaScript runs on the user's device and can be changed or bypassed, so the server must re-check anything that matters." markdown="0">
  <p class="knowledge-check__q">Quick check: why can't you rely on browser JavaScript alone to enforce a security rule?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Browser JavaScript is too slow to run security checks</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It runs on the user's device, so the user can change or bypass it — the server must re-check</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">JavaScript can't do comparisons, only display text</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **JavaScript** is the browser's language of **behaviour** — it makes a page
  interactive, completing the trio with HTML and CSS.
- It's **event-driven**: you attach **event listeners** (for `click`, `input`,
  `submit`, …) and the browser runs your function when the event fires.
- It's **single-threaded** but handles slow work **asynchronously** via the **event
  loop**, so the page stays responsive.
- The same language runs in **two places**: in the **browser** it touches the page
  but is sandboxed; on a **server** it touches files and databases but has no page.
- **Where code runs** determines its powers — browser JavaScript is the front end,
  server JavaScript is a back end.
- Client-side code is **untrusted**: use it for experience, but re-check anything
  that matters on the **server**.

Next up: [The DOM — manipulating the page](/learn/web-dev/the-dom/).
