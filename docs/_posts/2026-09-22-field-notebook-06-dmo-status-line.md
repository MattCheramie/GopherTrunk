---
title: "The Field Notebook, Part 6: The DMO Status Line — Noise Meters & Traffic Counters"
description: "How to read GopherTrunk's tetra dmo decode status line — dsb_total against dsb_schs_crc, the dnb_total noise meter against dnb_qualified, tch_crc, distinct_fn and the per-transmission scramble-seed fields — plus the seed recovered, seed changed and voice-follow lines around it, what healthy DMO prints, and when a counter means a defect."
category: tutorials
keywords: tetra dmo decode status, dnb_total noise meter, dnb_qualified traffic, tetra dmo scramble seed log, solve_rejects, seed_verified gophertrunk, tetra direct mode log reading, tetra dmo grant line, gophertrunk dmo status line
tags: [field-notebook, logs, tetra, dmo, direct-mode, diagnostics, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 6
---

*Part 6 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's debug.log one line family at a time — what each field measures,
what a healthy rig prints, which number is a noise meter and which one means
traffic, and which deep dive to open when a line goes wrong.
[Part 5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }})
read the DMR activity line, where every counter was decode evidence. This
part reads the one line in the log where a counter is **designed to race on
an empty channel**: the TETRA DMO decode-status line, the scramble-seed lines
that replaced its colour code, and the composer's voice-follow bookends.*

> **TL;DR:** `tetra dmo: decode status` (`internal/scanner/ccdecoder/pipelines_dmo.go`,
> `maybeLogStatus`, every 5 s at DEBUG) pairs each detector with its
> verifier: `dsb_total`/`dsb_schs_crc` grade the sync bursts, `dnb_total` is
> a **noise meter** (the correlator fires ~18×/s on an idle channel by
> construction), `dnb_qualified` counts bursts on the learned 255-dibit slot
> grid — the number that means traffic — and `tch_crc` counts CRC-valid
> speech blocks. `colour`/`colour_known` are
> gone: the TCH/S scramble seed is **per transmission** (the radio's source
> address under a fixed prefix), so the line prints `seed`, `seed_known`,
> `seed_verified`, `bursts_solved` and `solve_rejects` from
> `tetra.DMSeedTracker`. `scramble seed recovered` fires once per PTT; a
> `seed changed` flip-flop was a false soft solve, now counted in
> `solve_rejects`. Live DMO is on-air verified (13 Sep); the pre-roll and
> grant-on-adoption changes are capture-verified, live run pending
> ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)).

**Key takeaways**

- **`dnb_total` racing while `dsb_schs_crc` stands still is one conclusion,
  not two facts.** 1076→4541 in 185 s (18.7/s) beside a frozen 46 is noise;
  nothing is transmitting.
- **`seed_known` is not `seed_verified`.** Known can be a fallback the voice
  chain installed to stop buffering; verified means an exact solve or three
  CRC-confirmed decodes at the announced seed.
- **One `seed recovered` per PTT is healthy; a `seed changed … changed back`
  pair is not.** Every transmission carries a new seed; a change mid-PTT
  that reverts was a false solve.
- **`tch_crc` at the chance floor with signalling healthy is a decode
  question, not an encryption verdict.** This path was burned by that
  reading twice; the seed solver needs no configuration.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The status line | detectors beside verifiers, seed state, 5 s DEBUG | `pipelines_dmo.go` (`maybeLogStatus`, `dmoStatusInterval`) |
| Noise-meter math | ~18 spurious DNB leads/s; grid vote latches after 6 | `internal/radio/tetra/dmo_grid.go` (`DMSlotGrid`, `dmGridMinTrain`) |
| Seed tracking | exact solve adopts at once; hint after 3 CRC-valid decodes | `internal/radio/tetra/dmo_seed_tracker.go` (`DMSeedTracker`, `dmSeedHintMinCRC`) |
| Grant edge | fires on seed adoption or 4 qualified DNBs, re-arms after 3 s | `maybeGrant`, `dmoGrantMinDNB`, `dmoGrantRearm` |
| Voice bookends | `voice follow started` / `ended` with the chain's own seed state | `composer/tetra_dmo_voice.go` |
| The background | how each field earned its place | [TETRA End to End 11]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }}), [12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}), [13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}) |

