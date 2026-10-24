---
title: "The Legacy Family End to End, Part 1: What 'Legacy' Means — Two Generations, One Verification Ladder"
description: "The eight protocols the P25, DMR and TETRA series left out — SmartNet, EDACS, LTR, MPT 1327, NXDN, dPMR Mode 3, D-STAR and System Fusion — sorted into two generations, located in GopherTrunk's protocol enum and pipeline factories, and placed on a four-rung verification ladder that says, protocol by protocol, what is proven and what a capture would move."
category: deep-dives
keywords: legacy trunking protocols, motorola type ii smartnet decoder, edacs decoder, ltr trunking sdr, mpt 1327 ffsk, nxdn decoder, dpmr mode 3, d-star gmsk, yaesu system fusion, verification ladder, reference-pinned decoder, gophertrunk legacy family
tags: [legacy-family-end-to-end, smartnet, edacs, ltr, nxdn, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 1
---

*Part 1 of **The Legacy Family End to End**, a 14-part deep dive that
follows the protocols the three earlier End-to-End series left out —
the FM-era trunking generation (Motorola Type II / SmartNet, EDACS,
LTR, MPT 1327) and the AMBE-era narrowband and amateur modes (NXDN,
dPMR Mode 3, D-STAR, Yaesu System Fusion) — through the layers
[P25 End to End]({{ '/blog/series/p25-end-to-end/' | relative_url }}),
[DMR End to End]({{ '/blog/series/dmr-end-to-end/' | relative_url }}) and
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }})
walked. The difference is where these decoders stand.
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
drew the line between a green synthetic test and an on-air pass; most
of this family sits below it. This opener defines the ladder the series
measures every protocol against, and places all eight on it.*

