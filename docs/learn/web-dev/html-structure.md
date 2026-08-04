---
slug: html-structure
title: HTML — structure & semantics
description: "HTML is the skeleton of every web page. This lesson covers elements and tags, nesting, the shape of a document, and — most importantly — semantic markup: why choosing the tag that describes meaning matters for accessibility, search, and the developers who come after you."
keywords: HTML, elements, tags, semantic HTML, markup, document structure, HTML tags, semantic markup, attributes, headings, accessibility
level: beginner
status: full
prereq:
  - static-vs-dynamic
faq:
  - q: What is HTML?
    a: "**HTML** (HyperText Markup Language) is the language that describes the structure and content of a web page. It marks up text with **tags** — like `<h1>` for a heading or `<p>` for a paragraph — so the browser knows what each piece *is*. HTML is the skeleton; CSS styles it and JavaScript makes it interactive."
  - q: What does semantic HTML mean?
    a: "**Semantic** HTML means choosing tags that describe the *meaning* of content, not just its appearance — `<nav>` for navigation, `<button>` for a button, `<h1>` for the main heading. Semantic markup is understood by screen readers, search engines, and other developers, whereas a generic `<div>` says nothing about what it holds."
  - q: Is HTML a programming language?
    a: "Not in the usual sense — it has no logic, loops, or variables. HTML is a **markup language**: it describes and structures content. The logic on a web page comes from JavaScript. That said, writing good HTML is a genuine skill, because the structure you choose affects meaning, accessibility, and everything layered on top."
---

# HTML — structure & semantics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**HTML** is the skeleton of every web page — the language that marks up content
with **tags** so the browser knows what each piece *is*. An **element** is a tag,
its content, and any **attributes**; elements **nest** to form the tree the browser
turns into the [DOM](/learn/web-dev/the-dom/). The most important habit is
**semantic** markup: choosing the tag that describes **meaning** (`<nav>`,
`<button>`, `<h1>`) rather than a generic `<div>`, because meaning is what screen
readers, search engines, and future developers rely on. This is the first of the
front-end trio, building on
[how the browser works](/learn/web-dev/how-the-browser-works/).
</div>

We've talked about HTML as "the structure" since the first lesson. Now we meet it
directly. HTML is the first of the three browser languages — structure (HTML),
style ([CSS](/learn/web-dev/css-and-layout/)), and behaviour
([JavaScript](/learn/web-dev/javascript-in-the-browser/)) — and it's the
foundation the other two build on. Get the structure right and everything above it
is easier; get it wrong and no amount of styling or scripting fully rescues it.

## Elements and tags

HTML marks up content with **tags**. Most come in pairs — an opening tag and a
closing tag — wrapping some content between them. The opening tag, its content, and
the closing tag together make an **element**:

```html
<p>This is a paragraph.</p>
<h1>The main heading</h1>
<a href="/learn/">A link to the learn hub</a>
```

The `<p>` marks a paragraph, `<h1>` the top-level heading, `<a>` a link. Some
elements are **empty** — they have no content and no closing tag, like `<img>` for
an image or `<br>` for a line break. The browser reads these tags to know what each
piece of content *is*, which is exactly what it needs to build the page.

## Attributes

Tags can carry **attributes** — extra information, written as `name="value"` inside
the opening tag. Attributes configure an element without changing its content:

```html
<a href="/learn/networking/http/">Read the HTTP lesson</a>
<img src="antenna.jpg" alt="A discone antenna on a rooftop">
<button type="button">Refresh</button>
```

Here `href` says where the link goes, `src` where the image lives, and `alt` gives
a text description of the image for anyone who can't see it. A few attributes turn
up everywhere — `id` (a unique name for one element), `class` (a label used for
styling and grouping), and `alt` — and you'll lean on them constantly once CSS and
JavaScript enter the picture.

## The shape of a document

