---
slug: fuzzing
title: Fuzzing
description: Throw randomized, mutated inputs at your code until it crashes — Go's built-in fuzzer, coverage-guided mutation, and why parsers of untrusted input need fuzzing most.
keywords: fuzzing go, go fuzz tutorial, coverage guided fuzzing, fuzz testing, go test -fuzz, corpus seed, parser fuzzing, crash detection
level: intermediate
status: full
prereq:
  - property-based-testing
---

# Fuzzing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Fuzzing** hammers code with **mutated, mostly-malformed inputs**, hunting not
for wrong answers but for **crashes** — panics, hangs, out-of-range accesses.
Go builds it in: **`FuzzXxx`** functions, run with **`go test -fuzz`**, use
**coverage-guided mutation** — inputs that reach new code paths breed further —
and every crash is saved as a permanent regression case. Fuzz anything that
**parses untrusted input**: file formats, network bytes, radio frames — data
your code doesn't control.
</div>

Property testing generated *valid* inputs and checked deep claims about them.
Fuzzing is its feral sibling: generate *broken* inputs by the million and check
the shallowest claim of all — *does it survive?* For code that eats data from
the outside world, that shallow claim is worth a fortune.

## Why malformed input is a first-class threat

Think about what a parser really is: code that reads bytes it does not control
and builds structures from what it finds — lengths, counts, offsets. Feed it
bytes that lie ("the next field is 200 bytes long", says a 12-byte message) and
naive code walks off the end of a slice and panics. A decoder like
GopherTrunk's lives on the extreme end of this: its input is *whatever the
radio demodulated*, including noise-corrupted frames, truncated bursts, and
bit-flipped garbage, all day long. One panic on one malformed frame kills the
whole scanner mid-call. And where crashes live, security bugs live too — a
large fraction of real-world vulnerabilities are exactly "parser mishandles
malformed input."

Hand-writing malformed cases hits the limit fast: you can imagine a truncated
message, but not the specific 17-byte sequence where a length field interacts
with a padding rule to underflow a subtraction. Fuzzers find those hourly.

## Go's built-in fuzzer

Since Go 1.18, fuzzing is part of the standard toolchain. A fuzz target lives
in a `_test.go` file:

```go
func FuzzDecodeFrame(f *testing.F) {
    f.Add(cleanFrameBytes)      // seed corpus: known-interesting inputs
    f.Add(truncatedFrameBytes)

    f.Fuzz(func(t *testing.T, data []byte) {
        frame, err := DecodeFrame(data)
        if err != nil {
            return // rejecting bad input with an error is correct behavior
        }
        // If it claims success, the result must be sane:
        if frame.PayloadLen > len(data) {
            t.Errorf("PayloadLen %d exceeds input size %d", frame.PayloadLen, len(data))
        }
    })
}
```

```bash
go test -fuzz=FuzzDecodeFrame          # mutate until stopped or crashing
go test -fuzz=FuzzDecodeFrame -fuzztime=2m
```

Note the contract being tested: **errors are fine, panics are not**. A parser
facing garbage should return an error; the fuzzer's job is finding the garbage
that makes it *crash* instead. You can also assert cheap invariants on
"successful" parses, as above — a light dusting of
[property thinking](/learn/testing/property-based-testing/) that catches
silent nonsense, not just explosions.

## Coverage-guided mutation: why fuzzers are smart

The engine isn't uniform randomness — that would spend eternity on inputs the
first `if` rejects. Go's fuzzer instruments the code and watches **coverage**:

1. Start from the **seed corpus** (`f.Add(...)` — real, valid samples).
2. **Mutate** a corpus entry: flip bits, truncate, splice, tweak numbers.
3. If the mutant executes a **code path never seen before**, keep it in the
   corpus — it found new territory, so *its* mutants get a turn.
4. Repeat, millions of times, burrowing progressively deeper past each check.

This is why seeds matter: a valid frame as a seed means mutants start life
*almost* valid — past the magic-number check, into the deep parsing logic where
the interesting bugs live. When a crash is found, Go writes the exact input to
`testdata/fuzz/FuzzDecodeFrame/...`; committed, it becomes a permanent
[regression test](/learn/testing/regression-tests/) that plain `go test` runs
forever after — the same "random search finds it once, the suite guards it
forever" partnership as property testing.

> Rule of thumb: fuzz every function whose input crosses a trust boundary —
> file formats, network protocols, radio frames, user uploads. If your code
> didn't produce the bytes, fuzz the code that reads them.

## Where fuzzing fits in your suite

Fuzzing complements rather than replaces the layers you have:

| Technique | Inputs | Detects |
|-----------|--------|---------|
| Example / table tests | Hand-picked, valid & invalid | Wrong answers on known cases |
| Property tests | Generated **valid** | Violated claims on valid space |
| Fuzzing | Mutated, mostly **malformed** | Crashes, hangs — and saved cases feed the table |

Operationally: fuzz runs are open-ended, so they don't belong in the
per-commit loop — run them time-boxed in CI (`-fuzztime`), or let long
campaigns run on idle machines. The corpus and crash files, though, run fast
and belong in the everyday suite.

<div class="knowledge-check" data-quiz data-correct-msg="Right — coverage guidance keeps mutants that reach new code paths, so the fuzzer tunnels past validation into deep logic instead of bouncing off the first check." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a coverage-guided fuzzer so much more effective than uniformly random bytes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It only generates inputs that are fully valid per the format spec</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It formally proves the absence of crashes for all inputs</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It keeps and re-mutates inputs that reach new code paths, digging deeper past each check</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Fuzzing** bombards code with mutated, mostly-malformed inputs hunting
  **crashes**; errors are acceptable, panics are findings.
- **Coverage-guided mutation** breeds inputs that reach new paths — seeded with
  real samples, it tunnels deep past validation.
- Go builds it in: **`FuzzXxx`**, `go test -fuzz`, crashes saved under
  `testdata/fuzz/` as permanent regression cases.
- Fuzz whatever **parses untrusted input** — radio frames, file formats,
  network bytes; that's where crashes and vulnerabilities cluster.
- Fuzzing completes the trio: examples pin known cases, properties check valid
  space, fuzzing patrols the malformed wilderness.

Next up: [Linters & static analysis](/learn/testing/linters-and-static-analysis/)
