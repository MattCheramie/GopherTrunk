---
slug: the-provider-app
title: Using the provider's app
description: The provider's chat app — ChatGPT, Claude.ai, the Gemini app — is the simplest way to use AI for coding. Learn what it's great at, where copy-paste friction and missing repo context hold it back, and when it's the right tool.
keywords: chat app, ChatGPT, Claude.ai, Gemini app, AI coding, copy paste, multimodal, explain error, stack trace, planning, code snippets, beginner AI tools
level: beginner
status: full
prereq:
  - understanding-cost
faq:
  - q: "Do I need to install anything to use a provider's chat app?"
    a: "No. The chat app runs in a web browser, and most providers also offer a desktop and phone version. You sign in and start typing. That zero-setup quality is the whole appeal — it's the fastest way to get a question answered without touching your editor or terminal."
  - q: "Can the chat app see my project files?"
    a: "Not on its own. The model only knows what's in the conversation, so it sees a file only if you paste it (or upload it). It has no automatic access to your repository, your editor, or the rest of your codebase. That's the main limitation compared with editor-integrated and agentic tools."
  - q: "Is the chat app good for learning, or just for generating code?"
    a: "It's excellent for learning. Asking it to explain a concept, walk through an error, or compare two approaches is often where it shines most — better, for many people, than generating large chunks of code you then have to verify and wire in by hand."
---
# Using the provider's app

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Zero setup** — the chat app in your browser, desktop, or phone is the simplest on-ramp to AI for coding. **Great for thinking** — asking questions, learning concepts, planning, and explaining errors is where it excels. **Copy-paste friction** — the model can't see your repo unless you paste it, and nothing it produces is applied to your code automatically.
</div>

This is lesson 11 of the path, and the first in a short module about *where* you actually use AI in your day-to-day work. Earlier lessons covered how models work and how to think about cost; now we look at the concrete tools. We start with the simplest one of all: the provider's chat app. By the end of this lesson you'll understand what the chat app is, the kinds of work it's genuinely good at, the friction that makes it a poor fit for hands-on coding, and how to tell when it's the right tool to reach for.

## What "the provider's app" means

Every major AI provider ships a **chat app** — a conversational interface you open and type into. You've almost certainly seen them: OpenAI's **ChatGPT**, Anthropic's **Claude.ai**, Google's **Gemini** app. They run in a web browser, and most also offer a desktop application and a phone app that all sync to the same account.

The interaction model is a back-and-forth conversation. You type a message, the model replies, and the whole exchange stays on screen as **context** — the running text the model can see when it answers (we covered this in [Context windows](/learn/ai-software-dev/context-windows/)). There is no project, no file tree, no terminal. It is just you and a text box.

That simplicity is the point. Compared with the editor and command-line tools we'll meet in the next two lessons, the chat app asks almost nothing of you up front.

## The big advantage: zero setup

The chat app's defining strength is that there is nothing to install, configure, or wire into your workflow. You open a tab, sign in, and start asking. For someone new to AI-assisted development — or just trying it for the first time on a given problem — this removes every obstacle between having a question and getting an answer.

A second strength is that modern chat apps are usually **multimodal**, meaning they accept more than just text. You can paste or upload a screenshot of a crashing UI, a photo of a whiteboard sketch, a diagram, or a chunk of log output, and the model can reason about it. This is genuinely useful: instead of describing an error in prose, you paste the actual screen.

Here is where the chat app tends to be the best tool for the job:

| Task | Why the chat app fits |
|------|----------------------|
| Asking a "how does X work?" question | No code needs to change; you just want an explanation |
| Learning a concept or comparing approaches | A conversation is the natural shape for back-and-forth questions |
| Planning before you write code | Sketching an approach doesn't need access to your files yet |
| Generating an isolated snippet | A self-contained function or example stands alone |
| Explaining an error message or stack trace | You paste the text; the model reads it directly |

Notice the pattern: these are tasks where the relevant information is small enough to paste, and where the *output* is understanding or a self-contained piece of text — not a change spread across your project.

## A GopherTrunk example: decoding a compiler error

