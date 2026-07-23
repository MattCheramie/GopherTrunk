---
layout: page
title: Vocoders
description: IMBE and AMBE+2 in pure Go — patent status, licensing, and decoder architecture
nav_group: Reference
---

# Vocoders

Digital trunked-radio voice traffic is carried by one of three
vocoders — two DVSI-derived, plus TETRA's own ACELP:

- **IMBE** — used by P25 Phase 1 LDU1/LDU2 voice frames. Core US patents
  (filed early-to-mid-1990s, 20-year term) have **expired**. The
  algorithm is implementable in pure Go without licence concerns;
  GopherTrunk ships a pure-Go decoder at `internal/voice/imbe`.
- **AMBE+2** — used by P25 Phase 2, DMR (Tier II / III), and NXDN. AMBE+2
  is **patent-encumbered**. DVSI sells hardware vocoders (USB-3000 /
  AMBE-3003) and licences software ports. Open-source software
  implementations (e.g. `mbelib`) implement the algorithm; the *code*
  is permissively licensed (mbelib is ISC) but the *patents* are the
  user's risk to evaluate.
- **TETRA ACELP** — the full-rate speech codec used by TETRA voice
  (ETSI EN 300 395-2). This is **not** a DVSI/MBE-family vocoder; it is
  a conventional ACELP codec (LSP-quantised LPC + adaptive/algebraic
  codebooks), specified openly by ETSI. GopherTrunk ships a pure-Go
  decoder at `internal/voice/acelp`. See the TETRA ACELP section below
  for the clean-room / patent posture.

Re-implementing AMBE+2 in pure Go does not change the patent posture —
the algorithm itself is what the patents cover, regardless of
implementation language. Operators in licence-restrictive jurisdictions
should evaluate with counsel before deploying. GopherTrunk's default
build ships the pure-Go AMBE+2 decoder default-on; the legal
responsibility for operating it falls on the deployer, not the
project.

## How GopherTrunk handles this

The `internal/voice` package defines a `Vocoder` interface and a
process-global `Registry`. Each backend registers a factory at `init()`
time. The set of factories present in a binary is determined by the
import set:

```go
type Vocoder interface {
    Name() string
    FrameSize() int
    Decode(frame []byte) ([]int16, error)
    Reset()
    Close() error
}
```

| Backend                  | Build tag    | Default? | Status                                          |
| ------------------------ | ------------ | -------- | ----------------------------------------------- |
| `null` (silence)         | none         | yes      | Always available                                |
| `imbe` (pure-Go, P25 P1) | none         | yes      | Producing intelligible audio; level calibration pending reference data ([voice-calibration.md](voice-calibration.md)) |
| `ambe2` (pure-Go)        | none         | yes      | Producing audio; level calibration pending reference data; DTMF tones synthesise, knox tones via `ambe2.SetKnoxTone` or stay silent |
| `dvsi` (USB-3000 chip)   | `-tags dvsi` | **no**   | Wire-protocol + Vocoder scaffolding shipping; USB transport stub (returns `ErrNoDevice`) — hardware integration follows in a separate PR |

### Live-pipeline auto-decode

When CallStart fires, the recorder maps `Grant.Protocol` to a
vocoder name and instantiates a fresh vocoder per call. Each
`WriteRawFrame` call decodes its frame and appends the resulting
PCM to the call's WAV, alongside the optional `.raw` sidecar.

Default mapping (see `voice.DefaultVocoderForProtocol`):

| Grant.Protocol | Vocoder | Notes                            |
| -------------- | ------- | -------------------------------- |
| `p25`          | `imbe`  | P25 Phase 1 LDU1 / LDU2          |
| `p25-phase2`   | `ambe2` | P25 Phase 2                      |
| `dmr-tier2`    | `ambe2` | DMR Tier II conventional         |
| `dmr-tier3`    | `ambe2` | DMR Tier III trunked             |
| `nxdn`         | `ambe2` |                                  |
| `dpmr`         | `ambe2` | dPMR Mode 3                      |
| `tetra`        | `ambe2` |                                  |

