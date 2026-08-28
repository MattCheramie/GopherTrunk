---
slug: self-consistent-test-trap
title: Self-consistent test trap
entry_type: concept
category: testing-delivery
description: The self-consistent test trap is a round-trip test whose encoder and decoder share the same wrong constant, table, or convention — so the test passes either way and cannot see the very bug it exists to catch.
keywords: self-consistent test, round-trip test, encode decode test, shared constant bug, test independence, ground truth, reference implementation, synthetic fixtures, protocol testing, false green
aka: [round-trip trap, self-consistent synthetic trap]
autolink: true
infobox:
  - { label: Type, value: Testing anti-pattern }
  - { label: Shape, value: Encoder and decoder share the mistaken assumption }
  - { label: Symptom, value: Green tests, broken against the real world }
  - { label: Antidote, value: An independent source of truth }
see_also: [unit-testing, integration-testing, golden-test-vectors, mocking, testing-dsp-without-hardware, tetra-lock-facts]
related_reading:
  - { title: "From the Issue Tracker, Part 20: The Self-Consistent Trap — Round-Trip Tests That Validate Their Own Bugs", url: /blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software_testing
  - https://en.wikipedia.org/wiki/Test_oracle
---

**The self-consistent test trap** is the failure mode of round-trip testing: encode
something, decode it back, assert you got the original — and never notice that the encoder
and decoder *share* the wrong constant, the wrong table, or the wrong convention, so the
mistake cancels itself out.[^oracle] The test is real, the assertions are strict, the suite
is green, and the code is wrong in a way this test can never see, because a round trip is
blind to any error it makes twice. It is a test-oracle problem in disguise: the decoder is
being graded by an oracle that inherited its own misconceptions.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A round-trip test where data flows from an encoder to a decoder and back to an assertion that passes, while both encoder and decoder draw from the same shared wrong constant underneath." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="30" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="90" y="49" font-size="9" fill="currentColor" text-anchor="middle">encoder</text>
  <rect x="200" y="30" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="250" y="49" font-size="9" fill="currentColor" text-anchor="middle">decoder</text>
  <rect x="360" y="30" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="400" y="49" font-size="9" fill="currentColor" text-anchor="middle">PASS ✓</text>
  <line x1="140" y1="45" x2="196" y2="45" stroke="currentColor" stroke-width="1.2" marker-end="url(#scar)"/>
  <line x1="300" y1="45" x2="356" y2="45" stroke="currentColor" stroke-width="1.2" marker-end="url(#scar)"/>
  <rect x="130" y="100" width="180" height="28" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
  <text x="220" y="118" font-size="9" fill="currentColor" text-anchor="middle">shared wrong constant</text>
  <line x1="105" y1="62" x2="160" y2="98" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <line x1="335" y1="62" x2="280" y2="98" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The assertion checks that decode inverts encode — which stays true no matter what the shared constant says. The bug lives below the round trip, where no assertion looks.</figcaption>
</figure>

## How it bites

The trap recurs wherever a codebase implements both directions of a transform, and
GopherTrunk's issue tracker documents it in several independent dresses:

- **A skipped step, skipped twice.** The TETRA direct-mode voice path skipped descrambling
  at colour code 0 — wrong, since colour-0 scrambling is non-identity — but the synthetic
  tests scrambled *and* descrambled consistently at colour 0, so they passed either way and
  real on-air traffic decoded at the chance floor, misread as encryption (#1003).
- **A wrong constant, mirrored in the fake.** A SoapyRemote RPC call used opcode 600 where
  the protocol says 501 — and the fake test server switched on *the same constant*, so both
  sides moved together and every unit test passed while real hardware silently ignored the
  call.
- **A wrong table, validated against itself.** Placeholder TETRA sync-training constants
  correlated perfectly against signals synthesized *from those constants*, and never once
  against real air (#553).
- **An estimator grading its own homework.** A blind channel estimate contaminated by
  interference "verified" an interference-rejection combiner on synthetic scenes built from
  the same estimate's assumptions.

The common thread: the test's ground truth was manufactured by the code under test, so the
test measured self-consistency — a property even wrong code has.

## Escaping it

The antidote is always the same in structure — **inject an independent source of truth** —
and comes in escalating strengths:

1. **Independent references.** Decode fixtures produced by a different implementation
   (a reference codec, another project's decoder, upstream protocol literals), or check
   constants against the spec's numbers rather than your own header. This is
   [golden test vectors](/reference/golden-test-vectors/) with the emphasis on *whose gold*.
2. **Asymmetric fixtures.** Make the encode side unconditional where reality is
   unconditional — GopherTrunk's regression now scrambles on encode *always*, so a decoder
   that skips descrambling fails the test the way it fails the air.
3. **Consumption asserts.** A fake server that requires every request byte to be consumed
   catches argument-shape drift a switch-on-shared-constants fake cannot.
4. **Reality gates.** For signal-processing code, no synthetic pass closes the loop: the
   change is not verified until it works against a capture of the real phenomenon. A green
   round trip is a necessary condition, never a sufficient one.

## In the bigger picture

Round-trip tests remain excellent — fast, deterministic, great at catching *asymmetric*
regressions — and none of this argues for deleting them. The discipline is knowing what
they cannot certify: any property both directions share. [Unit tests](/reference/unit-testing/)
isolate code from its collaborators; this trap is what happens when a test fails to isolate
code from its *assumptions*. Budget at least one test per transform whose expected values
were produced by something that is not your code.

## Sources

[^oracle]: [Test oracle](https://en.wikipedia.org/wiki/Test_oracle) — Wikipedia, on the problem of obtaining correct expected outputs independently of the system under test.
