---
slug: the-pioneers
title: The people who built programming
description: Short profiles of the people who built programming — Lovelace, Turing, Hopper, von Neumann, Ritchie & Thompson, Kay, and Hamilton — and what each contributed.
keywords: Ada Lovelace, Alan Turing, Grace Hopper, John von Neumann, Dennis Ritchie, Ken Thompson, Alan Kay, Margaret Hamilton, history of programming, computing pioneers
level: beginner
status: full
faq:
  - q: "Who wrote the first computer program?"
    a: "**Ada Lovelace** is widely credited with the first published algorithm intended for a machine — her notes on Babbage's Analytical Engine in the 1840s included a method for computing Bernoulli numbers. She also saw, ahead of her time, that such a machine could manipulate any symbols, not just numbers, anticipating general-purpose computing."
  - q: "Why is Grace Hopper associated with \"debugging\"?"
    a: "**Grace Hopper's** team famously logged a real moth stuck in a relay of the Harvard Mark II, taping it into the logbook as the \"first actual case of bug being found.\" The term predates her, but the story made it iconic. Her far bigger contribution was pioneering the **compiler** and helping create **COBOL**, proving machines could translate human-readable code."
  - q: "What did Margaret Hamilton contribute?"
    a: "**Margaret Hamilton** led the team that wrote the onboard flight software for NASA's Apollo missions, and she championed treating software as a rigorous engineering discipline — she's often credited with popularizing the term **\"software engineering.\"** Her work on error handling and priority scheduling helped the Apollo 11 landing recover from an overload at a critical moment."
---
# The people who built programming

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Ideas have authors** — the abstractions we take for granted were each invented by someone. **Theory and practice** — Turing and von Neumann gave the field its foundations; Hopper, Ritchie, and Hamilton made it usable. **A broad cast** — programming was built by mathematicians, engineers, and visionaries across more than a century.
</div>

The previous lessons followed ideas — the stored program, the transistor, the compiler. This one follows people. Every abstraction you rely on was, at some point, an unproven idea in one person's head. Knowing who they were and what they actually did turns the history from a list of dates into a human story, and it gives you a map of *why* things are the way they are. These are tight profiles, not biographies — just enough to place each pioneer and their lasting contribution. They build on the hardware and language stories from [the previous lesson](/learn/intro-software-dev/birth-of-languages/).

## Ada Lovelace — the first algorithm

**Ada Lovelace** (1815–1852), an English mathematician, worked with Charles Babbage on his proposed Analytical Engine. In her published notes from the 1840s she included a step-by-step method for the machine to compute Bernoulli numbers — widely regarded as the first algorithm written for a computer. More striking still was her insight that the engine could operate on *any* symbols that could be encoded, not just numbers — it might, she mused, even compose music. That leap — seeing a calculating machine as a general symbol-manipulator — anticipated the idea of general-purpose computing by roughly a century.

## Alan Turing — what computing even means

**Alan Turing** (1912–1954), a British mathematician, gave the field its theoretical bedrock. In 1936 he described an abstract device — now called a **Turing machine** — a simple model that reads and writes symbols on a tape according to rules. With it he defined precisely what it means for a problem to be **computable**, and proved that some problems can never be solved by any machine, no matter how powerful. He also showed that a single "universal" machine could simulate any other — the theoretical seed of the general-purpose computer. During World War II he was central to breaking German ciphers at Bletchley Park, work with profound real-world impact.

## Grace Hopper — compilers, COBOL, and the "bug"

**Grace Hopper** (1906–1992), a US Navy rear admiral and mathematician, fought the prevailing belief that programs had to be written in machine-level code. She built one of the first **compilers**, proving a machine could translate readable instructions into executable code, and she was a driving force behind **COBOL**, the English-like business language that still runs banks today. She's also tied to the word **"debugging"**: her team taped a real moth — found jamming a relay in the Harvard Mark II — into their logbook as the "first actual case of bug being found." She spent decades insisting computing should be accessible to ordinary people, not just specialists.

