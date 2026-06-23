---
slug: choosing-your-method
title: Choosing your method
description: How you reach an AI model — chat app, IDE integration, agent, or a blend — should match how you actually work and your task mix. Build a personal setup, weigh learning curve, cost, and switching friction, and see worked setups for different kinds of developers.
keywords: choosing AI coding method, app vs IDE vs agent, personal AI setup, AI workflow, blended workflow, code completion, chat assisted coding, agentic coding, learning curve, switching friction, developer workflow
level: intermediate
prereq:
  - app-vs-ide-vs-both
  - choosing-model-and-provider
faq:
  - q: "Do I have to choose one method, or can I mix them?"
    a: "Mix them. The strongest setups **blend** methods: the chat app for learning and planning, IDE completion and chat for daily coding, and an agent for big multi-file changes. The goal isn't loyalty to one tool — it's reaching fluidly for whichever fits the task in front of you."
  - q: "I'm brand new to coding — where should I start?"
    a: "Start with the **chat app** or simple IDE chat. They're the gentlest on-ramp: low cost, easy to undo, and you read every line before it goes anywhere. Add code completion once that feels natural, and only reach for agents after you're comfortable reviewing changes you didn't type yourself."
  - q: "How much should I worry about switching tools later?"
    a: "A little, not a lot. Favour methods and config that travel well — portable prompts and [config files](/learn/ai-software-dev/skills-and-config-files/), tools that work across providers — so you're not locked in. Then treat your setup as something you adjust as you learn, not a permanent commitment."
---
# Choosing your method

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Match the method to how you work** — app, IDE, agent, or a blend, chosen by your task mix, not by fashion. **Build a personal setup** — e.g. the app for learning and planning, IDE completion and chat for daily coding, an agent for big multi-file changes. **Weigh the frictions** — learning curve, cost, and how hard it is to switch later.
</div>