## In this post

- **What this line is telling you** — five ratios and a seed.
- **What healthy looks like** — a camped channel, then one PTT end to end.
- **Field by field** — measures, healthy, worry-when.
- **When it doesn't look like that** — the 20 Aug race, the fallback that
  looked like a recovery, the flip-flop; then symptom → cause → read.

## What this line is telling you

[Cookbook Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})
introduced the pairing — `dnb_total` is a noise meter, `dnb_qualified` means
traffic — and
[Spec to Shipping 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
made it a design rule. This part reads the line as it prints today, because
two fields both posts quoted no longer exist:

```go
// internal/scanner/ccdecoder/pipelines_dmo.go (shape) — maybeLogStatus
p.log.Debug("tetra dmo: decode status",
    "system", p.system, "locked", p.locked,
    "carrier_off_hz", math.Round(p.rx.CarrierOffsetHz()*10)/10,
    "dsb_total", p.dsbTotal, "dsb_schs_crc", p.dsbCRC,   // sync bursts: found / CRC-valid
    "dnb_total", p.dnbTotal, "dnb_qualified", p.dnbQualified, // NOISE METER / on the slot grid
    "tch_crc", p.tchCRC,                                  // CRC-valid speech blocks
    "distinct_fn", len(p.fnSeen),                         // frame counter advancing = real lock
    "seed", fmt.Sprintf("%#010x", p.colour), "seed_known", p.colourKnown,
    "seed_verified", p.seeds.Verified(),
    "bursts_solved", p.seeds.Solved, "solve_rejects", p.seeds.SolveRejects,
    "grant_active", p.grantActive)
```

The rename matters. The earlier line printed `colour=3 colour_known=true`,
and every "colour" it ever recovered — 3, 39, 36, 31 — was a
partial-keystream artifact. The 12 Sep three-PTT capture, solved burst by
burst with an exact GF(2) solver (`tetra.SolveTCHScrambleSeed`), gave three
seeds for three PTTs — `0x012915c0`, `0x015d9c07`, `0x01671384` — each the
transmitting radio's **source address** under a fixed `000001` prefix,
announced in the DSB's SCH/H. No colour or MNI config could describe that.
So the field is a 30-bit seed printed `%#010x`, and its two flags differ:
`seed_known` is "the tracker has *something* to decode at", `seed_verified`
is "traffic confirmed it".

The noise meter is arithmetic. The DNB correlator matches an 11-dibit
training sequence at tolerance 2 under 8 filters, so
`Σ_{k≤2} C(11,k)·3^k = 529` of `4^11` patterns match by chance — ~18 false
leads per second at 18 kdibit/s. `DMSlotGrid` (`dmo_grid.go`) votes leads
onto the 255-dibit slot grid: one radio puts every burst on one residue,
noise spreads over all 255 (~0.21 per residue per second), and six agreeing
leads latch — ~340 ms into a real train. `dnb_qualified` counts leads on the
latched residue; the latch drops after `dmGridTrainGap`.

## What healthy looks like

A camped DMO channel between transmissions, as the current line would render
the 20 Aug operator's idle counters:

```
INF cchunt: camped on conventional channel — idle, waiting for traffic system=Site-DMO
DBG tetra dmo: decode status system=Site-DMO locked=true carrier_off_hz=-412.5 dsb_total=54 dsb_schs_crc=46 dnb_total=4541 dnb_qualified=0 tch_crc=0 distinct_fn=17 seed=0x00000000 seed_known=false seed_verified=false bursts_solved=0 solve_rejects=0 grant_active=false
```

`dnb_total` in the thousands next to `dnb_qualified=0` is a **healthy idle
channel**. The lock is sticky (no `cc.lost` between PTTs) and `distinct_fn`
proves it real: seventeen frame numbers cannot be a chance CRC. Now one
13 Sep PTT, end to end:

```
INF tetra dmo scramble seed recovered seed=0x01498b2a source_field=0x498b2a exact_adopts=1 hint_adopts=0 system=Site-DMO
INF tetra dmo grant (traffic detected) freq=438900000 seed=0x01498b2a seed_known=true system=Site-DMO
INF composer: tetra DMO voice follow started — DNB TCH/S decode + ACELP vocoder serial=cc:same-carrier:1 seed_hint=0x01498b2a rate_hz=18000
DBG tetra dmo: decode status system=Site-DMO locked=true carrier_off_hz=-410.8 dsb_total=60 dsb_schs_crc=52 dnb_total=4630 dnb_qualified=61 tch_crc=58 distinct_fn=18 seed=0x01498b2a seed_known=true seed_verified=true bursts_solved=57 solve_rejects=0 grant_active=true
INF composer: tetra DMO voice follow ended serial=cc:same-carrier:1 dnb_bursts=64 speech_frames=116 bfi_count=6 seed=0x01498b2a seed_known=true seed_verified=true seed_from_pipeline=false bursts_solved=58 solve_rejects=0 exact_adopts=1 hint_adopts=0
```

Read the order. The seed lands on the **first** grid-qualified DNB — an
exact solve is a 128-check-redundant proof that the burst is a TCH/S
codeword under that seed — and since the 13 Sep round `maybeGrant(true)`
fires on that adoption instead of waiting for `dmoGrantMinDNB=4` bursts,
which cost three decodable bursts per PTT offline and 1.0–1.8 s live. `source_field=0x498b2a` is the seed's low
24 bits. The composer starts with `seed_hint` from the grant; its ended line
carries the chain's **own** tracker state — `seed_from_pipeline=false`
because its own bursts solved it, `speech_frames=116` for 64 bursts being
two frames per CRC-valid DNB with six BFIs.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two rows of counters over one DMO transmission. Top: dnb_total climbs steadily at about eighteen per second before, during and after the PTT — a noise meter. Middle: dnb_qualified is flat at zero, rises only during the PTT once the slot grid latches, and stops when the train ends. Bottom: the seed tracker adopts the seed on the first qualified burst, the grant fires on that adoption, and a rejected false solve mid-transmission is counted in solve_rejects instead of changing the seed.">
  <text x="8" y="40" fill="var(--fg-muted)" font-size="9">dnb_total</text>
  <polyline points="70,70 200,60 330,48 460,38 600,28" fill="none" stroke="var(--fg-muted)"/>
  <text x="380" y="30" fill="var(--fg-muted)" font-size="8">~18/s always: noise meter</text>
  <text x="8" y="120" fill="currentColor" font-size="9">dnb_qualified</text>
  <polyline points="70,130 300,130 330,122 400,100 470,84 500,84 600,84" fill="none" stroke="currentColor"/>
  <text x="80" y="126" fill="var(--fg-muted)" font-size="8">0: nothing on the grid</text>
  <text x="505" y="92" fill="var(--fg-muted)" font-size="8">train ends → latch dropped</text>
  <line x1="300" y1="20" x2="300" y2="200" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="304" y="150" fill="var(--accent)" font-size="8">PTT: DSBs, then DNBs</text>
  <line x1="330" y1="20" x2="330" y2="200" stroke="var(--accent)" stroke-dasharray="2 4"/>
  <text x="334" y="162" fill="var(--accent)" font-size="8">grid latches (6 leads)</text>
  <rect x="332" y="176" width="150" height="20" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="407" y="190" text-anchor="middle" fill="var(--accent)" font-size="8">exact solve → seed adopted → grant</text>
  <rect x="490" y="176" width="110" height="20" rx="3" fill="none" stroke="currentColor"/>
  <text x="545" y="190" text-anchor="middle" fill="currentColor" font-size="8">false solve → solve_rejects</text>
</svg>
<figcaption>One transmission through the status line: the noise meter never stops, the qualified counter moves only on the slot grid, and the seed is adopted on the first exact solve — the grant now rides that adoption.</figcaption>
</figure>

## Field by field

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `locked` | DSB SCH/S CRC + SYNC PDU lock; sticky | `true` once any radio has keyed | never `true` while radios transmit — RF, frequency, gain |
| `dsb_total` / `dsb_schs_crc` | sync bursts detected / CRC-valid | ~90% (46/54, 105/117 on captures) | ratio collapsing — marginal signal |
| `dnb_total` | raw DNB correlator leads | climbs ~18/s **always** | it stops — the receiver stopped |
| `dnb_qualified` | leads on the latched slot grid | 0 idle; ~17/s in a PTT | climbs on a silent channel |
| `tch_crc` | CRC-valid TCH/S blocks | ≈ `dnb_qualified` once the seed is known | near 0 with `dnb_qualified` high — wrong seed or marginal signal |
| `distinct_fn` | distinct SYNC-PDU frame numbers | advances to 18 | stuck at 1–2 — a chance lock |
| `seed` / `seed_known` / `seed_verified` | tracker state | verified within one qualified burst | `known=true verified=false` — a fallback, decoding blind |
| `bursts_solved` / `solve_rejects` | exact solves / seed-changing solves that failed their own decode | rejects 0–1 per PTT | rejects climbing — a weak stretch |
| `grant_active` | the edge is latched | flips per PTT, re-arms after 3 s | stuck `true` on silence — the train never ended |

`tch_crc` is fed by the tracker's own decode, so it rises at a wrong seed
only by chance — the 8-bit CRC admits 1/256 at random, and a *related* wrong
seed CRC-passes ~9% of bursts (measured), which is why a CRC count was never
proof of a seed. `solve_rejects` exists because the soft-assisted solver's
~48-bit local parity windows overlap so heavily that its claimed 2^-24 false
rate was fiction: 7 of 1411 DNBs on the 13 Sep capture "solved" to seeds
that decode nothing.

## When it doesn't look like that

Three unhealthy readings, each from a real log. First, the **20 Aug race**,
the line that taught the pipeline its noise meter:

```
DBG tetra dmo: decode status … dsb_total=54 dsb_schs_crc=46 dnb_total=1076 dnb_qualified=0 tch_crc=0 …
DBG tetra dmo: decode status … dsb_total=54 dsb_schs_crc=46 dnb_total=4541 dnb_qualified=0 tch_crc=0 …
```

185 s apart: `dnb_total` +3465 (18.7/s) with `dsb_schs_crc` frozen at 46.
The first live build had no `dnb_qualified`; it granted on four raw DNBs
~230 ms after startup on a silent channel and never again, because the 3 s
re-arm drought could never elapse against a 55 ms mean inter-arrival.
`dnb_qualified` climbing while `dsb_*` freezes would be the same defect in a
new dress.

Second, the **fallback that read as a recovery**. The voice chain gives up
buffering at `dmoVoiceSeedMax=120` DNBs and installs a fallback:

```
INF composer: tetra DMO seed adopted from control pipeline (not confirmed on this chain's bursts) serial=cc:same-carrier:1 seed=0x00000027
INF composer: tetra DMO voice follow ended serial=cc:same-carrier:1 dnb_bursts=242 speech_frames=0 bfi_count=242 seed=0x00000027 seed_known=true seed_verified=false seed_from_pipeline=true …
```

That is the 20 Aug silent recording in today's vocabulary: 242 bursts, zero
speech, every DNB a BFI. The old line said `colour=0 colour_known=true`,
which read as "recovered colour 0" and sent the investigation after the
descrambler; `seed_known=true seed_verified=false seed_from_pipeline=true`
says the chain never confirmed anything on its own bursts. Zero speech also
explains "ends at a timeout": `boundaryTracker.onVoice` is never called, so
hangtime tears the call down.

Third, the **flip-flop**, from the 13 Sep live run:

```
INF composer: tetra DMO scramble seed changed serial=cc:same-carrier:1 seed=<a seed that decodes nothing>
INF composer: tetra DMO scramble seed changed serial=cc:same-carrier:1 seed=0x01733855
```

Two of five calls did this (the false seed is elided; the second line is
that PTT's real seed coming back). Every flip cost the flipped burst's speech
and, on a weak stretch, every burst until the next clean solve. A
seed-changing solve now has to CRC-decode the very burst it came from, so
the same event today shows as `solve_rejects=1` in the ended line, the burst
decoded at the known seed — pinned against two literal capture bursts by
`TestDMSeedTrackerRejectsSolveThatDoesNotDecode`.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| `dnb_total` climbing, everything else flat | idle channel — noise meter working | nothing; [Cookbook 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }}) |
| `dnb_qualified` climbing with `dsb_*` frozen | grid latched on noise or a foreign burst | save the log; [TETRA End to End 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}) |
| `tch_crc≈0`, `dnb_qualified` high, `seed_verified=false` | no exact solve — every burst errored; marginal RF | gain/antenna first; [TETRA End to End 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}), [TETRA scrambler]({{ '/reference/tetra-scrambler/' | relative_url }}) |
| `speech_frames=0`, `seed_from_pipeline=true` | voice chain never confirmed a seed; silent recording, hangtime end | [Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}) |
| `seed changed … changed` pairs mid-PTT | false soft solve on an old build | upgrade; `solve_rejects` counts them |
| Grant ~1–2 s into a PTT; `colour_known` in the log | pre-seed build waiting for 4 qualified DNBs, no pre-roll | upgrade; the 1 s pre-roll is capture-measured, live run pending (#1003); [Issue Tracker 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}) |

### How this line shapes operator practice

- **Start from `dnb_qualified`, never `dnb_total`.** A question that begins
  with the raw count begins with a noise meter.
- **Ask `seed_verified`, not `seed_known`.** Known is a decoding decision;
  verified is evidence. Report both with `seed_from_pipeline`.
- **A new seed per PTT is the protocol.** Do not pin `tetra_colour_code` to
  a seed you saw once.
- **Chance-floor speech on a clear channel is a defect report, not an
  encryption verdict.** `bursts_solved=0` beside a healthy `dsb_schs_crc`
  is worth a capture.

## Where this goes next

Every counter here assumes the pipeline keeps up with the tap. When it does
not, the log says so from a different direction —
[Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
reads `soapyremote: SDR overruns … host_drops`, the `decode can't keep up`
WARN and the runtime heartbeat, and shows why this very voice chain's colour
brute force once starved its own tap.

## FAQ

**Why does `dnb_total` keep climbing when nobody is transmitting?**
Because the traffic-burst correlator is deliberately loose — 11 dibits at
tolerance 2 under eight filters — and fires on noise about 18 times a second
by arithmetic. It is a noise meter proving the correlator is alive;
`dnb_qualified`, counting only bursts on the learned slot grid, is the field
that means traffic.

**What happened to `colour` and `colour_known` in the DMO status line?**
They became `seed`, `seed_known` and `seed_verified`. The TCH/S scramble
seed turned out to be per transmission — the radio's source address under a
fixed prefix — so a "colour code" could never describe it, and every colour
the old brute force recovered was a partial-keystream artifact.

**What does `solve_rejects` count?**
Seed-changing solves from the soft-assisted solver that failed to CRC-decode
their own burst. On the 13 Sep capture 7 of 1411 bursts produced such false
seeds and caused the "seed changed … changed back" flip-flops; they are now
rejected and the burst decodes at the known seed.

**Is `seed_known=true` with `seed_verified=false` a problem?**
It means the tracker is decoding at a fallback — the control pipeline's hint
or seed 0 — that traffic never confirmed on this chain's bursts. Paired with
`speech_frames=0` it explains a silent recording that ends on hangtime. A
later exact solve replaces the fallback.

**Is GopherTrunk's DMO decode verified on air?**
The core path is: five consecutive clear PTTs decoded live on 13 Sep, the
pipeline's seeds matching the offline replay burst for burst. Still
capture-verified only, live run pending under #1003: the 1 s pre-roll and
granting on seed adoption.

## Series navigation

**Part 6 of 14** · ←
[Part 5: The DMR Activity Line]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }})
· Next →
[Part 7: Overruns & host_drops — Reading a Downstream Signal]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
