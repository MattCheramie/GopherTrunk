# cryptolab — optional RF cryptographic-research toolkit

`cryptolab` is a byte-oriented cryptographic-research toolkit that ships
*inside* the `gophertrunk` binary but is **excluded from the default
install**. It collects the kinds of analysis you reach for when staring at
unfamiliar RF payloads — statistical triage, autocorrelation period
detection, a NIST SP 800-22 randomness battery, an obfuscation-class
classifier, keyspace brute force, LFSR / keystream analysis, keystream-reuse /
many-time-pad recovery, CRC parameter recovery, analog voice descrambling —
plus a pluggable "subject" framework for studying specific byte-oriented
obfuscators.

These are research tools: they recover obfuscation and weak/keyless
constructions and triage whether a captured payload is even breakable. They
make no claim to break strong keyed encryption; for that, the toolkit offers
known-plaintext keystream extraction and breakability triage only. The
keystream-reuse tool exploits operator misuse (a repeated IV/MI) and known
plaintext — never the cipher key itself.

## Opting in at build time

The toolkit is gated behind the `cryptolab` build tag, the same mechanism the
DVSI vocoder uses. The standard build does **not** link it in:

```
make build                 # default: `gophertrunk cryptolab` prints how to opt in
make build TAGS=cryptolab  # opt in: the full toolkit is linked
go build -tags cryptolab ./cmd/gophertrunk   # equivalent
make test-cryptolab        # run the toolkit's tests (incl. the tagged CLI)
```

The toolkit's engine and subject packages under `internal/cryptolab/` carry
no build tag, so they always compile and are covered by `make test`; only the
binary's `cryptolab` subcommand is gated, which is what keeps the default
operator install lean.

## Web console

`cryptolab` ships a browser console that mirrors the `siglab`/`configbuilder`
consoles (same stack, design tokens, and layout) and opens in its own window
like `siglab serve`:

```
make cryptolab-web-build           # bundle the SPA into web/cryptolab/dist/
make build TAGS=cryptolab          # link the toolkit + console into gophertrunk
gophertrunk cryptolab serve -open  # serve at http://127.0.0.1:8096/ and open a browser
```

The console exposes **every** tool, mode, and setting: it renders a form from
the backend's `GET /api/v1/cryptolab/tools` schema, so each parameter (file
upload, text, number, checkbox) gets a control, and new tools appear
automatically. A run uploads inputs, streams the live log, and shows the
structured result — summary, fields, ranked findings, notes, and downloadable
artifacts (survivor logs, checkpoints, descrambled output). It runs entirely
offline against uploaded files; no SDR or daemon required.

When the main `gophertrunk` daemon is built with `-tags cryptolab`, the same
console is also mounted inside it at `/cryptolab/` (its API lives under
`/api/v1/cryptolab/`, alongside the siglab routes), so you can reach it from
the running daemon without launching a separate `cryptolab serve`. Mutating
routes share the daemon's mutation gate. The default daemon build links a
no-op mount, so the toolkit stays out of the standard binary.

## Usage (CLI)

```
gophertrunk cryptolab [global flags] <tool> [<mode>] [tool flags]
gophertrunk cryptolab serve [-addr host:port] [-open]   # web console
gophertrunk cryptolab list      # list tools and modes
```

Global flags precede the tool name (`-out`, `-resume`, `-format`,
`-log-level`, `-log-format`). Output renders as text/json/jsonl/yaml/csv.

### Tools

| Tool | Modes | What it does |
|------|-------|--------------|
| `classify` | `auto` | triage an unknown payload and recommend the next tool |
| `stats` | `scan`, `period` | entropy / IC / chi-square / XOR key-length triage; autocorrelation period + repeated-n-gram detection |
| `randomness` | `battery`, `quick` | NIST SP 800-22 randomness tests on a keystream / payload bitstream |
| `brute` | `xor`, `caesar`, `vigenere`, `substitution` | classical-cipher recovery with English/crib scoring |
| `lfsr` | `bm`, `keystream` | Berlekamp–Massey LFSR recovery; keystream = pt⊕ct |
| `ks` | `reuse`, `mtp`, `extract` | keystream-reuse detection, many-time-pad recovery, keystream extraction |
| `crc` | `recover`, `compute` | recover / compute CRC parameters from sample frames |
| `descramble` | `invert`, `splitband`, `rolling` | analog spectral / split-band / rolling-code voice inversion |
| `alias` | `gauge`, `structure`, `cells`, `fromseed` | length-seeded byte-obfuscator recovery |

