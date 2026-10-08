---
title: "The Legacy Family End to End, Part 7: Analog Voice on Trunked FM — From a Grant to the Composer's FM Chain"
description: "What happens after a SmartNet, EDACS, LTR or MPT 1327 decoder publishes a grant: the band-plan resolver that must turn a channel number into hertz, the engine's zero-frequency drop and voice-pool bind, the composer's runFMChain stages and their config keys, how an analog call ends with no release message, and which of the four protocols can reach the recorder today."
category: deep-dives
keywords: analog trunking voice sdr, smartnet voice recording, edacs analog voice, ltr voice follow, runFMChain composer, analog fm call hangtime, band plan resolver lcn to frequency, dropping grant with zero frequency, fm_channel_bandwidth_hz, gophertrunk legacy family end to end
tags: [legacy-family-end-to-end, analog, fm, composer, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 7
---

*Part 7 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, each placed honestly
on one verification ladder.
[Part 6]({{ '/blog/deep-dives/legacy-family-06-mpt1327/' | relative_url }})
closed the control-channel half of the FM-era four and ended on a grant
carrying `FrequencyHz` 0. This part follows a grant from any of the four
decoders through the trunking engine into the composer's analog chain and
out to a WAV — and measures, protocol by protocol, which grants can make
that journey on a live daemon today.*

> **TL;DR:** Every FM-era decoder ends in `publishGrant` with a channel
> number and a frequency it had to resolve itself. Motorola resolves through
> its built-in `BandPlan` tables (`motorola_band_plan`: `800_standard`,
> `800_rebanded`, `800_splinter`, `900`); EDACS, LTR and MPT 1327 expose a
> `Resolver` (`LinearBandPlan` / `TableBandPlan`) that `newEDACSPipeline`,
> `newLTRPipeline` and `newMPT1327Pipeline` never install — no config key
> exists — so their live grants carry `FrequencyHz` 0 and `Engine.HandleGrant`
> drops them first thing. A surviving grant passes lockout, hold/avoid and
> the scan-list gate, binds a `VoiceDevice` (`VoicePool.Bind` →
> `SetCenterFreq`, fresh `CallID`) and publishes `CallStart`.
> `classifyVoiceKind` maps `motorola` / `ltr` / `mpt1327` / non-ProVoice
> `edacs` to `voiceKindFM` and starts `runFMChain`: 81-tap decimating FIR to
> 48 kHz, optional `fm_channel_bandwidth_hz` filter, `demod.FM`, 300 Hz
> high-pass, de-emphasis, 3.4 kHz low-pass, AGC, 8 kHz PCM. No FM-era
> decoder publishes a release; a call ends by hangtime (3.5 s after PCM
> stops), the 30 s watchdog, or the engine moving the device. Rung:
> composer-pinned by `TestComposerRunsFMChainForAnalogTrunking`; trunked
> analog voice is on-air unverified for all four and unreachable for three.

**Key takeaways**

- **A grant is only as good as its frequency.** The engine drops
  `FrequencyHz` 0 before any other gate, and three of the four FM-era
  pipelines produce exactly that.
- **The analog chain is the conventional scanner's chain.** `runFMChain`
  serves `fm-conv`, `am-conv` and the analog-trunk protocols alike; the
  field fixes from #1184 and #1090 landed there.
- **Nothing in the signalling ends an analog call.** With no release
  message and no squelch decision on a trunked tap, hangtime, the
  `call_timeout_ms` watchdog and pool preemption are the only ends.
- **Verified means the composer, not the air.** `status.md`'s "analog FM
  trunking decodes voice through the composer's FM chain" is pinned by tests
  that hand the composer a grant with its frequency already filled in.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Channel → Hz | Motorola: built-in `BandPlan.Frequency(ch)`; EDACS / LTR / MPT: `Resolver.Frequency`, not installed | `internal/radio/{motorola,edacs,ltr,mpt1327}/bandplan.go` |
| First gate | `if g.FrequencyHz == 0 { Warn("dropping grant with zero frequency"); return }` | `internal/trunking/engine.go` (`HandleGrant`) |
| Bind | `VoicePool.Bind` → `d.Tuner.SetCenterFreq`, `g.CallID = callSeq.Add(1)`, `KindCallStart` | `internal/trunking/voicepool.go`, `engine.go` (`startCall`) |
| Chain selection | `isAnalogTrunk` → `voiceKindFM`; EDACS ProVoice → `voiceKindUnsupported` | `internal/voice/composer/composer.go` (`classifyVoiceKind`) |
| The chain | decimate → channel filter → FM → HPF → de-emphasis → LPF → AGC → PCM | `composer.go` (`runFMChain`, `newFMChannelFilter`) |
| Knobs | `fm_deemphasis`, `fm_audio_highpass_hz` 300, `fm_audio_lowpass_hz` 3400, `fm_channel_bandwidth_hz` 0 | `config.example.yaml` (`recordings:`) |
| Call end | `boundaryTracker` hangtime (`voice_hangtime_ms` 3500), no-voice 2× window, `call_timeout_ms` 30000 | `composer/boundary.go`, `engine.go` (`runWatchdog`) |
| Pins | `TestComposerRunsFMChainForAnalogTrunking`, `TestComposerBypassesEDACSProVoice`, `TestFMChannelFilterSelectivity` | `internal/voice/composer/composer_test.go` |

## In this post

- **Four grants, one resolver gap** — who fills `FrequencyHz`, and who doesn't.
- **The engine's path to a tuner** — gates, bind, retune, preemption.
- **Inside `runFMChain`** — the stages, their order, and the keys that shape them.
- **Ending a call nobody signals** — hangtime, the watchdog, and the noise after the carrier.
- **The recorder's side, and the rung** — what lands on disk, and what has been verified.

## Four grants, one resolver gap

Parts 3 to 6 each ended in the same function shape: Motorola's
`publishGrant` takes a grant OSW whose address is the talkgroup and whose
command is the channel number, EDACS's a `GroupVoiceGrant` with an LCN,
LTR's an active `Status` with a repeater channel, MPT 1327's a
`GoToChannel`. All four stamp `ChannelNum` and `FrequencyHz` on a
`trunking.Grant`. The difference is where the hertz come from.

Motorola is self-contained. `motorola.BandPlan` ports trunk-recorder's
`get_freq` / `is_chan` tables, `motorola_band_plan` selects `800_standard`
(the default), `800_rebanded`, `800_splinter` or `900`, and
`newMotorolaPipeline` hands the plan to `motorola.New`. The plan is also
the OSW state machine's discriminator — a command inside the plan *is* a
channel number — so an unresolvable command is not a grant at all:

```go
// internal/radio/motorola/control.go (shape)
freq, ok := c.plan.Frequency(grantOSW.Command)
if !ok {
    return // not a channel in this plan — nothing to publish
}
```

The other three use the `Resolver` idiom — `LinearBandPlan`
(`BaseHz + (channel + Offset) × SpacingHz`) or a `TableBandPlan` map, each
unit-tested — and each `publishGrant` does the honest thing with a nil
resolver: sets `freq` to 0 and publishes anyway. The gap is upstream.
`newEDACSPipeline`, `newLTRPipeline` and `newMPT1327Pipeline` construct
their channels with `Bus`, `Log`, `SystemName` and `FrequencyHz` only; the
sole `Resolver:` in `pipelines.go` is DMR Tier III's
`tier3.ResolverFromPlan`, and `config.SystemConfig` carries
`motorola_band_plan`, `p25_band_plan`, `dmr_band_plan` and `nxdn_band_plan`
— nothing for EDACS, LTR or MPT 1327.
`TestControlChannelGrantWithoutResolverHasZeroFreq` pins the zero; nothing
pins the wiring, because there is none.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A pipeline from four FM-era decoders to a WAV file. Four boxes on the left, Motorola, EDACS, LTR and MPT 1327, each emit a grant with a channel number. A resolver stage follows: Motorola's is solid and labelled built-in band plan; the other three are dashed, labelled none, zero hertz. All feed the engine's HandleGrant, whose first gate drops a zero frequency; the three dashed paths stop there. The surviving path passes lockout, hold and avoid and scan-list gates to VoicePool.Bind and CallStart, into the composer's classifyVoiceKind and runFMChain, and finally the recorder's WritePCM and a WAV.">
  <text x="8" y="36" fill="currentColor" font-size="9" font-weight="bold">motorola</text>
  <text x="8" y="80" fill="currentColor" font-size="9" font-weight="bold">edacs</text>
  <text x="8" y="124" fill="currentColor" font-size="9" font-weight="bold">ltr</text>
  <text x="8" y="168" fill="currentColor" font-size="9" font-weight="bold">mpt1327</text>
  <rect x="66" y="22" width="90" height="22" fill="none" stroke="currentColor"/>
  <text x="111" y="37" text-anchor="middle" fill="currentColor" font-size="8">grant · ChannelNum</text>
  <rect x="66" y="66" width="90" height="22" fill="none" stroke="currentColor"/>
  <text x="111" y="81" text-anchor="middle" fill="currentColor" font-size="8">grant · LCN</text>
  <rect x="66" y="110" width="90" height="22" fill="none" stroke="currentColor"/>
  <text x="111" y="125" text-anchor="middle" fill="currentColor" font-size="8">grant · Channel</text>
  <rect x="66" y="154" width="90" height="22" fill="none" stroke="currentColor"/>
  <text x="111" y="169" text-anchor="middle" fill="currentColor" font-size="8">grant · Channel</text>
  <rect x="170" y="22" width="110" height="22" fill="none" stroke="var(--accent)"/>
  <text x="225" y="37" text-anchor="middle" fill="var(--accent)" font-size="8">built-in BandPlan → Hz</text>
  <rect x="170" y="66" width="110" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="225" y="81" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Resolver: none → 0 Hz</text>
  <rect x="170" y="110" width="110" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="225" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Resolver: none → 0 Hz</text>
  <rect x="170" y="154" width="110" height="22" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="225" y="169" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Resolver: none → 0 Hz</text>
  <line x1="280" y1="33" x2="300" y2="33" stroke="var(--accent)"/>
  <line x1="280" y1="77" x2="300" y2="77" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <line x1="280" y1="121" x2="300" y2="121" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <line x1="280" y1="165" x2="300" y2="165" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <rect x="300" y="22" width="96" height="154" fill="none" stroke="currentColor"/>
  <text x="348" y="40" text-anchor="middle" fill="currentColor" font-size="8" font-weight="bold">HandleGrant</text>
  <text x="348" y="58" text-anchor="middle" fill="var(--accent)" font-size="8">FrequencyHz == 0 → drop</text>
  <text x="348" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="8">lockout</text>
  <text x="348" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">hold / avoid</text>
  <text x="348" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">scan list</text>
  <text x="348" y="156" text-anchor="middle" fill="currentColor" font-size="8">startCall · Bind</text>
  <line x1="396" y1="33" x2="416" y2="33" stroke="var(--accent)"/>
  <rect x="416" y="22" width="110" height="22" fill="none" stroke="currentColor"/>
  <text x="471" y="37" text-anchor="middle" fill="currentColor" font-size="8">SetCenterFreq · CallStart</text>
  <line x1="526" y1="33" x2="546" y2="33" stroke="var(--accent)"/>
  <rect x="546" y="22" width="126" height="22" fill="none" stroke="currentColor"/>
  <text x="609" y="37" text-anchor="middle" fill="currentColor" font-size="8">classifyVoiceKind → FM</text>
  <line x1="609" y1="44" x2="609" y2="62" stroke="var(--accent)"/>
  <rect x="546" y="62" width="126" height="22" fill="none" stroke="var(--accent)"/>
  <text x="609" y="77" text-anchor="middle" fill="var(--accent)" font-size="8">runFMChain → PCM</text>
  <line x1="609" y1="84" x2="609" y2="102" stroke="var(--accent)"/>
  <rect x="546" y="102" width="126" height="22" fill="none" stroke="currentColor"/>
  <text x="609" y="117" text-anchor="middle" fill="currentColor" font-size="8">recorder.WritePCM → WAV</text>
  <text x="340" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="8">solid: a path a live grant can take today · dashed: stops at the engine's first gate</text>
</svg>
<figcaption>Only the Motorola lane carries hertz into the engine. The other three publish a channel number and a zero, and the engine's first gate ends their journey before the voice pool is consulted.</figcaption>
</figure>

## The engine's path to a tuner

A grant with a frequency meets the gates the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }})
series described, in order: the zero-frequency drop; the TETRA-only
radio/talkgroup reclassification (gated to `g.Protocol == "tetra"`); the
talkgroup lookup and lockout (`grant locked out`, emergency bypasses);
`holdAvoid.gate`; the scan-list mode. Then the pool. The same device
holding this call is a refresh (`grant already active; refreshed`); the
same call on another frequency, if the tuner `CanTune` it, is a
`pool.Retune` (`call followed to new frequency`); a free device gets
`startCall`; with none free, `CanPreempt` either ends a lower-priority
victim with `EndReasonPreempted` or publishes `KindGrantUnserved` with
`UnfollowedAllBusy`.

