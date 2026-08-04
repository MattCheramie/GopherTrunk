---
slug: responsive-design
title: Responsive design
description: How one website serves every screen — fluid layouts that flex to the viewport, CSS media queries that adapt at breakpoints, the mobile-first mindset, and the viewport meta tag that makes it all work on phones.
keywords: responsive design, media query, breakpoint, mobile-first, fluid layout, viewport meta tag, responsive web design, CSS breakpoints, flexbox grid responsive, adaptive layout
level: intermediate
status: full
prereq:
  - css-and-layout
faq:
  - q: What is responsive design?
    a: "Responsive design is building one website that adapts its layout to fit any screen — phone, tablet, laptop, or desktop — rather than shipping separate sites. It combines fluid, flexible layouts with CSS media queries that change the design at certain widths, so the page stays usable and readable everywhere."
  - q: What is a media query?
    a: "A media query is a CSS rule that applies styles only when a condition about the device is true — most often the viewport width. You write a block that says 'when the screen is at least this wide, use these styles,' letting one stylesheet describe several layouts for different screen sizes."
  - q: Why start mobile-first?
    a: "Because it forces you to prioritise the essential content and layout for the smallest, most constrained screen first, then progressively add complexity for larger ones with min-width media queries. It tends to produce simpler, faster pages than designing for desktop and trying to cram it down to a phone."
---

# Responsive design

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Responsive design** makes **one site work on every screen** — phone to desktop —
instead of building separate ones. It rests on **fluid layouts** that flex to the
viewport (relative units, [flexbox and grid](/learn/web-dev/css-and-layout/)) plus **CSS
media queries** that change the design at **breakpoints**. The modern default is
**mobile-first**: style the small screen first, then add for larger ones with
`min-width` queries. None of it works on phones without the **viewport meta tag** in your
HTML, which is the one line people most often forget.
</div>

Your users show up on a five-inch phone, a tablet, a laptop, and a wide desktop monitor,
and they expect the same site to be usable on all of them. **Responsive design** is how
you deliver that from a single codebase. It builds directly on
[CSS &amp; layout](/learn/web-dev/css-and-layout/) — the box model, flexbox, and grid — and
adds the tools that let one design adapt to whatever screen it lands on. This lesson is
those tools and the mindset that ties them together.

## The problem: one site, many screens

There is no single screen size to design for. Widths range from roughly 320 pixels on a
small phone to 2560 and beyond on a desktop, and orientation, pixel density, and window
size all vary too. A layout that looks great at 1440px — three columns, a wide sidebar —
is unusable crammed onto a phone, where those same columns become slivers. The old
answer was a separate "m-dot" mobile site; the modern answer is **one responsive site**
that reflows to fit. That's cheaper to build, cheaper to maintain, and better for search,
since there's a single URL for every device.

## Fluid layouts: flex, don't fix

The foundation of responsive design is refusing to hard-code sizes in pixels. Use
**relative units** and flexible layout systems so the page **fluidly** fills whatever
width it's given:

- **Relative units** — percentages, `rem`/`em`, `vw`/`vh`, `fr` in grid — size things
  relative to the viewport or the font, not to a fixed pixel count.
- **Flexbox and grid** — from the [layout lesson](/learn/web-dev/css-and-layout/) — wrap
  and redistribute space as the container changes width.
- **`max-width` on images and media** so they shrink to fit and never overflow.

```css
/* Fluid by default: the grid fits as many 250px columns as will fit, then wraps. */
.call-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
}
img { max-width: 100%; height: auto; }
```

A genuinely fluid layout already handles a lot of screen sizes before you write a single
media query. Media queries then handle the points where fluidity alone isn't enough —
where the design needs to *change*, not just stretch.

## Media queries: adapt at breakpoints

A **media query** applies CSS only when a condition about the device holds — almost always
the **viewport width**. The widths where you change the layout are called **breakpoints**.

```css
/* Base styles: single column, tuned for a phone. */
.layout { display: block; }

/* At 768px and up (tablet), switch to a sidebar + content layout. */
@media (min-width: 768px) {
  .layout { display: grid; grid-template-columns: 250px 1fr; }
}

/* At 1200px and up (desktop), widen and add a third column. */
@media (min-width: 1200px) {
  .layout { grid-template-columns: 250px 1fr 300px; }
}
```

One stylesheet now describes three layouts, each taking over at its breakpoint. Choose
breakpoints where **the content needs them**, not by chasing specific device models —
the moment a line of text gets too long or a row gets too cramped is a breakpoint, no
matter what device hits it first.

## Mobile-first

Notice the direction of those queries: the base styles target the **small** screen, and
`min-width` queries **add** complexity as the screen grows. That's **mobile-first**, the
modern default, and the direction matters. Designing for the phone first forces you to
decide what's essential — the constrained screen has no room for anything else — and then
you progressively enhance for the space larger screens offer.

The alternative, desktop-first (`max-width` queries stripping things away for phones),
tends to produce heavier pages that feel like a big design apologetically shrunk down.
Mobile-first also lines up with reality: for most sites, phones are the majority of
traffic, so the smallest screen deserves to be the one you get right first. It pairs
naturally with the performance mindset — smaller screens often mean slower networks, so a
lean mobile baseline helps the users who need it most.

## The viewport meta tag

Here's the line that trips people up: without it, none of the above works on phones. By
default a mobile browser pretends to be a ~980px-wide desktop and shrinks the whole page
to fit, so your media queries never see the real width. This one tag in your HTML
`<head>` tells the browser to use the device's actual width:

```html
<meta name="viewport" content="width=device-width, initial-scale=1" />
```

`width=device-width` makes the layout viewport match the physical screen, and
`initial-scale=1` starts at normal zoom. With it in place, a phone reports ~390px and
your mobile-first styles apply as intended; leave it out and a beautifully responsive
stylesheet still renders as a tiny, zoomed-out desktop page. It's the first thing to
check when "my media queries aren't working on mobile." Responsive design then feeds
straight into the [Core Web Vitals](/learn/web-dev/responsive-design/) and performance
work that a page needs to feel fast on real devices.

<div class="knowledge-check" data-quiz data-correct-msg="Right — mobile-first means styling the small screen as the base and adding complexity for larger screens with min-width media queries." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a mobile-first approach mean in practice?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Building a separate m-dot site just for phones</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Styling the small screen as the base, then adding for larger screens with min-width queries</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Designing for desktop first and stripping things away for phones</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Responsive design** serves **one site to every screen** by adapting the layout
  rather than shipping separate mobile and desktop versions.
- **Fluid layouts** — relative units plus [flexbox and grid](/learn/web-dev/css-and-layout/)
  — flex to the viewport and handle many sizes before any media query.
- **Media queries** apply CSS at **breakpoints**, letting one stylesheet describe
  several layouts; choose breakpoints where the content needs them.
- **Mobile-first** styles the small screen as the base and adds complexity upward with
  `min-width` queries — simpler, faster, and aligned with real traffic.
- The **viewport meta tag** (`width=device-width`) is required for any of it to work on
  phones — the most common thing people forget.

Next up: [What a back end does](/learn/web-dev/what-a-backend-does/).
