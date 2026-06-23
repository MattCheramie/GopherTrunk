---
slug: ide-integration
title: AI inside your editor
description: AI integrated into your editor — inline completion plugins, AI-native editors, and chat side-panels — can see your open files and selection. Learn the advantages of context-aware help, plus the cost, distraction, lock-in, and privacy trade-offs.
keywords: AI editor, GitHub Copilot, Cursor, Windsurf, code completion, inline suggestions, VS Code AI, JetBrains AI, chat side panel, editor integration, context-aware AI, privacy
level: beginner
status: full
prereq:
  - the-provider-app
faq:
  - q: "How is AI in my editor different from the chat app?"
    a: "The chat app only sees what you paste. Editor-integrated AI can see your **open files and your current selection**, so its suggestions are aware of your actual code without copy-pasting. You stay inside your editor instead of switching to a browser tab, which keeps you in flow."
  - q: "Is inline code completion free?"
    a: "Usually not for the better tools — most editor AI is a paid subscription, though many offer a free tier or trial. Pricing changes constantly, so check the tool's current pricing page rather than trusting a number you read somewhere. Some open-weight models can be run locally to avoid a subscription, at the cost of setup effort."
  - q: "Does using AI in my editor send my code to the provider?"
    a: "By default, yes — for cloud-based tools your open files and selection are sent to the provider's servers to generate suggestions. That's a real privacy consideration for proprietary or sensitive code. Some tools offer settings to exclude files, business plans with stricter data handling, or local models; read the data-handling policy before pointing it at confidential work."
---
# AI inside your editor

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Context-aware** — editor-integrated AI can see your open files and selection, so suggestions fit your real code without copy-pasting. **Stay in flow** — completion as you type, refactoring a selection, and chat about the current file all happen without leaving your editor. **Trade-offs** — usually a paid subscription, completions can distract or be subtly wrong, there's lock-in and privacy to weigh, and you still review everything.
</div>

This is lesson 12 of the path. In the [previous lesson](/learn/ai-software-dev/the-provider-app/) we used the provider's chat app and ran into its central limitation: it can't see your code, so you copy-paste constantly. This lesson moves the AI *into* the place you already write code — your editor — which removes most of that friction. By the end you'll understand the three main shapes editor AI takes, why context-awareness is such a leap, and the costs that come with it.

## What "AI inside your editor" means

Your **editor** (or **IDE**, integrated development environment) is the program where you write code — Visual Studio Code, JetBrains tools like IntelliJ or GoLand, Neovim, and others. Putting AI inside it comes in three broad shapes, and many setups combine them:

- **Inline completion plugins.** A plugin that suggests code as you type, appearing as faint "ghost text" you accept with a keystroke. GitHub Copilot popularized this, and there are many peers. It's like autocomplete, but instead of finishing a single word it can finish a whole line, block, or function.
- **AI-native editors.** Editors built from the ground up around AI, such as Cursor and Windsurf. These are typically forks of VS Code with deeper AI features woven through the whole experience rather than bolted on as a plugin.
- **Chat side-panels.** A chat window docked beside your code, inside VS Code, the JetBrains IDEs, and others. It's like the [provider's chat app](/learn/ai-software-dev/the-provider-app/), but it lives in your editor and can read the files you have open.

All three share one thing that changes everything.

## The key advantage: the model can see your code

The defining feature of editor AI is that the model has **context** about what you're working on. Depending on the tool, it can see your **open files**, your **current selection** (the code you've highlighted), nearby files, and sometimes a broader index of the project. You don't paste anything — the editor supplies the relevant code automatically.

That single change unlocks a workflow that the chat app can't match:

| Capability | What it looks like |
|------------|--------------------|
| **Completion as you type** | You start writing a function and the suggestion completes it, matching your naming and style |
| **Refactor a selection** | You highlight a block, ask for a change, and the model rewrites that exact code in place |
| **Chat about the current file** | You ask "what does this function do?" and the panel already knows which file you mean |