`startCall` is where the radio moves:

```go
// internal/trunking/voicepool.go (shape) — Bind
if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil { /* reacquire the handle, retry once */ }
g.CallID = p.callSeq.Add(1) // process-unique; the chain and recorder fence on it
// internal/trunking/engine.go (shape) — startCall
e.bus.Publish(events.Event{Kind: events.KindCallStart, Payload: CallStart{
    Grant: ac.Grant, Talkgroup: tg, DeviceSerial: d.Serial, StartedAt: ac.StartedAt,
}})
e.applyEncryptedPolicy(d.Serial, g, g.Encrypted)
```

The published `CallStart` carries `ac.Grant` because `Bind` stamped the
fresh `CallID`. `applyEncryptedPolicy` honours the `Encrypted` flag an
EDACS CCW or SmartNet OSW carries, so `skip_encrypted` applies to analog
trunking too. An EDACS grant also carries `ProVoice`; the engine ignores
it, the composer reads it next, and the recorder forces a `.raw` sidecar,
since no in-binary Aegis / ProVoice decoder exists.

## Inside `runFMChain`

`handleStart` computes one `voiceKind` per call. Digital protocols match by
name first; then:

```go
// internal/voice/composer/composer.go (shape) — classifyVoiceKind
isAnalogTrunk := proto == "motorola" || proto == "ltr" || proto == "mpt1327" ||
    (proto == "edacs" && !cs.Grant.ProVoice)
if proto == "" || proto == "fm" || proto == "fm-conv" || proto == "analog" || isAnalogTrunk {
    return voiceKindFM
}
return voiceKindUnsupported // YSF, EDACS ProVoice: "digital protocol not yet decoded; chain bypassed"
```

