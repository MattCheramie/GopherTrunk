---
layout: page
title: DMR & P25 encryption
description: Known-key RC4 decryption — DMR ARC4 "Enhanced Privacy" and P25 ADP — configuration, what the pipeline does, how to contribute a capture that verifies it
nav_group: Reference
---

# DMR & P25 encryption (RC4: "Enhanced Privacy" and ADP)

This page covers the two RC4-based air-interface encryptions GopherTrunk
can decrypt in-process with an operator-supplied key: DMR **Enhanced
Privacy** (below) and P25 **ADP** ([P25 ADP](#p25-adp-rc4-algid-0xaa),
verified on air). Both are configured with the same `encryption_keys`
block.

# DMR encryption (ARC4 / RC4 "Enhanced Privacy")

GopherTrunk can be configured with **known decryption keys** for DMR
systems that protect voice with the DMRA ARC4/RC4 "Enhanced Privacy"
algorithm. With a key configured, an encrypted call is descrambled
**in-process** and recorded as clear audio; without one, the call is
still captured (ciphertext) and its privacy header is decoded and
surfaced so the operator can see which key it needs.

This is **known-key** support only: the operator supplies a key they
are authorized to hold. GopherTrunk performs **no key recovery** of
any kind — it is the same model used by SDRTrunk, DSD-FME and OP25.
Only monitor systems you are legally permitted to monitor.

> **Verification status (issue #1187):** the descramble is pinned by
> reference vectors and a full-chain synthetic test, but it has **not yet
> been verified on air** — no known-key encrypted capture has been run
> through it. See [Contributing a known-key capture](#contributing-a-known-key-capture)
> below; that is the one thing that closes the loop.

## Configuration

Encryption keys are declared per trunking system, under
`trunking.systems[].encryption_keys`:

```yaml
trunking:
  systems:
    - name: "Example-DMR"
      protocol: dmr
      control_channels: [451_000_000]
      talkgroup_file: "/etc/gophertrunk/talkgroups-dmr.csv"
      encryption_keys:
        - key_id: 1            # matches the key ID in the privacy header
          algorithm: rc4       # only "rc4" / "arc4" is accepted today
          key: "0123456789"    # hex; whitespace and a "0x" prefix are ignored
```

Each entry has three fields:

| Field | Meaning |
| --- | --- |
| `key_id` | The key identifier the radios advertise in the Privacy Indicator header. A system that rotates between several keys resolves to the right one by ID. |
| `algorithm` | The cipher. Only `rc4` (alias `arc4`) is accepted. `aes` / `des` are rejected at config-load with an explicit "not supported yet" error so the schema can grow later without a config break. |
| `key` | The raw key, hex-encoded (Enhanced Privacy keys are 40 bits = 10 hex digits). Surrounding whitespace, internal spaces and an optional `0x`/`0X` prefix are tolerated. 1–32 bytes. |

The config is validated when the daemon loads it: an unknown
algorithm, malformed hex, an over-length key, or a duplicate `key_id`
within one system is a hard error. At startup the daemon logs
`in-process decryption keys configured (DMR enhanced privacy)` when
any system carries a key.

## What the pipeline does

A DMR voice call flows through these stages:

1. **Control-channel decode** — the DMR Tier II / Tier III decoders
   emit a `trunking.Grant`. The grant's `Encrypted` flag is read from
   the Full Link Control `ServiceOptions` bit, so an encrypted call is
   known as such before voice even starts.
2. **Privacy Indicator header** (`internal/radio/dmr/pi.go`,
   `dmr/voice/pi_detector.go`) — the data burst an encrypted
   transmission sends after its Voice LC Header carries the algorithm
   ID, the key ID and the 32-bit **Message Indicator** (the
   per-transmission IV). The voice chain decodes it (slot type →
   BPTC(196,96) → CRC-CCITT with the PI-header mask) and publishes a
   `call.encryption` event, so the call log, the web console and the
   webhook carry `algorithm_id` / `key_id` for DMR exactly as they do
   for P25. The header is logged with its raw octets
   (`composer: dmr PI header — call is encrypted`).
3. **Voice superframe decode** (`internal/radio/dmr/voice`) — the
   composer runs an IQ → DMR receiver → superframe-decoder chain on
   the granted voice channel, locking onto the A–F voice superframe
   and extracting its eighteen 72-bit on-air AMBE+2 frames.
4. **AMBE+2 forward-error-correction** (`ambefec.go`) — each 72-bit
   on-air frame is deinterleaved, Golay-corrected and assembled into
   the 49-bit vocoder payload.
5. **Enhanced Privacy descramble** (`dmr/voice/ep.go`,
   `composer/dmr_ep.go`) — for a call known to be encrypted, each
   superframe's Message Indicator is resolved (below) and, when a
   configured key matches the header's key ID, the 18 payloads are
   XORed with the RC4 keystream. Frames that failed FEC keep their
   keystream slot so the rest stay aligned; the AMBE+2 silence vector
   is passed through in clear, as radios transmit it.
6. **Vocoder → WAV** (`internal/voice/ambe2`, `"ambe2-dmr"`) — the
   (now clear) frames are rendered to 8 kHz PCM and written to the
   call's `.wav`; the 49-bit frames also land in the `.raw` sidecar.

### The keystream construction

Two independent open decoders agree on it (SDRTrunk's header and
embedded-parameter parsers; DSD-FME's on-air-verified decrypt), and
GopherTrunk's implementation is pinned against literal vectors computed
from their stated algorithms, not ported code:

- **RC4 key = key ‖ Message Indicator** (4 bytes, MSB first). The first
  **256** keystream bytes are discarded — the same warm-up as P25 ADP —
  then **7 bytes per 49-bit frame**, contiguous across the superframe.
- **The MI advances once per superframe** through a 32-bit LFSR
  (x³² + x⁴ + x² + 1, 32 shifts). The PI header's MI is the first
  superframe's.
- **Every superframe also carries its own MI** for late entry: each of
  the 18 frames donates a 4-bit nibble in its unprotected C3 bits, and
  the 72 bits reassemble into three Golay(24,12) codewords holding the
  32-bit MI plus a 4-bit CRC. GopherTrunk verifies this embedded IV on
  every superframe: it confirms (or, when it decodes perfectly clean,
  corrects) the LFSR prediction, and lets a chain that missed the PI
  header start decrypting from the next full superframe.

The counters behind all of this are in the debug-level
`composer: dmr enhanced privacy` line: `pi_headers`, `iv_verified`,
`mi_predicted`, `mi_mismatches`, `descrambled_frames`,
`silence_frames`, `no_key`, `no_mi`.

## Offline analysis: the crypto-frame capture

With `recordings.crypto_capture_path` set, every encrypted DMR
superframe is also appended to the cryptolab JSONL bridge — `protocol:
"dmr"`, the superframe's MI as `iv`, the 18 packed frames (126 bytes,
as on air) as `ct`, plus `algid` / `keyid` — the same artifact the P25
path writes and that `gophertrunk cryptolab` consumes. Superframes with
an FEC-failed frame are skipped rather than padded, so the keystream
positions in `ct` are exact.

## Contributing a known-key capture

A green synthetic test is not an on-air verification (this repo's
#764/#771 rule). What verifies the descramble is **one clear recording
of an Enhanced Privacy transmission from a radio whose key you hold**,
run through the replay harness. Please attach it to the issue (or link
a file drop) rather than sending it by email — the maintainers work
from the tracker, and the recipe has to be reproducible.

What to send:

1. **An IQ capture** of the encrypted call, made with the daemon's own
   tool so the rate and format are known:
   ```
   gophertrunk capture -freq <Hz> -sample-rate 2400000 -seconds 30 -protocol dmr -format cs16 -out ep-call.cs16
   ```
   (a Signal Lab wav/flac capture works too). A **DSD-FME-style
   discriminator-audio WAV** (16-bit mono) is also accepted.
2. **The key** (hex, e.g. `0123456789`) and the **key ID** the radios
   are programmed with, and the radio model / firmware if you know it.
3. Optionally the DSD-FME output for the same file — its `DMR PI H- ALG
   ID / KEY ID / MI` lines are the cross-check.

Then, on a build from `main`:

```
GT_DMR_EP_IQ=ep-call.cs16 GT_DMR_EP_RATE=2400000 GT_DMR_EP_KEY=0123456789 GT_DMR_EP_OUT=ep-call.wav \
  go test ./cmd/gophertrunk -run TestDMREnhancedPrivacyReplay -v
```

(`GT_DMR_EP_AUDIO=<wav>` instead of `GT_DMR_EP_IQ` for discriminator
audio.) The harness prints every PI header it decoded (algorithm, key
ID, MI, raw octets), the per-superframe embedded-IV / MI chain, the
descrambled frame count and the seconds of the output WAV that carry
speech, ending in a one-line `VERDICT`. **"Discriminator audio" means the
receiver's raw, unsquelched FM discriminator output** (what DSD-FME is fed
through a virtual cable), not a radio's speaker or a decoder's playback:
the #1187 DMR files were 96 kHz recordings of the *decoded* (garbled)
audio — white-noise bursts gated at the TDMA slot cadence — from which no
receiver can recover a burst, and the harness now says so ("NOT a
demodulable 4FSK discriminator tap") instead of a bare "nothing decoded".
When in doubt, record IQ with `gophertrunk capture`. Listen to the WAV: intelligible
speech is the confirmation; a flat hiss with everything else decoding
points at the key / key ID / construction, and the header lines are the
evidence to post.

## P25 ADP (RC4, ALGID 0xAA)

P25's ADP ("Advanced Digital Privacy") is the same cipher keyed the same
way — RC4 with the 40-bit key followed by the Message Indicator, the
first 256 keystream bytes discarded — and is **verified on air** (issue
#1187: a reporter's 6.2 s ADP call, NAC 239 / key ID 1, decodes to
speech with the configured key). Configure it exactly like a DMR key,
under the P25 system:

```yaml
trunking:
  systems:
    - name: "Example-P25-ADP"
      protocol: p25
      control_channels: [851_000_000]
      encryption_keys:
        - key_id: 1            # the KID the LDU2 Encryption Sync names
          algorithm: adp       # "adp" / "rc4" / "arc4" are one family
          key: "1234567890"    # 40-bit key, hex
```

What the Phase 1 voice chain does (`composer/p25_adp.go`,
`phase1/adp.go`):

1. Every LDU2 carries an **Encryption Sync** (algorithm ID, key ID,
   72-bit Message Indicator). The chain already surfaces it as
   `call.encryption`; with a configured key for that key ID and
   algorithm 0xAA it also decrypts.
2. **The ES names the MI of the *next* superframe** (LDU1 + LDU2, 360 ms),
   and successive MIs follow the 64-bit LFSR x⁶⁴+x⁶²+x⁴⁶+x³⁸+x²⁷+x¹⁵+1
   (TIA-102.AAAD). The first superframe's MI is normally carried by the
   HDU, which the LDU assembler does not deliver — so the chain **rewinds**
   the LFSR from the first ES it decodes to recover the current
   superframe's MI, holding that superframe's frames (once per call, at
   most 360 ms) until the ES arrives. An LDU2 whose ES fails RS advances
   the chain by the LFSR prediction instead.
3. Per superframe the keystream is applied to the FEC-decoded 88-bit IMBE
   frames: LDU1's nine frames from absolute keystream byte 267, LDU2's
   from 368, 11 bytes each, with a 2-byte gap before the ninth (the Low
   Speed Data word) — OP25's layout. The recorder then vocodes clear frames.

Both the layout and the MI schedule are pinned by the reporter's capture
(`internal/radio/p25/phase1/testdata/adp_issue1187_ldus.json`): under the
production rules the IMBE pitch track goes from ciphertext-random to
speech-continuous, every alternative (wrong key, MI applied to the current
superframe, no 11-byte skip) stays random, and all fourteen consecutive
MIs satisfy the LFSR.

Replay a capture of your own:

```
GT_P25_ADP_IQ=adp-call.cs16 GT_P25_ADP_RATE=2400000 GT_P25_ADP_KEY=1234567890 GT_P25_ADP_OUT=adp-call.wav \
  go test ./cmd/gophertrunk -run TestP25ADPReplay -v
```

(`GT_P25_ADP_AUDIO=<wav>` for discriminator audio.) It prints every LDU2's
ES, the descrambled frame count and a `VERDICT` based on IMBE pitch
continuity — never on loudness (random vocoder parameters are loud).

Not decrypted on P25: DES-OFB, TDES, AES-128/256 — identified and
published, and the cryptolab realises their keystreams, but no capture has
validated the IV expansion onto the voice frames.

## Decoding the `.raw` sidecar out-of-band

The `.raw` file is a flat concatenation of 7-byte frames, each holding
one FEC-decoded 49-bit AMBE+2 voice frame (MSB-first, 49 bits + 7 bits
of zero padding). This is a standard AMBE+2 frame format and can be
fed to an external AMBE decoder (an mbelib-based tool, DSD-FME, or
DVSI hardware) to produce audio. For an encrypted call with no
configured key it holds the *encrypted* frames.

## What is not implemented

- **Hytera Enhanced Privacy** (FID 0x68, a 40-bit MI and a vendor key
  schedule) and **Kirisun** privacy are recognised in the PI header and
  logged, but not decrypted.
- **DES / AES** DMRA algorithms: the cryptolab keystream generator
  realises the cores, but the voice path does not apply them (no
  capture to validate the IV expansion against).
- **Motorola Basic Privacy** (a 16-bit scrambler, not a cipher).

## References

- ETSI TS 102 361-1 (PI header data type, CRC masks §B.3.12).
- SDRTrunk (Apache-2.0): `EncryptionParameters`, `PiHeader`,
  `VoiceSuperFrameProcessor`, `Golay24` — header layout, embedded-IV
  fragment positions and reassembly, CRC-4.
- DSD-FME (GPL — read for the protocol conventions only, nothing
  ported): `dmr_pi.c` (header fields, MI LFSR), `dmr_le.c` (late-entry
  MI), `dsd_mbe.c` / `crypt-rc4.c` (key‖MI, 256-byte drop, 7 bytes per
  frame, silence passthrough).
- The AMBE+2 FEC is ported, with bit layouts preserved 1:1, from
  mbelib and szechyjs/dsd (ISC) — attribution in `THIRD_PARTY_LICENSES.md`.

See also [`docs/vocoders.md`](vocoders.md) for the IMBE / AMBE+2
licensing landscape and the vocoder plugin model.
