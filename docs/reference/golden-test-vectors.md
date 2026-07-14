---
slug: golden-test-vectors
title: Golden test vectors
entry_type: concept
category: sdr-app-building
description: "Golden test vectors are stored known-input/known-output pairs used to pin a DSP or decoder's behavior, so any deviation in a later build is caught as a regression."
keywords: golden test vectors, golden files, known answer test, KAT, reference vectors, regression testing, DSP test fixtures, bit-exact testing, expected output, snapshot testing
aka: ["golden files", "known-answer tests", "reference vectors", "KAT"]
autolink: true
infobox:
  - { label: Type, value: Regression-test fixture }
  - { label: Idea, value: "Freeze a known input and its correct output" }
  - { label: Used in, value: "DSP, codecs, FEC, crypto test suites" }
see_also: [testing-dsp-without-hardware, unit-testing, forward-error-correction, simulation-driven-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Test_vector
  - https://en.wikipedia.org/wiki/Characterization_test
---

**Golden test vectors** are stored pairs of a fixed input and its known-correct
output, used to pin down exactly what a piece of signal-processing or decoding
code is supposed to produce.[^tv] Run the code on the input, compare against the
saved "golden" output, and if they differ the test fails — the change is flagged
as a regression before it can ship. The term borrows from cryptography and codec
standardization, where a *known-answer test* (KAT) proves an implementation
matches the reference bit-for-bit.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A fixed input is run through code under test to produce an actual output, which is compared against a stored golden output; equal means pass, different means a regression." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gtvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="70" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="55" y="69" font-size="8">fixed input</text>
    <line x1="90" y1="65" x2="132" y2="65" stroke="currentColor" stroke-width="1.2" marker-end="url(#gtvar)"/>
    <rect x="134" y="50" width="86" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="177" y="63" font-size="8">code under</text><text x="177" y="74" font-size="8">test</text>
    <line x1="220" y1="65" x2="262" y2="65" stroke="currentColor" stroke-width="1.2" marker-end="url(#gtvar)"/>
    <rect x="264" y="50" width="80" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="304" y="63" font-size="8">actual</text><text x="304" y="74" font-size="8">output</text>
    <rect x="264" y="98" width="80" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="304" y="111" font-size="8">golden</text><text x="304" y="122" font-size="8">output</text>
    <line x1="344" y1="72" x2="392" y2="80" stroke="currentColor" stroke-width="1.1" marker-end="url(#gtvar)"/>
    <line x1="344" y1="112" x2="392" y2="98" stroke="currentColor" stroke-width="1.1" marker-end="url(#gtvar)"/>
    <text x="415" y="83" font-size="8">=?</text><text x="415" y="96" font-size="7">pass/fail</text>
  </g>
</svg>
<figcaption>The saved golden output is the oracle: the test passes only if the freshly computed output matches it.</figcaption>
</figure>

## How it works

You capture the output of a known-good implementation once — the moment you are
confident it is correct — and commit it to the repository as the reference. From
then on the test regenerates the output from the same input and asserts equality
against the stored copy. The input can be a synthetic IQ burst, a short off-air
[capture](/reference/iq-recording-playback/), or an abstract byte array; the
golden side can be decoded bits, demodulated audio, filter coefficients, or a
measured metric such as SNR or EVM.

The comparison comes in two flavors:

- **Bit-exact** — the output must match byte-for-byte. Appropriate for integer
  DSP, [forward-error-correction](/reference/forward-error-correction/) decoders,
  vocoders, and cryptographic primitives, where the standard defines one right
  answer.
- **Tolerance-based** — floating-point results are compared within an epsilon, and
  quality metrics are asserted to stay above a threshold (e.g. "demod SNR ≥ 19 dB
  on this capture"), guarding against silent degradation rather than demanding an
  identical number.

When the code legitimately changes behavior, you *regenerate and review* the
golden files — the diff in the fixture is itself a reviewable record of what the
change did to the output, which is the discipline's real value.

## In practice

Golden vectors live inside an ordinary [unit-test](/reference/unit-testing/) suite
and are the natural companion to
[testing without hardware](/reference/testing-dsp-without-hardware/): the fixed
input is a file, the golden output is a file, and neither needs a radio. The
standards world supplies ready-made vectors — DES and AES ship official KATs, and
speech codecs like AMBE/IMBE and Codec 2 publish reference input/output pairs so
independent decoders can prove conformance.

Two hazards deserve care. First, an *incorrect* golden file locks in a bug: the
vector is only as trustworthy as the moment it was captured, so it should be
generated from a verified reference or a hand-computed answer, not from whatever
the code happened to emit. Second, over-strict bit-exact comparison on
floating-point pipelines produces brittle tests that fail across compilers or
CPUs — reserve exactness for integer and specification-defined outputs, and use
tolerances elsewhere.

## Relevance to SDR

Golden vectors are how DSP correctness is kept honest over time. A digital filter,
a Viterbi decoder, a CRC check, or a full protocol framer each has a definable
right answer for a given input, so each can be pinned. Regenerating the golden
output and eyeballing the diff is often the fastest way to understand the effect
of a refactor on a numerically dense function.

**GopherTrunk** relies on this pattern throughout its decode chain. Its
file-replay tests assert concrete, checked-in expected results — decoded control
messages from a given capture, and demodulation quality metrics that must not
regress (a specific capture is pinned to lock at roughly a known demod SNR and
EVM, and a companion test asserts the decode chain reaches the receiver at the
same in-channel SNR whether the file is processed natively or resampled). Those
are golden vectors in spirit: a known input, a known-correct measured output, and
a failing test the instant a change moves the number. Combined with
[forward-error-correction](/reference/forward-error-correction/) known-answer
tests and [synthetic](/reference/simulation-driven-sdr/) inputs, they let the
whole receiver be regression-tested with no transmitter present.

## Sources

[^tv]: [Test vector](https://en.wikipedia.org/wiki/Test_vector) — Wikipedia, on fixed input/expected-output pairs used to validate implementations against a specification.
[^ct]: [Characterization (golden master) test](https://en.wikipedia.org/wiki/Characterization_test) — Wikipedia, on capturing existing behavior as a reference to detect later changes.
