---
title: "From the Issue Tracker, Season 2, Part 9: Sixteen Thousand Codewords per Slot — The AACH Decoder Behind 'Overruns at 200 kS/s'"
description: "Why a dual-TETRA wideband rig overran its SoapyRemote stream at a 'tiny' 200 kS/s with no decode-can't-keep-up warning — the RM(30,14) AACH decoder re-encoded all 16 384 codewords, allocating, on every downlink slot — and how a once-built codebook, popcount and a branch-free FIR window took the pump from 0.44× to 0.12× real time, bit-identical."
category: solution-postmortem
keywords: soapyremote host overruns 200 ks/s, tetra aach decoder cpu, rm(30,14) reed muller tetra, DecodeRM3014Tetra codebook, maximum likelihood block decoder popcount, fir ring buffer branch, mirrored history window fir, wideband pump profiling go, TestEngineDualTETRA200kThroughput, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, tetra, performance, dsp, wideband, go, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 9
---

*Part 9 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems that continues the
[first season]({{ '/blog/series/from-the-issue-tracker/' | relative_url }}):
one bug per part, with receipts.
[Part 8]({{ '/blog/solution-postmortem/issue-tracker-s2-08-sample-rate-880029/' | relative_url }})
closed two capture-tooling stories from the same September report. This
part returns to the decode pump: a report of SoapyRemote host overruns on
a wideband rig running at 200 kS/s — a rate a laptop should not notice —
where the profile put two thirds of the CPU inside a 30-bit block
decoder.*

> **TL;DR:** On 10 Sep the dual-TETRA wideband rig (two 144 kHz DDC
> channels off a 200 kS/s X310 stream) reported `soapyremote: SDR overruns
> … host_drops` "even at a tiny bandwidth", with no `decode can't keep up`
> WARN — that line does not exist on the wideband path.
> `TestEngineDualTETRA200kThroughput` measured the pump at **0.44× real
> time** on one 2.1 GHz core with 65 % inside `DecodeAACH`:
> `DecodeRM3014Tetra` / `DecodeRM3014TetraSoft` were maximum-likelihood
> searches that called the allocating `EncodeRM3014Tetra` for every one of
> the 16 384 codewords on every call — ~16 k allocations per decode, once
> per downlink slot (~70/s per carrier) plus once per traffic burst in the
> voice demux, and the ~19 GC/s on a 9 MB heap in the operator's heartbeat
> lines. Fix: a once-built `rm3014Table` searched by `bits.OnesCount32`
> (hard) or three 10-bit partial-sum tables (soft) — 17 µs / 36 µs,
> allocation-free, bit-identical to the brute force the codebook test keeps
> as reference. Next in the profile was `filter.FIR` (~50 %): its ring
> buffer branched per tap; a mirrored 2N history in the same float32
> summation order (`TestFIRMirroredWindowIsBitExact`) is ~1.8× faster.
> Pump: 0.44× → **0.12×**.

**Key takeaways**

- **`host_drops` is a downstream signal.** The driver sheds when its
  consumer stops draining; "overruns at a low rate" means profile the
  consumer, even when no decode-rate WARN fires.
- **An ML search over 2^14 codewords is fine; re-deriving the codebook
  inside it is not.** The codewords never change; building them once turns
  an allocating encode into an XOR and a popcount.
- **Bit-identical is a testable property.** Both rewrites keep the
  original loop as the reference and compare every output exactly, so no
  golden anywhere in the tree moved.
- **The second hotspot only appears once the first is gone.** The FIR was
  invisible behind the codebook; a profile after a fix is a new profile.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The codebook | all 2^14 RM(30,14) codewords, uint32-packed, built once under `sync.Once` | `internal/radio/framing/rm_30_14_tetra.go` (`rm3014Table`, `rm3014RowMask`) |