Because the suggestions are grounded in your real code, they're far more likely to use *your* function names, *your* types, and *your* conventions instead of inventing plausible-but-wrong ones. And because you never leave the editor, you stay in **flow** — the focused state where you're not constantly switching windows and breaking concentration.

A GopherTrunk example: suppose you're writing a new test in `ddc_test.go` and you type the start of a table-driven Go test. The completion plugin, seeing the `Downconverter` type defined in the file you have open, can suggest a sensible set of test cases using the real method names — something the chat app could only guess at unless you'd pasted the whole type definition first.

## The drawbacks

Moving AI into your editor is powerful, but it comes with real costs you should weigh honestly.

**It usually costs money.** The capable inline-completion tools and AI-native editors are typically a **paid subscription**. Many have a free tier or trial, but daily use generally means paying. As always, treat any specific price as illustrative and check the tool's live pricing page — these numbers change often (see [Understanding cost](/learn/ai-software-dev/understanding-cost/) for how to reason about it).

**Completions can distract or mislead.** Ghost text that pops up constantly can break your concentration as much as help it, especially while you're still thinking. Worse, a suggestion can be **subtly wrong** — syntactically perfect, confidently styled, and quietly incorrect. The model predicts likely-looking code; "likely" is not the same as "correct." A wrong completion that looks right is more dangerous than an obvious error, because it's easy to accept on autopilot.

**Editor lock-in.** AI-native editors and deeply integrated tooling can tie your workflow to one product. If your habits, keybindings, and AI features all live in a specific editor, switching later costs more. This isn't a dealbreaker, but it's worth knowing you're making a choice that's a little sticky.

**Privacy.** For cloud-based tools, your open files and selection are **sent to the provider** to generate suggestions. For open-source or personal projects that may be fine; for proprietary or regulated code it's a genuine concern. Most tools offer ways to exclude files or stricter business-plan data handling, and some support local models, but the default is that your code leaves your machine. We'll go deeper in [Security, privacy, and ethics](/learn/ai-software-dev/security-privacy-ethics/).

**You still must review everything.** Editor AI makes it *easier* to produce code, which makes it easier to produce code you didn't actually read. The reviewing burden doesn't go away just because the suggestion appeared in your editor instead of a chat window. Accepting a completion is committing to that code as if you wrote it.

## When it's the right tool

Editor AI is the right tool when you're **actively writing code** and want help that fits what's already on screen:

- You're in the middle of a file and want completions that match its style.
- You want to refactor or rename within a selection without describing it from scratch.
- You have a question about the specific file you're looking at.
- You value staying in your editor over the back-and-forth of a browser tab.

It's a weaker fit when the task spans many files and needs the model to *run* things — make a change, run the tests, see what failed, and try again. A chat side-panel can discuss your whole project, but it generally won't drive a multi-step loop of acting and checking on its own. That capability is the subject of the next lesson.

Think of editor AI as the sweet spot for *daily coding*: more capable than the chat app because it sees your code, but still firmly under your hand — every suggestion waits for you to accept it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — editor AI can see your open files and selection, so its suggestions fit your actual code without pasting." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the key advantage of AI integrated into your editor over the standalone chat app?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It is always free, with no subscription required</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It can see your open files and selection, so suggestions are context-aware without copy-pasting</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It never produces incorrect suggestions because it reads your code</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **AI inside your editor** — comes as inline completion plugins, AI-native editors, and chat side-panels in VS Code or JetBrains.
- **Context-aware** — the model can see your open files and selection, so suggestions match your real code instead of guessing.
- **Stay in flow** — completion as you type, in-place refactoring, and chat about the current file all happen without leaving the editor.
- **Costs and distractions** — it's usually a paid subscription, and completions can interrupt your thinking or be subtly wrong.
- **Lock-in and privacy** — deep integration can tie you to one editor, and cloud tools send your code to the provider.
- **Still your job to review** — easier code production doesn't remove the duty to read and verify everything you accept.

Next up: tools that don't just suggest — they read your whole repo, run commands, and edit files in a loop. See [Agentic & command-line tools](/learn/ai-software-dev/agentic-cli-tools/).
