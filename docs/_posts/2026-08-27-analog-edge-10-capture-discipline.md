---
title: "The Analog Edge, Part 10: Capture Discipline — cfile, cs16, SigMF & Metadata"
description: "Why \"get the raw capture\" is the most repeated sentence in the GopherTrunk issue tracker, the IQ formats and sidecar metadata that make a capture usable, the tap points that decide what a recording can prove, and how a good capture turns a complaint into a regression test."
category: tutorials
keywords: sdr iq capture, cfile format, cs16 raw iq, sigmf metadata, capture sample rate, iq recording sidecar, replay capture gophertrunk, pre-combine diversity capture, gophertrunk analog edge
tags: [analog-edge, capture, iq, formats, metadata, workflow]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 10
---

*Part 10 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system. [Part 9]({{ '/blog/tutorials/analog-edge-09-filters-lnas/' | relative_url }})
finished the hardware chain — antenna, feedline, filter, LNA, in the right
order. Our marginal reader's system is better, but one symptom survives, and
we've reached the point where arguing about it is worthless. This part is
about the skill that settles arguments: recording the actual samples. Half the
hard bugs in this project's tracker were "in the samples" — and every one of
them was solved the day someone captured them.*

> **TL;DR:** A useful capture is **raw IQ, at the same rate and gain as the
> failure, with a sidecar that says what it is**. GopherTrunk's formats:
> `.cfile` (GNU Radio float32, 8 bytes/sample), **cs16** (int16, 4
> bytes/sample — the default for `baseband.auto_record`), and `u8` (rtl_sdr
> native); none embeds rate or frequency, so a `.metadata.json` sidecar
> (`sample_rate_hz`, `center_freq_hz`) is **mandatory**, and `gophertrunk
> capture` writes one automatically (plus an optional SigMF sidecar via
> `-bundle -sigmf`). Tap point decides what a capture can prove: post-demod
> audio proves nothing for phase modulations, post-combine IQ can't evaluate a
> combiner, and only `diversity_capture`'s pre-combine tap answers diversity
> questions. Done right, a capture becomes a skip-gated regression test in
> `samples/` that keeps the bug fixed forever.

**Key takeaways**

- **Audio is not a capture.** For TETRA (π/4-DQPSK), P25, DMR, NXDN — anything
  carrying information in phase or 4-level amplitude — an MP3/WAV of demodulated
  audio has already destroyed the evidence. It must be IQ.
- **Capture the failure, not a nicer version of it.** Same sample rate, same
  gain, same antenna. The [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
  deficit only existed in 10 MS/s captures — a "cleaner" 2.5 MS/s recording
  would have proven the bug didn't exist.
- **A capture without a sidecar is a puzzle, not evidence.** Headerless IQ has
  no rate, no center frequency, no provenance; the loader can't even guess.
  `sample_rate_hz` and `center_freq_hz` are the two non-negotiable fields.
- **The tap point is part of the experiment design.** Wideband tap, per-channel
  `ddc` tap, pre-combine branch tap — each answers different questions, and
  picking the wrong one produces a file that answers none.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| One-shot manual capture | live SDR → `.cfile`/cs16 + metadata sidecar | `gophertrunk capture` (`cmd/gophertrunk/capture.go`) |
| Event-driven capture | auto-record IQ on sync loss, encrypted grant, … | `baseband.auto_record` (`config.example.yaml`) |
| Replay a capture | run any capture through the real decode path | `gophertrunk replay -in <f> -format u8\|f32\|cs16\|wav` |
| Shrink for sharing | post-DDC narrowband WAV of the exact decoded stream | `replay -record-ddc` |
| SigMF interop | emit a `.sigmf-meta` sidecar for external tooling | `gophertrunk capture -bundle <p> -sigmf` (`internal/gtbundle/sigmf.go`) |
| Pre-combine diversity tap | per-branch IQ before the MRC combiner | `sdr.soapy_remote[].diversity_capture` (`internal/sdr/soapyremote/branchcapture.go`) |
| What each decoder needs | per-protocol format / rate / duration / pass bar | [decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }}), `samples/*/README.md` |

