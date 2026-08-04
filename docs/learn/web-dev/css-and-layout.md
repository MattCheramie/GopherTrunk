---
slug: css-and-layout
title: CSS — styling & layout
description: CSS turns plain HTML into a designed page. This lesson covers how CSS attaches to markup with selectors, the properties that control appearance, the box model every element follows, and the two modern layout systems — flexbox and grid — that arrange elements on the page.
keywords: CSS, selectors, box model, flexbox, grid, styling, layout, CSS properties, padding margin border, responsive layout, cascade
level: beginner
status: full
prereq:
  - html-structure
faq:
  - q: What does CSS do?
    a: "**CSS** (Cascading Style Sheets) controls how a page *looks* — colours, fonts, spacing, sizes, and the arrangement of elements. HTML says what content is; CSS says how it should be presented. You write rules that select elements and set properties on them, and the browser applies those rules when it renders the page."
  - q: What is the box model?
    a: "Every element the browser lays out is a rectangular **box** with four layers from the inside out: the **content**, the **padding** around it, the **border** around that, and the **margin** separating it from other boxes. Almost all spacing and sizing in CSS comes down to adjusting these four layers."
  - q: Should I use flexbox or grid?
    a: "Both are modern layout tools and you'll use them together. **Flexbox** arranges items in a single direction — a row or a column — and is ideal for things like navigation bars and toolbars. **Grid** works in two dimensions at once — rows and columns — and suits page-level layouts. A common pattern is grid for the overall page, flexbox within its pieces."
---

# CSS — styling & layout

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**CSS** is how a page goes from plain text to designed. You write **rules** that
**select** elements and set **properties** — colour, font, spacing, size — and the
browser applies them. Every element is a **box** with content, **padding**,
**border**, and **margin** (the **box model**), and the two systems that arrange
those boxes are **flexbox** (one direction) and **grid** (two directions).
Together with [HTML](/learn/web-dev/html-structure/) for structure, CSS is the
second of the front-end trio — it controls everything about how the page *looks*.
</div>

HTML gives you a structured but unstyled page — black text on a white background,
stacked top to bottom. **CSS** is what makes it look like a designed product. It's
the second language of the browser, and where HTML answers "what is this content,"
CSS answers "how should it look and where should it go." The two are deliberately
separate: structure in HTML, presentation in CSS, so each can change without
disturbing the other.

## How CSS attaches to HTML

A CSS **rule** has two parts: a **selector** that picks which elements it applies
to, and a block of **declarations** — `property: value` pairs — that say what to do
to them:

```css
h1 {
  color: navy;
  font-size: 2rem;
}
```

That rule selects every `<h1>` and makes it navy and larger. Selectors are how you
target elements:

- A **type** selector like `h1` matches every element of that kind.
- A **class** selector like `.alert` matches elements with `class="alert"`.
- An **id** selector like `#header` matches the one element with `id="header"`.

Classes are the workhorse — you add a `class` to the HTML elements you want to
style as a group, then write one rule for that class. This is the main reason the
`class` attribute from the last lesson matters so much.

## The cascade

The "cascading" in CSS names how the browser resolves conflicts when **several
rules** target the same element. Rather than error, it decides which wins, based
mainly on:

- **Specificity** — a more specific selector (an id) beats a less specific one (a
  type).
- **Order** — when specificity ties, the rule written later wins.
- **Inheritance** — some properties, like `color` and `font`, pass down from a
  parent to its children unless overridden.

You don't need to master the rules today, but knowing they exist explains a
classic beginner moment: "my style isn't applying." Usually another, more specific
rule is winning the cascade. Reach for clear, consistent class names before you
reach for ever-more-specific selectors.

## The box model

Here's the idea that underlies all CSS layout: **every element is a rectangular
box**, and each box has four nested layers, from the inside out:

- **Content** — the text or image itself.
- **Padding** — space *inside* the box, between the content and the border.
- **Border** — a line around the padding.
- **Margin** — space *outside* the box, separating it from its neighbours.

```css
.card {
  padding: 16px;          /* space inside, around the content */
  border: 1px solid gray; /* the edge */
  margin: 24px;           /* space outside, between cards */
}
```

Nearly all spacing and sizing comes down to these four. A frequent stumbling block
is that, by default, `width` sets the *content* width and padding and border add on
top — so a "300px" box can end up wider. The common fix, `box-sizing: border-box`,
makes `width` include padding and border, which is far more intuitive and is set
almost universally today.

## Laying out with flexbox

Stacking boxes top to bottom is the default, but real designs arrange things in
rows, columns, and grids. **Flexbox** is the tool for laying out items along a
**single direction** — a row or a column — and distributing space among them.

```css
.toolbar {
  display: flex;
  gap: 12px;              /* space between items */
  justify-content: space-between;
  align-items: center;
}
```

Set `display: flex` on a container and its children become flex items you can
space, align, and size fluidly. Flexbox shines for navigation bars, toolbars,
button rows, and card strips — anything that's essentially a line of items you want
arranged and aligned. It also handles wrapping and flexible sizing, which is a big
part of adapting to different screens.

## Laying out with grid

Where flexbox thinks in one direction, **CSS grid** works in **two dimensions** at
once — rows *and* columns — which makes it the natural choice for the overall shape
of a page or any genuinely grid-like arrangement:

```css
.layout {
  display: grid;
  grid-template-columns: 200px 1fr;  /* sidebar + flexible main */
  gap: 24px;
}
```

That creates a fixed sidebar next to a main area that takes the remaining space
(`1fr` means "one fraction of what's left"). Grid excels at page layouts, image
galleries, and dashboards. In practice you **combine** the two: grid for the
big-picture structure, flexbox inside its regions. Both systems are also central
to [responsive design](/learn/web-dev/responsive-design/), where the same layout
rearranges itself to fit any screen.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the box model is content, padding, border, and margin, the four layers that control an element's spacing and size." markdown="0">
  <p class="knowledge-check__q">Quick check: which four layers make up the CSS box model, inside to out?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Header, body, footer, and sidebar</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Content, padding, border, and margin</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Selector, property, value, and rule</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **CSS** controls how a page looks — colour, font, spacing, size, and layout —
  keeping presentation separate from HTML's structure.
- A **rule** pairs a **selector** (type, class, or id) with **declarations**
  (`property: value`); **classes** are the everyday workhorse.
- The **cascade** resolves conflicting rules by **specificity**, **order**, and
  **inheritance** — the usual reason a style "won't apply."
- Every element is a **box** with **content, padding, border, and margin**; the
  **box model** governs spacing and sizing.
- **Flexbox** lays items out in **one direction** (rows or columns); **grid** works
  in **two dimensions** for page-level layout.
- Real designs **combine** grid and flexbox, and both underpin responsive layouts.

Next up: [JavaScript in the browser](/learn/web-dev/javascript-in-the-browser/).
