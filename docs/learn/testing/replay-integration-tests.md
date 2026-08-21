---
slug: replay-integration-tests
title: "Replay: testing a radio without a radio"
description: Recorded IQ captures let GopherTrunk test a whole radio decoder deterministically — the golden-file idea at system scale, turning live radio's worst testing properties into a repeatable suite.
keywords: replay testing, iq capture testing, deterministic radio testing, sdr replay, decode recorded capture, integration testing sdr, capture fixtures
level: intermediate
status: full
prereq:
  - integration-tests
  - golden-files-and-fixtures
gophertrunk_links:
  - title: Architecture overview
    url: /architecture.html
    note: see where the replay input path joins the same decode pipeline live radio uses.
  - title: Vocoder notes
    url: /vocoders.html
    note: the voice decoders that replay suites exercise end-to-end, capture to audio.
---

# Replay: testing a radio without a radio

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Live radio is the worst possible test input: unrepeatable, uncontrollable, and
different at every desk. GopherTrunk's answer is **replay**: record the raw
**IQ samples** an SDR produces into a **capture file**, then feed that file
through the **same decode pipeline** as live radio — same signal, bit for bit,
every run, on any machine. It's the
**[fixture idea](/learn/testing/golden-files-and-fixtures/) at system scale**,
and it makes whole-decoder behavior **deterministic**: assertable in CI,
[bisectable](/learn/testing/bisecting-history/), and shareable — a user's
capture file turns "it garbles sometimes" into a bug that fails identically on
a developer's desk.
</div>

Everything Unit 3 said about integration testing meets its hardest customer
here: how do you write a repeatable test for a system whose input is *the
radio spectrum*? The answer is one of GopherTrunk's central engineering moves,
and the pattern transfers to any system that eats an unrepeatable stream.

## The problem: the air never says the same thing twice

Test a decoder against live radio and every principle from this module breaks
at once. **Unrepeatable**: the transmission that triggered the bug is gone
forever; no failing run can be re-run. **Uncontrollable**: you can't ask the
local network to transmit a truncated burst so you can test that branch.
**Unportable**: a test needing an antenna and a nearby TETRA network runs on
approximately one desk on Earth — and never in
[CI](/learn/testing/continuous-integration/). **Nondeterministic**: signal
strength, fading, and interference differ minute to minute — the definition of
a [flaky test](/learn/testing/flaky-tests/), imposed by physics.

## The move: record the samples, replay the file

An SDR hands software a stream of raw **IQ samples** — numbers describing the
received signal itself, before any decoding (the
[RF & SDR module](/learn/rf-sdr/what-is-sdr/) explains the format). Write that
stream to a **capture file** and you've frozen the radio moment: not a
description of what happened, but the actual input, replayable forever.

```bash
gophertrunk replay -in tetra_cc_2s_144k.cs16 -format cs16
```

The decode pipeline neither knows nor cares that its samples come from a file
instead of an antenna — same filters, same demodulator, same protocol
decoders. That "same pipeline" property is load-bearing: a replay test
exercises the *real* code path, so what it proves is what live operation gets.
Determinism follows: the file contains the same bits every time, so decoding is
repeatable down to the frame count — a signal that faded at 3.2 seconds fades
at 3.2 seconds on every run, on every machine, forever.

## Captures as fixtures: the suite this enables

With determinism secured, the whole testing toolkit applies to an entire
radio system at once. A replay integration test is
[arrange-act-assert](/learn/testing/anatomy-of-a-test/) where "arrange" is a
file in `testdata/`:

```go
func TestReplayDecodesControlChannel(t *testing.T) {
    stats := replayCapture(t, "testdata/tetra_cc_sync_loss_2s_144k.cs16")

    // Deterministic input ⇒ exact expectations, like any fixture test:
    if stats.BSCHValid < 140 {
        t.Errorf("decoded %d valid BSCH bursts, want >= 140", stats.BSCHValid)
    }
}
```

Three properties make capture fixtures more than big test files:

- **They preserve hard-won conditions.** A capture of a weak, distorted,
  barely-decodable signal is *irreplaceable* — you can't order those
  conditions up again. GopherTrunk's suite keeps exactly such captures: the
  marginal signal that once broke the decoder, now pinned as the input a fix
  must handle, forever.
- **They're the bug-report gold standard.** A user whose reception garbles
  "sometimes" sends the capture; the developer replays it and sees the *same
  garble on the same frame*. Reproduction — Unit 5's
  [hardest step](/learn/testing/reproducing-a-bug/) — arrives as an email
  attachment.
- **They feed every tool downstream.** A capture-driven check is a command
  that exits pass/fail — exactly what
  [`git bisect run`](/learn/testing/bisecting-history/) wants, and exactly
  what [regression testing](/learn/testing/regression-tests/) wants: fix a
  decode bug, and the triggering capture joins the suite as the failing-first
  test.

Because whole-capture decoding takes seconds to minutes per file, replay
suites live on the [integration tier](/learn/testing/integration-tests/) —
GopherTrunk's `make integration`, not the per-commit
[`make vet test` gate](/learn/testing/make-vet-test/) — with the heaviest
capture harnesses skip-guarded behind environment variables naming a capture
file, so the suite passes cleanly on machines without the (large) files and
digs deep on machines with them.

> Rule of thumb: any system fed by an unrepeatable stream — radio, market
> data, sensors, user traffic — becomes testable the same way: **record the
> stream at the boundary, replay the recording as a fixture.**

One caveat keeps this honest, and it sets up the next lesson. Replay proves
the decoder handles *that captured signal* deterministically. It says nothing
about signals you never captured — and if your captures are all clean, strong,
and well-formed, the suite is green over a decoder that fails on the real
world's margins. Worse still is testing against signals you *synthesized*
yourself…

<div class="knowledge-check" data-quiz data-correct-msg="Right — the capture file contains identical samples every run, and the pipeline can't tell file from antenna, so whole-system decoding becomes exactly repeatable." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a replay test deterministic when live radio testing never is?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">The input is a recorded file — bit-identical every run — fed through the same pipeline live radio uses</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The replay pipeline uses a simplified decoder with the randomness removed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Replay runs slower than real time, which prevents timing variation</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Live radio input is **unrepeatable, uncontrollable, unportable,
  nondeterministic** — untestable by every standard in this module.
- **Replay** freezes the boundary: record raw **IQ samples** to a capture
  file, feed it through the **same decode pipeline** as live radio.
- Captures are **fixtures at system scale**: exact expectations, preserved
  hard-won signal conditions, and bug reports that reproduce on any desk.
- Replay checks power the whole toolkit — CI, **bisect run**, failing-first
  **regression tests** — from the `make integration` tier.
- Limit: replay proves behavior on **captured** signals only — which is why
  the next lesson's trap matters.

Next up: [The self-consistent synthetic trap](/learn/testing/the-self-consistent-synthetic-trap/)
