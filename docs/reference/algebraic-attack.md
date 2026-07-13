---
slug: algebraic-attack
title: Algebraic attack
entry_type: term
category: cryptography
description: An algebraic attack models a cipher as a system of equations relating known data to unknown key or state, then solves that system — over a finite field or modular ring — to recover the secret.
keywords: algebraic attack, system of equations, finite field, modular arithmetic, Gaussian elimination, Grobner basis, linearization, GF(2), linear cryptanalysis, LFSR, modular inverse
aka: [algebraic attack, algebraic cryptanalysis]
autolink: true
infobox:
  - { label: Type, value: Cryptanalytic technique }
  - { label: Method, value: Express as equations, then solve }
  - { label: Domains, value: "GF(2), ℤ/256, ℤ/2ⁿ, GF(2⁸)" }
  - { label: Tools, value: "Gaussian elimination, modular inverse" }
see_also: [sat-smt-solving, brute-force-attack, known-plaintext-attack, convolutional-code, tetra-tea]
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
prime, solves for the constants or proves no solution exists. Over GF(2) the same idea handles
bit-level ciphers: each output bit becomes a Boolean polynomial in the key bits. Nonlinear
ciphers resist this by pushing the polynomial degree up, so the analyst must linearize,
restrict to a subspace, or reach for heavier machinery.

## Variants

When the equations are nonlinear the toolbox grows. **Linearization** and its **XL/XSL**
extensions introduce a fresh variable for each nonlinear monomial so the enlarged system looks
linear, then solve it if enough independent equations exist. **Gröbner-basis** methods (Buchberger,
F4/F5) manipulate the polynomial ideal directly to eliminate variables systematically. For
stream ciphers built on [linear-feedback shift registers](/reference/linear-feedback-shift-register/),
**correlation** and **algebraic-immunity** attacks exploit low-degree relations between the
keystream and the register state. When the algebra becomes intractable by hand, the equations
are usually handed to a [SAT/SMT solver](/reference/sat-smt-solving/), which searches for a
satisfying assignment instead of solving symbolically.

## In practice

The attack's leverage is that a small algebraic weakness scales badly for the defender: one
exploitable linear relation among register bits can leak the whole state. This is why
register-based radio ciphers are scrutinised for algebraic immunity, and why the reduced-strength
[TETRA TEA1](/reference/tetra-tea/) drew attention — a compact keyed register is exactly the
kind of structure algebraic methods probe. Conversely, a *negative* algebraic result is
valuable evidence: showing that no linear or low-degree model fits the data proves the target's
core is genuinely nonlinear and steers the analysis toward search-based methods.

## Relevance to SDR

Algebraic solving is the fast first pass when reverse-engineering an encoder. GopherTrunk's
clean-room analysis of the Motorola P25 talker-alias obfuscation (issue #773) solved modular
linear systems (over ℤ/256 and ℤ/2¹⁶, with modular inverses) to recover the per-character
keystream and to *rule out* every linear and low-degree-polynomial update — the negative
result that proved the cipher's core is genuinely nonlinear.

## Sources

[^alg]: [Algebraic attack](https://en.wikipedia.org/wiki/Algebraic_attack) — Wikipedia, for modeling a cipher as a solvable equation system.
[^lin]: [Linear cryptanalysis](https://en.wikipedia.org/wiki/Linear_cryptanalysis) — Wikipedia, for why linear structure is exploitable and must be avoided in real ciphers.