### Examples

```
# Triage an unknown payload and get a recommended next command.
gophertrunk cryptolab classify auto -in unknown.bin

# Statistical triage of an unknown payload.
gophertrunk cryptolab stats scan -in payload.bin

# Find the period of a repeating-key cipher or periodic scrambler.
gophertrunk cryptolab stats period -in payload.bin -max-lag 256

# Is a recovered keystream strong (random) or an exploitable generator?
gophertrunk cryptolab randomness battery -in keystream.bin

# Find frames that reuse an IV/MI (P25 OFB/ADP keystream reuse) ...
gophertrunk cryptolab ks reuse -in frames.jsonl
# ... then recover a whole reuse group from one known plaintext.
gophertrunk cryptolab ks mtp -in frames.jsonl -known-label call-12 -known-pt known.bin
# ... or crib-drag the group with no known plaintext.
gophertrunk cryptolab ks mtp -in frames.jsonl -crib " the "

# Recover a repeating-XOR key with a known crib.
gophertrunk cryptolab brute xor -in cipher.bin -crib "UNIT "

# Identify the CRC on a protocol from captured (data,crc) frames.
gophertrunk cryptolab crc recover -in frames.txt -widths 16,8

# Recover a monoalphabetic substitution cipher (English plaintext assumed).
gophertrunk cryptolab brute substitution -in cipher.txt -restarts 40

# Descramble a frequency-inverted analog voice clip (run twice to undo).
gophertrunk cryptolab descramble invert -in scrambled.s16 -out clear.s16

# Undo a split-band inversion (low/high sub-bands inverted about a split point).
gophertrunk cryptolab descramble splitband -in scrambled.s16 -out clear.s16 -split 0.5

# Undo a rolling/hopping inversion, auto-detecting the per-frame split.
gophertrunk cryptolab descramble rolling -in scrambled.s16 -out clear.s16 -frame 1024 -schedule auto
```

## Subject framework: byte-obfuscator recovery (`alias`)

The toolkit's subject framework studies length-seeded, keyless, byte-oriented
obfuscators where the output substitution table and per-character decode
`char = int8(Modd·(LUT[eo] − Hodd))` are established but the per-character
state update is not. The `alias` tool provides four incremental, logging,
resumable recovery modes over a user-supplied ground-truth corpus
(`rid,talkgroup,encoded_hex,alias`; a trailing 2-byte CRC is stripped
automatically):

- **`gauge`** — brute-force all 32,768 affine gauges, looking for a
  coordinate frame in which the odd high byte becomes a clean function of a
  simple merged index.
- **`structure`** — enumerate merged-index table wirings over the dense,
  plaintext-free high-byte recurrence `H[k+1] = F(H[k-1], H[k], eo[k])` and
  report each wiring's conflict floor.
- **`cells`** — intersect per-context state candidates across the corpus;
  **monotone and resumable** (`-resume checkpoint.json`), so coverage only
  grows as you feed it more captures — even ciphertext-only ones, which still
  feed the high-byte recurrence.
- **`fromseed`** — simulate candidate accumulator updates from the length
  seed, auto-solve the affine output gauge, and cull by high-byte mismatch.

### What the data supports

The two state bytes form 256×256 = 65,536-cell functions; a passive corpus
typically exercises only a few hundred cells of each. The **high byte is
readable from ciphertext alone**, so the `structure`/`fromseed` modes can keep
chipping at it from passive captures with no transmitter. The **multiplier**
byte is only touched by labeled rows, so the `cells` mode improves
monotonically as more labeled data arrives. Each mode's report ends with what
additional data would close the remaining gap — the effort is always directed,
logged, and resumable.

The optional Z3 structural search under `internal/cryptolab/smt/` explores
richer multi-round / two-table update forms than the in-binary propagator. The
`alias structure` mode writes `high-transitions.csv` to the `-out` directory,
which the Z3 script consumes directly.

## Substitution and voice-descramble internals

- **Monoalphabetic substitution** (`brute substitution`) auto-solves a general
  substitution cipher by frequency-seeded hill-climbing with random restarts,
  scored against an embedded English trigram language model
  (`internal/cryptolab/engine/lang`, `…/engine/subst`). It assumes English
  plaintext; `-restarts` trades runtime for recovery on short ciphertexts.
- **Split-band inversion** (`descramble splitband`) inverts the low and high
  sub-bands independently about a `-split` point (fraction of Nyquist) using a
  disjoint-bin FFT so the operation stays self-inverse.
