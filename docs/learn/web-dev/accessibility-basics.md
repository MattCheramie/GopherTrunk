---
slug: accessibility-basics
title: Accessibility basics
description: Building for everyone. This lesson covers the accessibility fundamentals every web developer should know — semantic HTML, keyboard navigation, colour contrast, text alternatives, labelled forms, and ARIA — and why they are also just good engineering that helps every user.
keywords: accessibility, a11y, semantic HTML, keyboard navigation, ARIA, alt text, color contrast, screen reader, WCAG, accessible forms
level: intermediate
status: full
prereq:
  - forms-and-user-input
faq:
  - q: What is web accessibility?
    a: "**Accessibility** (often shortened to **a11y**) means building web pages that everyone can use, including people with visual, motor, hearing, or cognitive disabilities — for example, someone using a screen reader or navigating with only a keyboard. It overlaps heavily with good engineering: most accessible practices also make a site clearer and more robust for every user."
  - q: Do I need to learn ARIA to make an accessible site?
    a: "Mostly no. The single biggest win is **semantic HTML** — using the right elements, which are accessible by default. **ARIA** is a set of attributes for filling gaps semantic HTML can't cover, like custom widgets. The guiding rule is that no ARIA is better than bad ARIA, so reach for a native element first and use ARIA only when nothing native fits."
  - q: Is accessibility only for people with disabilities?
    a: "It's driven by that need, but the benefits are universal. Captions help in a noisy room, good contrast helps in bright sun, keyboard support helps power users, and clear structure helps search engines. Accessibility is often legally required, too — but even setting that aside, it's simply a mark of well-built software."
---

# Accessibility basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Accessibility** (a11y) means building pages **everyone** can use — including
people relying on screen readers or keyboards. The biggest win is **semantic
HTML**, which is accessible by default, backed by **keyboard** support, sufficient
**colour contrast**, **text alternatives** for images, and properly **labelled
forms**. **ARIA** fills the gaps semantic HTML can't, but native elements come
first. Almost every accessible practice is also just **good engineering** that
helps all users — and it's often legally required. This closes the front-end trio
unit, drawing on [HTML](/learn/web-dev/html-structure/) and
[forms](/learn/web-dev/forms-and-user-input/).
</div>

We've built structure, style, behaviour, and input. This lesson asks the question
that separates a page that works for *you* from one that works for *everyone*: can
someone who can't see the screen, can't use a mouse, or struggles with low
contrast still use what you built? **Accessibility** is the practice of making sure
the answer is yes — and, satisfyingly, most of it is things you should be doing
anyway. It's the natural close to the front-end trio, because it ties HTML, CSS,
and JavaScript back to the humans on the other end.

## Why accessibility matters

A large share of people use the web with some assistive technology or constraint:
screen readers that speak the page aloud, keyboard-only navigation, screen
magnifiers, captions, reduced motion. If a site assumes a sighted mouse user, it
quietly excludes all of them.

Two things make this more than a nice-to-have. First, it's frequently a **legal
requirement** — many jurisdictions mandate accessible public services and
products. Second, and more usefully day to day, accessible practices are **better
engineering for everyone**: captions help in a loud room, contrast helps in
sunlight, keyboard support helps power users, and clean structure helps search
engines and future maintainers. Accessibility and quality point the same
direction.

## Semantic HTML is most of the battle

The single highest-leverage thing you can do costs nothing extra: use **semantic
HTML**. This is the payoff of the [HTML lesson](/learn/web-dev/html-structure/) —
the right elements are **accessible by default**.

- A real `<button>` is focusable, works with Enter and Space, and is announced as a
  button. A `<div>` dressed up as one is none of those unless you rebuild it all by
  hand.
- Headings `<h1>`–`<h6>` give a screen-reader user an outline to navigate by, the
  way a sighted user skims.
- Landmarks like `<nav>`, `<main>`, and `<footer>` let users jump straight to a
  region.
- A `<label>` tied to an input tells the user what the field is for.

