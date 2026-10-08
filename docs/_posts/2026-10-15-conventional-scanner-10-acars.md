---
title: "The Conventional Scanner, Part 10: ACARS on a Scanner Channel — Coherent MSK and Parity Repair"
description: "How GopherTrunk decodes ACARS off a scanner.conventional AM channel with decoders: [acars] — the 2400 bit/s MSK line code settled empirically against acarsdec, the coherent detector in internal/radio/acars/receiver/msk.go that reads each bit from the absolute phase so one noise hit costs one bit, a Gardner loop on the one-bit phase difference, and a parity-located block repair accepted only when unique and printable (70 of 20 000 garbage blocks validated without the guards, 0 with)."
category: tutorials
keywords: acars decoder sdr, acars rtl-sdr scanner, acars msk demodulation, acars coherent detector, acars crc-16 kermit, acars parity repair, acarsdec comparison, acars 131.550 mhz, GET /api/v1/acars/messages, gophertrunk conventional scanner
tags: [conventional-scanner, acars, air-band, msk, data-decoders, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 10
---

*Part 10 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's conventional (non-trunked) scanner, `scanner.conventional`, from
the dwell loop to a verified kitchen-sink config.
[Part 9]({{ '/blog/tutorials/conventional-scanner-09-am-carrier-tracker/' | relative_url }})
finished the air-band voice path. The same band carries ACARS, the aircraft
data link, as 2400 bit/s MSK tones on an AM carrier — and the #1219
reporter's follow-up ask
([#1231](https://github.com/MattCheramie/GopherTrunk/issues/1231)) was to
decode it off the scanner the way MDC1200 and FleetSync hang off an FM
channel. This part reads what landed: a line code settled by experiment, a
detector chosen because the obvious one inverts whole blocks, and a block
repair that refuses to guess.*

> **TL;DR:** `decoders: [acars]` on a `mode: am` scan-list channel (rejected
> on FM by config validation) runs `internal/radio/acars/receiver` on the
> scanner's channel IQ: AM envelope `|x|` (no carrier recovery), a polyphase
> resample to `AudioRateHz` = 19.2 kHz (8 samples/bit), and a **coherent MSK
> detector** (`msk.go`): mix by the 1800 Hz centre, 1300 Hz low-pass, a
> Gardner timing loop on the one-bit phase difference (`mskTimingGainHunt`
> 0.06 → `mskTimingGainTrack` 0.015 once a block is framing), and the bit
> read from the absolute phase at each boundary after de-rotating by k·π/2.
> The line code — **2400 Hz when a data bit equals the previous one, 1200 Hz
> when it changes** — was settled empirically: of four mappings acarsdec
> decodes only this one. `acars.Framer` hunts a 48-bit sync in either
> polarity; `DecodeBlock` checks CRC-16/KERMIT and repairs up to three
> parity-located bit errors or one adjacent pair, **accepted only when the
> repair is unique and the block is printable** (`TestRepairNeverValidatesGarbage`:
> 70 of 20 000 without the guards, 0 with). Blocks land in `acars_log`,
> `GET /api/v1/acars/messages` and the `/acars` panel. Verified against
> acarsdec both ways; the reporter's live run is pending.

**Key takeaways**

- **A tone-deciding front end is the wrong tool for this MSK.** Data is the
  running XOR of tone decisions, so one wrong tone complements the rest of
  the block (byte 11 inverted 80 bytes; 5/7 messages). The coherent detector
  decodes 7/7.
- **A 16-bit check over many candidates validates garbage.** Three parity
  errors give 512 candidates (≈0.8 % false accept); repairs are accepted
  only when unique AND printable.
- **The line code was not taken on trust.** acarsdec decodes `SynthAudio`
  under one of four mappings only, and GopherTrunk decodes acarsdec's
  real-air recording to the same seven messages.
- **Reference-pinned ≠ on air.** `TestACARSReplay` is the gate; the
  reporter's capture has not been through it yet.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Config | `decoders: [acars]` requires `mode: am` | `config_validate.go`, `conventional.DecoderACARS` |
| Front end | envelope → resample to 19.2 kHz → coherent MSK → framer | `acars/receiver/receiver.go` (`ProcessIQ`, `ProcessAudio`) |
| Detector | Gardner on `arg(z·z*(t−T))/(π/2)`; bit = sign of de-rotated phase | `msk.go` (`mskDemod.events`, `boundary`) |
| Framing + check | 48-bit sync, ≤2 errors, either polarity; CRC-16/KERMIT = 0 | `framer.go`, `acars.CRC`, `TestCRCIsKermit` (0x2189) |
| Repair | ≤3 parity errors / 1 BCS bit / adjacent pair; unique + `plausible` | `DecodeBlock`, `correct`, `TestRepairNeverValidatesGarbage` |
| Outputs | `acars_log`, `GET /api/v1/acars/messages`, `/acars`, `acars.message` | `storage/acarslog.go`, `api/handlers_acars.go`, `events.KindACARSMessage` |

## In this post

- **The block on the wire** — characters, parity, sync, BCS.
- **A line code settled by experiment** — four mappings, one survivor.
- **Why coherent** — the XOR trap and the phase that undoes it.
- **A repair that refuses to guess** — syndromes, uniqueness, printability.
- **On the scanner, and what is verified** — the hold, the outputs, the open gate.

## The block on the wire

ACARS (ARINC 618) is 2400 bit/s on 131.550 MHz and friends. Every character is 7-bit ASCII, sent LSB first, with **odd parity**
in bit 7. One block, from `acars.go`'s package comment:

```text
pre-key   ≥16 × 0xFF          one steady tone; lets the clock settle
sync      '+' '*' SYN SYN     0xAB 0x2A 0x16 0x16 with parity
SOH · mode (1) · address (7, '.'-padded) · ack (1) · label (2) · block id (1)
STX       0x02                or ETX straight away: no text
text      ≤220 chars          downlink: msg no (4) + flight (6) + body
suffix    ETX 0x03 | ETB 0x17 (more blocks follow)
BCS (2, low byte first) · DEL 0x7F
```

The block check is CRC-16/KERMIT — reflected CRC-CCITT, polynomial `0x8408`,
initial value 0, no final XOR — over mode through the suffix **including the
parity bits**; over body plus BCS it yields zero. `TestCRCIsKermit` pins the
catalogue value `CRC("123456789") = 0x2189` independently of any frame.

Provenance: layout, parity sense, bit order, CRC convention and line code
were read from acarsdec (GPL — protocol facts only, nothing ported). Three
literal on-air blocks from acarsdec's
own `test.wav` pin the parser in `acars_test.go` (`realAirFrames`: LN-DYY
with a `_d` label and no text, PH-BXR's `5V` downlink `KL1681`, G-DBCK's
`BA031T`), and `TestEncoderReproducesRealAirFrames` demands `EncodeBlock`
re-create each byte for byte. The WAV is GPL content and is not committed.

## A line code settled by experiment

MSK with h = 0.5 at 2400 baud uses two tones, 1200 and 2400 Hz, and there
are four ways to map them onto data. The receiver's package comment records
how it was decided: `SynthAudio` rendered each mapping, and acarsdec — a
decoder proven on air — decoded **only one**: one cycle of **2400 Hz when a
data bit equals the previous one**, half a cycle of **1200 Hz when it
changes**. That is MDC1200's construction with the tones swapped
([Beyond Voice Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
has that side), and it makes the pre-key's run of ones a steady 2400 Hz
tone — which is why the pre-key carries no timing information.

## Why coherent

The first front end tried was the FFSK tone discriminator MDC1200 and
FleetSync use. It decoded 5 of the recording's 7 messages, and one failure
was instructive: a single tone error at byte 11 **complemented the remaining
80 bytes**. Deciding tones means integrating them back into data (the
running XOR of tone decisions), so one wrong decision inverts every later
bit; the 16-bit check catches it, nothing can repair it.

MSK's phase is the same integral, read directly. Mixed down by the 1800 Hz
centre, the phase advances +π/2 over a "same" bit and −π/2 over a "change"
bit, so at bit boundary k

```text
φ_k = φ_0 + (π/2)·(k − 2·n_k),   n_k = number of changes so far
```

and after de-rotating by k·π/2 the phase is φ_0 − π·n_k: BPSK whose sign is
the data bit, up to the one global sign the framer resolves. A noise hit now
costs **one bit**, which parity and the block check can repair.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Pipeline of the ACARS coherent MSK detector: 19.2 kHz audio through a DC blocker, a mix by the 1800 Hz centre, a 1300 Hz low-pass, then a Gardner timing loop on the one-bit phase difference and a per-boundary decision that de-rotates by k pi over two plus theta and slices the real part, feeding the framer. A footer contrasts a tone-deciding detector, where one tone error complemented 80 bytes, with the coherent one, where an error costs one bit.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">mskDemod · 19.2 kHz audio (8 samples/bit) → one soft data value per bit boundary</text>
  <g font-size="8" fill="currentColor">
    <rect x="8" y="34" width="70" height="34" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="43" y="55" text-anchor="middle">DC block</text>
    <rect x="90" y="34" width="90" height="34" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="135" y="49" text-anchor="middle">× e^(−j2π·1800t)</text>
    <text x="135" y="61" text-anchor="middle">to baseband</text>
    <rect x="192" y="34" width="80" height="34" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="232" y="49" text-anchor="middle">LPF 1300 Hz</text>
    <text x="232" y="61" text-anchor="middle">127 taps</text>
    <rect x="300" y="24" width="170" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="385" y="39" text-anchor="middle" fill="var(--accent)">timing: Gardner on x(t) = arg(z·z*(t−T))/(π/2)</text>
    <text x="385" y="53" text-anchor="middle" fill="var(--accent)">peaks ±1 at boundaries · 0 mid-bit on a change</text>
    <rect x="300" y="74" width="170" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="385" y="89" text-anchor="middle" fill="var(--accent)">data: at boundary k, y = z·e^(−j(kπ/2+θ))/|z|</text>
    <text x="385" y="103" text-anchor="middle" fill="var(--accent)">bit = sign Re(y) · θ += 0.08·err</text>
    <rect x="500" y="48" width="100" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="550" y="64" text-anchor="middle">acars.Framer</text>
    <text x="550" y="76" text-anchor="middle">Push(bit)</text>
    <rect x="612" y="48" width="60" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="642" y="64" text-anchor="middle">Decode</text>
    <text x="642" y="76" text-anchor="middle">Block</text>
  </g>
  <line x1="78" y1="51" x2="90" y2="51" stroke="currentColor"/>
  <line x1="180" y1="51" x2="192" y2="51" stroke="currentColor"/>
  <path d="M272 51 L286 51 L286 44 L300 44 M286 51 L286 94 L300 94" fill="none" stroke="currentColor"/>
  <path d="M470 94 L486 94 L486 68 L500 68" fill="none" stroke="currentColor"/>
  <line x1="600" y1="68" x2="612" y2="68" stroke="currentColor"/>
  <text x="385" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gain 0.06 while hunting · 0.015 once Framer.Busy()</text>
  <line x1="20" y1="146" x2="660" y2="146" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="20" y="164" fill="var(--fg-muted)" font-size="8">tone-deciding (FFSK) front end: one tone error at byte 11 complemented the remaining 80 bytes · 5 of 7 messages</text>
  <text x="20" y="182" fill="currentColor" font-size="8" font-weight="bold">coherent detector: a noise hit costs one bit · 7 of 7, ≥ acarsdec at every noise level tried</text>
</svg>
<figcaption>The tones encode whether a bit changed; reading the phase instead keeps one noise hit to one bit.</figcaption>
</figure>

In `msk.go`: a one-pole DC blocker (the envelope detector leaves the AM
carrier as DC), the complex mix by 1800 Hz, a 127-tap Kaiser low-pass at
1300 Hz (keeps the ±600 Hz deviation and MSK's main lobe, rejects the
3600 Hz image). Timing is a **Gardner loop on the one-bit phase difference**
`x(t) = arg(z(t)·z*(t−T))/(π/2)`: over a bit the phase moves ±π/2, so `x` is
a triangle wave peaking at ±1 on bit boundaries and crossing zero mid-bit
wherever the tone changes — Gardner's NRZ shape with the decision instants
on the boundaries (the per-sample frequency was tried first and slipped
bits under noise). The gain drops from `mskTimingGainHunt` = 0.06 to
`mskTimingGainTrack` = 0.015 once `Framer.Busy()` reports a block framing,
because a slip mid-block shifts every later character — the dominant loss at
~9 dB Eb/N0 with a single gain. At each boundary the sample is de-rotated by
`k·π/2` plus a decision-directed phase `θ` (`mskPhaseGain` = 0.08) and the
sign of the real part is the bit. `TestDecodesAtLowSNR` records the sweep:
40/40 at 11.4 dB Eb/N0, 37/40 at 8.9, 29/40 at 7, the cliff at ~5.5 dB.

## A repair that refuses to guess

`acars.Framer` hunts a 48-bit window — the last pre-key character plus
`+ * SYN SYN SOH` — at ≤ `syncMaxErrors` = 2 in either polarity (the sign's
start state is unknown, so the complement is accepted and the block's bits
inverted), collects characters to the ETX/ETB suffix, reads two BCS bytes
and hands the block to `DecodeBlock`. A DEL with no suffix seen means a bit
error hit the ETX/ETB; the three characters before it are suffix + BCS.

The CRC is linear (zero initial value, no final XOR), so a lone set bit's
contribution depends only on how many bytes follow it and one
`syndromeTable[d][i]` covers every position. `correct` searches a bounded
space: with 1..`maxParityErrors` = 3 characters failing parity, one wrong
bit in each (8ⁿ candidates) plus, for a single failing character, one wrong
BCS bit; with every character passing parity, one wrong BCS bit or **two
adjacent wrong bits inside one character** (parity cannot see an even
number; adjacent pairs are the burst a noise hit produces).

A 16-bit check over n candidates validates a wrong one with probability
≈ n/65536 — 0.8 % at three parity errors — which is why acarsdec-style
"first match wins" was not copied. Two guards:

```go
// internal/radio/acars/acars.go (shape) — DecodeBlock
n, ok := correct(fixed, bcs)          // ok only if EXACTLY ONE candidate validates
if ok && n > 0 && !plausible(fixed) { // repaired blocks must read as ACARS
    ok = false
}
```

`correct` stops as soon as a second candidate validates, and `plausible`
requires printable header characters (the ack may be NAK, the label's second
character DEL) and printable text (CR/LF/TAB allowed).
`TestRepairNeverValidatesGarbage` feeds 20 000 blocks of random parity-valid
characters with 0..3 broken ones and a random BCS — the worst case, since
the parity-located search runs every time — and demands **zero** accepted;
without the guards 70 of them "repair" into CRC-valid garbage (the figure
the `DecodeBlock` docstring records). `TestCorrectsSingleBitErrors` walks
every single-bit position of a long block;
`TestCorrectsThreeParityErrorsAndAdjacentPairs` requires ≥ 270/300 of each
class repaired to the *right* block. `Corrected` rides to the log and panel
as `fixed N`.

## On the scanner

The config is one line on an AM channel (`config.example.yaml`):

```yaml
scanner:
  conventional:
    - label: "ACARS primary"
      frequency_hz: 131550000
      mode: am
      decoders: [acars]
```

`validateConvChannel` rejects `acars` on anything but `mode: am` — "ACARS is
AM; got mode %q" — because an FM channel's dBFS squelch and voice chain are
the wrong tools for an air-band data channel. The daemon's
`convDataDecoderFactory` (`cmd/gophertrunk/conv_decoders.go`) builds
`acarsrx.New` with the ~48 kHz channel rate the scanner's data front end
hands it (the Part 11 mechanism), the scanner's serial and the channel's
frequency; the receiver envelope-detects the channel IQ itself.
`TestDecodesAMChannelAtScannerRates` pins the front end's rates (2.4 MS/s
÷ 50, 2.048 MS/s ÷ 40) with the carrier several kHz off centre — an envelope
detector does not care.

Two behaviours are ACARS-specific. A full block is ~0.86 s, longer than any
FFSK burst, so `holdMaxFor("acars")` returns `acarsScanHoldMax` = 1 s: a
`Busy` decoder extends the scan window in 50 ms steps up to that, and while
dwelling counts as activity so hangtime cannot cut a block
(`TestACARSDecoderHoldsScanWindowForALongBlock`). And `Reset` clears every
stage, framer included, on retune, so a block cut off by a hop is abandoned.
`TestScannerDecodesACARSOnAMChannel` runs a synthetic transmission through
the whole scanner at 2.4 MS/s and requires the production receiver to frame
the block.

Outputs follow the eleven-place pattern of
[Beyond Voice Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }}):
`events.KindACARSMessage` (`acars.message`) carries a `storage.ACARSMessage`
(registration, flight id, message number, label, block id, text, `CRCOK`,
`Corrected`, raw hex, serial + frequency); `storage.ACARSLog` drains it into
`acars_log` (swept by `retention.log_days`);
`GET /api/v1/acars/messages?limit=N` answers from it (503 without
`storage.path`); the `/acars` panel renders it (`web.tabs.acars: false`
hides it). Known limits, from `docs/acars.md`: a block also opens an
ordinary AM "call", ETB multi-block messages are not reassembled, labels are
raw, and VDL Mode 2 is a different air interface.

## What is verified

**Reference-pinned, both directions.** GopherTrunk decodes all 7 messages in
acarsdec's real-air `test.wav` (4-channel 12 kHz AM audio) to the fields
acarsdec prints, and acarsdec decodes `SynthAudio`; with white noise added,
the coherent detector decodes as many or more messages than acarsdec at
every level tried.

**Not yet on air.** The reporter's capture — rtl_fm AM audio at 12.5 kHz
plus acarsdec's text decode — has not been replayed. `TestACARSReplay`
(`cmd/gophertrunk/acars_replay_test.go`) is the gate; it takes AM audio
(`GT_ACARS_AUDIO`, any rate or channel count) or raw IQ (`GT_ACARS_IQ` with
`GT_ACARS_RATE`, `GT_ACARS_FORMAT`, `GT_ACARS_TUNE_HZ`):

```text
GT_ACARS_AUDIO=capture.wav go test ./cmd/gophertrunk -run TestACARSReplay -v
```

It prints `sync_locks= blocks= crc_ok= corrected= aborted=` per run and every
block with its time, `crc_ok`, `fixed` count, summary and raw hex;
`GT_ACARS_EXPECT=N` turns the report into an assertion. Its verdict line
says how to read a zero: `sync_locks=0` means no MSK/rate match (check the
rate, and that the file is AM audio, not FM); locks but no valid blocks
means too weak, or a framing defect. **The CRC is the verdict, never "it
printed something."** A live scanner run on the reporter's rig then closes
#1231.

## Where this goes next

ACARS rides the scanner's data front end — the decimate-to-48 kHz,
±8 kHz-filtered channel IQ the dwell loop feeds every decoder on a channel.
[Part 11]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }})
reads that front end for the FM-channel decoders it was built for, MDC1200
and FleetSync: why raw IQ decoded 0 of 2 bursts and the front end 2 of 2 on
the #1184 slices, why decoders ignore the tone gate, and the 28 September
on-air MDC1200 run that used exactly this path.

