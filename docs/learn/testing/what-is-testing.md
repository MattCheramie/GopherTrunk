---
slug: what-is-testing
title: What is testing?
description: Manual vs automated testing, verification vs validation, and the test pyramid — the map of unit, integration, and end-to-end tests that the rest of this module fills in.
keywords: what is software testing, test pyramid, unit test vs integration test, manual vs automated testing, verification vs validation, testing levels, automated tests explained
level: beginner
status: full
faq:
  - q: What is the test pyramid?
    a: A guideline for balancing test types. At the base are many small, fast unit tests covering individual functions; in the middle, fewer integration tests proving that real components work together; at the top, a handful of slow end-to-end tests driving the whole system like a user. The pyramid shape says most of your tests should be the cheap fast kind, with progressively fewer as tests get bigger and slower.
  - q: What's the difference between manual and automated testing?
    a: Manual testing is a human running the program and checking behavior by hand — flexible, but slow and unrepeatable. Automated testing is code that checks other code, so the same checks run identically in seconds, thousands of times, on every change. Real projects use both, but automation is what makes it practical to re-verify everything after every edit.
  - q: What's the difference between verification and validation?
    a: Verification asks "did we build the thing right?" — does the code meet its specification, which tests can check. Validation asks "did we build the right thing?" — does the software actually solve the user's problem, which requires putting it in front of reality. A program can pass every test and still be the wrong product; you need both questions.
---

# What is testing?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Testing** is deliberately running software to find out whether it behaves as
intended. **Manual** testing is a human checking by hand; **automated** testing
is code that checks code, repeatable in seconds forever. The **test pyramid**
organizes automated tests into layers — many fast **unit tests**, fewer
**integration tests**, a handful of **end-to-end tests** — and is the map for
Units 2 and 3. Testing does **verification** ("built the thing right?");
**validation** ("built the right thing?") needs reality too.
</div>

You know why software breaks and what that costs. This lesson introduces the
defense the rest of the module builds out — and gives you the map so every later
lesson has a place to hang on.

## What does "testing" actually mean?

Testing is running software **on purpose, with a prediction**. You choose an
input, you know what the correct output should be, you run the code, and you
compare. That's it — the entire field is elaborations of *choose input → predict
output → run → compare*.

The prediction is the part beginners skip. Poking at a program and nodding at
plausible-looking output isn't testing; it's rehearsal. A test exists when a
wrong answer would be *detected* — when there's a stated expectation the actual
behavior either meets or fails.

## Manual vs automated

| | Manual testing | Automated testing |
|--|----------------|-------------------|
| What it is | A human runs the program and checks by hand | Code runs the program and checks the results |
| Speed | Minutes per check | Milliseconds per check |
| Repeatability | Drifts — humans skip steps, forget cases | Identical every run |
| Best at | Exploring, judging look-and-feel, spotting the unexpected | Re-verifying everything, on every change, forever |
| Cost curve | Costs the same every single time | Costs once to write, ~free to re-run |

Manual testing never goes away — a human exploring the program finds problems no
scripted check anticipated. But the killer feature of automation is
**re-verification**. Lesson 1 said change is a main source of bugs; automated
tests mean that after *every* change, everything you've ever verified gets
re-verified in seconds. No team of humans can offer that. When this module says
"tests" without qualification, it means automated ones.

## The test pyramid

Automated tests come in sizes, and the classic way to balance them is the
**test pyramid**:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 220" role="img" aria-label="A pyramid with three layers: a wide base labeled unit tests, a middle band labeled integration tests, and a small top labeled end-to-end tests. Arrows beside it show speed and quantity increasing toward the base, realism increasing toward the top." xmlns="http://www.w3.org/2000/svg">
  <polygon points="260,20 380,110 140,110" fill="none" stroke="currentColor" stroke-width="2"/>
  <polygon points="140,110 380,110 440,185 80,185" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="140" y1="110" x2="380" y2="110" stroke="currentColor" stroke-width="1"/>
  <text x="260" y="90" text-anchor="middle" font-size="13" fill="currentColor">end-to-end</text>
  <text x="260" y="150" text-anchor="middle" font-size="13" fill="currentColor">integration</text>
  <text x="260" y="176" text-anchor="middle" font-size="13" fill="currentColor">unit tests — many, tiny, fast</text>
  <line x1="470" y1="180" x2="470" y2="40" stroke="currentColor" stroke-width="1.5" marker-end="none"/>
  <polygon points="470,32 465,44 475,44" fill="currentColor"/>
  <text x="482" y="70" font-size="11" fill="currentColor" transform="rotate(90 482 70)">realism, cost</text>
  <line x1="50" y1="40" x2="50" y2="180" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="50,188 45,176 55,176" fill="currentColor"/>
  <text x="38" y="66" font-size="11" fill="currentColor" transform="rotate(-90 38 130)">speed, quantity</text>
</svg>
<figcaption>The test pyramid: lots of fast unit tests at the base, fewer integration tests, a handful of end-to-end tests on top.</figcaption>
</figure>

- **Unit tests** (the base) check one small piece — usually a function — in
  isolation. Milliseconds each, so you can have thousands and run them
  constantly. [Unit 2](/learn/testing/what-is-a-unit-test/) is all about them.
- **Integration tests** (the middle) wire real pieces together — code plus a
  database, a file, a network — and check the *joints*, where lesson 3's Mars
  Orbiter–style bugs live. Fewer, slower, more realistic.
- **End-to-end tests** (the tip) drive the whole assembled system the way a user
  would. Most convincing, most expensive, so you keep only a few.

The shape is the advice: **mostly base, sparingly top**. An inverted pyramid —
a few giant end-to-end tests and no units — runs slowly, fails vaguely
("something, somewhere, is wrong"), and gets skipped. GopherTrunk follows the
shape: thousands of unit tests run on every commit via `make vet test`, while
the heavier replay integration suite (`make integration`) runs when the decode
paths change.

## Verification vs validation

One boundary to draw honestly. Tests perform **verification**: they check the
software against *your stated expectations*. But your expectations can be wrong
— you can build, and flawlessly verify, a program that solves the wrong problem.
Checking that the software actually serves its users is **validation**, and no
test suite can do it; it takes real users, real data, real conditions.

Keep both questions alive. "All tests pass" means *it does what I said it
should*. Whether what you said is right — that's a different check, and Unit 6's
[capture-gated verification](/learn/testing/capture-gated-verification/) lesson
shows what happens when a project takes that difference seriously.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the base is many small fast unit tests, thinning toward a few end-to-end tests at the top." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the test pyramid recommend?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Mostly end-to-end tests, since they're the most realistic</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Equal numbers of every kind of test</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Many fast unit tests, fewer integration tests, a handful of end-to-end tests</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Testing is **choose input → predict output → run → compare** — the prediction
  is what makes it a test.
- **Automated** tests re-verify everything after every change for ~free; that's
  the property manual testing can't match.
- The **test pyramid**: many **unit tests**, fewer **integration tests**, a few
  **end-to-end tests** — mostly base, sparingly top.
- Tests do **verification** (built the thing right); **validation** (built the
  right thing) requires reality.
- GopherTrunk lives the pyramid: `make vet test` on every commit, heavier replay
  integration runs when decode paths change.

Next up: [The testing mindset](/learn/testing/the-testing-mindset/)