This is lesson 24 of the path. The last lesson picked your *model*; this one picks *how you reach it*. Back in Module 3 you met the three ways to access a model — the [provider's chat app](/learn/ai-software-dev/the-provider-app/), [AI inside your editor](/learn/ai-software-dev/ide-integration/), and [agentic command-line tools](/learn/ai-software-dev/agentic-cli-tools/) — and [App, IDE, agent — or all three?](/learn/ai-software-dev/app-vs-ide-vs-both/) showed that the answer is usually "all three, by task." This lesson turns that map into *your* route: a concrete, personal setup. By the end you'll be able to assemble one that fits how you actually work, and know which frictions to weigh as you do.

## A quick recap of the three methods

The three methods aren't ranked worst-to-best; they're shaped for different jobs. A short refresher:

| Method | What it is | Where it shines |
|---|---|---|
| **Chat app** | A conversation with the model in a browser or desktop app | Learning, planning, explaining errors, exploring an idea before you touch code |
| **IDE integration** | The model built into your editor — [completion](/learn/ai-software-dev/code-completion/) as you type plus in-editor [chat](/learn/ai-software-dev/chat-assisted-coding/) | Everyday coding in flow: finishing functions, local refactors, asking about the open file |
| **Agent** | A tool that reads your repo, edits files, and runs commands in a loop | Big, well-scoped, multi-file changes; iterate-until-green tasks |
| **A blend** | Reaching for whichever of the above fits the moment | Real day-to-day work, which mixes all of the above |

Your method is really a *combination* of these, weighted by what your days look like.

## Match the method to how you actually work

The mistake mirrors the one from the last lesson: choosing by reputation instead of fit. The fix is the same — start from your reality. Ask three things:

- **What's my task mix?** If most of your day is steady coding in an editor, IDE integration should be the centre of gravity. If it's a lot of "help me understand this" and planning, the chat app earns more weight. If you frequently make large, sweeping changes, an agent pays off.
- **How do I like to work?** Some people think best in conversation; others want the AI invisible until they ask. Some are happy delegating and reviewing a diff; others want to type most lines themselves. There's no virtue in a setup that fights your instincts.
- **What are my constraints?** Budget, and especially privacy. A [privacy-restricted](/learn/ai-software-dev/security-privacy-ethics/) codebase narrows the field before preference does — an agent that reads your whole repo and sends it to a provider may simply be off the table.

The output of those questions isn't a single tool. It's a *setup*: which method is your default, which you reach for in specific situations, and what you deliberately leave out.

## Build a personal setup

A good setup names a primary method and a couple of supporting ones. A widely useful default looks like this:

- **Chat app for learning and planning.** Before writing code, talk through the approach, learn an unfamiliar API, or get an error explained. No files at risk — just understanding.
- **IDE completion + chat for daily coding.** The bulk of the work: let [completion](/learn/ai-software-dev/code-completion/) carry routine typing and use in-editor [chat](/learn/ai-software-dev/chat-assisted-coding/) for questions about the file you're in, staying in control of each change.
- **An agent for big multi-file changes.** When a task spans files and benefits from running tests in a loop, hand it to an [agent](/learn/ai-software-dev/agentic-and-vibe-coding/) — then review its diff carefully before keeping it.

That's a starting point, not a rule. The real skill is reaching fluidly between methods: plan in the app, code in the editor, delegate the big change to an agent. Where you sit on the [delegation spectrum](/learn/ai-software-dev/the-spectrum/) — how much you hand over versus drive yourself — shapes how much weight the agent gets in your blend.

## Worked setups for different developers

The same toolbox produces very different setups depending on who's holding it.

**A beginner learning to code.** Goals: understand, not just produce. Friction to avoid: being handed code you can't read. A fitting setup leans on the **chat app and simple IDE chat**, used to explain and to plan, with completion turned on lightly so it assists rather than races ahead. Agents wait until reviewing unfamiliar diffs feels comfortable. The whole point is to keep your judgement engaged — the AI is a tutor and a pair, not an autopilot.

**A busy professional.** Goals: ship reliable work fast. A fitting setup makes the **IDE the centre of gravity** — completion and chat all day for flow — with an **agent** for large, well-scoped changes (a cross-file refactor, a batch of failing tests) and the **chat app** kept open for planning and gnarly design questions. Capable models earn their cost here because time saved is the budget. The discipline that keeps it safe is [verification](/learn/ai-software-dev/verification-and-trust/): speed without review is how subtle bugs ship.

**Someone on a privacy-restricted codebase.** Goals: get help without code leaving the machine. Constraints decide before preferences do. A fitting setup pairs a **local or open-weight model** (chosen in [Choosing your model & provider](/learn/ai-software-dev/choosing-model-and-provider/)) with whichever methods can run against it — often IDE integration and an agent pointed at the local model — and treats any hosted chat app as for *non-code* questions only. The method follows the privacy gate, not the other way around.

A concrete pass through [GopherTrunk](/learn/intro-software-dev/what-is-software/): a contributor might plan a new decoder mode in the chat app, write it in the editor with completion and chat, and hand "wire it in across the packages and make `make vet test` pass" to an agent — exactly the blend above, applied to one real project.

## Weigh learning curve, cost, and switching friction

Three practical frictions should temper your setup.

**Learning curve.** Each method has one, and they're not equal. The chat app is nearly free to learn; IDE integration takes a little setup and habit-forming; agents demand the most, because using them safely means learning to scope tasks and review diffs. Adopt methods in order of comfort — you don't have to take on all three at once.

**Cost.** Cost roughly tracks how much the method does (see [Understanding cost](/learn/ai-software-dev/understanding-cost/)). A pasted snippet in the app is cheap; an agent looping over your repo and re-running tests consumes far more. Let your default method be the cheap one and reserve the expensive method for tasks that justify it.

**Switching friction.** Tools and models will change. Keep your setup portable: prefer [config files and prompts](/learn/ai-software-dev/skills-and-config-files/) that travel between tools, and avoid leaning so hard on one product's quirks that you can't leave. As the last lesson put it for models, the same holds for methods — pick what fits now, and make it cheap to change later.

| Friction | Cheapest method | Most demanding | What to do |
|---|---|---|---|
| **Learning curve** | Chat app | Agent | Adopt in order of comfort |
| **Cost per task** | Chat app (small snippets) | Agent (repo-wide loops) | Default to cheap, escalate deliberately |
| **Switching later** | Anything with portable config | Tightly coupled, product-specific setups | Keep prompts/config portable |

## Start small and adjust

You don't design the perfect setup up front — you grow one. A reasonable arc is: begin with the chat app, add IDE integration once it feels natural, and bring in an agent when you're comfortable with the guardrails it needs. Then, as your task mix or constraints change, adjust the weights. A setup is a living thing you tune, not a contract you sign.

<div class="knowledge-check" data-quiz data-correct-msg="Right — match the method (or blend of methods) to your task mix and how you work, not to fashion." markdown="0">
  <p class="knowledge-check__q">Quick check: how should you choose between the chat app, IDE integration, and agents?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick the single newest tool and use it for everything</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Match the method — or a blend — to your task mix, how you like to work, and your constraints</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Always use an agent, since it can do the most on its own</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Method follows fit** — choose by your task mix, how you like to work, and your constraints, not by what's fashionable.
- **A setup is a blend** — name a primary method and supporting ones; a common default is app for learning, IDE for daily coding, agent for big changes.
- **People differ** — a beginner leans on chat and gentle completion; a busy pro centres the IDE with an agent on call; a privacy-restricted dev follows the privacy gate first.
- **Weigh three frictions** — learning curve, cost per task, and how hard it is to switch later.
- **Start small, adjust** — grow your setup method by method and tune it as your work changes.

Next up: the discipline that makes any of this safe — reading, testing, and running AI-written code before you trust it. See [Verifying AI-written code](/learn/ai-software-dev/verification-and-trust/).