`TestComposerRunsFMChainForAnalogTrunking` publishes a `CallStart` for each
of `motorola`, `edacs`, `ltr` and `mpt1327` at 851 000 000 Hz and waits for
a chain on `VOICE-1`; `TestComposerBypassesEDACSProVoice` asserts a
`ProVoice: true` grant spawns none. The FM kind launches
`runFMChain(…, uint32(math.Round(rateHzF)), false, …)` — `false` is the AM
flag, and the rounded rate is fine because analog FM has no symbol clock to
drift.

The chain
[Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
sketched has grown stages since, each from a conventional-FM field report.
In order: **front-end decimation** — `newDecimatingFIR(iqHz, 48_000, c.bw,
true)`, an 81-tap Kaiser low-pass at `VoiceBandwidthHz` (12 500), 2.4 MS/s
down to 48 kHz; an optional **channel filter** — `newFMChannelFilter`, a
second 81-tap complex low-pass at 48 kHz, ±half of
`fm_channel_bandwidth_hz`, because at 2.4 MS/s an 81-tap FIR cannot
separate a 12.5 kHz channel from its neighbour while at 48 kHz it is
2–3 kHz sharp (`TestFMChannelFilterSelectivity`, #1184); an optional **CMA
equalizer** (`recordings.equalizer.enabled`); **`demod.FM`**; two cascaded
**high-pass** biquads at `fm_audio_highpass_hz` (300), stripping a carrier
offset's DC and the CTCSS/DCS tones *before* de-emphasis
(`TestComposerFMChainHighPassRemovesDC`); **de-emphasis** (`fm_deemphasis`:
`us` 75 µs default, `eu`, `off`); the **audio low-pass**
(`fm_audio_lowpass_hz` 3400); **AGC**; and **decimation to PCM** by 6, or
the opt-in polyphase `AudioResampler`, into `c.sink.WritePCM(serial, pcm)`.

Two details are trunked-analog-specific. `WritePCM` carries no `CallID`
fence because an analog chain keys on a stable physical serial. And the
#1090 squelch gate — freeze the AGC and fade to silence while the scanner
reports the channel closed — asks `squelch.SquelchOpen(serial)` and
**leaves the chain ungated on `ok = false`**, which is what every
analog-trunk chain gets: no scanner-side decision exists for a trunked tap
(`TestComposerFMChainIgnoresSquelchWithoutDecision`).

## Ending a call nobody signals

Here the FM-era protocols diverge from everything digital. No
`publishGrant` in `motorola`, `edacs` or `ltr` has a companion release;
none of the four publishes `KindCallRelease`. P25 has a terminator, DMR a
terminator with LC, TETRA a D-RELEASE; SmartNet has an OSW stream that
stops mentioning the call, LTR a status word whose F-bit clears, and
GopherTrunk turns neither into an event. An analog call ends only by the
mechanisms every chain shares.

`runFMChain` builds a `boundaryTracker` with `grantTG` 0 — gating
disabled — and calls `bt.onVoice(0)` after every PCM write. The tracker's
`run` loop ends the call `hangtime` (`voice_hangtime_ms`, 3500) after the
last write, or after `noVoiceStartupFactor` × hangtime (7 s) if PCM never
started (`EndReasonTimeout`), and `Touch`es the engine only when the
last-voice timestamp advanced — the #356 fix, so a stalled IQ source cannot
keep a call alive by heartbeat. Above it, `runWatchdog` fires at
`call_timeout_ms` (30 000) after `LastHeardAt` stops moving.

The chain's own comment states the consequence: analog FM "emits PCM
continuously, so in practice the engine's grant lifecycle / watchdog bounds
the call". While IQ flows, PCM flows — discriminator noise after the
carrier drops is still PCM — so neither hangtime nor the watchdog elapses.
What ends a trunked analog call live is the engine: a grant that retunes or
rebinds the device, pool preemption, or `EndCall` from the API. Nothing
quiets an analog-trunk tap on carrier loss; the noise-quieting squelch
CLAUDE.md names for the conventional scanner — the `dmrrx.carrierGate` idea
from
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
— is "not built" there either. Nor does the FM chain ever call
`bt.onTransmissionEnd()`, so `voice_call_grouping: transmission` has no
over boundary to roll on and an analog call is one file
([Recording, Composition & Streaming Part 3]({{ '/blog/deep-dives/recording-streaming-03-assembling-a-call/' | relative_url }})).
When the end comes, `handleEnd` cancels the chain and blocks on `done`;
`emitTail` writes a 10 ms fade so the cut does not click
(`TestComposerTailFadeOnCallEnd`); only then is the file finalized.

## The recorder's side, and the rung

For an analog protocol the recorder opens no vocoder:
`RecorderOptions.VocoderForProtocol`'s comment says protocols not in the
map "produce no decoded audio — typically analog protocols (motorola,
edacs, ltr, mpt1327) where the composer's FM chain feeds WritePCM
directly". `WritePCM` locks the session and calls `s.wav.WriteSamples` on
the file `NewAudioFileWriter` opened lazily on the first write, in the
configured `recordings.format`. No `.raw` sidecar exists for analog except
ProVoice. `CallComplete`, the call-log row and streaming are protocol-blind.

The composer half is **pinned in CI**:
`TestComposerRunsFMChainForAnalogTrunking`,
`TestComposerBypassesEDACSProVoice`, and the stage tests from
conventional-FM field reports (`TestFMChannelFilterSelectivity`,
`TestComposerFMChainHighPassRemovesDC`,
`TestComposerFMChainMutesSquelchClosedTail`). The FM chain is in daily use
under the conventional scanner, where those reporters' captures are
real-air fixtures
([From the Issue Tracker Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})).