| Hard decoder | minimum Hamming distance via `bits.OnesCount32(cw ^ r)`, early exit at 0 | `DecodeRM3014Tetra` |
| Soft decoder | ML correlation from three 10-bit partial-sum tables; `margin` = (best − second) / 2·Σ‖llr‖ | `DecodeRM3014TetraSoft` |
| Reference + pins | brute force kept; 400 random words at every error weight; ≤ 1 alloc/run | `rm_30_14_tetra_codebook_test.go` (`TestRM3014CodebookMatchesBruteForce`, `TestRM3014DecodeAllocations`) |
| Who calls it | per downlink slot (`aachClassifyMaxErrs` gate), per traffic burst in the demux, `dispatchSlice` | `tetra/downlink.go`, `tetra/traffic.go`, `tetra/process.go` (`DecodeAACH`) |
| The FIR window | 2N mirrored history, reversed taps, newest-first accumulation | `internal/dsp/filter/fir.go` (`FIR.Process`), `fir_window_test.go` |
| The instrument | seconds of 200 kS/s noise through two TETRA channels, ratio to real time, optional CPU profile | `widebandt2/engine_bench_tetra_test.go` (`GT_WB_BENCH_SECONDS`, `GT_WB_BENCH_PROFILE`) |

## In this post

- **The report, and the WARN that was not there** — reading overruns on the wideband path.
- **Seventy slots a second, sixteen thousand encodes each** — what `DecodeAACH` was doing.
- **A codebook built once** — XOR, popcount, and three partial-sum tables.
- **The filter behind the decoder** — a ring buffer that branched per tap.
- **Proving bit-identical** — the brute force as reference, the ring as reference.
- **What the pump measures now** — 0.44× to 0.12×, and what that leaves open.

## The report, and the WARN that was not there

The rig was the two-system wideband configuration from the 10 Sep "can't
monitor two TETRA systems at once" report: `role: wideband` on an X310 at
200 kS/s, one `channels:` entry per TETRA control channel, two 144 kHz DDC
taps and two full TETRA decoders on one pump goroutine. The operator's log carried `soapyremote: SDR overruns …
host_drops`, and the natural reading of that line at 200 kS/s is a network
or driver fault — a rate that small cannot be a CPU problem.

It can. `sendOrDrop` in `internal/sdr/soapyremote/driver.go` only sheds
when the consumer stops draining a ~400 ms channel; the driver has not
changed since the repo import, and its last defect was an opcode, not a
counter
([Season 1 Part 18]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})).
`host_drops` is the consumer falling behind
([Field Notebook Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
reads the line). The single-channel `ccdecoder` path has a companion WARN
— `decode can't keep up with real time` — that makes CPU obvious there;
the wideband engine has none, so its overruns arrive without a hint. The
standing rule from this report: an "overruns at low rate" report on a path
with no keep-up WARN still means profile the pump.

The profile instrument is `TestEngineDualTETRA200kThroughput` in
`internal/scanner/widebandt2/`: skipped unless `GT_WB_BENCH_SECONDS` is
set, it pumps that many seconds of 200 kS/s noise in 998-sample chunks
through a real engine with two TETRA channels at 467.9125 and 467.875 MHz
and logs processing time over stream time; `GT_WB_BENCH_PROFILE` writes a
CPU profile. The first run read **0.44× real time** on a 2.1 GHz Xeon
core — two channels consuming nearly half a core on noise alone — and the
profile put **65 %** of it in one function.

## Seventy slots a second, sixteen thousand encodes each

The TETRA AACH (Access Assignment Channel) carries a per-slot usage
marker as 14 information bits coded into 30 channel bits by the shortened
(30,14) Reed-Muller code of EN 300 392-2 §8.2.3.2 — systematic, generator
`[I₁₄ | P]` with the 14×16 parity matrix of equation (8.13), no
convolutional coding and no interleaving, so the AACH's type-4 block *is*
its type-2 block. GopherTrunk decodes it by maximum-likelihood search: try
every one of the 2^14 = 16 384 codewords, keep the nearest. That is a
reasonable decoder for a 30-bit code. What was not reasonable was how each
candidate was produced:

```go
// the old search (shape, kept as bruteForceRM3014 in the codebook test)
for v := 0; v < (1 << 14); v++ {
    var info [14]byte
    for i := 0; i < 14; i++ { info[i] = byte((v >> uint(13-i)) & 1) }
    cw := EncodeRM3014Tetra(info[:])     // allocates a 30-byte slice, 14×16 parity loop
    dist := 0
    for i := 0; i < 30; i++ { if cw[i] != received[i]&1 { dist++ } }
    …
}
```

