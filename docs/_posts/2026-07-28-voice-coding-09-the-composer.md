---
title: "Voice Coding, Part 9: The Composer — Wiring Frames to the Right Vocoder"
description: How GopherTrunk's composer turns a CallStart event into PCM — per-protocol demod chains, front-end decimation, boundary handling, and how the recorder maps each protocol to its vocoder.
category: deep-dives
keywords: voice composer, per-call demod chain, p25 phase 1 voice, p25 phase 2 voice, dmr voice chain, front-end decimation, vocoder selection, recording boundary, gophertrunk voice coding
tags: [composer, dsp, dmr, p25, architecture, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 9
---

*Part 9 of **Voice Coding**. The last two posts built vocoders in isolation.
This one is the wiring diagram: how a call's IQ becomes vocoder frames, which
chain runs for which protocol, and where the decision "use IMBE vs AMBE+2"
actually gets made. Calls arrive from the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series;
frames flow toward the recorder.*

> **TL;DR:** `internal/voice/composer` subscribes to `KindCallStart` /
> `KindCallEnd` and, per call, spins a goroutine running a **per-protocol demod
> chain**: analog FM, DMR, P25 Phase 1, or P25 Phase 2. Each chain decimates
> the wideband IQ to a protocol-friendly rate, recovers symbols, and emits
> either PCM (FM) or raw vocoder frames (digital). A **shared boundary
> tracker** handles hangtime, talkgroup gating, and per-transmission splitting
> uniformly. The composer picks the *chain*; the recorder picks the *vocoder*
> from `Grant.Protocol` via `DefaultVocoderForProtocol`.

**Key takeaways**

- The composer is **event-driven**, mirroring the engine's design: one
  subscription, one goroutine per active call, no direct calls into the
  recorder beyond a thin `WritePCM` / `WriteRawFrame` sink interface.
- Chain selection is a `switch` on `Grant.Protocol` — FM, DMR, P25 P1, P25 P2 —
  with everything else (NXDN, dPMR, TETRA, YSF, D-STAR, EDACS ProVoice)
  explicitly bypassed for now.
- Front-end decimation uses a **decimating FIR** that convolves only at output
  positions — byte-identical to the old full-rate filter, ~decim× cheaper.
- The digital chains emit **raw frames**; the recorder maps protocol → vocoder
  (`p25→imbe`, `p25-phase2→ambe2`, `dmr-tier*→ambe2-dmr`) and decodes to PCM.

## Cheat sheet

| `Grant.Protocol` | Chain | Emits | Vocoder (recorder) |
|---|---|---|---|
| `""`, `fm`, `motorola`, `ltr`, `mpt1327` | `runFMChain` | PCM | — (FM demod) |
| `p25` | `runP25Phase1VoiceChain` | raw frames | `imbe` |
| `p25-phase2` | `runP25Phase2VoiceChain` | raw frames | `ambe2` |
| `dmr-tier1/2/3` | `runDMRVoiceChain` | raw frames | `ambe2-dmr` |
| `nxdn`, `dpmr`, `tetra`, ProVoice | bypassed | — | — |

## In this post

- **What the composer is** — a CallStart-to-PCM bridge.
- **Chain selection** — the protocol switch and its fan-out.
- **Front-end decimation** — the decimating FIR and per-source rates.
- **Boundary handling** — hangtime, gating, and splitting in one place.
- **Vocoder selection** — why it lives in the recorder, not the composer.

## What the composer is

The [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }})
publishes a `KindCallStart` the moment it retunes a Voice SDR to a granted
channel. The composer is the subscriber that turns that event into audio. Its
`Run` loop is the same shape as the engine's — drain the bus, react — and it
spawns exactly one demod goroutine per active call:

```go
// internal/voice/composer/composer.go (shape)
switch ev.Kind {
case events.KindCallStart:
    if cs, ok := ev.Payload.(trunking.CallStart); ok {
        c.handleStart(ctx, cs) // look up device, open IQ, spawn chain
    }
case events.KindCallEnd:
    if ce, ok := ev.Payload.(trunking.CallEnd); ok {
        c.handleEnd(ce) // cancel the chain's context
    }
}
```

It touches the outside world through three tiny interfaces — `IQSource`
(stream IQ + report sample rate), `PCMSink` / `rawFrameSink` (the recorder),
and `EngineHooks` (`Touch`, `EndCall`, `UpdateSignal`). That decoupling is
deliberate: the composer imports no concrete SDR type and no concrete recorder,
so a test drives a whole chain with an in-memory channel and a map.

## Chain selection

`handleStart` classifies the grant's protocol and dispatches to the matching
chain. Analog trunking systems (Motorola Type II, EDACS non-ProVoice, LTR,
MPT-1327) carry plain narrowband FM voice, so they share the FM chain; the
digital protocols each get their own:

