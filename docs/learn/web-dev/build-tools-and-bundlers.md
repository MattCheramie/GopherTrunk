---
slug: build-tools-and-bundlers
title: Build tools & bundlers
description: What turns your source code into the files a browser downloads — why modern front ends have a build step at all, what a bundler like Vite or webpack does, and what transpiling, minifying, and tree-shaking each contribute.
keywords: build tool, bundler, Vite, webpack, esbuild, transpile, Babel, minify, tree shaking, source map, dev server, hot module replacement, build step
level: intermediate
status: full
prereq:
  - frontend-frameworks
faq:
  - q: Why can't the browser just run my source code directly?
    a: "Sometimes it can — plain HTML, CSS, and JavaScript run as-is. But modern front ends use things the browser doesn't understand directly: JSX, TypeScript, imported CSS, dozens of small module files. A build step converts all of that into the plain HTML, CSS, and JavaScript the browser does understand, bundled and optimised for fast loading."
  - q: What is the difference between a bundler and a transpiler?
    a: "A transpiler converts one flavour of code into another the browser accepts — JSX or TypeScript into plain JavaScript. A bundler combines many source files (and their dependencies) into a few optimised files to download. Modern tools like Vite do both, plus minifying and tree-shaking, in one pipeline."
  - q: Do I always need a build step?
    a: "No. A static page of hand-written HTML, CSS, and JavaScript needs no build at all — this site is largely like that. You need a build step when you use a framework, a language like TypeScript, or a component syntax like JSX, because those must be transformed into what the browser runs."
---

# Build tools & bundlers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Modern front ends have a **build step**: the code you write isn't the code the browser
runs. A **bundler** like **Vite** or **webpack** takes your many source files — plus
things the browser can't run directly, like **JSX** or **TypeScript** — and turns them
into a few optimised **HTML, CSS, and JavaScript** files to download. Along the way it
**transpiles** newer syntax to what browsers accept, **minifies** to shrink the files,
and **tree-shakes** away unused code. In development it runs a fast **dev server** with
instant updates; for production it emits a small, optimised bundle.
</div>

The [framework lessons](/learn/web-dev/frontend-frameworks/) quietly assumed something:
that the JSX and imported modules you write somehow become files a browser can run. That
conversion is the **build step**, and the tool that does it is a **bundler**. This lesson
is what that step is, why modern front ends need one, and what each part of it buys you.
It is also the honest answer to "why is there so much tooling?" — every piece earns its
place, but it helps to see what each does.

## Why a build step exists

The browser runs exactly three things: **HTML, CSS, and JavaScript**. But a modern
front end is written in things it can't run directly:

- **JSX** — the HTML-in-JavaScript syntax React uses is not valid JavaScript.
- **TypeScript** — JavaScript with types; the types must be stripped out first.
- **Many small modules** — you split code across dozens or hundreds of files and
  `import` between them, plus third-party packages from `node_modules`.
- **Imported CSS, images, and other assets** handled through your JavaScript.

A build step converts all of that into the plain HTML, CSS, and JavaScript the browser
understands — and, while it's at it, makes those files small and fast. Without it, you'd
be hand-writing browser-ready code and manually wiring together every file. With it, you
write in whatever is productive and the tooling produces what actually ships.

## What a bundler does

A **bundler** — Vite, webpack, Parcel, esbuild — is the engine of the build. Its core job
is in the name: take your entry file, follow every `import` to build a graph of all the
code your app actually uses, and **bundle** it into a small number of output files. It
does several jobs in that one pass:

- **Resolve modules.** Follow every `import` — your files and installed packages — into
  one dependency graph.
- **Transpile.** Convert JSX, TypeScript, and newer JavaScript features into plain
  JavaScript older browsers accept (the classic transpiler here is **Babel**).
- **Bundle.** Combine those many files into a few, so the browser makes a handful of
  requests instead of hundreds.
