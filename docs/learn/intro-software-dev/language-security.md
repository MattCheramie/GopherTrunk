---
slug: language-security
title: Language-level security
description: Why most severe CVEs trace to memory bugs in C/C++, what undefined behaviour and bounds checking mean, sandboxing, and how the software supply chain shapes attack surface.
keywords: memory safety, software security, undefined behaviour, bounds checking, buffer overflow, software supply chain, typosquatting, SBOM
level: intermediate
status: full
faq:
  - q: "Why are memory-safe languages considered more secure?"
    a: "A large share of severe security vulnerabilities come from **memory bugs** — buffer overflows, use-after-free — in memory-unsafe languages like C and C++. Memory-safe languages (Rust, Go, Java, Python) prevent these by construction, so choosing one removes an entire category of exploitable bugs from your code."
  - q: "What is the software supply chain risk?"
    a: "Modern programs depend on many third-party packages, each of which can introduce vulnerabilities or be malicious. **Supply-chain attacks** include typosquatting (a fake package with a near-identical name), compromised maintainers, and hidden malware. Lockfiles and SBOMs help you know and pin exactly what you depend on."
  - q: "What is undefined behaviour?"
    a: "**Undefined behaviour** is code whose result the language specification does not define — like reading past an array's end in C. The compiler may do anything: crash, return garbage, or silently keep going in an exploitable state. It is a major source of both bugs and security holes in low-level languages."
---
# Language-level security

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Memory safety** — removes the bug class behind most severe CVEs. **Undefined behaviour** — low-level code that can do *anything*. **Supply chain** — your dependencies are part of your attack surface.
</div>

Security is not only a matter of careful coding — it is partly decided the moment
you pick a language. The language's design determines which mistakes are even
*possible*, how much of your safety rests on every programmer being perfect, and
how large your **attack surface** is. This lesson looks at security from that
angle: memory safety, undefined behaviour, the protections languages do (or do
not) give you, and the increasingly important question of the code you did not
write yourself — your dependencies.

## Memory safety and the CVE record

The single biggest language-level security lever is **memory safety**. A
*memory-unsafe* language — chiefly **C and C++** — lets a program read or write
memory it should not: past the end of a buffer, after it has been freed, through a
dangling pointer. These are not just bugs; they are **exploitable**. An attacker
who can overflow a buffer can often overwrite adjacent data or inject code.

The evidence is stark and consistent. Studies from Microsoft and Google have both
reported that around **70% of their severe security vulnerabilities** stem from
memory-safety bugs in C/C++ code. This finding is so well-established that the US
NSA and CISA have publicly urged industry to move toward **memory-safe
languages**.