## In this post

- **Why the tracker keeps asking for captures** — evidence vs anecdote.
- **The formats** — cfile, cs16, u8, WAV, and what each costs.
- **The sidecar** — metadata that makes a file mean something.
- **Tap points** — wideband, ddc, and the pre-combine rule.
- **From capture to regression test** — how a file keeps a bug fixed.

## Why the tracker keeps asking for captures

"Please attach the raw capture" is the most repeated sentence in this
project's issue tracker, and it isn't bureaucracy. A symptom report is a
description of one observer's decode chain on one afternoon. A capture is the
*input itself* — replayable through any build, any branch, any candidate fix,
forever. The difference decided every hard case this series has drawn on: the
[ten-megasamples investigation]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
lived and died by the reporter's own `.cfile`s (the ~10 dB deficit was *in the
captured samples*, provable only by replaying them); the
[rail-pinned ADC]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
overturned a beautifully argued hypothesis because the raw capture was 50%
pinned to the rails; and the DMR two-slot work stalled for lack of one — the
only IQ grab available was a dead file at ~−75 dBFS RMS with no frame sync,
which could confirm nothing. A capture converts "it sounds bad on my system"
into a question with a reproducible answer. No capture, no verdict — that's
the project's [#764/#771 discipline]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}),
and it applies to your own debugging as much as to bug reports.

One rule above all: **capture the failure**. Same sample rate, same gain, same
antenna, same time of day if the symptom is load-dependent. The #764 deficit
existed only at 10 MS/s; a capture taken at the rate that *works* would have
been evidence for the wrong conclusion.

## The formats

All of GopherTrunk's production pipelines start at complex IQ baseband. The
formats it reads and writes, per the
[decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }}) page:

| Format | Encoding | Size @ 2.4 MS/s | Notes |
|---|---|---|---|
| `.cfile` / f32 | interleaved float32 I,Q | ~19 MB/s | GNU Radio native; largest, lossless |
| cs16 (`.raw`/`.bin`) | interleaved little-endian int16 | ~9.6 MB/s | `auto_record` default; full SDR dynamic range |
| u8 | interleaved uint8 | ~4.8 MB/s | rtl_sdr native 8-bit |
| baseband WAV | 2-ch 16-bit, header carries rate | small (channel rate) | post-DDC narrowband; SDRtrunk/SDR++-compatible |

None of the headerless formats embeds sample rate or center frequency — see
the sidecar section, which is not optional. And repeat the audio caveat from
`samples/README.md` until it's reflex: **an MP3/WAV of demodulated audio is
useless for phase modulations** — TETRA's π/4-DQPSK constellation is gone the
moment an FM discriminator touches it, and 128 kbps MP3 blurs 4-level FSK
enough to collapse the matched filter. IQ or nothing.

The manual tool is one command:

```
gophertrunk capture -freq 851000000 -sample-rate 2400000 -gain 297 \
    -seconds 30 -format cs16 -out cc-fail.raw -protocol p25 \
    -source "roof discone, marginal after 18:00"
```

That writes the IQ plus a `.metadata.json` sidecar, and `-bundle out.gtb.tar.gz
-sigmf` additionally packages capture + metadata (+ a carved narrowband slice)
with a SigMF `.sigmf-meta` sidecar so inspectrum, GNU Radio, and sigmf-python
can open it (`internal/gtbundle/sigmf.go` maps u8→`cu8`, f32→`cf32_le`). For
failures you can't predict, `baseband.auto_record` fires automatically —
`on_cc_sync_loss`, `on_encrypted`, `on_concurrent_calls`, `on_emergency` — and
its `tap: ddc` option records the small channelized stream (144 kHz for TETRA)
instead of the fat wideband, which is how the TETRA sync-loss investigation
got its 11 perfectly-timed captures without anyone touching a keyboard.

