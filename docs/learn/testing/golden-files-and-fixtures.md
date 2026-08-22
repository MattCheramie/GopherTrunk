---
slug: golden-files-and-fixtures
title: Golden files & fixtures
description: Recorded inputs and expected outputs stored beside your tests — how fixtures and golden files turn messy real-world data into repeatable tests, and the update-flag pattern that keeps them honest.
keywords: golden files go, test fixtures, testdata directory go, golden file testing, update golden flag, fixture data, snapshot testing go
level: intermediate
status: full
prereq:
  - your-first-go-test
---

# Golden files & fixtures

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **fixture** is stored data a test uses as input; a **golden file** is stored
data a test uses as the **expected output**, compared byte-for-byte against
what the code produces. Go convention keeps both in a **`testdata/`**
directory, which the toolchain ignores but tests can read. Golden testing
shines when outputs are **large or structured** — decoded messages, rendered
reports, protocol frames — and an **`-update` flag** regenerates goldens
deliberately. The catch: a golden file is only as right as the day someone
verified it, so **review golden diffs like code**.
</div>

Hand-written literals carry a test only so far. Real inputs — a captured radio
burst, a gnarly config file, a 2 KB JSON response — are too big to type and too
valuable to simplify. This lesson is about testing with *stored reality*, the
technique GopherTrunk leans on harder than almost any other project you'll
meet.

## Fixtures: recorded inputs

A **fixture** is input data checked into the repo for tests to load. Go blesses
a location for it: files under **`testdata/`** are ignored by the build but
available to tests in that package:

```text
internal/radio/tetra/
    tch.go
    tch_test.go
    testdata/
        control_burst_clean.bin
        control_burst_truncated.bin
```

```go
func TestDecodeBurst_Clean(t *testing.T) {
    raw, err := os.ReadFile("testdata/control_burst_clean.bin")
    if err != nil {
        t.Fatalf("reading fixture: %v", err)
    }
    msg, err := DecodeBurst(raw)
    if err != nil {
        t.Fatalf("DecodeBurst() error: %v", err)
    }
    if msg.Talkgroup != 4521 {
        t.Errorf("Talkgroup = %d, want 4521", msg.Talkgroup)
    }
}
```

The power move is *where the fixture came from*: not typed from the spec, but
**captured from a real radio**. A fixture recorded off the air carries every
quirk a real transmitter produces — quirks a hand-built input, constructed from
the same spec-reading as the decoder, cannot contain. That distinction becomes
a matter of doctrine in
[the self-consistent synthetic trap](/learn/testing/the-self-consistent-synthetic-trap/).

## Golden files: recorded expected outputs

Now the output side. Suppose the decoder emits a structured summary of
everything in a capture. Asserting field-by-field would take hundreds of lines
— and miss anything you didn't think to assert. Instead, store a known-good
output as the **golden file** and compare wholesale:

```go
func TestDecodeCapture_Summary(t *testing.T) {
    got := renderSummary(t, decodeFile(t, "testdata/site_capture.bin"))

    golden := "testdata/site_capture.golden.txt"
    if *update {
        os.WriteFile(golden, got, 0o644) // regenerate deliberately
    }
    want, err := os.ReadFile(golden)
    if err != nil {
        t.Fatalf("reading golden: %v", err)
    }
    if !bytes.Equal(got, want) {
        t.Errorf("summary differs from %s:\n%s", golden, diff(want, got))
    }
}
```

Byte-for-byte comparison means **every** aspect of the output is pinned — the
fields you thought about and the ones you didn't. Any behavior change, however
incidental, surfaces as a diff.

## The `-update` flag: regeneration as a deliberate act

Outputs legitimately change — you add a field to the summary, and fifty goldens
are now "wrong." Nobody re-types them; the suite regenerates them, but only
when explicitly told:

```go
var update = flag.Bool("update", false, "rewrite golden files with current output")
```

```bash
go test ./... -update   # regenerate, then REVIEW THE DIFF
git diff testdata/      # is every change one you intended?
```

That review step is the entire integrity of the system. A golden file is not
ground truth from the heavens — it's **the output somebody once looked at and
vouched for**. Run `-update` reflexively and you convert a behavior-pinning
suite into an it-does-whatever-it-does suite: any bug the change introduced is
now enshrined as "expected." Golden diffs get reviewed with the same care as
code diffs, because that's exactly what they are — a statement of intended
behavior.

> Rule of thumb: `-update` answers "the change is intended"; only a human diff
> review can answer "the change is *right*."

## Where the technique shines — and strains

| Fits golden testing | Strains it |
|---------------------|------------|
| Large structured outputs (decoded frames, rendered tables, generated configs) | Tiny outputs — a plain assert is clearer |
| Outputs where *everything* should be stable | Outputs with timestamps, randomness, or map ordering — normalize these first or diffs are noise |
| Formats a human can eyeball in review (text > binary) | Opaque binaries nobody can vouch for by reading |
| Regression pinning: "behavior today = behavior after refactor" | Deciding what's *correct* in the first place — goldens pin, they don't judge |

That last row is the honest limit: golden testing verifies **stability**, not
correctness. It's at its best wrapped around behavior that was verified
correct some *other* way — against a spec, a reference implementation, or
real-world validation. GopherTrunk's replay suite, coming in
[Unit 6](/learn/testing/replay-integration-tests/), is this lesson at system
scale: whole recorded IQ captures as fixtures, decoded end-to-end, with the
expected results pinned.

<div class="knowledge-check" data-quiz data-correct-msg="Right — regeneration makes the current output the new expectation, so an unreviewed -update can silently enshrine a bug as correct." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the danger of running <code>go test -update</code> and committing without reviewing the diff?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Whatever the code currently produces — bug included — becomes the new "expected" output</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The testdata directory grows too large for git</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Go's build cache stops working for that package</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Fixtures** are stored inputs; **golden files** are stored expected outputs,
  compared byte-for-byte — both live in **`testdata/`**.
- Fixtures **captured from reality** carry quirks hand-built inputs can't — the
  seed of Unit 6's biggest lesson.
- Golden comparison pins **everything** about an output, including what you
  didn't think to assert.
- Regenerate with an **`-update` flag**, then **review the diff like code** —
  a golden is only as right as the day someone vouched for it.
- Goldens verify **stability**, not correctness; pair them with independent
  verification of what "correct" is.

Next up: [Property-based testing](/learn/testing/property-based-testing/)