A **memory-safe language** — Rust, Go, Java, C#, Python, JavaScript — makes those
bugs impossible (or catches them at compile time, in Rust's case). The mechanism
varies: garbage collection, runtime bounds checks, or ownership (see
[memory management](/learn/intro-software-dev/memory-management/)). The
consequence is the same: choosing a memory-safe language *deletes the most common
category of severe vulnerability from your program* before you write a line.

## Undefined behaviour and bounds checking

Two related concepts explain *why* unsafe languages are so risky.

**Undefined behaviour (UB)** is code whose outcome the language specification
deliberately does not define. In C, reading past an array's end, signed integer
overflow, or using a freed pointer are all UB. The compiler is free to assume UB
never happens and optimise accordingly — so the program may crash, return garbage,
or, worst of all, *keep running in a corrupted, attacker-controllable state*. UB is
why C bugs are so unpredictable.

**Bounds checking** is the main defence. Before accessing element *i* of an array,
a bounds-checked language verifies *i* is in range and fails safely (an exception
or panic) if not.

```python
data = [1, 2, 3]
data[7]   # Python: IndexError — safe, predictable failure
```

```c
int data[3];
data[7];   // C: undefined behaviour — reads whatever is there, no check
```

Memory-safe languages bounds-check by default. The tiny per-access cost buys you a
predictable crash instead of a silent, exploitable overflow — almost always the
right trade outside the hottest inner loops.

## Sandboxing and isolation

Beyond the language core, **sandboxing** limits what running code can do even if it
is buggy or hostile. The idea is *least privilege*: give code only the capabilities
it needs.

- **WebAssembly** runs untrusted code in a tightly confined sandbox with no ambient
  access to the host system.
- **Browsers** isolate each tab and script in a restricted environment.
- **Containers and OS sandboxes** (seccomp, capabilities) restrict syscalls and file
  access.

Sandboxing is defence in depth: it does not fix the bug, but it shrinks the *blast
radius* when one is exploited. Combining a memory-safe language with a sandbox
gives you two independent layers of protection.

<div class="knowledge-check" data-quiz data-correct-msg="Right — industry studies (Microsoft, Google) consistently attribute roughly 70% of severe CVEs to memory-safety bugs in C/C++." markdown="0">
  <p class="knowledge-check__q">Quick check: where do the majority of severe security vulnerabilities tend to come from?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Weak passwords chosen by end users</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Memory-safety bugs in memory-unsafe languages like C and C++</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Programs written in interpreted languages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## The software supply chain

Most of the code in a modern program is not yours — it is **dependencies**: open-source
libraries pulled in through a package manager, each with its own dependencies, often
hundreds deep. Every one of those is part of your **attack surface**, and attackers
have noticed. **Supply-chain attacks** are now a leading threat:

- **Typosquatting** — a malicious package with a name one keystroke from a popular
  one (`reqeusts` instead of `requests`), hoping you mistype it.
- **Compromised maintainers** — an attacker takes over a legitimate package and ships
  malware in an update to its existing users.
- **Dependency confusion** — tricking a build into pulling a public package instead of
  an intended internal one.

Several practices reduce the risk:

- **Lockfiles** (`package-lock.json`, `Cargo.lock`, `go.sum`) pin the *exact* version
  and cryptographic hash of every dependency, so builds are reproducible and a
  swapped package is detected.
- **SBOMs** (Software Bill of Materials) — a machine-readable inventory of everything
  in your build, so when a vulnerability is announced you can instantly tell whether
  you are affected.
- **Minimal dependencies and vetting** — fewer, well-audited packages mean less
  surface to attack.

Language ecosystems differ here. Go's standard library is large enough that many
programs need few external dependencies; the npm ecosystem favours many tiny
packages, widening the surface. This is a real factor when
[choosing a language for a domain](/learn/intro-software-dev/language-for-the-domain/).

## Security is more than the language

Choosing a memory-safe language and pinning your dependencies are powerful, but
they are not the whole story — they remove *categories* of bugs, not all bugs. A
few classes of vulnerability are largely language-independent and still demand
care:

- **Injection** — SQL injection, command injection and cross-site scripting all
  come from mixing untrusted data into a command or query. The fix is the same in
  every language: never build commands by string concatenation; use parameterised
  queries and proper escaping.
- **Logic and authorisation flaws** — granting access you should not, trusting a
  client-supplied value, leaking secrets in logs. No type system catches a missing
  permission check.
- **Cryptographic mistakes** — rolling your own crypto, using weak algorithms, or
  mishandling keys. Use vetted libraries, never invent your own.
- **Misconfiguration** — exposed debug endpoints, default credentials, overly broad
  permissions.

The right mental model is **defence in depth**: a memory-safe language removes the
most common severe category, bounds checking and sandboxing add layers, careful
input handling closes the injection class, and good operational hygiene covers the
rest. Language choice raises the floor of your security; it does not let you stop
thinking. Treat untrusted input as hostile, keep dependencies few and pinned, and
assume any one layer can fail.

## Why this matters when parsing the air

Bring it back to radio. A tool like **GopherTrunk** spends much of its time
**parsing untrusted input**: demodulated frames pulled out of the air, from
transmitters you do not control, possibly malformed, possibly crafted. Parsing
attacker-influenced byte streams is *exactly* the scenario where a memory-safety
bug becomes a remote exploit — a malformed frame that overruns a fixed buffer in C
is a textbook attack vector.

Writing that frame-parsing code in a **memory-safe language** means a malformed
packet produces a safe error — a bounds-check panic, a rejected frame — instead of
a corrupted, attacker-controlled process. You cannot control what arrives over the
air, but you *can* control whether your parser turns a bad frame into a crash or a
compromise. Choosing memory safety for the code that touches untrusted radio data
is one of the highest-leverage security decisions you will make.

## Recap

- **Memory safety is the biggest lever** — roughly 70% of severe CVEs trace to memory
  bugs in C/C++; safe languages delete that bug class.
- **Undefined behaviour** — unspecified outcomes (like out-of-bounds reads) that
  compilers may turn into silent, exploitable corruption.
- **Bounds checking** — memory-safe languages fail predictably instead of reading
  past an array.
- **Sandboxing** — least-privilege isolation (WebAssembly, browsers, containers)
  limits the blast radius of any bug.
- **Supply chain** — dependencies are attack surface; typosquatting and compromised
  maintainers are real, mitigated by lockfiles and SBOMs.
- **Memory safety protects untrusted-input parsers** — like code reading demodulated
  frames off the air.

Next up: Module 3 turns from language internals to craft, starting with what makes
code readable and maintainable — [clean code](/learn/intro-software-dev/clean-code/).