## The sidecar

A headerless IQ file is a bag of numbers. The sidecar makes it evidence. The
committed example in `samples/p25/p25-450875-cc.metadata.json` shows the shape:

```json
{
  "source": "OTA RTL-SDR @ 450.500 MHz center, 2 MSPS; control channel at -625 kHz",
  "sample_rate_hz": 48000,
  "center_freq_hz": 449875000,
  "expected": { "demod_mode": "c4fm", "nac": "0x2C1", "min_tsbk": 28, "...": "..." }
}
```

`sample_rate_hz` and `center_freq_hz` are the two fields nothing can proceed
without. `source` is provenance — antenna, gain, conditions — which future-you
will not remember. And `expected` is the ambitious part: pass bars that turn
the file into a graded test (more below). Write the sidecar *at capture time*;
a `.cfile` found three weeks later with no note about its rate is, in
practice, garbage — every replay of it begins with guessing.

## Tap points: what a capture can prove

Where you tap the chain determines what questions the file can answer:

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="The GopherTrunk signal chain with its three IQ tap points marked: the wideband tap after the SDR captures everything including front-end behavior; the ddc tap after the downconverter captures one small channelized stream; and the pre-combine diversity tap sits before the MRC combiner, the only place a combiner question can be answered. Audio output at the far end is marked as not a capture.">
  <rect x="8" y="80" width="80" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="48" y="104" text-anchor="middle" fill="currentColor" font-size="10">SDR</text>
  <line x1="88" y1="100" x2="116" y2="100" stroke="currentColor"/><polygon points="116,96 126,100 116,104" fill="currentColor"/>
  <rect x="126" y="80" width="100" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="176" y="98" text-anchor="middle" fill="currentColor" font-size="10">combiner</text>
  <text x="176" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(diversity rigs)</text>
  <line x1="226" y1="100" x2="254" y2="100" stroke="currentColor"/><polygon points="254,96 264,100 254,104" fill="currentColor"/>
  <rect x="264" y="80" width="90" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="309" y="104" text-anchor="middle" fill="currentColor" font-size="10">DDC</text>
  <line x1="354" y1="100" x2="382" y2="100" stroke="currentColor"/><polygon points="382,96 392,100 382,104" fill="currentColor"/>
  <rect x="392" y="80" width="110" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="447" y="104" text-anchor="middle" fill="currentColor" font-size="10">demod + FEC</text>
  <line x1="502" y1="100" x2="530" y2="100" stroke="currentColor"/><polygon points="530,96 540,100 530,104" fill="currentColor"/>
  <rect x="540" y="80" width="80" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="580" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">audio</text>
  <text x="580" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">not a capture</text>
  <line x1="106" y1="80" x2="106" y2="46" stroke="var(--accent)"/><circle cx="106" cy="42" r="4" fill="var(--accent)"/>
  <text x="106" y="30" text-anchor="middle" fill="var(--accent)" font-size="9">pre-combine tap (diversity_capture)</text>
  <line x1="240" y1="120" x2="240" y2="158" stroke="currentColor"/><circle cx="240" cy="162" r="4" fill="currentColor"/>
  <text x="240" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">wideband tap (auto_record)</text>
  <line x1="372" y1="120" x2="372" y2="158" stroke="currentColor"/><circle cx="372" cy="162" r="4" fill="currentColor"/>
  <text x="372" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ddc tap (small, channelized)</text>
  <text x="340" y="203" text-anchor="middle" fill="var(--fg-muted)" font-size="10">every tap answers questions about what is UPSTREAM of it — and nothing about what it already baked in</text>
</svg>
<figcaption>A tap proves things about its upstream only: a post-combine capture has one combiner baked in, and audio has baked in the entire receiver.</figcaption>
</figure>

