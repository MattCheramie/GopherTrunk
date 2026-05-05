# Vocoders

Digital trunked-radio voice traffic is carried by one of two
DVSI-derived vocoders:

- **IMBE** — used by P25 Phase 1 LDU1/LDU2 voice frames. Core US patents
  (filed early-to-mid-1990s, 20-year term) have **expired**. The
  algorithm is implementable in pure Go without licence concerns; this
  is the path GopherTrunk plans to take in `internal/voice/imbe`.
- **AMBE+2** — used by P25 Phase 2, DMR (Tier II / III), and NXDN. AMBE+2
  is **patent-encumbered**. DVSI sells hardware vocoders (USB-3000 /
  AMBE-3003) and licences software ports. Open-source software
  implementations (e.g. `mbelib`) implement the algorithm; the *code*
  is permissively licensed (mbelib is ISC) but the *patents* are the
  user's risk to evaluate.

GopherTrunk does not ship an AMBE+2 implementation in default builds.

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

| Backend                  | Build tag       | Default? | Status                            |
| ------------------------ | --------------- | -------- | --------------------------------- |
| `null` (silence)         | none            | yes      | Always available                  |
| `imbe` (pure-Go, P25 P1) | none            | yes      | Stubbed; full decoder in progress |
| `mbelib` (AMBE+2 / IMBE) | `-tags mbelib`  | **no**   | CGO wrapper, off by default       |
| `dvsi` (USB-3000 chip)   | `-tags dvsi`    | **no**   | Hardware backend, planned         |

The recorder always emits a raw-frame sidecar (`.raw` next to the WAV)
when configured, so users can run their own decoder on the captured
frames without any vocoder linked into the daemon.

## Building with `mbelib`

`internal/voice/mbelib` ships a CGO wrapper around the szechyjs
[`mbelib`](https://github.com/szechyjs/mbelib) library, gated behind
the `mbelib` build tag. With `libmbe.so` and `mbelib.h` installed
on the host, opt in by running:

```sh
make build TAGS=mbelib
```

This registers two factories on `voice.DefaultRegistry`:

- `imbe`  — IMBE 4400 bps for P25 Phase 1 LDU1 / LDU2 voice frames
  (88-bit input, 11-byte packed). Targets `mbe_processImbe4400Dataf`.
- `ambe2` — AMBE+2 2400 bps for P25 Phase 2, DMR, and NXDN voice
  frames (49-bit input, 7-byte packed with 7 padding bits).
  Targets `mbe_processAmbe2400Dataf`.

Both produce 8 kHz / 20 ms / 160 samples of int16 PCM per call,
matching the recorder's PCM contract.

The default `make build` target compiles the stub at
`internal/voice/mbelib/cgo_stub.go` instead, registers nothing, and
links no extra libraries. CI exercises the stub path only — the
wrapper is verified at build time when an operator opts in.

If `make build TAGS=mbelib` fails with `mbelib.h: No such file or
directory`, install the library:

```sh
git clone https://github.com/szechyjs/mbelib && cd mbelib
mkdir build && cd build && cmake .. && make && sudo make install
sudo ldconfig
```

Then re-run the build.

## Why a plugin model

This is exactly what SDR# / OP25 / DSD do. The key benefits:

1. The default GopherTrunk binary has zero patent exposure and no
   external library dependencies for voice.
2. Users in jurisdictions where they hold (or don't need) AMBE+2
   licences can opt in by building with `-tags mbelib` or wiring a
   hardware DVSI dongle.
3. Captures contain raw frames so a researcher can defer the decoding
   choice to post-processing.

## Future work

- Pure-Go IMBE decoder for P25 Phase 1 (TIA-102.BABA reference).
- mbelib CGO wrapper (build-tag gated).
- DVSI USB-3000 / AMBE-3003 hardware backend.
- Optional Opus / FLAC re-encoding of the recorded WAVs to shrink
  long-running archives.