Say you're working on GopherTrunk — our pure-Go [software-defined radio](/learn/intro-software-dev/what-is-software/) scanner — and the Go compiler throws something opaque:

```text
./ddc.go:142:18: cannot use sampleRate (variable of type float64)
  as int value in argument to newDownconverter
```

If you've never seen Go's type rules before, that message can be intimidating. You copy it, paste it into the chat app, and ask: "I'm new to Go. What does this error mean and how do I fix it?" The model explains that Go won't silently convert a `float64` to an `int`, that you likely need an explicit conversion like `int(sampleRate)`, and that you should check whether truncating the fractional part is actually what you want here.

This is the chat app at its best. The error is small, self-contained, and pasteable. You wanted understanding, not an automatic edit. The same goes for a confusing **stack trace** — the list of function calls leading to a crash — which you can paste in full and have walked through line by line.

## The drawbacks: friction and missing context

The very thing that makes the chat app simple also limits it. Because it lives outside your editor, working with real code through it is full of friction.

**Copy-paste friction.** Every piece of code moves by hand. You copy a function out of your editor, paste it into the chat, read the reply, then copy the model's suggestion back into your editor and fit it into place. For one snippet that's fine. For anything spanning several files it becomes tedious and error-prone — it's easy to paste the wrong version or lose a change.

**The model can't see your repo.** The chat app knows only what's in the conversation. It cannot open your other files, follow an import to see how a function is actually defined, or check the rest of your codebase for how a pattern is used elsewhere. If a fix depends on code you didn't paste, the model is guessing. It may invent a function that doesn't exist or assume a structure your project doesn't have — a reminder that models predict plausible text and don't have ground truth about your project (we'll return to this in [Verification and trust](/learn/ai-software-dev/verification-and-trust/)).

**It's easy to lose track of context.** In a long conversation, details scroll away. The model still technically has the earlier turns in its [context window](/learn/ai-software-dev/context-windows/), but *you* may forget what version of a file you pasted ten messages ago, or which suggestion you already tried. Threads drift, and you end up re-pasting things to be sure.

**Nothing is applied automatically.** This is the core distinction from the tools in the next lessons. The chat app produces text. It never touches your files, runs your tests, or makes a change you can review as a diff. Every result is something *you* must transcribe and integrate. The model is a knowledgeable advisor sitting in another room, not a collaborator with its hands on your keyboard.

## When it's the right tool

None of those drawbacks make the chat app bad — they make it *specialized*. It is the right tool when:

- You want to **understand** something, not change a file.
- The relevant code or error is **small enough to paste** comfortably.
- You're **planning** an approach before writing anything.
- You need a **self-contained snippet** you'll integrate yourself.
- You're away from your dev machine and just want to think a problem through on your phone.

It is the *wrong* tool when the task is "make this change across my project," because then the friction and the missing repo context dominate. That's exactly the gap the next two lessons fill: tools that live inside your editor, and tools that operate on your whole project.

A practical habit many developers settle into: keep the chat app open as a thinking-and-learning space alongside whatever editor tooling they use for actual edits. The two complement each other rather than compete.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the chat app only sees what's in the conversation, so it can't read the rest of your repo on its own." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the main limitation of using a provider's chat app for hands-on coding?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It requires a complicated installation before you can use it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It can't see your repository or apply changes — you copy code in and out by hand</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can't explain error messages or stack traces</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The provider's app** — the chat interface (ChatGPT, Claude.ai, the Gemini app) in your browser, desktop, or phone; the simplest way to use AI.
- **Zero setup** — nothing to install or configure, and usually multimodal, so you can paste a screenshot or error directly.
- **Great for thinking** — best at questions, learning, planning, isolated snippets, and explaining errors and stack traces.
- **Missing repo context** — it only sees what you paste; it can't read the rest of your codebase or verify what's actually there.
- **No automatic changes** — it produces text you must copy back and integrate yourself, which adds friction for multi-file work.
- **Right tool, right job** — reach for it to understand and plan; reach for editor or agentic tools when you need to actually change code.

Next up: bringing the AI *into* your editor, so it can see your open files and you stop copy-pasting. See [AI inside your editor](/learn/ai-software-dev/ide-integration/).
