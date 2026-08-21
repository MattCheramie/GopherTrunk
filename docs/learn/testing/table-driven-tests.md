---
slug: table-driven-tests
title: Table-driven tests
description: Go's signature testing idiom — one test body, a table of cases, and t.Run subtests that name each row. How to write them, when to reach for them, and the mistakes to avoid.
keywords: table driven tests go, t.Run subtests, go testing idiom, test cases slice, go table test example, subtest go, parameterized tests go
level: intermediate
status: full
prereq:
  - your-first-go-test
---

# Table-driven tests

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **table-driven test** lists its cases as a slice of structs — inputs, expected
outputs, and a **name** — then loops once over the table running the same
arrange-act-assert body. **`t.Run`** turns each row into an independently named,
independently failing **subtest**. It's the dominant Go idiom because adding the
next edge case costs **one line**, and that low cost is what makes thorough
boundary coverage actually happen.
</div>

The last lesson ended with three tests that shared nine-tenths of their code.
Multiply that by the dozen boundary cases a serious function deserves and the
copy-paste becomes a maintenance problem. Go's answer is an idiom you'll meet in
virtually every Go codebase, GopherTrunk included.

## The idiom

Same `FormatMHz` function, all cases in one table:

```go
func TestFormatMHz(t *testing.T) {
    tests := []struct {
        name string
        hz   int64
        want string
    }{
        {"typical UHF frequency", 851_012_500, "851.0125 MHz"},
        {"zero", 0, "0.0000 MHz"},
        {"gigahertz range", 1_200_000_000, "1200.0000 MHz"},
        {"single hertz", 1, "0.0000 MHz"},
        {"negative offset", -812_500, "-0.8125 MHz"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatMHz(tt.hz)
            if got != tt.want {
                t.Errorf("FormatMHz(%d) = %q, want %q", tt.hz, got, tt.want)
            }
        })
    }
}
```

Three parts to notice:

- **The table** — an anonymous-struct slice, each row one complete case with a
  human-readable `name`.
- **The loop** — one arrange-act-assert body serving every row. Fix the
  assertion once, all cases benefit.
- **`t.Run(tt.name, …)`** — each row becomes a **subtest**: reported, passed,
  and failed *independently*.

## Why subtests earn their keep

`t.Run` looks like ceremony until the first failure. Without it, a mid-loop
`t.Errorf` leaves you guessing which row broke. With it:

```text
--- FAIL: TestFormatMHz (0.00s)
    --- FAIL: TestFormatMHz/negative_offset (0.00s)
        freq_test.go:24: FormatMHz(-812500) = "-0.8124 MHz", want "-0.8125 MHz"
```

The failing *row* is named in the output, every other row still runs (so one
broken case doesn't mask four more), and you can re-run exactly the offender
while fixing it:

```bash
go test -run 'TestFormatMHz/negative_offset'
```

That name-based addressing also means table rows show up individually in CI
logs and IDE test panels — each row is a first-class test in every tool.

## The real payoff: edge cases become one-liners

Here's the behavioral economics of the idiom. When adding a case costs a copied
function, developers add two or three and stop. When it costs **one line**,
they add ten. A decoder function in a project like GopherTrunk — where inputs
are whatever the airwaves produce — accumulates rows like *"truncated at byte
7"*, *"all zero bits"*, *"maximum talkgroup ID"*, *"parity bit flipped"*: the
whole [boundary checklist](/learn/testing/the-testing-mindset/) as data, not
code. And when a bug is fixed, its exact triggering input drops into the table
as a permanent [regression row](/learn/testing/regression-tests/) — one line
that guards the fix forever.

> Rule of thumb: the moment you're about to copy-paste a test and change two
> values, stop and build a table instead.

## Mistakes to avoid

- **Unnamed rows.** A table keyed only by index gives failures like
  `TestFormatMHz/#03` — you're back to guessing. The `name` field is not
  optional in spirit.
- **Logic in the table.** Rows should be *data*. When a row needs its own
  special-case `if` inside the loop body, that case has outgrown the table —
  give it its own test function instead of complicating every other row's path.
- **One giant table for unrelated behaviors.** A table shares one body, so it
  fits cases that share one *shape* of claim. "Formats correctly" and "returns
  an error on invalid input" often want two tables — the second's rows carry a
  `wantErr` instead of a `want`.
- **Reusing state across rows.** Each subtest should arrange from scratch.
  Rows that mutate a shared object make row 4's result depend on row 3 having
  run — a [flaky-test](/learn/testing/flaky-tests/) seed, and it breaks `-run`
  addressing of single rows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — t.Run makes each row an independently named subtest, so the output says exactly which case failed and the rest keep running." markdown="0">
  <p class="knowledge-check__q">Quick check: what does wrapping each table row in <code>t.Run(tt.name, …)</code> buy you?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The loop runs faster because rows execute in parallel by default</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The compiler verifies the table's expected values are correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Each row fails independently, by name, and can be re-run alone with -run</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **table-driven test** is a slice of case structs plus one loop of
  arrange-act-assert — Go's signature testing idiom.
- **`t.Run`** gives every row an independent name, independent failure, and
  `-run` addressability.
- The payoff is economic: **one-line edge cases** mean boundary coverage —
  and permanent regression rows — actually get written.
- Keep rows as **data**, name every row, and don't share state between them.
- Split tables when behaviors stop sharing one shape of claim.

Next up: [Fakes, stubs, and mocks](/learn/testing/test-doubles/)
