---
slug: dev-environment-workflow
title: Your environment & daily workflow
description: Set up an editor, terminal, Git, dotfiles, format-on-save and a tight edit-build-test loop, plus reproducible environments for fast, comfortable solo development.
keywords: development environment, daily workflow, edit build test loop, dotfiles, formatters linters, devcontainer, version managers, Git workflow
level: intermediate
status: full
faq:
  - q: "What is the edit-build-test loop?"
    a: "The **edit-build-test loop** is the core cycle of programming — change some code, build or run it, and check the result. The speed of this loop, often called **inner loop** time, is one of the biggest factors in how productive and happy you are. Solo developers should obsess over keeping it fast, because they run it thousands of times a week with no one else to share the wait."
  - q: "Why bother with dotfiles?"
    a: "**Dotfiles** are the small config files (for your shell, editor, Git and tools) that capture how your environment is set up. Keeping them in a Git repository means a new machine can be made to feel like home in minutes, your settings are backed up and versioned, and you never lose that one perfect config you spent an evening tuning."
  - q: "What makes an environment \"reproducible\"?"
    a: "A **reproducible environment** is one anyone — including future you — can recreate exactly, with the same language version and tools, rather than relying on whatever happens to be installed. **Version managers** pin language versions per project, and **containers or devcontainers** package the whole toolchain, so \"works on my machine\" stops being a problem even when the only machine is yours."
---
# Your environment & daily workflow

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Comfortable tools** — editor, terminal and Git you know cold. **Tight loop** — a fast edit-build-test cycle you run all day. **Reproducible** — dotfiles and containers so any machine, including future you, can rebuild it.
</div>

Your stack decides *what* you build with; your environment decides *how it feels* to
do it. Solo, you run the same small loop — change code, run it, check it — thousands
of times a week, with no one to share the waiting or the friction. So the payoff for
tuning your environment is enormous and entirely yours. This lesson sets up a
workplace that is fast, comfortable and reproducible: an editor you know cold, a
terminal that fits your hand, version control you trust, automatic formatting, and a
build-test loop tight enough that you stay in flow. None of it is fancy; all of it
compounds.

## Editor, terminal, and version control

Three tools form the floor of any setup. You do not need the *best* ones — you need
ones you know deeply:

- **Editor or IDE.** Whether it is a full IDE or a lean editor, learn its keybindings,
  its search, its jump-to-definition. Fluency here means your hands keep up with your
  thoughts. A good language server giving you autocomplete and inline errors as you
  type is worth far more than any theme.
- **Terminal.** You will live here — running builds, tests, Git and your program.
  Learn a handful of shell basics and let the terminal become an extension of the
  editor rather than a scary other place.
- **Version control with Git.** This is non-negotiable, even alone. Git is your undo
  button across the whole project, your record of *why* a change was made, and your
  safety net for trying risky ideas on a branch. If Git still feels like magic
  incantations, work through the dedicated [Git & GitHub path](/learn/git/) — it pays
  for itself within a week.

Commit small and often, with messages that explain *why*, not just *what*. Future you
is the new hire reading them, and you will not remember the context.

## Capture your setup in dotfiles

Once your environment feels right, **save it.** Your shell config, editor settings,
Git config and tool preferences are all just text files — your **dotfiles** — and
they belong in a Git repository of their own. The benefits are quietly huge:

- A **new machine** becomes home in minutes, not an afternoon of half-remembered
  tweaks.
- Your settings are **backed up and versioned**, so the perfect config you tuned one
  evening is never lost to a dead laptop.
- You can **see and undo** changes to your environment the same way you do for code.

You do not have to do this on day one, but the moment you find yourself re-configuring
the same thing on a second machine, start a dotfiles repo. It is the cheapest
insurance in software.

## Let tools do the boring checks

A solo developer has no one to nitpick style in review, so hand that job to machines
and forget about it. Two tools earn their place immediately:

- **A formatter** rewrites your code to a consistent style automatically. Run it
  **on save**. You stop thinking about indentation and spacing entirely, and you never
  again argue with yourself about it. Many languages ship one (Go has `gofmt`; others
  have equivalents).
