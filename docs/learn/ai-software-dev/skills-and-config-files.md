---
slug: skills-and-config-files
title: Skills & AI config files
description: How to give an AI coding tool standing instructions so you stop repeating yourself — custom instructions, project config files like CLAUDE.md and AGENTS.md, editor rules files, and loadable skills. Learn what belongs in each and how they differ.
keywords: AI config files, CLAUDE.md, AGENTS.md, custom instructions, system prompt, cursor rules, copilot instructions, agent skills, project conventions, standing instructions, GopherTrunk
level: intermediate
prereq:
  - prompting-for-code
faq:
  - q: "What is the difference between a config file and a skill?"
    a: "A **config file** is static standing context — the tool reads it automatically and applies it to the whole session (project conventions, build commands, do's and don'ts). A **skill** is a packaged, reusable capability the model loads *on demand* when a task calls for it — instructions, and sometimes scripts or templates, bundled together. The config file is always on; a skill is pulled in only when relevant, which keeps the context lean."
  - q: "Do I have to use a file named CLAUDE.md?"
    a: "No — the exact filename depends on the tool, and the conventions change over time. Claude Code reads `CLAUDE.md`; a cross-tool convention called `AGENTS.md` is gaining traction; editors like Cursor use a rules folder; Copilot has its own instructions file. The *idea* is identical across all of them: a project file with standing instructions the tool reads on its own. Check your tool's current docs for the exact name it looks for."
  - q: "What should I put in a project config file?"
    a: "The things you'd otherwise retype in every prompt: project conventions and style, the exact build and test commands, a short architecture orientation, and explicit do's and don'ts. Keep it short and concrete — a focused file the model actually follows beats a long one it half-ignores."
---
# Skills & AI config files

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Standing instructions stop the repetition** — give the model your conventions once instead of in every prompt. **Three levels** — account-wide custom instructions, project config files the tool reads automatically, and editor rules files. **Skills load on demand** — packaged, reusable capabilities the model pulls in only when a task needs them, unlike an always-on config file.
</div>

This is lesson 20 of the path. The previous lesson was about the *one* prompt you type right now; this one is about everything you'd otherwise have to type *every* time. If you find yourself ending prompt after prompt with "...and use the standard library, and write a test, and keep the change small," you're hand-feeding the model standing context it should already have. By the end of this lesson you'll know the three levels at which you can give a model permanent instructions, what belongs in each, and how a loadable *skill* differs from a static config file.

## Why standing instructions exist

A model has no memory between sessions on its own — each conversation starts blank, knowing only what's in its [context window](/learn/ai-software-dev/context-windows/). So your project's conventions, build commands, and rules have to get into the context *somehow* on every session. You can do that by hand in each prompt, which is tedious and error-prone, or you can write them down once in a place the tool loads automatically. That's all a config file is: standing context, applied for you.

The payoff is consistency. When the rules live in a file the tool always reads, the model produces code that matches your project the same way every time — not the way you happened to phrase it in this prompt versus that one.

## Three levels of standing instruction

Standing instructions live at three scopes, from broadest to most specific. They stack: account rules apply everywhere, project rules add to them for one repo, and a per-prompt instruction overrides for one request.

| Level | Scope | Where it lives | Good for |
|---|---|---|---|
| **Custom instructions / system prompt** | Every chat you have | Account or tool settings | Your personal style — "explain trade-offs," "be terse," "prefer the standard library." |
| **Project config file** | One repository | A file in the repo the tool reads automatically | Project conventions, build/test commands, architecture notes, do's and don'ts. |
| **Editor rules files** | One project, in your IDE | The editor's rules location | IDE-specific guidance, often scoped to certain folders or file types. |

**Account-wide custom instructions** (the [system/role prompt](/learn/ai-software-dev/prompting-for-code/) from the last lesson, exposed in settings) describe *you* — preferences that hold across every project. Keep these about your working style, not about any one codebase.

**Project config files** are the workhorse. Because they live *in the repo*, they travel with the code, get version-controlled, and are shared with everyone (and every agent) who works on it. The tool reads the file automatically at the start of a session, so the model starts already knowing how your project works.

**Editor rules files** do the same job inside an IDE, often with the ability to scope a rule to a folder or file pattern — "in `internal/dsp`, always preserve the existing buffer-reuse pattern."

## The filenames vary — the idea doesn't

Here the [evergreen warning](/learn/ai-software-dev/the-provider-landscape/) applies hard: the exact filenames are a moving target and differ by tool. As representative examples at the time of writing:

