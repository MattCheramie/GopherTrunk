---
layout: page
title: DMR, P25 & NXDN encryption
description: Known-key decryption — DMR ARC4 "Enhanced Privacy", P25 ADP / DES / AES and the NXDN 15-bit scrambler — configuration, what the pipeline does, how to contribute a capture that verifies it
nav_group: Reference
---

# DMR, P25 & NXDN encryption

This page covers the air-interface encryptions GopherTrunk can decrypt
in-process with an operator-supplied key: DMR **Enhanced Privacy**
(below), P25 **ADP** ([P25 ADP](#p25-adp-rc4-algid-0xaa), verified on
air), P25 [DES-OFB / TDES / AES](#p25-des-ofb-tdes-and-aes-algids-0x81--0x83--0x86--0x85--0x84--0x89),
and the [NXDN 15-bit scrambler](#nxdn-scrambler-cipher-type-1-15-bit-key)
(verified on two known-key captures). All are configured with the same
`encryption_keys` block.

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

> **Verification status (issue #1187):** the descramble is **capture-verified**
> — the reporter's two known-key discriminator captures (key IDs 11 and 22,
> colour code 2, TG 582743, a simplex handheld) decode to speech through
> `TestDMREnhancedPrivacyReplay`, and the first three superframes of one
> transmission are pinned as literal on-air vectors in
> `internal/radio/dmr/voice/testdata/ep_issue1187_ptt1.json`. What remains is
> the daemon path on air: a live call with the key configured that records
> intelligible audio. The [`replay -key`](#checking-a-key-offline-with-replay)
> path below runs that same daemon voice path on a recording, with the release
> binary.

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
| `algorithm` | The cipher: `rc4` (aliases `arc4`, `adp`) for DMR Enhanced Privacy and P25 ADP, `des` / `tdes` / `aes` for the P25 OFB family, `scrambler` (alias `nxdn`) for the NXDN 15-bit scrambler. Anything else is rejected at config load. |
| `key` | The raw key, hex-encoded (Enhanced Privacy keys are 40 bits = 10 hex digits). Surrounding whitespace, internal spaces and an optional `0x`/`0X` prefix are tolerated. 1–32 bytes. A `scrambler` key is the radio's decimal 15-bit key, 1..32767, as the codeplug shows it. |

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
- **Every superframe also carries the NEXT superframe's MI** for late
  entry: each of the 18 frames donates a 4-bit nibble in its unprotected
  C3 bits, and the 72 bits reassemble into three Golay(24,12) codewords
  holding the 32-bit MI plus a 4-bit CRC. That it names the *next*
  superframe (the P25 Encryption Sync convention) is capture-pinned: on
  the #1187 radio every embedded IV equals the LFSR advance of the MI
  before it, the first equals the advance of the PI header's, and
  descrambling each superframe with its *own* embedded IV stays at
  ciphertext level. It is also the order DSD-FME applies (the LFSR
  advance in `dmr_alg_refresh` runs before the late-entry comparison).
  GopherTrunk verifies the embedded IV on every superframe: it confirms
  (or, when it decodes perfectly clean, corrects) the prediction for the
  superframe that follows, and a chain that missed the PI header decrypts
  the superframe it first verified an IV on by rewinding the LFSR one
  step (`RewindMI`).

The counters behind all of this are in the debug-level
`composer: dmr enhanced privacy` line: `pi_headers`, `iv_verified`,
`mi_predicted`, `mi_mismatches`, `descrambled_frames`,
`silence_frames`, `no_key`, `no_mi`.

## Checking a key offline with `replay`

`gophertrunk replay` runs a recording through the daemon's own voice path —
decode → trunking engine → voice composer → recorder — and `-key` gives that
path a key exactly as `encryption_keys` gives it to the daemon. No config file
and no Go toolchain are needed; the release binary does it:

```
# A DSD-style discriminator-audio recording (the receiver's raw, unsquelched
# FM discriminator output — mono or stereo WAV, 8/16-bit or float, any rate)
gophertrunk replay -in ep-call.wav -format disc -protocol dmr-tier2 -freq 1 \
  -record-voice -out-dir rec -key rc4:11:4E77AD0B51

# An IQ capture (gophertrunk capture, Signal Lab, SDR# baseband…)
gophertrunk replay -in ep-call.cs16 -format cs16 -sample-rate 2400000 \
  -protocol dmr-tier2 -freq 1 -record-voice -out-dir rec -key rc4:11:4E77AD0B51
```

- **What `replay` can record.** It follows the grants it decodes, on the same
  carrier. A conventional DMR recording (direct mode or a Tier II repeater)
  grants from its own Voice LC headers, so it records. A P25 call is granted
  by the control channel, so `-key adp:…` only helps when the capture holds
  that control channel and the voice; a recording of a P25 voice channel on
  its own produces no grant and no recording. For that case use the
  `TestP25ADPReplay` harness (`GT_P25_ADP_*`, needs a Go toolchain) or a live
  daemon run with `encryption_keys`.

- `-key` is `ALG:KEYID:HEXKEY` (`rc4`/`arc4` for DMR Enhanced Privacy, `adp`
  for P25 ADP) or `KEYID:HEXKEY` for RC4, and may be repeated — one per key
  ID. It is validated like a config entry.
- `-freq` is required with `-record-voice` (any non-zero value works for a
  recording).
- The call lands under `rec/replay/<talkgroup>/` as a `.wav` you can listen
  to, plus the `.raw` frames and a `.json` sidecar. Intelligible speech is the
  confirmation; the log line `composer: dmr PI header — call is encrypted …
  key_id=11 … key_configured=true` shows the key was matched.
- `-format disc` prints a one-line check of the audio first: a real
  discriminator tap keeps ~90 % of its energy below 3 kHz in its keyed
  stretches. Decoded or re-recorded audio fails that check and cannot be
  decoded by any receiver.

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
speech, ending in a one-line `VERDICT`. `GT_DMR_EP_DUMP=<json>` writes
every header and every superframe's raw on-air and FEC-decoded frames,
embedded IV and MI *before* descrambling, so a keystream hypothesis can
be tested offline without re-running the receiver. The harness slices
voice with the cadence-detecting decoder production uses; a simplex
handheld sends one burst per 60 ms frame, and the back-to-back
single-slot slicer (`GT_DMR_EP_SINGLE_SLOT=1`) reads bursts B–F out of
the gaps — on the #1187 captures that produced frames whose Golay
codewords needed the random-word 3 corrections, which looked exactly
like post-FEC scrambling and was not. **"Discriminator audio" means the
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

## P25 DES-OFB, TDES and AES (ALGIDs 0x81 / 0x83 / 0x86 / 0x85 / 0x84 / 0x89)

The same Phase 1 voice path descrambles the OFB-family algorithms when a
key is configured. The construction is the one OP25 runs on air
(`op25_crypt_des.cc` / `op25_crypt_aes.cc`), so the frame mapping is ADP's
with a different leading discard — one cipher block instead of RC4's 256
warm-up bytes:

| Algorithm | Key | IV | Discard | LDU1 u0 | LDU2 u0 |
|---|---|---|---|---|---|
| ADP (0xAA) | 40-bit | RC4 keyed key‖MI[0:8] | 256 | 267 | 368 |
| DES-OFB (0x81), TDES (0x83 two-key, 0x86 three-key) | 64 / 128 / 192-bit | MI[0:8] | 8 | 19 | 120 |
| AES-128 (0x85), AES-256 (0x84, 0x89) | 128 / 256-bit | MI expanded to 128 bits by the TIA LFSR: MI ‖ LFSR64(MI) | 16 | 27 | 128 |

Within an LDU the nine 11-byte IMBE frames are contiguous with a 2-byte
Low Speed Data gap before u8; the MI schedule (an LDU2's ES announces the
NEXT superframe, successive MIs follow the 64-bit LFSR) is identical for
every algorithm. Configure the key with `algorithm: des`, `tdes` or `aes`
— the key LENGTH selects the variant (16 vs 24 bytes for TDES, 16 vs 32
for AES) and the chain refuses a key whose length disagrees with the ALGID
on air rather than guessing.

```yaml
encryption_keys:
  - key_id: 2
    algorithm: aes
    key: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
  - key_id: 3
    algorithm: des-ofb
    key: "0123456789abcdef"
```

**Verification status (#764/#771):** reference-pinned, not capture-pinned.
The layout and the AES IV expansion are pinned against OP25's literal
offsets and a transcription of its `expand_mi_to_128`, and the synthetic
chain test proves the daemon applies them end-to-end, but GopherTrunk has
no DES/TDES/AES call of its own. `TestP25ADPReplay` accepts a DES/AES
capture with the matching key (`GT_P25_ADP_KEY` — the ES's ALGID selects
the algorithm); pitch continuity off the chance floor is the confirmation.
Not decrypted on P25: DES-XL (0x9F, a different construction) and Phase 2
TDMA (the composer's Phase 2 chain only publishes the Encryption Sync).

## NXDN scrambler (cipher type 1, 15-bit key)

NXDN radios (Kenwood NX-series, Icom IDAS) offer a built-in "scrambler"
privacy mode: a 15-bit key, 1..32767, programmed per channel or per key
slot in the codeplug. On air the SACCH superframe's VCALL message names
the call's cipher (`cipher type 1` = scrambler, 2 = DES, 3 = AES) and a
6-bit **key ID**, and the 49-bit AMBE+2 payload of every voice slot is
XORed with a keystream from a 15-bit LFSR seeded by the key. With the
key configured, the NXDN voice chain descrambles in-process; the
recording is clear speech and the call is logged as encrypted with its
cipher and key ID. GopherTrunk never brute-forces a key on air (the
cryptolab `nxdn brute` tool does that offline against a frame dump, by
design separately — see [cryptolab](cryptolab.md)).

Keys go either on an NXDN **trunking system**, or — the common case, a
conventional NXDN channel in the scan list — on the **channel** itself
(`scanner.conventional[].encryption_keys`, also on a wideband dongle's
`conventional:` list). `key_id` is the key ID the VCALL carries, which is
the key slot number the radios were programmed with; a channel used by
several groups with different keys lists one entry per key ID:

```yaml
scanner:
  conventional:
    - label: "Plant ops (NXDN96)"
      frequency_hz: 447_100_000
      mode: nxdn                   # nxdn (12.5 kHz NXDN96) or nxdn48
      squelch_dbfs: -50
      encryption_keys:
        - key_id: 3
          algorithm: scrambler
          key: "32767"             # the radio's decimal 15-bit key
        - key_id: 1
          algorithm: scrambler
          key: "12345"
```

What the voice chain does:

1. Every LICH-valid traffic frame's SACCH block is decoded (deinterleave
   → depuncture → K=5 Viterbi → CRC-6, register preset 0x3F / taps 0x27,
   the MMDVMHost / DSD-FME CRC). Four blocks (structure field 3, 2, 1, 0)
   assemble the 72-bit VCALL: source, destination, call type, cipher
   type, key ID. It is logged (`composer: nxdn VCALL — call is encrypted
   … cipher=scrambler key_id=3 key_configured=true`) and published as
   `call.encryption` (`algorithm_id` = cipher type, `key_id`), so the
   call log, web UI and `recordings.mute_encrypted` treat it like a DMR
   or P25 encrypted call.
2. The key for `(algorithm scrambler, key_id)` is looked up on the
   call's system, then on the channel at the call's frequency.
3. While keys are configured, voice is held until the first VCALL names
   the cipher (one superframe, ~320 ms, capped at two); a clear call's
   frames are released as they are, a scrambled call's are descrambled
   when the key is known, otherwise released as ciphertext (the vocoder
   then renders noise — set `mute_encrypted` to silence it). With no
   keys configured anywhere the voice path is untouched.
4. The keystream schedule is the one the reporter's two known-key
   captures pinned (issue #1261): the register — output = state bit 0,
   feedback = bit 0 ⊕ bit 1 shifted in at bit 14 — is reseeded from the
   key at the first frame of every SACCH superframe and then runs 4 × 49
   bits per LICH-valid air frame, voice frame or not; slot *v* takes bits
   [49v, 49v+49). A brute force over all 32768 keys by AMBE pitch
   continuity ranks the radios' programmed key first on both captures
   (0.66 and 0.65 continuity against a 0.16 ciphertext floor). The call
   ended log line carries `sacch_ok` / `sacch_fail`, `descrambled_frames`,
   `ciphertext_frames` and `unaligned_frames` (frames released raw before
   any SACCH block fixed the superframe phase).

Check a capture offline with the harness (`GT_NXDN_SCR_IQ` = a cs16 IQ
capture of the voice channel, e.g. the composer's `voice_iq_debug` file):

```sh
GT_NXDN_SCR_IQ=447.100MHz.cs16 GT_NXDN_SCR_RATE=2400000 GT_NXDN_SCR_KEY=32767 \
  go test ./cmd/gophertrunk -run TestNXDNScramblerReplay -v
```

It prints the VCALL, the SACCH CRC yield and the pitch continuity of the
voice as received and descrambled; `GT_NXDN_SCR_BRUTE=1` sweeps every key
and ranks yours. Not decrypted: NXDN DES and AES (cipher types 2 and 3;
identified and logged, recorded as ciphertext).

**Status:** verified offline on the reporter's two known-key captures
(keys 32767 and 12345, key IDs 3 and 1, both 128+ voice frames of
speech after descrambling); a live daemon run with the key configured on
the scan-list channel is the on-air confirmation still to come.

## Keys from a KFDtool key container (`.ekc`)

[KFDtool](https://github.com/omahacommsys/KFDtool) is an open keyloader. Its
`.ekc` files hold an operator's P25 keys encrypted under a password
(PBKDF2-SHA512 and AES-256). GopherTrunk can load its keys from one instead
of listing them in plain text under `encryption_keys` (#1305):

```yaml
trunking:
  systems:
    - name: county
      protocol: p25
      control_channels: [851_012_500]
      encryption_key_files:
        - path: keys.ekc
          password_env: GT_EKC_PASSWORD
```

- **The password** comes from `password_env` (an environment variable),
  `password` (stored in the config in clear), or, with neither set, a prompt
  on the terminal at startup. Without a terminal the daemon refuses to start
  rather than run without its keys. A wrong password stops startup too: a
  typed one is asked for again, up to three times.
- **Which keys load:** traffic keys (TEKs) whose algorithm GopherTrunk
  decrypts: DES-OFB (0x81), TDES (0x83 / 0x86), AES (0x84 / 0x85 / 0x89) and
  ADP (0xAA). Key-encryption keys (KEKs) and other algorithms (DES-XL 0x9F)
  are skipped. When two keysets carry the same key ID, the active keyset's
  key wins; `keyset: N` loads only keyset N.
- **Explicit keys win.** An `encryption_keys` entry with the same `key_id`
  as a container key is used instead of it.
- **Nothing is written back.** The decrypted keys stay in the daemon's
  memory. The config, the API and the config builders never see them. The
  daemon logs how many keys it loaded and why any were skipped, never the
  keys themselves:

  ```
  daemon: decryption keys loaded from key container system=county file=keys.ekc keys=3 skipped="1 KEK, 1 0x9F (DES-XL), which GopherTrunk does not decrypt"
  ```

To check a container and its password without starting the daemon (here, the
test container in `internal/ekc/testdata`):

```
$ gophertrunk ekc -in keys.ekc
Password for key container keys.ekc:
ITEM  NAME             KEYSET      SLN  KEY ID         ALGORITHM       TYPE  BYTES  LOADED AS
1     County AES       1 (active)  1    1 (0x1)        0x84 (AES-256)  TEK   32     aes
2     Fire DES         1 (active)  2    42 (0x2A)      0x81 (DES-OFB)  TEK   8      des
3     Ops ADP          1 (active)  3    3 (0x3)        0xAA (ADP/RC4)  TEK   5      rc4
4     Master KEK       1 (active)  4    4 (0x4)        0x84 (AES-256)  KEK   32     -
5     County AES next  2           1    1 (0x1)        0x85 (AES-128)  TEK   16     -
6     Legacy DES-XL    1 (active)  6    6 (0x6)        0x9F (DES-XL)   TEK   8      -
7     Ops TDES         1 (active)  7    4660 (0x1234)  0x83 (TDES-2)   TEK   24     tdes

4 key(s) would load; skipped: 1 KEK, 1 superseded by the active keyset, 1 0x9F (DES-XL), which GopherTrunk does not decrypt
```

The reader is pinned against a container written by KFDtool's own code.

### Managing a container from GopherTrunk

`.ekc` is also GopherTrunk's native keystore: every key type GopherTrunk
decrypts has an ALGID a container holds, including DMR Enhanced Privacy
(RC4, stored as ADP 0xAA, the same cipher keyed the same way). So you can
create and edit containers with `gophertrunk ekc` and they stay KFDtool
containers, which open and edit in KFDtool too:

```
gophertrunk ekc create -out keys.ekc                       # asks for a new password, twice
gophertrunk ekc add -in keys.ekc -key-id 11 -algorithm rc4 -name "Ops DMR EP"
gophertrunk ekc add -in keys.ekc -key-id 1 -algorithm aes -name "County AES"
gophertrunk ekc list -in keys.ekc
gophertrunk ekc remove -in keys.ekc -item 2                # ITEM from the listing
gophertrunk ekc passwd -in keys.ekc
```

- **The key is never a command-line argument** (that would leave it in shell
  history and the process list). `add` asks for it at a prompt that does not
  show it, or reads it from an environment variable named by `-key-env`.
  Passwords come from a prompt or `-password-env` / `-new-password-env`.
  Nothing prints key material.
- `-algorithm` is `rc4` (or `adp`), `des`, `tdes` or `aes`, as in
  `encryption_keys`. The key length picks the variant: TDES 16 bytes → 0x83,
  24 → 0x86; AES 16 → 0x85, 32 → 0x84. `-keyset N` (default 1) and `-kek`
  match KFDtool's fields.
- A container is rewritten atomically with mode 0600. Everything KFDtool
  stores is kept: item numbers, key groups and its counters.
- A new password must be at least 8 characters.

To **move existing plain-text keys** out of the config:

```
gophertrunk ekc import -config gophertrunk.yaml -system county -out county.ekc
```

This writes the system's `encryption_keys` into the container (creating it,
or adding to an existing one) and prints the `encryption_key_files` entry to
put in their place. It does not edit the config file: delete the plain-text
keys yourself, and remember backups and version control may still hold them.

Containers GopherTrunk writes are checked byte for byte against KFDtool's
own: the encrypted inner key list is identical to what KFDtool's code writes
for the same keys, and the outer wrapper differs only in its random salt and
ciphertext. `internal/ekc/testdata/README.md` has a small .NET harness that
opens a GopherTrunk-written container with KFDtool's own reader.

### Plain-text keys and the config builder API

Keys that do stay in `encryption_keys`, and a container `password` kept in
the config, are never sent through the API. The config builder's
`GET /api/v1/config/file` (an open read, like the rest of the builder's
reads) returns them as `<redacted>`, and the web builder shows the key field
masked. Saving a draft puts the stored values back: a key by its system name
and `key_id`, a password by its system name and container `path`. A
`<redacted>` value with nothing to restore (a changed `key_id` or `path`, a
renamed system, a new file) is refused with a message saying to enter the
key again, so a save never writes the placeholder in place of a key. The
terminal config editor works on the file directly and shows keys as stored.

## Calls you cannot decrypt: `recordings.mute_encrypted`

Without a key the vocoder renders the ciphertext as random-parameter
"speech" — loud, unintelligible, and easy to mistake for a decode fault.
`recordings.mute_encrypted: true` records and streams **silence** for an
encrypted call no configured key covers (the behaviour of every hardware
scanner, SDRTrunk, trunk-recorder and DSD-FME), while the `.raw` / `.imb` /
`.amb` sidecars keep the ciphertext frames and the call is logged, shown and
uploaded exactly as before. A call whose key id matches an `encryption_keys`
entry is decrypted in-process and never muted; encryption discovered
mid-call (a P25 LDU2 Encryption Sync) mutes from that frame on. The
call-ended log line carries `muted_frames` so a silent recording of an
encrypted call reads as intended. `skip_encrypted` remains the stronger
option (no file at all).

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
- **DES / AES** DMRA algorithms on DMR: the cryptolab keystream generator
  realises the cores, but the DMR voice path does not apply them (no
  capture to validate the IV expansion against). On P25 they are applied
  (section above).
- **Motorola Basic Privacy** (a 16-bit scrambler, not a cipher).
- **NXDN DES / AES** (VCALL cipher types 2 and 3): identified from the
  SACCH and logged, not decrypted (no capture).

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
