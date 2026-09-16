---
title: "The Field Notebook, Part 9: Capture Lines — Aligning a Capture to the Log"
description: How to read GopherTrunk's siglab capture-started, capture-ended and capture-aborted log lines, line a recorded IQ file up with the daemon's decode log by timestamp, and recognise the two capture failures — a centre carved at the wrong frequency and a FLAC rate refusal — before they cost you an afternoon.
category: tutorials
keywords: siglab capture log, iq capture alignment, requested_center_hz, flac sample rate ceiling, capture aborted no samples, gophertrunk capture ppm, adc clipping warning, capture debug log, gophertrunk field notebook
tags: [field-notebook, logs, capture, siglab, iq, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 9
---

*Part 9 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field measures,
what a healthy rig prints, and which deep dive to open when a line goes wrong.
[Part 8]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }})
read the MRC health line. This part is about the lines that appear when you
deliberately record the air: the `siglab: capture` family. A capture is the most
valuable thing you can hand an investigation — but only if you can prove, from
the log alone, that it recorded the frequency you meant at the rate you think,
and can line its samples up against the decode log second-for-second.*

> **TL;DR:** A staged capture brackets itself in `debug.log` with
> `INF siglab: capture started` and `INF siglab: capture ended` (or
> `WRN siglab: capture aborted`). The start line carries `center_hz`,
> `sample_rate_hz`, `requested_center_hz` **and** `tuner_center_hz` — the last
> pair exists because a "442.8125 MHz" grab once recorded the tuner centre
> 441.7 MHz instead (15 Sep), and a centre without a bandwidth that differs
> from the tuner's is now refused with a 400. The end line carries `samples`,
> `recorded_seconds` and `elapsed`, so you can confirm the file is whole. A
> full-band FLAC over `baseband.FLACMaxSampleRateHz` (1048575 Hz) is rejected
> before the tuner is pinned — the old path aborted after 3908 samples with
> "unable to encode sample rate 880029". The `gophertrunk capture` CLI adds a
> measured-offset line, a clip WARNING, and a dropped-chunk WARNING. Align the
> capture's start clock to the decode log and every later conclusion is
> anchored.

**Key takeaways**

- **The start line is a contract, not a receipt.** `requested_center_hz` next
  to `tuner_center_hz` catches a capture carved at the wrong centre from the log
  alone — the failure that made a 60-second grab worthless on 15 Sep.
- **The end line proves the file is whole.** `samples`, `recorded_seconds` and
  `elapsed` say the capture ran to length; a short `recorded_seconds` or a
  `WRN … aborted` says it did not, and why.
- **FLAC has a hard rate ceiling and the daemon enforces it up front.** A
  full-band multi-MS/s FLAC is refused with a 400 before the tuner is touched,
  so you never wait out a capture that would die on its first block.
- **Alignment is the whole point.** One timestamp — the capture's start
  instant — ties every sample offset to a wall-clock line in the decode log,
  which is how the 12 Sep and 13 Sep captures were read.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Capture-started line | brackets the grab; carries requested vs tuner centre | `internal/api/capture.go` (`s.log.Info("siglab: capture started")`) |
| Capture-ended / aborted line | `samples`, `recorded_seconds`, `elapsed`, or `err` | `internal/api/capture.go` (`logEnd`) |
| Centre-without-bandwidth guard | 400s a centre that differs from the tuner's | `capture.go` (the `req.CenterHz != centerHz` check) |
| FLAC rate ceiling | refuse a full-band FLAC before pinning the tuner | `internal/sdr/baseband/flac.go` (`FLACMaxSampleRateHz`) |
| CLI measured-offset line | doubles as a ppm-measurement instrument | `cmd/gophertrunk/capture_carrier.go` (`carrierOffsetMeasurement`) |
| CLI clip / dropped-chunk WARNINGs | flag an overloaded front end or a slow drain | `capture.go`, `capture_carrier.go` (`formatClipWarning`) |
| What a capture must be | format / rate / duration / tap point | [Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}), [decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }}) |

## In this post

- **What the capture lines are telling you** — the started/ended/aborted trio.
- **What healthy looks like** — a grab that ran to length and lined up.
- **When it doesn't** — the wrong centre and the FLAC rate refusal.
- **Aligning a capture to the log** — one timestamp anchors everything.
- **The CLI capture verdicts** — offset, clipping, dropped chunks.

## What the capture lines are telling you

When you stage a capture from the web console, the daemon bookends it in
`debug.log` so you can line the file up against the decode log — a field request
from 13 Sep. The start line carries everything the metadata sidecar will, plus
the tuner's own centre and rate:

```
INF siglab: capture started serial=X310 center_hz=442387500 sample_rate_hz=200000 bandwidth_hz=200000 requested_center_hz=442387500 tuner_center_hz=441700000 tuner_rate_hz=6250000 format=flac seconds=25 protocol=tetra-dmo path=/data/siglab/captures/9f3a…flac
```

`center_hz`/`sample_rate_hz` are the file's *output* rate after any narrowband
slice; `bandwidth_hz` is the slice width (0 for full band).
`requested_center_hz` is the centre you asked for; `tuner_center_hz`/
`tuner_rate_hz` are what the SDR is physically tuned to. A slice down-converts
the live stream around `requested_center_hz` with no retuning, so the two
centres legitimately differ — the pair tells you the slice landed where you
meant. The grab closes with one of two lines. On success:

```
INF siglab: capture ended serial=X310 center_hz=442387500 sample_rate_hz=200000 format=flac samples=5000000 elapsed=25.01s recorded_seconds=25 path=/data/siglab/captures/9f3a…flac
```

`samples` and `recorded_seconds` are the receipt — 5 000 000 samples at 200 kS/s
is the 25 seconds requested — and `elapsed` confirms real time. On failure the
line escalates to `WRN siglab: capture aborted …` with an `err` naming the
cause: a device that stopped streaming, a finalize error, or "capture produced
no samples".

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The three siglab capture log lines annotated. The started line's requested center versus tuner center is the alignment pair; sample rate and bandwidth describe the slice. The ended line's samples, recorded seconds and elapsed are the receipt. The aborted line escalates to a warning whose error names the cause.">
  <rect x="14" y="40" width="652" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="24" y="56" fill="var(--fg-muted)" font-size="8" font-family="monospace">INF siglab: capture started</text>
  <text x="24" y="70" fill="currentColor" font-size="8.5" font-family="monospace">center_hz=442387500 sample_rate_hz=200000 bandwidth_hz=200000</text>
  <text x="24" y="81" fill="var(--accent)" font-size="8.5" font-family="monospace">requested_center_hz=442387500 tuner_center_hz=441700000</text>
  <line x1="120" y1="86" x2="120" y2="112" stroke="var(--accent)"/>
  <text x="24" y="126" fill="var(--accent)" font-size="9">alignment pair: did the slice land where you asked?</text>
  <line x1="430" y1="86" x2="430" y2="112" stroke="var(--fg-muted)"/>
  <text x="330" y="126" fill="var(--fg-muted)" font-size="9">slice width + output rate = the file's contents</text>
  <rect x="14" y="150" width="360" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="24" y="166" fill="var(--fg-muted)" font-size="8" font-family="monospace">INF siglab: capture ended</text>
  <text x="24" y="180" fill="currentColor" font-size="8.5" font-family="monospace">samples=5000000 recorded_seconds=25 elapsed=25.01s</text>
  <line x1="194" y1="194" x2="194" y2="214" stroke="currentColor"/>
  <text x="24" y="228" fill="currentColor" font-size="9">the receipt: the file ran to length</text>
  <rect x="392" y="150" width="274" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="402" y="166" fill="var(--fg-muted)" font-size="8" font-family="monospace">WRN siglab: capture aborted</text>
  <text x="402" y="180" fill="var(--fg-muted)" font-size="8.5" font-family="monospace">err="capture produced no samples"</text>
  <text x="402" y="228" fill="var(--fg-muted)" font-size="9">the failure names its own cause</text>
</svg>
<figcaption>The capture trio: the started line's centre pair proves alignment, the ended line's samples/recorded_seconds/elapsed prove the file is whole, and the aborted line names why it is not.</figcaption>
</figure>

## What healthy looks like

A healthy capture is three lines and no surprises — the second matching the
first. Here is a clean narrowband DMO grab, the shape the 13 Sep run produced
(a 60-second FLAC and its `debug.log` later replayed in sync):

```
INF siglab: capture started serial=X310 center_hz=438900000 sample_rate_hz=144000 bandwidth_hz=144000 requested_center_hz=438900000 tuner_center_hz=441700000 tuner_rate_hz=6250000 format=flac seconds=60 protocol=tetra-dmo path=/data/siglab/captures/dmo-13sep.flac
INF siglab: capture ended serial=X310 center_hz=438900000 sample_rate_hz=144000 format=flac samples=8640000 elapsed=60.02s recorded_seconds=60 path=/data/siglab/captures/dmo-13sep.flac
```