The trunked half is **on-air unverified for all four**. Motorola's grants
reach the engine with hertz, so its whole path is one #1143 capture away
([Part 3]({{ '/blog/deep-dives/legacy-family-03-smartnet-air-interface/' | relative_url }})).
EDACS, LTR and MPT 1327 cannot reach the engine until a resolver is wired
and a config key names it; their composer path is verified only with a
grant the test filled in by hand. `status.md`'s "Analog FM trunking
(Motorola Type II, EDACS, LTR, MPT 1327) decodes voice through the
composer's FM chain" is true of the composer and silent about the gap in
front of it. A green composer test is not a followed call.

### How analog trunking shaped the Go code

- **One classifier, computed once.** `voiceKind` replaced a fan of
  per-protocol booleans; `isAnalogTrunk` lives in exactly one place.
- **The AM flag reuses the chain.** `runFMChain(…, am bool, …)` swaps in
  `demod.AM` and skips de-emphasis and CMA; everything downstream is shared.
- **A second filter at the right rate.** `newFMChannelFilter` runs at
  48 kHz because that is where an 81-tap FIR is sharp; nil when unset.
- **Resolvers are optional by type, mandatory by physics.** Every
  `publishGrant` tolerates a nil `Resolver`; the engine does not tolerate
  the zero it produces.

