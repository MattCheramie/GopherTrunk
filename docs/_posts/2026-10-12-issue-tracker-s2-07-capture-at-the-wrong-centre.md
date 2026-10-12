---
title: "From the Issue Tracker, Season 2, Part 7: The Capture Carved at the Wrong Centre — A NaN That Fell Back Silently"
description: "How a 60 s SigLab grab labelled 442.8125 MHz came back with neither DMR repeater in it — the form coerced the centre with Number(), a NaN fell back to the tuner centre, and every log line agreed with the wrong file — and the strict parser, the 400, the requested_center_hz field and the captures-row centre that make the next grab honest."
category: solution-postmortem
keywords: siglab capture wrong centre, requested_center_hz, capture carved at tuner centre, number coercion nan fallback, parseCaptureTuning, center_hz without bandwidth_hz 400, sdr narrowband slice capture, sample synchronous multi slice capture, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, siglab, capture, web, api, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 7
---

*Part 7 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems that continues the
[first season]({{ '/blog/series/from-the-issue-tracker/' | relative_url }}):
one bug per part, with receipts.
[Part 6]({{ '/blog/solution-postmortem/issue-tracker-s2-06-false-exact-solves/' | relative_url }})
closed the DMO story on a capture that lined up with the log to the sample.
This part is about a capture that did not line up with anything: a 60 s
grab the operator asked for at 442.8125 MHz, in which neither of the two
repeaters it was meant to hold existed — because a text field had turned
the centre into `NaN` and nothing downstream said so.*

> **TL;DR:** The 15 Sep IPSC field material (X310 at 6.25 MS/s, Fire and
> Fire2 taps) included a 60 s "442.8125 MHz" cs16 whose only carriers sat
> at −11/−37/−61/−87 kHz and +213 kHz of the **tuner** centre, 441.7 MHz —
> the file covered 441.26–442.14 MHz, and the repeaters at 442.3875 and
> 443.2375 MHz were simply not in it. The daemon and the SPA at the
> operator's commit wired `center_hz` correctly; the centre was lost
> client-side, where the form coerced with `Number()` and silently fell
> back on `NaN`, and a centre without a bandwidth was dropped the same way.
> The start line said `center_hz=441700000 tuner_center_hz=441700000` and
> nothing disagreed. Now `web/siglab/src/lib/captureTuning.ts` parses
> strictly and errors (`parseCaptureTuning`), `resolveCaptureCenters` 400s
> a `center_hz` without `bandwidth_hz` that differs from the tuner centre,
> the `siglab: capture started` line carries `requested_center_hz`, and the
> staged-capture DTO and `CaptureRow` show the centre a file was actually
> carved at. Pinned by `captureTuning.test.ts`,
> `TestSiglabCaptureCentreWithoutBandwidthIs400` and
> `TestSiglabCaptureLogsStartAndEnd`.

**Key takeaways**

- **A silent fallback is a success-only log line in another dress.** The
  file, the sidecar and the start line all reported the tuner centre, and
  all were "correct"; the lie was that nobody had asked for it.
- **Parse the operator's text strictly.** A decimal comma, a pasted
  zero-width character or a unit suffix must be an error, not a `NaN` that
  quietly becomes a default.
- **Log the request beside the result.** `requested_center_hz` next to
  `center_hz` makes a grab at the wrong centre visible in one line.
- **A centre only means something with a bandwidth.** The capture never
  retunes; it carves a slice from the live stream, so a centre alone has
  no honest interpretation and now gets a 400.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Strict form parse | plain decimals only; errors on anything else, and on a centre without a bandwidth | `web/siglab/src/lib/captureTuning.ts` (`parseCaptureTuning`, `splitCentres`) |
| Route-side guard | 400 for `center_hz` without `bandwidth_hz` unless it equals the tuner centre | `internal/api/capture.go` (`resolveCaptureCenters`) |
| Span check | slice must fit the tuner's current span, no retune | `narrowbandParams` |
| The request in the log | `requested_center_hz` beside `center_hz` / `tuner_center_hz` | `siglab: capture started` line, `TestSiglabCaptureLogsStartAndEnd` |
| Where the file is | staged DTO `CenterHz`; captures row renders it | `internal/api/siglab_jobs.go`, `web/siglab/src/panels/CaptureRow.tsx` |
| Failing-first pins | bad centres rejected; centre-without-bandwidth 400s | `captureTuning.test.ts`, `TestSiglabCaptureCentreWithoutBandwidthIs400` |

## In this post

- **The file that held nothing** — what the capture contained and what it was labelled.
- **Where the centre went** — `Number()`, `NaN`, and two silent fallbacks.
- **Why everything agreed with the wrong file** — a success-only path in three layers.
- **The strict parser** — refuse, don't coerce.
- **The route, the log and the row** — the same number in three places.
- **What the next capture must be** — the still-unknown −20 kHz emitter.

