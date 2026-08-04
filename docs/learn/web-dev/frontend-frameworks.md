---
slug: frontend-frameworks
title: Front-end frameworks
description: Why React, Vue, Svelte, and their peers exist — the problem of keeping a complex UI in sync with changing data, why hand-written DOM code stops scaling, and what a framework actually gives you in return for the extra machinery.
keywords: front-end framework, React, Vue, Svelte, declarative UI, keeping UI in sync, virtual DOM, reactivity, component framework, why use a framework, hand-written DOM
level: intermediate
status: full
prereq:
  - the-dom
faq:
  - q: Do I need a framework to build a website?
    a: "No. Plenty of good sites are plain HTML, CSS, and a little JavaScript, and this very site is one of them. Frameworks earn their keep on apps with lots of interactive, data-driven UI that changes as the user works — a dashboard, an editor, a feed. For a mostly-static page, a framework is weight you don't need."
  - q: What is the difference between React, Vue, and Svelte?
    a: "They solve the same core problem — keeping the UI in sync with your data — with different trade-offs. React is the most widely used and has the biggest ecosystem; Vue aims for an approachable middle ground; Svelte pushes the work into a compile step so the browser ships less framework code. The mental model of components and state carries across all of them."
  - q: What does 'declarative' mean for a UI?
    a: "You describe what the page should look like for the current data, and the framework works out the DOM changes needed to get there. That's the opposite of imperative code, where you write each step — create this node, set that text, remove that child — by hand. Declarative UI is why frameworks scale to complex screens."
---

# Front-end frameworks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A front-end framework like **React**, **Vue**, or **Svelte** exists to solve one
stubborn problem: **keeping a complex UI in sync with changing data**. Hand-written
[DOM](/learn/web-dev/the-dom/) code works for a few elements, but as an app grows,
the manual "find the node, change it" updates multiply and drift out of sync. A
framework lets you write **declarative** UI — describe what the screen should show
for the current data — and it figures out the DOM changes for you. The price is a
**build step** and more machinery; the payoff is UI that stays correct as it scales.
</div>

You already know how to change a page by hand: grab an element from the
[DOM](/learn/web-dev/the-dom/), set its text or add a class, and the browser
repaints. That works beautifully for a handful of updates. This lesson is about what
happens when a handful becomes a hundred — when the same piece of data shows up in
five places, updates ten times a second, and every path through the code has to keep
all of it consistent. That is the wall hand-written DOM code hits, and it is exactly
the wall frameworks were built to get over.

## The problem: keeping the UI in sync

Picture a live scanner dashboard. A decoded call arrives, and now you must update:
the call list, the active-call banner, the talkgroup counter, the "last heard" time,
and maybe a little activity light. With plain DOM code, *you* write every one of
those updates, in the right order, for every event that could change them — a new
call, a call ending, the user filtering the list, a reconnect that reloads
everything.

```js
// Imperative: you spell out every DOM change, everywhere data can change.
function onCall(call) {
  document.querySelector("#count").textContent = String(calls.length);
  const li = document.createElement("li");
  li.textContent = call.talkgroup;
  document.querySelector("#call-list").prepend(li);
  document.querySelector("#banner").textContent = "Active: " + call.talkgroup;
  // ...and every other place this call is shown, kept in sync by hand.
}
```

Each line is simple. The trouble is that there are dozens of them, scattered across
dozens of event handlers, and **nothing enforces that they agree**. Miss one spot
and the counter says 12 while the list shows 13. This class of bug — the UI and the
data disagreeing — is the tax you pay for updating the page manually, and it grows
faster than the app does.

## The idea: describe the UI, don't mutate it

A framework flips the model. Instead of writing the *steps* to change the DOM, you
write a function of your data to the UI: "given these calls, here is what the screen
looks like." When the data changes, you don't touch the DOM — you hand the framework
the new data and let it work out the difference.