## John von Neumann — the stored-program architecture

**John von Neumann** (1903–1957), a Hungarian-American polymath, lent his name to the design that nearly every computer still follows. His 1945 report describing the **stored-program** computer — instructions and data sharing one memory — became the blueprint for the **von Neumann architecture** we met in lesson one. Whether you store a program or wire it in is *the* dividing line between a fixed machine and a general-purpose computer, and his clear articulation of it shaped the entire industry. He contributed across mathematics, physics, and economics too, but in computing this single idea is foundational.

## Dennis Ritchie & Ken Thompson — C and Unix

At Bell Labs around 1970, **Ken Thompson** and **Dennis Ritchie** built two things that still underpin modern computing. Thompson created the first version of the **Unix** operating system; Ritchie created the **C** programming language and, with Thompson, rebuilt Unix in C — proving an operating system could be written in a portable high-level language rather than assembly. The combination spread everywhere. Today **Linux** (a Unix-like system), the internet's servers, Android phones, and most embedded devices descend from that work. And as the last lesson noted, the DSP and radio code under tools like GopherTrunk is still largely written in Ritchie's C. Few pairs of people have left a wider footprint.

## Alan Kay — objects and personal computing

**Alan Kay** (born 1940) helped invent the way we think about and use computers. At Xerox PARC in the 1970s he led work on **Smalltalk** and helped popularize **object-oriented programming (OOP)** — organizing software around self-contained "objects" that bundle data with the operations on it, a style that came to dominate large-scale software. Kay also championed the vision of a personal, portable computer for everyone (his "Dynabook" concept) and contributed to the graphical, windowed interface that PARC pioneered and the rest of the industry later adopted. Much of how software is *structured* today traces back to his ideas.

## Margaret Hamilton — software engineering as a discipline

**Margaret Hamilton** (born 1936) led the team at MIT that wrote the onboard flight software for NASA's **Apollo** missions. She insisted software be built with the rigor of an engineering discipline — she's often credited with popularizing the term **"software engineering"** itself, at a time when programming was treated as an afterthought to hardware. Her emphasis on robust error handling proved its worth during the Apollo 11 landing: when the guidance computer was overloaded with tasks, her software's priority scheduling shed the less critical work and kept the landing on track. She made the case, decades early, that software's reliability is a matter of disciplined engineering, not luck.

## What this cast has in common

Look across these seven and a pattern emerges. Some were **theorists** (Turing, von Neumann) who defined what was possible; others were **builders** (Hopper, Ritchie, Thompson, Hamilton) who made it practical; and some were **visionaries** (Lovelace, Kay) who saw further than their hardware could reach. Progress needed all three. The abstractions you'll lean on for the rest of this path — compilers, operating systems, objects, reliable software — each began as one of these people's stubborn convictions.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Margaret Hamilton led the Apollo flight software and helped establish 'software engineering' as a discipline." markdown="0">
  <p class="knowledge-check__q">Quick check: Who led the Apollo onboard flight software team and helped popularize the term "software engineering"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Grace Hopper</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Margaret Hamilton</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Ada Lovelace</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Ada Lovelace** — wrote the first algorithm and foresaw general-purpose, symbol-manipulating machines.
- **Alan Turing** — defined computability with the Turing machine and the idea of a universal computer.
- **Grace Hopper** — pioneered the compiler, drove COBOL, and made the "bug" famous.
- **John von Neumann** — articulated the stored-program architecture nearly every computer still uses.
- **Ritchie & Thompson** — created C and Unix, the ancestors of Linux, the web's servers, and most embedded code.
- **Alan Kay & Margaret Hamilton** — gave us object-oriented programming and personal computing, and software engineering as a rigorous discipline.

Next up: from these foundations, computing escaped the lab and reached everyone. We trace mainframes to PCs to the internet to cheap edge devices — and how that flood of compute made software radio practical — in [PCs, the internet & the age of cheap compute](/learn/intro-software-dev/internet-to-edge/).