## FAQ

**How do I decode ACARS with GopherTrunk?**
Add a `scanner.conventional` entry with `mode: am`, the carrier (e.g.
`frequency_hz: 131550000`) and `decoders: [acars]`, and set `storage.path`.
Blocks land in `acars_log`, `GET /api/v1/acars/messages`, the `/acars` panel
and the `acars.message` event. Config validation rejects `acars` on an FM
channel.

**Why does GopherTrunk use a coherent MSK detector instead of deciding tones?**
ACARS's tones encode whether a bit *changed*, so one wrong tone decision
inverts the rest of the block (measured: byte 11 → 80 complemented bytes,
5/7 messages). The coherent detector in `msk.go` reads each bit from the
absolute phase after de-rotating by k·π/2, so a noise hit costs one bit; it
decodes 7/7.

**What does "fixed N" mean on an ACARS message?**
`DecodeBlock` repaired N bits — up to three parity-located single-bit
errors, one BCS bit, or one adjacent pair — before the block check
validated. A repair is accepted only when it is the unique candidate and
the result is printable ACARS; `TestRepairNeverValidatesGarbage` pins zero
false accepts over 20 000 garbage blocks.

**Is the ACARS decoder verified on air?**
Reference-pinned, not yet on air: it decodes the 7 real-air messages in
acarsdec's test recording and acarsdec decodes its synthesized audio. The
reporter's capture through `TestACARSReplay`, then a live scanner run, are
the open gates on #1231.

## Series navigation

**Part 10 of 14** · ←
[Part 9: The AM Carrier Tracker — Why a 3 kHz Tuning Error Cost a Sideband]({{ '/blog/tutorials/conventional-scanner-09-am-carrier-tracker/' | relative_url }})
· Next →
[Part 11: MDC1200 and FleetSync on Scan-List Channels — The Data Front End Behind the Dwell]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }})