- **Rolling-code inversion** (`descramble rolling`) applies a per-frame split
  schedule; `-schedule auto` detects each frame's inversion from its
  spectral energy balance, while a CSV schedule replays a known hop sequence.

## Randomness, classification, and keystream-reuse

These three tools turn the "what am I looking at, and can I break it?" workflow
into explicit, RF-framed steps:

- **`classify auto`** runs the cheap measurements the other tools depend on
  (entropy, index of coincidence, repeating-XOR key length, autocorrelation
  period, and the randomness battery) and emits a ranked verdict —
  `plaintext`, `substitution-or-shift`, `repeating-xor`, `periodic-scrambler`,
  `lfsr-or-keyless-scrambler`, or `strong-encrypted` — each with the exact next
  command to run. It is the front door for an unknown byte payload.
- **`randomness battery`** runs a NIST SP 800-22 subset (monobit, block
  frequency, runs, longest run, binary-matrix rank, spectral DFT, approximate
  entropy, serial, cumulative sums, and **linear complexity**) on a bitstream.
  The decisive RF question is whether a recovered keystream is statistically
  random (strong keyed encryption — nothing to exploit) or carries structure (a
  keyless scrambler or short LFSR). A failing linear-complexity or spectral
  test is the signature of an LFSR-based scrambler. `randomness quick` runs the
  fast, low-data subset for short captures. Input is packed bytes, MSB-first
  (the same convention as `lfsr bm`).
- **`ks`** exploits **keystream reuse**. Additive stream ciphers (P25 DES-OFB
  and ADP/RC4, TETRA AIE) produce an identical keystream whenever the same key
  and IV recur, so two frames sharing an IV satisfy `c1 ⊕ c2 = p1 ⊕ p2` — a
  running-key system recoverable with crib-dragging or one known plaintext, **no
  key recovery required**. For P25 the IV is the 72-bit Message Indicator the
  decoders already extract. `ks reuse` reports IV/MI collisions; `ks mtp`
  recovers content from a reuse group (from a known plaintext via
  `-known-label`/`-known-pt`, or by crib-drag via `-crib`); `ks extract`
  computes a keystream from a known plaintext/ciphertext pair so you can feed it
  straight to `randomness battery` or `lfsr bm`.

### Frames file format (`ks reuse` / `ks mtp`)

Each non-empty line is either a JSON object or a CSV triple. The JSON form is
exactly what the live decoder→cryptolab bridge emits (see below), one record
per encrypted frame:

```
{"label":"call-12","iv":"a1b2c3d4e5f60708090a","ct":"deadbeef…"}
call-12,a1b2c3d4e5f60708090a,deadbeef…
```

`label` identifies the source (call id / timestamp), `iv` is the hex IV / P25
Message Indicator, and `ct` is the hex ciphertext. Any extra JSON fields
(`system`, `protocol`, `tg`, `algid`, `keyid`, `at`) are preserved for
provenance and ignored by the parser. Frames with an empty IV are ignored.
Lines beginning with `#` are comments.

### Live capture bridge (decoder → cryptolab)

The daemon can feed `ks` straight from live decode. Set
`recordings.crypto_capture_path` in config.yaml:

```yaml
recordings:
  crypto_capture_path: /var/lib/gophertrunk/crypto-frames.jsonl
```

When set, the P25 Phase 1 voice composer appends one JSON record per encrypted
LDU2 superframe — `{label, iv (Message Indicator), ct (encrypted voice
frames), system, protocol, tg, algid, keyid, at}` — to that file. Point
`cryptolab ks reuse` / `ks mtp` at it to hunt for MI reuse across the capture.
Empty (the default) disables the bridge entirely: no extraction work runs on
the voice path, so the standard operator build is unaffected.

This records encrypted material and its IV; it performs **no decryption**. A
reused MI exposes `p1 ⊕ p2` for offline analysis — for P25 voice that is
IMBE-coded speech (recover one frame's plaintext to peel the rest), for
encrypted P25 data it is recoverable text. A non-reused capture confirms the
system is using its IV correctly.

## Scope note

The toolkit does not claim to break strong keyed RF crypto (P25 DES-OFB/AES,
ADP/RC4). For those it offers known-plaintext keystream extraction (`lfsr`,
`ks extract`), keystream-reuse / many-time-pad recovery when a transmitter
repeats an IV/MI (`ks`), weak/short-key brute force (`brute`), and breakability
triage (`stats`, `randomness`, `classify`); each report ends with what
additional data would close the remaining gap.