Analog protocols (`motorola`, `edacs`, `ltr`, `mpt1327`, etc.)
have no entry — for those, the composer's FM chain feeds
`WritePCM` directly. EDACS ProVoice (`Grant.ProVoice == true`)
has no in-binary decoder either; it always gets a `.raw` sidecar
regardless of the global `WriteRaw` flag, so researchers can
decode out-of-band.

Operators override the mapping via
`RecorderOptions.VocoderForProtocol`:

```go
voice.NewRecorder(voice.RecorderOptions{
    // …other fields…
    // Replace the IMBE mapping with the silence vocoder for
    // testing, leave AMBE+2 alone:
    VocoderForProtocol: map[string]string{
        "p25":        "null",
        "p25-phase2": "ambe2",
        // …other defaults…
    },
})
```

Pass an explicit empty (non-nil) map to disable auto-decode
entirely — the `.raw` sidecar then becomes the only audio
output for digital calls.

### Raw sidecar (escape hatch)

The recorder emits a raw-frame sidecar (`.raw` next to the WAV)
when `WriteRaw` is enabled or for ProVoice grants, so users can
run their own decoder on the captured frames without trusting
the in-binary vocoders. This is the escape hatch for operators
who want bit-exact mbelib / DSD-FME / OP25 output or who prefer
to defer the decoding choice to post-processing.

### Decoding a captured .raw sidecar

The daemon ships a `decode` subcommand that runs the registered
in-binary vocoders against a `.raw` frame stream out-of-band:

```sh
gophertrunk decode -in call.raw -out call.wav -vocoder imbe
gophertrunk decode -in dmr.raw  -out dmr.wav  -vocoder ambe2
gophertrunk decode -list-vocoders   # enumerate registered names
```

Stdin / stdout work for the input via `-in -`, so capture pipelines
can stream into the decoder without a temporary file:

```sh
some-source | gophertrunk decode -vocoder imbe -out out.wav
```

The library function backing this — `voice.DecodeStream(in,
vocoderName, out)` — is exported from `internal/voice` so other
consumers (web UIs, batch processors, post-mortem analysis tools)
can reuse the same decode path without spawning a binary. See
`internal/voice/streamdecode.go`.

## Implementation notes

- `internal/voice/mbe/` is the shared MBE-family synthesis core:
  cross-frame log-amplitude prediction, voiced harmonic generator,
  unvoiced FFT excitation + §6.4 overlap-add window, §6.2 spectral
  enhancement, and a per-frame fast-attack / slow-release AGC.
  Consumed by both `imbe` and `ambe2`.
- `internal/voice/imbe/` holds the IMBE-specific front half: 88-bit
  unpack, Golay/Hamming FEC inverse, scrambler, PRBA/HOC + inverse
  DCTs producing the spectral residuals the shared core consumes.
- `internal/voice/ambe2/` holds the AMBE+2-specific front half:
  49-bit unpack, codebook lookups (auto-generated from
  szechyjs/mbelib's `ambe3600x2400_const.h` under ISC,
  regenerable via `scripts/gen-ambe2-tables.sh`), inverse DCTs
  producing the spectral residuals, and the cross-frame `gamma`
  bookkeeping AMBE+2 requires.

Both decoders share the same constructor surface
(`New()` / `NewWithSeed(seed)` / `NewWithConfig(seed, mbe.AGCConfig)`)
so operators can pin reproducibility for tests or tune the AGC for
their downstream chain.

## Why a plugin model

This is exactly what SDR# / OP25 / DSD do. The key benefits:

1. The default binary has no external library dependencies for voice
   (no CGO, no system shared library, no install scripts).
2. Users with DVSI hardware can opt in by building with `-tags dvsi`.
   The Vocoder + AMBE-3003 wire protocol + voice.Vocoder interface
   conformance ship in [`internal/voice/dvsi/`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/dvsi/);
   the USB / FTDI transport that talks to the physical chip is a stub
   today (returns `ErrNoDevice`) so the recorder fallback chain
   activates cleanly. Hardware integration with a real DVSI USB-3000
   lands in a follow-up. CI exercises the wire protocol + Vocoder
   plumbing via the scripted mock Transport and the
   software-loopback Transport (`Options{LoopbackOnly: true}`); both
   live behind the `-tags dvsi` build tag.
