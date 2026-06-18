---
slug: sdlc-and-methodologies
title: The software life cycle — Waterfall to Agile
description: The stages of the software development life cycle, plus Waterfall, Agile values, Scrum and Kanban — and an honest take on when process helps and when it's overhead.
keywords: software development life cycle, SDLC, Waterfall, Agile, Scrum, Kanban, sprints, software methodologies, iterative development, WIP limits
level: beginner
status: full
faq:
  - q: "Is Agile always better than Waterfall?"
    a: "No. Agile shines when requirements are uncertain and feedback is valuable — most product software. Waterfall (or something close to it) still fits work where requirements are fixed and verification is expensive or regulated, such as aerospace, medical devices, or a firmware load that physically ships once. The honest answer is to match the process weight to the project's uncertainty and cost of change."
  - q: "What's the difference between Scrum and Kanban?"
    a: "Scrum organizes work into fixed-length sprints with defined roles and ceremonies (planning, standup, review, retrospective) and commits to a batch of work per sprint. Kanban has no sprints — it visualizes a continuous flow of tasks on a board and limits work in progress so the team finishes things before starting new ones. Scrum is cadence-driven; Kanban is flow-driven."
  - q: "Do solo developers need a methodology at all?"
    a: "Not a formal one. The value of heavyweight process is mostly coordinating many people. Solo, you still benefit from the underlying ideas — work in small iterations, keep something working, write down decisions — but you can drop the ceremonies. Use just enough structure to avoid losing track, and no more."
---
# The software life cycle — Waterfall to Agile

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The SDLC** — every project moves through requirements, design, build, test, deploy, maintain. **Waterfall vs Agile** — sequential up-front planning versus iterative feedback. **Process is a dial** — match its weight to the project's uncertainty, not a checklist to obey.
</div>

Writing code is only one slice of building software. Around it sits a whole life cycle — figuring out what to build, designing how, verifying it works, shipping it, and keeping it alive for years. *How* a team moves through that cycle is its **methodology**, and the industry has swung between two poles: plan everything first (Waterfall), or plan a little and learn fast (Agile). This lesson lays out the stages every project passes through, then the methodologies built on top, and stays honest about when heavier process earns its keep and when it's just overhead.

## The stages of the software development life cycle

Whatever the methodology, software passes through the same logical phases. They might be one big sequence or repeated in tight loops, but the activities are constant:

- **Requirements** — figuring out what problem you're solving and what "done" means. The most expensive mistakes are made here: building the wrong thing well.
- **Design** — deciding the architecture, data structures, interfaces, and technologies. How will the pieces fit together?
- **Implementation** — actually writing the code. This is what beginners picture as "software development," but it's often a minority of the total effort.
- **Testing** — verifying the software does what it should and breaks gracefully when it shouldn't. (A whole [testing lesson](/learn/intro-software-dev/testing/) is coming.)
- **Deployment** — getting the software into users' hands: a release, a package, a server rollout.
- **Maintenance** — fixing bugs, adapting to change, and extending the software over its life. For most software this is the *longest and largest* phase by far.

The crucial insight is that the **cost of fixing a mistake rises the later you catch it.** A misunderstood requirement is cheap to fix on a whiteboard, painful to fix after the architecture is built around it, and brutal to fix once it's shipped to thousands of users. Every methodology is really a different bet on how to manage that rising cost.

## Waterfall — one direction, plan first

The **Waterfall** model treats those stages as a strict sequence: finish requirements, then finish design, then implement, then test, then deploy. Each phase "flows" into the next like water down steps, and you don't go back up. It was the dominant model for decades, borrowed from manufacturing and construction where you genuinely can't pour a foundation after framing the walls.

**Where it works:** when requirements are well understood and stable, when the cost of change after release is enormous, or when regulation demands extensive up-front documentation and sign-off. A firmware image burned into a million devices, or avionics software, benefits from heavy front-loaded rigor — you get one shot.

**Where it fails:** most software isn't like that. Requirements are *discovered*, not handed down complete; users don't know what they want until they see something. Waterfall's failure mode is spending months building precisely the wrong thing, because the first time anyone tests against reality is near the end, when changing course is most expensive. The plan looks tidy; the world doesn't cooperate.