- **`CLAUDE.md`** — read by Claude Code, placed at the repo root (and optionally in subdirectories).
- **`AGENTS.md`** — an emerging *cross-tool* convention, an attempt at one file multiple agents can read so you don't maintain a separate file per tool.
- **Editor rules** — for example a `.cursor/rules` location in the Cursor editor.
- **Copilot instruction files** — GitHub Copilot's own instructions file for repo-level guidance.

Don't memorise these names; they will change, and new ones will appear. Memorise the *mechanism*: a project-scoped file, in a location your tool watches, that the tool loads on its own. When you adopt a new tool, the question to ask its docs is simply "what file do you read for project instructions?"

## What belongs in a project config file

Keep it short, concrete, and about *this* project. A focused file the model actually follows beats a long one it skims. The high-value contents:

- **Project conventions and style** — naming, error handling, formatting rules that aren't already enforced by a linter.
- **Build and test commands** — the exact commands, so the model can verify its own work instead of guessing.
- **Architecture orientation** — a few lines on how the codebase is laid out and where things go.
- **Explicit do's and don'ts** — the sharp-edged rules that prevent recurring mistakes.

Here's a short, realistic example in the GopherTrunk style:

```markdown
# GopherTrunk — guidance for AI assistants

GopherTrunk is a pure-Go SDR trunking scanner/decoder (P25, DMR, NXDN, TETRA).

## Build & test
- `make vet test` — vet + unit tests; must be green before any commit.
- Single package while iterating: `go test ./internal/scanner/ccdecoder/...`.

## Conventions
- Standard library first; justify every new dependency.
- Match the existing error style in the package you're editing.
- DSP code is rate-invariant — don't hard-code sample rates.

## Do / don't
- Bug fix = one narrow commit + a regression test that FAILS without the
  fix and PASSES with it. If you can't write the failing test, you haven't
  reproduced the bug yet — keep digging, don't guess.
- Keep refactors out of behaviour-change commits.
- Don't widen the scope of a change beyond what was asked.
```

Notice it gives the model what the [prompting lesson](/learn/ai-software-dev/prompting-for-code/) said it can't infer — commands, conventions, and the failing-first-test rule — once, so no prompt has to repeat them. This isn't hypothetical: the real GopherTrunk repository carries a file very much like this, and it shapes every AI session against the codebase.

## Skills: capabilities loaded on demand

A config file is *always on* — it's part of the context for the whole session whether or not the current task needs it. A **skill** is different: it's a packaged, reusable unit of instruction (and sometimes accompanying scripts, templates, or reference docs) that the model loads *only when a task calls for it*.

Think of the difference this way. Your project config is the standing house rules, read on entry. A skill is a specialist manual on the shelf — "how to add a new protocol decoder," "how to write a DSP benchmark" — that the model reaches for when, and only when, the work matches it. Anthropic's **Agent Skills** are one concrete implementation of this idea, but the pattern is broader than any one product.

| | Config file | Skill |
|---|---|---|
| **When it applies** | Always, for the whole session | On demand, when a task matches it |
| **Contains** | Standing context and rules | Instructions, often plus scripts or templates |
| **Purpose** | Make the model fit *this project* | Give the model a reusable *capability* |
| **Cost in context** | Occupies context the whole time | Loaded only when relevant, keeping context lean |

The reason this distinction matters is the same reason context management matters generally: context is finite, and relevant beats big. Loading every possible instruction all the time crowds out the code the model actually needs to see. Skills let you keep a large library of capabilities without paying for all of them on every request. That trade-off — what to feed the model and when — is the whole subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a config file is standing context the tool always loads; a skill is a capability pulled in only when a task needs it." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the key difference between a project config file and a skill?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A config file is for code and a skill is only for documentation</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A config file is always loaded; a skill is loaded on demand when a task matches it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A skill applies to every project while a config file applies to one chat</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Standing instructions stop repetition** — write your conventions once where the tool reads them, instead of in every prompt.
- **Three levels** — account-wide custom instructions for your style, a project config file for repo conventions, and editor rules files for IDE-scoped guidance.
- **Filenames vary** — `CLAUDE.md`, the cross-tool `AGENTS.md`, editor rules folders, Copilot instruction files; learn the mechanism, not the name, since both change over time.
- **Keep config files concrete** — conventions, exact build/test commands, brief architecture notes, and sharp do's and don'ts the model will actually follow.
- **Skills load on demand** — packaged, reusable capabilities the model pulls in only when a task matches, unlike an always-on config file.
- **Context is finite** — skills exist so a large library of capabilities doesn't crowd the context all at once.

Next up: the broader art of getting the right material in front of the model — attaching files, retrieval, MCP, and pruning what doesn't help — in [Feeding the model the right context](/learn/ai-software-dev/providing-context/).