3. Captures contain raw frames so a researcher can defer the decoding
   choice to post-processing.

## DVSI backend layout (`-tags dvsi`)

[`internal/voice/dvsi/`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/dvsi/):

- `packet.go` — AMBE-3003 wire format (sync byte + length + type +
  payload). Always compiled — no patent surface in describing a
  serial wire protocol.
- `doc.go` — exports `VocoderName = "dvsi"` so config validation
  paths can reference the key without `-tags dvsi` linked in.
- `dvsi_enabled.go` (`//go:build dvsi`) — `Vocoder`, `Transport`
  interface, `loopbackTransport`, `openUSBTransport` stub, and the
  `init()` registration into `voice.DefaultRegistry`.
- `dvsi_disabled.go` (`//go:build !dvsi`) — empty; default builds
  link nothing from the DVSI codepath.
- `dvsi_test.go` (`//go:build dvsi`) — Vocoder interface
  conformance, loopback round-trip, scripted-mock wire-format
  verification, frame-size validation, unexpected-reply rejection.

`make test-dvsi` runs the tagged unit tests; the `dvsi` CI job runs
the same target on Ubuntu.

## Voice calibration plumbing

The calibration harness ships end-to-end:

- [`internal/voice/calibrate`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/calibrate/) — RMS-
  ratio + best-alignment cross-correlation comparison against a
  reference WAV from DSD-FME / OP25.
- [`internal/voice/imbe/testdata/`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/imbe/testdata/)
  and
  [`internal/voice/ambe2/testdata/`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/ambe2/testdata/)
  — fixture drop zones with READMEs documenting the file layout
  the calibrate tests expect.
- [`docs/voice-calibration.md`](voice-calibration.md) — operator-
  facing capture-and-validate recipe.
