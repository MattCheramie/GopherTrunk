---
slug: how-the-browser-works
title: How the browser works
description: The browser is web development's most important program. This lesson traces how it turns a response into a page — fetching HTML, CSS, and JavaScript, parsing HTML into the DOM, applying styles, laying out and painting pixels, and running scripts — plus why script placement affects load speed.
keywords: how a browser works, rendering, parsing HTML, DOM, CSSOM, render tree, layout, paint, JavaScript engine, browser engine, critical rendering path
level: beginner
status: full
prereq:
  - what-is-web-development
faq:
  - q: What does a browser actually do with the HTML it receives?
    a: "It **parses** the HTML into a tree of objects called the **DOM**, parses the CSS into style rules, combines them to decide how every element looks, calculates where each one goes on the page (**layout**), and finally **paints** the pixels. It also downloads and runs any JavaScript, which can change the page after it first appears."
  - q: Why do people put <script> tags at the bottom of the page?
    a: "Because a plain script tag can block the browser from parsing the rest of the HTML while it downloads and runs. Putting scripts at the end — or marking them `defer` — lets the page's content appear first, then runs the JavaScript, so the user isn't staring at a blank screen."
  - q: Is the browser the same as the internet?
    a: "No. The **internet** is the network that carries data between machines; the **browser** is a program on your device that requests web pages over that network and turns the responses into something you can see and use. The browser is one of many programs that use the internet."
---

# How the browser works

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **browser** is the most important program in web development — it turns a
server's response into the page you see. It **fetches** the HTML, CSS, and
JavaScript, **parses** the HTML into a tree called the **DOM**, works out how each
element should look and where it goes (**layout**), and **paints** the result to
the screen. It also runs the **JavaScript**, which can change the page after it
loads. Everything a front-end developer writes is instructions for this one
program. Building on [what web development is](/learn/web-dev/what-is-web-development/),
this is the machine your front-end code actually runs on.
</div>

In the last lesson we said the front end runs "in the browser." That makes the
browser the single most important program in all of web development: it is the
runtime your HTML, CSS, and JavaScript execute inside, the way the JVM runs Java
or an operating system runs an app. Knowing roughly what it does with your code —
and in what order — explains an enormous number of things that otherwise look like
magic or bugs.

## From response to pixels

When a server sends back a response (as we traced in the previous lesson), what
arrives first is a text document: **HTML**. That HTML usually references other
files — stylesheets, scripts, images, fonts — which the browser then requests as
well. Turning all of that into a visible, interactive page is a pipeline, often
called the **rendering** process, and it runs in a fairly fixed order.

The stages are: **parse the HTML**, **parse the CSS**, **combine them** into what
gets shown, **lay out** the page geometry, and **paint** the pixels. Scripts can
run partway through and change things. Let's walk each stage.

## Parsing HTML into the DOM

The browser reads the HTML text top to bottom and builds it into a tree of
objects in memory. Each tag becomes a **node**; nesting becomes parent-and-child
relationships. This live, in-memory tree is the **DOM** — the Document Object
Model — and it is the single most important data structure on the front end.

```html
<body>
  <h1>Live calls</h1>
  <ul>
    <li>Fire dispatch</li>
    <li>Police tac 2</li>
  </ul>
</body>
```

That snippet becomes a tree: a `body` node with an `h1` child and a `ul` child,
and the `ul` with two `li` children. The DOM matters because it's not just how the
page is drawn — it's the thing JavaScript reads and changes to make the page
interactive. We give it a whole lesson later, [the DOM](/learn/web-dev/the-dom/).

## Applying CSS

In parallel, the browser parses the **CSS** — the style rules — into its own
structure. It then walks the DOM and, for every element, works out the final set
of styles that apply: colour, font, size, spacing, position, and so on. An element
with no matching rule still gets sensible defaults.

The result is a version of the tree annotated with how everything should look.
Only elements that will actually be shown are included — something hidden with
`display: none`, for instance, is computed but left out of what gets drawn. CSS
gets its own lesson in [CSS & layout](/learn/web-dev/css-and-layout/); here it's
enough to know the browser marries **structure (DOM)** with **style (CSS)** before
it can draw anything.

## Layout and paint

With structure and style in hand, the browser does two final jobs:

- **Layout** (sometimes called *reflow*): it calculates the exact geometry — how
  big every element is and where it sits on the page. This is where the box model,
  flexbox, and grid do their work. Change the width of the window and layout runs
  again.
- **Paint**: it fills in the actual pixels — text, colours, borders, images,
  shadows — turning the laid-out boxes into what you see.

These steps are not free. When JavaScript changes the page in a way that affects
size or position, the browser has to lay out and paint again, which is why sloppy
updates can make a page feel sluggish. You don't need to optimise this yet, but
it's worth knowing the pixels you see are the *end* of a pipeline, not the start.

## Running JavaScript

The browser also contains a **JavaScript engine** that downloads and runs your
scripts. JavaScript can read and change the DOM, respond to clicks and typing, and
fetch more data from a server — which is how a page updates without a full reload.
We cover what it does in [JavaScript in the browser](/learn/web-dev/javascript-in-the-browser/).

The catch is *timing*. A plain `<script>` tag, by default, **pauses HTML parsing**
while the browser downloads and executes it. Put a big script in the `<head>` and
the user can be staring at a blank page while it loads. That's why scripts
traditionally go at the **bottom** of the page, or are marked to load without
blocking:

```html
<!-- Blocks parsing until it finishes -->
<script src="app.js"></script>

<!-- Downloads alongside parsing, runs after the HTML is ready -->
<script src="app.js" defer></script>
```

The `defer` attribute lets the page's content appear first and runs the script
once the HTML is parsed — the usual choice today. This one detail explains a lot of
"why is my page slow to show up" questions.

## One program, many jobs

Under the hood, a browser is several coordinated engines: a **networking** layer
that fetches resources, a **rendering engine** that runs the parse-layout-paint
pipeline, and a **JavaScript engine** that executes scripts — all wrapped in the
interface with its tabs and address bar. Modern browsers also enforce important
**security** boundaries, keeping one site's code from reaching into another's.
When you write for "the browser," you're really writing instructions these engines
carry out on the user's behalf.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the browser parses HTML into the DOM, a live in-memory tree of the page's structure." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the browser build when it parses your HTML?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A compiled binary that runs on the server</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The DOM — a live in-memory tree of the page's structure</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A database record for each page element</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **browser** is web development's key program — the runtime your HTML, CSS,
  and JavaScript execute inside.
- It **fetches** the page's resources, then runs a **rendering** pipeline: parse
  HTML, apply CSS, lay out geometry, and paint pixels.
- Parsing HTML produces the **DOM**, a live in-memory tree that is also what
  JavaScript reads and changes.
- **Layout** computes size and position; **paint** draws the pixels — and changes
  that affect geometry make both run again.
- The **JavaScript engine** runs your scripts, which can update the page after it
  loads.
- A plain `<script>` **blocks parsing**; placing scripts at the end or using
  `defer` lets content appear first.

Next up: [The client-server web](/learn/web-dev/client-server-web/).