```jsx
// Declarative: describe the UI for the current data. React figures out the DOM.
function Dashboard({ calls }) {
  return (
    <div>
      <p>Active calls: {calls.length}</p>
      <ul>
        {calls.map((c) => <li key={c.id}>{c.talkgroup}</li>)}
      </ul>
    </div>
  );
}
```

There is no `createElement`, no `prepend`, no counter to update by hand. You state
that the count *is* `calls.length` and the list *is* one item per call. Add a call to
the data and every place it appears updates together, because they are all derived
from the same source. This is the core trade every framework makes: **you give up
direct control of the DOM, and in return the UI can't drift out of sync with the
data.**

## How frameworks pull it off

If you re-describe the whole UI on every change, why isn't it slow to rebuild the
page constantly? Frameworks avoid that with different strategies, but they share a
goal — apply the **smallest real DOM change** that matches your new description:

- **A virtual DOM (React, historically).** The framework builds a lightweight
  in-memory picture of the UI, compares the new picture to the old one, and touches
  only the DOM nodes that actually differ.
- **Fine-grained reactivity (Vue, Solid, Svelte).** The framework tracks exactly
  which bits of data each part of the UI depends on, so when one value changes it
  updates just the nodes bound to it — no diffing a whole tree.
- **A compile step (Svelte).** Much of the work happens ahead of time: your
  components compile to tight, direct DOM instructions, so less framework code ships
  to the browser.

You rarely need to think about which mechanism is running. What matters is the
promise they all keep: describe the UI declaratively, and the framework makes the
DOM match — efficiently.

## What you get, and what it costs

Frameworks bundle more than syncing. Most give you a **component model** (reusable,
self-contained UI pieces — the [next lesson](/learn/web-dev/components-and-state/)),
a way to manage **state**, **client-side routing** so navigation feels instant, and
a large **ecosystem** of libraries. That is real leverage for building an app.

It is not free. A framework adds a **build step** (covered in
[build tools &amp; bundlers](/learn/web-dev/build-tools-and-bundlers/)), ships
JavaScript the browser must download and run, and brings its own concepts to learn.
For a mostly-static page — a blog, a docs site, a marketing page — that machinery is
overhead with little return, which is why so much of the web, including this site, is
happily framework-free (see [static vs. dynamic](/learn/web-dev/static-vs-dynamic/)).
The honest rule: reach for a framework when the UI is **interactive and
data-driven** enough that keeping it in sync by hand has become the hard part.

## They're more alike than different

React, Vue, Svelte, Angular, Solid — the arguments about which is best are loud, but
the **mental model is shared**. All of them are built on components and state, all of
them are declarative, and all of them exist to keep the UI in sync with the data.
Learn that model once and you can read and reason about any of them; the syntax
differs, the idea does not. That is why the next two lessons focus on the ideas —
[components &amp; state](/learn/web-dev/components-and-state/) and
[fetching data](/learn/web-dev/fetching-data/) — rather than one framework's API.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the core job of a front-end framework is keeping the UI in sync with changing data, so you describe the UI declaratively instead of updating the DOM by hand." markdown="0">
  <p class="knowledge-check__q">Quick check: what core problem does a front-end framework primarily solve?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Making the network faster so pages download in less time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Keeping a complex UI in sync with changing data without manual DOM updates</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Replacing HTML and CSS so you no longer need them</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Hand-written [DOM](/learn/web-dev/the-dom/) updates work for small UIs but drift
  out of sync as an app grows and the same data appears in many places.
- A **front-end framework** solves that by letting you write **declarative** UI —
  describe what the screen shows for the current data — instead of mutating the DOM
  step by step.
- When the data changes, the framework computes the **smallest DOM change** needed,
  via a virtual DOM, fine-grained reactivity, or a compile step.
- Frameworks also bundle components, state management, routing, and a large
  ecosystem — real leverage for interactive, data-driven apps.
- The cost is a **build step**, shipped JavaScript, and more to learn, so a
  mostly-static page is often better off without one.
- **React, Vue, and Svelte share the same mental model** — components and state —
  so learning the idea transfers across all of them.

Next up: [Components &amp; state](/learn/web-dev/components-and-state/).