```go
// internal/voice/composer/composer.go (shape)
switch {
case isDMRVoice:  // dmr-tier1 / tier2 / tier3
    go c.runDMRVoiceChain(chainCtx, serial, iqCh, rateHzF, grp, interleaved, ch.done)
case isP25P2Voice: // p25-phase2
    go c.runP25Phase2VoiceChain(chainCtx, serial, system, macCfg, iqCh, rateHzF, ch.done)
case isP25P1Voice: // p25
    go c.runP25Phase1VoiceChain(chainCtx, serial, system, iqCh, rateHzF, ...)
default:           // analog FM family
    go c.runFMChain(chainCtx, serial, iqCh, uint32(math.Round(rateHzF)), ch.done)
}
```

Protocols without a chain yet — NXDN, dPMR, TETRA, YSF, D-STAR, EDACS ProVoice
— are logged and skipped rather than fed into an FM demod that would produce
garbage. EDACS ProVoice is called out specifically: it's digital and
patent-encumbered, so its bursts go to the recorder's `.raw` sidecar untouched.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A CallStart event fans out through a protocol switch into four demod chains — FM, P25 Phase 1, P25 Phase 2, and DMR — while unsupported protocols are bypassed">
  <rect x="12" y="80" width="140" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="82" y="100" text-anchor="middle" fill="var(--accent)" font-size="12">KindCallStart</text>
  <text x="82" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="10">handleStart</text>
  <line x1="152" y1="102" x2="196" y2="102" stroke="currentColor"/>
  <polygon points="196,98 206,102 196,106" fill="currentColor"/>
  <rect x="206" y="82" width="96" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="254" y="100" text-anchor="middle" fill="currentColor" font-size="11">protocol</text>
  <text x="254" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="10">switch</text>
  <g stroke="var(--fg-muted)">
    <line x1="302" y1="90" x2="470" y2="26"/><polygon points="467,22 477,25 469,31" fill="var(--fg-muted)"/>
    <line x1="302" y1="98" x2="470" y2="74"/><polygon points="467,70 477,74 468,79" fill="var(--fg-muted)"/>
    <line x1="302" y1="106" x2="470" y2="118"/><polygon points="468,113 477,118 467,122" fill="var(--fg-muted)"/>
    <line x1="302" y1="114" x2="470" y2="162"/><polygon points="468,157 477,162 466,165" fill="var(--fg-muted)"/>
  </g>
  <rect x="478" y="12" width="190" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="573" y="30" text-anchor="middle" fill="currentColor" font-size="11">runFMChain → PCM</text>
  <rect x="478" y="60" width="190" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="573" y="78" text-anchor="middle" fill="currentColor" font-size="11">P25 P1 → raw (IMBE)</text>
  <rect x="478" y="104" width="190" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="573" y="122" text-anchor="middle" fill="currentColor" font-size="11">P25 P2 → raw (AMBE+2)</text>
  <rect x="478" y="148" width="190" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="573" y="166" text-anchor="middle" fill="currentColor" font-size="11">DMR → raw (AMBE+2)</text>
  <text x="254" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">NXDN · dPMR · TETRA ·</text>
  <text x="254" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">YSF · ProVoice → bypass</text>
</svg>
<figcaption>One event fans out to four chains by protocol. Analog trunking shares the FM chain; each digital protocol has its own; the rest are explicitly skipped rather than mis-demodulated.</figcaption>
</figure>

## Front-end decimation

Every chain starts by bringing the wideband IQ down to a protocol-friendly
rate — 48 kHz for DMR and FM, an H-DQPSK-friendly rate for P25 Phase 2. The
naive way (filter every input sample, then keep every Nth) wastes ~98% of the
filter's work at 2.4 MS/s. GopherTrunk instead uses a **decimating FIR** that
computes only the samples it keeps:

```go
// internal/voice/composer/frontend_decim.go (shape)
// Byte-for-byte equivalent to LowpassKaiser(81, bw/iqHz, 8.6) then striding,
// but convolves only at output positions — cost drops ~decim× (≈50× at 2.4 MS/s).
func (f *decimatingFIR) Process(dst, raw []complex64) []complex64
```

That efficiency mattered for a real bug: at 2.4 MS/s the old full-rate filter
burned ~194M complex MACs/sec *per active voice call* on one goroutine
competing with the control-channel decode — enough to starve the SDR reader and
drop live IQ. The decimating FIR keeps the exact same coefficients and kept
samples (so the decode is unchanged) and just deletes the wasted multiplies.