## The file that held nothing

The 15 Sep material from the operator's dual-repeater IPSC rig (an X310
streaming 6.25 MS/s wideband, two DMR taps named Fire and Fire2) was a 60 s
cs16 and a `debug.log`. The file name said 442.8125 MHz — the midpoint the
operator chose so that an 880 kHz slice would hold both repeaters, at
442.3875 and 443.2375 MHz. A spectrum of the file told a different story:
its only carriers sat at −11, −37, −61 and −87 kHz and at +213 kHz
relative to its centre, and neither repeater was among them. Working back, the
file's centre was 441.7 MHz — the **tuner** centre — and its 880 kHz span
ran 441.26–442.14 MHz. Both repeaters were outside it.

The start line in the log read `center_hz=441700000
tuner_center_hz=441700000`, which was true: the slice had been carved at
the tuner centre. The sidecar agreed, and nothing in the UI disagreed. Every layer reported
accurately what it had done, and no layer reported what had been asked.
The capture tooling had just been extended with the capture
start/ended/aborted lines (Part 6) precisely so that a capture could be
lined up with the decode log without guessing — and the very next capture
was one that no log line could have caught, because the request had
already been lost before any of them ran.

## Where the centre went

The daemon's capture route and the SPA at the operator's commit both wired
`center_hz` correctly end to end, so the first suspicion — a dropped field
in the API — was wrong. The centre was lost **client-side**, before the
request existed. The SigLab form coerced its "Center MHz" text with
`Number()`. A decimal comma (`442,8125`), a stray invisible character
pasted along with the value, or a unit suffix produce `NaN`, and a `NaN`
centre fell back — silently — to "no centre", which the route reads as
"carve at the tuner centre". Separately, a centre supplied *without* a
bandwidth was dropped the same way: the capture does not retune the SDR, it
carves a narrowband slice from the live stream, so a centre only applies to
a slice, and the old code resolved the contradiction by discarding the
centre. Either path produces the same wrong file, labelled with the right
number.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A frequency axis from 441.0 to 443.6 megahertz. A dashed box marks the 880 kilohertz slice actually recorded, 441.26 to 442.14, centred on the tuner centre at 441.7 with its stray carriers inside. A solid box marks the slice the operator asked for, 442.3725 to 443.2525, centred on 442.8125, with the two repeaters at 442.3875 and 443.2375 inside it. Neither repeater falls inside the recorded box.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">15 Sep grab: requested 442.8125 MHz ± 440 kHz, recorded at the tuner centre</text>
  <line x1="30" y1="110" x2="650" y2="110" stroke="var(--fg-muted)"/>
  <g font-size="8" fill="var(--fg-muted)" text-anchor="middle">
    <text x="30" y="126">441.0</text><text x="186" y="126">441.7</text><text x="449" y="126">442.8125</text><text x="650" y="126">443.6</text>
    <text x="340" y="142">MHz</text>
  </g>
  <rect x="92" y="60" width="210" height="40" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="197" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">recorded: 441.26–442.14 (tuner centre)</text>
  <g stroke="var(--fg-muted)">
    <line x1="165" y1="80" x2="165" y2="100"/><line x1="177" y1="80" x2="177" y2="100"/><line x1="172" y1="80" x2="172" y2="100"/><line x1="183" y1="80" x2="183" y2="100"/><line x1="237" y1="80" x2="237" y2="100"/>
  </g>
  <text x="197" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−11/−37/−61/−87 kHz, +213 kHz: not repeaters</text>
  <rect x="358" y="60" width="210" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="463" y="52" text-anchor="middle" fill="var(--accent)" font-size="8">requested: 442.3725–443.2525</text>
  <line x1="362" y1="66" x2="362" y2="100" stroke="currentColor" stroke-width="2"/>
  <text x="362" y="160" text-anchor="middle" fill="currentColor" font-size="8">Fire 442.3875</text>
  <line x1="565" y1="66" x2="565" y2="100" stroke="currentColor" stroke-width="2"/>
  <text x="565" y="160" text-anchor="middle" fill="currentColor" font-size="8">Fire2 443.2375</text>
  <text x="340" y="186" text-anchor="middle" fill="var(--accent)" font-size="8">centre lost client-side (NaN) → slice carved around 441.7 MHz under the name 442.8125</text>
</svg>
<figcaption>The bandwidth survived; the centre did not. An 880 kHz slice was carved around the tuner centre and named for the centre that never reached the daemon.</figcaption>
</figure>

## Why everything agreed with the wrong file