Three things say "good": `requested_center_hz` equals the recorded `center_hz`
(carved around 438.9 MHz though the tuner sat at 441.7 MHz); `recorded_seconds`
matches the requested `seconds`, and `samples` = 60 s × 144 kS/s; and the level
is `ended`, not `aborted`. The daemon also writes a `.metadata.json` sidecar so
the file is a fixture, the contract
[Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
calls mandatory.

### How this line shapes operator practice

- **Compare the two centres before you trust the file.** `requested_center_hz`
  matching `center_hz` is one-glance proof the capture recorded the frequency
  you meant, not the tuner's parked centre.
- **Do the sample arithmetic once.** `samples ÷ sample_rate_hz` should equal
  `recorded_seconds`; if it doesn't, the file was truncated.
- **Treat `aborted` as a stop sign.** The `err` names the cause; re-running
  blindly reproduces it. Fix the named condition first.

## When it doesn't look like that

Two capture failures cost real time on 15 Sep; both now surface in the log.

**The wrong centre.** An operator staged what they typed as a "442.8125 MHz"
grab; the only carriers sat tens of kHz off a centre of 441.7 MHz, and neither
repeater they wanted was in it. The centre had been dropped client-side — the
console coerced the field with `Number()` and fell back to `NaN` on a decimal
comma. Two fixes now stand between you and that file:
`web/siglab/src/lib/captureTuning.ts` parses the centre strictly and errors on
anything but a plain decimal, and the daemon refuses a centre *without* a
bandwidth that differs from the tuner's — a centre only means something for a
narrowband slice:

```
siglab: center_hz 442812500 without bandwidth_hz: a capture is carved from the tuner's live stream without retuning (tuner centre 441.7000 MHz), so a centre only applies to a narrowband slice — add bandwidth_hz …
```

And the start line carries `requested_center_hz` so that, even if something
slips through, the mismatch is legible in the log.

**The FLAC rate refusal.** The same session tried a `format: flac` capture of an
880 kHz slice. The capped L/M ratio landed it at 880 029 Hz — a rate FLAC's
frame header cannot encode verbatim — and the old path aborted *after 3908
samples* with `unable to encode sample rate 880029`. Now the ceiling is checked
before the tuner is pinned: a full-band FLAC over `baseband.FLACMaxSampleRateHz`
(1 048 575 Hz, the 20-bit STREAMINFO limit) is refused with a 400:

```
siglab: flac cannot carry a 6.250 MS/s stream (FLAC's STREAMINFO ceiling is 1048575 Hz) — request a narrowband slice (center_hz + bandwidth_hz) under that rate, or use cs16/wav for the full band
```

The odd-rate case (880 029 Hz, under the ceiling but not encodable in a frame
header) was the subtler bug: `FLACFrameSampleRate` now stamps the spec's
"get from STREAMINFO" code for any rate the header cannot carry directly. Both
fixes are one lesson — refuse or repair at the boundary, never fail the first
block.

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `requested_center_hz` vs `center_hz` | did the slice land at the asked-for centre | equal | they differ and `bandwidth_hz`=0 — the grab is at the tuner centre |
| `bandwidth_hz` | narrowband slice width | matches the slice you asked for | 0 when you expected a slice (full-band grab by mistake) |
| `samples` / `recorded_seconds` | how much was actually written | `samples ≈ recorded_seconds × sample_rate_hz` | `recorded_seconds` well under `seconds` — truncated |
| `elapsed` | wall time the grab took | ≈ `seconds` | far over `seconds` (device stalling) |
| level (INF vs WRN) | `ended` vs `aborted` | `INF … ended` | `WRN … aborted` with an `err` |
| `format` | container written | matches request; FLAC only under the ceiling | a FLAC request 400'd before the tuner pinned |

## Aligning a capture to the log

A headerless IQ file has no clock; the capture's start instant ties sample
offset N to a wall-clock moment in the decode log. On 12 Sep an IPSC
investigation was anchored this way: the capture began at `23:41:31.28`, so a
grant the offline replay reported at 4.20 s into the file corresponds to
`23:41:35.48` in the daemon's `debug.log`. Take the `siglab: capture started`
time, add each replayed event's stream-time offset (the DMR IPSC timeline stamps
grants in stream seconds, dibits ÷ 4800), and read across. A symptom in both
replay and live log is in the samples; one only live is downstream of the
recording tap — the discipline of
[Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
and the offline microscope of
[Signal Lab]({{ '/blog/tutorials/signal-lab-01-first-capture-no-radio/' | relative_url }}).

## The CLI capture verdicts

The `gophertrunk capture` command — a file off a dongle, no daemon — prints its
own verdict lines that double as instruments. The always-on measured-offset line
turns it into a ppm meter:

```
capture: measured carrier offset +1240 Hz from centre (≈2.7 ppm at 442.3875 MHz), consistent across 6 of 8 probe windows spread over the capture. …
```

The "consistent across N of M windows" clause is the guard: a carrier seen in
only one window is reported as a transient, because a single-window probe once
turned a momentarily-keyed neighbour into a confident wrong "≈550.3 ppm"
warning. A real ppm offset lights every window; a burst lights one.

Two more WARNINGs decide whether the file is decodable at all. Front-end
overload prints when the rail-pinned fraction crosses `siglab.ClipOverloadRatio`
(0.5%): `capture: WARNING — 38.0% of the recorded samples are pinned at the ADC
rail … NO receiver … can decode this file … re-capture with a FIXED gain … issue
#836`. And a dropped-chunk WARNING fires when the SDR shed IQ mid-grab. A capture
that clears all three you can trust to reproduce a decode; one that fails any is
a front-end problem to fix *before* blaming the decoder — the #764 discipline
this project keeps returning to.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| Capture decodes nothing; carriers off the centre you typed | centre dropped client-side or carved at the tuner centre | Set a `bandwidth_hz` for a slice; check `requested_center_hz` vs `tuner_center_hz` — [Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}) |
| `WRN siglab: capture aborted err="…encode sample rate…"` | full-band or odd-rate FLAC over what the header carries | Slice under `FLACMaxSampleRateHz`, or use cs16/wav — [Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}) |
| `capture: WARNING — N% pinned at the ADC rail` | front end overloaded (AGC or too much gain) | Re-capture at a fixed lower gain; the file is undecodable as-is (#836) |
| `capture: WARNING — dropped N IQ chunk(s)` | drain can't keep up with the native rate | Lower `-sample-rate`, widen/drop `-bandwidth`, or write to faster storage |
| `recorded_seconds` far under `seconds` | device stalled or was cancelled mid-grab | Re-run; check the SDR is streaming ([SDR doctor]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})) |
| Replay disagrees with the live log at the same instant | fault is downstream of the recording tap | Align on the capture-started time and read both logs — [Signal Lab Part 2]({{ '/blog/tutorials/signal-lab-02-reading-the-dashboard/' | relative_url }}) |