## Where this goes next

The FM-era half of the family is done: four control channels, one analog
chain, one resolver gap. The AMBE-era half starts with the protocol that
has the most instrumentation and the least air.
[Part 8]({{ '/blog/deep-dives/legacy-family-08-nxdn-physical-layer/' | relative_url }})
takes NXDN's physical layer — 4FSK at 4800 symbols per second, the
direction-specific FSW, the doubled-bit LICH, the 15-bit scrambler model —
and the 1800 Hz deviation knob the committed `NXDN96 IQ.wav` says is wrong
for at least one transmitter.

## FAQ

**Why does GopherTrunk log "dropping grant with zero frequency" on an EDACS, LTR or MPT 1327 system?**
Those decoders resolve a channel number through an optional `Resolver`, and
`newEDACSPipeline`, `newLTRPipeline` and `newMPT1327Pipeline` install none —
there is no `edacs_` / `ltr_` / `mpt1327_` band-plan key. `publishGrant`
stamps `FrequencyHz` 0 and `Engine.HandleGrant` drops the grant as its
first check.

**How does GopherTrunk record analog trunked voice?**
`classifyVoiceKind` maps `motorola`, `ltr`, `mpt1327` and non-ProVoice
`edacs` to `voiceKindFM`; `runFMChain` decimates the voice IQ to 48 kHz,
FM-demodulates, applies the 300 Hz high-pass, de-emphasis, 3.4 kHz low-pass
and AGC, and writes 8 kHz PCM through `WritePCM`. The recorder opens no
vocoder and writes the PCM straight to the WAV or FLAC.

