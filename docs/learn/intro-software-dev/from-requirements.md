---
slug: from-requirements
title: Start with requirements
description: Drive a language choice from real requirements — performance, safety, real-time limits, platform, ecosystem, team and timeline — by turning fuzzy needs into concrete, testable criteria.
keywords: software requirements, language selection criteria, performance targets, real-time constraints, deployment targets, ecosystem, team skills, non-functional requirements
level: intermediate
status: full
faq:
  - q: "Why start a language choice from requirements rather than preferences?"
    a: "Because requirements are the only thing that can actually eliminate options. A real-time latency budget, a target platform, or a licensing constraint rules languages in or out on facts, not taste. Starting from preference tends to rationalise a choice you already made for fashion or familiarity."
  - q: "What is the difference between a functional and a non-functional requirement?"
    a: "A **functional** requirement is what the software must *do* (\"decode P25 trunked audio\"). A **non-functional** requirement is *how well* it must do it — speed, latency, reliability, resource limits, portability. Non-functional requirements are usually what drive a language choice, because they touch the performance ceiling and platform."
  - q: "How do I turn a vague need into a usable criterion?"
    a: "Attach a number or a hard fact. \"Fast\" becomes \"process a 2.4 MS/s stream with under 50 ms latency.\" \"Runs everywhere\" becomes \"ships as one binary for Linux, macOS and Windows on x86-64 and ARM.\" Concrete criteria can be tested against a candidate; vague ones can't."
---
# Start with requirements

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Requirements drive the choice** — performance, safety, real-time, platform, ecosystem, team and timeline, not fashion. **Make them concrete** — turn "fast" into a number. **Hard constraints eliminate** — some needs rule whole languages out instantly.
</div>

The previous lesson argued there is no "best" language, only the best fit for a
job. But "fit" is empty until you know what you're fitting *to*. That's the
requirements. The discipline here is simple to state and easy to skip: write down
what the software actually has to do and how well, *then* see which languages
clear the bar. Done well, this turns an argument about taste into a short list you
can defend.

## The requirements that move the needle

Not every requirement affects language choice. Many — the colour of a button, the
exact wording of an error — are language-agnostic. These are the categories that
genuinely steer the decision:

- **Performance targets.** Throughput and latency. "Handle 10k requests/sec" or
  "filter each sample buffer in under 5 ms." High enough targets push you toward
  compiled languages.
- **Correctness and safety needs.** How costly is a bug? Avionics, medical and
  financial code value languages that catch errors early — strong static types,
  memory safety. A throwaway script does not.
- **Real-time constraints.** A *deadline per operation*, not just average speed.
  Audio, control loops and radio DSP must finish each cycle on time, every time —
  this distrusts unpredictable pauses.
- **Platform and deployment targets.** Where does it run? A microcontroller, a phone,
  a browser, a Raspberry Pi, a user's laptop? The target can eliminate languages that
  don't run there at all.
- **Ecosystem and libraries.** Does a mature library already solve the hard part?
  Reaching for Python's NumPy or GNU Radio can outweigh a language's raw speed.
- **Team skills.** What can your people build well *now*, and what can they learn in
  the time you have?
- **Timeline.** A two-week prototype and a ten-year platform reward very different
  trade-offs between speed-to-write and long-term rigour.
- **Licensing and legal.** Some libraries, runtimes or compilers carry licences (GPL,
  commercial) that conflict with how you intend to ship. Check before you commit.

## Functional vs non-functional

A useful split: **functional** requirements are what the system does;
**non-functional** requirements are how well it does it.

| | Functional | Non-functional |
|---|---|---|
| Question | What must it *do*? | How *well*? |
| Examples | "Decode trunked audio", "export CSV" | latency, throughput, memory, portability, safety |
| Effect on language | usually neutral | often decisive |

Most languages can be made to *do* almost anything. It's the non-functional column —
the latency budget, the memory ceiling, the deployment story — that pushes you
toward or away from a language. So weight those heavily when choosing.

## Turning fuzzy needs into concrete criteria

"It needs to be fast and run everywhere" is not a requirement; it's a wish. The job
is to attach numbers and hard facts so a candidate language can be *tested* against
it.

| Fuzzy wish | Concrete criterion |
|---|---|
| "Fast" | "Process a 2.4 MS/s I/Q stream with end-to-end latency under 50 ms" |
| "Runs everywhere" | "Ship one self-contained binary for Linux, macOS and Windows, x86-64 and ARM" |
| "Reliable" | "No crashes over a 72-hour soak test; recover from a dropped USB device" |
| "Easy for users" | "Non-technical user downloads one file and double-clicks — no runtime install" |
| "Handles lots of users" | "Sustain 5,000 concurrent connections on a 2-core VM" |

Concrete criteria do two things: they let you *eliminate* options on facts, and they
become the acceptance tests you prototype against later (see
[the decision framework](/learn/intro-software-dev/decision-framework/)).

## How a hard constraint eliminates options

Some requirements are pass/fail. They don't *favour* a language — they *forbid*
others. These are the most valuable requirements you can find, because they shrink
the field fast.

Consider: **"must process a live sample stream in real time on a Raspberry Pi."**
Walk through what that single sentence rules out:

- **Real time** distrusts long, unpredictable garbage-collection pauses, so a
  heavily GC'd, JIT-warming runtime is a worry for the tightest inner loop.
- **Live stream, on a Pi** means a CPU and memory budget — a slow interpreted inner
  loop may simply not keep up, dropping samples.
- The combination steers the *DSP core* toward C, C++, Rust or carefully-written Go,
  and away from pure CPython in the hot path.

Notice it doesn't ban Python from the project — only from the inner loop. You might
still prototype in Python and push the heavy lifting into native code. Hard
constraints scope *which part* of the system each language can own. The related
timing reasoning lives in
[compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/) and
[memory management](/learn/intro-software-dev/memory-management/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a concrete criterion attaches a number or hard fact so candidate languages can be tested against it." markdown="0">
  <p class="knowledge-check__q">Quick check: which of these is a *concrete*, testable requirement?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">"The app should feel snappy and modern"</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">"Process a 2.4 MS/s stream with under 50 ms end-to-end latency"</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">"Use whichever language is most popular right now"</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Watch for fashion in disguise

Even disciplined teams smuggle preferences into "requirements." Guard against it:

- **"We need it to scale"** — to *what* number, by when? Often the honest answer is a
  few hundred users, which almost anything handles.
- **"It must be enterprise-grade"** — define the actual reliability and support needs;
  the phrase alone justifies nothing.
- **"X is the modern way"** — modern for whom? Momentum is a real signal for hiring
  and ecosystem, but it is not a functional requirement.

When a requirement can't be tested or counted, treat it as a *preference* and weight
it as such in the next lessons — honestly, not disguised as a hard constraint.

## Recap

- **Requirements come before language** — you can't judge fit without knowing what
  you're fitting to.
- **Some categories actually move the needle** — performance, safety, real-time,
  platform, ecosystem, team, timeline and licensing.
- **Non-functional requirements usually decide it** — most languages can *do* the
  job; how *well* is what separates them.
- **Make fuzzy needs concrete** — attach a number or hard fact so candidates can be
  tested.
- **Hard constraints eliminate** — one sentence like "real-time stream on a Pi" can
  rule whole languages out of the hot path.

Next up: the single biggest trade-off behind every language choice —
[performance versus productivity](/learn/intro-software-dev/performance-vs-productivity/).