## Where this goes next

A capture proves what reached the decoder; the *recorder* proves what reached
the disk. [Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
reads the `recorder: call started` / `call ended` family — the `reason=` field,
the `audio_pct` that catches a recording shorter than its call span, the
dead-key drops that leave nothing on disk, and the retention sweeper lines.

## FAQ

**What does requested_center_hz mean in a GopherTrunk capture log?**
The centre you asked the capture to record, printed on the `siglab: capture
started` line next to `tuner_center_hz` (the SDR's physical centre). For a
narrowband slice they differ; if they differ with no bandwidth, the grab
recorded the tuner centre, not what you typed.

**Why did my FLAC capture abort with "unable to encode sample rate"?**
FLAC's frame header cannot carry arbitrary rates, and its STREAMINFO ceiling is
1 048 575 Hz. A full-band multi-MS/s FLAC is now refused with a 400, and odd
sub-ceiling rates (e.g. 880 029 Hz) get the spec's STREAMINFO fallback code
instead of failing the first block. Slice under the ceiling, or use cs16/wav.

**How do I line an IQ capture up with the daemon's decode log?**
Take the `siglab: capture started` timestamp as sample offset zero, then add
each replayed event's stream-time offset (seconds = dibits ÷ symbol rate, which
the replay harnesses print). A symptom in both replay and live log is in the
samples; one only live is downstream of the recording tap.

**What sample rate proves my capture is complete?**
`samples` should equal `recorded_seconds × sample_rate_hz` on the `capture
ended` line, and `recorded_seconds` should match the `seconds` you requested. A
short `recorded_seconds`, or a `WRN … aborted`, means the file is truncated.

**Does the capture command measure my tuner's ppm error?**
Yes. `gophertrunk capture` prints a measured carrier offset consistent across
several probe windows; if the tuned frequency is a known-accurate carrier, that
offset is your tuner's ppm error for `-ppm` / `sdr.ppm`. A one-window carrier is
flagged transient, not reported as ppm.

## Series navigation

**Part 9 of 14** · ←
[Part 8: The MRC Health Line]({{ '/blog/tutorials/field-notebook-08-mrc-health-line/' | relative_url }})
· Next →
[Part 10: Recorder Lines — Spans, Segments & Dead Keys]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