Sixteen thousand encodes, each allocating, for every decode — several
hundred microseconds and ~16 k allocations per call. And the call sites
are hot. `decodeDownlinkSlot` (`tetra/downlink.go`) decodes the AACH on
every downlink slot to classify it, trusting the result only at
`errs ≤ aachClassifyMaxErrs` = 2; at four slots per 56.67 ms frame that is
~70 decodes per second per carrier. The voice demux decodes it again per
traffic burst (`tetra/traffic.go`, with the extractor's colour code) to
read the usage marker that routes each burst to its call, and
`dispatchSlice` (`tetra/process.go`) handles `ChannelAACH` on the generic
path. Two carriers, each with calls up, is hundreds of decodes a second
and millions of short-lived allocations — the ~19 garbage collections per
second on a 9 MB heap in the operator's heartbeat lines were this.

## A codebook built once

The codewords never change, so `rm3014Table` builds them once under
`sync.Once`. Each parity row of the matrix is packed into a 16-bit mask
(`rm3014RowMask`), a codeword's parity is the XOR of the row masks of its
set information bits, and the codeword is stored as `v<<16 | parity` —
30 bits in a `uint32`, indexed by `v`, which is the systematic prefix:

```go
// internal/radio/framing/rm_30_14_tetra.go (shape)
func rm3014Table() *[1 << 14]uint32 {
    rm3014Once.Do(func() {
        for r := 0; r < 14; r++ { /* pack parity row r into rm3014RowMask[r] */ }
        for v := 0; v < 1<<14; v++ {
            var parity uint32
            for r := 0; r < 14; r++ {
                if v&(1<<uint(13-r)) != 0 { parity ^= rm3014RowMask[r] }
            }
            rm3014Codebook[v] = uint32(v)<<16 | parity
        }
    })
    return &rm3014Codebook
}

func DecodeRM3014Tetra(received []byte) ([]byte, int) {
    table := rm3014Table()
    r := rm3014Pack(received)
    bestDist, bestV := 31, 0
    for v, cw := range table {
        d := bits.OnesCount32(cw ^ r)
        if d < bestDist {
            bestDist, bestV = d, v
            if d == 0 { break }
        }
    }
    return rm3014Info(bestV), bestDist
}
```

The hard decoder is now one XOR and one popcount per candidate — 17 µs
for the full search, with the early exit at distance 0 making a clean slot
cheaper still; ties resolve to the lowest information vector, as before.

The soft decoder, `DecodeRM3014TetraSoft`, maximises the correlation
Σ llr·(1 − 2c) over the same table. A 30-term dot product per candidate
would be ~500 k adds; instead the 30 LLRs are split into three 10-bit
groups and a 1024-entry partial-sum table is built per group
(`tab[k][x] = tab[k][x & (x−1)] + llr[lowest set bit]`), so each
candidate's correlation is three lookups and two adds — ~50 k operations,
36 µs, and no allocation on the search path. It returns the same `margin` = (best − second) / (2·Σ|llr|) as before, so
the callers' confidence gates are untouched.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Two pipelines for one AACH decode over 16384 candidates. Old: each candidate re-runs the allocating encoder then 30 byte compares, several hundred microseconds and about sixteen thousand allocations. New: each candidate is one XOR and one popcount against a once-built uint32 table, 17 microseconds hard or 36 soft, zero allocations. Called about 70 times per second per carrier plus once per traffic burst.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one AACH decode: 16 384 candidates either way</text>
  <text x="26" y="52" text-anchor="end" fill="var(--fg-muted)" font-size="8">old</text>
  <rect x="34" y="36" width="70" height="26" fill="none" stroke="currentColor"/>
  <text x="69" y="52" text-anchor="middle" fill="currentColor" font-size="8">30 bits</text>
  <path d="M104 49 L130 49" stroke="var(--fg-muted)"/>
  <rect x="130" y="30" width="330" height="38" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="295" y="44" text-anchor="middle" fill="currentColor" font-size="8">for v in 0..16383: EncodeRM3014Tetra(v) → alloc 30 B, 14×16 parity loop</text>
  <text x="295" y="60" text-anchor="middle" fill="currentColor" font-size="8">then 30 byte compares</text>
  <path d="M460 49 L486 49" stroke="var(--fg-muted)"/>
  <text x="490" y="46" fill="var(--fg-muted)" font-size="8">several hundred µs</text>
  <text x="490" y="60" fill="var(--fg-muted)" font-size="8">~16 k allocations</text>
  <text x="26" y="116" text-anchor="end" fill="var(--accent)" font-size="8">new</text>
  <rect x="34" y="100" width="70" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="69" y="116" text-anchor="middle" fill="var(--accent)" font-size="8">uint32 r</text>
  <path d="M104 113 L130 113" stroke="var(--accent)"/>
  <rect x="130" y="94" width="330" height="38" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="295" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">for cw in rm3014Table (built once): OnesCount32(cw ^ r)</text>
  <text x="295" y="124" text-anchor="middle" fill="var(--accent)" font-size="8">soft: tab[0][cw&gt;&gt;20] + tab[1][…] + tab[2][cw&amp;1023]</text>
  <path d="M460 113 L486 113" stroke="var(--accent)"/>
  <text x="490" y="110" fill="var(--accent)" font-size="8" font-weight="bold">17 µs hard · 36 µs soft</text>
  <text x="490" y="124" fill="var(--accent)" font-size="8" font-weight="bold">0 allocations</text>
  <line x1="34" y1="150" x2="650" y2="150" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="340" y="170" text-anchor="middle" fill="currentColor" font-size="8">called ~70×/s per carrier (every downlink slot) + once per traffic burst in the voice demux</text>
  <text x="340" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dual-TETRA 200 kS/s pump: 0.44× real time → 0.12× (65 % of the old figure was this search)</text>
</svg>
<figcaption>The search is the same; only the cost of producing each candidate changed. Bit-identical results are asserted against the old loop, not assumed.</figcaption>
</figure>

## The filter behind the decoder

With the codebook in place the profile moved, and `filter.FIR.Process`
was next at roughly half the remaining time — every narrowband receiver's
channel filter, run on every sample of every DDC tap. Its inner loop
walked a modulo-indexed ring buffer: for each of N taps, read the history
at `idx`, decrement, and branch on wrap. The fix is a classic one. The
history is kept at **2N** samples and every incoming sample is written
twice, at `histPos` and `histPos + N`, so `hist[histPos : histPos+N]` is
always the last N samples in arrival order with no wrap:

```go
// internal/dsp/filter/fir.go (shape) — FIR.Process
f.hist[f.histPos] = x
f.hist[f.histPos+N] = x
f.histPos++
if f.histPos == N { f.histPos = 0 }
w := f.hist[f.histPos : f.histPos+N]   // oldest-first, contiguous
var accI, accQ float32
for j := len(rt) - 1; j >= 0; j-- {    // newest tap first: same float32 order as the ring loop
    s, h := w[j], rt[j]
    accI += h * real(s)
    accQ += h * imag(s)
}
```

Two details carry the bit-exactness. The taps are stored reversed
(`rtaps[j] = taps[N−1−j]`) so the dot product runs over the contiguous
window, and the accumulation walks `j` **descending** — newest tap first —
which is the exact order the old ring loop summed in. Float32 addition is
not associative; a loop that summed the same terms in a different order
would produce outputs differing in the last bit, and every golden in the
tree that passes through a FIR (which is nearly all of them) would move.
The mirrored window is ~1.8× faster than the ring and produces the same
bits.

## Proving bit-identical

Both rewrites keep the code they replaced as the reference, which is the
part worth copying. `rm_30_14_tetra_codebook_test.go` holds
`bruteForceRM3014` and `bruteForceRM3014Soft` — the original re-encode
searches, verbatim — and `TestRM3014CodebookMatchesBruteForce` runs 400
random words through both: a random info vector encoded and corrupted
with 0–8 bit flips, every fifth word fully random so the uncorrectable,
tie-breaking cases are covered. Hard decodes must match info and distance
exactly; soft decodes info, distance and margin within 1e-9.
`TestRM3014DecodeAllocations` pins ≤ 1 allocation per decode with
`testing.AllocsPerRun`, where the old path made ~16 k; three benchmarks
(`BenchmarkDecodeRM3014Tetra`, `…Soft`, `…BruteForce`) keep the ratio
measurable.

`fir_window_test.go` does the same for the filter: `ringFIR` is the
original modulo-indexed loop, and `TestFIRMirroredWindowIsBitExact` runs
tap counts of 1, 2, 7, 33, 128 and 257 through both, twenty chunks of
random length (1–300 samples) each, comparing every complex64 output for
exact equality — then `Reset`s and checks an impulse. The approach is the
one
[Season 1 Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
recommends for decoders turned inside out: the only test that can
certify "nothing changed" is one that still runs the thing that was
replaced.

## What the pump measures now

With both changes, `TestEngineDualTETRA200kThroughput` reads **0.12×**
real time where it read 0.44× — two TETRA channels at 200 kS/s now take
about an eighth of one core, and the GC churn that rode on the codebook is
gone. The operator's overruns were the consumer; the consumer was a block
decoder, not the DSP; and the DSP's own largest cost was a branch per tap.
Neither change touched a decode decision: the AACH classification gate at
`aachClassifyMaxErrs` and the voice demux's usage-marker routing see the
same `(info, errs, margin)` they always did.

What this does not settle: the symbol-scope panels still build a full
`Downconverter` plus receiver per `/diag/symbols` subscriber, nothing
pooled — a separate CPU item, still open. And the wideband engine still has
no `can't keep up` line of its own; `host_drops` plus this bench is the
instrument until it does.

## Where this goes next

The pump is fast again; the next part is about a recording that vanished
for a different reason entirely. The 17 Sep IPSC capture showed one re-key
in five dropping *both* overs' recordings — the previous call's deferred
finalize met the next call's start on the same serial, and a drain signal
with no call identity finalised the wrong session.
[Part 10]({{ '/blog/solution-postmortem/issue-tracker-s2-10-rekey-dropped-both-overs/' | relative_url }})
fences every drain by `Grant.CallID`.

## FAQ

**Why did GopherTrunk report SoapyRemote host overruns at only 200 kS/s?**
Because `host_drops` is a downstream signal: the driver sheds only when its
consumer stops draining. On the dual-TETRA wideband rig the consumer was
the pump, running at 0.44× real time with 65 % of its CPU in
`DecodeAACH`'s re-encoding codeword search. The wideband path logs no
`decode can't keep up` WARN, so the overruns arrived without a hint.

**What does the RM(30,14) codebook in rm_30_14_tetra.go do?**
`rm3014Table` builds all 16 384 valid (30,14) Reed-Muller codewords once,
packed into `uint32`s and indexed by their 14 information bits.
`DecodeRM3014Tetra` then finds the nearest codeword with one XOR and
`bits.OnesCount32` per candidate (17 µs), and `DecodeRM3014TetraSoft`
correlates via three 10-bit partial-sum tables (36 µs), both allocation-
free and bit-identical to the old search.

**Is the faster AACH decoder guaranteed to give the same answers?**
Yes, tested rather than assumed: `TestRM3014CodebookMatchesBruteForce`
keeps the original re-encode-every-codeword search as the reference and
compares info, Hamming distance and soft margin on 400 random words at
every error weight, uncorrectable ones included.
`TestRM3014DecodeAllocations` pins ≤ 1 allocation per decode.

**What changed in filter.FIR and why is summation order mentioned?**
`FIR.Process` now keeps a 2N mirrored history so the last N samples are a
contiguous slice, removing a per-tap modulo branch (~1.8× faster). It
accumulates newest tap first because float32 addition is order-dependent;
`TestFIRMirroredWindowIsBitExact` compares every output against the old
ring-buffer loop so no FIR-dependent golden moved.

## Series navigation

**Part 9 of 14** · ←
[Part 8: 'unable to encode sample rate 880029' — FLAC's Frame-Header Table]({{ '/blog/solution-postmortem/issue-tracker-s2-08-sample-rate-880029/' | relative_url }})
· Next →
[Part 10: The Re-Key That Dropped Both Overs — Drain Coordination Fenced by Call ID]({{ '/blog/solution-postmortem/issue-tracker-s2-10-rekey-dropped-both-overs/' | relative_url }})
