---
title: "Beyond Voice, Part 11: LoRa — A Pure-Go Chirp Receiver"
description: How GopherTrunk decodes LoRa without Semtech silicon — chirp spread spectrum as cyclic shifts, the dechirp-and-fold FFT that reads a symbol as a bin, preamble and SFD sync that separates carrier offset from timing, the Gray, interleave, Hamming and whitening chain, LDRO, and LoRaWAN MIC verification and decryption with operator-held keys.
category: deep-dives
keywords: lora decoder sdr, chirp spread spectrum go, dechirp fft, lora spreading factor, lora diagonal interleaver, lora whitening lfsr, lorawan mic aes-cmac, low data rate optimize, gophertrunk lora
tags: [beyond-voice, lora, lorawan, chirp, iot, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 11
---

*Part 11 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats,
paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice
family — and the one eleven-place wiring pattern that carries each of them
from a burst on the air to a row in the web console.
[Part 10]({{ '/blog/deep-dives/beyond-voice-10-dsc-marine-calling/' | relative_url }})
closed the marine pair, both narrowband FSK through a discriminator. This
part is the series' one spread-spectrum decoder: LoRa, where a symbol is a
frequency sweep across the whole channel and the receiver's central move is
a multiply and an FFT rather than a slicer.*

> **TL;DR:** `internal/radio/lora` is a pure-Go LoRa modem (#586). A symbol is
> the base upchirp (`GenChirp`) cyclically shifted by its value, N = 2^SF
> chips, SF 7–12, BW 125/250/500 kHz, `DefaultOversample` 2. The
> `dechirper` multiplies a window by the downchirp, FFTs, **folds** the osf
> copies and takes the peak bin; `syncFrame` locks four stable preamble
> peaks, finds the SFD, and solves `cfo = bd/2`. Bits
> come out through `grayDecode` → `Deinterleave` → `HammingDecode4` →
> `Whiten` (LFSR 0xB8) → `PayloadCRC` (0x1021). The header is a compact
> **self-consistent** layout; bit-exact Semtech interop is gated on golden
> vectors not yet committed. `lora/receiver` channelizes one SDR into
> sub-channels, one demodulator per candidate SF, and on sync word 0x34
> parses LoRaWAN 1.0.x, verifying the AES-CMAC MIC and decrypting with
> operator-held keys. Output: `KindLoRaFrame` → `lora_log`
> → `GET /api/v1/lora/frames` → `/lora`. Synthetic-verified only.

**Key takeaways**

- **Dechirp turns modulation into a bin index.** Multiply by the conjugate
  chirp and every symbol collapses to a tone; the folded FFT's processing
  gain decodes SF12 at 0 dB SNR in the tests.
- **Preamble and SFD together separate offset from timing.** Upchirps peak
  at `timing + cfo`, SFD downchirps at `cfo − timing`; realign on the first
  and the second reads `2·cfo`.
- **The bit chain is exact inverse pairs, and that is also its limit.**
  Round-trips prove consistency; Semtech interop waits on a golden packet.
- **LoRaWAN decrypts only with keys you hold.** `KeyStore.Decode` verifies
  the MIC under NwkSKey and decrypts under AppSKey — no key recovery.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Chirp synthesis | base up/down chirp, cyclic shift by symbol value | `internal/radio/lora/chirp.go` (`GenChirp`, `GenSymbolChirp`) |
| Dechirp + sync | fold-FFT peak, preamble/SFD lock, `cfo = bd/2` | `internal/radio/lora/demod.go` (`peakOf`, `syncFrame`) |
| Bit chain + LDRO | Gray, diagonal interleave, Hamming(4+CR,4), whitening; SF−2 rows | `fec.go`, `whitening.go`, `ldro.go` (`Deinterleave`, `AutoLDRO`) |
| Streaming receiver | one SDR → `tuner.Bank` → demod per SF → bus | `internal/radio/lora/receiver/receiver.go` (`subChannel.bestDecode`) |
| LoRaWAN | MAC parse, AES-CMAC MIC, AES-CTR payload | `lorawan/mac.go`, `mic.go`, `crypt.go` (`KeyStore.Decode`) |
| Bus → storage → REST → panel | `KindLoRaFrame` → `lora_log` → `/api/v1/lora/frames` → `/lora` | `storage/loralog.go`, `api/handlers_lora.go`, `web/src/panels/LoRa.tsx` |

## In this post

- **Chirps and spreading factors** — what the receiver has to find.
- **Dechirp, fold, FFT** — a window becomes a bin.
- **The bit chain** — Gray, interleave, Hamming, whitening, CRC, LDRO.
- **One SDR, many sub-channels** — the streaming receiver.
- **LoRaWAN** — MIC and decryption with keys you already hold.
- **From bus event to the `/lora` panel** — the eleven places, with gaps.

## Chirps and spreading factors: what the receiver has to find

[LoRa]({{ '/reference/lora/' | relative_url }}) is chirp spread spectrum: a
symbol is an upchirp sweeping −BW/2 to +BW/2, and the *value* is where the
sweep starts — a cyclic shift of the base chirp by `v` of N = 2^SF chips. Bandwidth (`BW125`, `BW250`, `BW500`) sets the sweep
width; the spreading factor (7–12) sets its length and the processing gain.
(The reference page's "Not decoded" line predates #586, which landed the
decoder in v0.3.8.)

`GenChirp` defines the waveform at Fs = osf·BW, so it depends only on SF and
osf: `phase(n) = 2π(−n/2·osf + n²/2·N·osf²)`, conjugated for the downchirp
reference; `GenSymbolChirp` rotates it by `v·osf` samples.
`DefaultOversample` is 2 — enough to resolve the peak and a fractional
offset with a small FFT — and SF6 is out of scope (`MinSF`/`MaxSF` pin
7..12).

## Dechirp, fold, FFT: from a symbol window to a bin

The kernel is `dechirper.peakOf`. Multiply a symbol-length window by the
reference chirp (downchirp for data, upchirp for the SFD), FFT the
`sps = N·osf` product, and **fold**: bin `b` aliases with `b + r·N` for each
osf copy, so summing their powers puts a symbol's energy in one of N bins.

```go
// internal/radio/lora/demod.go (shape)
for i := range window { d.y[i] = window[i] * ref[i] }        // dechirp
d.out = d.plan.Forward(d.out, cmplx128(d.y))                  // FFT, size sps
for b := 0; b < d.n; b++ {                                     // fold osf copies
    for r := 0; r < d.osf; r++ { sum += power(d.out[b+r*d.n]) }
    d.mags[b] = sum
}
/* argmax → bin; interpolation → frac; peak/runner-up → snr; peak/mean → conc */
```

Two confidence figures come out with the bin: `snr`, peak over runner-up
(the panel's `Frame.SNRdB`), and `conc`, peak over the per-bin mean — the
detection gate, because a tone straddling two bins still dwarfs the mean
where a peak/runner-up ratio would collapse.

`syncFrame` slides one chip at a time until `preambleSyms` = 4 consecutive
windows exceed `concThresh` = 16 at the same bin (±1): the preamble
dechirps to a stable `bu = (timing + cfo) mod N`. It realigns the
grid by `bu·osf` samples, then steps symbol by symbol until a window
dechirps *better against the upchirp*: the SFD. On the realigned grid
the residual timing equals −cfo, so the SFD peak reads
`bd = cfo − timing = 2·cfo`. The header begins 2.25 symbols after the SFD,
nudged by `cfo` chips; `mix` removes the fractional offset as a phase ramp,
with
[SDR Internals Part 8]({{ '/blog/deep-dives/sdr-internals-08-equalization-diversity-fft/' | relative_url }})'s
FFT plan doing the heavy lifting.

The tests state the envelope: `TestModDemodClean` decodes every SF 7–12 at
every CR 1–4; `TestModDemodNoise` decodes SF7, 9 and 12 at **0 dB SNR**;
`TestModDemodCFO` survives offsets of −3, 1.5 and 4 bins;
`TestModDemodSFAutoDetect` proves an SF7 demodulator does *not* decode an
SF9 frame — what the streaming receiver's per-SF probing relies on.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Three upchirp windows, the third shifted, feed a multiply-by-downchirp box and an FFT-and-fold box, giving a spectrum with one dominant bin labelled bin equals v; below, the sync solve from preamble bin bu to SFD bin bd equals two cfo.">
  <g stroke="currentColor" fill="none">
    <rect x="12" y="20" width="60" height="60"/><line x1="14" y1="78" x2="70" y2="22"/>
    <rect x="76" y="20" width="60" height="60"/><line x1="78" y1="78" x2="134" y2="22"/>
    <rect x="140" y="20" width="60" height="60"/><line x1="142" y1="50" x2="170" y2="22"/><line x1="170" y1="78" x2="198" y2="50"/>
  </g>
  <text x="106" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="8">upchirps · third shifted: value v = cyclic shift</text>
  <line x1="204" y1="50" x2="232" y2="50" stroke="currentColor"/>
  <rect x="236" y="30" width="96" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="284" y="54" text-anchor="middle" fill="var(--accent)" font-size="9">× downchirp</text>
  <line x1="332" y1="50" x2="360" y2="50" stroke="currentColor"/>
  <rect x="364" y="30" width="96" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="412" y="47" text-anchor="middle" fill="currentColor" font-size="9">FFT (N·osf)</text>
  <text x="412" y="61" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fold osf copies → N bins</text>
  <line x1="460" y1="50" x2="488" y2="50" stroke="currentColor"/>
  <line x1="500" y1="80" x2="668" y2="80" stroke="var(--fg-muted)"/>
  <line x1="590" y1="80" x2="590" y2="24" stroke="var(--accent)" stroke-width="2"/>
  <text x="590" y="18" text-anchor="middle" fill="var(--accent)" font-size="9">bin = v</text>
  <text x="340" y="140" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">syncFrame</text>
  <text x="340" y="162" text-anchor="middle" fill="currentColor" font-size="9">4 stable preamble windows → bu = (timing + cfo) mod N → realign grid by bu·osf</text>
  <text x="340" y="184" text-anchor="middle" fill="var(--accent)" font-size="9">SFD dechirped with the upchirp: bd = cfo − timing = 2·cfo → cfo = bd/2</text>
</svg>
<figcaption>A shifted upchirp times the downchirp is a tone; the folded FFT reads its bin as the symbol, and the preamble/SFD bins solve carrier offset and timing jointly.</figcaption>
</figure>

## The bit chain: Gray, interleave, Hamming, whitening, CRC

`decodeFrameFromSymbols` decodes the header block first, learns the payload
geometry, then exactly that many payload blocks of `(4+CR)` symbols;
`decodeBlock` reverses exact-inverse transforms:

| Stage | RX direction | Purpose |
|---|---|---|
| [Gray]({{ '/reference/gray-code/' | relative_url }}) | `grayDecode` | a ±1 bin slip costs one bit |
| [Diagonal interleave]({{ '/reference/lora-diagonal-interleaver/' | relative_url }}) | `Deinterleave`: `symbol[k]` bit `(r+k) mod ppm` = `codeword[r]` bit `k` | one bad symbol, one bit per codeword |
| [Hamming]({{ '/reference/hamming-code/' | relative_url }})(4+CR, 4) | `HammingDecode4` | CR 1 parity, CR 2 detect, CR 3 correct, CR 4 SEC-DED |
| [Whitening]({{ '/reference/lora-whitening/' | relative_url }}) | `Whiten` — LFSR 0xB8, seed 0xFF | self-inverse XOR |
| CRC | `PayloadCRC` — 0x1021, init 0 | `Frame.CRCOK` |

`TestHammingSingleErrorCorrection` flips every bit of every CR 3/4 codeword
and expects the nibble back. The **explicit header** is one block at
`HeaderCR` = 4 — the payload's rate is not yet known — of four nibbles:
payload length, `CR | hasCRC<<3`, an XOR checksum (`ParseHeader`). The
source calls this "a compact, self-consistent header layout"; a bit-exact
Semtech header is a follow-up, and the same caveat covers the interleaver's
rotation sign, whitening sequence and CRC seed — "validated separately
against captured golden vectors", none committed. In
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
terms: round-trips, no literal vectors.

**LDRO** is the one place the payload geometry changes. At high SF clock
drift can smear the peak across the two lowest bins, so Low Data Rate
Optimize carries `ppm = SF − 2` bits per symbol in bins that are multiples
of four (`reducedRateShift`). `AutoLDRO` applies Semtech's Ts ≥ 16 ms rule
(SF11/SF12 at 125 kHz, SF12 at 250 kHz), `low_data_rate_optimize` overrides
it, and `TestLDROMismatchFailsToDecode` proves the mode changes the wire
format — the header still parses, the payload does not.

## One SDR, many sub-channels: the streaming receiver

`lora/receiver` is the only decoder in this series that owns its own tuner
bank. `newBank` picks a `tuner.NewChannelizerBank` at
`channelizerThreshold` = 7 or more sub-channels and a `tuner.NewDDCBank`
otherwise — the [channelizer]({{ '/reference/channelizer/' | relative_url }})
versus per-tap DDC trade-off — and each tap delivers narrowband IQ at
osf·BW to a `subChannel`.

```go
// internal/radio/lora/receiver/receiver.go (shape)
sfs := []int{cfg.SF}
if cfg.SF == 0 { for sf := lora.MinSF; sf <= lora.MaxSF; sf++ { sfs = append(sfs, sf) } }
for _, sf := range sfs {
    sc.demods = append(sc.demods, lora.NewDemodulatorMode(sf, osf, bw, cfg.LDRO))
}
sc.minDecode = maxSps * 700   // a 255-byte CR 4/8 frame must be fully buffered
```

A sub-channel with `spreading_factor: 0` runs six demodulators and
`bestDecode` tries each, preferring a CRC-valid frame — "this trades CPU for
coverage; pinning a single SF per sub-channel in config avoids the parallel
probes." `decodeReady` decodes only while `minDecode` samples sit past the
cursor, so a truncated frame is never mistaken for a corrupt one.
`TestWidebandTwoChannels` sums an SF7 packet at −200 kHz and an SF9 packet
at +180 kHz into one 1 MS/s stream and expects both on the bus with the
right SF; `TestWidebandLDROChannel` proves the override reaches the
demodulator.

On a waterfall the diagonal ramps are unmistakable — the signature
[Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
names in the wideband survey and
[The Hunt Part 4]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
classifies. This receiver turns that identification into rows.

## LoRaWAN: MIC and decryption with keys you already hold

When a sub-channel's `sync_word` is `0x34` (LoRaWAN; `0x12` is private),
`emit` hands the payload to `KeyStore.Decode`. `lorawan.Parse` reads the
[LoRaWAN]({{ '/reference/lorawan/' | relative_url }}) 1.0.x MAC layer:
MHDR's top three bits are the `MType`; data frames carry DevAddr, FCtrl,
FCnt, FOpts, optional FPort and FRMPayload, then the 4-byte MIC; a
join-request carries JoinEUI, DevEUI and DevNonce. Without keys for the DevAddr it stops there; the cleartext fields still
publish.

With keys, `VerifyMIC` recomputes the
[MIC]({{ '/reference/lorawan-mic/' | relative_url }}): AES-CMAC (RFC 4493)
over a `B0` block (`0x49`, direction, DevAddr, FCnt, length) plus
`MHDR|FHDR|FPort|FRMPayload`, keyed under NwkSKey, compared in constant
time. `TestAESCMACVectors` pins
the primitive against RFC 4493 §4's vectors — the package's one
literal-vector test — and `TestLoRaWANRoundTrip` decrypts `temp=21.5C` and
confirms a wrong NwkSKey fails the MIC. `DecryptFRMPayload` is the §4.3.3
AES-CTR-style scheme: encrypt `A` blocks (`0x01`, direction, DevAddr, FCnt,
block index) into a keystream and XOR, under AppSKey for FPort > 0 and
NwkSKey for FPort 0.

`FCnt32` assumes the upper 16 bits are zero (no session state is kept), and
join-accept bodies are left raw. The posture is the trunking one from
[Trunking Engine Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }}):
"GopherTrunk decrypts only with keys the operator already holds and is
authorised to use — it performs no key recovery." Keys arrive as
`lorawan_keys` (`dev_addr`, `nwk_skey`, `app_skey`); the `lora_log` row
carries `mic_ok`, `decrypted` and the plaintext as `decoded`.

## From bus event to the `/lora` panel

The [Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
pattern, walked for LoRa — with two places the walk finds empty:

1. **Front end + framer**: `lora/receiver` and `lora`.
2. **Bus event**: `events.KindLoRaFrame` (`"lora.frame"`), payload
   `storage.LoRaFrame`.
3. **Storage**: `storage.LoRaLog` → `lora_log`, indexed on `(received_at)`
   and `(dev_addr, received_at)`.
4. **REST**: `GET /api/v1/lora/frames?limit=N`.
5. **Panel**: `LoRa.tsx` polls every 5 s — SF/CR/BW, SNR, CFO, a `crc`
   badge, the LoRaWAN envelope.
6. **Config**: `lora.channels[]` — `serial`, `center_hz`, `bandwidth`,
   `sub_channels[]`, `lorawan_keys[]`.
7. **Field help**: `LoRaConfig.*` through `LoRaWANKeyConfig.*`.
8. **Route tests**: `"/lora"` in `web/src/nav/registry.test.ts`.
9. **Preflight**: **absent** — `preflight.go` names every other message
   decoder as needing `storage.path`, not lora.
10. **Retention**: **absent** — `lora_log` is not in
    `storage.decoderLogTables`, so `retention.log_days` never sweeps it.
11. **config.example.yaml**: the commented `lora:` block.

A third soft spot: the web Config Builder has no typed LoRa editor —
`webRoundTripAllow` in `configbuilder/webschema_test.go` records
`Config.LoRa` as round-tripping through "the generic index-signature editor;
a bespoke typed editor in types.ts is a follow-up." Such gaps are what the
pattern exists to make visible.

Verification is where Parts 9 and 10 left it: synthetic only. Every LoRa
test drives `LoRaModulate` into the package's own demodulator; the one
independent anchor is RFC 4493's CMAC vectors. No captured LoRa packet is
committed.

### How LoRa shaped the Go code

- **A modem, not just a receiver.** `LoRaModulate` and `Demodulator` are
  exact inverses in one package; every transform is round-trip testable.
- **The dechirper is the reusable kernel.** One struct owns the chirps, FFT
  plan and scratch buffers per (SF, osf); offline `Demodulator` and
  streaming `subChannel` both run on it.
- **Confidence travels with the symbol.** `peak` carries `bin`, `frac`,
  `snr` and `conc`; sync gates on concentration, the panel shows SNR.
- **Decoders never guess with keys.** `HasKeys`, `MICOK` and `Decrypted` are
  separate flags; a frame with no keys still reaches the log.

## Where this goes next

LoRa was data with no voice at all. The next decoder brings voice back,
open-source: M17's 4FSK link layer, its Link Setup Frame, Golay and
convolutional coding, and the Codec 2 payload.
[Part 12]({{ '/blog/deep-dives/beyond-voice-12-m17-open-digital-voice/' | relative_url }})
follows it from the 4800-baud carrier to stream and packet modes.

## FAQ

**Which LoRa settings does GopherTrunk decode?**
Spreading factors 7 through 12 (auto-detected when `spreading_factor: 0`),
coding rates 4/5 to 4/8 from the explicit header, bandwidths 125, 250 and
500 kHz, with or without payload CRC, and Low Data Rate Optimize by the
Semtech rule or an override. SF6 and implicit headers are out of scope.

**Does GopherTrunk decode real Semtech LoRa packets bit-exactly?**
Not yet verified. The chain is round-trip tested against its own modulator,
and the source marks the header layout, interleaver rotation sign, whitening
sequence and CRC seed as calibration steps gated on captured golden vectors,
none committed. Treat interop as synthetic-verified only.

**How does the receiver estimate carrier offset?**
From two dechirp peaks. The repeated preamble upchirps peak at
`timing + cfo`; after realigning the grid on that bin, the SFD downchirps
peak at `2·cfo`. Halving it gives the offset, which `mix` removes before
demodulation.

**Can GopherTrunk decrypt LoRaWAN traffic?**
Only for devices whose session keys the operator supplies under
`lorawan_keys` (DevAddr, NwkSKey, AppSKey). The MIC is verified by AES-CMAC
under NwkSKey and the payload decrypted under AppSKey (NwkSKey for FPort 0).
No key recovery; frames without keys publish their cleartext header fields.

**Why does `lora_log` grow without limit?**
Because `lora_log` is not in the retention sweeper's `decoderLogTables`
list, `retention.log_days` never deletes its rows — an omission the
eleven-place walk surfaces. Until it is added, prune it by hand.

## Series navigation

**Part 11 of 14** · ←
[Part 10: DSC — Marine Digital Selective Calling]({{ '/blog/deep-dives/beyond-voice-10-dsc-marine-calling/' | relative_url }})
· Next →
[Part 12: M17 — The Open Digital Voice Link Layer]({{ '/blog/deep-dives/beyond-voice-12-m17-open-digital-voice/' | relative_url }})