This is the lesson
[Season 1 Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
drew from a different set of bugs: a log line that reports only what
succeeded carries no information about what was attempted. Every layer
here was in that state. The form had a default ("blank means the tuner
centre") and let a malformed value *become* the default instead of an
error. The route had a precedence rule ("a centre only applies to a
slice") and resolved the contradiction by dropping the centre instead of
refusing. The start line printed the centre it was carving at, which was
the only centre it knew. The result is a file that is internally
consistent, metadata-complete, and wrong — the capture-tooling cousin of
the self-consistent trap
([Season 1 Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})):
nothing in the pipeline could disagree with itself, because the disagreeing
party had been eliminated at the first step.

The cost is not abstract. A 60 s wideband grab from an X310 is a
scheduling event for an operator, the analysis of it took real time, and
the conclusion — "neither repeater is in the file" — is one a reader
unfamiliar with the chain would reasonably have read as the rig's fault.

## The strict parser

`parseCaptureTuning` in `web/siglab/src/lib/captureTuning.ts` replaces the
coercion. The only accepted shape is a plain decimal with a dot — the form
every frequency in the console already uses — and everything else is an
error the operator can act on:

```ts
// web/siglab/src/lib/captureTuning.ts (shape)
const decimal = /^[+-]?(\d+\.?\d*|\.\d+)$/;

function parsePositive(label: string, raw: string, unit: string): number | string {
  const text = raw.trim();
  if (!decimal.test(text)) {
    return `${label} "${raw}" is not a number — use a plain decimal like 442.3875 (${unit}); clear it for the default`;
  }
  const v = Number(text);
  if (!(v > 0)) return `${label} must be greater than 0 ${unit}`;
  return v;
}
// …a centre with no bandwidth:
return { ok: false, error: "Center MHz needs a Bandwidth kHz: the tuner is not retuned, so a centre only applies to a narrowband slice …" };
```

The one silent path left is the one that was asked for: blank fields mean
a full-band grab at the tuner centre. `Captures.tsx` calls it before
building the request and shows the error instead of posting. The test file
enumerates the inputs that used to vanish — `"442,8125"`, a value with a
zero-width character appended, `"4.428125e2"`, `"442.8125 MHz"`, `"abc"` —
and asserts each is refused with a message matching `Center MHz .* is not
a number`, and that `"442.8125"` with an empty bandwidth is refused with
`needs a Bandwidth kHz`. A second parser detail is itself a guard:
`splitCentres` treats a comma as a list separator only when every piece
around it carries a decimal point, so `"442,8125"` stays one token and is
rejected as not-a-number rather than being read as two centres. The same
tokeniser is what later let the form take several centres for the
sample-synchronous multi-slice capture (`centers_hz`, up to
`maxCaptureSlices`), where every slice is carved from the same live chunks
through an identical DDC.

## The route, the log and the row

Client-side strictness is not enough on its own — the API is also driven
by `gophertrunk capture` and by scripts — so the route refuses the same
contradiction. `resolveCaptureCenters` in `internal/api/capture.go` 400s a
`center_hz` that arrives without a `bandwidth_hz` **and differs from the
tuner centre**; the tuner centre itself, which is what a full-band grab is
at anyway, stays fine:

```go
// internal/api/capture.go (shape) — resolveCaptureCenters
if req.BandwidthHz == 0 {
    if req.CenterHz != tunerCenterHz {
        return nil, fmt.Errorf(
            "center_hz %d without bandwidth_hz: a capture is carved from the tuner's live stream "+
                "without retuning (tuner centre %.4f MHz), so a centre only applies to a narrowband slice — "+
                "add bandwidth_hz to slice around %.4f MHz, or omit center_hz for a full-band grab", …)
    }
    return nil, nil
}
```

`TestSiglabCaptureCentreWithoutBandwidthIs400` posts the operator's exact
case against a fake 6.25 MS/s X310 at 441.7 MHz — `center_hz: 442812500`
with no bandwidth — and wants a 400 that names `bandwidth_hz`; then posts
`center_hz: 441700000` and wants a 200 whose staged capture reports
`CenterHz` = 441 700 000. `narrowbandParams` still rejects a slice that
falls outside the tuner's span, as before.

Then the number is written where the operator will look. The `siglab:
capture started` line gained `requested_center_hz`, printed beside
`center_hz` (the slice's actual centre) and `tuner_center_hz`:

```text
INF siglab: capture started serial=X310 center_hz=442387500 sample_rate_hz=200000 bandwidth_hz=200000 requested_center_hz=442387500 tuner_center_hz=441700000 tuner_rate_hz=6250000 format=flac seconds=25 protocol=tetra-dmo path=…
```

`TestSiglabCaptureLogsStartAndEnd` asserts `requested_center_hz=460400000`
appears for a `center_hz: 460400000` request. A grab at the wrong centre
would now print `requested_center_hz` ≠ `center_hz` — one line, one
comparison, which is how
[Field Notebook Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
teaches operators to read it. And the staged-capture DTO carries
`CenterHz`, so `CaptureRow.tsx` renders the centre a file was carved at
(`data-testid="center-mhz"`, four decimals): a slice that landed at the
tuner centre is no longer indistinguishable from the one asked for. Four
decimals is its own small fix from the same week — `captureName` used
`%.3f` and renamed a 442.3875 MHz slice "442.387MHz"; the standard
6.25/12.5 kHz steps put centres on quarter kilohertz, and
`TestCaptureNameKeepsQuarterKilohertz` pins it.

## What the next capture must be

None of this recovers the 15 Sep grab; it makes the next one honest. The
question that grab was supposed to answer is still open: the Fire2 tap's
deaf-heal WARN kept engaging a coarse carrier offset of ≈ −20.1 kHz, and
what sits at ≈ 442.3675 MHz is still unnamed — the file covered
441.26–442.14 MHz and so never saw it. Geometry rules out the channelizer
(the bin-4 image of Fire lands at +68.75 kHz, the bin centre at
+93.75 kHz, the tuner DC nowhere near). The instrument is a capture
centred on 442.3875 MHz with ±100 kHz over one full idle cycle, train and
gap, replayed through `TestDMRIPSCReplay` with a gap-phase spectrum. The
reject path in the deaf heal makes the tap immune either way, and the
16 Sep channel-select filter is the structural fix
([DMR End to End Part 10]({{ '/blog/deep-dives/dmr-end-to-end-10-wideband-bin-edge-deaf-heal/' | relative_url }}));
but the emitter's identity waits on a capture that actually contains it.

The same report's first attempt, before the cs16, had been a FLAC that
aborted after 3908 samples with `unable to encode sample rate 880029` —
which is the next part.

### How a NaN shaped the Go and TypeScript code

- **Refuse at the edge, in both edges.** The form and the route each
  validate; neither trusts the other to have done it.
- **The request travels with the result.** `requested_center_hz` is a log
  field, `CenterHz` a DTO field; the row renders what the daemon says, not
  what the form sent.
- **A default is reachable only by asking for it.** Blank fields are the
  one path to the tuner centre; a malformed value never becomes blank.
- **Literal bad inputs are the fixture.** `captureTuning.test.ts` holds the
  decimal comma and the zero-width character as strings, so the next
  coercion regression fails on the exact inputs that bit.

## Where this goes next

The same operator's first attempt at the same grab died in the FLAC
encoder: a slice carved at 880 029 Hz is a rate FLAC's frame header has
no code for, and the upstream encoder never falls back to the one code
that would carry it.
[Part 8]({{ '/blog/solution-postmortem/issue-tracker-s2-08-sample-rate-880029/' | relative_url }})
reads the frame-header sample-rate table and the two places a FLAC stream
stores the same number.

## FAQ

**Why did my GopherTrunk SigLab capture land at the tuner centre instead of the centre I typed?**
Before 15 Sep the form coerced the "Center MHz" text with `Number()`; a
decimal comma, a pasted invisible character or a unit suffix became `NaN`
and silently fell back to the tuner centre, and a centre without a
bandwidth was dropped the same way. `parseCaptureTuning` now refuses both
with an error, and the route returns 400 for a centre without a bandwidth.

**What does requested_center_hz mean in the capture started log line?**
It is the centre the request asked for, printed beside `center_hz` (the
centre the slice was actually carved at) and `tuner_center_hz`. On a
healthy slice `requested_center_hz` equals `center_hz`. If they differ with
`bandwidth_hz=0`, the grab was carved at the tuner centre — the 15 Sep
failure, now visible in one line.

**Why does a capture need a bandwidth to use a centre?**
A SigLab capture never retunes the SDR. It carves a narrowband slice from
the live stream through a streaming DDC, so a centre is only meaningful as
the middle of a slice. `resolveCaptureCenters` 400s a `center_hz` without
`bandwidth_hz` that differs from the tuner centre; omit the centre for a
full-band grab.

**Can I capture several frequencies at once in SigLab?**
Yes. List several centres in the "Center MHz" field (comma, semicolon or
whitespace separated) with one bandwidth; `splitCentres` builds
`centers_hz`, and the daemon carves every slice from the same live chunks
through identical DDCs so the files are sample-synchronous. A decimal
comma inside one value is still rejected as not-a-number.

## Series navigation

**Part 7 of 14** · ←
[Part 6: False Exact Solves — When 24 Redundant Checks Were Fiction]({{ '/blog/solution-postmortem/issue-tracker-s2-06-false-exact-solves/' | relative_url }})
· Next →
[Part 8: 'unable to encode sample rate 880029' — FLAC's Frame-Header Table]({{ '/blog/solution-postmortem/issue-tracker-s2-08-sample-rate-880029/' | relative_url }})