> **TL;DR:** "Legacy" here means two generations. The **FM-era** one
> signals in audio-band or sub-audible FSK and carries analog FM voice:
> SmartNet (3600-baud 2-FSK at ±1.2 kHz, 84-bit OSW frames), EDACS
> (9600-baud GFSK, 40-bit CCWs under BCH(40,28,2)), LTR (a 41-bit status
> word at 300 bps under every repeater's voice, no control channel) and
> MPT 1327 (1200-baud FFSK, 64-bit codewords). The **AMBE-era** one is
> 4-level or GMSK at 4800 symbols/s with AMBE voice: NXDN, dPMR Mode 3,
> D-STAR, YSF. All eight are one line in `trunking.Protocol`
> (`internal/trunking/site.go`) and one factory in `ccdecoder`'s
> `factories` map; the DDC channelizes all of them to 48 kHz except
> SmartNet (18 kHz, `motorolaDDCTargetRateHz`). On the four-rung ladder
> — **on-air verified / capture-pinned / reference-pinned /
> placeholder** — none stands on the top rung. SmartNet, EDACS, LTR and
> dPMR control are reference-pinned; MPT 1327 is pinned to committed
> audio captures; NXDN and YSF are capture-gated with harnesses waiting;
> the three AMBE voice chains carry placeholder interleave tables; YSF
> voice and EDACS ProVoice are bypassed.

**Key takeaways**

- **Two generations, one engine.** A SmartNet OSW, an EDACS CCW, an LTR
  status word and an NXDN CAC all end as the same `trunking.Grant`; the
  generations differ in modulation, voice coding and how much of each
  decoder has met real air.
- **Enum, factory, rate.** A protocol exists when it has a
  `trunking.Protocol` value, a `PipelineFactory` in `factories`, and a
  channel rate from `ddcTargetForProtocol`.
- **The ladder is the series' point.** Every part states its protocol's
  rung and the capture that would move it, sourced from the status and
  capture-needs documents and the code's own comments.
- **The #764/#771 rule applies to every row.** A green synthetic test
  is a claim of consistency, not correctness; for this family that is
  the shape of the whole table, not a caveat at the end.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Protocol names | `ProtocolEDACS`, `ProtocolMotorola`, `ProtocolLTR`, `ProtocolMPT1327`, `ProtocolNXDN`, `ProtocolDPMR`, `ProtocolDStar`, `ProtocolYSF` + `ParseProtocol` strings | `internal/trunking/site.go` |
| Pipeline factories | `newMotorolaPipeline`, `newEDACSPipeline`, `newLTRPipeline`, `newMPT1327Pipeline`, `newNXDNPipeline`, `newDPMRPipeline`, `newDStarPipeline`, `newYSFPipeline` | `internal/scanner/ccdecoder/pipelines.go` (`factories`) |
| Channel rate | 48 kHz default, 18 kHz Motorola, 144 kHz TETRA | `ccdecoder/ddc.go` (`ddcTargetForProtocol`) |
| Symbol boundary | 2-level → `BitSink`, 4-level → `DibitSink`; `tapBits` / `tapDibits` observe both | `pipelines.go` (`PipelineOptions.SymbolTap`) |
| Voice routing | analog trunk → `voiceKindFM`; `nxdn`/`dpmr`/`dstar` → AMBE (`ambe2-dmr`, `ambe2-dmr`, `ambe2`); YSF, ProVoice → `voiceKindUnsupported` | `composer.go` (`classifyVoiceKind`), `recorder.go` (`DefaultVocoderForProtocol`) |
| The ladder's sources | what ships, what is gated, what is a placeholder | `docs/status.md`, `docs/decoder-capture-needs.md` |

## In this post

- **Two generations** — the FM-era and AMBE-era families.
- **Where a protocol becomes a name** — enum, factory map, channel rate.
- **The symbol boundary** — bits versus dibits, and the voice fork.
- **The verification ladder** — four rungs, defined once.
- **Eight protocols, one table** — each rung and the capture that moves it.
- **What #764/#771 means here** — why the ladder is the story.

## Two generations

The earlier End-to-End series each had a reporter's capture or a live
run to anchor them. This family mostly does not, and it splits into two
groups with almost nothing in common at the physical layer.

The **FM-era trunking generation** carries its signalling where a 1980s
radio could put it. Motorola Type II / SmartNet transmits **84-bit
frames at 3600 baud over binary FSK at roughly ±1.2 kHz** — an 8-bit
sync, 76 coded bits, one 27-bit Outbound Status Word per frame
(`internal/radio/motorola`). EDACS runs a continuous **9600-baud GFSK**
control channel, BT = 0.3, with a 24-bit sync and 40-bit Control Channel
Words under a shortened **BCH(40,28,2)** (`internal/radio/edacs`). LTR
has no control channel: every repeater sends a **41-bit status word at
300 bps** underneath its own voice (`internal/radio/ltr`). MPT 1327
carries **1200-baud FFSK** — 1200 Hz mark, 1800 Hz space — inside a
narrowband FM audio channel, 64-bit codewords back to back
(`internal/radio/mpt1327`). Voice on all four is analog FM on the granted
channel, so the composer sends them down the `runFMChain` a conventional
scanner channel uses.

The **AMBE-era generation** replaced that with narrowband digital voice
at 4800 symbols/s. NXDN is 4-level FSK with a spec peak deviation of
**1800 Hz** (`nxdn_deviation_hz` overrides it) and a CAC under a K=5
rate-½ convolutional code. dPMR Mode 3 is the 6.25 kHz cousin, 4-level
at **2400 symbols/s** with half the deviation (`DeviationHz: 900.0` in
its factory). D-STAR is **GMSK at 4800 bps, BT = 0.5**, a 660-bit
on-wire shell around a 328-bit header. YSF is C4FM at 4800 with a
trellis-coded FICH. Their voice is AMBE — `ambe2-dmr` for NXDN and dPMR,
the base `ambe2` for D-STAR — and, as the status page says, each of
those chains is wired end to end "but not yet verified on air: each
chain's AMBE interleave table is a documented placeholder awaiting a
real voice capture." YSF voice is not rendered at all.

GopherTrunk carries both generations because operators still meet them:
utility fleets on LTR and MPT 1327, county systems still on SmartNet or
EDACS, amateur repeaters on D-STAR and Fusion.

## Where a protocol becomes a name

Three places in the tree decide that a protocol exists. The first is
the enum:

```go
// internal/trunking/site.go (shape)
const (
    ProtocolUnknown   Protocol = iota
    ProtocolP25
    ProtocolDMR
    ProtocolNXDN               // NXDN
    ProtocolDPMR               // dPMR Mode 3
    ProtocolEDACS              // EDACS / GE-Marc
    ProtocolMotorola           // Motorola Type II / SmartZone
    ProtocolLTR                // Logic Trunked Radio
    ProtocolMPT1327            // MPT 1327
    ProtocolP25Phase2
    ProtocolTETRA
    ProtocolYSF                // System Fusion (config "ysf")
    ProtocolDStar              // D-STAR (config "dstar")
    // …
)
```

`ParseProtocol` maps the YAML strings — `"edacs"`, `"motorola"`,
`"ltr"`, `"mpt1327"`, `"nxdn"`, `"dpmr"`, `"ysf"`, and `"dstar"` with
its `"d-star"`/`"d_star"` aliases — and `System.Validate` accepts any of
them with at least one `control_channels` frequency between 25 and
1300 MHz. The same `System` struct carries the per-protocol knobs the
series will keep meeting, from `MotorolaBandPlan` and `EDACSBCHMode` to
`NXDNDeviationHz` and `DStarFECMode`.

The second place is the factory map in `ccdecoder`:

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
var factories = map[trunking.Protocol]PipelineFactory{
    trunking.ProtocolP25:       newP25Phase1Pipeline,
    // …
    trunking.ProtocolDPMR:      newDPMRPipeline,
    trunking.ProtocolNXDN:      newNXDNPipeline,
    trunking.ProtocolEDACS:     newEDACSPipeline,
    trunking.ProtocolMotorola:  newMotorolaPipeline,
    trunking.ProtocolLTR:       newLTRPipeline,
    trunking.ProtocolMPT1327:   newMPT1327Pipeline,
    // …
    trunking.ProtocolYSF:       newYSFPipeline,
    trunking.ProtocolDStar:     newDStarPipeline,
}
```

Each factory builds a receiver from `internal/radio/<proto>/receiver`
and a `ControlChannel` from the parent package, and wires the receiver's
sink into `ControlChannel.Process`. `NewPipeline` and
`RegisteredProtocols` expose the same map to siglab, so every protocol
here can be driven through its production pipeline from a file.

The third place is the channel rate. `ddcTargetForProtocol` returns
`tetraDDCTargetRateHz` (144000) for TETRA, `motorolaDDCTargetRateHz`
(18000) for Motorola, and `ddcTargetRateHz` (48000) for everything else.
The 18 kHz exception is not only samples per symbol: the DDC's
anti-alias filter doubles as the channel-select filter, and at 48 kHz
its ±24 kHz passband admits the adjacent 25 kHz-spaced SmartNet channels
into the discriminator — the constant's comment cites
[#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143). At
48 kHz the other FM-era receivers get 5 samples per symbol (EDACS), 40
(MPT 1327) and 160 (LTR); the AMBE-era family gets the C4FM family's 10.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Two rows of protocol boxes: the FM-era generation (SmartNet, EDACS, LTR, MPT 1327) above the AMBE-era generation (NXDN, dPMR, D-STAR, YSF), each labelled with its modulation and symbol rate, all converging on one trunking.Grant box feeding the engine.">
  <text x="20" y="18" fill="var(--accent)" font-size="10" font-weight="bold">FM-era trunking generation — analog FM voice</text>
  <rect x="20" y="26" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="95" y="42" text-anchor="middle" fill="currentColor" font-size="10">SmartNet / Type II</text>
  <text x="95" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="8">3600 bd 2-FSK ±1.2 kHz · 84-bit OSW</text>
  <rect x="180" y="26" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="255" y="42" text-anchor="middle" fill="currentColor" font-size="10">EDACS</text>
  <text x="255" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="8">9600 bd GFSK · 40-bit CCW BCH(40,28,2)</text>
  <rect x="340" y="26" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="415" y="42" text-anchor="middle" fill="currentColor" font-size="10">LTR</text>
  <text x="415" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="8">300 bps sub-audible · 41-bit word · no CC</text>
  <rect x="500" y="26" width="160" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="580" y="42" text-anchor="middle" fill="currentColor" font-size="10">MPT 1327</text>
  <text x="580" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1200 bd FFSK 1200/1800 Hz · 64-bit codeword</text>
  <text x="20" y="98" fill="var(--accent)" font-size="10" font-weight="bold">AMBE-era narrowband / amateur generation — digital voice</text>
  <rect x="20" y="106" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="95" y="122" text-anchor="middle" fill="currentColor" font-size="10">NXDN</text>
  <text x="95" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">4FSK 4800 · 1800 Hz · ambe2-dmr</text>
  <rect x="180" y="106" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="255" y="122" text-anchor="middle" fill="currentColor" font-size="10">dPMR Mode 3</text>
  <text x="255" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">4FSK 2400 · 900 Hz · ambe2-dmr</text>
  <rect x="340" y="106" width="150" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="415" y="122" text-anchor="middle" fill="currentColor" font-size="10">D-STAR</text>
  <text x="415" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">GMSK 4800 BT 0.5 · ambe2</text>
  <rect x="500" y="106" width="160" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="580" y="122" text-anchor="middle" fill="currentColor" font-size="10">YSF</text>
  <text x="580" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">C4FM 4800 · FICH · voice bypassed</text>
  <line x1="95" y1="66" x2="300" y2="184" stroke="var(--fg-muted)"/>
  <line x1="255" y1="66" x2="320" y2="184" stroke="var(--fg-muted)"/>
  <line x1="415" y1="66" x2="360" y2="184" stroke="var(--fg-muted)"/>
  <line x1="580" y1="66" x2="380" y2="184" stroke="var(--fg-muted)"/>
  <line x1="95" y1="146" x2="300" y2="184" stroke="var(--fg-muted)"/>
  <line x1="255" y1="146" x2="320" y2="184" stroke="var(--fg-muted)"/>
  <line x1="415" y1="146" x2="360" y2="184" stroke="var(--fg-muted)"/>
  <line x1="580" y1="146" x2="380" y2="184" stroke="var(--fg-muted)"/>
  <rect x="250" y="184" width="180" height="36" rx="5" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="199" text-anchor="middle" fill="var(--accent)" font-size="10">trunking.Grant → engine</text>
  <text x="340" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="8">events.KindGrant · one shape for all eight</text>
</svg>
<figcaption>Two generations with nothing in common at the antenna converge on one grant shape; only the composer's voice fork tells them apart.</figcaption>
</figure>

## The symbol boundary

The cleanest line between the generations is the type of the receiver's
sink. The FM-era protocols are 2-level, so their receivers emit raw
bits through a `BitSink func(bits []byte, baseIdx int)` — each package
declares its own identical type. The 4FSK protocols emit a `DibitSink`. `PipelineOptions.SymbolTap` normalises both — `tapBits`
forwards with `isBits=true`, `tapDibits` with `false` — so siglab's
signal-quality analyser sees every protocol through one hook. `baseIdx`
is the absolute symbol index since the receiver's last `Reset`, pinned
monotonic by `TestReceiverBitSinkBaseIdxMonotonic` in the EDACS and LTR
receiver tests.

Behind the grant the generations fork again. The composer's
`classifyVoiceKind` sends `"motorola"`, `"ltr"`, `"mpt1327"` and
`"edacs"`-without-`ProVoice` to `voiceKindFM`; `"nxdn"`, `"dpmr"` and
`"dstar"` get their own AMBE kinds, the last two annotated `experimental,
unverified on air`; YSF and EDACS ProVoice fall through to
`voiceKindUnsupported` and log `composer: digital protocol not yet
decoded; chain bypassed`. The engine's `discoveredMode` makes the same
split for the talkgroup catalogue — `"A"` for the analog protocols,
`"D"` otherwise (a #1143 fix).

## The verification ladder

The earlier series could use verification language loosely because a
reporter's capture anchored them. This family needs the terms pinned
down, so the series uses four rungs throughout.

- **On-air verified.** A live daemon run, or an operator's capture
  replayed through the production path, decoded to the expected
  identities — the standard DMR direct mode and MDC1200 reached. Nothing
  in this family has.
- **Capture-pinned.** A real-air recording exists and a committed or
  skip-gated test decodes it to known values. TETRA's
  `tetra_cc_sync_loss_2s_144k.cs16` is the model; here only MPT 1327's
  audio captures qualify, and only for the audio-band half of the chain.
- **Reference-pinned.** Constants pinned by literal tests against an
  independent implementation proven on air — OP25, trunk-recorder,
  sdrtrunk, lwvmobile/edacs-fm, MMDVMHost — plus a failing-first
  regression, but no real-air capture through the chain.
- **Placeholder.** A documented stand-in the code itself labels
  best-effort or awaiting a capture, such as the AMBE interleave tables
  in `voice_ambe.go`.

The gap between the middle rungs is the lesson of
[the self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}):
a round-trip proves encoder and decoder agree; a reference literal
proves the decoder agrees with someone who has decoded real air; only a
capture proves the air agrees.

## Eight protocols, one table

The rungs below come from `docs/status.md`, `docs/decoder-capture-needs.md`,
`docs/protocol-feature-parity.md` and the packages' own comments.

| Protocol | Control decode | Voice | What would move it |
|---|---|---|---|
| Motorola Type II | **Reference-pinned**: OP25 + trunk-recorder literals (`TestOutboundSyncBitsMatchReference`, `TestXORMasksMatchReference`), failing-first `TestProcessDecodesRealAirFormat` | analog FM | the #1143 reporter's 854.5625 MHz, 3 MS/s capture |
| EDACS | **Reference-pinned** BCH (`lwvmobile/edacs-fm`, generator `0x1539`); sync `0x55D5AA` and field layout labelled best-effort in `sync.go` / `ccw.go` | analog FM; **ProVoice bypassed** | clean control-channel IQ with a known system ID |
| LTR | **Reference-pinned** FCS (sdrtrunk `CRCLTR.java` table); 41-bit layout per "the most-cited public reference", cross-check caveat in `status.go` | analog FM | a sub-audible capture with known area / home / group |
| MPT 1327 | **Capture-pinned (audio)**: `samples/mpt1327/MPT1327_423.6_{1,2}.mp3`; Tier 3 in capture-needs, "already decodes real audio end-to-end today" | analog FM | 48 kHz IQ to exercise the FFSK front end |
| NXDN | **Capture-gated**: harness ready (`integration_cc_nxdn_realair_test.go`, `TestReplayNXDNRealCapture`); soft decision measured 26/200 → 151/200 at σ=0.7 (synthetic) | `ambe2-dmr`, **placeholder** interleave | ≥ 5 s of RCCH IQ at 48 kHz |
| dPMR Mode 3 | **Reference-pinned** control (ETSI TS 102 658 layout, strict-validated opcodes) | `ambe2-dmr`, **placeholder** interleave | ≥ 10 s of clear voice IQ |
| D-STAR | header path, `dstar_fec_mode` **off by default** | `ambe2`, **placeholder** interleave, marked experimental | ≥ 10 s of clear voice IQ |
| YSF | **Capture-gated**: FICH codec per MMDVMHost (`EncodeFICHOnAir` / `DecodeFICHOnAir`) | **bypassed** | ≥ 10 s of DN-mode IQ |

Two things in that table matter later. The integration tests
`TestDaemonCCDecodesEDACS`, `TestDaemonCCDecodesLTR`,
`TestDaemonCCDecodesMotorola` and `TestDaemonCCDecodesMPT1327` all pass
and all synthesise their own IQ — they prove the daemon plumbing, not
the air interface. And the status page's "every trunked control
modulation in the Features table has an end-to-end IQ → CC chain
shipping" is true about *chains*, not *verification*. Both hold at
once; that is what the ladder is for.

## What #764/#771 means for this family

The repo's standing rule — a green synthetic is never an on-air pass —
was learned on P25 and TETRA, where the cost was a closed issue that had
to be reopened. For the legacy family it decides what this series may
claim. The SmartNet rebuild
([From Spec to Shipping Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}))
is the family's own worked example: a decoder green on every test for
months, framed on a format no transmitter ever sent; its replacement
cites every constant to a proven decoder and still calls itself "a
strong hypothesis with excellent provenance — nothing more."

So each part ends its protocol on an explicit rung, names the test that
pins what is pinned, and names the capture that is missing. Where the
code calls a layout best-effort, the post says so; where a chain renders
audio through a placeholder table, the post does not call it verified.
The overviews in
[Protocol Decoders Part 8]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
and
[Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
introduced these packages; this series goes a layer down, into the bit
layouts and the tests.

## Where this goes next

Before walking each protocol, the series needs the skeleton they share:
a `receiver/` subpackage with one `Options` shape, a
`ControlChannel.Process` adapter with three framing strategies, a
band-plan resolver (and the two packages where the production factory
never wires one), a `LockState` that satisfies the hunter's
`LockedPayload`, and the `strict_test.go` pattern.
[Part 2]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
lays that out once, so Parts 3 to 12 can spend their words on what
differs.

## FAQ

**Which protocols does "the legacy family" include in GopherTrunk?**
Eight: the FM-era trunking generation — Motorola Type II / SmartNet
(`motorola`), EDACS (`edacs`), LTR (`ltr`), MPT 1327 (`mpt1327`) — and
the AMBE-era modes — NXDN (`nxdn`), dPMR Mode 3 (`dpmr`), D-STAR
(`dstar`), Yaesu System Fusion (`ysf`). Each is a `trunking.Protocol`
value and a factory in `ccdecoder`'s `factories` map.

**Is any legacy protocol on-air verified in GopherTrunk?**
No. None of the eight has a live run or an operator capture decoded
through the production path. SmartNet, EDACS, LTR and dPMR control are
reference-pinned; MPT 1327 is pinned to committed audio captures; NXDN
and YSF are capture-gated; the NXDN, dPMR and D-STAR voice chains use
documented placeholder interleave tables.

**What does "reference-pinned" mean for a decoder?**
Its constants are checked by literal tests against an independent
implementation proven on air — for SmartNet, OP25's
`SMARTNET_SYNC_MAGIC = 0xAC` and `ID_XOR`/`CMD_XOR`; for LTR, sdrtrunk's
24-entry CRC-7 table — and a regression test fails against the old code.
No real signal has yet been through the chain.

**Do the legacy protocols record voice?**
The four FM-era protocols route to the composer's analog FM chain
(`voiceKindFM`); EDACS ProVoice grants are bypassed. NXDN, dPMR and
D-STAR render through `ambe2-dmr`, `ambe2-dmr` and `ambe2`, but their
interleave tables are documented placeholders, so that audio is
unverified. YSF voice is bypassed entirely.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Shared Skeleton — ControlChannel.Process, Band Plans, Topology, strict_test]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