- **Optimise.** **Minify** (strip whitespace, shorten names) and **tree-shake** (drop
  code nothing references) to shrink what downloads.

```js
// You write small, focused modules and import between them…
import { formatCall } from "./format.js";
import { Dashboard } from "./Dashboard.jsx";

// …the bundler follows every import, transpiles, and emits a few tight files.
```

The output is typically one or a few `.js` files, a `.css` file, and an `index.html`
wired to them — the artifact you actually deploy, covered in
[deploying a web app](/learn/deployment/what-is-deployment/).

## Transpiling, minifying, tree-shaking

Three transforms show up constantly, and it's worth knowing them by name:

- **Transpiling** turns code the browser can't run into code it can — JSX and
  TypeScript into JavaScript, or a brand-new language feature into an older equivalent.
  It's translation between flavours of the same kind of code.
- **Minifying** rewrites the shipped JavaScript and CSS to be as small as possible —
  removing whitespace and comments, shortening variable names — without changing
  behaviour. Smaller files download and parse faster.
- **Tree-shaking** analyses the import graph and **removes code nothing uses**. Import
  one helper from a big library and tree-shaking can drop the rest, so you ship only
  what you actually call.

Together these are why a huge, readable codebase can become a tiny, fast download — a
direct win for [performance](/learn/web-dev/responsive-design/) and the load times users
feel.

## The dev server and the production build

A bundler wears two hats. In **development** it runs a **dev server**: it serves your app
locally and, thanks to **hot module replacement (HMR)**, pushes each change into the
running page **instantly** — often without losing your place — so the edit-refresh loop
is near-immediate. It also emits **source maps**, which let the browser's debugger show
your original source instead of the transpiled, minified output.

For **production** you run a separate **build** command that does the full pipeline —
transpile, bundle, minify, tree-shake — and writes optimised files to an output folder
(often `dist/`). The two modes have opposite priorities: development optimises for
**fast feedback**, production optimises for **small, fast-loading output**. Newer tools
like **Vite** and **esbuild** made both dramatically faster than the older webpack setups,
which is much of why front-end tooling feels lighter than it did a few years ago.

## When you don't need one

None of this is mandatory. A page of hand-written HTML, CSS, and a little JavaScript
runs in the browser with **no build step at all** — and that's a perfectly good way to
build many sites, including largely this one (see
[static vs. dynamic](/learn/web-dev/static-vs-dynamic/) and
[templating &amp; static sites](/learn/web-dev/templating-and-static-sites/)). You take on
a build step when you adopt a framework, JSX, TypeScript, or a big dependency graph —
that is, when the productivity of writing in those outweighs the cost of the tooling. The
build step is a tool, not a rite of passage; reach for it when it pays for itself.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the browser only runs plain HTML, CSS, and JavaScript, so a build step transpiles and bundles things like JSX and TypeScript into what it can run." markdown="0">
  <p class="knowledge-check__q">Quick check: why do front ends using JSX or TypeScript need a build step?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To encrypt the source code so users can't read it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The browser only runs plain HTML, CSS, and JS, so JSX/TS must be transpiled and bundled first</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Browsers charge for JavaScript unless it's bundled into one file</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The browser runs only **HTML, CSS, and JavaScript**, so a **build step** converts
  everything else — **JSX, TypeScript, many modules** — into those.
- A **bundler** (Vite, webpack, esbuild) resolves imports into a dependency graph and
  combines many source files into a few optimised outputs.
- **Transpiling** translates newer or non-standard syntax to what browsers accept;
  **minifying** shrinks files; **tree-shaking** drops unused code.
- In **development** a dev server gives instant updates (**HMR**) and **source maps**;
  a separate **production build** emits small, optimised files.
- Newer tools like **Vite** made both modes much faster than older setups.
- A hand-written static page needs **no build step** — you take one on when a
  framework or typed language makes it worth it.

Next up: [Responsive design](/learn/web-dev/responsive-design/).
