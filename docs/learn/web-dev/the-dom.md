---
slug: the-dom
title: The DOM — manipulating the page
description: The DOM is the live tree the browser builds from your HTML, and the bridge JavaScript uses to change a page after it loads. This lesson explains what the DOM is, how to select and change elements, how to handle events, and why frequent DOM changes have a performance cost.
keywords: DOM, document object model, DOM manipulation, querySelector, event handling, update page without reload, DOM tree, nodes, reflow, JavaScript DOM
level: intermediate
status: full
prereq:
  - javascript-in-the-browser
faq:
  - q: What is the DOM?
    a: "The **DOM** (Document Object Model) is the browser's live, in-memory representation of the page — a tree of objects built from your HTML, where each element is a node. It's what JavaScript reads and changes to update the page. The HTML you write is the starting point; the DOM is the running version the browser and your code both work with."
  - q: How is the DOM different from HTML?
    a: "HTML is the static text you write and the server sends. The **DOM** is the live tree the browser builds from that HTML and then keeps in memory. Once the page is running, JavaScript changes the *DOM*, not the original HTML — which is why the page can look different from its source after scripts run."
  - q: Why can changing the DOM a lot make a page slow?
    a: "Because some DOM changes force the browser to recalculate layout and repaint — recomputing the size and position of elements. Doing that many times in quick succession, especially inside a loop, can make a page feel sluggish. The fix is to batch changes so the browser reflows once instead of many times."
---

# The DOM — manipulating the page

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **DOM** (Document Object Model) is the live, in-memory **tree** the browser
builds from your HTML — each element a **node**. It's the bridge
[JavaScript](/learn/web-dev/javascript-in-the-browser/) uses to change a page
**after it loads**: your code **selects** elements, **reads or changes** them, and
**listens** for events, and the browser updates the display — no reload needed.
Changing the DOM has a **cost**, though: some updates force **layout and paint**
again, so frequent changes can be slow. Understanding the DOM is the key to every
interactive page, and to why frameworks exist.
</div>

The last lesson said JavaScript can "change the page." This lesson is about *how* —
and the answer is the **DOM**. We first met it in
[how the browser works](/learn/web-dev/how-the-browser-works/): when the browser
parses your HTML, it builds a tree of objects in memory. That tree is the DOM, and
it's the thing your JavaScript actually manipulates. Master it and you understand
the machinery under every interactive page — and the problem that
[frameworks](/learn/web-dev/frontend-frameworks/) were invented to solve.

## What the DOM is

When the browser reads your HTML, it doesn't keep the text around — it builds a
**tree of objects**, one **node** per element, mirroring the nesting of your
markup. This is the **DOM**: a live model of the page, sitting in memory, that both
the browser and your JavaScript can work with.

The crucial word is **live**. The DOM is not a one-time snapshot of your HTML; it's
the running state of the page. When JavaScript changes a node, the browser
re-renders to match, and the page updates on screen. So the page you see is really
a view of the *current DOM*, which may have drifted far from the original HTML the
server sent. That's exactly what makes dynamic, no-reload interfaces possible.

## Selecting elements

To change something, you first have to **find** it. The browser gives JavaScript a
global `document` object — the root of the DOM — with methods to select nodes,
most commonly `querySelector` (first match) and `querySelectorAll` (all matches),
using the same selector syntax as CSS:

```js
// One element by id
const heading = document.querySelector("#title");

// A list of elements by class
const items = document.querySelectorAll(".call");
```

Because the selectors are the same ones you learned for
[CSS](/learn/web-dev/css-and-layout/), the `class` and `id` attributes you add to
your HTML do double duty: they're both styling hooks and the handles JavaScript
grabs elements by. Selecting is always step one — get a reference to the node, then
do something with it.

## Changing the page

With a node in hand, you can read and change it: its text, its attributes, its
styles, and the elements around it.

```js
const heading = document.querySelector("#title");

heading.textContent = "Live calls (updated)";   // change its text
heading.classList.add("active");                 // toggle a CSS class

// Build and insert a brand-new element
const li = document.createElement("li");
li.textContent = "New call on Fire dispatch";
document.querySelector("#calls").append(li);
```

