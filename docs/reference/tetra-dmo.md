---
slug: tetra-dmo
title: TETRA DMO
entry_type: term
category: trunked-radio
description: "TETRA Direct Mode Operation (DMO) is radio-to-radio TETRA with no infrastructure: a transmitting radio sends a Direct Mode Synchronisation Burst to let others acquire, then Direct Mode Normal Bursts carrying the call — reusing the trunked-mode physical layer with a different burst field layout."
keywords: TETRA DMO, direct mode operation, DSB, DNB, direct mode synchronisation burst, direct mode normal burst, SCH/S, EN 300 396-2, radio-to-radio, back-to-back
aka: [DMO, "direct mode operation", "TETRA direct mode"]
autolink: true
infobox:
  - { label: Mode, value: "Radio-to-radio, no infrastructure" }
  - { label: Bursts, value: "DSB (sync) + DNB (normal)" }
  - { label: Physical layer, value: "Reused from TMO (π/4-DQPSK)" }
  - { label: Spec, value: "ETSI EN 300 396-2 §9.4.3" }
see_also: [tetra, direct-mode-operation, tetra-burst-formats, tetra-logical-channels, tetra-tchs-speech-coding, tetra-scrambler, pi-4-dqpsk, tdma]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Direct_mode_operation
---

**TETRA DMO** (**Direct Mode Operation**) is TETRA's infrastructure-less peer-to-peer mode:
two radios talk [directly](/reference/direct-mode-operation/) with no base station or
[control channel](/reference/control-channel/) between them.[^tetra][^dmo] A transmitting
station sends a **Direct Mode Synchronisation Burst (DSB)** to let the other radios acquire,
then a train of **Direct Mode Normal Bursts (DNB)** carrying the call. Because there is no
trunking layer, the trunked-mode ingestion path — hunt a control channel, follow grants —
does not apply; a DMO receiver instead camps a direct-mode channel, detects the DSB, and
follows the burst train.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A DMO burst train: a Direct Mode Synchronisation Burst carrying a frequency-correction field, a 120-bit SCH/S block, the synchronisation training sequence and a 216-bit block, followed by Direct Mode Normal Bursts each carrying two 216-bit blocks around the normal training sequence." xmlns="http://www.w3.org/2000/svg">
  <text x="14" y="24" font-size="8.5" fill="currentColor">DSB (acquire)</text>
  <g font-size="6.5" fill="currentColor" text-anchor="middle">
    <rect x="14" y="30" width="52" height="24" fill="none" stroke="currentColor" stroke-width="1"/><text x="40" y="45">freq-corr</text>
    <rect x="66" y="30" width="70" height="24" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/><text x="101" y="42">SCH/S</text><text x="101" y="51">120 bits</text>
    <rect x="136" y="30" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="156" y="45">sync tr</text>
    <rect x="176" y="30" width="80" height="24" fill="none" stroke="currentColor" stroke-width="1"/><text x="216" y="45">BKN2 216</text>
  </g>
  <text x="14" y="82" font-size="8.5" fill="currentColor">DNB (call) × N</text>
  <g font-size="6.5" fill="currentColor" text-anchor="middle">
    <rect x="14" y="88" width="90" height="24" fill="none" stroke="currentColor" stroke-width="1"/><text x="59" y="103">BKN1 216</text>
    <rect x="104" y="88" width="40" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="124" y="103">norm tr</text>
    <rect x="144" y="88" width="90" height="24" fill="none" stroke="currentColor" stroke-width="1"/><text x="189" y="103">BKN2 216</text>
    <rect x="242" y="88" width="90" height="24" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/><text x="287" y="103">BKN1 …</text>
  </g>
  <text x="235" y="136" text-anchor="middle" font-size="8" fill="currentColor">same π/4-DQPSK, training sequences and scrambler as TMO — only field layout differs</text>
</svg>
<figcaption>A DMO transmission opens with a DSB (frequency correction, a 120-bit SCH/S, sync training sequence and a 216-bit block) then continues as a train of DNBs, each two 216-bit blocks around the normal training sequence.</figcaption>
</figure>

## A reused physical layer

The DMO air interface *reuses* the trunked-mode (TMO) physical layer wholesale: identical
[π/4-DQPSK](/reference/pi-4-dqpsk/) at 18 ksym/s, 25 kHz channels, 255-symbol (14.167 ms)
[TDMA](/reference/tdma/) timeslots, the four-slot frame / 18-frame multiframe, the *same*
normal and synchronisation training sequences, and the *same* 32-tap
[scrambler](/reference/tetra-scrambler/) polynomial (colour code 0 for the SCH/S and SCH/H of
a DSB, exactly as TMO scrambles its BSCH). So the receiver, sync-word correlation and
channel-coding machinery are shared between the two modes — only the burst **field layout**
differs, which is the one thing DMO needs to add.