- [`cmd/voice-calibrate`](https://github.com/MattCheramie/GopherTrunk/tree/main/cmd/voice-calibrate/) — CLI wrapper
  around `calibrate.Compare` so a one-off check doesn't require
  writing a test.

## Knox / call-alert extension hook

AMBE+2 tone frames with b1 ∈ [144, 163] are vendor-specific knox /
call-alert pairs. The public spec doesn't document them; without
registration, the decoder routes those frames through silence.

### Single-entry registration

Operators with a per-vendor reference can register one
(freqA, freqB) pair at a time via
[`ambe2.SetKnoxTone`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/ambe2/knox.go) (typically
from a per-vendor sub-package `init()`):

```go
import "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"

func init() {
    // Hypothetical Motorola Trbo call alert (frequencies illustrative).
    _ = ambe2.SetKnoxTone(150, 1100, 1750)
}
```

### Preset bundles

For curated tables (a service-manual extract, an open-source
receiver's vendor table), use
[`ambe2.RegisterPreset`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/ambe2/knox.go) instead
of repeated `SetKnoxTone` calls. The preset name shows up in
`ambe2.ListPresets()` for operator diagnostics:

```go
import "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"

func init() {
    // Hypothetical Vendor X call-alert table. Replace with values
    // sourced from a verifiable reference (DSDcc, DSD-FME, vendor
    // service manual) — the public AMBE+2 spec does not document
    // these frequencies.
    _ = ambe2.RegisterPreset(ambe2.KnoxPreset{
        Name: "vendor-x-call-alerts",
        Entries: map[int][2]float64{
            150: {1100, 1750},
            151: {1200, 1800},
        },
    })
}
```

Registered indices synthesise through the same summed-sinewave
dual-tone path as DTMF — phase-continuous across consecutive tone
frames, AGC-scaled, click-free.

### Sourcing vendor frequencies

The in-tree code ships no vendor presets because the AMBE+2 public
spec does not document the [144, 163] frequency range and the
maintainers have not had operator-confirmed reference data to copy
from. Contributors with a verifiable source (cite the file + commit
or document section in the PR) can land a per-vendor preset under
`internal/voice/ambe2/presets/<vendor>/`. Open-source receivers
worth checking: `szechyjs/mbelib`, DSDcc, DSD-FME, OP25 — search
the tone-decode paths for the b1 index range.

## TETRA ACELP (`internal/voice/acelp`)

TETRA full-rate voice uses an ACELP speech codec specified openly in
**ETSI EN 300 395-2** (13.167 kbit/s, 137-bit frames / 30 ms → 240
samples at 8 kHz). GopherTrunk ships a **pure-Go, clean-room** decoder.

**Chain.** A granted call's traffic bursts are recovered by the
composer's TETRA voice chain, TCH/S channel-decoded (EN 300 395-2 §5.5:
RCPC + interleave + class-2 CRC) into 137-bit speech frames, and rendered
to PCM by the ACELP decoder registered as the `tetra-acelp` vocoder. The
decoder is the usual ACELP pipeline: parameter unpack → LSP split-VQ
dequantisation → per-subframe adaptive (pitch) + algebraic (4-pulse)
codebooks → log-energy gain dequantisation → LSP→LPC conversion →
synthesis filter. Bit-exact fixed-point arithmetic (ITU-T-style 16/32-bit
saturating operators) underlies every block.

**Slot isolation.** The traffic extractor emits a burst for every TDMA
timeslot on the carrier. The TCH/S class-2 CRC is used as the slot filter:
only the granted call's TCH/S speech bursts verify, so other slots'
signalling bursts (and encrypted TEA1-4 traffic) are dropped. This cleanly
separates a *single* active call; two concurrent TCH/S calls on the same
carrier would still need TDMA frame-number demux (a follow-up).

**Provenance / licensing.** The decoder was ported from the *algorithmic*
description in EN 300 395-2 and the ITU-T fixed-point operator
definitions; the fixed quantiser tables (LSP codebooks, energy codebook,
interpolation filters) are numeric constants sourced from the ETSI
reference codec (via `github.com/curlyboi/libtetradec`). ACELP's core
algebraic-CELP patents (Sherbrooke/VoiceAge, filed late-1980s to
mid-1990s, 20-year term) have **expired**, so ACELP is far less
patent-encumbered than the DVSI/MBE family. As with IMBE/AMBE+2, the legal
responsibility for operating the codec falls on the deployer, not the
project; operators in licence-restrictive jurisdictions should evaluate
with counsel. The decoder lives behind an explicit vocoder registration
(`tetra-acelp`) so builds can omit it.

**Validation.** Each block carries unit tests validated *independently*
of the reference where possible (e.g. the LSP→LPC conversion is checked
against the LSP root property; log2/pow2 against `math`; the synthesis
filter against a float recursion). Bit-exact conformance against the ETSI
clause-4 test sequences is the remaining validation step (see Future work).

## Future work

- Absolute-level calibration thresholds documented; reference data
  (operator-supplied DSD-FME / OP25 decoded WAVs) is the remaining
  blocker. AGC per-frame gain tweaks if real frames show systematic
  level offset.
- Per-vendor knox tone tables (Motorola Trbo, Hytera, generic
  AMBE+2) — the extension hook ships; vendor reference data is the
  remaining piece.
- DVSI USB-3000 / AMBE-3003 USB / FTDI transport implementation —
  the wire-protocol + Vocoder + interface conformance ship now;
  the actual USB bulk-IN / bulk-OUT plumbing follows when a chip is
  available for round-trip testing.
- Optional Opus / FLAC re-encoding of the recorded WAVs to shrink
  long-running archives.
- Plain AMBE decoder for D-STAR voice (different algorithm from AMBE+2;
  same DVSI patent family).
- TETRA ACELP: bit-exact conformance against the ETSI EN 300 395-2
  clause-4 test sequences (add under `internal/voice/acelp/testdata/`),
  and TDMA frame-number slot demux so concurrent same-carrier TCH/S
  calls separate cleanly (today a single active call is isolated by the
  TCH/S CRC).
