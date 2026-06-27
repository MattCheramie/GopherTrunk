---
slug: algebraic-attack
title: Algebraic attack
entry_type: term
category: algorithms
description: An algebraic attack models a cipher as a system of equations relating known data to unknown key or state, then solves that system — over a finite field or modular ring — to recover the secret.
keywords: algebraic attack, system of equations, finite field, modular arithmetic, Gaussian elimination, GF(2), linear cryptanalysis, modular inverse
aka: [algebraic attack, algebraic cryptanalysis]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic technique }
  - { label: Method, value: Express as equations, then solve }
  - { label: Domains, value: "GF(2), ℤ/256, ℤ/2ⁿ, GF(2⁸)" }
  - { label: Tools, value: "Gaussian elimination, modular inverse" }
see_also: [sat-smt-solving, brute-force-attack, known-plaintext-attack, convolutional-code, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/Algebraic_attack
  - https://en.wikipedia.org/wiki/Linear_cryptanalysis
---

**An algebraic attack** writes a cipher as a system of equations in which the known plaintext
and ciphertext are coefficients and the key or internal state are the unknowns, then solves
the system directly.[^alg] When the equations are linear over a field or ring, the solve is
exact and fast (Gaussian elimination); a cipher whose update is truly linear falls
immediately, which is one reason real ciphers add nonlinearity.[^lin]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Observed bytes form a matrix equation A x equals y, solved for the unknown vector x." xmlns="http://www.w3.org/2000/svg">
  <text x="26" y="52" font-size="9" fill="currentColor">data →</text>
  <text x="92" y="52" font-family="monospace" font-size="13" fill="currentColor">A·x = y</text>
  <line x1="170" y1="48" x2="246" y2="48" stroke="currentColor" marker-end="url(#alar)"/><text x="208" y="42" text-anchor="middle" font-size="8" fill="currentColor">solve</text>
  <rect x="248" y="34" width="150" height="30" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="323" y="53" text-anchor="middle" font-size="8" fill="currentColor">x = key / state (mod m)</text>
  <defs><marker id="alar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each known byte is one equation; enough equations over-determine the unknowns and the linear algebra returns them — when the system really is linear.</figcaption>
</figure>

## How it works

The analyst posits a parametric form for the update — say `state' = A·state + B + input` — and
turns each observed transition into an equation modulo the cipher's word size (2⁸, 2¹⁶, or a
prime). Gaussian elimination over that ring, using modular inverses where the modulus is not
prime, solves for the constants or proves no solution exists. Nonlinear ciphers resist this:
the system becomes high-degree, and the analyst must either linearize, restrict to a subspace,
or hand the equations to a [SAT/SMT solver](/reference/sat-smt-solving/).

## Relevance to SDR

Algebraic solving is the fast first pass when reverse-engineering an encoder. GopherTrunk's
clean-room analysis of the Motorola P25 talker-alias obfuscation (issue #773) solved modular
linear systems (over ℤ/256 and ℤ/2¹⁶, with modular inverses) to recover the per-character
keystream and to *rule out* every linear and low-degree-polynomial update — the negative
result that proved the cipher's core is genuinely nonlinear.

## Sources

[^alg]: [Algebraic attack](https://en.wikipedia.org/wiki/Algebraic_attack) — Wikipedia, for modeling a cipher as a solvable equation system.
[^lin]: [Linear cryptanalysis](https://en.wikipedia.org/wiki/Linear_cryptanalysis) — Wikipedia, for why linear structure is exploitable and must be avoided in real ciphers.