The channel coding is likewise not a new code family: DMO's SCH/S, SCH/H, SCH/F and TCH/S run
the *same* per-channel chains as their TMO [logical-channel](/reference/tetra-logical-channels/)
counterparts, so the existing decoders handle them once the blocks are sliced, de-rotated and
descrambled. The only DMO-specific coding rules are the colour-0 seed for the DSB signalling
blocks and the field boundaries below.

## The two burst kinds

| Burst | Layout (relative to training-sequence lead dibit L) |
| --- | --- |
| DSB (sync) | 40-dibit frequency correction · 60-dibit SCH/S (120 type-5 bits, BKN1) · 19-dibit sync training sequence · 108-dibit BKN2 (216 bits) |
| DNB (normal) | 108-dibit BKN1 (216 bits) · 11-dibit normal training sequence · 108-dibit BKN2 (216 bits) |

The **DSB** is the acquisition burst: its frequency-correction field lets a cold receiver
lock, and its **SCH/S** (the 120-bit block ahead of the sync training sequence) carries the
synchronisation PDU — the DM colour code and the master's slot/frame numbering used to anchor
the DNB traffic that follows. The SCH/S decodes exactly like a TMO BSCH (colour 0). The
**DNB** is the payload burst: its two 216-bit blocks carry either TCH/S speech (decoded to the
two 137-bit ACELP frames by the shared [TCH/S](/reference/tetra-tchs-speech-coding/) chain) or
SCH/F short-data signalling — a receiver tells them apart by which decode's CRC passes. The
block boundaries relative to the training-sequence lead dibit are *not* the same as TMO's NDB,
so DMO needs its own slicer.

## Configuring a DMO system

A DMO channel is decoded by setting a system's `protocol: tetra-dmo` (aliases `dmo` /
`tetra_dmo`) and pointing `control_channels` at the direct-mode frequency — the daemon *camps*
that frequency rather than hunting, locks on the first DSB, auto-recovers the DM colour code,
and records the DNB voice train. An optional `tetra_colour_code` overrides the auto-recovery
when the traffic colour is known.

```yaml
trunking:
  systems:
    - name: DMO
      protocol: tetra-dmo
      control_channels: [438900000]
      # tetra_colour_code: 3   # optional; 0/omitted = auto-recover the DM colour
```

## Scope and honesty

GopherTrunk decodes DMO end to end in the daemon: `newTETRADMOPipeline`
(`internal/scanner/ccdecoder/pipelines_dmo.go`) locks on the DSB SCH/S, recovers the DM colour
code, and grants; a same-carrier voice chain (`runTETRADMOVoiceChain`,
`internal/voice/composer/tetra_dmo_voice.go`) decodes the DNB TCH/S speech — both hard- and
soft-decision (the same ~2× yield lever the TMO traffic path gets) — through the clean-room
ACELP vocoder to a recording. The DM call-control *protocol* that rides in SCH/S / SCH/F —
source and destination SSI, group, call type — is EN 300 396-3, a separate specification, and
is **not** yet decoded, so a DMO call is recorded without a talkgroup/party identity (it files
under group `0`). And while the decode chain is validated offline (synthetic round-trips +
`TestTETRADMOReplay` on captures) and with a synthetic full-daemon lock test, it has **not**
yet been A/B'd against a real on-air DMO capture through the full daemon — the standing lesson
(#764/#771) is that synthetic round-trips can pass while on-air decode fails, so treat DMO
voice as functional-but-unverified-on-air until that A/B lands.

## Relevance to SDR

`internal/radio/tetra/dmo.go` defines the `DMBurstKind` (DSB/DNB), the burst geometry, and the
`ExtractDMBursts` / `ExtractDMBurstsSoft` slicers that correlate the training sequences under
all four residual π/4-DQPSK rotations; `dmo_stream.go` wraps them in a bounded sliding-window
`DMStreamExtractor` for the live daemon. `dmo_decode.go` maps the sliced blocks onto the shared
decoders — `DecodeDMSCHS` (via the BSCH chain), `DecodeDMSCHH` (via SCH/HD), `DecodeDMSCHF`,
and `DMBurstTCHSpeech` / `DMBurstTCHSpeechSoft` (via the TCH/S chain) — de-rotating and
descrambling with the DM colour code (recovered by `RecoverDMColourCode`) before each decode.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA standard and its direct mode.
[^dmo]: [Direct mode operation](https://en.wikipedia.org/wiki/Direct_mode_operation) — Wikipedia, on infrastructure-less direct radio-to-radio operation.