## Agile — iterate and respond

The **Agile** movement (codified in the 2001 Agile Manifesto) was a reaction to exactly that failure. Rather than one long sequence, Agile runs the whole life cycle in short, repeated loops, delivering small slices of working software and adjusting based on feedback. Its core values are worth knowing in their original spirit:

- **Working software** over comprehensive documentation — a running slice you can demo beats a thick spec.
- **Individuals and interactions** over rigid processes and tools.
- **Customer collaboration** over contract negotiation.
- **Responding to change** over following a plan — because the plan *will* be wrong, and that's fine.

Agile doesn't mean "no planning" or "no documentation." It means planning continuously in small increments and treating change as expected rather than as failure. The bet is that frequent feedback catches misunderstandings early — while they're still cheap to fix.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Agile prioritizes adapting over rigidly following a plan, because requirements are discovered over time." markdown="0">
  <p class="knowledge-check__q">Quick check: which best captures a core Agile value?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Finish a complete specification before any code is written</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Deliver working software in small increments and respond to change</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Avoid talking to customers until the product is finished</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Scrum and Kanban — two ways to do Agile

Agile is a set of values, not a recipe. Two concrete frameworks dominate practice.

**Scrum** organizes work into fixed-length iterations called **sprints**, usually one to four weeks. The team commits to a chunk of work per sprint and aims to finish a potentially shippable increment by the end. It defines a few roles and ceremonies:

| Element | What it is |
| --- | --- |
| Product Owner | Owns and prioritizes the backlog — decides *what* matters next |
| Scrum Master | Facilitates the process, removes blockers |
| Development team | Builds the increment |
| Sprint planning | Picks what to do this sprint |
| Daily standup | Short sync: progress, plans, blockers |
| Sprint review | Demo the increment to stakeholders |
| Retrospective | Reflect on the *process* and improve it |

**Kanban** drops sprints entirely. Work is visualized as cards moving across a board (To Do → In Progress → Done), and the team sets **WIP (work-in-progress) limits** — a cap on how many items can be in each column at once. When a column is full, you can't start new work; you must finish or unblock something first. This pulls the team toward *flow* and exposes bottlenecks: if cards pile up in "In Review," that's where the team is starved. Kanban suits continuous streams of work — support, operations, or maintenance — where a fixed sprint commitment feels artificial.

Many teams blend the two ("Scrumban") or invent their own variant. The frameworks are tools, not religions.

## How much process is enough?

Here's the honest part. Process exists mostly to **coordinate people and manage risk.** Its value scales with team size, the cost of failure, and the number of stakeholders who must stay aligned. A regulated team of forty shipping medical software needs real ceremony; a two-person side project drowning in standups and burndown charts is paying overhead for coordination it doesn't need.

Consider a small team building a decoder for a new radio protocol in GopherTrunk. The requirements are genuinely fuzzy — nobody fully knows the protocol until they've captured and stared at real signals. Waterfall would be a disaster: you can't spec what you haven't reverse-engineered. A lightweight iterative loop fits perfectly — decode a little, compare against captured samples, learn, repeat. The methodology should *serve* the discovery, not fight it.

The skill isn't picking the "best" methodology — it's reading the situation. High uncertainty and cheap iteration favor Agile. Fixed requirements and expensive change favor more up-front rigor. Most real teams land somewhere in between and tune as they go. Process is a dial, not a switch.

## Recap

- **The SDLC** — requirements, design, implementation, testing, deployment, maintenance; the same stages underlie every methodology.
- **Cost of change rises late** — fixing a mistake gets more expensive the further downstream you catch it, which every methodology tries to manage.
- **Waterfall** — sequential and plan-first; great for stable, high-stakes, regulated work, dangerous when requirements are uncertain.
- **Agile** — iterative loops delivering working software and responding to change; bets that fast feedback catches mistakes early.
- **Scrum vs Kanban** — Scrum is cadence-driven (sprints, roles, ceremonies); Kanban is flow-driven (a board with WIP limits).
- **Process is a dial** — match its weight to team size, uncertainty, and the cost of failure; ceremony for its own sake is overhead.

Next up: the system every project relies on to track history and let people work together — [version control & collaboration](/learn/intro-software-dev/version-control-collab/).