```html
<!-- Accessible: native element, keyboard and screen-reader ready -->
<button type="button">Refresh</button>

<!-- Not accessible: no focus, no keyboard, no announcement -->
<div class="button" onclick="refresh()">Refresh</div>
```

Get the elements right and you've done most of the work before touching a single
accessibility-specific feature.

## Keyboard navigation

Many people don't use a mouse — because of motor disabilities, because they rely on
a screen reader, or just because the keyboard is faster. Everything interactive
must work with the **keyboard** alone.

The key ideas: users move between interactive elements with **Tab**, activate them
with **Enter** or **Space**, and there must always be a visible **focus indicator**
showing where they are. Native elements (`<button>`, `<a>`, `<input>`) get all of
this for free — another reason to use them. The most common accessibility failure
is a custom control built from `<div>`s that a keyboard simply cannot reach. Never
remove the focus outline just because it's not to your taste; if anything, make it
clearer.

## Text alternatives and contrast

Two visual essentials cover a lot of ground:

- **Text alternatives.** Every meaningful image needs **`alt`** text describing what
  it conveys, so a screen reader can speak it and it still communicates if the image
  fails to load. Purely decorative images get an *empty* `alt=""` so they're skipped
  rather than announced as noise.

```html
<img src="waterfall.png" alt="A P25 control channel spike at 851.3 MHz">
<img src="divider.png" alt="">
```

- **Colour contrast.** Text must stand out enough from its background to be legible,
  including for people with low vision or colour-blindness. Accessibility guidelines
  (WCAG) give minimum contrast ratios, and it's worth checking with a tool. A
  related rule: **don't rely on colour alone** to carry meaning — pair a red error
  state with an icon or text, so it still reads for someone who can't distinguish the
  colour.

## Accessible forms

Forms are where accessibility and the last lesson meet. An input with no label is a
mystery to a screen reader — it announces "edit text" with no clue what to type.
Every field needs a programmatically associated **`<label>`**:

```html
<label for="tg">Talkgroup</label>
<input id="tg" name="talkgroup" type="text" required>
```

The `for="tg"` binds the label to the input's `id`, so the screen reader speaks
"Talkgroup, edit text" and — a nice bonus for everyone — clicking the label focuses
the field. Beyond labels, group related fields, mark required ones clearly (not by
colour alone), and make error messages specific and tied to the field they concern.
Accessible validation is just good validation, spoken aloud.

## ARIA — the gap-filler

Sometimes semantic HTML can't express a complex, custom widget — a tab set, a
combobox, a live-updating region. That's what **ARIA** (Accessible Rich Internet
Applications) is for: attributes that add roles, states, and properties screen
readers understand.

```html
<div role="alert">Connection lost — retrying…</div>
```

That `role="alert"` tells assistive tech to announce the message immediately, handy
for the live updates a dashboard pushes. But ARIA comes with a firm warning: **no
ARIA is better than bad ARIA.** It's easy to misuse in ways that make things
*worse* than plain HTML. So the order of preference is always: use a **native
semantic element** first; reach for **ARIA only** when there is genuinely no native
element for what you're building. ARIA supplements semantic HTML — it never
replaces it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — using the correct semantic elements is accessible by default and is the highest-leverage step you can take." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the single most effective thing you can do for accessibility?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Add ARIA attributes to every element on the page</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Use semantic HTML — the right elements are accessible by default</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Increase the font size of all text to at least 24px</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Accessibility (a11y)** means building pages everyone can use; it's often legally
  required and is **good engineering** that benefits all users.
- **Semantic HTML** is the biggest win — native elements like `<button>` and real
  headings are **accessible by default**.
- Everything interactive must work by **keyboard**: Tab to move, Enter/Space to
  activate, and a visible **focus indicator**.
- Give images meaningful **`alt`** text (empty for decorative ones), ensure enough
  **colour contrast**, and never rely on **colour alone** for meaning.
- **Label** every form field with `<label for>`, and make errors specific and
  clearly marked.
- **ARIA** fills gaps semantic HTML can't, but **native elements come first** — no
  ARIA beats bad ARIA.

Next up: [Front-end frameworks](/learn/web-dev/frontend-frameworks/).
