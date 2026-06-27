# cryptolab — optional RF cryptographic-research toolkit

`cryptolab` is a byte-oriented cryptographic-research toolkit that ships
*inside* the `gophertrunk` binary but is **excluded from the default
install**. It collects the kinds of analysis you reach for when staring at
unfamiliar RF payloads — statistical triage, keyspace brute force, LFSR /
keystream analysis, CRC parameter recovery, analog voice descrambling — plus
a pluggable "subject" framework for studying specific byte-oriented
obfuscators.

These are research tools: they recover obfuscation and weak/keyless
constructions and triage whether a captured payload is even breakable. They
make no claim to break strong keyed encryption; for that, the toolkit offers
known-plaintext keystream extraction and breakability triage only.

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
| `brute` | `xor` | single / repeating-key XOR recovery with English/crib scoring |
| `lfsr` | `bm`, `keystream` | Berlekamp–Massey LFSR recovery; keystream = pt⊕ct |
| `crc` | `recover`, `compute` | recover / compute CRC parameters from sample frames |
| `stats` | `scan` | entropy / IC / chi-square / XOR key-length triage |
| `descramble` | `invert` | full-band analog spectral inversion (self-inverse) |
| `alias` | `gauge`, `structure`, `cells`, `fromseed` | length-seeded byte-obfuscator recovery |

### Examples

```
# Statistical triage of an unknown payload.
gophertrunk cryptolab stats scan -in payload.bin

# Recover a repeating-XOR key with a known crib.
gophertrunk cryptolab brute xor -in cipher.bin -crib "UNIT "

# Identify the CRC on a protocol from captured (data,crc) frames.
gophertrunk cryptolab crc recover -in frames.txt -widths 16,8

# Descramble a frequency-inverted analog voice clip (run twice to undo).
gophertrunk cryptolab descramble invert -in scrambled.s16 -out clear.s16
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
richer multi-round / two-table update forms than the in-binary propagator.