**What ends an analog trunked call?**
Not the signalling — no FM-era decoder publishes a release. The
`boundaryTracker` ends a call 3.5 s after PCM stops or 7 s after a start
with no PCM; the watchdog fires at `call_timeout_ms` (30 s). Because the FM
chain writes PCM as long as IQ flows, on a live system the engine's grant
lifecycle or pool preemption usually ends the call.

**Does `fm_channel_bandwidth_hz` affect digital voice?**
No. It builds a second complex low-pass at the 48 kHz intermediate rate in
`runFMChain` only, ±half the configured width ahead of the discriminator.
The digital chains size their own filters from `VoiceBandwidthHz`, and
`TestFMChannelFilterSelectivity` pins that the analog knob leaves them
untouched.

**Is analog trunked voice verified on air?**
No, for all four. The composer's FM chain is pinned in CI and field-proven
on conventional FM, but no trunked analog call has been followed live:
Motorola waits on the #1143 capture, and EDACS, LTR and MPT 1327 cannot
reach the engine until a band-plan resolver is wired through their pipeline
factories.

## Series navigation

**Part 7 of 14** · ←
[Part 6: MPT 1327 — FFSK Codewords, CWSC Tolerance and BCH(64,48)]({{ '/blog/deep-dives/legacy-family-06-mpt1327/' | relative_url }})
· Next →
[Part 8: NXDN Physical Layer — 4FSK at 4800, FSW, LICH and Scrambling]({{ '/blog/deep-dives/legacy-family-08-nxdn-physical-layer/' | relative_url }})