The chains also respect **per-source sample rates**. A physical SDR delivers
the daemon-wide rate (2.4 MS/s); a wideband-derived *virtual* voice tuner
delivers 48 kHz already. Sources can even expose their exact fractional rate
(`SampleRateExactHz`) so the digital chains clock their symbol-recovery loops
at the true rate — a rounded nominal rate drifts and periodically slips,
audible as voice spikes (the voice-path parity for issue #550). And the DMR /
P25 P2 front ends *channel-select* even when they don't decimate, so an adjacent
±12.5 kHz neighbour can't capture the FM discriminator during voice gaps.

## Boundary handling

Every chain — FM, DMR, P25 P1, P25 P2 — shares one `boundaryTracker`, so
call-boundary behaviour is identical across protocols instead of reimplemented
four times. It centralises three jobs:

- **Hangtime end-of-call.** Once voice has been decoding, the call ends
  `VoiceHangtime` (default 3.5 s) after the last decoded frame, rather than
  waiting out the engine's much longer watchdog — recordings stay tight to the
  actual transmission.
- **Talkgroup gating.** On protocols that decode an in-band talkgroup, audio
  whose TG differs from the granted one isn't written, and a sustained foreign
  TG (two consecutive frames) ends the call. This fixes the shared-frequency
  case where two virtual tuners decode the same IQ.
- **Per-transmission splitting.** At an over boundary the recorder rolls to a
  fresh file — but only when audio was written since the last roll, so a run of
  terminators doesn't spawn empty files.

```go
// internal/voice/composer/boundary.go (shape)
func (bt *boundaryTracker) onVoice(tg uint32) bool { /* gate + hangtime bookkeeping */ }
func (bt *boundaryTracker) onTransmissionEnd()     { /* split roll via KindCallSegment */ }
func (bt *boundaryTracker) run(ctx context.Context) { /* hangtime timer + throttled Touch */ }
```

The tracker also carries the call's signal-level meter (mean `|iq|²` → dBFS)
and, for P25 Phase 1 only, an EVM/SNR estimate from the receiver's soft/symbol
taps — the numbers stamped onto the call before the engine publishes `CallEnd`.

## Vocoder selection: it's the recorder's job

Here's the subtlety the series title hints at. The composer selects the
*chain*; it does **not** call a vocoder. The digital chains FEC-decode their
frames and hand the recorder raw vocoder payloads (`WriteRawFrame`). The
recorder is what maps a protocol to a vocoder factory:

```go
// internal/voice/recorder.go (shape)
func DefaultVocoderForProtocol() map[string]string {
    return map[string]string{
        "p25":        "imbe",      // P25 Phase 1 — IMBE 4400
        "p25-phase2": "ambe2",     // P25 Phase 2 — AMBE+2 3600x2400
        "dmr-tier1":  "ambe2-dmr", // DMR — AMBE+2 3600x2450
        "dmr-tier2":  "ambe2-dmr",
        "dmr-tier3":  "ambe2-dmr",
        "nxdn":       "ambe2",
    }
}
```

This is the same one-way discipline the engine uses. The composer knows how to
*recover frames* for a protocol; the recorder knows how to *decode frames* for
a protocol; neither reaches into the other's job. It's also why swapping in the
DVSI hardware vocoder (Part 11) is a config change — a different name in this
map — not a composer edit. And it's why the DMR chain writes its post-FEC
frames to a `.raw` sidecar *and* the recorder renders them: the frames stay
available for out-of-band tools while the WAV gets the pure-Go decode.

## Where this goes next

The chains and the recorder now hand a stream of 160-sample PCM frames toward
the output. [Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
takes that synthesized PCM and conditions it — DC blocking, spectral
enhancement, AGC, tone tilt, and loudness normalization — the difference
between "decoded" and "sounds right."

## FAQ

**Does the composer decode the vocoder itself?**
No. The composer recovers *frames* (PCM for FM, raw vocoder frames for digital)
and hands them to the recorder through a small sink interface. The recorder
selects the vocoder by protocol (`DefaultVocoderForProtocol`) and decodes to
PCM. Chain selection and vocoder selection are separate concerns in separate
packages.

**Why is there one goroutine per call?**
So each call's demod runs independently and a call ends cleanly by cancelling
its context. The composer's `Run` loop only spawns and reaps these goroutines;
the audio work happens off the event loop, exactly like the trunking engine's
subscribers.

**Which protocols can the composer produce voice for today?**
Analog FM (including Motorola/EDACS/LTR/MPT-1327 trunking), P25 Phase 1 (IMBE),
P25 Phase 2 (AMBE+2), and DMR (AMBE+2). NXDN, dPMR, TETRA, YSF, D-STAR, and
EDACS ProVoice are recognised but bypassed — logged, not mis-demodulated.

**How does a call know when to stop recording?**
The shared boundary tracker ends the call `VoiceHangtime` (default 3.5 s) after
the last matching voice frame, and immediately if a sustained foreign talkgroup
takes a shared frequency. A call that never decodes any voice is torn down after
a short no-voice window so a phantom grant frees its tuner.

## Series navigation

**Part 9 of 12** · ←
[Part 8: AMBE+2 FEC & the Knox Path]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }})
· Next →
[Part 10: Enhancement & Loudness]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})