- **A linter** flags likely bugs and bad smells — unused variables, suspicious
  comparisons, ignored errors. Run it on save or on every build so problems surface
  the instant you create them, not days later.

```text
on save:  format the file  →  run the linter  →  show errors inline
```

Wiring these into your editor means quality checks happen *for free*, thousands of
times a day, with zero willpower required. This is the first installment of the
"automation as your missing teammate" idea we develop in
[keeping quality high alone](/learn/intro-software-dev/quality-when-solo/).

## Keep the loop tight

The **edit-build-test loop** is the heartbeat of your day: change something, build or
run it, see the result. The shorter that cycle, the more you stay in flow and the
faster you learn whether an idea works. A loop that takes 30 seconds versus 2 seconds
is the difference between deep focus and constantly losing your train of thought.

Things that keep the loop fast:

- **Incremental builds** — only rebuild what changed, not the whole project.
- **Run the smallest unit** — execute the single test or function you are working on,
  not the entire suite, while iterating.
- **Watch mode** — tools that rerun automatically on save remove the manual step
  entirely.
- **Fast feedback inline** — errors appearing in the editor beat reading a wall of
  terminal output.

Treat a slow inner loop as a bug worth fixing. Every second you shave gets multiplied
by the thousands of times you run it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a fast edit-build-test loop keeps you in flow and gives near-instant feedback, which you run thousands of times a week." markdown="0">
  <p class="knowledge-check__q">Quick check: why do solo developers obsess over a fast edit-build-test loop?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because faster builds make the final program run faster too</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because they run the loop thousands of times a week, so its speed compounds into focus and productivity</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because slow loops use more electricity</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Make the environment reproducible

The classic disaster is "it works on my machine" — and when you are the only machine,
the disaster is a dead laptop or a fresh install wiping out a setup you can no longer
recreate. **Reproducible environments** solve this by capturing not just your code but
the tools that build it:

- **Version managers** pin the exact language version per project, so upgrading the
  language globally never silently breaks an old project.
- **Containers and devcontainers** package the whole toolchain — language, libraries,
  system tools — into a definition file you commit alongside the code. Anyone, on any
  machine, gets the identical environment from one command.

The payoff is that future you, on a new computer a year from now, can clone the repo
and be productive in minutes instead of rebuilding a fragile setup from memory. It is
the same bus-factor insurance as dotfiles, applied to the project itself.

## A concrete daily loop

Here is what a focused solo session can actually look like, tying it all together:

1. Open the project; the editor's language server lights up with autocomplete and
   inline errors.
2. Create a **branch** for today's task so the main branch stays shippable.
3. Write a small change; on save, the **formatter** tidies it and the **linter**
   flags an unused import you fix immediately.
4. Run just the **one test** for the function you are changing — under a second —
   watch it go red, make it green.
5. Run the program against real input. For a radio tool that might mean feeding it a
   captured sample file and watching it decode; the tight loop lets you tweak the
   parser and re-run in seconds.
6. **Commit** with a message explaining *why*, then move to the next small step.
7. When the task is done and tests pass, merge the branch back.

That is the rhythm: tiny changes, instant feedback, frequent commits, a clean main
branch. Repeat it and real software accumulates almost without you noticing.

## Recap

- **Know your core tools cold** — editor, terminal and Git fluency lets your hands
  keep up with your thinking.
- **Use Git even alone** — it is your undo, your history, and your safety net for
  risky ideas.
- **Save your setup in dotfiles** — version your environment so a new machine becomes
  home in minutes.
- **Automate formatting and linting** — run them on save so quality checks cost no
  willpower.
- **Keep the loop tight** — a fast edit-build-test cycle compounds into focus and
  productivity.
- **Make it reproducible** — version managers and containers mean future you can
  rebuild the whole environment from the repo.

Next up: turning a vague idea into a project you can actually finish in
[from idea to project](/learn/intro-software-dev/scoping-a-project/).
