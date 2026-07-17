---
slug: security-for-developers
title: Security for developers
description: A defender's guide to weaving security through the whole build — shifting left so it's cheaper, threat modeling at design time, a practical developer's checklist tying input validation, auth, secrets, patching, TLS, testing and monitoring together, and automating the guardrails in CI.
keywords: secure development, SDLC, shift left, threat modeling, dependency scanning, SAST, DAST, security checklist, developer security, least privilege, CI security, secure by design
level: intermediate
status: full
prereq:
  - secure-coding
faq:
  - q: What does "shift left" mean for a developer?
    a: It means moving security earlier in the process — into design and coding — instead of leaving it as a final check before launch. A flaw caught while you sketch the design costs minutes to fix; the same flaw caught in production can mean an incident, a rushed patch, and lost data. The earlier you think about how something could go wrong, the cheaper and easier it is to prevent.
  - q: Do I need to be a security expert to build secure software?
    a: No. Security for developers is mostly a set of repeatable habits — validate input, authenticate and authorize properly, manage secrets, patch dependencies, use TLS, test and scan — not deep specialist knowledge. Follow a checklist consistently and automate the checks you can, and you close off the large majority of real-world problems without being an expert.
  - q: What is threat modeling and when should I do it?
    a: Threat modeling is a quick, structured "what are we protecting, from whom, and how could it go wrong?" that you do at design time, before the code exists. It doesn't need special tools — a short conversation and a few notes are enough. Doing it early means you design defenses in rather than discovering the gaps after you have already shipped.
  - q: Can't I just run a security scan at the end and fix what it finds?
    a: A scan at the end catches some issues, but it's the most expensive place to find them and it misses design flaws entirely — a scanner can't tell you that you authorized the wrong user. Build security in from the start and put scans in CI so they run continuously, and the end-of-project scan becomes a confirmation rather than a fire drill.
---

# Security for developers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
For a builder, security isn't a final gate you clear before launch — it's a set
of habits woven through the whole process. **Shift left**: think about how things
break at design time, when fixing them is cheap. Keep **security in the workflow**
so it happens by default, not by heroics. And lean on **a repeatable checklist**
so nothing important gets skipped. This lesson ties the path together into how you
actually work, building on [secure coding](/learn/cybersecurity/secure-coding/).
</div>

You already know how to write safer code line by line. This lesson zooms out to
the whole build: where security fits in the process, how to spot problems before
you write the code, and how to make good habits automatic. The goal is
educational — understand *why* each step earns its place so you reach for it
without being told.

## Shift left

"Shift left" pictures your project as a timeline running left to right — design,
coding, testing, launch. The earlier (further **left**) you consider security,
the cheaper it is to get right.

A weakness spotted while you're sketching the design is a few minutes of
rethinking. The same weakness spotted in testing means reworking code that's
already written. Found in production, it can mean an incident, a scramble to
patch, and real data at risk. The cost climbs steeply the longer it hides.

So don't bolt security on at the end. Bake it into design and coding, the same
[security mindset](/learn/cybersecurity/security-mindset/) you'd bring to any part
of the craft. Security that lives at the start is prevention; security that only
shows up before launch is cleanup.

## Threat model early

Before you write code, spend a few minutes on a quick, structured question:
**what are we protecting, from whom, and how could it go wrong?** That's threat
modeling, and it belongs at design time.

It doesn't need special tools — a short conversation and a few notes will do.
Sketch what data and functions matter, who might want to abuse them, and where an
attacker could push. You're mapping the [threats, vulnerabilities and
risk](/learn/cybersecurity/threats-vulnerabilities-risk/) for *this* system while
the design is still soft enough to change cheaply. The output is a short list of
"here's what we should defend and how," which then drives the choices below.

## A developer's security checklist

Threat modeling tells you *what* matters; a checklist makes sure you actually
cover it. This one ties the whole path together — treat it as the defaults you
apply to every project.

- **Validate input and avoid injection.** Treat everything from outside as
  hostile until checked, and never let data become code — the heart of
  [secure coding](/learn/cybersecurity/secure-coding/) and the defense against
  [web application attacks](/learn/cybersecurity/web-application-attacks/).
- **Authenticate and authorize properly.** Prove who the user is
  ([authentication basics](/learn/cybersecurity/authentication-basics/)) *and*
  check what they're allowed to do
  ([authorization and access](/learn/cybersecurity/authorization-and-access/)) —
  they're two separate steps and skipping either is a hole.
- **Manage secrets.** Keep API keys, passwords, and tokens out of code and out of
  version history — see [secrets
  management](/learn/cybersecurity/secrets-management/).
- **Keep dependencies patched.** Most of what you ship is other people's code;
  update it and watch for known vulnerabilities, including in your
  [Git and repository security](/learn/git/cli-lfs-and-security/) hygiene.
- **Use TLS and secure config.** Encrypt data in transit and don't ship insecure
  defaults — [securing networks &
  services](/learn/cybersecurity/securing-networks-and-services/) and
  [hardening systems](/learn/cybersecurity/hardening-systems/).
- **Test and scan.** Layer in [testing](/learn/intro-software-dev/testing/) plus
  SAST (scans your source), DAST (probes the running app), and dependency
  scanning (flags vulnerable libraries).
- **Least privilege in production.** Give each part of the system only the access
  it truly needs, so a compromise stays small.
- **Log and monitor.** You can't respond to what you can't see — set up
  [monitoring and incident
  response](/learn/cybersecurity/monitoring-and-incident-response/) so problems
  surface.

## Automate the guardrails

A checklist only helps if it runs. The reliable way to make security
non-optional is to move the checks off your memory and into automation.

Put your scans and tests into CI so they run on every change: SAST, dependency
scanning, secret detection, and your test suite all as pipeline steps —
[GitHub Actions](/learn/git/github-actions/) is a common home for them. When a
check fails the build, security stops being something a busy developer can quietly
skip. The machine remembers the checklist so you don't have to.

## Plan for failure

Even with all of the above, assume something eventually gets through — that
humility is itself a security practice. The question isn't only "how do we keep
attackers out?" but "when one gets in, how fast do we notice and recover?"

That's [defense in depth](/learn/cybersecurity/defense-in-depth/): layers, so no
single failure is fatal. Make sure you have logging to reconstruct what happened,
backups you've actually tested restoring, and a written response plan so people
aren't inventing one under pressure. [Monitoring and incident
response](/learn/cybersecurity/monitoring-and-incident-response/) turns "we got
breached" from a catastrophe into an event you handle.

<div class="knowledge-check" data-quiz data-correct-msg="Right — shifting left means building security in from the start, where fixing problems is cheapest." markdown="0">
  <p class="knowledge-check__q">Quick check: "shift left" in security means…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Running one big security scan the day before launch</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Building security in from the start of development instead of bolting it on at the end</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Leaving security to a separate team so developers don't have to think about it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Security is a set of **habits woven through the whole build**, not a final gate.
- **Shift left**: problems are cheapest to fix at design time and most expensive
  in production.
- **Threat model early** — "what are we protecting, from whom, and how could it go
  wrong?" — before the code exists.
- Work the **developer's checklist**: input validation, auth, secrets, patching,
  TLS, testing/scanning, least privilege, and logging.
- **Automate the guardrails** in CI so security isn't optional, and **plan for
  failure** with layers, backups, and a response plan.

Next up: [staying safe online](/learn/cybersecurity/staying-safe-online/)