Setting `textContent` changes what the element says; `classList.add` flips a CSS
class on (often the cleanest way to restyle — let CSS define the look, let JS just
toggle the class); and `createElement` plus `append` adds a whole new element to
the page. This — reading and writing nodes — is the entire vocabulary of DOM
manipulation, and it's how a page rebuilds itself as data changes.

A safety note that connects to security: when you insert content that came from a
user, set it as **text** (`textContent`), not as raw HTML. Injecting untrusted
HTML into the DOM is the classic route to a cross-site scripting (XSS) attack,
covered in [web application attacks](/learn/cybersecurity/web-application-attacks/)
and [web security essentials](/learn/web-dev/web-security-essentials/).

## Handling events on the DOM

The DOM is also where **events** happen. As we saw in the last lesson, you attach a
listener to a node and the browser runs your function when the event fires on it —
and typically that function then *changes the DOM* in response:

```js
document.querySelector("#refresh").addEventListener("click", () => {
  const status = document.querySelector("#status");
  status.textContent = "Refreshing…";
});
```

Select, listen, change — that loop is the beating heart of front-end code. A click
comes in, your handler runs, it updates the DOM, and the user sees the result. Most
interactive features are just this pattern repeated: an event, a response, a DOM
update.

## The cost of DOM changes

DOM manipulation is powerful, but it isn't free, and this is where "intermediate"
starts to matter. Some changes — anything affecting an element's size or position —
force the browser to redo **layout** and **paint**, the pipeline stages from
[how the browser works](/learn/web-dev/how-the-browser-works/). One update is
nothing; a thousand in a tight loop, each triggering a fresh layout, can make a
page stutter.

```js
const list = document.querySelector("#calls");

// Costly: touches the live DOM on every iteration
for (const call of calls) {
  const li = document.createElement("li");
  li.textContent = call.name;
  list.append(li);           // may reflow each time
}
```

The general fix is to **batch**: build up your changes and apply them in as few
DOM operations as possible, so the browser reflows once instead of a thousand
times. You don't need micro-optimisation now, just the instinct that *touching the
DOM has a cost*.

## Why frameworks exist

Here's the payoff of this lesson for the rest of the module. Keeping the DOM in
sync with your data **by hand** — selecting the right nodes, updating exactly the
ones that changed, and doing it efficiently — gets fiddly and error-prone fast as
an interface grows. Miss a spot and the screen shows stale data; over-update and
the page is slow.

That precise pain is why **front-end frameworks** like React exist: you describe
what the page *should* look like for the current data, and the framework works out
the minimal DOM changes to get there. It's still manipulating this same DOM
underneath — the framework is a smarter layer over the exact machinery you've just
learned. We pick that thread up in
[front-end frameworks](/learn/web-dev/frontend-frameworks/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the DOM is the browser's live in-memory tree of the page, and JavaScript changes the DOM to update what's shown." markdown="0">
  <p class="knowledge-check__q">Quick check: when JavaScript updates the page, what is it actually changing?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The original HTML file on the server</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The DOM — the browser's live in-memory tree of the page</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The CSS stylesheet, which then redraws the HTML</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **DOM** is the browser's **live, in-memory tree** built from your HTML — one
  **node** per element — and it's what JavaScript reads and changes.
- It's **live**: change a node and the browser re-renders, so the page can drift
  from the original HTML source.
- You **select** nodes with `querySelector`/`querySelectorAll` (CSS selectors),
  then **read or change** their text, classes, attributes, and children.
- Insert user content as **text**, not raw HTML, to avoid **XSS**.
- Events happen on the DOM: **select, listen, change** is the core loop of
  interactive code.
- DOM changes can trigger **layout and paint**, so **batch** updates to keep pages
  fast — and this very difficulty is why **frameworks** exist.

Next up: [Forms & user input](/learn/web-dev/forms-and-user-input/).
