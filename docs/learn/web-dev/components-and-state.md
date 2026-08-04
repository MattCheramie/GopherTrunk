---
slug: components-and-state
title: Components & state
description: The two ideas at the heart of every modern front-end framework — the component as a reusable, self-contained piece of UI, and state as the data that drives what a component shows and re-renders it when it changes.
keywords: component, state, props, re-render, UI state, component tree, reusable UI, React state, useState, lifting state up, unidirectional data flow
level: intermediate
status: full
prereq:
  - frontend-frameworks
faq:
  - q: What is the difference between props and state?
    a: "Props are the inputs a component receives from its parent — read-only data passed down. State is data a component owns and can change over time. When state changes, the component re-renders; when a parent passes new props, the child re-renders too. Roughly: props flow in from outside, state lives inside."
  - q: What does 'the UI is a function of state' mean?
    a: "It means the screen is fully determined by the current state — feed the same state in and you get the same UI out. You never reach in and edit the DOM; you update the state, and the framework re-renders the affected components to match. This is what makes complex UIs predictable."
  - q: Why do frameworks make you call a setter instead of just reassigning a variable?
    a: "Because the framework needs to know the data changed so it can re-render. A plain reassignment is invisible to it. Calling the setter (or mutating a reactive object it's watching) is the signal that says 'this changed — update the UI,' which is how declarative rendering stays in sync."
---

# Components & state

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every modern UI is built from **components** — small, reusable, self-contained pieces
of UI — arranged into a tree. What a component shows is driven by **state**: the data
it holds, plus **props** passed in from its parent. The golden rule is that the **UI
is a function of state** — you never edit the page directly; you change the state and
the [framework](/learn/web-dev/frontend-frameworks/) re-renders the components that
depend on it. Data flows **down** through props and changes flow **up** through
events, which keeps even a large app predictable.
</div>

The [previous lesson](/learn/web-dev/frontend-frameworks/) said frameworks let you
describe the UI declaratively and keep it in sync with your data. This lesson is the
two ideas that make that work in practice: the **component**, which is how you break a
UI into pieces, and **state**, which is the data that drives those pieces. Get these
two and you have the mental model behind React, Vue, Svelte, and the rest — the syntax
is just detail on top.

## A component is a reusable piece of UI

A component is a self-contained chunk of interface — a button, a search box, a call
row, a whole panel — bundling its markup, its styling, and its behaviour in one place.
You define it once and use it wherever you need it, like a function you call to get UI.

```jsx
// A small component: takes inputs, returns the UI for one call.
function CallRow({ talkgroup, seconds }) {
  return (
    <li className="call-row">
      <span className="tg">{talkgroup}</span>
      <span className="len">{seconds}s</span>
    </li>
  );
}
```

Components **compose**: bigger ones are built from smaller ones, forming a **tree** with
one root at the top. A `Dashboard` contains a `CallList`, which renders many `CallRow`s.
This is the same "break a big thing into small, named, reusable pieces" instinct as
functions in ordinary code — applied to the interface — and it is why a screen with
thousands of elements stays comprehensible.

## Props: inputs passed down

A component almost never stands alone; it takes **inputs** from its parent, called
**props**. In `CallRow` above, `talkgroup` and `seconds` are props — the parent decides
their values, the child just renders them. Props are **read-only** to the child: it
displays them but does not change them. That one-way flow — data handed *down* the tree
— is what makes a component reusable, because the same `CallRow` works for any call the
parent gives it.

```jsx
// The parent supplies props for each child.
function CallList({ calls }) {
  return (
    <ul>
      {calls.map((c) => (
        <CallRow key={c.id} talkgroup={c.talkgroup} seconds={c.seconds} />
      ))}
    </ul>
  );
}
```

## State: data that changes over time

Props come from outside; **state** is data a component **owns and can change**. A
search box owns the text you've typed; a dashboard owns the list of calls received so
far; a toggle owns whether it's on. State is the memory of the UI between events.

The crucial move is *how* you change it. You don't reach into the DOM and edit the
page — you update the state through the framework, and it **re-renders** the component
to match.

```jsx
import { useState } from "react";

function Filter() {
  const [query, setQuery] = useState("");   // state, starts empty
  return (
    <input
      value={query}
      onChange={(e) => setQuery(e.target.value)}  // change state → re-render
      placeholder="Filter talkgroups"
    />
  );
}
```

Calling `setQuery` is you telling the framework "this data changed." That signal is
essential: a plain `query = "..."` reassignment would be invisible, and the UI would
never update. The setter is the bridge between your data and the declarative render.

## The UI is a function of state

Put props and state together and you get the single most important rule in front-end
development: **the UI is a function of state.** Given the current state, the screen is
fully determined — same state in, same UI out. You never manually sync the DOM; you
change the state and the framework recomputes the affected components.

This is why the counter-out-of-sync bug from the last lesson simply cannot happen: the
count and the list are both derived from the same `calls` state, so they can't
disagree. It also makes UIs easier to reason about and test — to check a screen, you set
up a state and look at what renders, no clicking through steps required.

## Data down, events up

If props only flow *down*, how does a child change something a parent owns — like a
row telling the dashboard it was clicked? The answer is the mirror of props: the parent
passes a **callback** down, and the child **calls it** to send a change back up. Data
flows **down** as props; changes flow **up** as events. State lives at the lowest
component that needs it, and when two siblings need the same state, you **lift it up** to
their shared parent.

```jsx
// Parent owns the state and passes a callback down; child calls it up.
function Dashboard() {
  const [selected, setSelected] = useState(null);
  return <CallRow talkgroup="Fire-1" onSelect={() => setSelected("Fire-1")} />;
}
```

This one-way, top-down flow — often called **unidirectional data flow** — is what keeps
a big component tree predictable: there is always one owner of each piece of state, and
one direction changes travel. It is the discipline that lets an app grow to hundreds of
components without becoming a tangle, and it carries directly into
[fetching data](/learn/web-dev/fetching-data/), where the data comes from a server
instead of a click.

<div class="knowledge-check" data-quiz data-correct-msg="Right — you change the state and the framework re-renders; you never edit the DOM directly. The UI is a function of state." markdown="0">
  <p class="knowledge-check__q">Quick check: in a component framework, how do you update what's on screen?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Find the DOM node and edit its text directly, as with plain JavaScript</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Change the component's state and let the framework re-render to match</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Reload the whole page so the new data is fetched again</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **component** is a reusable, self-contained piece of UI; components **compose**
  into a tree with one root.
- **Props** are read-only inputs a parent passes **down** to a child, which is what
  makes a component reusable.
- **State** is data a component owns and can change; changing state through the
  framework triggers a **re-render**.
- The core rule is that the **UI is a function of state** — you change state, never the
  DOM, and the framework keeps the screen in sync.
- Data flows **down** as props and changes flow **up** as events; shared state is
  **lifted** to a common parent (**unidirectional data flow**).
- This mental model — components plus state — is the same across
  [every framework](/learn/web-dev/frontend-frameworks/).

Next up: [Fetching data from the front end](/learn/web-dev/fetching-data/).
