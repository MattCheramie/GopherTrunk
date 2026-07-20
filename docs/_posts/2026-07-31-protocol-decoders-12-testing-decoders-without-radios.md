---
title: "Protocol Decoders, Part 12: Testing Decoders Without Radios"
description: How GopherTrunk tests every protocol decoder with no SDR and no live system — synthesized control channels modulated through the real TX chain, a decoder registry, golden fixtures, and the regression discipline that makes 'verified' mean something.
category: deep-dives
keywords: decoder integration testing, synthesized control channel, mock sdr replay, decoder registry, golden fixtures, regression test discipline, c4fm modulator test, gophertrunk testing
tags: [testing, integration, go, decoding, ci, architecture]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 12
---

*Part 12 of **Protocol Decoders** — the finale. Every post in this series has
leaned on a claim: "the framing is verified," "the state machine is functional,"
"this locks." Those claims are only worth anything if they're *tested*, and you
can't put an SDR and a live trunked system in a CI runner. This closing post shows
how GopherTrunk tests every decoder with no radio at all — and why that same
discipline is what let Part 11 honestly say the alias cipher is **not** verified.*

> **TL;DR:** GopherTrunk tests decoders by **synthesizing** a control channel:
> build a known dibit stream, modulate it through the *real* C4FM transmit chain to
> IQ, hand it to a **mock SDR**, and assert the production daemon locks and emits
> the right events. A per-protocol **pipeline registry** makes every decoder
> pluggable and testable; **golden fixtures** (including committed real-air
> captures) pin behavior; and a strict **regression discipline** — a failing-first
> test for every fix — is what makes "verified" a fact rather than a hope.

**Key takeaways**

- Decoders are tested end-to-end with a **mock SDR** replaying synthesized IQ —
  no hardware, fully deterministic, runs in CI.
- Synthesis goes through the **production TX chain**, so the test exercises the
  real demod, clock recovery, slicer, and state machine.
- A **registry** of `PipelineFactory` per protocol makes decoders pluggable and
  gives tests a clean seam (`SetTestFactory`).
- **Regression discipline** — failing-first tests and committed fixtures — is the
  reason a claim like `CipherVerified` can be trusted.

## Cheat sheet

| Piece | Where | Role |
|---|---|---|
| Synthesized dibits | `buildP25LockedIQDibits` | known FSW + NID + TSBK stream |
| TX chain | `demod.ModulateP25C4FM` | dibits → IQ through the real modulator |
| Mock SDR | `sdr.MockDriver` | replays an IQ `.cfile` as a fake device |
| Pipeline registry | `ccdecoder.factories` | `Protocol → PipelineFactory` dispatch |
| Test seam | `ccdecoder.SetTestFactory` | swap a factory for one protocol in a test |
| Golden fixtures | `integration_cc_*_test.go` | per-protocol end-to-end assertions |

## In this post

- **The problem** — why a live radio is the worst test rig.
- **Synthesizing a control channel** — dibits to IQ and back.
- **The registry** — decoders as pluggable factories.
- **Two levels of test** — full-chain lock vs. stubbed grant chain.
- **Golden fixtures and regression discipline** — and the series wrap.

## The problem: a radio is the worst test rig

Everything in this series is a decoder, and a decoder is only correct if it decodes.
The obvious way to check is to point it at a real system — which is also the *worst*
possible test. A live signal is non-deterministic (fades, drifts, disappears),
needs hardware CI can't have, and can't be replayed identically to reproduce a
failure. You can't gate a merge on "go stand near a trunked tower."

So GopherTrunk tests decoders the way it analyzes captures in
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}): offline, deterministic,
and against the *same production code* the live daemon runs. The trick is to
manufacture the signal.

## Synthesizing a control channel

The flagship integration test, `TestDaemonCCDecodesP25Phase1`, is the "lights up
live trunked reception" check — with no radio in the room. It builds a P25 Phase 1
dibit stream by hand: a warmup pattern cycling through every symbol so the clock
recovery has transitions to lock onto, then repeated frames of frame-sync word, NID,
and a trellis-encoded TSBK, with the P25 status symbols a real transmitter
interleaves:

```go
// cmd/gophertrunk/integration_cc_test.go (shape)
frame = append(frame, phase1.FrameSyncWord[:]...)
nidBits := phase1.EncodeNIDBits(nac, phase1.DUIDTrunkingSignaling)
// ... pack NID dibits ...
tsbk := phase1.AssembleTSBK(phase1.TSBK{LB: true, Opcode: phase1.OpRFSSStatusBroadcast})
frame = append(frame, phase1.EncodeTSBKChannel(tsbk)...)
frame = phase1.InjectControlStatusSymbols(frame) // status symbol every 36 dibits
```

Then — and this is the important part — it **modulates those dibits through the real
transmit chain**, not a shortcut:

```go
// 48 kHz @ 10 sps = 4800 baud (the spec rate); 1800 Hz peak deviation (TIA-102).
iq := demod.ModulateP25C4FM(dibits, sampleRateHz, deviationHz)
writeIQToU8File(iqPath, iq)              // interleaved u8, the mock SDR's format
sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})
```

`ModulateP25C4FM` runs the spec P25 C4FM TX path — impulse train, P25 transmit pulse
shape (RRC), FM modulator — so the resulting IQ is a faithful C4FM signal. The
`MockDriver` registers as an SDR device that simply replays that `.cfile`. From the
daemon's perspective it's a radio. The test then boots the **wired production
daemon** and asserts the full chain recovers the lock: IQ → C4FM demod →
Mueller-Müller clock recovery → 4-level slice → dibits → FSW + NID + TSBK trellis →
control-channel state machine → `cc.locked` on the bus → `/api/v1/scanner` reporting
`state=locked` → the `gophertrunk_control_channel_locked` gauge reaching 1.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A synthesized test round trip: known dibits are modulated through the real C4FM transmit chain to IQ, written as a u8 cfile, replayed by a mock SDR into the production daemon, which demodulates back to dibits, drives the control-channel state machine, and emits cc.locked which the test asserts through the API and metrics">
  <rect x="14" y="20" width="120" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="74" y="38" text-anchor="middle" fill="var(--accent)" font-size="11">known dibits</text>
  <text x="74" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="10">FSW+NID+TSBK</text>
  <line x1="134" y1="40" x2="170" y2="40" stroke="currentColor"/>
  <polygon points="170,36 180,40 170,44" fill="currentColor"/>
  <rect x="180" y="20" width="130" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="245" y="38" text-anchor="middle" fill="currentColor" font-size="11">ModulateP25C4FM</text>
  <text x="245" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="10">real TX chain → IQ</text>
  <line x1="310" y1="40" x2="346" y2="40" stroke="currentColor"/>
  <polygon points="346,36 356,40 346,44" fill="currentColor"/>
  <rect x="356" y="20" width="120" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="416" y="38" text-anchor="middle" fill="currentColor" font-size="11">u8 .cfile</text>
  <text x="416" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="10">MockDriver</text>
  <line x1="416" y1="60" x2="416" y2="88" stroke="currentColor"/>
  <polygon points="412,88 416,98 420,88" fill="currentColor"/>
  <rect x="180" y="98" width="296" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="328" y="118" text-anchor="middle" fill="var(--accent)" font-size="11">production daemon</text>
  <text x="328" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="10">demod → MM clock → slice → dibits → CC state machine</text>
  <line x1="180" y1="120" x2="144" y2="120" stroke="currentColor"/>
  <polygon points="144,116 134,120 144,124" fill="currentColor"/>
  <rect x="14" y="98" width="120" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="74" y="118" text-anchor="middle" fill="currentColor" font-size="11">cc.locked</text>
  <text x="74" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="10">on the bus</text>
  <line x1="74" y1="142" x2="74" y2="166" stroke="var(--accent)"/>
  <polygon points="70,166 74,176 78,166" fill="var(--accent)"/>
  <rect x="14" y="176" width="330" height="20" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="179" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="10">assert: /api/v1/scanner state=locked · metric gauge = 1</text>
</svg>
<figcaption>The test is a round trip: synthesize dibits, modulate them to IQ through the real TX chain, replay through a mock SDR, and assert the production daemon demodulates back to a lock — no hardware, fully deterministic.</figcaption>
</figure>

## The registry: decoders as factories

