---
title: "Signal Lab, Part 9: Dissecting P25 — TSBK PDUs, Receiver States & the Sync Landscape"
description: Signal Lab's deepest P25 view — collect_pdus surfaces every TSBK signaling block (decoded and CRC-failed) as a field-level dissection with a csv-pdus export, plus the C4FM and CQPSK receiver-state series, the soft-eye verdict, and the sync-landscape heatmap.
category: tutorials
keywords: p25 tsbk, collect_pdus, csv-pdus, pdu dissection, receiver state, afc agc, gardner timing, cma error, cqpsk, soft eye, sync landscape, p25 signaling
tags: [siglab, p25, tsbk, receiver-state, dsp, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 9
charts: true
---

*Part 9 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. This is the advanced end of the pool — the tools for
taking a P25 control channel apart down to the signaling block and the receiver
loop.*

> **TL;DR:** With `collect_pdus`, Signal Lab surfaces **every** P25 TSBK
> signaling block — decoded *and* CRC-failed — as a field-level dissection
> (opcode, source/dest/talkgroup, FEC/CRC status, raw payload), filterable and
> exportable as `csv-pdus`. Alongside it, the **receiver-state series** exposes
> the demod loop internals — AFC/AGC/M&M for the C4FM path, carrier/Gardner/CMA
> for the CQPSK path — plus a **soft-eye verdict** and a **sync-landscape
> heatmap** of variant, rotation, and hits.

**Key takeaways**

- **`collect_pdus` keeps the failures.** CRC-failed TSBKs are surfaced too — the
  errors are often more informative than the successes.
- **Field-level dissection** means opcode, addresses, FEC/CRC status, and raw
  payload for every block, filterable in the console.
- **`csv-pdus` exports** the whole dissection for offline analysis.
- **Receiver-state series differ by path** — C4FM (AFC/AGC/M&M/DDA) vs CQPSK
  (carrier/Gardner/CMA AGC/CMA error).
- **The sync-landscape heatmap** shows which sync variant and rotation the
  correlator is finding hits on.

## Cheat sheet

| Surface / knob | What it does |
|---|---|
| `collect_pdus` | Surface every P25 TSBK (decoded + CRC-failed) |
| `csv-pdus` | Export the PDU dissection as CSV |
| Receiver-state series (C4FM) | AFC, AGC level/target, M&M mu/sps, DDA |
| Receiver-state series (CQPSK) | Carrier Hz, Gardner mu/sps, CQPSK AGC gain, CMA error |
| Soft-eye verdict | Demod's own eye-quality judgment |
| Sync-landscape heatmap | Variant × rotation × hits |

## In this post

- **Why the CRC-failed PDUs matter** — the errors are the signal.
- **The TSBK dissection** — opcode to raw payload.
- **Receiver-state series** — watching the demod loop think.
- **C4FM vs CQPSK internals** — two paths, two state vectors.
- **The sync landscape** — where the correlator finds alignment.

## Why the CRC-failed PDUs matter

A normal decoder throws away what it can't verify. A P25 TSBK — a Trunking
Signaling Block, the control channel's unit of signaling — that fails its CRC
gets silently dropped, and the operator sees only the clean traffic. For running a
system that's fine; for *diagnosing* one it's the opposite of what you want.

`collect_pdus` inverts that. It surfaces **every** TSBK the demod recovered,
decoded and CRC-failed alike, because the failures carry the diagnosis. A control
channel that's *almost* decoding — TSBKs arriving but failing CRC in a steady
trickle — is a completely different problem from one that's silent, and only a
tool that keeps the failures can tell them apart. A burst of CRC failures lined up
with a fade in the EVM-vs-symbol trace (Part 7) or a dropout in the spectrogram
(Part 5) localizes the fault precisely. The garbage is the evidence.

## The TSBK dissection

Each collected TSBK is presented as a **field-level dissection**, not a hex blob:

| Field | What it is |
|---|---|
| **Opcode** | The TSBK type — grant, affiliation, status, etc. |
| **Source / destination** | The addressed radio IDs |
| **Talkgroup** | The group the block concerns |
| **FEC / CRC status** | Whether error-correction and the CRC passed |
| **Raw payload** | The underlying bytes, for when the parse is wrong |

The dissection is **filterable** in the browser console — narrow to a single
opcode, or to just the CRC failures, or to one talkgroup — which turns a wall of
signaling into a query. And it's **exportable as `csv-pdus`**, so you can pull the
whole set into a spreadsheet or a script and do statistics across a long capture:
how many grants, what fraction failed CRC, which opcodes cluster around the bad
moments. Keeping the raw payload alongside the parsed fields is the escape hatch —
when a field looks wrong, you check it against the bytes rather than trusting the
parser.

## Receiver-state series: watching the loop think

Beneath the PDUs, Signal Lab exposes the demod's own **receiver-state series** —
time series of the internal loop variables as they track the signal. This is the
demodulator narrating its own work: not "did it lock?" but "*how* did it lock, and
how hard was it working to stay locked?" A control loop that's fighting — AGC
hunting, timing error oscillating — produces a jittery state series even when it
technically holds lock, and that instability is an early warning the summary
metrics smooth over.

Because P25 has two distinct demod paths, there are two distinct state vectors,
and which one you're reading tells you which path is in play.

## C4FM vs CQPSK internals

The two paths track a signal with different machinery, so their state series
report different quantities:

**C4FM path** (4-level FM, the four-rail family from Part 4):

- **AFC** — automatic frequency control: the residual carrier correction the loop
  is applying.
- **AGC level / target** — the gain loop's current level against its target.
- **M&M mu / sps** — the Mueller & Müller timing recovery's fractional interval
  (`mu`) and samples-per-symbol estimate.
- **DDA** — the decision-directed adaptation term.

**CQPSK path** (phase modulation, the four-cluster family):

- **Carrier Hz** — the tracked carrier offset.
- **Gardner mu / sps** — the Gardner timing recovery's interval and
  samples-per-symbol.
- **CQPSK AGC gain** — the path's automatic gain.
- **CMA error** — the constant-modulus-algorithm equalizer's error term.

Seeing a *carrier Hz* and a *CMA error* series tells you the capture went down the
CQPSK path; seeing *AFC* and *M&M mu* tells you C4FM. A capture whose CMA error
won't settle is an equalizer struggling with multipath; a C4FM capture whose M&M
`mu` won't stop drifting is a timing-recovery problem — often the tail of a
sample-rate mismatch the Part 2 baud-deviation check should have caught. The state
series is where a stubborn "won't stay locked" turns from mysterious into
mechanistic.

## The soft-eye verdict and the sync landscape

Two more advanced readouts round out the P25 view.

The **soft-eye verdict** is the demodulator's own judgment on eye quality — a
distilled pass/marginal/fail on whether the decision margin (the open eye from
Part 4) is healthy, expressed as a verdict rather than a plot. It's the quick
"does the demod think this is clean?" to complement your own read of the eye
diagram.

The **sync-landscape heatmap** visualizes how the frame synchronizer is finding
alignment. P25 sync can appear in different **variants** and **rotations** (the
constellation can lock in any of several rotational ambiguities), and the heatmap
plots **hits** across that variant × rotation grid. A healthy capture concentrates
hits in one cell — one variant, one rotation — a clear, unambiguous sync. Energy
smeared across the grid means the synchronizer is finding weak, conflicting
correlations: it isn't sure where the frame starts, which is a lock that's about
to fail.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="heatmap" width="560" height="300" role="img"
        aria-label="Sync-landscape heatmap with hits concentrated in one variant and rotation"></canvas>
<script type="application/json" class="lab-chart-data">
{ "xlabel":"rotation","ylabel":"sync variant",
"matrix":[
[0.05,0.06,0.04,0.05],
[0.07,0.12,0.06,0.05],
[0.06,0.95,0.08,0.06],
[0.05,0.09,0.05,0.04]] }
</script>
<figcaption>A healthy sync landscape: hits concentrate in a single variant/rotation cell — the synchronizer is confident where the frame begins. A smear across cells means an uncertain, failing sync.</figcaption>
</figure>

Reese uses the sync landscape as his last stop on a hard capture. If the state
series are jittery, the soft-eye verdict is marginal, and the sync landscape is
smeared, the three agree: the capture is genuinely at the edge, and no knob will
save it — which, per the rate-invariance rule, sends you back to the recording.

## A PDU triage workflow

The PDU dissection is most powerful when you stop reading it block by block and
start querying it in aggregate. A practical triage on a suspect control channel
runs like this. First, export with `csv-pdus` and count opcodes: a healthy P25
control channel is mostly the steady heartbeat of network-status and
identifier-update blocks with grants sprinkled in; a wildly skewed distribution —
say, almost nothing but one opcode, or a flood of blocks that don't parse — is
itself a diagnosis. Second, compute the CRC-failure *fraction*, not the count: a
handful of failures on a long capture is normal, but a rising fraction is a
channel slipping toward the edge, and cross-referencing *when* those failures
cluster against the EVM-vs-symbol trace (Part 7) usually points straight at the
cause.

Third — and this is where keeping the raw payload pays off — spot-check a few
blocks whose parsed fields look implausible against their raw bytes. A talkgroup
that reads as a nonsense value on a block that *passed* CRC is a hint that the
opcode was misidentified or the dissector met a variant it doesn't fully model;
the raw payload lets you confirm which. Ada's first instinct was to trust every
green-CRC field completely; Reese's habit is to trust the CRC for *integrity* but
still eyeball the raw payload when a field surprises him, because a valid CRC only
proves the bits arrived intact, not that the parser read them the way the system
meant them.

Done as a set, those three passes — distribution, failure trend, raw spot-check —
turn a scrolling wall of TSBKs into a one-paragraph verdict on the channel's
health, and they compose directly with everything earlier in the series: the
spectrogram says *when* the channel was busy, the VSA says *how clean* it was, and
the PDU dissection says *what it was actually saying*. That's the whole point of a
workbench — the views reinforce each other, and a hard capture rarely survives all
of them pointing at the same moment.

## Where this goes next

You've now taken a single P25 capture apart to the signaling block and the loop
variable. [Part 10]({{ '/blog/tutorials/signal-lab-10-demod-bench-sweep/' | relative_url }})
zooms back out to the whole demodulator: `gophertrunk siglab sweep` synthesizes
P25 across an SNR ladder on both demod paths and measures demod quality against
theory — the regression instrument that ties the series off. The
[SigLab docs]({{ '/siglab.html' | relative_url }}) document `collect_pdus` and
`csv-pdus` in full.

## FAQ

**Why surface CRC-failed PDUs at all?**
Because they're diagnostic. A channel that's producing TSBKs that fail CRC is a
different (and more recoverable) problem than a silent one, and lining CRC
failures up with fades or dropouts localizes the fault. A tool that drops
failures hides exactly the evidence you need.

**What's in a TSBK dissection?**
Opcode, source/destination, talkgroup, FEC and CRC status, and the raw payload —
per block, filterable in the console and exportable via `csv-pdus`.

**How do I tell which demod path a capture used?**
Read the receiver-state series. AFC and M&M mu/sps mean the C4FM path; carrier Hz,
Gardner mu/sps, and CMA error mean the CQPSK path.

**What does a smeared sync landscape mean?**
The synchronizer is finding conflicting, weak correlations across sync variants
and rotations instead of one clear hit — an uncertain sync that's likely to lose
lock. Concentrated hits in one cell is healthy.

## Series navigation

**Part 9 of 10** · ←[Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }}) · Next →
[Part 10: The Demod Bench]({{ '/blog/tutorials/signal-lab-10-demod-bench-sweep/' | relative_url }})