The subtle case is diversity. On a two-antenna MRC rig
(coming in [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})),
the combiner lives *in the driver* — so `baseband.auto_record`, the iqtap
brokers, and the scope taps are all downstream of it. A capture from any of
them has one particular combiner already applied and **cannot be replayed
through a different one**: it answers nothing about whether tracking beats
static, or whether the second antenna contributes at all. The one tap that
can is `sdr.soapy_remote[].diversity_capture`, which dumps each branch
straight after de-interleave — `<prefix>.br0.cs16`, `<prefix>.br1.cs16`, plus
a `<prefix>.diversity.json` sidecar. Its alignment rule is worth internalizing
as a general principle: a datagram that didn't carry every branch is dropped
from **both** files and counted (`dropped_datagrams`), never written short,
because one short write silently desynchronizes the pair and quietly
invalidates every conclusion drawn afterwards. An unaligned multi-channel
capture is worse than no capture — it looks like evidence.

## From capture to regression test

The end state of a good capture is not an attachment on an issue — it's a
fixture. Drop the file in `samples/<protocol>/` with its `.metadata.json`, and
the skip-gated integration tests pick it up automatically; the sidecar's
`expected` block *is* the pass bar. The committed P25 example was graded EVM
12.7% / SNR 14.5 dB / 36 TSBKs with zero CRC failures on the day it was
measured, and its sidecar pins floors below those values — so any future demod
regression fails a test naming the exact capture that caught it. The
per-protocol `samples/*/README.md` files document each decoder's acceptance
criteria, and the [decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }})
page lists which decoders are still *blocked* waiting for a real capture
(with format, rate, and duration specified — e.g. TETRA wants ≥72 kHz because
a 48 kHz capture can't lock the 18 ksym/s symbol timing). This is the quiet
payoff of capture discipline: your bad afternoon becomes everyone's permanent
regression test.

## Where this goes next

With capture discipline in hand, we can finally buy hardware that raises
questions only captures can answer.
[Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
adds a second antenna: what diversity and maximal-ratio combining actually buy
on a fading trunking band, what GopherTrunk's `diversity: mrc` mode does with
the two streams, and the one log figure that tells you whether the pair is
earning its coax.

## FAQ

**How long should a capture be?**
Long enough to contain the failure plus context — the per-protocol guidance in
[decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }}) runs
5–30 s (TETRA asks for ≥30 s including an idle window for noise-floor
profiling). For intermittent symptoms, `baseband.auto_record` with the right
trigger beats any manual attempt.

**cs16 or cfile?**
cs16 at half the size is the practical default and what `auto_record` writes;
f32 `.cfile` is the GNU Radio interchange format. Both are lossless for
16-bit-class SDRs. Avoid u8 for marginal-signal work unless the source is
already 8-bit (RTL-SDR) — you can't add back dynamic range that was never
digitized.

**My capture is 2 GB. How do I share it?**
Carve the channel: `replay -record-ddc` writes the post-DDC narrowband stream
as a small 2-channel WAV that decodes identically (`replay -format wav`), and
`capture -bundle` packages a narrowband slice with metadata. A 10-second
channelized fixture is usually a few MB and reproduces the same decode.

**Does GopherTrunk read SigMF?**
It emits, not reads: `capture -bundle -sigmf` writes a `.sigmf-meta` sidecar
(SigMF 1.0.0, minimal global + captures objects) so external tooling can open
GopherTrunk captures. Internally the `.metadata.json` sidecar remains the
native contract — it carries decode expectations SigMF has no fields for.

**What gain should I capture at?**
The gain the failure happens at — that's the whole point. If you also have
time for a second capture at a stepped-down gain, take it; a pair of captures
bracketing the symptom let the replayer separate overload effects from
signal-limited effects, which is exactly how the
[rail-pinning diagnosis]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
was settled.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Filters & LNAs — Adding Gain Without Adding Garbage]({{ '/blog/tutorials/analog-edge-09-filters-lnas/' | relative_url }})
· Next →
[Part 11: Two Antennas — Diversity & MRC From the Operator's Seat]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