A complete HTML page has a standard skeleton. The whole document sits inside
`<html>`, split into a `<head>` (information *about* the page — its title, its
links to CSS and scripts) and a `<body>` (the content that's actually shown):

```html
<!doctype html>
<html lang="en">
  <head>
    <title>Live calls</title>
    <link rel="stylesheet" href="style.css">
  </head>
  <body>
    <h1>Live calls</h1>
    <p>The latest decoded traffic.</p>
  </body>
</html>
```

Elements **nest** inside one another — the `body` contains an `h1` and a `p`, and
those could contain more elements still. That nesting is what forms the **tree**
the browser parses into the DOM, as we saw in
[how the browser works](/learn/web-dev/how-the-browser-works/). Proper nesting
matters: tags should close in the reverse order they opened, so the tree stays
well-formed.

## Semantic HTML

Here is the idea that separates careful HTML from careless HTML: **semantics**.
Semantic markup means choosing the tag that describes what the content *means*, not
just how you want it to look.

HTML gives you meaningful elements for the common parts of a page:

```html
<header>…the top of the page…</header>
<nav>…the site navigation…</nav>
<main>
  <article>…a self-contained piece of content…</article>
</main>
<footer>…the bottom of the page…</footer>
```

Each of those *says something* about the content it holds. Compare that to wrapping
everything in a generic **`<div>`** — a box with no meaning at all. Both can be
made to look identical with CSS, but only the semantic version tells anyone *what
the content is*. Use `<button>` for a button, `<h1>`–`<h6>` for a heading
hierarchy, `<ul>`/`<ol>` for lists, `<a>` for links — reach for `<div>` and
`<span>` only when no meaningful element fits.

## Why semantics matter

Choosing the right tag isn't pedantry; it pays off in three concrete ways:

- **Accessibility.** Screen readers use semantics to navigate. A real `<button>` is
  focusable and announced as a button; a `<nav>` lets a user jump to navigation.
  Rebuilt out of `<div>`s, none of that works without heavy extra effort. This is
  the heart of [accessibility basics](/learn/web-dev/accessibility-basics/).
- **Search and machines.** Search engines and other tools read your markup to
  understand the page. Clear structure — a real heading hierarchy, articles marked
  as articles — helps them index and present it correctly.
- **The next developer.** Semantic HTML is self-documenting. Someone reading
  `<nav>…</nav>` knows instantly what it is; someone reading the fortieth nested
  `<div>` has to guess. That someone is often future-you.

The rule of thumb: **describe the content, don't just position it.** Styling is
CSS's job — HTML's job is to say what things are. A page built on honest semantics
is more accessible, more findable, and easier to maintain, all at once.

<div class="knowledge-check" data-quiz data-correct-msg="Right — semantic tags like <nav> and <button> describe what content means, which screen readers, search engines, and developers all rely on." markdown="0">
  <p class="knowledge-check__q">Quick check: why prefer a semantic tag like <code>&lt;button&gt;</code> over a styled <code>&lt;div&gt;</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It renders faster than a div in every browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It describes meaning, so screen readers, search engines, and developers understand it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A div can't be styled with CSS but a button can</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **HTML** is the structural skeleton of a page — a **markup** language that
  describes what content *is*, not logic.
- An **element** is an opening tag, content, and a closing tag; **empty** elements
  like `<img>` have no closing tag.
- **Attributes** (`href`, `src`, `alt`, `id`, `class`) configure elements with
  extra information.
- A document nests inside `<html>`, split into `<head>` (about the page) and
  `<body>` (shown content); nesting forms the **tree** the browser turns into the
  DOM.
- **Semantic** markup chooses tags by **meaning** — `<nav>`, `<button>`, `<h1>` —
  rather than reaching for a meaningless `<div>`.
- Semantics pay off in **accessibility**, **search**, and **maintainability**:
  describe the content, and leave the styling to CSS.

Next up: [CSS — styling & layout](/learn/web-dev/css-and-layout/).