What makes every protocol testable the same way is that decoders are **pluggable**.
The `ccdecoder` package keeps a registry mapping a `trunking.Protocol` to a
`PipelineFactory`, and every pipeline satisfies one small contract:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
type ProtocolPipeline interface {
    Process(iq []complex64) // consume one IQ chunk
    Reset()                 // clear symbol-domain state on re-sync
    Close() error           // idempotent
}

type PipelineFactory func(PipelineOptions) (ProtocolPipeline, error)
var factories = map[trunking.Protocol]PipelineFactory{ /* p25, tetra, dmr-tier3, ... */ }
```

This registry is the same seam the Signal Lab protocol picker enumerates
(`RegisteredProtocols`) and the same one the daemon dispatches through at retune. For
tests it exposes a deliberately narrow hook, `SetTestFactory`, which swaps the
factory for a single protocol and hands back a `restore` you defer. It exists
specifically so an out-of-package integration test can pump a known-good dibit stream
through the daemon's *real* decoder without owning a working modulator — and it's
documented "integration tests only; production must never call this."

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="The pipeline registry maps each protocol to a factory that builds a Process-Reset-Close pipeline; production dispatches through it at retune, Signal Lab enumerates it for the protocol picker, and integration tests swap one factory via SetTestFactory">
  <rect x="250" y="16" width="180" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="34" text-anchor="middle" fill="var(--accent)" font-size="12">factories map</text>
  <text x="340" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Protocol → PipelineFactory</text>
  <g stroke="var(--fg-muted)">
    <line x1="290" y1="58" x2="150" y2="96"/><polygon points="150,92 141,97 152,100" fill="var(--fg-muted)"/>
    <line x1="340" y1="58" x2="340" y2="96"/><polygon points="336,96 340,106 344,96" fill="var(--fg-muted)"/>
    <line x1="390" y1="58" x2="530" y2="96"/><polygon points="528,92 539,97 528,100" fill="var(--fg-muted)"/>
  </g>
  <rect x="40" y="98" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="130" y="117" text-anchor="middle" fill="currentColor" font-size="10">daemon: dispatch at retune</text>
  <rect x="250" y="98" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="340" y="117" text-anchor="middle" fill="currentColor" font-size="10">Signal Lab: protocol picker</text>
  <rect x="460" y="98" width="180" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="550" y="117" text-anchor="middle" fill="var(--accent)" font-size="10">tests: SetTestFactory</text>
  <rect x="250" y="138" width="180" height="24" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="340" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ProtocolPipeline: Process · Reset · Close</text>
</svg>
<figcaption>One registry, three consumers: production dispatches through it, Signal Lab enumerates it, and integration tests swap a single factory — all against the same tiny pipeline contract.</figcaption>
</figure>

## Two levels of test

There's a subtlety worth calling out, because it's a nice example of testing at the
right altitude. The full-chain test above proves **IQ → dibit → lock** works. But
asserting a multi-frame *grant chain* — status → identifier-update → group-voice-grant
— through the live clock loop is unreliable: the Mueller-Müller loop reliably lands
the *first* frame-sync word (enough to lock), but converging well enough to extract
every subsequent FSW + NID + 98-dibit TSBK trellis window in one streaming pass is a
tuning exercise orthogonal to the grant wiring.

So `TestDaemonCCDecodesP25Phase1GrantChain` uses `SetTestFactory` to install a stub
pipeline that **ignores the IQ and pumps the synthesized dibits straight into the
real `phase1.ControlChannel`**:

```go
// internal/scanner/ccdecoder pipeline stub (shape)
func (p *p25Phase1StubPipeline) Process(iq []complex64) {
    if p.consumed { return }
    p.consumed = true
    p.cc.Process(p.dibits, 0) // real state machine, band plan, bus, engine
}
```

Everything *above* IQ→dibit — factory dispatch, the state machine, the band-plan
resolution, the bus publication, the trunking engine, the supervisor, the API, the
metrics handler — runs through production code, and the test asserts the resolved
grant frequency exactly (`base + spacing × channel = 851_062_500`). One test owns the
demod; the other owns the message chain; neither is flaky because each isolates the
one thing it's proving.

## Golden fixtures and regression discipline

That synthesis-plus-mock-SDR pattern is repeated per protocol — there's an
`integration_cc_*_test.go` for P25 Phase 2, DMR (Tier 1/2/3), NXDN, dPMR, D-STAR,
YSF, EDACS, LTR, MPT-1327, Motorola Type II, and TETRA. Some go further and commit a
**real-air capture** (`integration_cc_tetra_realair_test.go`,
`integration_cc_nxdn_realair_test.go`) so the decoder is pinned against a genuine
signal, not just a self-consistent synthesis — the strongest kind of golden fixture,
because a synthesizer and a decoder written by the same author can agree on the same
mistake.

This is where the series closes the loop with its own capstone. The reason Part 11
could honestly report the alias cipher as **not verified** is the exact same
discipline: `CipherVerified` flips to true *only* alongside a committed regression
fixture mapping real encoded bytes to correct plaintext. It's the project's
[issue-closing policy]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
in code form — a claim isn't true until a failing-first test passes and the reality
is reproduced. The whole point of testing without radios isn't convenience; it's that
a *deterministic, reproducible* fixture is the only thing that can turn "it seems to
work" into "it is verified."

### How that principle shaped the Go code

- **The `MockDriver` is a real `sdr.Driver`.** It registers exactly like a physical
  device, so the daemon under test is unmodified — no test-only branches in
  production paths.
- **The registry has one narrow test seam.** `SetTestFactory` is the *only* way tests
  perturb decoder construction, and it restores itself, so a test can't leak state
  into the next one.
- **Metrics are polled, not scraped once.** Because the metrics collector is a
  separate bus subscriber, counters lag the test's own subscription; the helpers
  poll to a deadline instead of racing a single scrape — determinism all the way
  down.
- **Fixtures are committed, not generated on the fly.** A real-air `.cfile` in the
  repo is a permanent regression anchor; the decoder that passes it today must keep
  passing it forever.

## The series, wrapped

Twelve parts ago this series started with a single control-channel decoder and the
promise that every protocol — P25 Phase 1 and 2, DMR, NXDN, dPMR, TETRA, and the
legacy EDACS/LTR/MPT-1327 family — reduces to the same shape: symbols in, a
protocol-neutral `Grant` out, published on a bus the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) consumes
without caring which protocol produced it. We followed that shape through π/4-DQPSK
and channel coding, through distributed trunking with no control channel at all,
through conventional squelch and wideband channelization, and into the one field that
still resists us — the Motorola talker alias, GopherTrunk's white whale, honestly
gated and honestly unsolved. And we end where every one of those claims is settled:
in a test that needs no radio. That's the through-line — decoders that are decoupled,
composable, and *provable* offline. Thanks for reading.

## FAQ

**How does GopherTrunk test a decoder without an SDR?**
It synthesizes a known dibit stream, modulates it to IQ through the production C4FM
transmit chain, and replays that IQ through a `MockDriver` that registers as a fake
SDR. The real daemon demodulates it and the test asserts the lock, grant, API state,
and metrics — deterministically, in CI, with no hardware.

**Why modulate through the real TX chain instead of feeding dibits directly?**
So the test exercises the actual demodulator, clock recovery, and slicer, not just
the state machine. GopherTrunk does both: a full IQ→lock test proves the DSP path,
and a stub-factory test that injects dibits directly proves the multi-frame message
chain without depending on the clock loop converging perfectly.

**What is the pipeline registry?**
A map from `trunking.Protocol` to a `PipelineFactory` that builds that protocol's
receiver pipeline (`Process`/`Reset`/`Close`). It's how the daemon dispatches decode
at retune, how Signal Lab enumerates protocols, and — via `SetTestFactory` — how
integration tests swap in a controlled pipeline for one protocol.

**What makes a claim like "verified" trustworthy here?**
A committed, failing-first regression fixture. A fix ships with a test that fails
without it and passes with it; a real-air capture pins a decoder against genuine
signal; and a gate like `CipherVerified` only flips alongside such a fixture. Without
that, the project keeps the claim open — which is exactly why the alias cipher is
still marked unverified.

## Series navigation

**Part 12 of 12** · ←
[Part 11: The Alias Hunt II]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
